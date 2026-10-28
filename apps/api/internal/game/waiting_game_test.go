package game

import (
	"context"
	"errors"
	"testing"
	"time"

	"api/internal/domain"
)

func (h *harness) seedWaitingGame(t *testing.T) *domain.Game {
	t.Helper()
	g, err := h.svc.CreateWaitingGame(context.Background(), CreateGameInput{
		WhiteID: whiteID, BlackID: blackID, TimeControl: blitz(),
	})
	if err != nil {
		t.Fatalf("CreateWaitingGame: %v", err)
	}
	return g
}

func TestCreateWaitingGameArmsNoTimer(t *testing.T) {
	h := newHarness()

	g := h.seedWaitingGame(t)

	if g.Status != domain.GameStatusWaiting {
		t.Errorf("Status = %q, want waiting", g.Status)
	}
	if g.WhiteTimeRemaining != 3*time.Minute || g.BlackTimeRemaining != 3*time.Minute {
		t.Errorf("clocks = (%v, %v), want both at the 3m initial", g.WhiteTimeRemaining, g.BlackTimeRemaining)
	}
	if len(h.timers.schedules) != 0 {
		t.Errorf("armed %d timers for an unaccepted invite, want 0", len(h.timers.schedules))
	}
}

func TestCreateWaitingGameValidatesInput(t *testing.T) {
	h := newHarness()

	_, err := h.svc.CreateWaitingGame(context.Background(), CreateGameInput{
		WhiteID: "x", BlackID: "x", TimeControl: blitz(),
	})
	if err == nil {
		t.Error("expected an error for a player inviting themselves")
	}
}

// A waiting game isn't "active", so it must not stop either player from
// queueing or accepting a different invite.
func TestWaitingGameIsNotAnActiveGame(t *testing.T) {
	h := newHarness()
	h.seedWaitingGame(t)

	active, err := h.svc.HasActiveGame(context.Background(), whiteID)
	if err != nil {
		t.Fatalf("HasActiveGame: %v", err)
	}
	if active {
		t.Error("a waiting invite counted as an active game")
	}
}

func TestStartWaitingGameStartsTheClock(t *testing.T) {
	h := newHarness()
	g := h.seedWaitingGame(t)
	// Make the row look like it sat unanswered for a while.
	h.repo.stored(g.ID).LastMoveAt = time.Now().UTC().Add(-45 * time.Second)

	before := time.Now().UTC()
	started, err := h.svc.StartWaitingGame(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("StartWaitingGame: %v", err)
	}

	stored := h.repo.stored(g.ID)
	if stored.Status != domain.GameStatusActive || started.Status != domain.GameStatusActive {
		t.Errorf("status = %q (stored %q), want active", started.Status, stored.Status)
	}
	// Otherwise white's first move would be charged the 45s the invite waited.
	if stored.LastMoveAt.Before(before) {
		t.Errorf("LastMoveAt = %v, want reset to accept time (>= %v)", stored.LastMoveAt, before)
	}
	if len(h.timers.schedules) != 1 {
		t.Fatalf("armed %d timers, want 1", len(h.timers.schedules))
	}
	if s := h.timers.schedules[0]; s.gameID != g.ID || s.remaining != 3*time.Minute || s.color != domain.White {
		t.Errorf("schedule = %+v, want white's full 3m on %s", s, g.ID)
	}
}

func TestStartWaitingGameRejectsNonWaitingGames(t *testing.T) {
	h := newHarness()
	active := h.seedActiveGame(t, blitz())

	if _, err := h.svc.StartWaitingGame(context.Background(), active.ID); !errors.Is(err, domain.ErrGameNotFound) {
		t.Errorf("starting an active game: err = %v, want ErrGameNotFound", err)
	}
	if _, err := h.svc.StartWaitingGame(context.Background(), "nope"); !errors.Is(err, domain.ErrGameNotFound) {
		t.Errorf("starting a missing game: err = %v, want ErrGameNotFound", err)
	}
	if len(h.timers.schedules) != 0 {
		t.Error("a timer was armed for a rejected start")
	}
}

func TestDiscardWaitingGame(t *testing.T) {
	h := newHarness()
	g := h.seedWaitingGame(t)

	if err := h.svc.DiscardWaitingGame(context.Background(), g.ID); err != nil {
		t.Fatalf("DiscardWaitingGame: %v", err)
	}
	if h.repo.stored(g.ID) != nil {
		t.Error("the waiting game is still stored")
	}
	if err := h.svc.DiscardWaitingGame(context.Background(), g.ID); !errors.Is(err, domain.ErrGameNotFound) {
		t.Errorf("second discard: err = %v, want ErrGameNotFound", err)
	}
}

// Discarding must never delete a game that's been started.
func TestDiscardWaitingGameLeavesActiveGamesAlone(t *testing.T) {
	h := newHarness()
	active := h.seedActiveGame(t, blitz())

	if err := h.svc.DiscardWaitingGame(context.Background(), active.ID); !errors.Is(err, domain.ErrGameNotFound) {
		t.Errorf("err = %v, want ErrGameNotFound", err)
	}
	if h.repo.stored(active.ID) == nil {
		t.Error("an active game was deleted")
	}
}

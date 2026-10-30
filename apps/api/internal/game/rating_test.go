package game

import (
	"context"
	"errors"
	"testing"
	"time"

	"api/internal/domain"
)

func bullet() domain.TimeControl { return domain.TimeControl{Initial: time.Minute} }

func TestCreateGameSnapshotsTheCategoryRatings(t *testing.T) {
	h := newHarness()
	h.repo.setRating(whiteID, domain.CategoryBullet, domain.PlayerRating{Rating: 1500, GamesPlayed: 30})
	h.repo.setRating(blackID, domain.CategoryBullet, domain.PlayerRating{Rating: 900, GamesPlayed: 3})
	h.repo.setRating(whiteID, domain.CategoryBlitz, domain.PlayerRating{Rating: 2000})

	g := h.seedActiveGame(t, bullet())

	want := domain.GameRatings{
		Category: domain.CategoryBullet,
		White:    domain.SideRating{Before: 1500, K: 20},
		Black:    domain.SideRating{Before: 900, K: 40},
	}
	if g.Ratings == nil || *g.Ratings != want {
		t.Errorf("Ratings = %+v, want %+v", g.Ratings, want)
	}
}

// Each way a game ends applies the same Elo change, from the loser's side
// and the winner's.
func TestEveryEndingAppliesTheRatingChange(t *testing.T) {
	endings := []struct {
		name string
		end  func(h *harness, g *domain.Game) error
	}{
		{"checkmate", func(h *harness, g *domain.Game) error {
			h.validator.result = MoveResult{FEN: "8/8/8/8/8/8/8/8 b - - 0 1", SAN: "Qh5#", Status: PositionCheckmate}
			_, _, err := h.svc.SubmitMove(context.Background(), SubmitMoveInput{ID: g.ID, PlayerID: whiteID, From: "d1", To: "h5"})
			return err
		}},
		{"resignation", func(h *harness, g *domain.Game) error {
			_, err := h.svc.Resign(context.Background(), g.ID, blackID)
			return err
		}},
		{"timeout", func(h *harness, g *domain.Game) error {
			_, err := h.svc.HandleTimeExpired(context.Background(), g.ID, domain.Black)
			return err
		}},
	}
	for _, e := range endings {
		t.Run(e.name, func(t *testing.T) {
			h := newHarness()
			g := h.seedActiveGame(t, bullet()) // both 1200, provisional: K=40

			if err := e.end(h, g); err != nil {
				t.Fatalf("ending the game: %v", err)
			}

			if got := h.repo.rating(whiteID, domain.CategoryBullet); got != (domain.PlayerRating{Rating: 1220, GamesPlayed: 1}) {
				t.Errorf("white = %+v, want 1220 after 1 game", got)
			}
			if got := h.repo.rating(blackID, domain.CategoryBullet); got != (domain.PlayerRating{Rating: 1180, GamesPlayed: 1}) {
				t.Errorf("black = %+v, want 1180 after 1 game", got)
			}
			stored := h.repo.stored(g.ID).Ratings
			if stored.White.Delta == nil || *stored.White.Delta != 20 || *stored.Black.Delta != -20 {
				t.Errorf("stored deltas = %v / %v, want +20 / -20", stored.White.Delta, stored.Black.Delta)
			}
			if over := h.events.gameOver; len(over) != 1 || over[0].Ratings.White.Delta == nil {
				t.Error("the game_over event doesn't carry the deltas")
			}
		})
	}
}

func TestADrawMovesUnequalPlayersTogether(t *testing.T) {
	h := newHarness()
	h.repo.setRating(whiteID, domain.CategoryBullet, domain.PlayerRating{Rating: 1600, GamesPlayed: 50})
	h.repo.setRating(blackID, domain.CategoryBullet, domain.PlayerRating{Rating: 1200, GamesPlayed: 50})
	g := h.seedActiveGame(t, bullet())
	h.validator.result = MoveResult{FEN: "8/8/8/8/8/8/8/8 b - - 0 1", SAN: "Kb6", Status: PositionStalemate}

	if _, _, err := h.svc.SubmitMove(context.Background(), SubmitMoveInput{ID: g.ID, PlayerID: whiteID, From: "a1", To: "b6"}); err != nil {
		t.Fatalf("SubmitMove: %v", err)
	}

	if got := h.repo.rating(whiteID, domain.CategoryBullet).Rating; got != 1592 {
		t.Errorf("favourite = %d, want 1592", got)
	}
	if got := h.repo.rating(blackID, domain.CategoryBullet).Rating; got != 1208 {
		t.Errorf("underdog = %d, want 1208", got)
	}
}

// A timeout that fires just after a resignation (or the reverse) must not
// rate the game twice.
func TestASecondEndingAppliesNothing(t *testing.T) {
	h := newHarness()
	g := h.seedActiveGame(t, bullet())
	if _, err := h.svc.Resign(context.Background(), g.ID, blackID); err != nil {
		t.Fatalf("Resign: %v", err)
	}

	_, err := h.svc.HandleTimeExpired(context.Background(), g.ID, domain.White)

	if !errors.Is(err, domain.ErrGameAlreadyFinished) {
		t.Errorf("second ending = %v, want ErrGameAlreadyFinished", err)
	}
	if got := h.repo.rating(whiteID, domain.CategoryBullet); got.GamesPlayed != 1 || got.Rating != 1220 {
		t.Errorf("white = %+v, want rated once", got)
	}
}

func TestOnlyBlitzMovesTheDisplayRating(t *testing.T) {
	for _, c := range []struct {
		tc   domain.TimeControl
		want bool
	}{
		{bullet(), false},
		{blitz(), true},
		{domain.TimeControl{Initial: 10 * time.Minute}, false},
	} {
		h := newHarness()
		g := h.seedActiveGame(t, c.tc)
		if _, err := h.svc.Resign(context.Background(), g.ID, blackID); err != nil {
			t.Fatalf("Resign: %v", err)
		}
		_, moved := h.repo.blitzDisplay[whiteID]
		if moved != c.want {
			t.Errorf("%s: display rating moved = %v, want %v", g.Ratings.Category, moved, c.want)
		}
	}
}

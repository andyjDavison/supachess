package game

import (
	"context"
	"sync"
	"testing"
	"time"

	"api/internal/domain"
)

type expiryCall struct {
	gameID string
	color  domain.Color
}

type fakeExpirer struct {
	mu    sync.Mutex
	calls []expiryCall
	fired chan expiryCall
}

func newFakeExpirer() *fakeExpirer {
	return &fakeExpirer{fired: make(chan expiryCall, 10)}
}

func (f *fakeExpirer) HandleTimeExpired(_ context.Context, gameID string, expiredColor domain.Color) (*domain.Game, error) {
	call := expiryCall{gameID, expiredColor}
	f.mu.Lock()
	f.calls = append(f.calls, call)
	f.mu.Unlock()
	f.fired <- call
	return &domain.Game{ID: gameID}, nil
}

func (f *fakeExpirer) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func TestTimerManagerFiresWithTheExpiredColor(t *testing.T) {
	exp := newFakeExpirer()
	tm := NewTimerManager(exp)

	tm.Schedule("game-1", 10*time.Millisecond, domain.Black)

	select {
	case got := <-exp.fired:
		if got.gameID != "game-1" || got.color != domain.Black {
			t.Errorf("fired with %+v, want game-1/black", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the timer never fired")
	}
}

func TestTimerManagerCancelPreventsFiring(t *testing.T) {
	exp := newFakeExpirer()
	tm := NewTimerManager(exp)

	tm.Schedule("game-1", 50*time.Millisecond, domain.White)
	tm.Cancel("game-1")

	time.Sleep(150 * time.Millisecond)

	if n := exp.count(); n != 0 {
		t.Errorf("HandleTimeExpired ran %d times after Cancel, want 0", n)
	}
}

// Every move re-arms the clock, so a second Schedule must replace the
// first rather than leave both pending - otherwise a game would flag on a
// stale timer from an earlier move.
func TestTimerManagerRescheduleReplacesThePreviousTimer(t *testing.T) {
	exp := newFakeExpirer()
	tm := NewTimerManager(exp)

	tm.Schedule("game-1", 40*time.Millisecond, domain.White)
	tm.Schedule("game-1", 300*time.Millisecond, domain.Black)

	// Past the first deadline, well short of the second.
	time.Sleep(150 * time.Millisecond)
	if n := exp.count(); n != 0 {
		t.Fatalf("the replaced timer still fired (%d calls)", n)
	}

	select {
	case got := <-exp.fired:
		if got.color != domain.Black {
			t.Errorf("fired for %q, want black from the second Schedule", got.color)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the replacement timer never fired")
	}
}

func TestTimerManagerCancelUnknownGameIsANoop(t *testing.T) {
	tm := NewTimerManager(newFakeExpirer())
	tm.Cancel("never-scheduled") // must not panic
}

func TestTimerManagerTracksGamesIndependently(t *testing.T) {
	exp := newFakeExpirer()
	tm := NewTimerManager(exp)

	tm.Schedule("game-1", 10*time.Millisecond, domain.White)
	tm.Schedule("game-2", 10*time.Millisecond, domain.Black)

	seen := map[string]domain.Color{}
	for i := 0; i < 2; i++ {
		select {
		case got := <-exp.fired:
			seen[got.gameID] = got.color
		case <-time.After(2 * time.Second):
			t.Fatalf("only %d of 2 timers fired", i)
		}
	}

	if seen["game-1"] != domain.White || seen["game-2"] != domain.Black {
		t.Errorf("fired as %v, want game-1/white and game-2/black", seen)
	}
}

package matchmaking

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"api/internal/domain"
	"api/internal/game"
)

// --- fakes for the three seams MatchmakingService declares ---

type fakeGames struct {
	mu         sync.Mutex
	activeFor  map[string]bool
	activeErr  error
	created    []game.CreateGameInput
	createErr  error
	nextGameID string
}

func newFakeGames() *fakeGames {
	return &fakeGames{activeFor: map[string]bool{}, nextGameID: "game-1"}
}

func (f *fakeGames) HasActiveGame(_ context.Context, playerID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.activeErr != nil {
		return false, f.activeErr
	}
	return f.activeFor[playerID], nil
}

func (f *fakeGames) CreateGame(_ context.Context, in game.CreateGameInput) (*domain.Game, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.created = append(f.created, in)
	return &domain.Game{ID: f.nextGameID, WhiteID: in.WhiteID, BlackID: in.BlackID, TimeControl: in.TimeControl}, nil
}

func (f *fakeGames) createdGames() []game.CreateGameInput {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]game.CreateGameInput(nil), f.created...)
}

type fakeUsers struct{ ratings map[string]int32 }

func (f *fakeUsers) FindUserById(_ context.Context, id string) (*domain.User, error) {
	r, ok := f.ratings[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return &domain.User{UserID: id, Username: id, Rating: r}, nil
}

type notification struct {
	playerID string
	gameID   string
	opponent string
}

type fakeNotifier struct {
	mu   sync.Mutex
	sent []notification
}

func (f *fakeNotifier) NotifyMatched(_ context.Context, playerID string, g *domain.Game, opponent *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, notification{playerID: playerID, gameID: g.ID, opponent: opponent.Username})
	return nil
}

func (f *fakeNotifier) all() []notification {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]notification(nil), f.sent...)
}

// fixedClock makes bracket widening testable without sleeping.
type fixedClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fixedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fixedClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newHarness(ratings map[string]int32) (*MatchmakingService, *fakeGames, *fakeNotifier, *fixedClock) {
	games := newFakeGames()
	notifier := &fakeNotifier{}
	clock := &fixedClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	s := NewMatchmakingService(games, &fakeUsers{ratings: ratings}, notifier)
	s.clock = clock
	return s, games, notifier, clock
}

// --- JoinQueue guards ---

func TestJoinQueueRejectsUnknownPreset(t *testing.T) {
	s, games, _, _ := newHarness(map[string]int32{"alice": 1200})

	err := s.JoinQueue(context.Background(), "alice", "nonsense")

	if !errors.Is(err, domain.ErrUnknownPreset) {
		t.Errorf("got %v, want ErrUnknownPreset", err)
	}
	if len(games.createdGames()) != 0 {
		t.Error("a game was created for an unknown preset")
	}
}

func TestJoinQueueRejectsPlayerAlreadyInGame(t *testing.T) {
	s, _, _, _ := newHarness(map[string]int32{"alice": 1200})
	s.games.(*fakeGames).activeFor["alice"] = true

	err := s.JoinQueue(context.Background(), "alice", Preset3_0)

	if !errors.Is(err, domain.ErrAlreadyInGame) {
		t.Errorf("got %v, want ErrAlreadyInGame", err)
	}
	if len(s.queue.Entries(Preset3_0)) != 0 {
		t.Error("player was queued despite having an active game")
	}
}

func TestJoinQueuePropagatesLookupFailure(t *testing.T) {
	s, _, _, _ := newHarness(map[string]int32{}) // alice unknown

	if err := s.JoinQueue(context.Background(), "alice", Preset3_0); err == nil {
		t.Error("expected an error when the player can't be looked up")
	}
}

// --- pairing ---

func TestJoinQueuePairsWaitingOpponentImmediately(t *testing.T) {
	s, games, _, _ := newHarness(map[string]int32{"alice": 1200, "bob": 1220})
	ctx := context.Background()

	if err := s.JoinQueue(ctx, "alice", Preset3_0); err != nil {
		t.Fatalf("alice join: %v", err)
	}
	// No opponent yet, so alice waits.
	if len(games.createdGames()) != 0 {
		t.Fatal("a game was created with only one player queued")
	}

	if err := s.JoinQueue(ctx, "bob", Preset3_0); err != nil {
		t.Fatalf("bob join: %v", err)
	}

	created := games.createdGames()
	if len(created) != 1 {
		t.Fatalf("created %d games, want 1 - a compatible opponent should not wait for the sweep", len(created))
	}
	if created[0].TimeControl.Initial != 3*time.Minute {
		t.Errorf("game created with initial %v, want 3m from the preset", created[0].TimeControl.Initial)
	}
	if len(s.queue.Entries(Preset3_0)) != 0 {
		t.Error("matched players are still queued")
	}
}

// Each player must be told who they are playing, not who they are.
func TestPairingNotifiesBothPlayersWithTheOpponent(t *testing.T) {
	s, _, notifier, _ := newHarness(map[string]int32{"alice": 1200, "bob": 1220})
	ctx := context.Background()
	s.JoinQueue(ctx, "alice", Preset3_0)
	s.JoinQueue(ctx, "bob", Preset3_0)

	sent := notifier.all()
	if len(sent) != 2 {
		t.Fatalf("sent %d notifications, want 2", len(sent))
	}

	byPlayer := map[string]notification{}
	for _, n := range sent {
		byPlayer[n.playerID] = n
	}
	if byPlayer["alice"].opponent != "bob" {
		t.Errorf("alice was told her opponent is %q, want bob", byPlayer["alice"].opponent)
	}
	if byPlayer["bob"].opponent != "alice" {
		t.Errorf("bob was told his opponent is %q, want alice", byPlayer["bob"].opponent)
	}
	if byPlayer["alice"].gameID != byPlayer["bob"].gameID {
		t.Error("the two players were given different game ids")
	}
}

func TestNoPairingWhenRatingsTooFarApart(t *testing.T) {
	s, games, _, _ := newHarness(map[string]int32{"alice": 1200, "bob": 2000})
	ctx := context.Background()
	s.JoinQueue(ctx, "alice", Preset3_0)
	s.JoinQueue(ctx, "bob", Preset3_0)

	if len(games.createdGames()) != 0 {
		t.Error("paired players 800 apart on fresh brackets")
	}
	if len(s.queue.Entries(Preset3_0)) != 2 {
		t.Error("both players should still be waiting")
	}
}

func TestNoPairingAcrossDifferentPresets(t *testing.T) {
	s, games, _, _ := newHarness(map[string]int32{"alice": 1200, "bob": 1200})
	ctx := context.Background()
	s.JoinQueue(ctx, "alice", Preset3_0)
	s.JoinQueue(ctx, "bob", Preset10_0)

	if len(games.createdGames()) != 0 {
		t.Error("paired players queued for different time controls")
	}
}

// Eligibility is rechecked at pairing time, not just at join: time passes
// between the two.
func TestPairingSkipsPlayerWhoGainedAGameWhileWaiting(t *testing.T) {
	s, games, _, _ := newHarness(map[string]int32{"alice": 1200, "bob": 1200})
	ctx := context.Background()
	s.JoinQueue(ctx, "alice", Preset3_0)

	s.games.(*fakeGames).mu.Lock()
	s.games.(*fakeGames).activeFor["alice"] = true
	s.games.(*fakeGames).mu.Unlock()

	s.JoinQueue(ctx, "bob", Preset3_0)

	if len(games.createdGames()) != 0 {
		t.Error("paired a player who had acquired a game since joining")
	}
	if len(s.queue.Entries(Preset3_0)) != 1 {
		t.Errorf("queue holds %d, want 1 - alice should have been dropped, bob kept", len(s.queue.Entries(Preset3_0)))
	}
}

// --- leaving ---

func TestLeaveQueue(t *testing.T) {
	s, _, _, _ := newHarness(map[string]int32{"alice": 1200})
	s.JoinQueue(context.Background(), "alice", Preset3_0)

	s.LeaveQueue("alice", Preset3_0)

	if len(s.queue.Entries(Preset3_0)) != 0 {
		t.Error("player still queued after LeaveQueue")
	}
}

func TestHandleDisconnectClearsEveryPreset(t *testing.T) {
	s, _, _, _ := newHarness(map[string]int32{"alice": 1200})
	ctx := context.Background()
	s.JoinQueue(ctx, "alice", Preset3_0)
	s.JoinQueue(ctx, "alice", Preset10_0)

	s.HandleDisconnect("alice")

	for _, p := range []TimeControlPreset{Preset3_0, Preset10_0} {
		if n := len(s.queue.Entries(p)); n != 0 {
			t.Errorf("preset %q still holds %d entries after disconnect", p, n)
		}
	}
}

// --- sweep loop ---

// The reason the sweep exists: a pairing can become legal purely because
// time passed, with no join event to trigger a recheck.
func TestSweepLoopPairsOnceBracketsWiden(t *testing.T) {
	s, games, _, clock := newHarness(map[string]int32{"alice": 1200, "bob": 1450})
	ctx := context.Background()

	s.JoinQueue(ctx, "alice", Preset3_0)
	s.JoinQueue(ctx, "bob", Preset3_0)
	if len(games.createdGames()) != 0 {
		t.Fatal("250 apart should not pair on fresh brackets")
	}

	// Both have now waited long enough for +/-200 each, which covers 250.
	clock.advance(3 * widenInterval)

	loopCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go s.RunSweepLoop(loopCtx, time.Millisecond)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(games.createdGames()) == 1 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Error("sweep loop never paired the two players after their brackets widened")
}

func TestSweepLoopStopsOnContextCancel(t *testing.T) {
	s, _, _, _ := newHarness(map[string]int32{})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.RunSweepLoop(ctx, time.Millisecond)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("RunSweepLoop did not return after its context was cancelled")
	}
}

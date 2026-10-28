package invite

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"api/internal/domain"
	"api/internal/game"
	"api/internal/matchmaking"
)

var ctx = context.Background()

// --- doubles ---

type fakeGames struct {
	mu        sync.Mutex
	waiting   map[string]game.CreateGameInput
	started   []string
	discarded []string
	active    map[string]bool // playerID -> in an active game
	nextID    int
	createErr error
}

func (f *fakeGames) HasActiveGame(_ context.Context, playerID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.active[playerID], nil
}

func (f *fakeGames) CreateWaitingGame(_ context.Context, in game.CreateGameInput) (*domain.Game, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.nextID++
	id := fmt.Sprintf("game-%d", f.nextID)
	f.waiting[id] = in
	return &domain.Game{ID: id, WhiteID: in.WhiteID, BlackID: in.BlackID, Status: domain.GameStatusWaiting, TimeControl: in.TimeControl}, nil
}

func (f *fakeGames) StartWaitingGame(_ context.Context, gameID string) (*domain.Game, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	in, ok := f.waiting[gameID]
	if !ok {
		return nil, domain.ErrGameNotFound
	}
	delete(f.waiting, gameID)
	f.started = append(f.started, gameID)
	return &domain.Game{ID: gameID, WhiteID: in.WhiteID, BlackID: in.BlackID, Status: domain.GameStatusActive}, nil
}

func (f *fakeGames) DiscardWaitingGame(_ context.Context, gameID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.waiting[gameID]; !ok {
		return domain.ErrGameNotFound
	}
	delete(f.waiting, gameID)
	f.discarded = append(f.discarded, gameID)
	return nil
}

type fakeUsers map[string]*domain.User

func (u fakeUsers) FindUserById(_ context.Context, id string) (*domain.User, error) {
	if user, ok := u[id]; ok {
		return user, nil
	}
	return nil, domain.ErrUserNotFound
}

type fakeBlocks map[[2]string]bool

func (b fakeBlocks) IsBlocked(_ context.Context, x, y string) (bool, error) {
	return b[[2]string{x, y}] || b[[2]string{y, x}], nil
}

type fakePresence map[string]bool

func (p fakePresence) IsConnected(id string) bool { return p[id] }

type event struct {
	kind, to, gameID string
	reason           CloseReason
	opponent         string
}

type spyNotifier struct {
	mu     sync.Mutex
	events []event
}

func (n *spyNotifier) add(e event) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.events = append(n.events, e)
}

func (n *spyNotifier) NotifyInviteSent(_ context.Context, inv *Invite) error {
	n.add(event{kind: "sent", to: inv.Inviter.UserID, gameID: inv.GameID})
	return nil
}

func (n *spyNotifier) NotifyInviteReceived(_ context.Context, inv *Invite) error {
	n.add(event{kind: "received", to: inv.Invitee.UserID, gameID: inv.GameID})
	return nil
}

func (n *spyNotifier) NotifyInviteAccepted(_ context.Context, playerID string, g *domain.Game, opponent *domain.User) error {
	n.add(event{kind: "accepted", to: playerID, gameID: g.ID, opponent: opponent.UserID})
	return nil
}

func (n *spyNotifier) NotifyInviteClosed(_ context.Context, playerID, gameID string, reason CloseReason) error {
	n.add(event{kind: "closed", to: playerID, gameID: gameID, reason: reason})
	return nil
}

func (n *spyNotifier) take() []event {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := n.events
	n.events = nil
	return out
}

// manualTimer captures the expiry callback so tests can fire it on demand.
type manualTimer struct {
	d       time.Duration
	fire    func()
	stopped bool
}

func (t *manualTimer) Stop() bool {
	wasActive := !t.stopped
	t.stopped = true
	return wasActive
}

type harness struct {
	svc            *InviteService
	games          *fakeGames
	blocks         fakeBlocks
	presence       fakePresence
	notify         *spyNotifier
	timers         []*manualTimer
	whiteIsInviter bool
}

func newHarness() *harness {
	users := fakeUsers{}
	for _, name := range []string{"alice", "bob", "carol"} {
		users[name] = &domain.User{UserID: name, Username: name, Rating: 1200}
	}
	h := &harness{
		games:          &fakeGames{waiting: map[string]game.CreateGameInput{}, active: map[string]bool{}},
		blocks:         fakeBlocks{},
		presence:       fakePresence{"alice": true, "bob": true, "carol": true},
		notify:         &spyNotifier{},
		whiteIsInviter: true,
	}
	h.svc = NewInviteService(h.games, users, h.blocks, h.presence, h.notify)
	h.svc.now = func() time.Time { return time.Unix(1_700_000_000, 0).UTC() }
	h.svc.coinFlip = func() bool { return h.whiteIsInviter }
	h.svc.afterFunc = func(d time.Duration, f func()) stopper {
		t := &manualTimer{d: d, fire: f}
		h.timers = append(h.timers, t)
		return t
	}
	return h
}

func (h *harness) offer(t *testing.T, from, to string) *Invite {
	t.Helper()
	inv, err := h.svc.Offer(ctx, from, to, matchmaking.Preset3_2)
	if err != nil {
		t.Fatalf("Offer(%s -> %s): %v", from, to, err)
	}
	h.notify.take()
	return inv
}

// --- Offer ---

func TestOfferCreatesWaitingGameAndNotifiesBoth(t *testing.T) {
	h := newHarness()

	inv, err := h.svc.Offer(ctx, "alice", "bob", matchmaking.Preset3_2)
	if err != nil {
		t.Fatalf("Offer: %v", err)
	}

	in, ok := h.games.waiting[inv.GameID]
	if !ok {
		t.Fatal("no waiting game was created")
	}
	if in.TimeControl != (domain.TimeControl{Initial: 3 * time.Minute, Increment: 2 * time.Second}) {
		t.Errorf("time control = %+v, want 3+2", in.TimeControl)
	}
	if want := h.svc.now().Add(TTL); !inv.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", inv.ExpiresAt, want)
	}
	if len(h.timers) != 1 || h.timers[0].d != TTL {
		t.Errorf("expiry timers = %+v, want one at %v", h.timers, TTL)
	}

	got := h.notify.take()
	want := []event{
		{kind: "sent", to: "alice", gameID: inv.GameID},
		{kind: "received", to: "bob", gameID: inv.GameID},
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("events = %v, want %v", got, want)
	}
}

func TestOfferAssignsColorsFromCoinFlip(t *testing.T) {
	for _, inviterWhite := range []bool{true, false} {
		h := newHarness()
		h.whiteIsInviter = inviterWhite

		inv := h.offer(t, "alice", "bob")

		in := h.games.waiting[inv.GameID]
		wantWhite, wantBlack := "alice", "bob"
		if !inviterWhite {
			wantWhite, wantBlack = "bob", "alice"
		}
		if in.WhiteID != wantWhite || in.BlackID != wantBlack {
			t.Errorf("coin=%v: white/black = %s/%s, want %s/%s", inviterWhite, in.WhiteID, in.BlackID, wantWhite, wantBlack)
		}
	}
}

func TestOfferRejections(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(*harness)
		to     string
		preset matchmaking.TimeControlPreset
		want   error
	}{
		{"unknown preset", nil, "bob", "9+9", domain.ErrUnknownPreset},
		{"self", nil, "alice", matchmaking.Preset3_2, domain.ErrCannotInviteSelf},
		{"unknown user", nil, "nobody", matchmaking.Preset3_2, domain.ErrUserNotFound},
		// The silent block, from either side, looks like an unknown user.
		{"invitee blocked inviter", func(h *harness) { h.blocks[[2]string{"bob", "alice"}] = true }, "bob", matchmaking.Preset3_2, domain.ErrUserNotFound},
		{"inviter blocked invitee", func(h *harness) { h.blocks[[2]string{"alice", "bob"}] = true }, "bob", matchmaking.Preset3_2, domain.ErrUserNotFound},
		{"invitee offline", func(h *harness) { h.presence["bob"] = false }, "bob", matchmaking.Preset3_2, domain.ErrUserNotConnected},
		{"inviter in a game", func(h *harness) { h.games.active["alice"] = true }, "bob", matchmaking.Preset3_2, domain.ErrAlreadyInGame},
		{"invitee in a game", func(h *harness) { h.games.active["bob"] = true }, "bob", matchmaking.Preset3_2, domain.ErrAlreadyInGame},
		{"already has an outgoing invite", func(h *harness) { h.svc.Offer(ctx, "alice", "carol", matchmaking.Preset3_2) }, "bob", matchmaking.Preset3_2, domain.ErrInviteAlreadyPending},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness()
			if c.setup != nil {
				c.setup(h)
			}
			before := len(h.games.waiting)
			h.notify.take()

			if _, err := h.svc.Offer(ctx, "alice", c.to, c.preset); !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
			if len(h.games.waiting) != before {
				t.Error("a waiting game was created for a rejected offer")
			}
			if ev := h.notify.take(); len(ev) != 0 {
				t.Errorf("rejected offer sent notifications: %v", ev)
			}
		})
	}
}

// The client sees err.Error(), so a block and an unknown user must not
// differ even in their message text - or the silent block isn't silent.
// Real lookups wrap ErrUserNotFound ("finding user by ID: ..."), which is
// how this once leaked.
func TestUnknownUserAndBlockAreIndistinguishable(t *testing.T) {
	h := newHarness()
	h.svc.users = wrappingUsers{h.svc.users}
	h.blocks[[2]string{"bob", "alice"}] = true

	_, blockedErr := h.svc.Offer(ctx, "alice", "bob", matchmaking.Preset3_2)
	_, unknownErr := h.svc.Offer(ctx, "alice", "nobody", matchmaking.Preset3_2)

	if blockedErr == nil || unknownErr == nil || blockedErr.Error() != unknownErr.Error() {
		t.Errorf("blocked: %v; unknown: %v - want identical", blockedErr, unknownErr)
	}
}

// wrappingUsers wraps lookup errors the way user.UserService does.
type wrappingUsers struct{ inner userLookup }

func (w wrappingUsers) FindUserById(ctx context.Context, id string) (*domain.User, error) {
	u, err := w.inner.FindUserById(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("finding user by ID: %w", err)
	}
	return u, nil
}

// A failed offer must release the inviter's slot, or they could never
// invite anyone again.
func TestFailedOfferFreesTheOutgoingSlot(t *testing.T) {
	h := newHarness()
	h.games.createErr = errors.New("db down")

	if _, err := h.svc.Offer(ctx, "alice", "bob", matchmaking.Preset3_2); err == nil {
		t.Fatal("expected the create failure to surface")
	}

	h.games.createErr = nil
	h.offer(t, "alice", "bob")
}

func TestInviteeMayHaveSeveralIncoming(t *testing.T) {
	h := newHarness()
	h.offer(t, "alice", "carol")
	h.offer(t, "bob", "carol")
}

// --- Accept ---

func TestAcceptStartsGameAndNotifiesBoth(t *testing.T) {
	h := newHarness()
	inv := h.offer(t, "alice", "bob")

	g, err := h.svc.Accept(ctx, "bob", inv.GameID)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}

	if g.Status != domain.GameStatusActive || len(h.games.started) != 1 {
		t.Errorf("game not started: status %q, started %v", g.Status, h.games.started)
	}
	if !h.timers[0].stopped {
		t.Error("the expiry timer is still armed after accept")
	}
	got := h.notify.take()
	want := []event{
		{kind: "accepted", to: "alice", gameID: inv.GameID, opponent: "bob"},
		{kind: "accepted", to: "bob", gameID: inv.GameID, opponent: "alice"},
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("events = %v, want %v", got, want)
	}

	// The slot is free again, and the invite is gone.
	h.offer(t, "alice", "carol")
	if _, err := h.svc.Accept(ctx, "bob", inv.GameID); !errors.Is(err, domain.ErrInviteNotFound) {
		t.Errorf("second accept: err = %v, want ErrInviteNotFound", err)
	}
}

func TestOnlyTheInviteeCanAccept(t *testing.T) {
	h := newHarness()
	inv := h.offer(t, "alice", "bob")

	for _, who := range []string{"alice", "carol"} {
		if _, err := h.svc.Accept(ctx, who, inv.GameID); !errors.Is(err, domain.ErrInviteNotFound) {
			t.Errorf("%s accepting: err = %v, want ErrInviteNotFound", who, err)
		}
	}
	// The rejected attempts must not have consumed the invite.
	if _, err := h.svc.Accept(ctx, "bob", inv.GameID); err != nil {
		t.Errorf("bob accepting after the failed attempts: %v", err)
	}
}

// Either player may have been matched into a game while the invite sat.
func TestAcceptWhenSomeoneIsNowInAGame(t *testing.T) {
	h := newHarness()
	inv := h.offer(t, "alice", "bob")
	h.games.active["alice"] = true

	if _, err := h.svc.Accept(ctx, "bob", inv.GameID); !errors.Is(err, domain.ErrAlreadyInGame) {
		t.Fatalf("err = %v, want ErrAlreadyInGame", err)
	}
	if len(h.games.started) != 0 {
		t.Error("the invite game was started anyway")
	}
	if len(h.games.discarded) != 1 {
		t.Error("the waiting game was not discarded")
	}
	assertClosedForBoth(t, h, inv.GameID, ReasonUnavailable)
}

// --- Decline / Cancel / expiry / disconnect ---

func TestDeclineAndCancel(t *testing.T) {
	cases := []struct {
		name   string
		act    func(*InviteService, string) error
		reason CloseReason
	}{
		{"invitee declines", func(s *InviteService, id string) error { return s.Decline(ctx, "bob", id) }, ReasonDeclined},
		{"inviter cancels", func(s *InviteService, id string) error { return s.Cancel(ctx, "alice", id) }, ReasonCancelled},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness()
			inv := h.offer(t, "alice", "bob")

			if err := c.act(h.svc, inv.GameID); err != nil {
				t.Fatalf("err = %v", err)
			}
			if len(h.games.discarded) != 1 || !h.timers[0].stopped {
				t.Errorf("discarded %v, timer stopped %v; want the game discarded and timer stopped", h.games.discarded, h.timers[0].stopped)
			}
			assertClosedForBoth(t, h, inv.GameID, c.reason)
			h.offer(t, "alice", "bob") // slot freed
		})
	}
}

func TestWrongPartyCannotDeclineOrCancel(t *testing.T) {
	h := newHarness()
	inv := h.offer(t, "alice", "bob")

	if err := h.svc.Decline(ctx, "alice", inv.GameID); !errors.Is(err, domain.ErrInviteNotFound) {
		t.Errorf("inviter declining: err = %v, want ErrInviteNotFound", err)
	}
	if err := h.svc.Cancel(ctx, "bob", inv.GameID); !errors.Is(err, domain.ErrInviteNotFound) {
		t.Errorf("invitee cancelling: err = %v, want ErrInviteNotFound", err)
	}
	if err := h.svc.Decline(ctx, "carol", inv.GameID); !errors.Is(err, domain.ErrInviteNotFound) {
		t.Errorf("bystander declining: err = %v, want ErrInviteNotFound", err)
	}
	if len(h.games.discarded) != 0 {
		t.Error("a rejected decline/cancel discarded the game")
	}
}

func TestInviteExpires(t *testing.T) {
	h := newHarness()
	inv := h.offer(t, "alice", "bob")

	h.timers[0].fire()

	assertClosedForBoth(t, h, inv.GameID, ReasonExpired)
	if _, err := h.svc.Accept(ctx, "bob", inv.GameID); !errors.Is(err, domain.ErrInviteNotFound) {
		t.Errorf("accepting an expired invite: err = %v, want ErrInviteNotFound", err)
	}
	h.offer(t, "alice", "bob") // slot freed
}

// A timer that fires after the invite was already accepted must be a no-op.
func TestLateExpiryAfterAcceptDoesNothing(t *testing.T) {
	h := newHarness()
	inv := h.offer(t, "alice", "bob")
	h.svc.Accept(ctx, "bob", inv.GameID)
	h.notify.take()

	h.timers[0].fire()

	if ev := h.notify.take(); len(ev) != 0 {
		t.Errorf("late expiry sent %v", ev)
	}
	if len(h.games.discarded) != 0 {
		t.Error("late expiry discarded the started game")
	}
}

func TestDisconnectClosesEveryInviteTheUserIsIn(t *testing.T) {
	h := newHarness()
	sent := h.offer(t, "carol", "alice")   // carol is inviter
	received := h.offer(t, "bob", "carol") // carol is invitee
	untouched := h.offer(t, "alice", "bob")

	h.svc.HandleDisconnect("carol")

	closed := map[string]CloseReason{}
	for _, e := range h.notify.take() {
		if e.kind == "closed" {
			closed[e.gameID] = e.reason
		}
	}
	for _, id := range []string{sent.GameID, received.GameID} {
		if closed[id] != ReasonUnavailable {
			t.Errorf("invite %s: closed reason %q, want unavailable", id, closed[id])
		}
	}
	if _, ok := closed[untouched.GameID]; ok {
		t.Error("an invite carol wasn't part of was closed")
	}
	if _, err := h.svc.Accept(ctx, "bob", untouched.GameID); err != nil {
		t.Errorf("unrelated invite no longer acceptable: %v", err)
	}
}

func assertClosedForBoth(t *testing.T, h *harness, gameID string, reason CloseReason) {
	t.Helper()
	got := h.notify.take()
	want := []event{
		{kind: "closed", to: "alice", gameID: gameID, reason: reason},
		{kind: "closed", to: "bob", gameID: gameID, reason: reason},
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("events = %v, want %v", got, want)
	}
}

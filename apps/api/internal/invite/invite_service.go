package invite

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"api/internal/domain"
	"api/internal/game"
	"api/internal/matchmaking"
)

// TTL is how long an unanswered invite lives before it expires.
const TTL = 60 * time.Second

// gameService, userLookup, blockChecker and presence are the narrow slices
// of other packages this service calls - same pattern as matchmaking.
type gameService interface {
	HasActiveGame(ctx context.Context, playerID string) (bool, error)
	CreateWaitingGame(ctx context.Context, in game.CreateGameInput) (*domain.Game, error)
	StartWaitingGame(ctx context.Context, gameID string) (*domain.Game, error)
	DiscardWaitingGame(ctx context.Context, gameID string) error
}

type userLookup interface {
	FindUserById(ctx context.Context, id string) (*domain.User, error)
}

type blockChecker interface {
	IsBlocked(ctx context.Context, a, b string) (bool, error)
}

// presence is satisfied by ws.Hub. Invites are online-only: offering to
// someone with no open connection fails immediately.
type presence interface {
	IsConnected(userID string) bool
}

type CloseReason string

const (
	ReasonDeclined    CloseReason = "declined"
	ReasonCancelled   CloseReason = "cancelled"
	ReasonExpired     CloseReason = "expired"
	ReasonUnavailable CloseReason = "unavailable"
)

// Invite is a pending challenge. GameID is the waiting game's ID, which
// doubles as the invite's ID on the wire.
type Invite struct {
	GameID    string
	Inviter   *domain.User
	Invitee   *domain.User
	Preset    matchmaking.TimeControlPreset
	ExpiresAt time.Time
}

// Notifier is the seam to the WebSocket layer, implemented by
// ws.InviteNotifier. InviteService has no idea WebSockets exist.
type Notifier interface {
	NotifyInviteSent(ctx context.Context, inv *Invite) error
	NotifyInviteReceived(ctx context.Context, inv *Invite) error
	NotifyInviteAccepted(ctx context.Context, playerID string, g *domain.Game, opponent *domain.User) error
	NotifyInviteClosed(ctx context.Context, playerID, gameID string, reason CloseReason) error
}

type stopper interface{ Stop() bool }

type pending struct {
	invite *Invite
	timer  stopper
}

// InviteService runs the offer/accept/decline/cancel flow for challenging
// a specific player. Invites live in memory, like the matchmaking queue
// and game timers: a restart loses them, which is why main.go discards
// every waiting game on startup.
//
// Every path that ends an invite first removes it from the map under the
// lock, and only then does I/O. Whichever of accept, decline, cancel,
// expiry or disconnect claims it first wins; the rest find nothing.
type InviteService struct {
	games    gameService
	users    userLookup
	blocks   blockChecker
	presence presence
	notify   Notifier

	mu        sync.Mutex
	byGame    map[string]*pending
	byInviter map[string]string // inviterID -> gameID; "" while an offer is being created

	now       func() time.Time
	afterFunc func(time.Duration, func()) stopper
	coinFlip  func() bool // true: inviter plays white
}

func NewInviteService(games gameService, users userLookup, blocks blockChecker, presence presence, notify Notifier) *InviteService {
	return &InviteService{
		games:     games,
		users:     users,
		blocks:    blocks,
		presence:  presence,
		notify:    notify,
		byGame:    map[string]*pending{},
		byInviter: map[string]string{},
		now:       func() time.Time { return time.Now().UTC() },
		afterFunc: func(d time.Duration, f func()) stopper { return time.AfterFunc(d, f) },
		coinFlip:  func() bool { return rand.IntN(2) == 0 },
	}
}

// Offer challenges toUserID to a game at preset.
//
// A block in either direction answers ErrUserNotFound, exactly like a user
// that doesn't exist - the same silent block the friends feature uses.
func (s *InviteService) Offer(ctx context.Context, meID, toUserID string, preset matchmaking.TimeControlPreset) (*Invite, error) {
	tc, err := matchmaking.TimeControlForPreset(preset)
	if err != nil {
		return nil, err
	}
	if meID == toUserID {
		return nil, domain.ErrCannotInviteSelf
	}

	blocked, err := s.blocks.IsBlocked(ctx, meID, toUserID)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("checking blocks: %w", err)
	}
	if blocked || err != nil {
		return nil, domain.ErrUserNotFound
	}
	invitee, err := s.users.FindUserById(ctx, toUserID)
	if err != nil {
		return nil, err
	}
	if !s.presence.IsConnected(toUserID) {
		return nil, domain.ErrUserNotConnected
	}
	inviter, err := s.users.FindUserById(ctx, meID)
	if err != nil {
		return nil, fmt.Errorf("looking up inviter: %w", err)
	}

	// Reserve the inviter's one outgoing slot before any slow I/O, so two
	// quick offers can't both get through.
	s.mu.Lock()
	if _, busy := s.byInviter[meID]; busy {
		s.mu.Unlock()
		return nil, domain.ErrInviteAlreadyPending
	}
	s.byInviter[meID] = ""
	s.mu.Unlock()

	inv, err := s.createInvite(ctx, inviter, invitee, preset, tc)
	if err != nil {
		s.mu.Lock()
		delete(s.byInviter, meID)
		s.mu.Unlock()
		return nil, err
	}

	if err := s.notify.NotifyInviteSent(ctx, inv); err != nil {
		slog.Error("notifying inviter", "game", inv.GameID, "error", err)
	}
	if err := s.notify.NotifyInviteReceived(ctx, inv); err != nil {
		slog.Error("notifying invitee", "game", inv.GameID, "error", err)
	}
	return inv, nil
}

func (s *InviteService) createInvite(ctx context.Context, inviter, invitee *domain.User, preset matchmaking.TimeControlPreset, tc domain.TimeControl) (*Invite, error) {
	for _, id := range []string{inviter.UserID, invitee.UserID} {
		busy, err := s.games.HasActiveGame(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("checking active game: %w", err)
		}
		if busy {
			return nil, domain.ErrAlreadyInGame
		}
	}

	whiteID, blackID := inviter.UserID, invitee.UserID
	if !s.coinFlip() {
		whiteID, blackID = blackID, whiteID
	}
	g, err := s.games.CreateWaitingGame(ctx, game.CreateGameInput{WhiteID: whiteID, BlackID: blackID, TimeControl: tc})
	if err != nil {
		return nil, fmt.Errorf("creating waiting game: %w", err)
	}

	inv := &Invite{GameID: g.ID, Inviter: inviter, Invitee: invitee, Preset: preset, ExpiresAt: s.now().Add(TTL)}

	s.mu.Lock()
	s.byGame[g.ID] = &pending{
		invite: inv,
		timer:  s.afterFunc(TTL, func() { s.expire(g.ID) }),
	}
	s.byInviter[inviter.UserID] = g.ID
	s.mu.Unlock()
	return inv, nil
}

// Accept starts the game. Only the invitee can accept; for anyone else the
// invite doesn't exist.
func (s *InviteService) Accept(ctx context.Context, meID, gameID string) (*domain.Game, error) {
	inv, ok := s.claim(gameID, func(inv *Invite) bool { return inv.Invitee.UserID == meID })
	if !ok {
		return nil, domain.ErrInviteNotFound
	}

	// Either player may have been matched into a game since the offer.
	for _, id := range []string{inv.Inviter.UserID, inv.Invitee.UserID} {
		busy, err := s.games.HasActiveGame(ctx, id)
		if err == nil && !busy {
			continue
		}
		s.finish(ctx, inv, ReasonUnavailable)
		if err != nil {
			return nil, fmt.Errorf("checking active game: %w", err)
		}
		return nil, domain.ErrAlreadyInGame
	}

	g, err := s.games.StartWaitingGame(ctx, gameID)
	if err != nil {
		s.finish(ctx, inv, ReasonUnavailable)
		return nil, fmt.Errorf("starting invite game: %w", err)
	}

	if err := s.notify.NotifyInviteAccepted(ctx, inv.Inviter.UserID, g, inv.Invitee); err != nil {
		slog.Error("notifying inviter of accept", "game", gameID, "error", err)
	}
	if err := s.notify.NotifyInviteAccepted(ctx, inv.Invitee.UserID, g, inv.Inviter); err != nil {
		slog.Error("notifying invitee of accept", "game", gameID, "error", err)
	}
	return g, nil
}

// Decline is the invitee turning the invite down.
func (s *InviteService) Decline(ctx context.Context, meID, gameID string) error {
	return s.end(ctx, gameID, ReasonDeclined, func(inv *Invite) bool { return inv.Invitee.UserID == meID })
}

// Cancel is the inviter withdrawing the invite.
func (s *InviteService) Cancel(ctx context.Context, meID, gameID string) error {
	return s.end(ctx, gameID, ReasonCancelled, func(inv *Invite) bool { return inv.Inviter.UserID == meID })
}

// HandleDisconnect closes every invite the user is part of: invites are
// online-only, so one whose party has gone can't be answered.
func (s *InviteService) HandleDisconnect(userID string) {
	s.mu.Lock()
	var gameIDs []string
	for id, p := range s.byGame {
		if p.invite.Inviter.UserID == userID || p.invite.Invitee.UserID == userID {
			gameIDs = append(gameIDs, id)
		}
	}
	s.mu.Unlock()

	for _, id := range gameIDs {
		s.end(context.Background(), id, ReasonUnavailable, nil)
	}
}

func (s *InviteService) expire(gameID string) {
	s.end(context.Background(), gameID, ReasonExpired, nil)
}

func (s *InviteService) end(ctx context.Context, gameID string, reason CloseReason, allowed func(*Invite) bool) error {
	inv, ok := s.claim(gameID, allowed)
	if !ok {
		return domain.ErrInviteNotFound
	}
	s.finish(ctx, inv, reason)
	return nil
}

// claim removes the invite from memory if allowed (nil allows anyone),
// making the caller the only path that will finish it.
func (s *InviteService) claim(gameID string, allowed func(*Invite) bool) (*Invite, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.byGame[gameID]
	if !ok || (allowed != nil && !allowed(p.invite)) {
		return nil, false
	}
	p.timer.Stop()
	delete(s.byGame, gameID)
	delete(s.byInviter, p.invite.Inviter.UserID)
	return p.invite, true
}

// finish discards the waiting game and tells both players why it ended.
func (s *InviteService) finish(ctx context.Context, inv *Invite, reason CloseReason) {
	if err := s.games.DiscardWaitingGame(ctx, inv.GameID); err != nil && !errors.Is(err, domain.ErrGameNotFound) {
		slog.Error("discarding waiting game", "game", inv.GameID, "error", err)
	}
	for _, id := range []string{inv.Inviter.UserID, inv.Invitee.UserID} {
		if err := s.notify.NotifyInviteClosed(ctx, id, inv.GameID, reason); err != nil && !errors.Is(err, domain.ErrUserNotConnected) {
			slog.Error("notifying invite closed", "game", inv.GameID, "user", id, "error", err)
		}
	}
}

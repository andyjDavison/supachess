package matchmaking

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"api/internal/domain"
	"api/internal/game"
)

// gameService and userLookup are narrow slices of game.Service/whatever
// exposes GetUserByID — interface segregation, same pattern as the HTTP
// handlers depending on registerer/authenticator rather than concrete
// service types.
type gameService interface {
	HasActiveGame(ctx context.Context, playerID string) (bool, error)
	CreateGame(ctx context.Context, in game.CreateGameInput) (*domain.Game, error)
}

type userLookup interface {
	FindUserById(ctx context.Context, id string) (*domain.User, error)
}

// Notifier is the seam between Service and the WebSocket layer — same
// decoupling role game.GameEventPublisher plays for the game service.
// Service has no idea WebSockets exist.
type Notifier interface {
	NotifyMatched(ctx context.Context, playerID string, g *domain.Game, opponent *domain.User) error
}

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type MatchmakingService struct {
	queue  *Queue
	games  gameService
	users  userLookup
	notify Notifier
	clock  Clock
}

func NewMatchmakingService(games gameService, users userLookup, notify Notifier) *MatchmakingService {
	return &MatchmakingService{
		queue:  NewQueue(),
		games:  games,
		users:  users,
		notify: notify,
		clock:  realClock{},
	}
}

// JoinQueue enqueues playerID for preset. Confirms no active game first
// (the global, not per-preset, constraint), then runs an immediate match
// attempt before returning — a compatible opponent already waiting
// shouldn't require waiting for the next sweep tick.
func (s *MatchmakingService) JoinQueue(ctx context.Context, playerID string, preset TimeControlPreset) error {
	if _, err := timeControlForPreset(preset); err != nil {
		return err
	}

	hasGame, err := s.games.HasActiveGame(ctx, playerID)
	if err != nil {
		return fmt.Errorf("checking active game: %w", err)
	}
	if hasGame {
		return domain.ErrAlreadyInGame
	}

	user, err := s.users.FindUserById(ctx, playerID)
	if err != nil {
		return fmt.Errorf("looking up player: %w", err)
	}

	added := s.queue.Add(Entry{
		PlayerID: playerID,
		Rating:   int(user.Rating),
		Preset:   preset,
		JoinedAt: s.clock.Now(),
	})
	if !added {
		return nil // already queued for this preset
	}

	s.attemptMatch(ctx, preset)
	return nil
}

func (s *MatchmakingService) LeaveQueue(playerID string, preset TimeControlPreset) {
	s.queue.Remove(playerID, preset)
}

// HandleDisconnect removes playerID from every preset's queue — wired to
// fire from Hub's disconnect path (see hub.go changes below), since a
// dropped connection doesn't otherwise tell matchmaking anything.
func (s *MatchmakingService) HandleDisconnect(playerID string) {
	s.queue.RemoveFromAll(playerID)
}

// RunSweepLoop re-evaluates every non-empty preset queue on each tick —
// necessary because bracket widening can make a pairing valid purely
// from elapsed time, with no join event to trigger a recheck. Start once
// via `go matchmakingService.RunSweepLoop(ctx, interval)`; runs until
// ctx is cancelled.
func (s *MatchmakingService) RunSweepLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, preset := range s.queue.Presets() {
				s.attemptMatch(ctx, preset)
			}
		}
	}
}

func (s *MatchmakingService) attemptMatch(ctx context.Context, preset TimeControlPreset) {
	now := s.clock.Now()
	entries := s.queue.Entries(preset)

	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			a, b := entries[i], entries[j]
			if !compatible(a, b, now) {
				continue
			}

			// Re-check right before pairing, not just at join time —
			// time has passed since either entry joined.
			if !s.stillEligible(ctx, a.PlayerID) {
				s.queue.RemoveFromAll(a.PlayerID)
				continue
			}
			if !s.stillEligible(ctx, b.PlayerID) {
				s.queue.RemoveFromAll(b.PlayerID)
				continue
			}

			if !s.queue.Remove(a.PlayerID, preset) || !s.queue.Remove(b.PlayerID, preset) {
				continue // claimed by a concurrent attempt already
			}

			s.createMatch(ctx, preset, a, b)
			return // entries snapshot is stale now; next tick re-scans
		}
	}
}

func (s *MatchmakingService) stillEligible(ctx context.Context, playerID string) bool {
	hasGame, err := s.games.HasActiveGame(ctx, playerID)
	if err != nil {
		slog.Error("checking active game during match attempt", "player", playerID, "error", err)
		return false
	}
	return !hasGame
}

// createMatch assigns colors arbitrarily (a = white, b = black) — a
// known simplification, not fairness logic. Worth revisiting (e.g.
// alternating based on each player's recent color history) once this is
// otherwise working.
func (s *MatchmakingService) createMatch(ctx context.Context, preset TimeControlPreset, a, b Entry) {
	tc, err := timeControlForPreset(preset)
	if err != nil {
		slog.Error("unknown preset during match creation", "preset", preset, "error", err)
		return
	}

	g, err := s.games.CreateGame(ctx, game.CreateGameInput{
		WhiteID: a.PlayerID,
		BlackID: b.PlayerID,
		TimeControl:   tc,
	})
	if err != nil {
		slog.Error("creating matched game", "error", err)
		return
	}

	whiteUser, errA := s.users.FindUserById(ctx, a.PlayerID)
	blackUser, errB := s.users.FindUserById(ctx, b.PlayerID)
	if errA != nil || errB != nil {
		slog.Error("looking up matched players for notification", "errA", errA, "errB", errB)
		return
	}

	if err := s.notify.NotifyMatched(ctx, a.PlayerID, g, blackUser); err != nil {
		slog.Error("notifying player of match", "player", a.PlayerID, "error", err)
	}
	if err := s.notify.NotifyMatched(ctx, b.PlayerID, g, whiteUser); err != nil {
		slog.Error("notifying player of match", "player", b.PlayerID, "error", err)
	}
}
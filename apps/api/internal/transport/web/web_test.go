package web

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"api/internal/domain"
	"api/internal/game"
	"api/internal/user"
)

// Shared doubles for this package's handler tests. The handlers take
// concrete service types, so these build real services over fake
// repositories - the same approach internal/server/router_test.go uses.

const testPassword = "password123"

var errBoom = errorString("database is on fire")

type errorString string

func (e errorString) Error() string { return string(e) }

// quietLogs silences writeServiceError's slog output for the default branch.
func quietLogs(t *testing.T) {
	t.Helper()
	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(original) })
}

type fakeUserRepo struct {
	mu        sync.Mutex
	byID      map[string]*domain.User
	createErr error
	lookupErr error
	nextID    int
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byID: map[string]*domain.User{}}
}

// seed stores a user whose password hash really verifies, so login tests
// exercise bcrypt rather than a stub comparison.
func (r *fakeUserRepo) seed(t *testing.T, username, email string) *domain.User {
	t.Helper()
	hash, err := user.NewBcryptHasher(4).Hash(testPassword)
	if err != nil {
		t.Fatalf("hashing fixture password: %v", err)
	}
	u := &domain.User{
		UserID: "user-1", Username: username, Email: email,
		PasswordHash: hash, Rating: 800,
	}
	r.mu.Lock()
	r.byID[u.UserID] = u
	r.mu.Unlock()
	return u
}

func (r *fakeUserRepo) Create(_ context.Context, input *domain.User) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createErr != nil {
		return nil, r.createErr
	}
	r.nextID++
	stored := *input
	stored.UserID = "created-user"
	r.byID[stored.UserID] = &stored
	return &stored, nil
}

func (r *fakeUserRepo) find(match func(*domain.User) bool) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lookupErr != nil {
		return nil, r.lookupErr
	}
	for _, u := range r.byID {
		if match(u) {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (r *fakeUserRepo) FindUserById(_ context.Context, id string) (*domain.User, error) {
	return r.find(func(u *domain.User) bool { return u.UserID == id })
}

func (r *fakeUserRepo) FindUserByEmail(_ context.Context, email string) (*domain.User, error) {
	return r.find(func(u *domain.User) bool { return u.Email == email })
}

func (r *fakeUserRepo) FindUserByUsername(_ context.Context, username string) (*domain.User, error) {
	return r.find(func(u *domain.User) bool { return u.Username == username })
}

type fakeGameRepo struct {
	games   map[string]*domain.Game
	moves   map[string][]domain.Move
	findErr error
}

func newFakeGameRepo() *fakeGameRepo {
	return &fakeGameRepo{games: map[string]*domain.Game{}, moves: map[string][]domain.Move{}}
}

func (r *fakeGameRepo) Create(_ context.Context, g *domain.Game) (*domain.Game, error) {
	return g, nil
}

func (r *fakeGameRepo) FindByID(_ context.Context, id string) (*domain.Game, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	g, ok := r.games[id]
	if !ok {
		return nil, domain.ErrGameNotFound
	}
	return g, nil
}

func (r *fakeGameRepo) FindActiveGameByPlayerID(_ context.Context, _ string) (*domain.Game, error) {
	return nil, domain.ErrGameNotFound
}

func (r *fakeGameRepo) Update(_ context.Context, _ *domain.Game) error { return nil }

func (r *fakeGameRepo) DeleteWaitingGame(_ context.Context, _ string) error {
	return domain.ErrGameNotFound
}

func (r *fakeGameRepo) DeleteAllWaitingGames(_ context.Context) (int64, error) { return 0, nil }

func (r *fakeGameRepo) AppendMove(_ context.Context, _ string, _ *domain.Move) error { return nil }

func (r *fakeGameRepo) FindMovesByGameID(_ context.Context, gameID string) ([]domain.Move, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	return r.moves[gameID], nil
}

type noopValidator struct{}

func (noopValidator) ValidateMove([]domain.Move, string, domain.Square, domain.Square, domain.PieceType) (game.MoveResult, error) {
	return game.MoveResult{}, domain.ErrIllegalMove
}

type noopPublisher struct{}

func (noopPublisher) PublishMoveMade(context.Context, *domain.Game, *domain.Move) error { return nil }
func (noopPublisher) PublishGameOver(context.Context, *domain.Game) error               { return nil }

type noopScheduler struct{}

func (noopScheduler) Schedule(string, time.Duration, domain.Color) {}
func (noopScheduler) Cancel(string)                                {}

func newGameService(repo game.GameRepository) *game.GameService {
	return game.NewGameService(repo, noopValidator{}, noopPublisher{}, noopScheduler{})
}

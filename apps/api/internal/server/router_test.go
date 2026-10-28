package server

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"api/internal/auth"
	"api/internal/domain"
	"api/internal/friends"
	"api/internal/friends/friendstest"
	"api/internal/game"
	"api/internal/matchmaking"
	"api/internal/transport/web"
	"api/internal/transport/ws"
	"api/internal/user"
)

// Minimal doubles, just enough to construct the handler graph. The point
// of this file is routing and the auth boundary, not service behaviour -
// the services have their own tests.

type stubUserRepo struct{}

func (stubUserRepo) Create(context.Context, *domain.User) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}
func (stubUserRepo) FindUserById(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}
func (stubUserRepo) FindUserByEmail(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}
func (stubUserRepo) FindUserByUsername(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

type stubGameRepo struct{}

func (stubGameRepo) Create(context.Context, *domain.Game) (*domain.Game, error) {
	return nil, domain.ErrGameNotFound
}
func (stubGameRepo) FindByID(context.Context, string) (*domain.Game, error) {
	return nil, domain.ErrGameNotFound
}
func (stubGameRepo) FindActiveGameByPlayerID(context.Context, string) (*domain.Game, error) {
	return nil, domain.ErrGameNotFound
}
func (stubGameRepo) Update(context.Context, *domain.Game) error { return nil }
func (stubGameRepo) AppendMove(context.Context, string, *domain.Move) error {
	return nil
}
func (stubGameRepo) DeleteWaitingGame(context.Context, string) error {
	return domain.ErrGameNotFound
}
func (stubGameRepo) DeleteAllWaitingGames(context.Context) (int64, error) { return 0, nil }
func (stubGameRepo) FindMovesByGameID(context.Context, string) ([]domain.Move, error) {
	return nil, nil
}

type stubValidator struct{}

func (stubValidator) ValidateMove([]domain.Move, string, domain.Square, domain.Square, domain.PieceType) (game.MoveResult, error) {
	return game.MoveResult{}, domain.ErrIllegalMove
}

type stubPublisher struct{}

func (stubPublisher) PublishMoveMade(context.Context, *domain.Game, *domain.Move) error { return nil }
func (stubPublisher) PublishGameOver(context.Context, *domain.Game) error               { return nil }

type stubScheduler struct{}

func (stubScheduler) Schedule(string, time.Duration, domain.Color) {}
func (stubScheduler) Cancel(string)                                {}

type stubNotifier struct{}

func (stubNotifier) NotifyMatched(context.Context, string, *domain.Game, *domain.User) error {
	return nil
}

var testSecret = []byte("router-test-secret")

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	// The router's Logging middleware writes to the standard logger.
	original := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(original) })

	hasher := user.NewBcryptHasher(4)
	userService := user.NewUserService(stubUserRepo{}, hasher)
	authService := auth.NewAuthService(stubUserRepo{}, hasher)
	gameService := game.NewGameService(stubGameRepo{}, stubValidator{}, stubPublisher{}, stubScheduler{})
	matchmakingService := matchmaking.NewMatchmakingService(gameService, userService, stubNotifier{})

	hub := ws.NewHub()
	return NewRouter(
		testSecret,
		web.NewRegisterHandler(userService),
		web.NewAuthHandler(authService, testSecret, time.Hour, false),
		web.NewGameHandler(gameService),
		web.NewFriendsHandler(friends.NewFriendService(friendstest.NewRepo())),
		ws.NewGameMessageHandler(gameService, matchmakingService, nil, hub),
		hub,
	)
}

func do(t *testing.T, router http.Handler, method, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// routedTo reports whether the request reached a handler at all.
//
// A 404 status alone can't answer that: these handlers map a not-found
// record onto 404 too, so a correctly routed request against a stub
// repository looks identical to a missing route by status. Go's ServeMux
// writes "404 page not found" for an unmatched pattern, while the
// handlers write their own messages - that body is the discriminator.
func routedTo(rec *httptest.ResponseRecorder) bool {
	return !strings.Contains(rec.Body.String(), "page not found")
}

// The registered routes must actually resolve. Reaching a handler is all
// this asserts - what the handler then decides is its own test's business.
func TestRegisteredRoutesResolve(t *testing.T) {
	router := newTestRouter(t)

	cases := []struct{ method, path string }{
		{http.MethodPost, "/api/register"},
		{http.MethodGet, "/api/user/abc"},
		{http.MethodGet, "/api/user/email/a@b.com"},
		{http.MethodGet, "/api/user/name/alice"},
		{http.MethodPost, "/api/login"},
		{http.MethodPost, "/api/logout"},
		{http.MethodGet, "/api/games/game-1"},
		{http.MethodGet, "/api/games/game-1/moves"},
	}
	cases = append(cases, friendsRoutes...)

	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			if rec := do(t, router, c.method, c.path, nil); !routedTo(rec) {
				t.Errorf("route is not registered: %d %q", rec.Code, rec.Body.String())
			}
		})
	}
}

var friendsRoutes = []struct{ method, path string }{
	{http.MethodGet, "/api/friends"},
	{http.MethodDelete, "/api/friends/user-2"},
	{http.MethodGet, "/api/friends/requests"},
	{http.MethodPost, "/api/friends/requests"},
	{http.MethodPost, "/api/friends/requests/user-2/accept"},
	{http.MethodDelete, "/api/friends/requests/user-2"},
	{http.MethodGet, "/api/friends/blocks"},
	{http.MethodPut, "/api/friends/blocks/user-2"},
	{http.MethodDelete, "/api/friends/blocks/user-2"},
	{http.MethodGet, "/api/friends/search?q=a"},
}

// Every friends route acts on the caller's own relationships, so none of
// them may answer without a session.
func TestFriendsRoutesRequireAuth(t *testing.T) {
	router := newTestRouter(t)

	for _, c := range friendsRoutes {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			if rec := do(t, router, c.method, c.path, nil); rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d without a session cookie, want 401", rec.Code)
			}
		})
	}
}

func TestUnknownPathIs404(t *testing.T) {
	router := newTestRouter(t)

	rec := do(t, router, http.MethodGet, "/api/nope", nil)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if routedTo(rec) {
		t.Errorf("an unknown path reached a handler: %q", rec.Body.String())
	}
}

// Wrong method on a real path must not fall through to a handler.
func TestWrongMethodIsRejected(t *testing.T) {
	router := newTestRouter(t)

	rec := do(t, router, http.MethodGet, "/api/login", nil)

	if rec.Code == http.StatusOK {
		t.Error("GET /api/login succeeded, want it rejected - login is POST only")
	}
}

// The two protected routes must be closed by default. /api/register and
// /api/login must NOT be, or nobody could ever sign in.
func TestProtectedRoutesRequireAuth(t *testing.T) {
	router := newTestRouter(t)

	for _, path := range []string{"/api/me", "/ws"} {
		t.Run(path, func(t *testing.T) {
			if rec := do(t, router, http.MethodGet, path, nil); rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d without a session cookie, want 401", rec.Code)
			}
		})
	}
}

func TestPublicRoutesAreNotBehindAuth(t *testing.T) {
	router := newTestRouter(t)

	for _, c := range []struct{ method, path string }{
		{http.MethodPost, "/api/register"},
		{http.MethodPost, "/api/login"},
	} {
		t.Run(c.path, func(t *testing.T) {
			if rec := do(t, router, c.method, c.path, nil); rec.Code == http.StatusUnauthorized {
				t.Error("returned 401 without a cookie - this route must stay public")
			}
		})
	}
}

func TestMeRejectsATamperedCookie(t *testing.T) {
	router := newTestRouter(t)

	rec := do(t, router, http.MethodGet, "/api/me", &http.Cookie{Name: auth.CookieName, Value: "not-a-jwt"})

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

// With a valid cookie the request gets past RequireAuth and into the
// handler, which then fails its own lookup against the stub repo. Either
// way it must not be a 401.
func TestMeWithValidCookiePassesTheAuthBoundary(t *testing.T) {
	router := newTestRouter(t)
	token, err := auth.GenerateToken("user-1", testSecret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	rec := do(t, router, http.MethodGet, "/api/me", &http.Cookie{Name: auth.CookieName, Value: token})

	if rec.Code == http.StatusUnauthorized {
		t.Error("a validly signed cookie was rejected at the auth boundary")
	}
}

// Every response goes through the CORS middleware, including errors.
func TestCORSHeadersAreAppliedGlobally(t *testing.T) {
	router := newTestRouter(t)

	rec := do(t, router, http.MethodGet, "/api/nope", nil)

	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("a 404 response carries no CORS headers")
	}
}

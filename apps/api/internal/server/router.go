package server

import (
	"log/slog"
	"net/http"

	"api/internal/auth"
	"api/internal/middleware"
	"api/internal/transport/web"
	"api/internal/transport/ws"
)

func NewRouter(jwtSecret []byte,userHandler *web.RegisterHandler, authHandler *web.AuthHandler, gameHandler *web.GameHandler, gameMessageHandler *ws.GameMessageHandler, hub *ws.Hub) http.Handler {
	mux := http.NewServeMux()

	// User / registration
	mux.HandleFunc("POST /api/register", userHandler.CreateUserHandler)
	mux.HandleFunc("GET /api/user/{id}", userHandler.FindUserByIdHandler)
	mux.HandleFunc("GET /api/user/email/{email}", userHandler.FindUserByEmailHandler)
	mux.HandleFunc("GET /api/user/name/{username}", userHandler.FindUserByUsernameHandler)

	// Auth routes land here once internal/auth exists, e.g.:
	mux.HandleFunc("POST /api/login", authHandler.LoginHandler)
	mux.HandleFunc("POST /api/logout", authHandler.LogoutHandler)
	mux.Handle("GET /api/me", auth.RequireAuth(jwtSecret)(http.HandlerFunc(authHandler.MeHandler)))

	// Game routes
	mux.HandleFunc("GET /api/games/{id}", gameHandler.FindGameByIdHandler)
	mux.HandleFunc("GET /api/games/{id}/moves", gameHandler.FindMovesByGameIdHandler)

	registerWebSocketRoute(mux, hub, jwtSecret, gameMessageHandler)

	// Global middleware wraps everything. Route-specific middleware (like
	// auth.RequireAuth) wraps individual handlers above instead, since it
	// shouldn't apply to /api/register or /api/login themselves.
	return middleware.Logging(middleware.CORS(mux))
}

func registerWebSocketRoute(mux *http.ServeMux, hub *ws.Hub, jwtSecret []byte, gameMessageHandler *ws.GameMessageHandler) {
	wsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			// RequireAuth already rejects anything without a valid
			// session before this handler runs — this branch is
			// defensive, not a real path in practice.
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if err := ws.NewConnection(w, r, hub, userID, gameMessageHandler.Handle); err != nil {
			slog.Error("websocket upgrade failed", "error", err)
		}
	})

	mux.Handle("GET /ws", auth.RequireAuth(jwtSecret)(wsHandler))
}
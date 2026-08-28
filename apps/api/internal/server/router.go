package server

import (
	"net/http"

	"api/internal/auth"
	"api/internal/middleware"
	"api/internal/transport"
)

func NewRouter(jwtSecret []byte,userHandler *transport.RegisterHandler, authHandler *auth.AuthHandler) http.Handler {
	mux := http.NewServeMux()

	// User / registration
	mux.HandleFunc("POST /api/register", userHandler.CreateUserHandler)
	mux.HandleFunc("GET /api/user/{id}", userHandler.FindUserByIdHandler)
	mux.HandleFunc("GET /api/user/email/{email}", userHandler.FindUserByEmailHandler)
	mux.HandleFunc("GET /api/user/name/{username}", userHandler.FindUserByUsernameHandler)

	// Auth routes land here once internal/auth exists, e.g.:
	mux.HandleFunc("POST /api/login", authHandler.LoginHandler)
	// mux.HandleFunc("POST /api/logout", authHandler.LogoutHandler)
	mux.Handle("GET /api/me", auth.RequireAuth(jwtSecret)(http.HandlerFunc(authHandler.MeHandler)))

	// Global middleware wraps everything. Route-specific middleware (like
	// auth.RequireAuth) wraps individual handlers above instead, since it
	// shouldn't apply to /api/register or /api/login themselves.
	return middleware.Logging(middleware.CORS(mux))
}
package middleware

import (
	"net/http"
	"os"
)

// CORS wraps a handler with the headers browsers require for cross-origin
// requests from the frontend, and short-circuits preflight OPTIONS requests.
//
// FRONTEND_ORIGIN must be the exact origin (scheme + host + port) - a
// wildcard ("*") will not work once AllowCredentials-equivalent behavior is
// needed, i.e. as soon as the frontend sends cookies via `credentials: "include"`.
func CORS(next http.Handler) http.Handler {
	origin := getEnv("FRONTEND_ORIGIN", "http://localhost:5173")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
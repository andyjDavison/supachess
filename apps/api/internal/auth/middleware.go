package auth

import (
	"context"
	"net/http"
)

type contextKey string

const userIDContextKey contextKey = "userID"

// CookieName is the name of the cookie RequireAuth reads and that your
// login handler should write the signed JWT into.
const CookieName = "session"

// RequireAuth returns middleware that rejects any request without a valid
// session cookie, and - on success - attaches the authenticated user's id
// to the request context for downstream handlers to read via
// UserIDFromContext.
func RequireAuth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(CookieName)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			userID, err := ParseToken(cookie.Value, secret)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the authenticated user's id set by
// RequireAuth. The bool return is false if called on a request that never
// passed through RequireAuth - always check it rather than assuming a
// non-empty context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok
}
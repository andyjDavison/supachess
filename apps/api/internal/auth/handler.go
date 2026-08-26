package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// ErrInvalidCredentials is what UserProvider.VerifyCredentials should
// return when the email doesn't exist OR the password doesn't match.
// LoginHandler responds identically either way, so a failed login never
// reveals whether the email was even registered.
var ErrInvalidCredentials = errors.New("invalid email or password")

// UserProvider is the minimal capability these handlers need from the user
// domain. Implement it against your existing user.Service - directly if the
// method signatures already line up, or via a thin adapter type if they
// don't - so this package never has to import user's concrete types.
//
// A typical adapter, assuming user.Service already exposes FindByEmail,
// FindByID, and your BcryptHasher exposes Compare(hash, password) error:
//
//	type UserServiceAdapter struct{ svc *user.Service }
//
//	func (a *UserServiceAdapter) VerifyCredentials(ctx context.Context, email, password string) (string, error) {
//	    u, err := a.svc.FindByEmail(ctx, email)
//	    if err != nil {
//	        return "", auth.ErrInvalidCredentials
//	    }
//	    if err := a.svc.Hasher.Compare(u.PasswordHash, password); err != nil {
//	        return "", auth.ErrInvalidCredentials
//	    }
//	    return u.ID, nil
//	}
//
//	func (a *UserServiceAdapter) FindByID(ctx context.Context, id string) (auth.Profile, error) {
//	    u, err := a.svc.FindByID(ctx, id)
//	    if err != nil {
//	        return auth.Profile{}, err
//	    }
//	    return auth.Profile{ID: u.ID, Email: u.Email, Username: u.Username}, nil
//	}
type UserProvider interface {
	// VerifyCredentials checks email+password and returns the user's id on
	// success, or ErrInvalidCredentials on any failure.
	VerifyCredentials(ctx context.Context, email, password string) (userID string, err error)
	// FindByID returns the public profile fields the frontend needs.
	FindByID(ctx context.Context, userID string) (Profile, error)
}

// Profile is the JSON shape returned by MeHandler. Keep this in sync with
// the AuthUser interface in the frontend's AuthContext.tsx.
type Profile struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type Handler struct {
	users    UserProvider
	secret   []byte
	tokenTTL time.Duration
	// secure should be true in production (cookie only sent over HTTPS)
	// and false for local dev over plain http, or browsers will silently
	// drop the cookie and every request will look logged-out.
	secure bool
}

func NewHandler(users UserProvider, secret []byte, tokenTTL time.Duration, secure bool) *Handler {
	return &Handler{users: users, secret: secret, tokenTTL: tokenTTL, secure: secure}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginHandler verifies credentials and, on success, sets a signed JWT as
// an httpOnly session cookie. It never returns the token in the response
// body - only the cookie - so client-side JS never has direct access to it.
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	userID, err := h.users.VerifyCredentials(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, ErrInvalidCredentials.Error(), http.StatusUnauthorized)
		return
	}

	token, err := GenerateToken(userID, h.secret, h.tokenTTL)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.setSessionCookie(w, token, h.tokenTTL)
	w.WriteHeader(http.StatusOK)
}

// LogoutHandler clears the session cookie by re-setting it with the same
// name/path but MaxAge -1, which tells the browser to delete it immediately.
func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	h.setSessionCookie(w, "", -1*time.Second)
	w.WriteHeader(http.StatusOK)
}

// MeHandler returns the current user's profile. It must be mounted behind
// RequireAuth - it trusts that a valid session already put a user id in the
// request context and does not itself touch cookies or tokens.
func (h *Handler) MeHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	profile, err := h.users.FindByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
	})
}
package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

var ErrInvalidCredentials = errors.New("invalid email or password")
var ErrTokenGeneration = errors.New("error generating a token")

// Profile is the JSON shape returned by MeHandler. Keep this in sync with
// the AuthUser interface in the frontend's AuthContext.tsx.
type Profile struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type AuthHandler struct {
	authService *AuthService
	secret   []byte
	tokenTTL time.Duration
	// secure should be true in production (cookie only sent over HTTPS)
	// and false for local dev over plain http, or browsers will silently
	// drop the cookie and every request will look logged-out.
	secure bool
}

func NewAuthHandler(authService *AuthService, secret []byte, tokenTTL time.Duration, secure bool) *AuthHandler {
	return &AuthHandler{authService: authService, secret: secret, tokenTTL: tokenTTL, secure: secure}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	token, err := h.authService.VerifyCredentials(r.Context(), req.Email, req.Password, h.secret, h.tokenTTL)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.setSessionCookie(w, token, h.tokenTTL)
	w.WriteHeader(http.StatusOK)
}

// // LogoutHandler clears the session cookie by re-setting it with the same
// // name/path but MaxAge -1, which tells the browser to delete it immediately.
// func (h *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
// 	h.setSessionCookie(w, "", -1*time.Second)
// 	w.WriteHeader(http.StatusOK)
// }

// // MeHandler returns the current user's profile. It must be mounted behind
// // RequireAuth - it trusts that a valid session already put a user id in the
// // request context and does not itself touch cookies or tokens.
// func (h *AuthHandler) MeHandler(w http.ResponseWriter, r *http.Request) {
// 	userID, ok := UserIDFromContext(r.Context())
// 	if !ok {
// 		http.Error(w, "unauthorized", http.StatusUnauthorized)
// 		return
// 	}

// 	profile, err := h.users.FindByID(r.Context(), userID)
// 	if err != nil {
// 		http.Error(w, "user not found", http.StatusNotFound)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(profile)
// }

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string, ttl time.Duration) {
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
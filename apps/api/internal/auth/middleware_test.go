package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequireAuthRejectsMissingCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	RequireAuth(secret)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler ran without a session cookie")
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/me", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestRequireAuthRejectsBadToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "tampered"})

	rec := httptest.NewRecorder()
	RequireAuth(secret)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler ran with an invalid token")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestRequireAuthPutsUserIDOnContext(t *testing.T) {
	token, _ := GenerateToken("user-123", secret, time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})

	var seen string
	var ok bool
	rec := httptest.NewRecorder()
	RequireAuth(secret)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen, ok = UserIDFromContext(r.Context())
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if !ok || seen != "user-123" {
		t.Errorf("UserIDFromContext = (%q, %v), want (user-123, true)", seen, ok)
	}
}

// Callers must check the bool rather than assume a non-empty string.
func TestUserIDFromContextOnUnwrappedRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if id, ok := UserIDFromContext(req.Context()); ok || id != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", id, ok)
	}
}

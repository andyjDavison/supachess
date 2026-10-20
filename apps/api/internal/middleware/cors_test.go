package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSSetsHeadersAndCallsNext(t *testing.T) {
	called := false
	handler := CORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/me", nil))

	if !called {
		t.Error("next handler was not called for a normal request")
	}
	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want the downstream 418", rec.Code)
	}

	// Credentialed requests need a concrete origin and the credentials
	// header; "*" would break cookie auth.
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "" || got == "*" {
		t.Errorf("Allow-Origin = %q, want a concrete origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Allow-Credentials = %q, want true", got)
	}
	for _, h := range []string{"Access-Control-Allow-Methods", "Access-Control-Allow-Headers"} {
		if rec.Header().Get(h) == "" {
			t.Errorf("%s is empty", h)
		}
	}
}

func TestCORSShortCircuitsPreflight(t *testing.T) {
	handler := CORS(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("next handler ran for a preflight request")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/api/login", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("preflight status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("preflight response carries no Allow-Origin header")
	}
}

func TestCORSUsesFrontendOriginFromEnv(t *testing.T) {
	t.Setenv("FRONTEND_ORIGIN", "https://supachess.example")

	// The origin is read when the middleware is built, not per request.
	handler := CORS(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://supachess.example" {
		t.Errorf("Allow-Origin = %q, want the configured origin", got)
	}
}

package middleware

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The logger writes to the standard logger; keep test output clean.
func quietLogs(t *testing.T) {
	t.Helper()
	original := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(original) })
}

func TestLoggingPassesThrough(t *testing.T) {
	quietLogs(t)
	called := false

	rec := httptest.NewRecorder()
	Logging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("made it"))
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/register", nil))

	if !called {
		t.Error("next handler was not called")
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if rec.Body.String() != "made it" {
		t.Errorf("body = %q, want the downstream body", rec.Body.String())
	}
}

// A panic must become a real 500 rather than a closed connection with no
// response, which is what net/http does by default.
func TestLoggingRecoversPanicAsFiveHundred(t *testing.T) {
	quietLogs(t)

	rec := httptest.NewRecorder()
	handler := Logging(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("handler exploded")
	}))

	// The recover is inside the middleware, so this must not propagate.
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 after a panic", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Error("no response body after a panic - the client has nothing to react to")
	}
}

// If the handler already wrote a status before panicking, the middleware
// must not try to write a second header.
func TestLoggingPanicAfterWriteHeaderKeepsFirstStatus(t *testing.T) {
	quietLogs(t)

	rec := httptest.NewRecorder()
	Logging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		panic("too late")
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want the 200 already written", rec.Code)
	}
}

func TestStatusRecorderKeepsTheFirstStatus(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}

	rec.WriteHeader(http.StatusNotFound)
	rec.WriteHeader(http.StatusInternalServerError) // ignored

	if rec.status != http.StatusNotFound {
		t.Errorf("status = %d, want the first write (404)", rec.status)
	}
}

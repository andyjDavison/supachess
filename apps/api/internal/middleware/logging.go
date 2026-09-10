package middleware

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// statusRecorder wraps http.ResponseWriter to capture the status code that
// actually got written - the standard interface has no way to read that
// back afterward otherwise.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}

// Logging logs every request's method, path, resulting status, and
// duration, and recovers from any panic in a downstream handler.
//
// Without this, a panic mid-handler produces exactly the symptom of "the
// browser shows an empty Network tab entry with no status at all" - Go's
// default panic recovery in net/http closes the connection with no
// response whatsoever, and prints nothing but a stack trace to stderr.
// This instead logs the panic AND sends back a real 500, so the frontend
// gets a response to react to and the terminal shows what happened.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC %s %s: %v", r.Method, r.URL.Path, err)
				if !rec.wroteHeader {
					http.Error(rec, "internal server error", http.StatusInternalServerError)
				}
			}
			log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start))
		}()

		next.ServeHTTP(rec, r)
	})
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("underlying ResponseWriter does not support hijacking")
	}
	return hijacker.Hijack()
}
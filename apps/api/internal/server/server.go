package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	httpServer *http.Server
}

func New(addr string, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// Run starts the server and blocks until it exits - either because it
// failed to start, or because it received an interrupt/terminate signal.
// On signal, in-flight requests get up to shutdownTimeout to finish before
// the server is forced closed.
func (s *Server) Run(shutdownTimeout time.Duration) error {
	errCh := make(chan error, 1)
	go func() {
		log.Printf("Server starting on http://localhost%s...\n", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-stop:
		log.Println("Shutdown signal received, draining in-flight requests...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}
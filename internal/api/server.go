package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// Server wraps an HTTP server with graceful shutdown support.
type Server struct {
	server *http.Server
	router chi.Router
}

// NewServer creates a new Server with the given router and address.
func NewServer(router chi.Router, addr string) *Server {
	return &Server{
		router: router,
		server: &http.Server{
			Addr:         addr,
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// Start begins listening and serving HTTP requests.
// It blocks until the server is shut down.
func (s *Server) Start() error {
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the server with the given context.
// It waits for all active connections to finish or until the context is canceled.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Router returns the server's router.
func (s *Server) Router() chi.Router {
	return s.router
}

// Addr returns the server's address.
func (s *Server) Addr() string {
	return s.server.Addr
}

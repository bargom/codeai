package hooks

import (
	"context"
	"net/http"
	"time"

	"github.com/bargom/codeai/internal/shutdown"
)

// HTTPServer defines the interface for an HTTP server that can be shut down.
type HTTPServer interface {
	Shutdown(ctx context.Context) error
	SetKeepAlivesEnabled(v bool)
}

// HTTPServerShutdownFunc creates a shutdown hook for an HTTP server.
// It disables keep-alives and waits for active connections to drain.
func HTTPServerShutdownFunc(server HTTPServer, drainTimeout time.Duration) shutdown.HookFunc {
	return func(ctx context.Context) error {
		// Stop accepting new connections
		server.SetKeepAlivesEnabled(false)

		// Create shutdown context with drain timeout
		shutdownCtx, cancel := context.WithTimeout(ctx, drainTimeout)
		defer cancel()

		// Shutdown server (waits for active requests)
		return server.Shutdown(shutdownCtx)
	}
}

// HTTPServerShutdown creates a shutdown hook for the standard http.Server.
func HTTPServerShutdown(server *http.Server, drainTimeout time.Duration) shutdown.Hook {
	return shutdown.Hook{
		Name:     "http-server",
		Priority: shutdown.PriorityHTTPServer,
		Fn:       HTTPServerShutdownFunc(server, drainTimeout),
	}
}

// HTTPDrainerShutdownFunc creates a shutdown hook that drains HTTP connections.
func HTTPDrainerShutdownFunc(drainer *shutdown.HTTPDrainer, drainTimeout time.Duration) shutdown.HookFunc {
	return func(ctx context.Context) error {
		// Start draining
		drainer.StartDrain()

		// Wait for active requests with timeout
		shutdownCtx, cancel := context.WithTimeout(ctx, drainTimeout)
		defer cancel()

		return drainer.Wait(shutdownCtx)
	}
}

// HTTPDrainerShutdown creates a shutdown hook for an HTTPDrainer.
func HTTPDrainerShutdown(drainer *shutdown.HTTPDrainer, drainTimeout time.Duration) shutdown.Hook {
	return shutdown.Hook{
		Name:     "http-drainer",
		Priority: shutdown.PriorityHTTPServer,
		Fn:       HTTPDrainerShutdownFunc(drainer, drainTimeout),
	}
}

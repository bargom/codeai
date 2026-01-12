package shutdown

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Drainer tracks and waits for in-flight operations to complete.
type Drainer struct {
	count   atomic.Int64
	wg      sync.WaitGroup
	drainWg sync.WaitGroup
	done    chan struct{}
	once    sync.Once
}

// NewDrainer creates a new Drainer.
func NewDrainer() *Drainer {
	return &Drainer{
		done: make(chan struct{}),
	}
}

// Add increments the count of in-flight operations.
// Returns false if draining has started.
func (d *Drainer) Add() bool {
	select {
	case <-d.done:
		return false
	default:
		d.count.Add(1)
		d.wg.Add(1)
		return true
	}
}

// Done decrements the count of in-flight operations.
func (d *Drainer) Done() {
	d.count.Add(-1)
	d.wg.Done()
}

// Count returns the current number of in-flight operations.
func (d *Drainer) Count() int64 {
	return d.count.Load()
}

// StartDrain signals that draining has started.
// New operations will be rejected after this call.
func (d *Drainer) StartDrain() {
	d.once.Do(func() {
		close(d.done)
	})
}

// Wait blocks until all in-flight operations complete or the context is canceled.
func (d *Drainer) Wait(ctx context.Context) error {
	done := make(chan struct{})

	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WaitWithTimeout blocks until all operations complete or timeout expires.
func (d *Drainer) WaitWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return d.Wait(ctx)
}

// IsDraining returns true if draining has started.
func (d *Drainer) IsDraining() bool {
	select {
	case <-d.done:
		return true
	default:
		return false
	}
}

// HTTPDrainer wraps an HTTP handler to track in-flight requests.
type HTTPDrainer struct {
	drainer *Drainer
	handler http.Handler
}

// NewHTTPDrainer creates a new HTTP drainer wrapping the given handler.
func NewHTTPDrainer(handler http.Handler) *HTTPDrainer {
	return &HTTPDrainer{
		drainer: NewDrainer(),
		handler: handler,
	}
}

// ServeHTTP handles HTTP requests, tracking them for draining.
func (d *HTTPDrainer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !d.drainer.Add() {
		// Draining in progress, reject new requests
		w.Header().Set("Connection", "close")
		http.Error(w, "Service shutting down", http.StatusServiceUnavailable)
		return
	}
	defer d.drainer.Done()

	// Set Connection: close header during shutdown
	if d.drainer.IsDraining() {
		w.Header().Set("Connection", "close")
	}

	d.handler.ServeHTTP(w, r)
}

// StartDrain starts draining connections.
func (d *HTTPDrainer) StartDrain() {
	d.drainer.StartDrain()
}

// Wait waits for all in-flight requests to complete.
func (d *HTTPDrainer) Wait(ctx context.Context) error {
	return d.drainer.Wait(ctx)
}

// ActiveRequests returns the number of in-flight requests.
func (d *HTTPDrainer) ActiveRequests() int64 {
	return d.drainer.Count()
}

// IsDraining returns true if draining has started.
func (d *HTTPDrainer) IsDraining() bool {
	return d.drainer.IsDraining()
}

// DrainMiddleware is a middleware that tracks requests for graceful shutdown.
func DrainMiddleware(drainer *Drainer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !drainer.Add() {
				w.Header().Set("Connection", "close")
				http.Error(w, "Service shutting down", http.StatusServiceUnavailable)
				return
			}
			defer drainer.Done()

			if drainer.IsDraining() {
				w.Header().Set("Connection", "close")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ShutdownReadyMiddleware returns a middleware that checks if the manager is shutting down
// and returns 503 for the /health/ready endpoint.
func ShutdownReadyMiddleware(manager *Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only intercept /health/ready during shutdown
			if r.URL.Path == "/health/ready" && manager.IsShuttingDown() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"status":"unhealthy","reason":"service shutting down"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

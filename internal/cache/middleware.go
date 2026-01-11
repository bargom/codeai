package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"
)

// Middleware provides HTTP caching middleware.
type Middleware struct {
	cache Cache
}

// NewMiddleware creates a new caching middleware.
func NewMiddleware(cache Cache) *Middleware {
	return &Middleware{cache: cache}
}

// Handler returns a middleware handler that caches GET responses.
func (m *Middleware) Handler(ttl time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache GET requests
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			// Generate cache key
			key := m.generateKey(r)

			// Try to get from cache
			cached, err := m.cache.Get(r.Context(), key)
			if err == nil {
				w.Header().Set("X-Cache", "HIT")
				w.Header().Set("Content-Type", "application/json")
				w.Write(cached)
				return
			}

			// Capture response
			rec := &responseRecorder{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rec, r)

			// Cache successful responses
			if rec.statusCode >= 200 && rec.statusCode < 300 {
				_ = m.cache.Set(r.Context(), key, rec.body.Bytes(), ttl)
			}

			w.Header().Set("X-Cache", "MISS")
		})
	}
}

// HandlerWithKeyFunc returns a middleware with custom key generation.
func (m *Middleware) HandlerWithKeyFunc(ttl time.Duration, keyFunc func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			key := keyFunc(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			cached, err := m.cache.Get(r.Context(), key)
			if err == nil {
				w.Header().Set("X-Cache", "HIT")
				w.Header().Set("Content-Type", "application/json")
				w.Write(cached)
				return
			}

			rec := &responseRecorder{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rec, r)

			if rec.statusCode >= 200 && rec.statusCode < 300 {
				_ = m.cache.Set(r.Context(), key, rec.body.Bytes(), ttl)
			}

			w.Header().Set("X-Cache", "MISS")
		})
	}
}

// Invalidate removes a cached response by URL.
func (m *Middleware) Invalidate(r *http.Request) error {
	key := m.generateKey(r)
	return m.cache.Delete(r.Context(), key)
}

// InvalidatePattern removes all cached responses matching a pattern.
func (m *Middleware) InvalidatePattern(pattern string) error {
	return m.cache.DeletePattern(nil, "http:"+pattern)
}

func (m *Middleware) generateKey(r *http.Request) string {
	h := sha256.New()
	h.Write([]byte(r.URL.String()))
	return "http:" + hex.EncodeToString(h.Sum(nil))[:16]
}

// responseRecorder captures the response for caching.
type responseRecorder struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

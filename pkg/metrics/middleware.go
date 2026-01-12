package metrics

import (
	"net/http"
	"regexp"
	"time"
)

// metricsResponseWriter wraps http.ResponseWriter to capture status and size.
type metricsResponseWriter struct {
	http.ResponseWriter
	status int
	size   int64
}

func newMetricsResponseWriter(w http.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
}

func (w *metricsResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += int64(n)
	return n, err
}

// Flush implements http.Flusher for streaming responses.
func (w *metricsResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Unwrap returns the original ResponseWriter for http.ResponseController.
func (w *metricsResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// HTTPMiddleware returns an HTTP middleware that records metrics for each request.
func HTTPMiddleware(registry *Registry) func(http.Handler) http.Handler {
	return HTTPMiddlewareWithOptions(registry, MiddlewareOptions{})
}

// MiddlewareOptions configures the HTTP metrics middleware.
type MiddlewareOptions struct {
	// PathNormalizer customizes how paths are normalized for metrics grouping.
	// If nil, the default normalizer is used.
	PathNormalizer func(string) string

	// SkipPaths contains paths that should not be recorded in metrics.
	SkipPaths []string

	// MaxPathCardinality limits the number of unique paths tracked.
	// Paths beyond this limit will be recorded as "/other".
	// 0 means unlimited (default).
	MaxPathCardinality int
}

// HTTPMiddlewareWithOptions returns an HTTP middleware with custom options.
func HTTPMiddlewareWithOptions(registry *Registry, opts MiddlewareOptions) func(http.Handler) http.Handler {
	if opts.PathNormalizer == nil {
		opts.PathNormalizer = DefaultPathNormalizer
	}

	skipPathsMap := make(map[string]bool)
	for _, p := range opts.SkipPaths {
		skipPathsMap[p] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := opts.PathNormalizer(r.URL.Path)

			// Skip metrics for excluded paths
			if skipPathsMap[path] {
				next.ServeHTTP(w, r)
				return
			}

			method := r.Method
			httpMetrics := registry.HTTP()

			// Track active requests
			httpMetrics.IncActiveRequests(method, path)
			defer httpMetrics.DecActiveRequests(method, path)

			// Wrap response writer
			wrapped := newMetricsResponseWriter(w)

			// Record start time
			start := time.Now()

			// Process request
			next.ServeHTTP(wrapped, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			reqSize := r.ContentLength
			if reqSize < 0 {
				reqSize = 0
			}

			httpMetrics.RecordRequest(
				method,
				path,
				wrapped.status,
				duration,
				reqSize,
				wrapped.size,
			)
		})
	}
}

// Common regex patterns for path normalization
var (
	uuidPattern     = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	numericIDPattern = regexp.MustCompile(`/\d+(?:/|$)`)
	hexIDPattern    = regexp.MustCompile(`/[0-9a-fA-F]{24}(?:/|$)`)
)

// DefaultPathNormalizer normalizes paths by replacing dynamic segments with placeholders.
func DefaultPathNormalizer(path string) string {
	// Replace UUIDs with {id}
	path = uuidPattern.ReplaceAllString(path, "{id}")

	// Replace MongoDB ObjectIDs (24 hex chars) with {id}
	path = hexIDPattern.ReplaceAllStringFunc(path, func(s string) string {
		if s[len(s)-1] == '/' {
			return "/{id}/"
		}
		return "/{id}"
	})

	// Replace numeric IDs with {id}
	path = numericIDPattern.ReplaceAllStringFunc(path, func(s string) string {
		if s[len(s)-1] == '/' {
			return "/{id}/"
		}
		return "/{id}"
	})

	return path
}

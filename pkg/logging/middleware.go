package logging

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// RequestHeaderRequestID is the header name for request ID.
const RequestHeaderRequestID = "X-Request-ID"

// RequestHeaderTraceID is the header name for trace ID.
const RequestHeaderTraceID = "X-Trace-ID"

// HTTPMiddleware provides HTTP request logging middleware.
type HTTPMiddleware struct {
	logger    *slog.Logger
	verbosity Verbosity
}

// Verbosity controls how much request/response detail is logged.
type Verbosity int

const (
	// VerbosityMinimal logs only method, path, status, and duration.
	VerbosityMinimal Verbosity = iota
	// VerbosityStandard logs additional request metadata.
	VerbosityStandard
	// VerbosityVerbose logs request/response headers.
	VerbosityVerbose
)

// NewHTTPMiddleware creates a new HTTP logging middleware.
func NewHTTPMiddleware(logger *slog.Logger) *HTTPMiddleware {
	if logger == nil {
		logger = slog.Default()
	}
	return &HTTPMiddleware{
		logger:    logger.With("component", "http"),
		verbosity: VerbosityStandard,
	}
}

// WithVerbosity sets the logging verbosity level.
func (m *HTTPMiddleware) WithVerbosity(v Verbosity) *HTTPMiddleware {
	return &HTTPMiddleware{
		logger:    m.logger,
		verbosity: v,
	}
}

// Handler returns the middleware handler.
func (m *HTTPMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate or extract request ID
		requestID := r.Header.Get(RequestHeaderRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Extract or use request ID as trace ID
		traceID := r.Header.Get(RequestHeaderTraceID)
		if traceID == "" {
			traceID = requestID
		}

		// Create trace context
		tc := TraceContext{
			RequestID: requestID,
			TraceID:   traceID,
			SpanID:    uuid.New().String(),
		}

		// Add to context
		ctx := tc.ToContext(r.Context())

		// Add request ID to response headers
		w.Header().Set(RequestHeaderRequestID, requestID)
		if traceID != requestID {
			w.Header().Set(RequestHeaderTraceID, traceID)
		}

		// Wrap response writer to capture status and bytes
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// Calculate duration
		duration := time.Since(start)

		// Build log attributes
		attrs := m.buildLogAttrs(r, wrapped, duration)

		// Determine log level based on status code
		level := slog.LevelInfo
		if wrapped.status >= 500 {
			level = slog.LevelError
		} else if wrapped.status >= 400 {
			level = slog.LevelWarn
		}

		m.logger.LogAttrs(ctx, level, "http request", attrs...)
	})
}

// buildLogAttrs builds the log attributes based on verbosity.
func (m *HTTPMiddleware) buildLogAttrs(r *http.Request, w *responseWriter, duration time.Duration) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status", w.status),
		slog.Duration("duration", duration),
	}

	if m.verbosity >= VerbosityStandard {
		attrs = append(attrs,
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.Int64("response_bytes", w.bytes),
		)

		if r.URL.RawQuery != "" {
			attrs = append(attrs, slog.String("query", r.URL.RawQuery))
		}

		if r.ContentLength > 0 {
			attrs = append(attrs, slog.Int64("request_bytes", r.ContentLength))
		}

		if referer := r.Referer(); referer != "" {
			attrs = append(attrs, slog.String("referer", referer))
		}
	}

	if m.verbosity >= VerbosityVerbose {
		// Log request headers (with sensitive data redacted)
		headers := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				if IsSensitiveField(k) {
					headers[k] = RedactedValue
				} else {
					headers[k] = v[0]
				}
			}
		}
		attrs = append(attrs, slog.Any("request_headers", headers))
	}

	return attrs
}

// responseWriter wraps http.ResponseWriter to capture status and bytes written.
type responseWriter struct {
	http.ResponseWriter
	status      int
	bytes       int64
	wroteHeader bool
}

// WriteHeader captures the status code.
func (w *responseWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.status = status
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(status)
}

// Write captures the bytes written.
func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += int64(n)
	return n, err
}

// Unwrap returns the underlying ResponseWriter (for http.ResponseController).
func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// RequestLogger is a convenience function to create a middleware that logs requests.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	mw := NewHTTPMiddleware(logger)
	return mw.Handler
}

// InjectRequestID is a simple middleware that only injects request ID into context.
func InjectRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(RequestHeaderRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx := WithRequestID(r.Context(), requestID)
		w.Header().Set(RequestHeaderRequestID, requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoggerFromContext returns a logger with context values pre-populated.
func LoggerFromContext(ctx context.Context, baseLogger *slog.Logger) *slog.Logger {
	if baseLogger == nil {
		baseLogger = slog.Default()
	}

	tc := FromContext(ctx)
	attrs := make([]any, 0, 8)

	if tc.RequestID != "" {
		attrs = append(attrs, "request_id", tc.RequestID)
	}
	if tc.TraceID != "" {
		attrs = append(attrs, "trace_id", tc.TraceID)
	}
	if tc.UserID != "" {
		attrs = append(attrs, "user_id", tc.UserID)
	}

	return baseLogger.With(attrs...)
}

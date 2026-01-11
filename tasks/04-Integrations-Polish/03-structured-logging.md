# Task: Structured Logging with slog

## Overview
Implement structured logging throughout the runtime using Go's slog package.

## Phase
Phase 4: Integrations and Polish

## Priority
High - Essential for production observability.

## Dependencies
- Phase 1 complete

## Description
Create a comprehensive logging system using slog with support for JSON/text output, log levels, context propagation, and request tracing.

## Detailed Requirements

### 1. Logger Setup (internal/logging/logger.go)

```go
package logging

import (
    "context"
    "io"
    "log/slog"
    "os"
    "runtime"
    "time"
)

type LogConfig struct {
    Level      string // debug, info, warn, error
    Format     string // json, text
    Output     string // stdout, stderr, file path
    AddSource  bool
    TimeFormat string
}

func SetupLogger(config LogConfig) *slog.Logger {
    var level slog.Level
    switch config.Level {
    case "debug":
        level = slog.LevelDebug
    case "warn":
        level = slog.LevelWarn
    case "error":
        level = slog.LevelError
    default:
        level = slog.LevelInfo
    }

    var output io.Writer
    switch config.Output {
    case "", "stdout":
        output = os.Stdout
    case "stderr":
        output = os.Stderr
    default:
        f, err := os.OpenFile(config.Output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
        if err != nil {
            output = os.Stdout
        } else {
            output = f
        }
    }

    opts := &slog.HandlerOptions{
        Level:     level,
        AddSource: config.AddSource,
    }

    var handler slog.Handler
    if config.Format == "json" {
        handler = slog.NewJSONHandler(output, opts)
    } else {
        handler = slog.NewTextHandler(output, opts)
    }

    // Add default attributes
    handler = &ContextHandler{
        Handler: handler,
    }

    logger := slog.New(handler)
    slog.SetDefault(logger)

    return logger
}

// ContextHandler adds context values to log records
type ContextHandler struct {
    slog.Handler
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
    // Add request ID if present
    if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
        r.AddAttrs(slog.String("request_id", requestID))
    }

    // Add trace ID if present
    if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
        r.AddAttrs(slog.String("trace_id", traceID))
    }

    // Add user ID if present
    if userID, ok := ctx.Value(UserIDKey).(string); ok {
        r.AddAttrs(slog.String("user_id", userID))
    }

    return h.Handler.Handle(ctx, r)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &ContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
    return &ContextHandler{Handler: h.Handler.WithGroup(name)}
}

type contextKey string

const (
    RequestIDKey contextKey = "request_id"
    TraceIDKey   contextKey = "trace_id"
    UserIDKey    contextKey = "user_id"
)
```

### 2. HTTP Logging Middleware (internal/logging/http.go)

```go
package logging

import (
    "net/http"
    "time"

    "github.com/google/uuid"
    "log/slog"
)

type HTTPLogMiddleware struct {
    logger *slog.Logger
}

func NewHTTPLogMiddleware(logger *slog.Logger) *HTTPLogMiddleware {
    return &HTTPLogMiddleware{logger: logger}
}

func (m *HTTPLogMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Generate or extract request ID
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }

        // Extract trace ID
        traceID := r.Header.Get("X-Trace-ID")
        if traceID == "" {
            traceID = requestID
        }

        // Add to context
        ctx := r.Context()
        ctx = context.WithValue(ctx, RequestIDKey, requestID)
        ctx = context.WithValue(ctx, TraceIDKey, traceID)

        // Add to response headers
        w.Header().Set("X-Request-ID", requestID)

        // Wrap response writer to capture status
        wrapped := &responseWriter{ResponseWriter: w, status: 200}

        // Process request
        next.ServeHTTP(wrapped, r.WithContext(ctx))

        // Log request
        duration := time.Since(start)

        attrs := []slog.Attr{
            slog.String("method", r.Method),
            slog.String("path", r.URL.Path),
            slog.Int("status", wrapped.status),
            slog.Duration("duration", duration),
            slog.String("remote_addr", r.RemoteAddr),
            slog.String("user_agent", r.UserAgent()),
            slog.Int64("bytes", wrapped.bytes),
        }

        if r.URL.RawQuery != "" {
            attrs = append(attrs, slog.String("query", r.URL.RawQuery))
        }

        level := slog.LevelInfo
        if wrapped.status >= 500 {
            level = slog.LevelError
        } else if wrapped.status >= 400 {
            level = slog.LevelWarn
        }

        m.logger.LogAttrs(ctx, level, "http request", attrs...)
    })
}

type responseWriter struct {
    http.ResponseWriter
    status int
    bytes  int64
}

func (w *responseWriter) WriteHeader(status int) {
    w.status = status
    w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(b []byte) (int, error) {
    n, err := w.ResponseWriter.Write(b)
    w.bytes += int64(n)
    return n, err
}
```

### 3. Database Query Logging (internal/logging/database.go)

```go
package logging

import (
    "context"
    "time"

    "log/slog"
)

type QueryLogger struct {
    logger    *slog.Logger
    slowQuery time.Duration
}

func NewQueryLogger(logger *slog.Logger, slowQuery time.Duration) *QueryLogger {
    if slowQuery == 0 {
        slowQuery = 100 * time.Millisecond
    }
    return &QueryLogger{
        logger:    logger.With("component", "database"),
        slowQuery: slowQuery,
    }
}

func (l *QueryLogger) LogQuery(ctx context.Context, query string, args []any, duration time.Duration, err error) {
    attrs := []slog.Attr{
        slog.String("query", truncate(query, 500)),
        slog.Duration("duration", duration),
    }

    if len(args) > 0 {
        attrs = append(attrs, slog.Int("args_count", len(args)))
    }

    if err != nil {
        attrs = append(attrs, slog.String("error", err.Error()))
        l.logger.LogAttrs(ctx, slog.LevelError, "query failed", attrs...)
        return
    }

    level := slog.LevelDebug
    if duration > l.slowQuery {
        level = slog.LevelWarn
        attrs = append(attrs, slog.Bool("slow", true))
    }

    l.logger.LogAttrs(ctx, level, "query executed", attrs...)
}

func truncate(s string, max int) string {
    if len(s) <= max {
        return s
    }
    return s[:max] + "..."
}
```

### 4. Module Logging (internal/logging/module.go)

```go
package logging

import "log/slog"

// ModuleLogger creates a logger for a specific module
func ModuleLogger(module string) *slog.Logger {
    return slog.Default().With("module", module)
}

// WithOperation adds operation context
func WithOperation(logger *slog.Logger, operation string) *slog.Logger {
    return logger.With("operation", operation)
}

// WithEntity adds entity context
func WithEntity(logger *slog.Logger, entity, id string) *slog.Logger {
    return logger.With(
        slog.String("entity", entity),
        slog.String("entity_id", id),
    )
}

// WithWorkflow adds workflow context
func WithWorkflow(logger *slog.Logger, workflowID, executionID string) *slog.Logger {
    return logger.With(
        slog.String("workflow_id", workflowID),
        slog.String("execution_id", executionID),
    )
}

// WithJob adds job context
func WithJob(logger *slog.Logger, jobID string) *slog.Logger {
    return logger.With("job_id", jobID)
}
```

### 5. Sensitive Data Redaction (internal/logging/redact.go)

```go
package logging

import (
    "reflect"
    "regexp"
    "strings"
)

var sensitiveFields = map[string]bool{
    "password":      true,
    "secret":        true,
    "token":         true,
    "api_key":       true,
    "apikey":        true,
    "authorization": true,
    "credit_card":   true,
    "ssn":           true,
}

var sensitivePatterns = []*regexp.Regexp{
    regexp.MustCompile(`(?i)password[\"']?\s*[:=]\s*[\"']?[^\"'\s,}]+`),
    regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9\-_\.]+`),
}

const redactedValue = "[REDACTED]"

// RedactSensitive redacts sensitive fields from a map
func RedactSensitive(data map[string]any) map[string]any {
    result := make(map[string]any)

    for k, v := range data {
        if sensitiveFields[strings.ToLower(k)] {
            result[k] = redactedValue
            continue
        }

        switch val := v.(type) {
        case map[string]any:
            result[k] = RedactSensitive(val)
        case string:
            result[k] = redactString(val)
        default:
            result[k] = v
        }
    }

    return result
}

func redactString(s string) string {
    result := s
    for _, pattern := range sensitivePatterns {
        result = pattern.ReplaceAllString(result, redactedValue)
    }
    return result
}

// SafeAttrs creates slog attributes with sensitive data redacted
func SafeAttrs(data map[string]any) []slog.Attr {
    redacted := RedactSensitive(data)
    var attrs []slog.Attr

    for k, v := range redacted {
        attrs = append(attrs, slog.Any(k, v))
    }

    return attrs
}
```

## Acceptance Criteria
- [ ] JSON and text output formats
- [ ] Configurable log levels
- [ ] Request ID propagation
- [ ] HTTP request/response logging
- [ ] Database query logging with slow query detection
- [ ] Sensitive data redaction
- [ ] Module-specific loggers
- [ ] Context-aware logging

## Testing Strategy
- Unit tests for redaction
- Unit tests for log formatting
- Integration tests with HTTP middleware

## Files to Create
- `internal/logging/logger.go`
- `internal/logging/http.go`
- `internal/logging/database.go`
- `internal/logging/module.go`
- `internal/logging/redact.go`
- `internal/logging/logging_test.go`

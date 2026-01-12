package logging

import (
	"context"
	"io"
	"log/slog"
	"math/rand"
	"sync"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// RequestIDKey is the context key for request IDs.
	RequestIDKey contextKey = "request_id"
	// TraceIDKey is the context key for trace IDs.
	TraceIDKey contextKey = "trace_id"
	// UserIDKey is the context key for user IDs.
	UserIDKey contextKey = "user_id"
	// SpanIDKey is the context key for span IDs.
	SpanIDKey contextKey = "span_id"
)

// Logger wraps slog.Logger with additional functionality.
type Logger struct {
	*slog.Logger
	config Config
}

// New creates a new Logger with the given configuration.
func New(config Config) *Logger {
	output := config.GetOutput()
	return NewWithWriter(config, output)
}

// NewWithWriter creates a new Logger with a custom writer.
func NewWithWriter(config Config, w io.Writer) *Logger {
	level := ParseLevel(config.Level)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: config.AddSource,
	}

	var handler slog.Handler
	if config.Format == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	// Wrap with context handler to extract values from context
	contextHandler := &ContextHandler{
		Handler:    handler,
		sampleRate: config.SampleRate,
	}

	return &Logger{
		Logger: slog.New(contextHandler),
		config: config,
	}
}

// SetDefault sets this logger as the default slog logger.
func (l *Logger) SetDefault() {
	slog.SetDefault(l.Logger)
}

// With returns a new Logger with the given attributes.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
		config: l.config,
	}
}

// WithGroup returns a new Logger with the given group name.
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{
		Logger: l.Logger.WithGroup(name),
		config: l.config,
	}
}

// WithModule returns a new Logger with module context.
func (l *Logger) WithModule(module string) *Logger {
	return l.With("module", module)
}

// WithOperation returns a new Logger with operation context.
func (l *Logger) WithOperation(operation string) *Logger {
	return l.With("operation", operation)
}

// WithEntity returns a new Logger with entity context.
func (l *Logger) WithEntity(entity, id string) *Logger {
	return l.With(
		slog.String("entity", entity),
		slog.String("entity_id", id),
	)
}

// WithWorkflow returns a new Logger with workflow context.
func (l *Logger) WithWorkflow(workflowID, executionID string) *Logger {
	return l.With(
		slog.String("workflow_id", workflowID),
		slog.String("execution_id", executionID),
	)
}

// WithJob returns a new Logger with job context.
func (l *Logger) WithJob(jobID string) *Logger {
	return l.With("job_id", jobID)
}

// ContextHandler is a slog.Handler that extracts context values.
type ContextHandler struct {
	slog.Handler
	sampleRate float64
	mu         sync.RWMutex
}

// Enabled reports whether the handler handles records at the given level.
func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// Apply sampling for debug logs
	if level == slog.LevelDebug && h.sampleRate < 1.0 {
		if rand.Float64() > h.sampleRate {
			return false
		}
	}
	return h.Handler.Enabled(ctx, level)
}

// Handle adds context values to the log record and passes to the wrapped handler.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	// Add request ID if present
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok && requestID != "" {
		r.AddAttrs(slog.String("request_id", requestID))
	}

	// Add trace ID if present
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		r.AddAttrs(slog.String("trace_id", traceID))
	}

	// Add user ID if present
	if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
		r.AddAttrs(slog.String("user_id", userID))
	}

	// Add span ID if present
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok && spanID != "" {
		r.AddAttrs(slog.String("span_id", spanID))
	}

	return h.Handler.Handle(ctx, r)
}

// WithAttrs returns a new ContextHandler with the given attributes.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{
		Handler:    h.Handler.WithAttrs(attrs),
		sampleRate: h.sampleRate,
	}
}

// WithGroup returns a new ContextHandler with the given group.
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{
		Handler:    h.Handler.WithGroup(name),
		sampleRate: h.sampleRate,
	}
}

// Default returns a default logger using environment configuration.
func Default() *Logger {
	return New(ConfigFromEnv())
}

// ModuleLogger creates a logger for a specific module using the default logger.
func ModuleLogger(module string) *slog.Logger {
	return slog.Default().With("module", module)
}

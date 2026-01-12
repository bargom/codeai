package logging

import (
	"context"

	"github.com/google/uuid"
)

// TraceContext holds tracing information for a request.
type TraceContext struct {
	RequestID string
	TraceID   string
	SpanID    string
	UserID    string
}

// NewTraceContext creates a new TraceContext with generated IDs.
func NewTraceContext() TraceContext {
	requestID := uuid.New().String()
	return TraceContext{
		RequestID: requestID,
		TraceID:   requestID, // Use same ID as default
		SpanID:    uuid.New().String(),
	}
}

// NewTraceContextWithParent creates a new TraceContext inheriting the parent trace ID.
func NewTraceContextWithParent(parentTraceID string) TraceContext {
	tc := NewTraceContext()
	if parentTraceID != "" {
		tc.TraceID = parentTraceID
	}
	return tc
}

// WithRequestID sets the request ID.
func (tc TraceContext) WithRequestID(id string) TraceContext {
	tc.RequestID = id
	return tc
}

// WithTraceID sets the trace ID.
func (tc TraceContext) WithTraceID(id string) TraceContext {
	tc.TraceID = id
	return tc
}

// WithSpanID sets the span ID.
func (tc TraceContext) WithSpanID(id string) TraceContext {
	tc.SpanID = id
	return tc
}

// WithUserID sets the user ID.
func (tc TraceContext) WithUserID(id string) TraceContext {
	tc.UserID = id
	return tc
}

// ToContext adds the trace context to a context.Context.
func (tc TraceContext) ToContext(ctx context.Context) context.Context {
	if tc.RequestID != "" {
		ctx = context.WithValue(ctx, RequestIDKey, tc.RequestID)
	}
	if tc.TraceID != "" {
		ctx = context.WithValue(ctx, TraceIDKey, tc.TraceID)
	}
	if tc.SpanID != "" {
		ctx = context.WithValue(ctx, SpanIDKey, tc.SpanID)
	}
	if tc.UserID != "" {
		ctx = context.WithValue(ctx, UserIDKey, tc.UserID)
	}
	return ctx
}

// FromContext extracts a TraceContext from a context.Context.
func FromContext(ctx context.Context) TraceContext {
	tc := TraceContext{}

	if v, ok := ctx.Value(RequestIDKey).(string); ok {
		tc.RequestID = v
	}
	if v, ok := ctx.Value(TraceIDKey).(string); ok {
		tc.TraceID = v
	}
	if v, ok := ctx.Value(SpanIDKey).(string); ok {
		tc.SpanID = v
	}
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		tc.UserID = v
	}

	return tc
}

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequestIDKey, id)
}

// WithTraceID adds a trace ID to the context.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, TraceIDKey, id)
}

// WithSpanID adds a span ID to the context.
func WithSpanID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, SpanIDKey, id)
}

// WithUserID adds a user ID to the context.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, UserIDKey, id)
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(RequestIDKey).(string); ok {
		return v
	}
	return ""
}

// GetTraceID retrieves the trace ID from the context.
func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(TraceIDKey).(string); ok {
		return v
	}
	return ""
}

// GetSpanID retrieves the span ID from the context.
func GetSpanID(ctx context.Context) string {
	if v, ok := ctx.Value(SpanIDKey).(string); ok {
		return v
	}
	return ""
}

// GetUserID retrieves the user ID from the context.
func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

// GenerateRequestID generates a new UUID v4 request ID.
func GenerateRequestID() string {
	return uuid.New().String()
}

// GenerateSpanID generates a new UUID v4 span ID.
func GenerateSpanID() string {
	return uuid.New().String()
}

package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTraceContext(t *testing.T) {
	tc := NewTraceContext()

	assert.NotEmpty(t, tc.RequestID)
	assert.NotEmpty(t, tc.TraceID)
	assert.NotEmpty(t, tc.SpanID)
	assert.Empty(t, tc.UserID)

	// Request ID and Trace ID should match by default
	assert.Equal(t, tc.RequestID, tc.TraceID)
}

func TestNewTraceContextWithParent(t *testing.T) {
	parentTraceID := "parent-trace-123"
	tc := NewTraceContextWithParent(parentTraceID)

	assert.NotEmpty(t, tc.RequestID)
	assert.Equal(t, parentTraceID, tc.TraceID)
	assert.NotEqual(t, tc.RequestID, tc.TraceID) // Should be different
	assert.NotEmpty(t, tc.SpanID)
}

func TestNewTraceContextWithParent_EmptyParent(t *testing.T) {
	tc := NewTraceContextWithParent("")

	assert.NotEmpty(t, tc.TraceID)
	// When parent is empty, trace ID should equal request ID
	assert.Equal(t, tc.RequestID, tc.TraceID)
}

func TestTraceContext_WithMethods(t *testing.T) {
	tc := TraceContext{}

	tc = tc.WithRequestID("req-123")
	assert.Equal(t, "req-123", tc.RequestID)

	tc = tc.WithTraceID("trace-456")
	assert.Equal(t, "trace-456", tc.TraceID)

	tc = tc.WithSpanID("span-789")
	assert.Equal(t, "span-789", tc.SpanID)

	tc = tc.WithUserID("user-abc")
	assert.Equal(t, "user-abc", tc.UserID)
}

func TestTraceContext_ToContext(t *testing.T) {
	tc := TraceContext{
		RequestID: "req-123",
		TraceID:   "trace-456",
		SpanID:    "span-789",
		UserID:    "user-abc",
	}

	ctx := tc.ToContext(context.Background())

	assert.Equal(t, "req-123", ctx.Value(RequestIDKey))
	assert.Equal(t, "trace-456", ctx.Value(TraceIDKey))
	assert.Equal(t, "span-789", ctx.Value(SpanIDKey))
	assert.Equal(t, "user-abc", ctx.Value(UserIDKey))
}

func TestTraceContext_ToContext_PartialValues(t *testing.T) {
	tc := TraceContext{
		RequestID: "req-123",
	}

	ctx := tc.ToContext(context.Background())

	assert.Equal(t, "req-123", ctx.Value(RequestIDKey))
	assert.Nil(t, ctx.Value(TraceIDKey))
	assert.Nil(t, ctx.Value(SpanIDKey))
	assert.Nil(t, ctx.Value(UserIDKey))
}

func TestFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, RequestIDKey, "req-123")
	ctx = context.WithValue(ctx, TraceIDKey, "trace-456")
	ctx = context.WithValue(ctx, SpanIDKey, "span-789")
	ctx = context.WithValue(ctx, UserIDKey, "user-abc")

	tc := FromContext(ctx)

	assert.Equal(t, "req-123", tc.RequestID)
	assert.Equal(t, "trace-456", tc.TraceID)
	assert.Equal(t, "span-789", tc.SpanID)
	assert.Equal(t, "user-abc", tc.UserID)
}

func TestFromContext_EmptyContext(t *testing.T) {
	tc := FromContext(context.Background())

	assert.Empty(t, tc.RequestID)
	assert.Empty(t, tc.TraceID)
	assert.Empty(t, tc.SpanID)
	assert.Empty(t, tc.UserID)
}

func TestWithRequestID(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-123")
	assert.Equal(t, "req-123", GetRequestID(ctx))
}

func TestWithTraceID(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-456")
	assert.Equal(t, "trace-456", GetTraceID(ctx))
}

func TestWithSpanID(t *testing.T) {
	ctx := WithSpanID(context.Background(), "span-789")
	assert.Equal(t, "span-789", GetSpanID(ctx))
}

func TestWithUserID(t *testing.T) {
	ctx := WithUserID(context.Background(), "user-abc")
	assert.Equal(t, "user-abc", GetUserID(ctx))
}

func TestGetRequestID_Missing(t *testing.T) {
	ctx := context.Background()
	assert.Empty(t, GetRequestID(ctx))
}

func TestGetTraceID_Missing(t *testing.T) {
	ctx := context.Background()
	assert.Empty(t, GetTraceID(ctx))
}

func TestGetSpanID_Missing(t *testing.T) {
	ctx := context.Background()
	assert.Empty(t, GetSpanID(ctx))
}

func TestGetUserID_Missing(t *testing.T) {
	ctx := context.Background()
	assert.Empty(t, GetUserID(ctx))
}

func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)

	// Should be valid UUID format (36 characters with dashes)
	assert.Len(t, id1, 36)
}

func TestGenerateSpanID(t *testing.T) {
	id1 := GenerateSpanID()
	id2 := GenerateSpanID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)

	// Should be valid UUID format (36 characters with dashes)
	assert.Len(t, id1, 36)
}

func TestTraceContext_Immutability(t *testing.T) {
	tc1 := TraceContext{RequestID: "original"}
	tc2 := tc1.WithRequestID("modified")

	// Original should be unchanged
	assert.Equal(t, "original", tc1.RequestID)
	assert.Equal(t, "modified", tc2.RequestID)
}

func TestRoundTrip_ContextToTraceContext(t *testing.T) {
	original := TraceContext{
		RequestID: "req-123",
		TraceID:   "trace-456",
		SpanID:    "span-789",
		UserID:    "user-abc",
	}

	ctx := original.ToContext(context.Background())
	restored := FromContext(ctx)

	assert.Equal(t, original.RequestID, restored.RequestID)
	assert.Equal(t, original.TraceID, restored.TraceID)
	assert.Equal(t, original.SpanID, restored.SpanID)
	assert.Equal(t, original.UserID, restored.UserID)
}

package event

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvent(t *testing.T) {
	payload := map[string]string{"key": "value"}
	event := NewEvent(EventJobCreated, payload)

	assert.Equal(t, EventJobCreated, event.Type)
	assert.Equal(t, payload, event.Payload)
	assert.False(t, event.Timestamp.IsZero())
}

func TestDispatcher_Dispatch(t *testing.T) {
	d := NewDispatcher()
	ctx := context.Background()

	var called int32
	handler := func(ctx context.Context, e Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	}

	d.Subscribe(EventJobCreated, handler)

	event := NewEvent(EventJobCreated, nil)
	err := d.Dispatch(ctx, event)

	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&called))
}

func TestDispatcher_DispatchMultipleHandlers(t *testing.T) {
	d := NewDispatcher()
	ctx := context.Background()

	var called int32
	handler1 := func(ctx context.Context, e Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	}
	handler2 := func(ctx context.Context, e Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	}

	d.Subscribe(EventJobCreated, handler1)
	d.Subscribe(EventJobCreated, handler2)

	event := NewEvent(EventJobCreated, nil)
	err := d.Dispatch(ctx, event)

	require.NoError(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&called))
}

func TestDispatcher_DispatchNoHandlers(t *testing.T) {
	d := NewDispatcher()
	ctx := context.Background()

	event := NewEvent(EventJobCreated, nil)
	err := d.Dispatch(ctx, event)

	require.NoError(t, err)
}

func TestDispatcher_DispatchHandlerError(t *testing.T) {
	d := NewDispatcher()
	ctx := context.Background()

	var called int32
	handler1 := func(ctx context.Context, e Event) error {
		atomic.AddInt32(&called, 1)
		return errors.New("handler error")
	}
	handler2 := func(ctx context.Context, e Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	}

	d.Subscribe(EventJobCreated, handler1)
	d.Subscribe(EventJobCreated, handler2)

	event := NewEvent(EventJobCreated, nil)
	err := d.Dispatch(ctx, event)

	// Should not return error and continue with other handlers
	require.NoError(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&called))
}

func TestDispatcher_Subscribe(t *testing.T) {
	d := NewDispatcher()

	handler := func(ctx context.Context, e Event) error {
		return nil
	}

	d.Subscribe(EventJobCreated, handler)
	d.Subscribe(EventJobCompleted, handler)
	d.Subscribe(EventJobFailed, handler)

	// Verify handlers are registered by dispatching events
	ctx := context.Background()

	var called int32
	countingHandler := func(ctx context.Context, e Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	}

	d.Subscribe(EventJobStarted, countingHandler)

	_ = d.Dispatch(ctx, NewEvent(EventJobStarted, nil))
	assert.Equal(t, int32(1), atomic.LoadInt32(&called))
}

func TestNoOpDispatcher(t *testing.T) {
	d := NewNoOpDispatcher()
	ctx := context.Background()

	// Subscribe should not panic
	d.Subscribe(EventJobCreated, func(ctx context.Context, e Event) error {
		t.Fatal("handler should not be called")
		return nil
	})

	// Dispatch should not call handlers
	err := d.Dispatch(ctx, NewEvent(EventJobCreated, nil))
	require.NoError(t, err)

	// Unsubscribe should not panic
	d.Unsubscribe(EventJobCreated, nil)
}

func TestEventTypes(t *testing.T) {
	// Verify all event types are defined
	assert.Equal(t, EventType("job.created"), EventJobCreated)
	assert.Equal(t, EventType("job.scheduled"), EventJobScheduled)
	assert.Equal(t, EventType("job.started"), EventJobStarted)
	assert.Equal(t, EventType("job.completed"), EventJobCompleted)
	assert.Equal(t, EventType("job.failed"), EventJobFailed)
	assert.Equal(t, EventType("job.cancelled"), EventJobCancelled)
	assert.Equal(t, EventType("job.retrying"), EventJobRetrying)
}

func TestEvent_Timestamp(t *testing.T) {
	before := time.Now()
	event := NewEvent(EventJobCreated, nil)
	after := time.Now()

	assert.True(t, event.Timestamp.After(before) || event.Timestamp.Equal(before))
	assert.True(t, event.Timestamp.Before(after) || event.Timestamp.Equal(after))
}

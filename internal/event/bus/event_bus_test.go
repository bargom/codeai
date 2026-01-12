package bus

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger implements Logger for testing.
type mockLogger struct {
	mu       sync.Mutex
	messages []string
}

func (m *mockLogger) Info(msg string, args ...any)  { m.log("INFO", msg) }
func (m *mockLogger) Error(msg string, args ...any) { m.log("ERROR", msg) }
func (m *mockLogger) Debug(msg string, args ...any) { m.log("DEBUG", msg) }
func (m *mockLogger) Warn(msg string, args ...any)  { m.log("WARN", msg) }

func (m *mockLogger) log(level, msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, level+": "+msg)
}

func (m *mockLogger) getMessages() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.messages))
	copy(result, m.messages)
	return result
}

func TestNewEventBus(t *testing.T) {
	logger := &mockLogger{}
	eb := NewEventBus(logger)
	defer eb.Close()

	assert.NotNil(t, eb)
	assert.NotNil(t, eb.subscribers)
	assert.Equal(t, logger, eb.logger)
}

func TestEventBus_Subscribe(t *testing.T) {
	logger := &mockLogger{}
	eb := NewEventBus(logger)
	defer eb.Close()

	subscriber := SubscriberFunc(func(ctx context.Context, event Event) error {
		return nil
	})

	eb.Subscribe(EventWorkflowStarted, subscriber)

	assert.Equal(t, 1, eb.SubscriberCount(EventWorkflowStarted))
	assert.Equal(t, 0, eb.SubscriberCount(EventJobStarted))
}

func TestEventBus_Publish(t *testing.T) {
	logger := &mockLogger{}
	eb := NewEventBus(logger)
	defer eb.Close()

	var received atomic.Bool
	var receivedEvent Event

	subscriber := SubscriberFunc(func(ctx context.Context, event Event) error {
		received.Store(true)
		receivedEvent = event
		return nil
	})

	eb.Subscribe(EventWorkflowStarted, subscriber)

	event := Event{
		ID:        "test-1",
		Type:      EventWorkflowStarted,
		Source:    "test",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"key": "value"},
		Metadata:  map[string]string{"meta": "data"},
	}

	err := eb.Publish(context.Background(), event)

	require.NoError(t, err)
	assert.True(t, received.Load())
	assert.Equal(t, event.ID, receivedEvent.ID)
	assert.Equal(t, event.Type, receivedEvent.Type)
}

func TestEventBus_PublishToMultipleSubscribers(t *testing.T) {
	logger := &mockLogger{}
	eb := NewEventBus(logger)
	defer eb.Close()

	var count atomic.Int32

	for i := 0; i < 3; i++ {
		subscriber := SubscriberFunc(func(ctx context.Context, event Event) error {
			count.Add(1)
			return nil
		})
		eb.Subscribe(EventJobCompleted, subscriber)
	}

	event := Event{
		ID:   "test-1",
		Type: EventJobCompleted,
	}

	err := eb.Publish(context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, int32(3), count.Load())
}

func TestEventBus_SubscriberErrorIsolation(t *testing.T) {
	logger := &mockLogger{}
	eb := NewEventBus(logger)
	defer eb.Close()

	var successCount atomic.Int32

	// First subscriber succeeds
	eb.Subscribe(EventWorkflowFailed, SubscriberFunc(func(ctx context.Context, event Event) error {
		successCount.Add(1)
		return nil
	}))

	// Second subscriber fails
	eb.Subscribe(EventWorkflowFailed, SubscriberFunc(func(ctx context.Context, event Event) error {
		return errors.New("subscriber error")
	}))

	// Third subscriber succeeds
	eb.Subscribe(EventWorkflowFailed, SubscriberFunc(func(ctx context.Context, event Event) error {
		successCount.Add(1)
		return nil
	}))

	event := Event{
		ID:   "test-1",
		Type: EventWorkflowFailed,
	}

	err := eb.Publish(context.Background(), event)

	// No error returned even though one subscriber failed
	require.NoError(t, err)
	// Both successful subscribers ran
	assert.Equal(t, int32(2), successCount.Load())
}

func TestEventBus_SubscriberPanicRecovery(t *testing.T) {
	logger := &mockLogger{}
	eb := NewEventBus(logger)
	defer eb.Close()

	var successCount atomic.Int32

	// First subscriber succeeds
	eb.Subscribe(EventJobFailed, SubscriberFunc(func(ctx context.Context, event Event) error {
		successCount.Add(1)
		return nil
	}))

	// Second subscriber panics
	eb.Subscribe(EventJobFailed, SubscriberFunc(func(ctx context.Context, event Event) error {
		panic("subscriber panic")
	}))

	// Third subscriber succeeds
	eb.Subscribe(EventJobFailed, SubscriberFunc(func(ctx context.Context, event Event) error {
		successCount.Add(1)
		return nil
	}))

	event := Event{
		ID:   "test-1",
		Type: EventJobFailed,
	}

	// Should not panic
	err := eb.Publish(context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, int32(2), successCount.Load())
}

func TestEventBus_PublishAsync(t *testing.T) {
	logger := &mockLogger{}
	eb := NewEventBusWithConfig(logger, Config{
		AsyncBufferSize: 100,
		WorkerPoolSize:  2,
	})

	var received atomic.Bool
	var wg sync.WaitGroup
	wg.Add(1)

	subscriber := SubscriberFunc(func(ctx context.Context, event Event) error {
		received.Store(true)
		wg.Done()
		return nil
	})

	eb.Subscribe(EventAgentExecuted, subscriber)

	event := Event{
		ID:   "async-1",
		Type: EventAgentExecuted,
	}

	eb.PublishAsync(context.Background(), event)

	// Wait for async processing
	wg.Wait()
	eb.Close()

	assert.True(t, received.Load())
}

func TestEventBus_Close(t *testing.T) {
	logger := &mockLogger{}
	eb := NewEventBus(logger)

	var processedCount atomic.Int32

	subscriber := SubscriberFunc(func(ctx context.Context, event Event) error {
		processedCount.Add(1)
		return nil
	})

	eb.Subscribe(EventEmailSent, subscriber)

	// Queue some async events
	for i := 0; i < 5; i++ {
		event := Event{
			ID:   "close-test-" + string(rune('0'+i)),
			Type: EventEmailSent,
		}
		eb.PublishAsync(context.Background(), event)
	}

	// Close should drain the buffer
	eb.Close()

	assert.Equal(t, int32(5), processedCount.Load())
}

func TestEventBus_PublishAsyncAfterClose(t *testing.T) {
	logger := &mockLogger{}
	eb := NewEventBus(logger)
	eb.Close()

	// Should not panic
	event := Event{
		ID:   "after-close",
		Type: EventEmailSent,
	}
	eb.PublishAsync(context.Background(), event)

	// Check that warning was logged
	messages := logger.getMessages()
	found := false
	for _, msg := range messages {
		if msg == "WARN: attempted to publish async to closed bus" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestEventBus_NoSubscribers(t *testing.T) {
	logger := &mockLogger{}
	eb := NewEventBus(logger)
	defer eb.Close()

	event := Event{
		ID:   "no-subs",
		Type: EventWebhookTriggered,
	}

	// Should not error with no subscribers
	err := eb.Publish(context.Background(), event)

	require.NoError(t, err)
}

func TestSubscriberFunc(t *testing.T) {
	var called bool

	fn := SubscriberFunc(func(ctx context.Context, event Event) error {
		called = true
		return nil
	})

	event := Event{ID: "test"}
	err := fn.Handle(context.Background(), event)

	require.NoError(t, err)
	assert.True(t, called)
}

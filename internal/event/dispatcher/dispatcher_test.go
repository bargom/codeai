package dispatcher

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/event/bus"
	"github.com/bargom/codeai/internal/event/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger implements bus.Logger for testing.
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

// mockEventRepository implements repository.EventRepository for testing.
type mockEventRepository struct {
	mu         sync.Mutex
	events     []bus.Event
	saveError  error
	getError   error
	listEvents []bus.Event
}

func (m *mockEventRepository) SaveEvent(ctx context.Context, event bus.Event) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *mockEventRepository) GetEvent(ctx context.Context, eventID string) (*bus.Event, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.events {
		if e.ID == eventID {
			return &e, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockEventRepository) ListEvents(ctx context.Context, filter repository.EventFilter) ([]bus.Event, error) {
	return m.listEvents, nil
}

func (m *mockEventRepository) GetEventsByType(ctx context.Context, eventType bus.EventType, limit int) ([]bus.Event, error) {
	return nil, nil
}

func (m *mockEventRepository) GetEventsBySource(ctx context.Context, source string, limit int) ([]bus.Event, error) {
	return nil, nil
}

func (m *mockEventRepository) CountEvents(ctx context.Context, filter repository.EventFilter) (int64, error) {
	return 0, nil
}

func (m *mockEventRepository) DeleteOldEvents(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func (m *mockEventRepository) getSavedEvents() []bus.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]bus.Event, len(m.events))
	copy(result, m.events)
	return result
}

func TestNewDispatcher(t *testing.T) {
	logger := &mockLogger{}
	eb := bus.NewEventBus(logger)
	defer eb.Close()

	dispatcher := NewDispatcher(eb)

	assert.NotNil(t, dispatcher)
}

func TestDispatcher_DispatchWithoutPersistence(t *testing.T) {
	logger := &mockLogger{}
	eb := bus.NewEventBus(logger)
	defer eb.Close()

	dispatcher := NewDispatcher(eb, WithLogger(logger))

	var received atomic.Bool
	eb.Subscribe(bus.EventWorkflowStarted, bus.SubscriberFunc(func(ctx context.Context, event bus.Event) error {
		received.Store(true)
		return nil
	}))

	event := bus.Event{
		ID:        "dispatch-1",
		Type:      bus.EventWorkflowStarted,
		Source:    "test",
		Timestamp: time.Now(),
	}

	err := dispatcher.Dispatch(context.Background(), event)

	require.NoError(t, err)
	assert.True(t, received.Load())
}

func TestDispatcher_DispatchWithPersistence(t *testing.T) {
	logger := &mockLogger{}
	eb := bus.NewEventBus(logger)
	defer eb.Close()

	repo := &mockEventRepository{}
	dispatcher := NewDispatcher(eb, WithRepository(repo), WithLogger(logger))

	var received atomic.Bool
	eb.Subscribe(bus.EventJobCompleted, bus.SubscriberFunc(func(ctx context.Context, event bus.Event) error {
		received.Store(true)
		return nil
	}))

	event := bus.Event{
		ID:        "persist-1",
		Type:      bus.EventJobCompleted,
		Source:    "test",
		Timestamp: time.Now(),
	}

	err := dispatcher.Dispatch(context.Background(), event)

	require.NoError(t, err)
	assert.True(t, received.Load())

	// Verify event was persisted
	savedEvents := repo.getSavedEvents()
	require.Len(t, savedEvents, 1)
	assert.Equal(t, "persist-1", savedEvents[0].ID)
}

func TestDispatcher_DispatchPersistenceError(t *testing.T) {
	logger := &mockLogger{}
	eb := bus.NewEventBus(logger)
	defer eb.Close()

	repo := &mockEventRepository{
		saveError: errors.New("database error"),
	}
	dispatcher := NewDispatcher(eb, WithRepository(repo), WithLogger(logger))

	event := bus.Event{
		ID:   "error-1",
		Type: bus.EventWorkflowFailed,
	}

	err := dispatcher.Dispatch(context.Background(), event)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "persisting event")
}

func TestDispatcher_Subscribe(t *testing.T) {
	logger := &mockLogger{}
	eb := bus.NewEventBus(logger)
	defer eb.Close()

	dispatcher := NewDispatcher(eb)

	var received atomic.Bool
	dispatcher.Subscribe(bus.EventAgentExecuted, bus.SubscriberFunc(func(ctx context.Context, event bus.Event) error {
		received.Store(true)
		return nil
	}))

	event := bus.Event{
		ID:   "sub-1",
		Type: bus.EventAgentExecuted,
	}

	err := dispatcher.Dispatch(context.Background(), event)

	require.NoError(t, err)
	assert.True(t, received.Load())
}

func TestDispatcher_DispatchAsync(t *testing.T) {
	logger := &mockLogger{}
	eb := bus.NewEventBusWithConfig(logger, bus.Config{
		AsyncBufferSize: 100,
		WorkerPoolSize:  2,
	})

	repo := &mockEventRepository{}
	dispatcher := NewDispatcher(eb, WithRepository(repo))

	var wg sync.WaitGroup
	wg.Add(1)

	eb.Subscribe(bus.EventEmailSent, bus.SubscriberFunc(func(ctx context.Context, event bus.Event) error {
		wg.Done()
		return nil
	}))

	event := bus.Event{
		ID:   "async-dispatch-1",
		Type: bus.EventEmailSent,
	}

	dispatcher.DispatchAsync(context.Background(), event)

	wg.Wait()
	dispatcher.Close()

	// Verify event was persisted synchronously
	savedEvents := repo.getSavedEvents()
	require.Len(t, savedEvents, 1)
}

func TestDispatcher_Close(t *testing.T) {
	logger := &mockLogger{}
	eb := bus.NewEventBus(logger)
	dispatcher := NewDispatcher(eb)

	// Should not panic
	dispatcher.Close()
}

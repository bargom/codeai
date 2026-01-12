package handlers

import (
	"context"
	"sync"
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
	messages []logEntry
}

type logEntry struct {
	level string
	msg   string
}

func (m *mockLogger) Info(msg string, args ...any)  { m.log("INFO", msg) }
func (m *mockLogger) Error(msg string, args ...any) { m.log("ERROR", msg) }
func (m *mockLogger) Debug(msg string, args ...any) { m.log("DEBUG", msg) }
func (m *mockLogger) Warn(msg string, args ...any)  { m.log("WARN", msg) }

func (m *mockLogger) log(level, msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, logEntry{level: level, msg: msg})
}

func (m *mockLogger) getMessages() []logEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]logEntry, len(m.messages))
	copy(result, m.messages)
	return result
}

// mockEventRepository implements repository.EventRepository for testing.
type mockEventRepository struct {
	events []bus.Event
}

func (m *mockEventRepository) SaveEvent(ctx context.Context, event bus.Event) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockEventRepository) GetEvent(ctx context.Context, eventID string) (*bus.Event, error) {
	return nil, nil
}

func (m *mockEventRepository) ListEvents(ctx context.Context, filter repository.EventFilter) ([]bus.Event, error) {
	return nil, nil
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

func TestWorkflowEventHandler_Handle(t *testing.T) {
	logger := &mockLogger{}
	repo := &mockEventRepository{}
	handler := NewWorkflowEventHandler(repo, logger)

	tests := []struct {
		name      string
		event     bus.Event
		expectLog string
	}{
		{
			name: "workflow started",
			event: bus.Event{
				ID:   "wf-1",
				Type: bus.EventWorkflowStarted,
				Data: map[string]interface{}{"workflowID": "wf-123"},
			},
			expectLog: "workflow started",
		},
		{
			name: "workflow completed",
			event: bus.Event{
				ID:   "wf-2",
				Type: bus.EventWorkflowCompleted,
				Data: map[string]interface{}{"workflowID": "wf-456", "duration": 10.5},
			},
			expectLog: "workflow completed",
		},
		{
			name: "workflow failed",
			event: bus.Event{
				ID:   "wf-3",
				Type: bus.EventWorkflowFailed,
				Data: map[string]interface{}{"workflowID": "wf-789", "error": "timeout"},
			},
			expectLog: "workflow failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger.messages = nil // reset
			err := handler.Handle(context.Background(), tt.event)

			require.NoError(t, err)
			messages := logger.getMessages()
			require.NotEmpty(t, messages)
			assert.Equal(t, tt.expectLog, messages[0].msg)
		})
	}
}

func TestWorkflowEventHandler_SupportedEventTypes(t *testing.T) {
	handler := NewWorkflowEventHandler(nil, nil)

	types := handler.SupportedEventTypes()

	assert.Contains(t, types, bus.EventWorkflowStarted)
	assert.Contains(t, types, bus.EventWorkflowCompleted)
	assert.Contains(t, types, bus.EventWorkflowFailed)
}

func TestJobEventHandler_Handle(t *testing.T) {
	logger := &mockLogger{}
	repo := &mockEventRepository{}
	handler := NewJobEventHandler(repo, logger)

	tests := []struct {
		name      string
		event     bus.Event
		expectLog string
	}{
		{
			name: "job enqueued",
			event: bus.Event{
				ID:   "j-1",
				Type: bus.EventJobEnqueued,
				Data: map[string]interface{}{"jobID": "job-123", "jobType": "email"},
			},
			expectLog: "job enqueued",
		},
		{
			name: "job started",
			event: bus.Event{
				ID:   "j-2",
				Type: bus.EventJobStarted,
				Data: map[string]interface{}{"jobID": "job-456", "workerID": "worker-1"},
			},
			expectLog: "job started",
		},
		{
			name: "job completed",
			event: bus.Event{
				ID:   "j-3",
				Type: bus.EventJobCompleted,
				Data: map[string]interface{}{"jobID": "job-789", "duration": 5.0},
			},
			expectLog: "job completed",
		},
		{
			name: "job failed",
			event: bus.Event{
				ID:   "j-4",
				Type: bus.EventJobFailed,
				Data: map[string]interface{}{"jobID": "job-999", "error": "connection refused", "attempts": 3.0},
			},
			expectLog: "job failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger.messages = nil // reset
			err := handler.Handle(context.Background(), tt.event)

			require.NoError(t, err)
			messages := logger.getMessages()
			require.NotEmpty(t, messages)
			assert.Equal(t, tt.expectLog, messages[0].msg)
		})
	}
}

func TestJobEventHandler_SupportedEventTypes(t *testing.T) {
	handler := NewJobEventHandler(nil, nil)

	types := handler.SupportedEventTypes()

	assert.Contains(t, types, bus.EventJobEnqueued)
	assert.Contains(t, types, bus.EventJobStarted)
	assert.Contains(t, types, bus.EventJobCompleted)
	assert.Contains(t, types, bus.EventJobFailed)
}

func TestWorkflowEventHandler_UnknownEventType(t *testing.T) {
	logger := &mockLogger{}
	handler := NewWorkflowEventHandler(nil, logger)

	event := bus.Event{
		ID:   "unknown-1",
		Type: bus.EventType("unknown.event"),
	}

	err := handler.Handle(context.Background(), event)

	require.NoError(t, err)
	// No log should be produced for unknown events
	messages := logger.getMessages()
	assert.Empty(t, messages)
}

package subscribers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/event/bus"
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

func TestLoggingSubscriber_Handle(t *testing.T) {
	logger := &mockLogger{}
	subscriber := NewLoggingSubscriber(logger)

	event := bus.Event{
		ID:        "log-test-1",
		Type:      bus.EventWorkflowStarted,
		Source:    "test-source",
		Timestamp: time.Now(),
	}

	err := subscriber.Handle(context.Background(), event)

	require.NoError(t, err)
	messages := logger.getMessages()
	require.Len(t, messages, 1)
	assert.Equal(t, "INFO", messages[0].level)
	assert.Equal(t, "event received", messages[0].msg)
}

func TestMetricsSubscriber_Handle(t *testing.T) {
	subscriber := NewMetricsSubscriber()

	events := []bus.Event{
		{ID: "1", Type: bus.EventWorkflowStarted, Source: "source-a"},
		{ID: "2", Type: bus.EventWorkflowStarted, Source: "source-a"},
		{ID: "3", Type: bus.EventJobCompleted, Source: "source-b"},
		{ID: "4", Type: bus.EventWorkflowStarted, Source: "source-b"},
	}

	for _, event := range events {
		err := subscriber.Handle(context.Background(), event)
		require.NoError(t, err)
	}

	assert.Equal(t, int64(4), subscriber.GetTotalCount())
	assert.Equal(t, int64(3), subscriber.GetTypeCount(bus.EventWorkflowStarted))
	assert.Equal(t, int64(1), subscriber.GetTypeCount(bus.EventJobCompleted))
	assert.Equal(t, int64(2), subscriber.GetSourceCount("source-a"))
	assert.Equal(t, int64(2), subscriber.GetSourceCount("source-b"))
}

func TestMetricsSubscriber_GetStats(t *testing.T) {
	subscriber := NewMetricsSubscriber()

	events := []bus.Event{
		{ID: "1", Type: bus.EventWorkflowStarted, Source: "source-a"},
		{ID: "2", Type: bus.EventJobCompleted, Source: "source-b"},
	}

	for _, event := range events {
		_ = subscriber.Handle(context.Background(), event)
	}

	stats := subscriber.GetStats()

	assert.Equal(t, int64(2), stats["total"])

	byType, ok := stats["by_type"].(map[string]int64)
	require.True(t, ok)
	assert.Equal(t, int64(1), byType[string(bus.EventWorkflowStarted)])
	assert.Equal(t, int64(1), byType[string(bus.EventJobCompleted)])

	bySource, ok := stats["by_source"].(map[string]int64)
	require.True(t, ok)
	assert.Equal(t, int64(1), bySource["source-a"])
	assert.Equal(t, int64(1), bySource["source-b"])
}

func TestMetricsSubscriber_Reset(t *testing.T) {
	subscriber := NewMetricsSubscriber()

	event := bus.Event{ID: "1", Type: bus.EventWorkflowStarted, Source: "source-a"}
	_ = subscriber.Handle(context.Background(), event)

	assert.Equal(t, int64(1), subscriber.GetTotalCount())

	subscriber.Reset()

	assert.Equal(t, int64(0), subscriber.GetTotalCount())
	assert.Equal(t, int64(0), subscriber.GetTypeCount(bus.EventWorkflowStarted))
	assert.Equal(t, int64(0), subscriber.GetSourceCount("source-a"))
}

func TestMetricsSubscriber_Concurrent(t *testing.T) {
	subscriber := NewMetricsSubscriber()

	var wg sync.WaitGroup
	eventCount := 100

	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := bus.Event{
				ID:     "concurrent-" + string(rune('0'+id%10)),
				Type:   bus.EventJobStarted,
				Source: "concurrent-source",
			}
			_ = subscriber.Handle(context.Background(), event)
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int64(eventCount), subscriber.GetTotalCount())
}

func TestWebhookSubscriber_ShouldHandle(t *testing.T) {
	logger := &mockLogger{}
	subscriber := NewWebhookSubscriber(nil, logger)

	tests := []struct {
		name       string
		types      []bus.EventType
		eventType  bus.EventType
		shouldFind bool
	}{
		{
			name:       "type in list",
			types:      []bus.EventType{bus.EventWorkflowStarted, bus.EventWorkflowCompleted},
			eventType:  bus.EventWorkflowStarted,
			shouldFind: true,
		},
		{
			name:       "type not in list",
			types:      []bus.EventType{bus.EventJobStarted},
			eventType:  bus.EventWorkflowStarted,
			shouldFind: false,
		},
		{
			name:       "empty list",
			types:      []bus.EventType{},
			eventType:  bus.EventWorkflowStarted,
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := subscriber.shouldHandle(tt.types, tt.eventType)
			assert.Equal(t, tt.shouldFind, result)
		})
	}
}

func TestWebhookSubscriber_AddWebhook(t *testing.T) {
	logger := &mockLogger{}
	subscriber := NewWebhookSubscriber(nil, logger)

	assert.Len(t, subscriber.webhooks, 0)

	subscriber.AddWebhook(WebhookConfig{
		URL:    "https://example.com/webhook",
		Secret: "secret",
	})

	assert.Len(t, subscriber.webhooks, 1)
	assert.Equal(t, "https://example.com/webhook", subscriber.webhooks[0].URL)
}

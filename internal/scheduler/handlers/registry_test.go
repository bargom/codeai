package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bargom/codeai/internal/scheduler/tasks"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	// Verify default handlers are registered
	assert.True(t, registry.Has(tasks.TypeAIAgentExecution))
	assert.True(t, registry.Has(tasks.TypeTestSuiteRun))
	assert.True(t, registry.Has(tasks.TypeDataProcessing))
	assert.True(t, registry.Has(tasks.TypeCleanup))
	assert.True(t, registry.Has(tasks.TypeWebhook))
}

func TestNewEmptyRegistry(t *testing.T) {
	registry := NewEmptyRegistry()

	// Verify no handlers are registered
	assert.False(t, registry.Has(tasks.TypeAIAgentExecution))
	assert.Equal(t, 0, len(registry.ListTypes()))
}

func TestRegistry_Register(t *testing.T) {
	registry := NewEmptyRegistry()

	handler := func(ctx context.Context, t *asynq.Task) error {
		return nil
	}

	registry.Register("custom:task", handler)

	assert.True(t, registry.Has("custom:task"))

	retrieved, ok := registry.Get("custom:task")
	assert.True(t, ok)
	assert.NotNil(t, retrieved)
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()

	// Verify handler exists
	assert.True(t, registry.Has(tasks.TypeAIAgentExecution))

	// Unregister
	registry.Unregister(tasks.TypeAIAgentExecution)

	// Verify handler no longer exists
	assert.False(t, registry.Has(tasks.TypeAIAgentExecution))
}

func TestRegistry_ListTypes(t *testing.T) {
	registry := NewEmptyRegistry()

	registry.Register("task:a", func(ctx context.Context, t *asynq.Task) error { return nil })
	registry.Register("task:b", func(ctx context.Context, t *asynq.Task) error { return nil })
	registry.Register("task:c", func(ctx context.Context, t *asynq.Task) error { return nil })

	types := registry.ListTypes()
	assert.Len(t, types, 3)
	assert.Contains(t, types, "task:a")
	assert.Contains(t, types, "task:b")
	assert.Contains(t, types, "task:c")
}

func TestRegistry_GetMux(t *testing.T) {
	registry := NewEmptyRegistry()

	registry.Register("task:a", func(ctx context.Context, t *asynq.Task) error { return nil })

	mux := registry.GetMux()
	assert.NotNil(t, mux)
}

func TestLoggingMiddleware(t *testing.T) {
	var logs []string
	logFn := func(format string, args ...any) {
		logs = append(logs, format)
	}

	middleware := LoggingMiddleware(logFn)

	handler := func(ctx context.Context, t *asynq.Task) error {
		return nil
	}

	wrappedHandler := middleware(handler)

	task := asynq.NewTask("test:task", nil)
	err := wrappedHandler(context.Background(), task)

	require.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Contains(t, logs[0], "Starting task")
	assert.Contains(t, logs[1], "Task completed")
}

func TestLoggingMiddleware_Error(t *testing.T) {
	var logs []string
	logFn := func(format string, args ...any) {
		logs = append(logs, format)
	}

	middleware := LoggingMiddleware(logFn)

	handler := func(ctx context.Context, t *asynq.Task) error {
		return errors.New("test error")
	}

	wrappedHandler := middleware(handler)

	task := asynq.NewTask("test:task", nil)
	err := wrappedHandler(context.Background(), task)

	assert.Error(t, err)
	assert.Len(t, logs, 2)
	assert.Contains(t, logs[0], "Starting task")
	assert.Contains(t, logs[1], "Task failed")
}

func TestRecoveryMiddleware(t *testing.T) {
	var logs []string
	logFn := func(format string, args ...any) {
		logs = append(logs, format)
	}

	middleware := RecoveryMiddleware(logFn)

	handler := func(ctx context.Context, t *asynq.Task) error {
		panic("test panic")
	}

	wrappedHandler := middleware(handler)

	task := asynq.NewTask("test:task", nil)
	err := wrappedHandler(context.Background(), task)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task panicked")
	assert.Len(t, logs, 1)
	assert.Contains(t, logs[0], "Task panicked")
}

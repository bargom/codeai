// Package handlers provides the task handler registry for the scheduler.
package handlers

import (
	"context"
	"fmt"
	"sync"

	"github.com/hibiken/asynq"

	"github.com/bargom/codeai/internal/scheduler/tasks"
)

// HandlerFunc is a function that handles an Asynq task.
type HandlerFunc func(context.Context, *asynq.Task) error

// Registry manages task handlers.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]HandlerFunc
}

// NewRegistry creates a new handler registry with default handlers.
func NewRegistry() *Registry {
	r := &Registry{
		handlers: make(map[string]HandlerFunc),
	}

	// Register all default task handlers
	r.registerDefaultHandlers()

	return r
}

// NewEmptyRegistry creates an empty handler registry.
func NewEmptyRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]HandlerFunc),
	}
}

// registerDefaultHandlers registers all built-in task handlers.
func (r *Registry) registerDefaultHandlers() {
	// AI-related tasks
	r.Register(tasks.TypeAIAgentExecution, tasks.HandleAIAgentTask)
	r.Register(tasks.TypeTestSuiteRun, tasks.HandleTestSuiteTask)
	r.Register(tasks.TypeDataProcessing, tasks.HandleDataProcessingTask)

	// System tasks
	r.Register(tasks.TypeCleanup, tasks.HandleCleanupTask)
	r.Register(tasks.TypeWebhook, tasks.HandleWebhookTask)
}

// Register adds a handler for the given task type.
func (r *Registry) Register(taskType string, handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[taskType] = handler
}

// Unregister removes a handler for the given task type.
func (r *Registry) Unregister(taskType string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, taskType)
}

// Get returns the handler for the given task type.
func (r *Registry) Get(taskType string) (HandlerFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.handlers[taskType]
	return handler, ok
}

// Has checks if a handler is registered for the given task type.
func (r *Registry) Has(taskType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.handlers[taskType]
	return ok
}

// ListTypes returns all registered task types.
func (r *Registry) ListTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t)
	}
	return types
}

// GetMux returns an Asynq ServeMux configured with all registered handlers.
func (r *Registry) GetMux() *asynq.ServeMux {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mux := asynq.NewServeMux()
	for taskType, handler := range r.handlers {
		mux.HandleFunc(taskType, handler)
	}
	return mux
}

// MiddlewareFunc is middleware for task handlers.
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// WithMiddleware wraps all handlers with the given middleware.
func (r *Registry) WithMiddleware(middleware ...MiddlewareFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for taskType, handler := range r.handlers {
		wrapped := handler
		for i := len(middleware) - 1; i >= 0; i-- {
			wrapped = middleware[i](wrapped)
		}
		r.handlers[taskType] = wrapped
	}
}

// LoggingMiddleware creates a middleware that logs task execution.
func LoggingMiddleware(logFn func(format string, args ...any)) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, t *asynq.Task) error {
			logFn("Starting task: type=%s", t.Type())
			err := next(ctx, t)
			if err != nil {
				logFn("Task failed: type=%s error=%v", t.Type(), err)
			} else {
				logFn("Task completed: type=%s", t.Type())
			}
			return err
		}
	}
}

// RecoveryMiddleware creates a middleware that recovers from panics.
func RecoveryMiddleware(logFn func(format string, args ...any)) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, t *asynq.Task) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logFn("Task panicked: type=%s panic=%v", t.Type(), r)
					err = fmt.Errorf("task panicked: %v", r)
				}
			}()
			return next(ctx, t)
		}
	}
}

// TimeoutMiddleware creates a middleware that enforces task timeout.
// Note: Asynq already handles timeouts, this is for additional logging.
func TimeoutMiddleware(logFn func(format string, args ...any)) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, t *asynq.Task) error {
			select {
			case <-ctx.Done():
				logFn("Task timed out: type=%s", t.Type())
				return ctx.Err()
			default:
				return next(ctx, t)
			}
		}
	}
}

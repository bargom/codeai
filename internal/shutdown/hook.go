package shutdown

import (
	"context"
)

// Standard priorities for shutdown hooks (higher = earlier execution).
const (
	// PriorityHTTPServer is used for HTTP server shutdown (stop accepting new connections).
	PriorityHTTPServer = 90

	// PriorityBackgroundWorkers is used for background worker shutdown.
	PriorityBackgroundWorkers = 80

	// PriorityDatabase is used for database connection shutdown.
	PriorityDatabase = 70

	// PriorityCache is used for cache connection shutdown.
	PriorityCache = 60

	// PriorityMetrics is used for metrics/logging flush.
	PriorityMetrics = 50
)

// HookFunc is a function that performs shutdown logic.
// It receives a context that will be canceled when the hook timeout expires.
type HookFunc func(ctx context.Context) error

// Hook represents a shutdown hook with metadata.
type Hook struct {
	// Name identifies the hook for logging purposes.
	Name string

	// Priority determines execution order. Higher priorities execute first.
	// Use the Priority* constants for standard components.
	Priority int

	// Fn is the shutdown function to execute.
	Fn HookFunc
}

// HookResult contains the result of executing a shutdown hook.
type HookResult struct {
	Name     string
	Priority int
	Error    error
	Duration interface{} // time.Duration, but stored as interface for flexibility
}

// Registry manages shutdown hooks.
type Registry struct {
	hooks []Hook
}

// NewRegistry creates a new hook registry.
func NewRegistry() *Registry {
	return &Registry{
		hooks: make([]Hook, 0),
	}
}

// Register adds a shutdown hook to the registry.
func (r *Registry) Register(name string, priority int, fn HookFunc) {
	r.hooks = append(r.hooks, Hook{
		Name:     name,
		Priority: priority,
		Fn:       fn,
	})
}

// RegisterHook adds a Hook struct to the registry.
func (r *Registry) RegisterHook(hook Hook) {
	r.hooks = append(r.hooks, hook)
}

// Hooks returns all registered hooks.
func (r *Registry) Hooks() []Hook {
	result := make([]Hook, len(r.hooks))
	copy(result, r.hooks)
	return result
}

// Clear removes all registered hooks.
func (r *Registry) Clear() {
	r.hooks = r.hooks[:0]
}

// Count returns the number of registered hooks.
func (r *Registry) Count() int {
	return len(r.hooks)
}

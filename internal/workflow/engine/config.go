// Package engine provides the core workflow engine implementation.
package engine

import "time"

// Config holds the workflow engine configuration.
type Config struct {
	// TemporalHostPort is the Temporal server address.
	TemporalHostPort string
	// Namespace is the Temporal namespace.
	Namespace string
	// TaskQueue is the task queue name for workflows and activities.
	TaskQueue string
	// MaxConcurrentWorkflows is the maximum number of concurrent workflow executions.
	MaxConcurrentWorkflows int
	// MaxConcurrentActivities is the maximum number of concurrent activity executions.
	MaxConcurrentActivities int
	// DefaultTimeout is the default workflow execution timeout.
	DefaultTimeout time.Duration
	// WorkerID is the unique identifier for this worker.
	WorkerID string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		TemporalHostPort:        "localhost:7233",
		Namespace:               "default",
		TaskQueue:               "codeai-workflows",
		MaxConcurrentWorkflows:  100,
		MaxConcurrentActivities: 100,
		DefaultTimeout:          30 * time.Minute,
		WorkerID:                "codeai-worker",
	}
}

// Validate validates the configuration.
func (c Config) Validate() error {
	if c.TemporalHostPort == "" {
		return ErrConfigInvalid{Field: "TemporalHostPort", Reason: "cannot be empty"}
	}
	if c.Namespace == "" {
		return ErrConfigInvalid{Field: "Namespace", Reason: "cannot be empty"}
	}
	if c.TaskQueue == "" {
		return ErrConfigInvalid{Field: "TaskQueue", Reason: "cannot be empty"}
	}
	if c.MaxConcurrentWorkflows <= 0 {
		return ErrConfigInvalid{Field: "MaxConcurrentWorkflows", Reason: "must be positive"}
	}
	if c.MaxConcurrentActivities <= 0 {
		return ErrConfigInvalid{Field: "MaxConcurrentActivities", Reason: "must be positive"}
	}
	return nil
}

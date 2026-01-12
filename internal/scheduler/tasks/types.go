// Package tasks defines task types and handlers for the scheduler.
package tasks

import (
	"time"
)

// Task type constants.
const (
	// AI-related tasks
	TypeAIAgentExecution = "ai:agent:execute"
	TypeTestSuiteRun     = "ai:test:run"
	TypeDataProcessing   = "ai:data:process"

	// System tasks
	TypeCleanup  = "system:cleanup"
	TypeWebhook  = "system:webhook"
	TypeNotify   = "system:notify"

	// Code analysis tasks
	TypeCodeParse    = "code:parse"
	TypeCodeValidate = "code:validate"
	TypeCodeDeploy   = "code:deploy"
)

// RetryPolicy defines retry behavior for tasks.
type RetryPolicy struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// DefaultRetryPolicy returns a default exponential backoff retry policy.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:   3,
		InitialDelay: 10 * time.Second,
		MaxDelay:     10 * time.Minute,
		Multiplier:   2.0,
	}
}

// TaskResult represents the result of a task execution.
type TaskResult struct {
	Success bool
	Data    any
	Error   string
}

// NewSuccessResult creates a successful task result.
func NewSuccessResult(data any) TaskResult {
	return TaskResult{
		Success: true,
		Data:    data,
	}
}

// NewErrorResult creates a failed task result.
func NewErrorResult(err error) TaskResult {
	return TaskResult{
		Success: false,
		Error:   err.Error(),
	}
}

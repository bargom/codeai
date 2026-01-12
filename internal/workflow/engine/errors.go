package engine

import (
	"errors"
	"fmt"
)

var (
	// ErrEngineNotStarted is returned when engine operations are called before Start.
	ErrEngineNotStarted = errors.New("workflow engine not started")
	// ErrEngineAlreadyStarted is returned when Start is called on a running engine.
	ErrEngineAlreadyStarted = errors.New("workflow engine already started")
	// ErrWorkflowNotFound is returned when a workflow execution is not found.
	ErrWorkflowNotFound = errors.New("workflow not found")
)

// ErrConfigInvalid is returned when configuration validation fails.
type ErrConfigInvalid struct {
	Field  string
	Reason string
}

func (e ErrConfigInvalid) Error() string {
	return fmt.Sprintf("invalid config: %s %s", e.Field, e.Reason)
}

// ErrWorkflowFailed is returned when a workflow execution fails.
type ErrWorkflowFailed struct {
	WorkflowID string
	RunID      string
	Cause      error
}

func (e ErrWorkflowFailed) Error() string {
	return fmt.Sprintf("workflow %s (run: %s) failed: %v", e.WorkflowID, e.RunID, e.Cause)
}

func (e ErrWorkflowFailed) Unwrap() error {
	return e.Cause
}

// ErrActivityFailed is returned when an activity execution fails.
type ErrActivityFailed struct {
	ActivityType string
	Cause        error
}

func (e ErrActivityFailed) Error() string {
	return fmt.Sprintf("activity %s failed: %v", e.ActivityType, e.Cause)
}

func (e ErrActivityFailed) Unwrap() error {
	return e.Cause
}

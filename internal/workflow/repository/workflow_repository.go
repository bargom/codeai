// Package repository provides data persistence for workflow executions.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// ErrNotFound is returned when a workflow execution is not found.
var ErrNotFound = errors.New("workflow execution not found")

// Status represents the status of a workflow execution.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

// WorkflowExecution represents a workflow execution record.
type WorkflowExecution struct {
	ID            string          `json:"id"`
	WorkflowID    string          `json:"workflowId"`
	WorkflowType  string          `json:"workflowType"`
	RunID         string          `json:"runId,omitempty"`
	Status        Status          `json:"status"`
	Input         json.RawMessage `json:"input,omitempty"`
	Output        json.RawMessage `json:"output,omitempty"`
	Error         string          `json:"error,omitempty"`
	StartedAt     time.Time       `json:"startedAt"`
	CompletedAt   *time.Time      `json:"completedAt,omitempty"`
	Compensations []CompensationRecord `json:"compensations,omitempty"`
	Metadata      map[string]string   `json:"metadata,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

// CompensationRecord tracks a compensation action.
type CompensationRecord struct {
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Error      string    `json:"error,omitempty"`
	ExecutedAt time.Time `json:"executedAt,omitempty"`
}

// Filter defines filtering options for listing workflow executions.
type Filter struct {
	WorkflowType string
	Status       Status
	StartedAfter *time.Time
	StartedBefore *time.Time
	Limit        int
	Offset       int
}

// WorkflowRepository defines the interface for workflow execution persistence.
type WorkflowRepository interface {
	// SaveExecution saves a new workflow execution record.
	SaveExecution(ctx context.Context, exec *WorkflowExecution) error

	// GetExecution retrieves a workflow execution by its ID.
	GetExecution(ctx context.Context, id string) (*WorkflowExecution, error)

	// GetExecutionByWorkflowID retrieves a workflow execution by workflow ID.
	GetExecutionByWorkflowID(ctx context.Context, workflowID string) (*WorkflowExecution, error)

	// ListExecutions lists workflow executions with optional filtering.
	ListExecutions(ctx context.Context, filter Filter) ([]WorkflowExecution, error)

	// UpdateStatus updates the status of a workflow execution.
	UpdateStatus(ctx context.Context, id string, status Status, errorMsg string) error

	// UpdateOutput updates the output of a workflow execution.
	UpdateOutput(ctx context.Context, id string, output json.RawMessage) error

	// UpdateCompensations updates the compensation records of a workflow execution.
	UpdateCompensations(ctx context.Context, id string, compensations []CompensationRecord) error

	// DeleteExecution deletes a workflow execution by its ID.
	DeleteExecution(ctx context.Context, id string) error

	// CountByStatus counts workflow executions by status.
	CountByStatus(ctx context.Context, status Status) (int64, error)
}

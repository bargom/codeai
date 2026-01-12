// Package repository provides data access for compensation records.
package repository

import (
	"context"
	"encoding/json"
	"time"
)

// CompensationStatus represents the status of a compensation operation.
type CompensationStatus string

const (
	StatusPending   CompensationStatus = "pending"
	StatusRunning   CompensationStatus = "running"
	StatusCompleted CompensationStatus = "completed"
	StatusFailed    CompensationStatus = "failed"
	StatusSkipped   CompensationStatus = "skipped"
)

// CompensationRecord tracks a compensation operation and its state.
type CompensationRecord struct {
	ID               string            `json:"id" bson:"_id"`
	WorkflowID       string            `json:"workflowId" bson:"workflowId"`
	RunID            string            `json:"runId,omitempty" bson:"runId,omitempty"`
	ActivityName     string            `json:"activityName" bson:"activityName"`
	Status           CompensationStatus `json:"status" bson:"status"`
	Error            string            `json:"error,omitempty" bson:"error,omitempty"`
	StartedAt        time.Time         `json:"startedAt,omitempty" bson:"startedAt,omitempty"`
	CompletedAt      time.Time         `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
	Duration         time.Duration     `json:"duration,omitempty" bson:"duration,omitempty"`
	Retries          int               `json:"retries" bson:"retries"`
	CompensationData json.RawMessage   `json:"compensationData,omitempty" bson:"compensationData,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty" bson:"metadata,omitempty"`
	CreatedAt        time.Time         `json:"createdAt" bson:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt" bson:"updatedAt"`
}

// CompensationSummary provides an aggregated view of compensation history.
type CompensationSummary struct {
	WorkflowID       string `json:"workflowId"`
	TotalCompensations int   `json:"totalCompensations"`
	Completed        int    `json:"completed"`
	Failed           int    `json:"failed"`
	Skipped          int    `json:"skipped"`
	Pending          int    `json:"pending"`
}

// ListCompensationsFilter provides filtering options for listing compensations.
type ListCompensationsFilter struct {
	WorkflowID   string
	ActivityName string
	Status       CompensationStatus
	StartTime    time.Time
	EndTime      time.Time
	Limit        int
	Offset       int
}

// CompensationRepository defines the interface for compensation data access.
type CompensationRepository interface {
	// SaveCompensationRecord creates or updates a compensation record.
	SaveCompensationRecord(ctx context.Context, record *CompensationRecord) error

	// GetCompensationRecord retrieves a compensation record by ID.
	GetCompensationRecord(ctx context.Context, recordID string) (*CompensationRecord, error)

	// GetCompensationHistory retrieves all compensation records for a workflow.
	GetCompensationHistory(ctx context.Context, workflowID string) ([]CompensationRecord, error)

	// ListCompensations retrieves compensation records with filtering.
	ListCompensations(ctx context.Context, filter ListCompensationsFilter) ([]CompensationRecord, error)

	// MarkCompensationStarted updates record status to running.
	MarkCompensationStarted(ctx context.Context, recordID string) error

	// MarkCompensationCompleted updates record status to completed.
	MarkCompensationCompleted(ctx context.Context, recordID string) error

	// MarkCompensationFailed updates record status to failed with error.
	MarkCompensationFailed(ctx context.Context, recordID string, err error) error

	// GetCompensationSummary retrieves aggregated compensation statistics.
	GetCompensationSummary(ctx context.Context, workflowID string) (*CompensationSummary, error)

	// DeleteCompensationHistory removes all compensation records for a workflow.
	DeleteCompensationHistory(ctx context.Context, workflowID string) error

	// GetPendingCompensations retrieves all pending compensations for retry.
	GetPendingCompensations(ctx context.Context, limit int) ([]CompensationRecord, error)

	// IncrementRetryCount increments the retry counter for a compensation record.
	IncrementRetryCount(ctx context.Context, recordID string) error
}

// NewCompensationRecord creates a new compensation record with defaults.
func NewCompensationRecord(workflowID, runID, activityName string) *CompensationRecord {
	now := time.Now().UTC()
	return &CompensationRecord{
		ID:           generateID(),
		WorkflowID:   workflowID,
		RunID:        runID,
		ActivityName: activityName,
		Status:       StatusPending,
		Retries:      0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// generateID generates a unique ID for a compensation record.
func generateID() string {
	return time.Now().UTC().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of the specified length.
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[i%len(charset)]
	}
	return string(result)
}

// MarkComplete sets the record status to completed.
func (r *CompensationRecord) MarkComplete() {
	r.Status = StatusCompleted
	r.CompletedAt = time.Now().UTC()
	r.UpdatedAt = time.Now().UTC()
	if !r.StartedAt.IsZero() {
		r.Duration = r.CompletedAt.Sub(r.StartedAt)
	}
}

// MarkFailed sets the record status to failed with an error.
func (r *CompensationRecord) MarkFailed(err error) {
	r.Status = StatusFailed
	r.Error = err.Error()
	r.CompletedAt = time.Now().UTC()
	r.UpdatedAt = time.Now().UTC()
	if !r.StartedAt.IsZero() {
		r.Duration = r.CompletedAt.Sub(r.StartedAt)
	}
}

// MarkStarted sets the record status to running.
func (r *CompensationRecord) MarkStarted() {
	r.Status = StatusRunning
	r.StartedAt = time.Now().UTC()
	r.UpdatedAt = time.Now().UTC()
}

// MarkSkipped sets the record status to skipped.
func (r *CompensationRecord) MarkSkipped() {
	r.Status = StatusSkipped
	r.UpdatedAt = time.Now().UTC()
}

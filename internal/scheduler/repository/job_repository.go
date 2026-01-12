// Package repository provides data access for scheduler jobs.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Common errors.
var (
	ErrJobNotFound = errors.New("job not found")
)

// JobStatus represents the status of a job.
type JobStatus string

// Job status constants.
const (
	JobStatusPending   JobStatus = "pending"
	JobStatusScheduled JobStatus = "scheduled"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
	JobStatusRetrying  JobStatus = "retrying"
)

// Job represents a scheduled job in the system.
type Job struct {
	ID             string          `json:"id"`
	TaskType       string          `json:"task_type"`
	Payload        json.RawMessage `json:"payload"`
	Status         JobStatus       `json:"status"`
	Queue          string          `json:"queue"`
	ScheduledAt    *time.Time      `json:"scheduled_at,omitempty"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	FailedAt       *time.Time      `json:"failed_at,omitempty"`
	RetryCount     int             `json:"retry_count"`
	MaxRetries     int             `json:"max_retries"`
	Error          string          `json:"error,omitempty"`
	Result         json.RawMessage `json:"result,omitempty"`
	CronExpression string          `json:"cron_expression,omitempty"`
	CronEntryID    string          `json:"cron_entry_id,omitempty"`
	Timeout        time.Duration   `json:"timeout"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Metadata       map[string]any  `json:"metadata,omitempty"`
}

// JobFilter contains filter options for listing jobs.
type JobFilter struct {
	Status           []JobStatus
	TaskTypes        []string
	Queue            string
	ScheduledBefore  *time.Time
	ScheduledAfter   *time.Time
	CreatedBefore    *time.Time
	CreatedAfter     *time.Time
	WithCron         *bool
	Limit            int
	Offset           int
	OrderBy          string
	OrderDirection   string
}

// JobRepository defines the interface for job persistence.
type JobRepository interface {
	// CreateJob creates a new job record.
	CreateJob(ctx context.Context, job *Job) error

	// GetJob retrieves a job by ID.
	GetJob(ctx context.Context, jobID string) (*Job, error)

	// UpdateJob updates an existing job.
	UpdateJob(ctx context.Context, job *Job) error

	// UpdateJobStatus updates only the status and related timestamps.
	UpdateJobStatus(ctx context.Context, jobID string, status JobStatus, err error) error

	// DeleteJob removes a job.
	DeleteJob(ctx context.Context, jobID string) error

	// ListJobs lists jobs based on filter criteria.
	ListJobs(ctx context.Context, filter JobFilter) ([]Job, error)

	// CountJobs counts jobs based on filter criteria.
	CountJobs(ctx context.Context, filter JobFilter) (int64, error)

	// GetJobsByStatus retrieves jobs by status with a limit.
	GetJobsByStatus(ctx context.Context, status JobStatus, limit int) ([]Job, error)

	// GetPendingJobs retrieves all pending jobs ready to be executed.
	GetPendingJobs(ctx context.Context, limit int) ([]Job, error)

	// GetRecurringJobs retrieves all jobs with cron expressions.
	GetRecurringJobs(ctx context.Context) ([]Job, error)

	// SetJobResult stores the result of a job execution.
	SetJobResult(ctx context.Context, jobID string, result json.RawMessage) error

	// IncrementRetryCount increments the retry count for a job.
	IncrementRetryCount(ctx context.Context, jobID string) error
}

// DefaultFilter returns a JobFilter with sensible defaults.
func DefaultFilter() JobFilter {
	return JobFilter{
		Limit:          100,
		Offset:         0,
		OrderBy:        "created_at",
		OrderDirection: "DESC",
	}
}

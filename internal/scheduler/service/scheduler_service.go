// Package service provides the main scheduler service for job management.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/bargom/codeai/internal/event"
	"github.com/bargom/codeai/internal/scheduler/queue"
	"github.com/bargom/codeai/internal/scheduler/repository"
)

// JobRequest represents a request to create a new job.
type JobRequest struct {
	TaskType   string          `json:"task_type"`
	Payload    any             `json:"payload"`
	Queue      string          `json:"queue,omitempty"`
	MaxRetries int             `json:"max_retries,omitempty"`
	Timeout    time.Duration   `json:"timeout,omitempty"`
	Metadata   map[string]any  `json:"metadata,omitempty"`
}

// JobStatusResponse represents the status response for a job.
type JobStatusResponse struct {
	ID             string                `json:"id"`
	TaskType       string                `json:"task_type"`
	Status         repository.JobStatus  `json:"status"`
	Queue          string                `json:"queue"`
	ScheduledAt    *time.Time            `json:"scheduled_at,omitempty"`
	StartedAt      *time.Time            `json:"started_at,omitempty"`
	CompletedAt    *time.Time            `json:"completed_at,omitempty"`
	FailedAt       *time.Time            `json:"failed_at,omitempty"`
	RetryCount     int                   `json:"retry_count"`
	MaxRetries     int                   `json:"max_retries"`
	Error          string                `json:"error,omitempty"`
	Result         json.RawMessage       `json:"result,omitempty"`
	CronExpression string                `json:"cron_expression,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

// SchedulerService manages job scheduling and execution.
type SchedulerService struct {
	queueManager *queue.Manager
	repository   repository.JobRepository
	eventBus     event.Dispatcher
}

// NewSchedulerService creates a new scheduler service.
func NewSchedulerService(
	qm *queue.Manager,
	repo repository.JobRepository,
	eb event.Dispatcher,
) *SchedulerService {
	return &SchedulerService{
		queueManager: qm,
		repository:   repo,
		eventBus:     eb,
	}
}

// SubmitJob creates and enqueues a new job for immediate processing.
func (s *SchedulerService) SubmitJob(ctx context.Context, req JobRequest) (string, error) {
	// Generate job ID
	jobID := uuid.New().String()

	// Marshal payload
	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	// Set defaults
	queueName := req.Queue
	if queueName == "" {
		queueName = queue.QueueDefault
	}

	maxRetries := req.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	// Create job record
	job := &repository.Job{
		ID:         jobID,
		TaskType:   req.TaskType,
		Payload:    payloadBytes,
		Status:     repository.JobStatusPending,
		Queue:      queueName,
		MaxRetries: maxRetries,
		Timeout:    req.Timeout,
		Metadata:   req.Metadata,
	}

	// Save to repository
	if err := s.repository.CreateJob(ctx, job); err != nil {
		return "", fmt.Errorf("create job: %w", err)
	}

	// Emit job created event
	s.eventBus.Dispatch(ctx, event.NewEvent(event.EventJobCreated, map[string]any{
		"job_id":    jobID,
		"task_type": req.TaskType,
	}))

	// Create queue task
	task, err := queue.NewTask(req.TaskType, req.Payload)
	if err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}

	task.WithQueue(queueName).WithMaxRetry(maxRetries)
	if req.Timeout > 0 {
		task.WithTimeout(req.Timeout)
	}

	// Enqueue the task
	_, err = s.queueManager.EnqueueTask(ctx, task)
	if err != nil {
		// Update job status to failed if enqueue fails
		_ = s.repository.UpdateJobStatus(ctx, jobID, repository.JobStatusFailed, err)
		return "", fmt.Errorf("enqueue task: %w", err)
	}

	return jobID, nil
}

// ScheduleJob schedules a job for future execution.
func (s *SchedulerService) ScheduleJob(ctx context.Context, req JobRequest, scheduleTime time.Time) (string, error) {
	// Generate job ID
	jobID := uuid.New().String()

	// Marshal payload
	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	// Set defaults
	queueName := req.Queue
	if queueName == "" {
		queueName = queue.QueueDefault
	}

	maxRetries := req.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	// Create job record
	job := &repository.Job{
		ID:          jobID,
		TaskType:    req.TaskType,
		Payload:     payloadBytes,
		Status:      repository.JobStatusScheduled,
		Queue:       queueName,
		ScheduledAt: &scheduleTime,
		MaxRetries:  maxRetries,
		Timeout:     req.Timeout,
		Metadata:    req.Metadata,
	}

	// Save to repository
	if err := s.repository.CreateJob(ctx, job); err != nil {
		return "", fmt.Errorf("create job: %w", err)
	}

	// Emit job scheduled event
	s.eventBus.Dispatch(ctx, event.NewEvent(event.EventJobScheduled, map[string]any{
		"job_id":       jobID,
		"task_type":    req.TaskType,
		"scheduled_at": scheduleTime,
	}))

	// Create queue task
	task, err := queue.NewTask(req.TaskType, req.Payload)
	if err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}

	task.WithQueue(queueName).WithMaxRetry(maxRetries)
	if req.Timeout > 0 {
		task.WithTimeout(req.Timeout)
	}

	// Schedule the task
	_, err = s.queueManager.ScheduleTask(ctx, task, scheduleTime)
	if err != nil {
		// Update job status to failed if schedule fails
		_ = s.repository.UpdateJobStatus(ctx, jobID, repository.JobStatusFailed, err)
		return "", fmt.Errorf("schedule task: %w", err)
	}

	return jobID, nil
}

// CreateRecurringJob sets up a cron-based recurring job.
func (s *SchedulerService) CreateRecurringJob(ctx context.Context, req JobRequest, cronSpec string) (string, error) {
	// Generate job ID
	jobID := uuid.New().String()

	// Marshal payload
	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	// Set defaults
	queueName := req.Queue
	if queueName == "" {
		queueName = queue.QueueDefault
	}

	maxRetries := req.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	// Create queue task
	task, err := queue.NewTask(req.TaskType, req.Payload)
	if err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}

	task.WithQueue(queueName).WithMaxRetry(maxRetries)
	if req.Timeout > 0 {
		task.WithTimeout(req.Timeout)
	}

	// Register recurring task
	entryID, err := s.queueManager.EnqueueRecurringTask(task, cronSpec, jobID)
	if err != nil {
		return "", fmt.Errorf("register recurring task: %w", err)
	}

	// Create job record
	job := &repository.Job{
		ID:             jobID,
		TaskType:       req.TaskType,
		Payload:        payloadBytes,
		Status:         repository.JobStatusScheduled,
		Queue:          queueName,
		MaxRetries:     maxRetries,
		Timeout:        req.Timeout,
		CronExpression: cronSpec,
		CronEntryID:    entryID,
		Metadata:       req.Metadata,
	}

	// Save to repository
	if err := s.repository.CreateJob(ctx, job); err != nil {
		// Cleanup: unregister the recurring task
		_ = s.queueManager.UnregisterRecurringTask(entryID)
		return "", fmt.Errorf("create job: %w", err)
	}

	// Emit job scheduled event
	s.eventBus.Dispatch(ctx, event.NewEvent(event.EventJobScheduled, map[string]any{
		"job_id":          jobID,
		"task_type":       req.TaskType,
		"cron_expression": cronSpec,
	}))

	return jobID, nil
}

// CancelJob cancels a pending or scheduled job.
func (s *SchedulerService) CancelJob(ctx context.Context, jobID string) error {
	// Get job
	job, err := s.repository.GetJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}

	// Check if job can be cancelled
	if job.Status == repository.JobStatusCompleted || job.Status == repository.JobStatusCancelled {
		return fmt.Errorf("job already %s", job.Status)
	}

	// If it's a recurring job, unregister it
	if job.CronEntryID != "" {
		if err := s.queueManager.UnregisterRecurringTask(job.CronEntryID); err != nil {
			// Log but continue
		}
	}

	// Update job status
	if err := s.repository.UpdateJobStatus(ctx, jobID, repository.JobStatusCancelled, nil); err != nil {
		return fmt.Errorf("update job status: %w", err)
	}

	// Emit job cancelled event
	s.eventBus.Dispatch(ctx, event.NewEvent(event.EventJobCancelled, map[string]any{
		"job_id": jobID,
	}))

	return nil
}

// GetJobStatus retrieves current job status.
func (s *SchedulerService) GetJobStatus(ctx context.Context, jobID string) (*JobStatusResponse, error) {
	job, err := s.repository.GetJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}

	return &JobStatusResponse{
		ID:             job.ID,
		TaskType:       job.TaskType,
		Status:         job.Status,
		Queue:          job.Queue,
		ScheduledAt:    job.ScheduledAt,
		StartedAt:      job.StartedAt,
		CompletedAt:    job.CompletedAt,
		FailedAt:       job.FailedAt,
		RetryCount:     job.RetryCount,
		MaxRetries:     job.MaxRetries,
		Error:          job.Error,
		Result:         job.Result,
		CronExpression: job.CronExpression,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
	}, nil
}

// ListJobs lists jobs based on filter criteria.
func (s *SchedulerService) ListJobs(ctx context.Context, filter repository.JobFilter) ([]repository.Job, int64, error) {
	jobs, err := s.repository.ListJobs(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("list jobs: %w", err)
	}

	count, err := s.repository.CountJobs(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count jobs: %w", err)
	}

	return jobs, count, nil
}

// GetQueueStats retrieves statistics for the queues.
func (s *SchedulerService) GetQueueStats(ctx context.Context) (map[string]QueueStats, error) {
	queues, err := s.queueManager.ListQueues()
	if err != nil {
		return nil, fmt.Errorf("list queues: %w", err)
	}

	stats := make(map[string]QueueStats)
	for _, queueName := range queues {
		info, err := s.queueManager.GetQueueInfo(queueName)
		if err != nil {
			continue
		}

		stats[queueName] = QueueStats{
			Queue:      queueName,
			Size:       info.Size,
			Pending:    info.Pending,
			Active:     info.Active,
			Scheduled:  info.Scheduled,
			Retry:      info.Retry,
			Archived:   info.Archived,
			Completed:  info.Completed,
			Processed:  info.Processed,
			Failed:     info.Failed,
		}
	}

	return stats, nil
}

// Start starts the scheduler service.
func (s *SchedulerService) Start() error {
	return s.queueManager.Start()
}

// Stop stops the scheduler service.
func (s *SchedulerService) Stop() error {
	return s.queueManager.Stop()
}

// QueueStats represents statistics for a queue.
type QueueStats struct {
	Queue     string `json:"queue"`
	Size      int    `json:"size"`
	Pending   int    `json:"pending"`
	Active    int    `json:"active"`
	Scheduled int    `json:"scheduled"`
	Retry     int    `json:"retry"`
	Archived  int    `json:"archived"`
	Completed int    `json:"completed"`
	Processed int    `json:"processed"`
	Failed    int    `json:"failed"`
}

// MarkJobStarted marks a job as started.
func (s *SchedulerService) MarkJobStarted(ctx context.Context, jobID string) error {
	if err := s.repository.UpdateJobStatus(ctx, jobID, repository.JobStatusRunning, nil); err != nil {
		return fmt.Errorf("update job status: %w", err)
	}

	s.eventBus.Dispatch(ctx, event.NewEvent(event.EventJobStarted, map[string]any{
		"job_id": jobID,
	}))

	return nil
}

// MarkJobCompleted marks a job as completed with result.
func (s *SchedulerService) MarkJobCompleted(ctx context.Context, jobID string, result any) error {
	if err := s.repository.UpdateJobStatus(ctx, jobID, repository.JobStatusCompleted, nil); err != nil {
		return fmt.Errorf("update job status: %w", err)
	}

	if result != nil {
		resultBytes, err := json.Marshal(result)
		if err == nil {
			_ = s.repository.SetJobResult(ctx, jobID, resultBytes)
		}
	}

	s.eventBus.Dispatch(ctx, event.NewEvent(event.EventJobCompleted, map[string]any{
		"job_id": jobID,
	}))

	return nil
}

// MarkJobFailed marks a job as failed with error.
func (s *SchedulerService) MarkJobFailed(ctx context.Context, jobID string, jobErr error) error {
	if err := s.repository.UpdateJobStatus(ctx, jobID, repository.JobStatusFailed, jobErr); err != nil {
		return fmt.Errorf("update job status: %w", err)
	}

	s.eventBus.Dispatch(ctx, event.NewEvent(event.EventJobFailed, map[string]any{
		"job_id": jobID,
		"error":  jobErr.Error(),
	}))

	return nil
}

// MarkJobRetrying marks a job as retrying.
func (s *SchedulerService) MarkJobRetrying(ctx context.Context, jobID string) error {
	if err := s.repository.UpdateJobStatus(ctx, jobID, repository.JobStatusRetrying, nil); err != nil {
		return fmt.Errorf("update job status: %w", err)
	}

	if err := s.repository.IncrementRetryCount(ctx, jobID); err != nil {
		return fmt.Errorf("increment retry count: %w", err)
	}

	s.eventBus.Dispatch(ctx, event.NewEvent(event.EventJobRetrying, map[string]any{
		"job_id": jobID,
	}))

	return nil
}

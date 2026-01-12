package repository

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"
)

// MemoryJobRepository implements JobRepository using in-memory storage.
// Useful for testing and development.
type MemoryJobRepository struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

// NewMemoryJobRepository creates a new in-memory job repository.
func NewMemoryJobRepository() *MemoryJobRepository {
	return &MemoryJobRepository{
		jobs: make(map[string]*Job),
	}
}

// CreateJob creates a new job record.
func (r *MemoryJobRepository) CreateJob(ctx context.Context, job *Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()

	// Deep copy the job
	jobCopy := *job
	r.jobs[job.ID] = &jobCopy

	return nil
}

// GetJob retrieves a job by ID.
func (r *MemoryJobRepository) GetJob(ctx context.Context, jobID string) (*Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job, ok := r.jobs[jobID]
	if !ok {
		return nil, ErrJobNotFound
	}

	// Return a copy
	jobCopy := *job
	return &jobCopy, nil
}

// UpdateJob updates an existing job.
func (r *MemoryJobRepository) UpdateJob(ctx context.Context, job *Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.jobs[job.ID]; !ok {
		return ErrJobNotFound
	}

	job.UpdatedAt = time.Now()
	jobCopy := *job
	r.jobs[job.ID] = &jobCopy

	return nil
}

// UpdateJobStatus updates only the status and related timestamps.
func (r *MemoryJobRepository) UpdateJobStatus(ctx context.Context, jobID string, status JobStatus, jobErr error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, ok := r.jobs[jobID]
	if !ok {
		return ErrJobNotFound
	}

	now := time.Now()
	job.Status = status
	job.UpdatedAt = now

	switch status {
	case JobStatusRunning:
		job.StartedAt = &now
	case JobStatusCompleted:
		job.CompletedAt = &now
	case JobStatusFailed:
		job.FailedAt = &now
		if jobErr != nil {
			job.Error = jobErr.Error()
		}
	}

	return nil
}

// DeleteJob removes a job.
func (r *MemoryJobRepository) DeleteJob(ctx context.Context, jobID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.jobs[jobID]; !ok {
		return ErrJobNotFound
	}

	delete(r.jobs, jobID)
	return nil
}

// ListJobs lists jobs based on filter criteria.
func (r *MemoryJobRepository) ListJobs(ctx context.Context, filter JobFilter) ([]Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Job

	for _, job := range r.jobs {
		if r.matchesFilter(job, filter) {
			result = append(result, *job)
		}
	}

	// Sort
	r.sortJobs(result, filter.OrderBy, filter.OrderDirection)

	// Apply pagination
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	} else if filter.Offset >= len(result) {
		return []Job{}, nil
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if limit < len(result) {
		result = result[:limit]
	}

	return result, nil
}

// CountJobs counts jobs based on filter criteria.
func (r *MemoryJobRepository) CountJobs(ctx context.Context, filter JobFilter) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var count int64
	for _, job := range r.jobs {
		if r.matchesFilter(job, filter) {
			count++
		}
	}

	return count, nil
}

// GetJobsByStatus retrieves jobs by status with a limit.
func (r *MemoryJobRepository) GetJobsByStatus(ctx context.Context, status JobStatus, limit int) ([]Job, error) {
	return r.ListJobs(ctx, JobFilter{
		Status: []JobStatus{status},
		Limit:  limit,
	})
}

// GetPendingJobs retrieves all pending jobs ready to be executed.
func (r *MemoryJobRepository) GetPendingJobs(ctx context.Context, limit int) ([]Job, error) {
	now := time.Now()
	return r.ListJobs(ctx, JobFilter{
		Status:          []JobStatus{JobStatusPending, JobStatusScheduled},
		ScheduledBefore: &now,
		Limit:           limit,
		OrderBy:         "scheduled_at",
		OrderDirection:  "ASC",
	})
}

// GetRecurringJobs retrieves all jobs with cron expressions.
func (r *MemoryJobRepository) GetRecurringJobs(ctx context.Context) ([]Job, error) {
	withCron := true
	return r.ListJobs(ctx, JobFilter{
		WithCron: &withCron,
		Limit:    1000,
	})
}

// SetJobResult stores the result of a job execution.
func (r *MemoryJobRepository) SetJobResult(ctx context.Context, jobID string, result json.RawMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, ok := r.jobs[jobID]
	if !ok {
		return ErrJobNotFound
	}

	job.Result = result
	job.UpdatedAt = time.Now()

	return nil
}

// IncrementRetryCount increments the retry count for a job.
func (r *MemoryJobRepository) IncrementRetryCount(ctx context.Context, jobID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, ok := r.jobs[jobID]
	if !ok {
		return ErrJobNotFound
	}

	job.RetryCount++
	job.UpdatedAt = time.Now()

	return nil
}

// matchesFilter checks if a job matches the filter criteria.
func (r *MemoryJobRepository) matchesFilter(job *Job, filter JobFilter) bool {
	// Check status
	if len(filter.Status) > 0 {
		found := false
		for _, s := range filter.Status {
			if job.Status == s {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check task types
	if len(filter.TaskTypes) > 0 {
		found := false
		for _, t := range filter.TaskTypes {
			if job.TaskType == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check queue
	if filter.Queue != "" && job.Queue != filter.Queue {
		return false
	}

	// Check scheduled time
	if filter.ScheduledBefore != nil {
		if job.ScheduledAt != nil && job.ScheduledAt.After(*filter.ScheduledBefore) {
			return false
		}
	}

	if filter.ScheduledAfter != nil {
		if job.ScheduledAt == nil || job.ScheduledAt.Before(*filter.ScheduledAfter) {
			return false
		}
	}

	// Check created time
	if filter.CreatedBefore != nil && job.CreatedAt.After(*filter.CreatedBefore) {
		return false
	}

	if filter.CreatedAfter != nil && job.CreatedAt.Before(*filter.CreatedAfter) {
		return false
	}

	// Check cron
	if filter.WithCron != nil {
		hasCron := job.CronExpression != ""
		if *filter.WithCron != hasCron {
			return false
		}
	}

	return true
}

// sortJobs sorts jobs by the given field and direction.
func (r *MemoryJobRepository) sortJobs(jobs []Job, orderBy, direction string) {
	if orderBy == "" {
		orderBy = "created_at"
	}

	ascending := direction == "ASC"

	sort.Slice(jobs, func(i, j int) bool {
		var less bool
		switch orderBy {
		case "created_at":
			less = jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
		case "updated_at":
			less = jobs[i].UpdatedAt.Before(jobs[j].UpdatedAt)
		case "scheduled_at":
			if jobs[i].ScheduledAt == nil && jobs[j].ScheduledAt == nil {
				less = false
			} else if jobs[i].ScheduledAt == nil {
				less = false
			} else if jobs[j].ScheduledAt == nil {
				less = true
			} else {
				less = jobs[i].ScheduledAt.Before(*jobs[j].ScheduledAt)
			}
		case "status":
			less = jobs[i].Status < jobs[j].Status
		case "task_type":
			less = jobs[i].TaskType < jobs[j].TaskType
		default:
			less = jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
		}

		if ascending {
			return less
		}
		return !less
	})
}

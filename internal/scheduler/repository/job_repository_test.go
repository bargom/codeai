package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryJobRepository_CreateJob(t *testing.T) {
	repo := NewMemoryJobRepository()
	ctx := context.Background()

	job := &Job{
		ID:         "job-1",
		TaskType:   "test:task",
		Payload:    json.RawMessage(`{"key": "value"}`),
		Status:     JobStatusPending,
		Queue:      "default",
		MaxRetries: 3,
	}

	err := repo.CreateJob(ctx, job)
	require.NoError(t, err)

	// Verify job was created
	retrieved, err := repo.GetJob(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, job.ID, retrieved.ID)
	assert.Equal(t, job.TaskType, retrieved.TaskType)
	assert.Equal(t, job.Status, retrieved.Status)
}

func TestMemoryJobRepository_GetJob_NotFound(t *testing.T) {
	repo := NewMemoryJobRepository()
	ctx := context.Background()

	_, err := repo.GetJob(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrJobNotFound)
}

func TestMemoryJobRepository_UpdateJobStatus(t *testing.T) {
	repo := NewMemoryJobRepository()
	ctx := context.Background()

	// Create a job
	job := &Job{
		ID:       "job-1",
		TaskType: "test:task",
		Status:   JobStatusPending,
		Queue:    "default",
	}
	err := repo.CreateJob(ctx, job)
	require.NoError(t, err)

	// Update status to running
	err = repo.UpdateJobStatus(ctx, "job-1", JobStatusRunning, nil)
	require.NoError(t, err)

	// Verify status was updated
	retrieved, err := repo.GetJob(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, JobStatusRunning, retrieved.Status)
	assert.NotNil(t, retrieved.StartedAt)

	// Update status to completed
	err = repo.UpdateJobStatus(ctx, "job-1", JobStatusCompleted, nil)
	require.NoError(t, err)

	// Verify status was updated
	retrieved, err = repo.GetJob(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, JobStatusCompleted, retrieved.Status)
	assert.NotNil(t, retrieved.CompletedAt)
}

func TestMemoryJobRepository_UpdateJobStatus_Failed(t *testing.T) {
	repo := NewMemoryJobRepository()
	ctx := context.Background()

	// Create a job
	job := &Job{
		ID:       "job-1",
		TaskType: "test:task",
		Status:   JobStatusRunning,
		Queue:    "default",
	}
	err := repo.CreateJob(ctx, job)
	require.NoError(t, err)

	// Update status to failed with error
	testErr := assert.AnError
	err = repo.UpdateJobStatus(ctx, "job-1", JobStatusFailed, testErr)
	require.NoError(t, err)

	// Verify status was updated
	retrieved, err := repo.GetJob(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, JobStatusFailed, retrieved.Status)
	assert.NotNil(t, retrieved.FailedAt)
	assert.Contains(t, retrieved.Error, "assert.AnError")
}

func TestMemoryJobRepository_DeleteJob(t *testing.T) {
	repo := NewMemoryJobRepository()
	ctx := context.Background()

	// Create a job
	job := &Job{
		ID:       "job-1",
		TaskType: "test:task",
		Status:   JobStatusPending,
		Queue:    "default",
	}
	err := repo.CreateJob(ctx, job)
	require.NoError(t, err)

	// Delete the job
	err = repo.DeleteJob(ctx, "job-1")
	require.NoError(t, err)

	// Verify job was deleted
	_, err = repo.GetJob(ctx, "job-1")
	assert.ErrorIs(t, err, ErrJobNotFound)
}

func TestMemoryJobRepository_ListJobs(t *testing.T) {
	repo := NewMemoryJobRepository()
	ctx := context.Background()

	// Create multiple jobs
	for i := 0; i < 5; i++ {
		job := &Job{
			ID:       "job-" + string(rune('a'+i)),
			TaskType: "test:task",
			Status:   JobStatusPending,
			Queue:    "default",
		}
		err := repo.CreateJob(ctx, job)
		require.NoError(t, err)
	}

	// Update some jobs to different statuses
	_ = repo.UpdateJobStatus(ctx, "job-a", JobStatusRunning, nil)
	_ = repo.UpdateJobStatus(ctx, "job-b", JobStatusCompleted, nil)

	// List all jobs
	jobs, err := repo.ListJobs(ctx, JobFilter{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, jobs, 5)

	// List only pending jobs
	jobs, err = repo.ListJobs(ctx, JobFilter{
		Status: []JobStatus{JobStatusPending},
		Limit:  10,
	})
	require.NoError(t, err)
	assert.Len(t, jobs, 3)

	// List running and completed jobs
	jobs, err = repo.ListJobs(ctx, JobFilter{
		Status: []JobStatus{JobStatusRunning, JobStatusCompleted},
		Limit:  10,
	})
	require.NoError(t, err)
	assert.Len(t, jobs, 2)
}

func TestMemoryJobRepository_CountJobs(t *testing.T) {
	repo := NewMemoryJobRepository()
	ctx := context.Background()

	// Create multiple jobs
	for i := 0; i < 10; i++ {
		status := JobStatusPending
		if i%2 == 0 {
			status = JobStatusCompleted
		}
		job := &Job{
			ID:       "job-" + string(rune('a'+i)),
			TaskType: "test:task",
			Status:   status,
			Queue:    "default",
		}
		err := repo.CreateJob(ctx, job)
		require.NoError(t, err)
	}

	// Count all jobs
	count, err := repo.CountJobs(ctx, JobFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(10), count)

	// Count only pending jobs
	count, err = repo.CountJobs(ctx, JobFilter{
		Status: []JobStatus{JobStatusPending},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestMemoryJobRepository_GetPendingJobs(t *testing.T) {
	repo := NewMemoryJobRepository()
	ctx := context.Background()

	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	// Create jobs with different scheduled times
	job1 := &Job{
		ID:          "job-1",
		TaskType:    "test:task",
		Status:      JobStatusPending,
		Queue:       "default",
		ScheduledAt: &past, // Should be included
	}
	job2 := &Job{
		ID:          "job-2",
		TaskType:    "test:task",
		Status:      JobStatusScheduled,
		Queue:       "default",
		ScheduledAt: &future, // Should not be included (future)
	}
	job3 := &Job{
		ID:       "job-3",
		TaskType: "test:task",
		Status:   JobStatusPending,
		Queue:    "default",
		// No scheduled time - should be included
	}

	_ = repo.CreateJob(ctx, job1)
	_ = repo.CreateJob(ctx, job2)
	_ = repo.CreateJob(ctx, job3)

	jobs, err := repo.GetPendingJobs(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, jobs, 2) // job-1 and job-3
}

func TestMemoryJobRepository_GetRecurringJobs(t *testing.T) {
	repo := NewMemoryJobRepository()
	ctx := context.Background()

	// Create jobs with and without cron expressions
	job1 := &Job{
		ID:             "job-1",
		TaskType:       "test:task",
		Status:         JobStatusScheduled,
		Queue:          "default",
		CronExpression: "0 * * * *",
	}
	job2 := &Job{
		ID:       "job-2",
		TaskType: "test:task",
		Status:   JobStatusPending,
		Queue:    "default",
	}
	job3 := &Job{
		ID:             "job-3",
		TaskType:       "test:task",
		Status:         JobStatusScheduled,
		Queue:          "default",
		CronExpression: "0 0 * * *",
	}

	_ = repo.CreateJob(ctx, job1)
	_ = repo.CreateJob(ctx, job2)
	_ = repo.CreateJob(ctx, job3)

	jobs, err := repo.GetRecurringJobs(ctx)
	require.NoError(t, err)
	assert.Len(t, jobs, 2) // job-1 and job-3
}

func TestMemoryJobRepository_SetJobResult(t *testing.T) {
	repo := NewMemoryJobRepository()
	ctx := context.Background()

	job := &Job{
		ID:       "job-1",
		TaskType: "test:task",
		Status:   JobStatusCompleted,
		Queue:    "default",
	}
	err := repo.CreateJob(ctx, job)
	require.NoError(t, err)

	result := json.RawMessage(`{"success": true, "count": 42}`)
	err = repo.SetJobResult(ctx, "job-1", result)
	require.NoError(t, err)

	retrieved, err := repo.GetJob(ctx, "job-1")
	require.NoError(t, err)
	assert.JSONEq(t, string(result), string(retrieved.Result))
}

func TestMemoryJobRepository_IncrementRetryCount(t *testing.T) {
	repo := NewMemoryJobRepository()
	ctx := context.Background()

	job := &Job{
		ID:         "job-1",
		TaskType:   "test:task",
		Status:     JobStatusRetrying,
		Queue:      "default",
		RetryCount: 0,
	}
	err := repo.CreateJob(ctx, job)
	require.NoError(t, err)

	// Increment retry count
	err = repo.IncrementRetryCount(ctx, "job-1")
	require.NoError(t, err)

	retrieved, err := repo.GetJob(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, 1, retrieved.RetryCount)

	// Increment again
	err = repo.IncrementRetryCount(ctx, "job-1")
	require.NoError(t, err)

	retrieved, err = repo.GetJob(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, 2, retrieved.RetryCount)
}

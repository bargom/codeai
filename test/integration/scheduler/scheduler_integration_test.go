//go:build integration

// Package scheduler provides integration tests for the scheduler service.
package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bargom/codeai/internal/event"
	schedrepo "github.com/bargom/codeai/internal/scheduler/repository"
	"github.com/bargom/codeai/internal/scheduler/service"
	"github.com/bargom/codeai/test/integration/testutil"
)

// SchedulerTestSuite holds resources for scheduler integration tests.
type SchedulerTestSuite struct {
	Repository      *schedrepo.MemoryJobRepository
	EventDispatcher event.Dispatcher
	Fixtures        *testutil.FixtureBuilder
	EventCollector  *testutil.EventCollector
	Ctx             context.Context
	Cancel          context.CancelFunc
}

// NewSchedulerTestSuite creates a new scheduler test suite.
func NewSchedulerTestSuite(t *testing.T) *SchedulerTestSuite {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	suite := &SchedulerTestSuite{
		Repository:      schedrepo.NewMemoryJobRepository(),
		EventDispatcher: event.NewDispatcher(),
		Fixtures:        testutil.NewFixtureBuilder(),
		EventCollector:  testutil.NewEventCollector(),
		Ctx:             ctx,
		Cancel:          cancel,
	}

	// Subscribe to job events to collect them
	suite.EventDispatcher.Subscribe(event.EventJobCreated, func(ctx context.Context, e event.Event) error {
		suite.EventCollector.Collect(e)
		return nil
	})
	suite.EventDispatcher.Subscribe(event.EventJobCompleted, func(ctx context.Context, e event.Event) error {
		suite.EventCollector.Collect(e)
		return nil
	})
	suite.EventDispatcher.Subscribe(event.EventJobFailed, func(ctx context.Context, e event.Event) error {
		suite.EventCollector.Collect(e)
		return nil
	})

	return suite
}

// Teardown cleans up the test suite.
func (s *SchedulerTestSuite) Teardown() {
	s.Cancel()
}

// Reset clears the repository and event collector.
func (s *SchedulerTestSuite) Reset() {
	s.Repository = schedrepo.NewMemoryJobRepository()
	s.EventCollector.Reset()
}

func TestJobRepositoryCRUD(t *testing.T) {
	suite := NewSchedulerTestSuite(t)
	defer suite.Teardown()

	t.Run("create and retrieve job", func(t *testing.T) {
		suite.Reset()

		job := suite.Fixtures.CreateTestJob("test-task", map[string]interface{}{"key": "value"})

		err := suite.Repository.CreateJob(suite.Ctx, job)
		require.NoError(t, err)

		retrieved, err := suite.Repository.GetJob(suite.Ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, job.ID, retrieved.ID)
		assert.Equal(t, job.TaskType, retrieved.TaskType)
		assert.Equal(t, schedrepo.JobStatusPending, retrieved.Status)
	})

	t.Run("update job status", func(t *testing.T) {
		suite.Reset()

		job := suite.Fixtures.CreateTestJob("test-task", nil)
		err := suite.Repository.CreateJob(suite.Ctx, job)
		require.NoError(t, err)

		// Update to running
		err = suite.Repository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusRunning, nil)
		require.NoError(t, err)

		retrieved, err := suite.Repository.GetJob(suite.Ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, schedrepo.JobStatusRunning, retrieved.Status)
		assert.NotNil(t, retrieved.StartedAt)
	})

	t.Run("update job to completed", func(t *testing.T) {
		suite.Reset()

		job := suite.Fixtures.CreateTestJob("test-task", nil)
		err := suite.Repository.CreateJob(suite.Ctx, job)
		require.NoError(t, err)

		err = suite.Repository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusCompleted, nil)
		require.NoError(t, err)

		retrieved, err := suite.Repository.GetJob(suite.Ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, schedrepo.JobStatusCompleted, retrieved.Status)
		assert.NotNil(t, retrieved.CompletedAt)
	})

	t.Run("update job to failed with error", func(t *testing.T) {
		suite.Reset()

		job := suite.Fixtures.CreateTestJob("test-task", nil)
		err := suite.Repository.CreateJob(suite.Ctx, job)
		require.NoError(t, err)

		testErr := assert.AnError
		err = suite.Repository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusFailed, testErr)
		require.NoError(t, err)

		retrieved, err := suite.Repository.GetJob(suite.Ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, schedrepo.JobStatusFailed, retrieved.Status)
		assert.NotNil(t, retrieved.FailedAt)
		assert.Contains(t, retrieved.Error, "error")
	})

	t.Run("delete job", func(t *testing.T) {
		suite.Reset()

		job := suite.Fixtures.CreateTestJob("test-task", nil)
		err := suite.Repository.CreateJob(suite.Ctx, job)
		require.NoError(t, err)

		err = suite.Repository.DeleteJob(suite.Ctx, job.ID)
		require.NoError(t, err)

		_, err = suite.Repository.GetJob(suite.Ctx, job.ID)
		assert.ErrorIs(t, err, schedrepo.ErrJobNotFound)
	})

	t.Run("get non-existent job returns error", func(t *testing.T) {
		suite.Reset()

		_, err := suite.Repository.GetJob(suite.Ctx, "non-existent-id")
		assert.ErrorIs(t, err, schedrepo.ErrJobNotFound)
	})
}

func TestJobListingAndFiltering(t *testing.T) {
	suite := NewSchedulerTestSuite(t)
	defer suite.Teardown()

	t.Run("list jobs by status", func(t *testing.T) {
		suite.Reset()

		// Create jobs with different statuses
		pendingJob := suite.Fixtures.CreateTestJob("task-pending", nil)
		pendingJob.Status = schedrepo.JobStatusPending
		require.NoError(t, suite.Repository.CreateJob(suite.Ctx, pendingJob))

		runningJob := suite.Fixtures.CreateTestJob("task-running", nil)
		runningJob.Status = schedrepo.JobStatusRunning
		require.NoError(t, suite.Repository.CreateJob(suite.Ctx, runningJob))

		completedJob := suite.Fixtures.CreateTestJob("task-completed", nil)
		completedJob.Status = schedrepo.JobStatusCompleted
		require.NoError(t, suite.Repository.CreateJob(suite.Ctx, completedJob))

		// Filter by pending status
		jobs, err := suite.Repository.GetJobsByStatus(suite.Ctx, schedrepo.JobStatusPending, 10)
		require.NoError(t, err)
		assert.Len(t, jobs, 1)
		assert.Equal(t, pendingJob.ID, jobs[0].ID)

		// Filter by completed status
		jobs, err = suite.Repository.GetJobsByStatus(suite.Ctx, schedrepo.JobStatusCompleted, 10)
		require.NoError(t, err)
		assert.Len(t, jobs, 1)
		assert.Equal(t, completedJob.ID, jobs[0].ID)
	})

	t.Run("list jobs with pagination", func(t *testing.T) {
		suite.Reset()

		// Create 5 jobs
		for i := 0; i < 5; i++ {
			job := suite.Fixtures.CreateTestJob("task", nil)
			require.NoError(t, suite.Repository.CreateJob(suite.Ctx, job))
			time.Sleep(time.Millisecond) // Ensure different created times
		}

		// Get first page
		filter := schedrepo.JobFilter{
			Limit:  2,
			Offset: 0,
		}
		jobs, err := suite.Repository.ListJobs(suite.Ctx, filter)
		require.NoError(t, err)
		assert.Len(t, jobs, 2)

		// Get second page
		filter.Offset = 2
		jobs, err = suite.Repository.ListJobs(suite.Ctx, filter)
		require.NoError(t, err)
		assert.Len(t, jobs, 2)

		// Get last page
		filter.Offset = 4
		jobs, err = suite.Repository.ListJobs(suite.Ctx, filter)
		require.NoError(t, err)
		assert.Len(t, jobs, 1)
	})

	t.Run("list jobs by task type", func(t *testing.T) {
		suite.Reset()

		job1 := suite.Fixtures.CreateTestJob("type-a", nil)
		require.NoError(t, suite.Repository.CreateJob(suite.Ctx, job1))

		job2 := suite.Fixtures.CreateTestJob("type-b", nil)
		require.NoError(t, suite.Repository.CreateJob(suite.Ctx, job2))

		job3 := suite.Fixtures.CreateTestJob("type-a", nil)
		require.NoError(t, suite.Repository.CreateJob(suite.Ctx, job3))

		filter := schedrepo.JobFilter{
			TaskTypes: []string{"type-a"},
			Limit:     10,
		}
		jobs, err := suite.Repository.ListJobs(suite.Ctx, filter)
		require.NoError(t, err)
		assert.Len(t, jobs, 2)
	})

	t.Run("count jobs by filter", func(t *testing.T) {
		suite.Reset()

		for i := 0; i < 3; i++ {
			job := suite.Fixtures.CreateTestJob("task", nil)
			job.Status = schedrepo.JobStatusCompleted
			require.NoError(t, suite.Repository.CreateJob(suite.Ctx, job))
		}

		for i := 0; i < 2; i++ {
			job := suite.Fixtures.CreateTestJob("task", nil)
			job.Status = schedrepo.JobStatusFailed
			require.NoError(t, suite.Repository.CreateJob(suite.Ctx, job))
		}

		filter := schedrepo.JobFilter{
			Status: []schedrepo.JobStatus{schedrepo.JobStatusCompleted},
		}
		count, err := suite.Repository.CountJobs(suite.Ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("get recurring jobs", func(t *testing.T) {
		suite.Reset()

		regularJob := suite.Fixtures.CreateTestJob("regular-task", nil)
		require.NoError(t, suite.Repository.CreateJob(suite.Ctx, regularJob))

		recurringJob := suite.Fixtures.CreateRecurringJob("recurring-task", nil, "*/5 * * * *")
		require.NoError(t, suite.Repository.CreateJob(suite.Ctx, recurringJob))

		jobs, err := suite.Repository.GetRecurringJobs(suite.Ctx)
		require.NoError(t, err)
		assert.Len(t, jobs, 1)
		assert.Equal(t, recurringJob.ID, jobs[0].ID)
		assert.NotEmpty(t, jobs[0].CronExpression)
	})
}

func TestJobRetryMechanics(t *testing.T) {
	suite := NewSchedulerTestSuite(t)
	defer suite.Teardown()

	t.Run("increment retry count", func(t *testing.T) {
		suite.Reset()

		job := suite.Fixtures.CreateTestJob("retry-task", nil)
		job.MaxRetries = 3
		require.NoError(t, suite.Repository.CreateJob(suite.Ctx, job))

		assert.Equal(t, 0, job.RetryCount)

		// Increment retry count
		err := suite.Repository.IncrementRetryCount(suite.Ctx, job.ID)
		require.NoError(t, err)

		retrieved, err := suite.Repository.GetJob(suite.Ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, retrieved.RetryCount)

		// Increment again
		err = suite.Repository.IncrementRetryCount(suite.Ctx, job.ID)
		require.NoError(t, err)

		retrieved, err = suite.Repository.GetJob(suite.Ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, retrieved.RetryCount)
	})

	t.Run("set job result", func(t *testing.T) {
		suite.Reset()

		job := suite.Fixtures.CreateTestJob("result-task", nil)
		require.NoError(t, suite.Repository.CreateJob(suite.Ctx, job))

		result := []byte(`{"success": true, "output": "test result"}`)
		err := suite.Repository.SetJobResult(suite.Ctx, job.ID, result)
		require.NoError(t, err)

		retrieved, err := suite.Repository.GetJob(suite.Ctx, job.ID)
		require.NoError(t, err)
		assert.JSONEq(t, string(result), string(retrieved.Result))
	})
}

func TestScheduledJobsRetrieval(t *testing.T) {
	suite := NewSchedulerTestSuite(t)
	defer suite.Teardown()

	t.Run("get pending jobs ready to execute", func(t *testing.T) {
		suite.Reset()

		// Create a job scheduled in the past (ready to execute)
		pastTime := time.Now().Add(-1 * time.Hour)
		pastJob := suite.Fixtures.CreateScheduledJob("past-task", nil, pastTime)
		require.NoError(t, suite.Repository.CreateJob(suite.Ctx, pastJob))

		// Create a job scheduled in the future (not ready)
		futureTime := time.Now().Add(1 * time.Hour)
		futureJob := suite.Fixtures.CreateScheduledJob("future-task", nil, futureTime)
		require.NoError(t, suite.Repository.CreateJob(suite.Ctx, futureJob))

		// Create a regular pending job (no schedule time)
		pendingJob := suite.Fixtures.CreateTestJob("pending-task", nil)
		require.NoError(t, suite.Repository.CreateJob(suite.Ctx, pendingJob))

		// Get pending jobs
		jobs, err := suite.Repository.GetPendingJobs(suite.Ctx, 10)
		require.NoError(t, err)

		// Should include past scheduled job and regular pending job, but not future job
		assert.GreaterOrEqual(t, len(jobs), 1)

		// Verify the past job is included
		var foundPast bool
		for _, j := range jobs {
			if j.ID == pastJob.ID {
				foundPast = true
				break
			}
		}
		assert.True(t, foundPast, "Past scheduled job should be in pending jobs")
	})
}

func TestJobServiceIntegration(t *testing.T) {
	suite := NewSchedulerTestSuite(t)
	defer suite.Teardown()

	// Note: Full service tests require Redis and Asynq
	// These tests focus on repository-level integration

	t.Run("job lifecycle simulation", func(t *testing.T) {
		suite.Reset()

		// Create job
		job := suite.Fixtures.CreateTestJob("lifecycle-task", map[string]interface{}{
			"action": "process",
		})
		err := suite.Repository.CreateJob(suite.Ctx, job)
		require.NoError(t, err)

		// Simulate job starting
		err = suite.Repository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusRunning, nil)
		require.NoError(t, err)

		// Verify running state
		retrieved, err := suite.Repository.GetJob(suite.Ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, schedrepo.JobStatusRunning, retrieved.Status)
		assert.NotNil(t, retrieved.StartedAt)

		// Simulate job completion
		result := []byte(`{"processed": true}`)
		err = suite.Repository.SetJobResult(suite.Ctx, job.ID, result)
		require.NoError(t, err)

		err = suite.Repository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusCompleted, nil)
		require.NoError(t, err)

		// Verify completed state
		retrieved, err = suite.Repository.GetJob(suite.Ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, schedrepo.JobStatusCompleted, retrieved.Status)
		assert.NotNil(t, retrieved.CompletedAt)
		assert.JSONEq(t, `{"processed": true}`, string(retrieved.Result))
	})

	t.Run("job failure and retry simulation", func(t *testing.T) {
		suite.Reset()

		// Create job with retry limit
		job := suite.Fixtures.CreateTestJob("failing-task", nil)
		job.MaxRetries = 3
		err := suite.Repository.CreateJob(suite.Ctx, job)
		require.NoError(t, err)

		// Simulate multiple failure attempts
		for i := 0; i < 3; i++ {
			// Start job
			err = suite.Repository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusRunning, nil)
			require.NoError(t, err)

			// Job fails
			err = suite.Repository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusRetrying, nil)
			require.NoError(t, err)

			err = suite.Repository.IncrementRetryCount(suite.Ctx, job.ID)
			require.NoError(t, err)
		}

		// Final failure
		err = suite.Repository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusFailed, assert.AnError)
		require.NoError(t, err)

		// Verify final state
		retrieved, err := suite.Repository.GetJob(suite.Ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, schedrepo.JobStatusFailed, retrieved.Status)
		assert.Equal(t, 3, retrieved.RetryCount)
		assert.NotNil(t, retrieved.FailedAt)
	})
}

func TestJobServiceRequestValidation(t *testing.T) {
	t.Run("job request structure", func(t *testing.T) {
		req := service.JobRequest{
			TaskType:   "test-task",
			Payload:    map[string]interface{}{"key": "value"},
			Queue:      "default",
			MaxRetries: 3,
			Timeout:    30 * time.Second,
			Metadata:   map[string]interface{}{"env": "test"},
		}

		assert.NotEmpty(t, req.TaskType)
		assert.NotNil(t, req.Payload)
		assert.Equal(t, "default", req.Queue)
		assert.Equal(t, 3, req.MaxRetries)
		assert.Equal(t, 30*time.Second, req.Timeout)
	})
}

func TestConcurrentJobOperations(t *testing.T) {
	suite := NewSchedulerTestSuite(t)
	defer suite.Teardown()

	t.Run("concurrent job creation", func(t *testing.T) {
		suite.Reset()

		const numJobs = 100
		done := make(chan bool, numJobs)

		for i := 0; i < numJobs; i++ {
			go func(idx int) {
				job := suite.Fixtures.CreateTestJob("concurrent-task", map[string]interface{}{"index": idx})
				err := suite.Repository.CreateJob(suite.Ctx, job)
				if err != nil {
					t.Errorf("failed to create job %d: %v", idx, err)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numJobs; i++ {
			<-done
		}

		// Verify all jobs were created
		jobs, err := suite.Repository.ListJobs(suite.Ctx, schedrepo.JobFilter{Limit: 200})
		require.NoError(t, err)
		assert.Len(t, jobs, numJobs)
	})

	t.Run("concurrent status updates", func(t *testing.T) {
		suite.Reset()

		// Create jobs
		const numJobs = 50
		jobIDs := make([]string, numJobs)
		for i := 0; i < numJobs; i++ {
			job := suite.Fixtures.CreateTestJob("status-task", nil)
			err := suite.Repository.CreateJob(suite.Ctx, job)
			require.NoError(t, err)
			jobIDs[i] = job.ID
		}

		done := make(chan bool, numJobs)

		// Concurrently update all jobs to running
		for _, id := range jobIDs {
			go func(jobID string) {
				err := suite.Repository.UpdateJobStatus(suite.Ctx, jobID, schedrepo.JobStatusRunning, nil)
				if err != nil {
					t.Errorf("failed to update job %s: %v", jobID, err)
				}
				done <- true
			}(id)
		}

		// Wait for all updates
		for i := 0; i < numJobs; i++ {
			<-done
		}

		// Verify all jobs are running
		jobs, err := suite.Repository.GetJobsByStatus(suite.Ctx, schedrepo.JobStatusRunning, 100)
		require.NoError(t, err)
		assert.Len(t, jobs, numJobs)
	})
}

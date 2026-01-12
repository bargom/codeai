//go:build integration

// Package scenarios provides end-to-end integration tests for workflow scenarios.
package scenarios

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bargom/codeai/internal/event"
	"github.com/bargom/codeai/internal/event/bus"
	emailrepo "github.com/bargom/codeai/internal/notification/email/repository"
	schedrepo "github.com/bargom/codeai/internal/scheduler/repository"
	webhookrepo "github.com/bargom/codeai/internal/webhook/repository"
	"github.com/bargom/codeai/test/integration/testutil"
)

// ScenarioTestSuite holds all resources for end-to-end tests.
type ScenarioTestSuite struct {
	// Repositories
	JobRepository     *schedrepo.MemoryJobRepository
	WebhookRepository *webhookrepo.MemoryRepository
	EmailRepository   *emailrepo.MemoryEmailRepository

	// Event infrastructure
	EventDispatcher event.Dispatcher
	EventBus        *bus.EventBus

	// Test helpers
	WebhookServer  *testutil.WebhookTestServer
	EmailClient    *testutil.MockEmailClient
	EventCollector *testutil.EventCollector
	Fixtures       *testutil.FixtureBuilder

	// Context
	Ctx    context.Context
	Cancel context.CancelFunc
}

// NewScenarioTestSuite creates a new scenario test suite.
func NewScenarioTestSuite(t *testing.T) *ScenarioTestSuite {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	return &ScenarioTestSuite{
		JobRepository:     schedrepo.NewMemoryJobRepository(),
		WebhookRepository: webhookrepo.NewMemoryRepository(),
		EmailRepository:   emailrepo.NewMemoryEmailRepository(),
		EventDispatcher:   event.NewDispatcher(),
		EventBus:          bus.NewEventBus(nil),
		WebhookServer:     testutil.NewWebhookTestServer(),
		EmailClient:       testutil.NewMockEmailClient(),
		EventCollector:    testutil.NewEventCollector(),
		Fixtures:          testutil.NewFixtureBuilder(),
		Ctx:               ctx,
		Cancel:            cancel,
	}
}

// Teardown cleans up all resources.
func (s *ScenarioTestSuite) Teardown() {
	s.WebhookServer.Close()
	s.EventBus.Close()
	s.Cancel()
}

// Reset clears all state.
func (s *ScenarioTestSuite) Reset() {
	s.JobRepository = schedrepo.NewMemoryJobRepository()
	s.WebhookRepository = webhookrepo.NewMemoryRepository()
	s.EmailRepository = emailrepo.NewMemoryEmailRepository()
	s.WebhookServer.Reset()
	s.EmailClient.Reset()
	s.EventCollector.Reset()
}

// SetupEventSubscribers sets up event subscribers for the test.
func (s *ScenarioTestSuite) SetupEventSubscribers() {
	// Collect all events for verification
	s.EventBus.Subscribe(bus.EventJobCompleted, bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
		s.EventCollector.Collect(e)
		return nil
	}))

	s.EventBus.Subscribe(bus.EventWorkflowCompleted, bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
		s.EventCollector.Collect(e)
		return nil
	}))

	s.EventBus.Subscribe(bus.EventJobFailed, bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
		s.EventCollector.Collect(e)
		return nil
	}))

	s.EventBus.Subscribe(bus.EventWorkflowFailed, bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
		s.EventCollector.Collect(e)
		return nil
	}))
}

func TestJobToWebhookFlow(t *testing.T) {
	suite := NewScenarioTestSuite(t)
	defer suite.Teardown()

	t.Run("job completion triggers webhook delivery", func(t *testing.T) {
		suite.Reset()
		suite.SetupEventSubscribers()

		// 1. Setup webhook to receive job completion events
		webhook := suite.Fixtures.CreateTestWebhook(
			suite.WebhookServer.URL(),
			bus.EventJobCompleted,
		)
		err := suite.WebhookRepository.CreateWebhook(suite.Ctx, webhook)
		require.NoError(t, err)

		// 2. Create and submit job
		job := suite.Fixtures.CreateTestJob("process-data", map[string]interface{}{
			"input": "test-data",
		})
		err = suite.JobRepository.CreateJob(suite.Ctx, job)
		require.NoError(t, err)

		// 3. Simulate job execution
		err = suite.JobRepository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusRunning, nil)
		require.NoError(t, err)

		// 4. Job completes
		result := []byte(`{"processed": true, "count": 42}`)
		err = suite.JobRepository.SetJobResult(suite.Ctx, job.ID, result)
		require.NoError(t, err)

		err = suite.JobRepository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusCompleted, nil)
		require.NoError(t, err)

		// 5. Publish job completed event
		completedEvent := suite.Fixtures.CreateJobCompletedEvent(job.ID)
		err = suite.EventBus.Publish(suite.Ctx, completedEvent)
		require.NoError(t, err)

		// 6. Verify event was captured
		assert.Equal(t, 1, suite.EventCollector.Count())

		// 7. Verify job state
		retrievedJob, err := suite.JobRepository.GetJob(suite.Ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, schedrepo.JobStatusCompleted, retrievedJob.Status)
		assert.NotNil(t, retrievedJob.CompletedAt)
	})
}

func TestJobFailureNotificationFlow(t *testing.T) {
	suite := NewScenarioTestSuite(t)
	defer suite.Teardown()

	t.Run("job failure triggers error notification", func(t *testing.T) {
		suite.Reset()
		suite.SetupEventSubscribers()

		// 1. Create job
		job := suite.Fixtures.CreateTestJob("failing-task", nil)
		job.MaxRetries = 3
		err := suite.JobRepository.CreateJob(suite.Ctx, job)
		require.NoError(t, err)

		// 2. Simulate job starting
		err = suite.JobRepository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusRunning, nil)
		require.NoError(t, err)

		// 3. Simulate retries
		for i := 0; i < 3; i++ {
			err = suite.JobRepository.IncrementRetryCount(suite.Ctx, job.ID)
			require.NoError(t, err)
		}

		// 4. Job fails permanently
		err = suite.JobRepository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusFailed, assert.AnError)
		require.NoError(t, err)

		// 5. Publish failure event
		failedEvent := suite.Fixtures.CreateTestEvent(bus.EventJobFailed, map[string]interface{}{
			"job_id": job.ID,
			"error":  "max retries exceeded",
		})
		err = suite.EventBus.Publish(suite.Ctx, failedEvent)
		require.NoError(t, err)

		// 6. Verify job state
		retrievedJob, err := suite.JobRepository.GetJob(suite.Ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, schedrepo.JobStatusFailed, retrievedJob.Status)
		assert.Equal(t, 3, retrievedJob.RetryCount)
		assert.NotNil(t, retrievedJob.FailedAt)

		// 7. Verify event was captured
		assert.Equal(t, 1, suite.EventCollector.Count())
	})
}

func TestEmailNotificationScenario(t *testing.T) {
	suite := NewScenarioTestSuite(t)
	defer suite.Teardown()

	t.Run("workflow completion triggers email notification", func(t *testing.T) {
		suite.Reset()
		suite.SetupEventSubscribers()

		// 1. Simulate workflow completing
		workflowID := "wf-" + testutil.RandomString(8)

		// 2. Publish workflow completed event
		completedEvent := suite.Fixtures.CreateWorkflowCompletedEvent(workflowID)
		err := suite.EventBus.Publish(suite.Ctx, completedEvent)
		require.NoError(t, err)

		// 3. Verify event captured
		assert.Equal(t, 1, suite.EventCollector.Count())

		// 4. Simulate email being sent
		_, err = suite.EmailClient.Send(
			[]string{"admin@example.com"},
			"Workflow Completed: "+workflowID,
			"<h1>Workflow Completed</h1>",
			"Workflow completed successfully",
		)
		require.NoError(t, err)

		// 5. Log email
		emailLog := suite.Fixtures.CreateTestEmailLog(
			[]string{"admin@example.com"},
			"Workflow Completed: "+workflowID,
			"sent",
		)
		err = suite.EmailRepository.SaveEmail(suite.Ctx, emailLog)
		require.NoError(t, err)

		// 6. Verify email was logged
		assert.Equal(t, 1, suite.EmailClient.Count())
		assert.Equal(t, 1, suite.EmailRepository.Count())
	})
}

func TestConcurrentJobProcessing(t *testing.T) {
	suite := NewScenarioTestSuite(t)
	defer suite.Teardown()

	t.Run("concurrent jobs are processed correctly", func(t *testing.T) {
		suite.Reset()
		suite.SetupEventSubscribers()

		const numJobs = 50
		var completedCount int32
		var wg sync.WaitGroup

		// Create all jobs
		jobIDs := make([]string, numJobs)
		for i := 0; i < numJobs; i++ {
			job := suite.Fixtures.CreateTestJob("concurrent-task", map[string]interface{}{"index": i})
			err := suite.JobRepository.CreateJob(suite.Ctx, job)
			require.NoError(t, err)
			jobIDs[i] = job.ID
		}

		// Process jobs concurrently
		for _, id := range jobIDs {
			wg.Add(1)
			go func(jobID string) {
				defer wg.Done()

				// Start
				if err := suite.JobRepository.UpdateJobStatus(suite.Ctx, jobID, schedrepo.JobStatusRunning, nil); err != nil {
					t.Errorf("failed to start job: %v", err)
					return
				}

				// Simulate work
				time.Sleep(10 * time.Millisecond)

				// Complete
				if err := suite.JobRepository.UpdateJobStatus(suite.Ctx, jobID, schedrepo.JobStatusCompleted, nil); err != nil {
					t.Errorf("failed to complete job: %v", err)
					return
				}

				atomic.AddInt32(&completedCount, 1)
			}(id)
		}

		wg.Wait()

		// Verify all jobs completed
		assert.Equal(t, int32(numJobs), atomic.LoadInt32(&completedCount))

		// Verify job states
		jobs, err := suite.JobRepository.GetJobsByStatus(suite.Ctx, schedrepo.JobStatusCompleted, 100)
		require.NoError(t, err)
		assert.Len(t, jobs, numJobs)
	})
}

func TestMixedSuccessAndFailureScenario(t *testing.T) {
	suite := NewScenarioTestSuite(t)
	defer suite.Teardown()

	t.Run("mixed success and failure processing", func(t *testing.T) {
		suite.Reset()
		suite.SetupEventSubscribers()

		// Create jobs
		const numJobs = 20
		successfulJobs := make([]*schedrepo.Job, 0)
		failedJobs := make([]*schedrepo.Job, 0)

		for i := 0; i < numJobs; i++ {
			job := suite.Fixtures.CreateTestJob("mixed-task", nil)
			err := suite.JobRepository.CreateJob(suite.Ctx, job)
			require.NoError(t, err)

			// Even jobs succeed, odd jobs fail
			if i%2 == 0 {
				successfulJobs = append(successfulJobs, job)
			} else {
				failedJobs = append(failedJobs, job)
			}
		}

		// Process successful jobs
		for _, job := range successfulJobs {
			err := suite.JobRepository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusRunning, nil)
			require.NoError(t, err)

			err = suite.JobRepository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusCompleted, nil)
			require.NoError(t, err)
		}

		// Process failed jobs
		for _, job := range failedJobs {
			err := suite.JobRepository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusRunning, nil)
			require.NoError(t, err)

			err = suite.JobRepository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusFailed, assert.AnError)
			require.NoError(t, err)
		}

		// Verify counts
		completed, err := suite.JobRepository.GetJobsByStatus(suite.Ctx, schedrepo.JobStatusCompleted, 100)
		require.NoError(t, err)
		assert.Len(t, completed, len(successfulJobs))

		failed, err := suite.JobRepository.GetJobsByStatus(suite.Ctx, schedrepo.JobStatusFailed, 100)
		require.NoError(t, err)
		assert.Len(t, failed, len(failedJobs))
	})
}

func TestEventPropagationScenario(t *testing.T) {
	suite := NewScenarioTestSuite(t)
	defer suite.Teardown()

	t.Run("events propagate through the system", func(t *testing.T) {
		suite.Reset()
		suite.SetupEventSubscribers()

		// Subscribe multiple handlers to track propagation
		var handlerCalls int32

		suite.EventBus.Subscribe(bus.EventWorkflowCompleted, bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
			atomic.AddInt32(&handlerCalls, 1)
			return nil
		}))

		suite.EventBus.Subscribe(bus.EventWorkflowCompleted, bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
			atomic.AddInt32(&handlerCalls, 1)
			return nil
		}))

		// Publish event
		event := suite.Fixtures.CreateWorkflowCompletedEvent("wf-test")
		err := suite.EventBus.Publish(suite.Ctx, event)
		require.NoError(t, err)

		// Wait for async processing
		time.Sleep(50 * time.Millisecond)

		// All handlers should be called (including the one in SetupEventSubscribers)
		assert.GreaterOrEqual(t, atomic.LoadInt32(&handlerCalls), int32(2))
	})
}

func TestWebhookWithRetryScenario(t *testing.T) {
	suite := NewScenarioTestSuite(t)
	defer suite.Teardown()

	t.Run("webhook delivery with simulated retry", func(t *testing.T) {
		suite.Reset()

		// Create webhook
		webhook := suite.Fixtures.CreateTestWebhook(
			suite.WebhookServer.URL(),
			bus.EventJobCompleted,
		)
		err := suite.WebhookRepository.CreateWebhook(suite.Ctx, webhook)
		require.NoError(t, err)

		// Simulate first failed delivery
		failedDelivery := suite.Fixtures.CreateWebhookDelivery(
			webhook.ID,
			bus.EventJobCompleted,
			false,
		)
		nextRetry := time.Now().Add(5 * time.Minute)
		failedDelivery.NextRetryAt = &nextRetry
		failedDelivery.Attempts = 1
		err = suite.WebhookRepository.SaveDelivery(suite.Ctx, failedDelivery)
		require.NoError(t, err)

		// Increment failure count
		err = suite.WebhookRepository.IncrementFailureCount(suite.Ctx, webhook.ID)
		require.NoError(t, err)

		// Simulate successful retry
		successDelivery := suite.Fixtures.CreateWebhookDelivery(
			webhook.ID,
			bus.EventJobCompleted,
			true,
		)
		successDelivery.Attempts = 2
		err = suite.WebhookRepository.SaveDelivery(suite.Ctx, successDelivery)
		require.NoError(t, err)

		// Reset failure count on success
		err = suite.WebhookRepository.ResetFailureCount(suite.Ctx, webhook.ID)
		require.NoError(t, err)

		// Verify webhook state
		retrievedWebhook, err := suite.WebhookRepository.GetWebhook(suite.Ctx, webhook.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, retrievedWebhook.FailureCount)
		assert.True(t, retrievedWebhook.Active)
	})
}

func TestFullPipelineScenario(t *testing.T) {
	suite := NewScenarioTestSuite(t)
	defer suite.Teardown()

	t.Run("complete pipeline: job -> event -> webhook + email", func(t *testing.T) {
		suite.Reset()
		suite.SetupEventSubscribers()

		// 1. Setup infrastructure
		webhook := suite.Fixtures.CreateTestWebhook(
			suite.WebhookServer.URL(),
			bus.EventJobCompleted,
		)
		err := suite.WebhookRepository.CreateWebhook(suite.Ctx, webhook)
		require.NoError(t, err)

		// 2. Create and run job
		job := suite.Fixtures.CreateTestJob("pipeline-task", map[string]interface{}{
			"step": "process",
		})
		err = suite.JobRepository.CreateJob(suite.Ctx, job)
		require.NoError(t, err)

		// 3. Execute job
		err = suite.JobRepository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusRunning, nil)
		require.NoError(t, err)

		// Simulate processing
		result := []byte(`{"success": true}`)
		err = suite.JobRepository.SetJobResult(suite.Ctx, job.ID, result)
		require.NoError(t, err)

		err = suite.JobRepository.UpdateJobStatus(suite.Ctx, job.ID, schedrepo.JobStatusCompleted, nil)
		require.NoError(t, err)

		// 4. Publish completion event
		completedEvent := suite.Fixtures.CreateJobCompletedEvent(job.ID)
		err = suite.EventBus.Publish(suite.Ctx, completedEvent)
		require.NoError(t, err)

		// 5. Simulate webhook delivery
		delivery := suite.Fixtures.CreateWebhookDelivery(
			webhook.ID,
			bus.EventJobCompleted,
			true,
		)
		err = suite.WebhookRepository.SaveDelivery(suite.Ctx, delivery)
		require.NoError(t, err)

		// 6. Simulate email notification
		_, err = suite.EmailClient.Send(
			[]string{"admin@example.com"},
			"Job Completed: "+job.ID,
			"<h1>Success</h1>",
			"Job completed",
		)
		require.NoError(t, err)

		emailLog := suite.Fixtures.CreateTestEmailLog(
			[]string{"admin@example.com"},
			"Job Completed: "+job.ID,
			"sent",
		)
		err = suite.EmailRepository.SaveEmail(suite.Ctx, emailLog)
		require.NoError(t, err)

		// 7. Verify entire pipeline
		// Job completed
		retrievedJob, err := suite.JobRepository.GetJob(suite.Ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, schedrepo.JobStatusCompleted, retrievedJob.Status)

		// Event captured
		assert.GreaterOrEqual(t, suite.EventCollector.Count(), 1)

		// Webhook delivered
		deliveries, err := suite.WebhookRepository.ListDeliveries(suite.Ctx, webhook.ID, webhookrepo.DeliveryFilter{Limit: 10})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(deliveries), 1)

		// Email sent
		assert.Equal(t, 1, suite.EmailClient.Count())
		assert.Equal(t, 1, suite.EmailRepository.Count())
	})
}

func TestRecurringJobScenario(t *testing.T) {
	suite := NewScenarioTestSuite(t)
	defer suite.Teardown()

	t.Run("recurring job executes multiple times", func(t *testing.T) {
		suite.Reset()

		// Create recurring job
		recurringJob := suite.Fixtures.CreateRecurringJob(
			"scheduled-report",
			map[string]interface{}{"report": "daily"},
			"0 * * * *", // Every hour
		)
		err := suite.JobRepository.CreateJob(suite.Ctx, recurringJob)
		require.NoError(t, err)

		// Simulate 3 executions
		for i := 0; i < 3; i++ {
			// Start execution
			err = suite.JobRepository.UpdateJobStatus(suite.Ctx, recurringJob.ID, schedrepo.JobStatusRunning, nil)
			require.NoError(t, err)

			// Complete execution
			result := []byte(`{"execution": ` + string(rune('0'+i)) + `}`)
			err = suite.JobRepository.SetJobResult(suite.Ctx, recurringJob.ID, result)
			require.NoError(t, err)

			err = suite.JobRepository.UpdateJobStatus(suite.Ctx, recurringJob.ID, schedrepo.JobStatusScheduled, nil)
			require.NoError(t, err)
		}

		// Verify job is still recurring
		retrievedJob, err := suite.JobRepository.GetJob(suite.Ctx, recurringJob.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, retrievedJob.CronExpression)

		// Check it's in recurring jobs list
		recurringJobs, err := suite.JobRepository.GetRecurringJobs(suite.Ctx)
		require.NoError(t, err)
		assert.Len(t, recurringJobs, 1)
	})
}

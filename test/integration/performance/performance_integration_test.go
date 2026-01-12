//go:build integration

// Package performance provides performance integration tests.
package performance

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bargom/codeai/internal/event/bus"
	emailrepo "github.com/bargom/codeai/internal/notification/email/repository"
	schedrepo "github.com/bargom/codeai/internal/scheduler/repository"
	webhookrepo "github.com/bargom/codeai/internal/webhook/repository"
	"github.com/bargom/codeai/test/integration/testutil"
)

// PerformanceTestSuite holds resources for performance tests.
type PerformanceTestSuite struct {
	JobRepository     *schedrepo.MemoryJobRepository
	WebhookRepository *webhookrepo.MemoryRepository
	EmailRepository   *emailrepo.MemoryEmailRepository
	EventBus          *bus.EventBus
	Fixtures          *testutil.FixtureBuilder
	Ctx               context.Context
	Cancel            context.CancelFunc
}

// NewPerformanceTestSuite creates a new performance test suite.
func NewPerformanceTestSuite(t *testing.T) *PerformanceTestSuite {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	return &PerformanceTestSuite{
		JobRepository:     schedrepo.NewMemoryJobRepository(),
		WebhookRepository: webhookrepo.NewMemoryRepository(),
		EmailRepository:   emailrepo.NewMemoryEmailRepository(),
		EventBus:          bus.NewEventBus(nil),
		Fixtures:          testutil.NewFixtureBuilder(),
		Ctx:               ctx,
		Cancel:            cancel,
	}
}

// Teardown cleans up the test suite.
func (s *PerformanceTestSuite) Teardown() {
	s.EventBus.Close()
	s.Cancel()
}

// Reset clears all state.
func (s *PerformanceTestSuite) Reset() {
	s.JobRepository = schedrepo.NewMemoryJobRepository()
	s.WebhookRepository = webhookrepo.NewMemoryRepository()
	s.EmailRepository = emailrepo.NewMemoryEmailRepository()
}

func TestJobThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	suite := NewPerformanceTestSuite(t)
	defer suite.Teardown()

	t.Run("create 1000 jobs", func(t *testing.T) {
		suite.Reset()

		const numJobs = 1000
		start := time.Now()

		for i := 0; i < numJobs; i++ {
			job := suite.Fixtures.CreateTestJob("throughput-task", map[string]interface{}{"index": i})
			err := suite.JobRepository.CreateJob(suite.Ctx, job)
			require.NoError(t, err)
		}

		elapsed := time.Since(start)
		t.Logf("Created %d jobs in %v (%.2f jobs/sec)", numJobs, elapsed, float64(numJobs)/elapsed.Seconds())

		// Verify all jobs created
		jobs, err := suite.JobRepository.ListJobs(suite.Ctx, schedrepo.JobFilter{Limit: 2000})
		require.NoError(t, err)
		assert.Len(t, jobs, numJobs)

		// Should complete within reasonable time
		assert.Less(t, elapsed, 10*time.Second)
	})

	t.Run("concurrent job creation throughput", func(t *testing.T) {
		suite.Reset()

		const numJobs = 1000
		const concurrency = 50

		var wg sync.WaitGroup
		var created int32

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				jobsPerWorker := numJobs / concurrency
				for j := 0; j < jobsPerWorker; j++ {
					job := suite.Fixtures.CreateTestJob("concurrent-throughput", map[string]interface{}{
						"worker": workerID,
						"job":    j,
					})
					if err := suite.JobRepository.CreateJob(suite.Ctx, job); err == nil {
						atomic.AddInt32(&created, 1)
					}
				}
			}(i)
		}

		wg.Wait()
		elapsed := time.Since(start)

		t.Logf("Created %d jobs concurrently in %v (%.2f jobs/sec)", created, elapsed, float64(created)/elapsed.Seconds())

		assert.Equal(t, int32(numJobs), created)
		assert.Less(t, elapsed, 10*time.Second)
	})

	t.Run("job status updates throughput", func(t *testing.T) {
		suite.Reset()

		const numJobs = 500

		// Create jobs
		jobIDs := make([]string, numJobs)
		for i := 0; i < numJobs; i++ {
			job := suite.Fixtures.CreateTestJob("status-throughput", nil)
			require.NoError(t, suite.JobRepository.CreateJob(suite.Ctx, job))
			jobIDs[i] = job.ID
		}

		start := time.Now()

		// Update all jobs to running, then completed
		for _, id := range jobIDs {
			require.NoError(t, suite.JobRepository.UpdateJobStatus(suite.Ctx, id, schedrepo.JobStatusRunning, nil))
			require.NoError(t, suite.JobRepository.UpdateJobStatus(suite.Ctx, id, schedrepo.JobStatusCompleted, nil))
		}

		elapsed := time.Since(start)
		updates := numJobs * 2

		t.Logf("Performed %d status updates in %v (%.2f updates/sec)", updates, elapsed, float64(updates)/elapsed.Seconds())

		assert.Less(t, elapsed, 10*time.Second)
	})
}

func TestEventThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("publish 10000 events", func(t *testing.T) {
		eb := bus.NewEventBus(nil)
		defer eb.Close()

		fixtures := testutil.NewFixtureBuilder()
		ctx := context.Background()

		var received int32

		subscriber := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
			atomic.AddInt32(&received, 1)
			return nil
		})

		eb.Subscribe(bus.EventType("perf.test"), subscriber)

		const numEvents = 10000
		start := time.Now()

		for i := 0; i < numEvents; i++ {
			event := fixtures.CreateTestEvent(bus.EventType("perf.test"), nil)
			_ = eb.Publish(ctx, event)
		}

		elapsed := time.Since(start)
		t.Logf("Published %d events in %v (%.2f events/sec)", numEvents, elapsed, float64(numEvents)/elapsed.Seconds())

		// Wait for all events to be processed
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, int32(numEvents), atomic.LoadInt32(&received))
		assert.Less(t, elapsed, 5*time.Second)
	})

	t.Run("async event throughput", func(t *testing.T) {
		eb := bus.NewEventBus(nil)
		defer eb.Close()

		fixtures := testutil.NewFixtureBuilder()
		ctx := context.Background()

		var received int32
		var wg sync.WaitGroup

		const numEvents = 5000
		wg.Add(numEvents)

		subscriber := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
			atomic.AddInt32(&received, 1)
			wg.Done()
			return nil
		})

		eb.Subscribe(bus.EventType("async.perf"), subscriber)

		start := time.Now()

		for i := 0; i < numEvents; i++ {
			event := fixtures.CreateTestEvent(bus.EventType("async.perf"), nil)
			eb.PublishAsync(ctx, event)
		}

		publishElapsed := time.Since(start)
		t.Logf("Async published %d events in %v (%.2f events/sec)", numEvents, publishElapsed, float64(numEvents)/publishElapsed.Seconds())

		// Wait for all events to be delivered
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(30 * time.Second):
			t.Fatal("timeout waiting for async events")
		}

		totalElapsed := time.Since(start)
		t.Logf("All %d async events processed in %v", numEvents, totalElapsed)

		assert.Equal(t, int32(numEvents), atomic.LoadInt32(&received))
	})

	t.Run("multiple subscribers throughput", func(t *testing.T) {
		eb := bus.NewEventBus(nil)
		defer eb.Close()

		fixtures := testutil.NewFixtureBuilder()
		ctx := context.Background()

		const numSubscribers = 10
		const numEvents = 1000

		var totalReceived int32

		for i := 0; i < numSubscribers; i++ {
			subscriber := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
				atomic.AddInt32(&totalReceived, 1)
				return nil
			})
			eb.Subscribe(bus.EventType("multi.sub"), subscriber)
		}

		start := time.Now()

		for i := 0; i < numEvents; i++ {
			event := fixtures.CreateTestEvent(bus.EventType("multi.sub"), nil)
			_ = eb.Publish(ctx, event)
		}

		elapsed := time.Since(start)
		expectedTotal := numEvents * numSubscribers

		t.Logf("Published %d events to %d subscribers in %v", numEvents, numSubscribers, elapsed)

		// Wait for processing
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, int32(expectedTotal), atomic.LoadInt32(&totalReceived))
	})
}

func TestWebhookThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	suite := NewPerformanceTestSuite(t)
	defer suite.Teardown()

	t.Run("create and query webhooks", func(t *testing.T) {
		suite.Reset()

		const numWebhooks = 500

		start := time.Now()

		// Create webhooks
		webhookIDs := make([]string, numWebhooks)
		for i := 0; i < numWebhooks; i++ {
			webhook := suite.Fixtures.CreateTestWebhook(
				"https://example.com/webhook",
				bus.EventWorkflowCompleted,
			)
			require.NoError(t, suite.WebhookRepository.CreateWebhook(suite.Ctx, webhook))
			webhookIDs[i] = webhook.ID
		}

		createElapsed := time.Since(start)
		t.Logf("Created %d webhooks in %v", numWebhooks, createElapsed)

		// Query webhooks by event
		queryStart := time.Now()
		webhooks, err := suite.WebhookRepository.GetWebhooksByEvent(suite.Ctx, bus.EventWorkflowCompleted)
		require.NoError(t, err)
		queryElapsed := time.Since(queryStart)

		t.Logf("Queried %d webhooks in %v", len(webhooks), queryElapsed)

		assert.Len(t, webhooks, numWebhooks)
		assert.Less(t, createElapsed, 5*time.Second)
		assert.Less(t, queryElapsed, 1*time.Second)
	})

	t.Run("delivery logging throughput", func(t *testing.T) {
		suite.Reset()

		webhook := suite.Fixtures.CreateTestWebhook(
			"https://example.com/webhook",
			bus.EventWorkflowCompleted,
		)
		require.NoError(t, suite.WebhookRepository.CreateWebhook(suite.Ctx, webhook))

		const numDeliveries = 1000

		start := time.Now()

		for i := 0; i < numDeliveries; i++ {
			delivery := suite.Fixtures.CreateWebhookDelivery(
				webhook.ID,
				bus.EventWorkflowCompleted,
				true,
			)
			require.NoError(t, suite.WebhookRepository.SaveDelivery(suite.Ctx, delivery))
		}

		elapsed := time.Since(start)
		t.Logf("Saved %d deliveries in %v (%.2f deliveries/sec)", numDeliveries, elapsed, float64(numDeliveries)/elapsed.Seconds())

		// Verify
		deliveries, err := suite.WebhookRepository.ListDeliveries(suite.Ctx, webhook.ID, webhookrepo.DeliveryFilter{Limit: 2000})
		require.NoError(t, err)
		assert.Len(t, deliveries, numDeliveries)

		assert.Less(t, elapsed, 5*time.Second)
	})
}

func TestEmailThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	suite := NewPerformanceTestSuite(t)
	defer suite.Teardown()

	t.Run("email logging throughput", func(t *testing.T) {
		suite.Reset()

		const numEmails = 1000

		start := time.Now()

		for i := 0; i < numEmails; i++ {
			email := suite.Fixtures.CreateTestEmailLog(
				[]string{"test@example.com"},
				"Test Email",
				"sent",
			)
			require.NoError(t, suite.EmailRepository.SaveEmail(suite.Ctx, email))
		}

		elapsed := time.Since(start)
		t.Logf("Saved %d emails in %v (%.2f emails/sec)", numEmails, elapsed, float64(numEmails)/elapsed.Seconds())

		assert.Equal(t, numEmails, suite.EmailRepository.Count())
		assert.Less(t, elapsed, 5*time.Second)
	})
}

func TestConcurrentOperationsThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	suite := NewPerformanceTestSuite(t)
	defer suite.Teardown()

	t.Run("mixed concurrent operations", func(t *testing.T) {
		suite.Reset()

		const opsPerType = 200
		const concurrency = 20

		var wg sync.WaitGroup
		var totalOps int32

		start := time.Now()

		// Concurrent job operations
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < opsPerType/concurrency; j++ {
					job := suite.Fixtures.CreateTestJob("mixed-ops", nil)
					if suite.JobRepository.CreateJob(suite.Ctx, job) == nil {
						atomic.AddInt32(&totalOps, 1)
					}
				}
			}()
		}

		// Concurrent webhook operations
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < opsPerType/concurrency; j++ {
					webhook := suite.Fixtures.CreateTestWebhook("https://example.com/webhook")
					if suite.WebhookRepository.CreateWebhook(suite.Ctx, webhook) == nil {
						atomic.AddInt32(&totalOps, 1)
					}
				}
			}()
		}

		// Concurrent email operations
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < opsPerType/concurrency; j++ {
					email := suite.Fixtures.CreateTestEmailLog([]string{"test@example.com"}, "Test", "sent")
					if suite.EmailRepository.SaveEmail(suite.Ctx, email) == nil {
						atomic.AddInt32(&totalOps, 1)
					}
				}
			}()
		}

		wg.Wait()

		elapsed := time.Since(start)
		t.Logf("Completed %d mixed operations in %v (%.2f ops/sec)", totalOps, elapsed, float64(totalOps)/elapsed.Seconds())

		expectedOps := int32(opsPerType * 3)
		assert.Equal(t, expectedOps, atomic.LoadInt32(&totalOps))
		assert.Less(t, elapsed, 10*time.Second)
	})
}

func TestQueryPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	suite := NewPerformanceTestSuite(t)
	defer suite.Teardown()

	t.Run("job listing with filters", func(t *testing.T) {
		suite.Reset()

		// Create jobs with different statuses
		const numJobs = 1000
		for i := 0; i < numJobs; i++ {
			job := suite.Fixtures.CreateTestJob("query-test", nil)
			if i%3 == 0 {
				job.Status = schedrepo.JobStatusCompleted
			} else if i%3 == 1 {
				job.Status = schedrepo.JobStatusFailed
			}
			require.NoError(t, suite.JobRepository.CreateJob(suite.Ctx, job))
		}

		// Query with filter
		filter := schedrepo.JobFilter{
			Status: []schedrepo.JobStatus{schedrepo.JobStatusCompleted},
			Limit:  100,
		}

		start := time.Now()
		jobs, err := suite.JobRepository.ListJobs(suite.Ctx, filter)
		elapsed := time.Since(start)

		require.NoError(t, err)
		t.Logf("Queried filtered jobs in %v, found %d", elapsed, len(jobs))

		assert.Less(t, elapsed, 100*time.Millisecond)
	})

	t.Run("count operations", func(t *testing.T) {
		suite.Reset()

		const numJobs = 1000
		for i := 0; i < numJobs; i++ {
			job := suite.Fixtures.CreateTestJob("count-test", nil)
			require.NoError(t, suite.JobRepository.CreateJob(suite.Ctx, job))
		}

		filter := schedrepo.JobFilter{}

		start := time.Now()
		count, err := suite.JobRepository.CountJobs(suite.Ctx, filter)
		elapsed := time.Since(start)

		require.NoError(t, err)
		t.Logf("Counted %d jobs in %v", count, elapsed)

		assert.Equal(t, int64(numJobs), count)
		assert.Less(t, elapsed, 50*time.Millisecond)
	})
}

func BenchmarkJobCreation(b *testing.B) {
	repo := schedrepo.NewMemoryJobRepository()
	fixtures := testutil.NewFixtureBuilder()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job := fixtures.CreateTestJob("benchmark-task", nil)
		_ = repo.CreateJob(ctx, job)
	}
}

func BenchmarkEventPublish(b *testing.B) {
	eb := bus.NewEventBus(nil)
	defer eb.Close()

	fixtures := testutil.NewFixtureBuilder()
	ctx := context.Background()

	subscriber := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
		return nil
	})
	eb.Subscribe(bus.EventType("bench.test"), subscriber)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := fixtures.CreateTestEvent(bus.EventType("bench.test"), nil)
		_ = eb.Publish(ctx, event)
	}
}

func BenchmarkWebhookQuery(b *testing.B) {
	repo := webhookrepo.NewMemoryRepository()
	fixtures := testutil.NewFixtureBuilder()
	ctx := context.Background()

	// Create webhooks
	for i := 0; i < 100; i++ {
		webhook := fixtures.CreateTestWebhook("https://example.com", bus.EventWorkflowCompleted)
		_ = repo.CreateWebhook(ctx, webhook)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.GetWebhooksByEvent(ctx, bus.EventWorkflowCompleted)
	}
}

func BenchmarkEmailSave(b *testing.B) {
	repo := emailrepo.NewMemoryEmailRepository()
	fixtures := testutil.NewFixtureBuilder()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		email := fixtures.CreateTestEmailLog([]string{"test@example.com"}, "Test", "sent")
		_ = repo.SaveEmail(ctx, email)
	}
}

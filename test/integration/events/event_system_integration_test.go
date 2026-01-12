//go:build integration

// Package events provides integration tests for the event system.
package events

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bargom/codeai/internal/event"
	"github.com/bargom/codeai/internal/event/bus"
	"github.com/bargom/codeai/test/integration/testutil"
)

// EventTestSuite holds resources for event system integration tests.
type EventTestSuite struct {
	EventBus        *bus.EventBus
	EventDispatcher event.Dispatcher
	Fixtures        *testutil.FixtureBuilder
	Logger          *testLogger
	Ctx             context.Context
	Cancel          context.CancelFunc
}

// testLogger implements bus.Logger for tests.
type testLogger struct {
	messages []string
	mu       sync.Mutex
}

func newTestLogger() *testLogger {
	return &testLogger{messages: make([]string, 0)}
}

func (l *testLogger) Info(msg string, args ...any)  { l.log("INFO", msg) }
func (l *testLogger) Error(msg string, args ...any) { l.log("ERROR", msg) }
func (l *testLogger) Debug(msg string, args ...any) { l.log("DEBUG", msg) }
func (l *testLogger) Warn(msg string, args ...any)  { l.log("WARN", msg) }

func (l *testLogger) log(level, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, level+": "+msg)
}

func (l *testLogger) getMessages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]string, len(l.messages))
	copy(result, l.messages)
	return result
}

// NewEventTestSuite creates a new event test suite.
func NewEventTestSuite(t *testing.T) *EventTestSuite {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	logger := newTestLogger()

	return &EventTestSuite{
		EventBus:        bus.NewEventBus(logger),
		EventDispatcher: event.NewDispatcher(),
		Fixtures:        testutil.NewFixtureBuilder(),
		Ctx:             ctx,
		Cancel:          cancel,
	}
}

// Teardown cleans up the test suite.
func (s *EventTestSuite) Teardown() {
	s.EventBus.Close()
	s.Cancel()
}

func TestEventBusBasicOperations(t *testing.T) {
	suite := NewEventTestSuite(t)
	defer suite.Teardown()

	t.Run("subscribe and receive event", func(t *testing.T) {
		received := make(chan bus.Event, 1)

		subscriber := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
			received <- e
			return nil
		})

		suite.EventBus.Subscribe(bus.EventWorkflowCompleted, subscriber)

		event := suite.Fixtures.CreateTestEvent(bus.EventWorkflowCompleted, map[string]interface{}{
			"workflow_id": "test-123",
		})

		err := suite.EventBus.Publish(suite.Ctx, event)
		require.NoError(t, err)

		select {
		case e := <-received:
			assert.Equal(t, bus.EventWorkflowCompleted, e.Type)
			assert.Equal(t, "test-123", e.Data["workflow_id"])
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("multiple subscribers receive same event", func(t *testing.T) {
		var count int32
		wg := sync.WaitGroup{}
		wg.Add(3)

		for i := 0; i < 3; i++ {
			subscriber := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
				atomic.AddInt32(&count, 1)
				wg.Done()
				return nil
			})
			suite.EventBus.Subscribe(bus.EventJobCompleted, subscriber)
		}

		event := suite.Fixtures.CreateTestEvent(bus.EventJobCompleted, nil)
		err := suite.EventBus.Publish(suite.Ctx, event)
		require.NoError(t, err)

		wg.Wait()
		assert.Equal(t, int32(3), atomic.LoadInt32(&count))
	})

	t.Run("subscriber count is tracked", func(t *testing.T) {
		eventType := bus.EventType("test.subscriber.count")

		assert.Equal(t, 0, suite.EventBus.SubscriberCount(eventType))

		subscriber1 := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error { return nil })
		subscriber2 := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error { return nil })

		suite.EventBus.Subscribe(eventType, subscriber1)
		assert.Equal(t, 1, suite.EventBus.SubscriberCount(eventType))

		suite.EventBus.Subscribe(eventType, subscriber2)
		assert.Equal(t, 2, suite.EventBus.SubscriberCount(eventType))
	})

	t.Run("no subscribers handles gracefully", func(t *testing.T) {
		event := suite.Fixtures.CreateTestEvent(bus.EventType("no.subscribers"), nil)

		err := suite.EventBus.Publish(suite.Ctx, event)
		assert.NoError(t, err)
	})
}

func TestAsyncEventPublishing(t *testing.T) {
	logger := newTestLogger()
	eb := bus.NewEventBus(logger)
	defer eb.Close()

	ctx := context.Background()
	fixtures := testutil.NewFixtureBuilder()

	t.Run("async publish returns immediately", func(t *testing.T) {
		received := make(chan struct{}, 1)

		subscriber := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
			time.Sleep(100 * time.Millisecond) // Simulate slow subscriber
			received <- struct{}{}
			return nil
		})

		eb.Subscribe(bus.EventType("async.test"), subscriber)

		event := fixtures.CreateTestEvent(bus.EventType("async.test"), nil)

		start := time.Now()
		eb.PublishAsync(ctx, event)
		elapsed := time.Since(start)

		// PublishAsync should return almost immediately
		assert.Less(t, elapsed, 50*time.Millisecond)

		// Event should still be delivered
		select {
		case <-received:
			// Success
		case <-time.After(1 * time.Second):
			t.Fatal("async event was not delivered")
		}
	})

	t.Run("multiple async events are delivered", func(t *testing.T) {
		var count int32
		wg := sync.WaitGroup{}
		wg.Add(10)

		subscriber := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
			atomic.AddInt32(&count, 1)
			wg.Done()
			return nil
		})

		eb.Subscribe(bus.EventType("async.batch"), subscriber)

		for i := 0; i < 10; i++ {
			event := fixtures.CreateTestEvent(bus.EventType("async.batch"), nil)
			eb.PublishAsync(ctx, event)
		}

		wg.Wait()
		assert.Equal(t, int32(10), atomic.LoadInt32(&count))
	})
}

func TestEventErrorHandling(t *testing.T) {
	logger := newTestLogger()
	eb := bus.NewEventBus(logger)
	defer eb.Close()

	ctx := context.Background()
	fixtures := testutil.NewFixtureBuilder()

	t.Run("subscriber error does not affect other subscribers", func(t *testing.T) {
		var successCount int32

		// Failing subscriber
		failingSubscriber := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
			return errors.New("subscriber failed")
		})

		// Successful subscribers
		successSubscriber := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
			atomic.AddInt32(&successCount, 1)
			return nil
		})

		eb.Subscribe(bus.EventType("error.test"), failingSubscriber)
		eb.Subscribe(bus.EventType("error.test"), successSubscriber)
		eb.Subscribe(bus.EventType("error.test"), successSubscriber)

		event := fixtures.CreateTestEvent(bus.EventType("error.test"), nil)
		err := eb.Publish(ctx, event)

		// Publish should still succeed even with failing subscriber
		assert.NoError(t, err)

		// Both success subscribers should have been called
		assert.Equal(t, int32(2), atomic.LoadInt32(&successCount))
	})

	t.Run("subscriber panic is recovered", func(t *testing.T) {
		var called bool

		panicSubscriber := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
			panic("subscriber panicked")
		})

		safeSubscriber := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
			called = true
			return nil
		})

		eb.Subscribe(bus.EventType("panic.test"), panicSubscriber)
		eb.Subscribe(bus.EventType("panic.test"), safeSubscriber)

		event := fixtures.CreateTestEvent(bus.EventType("panic.test"), nil)

		// Should not panic
		assert.NotPanics(t, func() {
			_ = eb.Publish(ctx, event)
		})

		// Safe subscriber should still be called
		assert.True(t, called)
	})
}

func TestEventDispatcher(t *testing.T) {
	suite := NewEventTestSuite(t)
	defer suite.Teardown()

	t.Run("dispatch event to handler", func(t *testing.T) {
		received := make(chan event.Event, 1)

		handler := func(ctx context.Context, e event.Event) error {
			received <- e
			return nil
		}

		suite.EventDispatcher.Subscribe(event.EventJobCreated, handler)

		evt := event.NewEvent(event.EventJobCreated, map[string]interface{}{"job_id": "123"})

		err := suite.EventDispatcher.Dispatch(suite.Ctx, evt)
		require.NoError(t, err)

		select {
		case e := <-received:
			assert.Equal(t, event.EventJobCreated, e.Type)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("multiple handlers for same event", func(t *testing.T) {
		var count int32

		for i := 0; i < 3; i++ {
			handler := func(ctx context.Context, e event.Event) error {
				atomic.AddInt32(&count, 1)
				return nil
			}
			suite.EventDispatcher.Subscribe(event.EventJobCompleted, handler)
		}

		evt := event.NewEvent(event.EventJobCompleted, nil)
		err := suite.EventDispatcher.Dispatch(suite.Ctx, evt)
		require.NoError(t, err)

		// Small delay to allow all handlers to complete
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, int32(3), atomic.LoadInt32(&count))
	})

	t.Run("noop dispatcher does nothing", func(t *testing.T) {
		noop := event.NewNoOpDispatcher()

		var called bool
		handler := func(ctx context.Context, e event.Event) error {
			called = true
			return nil
		}

		noop.Subscribe(event.EventJobCreated, handler)
		evt := event.NewEvent(event.EventJobCreated, nil)

		err := noop.Dispatch(suite.Ctx, evt)
		assert.NoError(t, err)
		assert.False(t, called)
	})
}

func TestEventTypes(t *testing.T) {
	t.Run("job lifecycle events", func(t *testing.T) {
		assert.Equal(t, event.EventType("job.created"), event.EventJobCreated)
		assert.Equal(t, event.EventType("job.scheduled"), event.EventJobScheduled)
		assert.Equal(t, event.EventType("job.started"), event.EventJobStarted)
		assert.Equal(t, event.EventType("job.completed"), event.EventJobCompleted)
		assert.Equal(t, event.EventType("job.failed"), event.EventJobFailed)
		assert.Equal(t, event.EventType("job.cancelled"), event.EventJobCancelled)
	})

	t.Run("bus event types", func(t *testing.T) {
		assert.Equal(t, bus.EventType("workflow.started"), bus.EventWorkflowStarted)
		assert.Equal(t, bus.EventType("workflow.completed"), bus.EventWorkflowCompleted)
		assert.Equal(t, bus.EventType("workflow.failed"), bus.EventWorkflowFailed)
		assert.Equal(t, bus.EventType("job.completed"), bus.EventJobCompleted)
		assert.Equal(t, bus.EventType("job.failed"), bus.EventJobFailed)
	})
}

func TestEventCreation(t *testing.T) {
	fixtures := testutil.NewFixtureBuilder()

	t.Run("create event with data", func(t *testing.T) {
		event := fixtures.CreateTestEvent(bus.EventWorkflowCompleted, map[string]interface{}{
			"workflow_id": "wf-123",
			"duration":    1500,
		})

		assert.NotEmpty(t, event.ID)
		assert.Equal(t, bus.EventWorkflowCompleted, event.Type)
		assert.Equal(t, "wf-123", event.Data["workflow_id"])
		assert.Equal(t, 1500, event.Data["duration"])
		assert.WithinDuration(t, time.Now(), event.Timestamp, 1*time.Second)
	})

	t.Run("create workflow completed event", func(t *testing.T) {
		event := fixtures.CreateWorkflowCompletedEvent("wf-456")

		assert.Equal(t, bus.EventWorkflowCompleted, event.Type)
		assert.Equal(t, "wf-456", event.Data["workflow_id"])
		assert.Equal(t, "completed", event.Data["status"])
	})

	t.Run("create job completed event", func(t *testing.T) {
		event := fixtures.CreateJobCompletedEvent("job-789")

		assert.Equal(t, bus.EventJobCompleted, event.Type)
		assert.Equal(t, "job-789", event.Data["job_id"])
		assert.Equal(t, "completed", event.Data["status"])
	})
}

func TestConcurrentEventOperations(t *testing.T) {
	logger := newTestLogger()
	eb := bus.NewEventBus(logger)
	defer eb.Close()

	ctx := context.Background()
	fixtures := testutil.NewFixtureBuilder()

	t.Run("concurrent publish is safe", func(t *testing.T) {
		var count int32
		wg := sync.WaitGroup{}

		subscriber := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
			atomic.AddInt32(&count, 1)
			return nil
		})

		eb.Subscribe(bus.EventType("concurrent.test"), subscriber)

		const numPublishers = 50
		wg.Add(numPublishers)

		for i := 0; i < numPublishers; i++ {
			go func() {
				defer wg.Done()
				event := fixtures.CreateTestEvent(bus.EventType("concurrent.test"), nil)
				_ = eb.Publish(ctx, event)
			}()
		}

		wg.Wait()

		// Wait a bit for all async processing
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, int32(numPublishers), atomic.LoadInt32(&count))
	})

	t.Run("concurrent subscribe is safe", func(t *testing.T) {
		wg := sync.WaitGroup{}
		const numSubscribers = 50

		wg.Add(numSubscribers)
		for i := 0; i < numSubscribers; i++ {
			go func() {
				defer wg.Done()
				subscriber := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
					return nil
				})
				eb.Subscribe(bus.EventType("concurrent.subscribe"), subscriber)
			}()
		}

		wg.Wait()
		assert.Equal(t, numSubscribers, eb.SubscriberCount(bus.EventType("concurrent.subscribe")))
	})
}

func TestEventBusShutdown(t *testing.T) {
	t.Run("close drains async buffer", func(t *testing.T) {
		logger := newTestLogger()
		eb := bus.NewEventBus(logger)
		fixtures := testutil.NewFixtureBuilder()

		var processed int32

		subscriber := bus.SubscriberFunc(func(ctx context.Context, e bus.Event) error {
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&processed, 1)
			return nil
		})

		eb.Subscribe(bus.EventType("shutdown.test"), subscriber)

		// Publish several async events
		for i := 0; i < 5; i++ {
			event := fixtures.CreateTestEvent(bus.EventType("shutdown.test"), nil)
			eb.PublishAsync(context.Background(), event)
		}

		// Close should wait for all events to be processed
		eb.Close()

		assert.Equal(t, int32(5), atomic.LoadInt32(&processed))
	})

	t.Run("publish async to closed bus is ignored", func(t *testing.T) {
		logger := newTestLogger()
		eb := bus.NewEventBus(logger)
		fixtures := testutil.NewFixtureBuilder()

		eb.Close()

		// Should not panic
		assert.NotPanics(t, func() {
			event := fixtures.CreateTestEvent(bus.EventType("closed.test"), nil)
			eb.PublishAsync(context.Background(), event)
		})
	})
}

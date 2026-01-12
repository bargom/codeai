//go:build integration

// Package webhook provides integration tests for the webhook service.
package webhook

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bargom/codeai/internal/event/bus"
	webhookrepo "github.com/bargom/codeai/internal/webhook/repository"
	"github.com/bargom/codeai/test/integration/testutil"
)

// WebhookTestSuite holds resources for webhook integration tests.
type WebhookTestSuite struct {
	Repository    *webhookrepo.MemoryRepository
	WebhookServer *testutil.WebhookTestServer
	Fixtures      *testutil.FixtureBuilder
	Ctx           context.Context
	Cancel        context.CancelFunc
}

// NewWebhookTestSuite creates a new webhook test suite.
func NewWebhookTestSuite(t *testing.T) *WebhookTestSuite {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	return &WebhookTestSuite{
		Repository:    webhookrepo.NewMemoryRepository(),
		WebhookServer: testutil.NewWebhookTestServer(),
		Fixtures:      testutil.NewFixtureBuilder(),
		Ctx:           ctx,
		Cancel:        cancel,
	}
}

// Teardown cleans up the test suite.
func (s *WebhookTestSuite) Teardown() {
	s.WebhookServer.Close()
	s.Cancel()
}

// Reset clears the repository and webhook server.
func (s *WebhookTestSuite) Reset() {
	s.Repository = webhookrepo.NewMemoryRepository()
	s.WebhookServer.Reset()
}

func TestWebhookRepositoryCRUD(t *testing.T) {
	suite := NewWebhookTestSuite(t)
	defer suite.Teardown()

	t.Run("create and retrieve webhook", func(t *testing.T) {
		suite.Reset()

		webhook := suite.Fixtures.CreateTestWebhook(
			suite.WebhookServer.URL(),
			bus.EventWorkflowCompleted,
		)

		err := suite.Repository.CreateWebhook(suite.Ctx, webhook)
		require.NoError(t, err)

		retrieved, err := suite.Repository.GetWebhook(suite.Ctx, webhook.ID)
		require.NoError(t, err)
		assert.Equal(t, webhook.ID, retrieved.ID)
		assert.Equal(t, webhook.URL, retrieved.URL)
		assert.True(t, retrieved.Active)
	})

	t.Run("update webhook configuration", func(t *testing.T) {
		suite.Reset()

		webhook := suite.Fixtures.CreateTestWebhook(
			suite.WebhookServer.URL(),
			bus.EventWorkflowCompleted,
		)
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, webhook))

		newURL := "https://new-endpoint.example.com/webhook"
		update := webhookrepo.WebhookUpdate{
			URL: &newURL,
		}

		err := suite.Repository.UpdateWebhook(suite.Ctx, webhook.ID, update)
		require.NoError(t, err)

		retrieved, err := suite.Repository.GetWebhook(suite.Ctx, webhook.ID)
		require.NoError(t, err)
		assert.Equal(t, newURL, retrieved.URL)
	})

	t.Run("delete webhook", func(t *testing.T) {
		suite.Reset()

		webhook := suite.Fixtures.CreateTestWebhook(suite.WebhookServer.URL())
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, webhook))

		err := suite.Repository.DeleteWebhook(suite.Ctx, webhook.ID)
		require.NoError(t, err)

		_, err = suite.Repository.GetWebhook(suite.Ctx, webhook.ID)
		assert.Error(t, err)
	})

	t.Run("get non-existent webhook returns error", func(t *testing.T) {
		suite.Reset()

		_, err := suite.Repository.GetWebhook(suite.Ctx, "non-existent")
		assert.Error(t, err)
	})

	t.Run("create duplicate webhook fails", func(t *testing.T) {
		suite.Reset()

		webhook := suite.Fixtures.CreateTestWebhook(suite.WebhookServer.URL())
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, webhook))

		err := suite.Repository.CreateWebhook(suite.Ctx, webhook)
		assert.Error(t, err)
	})
}

func TestWebhookListingAndFiltering(t *testing.T) {
	suite := NewWebhookTestSuite(t)
	defer suite.Teardown()

	t.Run("list webhooks by active status", func(t *testing.T) {
		suite.Reset()

		// Create active webhook
		activeWebhook := suite.Fixtures.CreateTestWebhook(
			suite.WebhookServer.URL(),
			bus.EventWorkflowCompleted,
		)
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, activeWebhook))

		// Create inactive webhook
		inactiveWebhook := suite.Fixtures.CreateDisabledWebhook(
			suite.WebhookServer.URL(),
			bus.EventJobCompleted,
		)
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, inactiveWebhook))

		// Filter active only
		active := true
		filter := webhookrepo.WebhookFilter{
			Active: &active,
		}
		webhooks, err := suite.Repository.ListWebhooks(suite.Ctx, filter)
		require.NoError(t, err)
		assert.Len(t, webhooks, 1)
		assert.True(t, webhooks[0].Active)

		// Filter inactive only
		active = false
		filter.Active = &active
		webhooks, err = suite.Repository.ListWebhooks(suite.Ctx, filter)
		require.NoError(t, err)
		assert.Len(t, webhooks, 1)
		assert.False(t, webhooks[0].Active)
	})

	t.Run("list webhooks with pagination", func(t *testing.T) {
		suite.Reset()

		// Create 5 webhooks
		for i := 0; i < 5; i++ {
			webhook := suite.Fixtures.CreateTestWebhook(suite.WebhookServer.URL())
			require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, webhook))
		}

		// Get first page
		filter := webhookrepo.WebhookFilter{
			Limit:  2,
			Offset: 0,
		}
		webhooks, err := suite.Repository.ListWebhooks(suite.Ctx, filter)
		require.NoError(t, err)
		assert.Len(t, webhooks, 2)

		// Get second page
		filter.Offset = 2
		webhooks, err = suite.Repository.ListWebhooks(suite.Ctx, filter)
		require.NoError(t, err)
		assert.Len(t, webhooks, 2)
	})

	t.Run("get webhooks by event type", func(t *testing.T) {
		suite.Reset()

		// Create webhook for workflow events
		workflowWebhook := suite.Fixtures.CreateTestWebhook(
			suite.WebhookServer.URL(),
			bus.EventWorkflowCompleted,
			bus.EventWorkflowFailed,
		)
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, workflowWebhook))

		// Create webhook for job events
		jobWebhook := suite.Fixtures.CreateTestWebhook(
			suite.WebhookServer.URL(),
			bus.EventJobCompleted,
		)
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, jobWebhook))

		// Get webhooks for workflow.completed
		webhooks, err := suite.Repository.GetWebhooksByEvent(suite.Ctx, bus.EventWorkflowCompleted)
		require.NoError(t, err)
		assert.Len(t, webhooks, 1)
		assert.Equal(t, workflowWebhook.ID, webhooks[0].ID)

		// Get webhooks for job.completed
		webhooks, err = suite.Repository.GetWebhooksByEvent(suite.Ctx, bus.EventJobCompleted)
		require.NoError(t, err)
		assert.Len(t, webhooks, 1)
		assert.Equal(t, jobWebhook.ID, webhooks[0].ID)
	})

	t.Run("webhook with no events receives all events", func(t *testing.T) {
		suite.Reset()

		// Create webhook without specific events (catches all)
		webhook := suite.Fixtures.CreateTestWebhook(suite.WebhookServer.URL())
		webhook.Events = nil // No specific events
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, webhook))

		// Should receive any event type
		webhooks, err := suite.Repository.GetWebhooksByEvent(suite.Ctx, bus.EventWorkflowCompleted)
		require.NoError(t, err)
		assert.Len(t, webhooks, 1)

		webhooks, err = suite.Repository.GetWebhooksByEvent(suite.Ctx, bus.EventJobCompleted)
		require.NoError(t, err)
		assert.Len(t, webhooks, 1)
	})
}

func TestWebhookFailureTracking(t *testing.T) {
	suite := NewWebhookTestSuite(t)
	defer suite.Teardown()

	t.Run("increment failure count", func(t *testing.T) {
		suite.Reset()

		webhook := suite.Fixtures.CreateTestWebhook(suite.WebhookServer.URL())
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, webhook))

		// Increment failure count
		err := suite.Repository.IncrementFailureCount(suite.Ctx, webhook.ID)
		require.NoError(t, err)

		retrieved, err := suite.Repository.GetWebhook(suite.Ctx, webhook.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, retrieved.FailureCount)

		// Increment again
		err = suite.Repository.IncrementFailureCount(suite.Ctx, webhook.ID)
		require.NoError(t, err)

		retrieved, err = suite.Repository.GetWebhook(suite.Ctx, webhook.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, retrieved.FailureCount)
	})

	t.Run("reset failure count", func(t *testing.T) {
		suite.Reset()

		webhook := suite.Fixtures.CreateTestWebhook(suite.WebhookServer.URL())
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, webhook))

		// Increment multiple times
		for i := 0; i < 5; i++ {
			require.NoError(t, suite.Repository.IncrementFailureCount(suite.Ctx, webhook.ID))
		}

		retrieved, err := suite.Repository.GetWebhook(suite.Ctx, webhook.ID)
		require.NoError(t, err)
		assert.Equal(t, 5, retrieved.FailureCount)

		// Reset
		err = suite.Repository.ResetFailureCount(suite.Ctx, webhook.ID)
		require.NoError(t, err)

		retrieved, err = suite.Repository.GetWebhook(suite.Ctx, webhook.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, retrieved.FailureCount)
	})

	t.Run("disable webhook after failures", func(t *testing.T) {
		suite.Reset()

		webhook := suite.Fixtures.CreateTestWebhook(suite.WebhookServer.URL())
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, webhook))

		// Disable webhook
		active := false
		update := webhookrepo.WebhookUpdate{
			Active: &active,
		}
		err := suite.Repository.UpdateWebhook(suite.Ctx, webhook.ID, update)
		require.NoError(t, err)

		retrieved, err := suite.Repository.GetWebhook(suite.Ctx, webhook.ID)
		require.NoError(t, err)
		assert.False(t, retrieved.Active)

		// Inactive webhook should not be returned for events
		webhooks, err := suite.Repository.GetWebhooksByEvent(suite.Ctx, bus.EventWorkflowCompleted)
		require.NoError(t, err)
		assert.Len(t, webhooks, 0)
	})
}

func TestWebhookDeliveryTracking(t *testing.T) {
	suite := NewWebhookTestSuite(t)
	defer suite.Teardown()

	t.Run("save and retrieve delivery", func(t *testing.T) {
		suite.Reset()

		webhook := suite.Fixtures.CreateTestWebhook(suite.WebhookServer.URL())
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, webhook))

		delivery := suite.Fixtures.CreateWebhookDelivery(
			webhook.ID,
			bus.EventWorkflowCompleted,
			true,
		)
		err := suite.Repository.SaveDelivery(suite.Ctx, delivery)
		require.NoError(t, err)

		retrieved, err := suite.Repository.GetDelivery(suite.Ctx, delivery.ID)
		require.NoError(t, err)
		assert.Equal(t, delivery.ID, retrieved.ID)
		assert.Equal(t, webhook.ID, retrieved.WebhookID)
		assert.True(t, retrieved.Success)
	})

	t.Run("list deliveries for webhook", func(t *testing.T) {
		suite.Reset()

		webhook := suite.Fixtures.CreateTestWebhook(suite.WebhookServer.URL())
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, webhook))

		// Create multiple deliveries
		for i := 0; i < 5; i++ {
			success := i%2 == 0
			delivery := suite.Fixtures.CreateWebhookDelivery(
				webhook.ID,
				bus.EventWorkflowCompleted,
				success,
			)
			require.NoError(t, suite.Repository.SaveDelivery(suite.Ctx, delivery))
		}

		// Get all deliveries
		filter := webhookrepo.DeliveryFilter{
			Limit: 10,
		}
		deliveries, err := suite.Repository.ListDeliveries(suite.Ctx, webhook.ID, filter)
		require.NoError(t, err)
		assert.Len(t, deliveries, 5)

		// Filter by success
		success := true
		filter.Success = &success
		deliveries, err = suite.Repository.ListDeliveries(suite.Ctx, webhook.ID, filter)
		require.NoError(t, err)
		assert.Len(t, deliveries, 3) // 0, 2, 4 are successful
	})

	t.Run("get failed deliveries for retry", func(t *testing.T) {
		suite.Reset()

		webhook := suite.Fixtures.CreateTestWebhook(suite.WebhookServer.URL())
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, webhook))

		// Create failed delivery with past retry time
		failedDelivery := suite.Fixtures.CreateWebhookDelivery(
			webhook.ID,
			bus.EventWorkflowCompleted,
			false,
		)
		pastRetry := time.Now().Add(-1 * time.Hour)
		failedDelivery.NextRetryAt = &pastRetry
		require.NoError(t, suite.Repository.SaveDelivery(suite.Ctx, failedDelivery))

		// Create failed delivery with future retry time
		futureDelivery := suite.Fixtures.CreateWebhookDelivery(
			webhook.ID,
			bus.EventWorkflowCompleted,
			false,
		)
		futureRetry := time.Now().Add(1 * time.Hour)
		futureDelivery.NextRetryAt = &futureRetry
		require.NoError(t, suite.Repository.SaveDelivery(suite.Ctx, futureDelivery))

		// Only past retry should be returned
		deliveries, err := suite.Repository.GetFailedDeliveries(suite.Ctx, 10)
		require.NoError(t, err)
		assert.Len(t, deliveries, 1)
		assert.Equal(t, failedDelivery.ID, deliveries[0].ID)
	})

	t.Run("update delivery retry time", func(t *testing.T) {
		suite.Reset()

		webhook := suite.Fixtures.CreateTestWebhook(suite.WebhookServer.URL())
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, webhook))

		delivery := suite.Fixtures.CreateWebhookDelivery(
			webhook.ID,
			bus.EventWorkflowCompleted,
			false,
		)
		require.NoError(t, suite.Repository.SaveDelivery(suite.Ctx, delivery))

		nextRetry := time.Now().Add(5 * time.Minute)
		err := suite.Repository.UpdateDeliveryRetry(suite.Ctx, delivery.ID, nextRetry)
		require.NoError(t, err)

		retrieved, err := suite.Repository.GetDelivery(suite.Ctx, delivery.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.NextRetryAt)
		assert.WithinDuration(t, nextRetry, *retrieved.NextRetryAt, time.Second)
	})

	t.Run("delete old deliveries", func(t *testing.T) {
		suite.Reset()

		webhook := suite.Fixtures.CreateTestWebhook(suite.WebhookServer.URL())
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, webhook))

		// Create old delivery
		oldDelivery := suite.Fixtures.CreateWebhookDelivery(
			webhook.ID,
			bus.EventWorkflowCompleted,
			true,
		)
		oldDelivery.DeliveredAt = time.Now().Add(-48 * time.Hour)
		require.NoError(t, suite.Repository.SaveDelivery(suite.Ctx, oldDelivery))

		// Create recent delivery
		recentDelivery := suite.Fixtures.CreateWebhookDelivery(
			webhook.ID,
			bus.EventWorkflowCompleted,
			true,
		)
		require.NoError(t, suite.Repository.SaveDelivery(suite.Ctx, recentDelivery))

		// Delete deliveries older than 24 hours
		cutoff := time.Now().Add(-24 * time.Hour)
		deleted, err := suite.Repository.DeleteOldDeliveries(suite.Ctx, cutoff)
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		// Old delivery should be gone
		_, err = suite.Repository.GetDelivery(suite.Ctx, oldDelivery.ID)
		assert.Error(t, err)

		// Recent delivery should still exist
		_, err = suite.Repository.GetDelivery(suite.Ctx, recentDelivery.ID)
		assert.NoError(t, err)
	})
}

func TestWebhookTestServer(t *testing.T) {
	t.Run("captures webhook deliveries", func(t *testing.T) {
		server := testutil.NewWebhookTestServer()
		defer server.Close()

		assert.Equal(t, 0, server.DeliveryCount())
		assert.NotEmpty(t, server.URL())
	})

	t.Run("can simulate failures", func(t *testing.T) {
		server := testutil.NewWebhookTestServer()
		defer server.Close()

		server.SetFailRequests(true)
		assert.True(t, server.FailRequests)

		server.SetFailRequests(false)
		assert.False(t, server.FailRequests)
	})

	t.Run("reset clears deliveries", func(t *testing.T) {
		server := testutil.NewWebhookTestServer()
		defer server.Close()

		server.SetFailRequests(true)
		server.Reset()

		assert.False(t, server.FailRequests)
		assert.Equal(t, 0, server.DeliveryCount())
	})
}

func TestConcurrentWebhookOperations(t *testing.T) {
	suite := NewWebhookTestSuite(t)
	defer suite.Teardown()

	t.Run("concurrent webhook creation", func(t *testing.T) {
		suite.Reset()

		const numWebhooks = 50
		done := make(chan bool, numWebhooks)

		for i := 0; i < numWebhooks; i++ {
			go func() {
				webhook := suite.Fixtures.CreateTestWebhook(suite.WebhookServer.URL())
				err := suite.Repository.CreateWebhook(suite.Ctx, webhook)
				if err != nil {
					t.Errorf("failed to create webhook: %v", err)
				}
				done <- true
			}()
		}

		for i := 0; i < numWebhooks; i++ {
			<-done
		}

		webhooks, err := suite.Repository.ListWebhooks(suite.Ctx, webhookrepo.WebhookFilter{Limit: 100})
		require.NoError(t, err)
		assert.Len(t, webhooks, numWebhooks)
	})

	t.Run("concurrent delivery saves", func(t *testing.T) {
		suite.Reset()

		webhook := suite.Fixtures.CreateTestWebhook(suite.WebhookServer.URL())
		require.NoError(t, suite.Repository.CreateWebhook(suite.Ctx, webhook))

		const numDeliveries = 100
		done := make(chan bool, numDeliveries)

		for i := 0; i < numDeliveries; i++ {
			go func() {
				delivery := suite.Fixtures.CreateWebhookDelivery(
					webhook.ID,
					bus.EventWorkflowCompleted,
					true,
				)
				err := suite.Repository.SaveDelivery(suite.Ctx, delivery)
				if err != nil {
					t.Errorf("failed to save delivery: %v", err)
				}
				done <- true
			}()
		}

		for i := 0; i < numDeliveries; i++ {
			<-done
		}

		deliveries, err := suite.Repository.ListDeliveries(suite.Ctx, webhook.ID, webhookrepo.DeliveryFilter{Limit: 200})
		require.NoError(t, err)
		assert.Len(t, deliveries, numDeliveries)
	})
}

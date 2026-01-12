package repository

import (
	"context"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/event/bus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRepository_CreateWebhook(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	webhook := &WebhookConfig{
		ID:        "wh-1",
		URL:       "https://example.com/webhook",
		Events:    []bus.EventType{bus.EventJobCompleted},
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.CreateWebhook(ctx, webhook)
	require.NoError(t, err)

	// Verify it was created
	retrieved, err := repo.GetWebhook(ctx, "wh-1")
	require.NoError(t, err)
	assert.Equal(t, webhook.URL, retrieved.URL)
	assert.True(t, retrieved.Active)
}

func TestMemoryRepository_CreateWebhook_Duplicate(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	webhook := &WebhookConfig{
		ID:  "wh-1",
		URL: "https://example.com/webhook",
	}

	err := repo.CreateWebhook(ctx, webhook)
	require.NoError(t, err)

	err = repo.CreateWebhook(ctx, webhook)
	assert.Error(t, err)
}

func TestMemoryRepository_GetWebhook_NotFound(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	_, err := repo.GetWebhook(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestMemoryRepository_ListWebhooks(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Create some webhooks
	for i := 0; i < 5; i++ {
		active := i%2 == 0
		webhook := &WebhookConfig{
			ID:     string(rune('A' + i)),
			URL:    "https://example.com/webhook",
			Active: active,
		}
		require.NoError(t, repo.CreateWebhook(ctx, webhook))
	}

	// List all
	webhooks, err := repo.ListWebhooks(ctx, WebhookFilter{})
	require.NoError(t, err)
	assert.Len(t, webhooks, 5)

	// List active only
	active := true
	webhooks, err = repo.ListWebhooks(ctx, WebhookFilter{Active: &active})
	require.NoError(t, err)
	assert.Len(t, webhooks, 3)

	// List with limit
	webhooks, err = repo.ListWebhooks(ctx, WebhookFilter{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, webhooks, 2)
}

func TestMemoryRepository_GetWebhooksByEvent(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Create webhooks with different event subscriptions
	webhook1 := &WebhookConfig{
		ID:     "wh-1",
		URL:    "https://example.com/1",
		Events: []bus.EventType{bus.EventJobCompleted, bus.EventJobFailed},
		Active: true,
	}
	webhook2 := &WebhookConfig{
		ID:     "wh-2",
		URL:    "https://example.com/2",
		Events: []bus.EventType{bus.EventWorkflowCompleted},
		Active: true,
	}
	webhook3 := &WebhookConfig{
		ID:     "wh-3",
		URL:    "https://example.com/3",
		Events: []bus.EventType{}, // All events
		Active: true,
	}
	webhook4 := &WebhookConfig{
		ID:     "wh-4",
		URL:    "https://example.com/4",
		Events: []bus.EventType{bus.EventJobCompleted},
		Active: false, // Inactive
	}

	require.NoError(t, repo.CreateWebhook(ctx, webhook1))
	require.NoError(t, repo.CreateWebhook(ctx, webhook2))
	require.NoError(t, repo.CreateWebhook(ctx, webhook3))
	require.NoError(t, repo.CreateWebhook(ctx, webhook4))

	// Get webhooks for JobCompleted
	webhooks, err := repo.GetWebhooksByEvent(ctx, bus.EventJobCompleted)
	require.NoError(t, err)
	assert.Len(t, webhooks, 2) // wh-1 and wh-3 (all events)

	// Get webhooks for WorkflowCompleted
	webhooks, err = repo.GetWebhooksByEvent(ctx, bus.EventWorkflowCompleted)
	require.NoError(t, err)
	assert.Len(t, webhooks, 2) // wh-2 and wh-3
}

func TestMemoryRepository_UpdateWebhook(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	webhook := &WebhookConfig{
		ID:        "wh-1",
		URL:       "https://example.com/webhook",
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repo.CreateWebhook(ctx, webhook))

	// Update URL
	newURL := "https://example.com/new-webhook"
	err := repo.UpdateWebhook(ctx, "wh-1", WebhookUpdate{URL: &newURL})
	require.NoError(t, err)

	retrieved, err := repo.GetWebhook(ctx, "wh-1")
	require.NoError(t, err)
	assert.Equal(t, newURL, retrieved.URL)
	assert.True(t, retrieved.Active)
}

func TestMemoryRepository_DeleteWebhook(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	webhook := &WebhookConfig{ID: "wh-1", URL: "https://example.com"}
	require.NoError(t, repo.CreateWebhook(ctx, webhook))

	err := repo.DeleteWebhook(ctx, "wh-1")
	require.NoError(t, err)

	_, err = repo.GetWebhook(ctx, "wh-1")
	assert.Error(t, err)
}

func TestMemoryRepository_FailureCount(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	webhook := &WebhookConfig{ID: "wh-1", URL: "https://example.com", FailureCount: 0}
	require.NoError(t, repo.CreateWebhook(ctx, webhook))

	// Increment
	require.NoError(t, repo.IncrementFailureCount(ctx, "wh-1"))
	require.NoError(t, repo.IncrementFailureCount(ctx, "wh-1"))
	require.NoError(t, repo.IncrementFailureCount(ctx, "wh-1"))

	retrieved, err := repo.GetWebhook(ctx, "wh-1")
	require.NoError(t, err)
	assert.Equal(t, 3, retrieved.FailureCount)

	// Reset
	require.NoError(t, repo.ResetFailureCount(ctx, "wh-1"))

	retrieved, err = repo.GetWebhook(ctx, "wh-1")
	require.NoError(t, err)
	assert.Equal(t, 0, retrieved.FailureCount)
}

func TestMemoryRepository_Deliveries(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Create deliveries
	for i := 0; i < 3; i++ {
		delivery := &WebhookDelivery{
			ID:          string(rune('A' + i)),
			WebhookID:   "wh-1",
			EventID:     "evt-1",
			Success:     i%2 == 0,
			DeliveredAt: time.Now(),
		}
		require.NoError(t, repo.SaveDelivery(ctx, delivery))
	}

	// List all deliveries for webhook
	deliveries, err := repo.ListDeliveries(ctx, "wh-1", DeliveryFilter{})
	require.NoError(t, err)
	assert.Len(t, deliveries, 3)

	// List successful only
	success := true
	deliveries, err = repo.ListDeliveries(ctx, "wh-1", DeliveryFilter{Success: &success})
	require.NoError(t, err)
	assert.Len(t, deliveries, 2)
}

func TestMemoryRepository_GetFailedDeliveries(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	now := time.Now()

	// Create failed deliveries with different retry times
	d1 := &WebhookDelivery{
		ID:          "d-1",
		WebhookID:   "wh-1",
		Success:     false,
		DeliveredAt: now.Add(-1 * time.Hour),
		NextRetryAt: ptr(now.Add(-10 * time.Minute)), // Past - should retry
	}
	d2 := &WebhookDelivery{
		ID:          "d-2",
		WebhookID:   "wh-1",
		Success:     false,
		DeliveredAt: now.Add(-30 * time.Minute),
		NextRetryAt: ptr(now.Add(10 * time.Minute)), // Future - not ready
	}
	d3 := &WebhookDelivery{
		ID:          "d-3",
		WebhookID:   "wh-1",
		Success:     true, // Successful - should not be in failed
		DeliveredAt: now,
	}

	require.NoError(t, repo.SaveDelivery(ctx, d1))
	require.NoError(t, repo.SaveDelivery(ctx, d2))
	require.NoError(t, repo.SaveDelivery(ctx, d3))

	failed, err := repo.GetFailedDeliveries(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, failed, 1)
	assert.Equal(t, "d-1", failed[0].ID)
}

func TestMemoryRepository_DeleteOldDeliveries(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	now := time.Now()

	// Create deliveries at different times
	// i=0: now (0h ago)
	// i=1: -24h ago
	// i=2: -48h ago
	// i=3: -72h ago
	// i=4: -96h ago
	for i := 0; i < 5; i++ {
		delivery := &WebhookDelivery{
			ID:          string(rune('A' + i)),
			WebhookID:   "wh-1",
			DeliveredAt: now.Add(time.Duration(-i*24) * time.Hour),
		}
		require.NoError(t, repo.SaveDelivery(ctx, delivery))
	}

	// Delete deliveries older than 2 days (before -48h)
	// This should delete: -72h (D), -96h (E) = 2 deliveries
	// Keeping: 0h (A), -24h (B), -48h (C) = 3 deliveries
	deleted, err := repo.DeleteOldDeliveries(ctx, now.Add(-48*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// Verify remaining
	deliveries, err := repo.ListDeliveries(ctx, "wh-1", DeliveryFilter{})
	require.NoError(t, err)
	assert.Len(t, deliveries, 3)
}

func ptr[T any](v T) *T {
	return &v
}

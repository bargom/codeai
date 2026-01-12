package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bargom/codeai/internal/event/bus"
)

// MemoryRepository is an in-memory implementation of WebhookRepository.
// Useful for testing and development.
type MemoryRepository struct {
	mu         sync.RWMutex
	webhooks   map[string]*WebhookConfig
	deliveries map[string]*WebhookDelivery
}

// NewMemoryRepository creates a new in-memory webhook repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		webhooks:   make(map[string]*WebhookConfig),
		deliveries: make(map[string]*WebhookDelivery),
	}
}

// CreateWebhook creates a new webhook configuration.
func (r *MemoryRepository) CreateWebhook(ctx context.Context, webhook *WebhookConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.webhooks[webhook.ID]; exists {
		return fmt.Errorf("webhook %s already exists", webhook.ID)
	}

	webhookCopy := *webhook
	r.webhooks[webhook.ID] = &webhookCopy
	return nil
}

// GetWebhook retrieves a webhook by its ID.
func (r *MemoryRepository) GetWebhook(ctx context.Context, webhookID string) (*WebhookConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	webhook, exists := r.webhooks[webhookID]
	if !exists {
		return nil, fmt.Errorf("webhook %s not found", webhookID)
	}

	webhookCopy := *webhook
	return &webhookCopy, nil
}

// ListWebhooks retrieves webhooks matching the filter criteria.
func (r *MemoryRepository) ListWebhooks(ctx context.Context, filter WebhookFilter) ([]WebhookConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var webhooks []WebhookConfig
	for _, webhook := range r.webhooks {
		if filter.Active != nil && webhook.Active != *filter.Active {
			continue
		}
		webhooks = append(webhooks, *webhook)
	}

	// Apply offset and limit
	if filter.Offset > 0 {
		if filter.Offset >= len(webhooks) {
			return nil, nil
		}
		webhooks = webhooks[filter.Offset:]
	}

	if filter.Limit > 0 && len(webhooks) > filter.Limit {
		webhooks = webhooks[:filter.Limit]
	}

	return webhooks, nil
}

// GetWebhooksByEvent retrieves active webhooks subscribed to an event type.
func (r *MemoryRepository) GetWebhooksByEvent(ctx context.Context, eventType bus.EventType) ([]WebhookConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var webhooks []WebhookConfig
	for _, webhook := range r.webhooks {
		if !webhook.Active {
			continue
		}

		// If no events specified, webhook receives all events
		if len(webhook.Events) == 0 {
			webhooks = append(webhooks, *webhook)
			continue
		}

		// Check if event type is in the list
		for _, et := range webhook.Events {
			if et == eventType {
				webhooks = append(webhooks, *webhook)
				break
			}
		}
	}

	return webhooks, nil
}

// UpdateWebhook updates a webhook configuration.
func (r *MemoryRepository) UpdateWebhook(ctx context.Context, webhookID string, update WebhookUpdate) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	webhook, exists := r.webhooks[webhookID]
	if !exists {
		return fmt.Errorf("webhook %s not found", webhookID)
	}

	if update.URL != nil {
		webhook.URL = *update.URL
	}
	if update.Events != nil {
		webhook.Events = update.Events
	}
	if update.Secret != nil {
		webhook.Secret = *update.Secret
	}
	if update.Headers != nil {
		webhook.Headers = update.Headers
	}
	if update.Active != nil {
		webhook.Active = *update.Active
	}
	if update.LastDelivery != nil {
		webhook.LastDelivery = update.LastDelivery
	}
	if update.FailureCount != nil {
		webhook.FailureCount = *update.FailureCount
	}
	if update.Metadata != nil {
		webhook.Metadata = update.Metadata
	}
	webhook.UpdatedAt = time.Now()

	return nil
}

// DeleteWebhook removes a webhook configuration.
func (r *MemoryRepository) DeleteWebhook(ctx context.Context, webhookID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.webhooks[webhookID]; !exists {
		return fmt.Errorf("webhook %s not found", webhookID)
	}

	delete(r.webhooks, webhookID)
	return nil
}

// IncrementFailureCount increments the failure count for a webhook.
func (r *MemoryRepository) IncrementFailureCount(ctx context.Context, webhookID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	webhook, exists := r.webhooks[webhookID]
	if !exists {
		return fmt.Errorf("webhook %s not found", webhookID)
	}

	webhook.FailureCount++
	webhook.UpdatedAt = time.Now()
	return nil
}

// ResetFailureCount resets the failure count for a webhook.
func (r *MemoryRepository) ResetFailureCount(ctx context.Context, webhookID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	webhook, exists := r.webhooks[webhookID]
	if !exists {
		return fmt.Errorf("webhook %s not found", webhookID)
	}

	webhook.FailureCount = 0
	webhook.UpdatedAt = time.Now()
	return nil
}

// SaveDelivery persists a delivery attempt.
func (r *MemoryRepository) SaveDelivery(ctx context.Context, delivery *WebhookDelivery) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	deliveryCopy := *delivery
	r.deliveries[delivery.ID] = &deliveryCopy
	return nil
}

// GetDelivery retrieves a delivery by its ID.
func (r *MemoryRepository) GetDelivery(ctx context.Context, deliveryID string) (*WebhookDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	delivery, exists := r.deliveries[deliveryID]
	if !exists {
		return nil, fmt.Errorf("delivery %s not found", deliveryID)
	}

	deliveryCopy := *delivery
	return &deliveryCopy, nil
}

// ListDeliveries retrieves deliveries for a webhook.
func (r *MemoryRepository) ListDeliveries(ctx context.Context, webhookID string, filter DeliveryFilter) ([]WebhookDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var deliveries []WebhookDelivery
	for _, delivery := range r.deliveries {
		if delivery.WebhookID != webhookID {
			continue
		}
		if filter.Success != nil && delivery.Success != *filter.Success {
			continue
		}
		deliveries = append(deliveries, *delivery)
	}

	// Apply offset and limit
	if filter.Offset > 0 {
		if filter.Offset >= len(deliveries) {
			return nil, nil
		}
		deliveries = deliveries[filter.Offset:]
	}

	if filter.Limit > 0 && len(deliveries) > filter.Limit {
		deliveries = deliveries[:filter.Limit]
	}

	return deliveries, nil
}

// GetFailedDeliveries retrieves failed deliveries ready for retry.
func (r *MemoryRepository) GetFailedDeliveries(ctx context.Context, limit int) ([]WebhookDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	var deliveries []WebhookDelivery

	for _, delivery := range r.deliveries {
		if delivery.Success {
			continue
		}
		if delivery.NextRetryAt == nil || now.Before(*delivery.NextRetryAt) {
			continue
		}
		deliveries = append(deliveries, *delivery)

		if limit > 0 && len(deliveries) >= limit {
			break
		}
	}

	return deliveries, nil
}

// UpdateDeliveryRetry updates the next retry time for a delivery.
func (r *MemoryRepository) UpdateDeliveryRetry(ctx context.Context, deliveryID string, nextRetryAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delivery, exists := r.deliveries[deliveryID]
	if !exists {
		return fmt.Errorf("delivery %s not found", deliveryID)
	}

	delivery.NextRetryAt = &nextRetryAt
	return nil
}

// DeleteOldDeliveries removes deliveries older than the specified time.
func (r *MemoryRepository) DeleteOldDeliveries(ctx context.Context, before time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var deleted int64
	for id, delivery := range r.deliveries {
		if delivery.DeliveredAt.Before(before) {
			delete(r.deliveries, id)
			deleted++
		}
	}

	return deleted, nil
}

// Package repository provides data access for webhook configurations and deliveries.
package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bargom/codeai/internal/event/bus"
)

// WebhookConfig represents a webhook subscription configuration.
type WebhookConfig struct {
	ID           string                 `json:"id" bson:"_id"`
	URL          string                 `json:"url" bson:"url"`
	Events       []bus.EventType        `json:"events" bson:"events"`
	Secret       string                 `json:"-" bson:"secret"` // Hidden in JSON responses
	Headers      map[string]string      `json:"headers,omitempty" bson:"headers,omitempty"`
	Active       bool                   `json:"active" bson:"active"`
	CreatedAt    time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" bson:"updated_at"`
	LastDelivery *time.Time             `json:"last_delivery,omitempty" bson:"last_delivery,omitempty"`
	FailureCount int                    `json:"failure_count" bson:"failure_count"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
}

// WebhookUpdate represents fields to update on a webhook.
type WebhookUpdate struct {
	URL          *string
	Events       []bus.EventType
	Secret       *string
	Headers      map[string]string
	Active       *bool
	LastDelivery *time.Time
	FailureCount *int
	Metadata     map[string]interface{}
}

// WebhookFilter specifies criteria for filtering webhooks.
type WebhookFilter struct {
	Active *bool
	Limit  int
	Offset int
}

// WebhookDelivery represents a webhook delivery attempt.
type WebhookDelivery struct {
	ID           string          `json:"id" bson:"_id"`
	WebhookID    string          `json:"webhook_id" bson:"webhook_id"`
	EventID      string          `json:"event_id" bson:"event_id"`
	EventType    bus.EventType   `json:"event_type" bson:"event_type"`
	URL          string          `json:"url" bson:"url"`
	StatusCode   int             `json:"status_code" bson:"status_code"`
	RequestBody  json.RawMessage `json:"request_body" bson:"request_body"`
	ResponseBody string          `json:"response_body,omitempty" bson:"response_body,omitempty"`
	Duration     time.Duration   `json:"duration_ms" bson:"duration_ms"`
	Attempts     int             `json:"attempts" bson:"attempts"`
	Success      bool            `json:"success" bson:"success"`
	Error        string          `json:"error,omitempty" bson:"error,omitempty"`
	DeliveredAt  time.Time       `json:"delivered_at" bson:"delivered_at"`
	NextRetryAt  *time.Time      `json:"next_retry_at,omitempty" bson:"next_retry_at,omitempty"`
}

// DeliveryFilter specifies criteria for filtering deliveries.
type DeliveryFilter struct {
	Success *bool
	Limit   int
	Offset  int
}

// WebhookRepository defines the interface for webhook persistence.
type WebhookRepository interface {
	// CreateWebhook creates a new webhook configuration.
	CreateWebhook(ctx context.Context, webhook *WebhookConfig) error

	// GetWebhook retrieves a webhook by its ID.
	GetWebhook(ctx context.Context, webhookID string) (*WebhookConfig, error)

	// ListWebhooks retrieves webhooks matching the filter criteria.
	ListWebhooks(ctx context.Context, filter WebhookFilter) ([]WebhookConfig, error)

	// GetWebhooksByEvent retrieves active webhooks subscribed to an event type.
	GetWebhooksByEvent(ctx context.Context, eventType bus.EventType) ([]WebhookConfig, error)

	// UpdateWebhook updates a webhook configuration.
	UpdateWebhook(ctx context.Context, webhookID string, update WebhookUpdate) error

	// DeleteWebhook removes a webhook configuration.
	DeleteWebhook(ctx context.Context, webhookID string) error

	// IncrementFailureCount increments the failure count for a webhook.
	IncrementFailureCount(ctx context.Context, webhookID string) error

	// ResetFailureCount resets the failure count for a webhook.
	ResetFailureCount(ctx context.Context, webhookID string) error

	// Delivery tracking
	// SaveDelivery persists a delivery attempt.
	SaveDelivery(ctx context.Context, delivery *WebhookDelivery) error

	// GetDelivery retrieves a delivery by its ID.
	GetDelivery(ctx context.Context, deliveryID string) (*WebhookDelivery, error)

	// ListDeliveries retrieves deliveries for a webhook.
	ListDeliveries(ctx context.Context, webhookID string, filter DeliveryFilter) ([]WebhookDelivery, error)

	// GetFailedDeliveries retrieves failed deliveries ready for retry.
	GetFailedDeliveries(ctx context.Context, limit int) ([]WebhookDelivery, error)

	// UpdateDeliveryRetry updates the next retry time for a delivery.
	UpdateDeliveryRetry(ctx context.Context, deliveryID string, nextRetryAt time.Time) error

	// DeleteOldDeliveries removes deliveries older than the specified time.
	DeleteOldDeliveries(ctx context.Context, before time.Time) (int64, error)
}

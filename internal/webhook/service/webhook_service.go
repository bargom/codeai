// Package service provides the webhook business logic layer.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/bargom/codeai/internal/event/bus"
	"github.com/bargom/codeai/internal/webhook/repository"
	"github.com/bargom/codeai/pkg/integration/webhook"
)

// Logger defines the logging interface for the webhook service.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
}

// Config holds configuration for the webhook service.
type Config struct {
	MaxFailureCount int           // Disable webhook after this many consecutive failures
	DefaultTimeout  time.Duration // Default timeout for webhook delivery
}

// DefaultConfig returns a default service configuration.
func DefaultConfig() Config {
	return Config{
		MaxFailureCount: 10,
		DefaultTimeout:  30 * time.Second,
	}
}

// WebhookService orchestrates webhook delivery and management.
type WebhookService struct {
	client     *webhook.Client
	repository repository.WebhookRepository
	config     Config
	logger     Logger
}

// NewWebhookService creates a new webhook service.
func NewWebhookService(
	client *webhook.Client,
	repo repository.WebhookRepository,
	opts ...Option,
) *WebhookService {
	s := &WebhookService{
		client:     client,
		repository: repo,
		config:     DefaultConfig(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Option configures the WebhookService.
type Option func(*WebhookService)

// WithLogger sets the logger for the service.
func WithLogger(logger Logger) Option {
	return func(s *WebhookService) {
		s.logger = logger
	}
}

// WithConfig sets the configuration for the service.
func WithConfig(cfg Config) Option {
	return func(s *WebhookService) {
		s.config = cfg
	}
}

// RegisterWebhookRequest represents a request to register a new webhook.
type RegisterWebhookRequest struct {
	URL      string                 `json:"url" validate:"required,url"`
	Events   []bus.EventType        `json:"events"`
	Secret   string                 `json:"secret,omitempty"`
	Headers  map[string]string      `json:"headers,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RegisterWebhook creates a new webhook subscription.
func (s *WebhookService) RegisterWebhook(ctx context.Context, req RegisterWebhookRequest) (string, error) {
	webhookID := uuid.New().String()
	now := time.Now()

	config := &repository.WebhookConfig{
		ID:        webhookID,
		URL:       req.URL,
		Events:    req.Events,
		Secret:    req.Secret,
		Headers:   req.Headers,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  req.Metadata,
	}

	if err := s.repository.CreateWebhook(ctx, config); err != nil {
		return "", fmt.Errorf("creating webhook: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("webhook registered",
			"webhookID", webhookID,
			"url", req.URL,
			"events", req.Events,
		)
	}

	return webhookID, nil
}

// GetWebhook retrieves a webhook by its ID.
func (s *WebhookService) GetWebhook(ctx context.Context, webhookID string) (*repository.WebhookConfig, error) {
	return s.repository.GetWebhook(ctx, webhookID)
}

// ListWebhooks retrieves all webhooks with optional filtering.
func (s *WebhookService) ListWebhooks(ctx context.Context, filter repository.WebhookFilter) ([]repository.WebhookConfig, error) {
	return s.repository.ListWebhooks(ctx, filter)
}

// UpdateWebhookRequest represents a request to update a webhook.
type UpdateWebhookRequest struct {
	URL      *string                `json:"url,omitempty"`
	Events   []bus.EventType        `json:"events,omitempty"`
	Secret   *string                `json:"secret,omitempty"`
	Headers  map[string]string      `json:"headers,omitempty"`
	Active   *bool                  `json:"active,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateWebhook updates a webhook configuration.
func (s *WebhookService) UpdateWebhook(ctx context.Context, webhookID string, req UpdateWebhookRequest) error {
	update := repository.WebhookUpdate{
		URL:      req.URL,
		Events:   req.Events,
		Secret:   req.Secret,
		Headers:  req.Headers,
		Active:   req.Active,
		Metadata: req.Metadata,
	}

	if err := s.repository.UpdateWebhook(ctx, webhookID, update); err != nil {
		return fmt.Errorf("updating webhook: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("webhook updated", "webhookID", webhookID)
	}

	return nil
}

// DeleteWebhook removes a webhook subscription.
func (s *WebhookService) DeleteWebhook(ctx context.Context, webhookID string) error {
	if err := s.repository.DeleteWebhook(ctx, webhookID); err != nil {
		return fmt.Errorf("deleting webhook: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("webhook deleted", "webhookID", webhookID)
	}

	return nil
}

// DeliverWebhooksForEvent sends webhooks to all subscribed endpoints for an event.
func (s *WebhookService) DeliverWebhooksForEvent(ctx context.Context, event bus.Event) error {
	webhooks, err := s.repository.GetWebhooksByEvent(ctx, event.Type)
	if err != nil {
		return fmt.Errorf("getting webhooks for event: %w", err)
	}

	if len(webhooks) == 0 {
		return nil
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	for _, wh := range webhooks {
		if err := s.deliverToWebhook(ctx, &wh, event, payload); err != nil {
			if s.logger != nil {
				s.logger.Error("webhook delivery failed",
					"webhookID", wh.ID,
					"url", wh.URL,
					"eventID", event.ID,
					"error", err.Error(),
				)
			}
			// Continue to other webhooks despite error
		}
	}

	return nil
}

// deliverToWebhook sends an event to a specific webhook and records the delivery.
func (s *WebhookService) deliverToWebhook(ctx context.Context, config *repository.WebhookConfig, event bus.Event, payload []byte) error {
	deliveryID := uuid.New().String()

	wh := &webhook.Webhook{
		ID:        deliveryID,
		URL:       config.URL,
		EventType: string(event.Type),
		EventID:   event.ID,
		Payload:   payload,
		Headers:   config.Headers,
		Secret:    config.Secret,
		Timeout:   s.config.DefaultTimeout,
	}

	result, err := s.client.Send(ctx, wh)

	// Record delivery attempt
	delivery := &repository.WebhookDelivery{
		ID:           deliveryID,
		WebhookID:    config.ID,
		EventID:      event.ID,
		EventType:    event.Type,
		URL:          config.URL,
		RequestBody:  payload,
		DeliveredAt:  time.Now(),
	}

	if result != nil {
		delivery.StatusCode = result.StatusCode
		delivery.ResponseBody = result.ResponseBody
		delivery.Duration = result.Duration
		delivery.Attempts = result.Attempts
		delivery.Success = result.Success
		delivery.Error = result.Error
	} else if err != nil {
		delivery.Error = err.Error()
	}

	// Save delivery record
	if saveErr := s.repository.SaveDelivery(ctx, delivery); saveErr != nil {
		if s.logger != nil {
			s.logger.Error("failed to save delivery", "deliveryID", deliveryID, "error", saveErr.Error())
		}
	}

	// Update webhook stats
	now := time.Now()
	if delivery.Success {
		if err := s.repository.ResetFailureCount(ctx, config.ID); err != nil {
			if s.logger != nil {
				s.logger.Error("failed to reset failure count", "webhookID", config.ID, "error", err.Error())
			}
		}
		if err := s.repository.UpdateWebhook(ctx, config.ID, repository.WebhookUpdate{
			LastDelivery: &now,
		}); err != nil {
			if s.logger != nil {
				s.logger.Error("failed to update last delivery", "webhookID", config.ID, "error", err.Error())
			}
		}
	} else {
		if err := s.repository.IncrementFailureCount(ctx, config.ID); err != nil {
			if s.logger != nil {
				s.logger.Error("failed to increment failure count", "webhookID", config.ID, "error", err.Error())
			}
		}

		// Check if webhook should be disabled
		wh, _ := s.repository.GetWebhook(ctx, config.ID)
		if wh != nil && wh.FailureCount >= s.config.MaxFailureCount {
			if disableErr := s.DisableWebhook(ctx, config.ID); disableErr != nil {
				if s.logger != nil {
					s.logger.Error("failed to disable webhook", "webhookID", config.ID, "error", disableErr.Error())
				}
			}
		}

		// Schedule retry
		retryPolicy := webhook.DefaultRetryPolicy()
		nextRetryAt := time.Now().Add(retryPolicy.CalculateBackoff(delivery.Attempts))
		if retryErr := s.repository.UpdateDeliveryRetry(ctx, deliveryID, nextRetryAt); retryErr != nil {
			if s.logger != nil {
				s.logger.Error("failed to schedule retry", "deliveryID", deliveryID, "error", retryErr.Error())
			}
		}
	}

	return err
}

// RetryFailedWebhook retries a failed delivery.
func (s *WebhookService) RetryFailedWebhook(ctx context.Context, deliveryID string) error {
	delivery, err := s.repository.GetDelivery(ctx, deliveryID)
	if err != nil {
		return fmt.Errorf("getting delivery: %w", err)
	}

	config, err := s.repository.GetWebhook(ctx, delivery.WebhookID)
	if err != nil {
		return fmt.Errorf("getting webhook config: %w", err)
	}

	if !config.Active {
		return fmt.Errorf("webhook %s is disabled", config.ID)
	}

	wh := &webhook.Webhook{
		ID:        delivery.ID,
		URL:       config.URL,
		EventType: string(delivery.EventType),
		EventID:   delivery.EventID,
		Payload:   delivery.RequestBody,
		Headers:   config.Headers,
		Secret:    config.Secret,
		Timeout:   s.config.DefaultTimeout,
	}

	result, sendErr := s.client.Send(ctx, wh)

	// Update delivery record
	if result != nil {
		delivery.StatusCode = result.StatusCode
		delivery.ResponseBody = result.ResponseBody
		delivery.Duration = result.Duration
		delivery.Attempts = result.Attempts
		delivery.Success = result.Success
		delivery.Error = result.Error
	} else if sendErr != nil {
		delivery.Error = sendErr.Error()
		delivery.Attempts++
	}
	delivery.DeliveredAt = time.Now()

	if delivery.Success {
		delivery.NextRetryAt = nil
	} else {
		retryPolicy := webhook.DefaultRetryPolicy()
		if delivery.Attempts < retryPolicy.MaxAttempts {
			nextRetryAt := time.Now().Add(retryPolicy.CalculateBackoff(delivery.Attempts))
			delivery.NextRetryAt = &nextRetryAt
		} else {
			delivery.NextRetryAt = nil // Max retries reached
		}
	}

	if saveErr := s.repository.SaveDelivery(ctx, delivery); saveErr != nil {
		if s.logger != nil {
			s.logger.Error("failed to update delivery", "deliveryID", deliveryID, "error", saveErr.Error())
		}
	}

	if s.logger != nil {
		s.logger.Info("retry attempted",
			"deliveryID", deliveryID,
			"success", delivery.Success,
			"attempts", delivery.Attempts,
		)
	}

	return sendErr
}

// DisableWebhook disables a webhook that has been failing.
func (s *WebhookService) DisableWebhook(ctx context.Context, webhookID string) error {
	active := false
	if err := s.repository.UpdateWebhook(ctx, webhookID, repository.WebhookUpdate{
		Active: &active,
	}); err != nil {
		return fmt.Errorf("disabling webhook: %w", err)
	}

	if s.logger != nil {
		s.logger.Warn("webhook disabled due to failures", "webhookID", webhookID)
	}

	return nil
}

// ValidateWebhookURL verifies that a webhook endpoint is reachable.
func (s *WebhookService) ValidateWebhookURL(ctx context.Context, url string) error {
	return s.client.ValidateEndpoint(ctx, url)
}

// GetDeliveries retrieves delivery history for a webhook.
func (s *WebhookService) GetDeliveries(ctx context.Context, webhookID string, filter repository.DeliveryFilter) ([]repository.WebhookDelivery, error) {
	return s.repository.ListDeliveries(ctx, webhookID, filter)
}

// SendTestWebhook sends a test event to a webhook to verify it's working.
func (s *WebhookService) SendTestWebhook(ctx context.Context, webhookID string) (*repository.WebhookDelivery, error) {
	config, err := s.repository.GetWebhook(ctx, webhookID)
	if err != nil {
		return nil, fmt.Errorf("getting webhook: %w", err)
	}

	testEvent := bus.Event{
		ID:        uuid.New().String(),
		Type:      "webhook.test",
		Source:    "codeai.webhook.test",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"test":    true,
			"message": "This is a test webhook delivery from CodeAI",
		},
		Metadata: map[string]string{
			"test": "true",
		},
	}

	payload, err := json.Marshal(testEvent)
	if err != nil {
		return nil, fmt.Errorf("marshaling test event: %w", err)
	}

	deliveryID := uuid.New().String()

	wh := &webhook.Webhook{
		ID:        deliveryID,
		URL:       config.URL,
		EventType: string(testEvent.Type),
		EventID:   testEvent.ID,
		Payload:   payload,
		Headers:   config.Headers,
		Secret:    config.Secret,
		Timeout:   s.config.DefaultTimeout,
	}

	result, sendErr := s.client.Send(ctx, wh)

	delivery := &repository.WebhookDelivery{
		ID:          deliveryID,
		WebhookID:   config.ID,
		EventID:     testEvent.ID,
		EventType:   testEvent.Type,
		URL:         config.URL,
		RequestBody: payload,
		DeliveredAt: time.Now(),
	}

	if result != nil {
		delivery.StatusCode = result.StatusCode
		delivery.ResponseBody = result.ResponseBody
		delivery.Duration = result.Duration
		delivery.Attempts = result.Attempts
		delivery.Success = result.Success
		delivery.Error = result.Error
	} else if sendErr != nil {
		delivery.Error = sendErr.Error()
	}

	// Save test delivery for reference
	if saveErr := s.repository.SaveDelivery(ctx, delivery); saveErr != nil {
		if s.logger != nil {
			s.logger.Error("failed to save test delivery", "deliveryID", deliveryID, "error", saveErr.Error())
		}
	}

	return delivery, sendErr
}

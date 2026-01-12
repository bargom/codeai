package subscribers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bargom/codeai/internal/event/bus"
)

// WebhookConfig holds configuration for a webhook endpoint.
type WebhookConfig struct {
	URL           string
	Secret        string
	Timeout       time.Duration
	RetryAttempts int
	EventTypes    []bus.EventType // Empty means all events
}

// WebhookSubscriber forwards events to configured webhook endpoints.
type WebhookSubscriber struct {
	webhooks []WebhookConfig
	client   *http.Client
	logger   bus.Logger
}

// NewWebhookSubscriber creates a new WebhookSubscriber.
func NewWebhookSubscriber(webhooks []WebhookConfig, logger bus.Logger) *WebhookSubscriber {
	return &WebhookSubscriber{
		webhooks: webhooks,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Handle forwards the event to all configured webhooks.
func (s *WebhookSubscriber) Handle(ctx context.Context, event bus.Event) error {
	for _, wh := range s.webhooks {
		// Check if this webhook should receive this event type
		if len(wh.EventTypes) > 0 && !s.shouldHandle(wh.EventTypes, event.Type) {
			continue
		}

		if err := s.sendWebhook(ctx, wh, event); err != nil {
			if s.logger != nil {
				s.logger.Error("webhook delivery failed",
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

// shouldHandle checks if an event type is in the list.
func (s *WebhookSubscriber) shouldHandle(types []bus.EventType, eventType bus.EventType) bool {
	for _, t := range types {
		if t == eventType {
			return true
		}
	}
	return false
}

// sendWebhook sends an event to a webhook endpoint with retries.
func (s *WebhookSubscriber) sendWebhook(ctx context.Context, wh WebhookConfig, event bus.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	retries := wh.RetryAttempts
	if retries <= 0 {
		retries = 1
	}

	timeout := wh.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	var lastErr error
	for attempt := 0; attempt < retries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
		}

		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		err := s.doSend(reqCtx, wh.URL, wh.Secret, payload)
		cancel()

		if err == nil {
			if s.logger != nil {
				s.logger.Debug("webhook delivered",
					"url", wh.URL,
					"eventID", event.ID,
					"attempt", attempt+1,
				)
			}
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("webhook delivery failed after %d attempts: %w", retries, lastErr)
}

// doSend performs the actual HTTP request.
func (s *WebhookSubscriber) doSend(ctx context.Context, url, secret string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("X-Webhook-Secret", secret)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// AddWebhook adds a webhook configuration.
func (s *WebhookSubscriber) AddWebhook(wh WebhookConfig) {
	s.webhooks = append(s.webhooks, wh)
}

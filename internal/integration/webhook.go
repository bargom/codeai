// Package integration provides webhook dispatching capabilities.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/bargom/codeai/internal/ast"
)

// WebhookDispatcher manages and dispatches webhooks.
type WebhookDispatcher struct {
	mu         sync.RWMutex
	webhooks   map[string]*RegisteredWebhook
	httpClient *http.Client
	logger     *slog.Logger
}

// RegisteredWebhook represents a webhook registered from the DSL.
type RegisteredWebhook struct {
	Name    string
	Event   string
	URL     string
	Method  string
	Headers map[string]string
	Retry   *RetryPolicy
}

// RetryPolicy defines retry behavior for webhook delivery.
type RetryPolicy struct {
	MaxAttempts       int
	InitialInterval   time.Duration
	BackoffMultiplier float64
}

// WebhookResult represents the result of a webhook dispatch.
type WebhookResult struct {
	Success    bool
	StatusCode int
	Attempts   int
	Error      error
	Duration   time.Duration
}

// NewWebhookDispatcher creates a new webhook dispatcher.
func NewWebhookDispatcher(logger *slog.Logger) *WebhookDispatcher {
	if logger == nil {
		logger = slog.Default()
	}
	return &WebhookDispatcher{
		webhooks: make(map[string]*RegisteredWebhook),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// RegisterWebhookFromAST registers a webhook from AST definition.
func (d *WebhookDispatcher) RegisterWebhookFromAST(webhook *ast.WebhookDecl) error {
	if webhook == nil {
		return fmt.Errorf("webhook declaration is nil")
	}

	// Validate URL
	if _, err := url.Parse(webhook.URL); err != nil {
		return fmt.Errorf("invalid webhook URL '%s': %w", webhook.URL, err)
	}

	// Build headers map
	headers := make(map[string]string)
	for _, h := range webhook.Headers {
		headers[h.Key] = h.Value
	}

	// Build retry policy
	var retryPolicy *RetryPolicy
	if webhook.Retry != nil {
		interval, _ := time.ParseDuration(webhook.Retry.InitialInterval)
		if interval == 0 {
			interval = time.Second
		}
		retryPolicy = &RetryPolicy{
			MaxAttempts:       webhook.Retry.MaxAttempts,
			InitialInterval:   interval,
			BackoffMultiplier: webhook.Retry.BackoffMultiplier,
		}
	}

	registered := &RegisteredWebhook{
		Name:    webhook.Name,
		Event:   webhook.Event,
		URL:     webhook.URL,
		Method:  string(webhook.Method),
		Headers: headers,
		Retry:   retryPolicy,
	}

	d.mu.Lock()
	d.webhooks[webhook.Name] = registered
	d.mu.Unlock()

	return nil
}

// DispatchWebhook sends an HTTP request to the webhook URL.
func (d *WebhookDispatcher) DispatchWebhook(ctx context.Context, webhookName string, payload interface{}) (*WebhookResult, error) {
	d.mu.RLock()
	webhook, exists := d.webhooks[webhookName]
	d.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("webhook '%s' not found", webhookName)
	}

	return d.dispatch(ctx, webhook, payload)
}

// DispatchByEvent dispatches to all webhooks registered for an event.
func (d *WebhookDispatcher) DispatchByEvent(ctx context.Context, eventName string, payload interface{}) ([]*WebhookResult, error) {
	d.mu.RLock()
	var matchingWebhooks []*RegisteredWebhook
	for _, wh := range d.webhooks {
		if wh.Event == eventName {
			matchingWebhooks = append(matchingWebhooks, wh)
		}
	}
	d.mu.RUnlock()

	if len(matchingWebhooks) == 0 {
		return nil, nil
	}

	results := make([]*WebhookResult, len(matchingWebhooks))
	for i, wh := range matchingWebhooks {
		result, err := d.dispatch(ctx, wh, payload)
		if err != nil {
			result = &WebhookResult{
				Success: false,
				Error:   err,
			}
		}
		results[i] = result
	}

	return results, nil
}

// dispatch sends the actual HTTP request with retry logic.
func (d *WebhookDispatcher) dispatch(ctx context.Context, webhook *RegisteredWebhook, payload interface{}) (*WebhookResult, error) {
	start := time.Now()

	// Serialize payload
	body, err := json.Marshal(payload)
	if err != nil {
		return &WebhookResult{
			Success:  false,
			Error:    fmt.Errorf("failed to marshal payload: %w", err),
			Duration: time.Since(start),
		}, nil
	}

	maxAttempts := 1
	initialInterval := time.Second
	backoff := 1.0

	if webhook.Retry != nil {
		maxAttempts = webhook.Retry.MaxAttempts
		initialInterval = webhook.Retry.InitialInterval
		backoff = webhook.Retry.BackoffMultiplier
	}

	var lastErr error
	var lastStatusCode int

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Create request
		req, err := http.NewRequestWithContext(ctx, webhook.Method, webhook.URL, bytes.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}

		// Set default content type if not specified
		if _, hasContentType := webhook.Headers["Content-Type"]; !hasContentType {
			req.Header.Set("Content-Type", "application/json")
		}

		// Apply custom headers
		for key, value := range webhook.Headers {
			req.Header.Set(key, value)
		}

		// Execute request
		resp, err := d.httpClient.Do(req)
		if err != nil {
			lastErr = err
			d.logger.Warn("webhook request failed",
				"webhook", webhook.Name,
				"attempt", attempt,
				"error", err,
			)
		} else {
			lastStatusCode = resp.StatusCode
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return &WebhookResult{
					Success:    true,
					StatusCode: resp.StatusCode,
					Attempts:   attempt,
					Duration:   time.Since(start),
				}, nil
			}

			lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
			d.logger.Warn("webhook returned non-success status",
				"webhook", webhook.Name,
				"attempt", attempt,
				"status", resp.StatusCode,
			)
		}

		// Wait before retry (if not last attempt)
		if attempt < maxAttempts {
			waitTime := initialInterval * time.Duration(math.Pow(backoff, float64(attempt-1)))
			select {
			case <-ctx.Done():
				return &WebhookResult{
					Success:    false,
					StatusCode: lastStatusCode,
					Attempts:   attempt,
					Error:      ctx.Err(),
					Duration:   time.Since(start),
				}, nil
			case <-time.After(waitTime):
				// Continue to next attempt
			}
		}
	}

	return &WebhookResult{
		Success:    false,
		StatusCode: lastStatusCode,
		Attempts:   maxAttempts,
		Error:      lastErr,
		Duration:   time.Since(start),
	}, nil
}

// GetWebhook returns a registered webhook by name.
func (d *WebhookDispatcher) GetWebhook(name string) (*RegisteredWebhook, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	wh, exists := d.webhooks[name]
	return wh, exists
}

// Count returns the number of registered webhooks.
func (d *WebhookDispatcher) Count() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.webhooks)
}

// GetWebhooksForEvent returns all webhooks registered for a specific event.
func (d *WebhookDispatcher) GetWebhooksForEvent(eventName string) []*RegisteredWebhook {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var webhooks []*RegisteredWebhook
	for _, wh := range d.webhooks {
		if wh.Event == eventName {
			webhooks = append(webhooks, wh)
		}
	}
	return webhooks
}

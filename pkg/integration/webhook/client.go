package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Config holds configuration for the webhook client.
type Config struct {
	Timeout       time.Duration
	MaxRetries    int
	RetryBackoff  time.Duration
	MaxConcurrent int
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Timeout:       30 * time.Second,
		MaxRetries:    3,
		RetryBackoff:  5 * time.Second,
		MaxConcurrent: 50,
	}
}

// Client sends webhooks to external endpoints with retry logic.
type Client struct {
	httpClient *http.Client
	config     Config
	semaphore  chan struct{}
}

// NewClient creates a new webhook client with the given configuration.
func NewClient(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.MaxConcurrent == 0 {
		cfg.MaxConcurrent = 50
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		config:    cfg,
		semaphore: make(chan struct{}, cfg.MaxConcurrent),
	}
}

// Send delivers a webhook to its target URL with retry logic.
func (c *Client) Send(ctx context.Context, webhook *Webhook) (*DeliveryResult, error) {
	// Acquire semaphore for concurrency limiting
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	retryPolicy := webhook.RetryPolicy
	if retryPolicy == nil {
		retryPolicy = DefaultRetryPolicy()
	}

	maxAttempts := retryPolicy.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	result := &DeliveryResult{
		WebhookID: webhook.ID,
	}

	var lastErr error
	startTime := time.Now()

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result.Attempts = attempt

		if attempt > 1 {
			backoff := retryPolicy.CalculateBackoff(attempt - 1)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				result.Error = ctx.Err().Error()
				result.DeliveredAt = time.Now()
				result.Duration = time.Since(startTime)
				return result, ctx.Err()
			}
		}

		statusCode, responseBody, err := c.doSend(ctx, webhook)
		result.StatusCode = statusCode
		result.ResponseBody = responseBody

		if err == nil && statusCode >= 200 && statusCode < 300 {
			result.Success = true
			result.DeliveredAt = time.Now()
			result.Duration = time.Since(startTime)
			return result, nil
		}

		if err != nil {
			lastErr = err
			result.Error = err.Error()
		} else {
			lastErr = fmt.Errorf("unexpected status code: %d", statusCode)
			result.Error = lastErr.Error()
		}

		// Don't retry for client errors (4xx) except 429 (rate limit)
		if statusCode >= 400 && statusCode < 500 && statusCode != 429 {
			break
		}
	}

	result.DeliveredAt = time.Now()
	result.Duration = time.Since(startTime)

	if lastErr != nil {
		return result, fmt.Errorf("webhook delivery failed after %d attempts: %w", result.Attempts, lastErr)
	}
	return result, nil
}

// SendBatch delivers multiple webhooks concurrently and returns all results.
func (c *Client) SendBatch(ctx context.Context, webhooks []*Webhook) ([]DeliveryResult, error) {
	results := make([]DeliveryResult, len(webhooks))
	var wg sync.WaitGroup

	for i, webhook := range webhooks {
		wg.Add(1)
		go func(idx int, wh *Webhook) {
			defer wg.Done()
			result, err := c.Send(ctx, wh)
			if result != nil {
				results[idx] = *result
			} else if err != nil {
				results[idx] = DeliveryResult{
					WebhookID:   wh.ID,
					Error:       err.Error(),
					DeliveredAt: time.Now(),
				}
			}
		}(i, webhook)
	}

	wg.Wait()
	return results, nil
}

// ValidateEndpoint checks if a webhook endpoint is reachable.
func (c *Client) ValidateEndpoint(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("endpoint unreachable: %w", err)
	}
	defer resp.Body.Close()

	// Accept any 2xx or 4xx (method not allowed is ok for HEAD)
	if resp.StatusCode >= 500 {
		return fmt.Errorf("endpoint returned server error: %d", resp.StatusCode)
	}

	return nil
}

// doSend performs the actual HTTP request.
func (c *Client) doSend(ctx context.Context, webhook *Webhook) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(webhook.Payload))
	if err != nil {
		return 0, "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CodeAI-Webhook/1.0")

	// Add custom headers
	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}

	// Add HMAC signature if secret is provided
	if webhook.Secret != "" {
		signature := signPayload(webhook.Secret, webhook.Payload)
		req.Header.Set("X-Webhook-Signature", signature)
		req.Header.Set("X-Webhook-Signature-Algorithm", "sha256")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024)) // Limit to 64KB
	if err != nil {
		return resp.StatusCode, "", fmt.Errorf("reading response: %w", err)
	}

	return resp.StatusCode, string(body), nil
}

// signPayload generates an HMAC-SHA256 signature for the payload.
func signPayload(secret string, payload []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
)

// WebhookPayload represents the payload for a webhook notification task.
type WebhookPayload struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        json.RawMessage   `json:"body,omitempty"`
	Timeout     time.Duration     `json:"timeout"`
	RetryPolicy RetryPolicy       `json:"retry_policy,omitempty"`
}

// WebhookResult represents the result of a webhook notification.
type WebhookResult struct {
	URL         string        `json:"url"`
	StatusCode  int           `json:"status_code"`
	Response    string        `json:"response"`
	Duration    time.Duration `json:"duration"`
	CompletedAt time.Time     `json:"completed_at"`
	Success     bool          `json:"success"`
	Error       string        `json:"error,omitempty"`
}

// WebhookHandler handles webhook notification tasks.
type WebhookHandler struct {
	httpClient *http.Client
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProcessTask handles the webhook notification.
func (h *WebhookHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload WebhookPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	// Set defaults
	if payload.Method == "" {
		payload.Method = http.MethodPost
	}
	if payload.Timeout > 0 {
		h.httpClient.Timeout = payload.Timeout
	}

	startTime := time.Now()

	// Create the request
	var body io.Reader
	if len(payload.Body) > 0 {
		body = bytes.NewReader(payload.Body)
	}

	req, err := http.NewRequestWithContext(ctx, payload.Method, payload.URL, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range payload.Headers {
		req.Header.Set(key, value)
	}

	// Execute the request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	result := WebhookResult{
		URL:         payload.URL,
		StatusCode:  resp.StatusCode,
		Response:    string(respBody),
		Duration:    time.Since(startTime),
		CompletedAt: time.Now(),
		Success:     resp.StatusCode >= 200 && resp.StatusCode < 300,
	}

	// Return error for non-2xx status codes to trigger retry
	if !result.Success {
		return fmt.Errorf("webhook failed with status %d: %s", resp.StatusCode, result.Response)
	}

	return nil
}

// HandleWebhookTask is the handler function for webhook tasks.
func HandleWebhookTask(ctx context.Context, t *asynq.Task) error {
	handler := NewWebhookHandler()
	return handler.ProcessTask(ctx, t)
}

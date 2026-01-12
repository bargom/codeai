// Package webhook provides HTTP webhook client and types for external integrations.
package webhook

import (
	"encoding/json"
	"time"
)

// Webhook represents a webhook request to be sent to an external endpoint.
type Webhook struct {
	ID          string            `json:"id"`
	URL         string            `json:"url"`
	EventType   string            `json:"event_type"`
	EventID     string            `json:"event_id"`
	Payload     json.RawMessage   `json:"payload"`
	Headers     map[string]string `json:"headers,omitempty"`
	Secret      string            `json:"-"` // For HMAC signing, not serialized
	Timeout     time.Duration     `json:"-"`
	RetryPolicy *RetryPolicy      `json:"-"`
}

// DeliveryResult represents the outcome of a webhook delivery attempt.
type DeliveryResult struct {
	WebhookID    string        `json:"webhook_id"`
	StatusCode   int           `json:"status_code"`
	ResponseBody string        `json:"response_body,omitempty"`
	Duration     time.Duration `json:"duration_ms"`
	Attempts     int           `json:"attempts"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	DeliveredAt  time.Time     `json:"delivered_at"`
}

// RetryPolicy defines the retry behavior for failed webhook deliveries.
type RetryPolicy struct {
	MaxAttempts    int           `json:"max_attempts"`
	InitialBackoff time.Duration `json:"initial_backoff"`
	MaxBackoff     time.Duration `json:"max_backoff"`
	Multiplier     float64       `json:"multiplier"`
}

// DefaultRetryPolicy returns a sensible default retry policy.
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     5 * time.Minute,
		Multiplier:     2.0,
	}
}

// CalculateBackoff returns the backoff duration for the given attempt number.
func (p *RetryPolicy) CalculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return p.InitialBackoff
	}

	backoff := p.InitialBackoff
	for i := 1; i < attempt; i++ {
		backoff = time.Duration(float64(backoff) * p.Multiplier)
		if backoff > p.MaxBackoff {
			return p.MaxBackoff
		}
	}
	return backoff
}

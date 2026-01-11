# Task: Webhook Publisher

## Overview
Implement a webhook publisher for sending events to external HTTP endpoints.

## Phase
Phase 3: Workflows and Jobs

## Priority
Medium - Important for integrations.

## Dependencies
- 03-Workflows-Jobs/04-event-system.md

## Description
Create a webhook publisher that sends events to configured HTTP endpoints with retry logic, signature verification, and delivery tracking.

## Detailed Requirements

### 1. Webhook Publisher (internal/modules/event/publishers/webhook.go)

```go
package publishers

import (
    "bytes"
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "log/slog"

    "github.com/codeai/codeai/internal/modules/event"
)

type WebhookPublisher struct {
    url        string
    secret     string
    headers    map[string]string
    timeout    time.Duration
    retry      RetryConfig
    client     *http.Client
    logger     *slog.Logger
}

type WebhookConfig struct {
    URL     string
    Secret  string            // For signature
    Headers map[string]string
    Timeout time.Duration
    Retry   RetryConfig
}

type RetryConfig struct {
    MaxAttempts int
    InitialWait time.Duration
    MaxWait     time.Duration
    Multiplier  float64
}

func NewWebhookPublisher(config WebhookConfig) *WebhookPublisher {
    timeout := config.Timeout
    if timeout == 0 {
        timeout = 30 * time.Second
    }

    retry := config.Retry
    if retry.MaxAttempts == 0 {
        retry.MaxAttempts = 3
        retry.InitialWait = time.Second
        retry.MaxWait = 30 * time.Second
        retry.Multiplier = 2.0
    }

    return &WebhookPublisher{
        url:     config.URL,
        secret:  config.Secret,
        headers: config.Headers,
        timeout: timeout,
        retry:   retry,
        client: &http.Client{
            Timeout: timeout,
        },
        logger: slog.Default().With("publisher", "webhook"),
    }
}

func (p *WebhookPublisher) Name() string {
    return "webhook:" + p.url
}

func (p *WebhookPublisher) Publish(ctx context.Context, eventType string, msg *event.EventMessage) error {
    payload, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("marshal error: %w", err)
    }

    var lastErr error
    wait := p.retry.InitialWait

    for attempt := 1; attempt <= p.retry.MaxAttempts; attempt++ {
        err := p.send(ctx, eventType, payload)
        if err == nil {
            p.logger.Info("webhook delivered",
                "url", p.url,
                "event", eventType,
                "attempt", attempt,
            )
            return nil
        }

        lastErr = err
        p.logger.Warn("webhook failed",
            "url", p.url,
            "event", eventType,
            "attempt", attempt,
            "error", err,
        )

        if attempt < p.retry.MaxAttempts {
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(wait):
            }

            wait = time.Duration(float64(wait) * p.retry.Multiplier)
            if wait > p.retry.MaxWait {
                wait = p.retry.MaxWait
            }
        }
    }

    return fmt.Errorf("webhook delivery failed after %d attempts: %w",
        p.retry.MaxAttempts, lastErr)
}

func (p *WebhookPublisher) send(ctx context.Context, eventType string, payload []byte) error {
    req, err := http.NewRequestWithContext(ctx, "POST", p.url, bytes.NewReader(payload))
    if err != nil {
        return err
    }

    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Event-Type", eventType)
    req.Header.Set("X-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))

    for k, v := range p.headers {
        req.Header.Set(k, v)
    }

    // Add signature if secret is configured
    if p.secret != "" {
        signature := p.sign(payload)
        req.Header.Set("X-Signature", signature)
        req.Header.Set("X-Signature-256", "sha256="+signature)
    }

    resp, err := p.client.Do(req)
    if err != nil {
        return fmt.Errorf("request error: %w", err)
    }
    defer resp.Body.Close()

    // Read response body for error details
    body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
    }

    return nil
}

func (p *WebhookPublisher) sign(payload []byte) string {
    mac := hmac.New(sha256.New, []byte(p.secret))
    mac.Write(payload)
    return hex.EncodeToString(mac.Sum(nil))
}

func (p *WebhookPublisher) Close() error {
    return nil
}

// VerifySignature verifies a webhook signature (for receiving webhooks)
func VerifySignature(payload []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

### 2. Webhook Registry (internal/modules/event/publishers/webhook_registry.go)

```go
package publishers

import (
    "sync"
)

type WebhookRegistry struct {
    webhooks map[string][]*WebhookPublisher
    mu       sync.RWMutex
}

func NewWebhookRegistry() *WebhookRegistry {
    return &WebhookRegistry{
        webhooks: make(map[string][]*WebhookPublisher),
    }
}

func (r *WebhookRegistry) Register(eventType string, webhook *WebhookPublisher) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.webhooks[eventType] = append(r.webhooks[eventType], webhook)
}

func (r *WebhookRegistry) GetWebhooks(eventType string) []*WebhookPublisher {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.webhooks[eventType]
}

func (r *WebhookRegistry) Unregister(eventType, url string) {
    r.mu.Lock()
    defer r.mu.Unlock()

    hooks := r.webhooks[eventType]
    filtered := make([]*WebhookPublisher, 0, len(hooks))

    for _, h := range hooks {
        if h.url != url {
            filtered = append(filtered, h)
        }
    }

    r.webhooks[eventType] = filtered
}
```

### 3. Delivery Tracking

```go
// internal/modules/event/publishers/delivery.go
package publishers

import (
    "context"
    "time"
)

type DeliveryRecord struct {
    ID          string
    EventID     string
    EventType   string
    WebhookURL  string
    Status      DeliveryStatus
    Attempts    int
    LastAttempt time.Time
    Error       string
    CreatedAt   time.Time
}

type DeliveryStatus string

const (
    DeliveryPending   DeliveryStatus = "pending"
    DeliverySuccess   DeliveryStatus = "success"
    DeliveryFailed    DeliveryStatus = "failed"
)

type DeliveryStore interface {
    Save(ctx context.Context, record *DeliveryRecord) error
    Get(ctx context.Context, id string) (*DeliveryRecord, error)
    ListByEvent(ctx context.Context, eventID string) ([]*DeliveryRecord, error)
    ListFailed(ctx context.Context, limit int) ([]*DeliveryRecord, error)
}

// TrackedWebhookPublisher wraps webhook publisher with delivery tracking
type TrackedWebhookPublisher struct {
    *WebhookPublisher
    store DeliveryStore
}

func NewTrackedWebhookPublisher(config WebhookConfig, store DeliveryStore) *TrackedWebhookPublisher {
    return &TrackedWebhookPublisher{
        WebhookPublisher: NewWebhookPublisher(config),
        store:           store,
    }
}

func (p *TrackedWebhookPublisher) Publish(ctx context.Context, eventType string, msg *event.EventMessage) error {
    record := &DeliveryRecord{
        ID:        uuid.New().String(),
        EventID:   msg.ID,
        EventType: eventType,
        WebhookURL: p.url,
        Status:    DeliveryPending,
        CreatedAt: time.Now(),
    }

    err := p.WebhookPublisher.Publish(ctx, eventType, msg)

    record.Attempts = 1 // Simplified; actual retry count tracked differently
    record.LastAttempt = time.Now()

    if err != nil {
        record.Status = DeliveryFailed
        record.Error = err.Error()
    } else {
        record.Status = DeliverySuccess
    }

    p.store.Save(ctx, record)
    return err
}
```

## Acceptance Criteria
- [ ] HTTP POST to configured URLs
- [ ] HMAC signature generation
- [ ] Retry with exponential backoff
- [ ] Custom headers support
- [ ] Timeout configuration
- [ ] Delivery tracking
- [ ] Webhook registry per event type

## Testing Strategy
- Unit tests with mock HTTP server
- Integration tests with real endpoints
- Retry behavior tests
- Signature verification tests

## Files to Create
- `internal/modules/event/publishers/webhook.go`
- `internal/modules/event/publishers/webhook_registry.go`
- `internal/modules/event/publishers/delivery.go`
- `internal/modules/event/publishers/webhook_test.go`

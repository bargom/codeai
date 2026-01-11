# Task: Integration Module with Circuit Breaker

## Overview
Implement an integration module for connecting to external REST/GraphQL APIs with built-in circuit breaker, retry logic, and timeout handling.

## Phase
Phase 4: Integrations and Polish

## Priority
High - Core feature for external service communication.

## Dependencies
- Phase 1 complete

## Description
Create an integration module that manages connections to external services with resilience patterns including circuit breakers, timeouts, and automatic retries.

## Detailed Requirements

### 1. Integration Types (internal/modules/integration/types.go)

```go
package integration

import (
    "time"
)

type Integration struct {
    ID             string
    Description    string
    Type           IntegrationType
    BaseURL        string
    Auth           AuthConfig
    Headers        map[string]string
    Timeout        time.Duration
    Retry          *RetryConfig
    CircuitBreaker *CircuitBreakerConfig
    Operations     map[string]*Operation
}

type IntegrationType string

const (
    IntegrationREST    IntegrationType = "rest"
    IntegrationGraphQL IntegrationType = "graphql"
    IntegrationGRPC    IntegrationType = "grpc"
)

type AuthConfig struct {
    Type   AuthType
    Token  string            // For bearer
    Header string            // Header name for API key
    Key    string            // API key value
    Params map[string]string // For OAuth
}

type AuthType string

const (
    AuthNone   AuthType = "none"
    AuthBearer AuthType = "bearer"
    AuthAPIKey AuthType = "api_key"
    AuthBasic  AuthType = "basic"
    AuthOAuth2 AuthType = "oauth2"
)

type Operation struct {
    ID       string
    Method   string
    Path     string
    Body     *BodyDef
    Returns  *ReturnDef
    Headers  map[string]string
    Timeout  time.Duration
}

type RetryConfig struct {
    MaxAttempts int
    InitialWait time.Duration
    MaxWait     time.Duration
    Multiplier  float64
    RetryOn     []int // HTTP status codes to retry
}

type CircuitBreakerConfig struct {
    Threshold   int           // Failures before opening
    Window      time.Duration // Time window for threshold
    ResetAfter  time.Duration // Time before trying again
}
```

### 2. Circuit Breaker (internal/modules/integration/circuit.go)

```go
package integration

import (
    "errors"
    "sync"
    "time"
)

type CircuitState int

const (
    StateClosed CircuitState = iota
    StateOpen
    StateHalfOpen
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type CircuitBreaker struct {
    config      CircuitBreakerConfig
    state       CircuitState
    failures    int
    lastFailure time.Time
    lastSuccess time.Time
    mu          sync.RWMutex
}

func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
    if config.Threshold == 0 {
        config.Threshold = 5
    }
    if config.Window == 0 {
        config.Window = time.Minute
    }
    if config.ResetAfter == 0 {
        config.ResetAfter = 30 * time.Second
    }

    return &CircuitBreaker{
        config: config,
        state:  StateClosed,
    }
}

func (cb *CircuitBreaker) Allow() error {
    cb.mu.RLock()
    state := cb.state
    lastFailure := cb.lastFailure
    cb.mu.RUnlock()

    switch state {
    case StateClosed:
        return nil

    case StateOpen:
        if time.Since(lastFailure) > cb.config.ResetAfter {
            cb.mu.Lock()
            cb.state = StateHalfOpen
            cb.mu.Unlock()
            return nil
        }
        return ErrCircuitOpen

    case StateHalfOpen:
        return nil
    }

    return nil
}

func (cb *CircuitBreaker) RecordSuccess() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failures = 0
    cb.lastSuccess = time.Now()

    if cb.state == StateHalfOpen {
        cb.state = StateClosed
    }
}

func (cb *CircuitBreaker) RecordFailure() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    now := time.Now()

    // Reset counter if outside window
    if time.Since(cb.lastFailure) > cb.config.Window {
        cb.failures = 0
    }

    cb.failures++
    cb.lastFailure = now

    if cb.failures >= cb.config.Threshold {
        cb.state = StateOpen
    }
}

func (cb *CircuitBreaker) State() CircuitState {
    cb.mu.RLock()
    defer cb.mu.RUnlock()
    return cb.state
}

func (cb *CircuitBreaker) Stats() CircuitStats {
    cb.mu.RLock()
    defer cb.mu.RUnlock()

    return CircuitStats{
        State:       cb.state,
        Failures:    cb.failures,
        LastFailure: cb.lastFailure,
        LastSuccess: cb.lastSuccess,
    }
}

type CircuitStats struct {
    State       CircuitState
    Failures    int
    LastFailure time.Time
    LastSuccess time.Time
}
```

### 3. HTTP Client (internal/modules/integration/client.go)

```go
package integration

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"

    "log/slog"
)

type Client struct {
    integration    *Integration
    httpClient     *http.Client
    circuitBreaker *CircuitBreaker
    logger         *slog.Logger
}

func NewClient(integration *Integration) *Client {
    timeout := integration.Timeout
    if timeout == 0 {
        timeout = 30 * time.Second
    }

    c := &Client{
        integration: integration,
        httpClient: &http.Client{
            Timeout: timeout,
        },
        logger: slog.Default().With("integration", integration.ID),
    }

    if integration.CircuitBreaker != nil {
        c.circuitBreaker = NewCircuitBreaker(*integration.CircuitBreaker)
    }

    return c
}

func (c *Client) Call(ctx context.Context, operation string, params map[string]any) (any, error) {
    op, ok := c.integration.Operations[operation]
    if !ok {
        return nil, fmt.Errorf("unknown operation: %s", operation)
    }

    // Check circuit breaker
    if c.circuitBreaker != nil {
        if err := c.circuitBreaker.Allow(); err != nil {
            return nil, err
        }
    }

    // Execute with retry
    result, err := c.executeWithRetry(ctx, op, params)

    // Record result for circuit breaker
    if c.circuitBreaker != nil {
        if err != nil {
            c.circuitBreaker.RecordFailure()
        } else {
            c.circuitBreaker.RecordSuccess()
        }
    }

    return result, err
}

func (c *Client) executeWithRetry(ctx context.Context, op *Operation, params map[string]any) (any, error) {
    retry := c.integration.Retry
    if retry == nil {
        retry = &RetryConfig{MaxAttempts: 1}
    }

    var lastErr error
    wait := retry.InitialWait
    if wait == 0 {
        wait = time.Second
    }

    for attempt := 1; attempt <= retry.MaxAttempts; attempt++ {
        result, err, statusCode := c.execute(ctx, op, params)

        if err == nil {
            return result, nil
        }

        lastErr = err

        // Check if we should retry
        if !c.shouldRetry(retry, statusCode, attempt) {
            break
        }

        c.logger.Warn("retrying request",
            "operation", op.ID,
            "attempt", attempt,
            "error", err,
        )

        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-time.After(wait):
        }

        // Calculate next wait
        multiplier := retry.Multiplier
        if multiplier == 0 {
            multiplier = 2.0
        }
        wait = time.Duration(float64(wait) * multiplier)
        if retry.MaxWait > 0 && wait > retry.MaxWait {
            wait = retry.MaxWait
        }
    }

    return nil, lastErr
}

func (c *Client) execute(ctx context.Context, op *Operation, params map[string]any) (any, error, int) {
    url := c.buildURL(op, params)

    var body io.Reader
    if op.Method != "GET" && params != nil {
        data, err := json.Marshal(params)
        if err != nil {
            return nil, err, 0
        }
        body = bytes.NewReader(data)
    }

    req, err := http.NewRequestWithContext(ctx, op.Method, url, body)
    if err != nil {
        return nil, err, 0
    }

    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    for k, v := range c.integration.Headers {
        req.Header.Set(k, v)
    }
    for k, v := range op.Headers {
        req.Header.Set(k, v)
    }

    // Set auth
    c.applyAuth(req)

    // Execute
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err, 0
    }
    defer resp.Body.Close()

    // Read response
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err, resp.StatusCode
    }

    // Check status
    if resp.StatusCode >= 400 {
        return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody)), resp.StatusCode
    }

    // Parse response
    var result any
    if len(respBody) > 0 {
        if err := json.Unmarshal(respBody, &result); err != nil {
            return nil, err, resp.StatusCode
        }
    }

    return result, nil, resp.StatusCode
}

func (c *Client) buildURL(op *Operation, params map[string]any) string {
    url := c.integration.BaseURL + op.Path

    // Replace path parameters
    for k, v := range params {
        placeholder := "{" + k + "}"
        if strings.Contains(url, placeholder) {
            url = strings.Replace(url, placeholder, fmt.Sprintf("%v", v), 1)
        }
    }

    return url
}

func (c *Client) applyAuth(req *http.Request) {
    auth := c.integration.Auth

    switch auth.Type {
    case AuthBearer:
        req.Header.Set("Authorization", "Bearer "+auth.Token)

    case AuthAPIKey:
        header := auth.Header
        if header == "" {
            header = "X-API-Key"
        }
        req.Header.Set(header, auth.Key)

    case AuthBasic:
        req.SetBasicAuth(auth.Params["username"], auth.Params["password"])
    }
}

func (c *Client) shouldRetry(retry *RetryConfig, statusCode, attempt int) bool {
    if attempt >= retry.MaxAttempts {
        return false
    }

    // Retry on specific status codes
    if len(retry.RetryOn) > 0 {
        for _, code := range retry.RetryOn {
            if code == statusCode {
                return true
            }
        }
        return false
    }

    // Default: retry on 5xx and 429
    return statusCode >= 500 || statusCode == 429
}
```

### 4. Integration Module (internal/modules/integration/module.go)

```go
package integration

import (
    "context"
    "sync"
)

type IntegrationModule interface {
    Module
    RegisterIntegration(integration *Integration) error
    Call(ctx context.Context, integrationID, operation string, params map[string]any) (any, error)
    GetStats(integrationID string) *IntegrationStats
}

type integrationModule struct {
    integrations map[string]*Integration
    clients      map[string]*Client
    mu           sync.RWMutex
    logger       *slog.Logger
}

func NewIntegrationModule() IntegrationModule {
    return &integrationModule{
        integrations: make(map[string]*Integration),
        clients:      make(map[string]*Client),
        logger:       slog.Default().With("module", "integration"),
    }
}

func (m *integrationModule) Name() string { return "integration" }

func (m *integrationModule) Initialize(config *Config) error {
    return nil
}

func (m *integrationModule) Start(ctx context.Context) error {
    return nil
}

func (m *integrationModule) Stop(ctx context.Context) error {
    return nil
}

func (m *integrationModule) Health() HealthStatus {
    return HealthStatus{Status: "healthy"}
}

func (m *integrationModule) RegisterIntegration(integration *Integration) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.integrations[integration.ID] = integration
    m.clients[integration.ID] = NewClient(integration)

    m.logger.Info("registered integration", "id", integration.ID)
    return nil
}

func (m *integrationModule) Call(ctx context.Context, integrationID, operation string, params map[string]any) (any, error) {
    m.mu.RLock()
    client, ok := m.clients[integrationID]
    m.mu.RUnlock()

    if !ok {
        return nil, fmt.Errorf("integration not found: %s", integrationID)
    }

    return client.Call(ctx, operation, params)
}

func (m *integrationModule) GetStats(integrationID string) *IntegrationStats {
    m.mu.RLock()
    client, ok := m.clients[integrationID]
    m.mu.RUnlock()

    if !ok {
        return nil
    }

    stats := &IntegrationStats{
        IntegrationID: integrationID,
    }

    if client.circuitBreaker != nil {
        cbStats := client.circuitBreaker.Stats()
        stats.CircuitState = cbStats.State
        stats.Failures = cbStats.Failures
        stats.LastFailure = cbStats.LastFailure
    }

    return stats
}

type IntegrationStats struct {
    IntegrationID string
    CircuitState  CircuitState
    Failures      int
    LastFailure   time.Time
}
```

## Acceptance Criteria
- [ ] Register integrations with operations
- [ ] Circuit breaker with configurable threshold
- [ ] Retry with exponential backoff
- [ ] Timeout configuration
- [ ] Multiple auth types (Bearer, API Key, Basic)
- [ ] Path parameter substitution
- [ ] Integration stats/monitoring

## Testing Strategy
- Unit tests for circuit breaker
- Unit tests for retry logic
- Integration tests with mock server
- Performance tests

## Files to Create
- `internal/modules/integration/types.go`
- `internal/modules/integration/circuit.go`
- `internal/modules/integration/client.go`
- `internal/modules/integration/module.go`
- `internal/modules/integration/integration_test.go`

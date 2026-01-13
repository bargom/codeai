// Package integration provides external API integration capabilities.
package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/bargom/codeai/internal/ast"
)

// Client represents an HTTP client for an external integration.
type Client struct {
	Name           string
	IntgType       ast.IntegrationType
	BaseURL        string
	Auth           *AuthConfig
	Timeout        time.Duration
	CircuitBreaker *CircuitBreaker
	httpClient     *http.Client
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	Type   ast.IntegrationAuthType
	Token  string            // For bearer auth
	Header string            // For API key auth - header name
	Value  string            // For API key auth - value
	Config map[string]string // Additional config
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu               sync.Mutex
	failureThreshold int
	timeout          time.Duration
	maxConcurrent    int
	failures         int
	lastFailure      time.Time
	state            CircuitState
	concurrent       int
}

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// IntegrationRegistry manages registered integrations.
type IntegrationRegistry struct {
	mu           sync.RWMutex
	integrations map[string]*Client
}

// NewIntegrationRegistry creates a new integration registry.
func NewIntegrationRegistry() *IntegrationRegistry {
	return &IntegrationRegistry{
		integrations: make(map[string]*Client),
	}
}

// LoadIntegrationFromAST creates and registers an integration client from AST.
func (r *IntegrationRegistry) LoadIntegrationFromAST(intg *ast.IntegrationDecl) (*Client, error) {
	if intg == nil {
		return nil, fmt.Errorf("integration declaration is nil")
	}

	// Validate base URL
	if _, err := url.Parse(intg.BaseURL); err != nil {
		return nil, fmt.Errorf("invalid base URL '%s': %w", intg.BaseURL, err)
	}

	// Parse timeout
	timeout := 30 * time.Second // default
	if intg.Timeout != "" {
		d, err := time.ParseDuration(intg.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout '%s': %w", intg.Timeout, err)
		}
		timeout = d
	}

	// Build auth config
	var authConfig *AuthConfig
	if intg.Auth != nil {
		authConfig = buildAuthConfig(intg.Auth)
	}

	// Build circuit breaker
	var cb *CircuitBreaker
	if intg.CircuitBreaker != nil {
		cb = ConfigureCircuitBreaker(intg.CircuitBreaker)
	}

	client := &Client{
		Name:           intg.Name,
		IntgType:       intg.IntgType,
		BaseURL:        intg.BaseURL,
		Auth:           authConfig,
		Timeout:        timeout,
		CircuitBreaker: cb,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}

	// Register the client
	r.mu.Lock()
	r.integrations[intg.Name] = client
	r.mu.Unlock()

	return client, nil
}

// buildAuthConfig builds auth configuration from AST.
func buildAuthConfig(auth *ast.IntegrationAuthDecl) *AuthConfig {
	config := &AuthConfig{
		Type:   auth.AuthType,
		Config: make(map[string]string),
	}

	// Extract config values
	for key, expr := range auth.Config {
		if strLit, ok := expr.(*ast.StringLiteral); ok {
			config.Config[key] = strLit.Value
		} else if funcCall, ok := expr.(*ast.FunctionCall); ok {
			// Handle env() function calls
			if funcCall.Name == "env" && len(funcCall.Args) > 0 {
				if arg, ok := funcCall.Args[0].(*ast.StringLiteral); ok {
					// Store as env reference to be resolved later
					config.Config[key] = "${" + arg.Value + "}"
				}
			}
		}
	}

	// Extract known fields
	if token, ok := config.Config["token"]; ok {
		config.Token = token
	}
	if header, ok := config.Config["header"]; ok {
		config.Header = header
	}
	if value, ok := config.Config["value"]; ok {
		config.Value = value
	}

	return config
}

// ConfigureCircuitBreaker creates a circuit breaker from AST config.
func ConfigureCircuitBreaker(config *ast.CircuitBreakerConfig) *CircuitBreaker {
	timeout, _ := time.ParseDuration(config.Timeout)
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &CircuitBreaker{
		failureThreshold: config.FailureThreshold,
		timeout:          timeout,
		maxConcurrent:    config.MaxConcurrent,
		state:            CircuitClosed,
	}
}

// GetClient returns a registered integration client.
func (r *IntegrationRegistry) GetClient(name string) (*Client, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	client, exists := r.integrations[name]
	return client, exists
}

// Count returns the number of registered integrations.
func (r *IntegrationRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.integrations)
}

// Do executes an HTTP request through the integration client.
func (c *Client) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	// Check circuit breaker
	if c.CircuitBreaker != nil {
		if err := c.CircuitBreaker.Allow(); err != nil {
			return nil, err
		}
		defer c.CircuitBreaker.Done()
	}

	// Build URL
	fullURL := c.BaseURL + path

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		c.recordFailure()
		return nil, err
	}

	// Apply authentication
	if c.Auth != nil {
		applyAuth(req, c.Auth)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.recordFailure()
		return nil, err
	}

	// Record success or failure based on status code
	if resp.StatusCode >= 500 {
		c.recordFailure()
	} else {
		c.recordSuccess()
	}

	return resp, nil
}

// applyAuth applies authentication to an HTTP request.
func applyAuth(req *http.Request, auth *AuthConfig) {
	switch auth.Type {
	case ast.IntegrationAuthBearer:
		req.Header.Set("Authorization", "Bearer "+auth.Token)
	case ast.IntegrationAuthAPIKey:
		if auth.Header != "" {
			req.Header.Set(auth.Header, auth.Value)
		}
	case ast.IntegrationAuthBasic:
		// Basic auth would be set here
	}
}

// recordFailure records a failure for circuit breaker.
func (c *Client) recordFailure() {
	if c.CircuitBreaker != nil {
		c.CircuitBreaker.RecordFailure()
	}
}

// recordSuccess records a success for circuit breaker.
func (c *Client) recordSuccess() {
	if c.CircuitBreaker != nil {
		c.CircuitBreaker.RecordSuccess()
	}
}

// Allow checks if a request is allowed by the circuit breaker.
func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = CircuitHalfOpen
			cb.concurrent++
			return nil
		}
		return fmt.Errorf("circuit breaker is open")
	case CircuitHalfOpen:
		// Allow limited requests in half-open state
		if cb.concurrent >= 1 {
			return fmt.Errorf("circuit breaker is half-open, waiting for test request")
		}
		cb.concurrent++
		return nil
	case CircuitClosed:
		// Check concurrent limit
		if cb.maxConcurrent > 0 && cb.concurrent >= cb.maxConcurrent {
			return fmt.Errorf("max concurrent requests reached")
		}
		cb.concurrent++
		return nil
	}

	return nil
}

// Done decrements the concurrent request counter.
func (cb *CircuitBreaker) Done() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.concurrent > 0 {
		cb.concurrent--
	}
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.failureThreshold {
		cb.state = CircuitOpen
	}
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitHalfOpen {
		// Reset on success in half-open state
		cb.state = CircuitClosed
		cb.failures = 0
	}
}

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Package graphql provides a GraphQL client with resilience patterns.
package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/bargom/codeai/pkg/integration"
	"github.com/bargom/codeai/pkg/metrics"
)

// Client is a GraphQL client with resilience patterns.
type Client struct {
	config         integration.Config
	httpClient     *http.Client
	circuitBreaker *integration.CircuitBreaker
	retryer        *integration.Retryer
	timeoutManager *integration.TimeoutManager
	logger         *slog.Logger
}

// New creates a new GraphQL client with the given configuration.
func New(config integration.Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	client := &Client{
		config: config,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   config.Timeout.Default,
		},
		circuitBreaker: integration.NewCircuitBreaker(config.ServiceName, config.CircuitBreaker),
		retryer:        integration.NewRetryer(config.Retry).WithService(config.ServiceName, "graphql"),
		timeoutManager: integration.NewTimeoutManager(config.Timeout).WithService(config.ServiceName, "graphql"),
		logger:         slog.Default().With("component", "graphql_client", "service", config.ServiceName),
	}

	return client, nil
}

// Request represents a GraphQL request.
type Request struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

// Response represents a GraphQL response.
type Response struct {
	Data       json.RawMessage `json:"data"`
	Errors     []Error         `json:"errors,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
	StatusCode int
	Duration   time.Duration
}

// Error represents a GraphQL error.
type Error struct {
	Message    string                 `json:"message"`
	Locations  []Location             `json:"locations,omitempty"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// Error implements the error interface.
func (e Error) Error() string {
	return e.Message
}

// Location represents a location in a GraphQL document.
type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// ExecuteOptions configures a GraphQL execution.
type ExecuteOptions struct {
	Timeout     time.Duration
	SkipRetry   bool
	SkipCircuit bool
	Headers     http.Header
}

// Execute executes a GraphQL query or mutation.
func (c *Client) Execute(ctx context.Context, req *Request, opts *ExecuteOptions) (*Response, error) {
	if opts == nil {
		opts = &ExecuteOptions{}
	}

	timer := c.newTimer()

	// Check circuit breaker
	if !opts.SkipCircuit {
		if err := c.circuitBreaker.Allow(); err != nil {
			if timer != nil {
				timer.Error("circuit_open")
			}
			return nil, err
		}
	}

	var resp *Response
	var err error

	// Execute with retry for network errors only
	if opts.SkipRetry {
		resp, err = c.execute(ctx, req, opts)
	} else {
		// Only retry on network errors, not GraphQL errors
		resp, err = integration.DoWithResult(ctx, c.retryer, func(ctx context.Context) (*Response, error) {
			return c.execute(ctx, req, opts)
		})
	}

	// Record circuit breaker result (only for network errors, not GraphQL errors)
	if !opts.SkipCircuit {
		if err != nil {
			c.circuitBreaker.RecordFailure()
		} else {
			c.circuitBreaker.RecordSuccess()
		}
	}

	// Record metrics
	if err != nil {
		if timer != nil {
			timer.Error(metrics.ClassifyError(err))
		}
		return nil, err
	}

	if timer != nil {
		timer.Done(resp.StatusCode)
	}
	return resp, nil
}

// execute performs the actual GraphQL request.
func (c *Client) execute(ctx context.Context, req *Request, opts *ExecuteOptions) (*Response, error) {
	start := time.Now()

	// Marshal request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if c.config.UserAgent != "" {
		httpReq.Header.Set("User-Agent", c.config.UserAgent)
	}

	// Set config headers
	for key, value := range c.config.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set request-specific headers
	if opts.Headers != nil {
		for key, values := range opts.Headers {
			for _, value := range values {
				httpReq.Header.Add(key, value)
			}
		}
	}

	// Apply authentication
	c.applyAuth(httpReq)

	// Apply timeout
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = c.config.Timeout.Default
	}

	var httpResp *http.Response
	err = c.timeoutManager.Execute(ctx, timeout, "graphql", func(timeoutCtx context.Context) error {
		httpReq = httpReq.WithContext(timeoutCtx)
		var execErr error
		httpResp, execErr = c.httpClient.Do(httpReq)
		return execErr
	})

	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if httpResp.StatusCode >= 400 {
		return nil, integration.NewHTTPErrorWithBody(
			httpResp.StatusCode,
			fmt.Sprintf("HTTP %d: %s", httpResp.StatusCode, http.StatusText(httpResp.StatusCode)),
			respBody,
		)
	}

	// Parse GraphQL response
	var resp Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	resp.StatusCode = httpResp.StatusCode
	resp.Duration = time.Since(start)

	// Log GraphQL errors (but don't fail - let caller handle them)
	if len(resp.Errors) > 0 {
		c.logger.Warn("graphql errors in response",
			"error_count", len(resp.Errors),
			"first_error", resp.Errors[0].Message,
		)
	}

	return &resp, nil
}

// applyAuth applies authentication to the request.
func (c *Client) applyAuth(req *http.Request) {
	auth := c.config.Auth

	switch auth.Type {
	case integration.AuthBearer:
		req.Header.Set("Authorization", "Bearer "+auth.Token)

	case integration.AuthAPIKey:
		header := auth.APIKeyHeader
		if header == "" {
			header = "X-API-Key"
		}
		req.Header.Set(header, auth.APIKey)

	case integration.AuthBasic:
		req.SetBasicAuth(auth.Username, auth.Password)

	case integration.AuthOAuth2:
		if auth.OAuth2Config != nil && auth.OAuth2Config.Token != "" {
			req.Header.Set("Authorization", "Bearer "+auth.OAuth2Config.Token)
		}
	}
}

// newTimer creates a new metrics timer.
func (c *Client) newTimer() *metrics.IntegrationCallTimer {
	if !c.config.EnableMetrics {
		return nil
	}
	reg := metrics.Global()
	if reg == nil {
		return nil
	}
	return reg.Integration().NewCallTimer(c.config.ServiceName, "graphql")
}

// Query executes a GraphQL query.
func (c *Client) Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	req := &Request{
		Query:     query,
		Variables: variables,
	}

	resp, err := c.Execute(ctx, req, nil)
	if err != nil {
		return err
	}

	if len(resp.Errors) > 0 {
		return &GraphQLError{Errors: resp.Errors}
	}

	if result != nil && len(resp.Data) > 0 {
		if err := json.Unmarshal(resp.Data, result); err != nil {
			return fmt.Errorf("failed to unmarshal data: %w", err)
		}
	}

	return nil
}

// Mutate executes a GraphQL mutation.
func (c *Client) Mutate(ctx context.Context, mutation string, variables map[string]interface{}, result interface{}) error {
	return c.Query(ctx, mutation, variables, result)
}

// QueryWithOperation executes a named GraphQL query.
func (c *Client) QueryWithOperation(ctx context.Context, query, operationName string, variables map[string]interface{}, result interface{}) error {
	req := &Request{
		Query:         query,
		OperationName: operationName,
		Variables:     variables,
	}

	resp, err := c.Execute(ctx, req, nil)
	if err != nil {
		return err
	}

	if len(resp.Errors) > 0 {
		return &GraphQLError{Errors: resp.Errors}
	}

	if result != nil && len(resp.Data) > 0 {
		if err := json.Unmarshal(resp.Data, result); err != nil {
			return fmt.Errorf("failed to unmarshal data: %w", err)
		}
	}

	return nil
}

// GraphQLError represents a GraphQL error response.
type GraphQLError struct {
	Errors []Error
}

// Error implements the error interface.
func (e *GraphQLError) Error() string {
	if len(e.Errors) == 0 {
		return "unknown graphql error"
	}
	return e.Errors[0].Message
}

// HasError checks if the response contains a specific error.
func (r *Response) HasError() bool {
	return len(r.Errors) > 0
}

// FirstError returns the first error if present.
func (r *Response) FirstError() *Error {
	if len(r.Errors) > 0 {
		return &r.Errors[0]
	}
	return nil
}

// UnmarshalData unmarshals the response data into the given value.
func (r *Response) UnmarshalData(v interface{}) error {
	if len(r.Data) == 0 {
		return nil
	}
	return json.Unmarshal(r.Data, v)
}

// CircuitBreaker returns the circuit breaker for this client.
func (c *Client) CircuitBreaker() *integration.CircuitBreaker {
	return c.circuitBreaker
}

// Config returns the client configuration.
func (c *Client) Config() integration.Config {
	return c.config
}

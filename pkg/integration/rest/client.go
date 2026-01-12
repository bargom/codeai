// Package rest provides a REST client with resilience patterns.
package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bargom/codeai/pkg/integration"
	"github.com/bargom/codeai/pkg/metrics"
)

// Client is an HTTP client with resilience patterns.
type Client struct {
	config         integration.Config
	httpClient     *http.Client
	circuitBreaker *integration.CircuitBreaker
	retryer        *integration.Retryer
	timeoutManager *integration.TimeoutManager
	logger         *slog.Logger
	middleware     []Middleware
}

// New creates a new REST client with the given configuration.
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
		retryer:        integration.NewRetryer(config.Retry).WithService(config.ServiceName, ""),
		timeoutManager: integration.NewTimeoutManager(config.Timeout).WithService(config.ServiceName, ""),
		logger:         slog.Default().With("component", "rest_client", "service", config.ServiceName),
		middleware:     make([]Middleware, 0),
	}

	// Add default middleware
	if config.EnableLogging {
		client.middleware = append(client.middleware, NewLoggingMiddleware(config.RedactFields))
	}
	if config.EnableMetrics {
		client.middleware = append(client.middleware, NewMetricsMiddleware(config.ServiceName))
	}

	return client, nil
}

// Use adds middleware to the client.
func (c *Client) Use(mw ...Middleware) *Client {
	c.middleware = append(c.middleware, mw...)
	return c
}

// Request represents an HTTP request.
type Request struct {
	Method      string
	Path        string
	Query       url.Values
	Headers     http.Header
	Body        interface{}
	Timeout     time.Duration
	SkipRetry   bool
	SkipCircuit bool
}

// Response represents an HTTP response.
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Duration   time.Duration
}

// Do executes an HTTP request.
func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	endpoint := req.Path
	timer := c.newTimer(endpoint)

	// Check circuit breaker
	if !req.SkipCircuit {
		if err := c.circuitBreaker.Allow(); err != nil {
			if timer != nil {
				timer.Error("circuit_open")
			}
			return nil, err
		}
	}

	var resp *Response
	var err error

	// Execute with retry
	if req.SkipRetry {
		resp, err = c.execute(ctx, req)
	} else {
		retryer := c.retryer.WithService(c.config.ServiceName, endpoint)
		resp, err = integration.DoWithResult(ctx, retryer, func(ctx context.Context) (*Response, error) {
			return c.execute(ctx, req)
		})
	}

	// Record circuit breaker result
	if !req.SkipCircuit {
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

// execute performs the actual HTTP request.
func (c *Client) execute(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	// Build URL
	reqURL, err := c.buildURL(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Build body
	var bodyReader io.Reader
	if req.Body != nil {
		bodyData, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyData)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if c.config.UserAgent != "" {
		httpReq.Header.Set("User-Agent", c.config.UserAgent)
	}

	// Set config headers
	for key, value := range c.config.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set request headers
	for key, values := range req.Headers {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	// Apply authentication
	c.applyAuth(httpReq)

	// Apply middleware (request)
	for _, mw := range c.middleware {
		if err := mw.HandleRequest(httpReq); err != nil {
			return nil, err
		}
	}

	// Apply timeout
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = c.config.Timeout.Default
	}

	var httpResp *http.Response
	err = c.timeoutManager.Execute(ctx, timeout, req.Path, func(timeoutCtx context.Context) error {
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

	resp := &Response{
		StatusCode: httpResp.StatusCode,
		Headers:    httpResp.Header,
		Body:       respBody,
		Duration:   time.Since(start),
	}

	// Apply middleware (response)
	for _, mw := range c.middleware {
		if err := mw.HandleResponse(resp); err != nil {
			return nil, err
		}
	}

	// Check for HTTP errors
	if httpResp.StatusCode >= 400 {
		return resp, integration.NewHTTPErrorWithBody(
			httpResp.StatusCode,
			fmt.Sprintf("HTTP %d: %s", httpResp.StatusCode, http.StatusText(httpResp.StatusCode)),
			respBody,
		)
	}

	return resp, nil
}

// buildURL builds the full URL for the request.
func (c *Client) buildURL(req *Request) (string, error) {
	base := strings.TrimSuffix(c.config.BaseURL, "/")
	path := req.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	fullURL := base + path

	if len(req.Query) > 0 {
		fullURL += "?" + req.Query.Encode()
	}

	return fullURL, nil
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

// newTimer creates a new metrics timer for the request.
func (c *Client) newTimer(endpoint string) *metrics.IntegrationCallTimer {
	if !c.config.EnableMetrics {
		return nil
	}
	reg := metrics.Global()
	if reg == nil {
		return nil
	}
	return reg.Integration().NewCallTimer(c.config.ServiceName, endpoint)
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string, opts ...RequestOption) (*Response, error) {
	req := &Request{
		Method: http.MethodGet,
		Path:   path,
	}
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body interface{}, opts ...RequestOption) (*Response, error) {
	req := &Request{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,
	}
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req)
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, path string, body interface{}, opts ...RequestOption) (*Response, error) {
	req := &Request{
		Method: http.MethodPut,
		Path:   path,
		Body:   body,
	}
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req)
}

// Patch performs a PATCH request.
func (c *Client) Patch(ctx context.Context, path string, body interface{}, opts ...RequestOption) (*Response, error) {
	req := &Request{
		Method: http.MethodPatch,
		Path:   path,
		Body:   body,
	}
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string, opts ...RequestOption) (*Response, error) {
	req := &Request{
		Method: http.MethodDelete,
		Path:   path,
	}
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req)
}

// RequestOption configures a request.
type RequestOption func(*Request)

// WithQuery adds query parameters.
func WithQuery(params url.Values) RequestOption {
	return func(r *Request) {
		r.Query = params
	}
}

// WithHeader adds a header.
func WithHeader(key, value string) RequestOption {
	return func(r *Request) {
		if r.Headers == nil {
			r.Headers = make(http.Header)
		}
		r.Headers.Set(key, value)
	}
}

// WithHeaders adds multiple headers.
func WithHeaders(headers http.Header) RequestOption {
	return func(r *Request) {
		if r.Headers == nil {
			r.Headers = make(http.Header)
		}
		for k, v := range headers {
			r.Headers[k] = v
		}
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) RequestOption {
	return func(r *Request) {
		r.Timeout = timeout
	}
}

// WithSkipRetry skips retry for this request.
func WithSkipRetry() RequestOption {
	return func(r *Request) {
		r.SkipRetry = true
	}
}

// WithSkipCircuit skips circuit breaker for this request.
func WithSkipCircuit() RequestOption {
	return func(r *Request) {
		r.SkipCircuit = true
	}
}

// UnmarshalJSON unmarshals the response body into the given value.
func (r *Response) UnmarshalJSON(v interface{}) error {
	if len(r.Body) == 0 {
		return nil
	}
	return json.Unmarshal(r.Body, v)
}

// IsSuccess returns true if the response has a 2xx status code.
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsClientError returns true if the response has a 4xx status code.
func (r *Response) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// IsServerError returns true if the response has a 5xx status code.
func (r *Response) IsServerError() bool {
	return r.StatusCode >= 500
}

// CircuitBreaker returns the circuit breaker for this client.
func (c *Client) CircuitBreaker() *integration.CircuitBreaker {
	return c.circuitBreaker
}

// Config returns the client configuration.
func (c *Client) Config() integration.Config {
	return c.config
}

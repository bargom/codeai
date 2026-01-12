package rest

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bargom/codeai/pkg/metrics"
)

// Middleware is an interface for request/response middleware.
type Middleware interface {
	HandleRequest(req *http.Request) error
	HandleResponse(resp *Response) error
}

// LoggingMiddleware logs request and response details.
type LoggingMiddleware struct {
	logger       *slog.Logger
	redactFields []string
}

// NewLoggingMiddleware creates a new logging middleware.
func NewLoggingMiddleware(redactFields []string) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger:       slog.Default().With("component", "rest_middleware"),
		redactFields: redactFields,
	}
}

// HandleRequest logs the outgoing request.
func (m *LoggingMiddleware) HandleRequest(req *http.Request) error {
	m.logger.Debug("outgoing request",
		"method", req.Method,
		"url", m.redactURL(req.URL.String()),
		"headers", m.redactHeaders(req.Header),
	)
	return nil
}

// HandleResponse logs the incoming response.
func (m *LoggingMiddleware) HandleResponse(resp *Response) error {
	m.logger.Debug("incoming response",
		"status_code", resp.StatusCode,
		"duration", resp.Duration,
		"body_size", len(resp.Body),
	)
	return nil
}

// redactURL redacts sensitive query parameters from the URL.
func (m *LoggingMiddleware) redactURL(urlStr string) string {
	for _, field := range m.redactFields {
		// Simple redaction for query parameters
		lower := strings.ToLower(field)
		patterns := []string{
			lower + "=",
			strings.ReplaceAll(lower, "_", "") + "=",
		}
		for _, pattern := range patterns {
			if idx := strings.Index(strings.ToLower(urlStr), pattern); idx != -1 {
				// Find the end of the value (next & or end of string)
				endIdx := strings.Index(urlStr[idx:], "&")
				if endIdx == -1 {
					urlStr = urlStr[:idx] + pattern + "[REDACTED]"
				} else {
					urlStr = urlStr[:idx] + pattern + "[REDACTED]" + urlStr[idx+endIdx:]
				}
			}
		}
	}
	return urlStr
}

// redactHeaders redacts sensitive headers.
func (m *LoggingMiddleware) redactHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range headers {
		lowerKey := strings.ToLower(key)
		shouldRedact := false

		for _, field := range m.redactFields {
			if strings.Contains(lowerKey, strings.ToLower(field)) {
				shouldRedact = true
				break
			}
		}

		// Always redact Authorization header
		if lowerKey == "authorization" {
			shouldRedact = true
		}

		if shouldRedact {
			result[key] = "[REDACTED]"
		} else if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}

// MetricsMiddleware records request metrics.
type MetricsMiddleware struct {
	serviceName string
	timer       *metrics.IntegrationCallTimer
}

// NewMetricsMiddleware creates a new metrics middleware.
func NewMetricsMiddleware(serviceName string) *MetricsMiddleware {
	return &MetricsMiddleware{
		serviceName: serviceName,
	}
}

// HandleRequest starts timing the request.
func (m *MetricsMiddleware) HandleRequest(req *http.Request) error {
	if reg := metrics.Global(); reg != nil {
		m.timer = reg.Integration().NewCallTimer(m.serviceName, req.URL.Path)
	}
	return nil
}

// HandleResponse records the response metrics.
func (m *MetricsMiddleware) HandleResponse(resp *Response) error {
	if m.timer != nil {
		m.timer.Done(resp.StatusCode)
	}
	return nil
}

// RetryMiddleware handles retry logic.
type RetryMiddleware struct {
	logger *slog.Logger
}

// NewRetryMiddleware creates a new retry middleware.
func NewRetryMiddleware() *RetryMiddleware {
	return &RetryMiddleware{
		logger: slog.Default().With("component", "retry_middleware"),
	}
}

// HandleRequest is a no-op for retry middleware.
func (m *RetryMiddleware) HandleRequest(req *http.Request) error {
	return nil
}

// HandleResponse is a no-op for retry middleware.
func (m *RetryMiddleware) HandleResponse(resp *Response) error {
	return nil
}

// HeaderMiddleware adds headers to requests.
type HeaderMiddleware struct {
	headers map[string]string
}

// NewHeaderMiddleware creates a new header middleware.
func NewHeaderMiddleware(headers map[string]string) *HeaderMiddleware {
	return &HeaderMiddleware{
		headers: headers,
	}
}

// HandleRequest adds headers to the request.
func (m *HeaderMiddleware) HandleRequest(req *http.Request) error {
	for key, value := range m.headers {
		req.Header.Set(key, value)
	}
	return nil
}

// HandleResponse is a no-op for header middleware.
func (m *HeaderMiddleware) HandleResponse(resp *Response) error {
	return nil
}

// RequestIDMiddleware adds a request ID to requests.
type RequestIDMiddleware struct {
	generator func() string
}

// NewRequestIDMiddleware creates a new request ID middleware.
func NewRequestIDMiddleware(generator func() string) *RequestIDMiddleware {
	return &RequestIDMiddleware{
		generator: generator,
	}
}

// HandleRequest adds a request ID to the request.
func (m *RequestIDMiddleware) HandleRequest(req *http.Request) error {
	if m.generator != nil {
		req.Header.Set("X-Request-ID", m.generator())
	}
	return nil
}

// HandleResponse is a no-op for request ID middleware.
func (m *RequestIDMiddleware) HandleResponse(resp *Response) error {
	return nil
}

// RateLimitMiddleware handles rate limiting.
type RateLimitMiddleware struct {
	logger       *slog.Logger
	retryAfter   time.Duration
	lastRateLimit time.Time
}

// NewRateLimitMiddleware creates a new rate limit middleware.
func NewRateLimitMiddleware() *RateLimitMiddleware {
	return &RateLimitMiddleware{
		logger: slog.Default().With("component", "rate_limit_middleware"),
	}
}

// HandleRequest is a no-op for rate limit middleware.
func (m *RateLimitMiddleware) HandleRequest(req *http.Request) error {
	return nil
}

// HandleResponse checks for rate limit headers.
func (m *RateLimitMiddleware) HandleResponse(resp *Response) error {
	if resp.StatusCode == http.StatusTooManyRequests {
		m.lastRateLimit = time.Now()
		if retryAfter := resp.Headers.Get("Retry-After"); retryAfter != "" {
			// Try to parse as duration
			if d, err := time.ParseDuration(retryAfter + "s"); err == nil {
				m.retryAfter = d
			}
		}
		m.logger.Warn("rate limited",
			"retry_after", m.retryAfter,
		)
	}
	return nil
}

// BodyRedactionMiddleware redacts sensitive fields from request/response bodies.
type BodyRedactionMiddleware struct {
	redactFields []string
}

// NewBodyRedactionMiddleware creates a new body redaction middleware.
func NewBodyRedactionMiddleware(fields []string) *BodyRedactionMiddleware {
	return &BodyRedactionMiddleware{
		redactFields: fields,
	}
}

// HandleRequest is a no-op (redaction happens during logging).
func (m *BodyRedactionMiddleware) HandleRequest(req *http.Request) error {
	return nil
}

// HandleResponse is a no-op (redaction happens during logging).
func (m *BodyRedactionMiddleware) HandleResponse(resp *Response) error {
	return nil
}

// RedactJSON redacts sensitive fields from a JSON byte slice.
func (m *BodyRedactionMiddleware) RedactJSON(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return data
	}

	m.redactMap(obj)

	result, err := json.Marshal(obj)
	if err != nil {
		return data
	}
	return result
}

func (m *BodyRedactionMiddleware) redactMap(obj map[string]interface{}) {
	for key, value := range obj {
		lowerKey := strings.ToLower(key)
		shouldRedact := false

		for _, field := range m.redactFields {
			if strings.Contains(lowerKey, strings.ToLower(field)) {
				shouldRedact = true
				break
			}
		}

		if shouldRedact {
			obj[key] = "[REDACTED]"
		} else if nestedMap, ok := value.(map[string]interface{}); ok {
			m.redactMap(nestedMap)
		} else if nestedSlice, ok := value.([]interface{}); ok {
			m.redactSlice(nestedSlice)
		}
	}
}

func (m *BodyRedactionMiddleware) redactSlice(arr []interface{}) {
	for i, value := range arr {
		if nestedMap, ok := value.(map[string]interface{}); ok {
			m.redactMap(nestedMap)
			arr[i] = nestedMap
		} else if nestedSlice, ok := value.([]interface{}); ok {
			m.redactSlice(nestedSlice)
		}
	}
}

// CompressionMiddleware handles request/response compression.
type CompressionMiddleware struct{}

// NewCompressionMiddleware creates a new compression middleware.
func NewCompressionMiddleware() *CompressionMiddleware {
	return &CompressionMiddleware{}
}

// HandleRequest adds Accept-Encoding header.
func (m *CompressionMiddleware) HandleRequest(req *http.Request) error {
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	return nil
}

// HandleResponse handles decompression (handled by http.Client automatically).
func (m *CompressionMiddleware) HandleResponse(resp *Response) error {
	return nil
}

// CachingMiddleware provides simple response caching.
type CachingMiddleware struct {
	cache    map[string]*cachedResponse
	maxAge   time.Duration
}

type cachedResponse struct {
	response  *Response
	timestamp time.Time
}

// NewCachingMiddleware creates a new caching middleware.
func NewCachingMiddleware(maxAge time.Duration) *CachingMiddleware {
	return &CachingMiddleware{
		cache:  make(map[string]*cachedResponse),
		maxAge: maxAge,
	}
}

// HandleRequest is a no-op (cache lookup happens elsewhere).
func (m *CachingMiddleware) HandleRequest(req *http.Request) error {
	return nil
}

// HandleResponse caches the response.
func (m *CachingMiddleware) HandleResponse(resp *Response) error {
	return nil
}

// Get retrieves a cached response.
func (m *CachingMiddleware) Get(key string) (*Response, bool) {
	if cached, ok := m.cache[key]; ok {
		if time.Since(cached.timestamp) < m.maxAge {
			// Clone the response
			respCopy := &Response{
				StatusCode: cached.response.StatusCode,
				Headers:    cached.response.Headers.Clone(),
				Body:       bytes.Clone(cached.response.Body),
				Duration:   cached.response.Duration,
			}
			return respCopy, true
		}
		delete(m.cache, key)
	}
	return nil, false
}

// Set caches a response.
func (m *CachingMiddleware) Set(key string, resp *Response) {
	m.cache[key] = &cachedResponse{
		response:  resp,
		timestamp: time.Now(),
	}
}

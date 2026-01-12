package integration

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bargom/codeai/pkg/metrics"
)

// RetryConfig configures the retry behavior.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (including the first one).
	// Default: 3
	MaxAttempts int

	// BaseDelay is the initial delay between retries.
	// Default: 100ms
	BaseDelay time.Duration

	// MaxDelay is the maximum delay between retries.
	// Default: 30s
	MaxDelay time.Duration

	// Multiplier is the factor by which the delay increases.
	// Default: 2.0
	Multiplier float64

	// Jitter adds randomness to delays to prevent thundering herd.
	// Value between 0 and 1 representing the percentage of jitter (e.g., 0.25 = ±25%).
	// Default: 0.25
	Jitter float64

	// RetryIf is a function that determines if an error should be retried.
	// If nil, uses the default retryable check.
	RetryIf func(err error) bool

	// OnRetry is called before each retry attempt.
	OnRetry func(attempt int, err error, delay time.Duration)
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
		Jitter:      0.25,
	}
}

// Retryer implements retry logic with exponential backoff.
type Retryer struct {
	config      RetryConfig
	logger      *slog.Logger
	serviceName string
	endpoint    string
}

// NewRetryer creates a new retryer with the given configuration.
func NewRetryer(config RetryConfig) *Retryer {
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 3
	}
	if config.BaseDelay <= 0 {
		config.BaseDelay = 100 * time.Millisecond
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 30 * time.Second
	}
	if config.Multiplier <= 0 {
		config.Multiplier = 2.0
	}
	if config.Jitter < 0 || config.Jitter > 1 {
		config.Jitter = 0.25
	}

	return &Retryer{
		config: config,
		logger: slog.Default().With("component", "retryer"),
	}
}

// WithService returns a new retryer configured for a specific service and endpoint.
func (r *Retryer) WithService(serviceName, endpoint string) *Retryer {
	return &Retryer{
		config:      r.config,
		logger:      r.logger.With("service", serviceName, "endpoint", endpoint),
		serviceName: serviceName,
		endpoint:    endpoint,
	}
}

// Do executes the function with retry logic.
func (r *Retryer) Do(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error
	delay := r.config.BaseDelay

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if context is cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Check if this is the last attempt
		if attempt >= r.config.MaxAttempts {
			break
		}

		// Check if error is retryable
		if !r.isRetryable(err) {
			return err
		}

		// Calculate delay with jitter
		actualDelay := r.addJitter(delay)

		r.logger.Warn("retrying request",
			"attempt", attempt,
			"max_attempts", r.config.MaxAttempts,
			"error", err.Error(),
			"delay", actualDelay,
		)

		if r.config.OnRetry != nil {
			r.config.OnRetry(attempt, err, actualDelay)
		}

		// Record retry metric
		if r.serviceName != "" {
			if reg := metrics.Global(); reg != nil {
				reg.Integration().RecordRetry(r.serviceName, r.endpoint)
			}
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(actualDelay):
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * r.config.Multiplier)
		if delay > r.config.MaxDelay {
			delay = r.config.MaxDelay
		}
	}

	return lastErr
}

// DoWithResult executes a function that returns a result with retry logic.
func DoWithResult[T any](ctx context.Context, r *Retryer, fn func(context.Context) (T, error)) (T, error) {
	var result T
	var lastErr error
	delay := r.config.BaseDelay

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		res, err := fn(ctx)
		if err == nil {
			return res, nil
		}

		lastErr = err

		// Check if context is cancelled
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		// Check if this is the last attempt
		if attempt >= r.config.MaxAttempts {
			break
		}

		// Check if error is retryable
		if !r.isRetryable(err) {
			return result, err
		}

		// Calculate delay with jitter
		actualDelay := r.addJitter(delay)

		r.logger.Warn("retrying request",
			"attempt", attempt,
			"max_attempts", r.config.MaxAttempts,
			"error", err.Error(),
			"delay", actualDelay,
		)

		if r.config.OnRetry != nil {
			r.config.OnRetry(attempt, err, actualDelay)
		}

		// Record retry metric
		if r.serviceName != "" {
			if reg := metrics.Global(); reg != nil {
				reg.Integration().RecordRetry(r.serviceName, r.endpoint)
			}
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(actualDelay):
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * r.config.Multiplier)
		if delay > r.config.MaxDelay {
			delay = r.config.MaxDelay
		}
	}

	return result, lastErr
}

// isRetryable checks if an error should be retried.
func (r *Retryer) isRetryable(err error) bool {
	if r.config.RetryIf != nil {
		return r.config.RetryIf(err)
	}
	return IsRetryable(err)
}

// addJitter adds randomness to the delay.
func (r *Retryer) addJitter(delay time.Duration) time.Duration {
	if r.config.Jitter <= 0 {
		return delay
	}

	// Random value between -jitter and +jitter
	jitterRange := float64(delay) * r.config.Jitter
	jitter := (rand.Float64()*2 - 1) * jitterRange
	result := time.Duration(float64(delay) + jitter)

	if result < 0 {
		return delay
	}
	return result
}

// IsRetryable checks if an error is retryable using the default logic.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for timeout errors
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Check for connection errors (OpError) first since it implements net.Error
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// Check for temporary network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() || isTemporaryNetError(netErr) {
			return true
		}
	}

	// Check for specific HTTP status codes via HTTPError
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return IsRetryableStatusCode(httpErr.StatusCode)
	}

	// Check error message for common retryable conditions
	errMsg := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"no such host",
		"temporary failure",
		"service unavailable",
		"bad gateway",
		"gateway timeout",
		"too many requests",
		"i/o timeout",
		"network is unreachable",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// isTemporaryNetError checks if the net.Error is temporary.
// The Temporary() method is deprecated but we still check it for compatibility.
func isTemporaryNetError(err net.Error) bool {
	// Check if the error has a Temporary method that returns true
	type temporary interface {
		Temporary() bool
	}
	if t, ok := err.(temporary); ok {
		return t.Temporary()
	}
	return false
}

// IsRetryableStatusCode checks if an HTTP status code should be retried.
func IsRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,    // 429
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return statusCode >= 500
	}
}

// HTTPError represents an HTTP error with status code.
type HTTPError struct {
	StatusCode int
	Message    string
	Body       []byte
}

// Error implements the error interface.
func (e *HTTPError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return http.StatusText(e.StatusCode)
}

// NewHTTPError creates a new HTTP error.
func NewHTTPError(statusCode int, message string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// NewHTTPErrorWithBody creates a new HTTP error with response body.
func NewHTTPErrorWithBody(statusCode int, message string, body []byte) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
		Body:       body,
	}
}

// Retry executes a function with the default retry configuration.
func Retry(ctx context.Context, fn func(context.Context) error) error {
	return NewRetryer(DefaultRetryConfig()).Do(ctx, fn)
}

// RetryWithConfig executes a function with the given retry configuration.
func RetryWithConfig(ctx context.Context, config RetryConfig, fn func(context.Context) error) error {
	return NewRetryer(config).Do(ctx, fn)
}

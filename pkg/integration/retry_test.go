package integration

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, config.BaseDelay)
	assert.Equal(t, 30*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.Multiplier)
	assert.Equal(t, 0.25, config.Jitter)
}

func TestNewRetryer(t *testing.T) {
	t.Run("uses defaults for zero values", func(t *testing.T) {
		r := NewRetryer(RetryConfig{})

		assert.Equal(t, 3, r.config.MaxAttempts)
		assert.Equal(t, 100*time.Millisecond, r.config.BaseDelay)
		assert.Equal(t, 30*time.Second, r.config.MaxDelay)
		assert.Equal(t, 2.0, r.config.Multiplier)
		// Note: Jitter of 0 is valid (no jitter), so it's not replaced with default
		assert.Equal(t, 0.0, r.config.Jitter)
	})

	t.Run("uses provided values", func(t *testing.T) {
		config := RetryConfig{
			MaxAttempts: 5,
			BaseDelay:   50 * time.Millisecond,
			MaxDelay:    10 * time.Second,
			Multiplier:  1.5,
			Jitter:      0.1,
		}
		r := NewRetryer(config)

		assert.Equal(t, 5, r.config.MaxAttempts)
		assert.Equal(t, 50*time.Millisecond, r.config.BaseDelay)
		assert.Equal(t, 10*time.Second, r.config.MaxDelay)
		assert.Equal(t, 1.5, r.config.Multiplier)
		assert.Equal(t, 0.1, r.config.Jitter)
	})

	t.Run("clamps invalid jitter", func(t *testing.T) {
		r := NewRetryer(RetryConfig{Jitter: 1.5})
		assert.Equal(t, 0.25, r.config.Jitter)

		r = NewRetryer(RetryConfig{Jitter: -0.5})
		assert.Equal(t, 0.25, r.config.Jitter)
	})
}

func TestRetryerDo(t *testing.T) {
	t.Run("returns nil on success", func(t *testing.T) {
		r := NewRetryer(DefaultRetryConfig())

		err := r.Do(context.Background(), func(ctx context.Context) error {
			return nil
		})

		assert.NoError(t, err)
	})

	t.Run("returns error after max attempts", func(t *testing.T) {
		r := NewRetryer(RetryConfig{
			MaxAttempts: 3,
			BaseDelay:   1 * time.Millisecond,
			Jitter:      0,
		})

		var attempts int
		// Use retryable error (HTTP 500)
		expectedErr := NewHTTPError(http.StatusInternalServerError, "server error")

		err := r.Do(context.Background(), func(ctx context.Context) error {
			attempts++
			return expectedErr
		})

		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("stops on non-retryable error", func(t *testing.T) {
		r := NewRetryer(RetryConfig{
			MaxAttempts: 5,
			BaseDelay:   1 * time.Millisecond,
		})

		var attempts int
		// HTTP 400 is not retryable
		httpErr := NewHTTPError(http.StatusBadRequest, "bad request")

		err := r.Do(context.Background(), func(ctx context.Context) error {
			attempts++
			return httpErr
		})

		assert.Equal(t, httpErr, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("retries on retryable error", func(t *testing.T) {
		r := NewRetryer(RetryConfig{
			MaxAttempts: 3,
			BaseDelay:   1 * time.Millisecond,
			Jitter:      0,
		})

		var attempts int
		// HTTP 503 is retryable
		httpErr := NewHTTPError(http.StatusServiceUnavailable, "service unavailable")

		err := r.Do(context.Background(), func(ctx context.Context) error {
			attempts++
			return httpErr
		})

		assert.Equal(t, httpErr, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("succeeds after retry", func(t *testing.T) {
		r := NewRetryer(RetryConfig{
			MaxAttempts: 3,
			BaseDelay:   1 * time.Millisecond,
			Jitter:      0,
		})

		var attempts int
		err := r.Do(context.Background(), func(ctx context.Context) error {
			attempts++
			if attempts < 3 {
				// Use retryable error (HTTP 503)
				return NewHTTPError(http.StatusServiceUnavailable, "service unavailable")
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		r := NewRetryer(RetryConfig{
			MaxAttempts: 10,
			BaseDelay:   100 * time.Millisecond,
		})

		ctx, cancel := context.WithCancel(context.Background())
		var attempts int32

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		err := r.Do(ctx, func(ctx context.Context) error {
			atomic.AddInt32(&attempts, 1)
			// Use retryable error
			return NewHTTPError(http.StatusServiceUnavailable, "service unavailable")
		})

		assert.Equal(t, context.Canceled, err)
		assert.LessOrEqual(t, atomic.LoadInt32(&attempts), int32(3))
	})

	t.Run("calls OnRetry callback", func(t *testing.T) {
		var retryAttempts []int

		r := NewRetryer(RetryConfig{
			MaxAttempts: 3,
			BaseDelay:   1 * time.Millisecond,
			Jitter:      0,
			OnRetry: func(attempt int, err error, delay time.Duration) {
				retryAttempts = append(retryAttempts, attempt)
			},
		})

		r.Do(context.Background(), func(ctx context.Context) error {
			// Use retryable error
			return NewHTTPError(http.StatusInternalServerError, "server error")
		})

		assert.Equal(t, []int{1, 2}, retryAttempts)
	})

	t.Run("uses custom RetryIf", func(t *testing.T) {
		r := NewRetryer(RetryConfig{
			MaxAttempts: 3,
			BaseDelay:   1 * time.Millisecond,
			RetryIf: func(err error) bool {
				return err.Error() == "retry me"
			},
		})

		var attempts int
		err := r.Do(context.Background(), func(ctx context.Context) error {
			attempts++
			return errors.New("don't retry me")
		})

		assert.Error(t, err)
		assert.Equal(t, 1, attempts)
	})
}

func TestDoWithResult(t *testing.T) {
	t.Run("returns result on success", func(t *testing.T) {
		r := NewRetryer(DefaultRetryConfig())

		result, err := DoWithResult(context.Background(), r, func(ctx context.Context) (int, error) {
			return 42, nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 42, result)
	})

	t.Run("returns error after max attempts", func(t *testing.T) {
		r := NewRetryer(RetryConfig{
			MaxAttempts: 2,
			BaseDelay:   1 * time.Millisecond,
			Jitter:      0,
		})

		var attempts int
		result, err := DoWithResult(context.Background(), r, func(ctx context.Context) (string, error) {
			attempts++
			// Use retryable error
			return "", NewHTTPError(http.StatusInternalServerError, "server error")
		})

		assert.Error(t, err)
		assert.Empty(t, result)
		assert.Equal(t, 2, attempts)
	})

	t.Run("returns result after retry", func(t *testing.T) {
		r := NewRetryer(RetryConfig{
			MaxAttempts: 3,
			BaseDelay:   1 * time.Millisecond,
			Jitter:      0,
		})

		var attempts int
		result, err := DoWithResult(context.Background(), r, func(ctx context.Context) (string, error) {
			attempts++
			if attempts < 2 {
				// Use retryable error
				return "", NewHTTPError(http.StatusServiceUnavailable, "service unavailable")
			}
			return "success", nil
		})

		assert.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, 2, attempts)
	})
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "HTTP 500",
			err:      NewHTTPError(500, "internal server error"),
			expected: true,
		},
		{
			name:     "HTTP 502",
			err:      NewHTTPError(502, "bad gateway"),
			expected: true,
		},
		{
			name:     "HTTP 503",
			err:      NewHTTPError(503, "service unavailable"),
			expected: true,
		},
		{
			name:     "HTTP 504",
			err:      NewHTTPError(504, "gateway timeout"),
			expected: true,
		},
		{
			name:     "HTTP 429",
			err:      NewHTTPError(429, "too many requests"),
			expected: true,
		},
		{
			name:     "HTTP 400",
			err:      NewHTTPError(400, "bad request"),
			expected: false,
		},
		{
			name:     "HTTP 404",
			err:      NewHTTPError(404, "not found"),
			expected: false,
		},
		{
			name:     "HTTP 401",
			err:      NewHTTPError(401, "unauthorized"),
			expected: false,
		},
		{
			name:     "connection refused error",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "connection reset error",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "service unavailable error",
			err:      errors.New("service unavailable"),
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("some random error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsRetryable(tt.err))
		})
	}
}

func TestIsRetryableStatusCode(t *testing.T) {
	tests := []struct {
		code     int
		expected bool
	}{
		{200, false},
		{201, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true},  // Too Many Requests
		{500, true},  // Internal Server Error
		{501, true},  // Not Implemented
		{502, true},  // Bad Gateway
		{503, true},  // Service Unavailable
		{504, true},  // Gateway Timeout
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.code), func(t *testing.T) {
			assert.Equal(t, tt.expected, IsRetryableStatusCode(tt.code))
		})
	}
}

func TestHTTPError(t *testing.T) {
	t.Run("Error returns message", func(t *testing.T) {
		err := NewHTTPError(500, "custom message")
		assert.Equal(t, "custom message", err.Error())
	})

	t.Run("Error returns status text when no message", func(t *testing.T) {
		err := NewHTTPError(404, "")
		assert.Equal(t, "Not Found", err.Error())
	})

	t.Run("NewHTTPErrorWithBody stores body", func(t *testing.T) {
		body := []byte(`{"error": "details"}`)
		err := NewHTTPErrorWithBody(500, "error", body)
		assert.Equal(t, body, err.Body)
	})
}

func TestRetryWithConfig(t *testing.T) {
	var attempts int
	config := RetryConfig{
		MaxAttempts: 2,
		BaseDelay:   1 * time.Millisecond,
		Jitter:      0,
	}

	err := RetryWithConfig(context.Background(), config, func(ctx context.Context) error {
		attempts++
		if attempts < 2 {
			// Use retryable error
			return NewHTTPError(http.StatusServiceUnavailable, "service unavailable")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestRetry(t *testing.T) {
	var attempts int

	err := Retry(context.Background(), func(ctx context.Context) error {
		attempts++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestRetryerWithService(t *testing.T) {
	r := NewRetryer(DefaultRetryConfig())
	serviceRetryer := r.WithService("test-service", "/api/users")

	assert.Equal(t, "test-service", serviceRetryer.serviceName)
	assert.Equal(t, "/api/users", serviceRetryer.endpoint)
}

func TestNetErrorRetry(t *testing.T) {
	t.Run("retries on net.OpError", func(t *testing.T) {
		opErr := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: errors.New("connection refused"),
		}

		assert.True(t, IsRetryable(opErr))
	})
}

func TestExponentialBackoff(t *testing.T) {
	r := NewRetryer(RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      0, // No jitter for predictable timing
	})

	var timestamps []time.Time
	start := time.Now()

	r.Do(context.Background(), func(ctx context.Context) error {
		timestamps = append(timestamps, time.Now())
		// Use retryable error
		return NewHTTPError(http.StatusInternalServerError, "server error")
	})

	require.Len(t, timestamps, 5)

	// Check that delays increase exponentially (approximately)
	for i := 1; i < len(timestamps); i++ {
		elapsed := timestamps[i].Sub(timestamps[i-1])
		// Allow some tolerance for timing
		expectedMinDelay := time.Duration(float64(10*time.Millisecond) * float64(int(1)<<(i-1)))
		if expectedMinDelay > 100*time.Millisecond {
			expectedMinDelay = 100 * time.Millisecond
		}
		// Check that delay is at least half the expected (to account for timing jitter)
		assert.GreaterOrEqual(t, elapsed, expectedMinDelay/2)
	}

	totalElapsed := time.Since(start)
	// Should take at least: 10 + 20 + 40 + 80 = 150ms (capped at 100ms for last delay)
	// So at least: 10 + 20 + 40 + 100 = 170ms, but with tolerance
	assert.GreaterOrEqual(t, totalElapsed, 50*time.Millisecond)
}

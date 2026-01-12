package integration

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/bargom/codeai/pkg/metrics"
)

// ErrTimeout is returned when an operation times out.
var ErrTimeout = errors.New("operation timed out")

// TimeoutConfig configures timeout behavior.
type TimeoutConfig struct {
	// Default is the default timeout for operations.
	// Default: 30s
	Default time.Duration

	// Connect is the timeout for establishing connections.
	// Default: 10s
	Connect time.Duration

	// Read is the timeout for reading response.
	// Default: 30s
	Read time.Duration

	// Write is the timeout for writing request.
	// Default: 30s
	Write time.Duration

	// OnTimeout is called when a timeout occurs.
	OnTimeout func(operation string, timeout time.Duration)
}

// DefaultTimeoutConfig returns the default timeout configuration.
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Default: 30 * time.Second,
		Connect: 10 * time.Second,
		Read:    30 * time.Second,
		Write:   30 * time.Second,
	}
}

// TimeoutManager manages timeouts for operations.
type TimeoutManager struct {
	config      TimeoutConfig
	logger      *slog.Logger
	serviceName string
	endpoint    string
}

// NewTimeoutManager creates a new timeout manager.
func NewTimeoutManager(config TimeoutConfig) *TimeoutManager {
	if config.Default <= 0 {
		config.Default = 30 * time.Second
	}
	if config.Connect <= 0 {
		config.Connect = 10 * time.Second
	}
	if config.Read <= 0 {
		config.Read = 30 * time.Second
	}
	if config.Write <= 0 {
		config.Write = 30 * time.Second
	}

	return &TimeoutManager{
		config: config,
		logger: slog.Default().With("component", "timeout_manager"),
	}
}

// WithService returns a new timeout manager configured for a specific service.
func (tm *TimeoutManager) WithService(serviceName, endpoint string) *TimeoutManager {
	return &TimeoutManager{
		config:      tm.config,
		logger:      tm.logger.With("service", serviceName, "endpoint", endpoint),
		serviceName: serviceName,
		endpoint:    endpoint,
	}
}

// WithTimeout creates a context with the specified timeout.
// If the parent context has a tighter deadline, that deadline is preserved.
func (tm *TimeoutManager) WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = tm.config.Default
	}

	// Check if parent context already has a tighter deadline
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < timeout {
			// Parent has tighter deadline, just add cancel
			return context.WithCancel(ctx)
		}
	}

	return context.WithTimeout(ctx, timeout)
}

// WithDefaultTimeout creates a context with the default timeout.
func (tm *TimeoutManager) WithDefaultTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return tm.WithTimeout(ctx, tm.config.Default)
}

// Execute runs a function with the specified timeout.
func (tm *TimeoutManager) Execute(ctx context.Context, timeout time.Duration, operation string, fn func(context.Context) error) error {
	timeoutCtx, cancel := tm.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)
	start := time.Now()

	go func() {
		done <- fn(timeoutCtx)
	}()

	select {
	case err := <-done:
		return err

	case <-timeoutCtx.Done():
		elapsed := time.Since(start)

		if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
			tm.logger.Warn("operation timed out",
				"operation", operation,
				"timeout", timeout,
				"elapsed", elapsed,
			)

			if tm.config.OnTimeout != nil {
				tm.config.OnTimeout(operation, timeout)
			}

			// Record timeout metric
			if tm.serviceName != "" {
				if reg := metrics.Global(); reg != nil {
					reg.Integration().RecordError(tm.serviceName, tm.endpoint, "timeout")
				}
			}

			return ErrTimeout
		}

		return timeoutCtx.Err()
	}
}

// ExecuteWithResult runs a function that returns a result with the specified timeout.
func ExecuteWithResult[T any](ctx context.Context, tm *TimeoutManager, timeout time.Duration, operation string, fn func(context.Context) (T, error)) (T, error) {
	var result T
	timeoutCtx, cancel := tm.WithTimeout(ctx, timeout)
	defer cancel()

	type resultWithError struct {
		result T
		err    error
	}

	done := make(chan resultWithError, 1)
	start := time.Now()

	go func() {
		res, err := fn(timeoutCtx)
		done <- resultWithError{result: res, err: err}
	}()

	select {
	case res := <-done:
		return res.result, res.err

	case <-timeoutCtx.Done():
		elapsed := time.Since(start)

		if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
			tm.logger.Warn("operation timed out",
				"operation", operation,
				"timeout", timeout,
				"elapsed", elapsed,
			)

			if tm.config.OnTimeout != nil {
				tm.config.OnTimeout(operation, timeout)
			}

			// Record timeout metric
			if tm.serviceName != "" {
				if reg := metrics.Global(); reg != nil {
					reg.Integration().RecordError(tm.serviceName, tm.endpoint, "timeout")
				}
			}

			return result, ErrTimeout
		}

		return result, timeoutCtx.Err()
	}
}

// Config returns the timeout configuration.
func (tm *TimeoutManager) Config() TimeoutConfig {
	return tm.config
}

// WithTimeout is a convenience function that runs a function with a timeout.
func WithTimeout(ctx context.Context, timeout time.Duration, fn func(context.Context) error) error {
	return NewTimeoutManager(TimeoutConfig{Default: timeout}).Execute(ctx, timeout, "operation", fn)
}

// TimeoutContext wraps a context with timeout tracking.
type TimeoutContext struct {
	ctx       context.Context
	cancel    context.CancelFunc
	timeout   time.Duration
	startTime time.Time
}

// NewTimeoutContext creates a new timeout context.
func NewTimeoutContext(ctx context.Context, timeout time.Duration) *TimeoutContext {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	return &TimeoutContext{
		ctx:       timeoutCtx,
		cancel:    cancel,
		timeout:   timeout,
		startTime: time.Now(),
	}
}

// Context returns the underlying context.
func (tc *TimeoutContext) Context() context.Context {
	return tc.ctx
}

// Cancel cancels the context.
func (tc *TimeoutContext) Cancel() {
	tc.cancel()
}

// Remaining returns the remaining time before timeout.
func (tc *TimeoutContext) Remaining() time.Duration {
	if deadline, ok := tc.ctx.Deadline(); ok {
		return time.Until(deadline)
	}
	return tc.timeout - time.Since(tc.startTime)
}

// Elapsed returns the time elapsed since the context was created.
func (tc *TimeoutContext) Elapsed() time.Duration {
	return time.Since(tc.startTime)
}

// IsExpired returns true if the context has expired.
func (tc *TimeoutContext) IsExpired() bool {
	select {
	case <-tc.ctx.Done():
		return true
	default:
		return false
	}
}

// Extend extends the timeout by the specified duration.
// Note: This creates a new context with a new deadline.
func (tc *TimeoutContext) Extend(additional time.Duration) *TimeoutContext {
	tc.cancel() // Cancel the old context

	newTimeout := tc.Remaining() + additional
	if newTimeout < 0 {
		newTimeout = additional
	}

	return NewTimeoutContext(context.Background(), newTimeout)
}

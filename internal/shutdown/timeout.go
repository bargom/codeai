package shutdown

import (
	"context"
	"fmt"
	"time"
)

// TimeoutError is returned when a shutdown operation times out.
type TimeoutError struct {
	Operation string
	Timeout   time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("shutdown operation %q timed out after %v", e.Operation, e.Timeout)
}

// IsTimeout returns true if the error is a TimeoutError.
func IsTimeout(err error) bool {
	_, ok := err.(*TimeoutError)
	return ok
}

// WithTimeout executes a function with a timeout.
// Returns TimeoutError if the function doesn't complete within the timeout.
func WithTimeout(ctx context.Context, timeout time.Duration, name string, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		done <- fn(ctx)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return &TimeoutError{
				Operation: name,
				Timeout:   timeout,
			}
		}
		return ctx.Err()
	}
}

// WithTimeoutAndPanicRecovery executes a function with timeout and panic recovery.
func WithTimeoutAndPanicRecovery(ctx context.Context, timeout time.Duration, name string, fn func(ctx context.Context) error) (err error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- &PanicError{
					Operation: name,
					Value:     r,
				}
			}
		}()
		done <- fn(ctx)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return &TimeoutError{
				Operation: name,
				Timeout:   timeout,
			}
		}
		return ctx.Err()
	}
}

// PanicError is returned when a shutdown hook panics.
type PanicError struct {
	Operation string
	Value     interface{}
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("shutdown operation %q panicked: %v", e.Operation, e.Value)
}

// IsPanic returns true if the error is a PanicError.
func IsPanic(err error) bool {
	_, ok := err.(*PanicError)
	return ok
}

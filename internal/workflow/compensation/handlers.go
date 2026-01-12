package compensation

import (
	"context"
	"encoding/json"
	"fmt"
)

// ResourceCleanup provides common compensation handlers for resource cleanup.
type ResourceCleanup struct{}

// NewResourceCleanup creates a new ResourceCleanup instance.
func NewResourceCleanup() *ResourceCleanup {
	return &ResourceCleanup{}
}

// DeleteResource creates a compensation handler to delete a resource.
func (r *ResourceCleanup) DeleteResource(resourceType, resourceID string, deleteFn func(ctx context.Context, id string) error) CompensationFunc {
	return func(ctx context.Context) error {
		return deleteFn(ctx, resourceID)
	}
}

// RevertState creates a compensation handler to revert state.
func (r *ResourceCleanup) RevertState(resourceType, resourceID string, previousState json.RawMessage, revertFn func(ctx context.Context, id string, state json.RawMessage) error) CompensationFunc {
	return func(ctx context.Context) error {
		return revertFn(ctx, resourceID, previousState)
	}
}

// CancelOperation creates a compensation handler to cancel an operation.
func (r *ResourceCleanup) CancelOperation(operationType, operationID string, cancelFn func(ctx context.Context, id string) error) CompensationFunc {
	return func(ctx context.Context) error {
		return cancelFn(ctx, operationID)
	}
}

// CompensationBuilder provides a fluent interface for building compensation handlers.
type CompensationBuilder struct {
	name string
	fn   CompensationFunc
}

// NewCompensation creates a new compensation builder.
func NewCompensation(name string) *CompensationBuilder {
	return &CompensationBuilder{name: name}
}

// WithAction sets the compensation action.
func (b *CompensationBuilder) WithAction(fn CompensationFunc) *CompensationBuilder {
	b.fn = fn
	return b
}

// WithRetry wraps the compensation with retry logic.
func (b *CompensationBuilder) WithRetry(maxAttempts int) *CompensationBuilder {
	originalFn := b.fn
	b.fn = func(ctx context.Context) error {
		var lastErr error
		for i := 0; i < maxAttempts; i++ {
			if err := originalFn(ctx); err != nil {
				lastErr = err
				continue
			}
			return nil
		}
		return fmt.Errorf("compensation failed after %d attempts: %w", maxAttempts, lastErr)
	}
	return b
}

// WithTimeout wraps the compensation with a timeout.
func (b *CompensationBuilder) WithTimeout(ctx context.Context) *CompensationBuilder {
	originalFn := b.fn
	b.fn = func(execCtx context.Context) error {
		done := make(chan error, 1)
		go func() {
			done <- originalFn(execCtx)
		}()

		select {
		case err := <-done:
			return err
		case <-execCtx.Done():
			return execCtx.Err()
		}
	}
	return b
}

// Build returns the compensation function.
func (b *CompensationBuilder) Build() (string, CompensationFunc) {
	return b.name, b.fn
}

// TransactionalSaga extends SagaManager with transaction-like semantics.
type TransactionalSaga struct {
	*SagaManager
	committed bool
}

// NewTransactionalSaga creates a new transactional saga.
func NewTransactionalSaga() *TransactionalSaga {
	return &TransactionalSaga{
		SagaManager: NewSagaManager(),
	}
}

// Commit marks the saga as committed, preventing automatic compensation.
func (s *TransactionalSaga) Commit() {
	s.committed = true
}

// Rollback executes compensations if not committed.
func (s *TransactionalSaga) Rollback(ctx context.Context) error {
	if s.committed {
		return nil
	}
	return s.Compensate(ctx)
}

// IsCommitted returns true if the saga has been committed.
func (s *TransactionalSaga) IsCommitted() bool {
	return s.committed
}

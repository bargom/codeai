// Package compensation provides saga pattern support for workflow compensation.
package compensation

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CompensationFunc is a function that compensates for a previous action.
type CompensationFunc func(ctx context.Context) error

// CompensationRecord tracks a compensation action and its state.
type CompensationRecord struct {
	Name        string    `json:"name"`
	Executed    bool      `json:"executed"`
	ExecutedAt  time.Time `json:"executedAt,omitempty"`
	Error       string    `json:"error,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
}

// SagaManager manages compensation transactions for workflows.
type SagaManager struct {
	mu            sync.Mutex
	compensations []namedCompensation
	records       []CompensationRecord
}

type namedCompensation struct {
	name string
	fn   CompensationFunc
}

// NewSagaManager creates a new SagaManager instance.
func NewSagaManager() *SagaManager {
	return &SagaManager{
		compensations: make([]namedCompensation, 0),
		records:       make([]CompensationRecord, 0),
	}
}

// AddCompensation registers a compensation function to be executed on rollback.
// Compensations are executed in reverse order (LIFO).
func (s *SagaManager) AddCompensation(name string, fn CompensationFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.compensations = append(s.compensations, namedCompensation{name: name, fn: fn})
}

// Compensate executes all registered compensations in reverse order.
// It continues executing even if individual compensations fail.
func (s *SagaManager) Compensate(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errors []error

	// Execute compensations in reverse order (LIFO)
	for i := len(s.compensations) - 1; i >= 0; i-- {
		comp := s.compensations[i]
		record := CompensationRecord{
			Name: comp.name,
		}

		start := time.Now()
		if err := comp.fn(ctx); err != nil {
			record.Error = err.Error()
			errors = append(errors, fmt.Errorf("compensation %s failed: %w", comp.name, err))
		}
		record.Executed = true
		record.ExecutedAt = time.Now()
		record.Duration = time.Since(start)

		s.records = append(s.records, record)
	}

	if len(errors) > 0 {
		return &CompensationError{Errors: errors}
	}

	return nil
}

// Clear removes all registered compensations.
func (s *SagaManager) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.compensations = make([]namedCompensation, 0)
}

// Records returns the compensation execution records.
func (s *SagaManager) Records() []CompensationRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]CompensationRecord, len(s.records))
	copy(result, s.records)
	return result
}

// Count returns the number of registered compensations.
func (s *SagaManager) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.compensations)
}

// CompensationError aggregates multiple compensation failures.
type CompensationError struct {
	Errors []error
}

// Error implements the error interface.
func (e *CompensationError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("%d compensation failures", len(e.Errors))
}

// Unwrap returns the first error for errors.Is/As compatibility.
func (e *CompensationError) Unwrap() error {
	if len(e.Errors) > 0 {
		return e.Errors[0]
	}
	return nil
}

// SagaStep represents a single step in a saga with its action and compensation.
type SagaStep struct {
	Name         string
	Action       func(ctx context.Context) error
	Compensation CompensationFunc
}

// ExecuteSaga executes a series of steps with automatic compensation on failure.
func ExecuteSaga(ctx context.Context, steps []SagaStep) ([]CompensationRecord, error) {
	saga := NewSagaManager()

	for _, step := range steps {
		// Execute the action
		if err := step.Action(ctx); err != nil {
			// Action failed, compensate and return
			compensateErr := saga.Compensate(ctx)
			if compensateErr != nil {
				return saga.Records(), fmt.Errorf("step %s failed: %w (compensation also failed: %v)", step.Name, err, compensateErr)
			}
			return saga.Records(), fmt.Errorf("step %s failed: %w (compensated)", step.Name, err)
		}

		// Action succeeded, register compensation for rollback
		if step.Compensation != nil {
			saga.AddCompensation(step.Name, step.Compensation)
		}
	}

	return nil, nil
}

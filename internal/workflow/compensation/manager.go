// Package compensation provides enhanced saga pattern support for workflow compensation.
package compensation

import (
	"fmt"
	"sync"
	"time"

	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// CompensationFunc is a function executed via Temporal workflow context.
type WorkflowCompensationFunc func(ctx workflow.Context, input interface{}) error

// CompensationStep represents a single compensation operation.
type CompensationStep struct {
	ActivityName   string
	CompensateFn   WorkflowCompensationFunc
	Input          interface{}
	Timeout        time.Duration
	RetryPolicy    *temporal.RetryPolicy
	Idempotent     bool
	AllowSkip      bool // If true, failure doesn't stop other compensations
}

// CompensationManager manages workflow-level compensation transactions.
type CompensationManager struct {
	mu            sync.Mutex
	compensations []CompensationStep
	executed      []string
	records       []CompensationExecutionRecord
	logger        log.Logger
}

// CompensationExecutionRecord tracks compensation execution details.
type CompensationExecutionRecord struct {
	ActivityName  string        `json:"activityName"`
	Status        CompensationStatus `json:"status"`
	ExecutedAt    time.Time     `json:"executedAt,omitempty"`
	Duration      time.Duration `json:"duration,omitempty"`
	Error         string        `json:"error,omitempty"`
	Retries       int           `json:"retries,omitempty"`
}

// CompensationStatus represents the status of a compensation operation.
type CompensationStatus string

const (
	CompensationPending   CompensationStatus = "pending"
	CompensationRunning   CompensationStatus = "running"
	CompensationCompleted CompensationStatus = "completed"
	CompensationFailed    CompensationStatus = "failed"
	CompensationSkipped   CompensationStatus = "skipped"
)

// NewCompensationManager creates a new CompensationManager instance.
func NewCompensationManager(ctx workflow.Context) *CompensationManager {
	return &CompensationManager{
		compensations: make([]CompensationStep, 0),
		executed:      make([]string, 0),
		records:       make([]CompensationExecutionRecord, 0),
		logger:        workflow.GetLogger(ctx),
	}
}

// RegisterCompensation adds a compensation step for an activity.
func (cm *CompensationManager) RegisterCompensation(step CompensationStep) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if step.Timeout == 0 {
		step.Timeout = 30 * time.Second
	}

	cm.compensations = append(cm.compensations, step)
}

// RegisterSimple adds a simple compensation with default settings.
func (cm *CompensationManager) RegisterSimple(activityName string, fn WorkflowCompensationFunc, input interface{}) {
	cm.RegisterCompensation(CompensationStep{
		ActivityName: activityName,
		CompensateFn: fn,
		Input:        input,
		Timeout:      30 * time.Second,
		Idempotent:   true,
	})
}

// RecordExecution marks an activity as successfully executed.
func (cm *CompensationManager) RecordExecution(activityName string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.executed = append(cm.executed, activityName)
}

// IsExecuted checks if an activity was executed.
func (cm *CompensationManager) IsExecuted(activityName string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, name := range cm.executed {
		if name == activityName {
			return true
		}
	}
	return false
}

// Compensate runs all registered compensations in reverse order.
func (cm *CompensationManager) Compensate(ctx workflow.Context) error {
	cm.mu.Lock()
	compensations := make([]CompensationStep, len(cm.compensations))
	copy(compensations, cm.compensations)
	cm.mu.Unlock()

	var errors []error

	// Execute compensations in reverse order (LIFO)
	for i := len(compensations) - 1; i >= 0; i-- {
		comp := compensations[i]

		// Only compensate activities that were executed
		if !cm.IsExecuted(comp.ActivityName) {
			cm.logger.Info("Skipping compensation for non-executed activity",
				"activity", comp.ActivityName,
			)
			continue
		}

		record := CompensationExecutionRecord{
			ActivityName: comp.ActivityName,
			Status:       CompensationRunning,
		}

		cm.logger.Info("Executing compensation",
			"activity", comp.ActivityName,
		)

		start := workflow.Now(ctx)
		err := cm.executeCompensation(ctx, comp)
		record.ExecutedAt = workflow.Now(ctx)
		record.Duration = record.ExecutedAt.Sub(start)

		if err != nil {
			record.Status = CompensationFailed
			record.Error = err.Error()

			cm.logger.Error("Compensation failed",
				"activity", comp.ActivityName,
				"error", err,
			)

			if !comp.AllowSkip {
				errors = append(errors, fmt.Errorf("compensation %s failed: %w", comp.ActivityName, err))
			}
		} else {
			record.Status = CompensationCompleted
			cm.logger.Info("Compensation completed",
				"activity", comp.ActivityName,
				"duration", record.Duration,
			)
		}

		cm.mu.Lock()
		cm.records = append(cm.records, record)
		cm.mu.Unlock()
	}

	if len(errors) > 0 {
		return &CompensationError{Errors: errors}
	}

	return nil
}

// CompensatePartial runs compensation for specific activities only.
func (cm *CompensationManager) CompensatePartial(ctx workflow.Context, activityNames []string) error {
	nameSet := make(map[string]bool)
	for _, name := range activityNames {
		nameSet[name] = true
	}

	cm.mu.Lock()
	compensations := make([]CompensationStep, 0)
	for _, comp := range cm.compensations {
		if nameSet[comp.ActivityName] {
			compensations = append(compensations, comp)
		}
	}
	cm.mu.Unlock()

	var errors []error

	// Execute in reverse order
	for i := len(compensations) - 1; i >= 0; i-- {
		comp := compensations[i]

		if !cm.IsExecuted(comp.ActivityName) {
			continue
		}

		record := CompensationExecutionRecord{
			ActivityName: comp.ActivityName,
			Status:       CompensationRunning,
		}

		start := workflow.Now(ctx)
		err := cm.executeCompensation(ctx, comp)
		record.ExecutedAt = workflow.Now(ctx)
		record.Duration = record.ExecutedAt.Sub(start)

		if err != nil {
			record.Status = CompensationFailed
			record.Error = err.Error()
			if !comp.AllowSkip {
				errors = append(errors, fmt.Errorf("compensation %s failed: %w", comp.ActivityName, err))
			}
		} else {
			record.Status = CompensationCompleted
		}

		cm.mu.Lock()
		cm.records = append(cm.records, record)
		cm.mu.Unlock()
	}

	if len(errors) > 0 {
		return &CompensationError{Errors: errors}
	}

	return nil
}

// executeCompensation runs a single compensation step with retry logic.
func (cm *CompensationManager) executeCompensation(ctx workflow.Context, step CompensationStep) error {
	// Create activity options for compensation
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: step.Timeout,
	}

	if step.RetryPolicy != nil {
		activityOptions.RetryPolicy = step.RetryPolicy
	} else {
		// Default retry policy for compensations
		activityOptions.RetryPolicy = &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		}
	}

	compensationCtx := workflow.WithActivityOptions(ctx, activityOptions)
	return step.CompensateFn(compensationCtx, step.Input)
}

// Records returns the compensation execution records.
func (cm *CompensationManager) Records() []CompensationExecutionRecord {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	result := make([]CompensationExecutionRecord, len(cm.records))
	copy(result, cm.records)
	return result
}

// ExecutedCount returns the number of executed activities.
func (cm *CompensationManager) ExecutedCount() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return len(cm.executed)
}

// CompensationCount returns the number of registered compensations.
func (cm *CompensationManager) CompensationCount() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return len(cm.compensations)
}

// Clear removes all registered compensations and records.
func (cm *CompensationManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.compensations = make([]CompensationStep, 0)
	cm.executed = make([]string, 0)
	cm.records = make([]CompensationExecutionRecord, 0)
}

// GetExecutedActivities returns a copy of the executed activity names.
func (cm *CompensationManager) GetExecutedActivities() []string {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	result := make([]string, len(cm.executed))
	copy(result, cm.executed)
	return result
}

// Package testing provides test utilities for compensation workflows.
package testing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.temporal.io/sdk/workflow"
)

// FailureInjector allows controlled injection of failures for testing compensation.
type FailureInjector struct {
	mu           sync.RWMutex
	failAtStep   map[string]bool
	failCount    map[string]int    // Fail only N times then succeed
	delayAtStep  map[string]time.Duration
	errorType    map[string]string // Custom error types
	failureOrder []string         // Order in which failures occurred
}

// NewFailureInjector creates a new FailureInjector instance.
func NewFailureInjector() *FailureInjector {
	return &FailureInjector{
		failAtStep:   make(map[string]bool),
		failCount:    make(map[string]int),
		delayAtStep:  make(map[string]time.Duration),
		errorType:    make(map[string]string),
		failureOrder: make([]string, 0),
	}
}

// ConfigureFailure sets a step to always fail.
func (fi *FailureInjector) ConfigureFailure(stepName string) {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.failAtStep[stepName] = true
}

// ConfigureFailureWithCount sets a step to fail N times then succeed.
func (fi *FailureInjector) ConfigureFailureWithCount(stepName string, count int) {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.failCount[stepName] = count
}

// ConfigureDelay sets a delay for a step.
func (fi *FailureInjector) ConfigureDelay(stepName string, delay time.Duration) {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.delayAtStep[stepName] = delay
}

// ConfigureErrorType sets a custom error type for a step.
func (fi *FailureInjector) ConfigureErrorType(stepName string, errorType string) {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.errorType[stepName] = errorType
}

// ShouldFail returns true if the step should fail.
func (fi *FailureInjector) ShouldFail(stepName string) bool {
	fi.mu.Lock()
	defer fi.mu.Unlock()

	// Check if step has fail count
	if count, exists := fi.failCount[stepName]; exists {
		if count > 0 {
			fi.failCount[stepName] = count - 1
			fi.failureOrder = append(fi.failureOrder, stepName)
			return true
		}
		return false
	}

	// Check if step should always fail
	if fi.failAtStep[stepName] {
		fi.failureOrder = append(fi.failureOrder, stepName)
		return true
	}

	return false
}

// GetDelay returns the configured delay for a step.
func (fi *FailureInjector) GetDelay(stepName string) time.Duration {
	fi.mu.RLock()
	defer fi.mu.RUnlock()
	return fi.delayAtStep[stepName]
}

// GetErrorType returns the configured error type for a step.
func (fi *FailureInjector) GetErrorType(stepName string) string {
	fi.mu.RLock()
	defer fi.mu.RUnlock()
	if errType, exists := fi.errorType[stepName]; exists {
		return errType
	}
	return "InjectedFailure"
}

// GetFailureOrder returns the order in which failures occurred.
func (fi *FailureInjector) GetFailureOrder() []string {
	fi.mu.RLock()
	defer fi.mu.RUnlock()
	result := make([]string, len(fi.failureOrder))
	copy(result, fi.failureOrder)
	return result
}

// Reset clears all configured failures.
func (fi *FailureInjector) Reset() {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.failAtStep = make(map[string]bool)
	fi.failCount = make(map[string]int)
	fi.delayAtStep = make(map[string]time.Duration)
	fi.errorType = make(map[string]string)
	fi.failureOrder = make([]string, 0)
}

// CompensationTestScenario defines a test scenario for compensation.
type CompensationTestScenario struct {
	Name               string
	FailAtStep         string
	ExpectCompensated  []string
	ExpectNotCompensated []string
	ExpectedError      string
}

// DefaultTestScenarios returns a set of common test scenarios.
func DefaultTestScenarios() []CompensationTestScenario {
	return []CompensationTestScenario{
		{
			Name:               "All steps succeed",
			FailAtStep:         "",
			ExpectCompensated:  []string{},
			ExpectedError:      "",
		},
		{
			Name:               "Fail at first step",
			FailAtStep:         "step-1",
			ExpectCompensated:  []string{},
			ExpectedError:      "step-1 failed",
		},
		{
			Name:               "Fail at second step",
			FailAtStep:         "step-2",
			ExpectCompensated:  []string{"step-1"},
			ExpectedError:      "step-2 failed",
		},
		{
			Name:               "Fail at third step",
			FailAtStep:         "step-3",
			ExpectCompensated:  []string{"step-2", "step-1"},
			ExpectedError:      "step-3 failed",
		},
		{
			Name:               "Fail at last step",
			FailAtStep:         "step-4",
			ExpectCompensated:  []string{"step-3", "step-2", "step-1"},
			ExpectedError:      "step-4 failed",
		},
	}
}

// CompensationRecorder records compensation execution for verification.
type CompensationRecorder struct {
	mu           sync.Mutex
	compensated  []string
	timestamps   map[string]time.Time
	errors       map[string]error
	callCount    map[string]int
}

// NewCompensationRecorder creates a new CompensationRecorder.
func NewCompensationRecorder() *CompensationRecorder {
	return &CompensationRecorder{
		compensated: make([]string, 0),
		timestamps:  make(map[string]time.Time),
		errors:      make(map[string]error),
		callCount:   make(map[string]int),
	}
}

// Record records a compensation execution.
func (cr *CompensationRecorder) Record(stepName string, err error) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.compensated = append(cr.compensated, stepName)
	cr.timestamps[stepName] = time.Now()
	cr.callCount[stepName]++
	if err != nil {
		cr.errors[stepName] = err
	}
}

// GetCompensated returns the list of compensated steps in order.
func (cr *CompensationRecorder) GetCompensated() []string {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	result := make([]string, len(cr.compensated))
	copy(result, cr.compensated)
	return result
}

// WasCompensated returns true if the step was compensated.
func (cr *CompensationRecorder) WasCompensated(stepName string) bool {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	for _, name := range cr.compensated {
		if name == stepName {
			return true
		}
	}
	return false
}

// GetCallCount returns how many times a step was compensated.
func (cr *CompensationRecorder) GetCallCount(stepName string) int {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.callCount[stepName]
}

// GetError returns the error for a compensated step.
func (cr *CompensationRecorder) GetError(stepName string) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.errors[stepName]
}

// Reset clears all recorded compensations.
func (cr *CompensationRecorder) Reset() {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.compensated = make([]string, 0)
	cr.timestamps = make(map[string]time.Time)
	cr.errors = make(map[string]error)
	cr.callCount = make(map[string]int)
}

// VerifyCompensationOrder verifies compensations occurred in expected order.
func (cr *CompensationRecorder) VerifyCompensationOrder(expected []string) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if len(cr.compensated) != len(expected) {
		return fmt.Errorf("compensation count mismatch: got %d, expected %d", len(cr.compensated), len(expected))
	}

	for i, name := range expected {
		if i >= len(cr.compensated) || cr.compensated[i] != name {
			return fmt.Errorf("compensation order mismatch at position %d: got %s, expected %s", i, cr.compensated[i], name)
		}
	}

	return nil
}

// MockCompensationFunc creates a mock compensation function that records calls.
func MockCompensationFunc(recorder *CompensationRecorder, stepName string) func(ctx workflow.Context, input interface{}) error {
	return func(ctx workflow.Context, input interface{}) error {
		recorder.Record(stepName, nil)
		return nil
	}
}

// MockCompensationFuncWithError creates a mock that records and returns an error.
func MockCompensationFuncWithError(recorder *CompensationRecorder, stepName string, err error) func(ctx workflow.Context, input interface{}) error {
	return func(ctx workflow.Context, input interface{}) error {
		recorder.Record(stepName, err)
		return err
	}
}

// InjectedError represents a test-injected error.
type InjectedError struct {
	StepName string
	ErrorType string
	Message  string
}

// Error implements the error interface.
func (e *InjectedError) Error() string {
	return fmt.Sprintf("%s: %s at step %s", e.ErrorType, e.Message, e.StepName)
}

// NewInjectedError creates a new injected error.
func NewInjectedError(stepName, errorType, message string) *InjectedError {
	return &InjectedError{
		StepName:  stepName,
		ErrorType: errorType,
		Message:   message,
	}
}

// TestActivity is a mock activity for testing.
type TestActivity struct {
	injector *FailureInjector
	recorder *CompensationRecorder
}

// NewTestActivity creates a new TestActivity.
func NewTestActivity(injector *FailureInjector, recorder *CompensationRecorder) *TestActivity {
	return &TestActivity{
		injector: injector,
		recorder: recorder,
	}
}

// Execute executes the test activity with failure injection.
func (ta *TestActivity) Execute(ctx context.Context, stepName string) (string, error) {
	// Apply configured delay
	if delay := ta.injector.GetDelay(stepName); delay > 0 {
		time.Sleep(delay)
	}

	// Check if should fail
	if ta.injector.ShouldFail(stepName) {
		errorType := ta.injector.GetErrorType(stepName)
		return "", NewInjectedError(stepName, errorType, "injected failure")
	}

	return fmt.Sprintf("success at %s", stepName), nil
}

// Compensate executes compensation with recording.
func (ta *TestActivity) Compensate(ctx context.Context, stepName string) error {
	// Apply configured delay for compensation
	compensateKey := fmt.Sprintf("compensate-%s", stepName)
	if delay := ta.injector.GetDelay(compensateKey); delay > 0 {
		time.Sleep(delay)
	}

	// Check if compensation should fail
	if ta.injector.ShouldFail(compensateKey) {
		errorType := ta.injector.GetErrorType(compensateKey)
		err := NewInjectedError(compensateKey, errorType, "compensation failure")
		ta.recorder.Record(stepName, err)
		return err
	}

	ta.recorder.Record(stepName, nil)
	return nil
}

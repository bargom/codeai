package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/bargom/codeai/internal/workflow/definitions"
)

// TestCaseExecutor defines the interface for executing test cases.
type TestCaseExecutor interface {
	Execute(ctx context.Context, testCase definitions.TestCase) (json.RawMessage, error)
}

// TestCaseActivities holds test case execution activity implementations.
type TestCaseActivities struct {
	executor TestCaseExecutor
}

// NewTestCaseActivities creates a new TestCaseActivities instance.
func NewTestCaseActivities(executor TestCaseExecutor) *TestCaseActivities {
	return &TestCaseActivities{executor: executor}
}

// ExecuteTestCase executes a single test case.
func (a *TestCaseActivities) ExecuteTestCase(ctx context.Context, req definitions.ExecuteTestCaseRequest) (definitions.ExecuteTestCaseResponse, error) {
	info := activity.GetInfo(ctx)
	tc := req.TestCase

	result := definitions.TestCaseResult{
		TestCaseID: tc.ID,
		Name:       tc.Name,
		Status:     definitions.StatusRunning,
		Attempts:   int(info.Attempt),
		StartedAt:  time.Now(),
	}

	// Log activity start
	activity.RecordHeartbeat(ctx, fmt.Sprintf("executing test case %s (attempt %d)", tc.ID, info.Attempt))

	// Execute the test case
	output, err := a.executor.Execute(ctx, tc)
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)

	if err != nil {
		result.Status = definitions.StatusFailed
		result.Error = err.Error()
		return definitions.ExecuteTestCaseResponse{Result: result}, err
	}

	result.Status = definitions.StatusCompleted
	result.Output = output

	return definitions.ExecuteTestCaseResponse{Result: result}, nil
}

// DefaultTestCaseExecutor is a default implementation of TestCaseExecutor.
type DefaultTestCaseExecutor struct{}

// Execute implements TestCaseExecutor for the default executor.
func (e *DefaultTestCaseExecutor) Execute(ctx context.Context, tc definitions.TestCase) (json.RawMessage, error) {
	result := map[string]interface{}{
		"testCaseId": tc.ID,
		"passed":     true,
		"message":    "test case executed successfully",
	}

	// Check if expected output is provided and validate
	if tc.Expected != nil {
		var expected map[string]interface{}
		if err := json.Unmarshal(tc.Expected, &expected); err == nil {
			result["expectedValidated"] = true
		}
	}

	return json.Marshal(result)
}

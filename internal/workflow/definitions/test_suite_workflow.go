package definitions

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// TestSuiteWorkflow executes AI test suites with retries.
func TestSuiteWorkflow(ctx workflow.Context, input TestSuiteInput) (TestSuiteOutput, error) {
	output := TestSuiteOutput{
		SuiteID:    input.SuiteID,
		Name:       input.Name,
		Status:     StatusRunning,
		Results:    make([]TestCaseResult, 0, len(input.TestCases)),
		TotalTests: len(input.TestCases),
		StartedAt:  workflow.Now(ctx),
	}

	// Set workflow timeout
	timeout := input.Timeout
	if timeout == 0 {
		timeout = 60 * time.Minute
	}

	// Configure activity options
	retryPolicy := input.RetryPolicy
	if retryPolicy.MaxAttempts == 0 {
		retryPolicy = DefaultRetryConfig()
	}

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: timeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    retryPolicy.InitialInterval,
			BackoffCoefficient: retryPolicy.BackoffCoefficient,
			MaximumInterval:    retryPolicy.MaximumInterval,
			MaximumAttempts:    int32(retryPolicy.MaxAttempts),
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Validate test suite input
	var validationResult ValidationResult
	if err := workflow.ExecuteActivity(ctx, ValidateTestSuiteActivity, ValidateTestSuiteRequest{
		Input: input,
	}).Get(ctx, &validationResult); err != nil {
		output.Status = StatusFailed
		output.Error = fmt.Sprintf("validation failed: %v", err)
		output.CompletedAt = workflow.Now(ctx)
		output.TotalDuration = output.CompletedAt.Sub(output.StartedAt)
		return output, err
	}

	if !validationResult.Valid {
		output.Status = StatusFailed
		output.Error = fmt.Sprintf("invalid test suite: %s", validationResult.Error)
		output.CompletedAt = workflow.Now(ctx)
		output.TotalDuration = output.CompletedAt.Sub(output.StartedAt)
		return output, fmt.Errorf("invalid test suite: %s", validationResult.Error)
	}

	// Execute test cases
	if input.Parallel {
		results, err := executeTestCasesParallel(ctx, input.TestCases)
		output.Results = results
		if err != nil && input.StopOnFailure {
			output.Status = StatusFailed
			output.Error = err.Error()
		}
	} else {
		results, err := executeTestCasesSequential(ctx, input.TestCases, input.StopOnFailure)
		output.Results = results
		if err != nil {
			output.Status = StatusFailed
			output.Error = err.Error()
		}
	}

	// Calculate summary
	for _, result := range output.Results {
		switch result.Status {
		case StatusCompleted:
			output.PassedTests++
		case StatusFailed:
			output.FailedTests++
		case StatusCanceled:
			output.SkippedTests++
		}
	}

	// Determine final status
	if output.Status != StatusFailed {
		if output.FailedTests > 0 {
			output.Status = StatusFailed
		} else {
			output.Status = StatusCompleted
		}
	}

	// Store results
	storageCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	})

	var storageResult StorageResult
	_ = workflow.ExecuteActivity(storageCtx, StoreTestSuiteResultActivity, StoreTestSuiteResultRequest{
		SuiteID: input.SuiteID,
		Output:  output,
	}).Get(ctx, &storageResult)

	output.CompletedAt = workflow.Now(ctx)
	output.TotalDuration = output.CompletedAt.Sub(output.StartedAt)

	if output.Status == StatusFailed {
		return output, fmt.Errorf("test suite failed: %d/%d tests failed", output.FailedTests, output.TotalTests)
	}

	return output, nil
}

func executeTestCasesSequential(ctx workflow.Context, testCases []TestCase, stopOnFailure bool) ([]TestCaseResult, error) {
	results := make([]TestCaseResult, 0, len(testCases))

	for _, tc := range testCases {
		result, err := executeTestCase(ctx, tc)
		results = append(results, result)

		if err != nil && stopOnFailure {
			// Mark remaining tests as skipped
			for i := len(results); i < len(testCases); i++ {
				results = append(results, TestCaseResult{
					TestCaseID: testCases[i].ID,
					Name:       testCases[i].Name,
					Status:     StatusCanceled,
					Error:      "skipped due to previous failure",
				})
			}
			return results, err
		}
	}

	return results, nil
}

func executeTestCasesParallel(ctx workflow.Context, testCases []TestCase) ([]TestCaseResult, error) {
	var futures []workflow.Future
	results := make([]TestCaseResult, len(testCases))

	// Start all test cases
	for i, tc := range testCases {
		future := workflow.ExecuteActivity(ctx, ExecuteTestCaseActivity, ExecuteTestCaseRequest{
			TestCase: tc,
		})
		futures = append(futures, future)
		results[i] = TestCaseResult{
			TestCaseID: tc.ID,
			Name:       tc.Name,
			Status:     StatusRunning,
		}
	}

	// Collect results
	var firstError error
	for i, future := range futures {
		var response ExecuteTestCaseResponse
		if err := future.Get(ctx, &response); err != nil {
			results[i].Status = StatusFailed
			results[i].Error = err.Error()
			results[i].FinishedAt = workflow.Now(ctx)
			if firstError == nil {
				firstError = fmt.Errorf("test case %s failed: %w", testCases[i].ID, err)
			}
		} else {
			results[i] = response.Result
		}
	}

	return results, firstError
}

func executeTestCase(ctx workflow.Context, tc TestCase) (TestCaseResult, error) {
	var response ExecuteTestCaseResponse

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: tc.Timeout,
	}
	if tc.Timeout == 0 {
		activityOptions.StartToCloseTimeout = 5 * time.Minute
	}

	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteActivity(ctx, ExecuteTestCaseActivity, ExecuteTestCaseRequest{
		TestCase: tc,
	}).Get(ctx, &response)

	if err != nil {
		return TestCaseResult{
			TestCaseID: tc.ID,
			Name:       tc.Name,
			Status:     StatusFailed,
			Error:      err.Error(),
			FinishedAt: workflow.Now(ctx),
		}, err
	}

	return response.Result, nil
}

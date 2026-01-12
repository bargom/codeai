package definitions

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// AIAgentPipelineWorkflow orchestrates multi-agent AI tasks.
func AIAgentPipelineWorkflow(ctx workflow.Context, input PipelineInput) (PipelineOutput, error) {
	output := PipelineOutput{
		WorkflowID: input.WorkflowID,
		Status:     StatusRunning,
		Results:    make([]AgentResult, 0, len(input.Agents)),
		StartedAt:  workflow.Now(ctx),
	}

	// Set workflow timeout
	timeout := input.Timeout
	if timeout == 0 {
		timeout = 30 * time.Minute
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

	// Validate input
	var validationResult ValidationResult
	if err := workflow.ExecuteActivity(ctx, ValidateInputActivity, ValidateInputRequest{
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
		output.Error = fmt.Sprintf("invalid input: %s", validationResult.Error)
		output.CompletedAt = workflow.Now(ctx)
		output.TotalDuration = output.CompletedAt.Sub(output.StartedAt)
		return output, fmt.Errorf("invalid input: %s", validationResult.Error)
	}

	// Execute agents
	if input.Parallel {
		// Execute agents in parallel
		results, err := executeAgentsParallel(ctx, input.Agents)
		if err != nil {
			output.Status = StatusFailed
			output.Error = err.Error()
		} else {
			output.Results = results
			output.Status = StatusCompleted
		}
	} else {
		// Execute agents sequentially
		results, err := executeAgentsSequential(ctx, input.Agents)
		if err != nil {
			output.Status = StatusFailed
			output.Error = err.Error()
		} else {
			output.Results = results
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
	_ = workflow.ExecuteActivity(storageCtx, StorePipelineResultActivity, StorePipelineResultRequest{
		WorkflowID: input.WorkflowID,
		Output:     output,
	}).Get(ctx, &storageResult)

	output.CompletedAt = workflow.Now(ctx)
	output.TotalDuration = output.CompletedAt.Sub(output.StartedAt)

	if output.Status == StatusFailed {
		return output, fmt.Errorf("pipeline failed: %s", output.Error)
	}

	return output, nil
}

func executeAgentsSequential(ctx workflow.Context, agents []AgentConfig) ([]AgentResult, error) {
	results := make([]AgentResult, 0, len(agents))

	for _, agent := range agents {
		result, err := executeAgent(ctx, agent)
		if err != nil {
			result.Status = StatusFailed
			result.Error = err.Error()
		}
		results = append(results, result)

		if result.Status == StatusFailed {
			return results, fmt.Errorf("agent %s failed: %s", agent.Name, result.Error)
		}
	}

	return results, nil
}

func executeAgentsParallel(ctx workflow.Context, agents []AgentConfig) ([]AgentResult, error) {
	var futures []workflow.Future
	results := make([]AgentResult, len(agents))

	// Start all agents
	for i, agent := range agents {
		future := workflow.ExecuteActivity(ctx, ExecuteAgentActivity, ExecuteAgentRequest{
			Agent: agent,
		})
		futures = append(futures, future)
		results[i] = AgentResult{
			AgentName: agent.Name,
			Status:    StatusRunning,
		}
	}

	// Collect results
	var firstError error
	for i, future := range futures {
		var response ExecuteAgentResponse
		if err := future.Get(ctx, &response); err != nil {
			results[i].Status = StatusFailed
			results[i].Error = err.Error()
			results[i].FinishedAt = workflow.Now(ctx)
			if firstError == nil {
				firstError = fmt.Errorf("agent %s failed: %w", agents[i].Name, err)
			}
		} else {
			results[i] = response.Result
		}
	}

	return results, firstError
}

func executeAgent(ctx workflow.Context, agent AgentConfig) (AgentResult, error) {
	var response ExecuteAgentResponse

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: agent.Timeout,
	}
	if agent.Timeout == 0 {
		activityOptions.StartToCloseTimeout = 10 * time.Minute
	}

	if agent.RetryPolicy.MaxAttempts > 0 {
		activityOptions.RetryPolicy = &temporal.RetryPolicy{
			InitialInterval:    agent.RetryPolicy.InitialInterval,
			BackoffCoefficient: agent.RetryPolicy.BackoffCoefficient,
			MaximumInterval:    agent.RetryPolicy.MaximumInterval,
			MaximumAttempts:    int32(agent.RetryPolicy.MaxAttempts),
		}
	}

	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteActivity(ctx, ExecuteAgentActivity, ExecuteAgentRequest{
		Agent: agent,
	}).Get(ctx, &response)

	return response.Result, err
}

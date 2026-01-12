package definitions

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/bargom/codeai/internal/workflow/compensation"
	compActivities "github.com/bargom/codeai/internal/workflow/activities/compensation"
)

// AIAgentPipelineWorkflowWithCompensation orchestrates multi-agent AI tasks with compensation support.
func AIAgentPipelineWorkflowWithCompensation(ctx workflow.Context, input PipelineInput) (PipelineOutput, error) {
	logger := workflow.GetLogger(ctx)
	cm := compensation.NewCompensationManager(ctx)

	output := PipelineOutput{
		WorkflowID: input.WorkflowID,
		Status:     StatusRunning,
		Results:    make([]AgentResult, 0, len(input.Agents)),
		StartedAt:  workflow.Now(ctx),
	}

	logger.Info("Starting AI pipeline with compensation",
		"workflowId", input.WorkflowID,
		"agents", len(input.Agents),
	)

	// Set workflow timeout
	timeout := input.Timeout
	if timeout == 0 {
		timeout = 30 * time.Minute
	}

	// Configure default activity options
	defaultRetryPolicy := &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    60 * time.Second,
		MaximumAttempts:    3,
	}

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: timeout,
		RetryPolicy:         defaultRetryPolicy,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Step 1: Validate input
	logger.Info("Validating pipeline input")
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

	// Execute agents with compensation
	if input.Parallel {
		results, err := executeAgentsWithCompensationParallel(ctx, cm, input)
		if err != nil {
			output.Status = StatusFailed
			output.Error = err.Error()
			output.Results = results
		} else {
			output.Results = results
			output.Status = StatusCompleted
		}
	} else {
		results, err := executeAgentsWithCompensationSequential(ctx, cm, input)
		if err != nil {
			output.Status = StatusFailed
			output.Error = err.Error()
			output.Results = results
		} else {
			output.Results = results
			output.Status = StatusCompleted
		}
	}

	// Step 3: Store results with compensation
	cm.RegisterCompensation(compensation.CompensationStep{
		ActivityName: "store-results",
		CompensateFn: func(ctx workflow.Context, input interface{}) error {
			rollbackInput := input.(compActivities.FileRollbackInput)
			var rollbackOutput compActivities.RollbackOutput
			return workflow.ExecuteActivity(ctx, RollbackFileOperationActivity, rollbackInput).Get(ctx, &rollbackOutput)
		},
		Input: compActivities.FileRollbackInput{
			RollbackInput: compActivities.RollbackInput{
				WorkflowID:   input.WorkflowID,
				ActivityName: "store-results",
				ResourceType: "pipeline_result",
			},
			Operation: "create",
		},
		Timeout:   30 * time.Second,
		AllowSkip: true, // Storage failure is non-critical
	})

	storageCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	})

	var storageResult StorageResult
	err := workflow.ExecuteActivity(storageCtx, StorePipelineResultActivity, StorePipelineResultRequest{
		WorkflowID: input.WorkflowID,
		Output:     output,
	}).Get(ctx, &storageResult)

	if err != nil {
		logger.Warn("Failed to store pipeline results", "error", err)
		// Storage failure is logged but doesn't fail the workflow
	} else {
		cm.RecordExecution("store-results")
	}

	// Step 4: Send notification with compensation (non-critical)
	cm.RegisterCompensation(compensation.CompensationStep{
		ActivityName: "send-notification",
		CompensateFn: func(ctx workflow.Context, input interface{}) error {
			// Notifications cannot typically be "unsent"
			logger.Warn("Cannot compensate notification - already sent")
			return nil
		},
		AllowSkip: true, // Notification failure is non-critical
	})

	if output.Status == StatusCompleted {
		notifyCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 2,
			},
		})

		err := workflow.ExecuteActivity(notifyCtx, SendNotificationActivity, SendNotificationRequest{
			WorkflowID: input.WorkflowID,
			Type:       "pipeline_completed",
			Message:    fmt.Sprintf("Pipeline %s completed successfully", input.WorkflowID),
		}).Get(ctx, nil)

		if err != nil {
			logger.Warn("Failed to send completion notification", "error", err)
		} else {
			cm.RecordExecution("send-notification")
		}
	}

	output.CompletedAt = workflow.Now(ctx)
	output.TotalDuration = output.CompletedAt.Sub(output.StartedAt)

	logger.Info("AI pipeline with compensation completed",
		"status", output.Status,
		"duration", output.TotalDuration,
	)

	if output.Status == StatusFailed {
		return output, fmt.Errorf("pipeline failed: %s", output.Error)
	}

	return output, nil
}

// executeAgentsWithCompensationSequential executes agents sequentially with compensation.
func executeAgentsWithCompensationSequential(ctx workflow.Context, cm *compensation.CompensationManager, input PipelineInput) ([]AgentResult, error) {
	logger := workflow.GetLogger(ctx)
	results := make([]AgentResult, 0, len(input.Agents))

	for _, agent := range input.Agents {
		// Register compensation before execution
		cm.RegisterCompensation(compensation.CompensationStep{
			ActivityName: fmt.Sprintf("agent-%s", agent.Name),
			CompensateFn: func(ctx workflow.Context, input interface{}) error {
				rollbackInput := input.(compActivities.AgentRollbackInput)
				var rollbackOutput compActivities.RollbackOutput
				return workflow.ExecuteActivity(ctx, RollbackAgentExecutionActivity, rollbackInput).Get(ctx, &rollbackOutput)
			},
			Input: compActivities.AgentRollbackInput{
				RollbackInput: compActivities.RollbackInput{
					WorkflowID:   input.WorkflowID,
					ActivityName: fmt.Sprintf("agent-%s", agent.Name),
					ResourceType: "agent_execution",
				},
				AgentType: agent.Type,
				AgentName: agent.Name,
			},
			Timeout: 60 * time.Second,
		})

		// Execute agent
		result, err := executeAgentWithCompensation(ctx, agent)
		if err != nil {
			result.Status = StatusFailed
			result.Error = err.Error()
			results = append(results, result)

			logger.Error("Agent execution failed, starting compensation",
				"agent", agent.Name,
				"error", err,
			)

			// Compensate all previously executed agents
			if compErr := cm.Compensate(ctx); compErr != nil {
				return results, fmt.Errorf("agent %s failed: %w (compensation failed: %v)", agent.Name, err, compErr)
			}

			return results, fmt.Errorf("agent %s failed: %w (compensated)", agent.Name, err)
		}

		cm.RecordExecution(fmt.Sprintf("agent-%s", agent.Name))
		results = append(results, result)
	}

	return results, nil
}

// executeAgentsWithCompensationParallel executes agents in parallel with compensation.
func executeAgentsWithCompensationParallel(ctx workflow.Context, cm *compensation.CompensationManager, input PipelineInput) ([]AgentResult, error) {
	logger := workflow.GetLogger(ctx)
	var futures []workflow.Future
	results := make([]AgentResult, len(input.Agents))

	// Register compensation and start all agents
	for i, agent := range input.Agents {
		cm.RegisterCompensation(compensation.CompensationStep{
			ActivityName: fmt.Sprintf("agent-%s", agent.Name),
			CompensateFn: func(ctx workflow.Context, input interface{}) error {
				rollbackInput := input.(compActivities.AgentRollbackInput)
				var rollbackOutput compActivities.RollbackOutput
				return workflow.ExecuteActivity(ctx, RollbackAgentExecutionActivity, rollbackInput).Get(ctx, &rollbackOutput)
			},
			Input: compActivities.AgentRollbackInput{
				RollbackInput: compActivities.RollbackInput{
					WorkflowID:   input.WorkflowID,
					ActivityName: fmt.Sprintf("agent-%s", agent.Name),
					ResourceType: "agent_execution",
				},
				AgentType: agent.Type,
				AgentName: agent.Name,
			},
			Timeout: 60 * time.Second,
		})

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
	executedAgents := make([]string, 0)

	for i, future := range futures {
		var response ExecuteAgentResponse
		if err := future.Get(ctx, &response); err != nil {
			results[i].Status = StatusFailed
			results[i].Error = err.Error()
			results[i].FinishedAt = workflow.Now(ctx)
			if firstError == nil {
				firstError = fmt.Errorf("agent %s failed: %w", input.Agents[i].Name, err)
			}
		} else {
			results[i] = response.Result
			executedAgents = append(executedAgents, fmt.Sprintf("agent-%s", input.Agents[i].Name))
			cm.RecordExecution(fmt.Sprintf("agent-%s", input.Agents[i].Name))
		}
	}

	if firstError != nil {
		logger.Error("Parallel agent execution had failures, starting compensation")

		// Compensate only executed agents
		if compErr := cm.Compensate(ctx); compErr != nil {
			return results, fmt.Errorf("parallel execution failed: %w (compensation failed: %v)", firstError, compErr)
		}

		return results, fmt.Errorf("parallel execution failed: %w (compensated)", firstError)
	}

	return results, nil
}

// executeAgentWithCompensation executes a single agent.
func executeAgentWithCompensation(ctx workflow.Context, agent AgentConfig) (AgentResult, error) {
	var response ExecuteAgentResponse

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: agent.Timeout,
	}
	if activityOptions.StartToCloseTimeout == 0 {
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

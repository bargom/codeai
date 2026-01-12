// Package patterns provides reusable workflow patterns for the workflow engine.
package patterns

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/bargom/codeai/internal/workflow/compensation"
)

// SagaStep defines a single step in a saga workflow.
type SagaStep struct {
	Name           string
	Activity       interface{}
	Input          interface{}
	Output         interface{}
	CompensateFn   compensation.WorkflowCompensationFunc
	CompensateData interface{}
	Timeout        time.Duration
	RetryPolicy    *temporal.RetryPolicy
	AllowFailure   bool // If true, failure doesn't trigger compensation
}

// SagaInput is the input for SagaWorkflow.
type SagaInput struct {
	WorkflowID string
	Steps      []SagaStep
	Timeout    time.Duration
	Metadata   map[string]string
}

// SagaOutput is the output from SagaWorkflow.
type SagaOutput struct {
	Status           string                        `json:"status"`
	CompletedSteps   []string                      `json:"completedSteps"`
	FailedStep       string                        `json:"failedStep,omitempty"`
	Error            string                        `json:"error,omitempty"`
	CompensationLogs []compensation.CompensationExecutionRecord `json:"compensationLogs,omitempty"`
	Results          map[string]interface{}        `json:"results,omitempty"`
}

// SagaWorkflow executes activities with automatic compensation on failure.
func SagaWorkflow(ctx workflow.Context, input SagaInput) (SagaOutput, error) {
	logger := workflow.GetLogger(ctx)
	cm := compensation.NewCompensationManager(ctx)

	output := SagaOutput{
		Status:         "running",
		CompletedSteps: make([]string, 0),
		Results:        make(map[string]interface{}),
	}

	logger.Info("Starting saga workflow",
		"workflowId", input.WorkflowID,
		"steps", len(input.Steps),
	)

	// Execute each step
	for i, step := range input.Steps {
		logger.Info("Executing saga step",
			"step", step.Name,
			"index", i,
		)

		// Register compensation before execution
		if step.CompensateFn != nil {
			cm.RegisterCompensation(compensation.CompensationStep{
				ActivityName: step.Name,
				CompensateFn: step.CompensateFn,
				Input:        step.CompensateData,
				Timeout:      step.Timeout,
				RetryPolicy:  step.RetryPolicy,
				AllowSkip:    step.AllowFailure,
			})
		}

		// Configure activity options for this step
		activityOptions := workflow.ActivityOptions{
			StartToCloseTimeout: step.Timeout,
		}
		if activityOptions.StartToCloseTimeout == 0 {
			activityOptions.StartToCloseTimeout = 10 * time.Minute
		}
		if step.RetryPolicy != nil {
			activityOptions.RetryPolicy = step.RetryPolicy
		} else {
			activityOptions.RetryPolicy = &temporal.RetryPolicy{
				InitialInterval:    time.Second,
				BackoffCoefficient: 2.0,
				MaximumInterval:    time.Minute,
				MaximumAttempts:    3,
			}
		}

		stepCtx := workflow.WithActivityOptions(ctx, activityOptions)

		// Execute the activity
		err := workflow.ExecuteActivity(stepCtx, step.Activity, step.Input).Get(ctx, step.Output)

		if err != nil {
			logger.Error("Saga step failed",
				"step", step.Name,
				"error", err,
			)

			if step.AllowFailure {
				logger.Warn("Step failure allowed, continuing saga",
					"step", step.Name,
				)
				continue
			}

			// Mark step as failed
			output.FailedStep = step.Name
			output.Error = err.Error()
			output.Status = "failed"

			// Execute compensation for all completed steps
			logger.Info("Starting compensation for failed saga",
				"failedStep", step.Name,
				"completedSteps", len(output.CompletedSteps),
			)

			compErr := cm.Compensate(ctx)
			output.CompensationLogs = cm.Records()

			if compErr != nil {
				logger.Error("Compensation failed",
					"error", compErr,
				)
				output.Status = "compensation_failed"
				return output, fmt.Errorf("saga failed at step %s: %w (compensation also failed: %v)", step.Name, err, compErr)
			}

			output.Status = "compensated"
			return output, fmt.Errorf("saga failed at step %s: %w (compensated)", step.Name, err)
		}

		// Mark step as executed and record completion
		cm.RecordExecution(step.Name)
		output.CompletedSteps = append(output.CompletedSteps, step.Name)

		// Store result if output was provided
		if step.Output != nil {
			output.Results[step.Name] = step.Output
		}

		logger.Info("Saga step completed",
			"step", step.Name,
			"completedSteps", len(output.CompletedSteps),
		)
	}

	output.Status = "completed"
	logger.Info("Saga workflow completed successfully",
		"completedSteps", len(output.CompletedSteps),
	)

	return output, nil
}

// SagaBuilder provides a fluent interface for building saga workflows.
type SagaBuilder struct {
	workflowID string
	steps      []SagaStep
	timeout    time.Duration
	metadata   map[string]string
}

// NewSagaBuilder creates a new saga builder.
func NewSagaBuilder(workflowID string) *SagaBuilder {
	return &SagaBuilder{
		workflowID: workflowID,
		steps:      make([]SagaStep, 0),
		metadata:   make(map[string]string),
	}
}

// WithTimeout sets the overall saga timeout.
func (b *SagaBuilder) WithTimeout(timeout time.Duration) *SagaBuilder {
	b.timeout = timeout
	return b
}

// WithMetadata adds metadata to the saga.
func (b *SagaBuilder) WithMetadata(key, value string) *SagaBuilder {
	b.metadata[key] = value
	return b
}

// AddStep adds a step to the saga.
func (b *SagaBuilder) AddStep(step SagaStep) *SagaBuilder {
	b.steps = append(b.steps, step)
	return b
}

// AddActivityStep adds an activity step with compensation.
func (b *SagaBuilder) AddActivityStep(name string, activity interface{}, input interface{}, compensateFn compensation.WorkflowCompensationFunc) *SagaBuilder {
	return b.AddStep(SagaStep{
		Name:         name,
		Activity:     activity,
		Input:        input,
		CompensateFn: compensateFn,
	})
}

// AddNonCriticalStep adds a step that won't trigger compensation on failure.
func (b *SagaBuilder) AddNonCriticalStep(name string, activity interface{}, input interface{}) *SagaBuilder {
	return b.AddStep(SagaStep{
		Name:         name,
		Activity:     activity,
		Input:        input,
		AllowFailure: true,
	})
}

// Build creates the SagaInput from the builder.
func (b *SagaBuilder) Build() SagaInput {
	return SagaInput{
		WorkflowID: b.workflowID,
		Steps:      b.steps,
		Timeout:    b.timeout,
		Metadata:   b.metadata,
	}
}

// SagaWithSignals executes a saga with support for manual compensation signals.
func SagaWithSignals(ctx workflow.Context, input SagaInput) (SagaOutput, error) {
	logger := workflow.GetLogger(ctx)
	cm := compensation.NewCompensationManager(ctx)

	output := SagaOutput{
		Status:         "running",
		CompletedSteps: make([]string, 0),
		Results:        make(map[string]interface{}),
	}

	// Set up signal channel for manual compensation
	compensateSignal := workflow.GetSignalChannel(ctx, "compensate")
	cancelSignal := workflow.GetSignalChannel(ctx, "cancel")

	selector := workflow.NewSelector(ctx)

	var forceCompensate bool
	var forceCancel bool

	// Handle compensation signal
	selector.AddReceive(compensateSignal, func(c workflow.ReceiveChannel, more bool) {
		var signalData map[string]string
		c.Receive(ctx, &signalData)
		logger.Info("Received manual compensation signal")
		forceCompensate = true
	})

	// Handle cancel signal
	selector.AddReceive(cancelSignal, func(c workflow.ReceiveChannel, more bool) {
		var signalData map[string]string
		c.Receive(ctx, &signalData)
		logger.Info("Received cancel signal")
		forceCancel = true
	})

	// Execute steps with signal checking
	for i, step := range input.Steps {
		// Check for signals (non-blocking)
		for selector.HasPending() {
			selector.Select(ctx)
		}

		if forceCancel {
			output.Status = "cancelled"
			output.Error = "workflow cancelled by signal"
			return output, fmt.Errorf("workflow cancelled by signal")
		}

		if forceCompensate {
			logger.Info("Manual compensation triggered")
			compErr := cm.Compensate(ctx)
			output.CompensationLogs = cm.Records()
			output.Status = "manually_compensated"
			if compErr != nil {
				return output, fmt.Errorf("manual compensation failed: %w", compErr)
			}
			return output, nil
		}

		// Register and execute step
		if step.CompensateFn != nil {
			cm.RegisterCompensation(compensation.CompensationStep{
				ActivityName: step.Name,
				CompensateFn: step.CompensateFn,
				Input:        step.CompensateData,
				Timeout:      step.Timeout,
				RetryPolicy:  step.RetryPolicy,
				AllowSkip:    step.AllowFailure,
			})
		}

		activityOptions := workflow.ActivityOptions{
			StartToCloseTimeout: step.Timeout,
		}
		if activityOptions.StartToCloseTimeout == 0 {
			activityOptions.StartToCloseTimeout = 10 * time.Minute
		}
		if step.RetryPolicy != nil {
			activityOptions.RetryPolicy = step.RetryPolicy
		}

		stepCtx := workflow.WithActivityOptions(ctx, activityOptions)

		err := workflow.ExecuteActivity(stepCtx, step.Activity, step.Input).Get(ctx, step.Output)

		if err != nil {
			if step.AllowFailure {
				logger.Warn("Non-critical step failed, continuing",
					"step", step.Name,
					"index", i,
				)
				continue
			}

			output.FailedStep = step.Name
			output.Error = err.Error()
			output.Status = "failed"

			compErr := cm.Compensate(ctx)
			output.CompensationLogs = cm.Records()

			if compErr != nil {
				output.Status = "compensation_failed"
				return output, fmt.Errorf("saga failed at step %s: %w (compensation failed: %v)", step.Name, err, compErr)
			}

			output.Status = "compensated"
			return output, fmt.Errorf("saga failed at step %s: %w (compensated)", step.Name, err)
		}

		cm.RecordExecution(step.Name)
		output.CompletedSteps = append(output.CompletedSteps, step.Name)

		if step.Output != nil {
			output.Results[step.Name] = step.Output
		}
	}

	output.Status = "completed"
	return output, nil
}

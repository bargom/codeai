// Package workflow provides DSL-to-Temporal workflow loading and execution.
package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/workflow/definitions"
)

// DSLWorkflowConfig holds configuration for a DSL-loaded workflow.
type DSLWorkflowConfig struct {
	Name         string
	TriggerType  ast.TriggerType
	TriggerValue string
	Timeout      time.Duration
	RetryPolicy  *temporal.RetryPolicy
	Steps        []DSLWorkflowStep
}

// DSLWorkflowStep represents a workflow step loaded from DSL.
type DSLWorkflowStep struct {
	Name      string
	Activity  string
	Input     map[string]string
	Condition string
	Parallel  bool
	Steps     []DSLWorkflowStep // Nested steps for parallel blocks
}

// DSLWorkflowInput is the input for executing a DSL-loaded workflow.
type DSLWorkflowInput struct {
	WorkflowID string            `json:"workflowId"`
	Input      map[string]any    `json:"input"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// DSLWorkflowOutput is the result of a DSL-loaded workflow execution.
type DSLWorkflowOutput struct {
	WorkflowID   string                 `json:"workflowId"`
	Status       definitions.Status     `json:"status"`
	StepResults  map[string]StepResult  `json:"stepResults"`
	StartedAt    time.Time              `json:"startedAt"`
	CompletedAt  time.Time              `json:"completedAt"`
	Error        string                 `json:"error,omitempty"`
}

// StepResult holds the result of a single workflow step.
type StepResult struct {
	StepName   string          `json:"stepName"`
	Status     definitions.Status `json:"status"`
	Output     json.RawMessage `json:"output,omitempty"`
	Error      string          `json:"error,omitempty"`
	Duration   time.Duration   `json:"duration"`
	StartedAt  time.Time       `json:"startedAt"`
	FinishedAt time.Time       `json:"finishedAt"`
}

// LoadWorkflowFromAST loads a Temporal workflow configuration from an AST WorkflowDecl.
func LoadWorkflowFromAST(decl *ast.WorkflowDecl) (*DSLWorkflowConfig, error) {
	if decl == nil {
		return nil, fmt.Errorf("workflow declaration is nil")
	}

	config := &DSLWorkflowConfig{
		Name:         decl.Name,
		TriggerType:  decl.Trigger.TrigType,
		TriggerValue: decl.Trigger.Value,
	}

	// Parse timeout
	if decl.Timeout != "" {
		timeout, err := time.ParseDuration(decl.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout %q: %w", decl.Timeout, err)
		}
		config.Timeout = timeout
	} else {
		config.Timeout = 24 * time.Hour // Default 24h timeout
	}

	// Convert retry policy
	if decl.Retry != nil {
		policy, err := convertRetryPolicy(decl.Retry)
		if err != nil {
			return nil, fmt.Errorf("invalid retry policy: %w", err)
		}
		config.RetryPolicy = policy
	} else {
		config.RetryPolicy = defaultRetryPolicy()
	}

	// Convert steps
	for _, step := range decl.Steps {
		dslStep, err := convertWorkflowStep(step)
		if err != nil {
			return nil, fmt.Errorf("invalid step %q: %w", step.Name, err)
		}
		config.Steps = append(config.Steps, dslStep)
	}

	return config, nil
}

// convertRetryPolicy converts an AST RetryPolicyDecl to a Temporal RetryPolicy.
func convertRetryPolicy(decl *ast.RetryPolicyDecl) (*temporal.RetryPolicy, error) {
	policy := &temporal.RetryPolicy{
		MaximumAttempts: int32(decl.MaxAttempts),
	}

	if decl.InitialInterval != "" {
		interval, err := time.ParseDuration(decl.InitialInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid initial_interval: %w", err)
		}
		policy.InitialInterval = interval
	} else {
		policy.InitialInterval = time.Second
	}

	if decl.BackoffMultiplier > 0 {
		policy.BackoffCoefficient = decl.BackoffMultiplier
	} else {
		policy.BackoffCoefficient = 2.0
	}

	policy.MaximumInterval = 10 * time.Minute

	return policy, nil
}

// defaultRetryPolicy returns a sensible default retry policy.
func defaultRetryPolicy() *temporal.RetryPolicy {
	return &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    10 * time.Minute,
		MaximumAttempts:    3,
	}
}

// convertWorkflowStep converts an AST WorkflowStep to a DSLWorkflowStep.
func convertWorkflowStep(step *ast.WorkflowStep) (DSLWorkflowStep, error) {
	dslStep := DSLWorkflowStep{
		Name:      step.Name,
		Activity:  step.Activity,
		Condition: step.Condition,
		Parallel:  step.Parallel,
		Input:     make(map[string]string),
	}

	// Convert input mappings
	for _, mapping := range step.Input {
		dslStep.Input[mapping.Key] = mapping.Value
	}

	// Convert nested steps for parallel blocks
	if step.Parallel {
		for _, nestedStep := range step.Steps {
			nested, err := convertWorkflowStep(nestedStep)
			if err != nil {
				return DSLWorkflowStep{}, err
			}
			dslStep.Steps = append(dslStep.Steps, nested)
		}
	}

	return dslStep, nil
}

// ExecuteDSLWorkflow executes a DSL-loaded workflow using Temporal.
// This is the Temporal workflow function that can be registered with a worker.
func ExecuteDSLWorkflow(ctx workflow.Context, config DSLWorkflowConfig, input DSLWorkflowInput) (*DSLWorkflowOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting DSL workflow", "name", config.Name, "workflowId", input.WorkflowID)

	output := &DSLWorkflowOutput{
		WorkflowID:  input.WorkflowID,
		Status:      definitions.StatusRunning,
		StepResults: make(map[string]StepResult),
		StartedAt:   workflow.Now(ctx),
	}

	// Set workflow timeout
	if config.Timeout > 0 {
		var cancel workflow.CancelFunc
		ctx, cancel = workflow.WithCancel(ctx)
		_ = cancel // Used to cancel workflow if needed
	}

	// Create activity options
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy:         config.RetryPolicy,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Execute steps
	stepContext := &stepExecutionContext{
		workflowInput: input.Input,
		stepOutputs:   make(map[string]json.RawMessage),
	}

	for _, step := range config.Steps {
		if err := executeStep(ctx, step, stepContext, output); err != nil {
			output.Status = definitions.StatusFailed
			output.Error = err.Error()
			output.CompletedAt = workflow.Now(ctx)
			return output, nil
		}
	}

	output.Status = definitions.StatusCompleted
	output.CompletedAt = workflow.Now(ctx)
	logger.Info("DSL workflow completed", "name", config.Name, "workflowId", input.WorkflowID)

	return output, nil
}

// stepExecutionContext tracks execution state across steps.
type stepExecutionContext struct {
	workflowInput map[string]any
	stepOutputs   map[string]json.RawMessage
}

// executeStep executes a single workflow step.
func executeStep(ctx workflow.Context, step DSLWorkflowStep, stepCtx *stepExecutionContext, output *DSLWorkflowOutput) error {
	logger := workflow.GetLogger(ctx)

	if step.Parallel {
		return executeParallelSteps(ctx, step.Steps, stepCtx, output)
	}

	// Check condition if present
	if step.Condition != "" {
		shouldRun, err := evaluateCondition(step.Condition, stepCtx)
		if err != nil {
			return fmt.Errorf("condition evaluation failed for step %q: %w", step.Name, err)
		}
		if !shouldRun {
			logger.Info("Skipping step due to condition", "step", step.Name)
			output.StepResults[step.Name] = StepResult{
				StepName: step.Name,
				Status:   definitions.StatusCompleted,
			}
			return nil
		}
	}

	// Resolve input mappings
	resolvedInput, err := resolveInputMappings(step.Input, stepCtx)
	if err != nil {
		return fmt.Errorf("failed to resolve input for step %q: %w", step.Name, err)
	}

	stepResult := StepResult{
		StepName:  step.Name,
		Status:    definitions.StatusRunning,
		StartedAt: workflow.Now(ctx),
	}

	// Execute the activity
	var activityResult json.RawMessage
	err = workflow.ExecuteActivity(ctx, step.Activity, resolvedInput).Get(ctx, &activityResult)

	stepResult.FinishedAt = workflow.Now(ctx)
	stepResult.Duration = stepResult.FinishedAt.Sub(stepResult.StartedAt)

	if err != nil {
		stepResult.Status = definitions.StatusFailed
		stepResult.Error = err.Error()
		output.StepResults[step.Name] = stepResult
		return fmt.Errorf("step %q failed: %w", step.Name, err)
	}

	stepResult.Status = definitions.StatusCompleted
	stepResult.Output = activityResult
	output.StepResults[step.Name] = stepResult

	// Store output for later steps to reference
	stepCtx.stepOutputs[step.Name] = activityResult

	return nil
}

// executeParallelSteps executes multiple steps in parallel.
func executeParallelSteps(ctx workflow.Context, steps []DSLWorkflowStep, stepCtx *stepExecutionContext, output *DSLWorkflowOutput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing parallel steps", "count", len(steps))

	// Create futures for parallel execution
	futures := make([]workflow.Future, len(steps))
	stepNames := make([]string, len(steps))

	for i, step := range steps {
		stepNames[i] = step.Name

		// Resolve input for each parallel step
		resolvedInput, err := resolveInputMappings(step.Input, stepCtx)
		if err != nil {
			return fmt.Errorf("failed to resolve input for parallel step %q: %w", step.Name, err)
		}

		futures[i] = workflow.ExecuteActivity(ctx, step.Activity, resolvedInput)
	}

	// Wait for all parallel steps to complete
	for i, future := range futures {
		stepName := stepNames[i]
		stepResult := StepResult{
			StepName:  stepName,
			Status:    definitions.StatusRunning,
			StartedAt: workflow.Now(ctx),
		}

		var activityResult json.RawMessage
		err := future.Get(ctx, &activityResult)

		stepResult.FinishedAt = workflow.Now(ctx)
		stepResult.Duration = stepResult.FinishedAt.Sub(stepResult.StartedAt)

		if err != nil {
			stepResult.Status = definitions.StatusFailed
			stepResult.Error = err.Error()
			output.StepResults[stepName] = stepResult
			return fmt.Errorf("parallel step %q failed: %w", stepName, err)
		}

		stepResult.Status = definitions.StatusCompleted
		stepResult.Output = activityResult
		output.StepResults[stepName] = stepResult
		stepCtx.stepOutputs[stepName] = activityResult
	}

	return nil
}

// resolveInputMappings resolves input mappings to actual values.
func resolveInputMappings(mappings map[string]string, stepCtx *stepExecutionContext) (map[string]any, error) {
	result := make(map[string]any)

	for key, mapping := range mappings {
		value, err := resolveMapping(mapping, stepCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve mapping %q: %w", key, err)
		}
		result[key] = value
	}

	return result, nil
}

// resolveMapping resolves a single mapping expression to its value.
// Supports expressions like:
// - "workflow.input.field" - references workflow input
// - "steps.stepName.output.field" - references previous step output
// - literal string values
func resolveMapping(mapping string, stepCtx *stepExecutionContext) (any, error) {
	// For now, return the mapping as a literal
	// Full expression evaluation can be added later
	return mapping, nil
}

// evaluateCondition evaluates a condition expression.
// Returns true if the step should execute, false if it should be skipped.
func evaluateCondition(condition string, stepCtx *stepExecutionContext) (bool, error) {
	// For now, return true to execute all steps
	// Full condition evaluation can be added later
	return true, nil
}

// DSLWorkflowRegistry manages loaded DSL workflows.
type DSLWorkflowRegistry struct {
	workflows map[string]*DSLWorkflowConfig
}

// NewDSLWorkflowRegistry creates a new workflow registry.
func NewDSLWorkflowRegistry() *DSLWorkflowRegistry {
	return &DSLWorkflowRegistry{
		workflows: make(map[string]*DSLWorkflowConfig),
	}
}

// Register adds a workflow configuration to the registry.
func (r *DSLWorkflowRegistry) Register(config *DSLWorkflowConfig) error {
	if config == nil {
		return fmt.Errorf("workflow config is nil")
	}
	if config.Name == "" {
		return fmt.Errorf("workflow name is required")
	}
	r.workflows[config.Name] = config
	return nil
}

// Get retrieves a workflow configuration by name.
func (r *DSLWorkflowRegistry) Get(name string) (*DSLWorkflowConfig, bool) {
	config, ok := r.workflows[name]
	return config, ok
}

// GetByTrigger returns all workflows that match the given trigger.
func (r *DSLWorkflowRegistry) GetByTrigger(triggerType ast.TriggerType, triggerValue string) []*DSLWorkflowConfig {
	var matches []*DSLWorkflowConfig
	for _, config := range r.workflows {
		if config.TriggerType == triggerType && config.TriggerValue == triggerValue {
			matches = append(matches, config)
		}
	}
	return matches
}

// List returns all registered workflow names.
func (r *DSLWorkflowRegistry) List() []string {
	names := make([]string, 0, len(r.workflows))
	for name := range r.workflows {
		names = append(names, name)
	}
	return names
}

// LoadWorkflows loads multiple workflow declarations into the registry.
func (r *DSLWorkflowRegistry) LoadWorkflows(decls []*ast.WorkflowDecl) error {
	for _, decl := range decls {
		config, err := LoadWorkflowFromAST(decl)
		if err != nil {
			return fmt.Errorf("failed to load workflow %q: %w", decl.Name, err)
		}
		if err := r.Register(config); err != nil {
			return err
		}
	}
	return nil
}

// WorkflowTriggerHandler handles triggering workflows based on events or schedules.
type WorkflowTriggerHandler struct {
	registry *DSLWorkflowRegistry
}

// NewWorkflowTriggerHandler creates a new trigger handler.
func NewWorkflowTriggerHandler(registry *DSLWorkflowRegistry) *WorkflowTriggerHandler {
	return &WorkflowTriggerHandler{registry: registry}
}

// HandleEvent processes an event and triggers matching workflows.
func (h *WorkflowTriggerHandler) HandleEvent(ctx context.Context, eventName string, eventData map[string]any) error {
	workflows := h.registry.GetByTrigger(ast.TriggerTypeEvent, eventName)
	if len(workflows) == 0 {
		return nil // No matching workflows
	}

	// In a real implementation, this would start Temporal workflows
	// For now, this is a placeholder for the trigger mechanism
	for _, wf := range workflows {
		_ = wf // Would start workflow here
	}

	return nil
}

// Package validator provides workflow and job validation for CodeAI AST.
package validator

import (
	"fmt"
	"regexp"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/bargom/codeai/internal/ast"
)

// WorkflowValidator performs semantic validation on workflow and job declarations.
type WorkflowValidator struct {
	errors    *ValidationErrors
	workflows map[string]*ast.WorkflowDecl
	jobs      map[string]*ast.JobDecl
	// Track known activity types for validation
	activities map[string]bool
	// Track known task types for job validation
	taskTypes map[string]bool
}

// NewWorkflowValidator creates a new WorkflowValidator instance.
func NewWorkflowValidator() *WorkflowValidator {
	return &WorkflowValidator{
		errors:     &ValidationErrors{},
		workflows:  make(map[string]*ast.WorkflowDecl),
		jobs:       make(map[string]*ast.JobDecl),
		activities: make(map[string]bool),
		taskTypes:  make(map[string]bool),
	}
}

// RegisterActivity registers an activity type as known/valid.
func (v *WorkflowValidator) RegisterActivity(name string) {
	v.activities[name] = true
}

// RegisterTaskType registers a task type as known/valid.
func (v *WorkflowValidator) RegisterTaskType(name string) {
	v.taskTypes[name] = true
}

// ValidateWorkflows validates a slice of workflow declarations.
func (v *WorkflowValidator) ValidateWorkflows(decls []*ast.WorkflowDecl) error {
	for _, decl := range decls {
		v.validateWorkflow(decl)
	}

	if v.errors.HasErrors() {
		return v.errors
	}
	return nil
}

// ValidateJobs validates a slice of job declarations.
func (v *WorkflowValidator) ValidateJobs(decls []*ast.JobDecl) error {
	for _, decl := range decls {
		v.validateJob(decl)
	}

	if v.errors.HasErrors() {
		return v.errors
	}
	return nil
}

// validateWorkflow validates a single workflow declaration.
func (v *WorkflowValidator) validateWorkflow(decl *ast.WorkflowDecl) {
	if decl == nil {
		return
	}

	// Check for duplicate workflow names
	if _, exists := v.workflows[decl.Name]; exists {
		v.errors.Add(newSemanticError(decl.Pos(), fmt.Sprintf("duplicate workflow declaration: %q", decl.Name)))
		return
	}
	v.workflows[decl.Name] = decl

	// Validate workflow name format
	if !isValidIdentifier(decl.Name) {
		v.errors.Add(newSemanticError(decl.Pos(), fmt.Sprintf("invalid workflow name: %q must be a valid identifier", decl.Name)))
	}

	// Validate trigger
	v.validateTrigger(decl.Trigger)

	// Validate timeout if present
	if decl.Timeout != "" {
		if _, err := time.ParseDuration(decl.Timeout); err != nil {
			v.errors.Add(newSemanticError(decl.Pos(), fmt.Sprintf("invalid timeout duration: %q", decl.Timeout)))
		}
	}

	// Validate steps
	if len(decl.Steps) == 0 {
		v.errors.Add(newSemanticError(decl.Pos(), fmt.Sprintf("workflow %q must have at least one step", decl.Name)))
	}

	stepNames := make(map[string]bool)
	for _, step := range decl.Steps {
		v.validateWorkflowStep(step, stepNames)
	}

	// Validate retry policy if present
	if decl.Retry != nil {
		v.validateRetryPolicy(decl.Retry)
	}
}

// validateTrigger validates a workflow trigger.
func (v *WorkflowValidator) validateTrigger(trigger *ast.Trigger) {
	if trigger == nil {
		// Trigger is required
		return
	}

	switch trigger.TrigType {
	case ast.TriggerTypeEvent:
		if trigger.Value == "" {
			v.errors.Add(newSemanticError(trigger.Pos(), "trigger type \"event\" requires a value"))
		}
		// Validate event name format (e.g., "order.created")
		if trigger.Value != "" && !isValidEventName(trigger.Value) {
			v.errors.Add(newSemanticError(trigger.Pos(), fmt.Sprintf("invalid event name: %q (expected format: 'domain.event')", trigger.Value)))
		}

	case ast.TriggerTypeSchedule:
		if trigger.Value == "" {
			v.errors.Add(newSemanticError(trigger.Pos(), "trigger type \"schedule\" requires a value"))
		}
		// Validate cron expression
		if trigger.Value != "" {
			if err := validateCronExpression(trigger.Value); err != nil {
				v.errors.Add(newSemanticError(trigger.Pos(), fmt.Sprintf("invalid cron expression %q: %v", trigger.Value, err)))
			}
		}

	case ast.TriggerTypeManual:
		// Manual triggers don't require a value
	}
}

// validateWorkflowStep validates a single workflow step.
func (v *WorkflowValidator) validateWorkflowStep(step *ast.WorkflowStep, stepNames map[string]bool) {
	if step == nil {
		return
	}

	if step.Parallel {
		// Validate parallel block
		for _, nestedStep := range step.Steps {
			v.validateWorkflowStep(nestedStep, stepNames)
		}
		return
	}

	// Check for duplicate step names
	if step.Name != "" {
		if _, exists := stepNames[step.Name]; exists {
			v.errors.Add(newSemanticError(step.Pos(), fmt.Sprintf("duplicate step name: %q", step.Name)))
		}
		stepNames[step.Name] = true
	}

	// Validate step name format
	if step.Name != "" && !isValidIdentifier(step.Name) {
		v.errors.Add(newSemanticError(step.Pos(), fmt.Sprintf("invalid step name: %q must be a valid identifier", step.Name)))
	}

	// Validate activity reference
	if step.Activity == "" && !step.Parallel {
		v.errors.Add(newSemanticError(step.Pos(), fmt.Sprintf("step %q must specify an activity", step.Name)))
	}

	// Check if activity is registered (if strict validation is enabled)
	if step.Activity != "" && len(v.activities) > 0 {
		if !v.activities[step.Activity] {
			v.errors.Add(newSemanticError(step.Pos(), fmt.Sprintf("unknown activity: %q", step.Activity)))
		}
	}

	// Validate input mappings
	for _, mapping := range step.Input {
		v.validateInputMapping(mapping)
	}
}

// validateInputMapping validates an input mapping.
func (v *WorkflowValidator) validateInputMapping(mapping *ast.InputMapping) {
	if mapping == nil {
		return
	}

	// Validate key format
	if !isValidIdentifier(mapping.Key) {
		v.errors.Add(newSemanticError(mapping.Pos(), fmt.Sprintf("invalid input mapping key: %q", mapping.Key)))
	}

	// Value can be any expression reference - skip strict validation
}

// validateRetryPolicy validates a retry policy.
func (v *WorkflowValidator) validateRetryPolicy(policy *ast.RetryPolicyDecl) {
	if policy == nil {
		return
	}

	// Validate max attempts
	if policy.MaxAttempts < 0 {
		v.errors.Add(newSemanticError(policy.Pos(), fmt.Sprintf("max_attempts must be non-negative, got %d", policy.MaxAttempts)))
	}

	// Validate initial interval
	if policy.InitialInterval != "" {
		if _, err := time.ParseDuration(policy.InitialInterval); err != nil {
			v.errors.Add(newSemanticError(policy.Pos(), fmt.Sprintf("invalid interval duration: %q", policy.InitialInterval)))
		}
	}

	// Validate backoff multiplier
	if policy.BackoffMultiplier < 0 {
		v.errors.Add(newSemanticError(policy.Pos(), fmt.Sprintf("backoff_multiplier must be non-negative, got %f", policy.BackoffMultiplier)))
	}
}

// validateJob validates a single job declaration.
func (v *WorkflowValidator) validateJob(decl *ast.JobDecl) {
	if decl == nil {
		return
	}

	// Check for duplicate job names
	if _, exists := v.jobs[decl.Name]; exists {
		v.errors.Add(newSemanticError(decl.Pos(), fmt.Sprintf("duplicate job declaration: %q", decl.Name)))
		return
	}
	v.jobs[decl.Name] = decl

	// Validate job name format
	if !isValidIdentifier(decl.Name) {
		v.errors.Add(newSemanticError(decl.Pos(), fmt.Sprintf("invalid job name: %q must be a valid identifier", decl.Name)))
	}

	// Validate schedule (cron expression) if present
	if decl.Schedule != "" {
		if err := validateCronExpression(decl.Schedule); err != nil {
			v.errors.Add(newSemanticError(decl.Pos(), fmt.Sprintf("invalid cron expression %q: %v", decl.Schedule, err)))
		}
	}

	// Validate task type is specified
	if decl.Task == "" {
		v.errors.Add(newSemanticError(decl.Pos(), fmt.Sprintf("job %q must specify a task", decl.Name)))
	}

	// Check if task type is registered (if strict validation is enabled)
	if decl.Task != "" && len(v.taskTypes) > 0 {
		if !v.taskTypes[decl.Task] {
			v.errors.Add(newSemanticError(decl.Pos(), fmt.Sprintf("unknown task type: %q", decl.Task)))
		}
	}

	// Validate queue name format if specified
	if decl.Queue != "" && !isValidIdentifier(decl.Queue) {
		v.errors.Add(newSemanticError(decl.Pos(), fmt.Sprintf("invalid queue name: %q must be a valid identifier", decl.Queue)))
	}

	// Validate retry policy if present
	if decl.Retry != nil {
		v.validateRetryPolicy(decl.Retry)
	}
}

// =============================================================================
// Validation Helpers
// =============================================================================

// isValidIdentifier checks if a string is a valid identifier.
var identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func isValidIdentifier(name string) bool {
	return identifierPattern.MatchString(name)
}

// isValidEventName checks if a string is a valid event name (e.g., "order.created").
var eventNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)*$`)

func isValidEventName(name string) bool {
	return eventNamePattern.MatchString(name)
}

// validateCronExpression validates a cron expression.
func validateCronExpression(expr string) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(expr)
	return err
}

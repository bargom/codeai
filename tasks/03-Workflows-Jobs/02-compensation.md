# Task: Compensation (Rollback) Handling

## Overview
Implement saga-pattern compensation for workflows to ensure data consistency when steps fail.

## Phase
Phase 3: Workflows and Jobs

## Priority
High - Critical for data integrity in workflows.

## Dependencies
- 03-Workflows-Jobs/01-workflow-engine.md

## Description
Create a compensation system that automatically rolls back completed steps when a workflow fails, implementing the saga pattern for distributed transactions.

## Detailed Requirements

### 1. Compensation Manager (internal/modules/workflow/compensation.go)

```go
package workflow

import (
    "context"
    "fmt"
    "log/slog"
)

type CompensationManager struct {
    logger *slog.Logger
}

func NewCompensationManager() *CompensationManager {
    return &CompensationManager{
        logger: slog.Default().With("module", "compensation"),
    }
}

type CompensationRecord struct {
    StepName   string
    Action     *Action
    Data       map[string]any
    ExecutedAt time.Time
}

// ExecuteCompensations runs compensation actions in reverse order
func (m *CompensationManager) ExecuteCompensations(
    ctx context.Context,
    exec *Execution,
    executor *Executor,
    compensations []CompensationRecord,
) error {
    m.logger.Info("starting compensation",
        "execution", exec.ID,
        "steps_to_compensate", len(compensations),
    )

    var errors []error

    // Execute in reverse order (LIFO)
    for i := len(compensations) - 1; i >= 0; i-- {
        comp := compensations[i]

        m.logger.Info("compensating step", "step", comp.StepName)

        // Merge compensation data into variables
        vars := make(map[string]any)
        for k, v := range exec.Variables {
            vars[k] = v
        }
        for k, v := range comp.Data {
            vars[k] = v
        }

        // Create compensation context
        compExec := &Execution{
            ID:        exec.ID,
            Variables: vars,
        }

        _, err := executor.executeAction(ctx, comp.Action, compExec)
        if err != nil {
            m.logger.Error("compensation failed",
                "step", comp.StepName,
                "error", err,
            )
            errors = append(errors, fmt.Errorf("compensate %s: %w", comp.StepName, err))
            // Continue with other compensations
        } else {
            m.logger.Info("compensation successful", "step", comp.StepName)
        }
    }

    if len(errors) > 0 {
        return &CompensationErrors{Errors: errors}
    }

    return nil
}

type CompensationErrors struct {
    Errors []error
}

func (e *CompensationErrors) Error() string {
    return fmt.Sprintf("compensation failed with %d errors", len(e.Errors))
}
```

### 2. Enhanced Executor with Compensation

```go
// internal/modules/workflow/executor_compensation.go
package workflow

func (e *Executor) executeStepsWithCompensation(
    ctx context.Context,
    steps []*Step,
    exec *Execution,
) error {
    var compensations []CompensationRecord

    for _, step := range steps {
        select {
        case <-ctx.Done():
            // Timeout or cancellation - trigger compensation
            e.compensate(ctx, exec, compensations)
            return ctx.Err()
        default:
        }

        exec.CurrentStep = step.Name
        e.stateStore.Save(ctx, exec)

        // Record compensation before executing (if defined)
        if step.Compensate != nil {
            comp := CompensationRecord{
                StepName:   step.Name,
                Action:     step.Compensate,
                Data:       copyMap(exec.Variables),
                ExecutedAt: time.Now(),
            }
            compensations = append(compensations, comp)
        }

        err := e.executeStep(ctx, step, exec)
        if err != nil {
            // Step failed - execute compensations
            e.compensate(ctx, exec, compensations)
            return err
        }

        exec.CompletedSteps = append(exec.CompletedSteps, step.Name)
    }

    return nil
}

func (e *Executor) compensate(
    ctx context.Context,
    exec *Execution,
    compensations []CompensationRecord,
) {
    if len(compensations) == 0 {
        return
    }

    e.logger.Info("initiating rollback",
        "execution", exec.ID,
        "steps", len(compensations),
    )

    manager := NewCompensationManager()
    err := manager.ExecuteCompensations(ctx, exec, e, compensations)

    if err != nil {
        e.logger.Error("rollback failed", "execution", exec.ID, "error", err)
        // Store compensation failure for manual intervention
        exec.Variables["_compensation_error"] = err.Error()
    } else {
        e.logger.Info("rollback completed", "execution", exec.ID)
    }
}

func copyMap(m map[string]any) map[string]any {
    result := make(map[string]any)
    for k, v := range m {
        result[k] = v
    }
    return result
}
```

### 3. Common Compensation Actions

```go
// internal/modules/workflow/compensation_actions.go
package workflow

// BuildDeleteCompensation creates a compensation that deletes a created record
func BuildDeleteCompensation(entity, idField string) *Action {
    return &Action{
        Type:      ActionDelete,
        Target:    entity,
        Operation: "delete",
        Params: map[string]any{
            "id": fmt.Sprintf("{%s}", idField),
        },
    }
}

// BuildRestoreCompensation creates a compensation that restores original values
func BuildRestoreCompensation(entity, idField string, fields []string) *Action {
    params := map[string]any{
        "id": fmt.Sprintf("{%s}", idField),
    }
    for _, field := range fields {
        params[field] = fmt.Sprintf("{_original_%s}", field)
    }

    return &Action{
        Type:      ActionUpdate,
        Target:    entity,
        Operation: "update",
        Params:    params,
    }
}

// BuildIncrementCompensation creates a compensation that reverses a decrement
func BuildIncrementCompensation(entity, idField, field string) *Action {
    return &Action{
        Type:      ActionUpdate,
        Target:    entity,
        Operation: "increment",
        Params: map[string]any{
            "id":    fmt.Sprintf("{%s}", idField),
            "field": field,
            "value": fmt.Sprintf("{_decremented_%s}", field),
        },
    }
}

// BuildRefundCompensation creates a compensation for payment refund
func BuildRefundCompensation(integrationID string) *Action {
    return &Action{
        Type:      ActionCall,
        Target:    integrationID,
        Operation: "refund",
        Params: map[string]any{
            "charge_id": "{payment_charge_id}",
            "amount":    "{payment_amount}",
        },
    }
}
```

### 4. Compensation from CodeAI Syntax

```go
// internal/modules/workflow/codeai_compensation.go
package workflow

import "github.com/codeai/codeai/internal/parser"

// BuildCompensationFromAST creates compensation action from CodeAI step definition
func BuildCompensationFromAST(step *parser.WorkflowStep) *Action {
    if step.Compensate == nil {
        return nil
    }

    return convertAction(step.Compensate)
}

func convertAction(ast *parser.ActionAST) *Action {
    action := &Action{
        Params: make(map[string]any),
    }

    switch ast.Type {
    case "increment":
        action.Type = ActionUpdate
        action.Operation = "increment"
        action.Target = ast.Target
        action.Params["field"] = ast.Field
        action.Params["value"] = ast.Value

    case "decrement":
        action.Type = ActionUpdate
        action.Operation = "decrement"
        action.Target = ast.Target
        action.Params["field"] = ast.Field
        action.Params["value"] = ast.Value

    case "update":
        action.Type = ActionUpdate
        action.Target = ast.Target
        for k, v := range ast.Params {
            action.Params[k] = v
        }

    case "delete":
        action.Type = ActionDelete
        action.Target = ast.Target
        action.Params["id"] = ast.ID

    case "call":
        action.Type = ActionCall
        action.Target = ast.Integration
        action.Operation = ast.Operation
        action.Params = ast.Params
    }

    return action
}
```

## Acceptance Criteria
- [ ] Automatic rollback on step failure
- [ ] Reverse-order compensation execution
- [ ] Continue compensation even if some fail
- [ ] Record compensation results
- [ ] Common compensation builders
- [ ] Integration with CodeAI syntax

## Testing Strategy
- Unit tests for compensation manager
- Integration tests with failing workflows
- Tests for partial compensation failures

## Files to Create
- `internal/modules/workflow/compensation.go`
- `internal/modules/workflow/compensation_actions.go`
- `internal/modules/workflow/executor_compensation.go`
- `internal/modules/workflow/compensation_test.go`

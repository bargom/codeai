# Task: Workflow Engine with State Persistence

## Overview
Implement a workflow execution engine that handles multi-step processes with state persistence, retries, and failure handling.

## Phase
Phase 3: Workflows and Jobs

## Priority
High - Core feature for business process automation.

## Dependencies
- Phase 1 complete
- Phase 2: Database Module, Event System

## Description
Create a workflow engine that executes multi-step processes triggered by events, with durable state persistence, step-level retries, and compensation (rollback) support.

## Detailed Requirements

### 1. Workflow Types (internal/modules/workflow/types.go)

```go
package workflow

import (
    "time"
)

type Workflow struct {
    ID          string
    Description string
    Trigger     string // Event name
    Steps       []*Step
    Timeout     time.Duration
    OnComplete  *Action
    OnFail      *Action
}

type Step struct {
    Name       string
    Type       StepType
    Action     *Action
    Condition  *Condition
    ForEach    *ForEach
    Parallel   []*Step
    Timeout    time.Duration
    Retry      *RetryConfig
    Compensate *Action
    OnSuccess  *Action
    OnFail     *Action
}

type StepType int

const (
    StepAction StepType = iota
    StepCondition
    StepForEach
    StepParallel
    StepWait
)

type Action struct {
    Type       ActionType
    Target     string            // Entity, integration, function name
    Operation  string            // Method/operation name
    Params     map[string]any
    Assign     string            // Variable to assign result to
}

type ActionType int

const (
    ActionQuery ActionType = iota
    ActionUpdate
    ActionDelete
    ActionCall      // Integration call
    ActionEmit      // Event emission
    ActionSend      // Email/notification
    ActionFunction  // Custom function
)

type Condition struct {
    Expression string
    Then       []*Step
    Else       []*Step
}

type ForEach struct {
    Source   string  // Expression for collection
    Item     string  // Variable name for current item
    Index    string  // Variable name for index
    Steps    []*Step
}

type RetryConfig struct {
    MaxAttempts int
    Delay       time.Duration
    Backoff     BackoffType
    MaxDelay    time.Duration
}

type BackoffType int

const (
    BackoffFixed BackoffType = iota
    BackoffExponential
    BackoffLinear
)

type ExecutionStatus string

const (
    StatusPending   ExecutionStatus = "pending"
    StatusRunning   ExecutionStatus = "running"
    StatusCompleted ExecutionStatus = "completed"
    StatusFailed    ExecutionStatus = "failed"
    StatusCancelled ExecutionStatus = "cancelled"
    StatusPaused    ExecutionStatus = "paused"
)
```

### 2. Execution State (internal/modules/workflow/state.go)

```go
package workflow

import (
    "encoding/json"
    "time"
)

type Execution struct {
    ID            string          `json:"id"`
    WorkflowID    string          `json:"workflow_id"`
    Status        ExecutionStatus `json:"status"`
    TriggerData   any             `json:"trigger_data"`
    Variables     map[string]any  `json:"variables"`
    CurrentStep   string          `json:"current_step"`
    StepResults   map[string]any  `json:"step_results"`
    StepStatuses  map[string]StepStatus `json:"step_statuses"`
    CompletedSteps []string       `json:"completed_steps"`
    Error         string          `json:"error,omitempty"`
    StartedAt     time.Time       `json:"started_at"`
    CompletedAt   *time.Time      `json:"completed_at,omitempty"`
    UpdatedAt     time.Time       `json:"updated_at"`
}

type StepStatus struct {
    Status     ExecutionStatus `json:"status"`
    Attempts   int             `json:"attempts"`
    LastError  string          `json:"last_error,omitempty"`
    StartedAt  *time.Time      `json:"started_at,omitempty"`
    CompletedAt *time.Time     `json:"completed_at,omitempty"`
    Result     any             `json:"result,omitempty"`
}

type StateStore interface {
    Save(ctx context.Context, exec *Execution) error
    Load(ctx context.Context, id string) (*Execution, error)
    List(ctx context.Context, workflowID string, status ExecutionStatus) ([]*Execution, error)
    Delete(ctx context.Context, id string) error
}

// PostgresStateStore implements StateStore
type PostgresStateStore struct {
    db database.Module
}

func NewPostgresStateStore(db database.Module) *PostgresStateStore {
    return &PostgresStateStore{db: db}
}

func (s *PostgresStateStore) Save(ctx context.Context, exec *Execution) error {
    exec.UpdatedAt = time.Now()

    varsJSON, _ := json.Marshal(exec.Variables)
    stepResultsJSON, _ := json.Marshal(exec.StepResults)
    stepStatusesJSON, _ := json.Marshal(exec.StepStatuses)
    triggerJSON, _ := json.Marshal(exec.TriggerData)

    _, err := s.db.Execute(ctx, `
        INSERT INTO workflow_executions (
            id, workflow_id, status, trigger_data, variables,
            current_step, step_results, step_statuses, completed_steps,
            error, started_at, completed_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
        ON CONFLICT (id) DO UPDATE SET
            status = EXCLUDED.status,
            variables = EXCLUDED.variables,
            current_step = EXCLUDED.current_step,
            step_results = EXCLUDED.step_results,
            step_statuses = EXCLUDED.step_statuses,
            completed_steps = EXCLUDED.completed_steps,
            error = EXCLUDED.error,
            completed_at = EXCLUDED.completed_at,
            updated_at = EXCLUDED.updated_at
    `, exec.ID, exec.WorkflowID, exec.Status, triggerJSON, varsJSON,
        exec.CurrentStep, stepResultsJSON, stepStatusesJSON, exec.CompletedSteps,
        exec.Error, exec.StartedAt, exec.CompletedAt, exec.UpdatedAt)

    return err
}

func (s *PostgresStateStore) Load(ctx context.Context, id string) (*Execution, error) {
    row, err := s.db.QueryOne(ctx, `
        SELECT id, workflow_id, status, trigger_data, variables,
               current_step, step_results, step_statuses, completed_steps,
               error, started_at, completed_at, updated_at
        FROM workflow_executions WHERE id = $1
    `, id)
    if err != nil {
        return nil, err
    }

    exec := &Execution{}
    // Parse fields...
    return exec, nil
}
```

### 3. Workflow Executor (internal/modules/workflow/executor.go)

```go
package workflow

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "log/slog"
)

type Executor struct {
    workflows  map[string]*Workflow
    stateStore StateStore
    evaluator  *Evaluator
    db         database.Module
    events     event.Module
    integrations integration.Module
    logger     *slog.Logger
}

func NewExecutor(
    stateStore StateStore,
    db database.Module,
    events event.Module,
    integrations integration.Module,
) *Executor {
    return &Executor{
        workflows:    make(map[string]*Workflow),
        stateStore:   stateStore,
        evaluator:    NewEvaluator(),
        db:           db,
        events:       events,
        integrations: integrations,
        logger:       slog.Default().With("module", "workflow"),
    }
}

func (e *Executor) RegisterWorkflow(wf *Workflow) {
    e.workflows[wf.ID] = wf
}

func (e *Executor) Start(ctx context.Context, workflowID string, triggerData any) (string, error) {
    wf, ok := e.workflows[workflowID]
    if !ok {
        return "", fmt.Errorf("workflow not found: %s", workflowID)
    }

    exec := &Execution{
        ID:           uuid.New().String(),
        WorkflowID:   workflowID,
        Status:       StatusRunning,
        TriggerData:  triggerData,
        Variables:    make(map[string]any),
        StepResults:  make(map[string]any),
        StepStatuses: make(map[string]StepStatus),
        StartedAt:    time.Now(),
    }

    // Initialize variables with trigger data
    exec.Variables["trigger"] = triggerData

    // Save initial state
    if err := e.stateStore.Save(ctx, exec); err != nil {
        return "", fmt.Errorf("failed to save execution: %w", err)
    }

    // Execute asynchronously
    go e.execute(context.Background(), wf, exec)

    return exec.ID, nil
}

func (e *Executor) execute(ctx context.Context, wf *Workflow, exec *Execution) {
    defer func() {
        if r := recover(); r != nil {
            e.logger.Error("workflow panic", "workflow", wf.ID, "execution", exec.ID, "panic", r)
            exec.Status = StatusFailed
            exec.Error = fmt.Sprintf("panic: %v", r)
            e.stateStore.Save(ctx, exec)
        }
    }()

    // Apply workflow timeout
    if wf.Timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, wf.Timeout)
        defer cancel()
    }

    // Execute steps
    err := e.executeSteps(ctx, wf.Steps, exec)

    now := time.Now()
    exec.CompletedAt = &now

    if err != nil {
        exec.Status = StatusFailed
        exec.Error = err.Error()

        // Execute OnFail action
        if wf.OnFail != nil {
            e.executeAction(ctx, wf.OnFail, exec)
        }
    } else {
        exec.Status = StatusCompleted

        // Execute OnComplete action
        if wf.OnComplete != nil {
            e.executeAction(ctx, wf.OnComplete, exec)
        }
    }

    e.stateStore.Save(ctx, exec)
    e.logger.Info("workflow completed", "workflow", wf.ID, "execution", exec.ID, "status", exec.Status)
}

func (e *Executor) executeSteps(ctx context.Context, steps []*Step, exec *Execution) error {
    for _, step := range steps {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        exec.CurrentStep = step.Name
        e.stateStore.Save(ctx, exec)

        if err := e.executeStep(ctx, step, exec); err != nil {
            return err
        }

        exec.CompletedSteps = append(exec.CompletedSteps, step.Name)
    }

    return nil
}

func (e *Executor) executeStep(ctx context.Context, step *Step, exec *Execution) error {
    status := StepStatus{
        Status:    StatusRunning,
        StartedAt: timePtr(time.Now()),
    }
    exec.StepStatuses[step.Name] = status

    // Apply step timeout
    if step.Timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, step.Timeout)
        defer cancel()
    }

    var err error
    var result any

    switch step.Type {
    case StepAction:
        result, err = e.executeActionWithRetry(ctx, step, exec)

    case StepCondition:
        err = e.executeCondition(ctx, step.Condition, exec)

    case StepForEach:
        err = e.executeForEach(ctx, step.ForEach, exec)

    case StepParallel:
        err = e.executeParallel(ctx, step.Parallel, exec)

    case StepWait:
        // Wait for duration or signal
    }

    now := time.Now()
    status.CompletedAt = &now

    if err != nil {
        status.Status = StatusFailed
        status.LastError = err.Error()
        exec.StepStatuses[step.Name] = status

        // Execute compensation if defined
        if step.Compensate != nil {
            e.logger.Info("executing compensation", "step", step.Name)
            e.executeAction(ctx, step.Compensate, exec)
        }

        // Execute OnFail
        if step.OnFail != nil {
            e.executeAction(ctx, step.OnFail, exec)
        }

        return err
    }

    status.Status = StatusCompleted
    status.Result = result
    exec.StepStatuses[step.Name] = status
    exec.StepResults[step.Name] = result

    // Execute OnSuccess
    if step.OnSuccess != nil {
        e.executeAction(ctx, step.OnSuccess, exec)
    }

    return nil
}

func (e *Executor) executeActionWithRetry(ctx context.Context, step *Step, exec *Execution) (any, error) {
    retry := step.Retry
    if retry == nil {
        retry = &RetryConfig{MaxAttempts: 1}
    }

    var lastErr error
    delay := retry.Delay

    for attempt := 1; attempt <= retry.MaxAttempts; attempt++ {
        status := exec.StepStatuses[step.Name]
        status.Attempts = attempt
        exec.StepStatuses[step.Name] = status

        result, err := e.executeAction(ctx, step.Action, exec)
        if err == nil {
            return result, nil
        }

        lastErr = err
        e.logger.Warn("step failed", "step", step.Name, "attempt", attempt, "error", err)

        if attempt < retry.MaxAttempts {
            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            case <-time.After(delay):
            }

            // Calculate next delay
            switch retry.Backoff {
            case BackoffExponential:
                delay *= 2
            case BackoffLinear:
                delay += retry.Delay
            }

            if retry.MaxDelay > 0 && delay > retry.MaxDelay {
                delay = retry.MaxDelay
            }
        }
    }

    return nil, lastErr
}

func (e *Executor) executeAction(ctx context.Context, action *Action, exec *Execution) (any, error) {
    switch action.Type {
    case ActionQuery:
        return e.executeQuery(ctx, action, exec)
    case ActionUpdate:
        return e.executeUpdate(ctx, action, exec)
    case ActionCall:
        return e.executeIntegrationCall(ctx, action, exec)
    case ActionEmit:
        return nil, e.executeEmit(ctx, action, exec)
    default:
        return nil, fmt.Errorf("unknown action type: %d", action.Type)
    }
}

func (e *Executor) executeQuery(ctx context.Context, action *Action, exec *Execution) (any, error) {
    // Build and execute query using evaluator
    params := e.evaluator.EvaluateParams(action.Params, exec.Variables)
    result, err := e.db.Query(ctx, action.Operation, params)
    if err != nil {
        return nil, err
    }

    if action.Assign != "" {
        exec.Variables[action.Assign] = result
    }

    return result, nil
}

func (e *Executor) executeIntegrationCall(ctx context.Context, action *Action, exec *Execution) (any, error) {
    params := e.evaluator.EvaluateParams(action.Params, exec.Variables)
    result, err := e.integrations.Call(ctx, action.Target, action.Operation, params)
    if err != nil {
        return nil, err
    }

    if action.Assign != "" {
        exec.Variables[action.Assign] = result
    }

    return result, nil
}

func (e *Executor) executeEmit(ctx context.Context, action *Action, exec *Execution) error {
    payload := e.evaluator.EvaluateParams(action.Params, exec.Variables)
    return e.events.Emit(ctx, action.Target, payload)
}

func timePtr(t time.Time) *time.Time {
    return &t
}
```

### 4. Workflow Module (internal/modules/workflow/module.go)

```go
package workflow

import (
    "context"
)

type WorkflowModule interface {
    Module
    RegisterWorkflow(wf *Workflow) error
    Start(ctx context.Context, workflowID string, trigger any) (string, error)
    Resume(ctx context.Context, executionID string) error
    Cancel(ctx context.Context, executionID string) error
    GetStatus(ctx context.Context, executionID string) (*Execution, error)
}

type workflowModule struct {
    executor *Executor
    events   event.Module
}

func NewWorkflowModule(
    db database.Module,
    events event.Module,
    integrations integration.Module,
) WorkflowModule {
    stateStore := NewPostgresStateStore(db)
    executor := NewExecutor(stateStore, db, events, integrations)

    return &workflowModule{
        executor: executor,
        events:   events,
    }
}

func (m *workflowModule) Name() string { return "workflow" }

func (m *workflowModule) Initialize(config *Config) error {
    return nil
}

func (m *workflowModule) Start(ctx context.Context) error {
    // Subscribe to workflow triggers
    for id, wf := range m.executor.workflows {
        if wf.Trigger != "" {
            m.events.Subscribe(wf.Trigger, func(ctx *ExecutionContext, payload any) error {
                _, err := m.executor.Start(ctx, id, payload)
                return err
            })
        }
    }
    return nil
}

func (m *workflowModule) RegisterWorkflow(wf *Workflow) error {
    m.executor.RegisterWorkflow(wf)
    return nil
}
```

## Acceptance Criteria
- [ ] Execute multi-step workflows
- [ ] State persistence to database
- [ ] Step-level retries with backoff
- [ ] Event-triggered workflows
- [ ] Parallel step execution
- [ ] ForEach loop support
- [ ] Conditional branching
- [ ] Timeout handling

## Testing Strategy
- Unit tests for executor logic
- Integration tests with database
- End-to-end workflow tests

## Files to Create
- `internal/modules/workflow/types.go`
- `internal/modules/workflow/state.go`
- `internal/modules/workflow/executor.go`
- `internal/modules/workflow/evaluator.go`
- `internal/modules/workflow/module.go`
- `internal/modules/workflow/executor_test.go`

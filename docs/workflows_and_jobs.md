# Workflows and Jobs Guide

**Version**: 1.0
**Date**: 2026-01-12
**Status**: Production Ready

---

## Table of Contents

1. [Overview](#1-overview)
2. [Workflow Orchestration (Temporal)](#2-workflow-orchestration-temporal)
3. [Job Scheduling (Asynq)](#3-job-scheduling-asynq)
4. [Event-Driven Patterns](#4-event-driven-patterns)
5. [Best Practices](#5-best-practices)
6. [Monitoring and Observability](#6-monitoring-and-observability)
7. [Examples](#7-examples)

---

## 1. Overview

CodeAI provides two complementary systems for asynchronous work:

| System | Technology | Use Case |
|--------|------------|----------|
| **Workflows** | Temporal | Long-running, multi-step processes with state persistence and compensation |
| **Jobs** | Asynq (Redis) | Background tasks, scheduled jobs, and recurring cron tasks |

### 1.1 When to Use Each

| Scenario | Recommended System |
|----------|-------------------|
| Multi-step business process | Workflow |
| Requires rollback/compensation on failure | Workflow |
| Simple background task | Job |
| Scheduled/recurring task | Job |
| Event-triggered processing | Either (depends on complexity) |
| Long-running (hours/days) | Workflow |
| Short-lived (seconds/minutes) | Job |

### 1.2 Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           CodeAI Runtime                                │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────┐         ┌─────────────────────┐               │
│  │   Workflow Engine   │         │  Scheduler Service  │               │
│  │     (Temporal)      │         │     (Asynq)         │               │
│  │                     │         │                     │               │
│  │  ┌───────────────┐  │         │  ┌───────────────┐  │               │
│  │  │   Workflows   │  │         │  │    Tasks      │  │               │
│  │  │  - Pipeline   │  │         │  │  - AI Agent   │  │               │
│  │  │  - TestSuite  │  │         │  │  - Cleanup    │  │               │
│  │  │  - Saga       │  │         │  │  - Webhook    │  │               │
│  │  └───────────────┘  │         │  └───────────────┘  │               │
│  │                     │         │                     │               │
│  │  ┌───────────────┐  │         │  ┌───────────────┐  │               │
│  │  │  Activities   │  │         │  │   Queues      │  │               │
│  │  │  - Agent      │  │         │  │  - Critical   │  │               │
│  │  │  - Storage    │  │         │  │  - Default    │  │               │
│  │  │  - Notify     │  │         │  │  - Low        │  │               │
│  │  └───────────────┘  │         │  └───────────────┘  │               │
│  └──────────┬──────────┘         └──────────┬──────────┘               │
│             │                               │                          │
│             └───────────┬───────────────────┘                          │
│                         │                                              │
│                         ▼                                              │
│              ┌─────────────────────┐                                   │
│              │     Event Bus       │                                   │
│              │  (Pub/Sub System)   │                                   │
│              └─────────────────────┘                                   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Workflow Orchestration (Temporal)

### 2.1 Core Concepts

#### Workflow
A **workflow** is a durable function that orchestrates multiple activities. It survives process restarts and can run for extended periods (hours, days, or longer).

#### Activity
An **activity** is a single unit of work within a workflow. Activities are the building blocks that perform actual business logic.

#### Saga Pattern
The **saga pattern** provides automatic compensation (rollback) when workflow steps fail.

### 2.2 Workflow Definition

#### Basic Workflow Structure

```go
// internal/workflow/definitions/my_workflow.go
package definitions

import (
    "time"
    "go.temporal.io/sdk/workflow"
    "go.temporal.io/sdk/temporal"
)

// MyWorkflowInput defines the input for the workflow
type MyWorkflowInput struct {
    WorkflowID string            `json:"workflowId"`
    Data       map[string]string `json:"data"`
    Timeout    time.Duration     `json:"timeout,omitempty"`
}

// MyWorkflowOutput defines the output of the workflow
type MyWorkflowOutput struct {
    WorkflowID string        `json:"workflowId"`
    Status     Status        `json:"status"`
    Results    []StepResult  `json:"results"`
    Error      string        `json:"error,omitempty"`
}

// MyWorkflow orchestrates the business process
func MyWorkflow(ctx workflow.Context, input MyWorkflowInput) (MyWorkflowOutput, error) {
    logger := workflow.GetLogger(ctx)

    output := MyWorkflowOutput{
        WorkflowID: input.WorkflowID,
        Status:     StatusRunning,
    }

    // Configure activity options
    activityOptions := workflow.ActivityOptions{
        StartToCloseTimeout: 10 * time.Minute,
        RetryPolicy: &temporal.RetryPolicy{
            InitialInterval:    time.Second,
            BackoffCoefficient: 2.0,
            MaximumInterval:    60 * time.Second,
            MaximumAttempts:    3,
        },
    }
    ctx = workflow.WithActivityOptions(ctx, activityOptions)

    // Step 1: Validate
    var validationResult ValidationResult
    err := workflow.ExecuteActivity(ctx, ValidateActivity, input).Get(ctx, &validationResult)
    if err != nil {
        output.Status = StatusFailed
        output.Error = err.Error()
        return output, err
    }

    // Step 2: Process
    var processResult ProcessResult
    err = workflow.ExecuteActivity(ctx, ProcessActivity, input).Get(ctx, &processResult)
    if err != nil {
        output.Status = StatusFailed
        output.Error = err.Error()
        return output, err
    }

    output.Status = StatusCompleted
    return output, nil
}
```

### 2.3 Activity Implementation

```go
// internal/workflow/activities/my_activities.go
package activities

import (
    "context"
    "go.temporal.io/sdk/activity"
)

// MyActivities holds activity implementations
type MyActivities struct {
    // Inject dependencies
    db Database
}

// NewMyActivities creates a new activity holder
func NewMyActivities(db Database) *MyActivities {
    return &MyActivities{db: db}
}

// Validate performs input validation
func (a *MyActivities) Validate(ctx context.Context, input ValidateInput) (ValidateOutput, error) {
    info := activity.GetInfo(ctx)

    // Send heartbeat for long-running activities
    activity.RecordHeartbeat(ctx, "validating...")

    // Perform validation logic
    if input.Data == nil {
        return ValidateOutput{Valid: false}, fmt.Errorf("data is required")
    }

    return ValidateOutput{Valid: true}, nil
}

// Process performs the main processing
func (a *MyActivities) Process(ctx context.Context, input ProcessInput) (ProcessOutput, error) {
    info := activity.GetInfo(ctx)

    // Check if this is a retry
    if info.Attempt > 1 {
        activity.GetLogger(ctx).Info("Retry attempt", "attempt", info.Attempt)
    }

    // Perform processing
    result, err := a.db.Process(ctx, input.Data)
    if err != nil {
        return ProcessOutput{}, err
    }

    return ProcessOutput{Result: result}, nil
}
```

### 2.4 Saga Pattern and Compensation

The saga pattern ensures that when a workflow step fails, all previously completed steps are automatically compensated (rolled back).

#### Compensation Manager

```go
// Using the built-in compensation manager
package definitions

import (
    "github.com/bargom/codeai/internal/workflow/compensation"
)

func OrderWorkflow(ctx workflow.Context, input OrderInput) (OrderOutput, error) {
    logger := workflow.GetLogger(ctx)
    cm := compensation.NewCompensationManager(ctx)

    // Step 1: Reserve Inventory
    cm.RegisterCompensation(compensation.CompensationStep{
        ActivityName: "reserve-inventory",
        CompensateFn: func(ctx workflow.Context, input interface{}) error {
            return workflow.ExecuteActivity(ctx, ReleaseInventoryActivity, input).Get(ctx, nil)
        },
        Input: ReleaseInventoryInput{OrderID: input.OrderID},
        Timeout: 30 * time.Second,
    })

    var reserveResult ReserveResult
    err := workflow.ExecuteActivity(ctx, ReserveInventoryActivity, input).Get(ctx, &reserveResult)
    if err != nil {
        return OrderOutput{}, err
    }
    cm.RecordExecution("reserve-inventory")

    // Step 2: Charge Payment
    cm.RegisterCompensation(compensation.CompensationStep{
        ActivityName: "charge-payment",
        CompensateFn: func(ctx workflow.Context, input interface{}) error {
            return workflow.ExecuteActivity(ctx, RefundPaymentActivity, input).Get(ctx, nil)
        },
        Input: RefundInput{TransactionID: reserveResult.TransactionID},
        Timeout: 60 * time.Second,
    })

    var chargeResult ChargeResult
    err = workflow.ExecuteActivity(ctx, ChargePaymentActivity, ChargeInput{
        OrderID: input.OrderID,
        Amount:  input.TotalAmount,
    }).Get(ctx, &chargeResult)

    if err != nil {
        // Payment failed - compensate by releasing inventory
        logger.Error("Payment failed, starting compensation", "error", err)
        if compErr := cm.Compensate(ctx); compErr != nil {
            return OrderOutput{}, fmt.Errorf("payment failed: %w (compensation failed: %v)", err, compErr)
        }
        return OrderOutput{Status: "cancelled", Error: err.Error()}, err
    }
    cm.RecordExecution("charge-payment")

    // Step 3: Ship Order (non-compensatable, so comes last)
    err = workflow.ExecuteActivity(ctx, ShipOrderActivity, ShipInput{
        OrderID: input.OrderID,
    }).Get(ctx, nil)

    if err != nil {
        // Shipping failed - compensate all previous steps
        logger.Error("Shipping failed, starting compensation", "error", err)
        if compErr := cm.Compensate(ctx); compErr != nil {
            return OrderOutput{}, fmt.Errorf("shipping failed: %w (compensation failed: %v)", err, compErr)
        }
        return OrderOutput{Status: "cancelled", Error: err.Error()}, err
    }

    return OrderOutput{Status: "completed"}, nil
}
```

#### Using SagaBuilder

```go
// Fluent saga builder for cleaner code
import "github.com/bargom/codeai/internal/workflow/patterns"

func OrderWorkflowWithBuilder(ctx workflow.Context, input OrderInput) (patterns.SagaOutput, error) {
    saga := patterns.NewSagaBuilder(input.OrderID).
        WithTimeout(30 * time.Minute).
        WithMetadata("orderType", input.Type).
        AddActivityStep(
            "reserve-inventory",
            ReserveInventoryActivity,
            ReserveInput{OrderID: input.OrderID},
            func(ctx workflow.Context, input interface{}) error {
                return workflow.ExecuteActivity(ctx, ReleaseInventoryActivity, input).Get(ctx, nil)
            },
        ).
        AddActivityStep(
            "charge-payment",
            ChargePaymentActivity,
            ChargeInput{OrderID: input.OrderID, Amount: input.TotalAmount},
            func(ctx workflow.Context, input interface{}) error {
                return workflow.ExecuteActivity(ctx, RefundPaymentActivity, input).Get(ctx, nil)
            },
        ).
        AddNonCriticalStep(
            "send-notification",
            SendNotificationActivity,
            NotifyInput{OrderID: input.OrderID, Type: "order_placed"},
        ).
        Build()

    return patterns.SagaWorkflow(ctx, saga)
}
```

### 2.5 Parallel Execution

```go
func ParallelProcessingWorkflow(ctx workflow.Context, input PipelineInput) (PipelineOutput, error) {
    var futures []workflow.Future
    results := make([]AgentResult, len(input.Agents))

    // Start all agents in parallel
    for i, agent := range input.Agents {
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
            if firstError == nil {
                firstError = err
            }
        } else {
            results[i] = response.Result
        }
    }

    return PipelineOutput{Results: results}, firstError
}
```

### 2.6 Workflow Signals and Queries

```go
func InteractiveWorkflow(ctx workflow.Context, input WorkflowInput) (WorkflowOutput, error) {
    logger := workflow.GetLogger(ctx)

    // Set up signal channels
    pauseSignal := workflow.GetSignalChannel(ctx, "pause")
    resumeSignal := workflow.GetSignalChannel(ctx, "resume")
    cancelSignal := workflow.GetSignalChannel(ctx, "cancel")

    var isPaused bool
    var isCancelled bool

    // Register query handler for status
    err := workflow.SetQueryHandler(ctx, "status", func() (string, error) {
        if isCancelled {
            return "cancelled", nil
        }
        if isPaused {
            return "paused", nil
        }
        return "running", nil
    })
    if err != nil {
        return WorkflowOutput{}, err
    }

    selector := workflow.NewSelector(ctx)

    selector.AddReceive(pauseSignal, func(c workflow.ReceiveChannel, more bool) {
        var data map[string]string
        c.Receive(ctx, &data)
        logger.Info("Workflow paused")
        isPaused = true
    })

    selector.AddReceive(resumeSignal, func(c workflow.ReceiveChannel, more bool) {
        var data map[string]string
        c.Receive(ctx, &data)
        logger.Info("Workflow resumed")
        isPaused = false
    })

    selector.AddReceive(cancelSignal, func(c workflow.ReceiveChannel, more bool) {
        var data map[string]string
        c.Receive(ctx, &data)
        logger.Info("Workflow cancelled")
        isCancelled = true
    })

    // Main workflow loop
    for _, step := range input.Steps {
        // Check for signals
        for selector.HasPending() {
            selector.Select(ctx)
        }

        if isCancelled {
            return WorkflowOutput{Status: "cancelled"}, nil
        }

        // Wait while paused
        for isPaused && !isCancelled {
            selector.Select(ctx)
        }

        // Execute step
        err := workflow.ExecuteActivity(ctx, step.Activity, step.Input).Get(ctx, nil)
        if err != nil {
            return WorkflowOutput{Status: "failed", Error: err.Error()}, err
        }
    }

    return WorkflowOutput{Status: "completed"}, nil
}
```

### 2.7 Registering Workflows

```go
// main.go or initialization code
package main

import (
    "github.com/bargom/codeai/internal/workflow/engine"
    "github.com/bargom/codeai/internal/workflow/definitions"
    "github.com/bargom/codeai/internal/workflow/activities"
)

func main() {
    cfg := engine.Config{
        TemporalHostPort:       "localhost:7233",
        Namespace:              "codeai",
        TaskQueue:              "codeai-tasks",
        MaxConcurrentWorkflows: 100,
        MaxConcurrentActivities: 50,
        DefaultTimeout:         30 * time.Minute,
    }

    eng, err := engine.NewEngine(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // Register workflows
    eng.RegisterWorkflow(definitions.AIAgentPipelineWorkflow)
    eng.RegisterWorkflow(definitions.TestSuiteWorkflow)
    eng.RegisterWorkflow(patterns.SagaWorkflow)

    // Register activities
    agentActivities := activities.NewAgentActivities(agentExecutor)
    eng.RegisterActivity(agentActivities.ExecuteAgent)

    storageActivities := activities.NewStorageActivities(storageClient)
    eng.RegisterActivity(storageActivities.Store)

    // Start the engine
    if err := eng.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // Execute a workflow
    run, err := eng.ExecuteWorkflow(ctx, "workflow-123", definitions.AIAgentPipelineWorkflow, input)
    if err != nil {
        log.Fatal(err)
    }

    // Get result
    var output definitions.PipelineOutput
    if err := run.Get(ctx, &output); err != nil {
        log.Fatal(err)
    }
}
```

### 2.8 Testing Workflows

```go
// internal/workflow/definitions/my_workflow_test.go
package definitions_test

import (
    "testing"
    "go.temporal.io/sdk/testsuite"
)

func TestMyWorkflow(t *testing.T) {
    testSuite := &testsuite.WorkflowTestSuite{}
    env := testSuite.NewTestWorkflowEnvironment()

    // Mock activities
    env.OnActivity(ValidateActivity, mock.Anything, mock.Anything).Return(
        ValidateOutput{Valid: true}, nil,
    )
    env.OnActivity(ProcessActivity, mock.Anything, mock.Anything).Return(
        ProcessOutput{Result: "success"}, nil,
    )

    // Execute workflow
    env.ExecuteWorkflow(MyWorkflow, MyWorkflowInput{
        WorkflowID: "test-123",
        Data:       map[string]string{"key": "value"},
    })

    // Verify completion
    require.True(t, env.IsWorkflowCompleted())
    require.NoError(t, env.GetWorkflowError())

    // Check output
    var output MyWorkflowOutput
    require.NoError(t, env.GetWorkflowResult(&output))
    require.Equal(t, StatusCompleted, output.Status)
}

func TestMyWorkflowCompensation(t *testing.T) {
    testSuite := &testsuite.WorkflowTestSuite{}
    env := testSuite.NewTestWorkflowEnvironment()

    // Mock first activity succeeding
    env.OnActivity(ReserveInventoryActivity, mock.Anything, mock.Anything).Return(
        ReserveResult{TransactionID: "txn-123"}, nil,
    )

    // Mock second activity failing
    env.OnActivity(ChargePaymentActivity, mock.Anything, mock.Anything).Return(
        ChargeResult{}, errors.New("payment declined"),
    )

    // Mock compensation activity
    env.OnActivity(ReleaseInventoryActivity, mock.Anything, mock.Anything).Return(nil)

    env.ExecuteWorkflow(OrderWorkflow, OrderInput{OrderID: "order-123"})

    require.True(t, env.IsWorkflowCompleted())

    // Verify compensation was called
    env.AssertExpectations(t)
}
```

---

## 3. Job Scheduling (Asynq)

### 3.1 Core Concepts

| Concept | Description |
|---------|-------------|
| **Task** | A unit of work with type and payload |
| **Queue** | Priority-based task queue (critical, default, low) |
| **Handler** | Function that processes a task |
| **Scheduler** | Cron-based recurring task scheduler |

### 3.2 Task Definition

```go
// internal/scheduler/tasks/report_task.go
package tasks

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    "github.com/hibiken/asynq"
)

// Task type constant
const TypeDailyReport = "report:daily"

// DailyReportPayload defines the task payload
type DailyReportPayload struct {
    ReportType   string    `json:"report_type"`
    StartDate    time.Time `json:"start_date"`
    EndDate      time.Time `json:"end_date"`
    Recipients   []string  `json:"recipients"`
    Format       string    `json:"format"` // pdf, csv, html
    IncludeCharts bool     `json:"include_charts"`
}

// DailyReportResult defines the task result
type DailyReportResult struct {
    ReportID    string    `json:"report_id"`
    FileURL     string    `json:"file_url"`
    GeneratedAt time.Time `json:"generated_at"`
    RecordCount int       `json:"record_count"`
}

// DailyReportHandler handles daily report generation
type DailyReportHandler struct {
    db          Database
    storage     Storage
    emailClient EmailClient
}

// NewDailyReportHandler creates a new handler
func NewDailyReportHandler(db Database, storage Storage, email EmailClient) *DailyReportHandler {
    return &DailyReportHandler{
        db:          db,
        storage:     storage,
        emailClient: email,
    }
}

// ProcessTask handles the report generation
func (h *DailyReportHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
    var payload DailyReportPayload
    if err := json.Unmarshal(t.Payload(), &payload); err != nil {
        return fmt.Errorf("unmarshal payload: %w", err)
    }

    // Fetch data
    data, err := h.db.GetReportData(ctx, payload.StartDate, payload.EndDate)
    if err != nil {
        return fmt.Errorf("fetch data: %w", err)
    }

    // Generate report
    report, err := h.generateReport(payload, data)
    if err != nil {
        return fmt.Errorf("generate report: %w", err)
    }

    // Upload to storage
    url, err := h.storage.Upload(ctx, report)
    if err != nil {
        return fmt.Errorf("upload report: %w", err)
    }

    // Send email notification
    for _, recipient := range payload.Recipients {
        if err := h.emailClient.Send(ctx, recipient, "Daily Report", url); err != nil {
            // Log but don't fail the task
            log.Printf("failed to send email to %s: %v", recipient, err)
        }
    }

    return nil
}

func (h *DailyReportHandler) generateReport(payload DailyReportPayload, data []ReportRow) ([]byte, error) {
    // Report generation logic
    return nil, nil
}
```

### 3.3 Queue Configuration

```go
// internal/scheduler/queue/config.go
package queue

const (
    QueueCritical = "critical"  // Priority 6
    QueueDefault  = "default"   // Priority 3
    QueueLow      = "low"       // Priority 1
)

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
    return Config{
        RedisAddr:   "localhost:6379",
        RedisDB:     0,
        Concurrency: 10,
        Queues: map[string]int{
            QueueCritical: 6,  // 60% of workers
            QueueDefault:  3,  // 30% of workers
            QueueLow:      1,  // 10% of workers
        },
        MaxRetry:        3,
        ShutdownTimeout: 30 * time.Second,
    }
}
```

### 3.4 Creating and Enqueueing Tasks

```go
// Using the scheduler service
service := scheduler.NewSchedulerService(queueManager, jobRepo, eventBus)

// Immediate execution
jobID, err := service.SubmitJob(ctx, scheduler.JobRequest{
    TaskType: tasks.TypeDailyReport,
    Payload: tasks.DailyReportPayload{
        ReportType: "sales",
        StartDate:  time.Now().AddDate(0, 0, -1),
        EndDate:    time.Now(),
        Recipients: []string{"manager@company.com"},
        Format:     "pdf",
    },
    Queue:      queue.QueueDefault,
    MaxRetries: 3,
    Timeout:    10 * time.Minute,
})

// Scheduled execution (future time)
jobID, err := service.ScheduleJob(ctx, scheduler.JobRequest{
    TaskType: tasks.TypeDailyReport,
    Payload:  payload,
}, time.Now().Add(2 * time.Hour))

// Recurring execution (cron)
jobID, err := service.CreateRecurringJob(ctx, scheduler.JobRequest{
    TaskType: tasks.TypeDailyReport,
    Payload:  payload,
}, "0 6 * * *") // Every day at 6 AM
```

### 3.5 Cron Schedule Syntax

CodeAI uses standard cron syntax with optional seconds:

```
┌───────────── minute (0 - 59)
│ ┌───────────── hour (0 - 23)
│ │ ┌───────────── day of month (1 - 31)
│ │ │ ┌───────────── month (1 - 12)
│ │ │ │ ┌───────────── day of week (0 - 6) (Sunday = 0)
│ │ │ │ │
* * * * *
```

| Expression | Description |
|------------|-------------|
| `0 6 * * *` | Every day at 6:00 AM |
| `*/15 * * * *` | Every 15 minutes |
| `0 0 * * 0` | Every Sunday at midnight |
| `0 9-17 * * 1-5` | Every hour 9 AM to 5 PM, Monday-Friday |
| `0 0 1 * *` | First day of every month at midnight |
| `@hourly` | Every hour |
| `@daily` | Every day at midnight |
| `@weekly` | Every Sunday at midnight |
| `@monthly` | First day of month at midnight |

### 3.6 Job Priorities

```go
// Critical: Payment processing, alerts
task, _ := queue.NewTask(tasks.TypePaymentProcess, payload)
task.WithQueue(queue.QueueCritical).WithMaxRetry(5)

// Default: Report generation, data processing
task, _ := queue.NewTask(tasks.TypeDailyReport, payload)
task.WithQueue(queue.QueueDefault).WithMaxRetry(3)

// Low: Cleanup, analytics, non-urgent tasks
task, _ := queue.NewTask(tasks.TypeCleanup, payload)
task.WithQueue(queue.QueueLow).WithMaxRetry(1)
```

### 3.7 Retry Policies

```go
// Using the Task fluent API
task, _ := queue.NewTask(taskType, payload)
task.
    WithQueue(queue.QueueDefault).
    WithMaxRetry(5).                    // Max 5 retry attempts
    WithTimeout(10 * time.Minute).      // Task timeout
    WithDeadline(time.Now().Add(1*time.Hour)). // Must complete by this time
    WithRetention(24 * time.Hour).      // Keep completed task for 24h
    WithUnique("report-2024-01-12", 1*time.Hour) // Prevent duplicates

// Default retry policy with exponential backoff
// Attempt 1: immediate
// Attempt 2: 10 seconds
// Attempt 3: 20 seconds
// Attempt 4: 40 seconds
// Attempt 5: 80 seconds (capped at MaxDelay)
```

### 3.8 Job Payload Structure

```go
// Generic payload pattern
type JobPayload struct {
    // Required fields
    Type      string `json:"type"`
    RequestID string `json:"request_id"`

    // Task-specific data
    Data json.RawMessage `json:"data"`

    // Metadata
    CreatedAt  time.Time         `json:"created_at"`
    CreatedBy  string            `json:"created_by,omitempty"`
    Priority   int               `json:"priority,omitempty"`
    Tags       []string          `json:"tags,omitempty"`
    Metadata   map[string]string `json:"metadata,omitempty"`
}

// Type-specific payloads
type AIAgentPayload struct {
    AgentType  string            `json:"agent_type"`
    Input      json.RawMessage   `json:"input"`
    Config     map[string]string `json:"config"`
    Timeout    time.Duration     `json:"timeout"`
}

type CleanupPayload struct {
    Type       string        `json:"type"`        // jobs, logs, temp
    OlderThan  time.Duration `json:"older_than"`
    BatchSize  int           `json:"batch_size"`
    DryRun     bool          `json:"dry_run"`
}

type WebhookPayload struct {
    URL     string            `json:"url"`
    Method  string            `json:"method"`
    Headers map[string]string `json:"headers"`
    Body    json.RawMessage   `json:"body"`
    Timeout time.Duration     `json:"timeout"`
}
```

### 3.9 Registering Handlers

```go
// main.go
func main() {
    cfg := queue.DefaultConfig()
    manager, err := queue.NewManager(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // Register handlers
    reportHandler := tasks.NewDailyReportHandler(db, storage, emailClient)
    manager.RegisterHandler(tasks.TypeDailyReport, reportHandler.ProcessTask)

    cleanupHandler := tasks.NewCleanupHandler()
    manager.RegisterHandler(tasks.TypeCleanup, cleanupHandler.ProcessTask)

    webhookHandler := tasks.NewWebhookHandler(httpClient)
    manager.RegisterHandler(tasks.TypeWebhook, webhookHandler.ProcessTask)

    // Start processing
    if err := manager.Start(); err != nil {
        log.Fatal(err)
    }

    // Graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
    <-sigChan

    if err := manager.Stop(); err != nil {
        log.Printf("error stopping manager: %v", err)
    }
}
```

### 3.10 Monitoring Job Execution

```go
// Get job status
status, err := service.GetJobStatus(ctx, jobID)
fmt.Printf("Job %s: %s\n", jobID, status.Status)

// List jobs with filters
jobs, total, err := service.ListJobs(ctx, repository.JobFilter{
    Status:    []repository.JobStatus{repository.JobStatusRunning},
    TaskTypes: []string{tasks.TypeDailyReport},
    Limit:     10,
    OrderBy:   "created_at",
    OrderDirection: "DESC",
})

// Get queue statistics
stats, err := service.GetQueueStats(ctx)
for queueName, queueStats := range stats {
    fmt.Printf("Queue %s: pending=%d, active=%d, completed=%d\n",
        queueName, queueStats.Pending, queueStats.Active, queueStats.Completed)
}
```

---

## 4. Event-Driven Patterns

### 4.1 Event Types

```go
// internal/event/bus/types.go
package bus

// Workflow lifecycle events
const (
    EventWorkflowStarted   EventType = "workflow.started"
    EventWorkflowCompleted EventType = "workflow.completed"
    EventWorkflowFailed    EventType = "workflow.failed"
)

// Job lifecycle events
const (
    EventJobEnqueued  EventType = "job.enqueued"
    EventJobStarted   EventType = "job.started"
    EventJobCompleted EventType = "job.completed"
    EventJobFailed    EventType = "job.failed"
)

// Business events
const (
    EventAgentExecuted      EventType = "agent.executed"
    EventTestSuiteCompleted EventType = "test.suite.completed"
    EventWebhookTriggered   EventType = "webhook.triggered"
)
```

### 4.2 Event Structure

```go
type Event struct {
    ID        string                 `json:"id"`        // Unique event ID
    Type      EventType              `json:"type"`      // Event type
    Source    string                 `json:"source"`    // Event source (service name)
    Timestamp time.Time              `json:"timestamp"` // When event occurred
    Data      map[string]interface{} `json:"data"`      // Event payload
    Metadata  map[string]string      `json:"metadata"`  // Additional metadata
}
```

### 4.3 Publishing Events

```go
// Using the event dispatcher
dispatcher := dispatcher.NewDispatcher(eventBus,
    dispatcher.WithRepository(eventRepo),
    dispatcher.WithLogger(logger),
)

// Synchronous publish
event := bus.Event{
    ID:        uuid.New().String(),
    Type:      bus.EventWorkflowCompleted,
    Source:    "workflow-engine",
    Timestamp: time.Now(),
    Data: map[string]interface{}{
        "workflowID": "wf-123",
        "status":     "completed",
        "duration":   time.Since(startTime).Seconds(),
    },
}

if err := dispatcher.Dispatch(ctx, event); err != nil {
    log.Printf("failed to dispatch event: %v", err)
}

// Asynchronous publish (non-blocking)
dispatcher.DispatchAsync(ctx, event)
```

### 4.4 Subscribing to Events

```go
// Function subscriber
dispatcher.Subscribe(bus.EventWorkflowCompleted, bus.SubscriberFunc(
    func(ctx context.Context, event bus.Event) error {
        workflowID := event.Data["workflowID"].(string)
        log.Printf("Workflow %s completed", workflowID)
        return nil
    },
))

// Struct subscriber
type MetricsSubscriber struct {
    metrics MetricsClient
}

func (s *MetricsSubscriber) Handle(ctx context.Context, event bus.Event) error {
    switch event.Type {
    case bus.EventWorkflowCompleted:
        s.metrics.Increment("workflows.completed")
    case bus.EventWorkflowFailed:
        s.metrics.Increment("workflows.failed")
    case bus.EventJobCompleted:
        duration := event.Data["duration"].(float64)
        s.metrics.Histogram("jobs.duration", duration)
    }
    return nil
}

metricsSubscriber := &MetricsSubscriber{metrics: metricsClient}
dispatcher.Subscribe(bus.EventWorkflowCompleted, metricsSubscriber)
dispatcher.Subscribe(bus.EventWorkflowFailed, metricsSubscriber)
dispatcher.Subscribe(bus.EventJobCompleted, metricsSubscriber)
```

### 4.5 Event-Triggered Workflows

```go
// Subscriber that triggers a workflow on event
type WorkflowTriggerSubscriber struct {
    engine *engine.Engine
}

func (s *WorkflowTriggerSubscriber) Handle(ctx context.Context, event bus.Event) error {
    if event.Type == bus.EventAgentExecuted {
        // Trigger analysis workflow when agent completes
        agentType := event.Data["agentType"].(string)
        if agentType == "code-analysis" {
            input := definitions.AnalysisInput{
                WorkflowID: uuid.New().String(),
                SourceData: event.Data,
            }
            _, err := s.engine.ExecuteWorkflow(ctx, input.WorkflowID,
                definitions.FollowUpAnalysisWorkflow, input)
            return err
        }
    }
    return nil
}
```

### 4.6 Event Schema Evolution

```go
// Version events for backward compatibility
type EventV1 struct {
    ID   string `json:"id"`
    Data string `json:"data"`
}

type EventV2 struct {
    ID       string            `json:"id"`
    Data     json.RawMessage   `json:"data"`
    Metadata map[string]string `json:"metadata"`
    Version  int               `json:"version"`
}

// Handler that supports multiple versions
func HandleEvent(ctx context.Context, event bus.Event) error {
    version := 1
    if v, ok := event.Metadata["version"]; ok {
        version, _ = strconv.Atoi(v)
    }

    switch version {
    case 1:
        return handleV1(ctx, event)
    case 2:
        return handleV2(ctx, event)
    default:
        return fmt.Errorf("unsupported event version: %d", version)
    }
}
```

### 4.7 Async Event Handlers

```go
// Configure event bus for async processing
cfg := bus.Config{
    AsyncBufferSize: 10000,  // Buffer up to 10k events
    WorkerPoolSize:  20,     // 20 concurrent workers
}

eventBus := bus.NewEventBusWithConfig(logger, cfg)

// Events published via PublishAsync are processed by worker pool
eventBus.PublishAsync(ctx, event)
```

---

## 5. Best Practices

### 5.1 Idempotent Activities

Activities should be idempotent - safe to retry without side effects.

```go
// BAD: Not idempotent
func (a *Activities) CreateOrder(ctx context.Context, input CreateOrderInput) error {
    // Creates duplicate orders on retry
    return a.db.Insert(ctx, "orders", input)
}

// GOOD: Idempotent with unique constraint
func (a *Activities) CreateOrder(ctx context.Context, input CreateOrderInput) error {
    // Check if order already exists
    existing, err := a.db.GetByID(ctx, "orders", input.OrderID)
    if err == nil && existing != nil {
        return nil // Already created, idempotent success
    }

    // Use idempotency key or unique constraint
    return a.db.InsertWithKey(ctx, "orders", input.OrderID, input)
}

// GOOD: Upsert pattern
func (a *Activities) UpdateInventory(ctx context.Context, input UpdateInput) error {
    return a.db.Upsert(ctx, "inventory", input.ProductID, input.Quantity)
}
```

### 5.2 Timeout Recommendations

| Activity Type | Recommended Timeout |
|---------------|---------------------|
| Database query | 30 seconds |
| External API call | 30-60 seconds |
| File upload | 5-10 minutes |
| Report generation | 10-30 minutes |
| AI agent execution | 10-60 minutes |
| Batch processing | 1-4 hours |

```go
// Set appropriate timeouts per activity
fastActivityOptions := workflow.ActivityOptions{
    StartToCloseTimeout: 30 * time.Second,
}

slowActivityOptions := workflow.ActivityOptions{
    StartToCloseTimeout: 30 * time.Minute,
    HeartbeatTimeout:    1 * time.Minute, // For long-running activities
}
```

### 5.3 Resource Cleanup in Compensation

```go
// Proper compensation with resource cleanup
cm.RegisterCompensation(compensation.CompensationStep{
    ActivityName: "create-resources",
    CompensateFn: func(ctx workflow.Context, input interface{}) error {
        rollbackInput := input.(RollbackInput)

        // Clean up in reverse order of creation
        errors := make([]error, 0)

        // 1. Delete created files
        if err := workflow.ExecuteActivity(ctx, DeleteFilesActivity,
            rollbackInput.FileIDs).Get(ctx, nil); err != nil {
            errors = append(errors, fmt.Errorf("delete files: %w", err))
        }

        // 2. Release reserved resources
        if err := workflow.ExecuteActivity(ctx, ReleaseResourcesActivity,
            rollbackInput.ResourceIDs).Get(ctx, nil); err != nil {
            errors = append(errors, fmt.Errorf("release resources: %w", err))
        }

        // 3. Revert database changes
        if err := workflow.ExecuteActivity(ctx, RevertDatabaseActivity,
            rollbackInput.TransactionID).Get(ctx, nil); err != nil {
            errors = append(errors, fmt.Errorf("revert database: %w", err))
        }

        if len(errors) > 0 {
            return fmt.Errorf("compensation errors: %v", errors)
        }
        return nil
    },
    Input: RollbackInput{
        FileIDs:       createdFileIDs,
        ResourceIDs:   reservedResourceIDs,
        TransactionID: txnID,
    },
    Timeout: 5 * time.Minute,
})
```

### 5.4 Workflow Versioning

```go
// Use GetVersion for workflow changes
func MyWorkflow(ctx workflow.Context, input Input) (Output, error) {
    v := workflow.GetVersion(ctx, "new-validation-step", workflow.DefaultVersion, 1)

    if v == workflow.DefaultVersion {
        // Old logic for existing executions
        return oldValidation(ctx, input)
    }

    // New logic for new executions
    return newValidation(ctx, input)
}
```

### 5.5 Job Deduplication

```go
// Prevent duplicate job execution
task, _ := queue.NewTask(taskType, payload)
task.WithUnique(
    fmt.Sprintf("report-%s-%s", reportType, date.Format("2006-01-02")),
    24 * time.Hour, // Unique for 24 hours
)

// Or use idempotency at the handler level
func (h *Handler) ProcessTask(ctx context.Context, t *asynq.Task) error {
    var payload Payload
    json.Unmarshal(t.Payload(), &payload)

    // Check if already processed
    processed, err := h.cache.Get(ctx, fmt.Sprintf("processed:%s", payload.RequestID))
    if err == nil && processed {
        return nil // Already processed, skip
    }

    // Process task
    if err := h.doWork(ctx, payload); err != nil {
        return err
    }

    // Mark as processed
    h.cache.Set(ctx, fmt.Sprintf("processed:%s", payload.RequestID), true, 24*time.Hour)
    return nil
}
```

### 5.6 Error Handling

```go
// Distinguish retriable vs non-retriable errors
type NonRetriableError struct {
    Cause error
}

func (e *NonRetriableError) Error() string {
    return e.Cause.Error()
}

func (h *Handler) ProcessTask(ctx context.Context, t *asynq.Task) error {
    var payload Payload
    if err := json.Unmarshal(t.Payload(), &payload); err != nil {
        // Invalid payload - don't retry
        return &NonRetriableError{Cause: err}
    }

    result, err := h.externalAPI.Call(ctx, payload)
    if err != nil {
        if isRateLimitError(err) {
            // Rate limited - retry later
            return fmt.Errorf("rate limited: %w", err)
        }
        if isAuthError(err) {
            // Auth error - don't retry
            return &NonRetriableError{Cause: err}
        }
        // Other errors - retry
        return err
    }

    return nil
}
```

---

## 6. Monitoring and Observability

### 6.1 Workflow Metrics

```go
// internal/workflow/monitoring/metrics.go
var (
    workflowsStarted = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codeai_workflows_started_total",
            Help: "Total number of workflows started",
        },
        []string{"workflow_type"},
    )

    workflowsCompleted = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codeai_workflows_completed_total",
            Help: "Total number of workflows completed",
        },
        []string{"workflow_type", "status"},
    )

    workflowDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "codeai_workflow_duration_seconds",
            Help:    "Workflow execution duration",
            Buckets: []float64{1, 5, 15, 30, 60, 120, 300, 600},
        },
        []string{"workflow_type"},
    )

    activityDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "codeai_activity_duration_seconds",
            Help:    "Activity execution duration",
            Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60},
        },
        []string{"activity_type"},
    )
)
```

### 6.2 Job Metrics

```go
// internal/scheduler/monitoring/metrics.go
var (
    jobsEnqueued = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codeai_jobs_enqueued_total",
            Help: "Total number of jobs enqueued",
        },
        []string{"task_type", "queue"},
    )

    jobsProcessed = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codeai_jobs_processed_total",
            Help: "Total number of jobs processed",
        },
        []string{"task_type", "status"},
    )

    jobProcessingDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "codeai_job_processing_duration_seconds",
            Help:    "Job processing duration",
            Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60, 300},
        },
        []string{"task_type"},
    )

    queueDepth = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "codeai_queue_depth",
            Help: "Current queue depth",
        },
        []string{"queue", "state"}, // state: pending, active, scheduled, retry
    )
)
```

### 6.3 Logging Best Practices

```go
// Structured logging with context
func (a *Activities) ProcessOrder(ctx context.Context, input OrderInput) error {
    logger := activity.GetLogger(ctx)
    info := activity.GetInfo(ctx)

    logger.Info("Starting order processing",
        "orderID", input.OrderID,
        "attempt", info.Attempt,
        "workflowID", info.WorkflowExecution.ID,
    )

    // Log at appropriate levels
    logger.Debug("Validating inventory", "items", len(input.Items))

    if err := validateInventory(input.Items); err != nil {
        logger.Error("Inventory validation failed",
            "orderID", input.OrderID,
            "error", err,
        )
        return err
    }

    logger.Info("Order processing completed",
        "orderID", input.OrderID,
        "duration", time.Since(start),
    )

    return nil
}
```

---

## 7. Examples

### 7.1 Order Processing Workflow with Compensation

```go
// Complete order processing workflow with saga pattern
package definitions

import (
    "fmt"
    "time"

    "go.temporal.io/sdk/temporal"
    "go.temporal.io/sdk/workflow"

    "github.com/bargom/codeai/internal/workflow/compensation"
)

type OrderInput struct {
    OrderID     string        `json:"orderId"`
    CustomerID  string        `json:"customerId"`
    Items       []OrderItem   `json:"items"`
    TotalAmount float64       `json:"totalAmount"`
    ShippingAddr Address      `json:"shippingAddress"`
}

type OrderOutput struct {
    OrderID        string    `json:"orderId"`
    Status         string    `json:"status"`
    TrackingNumber string    `json:"trackingNumber,omitempty"`
    Error          string    `json:"error,omitempty"`
    CompletedAt    time.Time `json:"completedAt"`
}

func OrderProcessingWorkflow(ctx workflow.Context, input OrderInput) (OrderOutput, error) {
    logger := workflow.GetLogger(ctx)
    cm := compensation.NewCompensationManager(ctx)

    output := OrderOutput{
        OrderID: input.OrderID,
        Status:  "processing",
    }

    logger.Info("Starting order processing",
        "orderID", input.OrderID,
        "items", len(input.Items),
        "total", input.TotalAmount,
    )

    // Configure activity options
    activityOptions := workflow.ActivityOptions{
        StartToCloseTimeout: 5 * time.Minute,
        RetryPolicy: &temporal.RetryPolicy{
            InitialInterval:    time.Second,
            BackoffCoefficient: 2.0,
            MaximumInterval:    time.Minute,
            MaximumAttempts:    3,
        },
    }
    ctx = workflow.WithActivityOptions(ctx, activityOptions)

    // Step 1: Validate Order
    logger.Info("Step 1: Validating order")
    var validationResult OrderValidationResult
    err := workflow.ExecuteActivity(ctx, ValidateOrderActivity, ValidateOrderInput{
        OrderID: input.OrderID,
        Items:   input.Items,
    }).Get(ctx, &validationResult)

    if err != nil || !validationResult.Valid {
        output.Status = "validation_failed"
        output.Error = getErrorMessage(err, validationResult.Error)
        return output, fmt.Errorf("validation failed: %s", output.Error)
    }

    // Step 2: Reserve Inventory (with compensation)
    logger.Info("Step 2: Reserving inventory")
    cm.RegisterCompensation(compensation.CompensationStep{
        ActivityName: "reserve-inventory",
        CompensateFn: func(ctx workflow.Context, data interface{}) error {
            releaseInput := data.(ReleaseInventoryInput)
            return workflow.ExecuteActivity(ctx, ReleaseInventoryActivity, releaseInput).Get(ctx, nil)
        },
        Input: ReleaseInventoryInput{
            OrderID: input.OrderID,
            Items:   input.Items,
        },
        Timeout: 2 * time.Minute,
    })

    var reserveResult ReserveInventoryResult
    err = workflow.ExecuteActivity(ctx, ReserveInventoryActivity, ReserveInventoryInput{
        OrderID: input.OrderID,
        Items:   input.Items,
    }).Get(ctx, &reserveResult)

    if err != nil {
        logger.Error("Failed to reserve inventory, compensating", "error", err)
        cm.Compensate(ctx)
        output.Status = "inventory_failed"
        output.Error = err.Error()
        return output, err
    }
    cm.RecordExecution("reserve-inventory")

    // Step 3: Process Payment (with compensation)
    logger.Info("Step 3: Processing payment")
    cm.RegisterCompensation(compensation.CompensationStep{
        ActivityName: "process-payment",
        CompensateFn: func(ctx workflow.Context, data interface{}) error {
            refundInput := data.(RefundPaymentInput)
            return workflow.ExecuteActivity(ctx, RefundPaymentActivity, refundInput).Get(ctx, nil)
        },
        Input: RefundPaymentInput{
            OrderID: input.OrderID,
        },
        Timeout: 5 * time.Minute,
    })

    paymentCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout: 2 * time.Minute,
        RetryPolicy: &temporal.RetryPolicy{
            MaximumAttempts: 5, // More retries for payment
        },
    })

    var paymentResult ProcessPaymentResult
    err = workflow.ExecuteActivity(paymentCtx, ProcessPaymentActivity, ProcessPaymentInput{
        OrderID:     input.OrderID,
        CustomerID:  input.CustomerID,
        Amount:      input.TotalAmount,
    }).Get(ctx, &paymentResult)

    if err != nil {
        logger.Error("Payment failed, compensating", "error", err)
        cm.Compensate(ctx)
        output.Status = "payment_failed"
        output.Error = err.Error()
        return output, err
    }
    cm.RecordExecution("process-payment")

    // Step 4: Create Shipment (with compensation)
    logger.Info("Step 4: Creating shipment")
    cm.RegisterCompensation(compensation.CompensationStep{
        ActivityName: "create-shipment",
        CompensateFn: func(ctx workflow.Context, data interface{}) error {
            cancelInput := data.(CancelShipmentInput)
            return workflow.ExecuteActivity(ctx, CancelShipmentActivity, cancelInput).Get(ctx, nil)
        },
        Input: CancelShipmentInput{
            OrderID: input.OrderID,
        },
        Timeout:   2 * time.Minute,
        AllowSkip: true, // Shipment cancellation is best-effort
    })

    var shipmentResult CreateShipmentResult
    err = workflow.ExecuteActivity(ctx, CreateShipmentActivity, CreateShipmentInput{
        OrderID:  input.OrderID,
        Items:    input.Items,
        Address:  input.ShippingAddr,
    }).Get(ctx, &shipmentResult)

    if err != nil {
        logger.Error("Shipment creation failed, compensating", "error", err)
        cm.Compensate(ctx)
        output.Status = "shipment_failed"
        output.Error = err.Error()
        return output, err
    }
    cm.RecordExecution("create-shipment")
    output.TrackingNumber = shipmentResult.TrackingNumber

    // Step 5: Send Confirmation (non-critical, no compensation needed)
    logger.Info("Step 5: Sending confirmation")
    notifyCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout: 30 * time.Second,
        RetryPolicy: &temporal.RetryPolicy{
            MaximumAttempts: 2,
        },
    })

    // Fire and forget - don't fail order if notification fails
    workflow.ExecuteActivity(notifyCtx, SendOrderConfirmationActivity, SendConfirmationInput{
        OrderID:        input.OrderID,
        CustomerID:     input.CustomerID,
        TrackingNumber: shipmentResult.TrackingNumber,
    })

    // Complete
    output.Status = "completed"
    output.CompletedAt = workflow.Now(ctx)

    logger.Info("Order processing completed",
        "orderID", input.OrderID,
        "trackingNumber", output.TrackingNumber,
    )

    return output, nil
}
```

### 7.2 Daily Report Generation Job

```go
// Complete daily report job implementation
package tasks

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/hibiken/asynq"
)

const TypeDailyReport = "report:daily:generate"

type DailyReportPayload struct {
    ReportType   string    `json:"report_type"`   // sales, inventory, activity
    ReportDate   time.Time `json:"report_date"`
    Recipients   []string  `json:"recipients"`
    Format       string    `json:"format"`        // pdf, csv, xlsx
    Filters      map[string]string `json:"filters,omitempty"`
    IncludeCharts bool     `json:"include_charts"`
}

type DailyReportHandler struct {
    db          Database
    storage     ObjectStorage
    emailClient EmailClient
    reportGen   ReportGenerator
}

func NewDailyReportHandler(
    db Database,
    storage ObjectStorage,
    email EmailClient,
    gen ReportGenerator,
) *DailyReportHandler {
    return &DailyReportHandler{
        db:          db,
        storage:     storage,
        emailClient: email,
        reportGen:   gen,
    }
}

func (h *DailyReportHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
    var payload DailyReportPayload
    if err := json.Unmarshal(t.Payload(), &payload); err != nil {
        return fmt.Errorf("unmarshal payload: %w", err)
    }

    logger := log.With(
        "taskID", t.ResultWriter().TaskID(),
        "reportType", payload.ReportType,
        "reportDate", payload.ReportDate.Format("2006-01-02"),
    )
    logger.Info("Starting report generation")

    startTime := time.Now()

    // Step 1: Fetch data for the report
    logger.Debug("Fetching report data")
    data, err := h.fetchReportData(ctx, payload)
    if err != nil {
        return fmt.Errorf("fetch data: %w", err)
    }
    logger.Debug("Data fetched", "recordCount", len(data))

    // Step 2: Generate the report
    logger.Debug("Generating report")
    reportBytes, err := h.reportGen.Generate(ctx, ReportConfig{
        Type:          payload.ReportType,
        Format:        payload.Format,
        Data:          data,
        IncludeCharts: payload.IncludeCharts,
        ReportDate:    payload.ReportDate,
    })
    if err != nil {
        return fmt.Errorf("generate report: %w", err)
    }
    logger.Debug("Report generated", "sizeBytes", len(reportBytes))

    // Step 3: Upload to storage
    logger.Debug("Uploading report")
    filename := fmt.Sprintf("%s-report-%s.%s",
        payload.ReportType,
        payload.ReportDate.Format("2006-01-02"),
        payload.Format,
    )

    url, err := h.storage.Upload(ctx, filename, reportBytes)
    if err != nil {
        return fmt.Errorf("upload report: %w", err)
    }
    logger.Debug("Report uploaded", "url", url)

    // Step 4: Send email notifications
    logger.Debug("Sending notifications", "recipientCount", len(payload.Recipients))

    var emailErrors []error
    for _, recipient := range payload.Recipients {
        err := h.emailClient.Send(ctx, EmailRequest{
            To:      recipient,
            Subject: fmt.Sprintf("%s Report - %s",
                capitalize(payload.ReportType),
                payload.ReportDate.Format("January 2, 2006"),
            ),
            Body:    h.generateEmailBody(payload, url),
            Attachments: []Attachment{{
                Filename: filename,
                Data:     reportBytes,
            }},
        })
        if err != nil {
            emailErrors = append(emailErrors, fmt.Errorf("%s: %w", recipient, err))
        }
    }

    duration := time.Since(startTime)

    if len(emailErrors) > 0 {
        logger.Warn("Some notifications failed",
            "successCount", len(payload.Recipients)-len(emailErrors),
            "failCount", len(emailErrors),
            "errors", emailErrors,
        )
    }

    logger.Info("Report generation completed",
        "duration", duration,
        "recordCount", len(data),
        "url", url,
    )

    // Write result for inspection
    result := map[string]interface{}{
        "url":         url,
        "recordCount": len(data),
        "duration":    duration.String(),
        "errors":      len(emailErrors),
    }
    resultBytes, _ := json.Marshal(result)
    t.ResultWriter().Write(resultBytes)

    return nil
}

func (h *DailyReportHandler) fetchReportData(ctx context.Context, payload DailyReportPayload) ([]ReportRow, error) {
    switch payload.ReportType {
    case "sales":
        return h.db.GetSalesData(ctx, payload.ReportDate, payload.Filters)
    case "inventory":
        return h.db.GetInventoryData(ctx, payload.ReportDate, payload.Filters)
    case "activity":
        return h.db.GetActivityData(ctx, payload.ReportDate, payload.Filters)
    default:
        return nil, fmt.Errorf("unknown report type: %s", payload.ReportType)
    }
}

func (h *DailyReportHandler) generateEmailBody(payload DailyReportPayload, url string) string {
    return fmt.Sprintf(`
Your %s report for %s is ready.

You can download it from: %s

This report includes data through %s and was generated automatically.

If you have questions, please contact the data team.
`,
        payload.ReportType,
        payload.ReportDate.Format("January 2, 2006"),
        url,
        payload.ReportDate.Format("January 2, 2006"),
    )
}

// Schedule the daily report job
func ScheduleDailyReport(service *scheduler.SchedulerService) (string, error) {
    return service.CreateRecurringJob(context.Background(), scheduler.JobRequest{
        TaskType: TypeDailyReport,
        Payload: DailyReportPayload{
            ReportType:    "sales",
            Recipients:    []string{"sales@company.com", "manager@company.com"},
            Format:        "pdf",
            IncludeCharts: true,
        },
        Queue:      queue.QueueDefault,
        MaxRetries: 3,
        Timeout:    30 * time.Minute,
    }, "0 6 * * *") // Run at 6 AM daily
}
```

### 7.3 User Registration Event Chain

```go
// Event-driven user registration flow
package registration

import (
    "context"

    "github.com/bargom/codeai/internal/event/bus"
)

// Event types
const (
    EventUserCreated           bus.EventType = "user.created"
    EventUserEmailVerified     bus.EventType = "user.email.verified"
    EventUserOnboardingStarted bus.EventType = "user.onboarding.started"
    EventUserOnboardingComplete bus.EventType = "user.onboarding.complete"
)

// UserRegistrationService orchestrates user registration
type UserRegistrationService struct {
    userRepo    UserRepository
    dispatcher  *dispatcher.EventDispatcher
    emailClient EmailClient
}

func (s *UserRegistrationService) RegisterUser(ctx context.Context, input RegisterInput) (*User, error) {
    // Create user
    user, err := s.userRepo.Create(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("create user: %w", err)
    }

    // Emit user created event
    s.dispatcher.Dispatch(ctx, bus.Event{
        ID:        uuid.New().String(),
        Type:      EventUserCreated,
        Source:    "registration-service",
        Timestamp: time.Now(),
        Data: map[string]interface{}{
            "userID":    user.ID,
            "email":     user.Email,
            "createdAt": user.CreatedAt,
        },
    })

    return user, nil
}

// Email verification handler
type EmailVerificationHandler struct {
    userRepo   UserRepository
    dispatcher *dispatcher.EventDispatcher
}

func (h *EmailVerificationHandler) Handle(ctx context.Context, event bus.Event) error {
    if event.Type != EventUserCreated {
        return nil
    }

    userID := event.Data["userID"].(string)
    email := event.Data["email"].(string)

    // Generate verification token
    token, err := generateVerificationToken(userID)
    if err != nil {
        return err
    }

    // Send verification email
    if err := h.sendVerificationEmail(ctx, email, token); err != nil {
        return err
    }

    return nil
}

func (h *EmailVerificationHandler) VerifyEmail(ctx context.Context, token string) error {
    userID, err := validateVerificationToken(token)
    if err != nil {
        return err
    }

    // Mark email as verified
    if err := h.userRepo.VerifyEmail(ctx, userID); err != nil {
        return err
    }

    // Emit verified event
    h.dispatcher.Dispatch(ctx, bus.Event{
        ID:        uuid.New().String(),
        Type:      EventUserEmailVerified,
        Source:    "registration-service",
        Timestamp: time.Now(),
        Data: map[string]interface{}{
            "userID":     userID,
            "verifiedAt": time.Now(),
        },
    })

    return nil
}

// Onboarding handler
type OnboardingHandler struct {
    onboardingService OnboardingService
    dispatcher        *dispatcher.EventDispatcher
}

func (h *OnboardingHandler) Handle(ctx context.Context, event bus.Event) error {
    if event.Type != EventUserEmailVerified {
        return nil
    }

    userID := event.Data["userID"].(string)

    // Start onboarding
    h.dispatcher.Dispatch(ctx, bus.Event{
        ID:        uuid.New().String(),
        Type:      EventUserOnboardingStarted,
        Source:    "onboarding-service",
        Timestamp: time.Now(),
        Data: map[string]interface{}{
            "userID": userID,
        },
    })

    // Create onboarding tasks
    if err := h.onboardingService.CreateTasks(ctx, userID); err != nil {
        return err
    }

    // Send welcome email
    if err := h.onboardingService.SendWelcomeEmail(ctx, userID); err != nil {
        // Log but don't fail
        log.Printf("failed to send welcome email: %v", err)
    }

    return nil
}

// Analytics handler - collects metrics from all events
type AnalyticsHandler struct {
    analytics AnalyticsClient
}

func (h *AnalyticsHandler) Handle(ctx context.Context, event bus.Event) error {
    switch event.Type {
    case EventUserCreated:
        h.analytics.Track("user_registered", event.Data)
    case EventUserEmailVerified:
        h.analytics.Track("email_verified", event.Data)
    case EventUserOnboardingStarted:
        h.analytics.Track("onboarding_started", event.Data)
    case EventUserOnboardingComplete:
        h.analytics.Track("onboarding_complete", event.Data)
    }
    return nil
}

// Wire up the event chain
func SetupRegistrationEventChain(dispatcher *dispatcher.EventDispatcher, deps Dependencies) {
    // Email verification on user created
    emailHandler := &EmailVerificationHandler{
        userRepo:   deps.UserRepo,
        dispatcher: dispatcher,
    }
    dispatcher.Subscribe(EventUserCreated, emailHandler)

    // Onboarding on email verified
    onboardingHandler := &OnboardingHandler{
        onboardingService: deps.OnboardingService,
        dispatcher:        dispatcher,
    }
    dispatcher.Subscribe(EventUserEmailVerified, onboardingHandler)

    // Analytics on all events
    analyticsHandler := &AnalyticsHandler{
        analytics: deps.Analytics,
    }
    dispatcher.Subscribe(EventUserCreated, analyticsHandler)
    dispatcher.Subscribe(EventUserEmailVerified, analyticsHandler)
    dispatcher.Subscribe(EventUserOnboardingStarted, analyticsHandler)
    dispatcher.Subscribe(EventUserOnboardingComplete, analyticsHandler)
}
```

---

## Appendix A: DSL Syntax (Planned)

The following DSL syntax is planned for future implementation:

### Workflow Definition (DSL)

```codeai
workflow OrderFulfillment {
    description: "Process order from placement to delivery"
    trigger: OrderPlaced

    steps {
        validate_inventory {
            for_each: trigger.order.items
            check: item.product.quantity >= item.quantity
            on_fail: cancel_order("Insufficient inventory")
        }

        process_payment {
            call: PaymentGateway.charge {
                amount: trigger.order.total
            }
            timeout: 30s
            retry: 3 times with exponential_backoff
            on_fail: rollback
        }

        create_shipment {
            call: ShippingService.create {
                order_id: trigger.order.id
                address: trigger.order.shipping_address
            }
            on_fail: rollback
        }

        notify_customer {
            send: email(trigger.order.customer.email) {
                template: "order_shipped"
                data: { tracking_number: create_shipment.tracking_number }
            }
        }
    }

    on_complete: update(trigger.order.status = "completed")
    on_fail: emit(OrderFailed)
}
```

### Job Definition (DSL)

```codeai
job DailyInventoryReport {
    description: "Generate daily inventory report"
    schedule: "0 6 * * *"
    timezone: "UTC"

    steps {
        fetch_data {
            query: select Product where quantity < 10
            as: low_stock_items
        }

        generate_report {
            template: "inventory_report"
            data: { low_stock: low_stock_items }
            format: pdf
        }

        distribute {
            send: email(config.reports.recipients) {
                subject: "Daily Inventory Report"
                attachment: report_file
            }
        }
    }

    retry: 3 times
    timeout: 10m
}
```

---

## Appendix B: Implementation Reference

| Component | Location | Description |
|-----------|----------|-------------|
| Workflow Engine | `internal/workflow/engine/engine.go` | Temporal client wrapper |
| Workflow Definitions | `internal/workflow/definitions/` | Workflow implementations |
| Activities | `internal/workflow/activities/` | Activity implementations |
| Compensation | `internal/workflow/compensation/` | Saga pattern support |
| Saga Patterns | `internal/workflow/patterns/saga_workflow.go` | Reusable saga workflow |
| Queue Manager | `internal/scheduler/queue/manager.go` | Asynq queue management |
| Task Definitions | `internal/scheduler/tasks/` | Task type definitions |
| Scheduler Service | `internal/scheduler/service/` | Job scheduling service |
| Event Bus | `internal/event/bus/event_bus.go` | Pub/sub event system |
| Event Dispatcher | `internal/event/dispatcher/` | Event routing |
| Event Handlers | `internal/event/handlers/` | Event processing |

---

*This document describes the CodeAI workflow and job systems. For API documentation, see the API Reference. For DSL syntax, see the DSL Language Specification.*

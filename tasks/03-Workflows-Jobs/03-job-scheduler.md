# Task: Job Scheduler with Asynq

## Overview
Implement a job scheduling system using Asynq for scheduled and recurring background tasks.

## Phase
Phase 3: Workflows and Jobs

## Priority
High - Required for background processing.

## Dependencies
- Phase 1 complete
- Redis cache module

## Description
Create a job scheduling module that supports cron-based scheduling, one-time delayed jobs, and job queues with Asynq and Redis.

## Detailed Requirements

### 1. Job Types (internal/modules/job/types.go)

```go
package job

import (
    "time"
)

type Job struct {
    ID          string
    Description string
    Schedule    string        // Cron expression or "every 1h"
    Timezone    string
    Steps       []*JobStep
    Timeout     time.Duration
    Retry       int
    Queue       string        // Queue name for priority
    OnFail      *Action
}

type JobStep struct {
    Name   string
    Action *Action
}

type JobExecution struct {
    ID          string
    JobID       string
    Status      JobStatus
    Payload     any
    Result      any
    Error       string
    StartedAt   time.Time
    CompletedAt *time.Time
    Attempts    int
}

type JobStatus string

const (
    JobStatusPending   JobStatus = "pending"
    JobStatusRunning   JobStatus = "running"
    JobStatusCompleted JobStatus = "completed"
    JobStatusFailed    JobStatus = "failed"
    JobStatusRetrying  JobStatus = "retrying"
)
```

### 2. Asynq Job Module (internal/modules/job/asynq.go)

```go
package job

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/hibiken/asynq"
    "log/slog"
)

type AsynqJobModule struct {
    client     *asynq.Client
    server     *asynq.Server
    scheduler  *asynq.Scheduler
    inspector  *asynq.Inspector
    jobs       map[string]*Job
    handlers   map[string]asynq.Handler
    redisOpt   asynq.RedisClientOpt
    logger     *slog.Logger
}

type JobConfig struct {
    RedisURL    string
    Concurrency int
    Queues      map[string]int // Queue name -> priority
}

func NewAsynqJobModule(config JobConfig) (*AsynqJobModule, error) {
    redisOpt, err := asynq.ParseRedisURI(config.RedisURL)
    if err != nil {
        return nil, fmt.Errorf("invalid redis URL: %w", err)
    }

    queues := config.Queues
    if queues == nil {
        queues = map[string]int{
            "critical": 6,
            "default":  3,
            "low":      1,
        }
    }

    concurrency := config.Concurrency
    if concurrency == 0 {
        concurrency = 10
    }

    m := &AsynqJobModule{
        client:    asynq.NewClient(redisOpt),
        inspector: asynq.NewInspector(redisOpt),
        jobs:      make(map[string]*Job),
        handlers:  make(map[string]asynq.Handler),
        redisOpt:  redisOpt,
        logger:    slog.Default().With("module", "job"),
    }

    m.server = asynq.NewServer(
        redisOpt,
        asynq.Config{
            Concurrency: concurrency,
            Queues:      queues,
            ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
                m.logger.Error("job failed",
                    "type", task.Type(),
                    "error", err,
                )
            }),
            Logger: &asynqLogger{logger: m.logger},
        },
    )

    m.scheduler = asynq.NewScheduler(redisOpt, nil)

    return m, nil
}

func (m *AsynqJobModule) Name() string { return "job-scheduler" }

func (m *AsynqJobModule) Initialize(config *Config) error {
    return nil
}

func (m *AsynqJobModule) Start(ctx context.Context) error {
    // Register scheduled jobs
    for _, job := range m.jobs {
        if job.Schedule != "" {
            schedule := m.parseSchedule(job.Schedule)
            _, err := m.scheduler.Register(schedule, asynq.NewTask(job.ID, nil))
            if err != nil {
                return fmt.Errorf("failed to register job %s: %w", job.ID, err)
            }
            m.logger.Info("registered scheduled job", "job", job.ID, "schedule", job.Schedule)
        }
    }

    // Start scheduler
    go func() {
        if err := m.scheduler.Start(); err != nil {
            m.logger.Error("scheduler error", "error", err)
        }
    }()

    // Build mux and start server
    mux := asynq.NewServeMux()
    for jobID := range m.jobs {
        handler := m.createHandler(jobID)
        mux.HandleFunc(jobID, handler)
    }

    go func() {
        if err := m.server.Start(mux); err != nil {
            m.logger.Error("server error", "error", err)
        }
    }()

    m.logger.Info("job scheduler started")
    return nil
}

func (m *AsynqJobModule) Stop(ctx context.Context) error {
    m.scheduler.Shutdown()
    m.server.Shutdown()
    m.client.Close()
    return nil
}

func (m *AsynqJobModule) Health() HealthStatus {
    info, err := m.inspector.CurrentStats()
    if err != nil {
        return HealthStatus{Status: "unhealthy", Error: err.Error()}
    }

    return HealthStatus{
        Status: "healthy",
        Details: map[string]any{
            "pending":   info.Pending,
            "active":    info.Active,
            "completed": info.Completed,
            "failed":    info.Failed,
        },
    }
}

func (m *AsynqJobModule) RegisterJob(job *Job) error {
    m.jobs[job.ID] = job
    return nil
}

func (m *AsynqJobModule) Enqueue(ctx context.Context, jobID string, payload any) error {
    data, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    job := m.jobs[jobID]
    opts := []asynq.Option{}

    if job != nil {
        if job.Queue != "" {
            opts = append(opts, asynq.Queue(job.Queue))
        }
        if job.Retry > 0 {
            opts = append(opts, asynq.MaxRetry(job.Retry))
        }
        if job.Timeout > 0 {
            opts = append(opts, asynq.Timeout(job.Timeout))
        }
    }

    task := asynq.NewTask(jobID, data, opts...)
    _, err = m.client.EnqueueContext(ctx, task)
    return err
}

func (m *AsynqJobModule) EnqueueAt(ctx context.Context, jobID string, payload any, at time.Time) error {
    data, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    task := asynq.NewTask(jobID, data)
    _, err = m.client.EnqueueContext(ctx, task, asynq.ProcessAt(at))
    return err
}

func (m *AsynqJobModule) EnqueueIn(ctx context.Context, jobID string, payload any, delay time.Duration) error {
    data, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    task := asynq.NewTask(jobID, data)
    _, err = m.client.EnqueueContext(ctx, task, asynq.ProcessIn(delay))
    return err
}

func (m *AsynqJobModule) createHandler(jobID string) func(context.Context, *asynq.Task) error {
    return func(ctx context.Context, task *asynq.Task) error {
        job := m.jobs[jobID]
        if job == nil {
            return fmt.Errorf("job not found: %s", jobID)
        }

        m.logger.Info("executing job", "job", jobID)

        var payload any
        if len(task.Payload()) > 0 {
            json.Unmarshal(task.Payload(), &payload)
        }

        // Execute job steps
        for _, step := range job.Steps {
            if err := m.executeStep(ctx, step, payload); err != nil {
                m.logger.Error("job step failed", "job", jobID, "step", step.Name, "error", err)

                // Execute OnFail if defined
                if job.OnFail != nil {
                    m.executeAction(ctx, job.OnFail, payload)
                }

                return err
            }
        }

        m.logger.Info("job completed", "job", jobID)
        return nil
    }
}

func (m *AsynqJobModule) executeStep(ctx context.Context, step *JobStep, payload any) error {
    // Implementation depends on action type
    return m.executeAction(ctx, step.Action, payload)
}

func (m *AsynqJobModule) executeAction(ctx context.Context, action *Action, payload any) error {
    // Execute action based on type
    return nil
}

func (m *AsynqJobModule) parseSchedule(schedule string) string {
    // Handle "every 1h" syntax
    if strings.HasPrefix(schedule, "every ") {
        duration := strings.TrimPrefix(schedule, "every ")
        d, err := time.ParseDuration(duration)
        if err == nil {
            // Convert to cron expression
            switch {
            case d == time.Hour:
                return "0 * * * *"
            case d == 24*time.Hour:
                return "0 0 * * *"
            default:
                // For arbitrary durations, use @every
                return "@every " + duration
            }
        }
    }
    return schedule
}

func (m *AsynqJobModule) GetScheduled(ctx context.Context) ([]*ScheduledJob, error) {
    entries, err := m.scheduler.Entries()
    if err != nil {
        return nil, err
    }

    var jobs []*ScheduledJob
    for _, entry := range entries {
        jobs = append(jobs, &ScheduledJob{
            ID:       entry.ID,
            Schedule: entry.Spec,
            Next:     entry.Next,
            Prev:     entry.Prev,
        })
    }

    return jobs, nil
}

type ScheduledJob struct {
    ID       string
    Schedule string
    Next     time.Time
    Prev     time.Time
}

type asynqLogger struct {
    logger *slog.Logger
}

func (l *asynqLogger) Debug(args ...interface{}) {
    l.logger.Debug(fmt.Sprint(args...))
}

func (l *asynqLogger) Info(args ...interface{}) {
    l.logger.Info(fmt.Sprint(args...))
}

func (l *asynqLogger) Warn(args ...interface{}) {
    l.logger.Warn(fmt.Sprint(args...))
}

func (l *asynqLogger) Error(args ...interface{}) {
    l.logger.Error(fmt.Sprint(args...))
}

func (l *asynqLogger) Fatal(args ...interface{}) {
    l.logger.Error(fmt.Sprint(args...))
}
```

### 3. Job Module Interface

```go
// internal/modules/job/module.go
package job

type JobModule interface {
    Module
    RegisterJob(job *Job) error
    Enqueue(ctx context.Context, jobID string, payload any) error
    EnqueueAt(ctx context.Context, jobID string, payload any, at time.Time) error
    EnqueueIn(ctx context.Context, jobID string, payload any, delay time.Duration) error
    GetScheduled(ctx context.Context) ([]*ScheduledJob, error)
}
```

## Acceptance Criteria
- [ ] Register jobs with cron schedules
- [ ] Support "every X" schedule syntax
- [ ] Enqueue one-time jobs
- [ ] Delayed job execution
- [ ] Multiple queue priorities
- [ ] Retry configuration
- [ ] Job timeout handling
- [ ] Job status inspection

## Testing Strategy
- Unit tests with mock Redis
- Integration tests with Redis container
- Scheduler timing tests

## Files to Create
- `internal/modules/job/types.go`
- `internal/modules/job/module.go`
- `internal/modules/job/asynq.go`
- `internal/modules/job/job_test.go`

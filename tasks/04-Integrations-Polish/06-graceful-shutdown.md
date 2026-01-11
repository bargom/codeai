# Task: Graceful Shutdown

## Overview
Implement graceful shutdown handling for the runtime to ensure clean termination of all components.

## Phase
Phase 4: Integrations and Polish

## Priority
High - Essential for production deployments.

## Dependencies
- Phase 1 complete

## Description
Create a shutdown manager that coordinates graceful termination of HTTP server, database connections, background jobs, and active workflows.

## Detailed Requirements

### 1. Shutdown Manager (internal/shutdown/manager.go)

```go
package shutdown

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"

    "log/slog"
)

type ShutdownManager struct {
    hooks        []ShutdownHook
    timeout      time.Duration
    logger       *slog.Logger
    shutdownOnce sync.Once
    done         chan struct{}
}

type ShutdownHook struct {
    Name     string
    Priority int // Lower = earlier shutdown
    Fn       func(ctx context.Context) error
}

func NewShutdownManager(timeout time.Duration) *ShutdownManager {
    if timeout == 0 {
        timeout = 30 * time.Second
    }

    return &ShutdownManager{
        hooks:   make([]ShutdownHook, 0),
        timeout: timeout,
        logger:  slog.Default().With("component", "shutdown"),
        done:    make(chan struct{}),
    }
}

// Register adds a shutdown hook
func (m *ShutdownManager) Register(name string, priority int, fn func(ctx context.Context) error) {
    m.hooks = append(m.hooks, ShutdownHook{
        Name:     name,
        Priority: priority,
        Fn:       fn,
    })
}

// ListenForSignals starts listening for shutdown signals
func (m *ShutdownManager) ListenForSignals() <-chan struct{} {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

    go func() {
        sig := <-sigChan
        m.logger.Info("received shutdown signal", "signal", sig)
        m.Shutdown()
    }()

    return m.done
}

// Shutdown performs graceful shutdown
func (m *ShutdownManager) Shutdown() {
    m.shutdownOnce.Do(func() {
        m.logger.Info("starting graceful shutdown", "timeout", m.timeout)

        ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
        defer cancel()

        // Sort hooks by priority
        sortHooks(m.hooks)

        // Execute hooks
        var wg sync.WaitGroup
        errors := make(chan error, len(m.hooks))

        currentPriority := -1
        var priorityWg sync.WaitGroup

        for _, hook := range m.hooks {
            // If priority changed, wait for previous priority group
            if hook.Priority != currentPriority {
                priorityWg.Wait()
                currentPriority = hook.Priority
            }

            priorityWg.Add(1)
            wg.Add(1)

            go func(h ShutdownHook) {
                defer priorityWg.Done()
                defer wg.Done()

                m.logger.Info("executing shutdown hook", "name", h.Name, "priority", h.Priority)

                if err := h.Fn(ctx); err != nil {
                    m.logger.Error("shutdown hook failed", "name", h.Name, "error", err)
                    errors <- fmt.Errorf("%s: %w", h.Name, err)
                } else {
                    m.logger.Info("shutdown hook completed", "name", h.Name)
                }
            }(hook)
        }

        // Wait for all hooks or timeout
        done := make(chan struct{})
        go func() {
            wg.Wait()
            close(done)
        }()

        select {
        case <-done:
            m.logger.Info("graceful shutdown completed")
        case <-ctx.Done():
            m.logger.Warn("shutdown timeout exceeded, forcing exit")
        }

        close(errors)

        // Collect errors
        var shutdownErrors []error
        for err := range errors {
            shutdownErrors = append(shutdownErrors, err)
        }

        if len(shutdownErrors) > 0 {
            m.logger.Error("shutdown completed with errors", "count", len(shutdownErrors))
        }

        close(m.done)
    })
}

func sortHooks(hooks []ShutdownHook) {
    // Simple bubble sort for small slice
    for i := 0; i < len(hooks); i++ {
        for j := i + 1; j < len(hooks); j++ {
            if hooks[j].Priority < hooks[i].Priority {
                hooks[i], hooks[j] = hooks[j], hooks[i]
            }
        }
    }
}
```

### 2. HTTP Server Shutdown (internal/shutdown/http.go)

```go
package shutdown

import (
    "context"
    "net/http"
    "time"
)

// HTTPServerShutdown creates a shutdown hook for HTTP server
func HTTPServerShutdown(server *http.Server, drainTimeout time.Duration) func(ctx context.Context) error {
    return func(ctx context.Context) error {
        // Stop accepting new connections
        server.SetKeepAlivesEnabled(false)

        // Create shutdown context with drain timeout
        shutdownCtx, cancel := context.WithTimeout(ctx, drainTimeout)
        defer cancel()

        // Shutdown server (waits for active requests)
        return server.Shutdown(shutdownCtx)
    }
}
```

### 3. Database Shutdown (internal/shutdown/database.go)

```go
package shutdown

import (
    "context"
)

type DatabaseCloser interface {
    Close() error
}

// DatabaseShutdown creates a shutdown hook for database connections
func DatabaseShutdown(db DatabaseCloser) func(ctx context.Context) error {
    return func(ctx context.Context) error {
        return db.Close()
    }
}
```

### 4. Job Scheduler Shutdown (internal/shutdown/jobs.go)

```go
package shutdown

import (
    "context"
    "time"
)

type JobScheduler interface {
    Shutdown()
    WaitForCompletion(timeout time.Duration)
}

// JobSchedulerShutdown creates a shutdown hook for job scheduler
func JobSchedulerShutdown(scheduler JobScheduler, waitTimeout time.Duration) func(ctx context.Context) error {
    return func(ctx context.Context) error {
        // Stop accepting new jobs
        scheduler.Shutdown()

        // Wait for running jobs to complete
        done := make(chan struct{})
        go func() {
            scheduler.WaitForCompletion(waitTimeout)
            close(done)
        }()

        select {
        case <-done:
            return nil
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

### 5. Workflow Shutdown (internal/shutdown/workflow.go)

```go
package shutdown

import (
    "context"
    "time"
)

type WorkflowEngine interface {
    PauseNewExecutions()
    WaitForRunning(timeout time.Duration) error
}

// WorkflowShutdown creates a shutdown hook for workflow engine
func WorkflowShutdown(engine WorkflowEngine, waitTimeout time.Duration) func(ctx context.Context) error {
    return func(ctx context.Context) error {
        // Stop accepting new workflow executions
        engine.PauseNewExecutions()

        // Wait for running workflows (they will be persisted and can resume)
        return engine.WaitForRunning(waitTimeout)
    }
}
```

### 6. Integration Example (internal/shutdown/example.go)

```go
package shutdown

import (
    "context"
    "net/http"
    "time"
)

// SetupShutdown configures graceful shutdown for all components
func SetupShutdown(
    httpServer *http.Server,
    db DatabaseCloser,
    cache CacheCloser,
    jobScheduler JobScheduler,
    workflowEngine WorkflowEngine,
) *ShutdownManager {
    manager := NewShutdownManager(30 * time.Second)

    // Priority 1: Stop accepting new requests/jobs
    manager.Register("http-server", 1, HTTPServerShutdown(httpServer, 10*time.Second))
    manager.Register("workflow-engine", 1, WorkflowShutdown(workflowEngine, 10*time.Second))

    // Priority 2: Wait for active work to complete
    manager.Register("job-scheduler", 2, JobSchedulerShutdown(jobScheduler, 15*time.Second))

    // Priority 3: Close connections
    manager.Register("cache", 3, func(ctx context.Context) error {
        return cache.Close()
    })
    manager.Register("database", 3, DatabaseShutdown(db))

    return manager
}

type CacheCloser interface {
    Close() error
}
```

### 7. Runtime Integration

```go
// internal/runtime/engine.go additions

func (e *Engine) Run(ctx context.Context) error {
    // Setup shutdown manager
    shutdownManager := shutdown.SetupShutdown(
        e.httpServer,
        e.database,
        e.cache,
        e.jobScheduler,
        e.workflowEngine,
    )

    // Start listening for shutdown signals
    done := shutdownManager.ListenForSignals()

    // Start all modules
    if err := e.startModules(ctx); err != nil {
        return err
    }

    // Wait for shutdown
    <-done

    return nil
}
```

## Acceptance Criteria
- [ ] Signal handling (SIGINT, SIGTERM, SIGQUIT)
- [ ] Prioritized shutdown hooks
- [ ] HTTP server drain (complete in-flight requests)
- [ ] Database connection cleanup
- [ ] Job scheduler stop and wait
- [ ] Workflow persistence before exit
- [ ] Configurable shutdown timeout
- [ ] Logging of shutdown progress

## Testing Strategy
- Unit tests for shutdown manager
- Integration tests with signal simulation
- Timeout behavior tests

## Files to Create
- `internal/shutdown/manager.go`
- `internal/shutdown/http.go`
- `internal/shutdown/database.go`
- `internal/shutdown/jobs.go`
- `internal/shutdown/workflow.go`
- `internal/shutdown/shutdown_test.go`

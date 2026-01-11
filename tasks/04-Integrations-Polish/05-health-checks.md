# Task: Health Check Endpoints

## Overview
Implement health check endpoints for Kubernetes liveness/readiness probes and service monitoring.

## Phase
Phase 4: Integrations and Polish

## Priority
High - Required for container orchestration.

## Dependencies
- 01-Foundation/06-http-module.md

## Description
Create health check endpoints that report the status of the application and its dependencies, supporting Kubernetes health probes.

## Detailed Requirements

### 1. Health Types (internal/health/types.go)

```go
package health

import "time"

type Status string

const (
    StatusHealthy   Status = "healthy"
    StatusUnhealthy Status = "unhealthy"
    StatusDegraded  Status = "degraded"
)

type HealthResponse struct {
    Status    Status                  `json:"status"`
    Timestamp time.Time               `json:"timestamp"`
    Version   string                  `json:"version,omitempty"`
    Uptime    string                  `json:"uptime,omitempty"`
    Checks    map[string]CheckResult  `json:"checks,omitempty"`
}

type CheckResult struct {
    Status  Status         `json:"status"`
    Message string         `json:"message,omitempty"`
    Time    time.Duration  `json:"time,omitempty"`
    Details map[string]any `json:"details,omitempty"`
}

type Checker interface {
    Name() string
    Check(ctx context.Context) CheckResult
}
```

### 2. Health Module (internal/health/health.go)

```go
package health

import (
    "context"
    "sync"
    "time"
)

type HealthService struct {
    checkers  []Checker
    startTime time.Time
    version   string
    mu        sync.RWMutex
}

func NewHealthService(version string) *HealthService {
    return &HealthService{
        checkers:  make([]Checker, 0),
        startTime: time.Now(),
        version:   version,
    }
}

func (h *HealthService) Register(checker Checker) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.checkers = append(h.checkers, checker)
}

// Liveness check - is the application running?
func (h *HealthService) Liveness(ctx context.Context) HealthResponse {
    return HealthResponse{
        Status:    StatusHealthy,
        Timestamp: time.Now(),
        Version:   h.version,
        Uptime:    time.Since(h.startTime).String(),
    }
}

// Readiness check - is the application ready to serve traffic?
func (h *HealthService) Readiness(ctx context.Context) HealthResponse {
    h.mu.RLock()
    checkers := h.checkers
    h.mu.RUnlock()

    checks := make(map[string]CheckResult)
    overallStatus := StatusHealthy

    // Run checks concurrently with timeout
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    var wg sync.WaitGroup
    var mu sync.Mutex

    for _, checker := range checkers {
        wg.Add(1)
        go func(c Checker) {
            defer wg.Done()

            start := time.Now()
            result := c.Check(ctx)
            result.Time = time.Since(start)

            mu.Lock()
            checks[c.Name()] = result

            if result.Status == StatusUnhealthy {
                overallStatus = StatusUnhealthy
            } else if result.Status == StatusDegraded && overallStatus != StatusUnhealthy {
                overallStatus = StatusDegraded
            }
            mu.Unlock()
        }(checker)
    }

    wg.Wait()

    return HealthResponse{
        Status:    overallStatus,
        Timestamp: time.Now(),
        Version:   h.version,
        Uptime:    time.Since(h.startTime).String(),
        Checks:    checks,
    }
}

// Full health check with all details
func (h *HealthService) Health(ctx context.Context) HealthResponse {
    return h.Readiness(ctx)
}
```

### 3. Built-in Checkers (internal/health/checkers.go)

```go
package health

import (
    "context"
    "fmt"
    "runtime"
)

// DatabaseChecker checks database connectivity
type DatabaseChecker struct {
    db interface {
        Ping(ctx context.Context) error
        Stats() DBStats
    }
}

type DBStats struct {
    MaxOpenConnections int
    OpenConnections    int
    InUse              int
    Idle               int
}

func NewDatabaseChecker(db interface {
    Ping(ctx context.Context) error
    Stats() DBStats
}) *DatabaseChecker {
    return &DatabaseChecker{db: db}
}

func (c *DatabaseChecker) Name() string { return "database" }

func (c *DatabaseChecker) Check(ctx context.Context) CheckResult {
    if err := c.db.Ping(ctx); err != nil {
        return CheckResult{
            Status:  StatusUnhealthy,
            Message: fmt.Sprintf("database ping failed: %v", err),
        }
    }

    stats := c.db.Stats()
    return CheckResult{
        Status: StatusHealthy,
        Details: map[string]any{
            "max_connections":  stats.MaxOpenConnections,
            "open_connections": stats.OpenConnections,
            "in_use":          stats.InUse,
            "idle":            stats.Idle,
        },
    }
}

// RedisChecker checks Redis connectivity
type RedisChecker struct {
    redis interface {
        Ping(ctx context.Context) error
    }
}

func NewRedisChecker(redis interface {
    Ping(ctx context.Context) error
}) *RedisChecker {
    return &RedisChecker{redis: redis}
}

func (c *RedisChecker) Name() string { return "redis" }

func (c *RedisChecker) Check(ctx context.Context) CheckResult {
    if err := c.redis.Ping(ctx); err != nil {
        return CheckResult{
            Status:  StatusUnhealthy,
            Message: fmt.Sprintf("redis ping failed: %v", err),
        }
    }

    return CheckResult{Status: StatusHealthy}
}

// MemoryChecker checks memory usage
type MemoryChecker struct {
    threshold float64 // Percentage (0-100)
}

func NewMemoryChecker(threshold float64) *MemoryChecker {
    if threshold == 0 {
        threshold = 90
    }
    return &MemoryChecker{threshold: threshold}
}

func (c *MemoryChecker) Name() string { return "memory" }

func (c *MemoryChecker) Check(ctx context.Context) CheckResult {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)

    allocMB := float64(m.Alloc) / 1024 / 1024
    sysMB := float64(m.Sys) / 1024 / 1024
    usagePercent := (float64(m.Alloc) / float64(m.Sys)) * 100

    status := StatusHealthy
    if usagePercent > c.threshold {
        status = StatusDegraded
    }

    return CheckResult{
        Status: status,
        Details: map[string]any{
            "alloc_mb":      fmt.Sprintf("%.2f", allocMB),
            "sys_mb":        fmt.Sprintf("%.2f", sysMB),
            "usage_percent": fmt.Sprintf("%.2f", usagePercent),
            "num_gc":        m.NumGC,
            "goroutines":    runtime.NumGoroutine(),
        },
    }
}

// DiskChecker checks disk space (simplified)
type DiskChecker struct {
    path      string
    threshold float64
}

func NewDiskChecker(path string, threshold float64) *DiskChecker {
    if threshold == 0 {
        threshold = 90
    }
    return &DiskChecker{path: path, threshold: threshold}
}

func (c *DiskChecker) Name() string { return "disk" }

func (c *DiskChecker) Check(ctx context.Context) CheckResult {
    // Implementation would use syscall to get disk stats
    return CheckResult{Status: StatusHealthy}
}

// CustomChecker allows custom health checks
type CustomChecker struct {
    name    string
    checkFn func(ctx context.Context) CheckResult
}

func NewCustomChecker(name string, fn func(ctx context.Context) CheckResult) *CustomChecker {
    return &CustomChecker{name: name, checkFn: fn}
}

func (c *CustomChecker) Name() string { return c.name }

func (c *CustomChecker) Check(ctx context.Context) CheckResult {
    return c.checkFn(ctx)
}
```

### 4. HTTP Handlers (internal/health/handlers.go)

```go
package health

import (
    "encoding/json"
    "net/http"
)

type Handler struct {
    service *HealthService
}

func NewHandler(service *HealthService) *Handler {
    return &Handler{service: service}
}

// LivenessHandler handles /health/live endpoint
func (h *Handler) LivenessHandler(w http.ResponseWriter, r *http.Request) {
    resp := h.service.Liveness(r.Context())
    h.writeResponse(w, resp)
}

// ReadinessHandler handles /health/ready endpoint
func (h *Handler) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
    resp := h.service.Readiness(r.Context())
    h.writeResponse(w, resp)
}

// HealthHandler handles /health endpoint with full details
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
    resp := h.service.Health(r.Context())
    h.writeResponse(w, resp)
}

func (h *Handler) writeResponse(w http.ResponseWriter, resp HealthResponse) {
    w.Header().Set("Content-Type", "application/json")

    status := http.StatusOK
    if resp.Status == StatusUnhealthy {
        status = http.StatusServiceUnavailable
    }

    w.WriteHeader(status)
    json.NewEncoder(w).Encode(resp)
}

// RegisterRoutes registers health check routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/health", h.HealthHandler)
    mux.HandleFunc("/health/live", h.LivenessHandler)
    mux.HandleFunc("/health/ready", h.ReadinessHandler)
}
```

## Acceptance Criteria
- [ ] /health endpoint with full status
- [ ] /health/live for Kubernetes liveness probe
- [ ] /health/ready for Kubernetes readiness probe
- [ ] Database connectivity check
- [ ] Redis connectivity check
- [ ] Memory usage check
- [ ] Custom checker support
- [ ] Concurrent check execution with timeout

## Testing Strategy
- Unit tests for health service
- Unit tests for each checker
- Integration tests with mock dependencies

## Files to Create
- `internal/health/types.go`
- `internal/health/health.go`
- `internal/health/checkers.go`
- `internal/health/handlers.go`
- `internal/health/health_test.go`

package health

import (
	"context"
	"sync"
	"time"
)

// Registry manages health checkers and executes checks.
type Registry struct {
	checkers  []Checker
	startTime time.Time
	version   string
	mu        sync.RWMutex
}

// NewRegistry creates a new health check registry.
func NewRegistry(version string) *Registry {
	return &Registry{
		checkers:  make([]Checker, 0),
		startTime: time.Now(),
		version:   version,
	}
}

// Register adds a health checker to the registry.
func (r *Registry) Register(checker Checker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkers = append(r.checkers, checker)
}

// Checkers returns a copy of the registered checkers.
func (r *Registry) Checkers() []Checker {
	r.mu.RLock()
	defer r.mu.RUnlock()
	checkers := make([]Checker, len(r.checkers))
	copy(checkers, r.checkers)
	return checkers
}

// Liveness returns a liveness check response.
// Liveness checks only fail if the process is broken (e.g., deadlock).
func (r *Registry) Liveness(ctx context.Context) Response {
	return Response{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Version:   r.version,
		Uptime:    time.Since(r.startTime).String(),
	}
}

// Readiness checks if the application is ready to serve traffic.
// It runs all critical health checks concurrently.
func (r *Registry) Readiness(ctx context.Context) Response {
	return r.runChecks(ctx, true)
}

// Health returns a full health check with all registered checkers.
func (r *Registry) Health(ctx context.Context) Response {
	return r.runChecks(ctx, false)
}

// runChecks executes health checks concurrently with a timeout.
// If criticalOnly is true, only critical severity checks are run.
func (r *Registry) runChecks(ctx context.Context, criticalOnly bool) Response {
	r.mu.RLock()
	checkers := r.checkers
	r.mu.RUnlock()

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	checks := make(map[string]CheckResult)
	overallStatus := StatusHealthy

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, checker := range checkers {
		if criticalOnly && checker.Severity() != SeverityCritical {
			continue
		}

		wg.Add(1)
		go func(c Checker) {
			defer wg.Done()

			start := time.Now()
			result := c.Check(ctx)
			result.Duration = time.Since(start)

			mu.Lock()
			defer mu.Unlock()

			checks[c.Name()] = result

			// Update overall status based on check result and severity
			if result.Status == StatusUnhealthy {
				if c.Severity() == SeverityCritical {
					overallStatus = StatusUnhealthy
				} else if overallStatus == StatusHealthy {
					overallStatus = StatusDegraded
				}
			} else if result.Status == StatusDegraded && overallStatus == StatusHealthy {
				overallStatus = StatusDegraded
			}
		}(checker)
	}

	wg.Wait()

	return Response{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Version:   r.version,
		Uptime:    time.Since(r.startTime).String(),
		Checks:    checks,
	}
}

// StartTime returns when the registry was created.
func (r *Registry) StartTime() time.Time {
	return r.startTime
}

// Version returns the version string.
func (r *Registry) Version() string {
	return r.version
}

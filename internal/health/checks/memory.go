package checks

import (
	"context"
	"fmt"
	"runtime"

	"github.com/bargom/codeai/internal/health"
)

// MemoryChecker checks memory usage.
type MemoryChecker struct {
	threshold float64 // Percentage (0-100)
	severity  health.Severity
}

// MemoryOption is a functional option for MemoryChecker.
type MemoryOption func(*MemoryChecker)

// WithMemoryThreshold sets the threshold percentage.
func WithMemoryThreshold(t float64) MemoryOption {
	return func(c *MemoryChecker) {
		c.threshold = t
	}
}

// WithMemorySeverity sets the severity level.
func WithMemorySeverity(s health.Severity) MemoryOption {
	return func(c *MemoryChecker) {
		c.severity = s
	}
}

// NewMemoryChecker creates a new memory health checker.
func NewMemoryChecker(opts ...MemoryOption) *MemoryChecker {
	c := &MemoryChecker{
		threshold: 90, // Default: warn if > 90% usage
		severity:  health.SeverityWarning,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Name returns the name of this health check.
func (c *MemoryChecker) Name() string {
	return "memory"
}

// Severity returns the severity level of this check.
func (c *MemoryChecker) Severity() health.Severity {
	return c.severity
}

// Check performs the memory health check.
func (c *MemoryChecker) Check(ctx context.Context) health.CheckResult {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := float64(m.Alloc) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024
	usagePercent := (float64(m.Alloc) / float64(m.Sys)) * 100

	status := health.StatusHealthy
	var message string
	if usagePercent > c.threshold {
		status = health.StatusDegraded
		message = fmt.Sprintf("memory usage %.2f%% exceeds threshold %.2f%%", usagePercent, c.threshold)
	}

	return health.CheckResult{
		Status:  status,
		Message: message,
		Details: map[string]any{
			"alloc_mb":      fmt.Sprintf("%.2f", allocMB),
			"sys_mb":        fmt.Sprintf("%.2f", sysMB),
			"usage_percent": fmt.Sprintf("%.2f", usagePercent),
			"num_gc":        m.NumGC,
			"goroutines":    runtime.NumGoroutine(),
		},
	}
}

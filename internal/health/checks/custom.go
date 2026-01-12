package checks

import (
	"context"

	"github.com/bargom/codeai/internal/health"
)

// CustomChecker allows defining custom health checks with a function.
type CustomChecker struct {
	name     string
	checkFn  func(ctx context.Context) health.CheckResult
	severity health.Severity
}

// CustomOption is a functional option for CustomChecker.
type CustomOption func(*CustomChecker)

// WithCustomSeverity sets the severity level.
func WithCustomSeverity(s health.Severity) CustomOption {
	return func(c *CustomChecker) {
		c.severity = s
	}
}

// NewCustomChecker creates a new custom health checker.
func NewCustomChecker(name string, checkFn func(ctx context.Context) health.CheckResult, opts ...CustomOption) *CustomChecker {
	c := &CustomChecker{
		name:     name,
		checkFn:  checkFn,
		severity: health.SeverityWarning,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Name returns the name of this health check.
func (c *CustomChecker) Name() string {
	return c.name
}

// Severity returns the severity level of this check.
func (c *CustomChecker) Severity() health.Severity {
	return c.severity
}

// Check performs the custom health check.
func (c *CustomChecker) Check(ctx context.Context) health.CheckResult {
	return c.checkFn(ctx)
}

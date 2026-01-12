// Package health provides health check functionality for the application.
package health

import (
	"context"
	"time"
)

// Status represents the health status of a component.
type Status string

const (
	// StatusHealthy indicates the component is functioning normally.
	StatusHealthy Status = "healthy"
	// StatusUnhealthy indicates the component is not functioning.
	StatusUnhealthy Status = "unhealthy"
	// StatusDegraded indicates the component is functioning but with issues.
	StatusDegraded Status = "degraded"
)

// Severity represents the severity level of a health check.
type Severity string

const (
	// SeverityCritical affects readiness (affects /ready endpoint).
	SeverityCritical Severity = "critical"
	// SeverityWarning is logged but doesn't affect readiness.
	SeverityWarning Severity = "warning"
)

// Response represents a health check response.
type Response struct {
	Status    Status                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version,omitempty"`
	Uptime    string                 `json:"uptime,omitempty"`
	Checks    map[string]CheckResult `json:"checks,omitempty"`
}

// CheckResult represents the result of an individual health check.
type CheckResult struct {
	Status   Status         `json:"status"`
	Message  string         `json:"message,omitempty"`
	Duration time.Duration  `json:"duration,omitempty"`
	Details  map[string]any `json:"details,omitempty"`
}

// Checker is the interface that health checks must implement.
type Checker interface {
	// Name returns the name of the health check.
	Name() string
	// Check performs the health check and returns the result.
	Check(ctx context.Context) CheckResult
	// Severity returns the severity level of this check.
	Severity() Severity
}

// CheckerFunc is a function adapter for simple health checks.
type CheckerFunc func(ctx context.Context) CheckResult

// LiveResponse represents a liveness check response.
type LiveResponse struct {
	Status string `json:"status"`
}

// ReadyResponse represents a readiness check response.
type ReadyResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

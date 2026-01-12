// Package checks provides built-in health checkers.
package checks

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bargom/codeai/internal/health"
)

// DatabaseChecker checks database connectivity.
type DatabaseChecker struct {
	db       *sql.DB
	timeout  time.Duration
	severity health.Severity
}

// DatabaseOption is a functional option for DatabaseChecker.
type DatabaseOption func(*DatabaseChecker)

// WithDatabaseTimeout sets the ping timeout.
func WithDatabaseTimeout(d time.Duration) DatabaseOption {
	return func(c *DatabaseChecker) {
		c.timeout = d
	}
}

// WithDatabaseSeverity sets the severity level.
func WithDatabaseSeverity(s health.Severity) DatabaseOption {
	return func(c *DatabaseChecker) {
		c.severity = s
	}
}

// NewDatabaseChecker creates a new database health checker.
func NewDatabaseChecker(db *sql.DB, opts ...DatabaseOption) *DatabaseChecker {
	c := &DatabaseChecker{
		db:       db,
		timeout:  2 * time.Second,
		severity: health.SeverityCritical,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Name returns the name of this health check.
func (c *DatabaseChecker) Name() string {
	return "database"
}

// Severity returns the severity level of this check.
func (c *DatabaseChecker) Severity() health.Severity {
	return c.severity
}

// Check performs the database health check.
func (c *DatabaseChecker) Check(ctx context.Context) health.CheckResult {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	if err := c.db.PingContext(ctx); err != nil {
		return health.CheckResult{
			Status:  health.StatusUnhealthy,
			Message: fmt.Sprintf("database ping failed: %v", err),
		}
	}

	stats := c.db.Stats()
	return health.CheckResult{
		Status: health.StatusHealthy,
		Details: map[string]any{
			"max_connections":  stats.MaxOpenConnections,
			"open_connections": stats.OpenConnections,
			"in_use":           stats.InUse,
			"idle":             stats.Idle,
		},
	}
}

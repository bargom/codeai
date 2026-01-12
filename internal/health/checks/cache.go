package checks

import (
	"context"
	"fmt"
	"time"

	"github.com/bargom/codeai/internal/health"
)

// Pinger is an interface for services that can be pinged.
type Pinger interface {
	Ping(ctx context.Context) error
}

// CacheChecker checks cache (Redis) connectivity.
type CacheChecker struct {
	cache    Pinger
	timeout  time.Duration
	severity health.Severity
}

// CacheOption is a functional option for CacheChecker.
type CacheOption func(*CacheChecker)

// WithCacheTimeout sets the ping timeout.
func WithCacheTimeout(d time.Duration) CacheOption {
	return func(c *CacheChecker) {
		c.timeout = d
	}
}

// WithCacheSeverity sets the severity level.
func WithCacheSeverity(s health.Severity) CacheOption {
	return func(c *CacheChecker) {
		c.severity = s
	}
}

// NewCacheChecker creates a new cache health checker.
func NewCacheChecker(cache Pinger, opts ...CacheOption) *CacheChecker {
	c := &CacheChecker{
		cache:    cache,
		timeout:  1 * time.Second,
		severity: health.SeverityCritical,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Name returns the name of this health check.
func (c *CacheChecker) Name() string {
	return "cache"
}

// Severity returns the severity level of this check.
func (c *CacheChecker) Severity() health.Severity {
	return c.severity
}

// Check performs the cache health check.
func (c *CacheChecker) Check(ctx context.Context) health.CheckResult {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	if err := c.cache.Ping(ctx); err != nil {
		return health.CheckResult{
			Status:  health.StatusUnhealthy,
			Message: fmt.Sprintf("cache ping failed: %v", err),
		}
	}

	return health.CheckResult{
		Status: health.StatusHealthy,
	}
}

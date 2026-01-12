package checks

import (
	"context"
	"fmt"

	"github.com/bargom/codeai/internal/health"
	"golang.org/x/sys/unix"
)

// DiskChecker checks disk space availability.
type DiskChecker struct {
	path      string
	threshold float64 // Percentage free space threshold (0-100)
	severity  health.Severity
}

// DiskOption is a functional option for DiskChecker.
type DiskOption func(*DiskChecker)

// WithDiskThreshold sets the free space threshold percentage.
// If free space falls below this threshold, status will be degraded.
func WithDiskThreshold(t float64) DiskOption {
	return func(c *DiskChecker) {
		c.threshold = t
	}
}

// WithDiskSeverity sets the severity level.
func WithDiskSeverity(s health.Severity) DiskOption {
	return func(c *DiskChecker) {
		c.severity = s
	}
}

// NewDiskChecker creates a new disk space health checker.
func NewDiskChecker(path string, opts ...DiskOption) *DiskChecker {
	c := &DiskChecker{
		path:      path,
		threshold: 10, // Default: warn if < 10% free
		severity:  health.SeverityWarning,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Name returns the name of this health check.
func (c *DiskChecker) Name() string {
	return "disk"
}

// Severity returns the severity level of this check.
func (c *DiskChecker) Severity() health.Severity {
	return c.severity
}

// Check performs the disk space health check.
func (c *DiskChecker) Check(ctx context.Context) health.CheckResult {
	var stat unix.Statfs_t
	if err := unix.Statfs(c.path, &stat); err != nil {
		return health.CheckResult{
			Status:  health.StatusUnhealthy,
			Message: fmt.Sprintf("failed to get disk stats for %s: %v", c.path, err),
		}
	}

	totalBytes := stat.Blocks * uint64(stat.Bsize)
	freeBytes := stat.Bavail * uint64(stat.Bsize)
	usedBytes := totalBytes - freeBytes
	freePercent := (float64(freeBytes) / float64(totalBytes)) * 100
	usedPercent := (float64(usedBytes) / float64(totalBytes)) * 100

	status := health.StatusHealthy
	var message string
	if freePercent < c.threshold {
		status = health.StatusDegraded
		message = fmt.Sprintf("disk free space %.2f%% below threshold %.2f%%", freePercent, c.threshold)
	}

	return health.CheckResult{
		Status:  status,
		Message: message,
		Details: map[string]any{
			"path":         c.path,
			"total_gb":     fmt.Sprintf("%.2f", float64(totalBytes)/1024/1024/1024),
			"free_gb":      fmt.Sprintf("%.2f", float64(freeBytes)/1024/1024/1024),
			"used_gb":      fmt.Sprintf("%.2f", float64(usedBytes)/1024/1024/1024),
			"free_percent": fmt.Sprintf("%.2f", freePercent),
			"used_percent": fmt.Sprintf("%.2f", usedPercent),
		},
	}
}

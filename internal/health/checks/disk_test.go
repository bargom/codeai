package checks

import (
	"context"
	"testing"

	"github.com/bargom/codeai/internal/health"
	"github.com/stretchr/testify/assert"
)

func TestDiskChecker(t *testing.T) {
	t.Run("healthy disk", func(t *testing.T) {
		// Check root path which should always exist
		checker := NewDiskChecker("/")

		result := checker.Check(context.Background())

		assert.Equal(t, health.StatusHealthy, result.Status)
		assert.NotNil(t, result.Details)
	})

	t.Run("invalid path returns unhealthy", func(t *testing.T) {
		checker := NewDiskChecker("/nonexistent/path/that/does/not/exist")

		result := checker.Check(context.Background())

		assert.Equal(t, health.StatusUnhealthy, result.Status)
		assert.Contains(t, result.Message, "failed to get disk stats")
	})

	t.Run("name returns disk", func(t *testing.T) {
		checker := NewDiskChecker("/")
		assert.Equal(t, "disk", checker.Name())
	})

	t.Run("default severity is warning", func(t *testing.T) {
		checker := NewDiskChecker("/")
		assert.Equal(t, health.SeverityWarning, checker.Severity())
	})

	t.Run("custom threshold", func(t *testing.T) {
		checker := NewDiskChecker("/", WithDiskThreshold(20))
		assert.Equal(t, 20.0, checker.threshold)
	})

	t.Run("custom severity", func(t *testing.T) {
		checker := NewDiskChecker("/", WithDiskSeverity(health.SeverityCritical))
		assert.Equal(t, health.SeverityCritical, checker.Severity())
	})

	t.Run("returns details", func(t *testing.T) {
		checker := NewDiskChecker("/")
		result := checker.Check(context.Background())

		if result.Status == health.StatusHealthy || result.Status == health.StatusDegraded {
			assert.Contains(t, result.Details, "path")
			assert.Contains(t, result.Details, "total_gb")
			assert.Contains(t, result.Details, "free_gb")
			assert.Contains(t, result.Details, "used_gb")
			assert.Contains(t, result.Details, "free_percent")
			assert.Contains(t, result.Details, "used_percent")
		}
	})

	t.Run("degraded when under threshold", func(t *testing.T) {
		// Set an extremely high threshold that will always trigger degraded
		checker := NewDiskChecker("/", WithDiskThreshold(99.99))

		result := checker.Check(context.Background())

		// Unless disk is 99.99% free (unlikely), it should be degraded
		assert.Equal(t, health.StatusDegraded, result.Status)
		assert.Contains(t, result.Message, "below threshold")
	})
}

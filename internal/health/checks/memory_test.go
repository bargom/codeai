package checks

import (
	"context"
	"testing"

	"github.com/bargom/codeai/internal/health"
	"github.com/stretchr/testify/assert"
)

func TestMemoryChecker(t *testing.T) {
	t.Run("returns healthy by default", func(t *testing.T) {
		checker := NewMemoryChecker()

		result := checker.Check(context.Background())

		// With default 90% threshold, normal memory usage should be healthy
		assert.Equal(t, health.StatusHealthy, result.Status)
		assert.NotNil(t, result.Details)
	})

	t.Run("name returns memory", func(t *testing.T) {
		checker := NewMemoryChecker()
		assert.Equal(t, "memory", checker.Name())
	})

	t.Run("default severity is warning", func(t *testing.T) {
		checker := NewMemoryChecker()
		assert.Equal(t, health.SeverityWarning, checker.Severity())
	})

	t.Run("custom threshold", func(t *testing.T) {
		checker := NewMemoryChecker(WithMemoryThreshold(50))
		assert.Equal(t, 50.0, checker.threshold)
	})

	t.Run("custom severity", func(t *testing.T) {
		checker := NewMemoryChecker(WithMemorySeverity(health.SeverityCritical))
		assert.Equal(t, health.SeverityCritical, checker.Severity())
	})

	t.Run("returns details", func(t *testing.T) {
		checker := NewMemoryChecker()
		result := checker.Check(context.Background())

		assert.Contains(t, result.Details, "alloc_mb")
		assert.Contains(t, result.Details, "sys_mb")
		assert.Contains(t, result.Details, "usage_percent")
		assert.Contains(t, result.Details, "num_gc")
		assert.Contains(t, result.Details, "goroutines")
	})

	t.Run("degraded when over threshold", func(t *testing.T) {
		// Set an extremely low threshold that will always be exceeded
		checker := NewMemoryChecker(WithMemoryThreshold(0.001))

		result := checker.Check(context.Background())

		assert.Equal(t, health.StatusDegraded, result.Status)
		assert.Contains(t, result.Message, "exceeds threshold")
	})
}

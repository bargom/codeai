package checks

import (
	"context"
	"testing"

	"github.com/bargom/codeai/internal/health"
	"github.com/stretchr/testify/assert"
)

func TestCustomChecker(t *testing.T) {
	t.Run("executes custom function", func(t *testing.T) {
		called := false
		checker := NewCustomChecker("test", func(ctx context.Context) health.CheckResult {
			called = true
			return health.CheckResult{Status: health.StatusHealthy}
		})

		result := checker.Check(context.Background())

		assert.True(t, called)
		assert.Equal(t, health.StatusHealthy, result.Status)
	})

	t.Run("name returns configured name", func(t *testing.T) {
		checker := NewCustomChecker("my-check", func(ctx context.Context) health.CheckResult {
			return health.CheckResult{Status: health.StatusHealthy}
		})
		assert.Equal(t, "my-check", checker.Name())
	})

	t.Run("default severity is warning", func(t *testing.T) {
		checker := NewCustomChecker("test", func(ctx context.Context) health.CheckResult {
			return health.CheckResult{Status: health.StatusHealthy}
		})
		assert.Equal(t, health.SeverityWarning, checker.Severity())
	})

	t.Run("custom severity", func(t *testing.T) {
		checker := NewCustomChecker("test", func(ctx context.Context) health.CheckResult {
			return health.CheckResult{Status: health.StatusHealthy}
		}, WithCustomSeverity(health.SeverityCritical))
		assert.Equal(t, health.SeverityCritical, checker.Severity())
	})

	t.Run("returns custom result", func(t *testing.T) {
		checker := NewCustomChecker("test", func(ctx context.Context) health.CheckResult {
			return health.CheckResult{
				Status:  health.StatusDegraded,
				Message: "custom message",
				Details: map[string]any{"key": "value"},
			}
		})

		result := checker.Check(context.Background())

		assert.Equal(t, health.StatusDegraded, result.Status)
		assert.Equal(t, "custom message", result.Message)
		assert.Equal(t, "value", result.Details["key"])
	})

	t.Run("receives context", func(t *testing.T) {
		type ctxKey string
		key := ctxKey("test")

		checker := NewCustomChecker("test", func(ctx context.Context) health.CheckResult {
			if ctx.Value(key) == "value" {
				return health.CheckResult{Status: health.StatusHealthy}
			}
			return health.CheckResult{Status: health.StatusUnhealthy}
		})

		ctx := context.WithValue(context.Background(), key, "value")
		result := checker.Check(ctx)

		assert.Equal(t, health.StatusHealthy, result.Status)
	})
}

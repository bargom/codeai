package checks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/health"
	"github.com/stretchr/testify/assert"
)

// mockCache implements Pinger for testing
type mockCache struct {
	pingErr error
}

func (m *mockCache) Ping(ctx context.Context) error {
	return m.pingErr
}

func TestCacheChecker(t *testing.T) {
	t.Run("healthy cache", func(t *testing.T) {
		cache := &mockCache{pingErr: nil}
		checker := NewCacheChecker(cache)

		result := checker.Check(context.Background())

		assert.Equal(t, health.StatusHealthy, result.Status)
		assert.Empty(t, result.Message)
	})

	t.Run("unhealthy cache", func(t *testing.T) {
		cache := &mockCache{pingErr: errors.New("connection refused")}
		checker := NewCacheChecker(cache)

		result := checker.Check(context.Background())

		assert.Equal(t, health.StatusUnhealthy, result.Status)
		assert.Contains(t, result.Message, "cache ping failed")
		assert.Contains(t, result.Message, "connection refused")
	})

	t.Run("name returns cache", func(t *testing.T) {
		cache := &mockCache{}
		checker := NewCacheChecker(cache)
		assert.Equal(t, "cache", checker.Name())
	})

	t.Run("default severity is critical", func(t *testing.T) {
		cache := &mockCache{}
		checker := NewCacheChecker(cache)
		assert.Equal(t, health.SeverityCritical, checker.Severity())
	})

	t.Run("custom timeout", func(t *testing.T) {
		cache := &mockCache{}
		checker := NewCacheChecker(cache, WithCacheTimeout(5*time.Second))
		assert.Equal(t, 5*time.Second, checker.timeout)
	})

	t.Run("custom severity", func(t *testing.T) {
		cache := &mockCache{}
		checker := NewCacheChecker(cache, WithCacheSeverity(health.SeverityWarning))
		assert.Equal(t, health.SeverityWarning, checker.Severity())
	})
}

// slowCache simulates a slow response
type slowCache struct {
	delay time.Duration
}

func (s *slowCache) Ping(ctx context.Context) error {
	select {
	case <-time.After(s.delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestCacheCheckerTimeout(t *testing.T) {
	t.Run("respects context timeout", func(t *testing.T) {
		cache := &slowCache{delay: 10 * time.Second}
		checker := NewCacheChecker(cache, WithCacheTimeout(50*time.Millisecond))

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()
		result := checker.Check(ctx)
		elapsed := time.Since(start)

		// Should timeout quickly
		assert.Less(t, elapsed, 500*time.Millisecond)
		assert.Equal(t, health.StatusUnhealthy, result.Status)
	})
}

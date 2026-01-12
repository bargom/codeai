package health

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChecker is a mock health checker for testing.
type mockChecker struct {
	name      string
	severity  Severity
	result    CheckResult
	delay     time.Duration
	callCount int64
}

func (m *mockChecker) Name() string {
	return m.name
}

func (m *mockChecker) Severity() Severity {
	return m.severity
}

func (m *mockChecker) Check(ctx context.Context) CheckResult {
	atomic.AddInt64(&m.callCount, 1)
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: "timeout",
			}
		}
	}
	return m.result
}

func (m *mockChecker) CallCount() int64 {
	return atomic.LoadInt64(&m.callCount)
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry("1.0.0")

	assert.NotNil(t, r)
	assert.Equal(t, "1.0.0", r.Version())
	assert.Empty(t, r.Checkers())
	assert.WithinDuration(t, time.Now(), r.StartTime(), time.Second)
}

func TestRegistryRegister(t *testing.T) {
	r := NewRegistry("1.0.0")

	checker1 := &mockChecker{name: "test1", severity: SeverityCritical}
	checker2 := &mockChecker{name: "test2", severity: SeverityWarning}

	r.Register(checker1)
	r.Register(checker2)

	checkers := r.Checkers()
	assert.Len(t, checkers, 2)
	assert.Equal(t, "test1", checkers[0].Name())
	assert.Equal(t, "test2", checkers[1].Name())
}

func TestRegistryLiveness(t *testing.T) {
	r := NewRegistry("1.0.0")

	// Liveness should always return healthy
	resp := r.Liveness(context.Background())

	assert.Equal(t, StatusHealthy, resp.Status)
	assert.Equal(t, "1.0.0", resp.Version)
	assert.NotEmpty(t, resp.Uptime)
	assert.WithinDuration(t, time.Now(), resp.Timestamp, time.Second)
}

func TestRegistryHealth(t *testing.T) {
	tests := []struct {
		name           string
		checkers       []*mockChecker
		expectedStatus Status
	}{
		{
			name:           "no checkers - healthy",
			checkers:       nil,
			expectedStatus: StatusHealthy,
		},
		{
			name: "all healthy",
			checkers: []*mockChecker{
				{name: "test1", severity: SeverityCritical, result: CheckResult{Status: StatusHealthy}},
				{name: "test2", severity: SeverityCritical, result: CheckResult{Status: StatusHealthy}},
			},
			expectedStatus: StatusHealthy,
		},
		{
			name: "one critical unhealthy",
			checkers: []*mockChecker{
				{name: "test1", severity: SeverityCritical, result: CheckResult{Status: StatusHealthy}},
				{name: "test2", severity: SeverityCritical, result: CheckResult{Status: StatusUnhealthy}},
			},
			expectedStatus: StatusUnhealthy,
		},
		{
			name: "one warning unhealthy",
			checkers: []*mockChecker{
				{name: "test1", severity: SeverityCritical, result: CheckResult{Status: StatusHealthy}},
				{name: "test2", severity: SeverityWarning, result: CheckResult{Status: StatusUnhealthy}},
			},
			expectedStatus: StatusDegraded,
		},
		{
			name: "one degraded",
			checkers: []*mockChecker{
				{name: "test1", severity: SeverityCritical, result: CheckResult{Status: StatusHealthy}},
				{name: "test2", severity: SeverityCritical, result: CheckResult{Status: StatusDegraded}},
			},
			expectedStatus: StatusDegraded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry("1.0.0")
			for _, c := range tt.checkers {
				r.Register(c)
			}

			resp := r.Health(context.Background())

			assert.Equal(t, tt.expectedStatus, resp.Status)
			assert.Len(t, resp.Checks, len(tt.checkers))
		})
	}
}

func TestRegistryReadiness(t *testing.T) {
	t.Run("runs only critical checks", func(t *testing.T) {
		r := NewRegistry("1.0.0")

		critical := &mockChecker{name: "critical", severity: SeverityCritical, result: CheckResult{Status: StatusHealthy}}
		warning := &mockChecker{name: "warning", severity: SeverityWarning, result: CheckResult{Status: StatusHealthy}}

		r.Register(critical)
		r.Register(warning)

		resp := r.Readiness(context.Background())

		assert.Equal(t, StatusHealthy, resp.Status)
		// Readiness only runs critical checks
		assert.Equal(t, int64(1), critical.CallCount())
		assert.Equal(t, int64(0), warning.CallCount())
	})

	t.Run("returns unhealthy if critical check fails", func(t *testing.T) {
		r := NewRegistry("1.0.0")

		critical := &mockChecker{
			name:     "critical",
			severity: SeverityCritical,
			result:   CheckResult{Status: StatusUnhealthy, Message: "db down"},
		}

		r.Register(critical)

		resp := r.Readiness(context.Background())

		assert.Equal(t, StatusUnhealthy, resp.Status)
		require.Contains(t, resp.Checks, "critical")
		assert.Equal(t, "db down", resp.Checks["critical"].Message)
	})
}

func TestRegistryParallelExecution(t *testing.T) {
	r := NewRegistry("1.0.0")

	// Create slow checkers
	checker1 := &mockChecker{
		name:     "slow1",
		severity: SeverityCritical,
		delay:    100 * time.Millisecond,
		result:   CheckResult{Status: StatusHealthy},
	}
	checker2 := &mockChecker{
		name:     "slow2",
		severity: SeverityCritical,
		delay:    100 * time.Millisecond,
		result:   CheckResult{Status: StatusHealthy},
	}

	r.Register(checker1)
	r.Register(checker2)

	start := time.Now()
	resp := r.Health(context.Background())
	elapsed := time.Since(start)

	// Should complete in ~100ms if parallel, ~200ms if sequential
	assert.Less(t, elapsed, 150*time.Millisecond)
	assert.Equal(t, StatusHealthy, resp.Status)
}

func TestRegistryTimeout(t *testing.T) {
	r := NewRegistry("1.0.0")

	// Create a checker that takes too long
	slow := &mockChecker{
		name:     "slow",
		severity: SeverityCritical,
		delay:    10 * time.Second, // Way longer than timeout
		result:   CheckResult{Status: StatusHealthy},
	}

	r.Register(slow)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	resp := r.Health(ctx)
	elapsed := time.Since(start)

	// Should timeout quickly
	assert.Less(t, elapsed, time.Second)
	assert.Equal(t, StatusUnhealthy, resp.Status)
}

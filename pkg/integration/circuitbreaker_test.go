package integration

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCircuitBreaker(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		cb := NewCircuitBreaker("test", CircuitBreakerConfig{})

		assert.Equal(t, "test", cb.Name())
		assert.Equal(t, StateClosed, cb.State())
		assert.Equal(t, 5, cb.config.FailureThreshold)
		assert.Equal(t, 60*time.Second, cb.config.Timeout)
		assert.Equal(t, 3, cb.config.HalfOpenRequests)
	})

	t.Run("custom config", func(t *testing.T) {
		config := CircuitBreakerConfig{
			FailureThreshold: 10,
			Timeout:          30 * time.Second,
			HalfOpenRequests: 5,
		}
		cb := NewCircuitBreaker("test", config)

		assert.Equal(t, 10, cb.config.FailureThreshold)
		assert.Equal(t, 30*time.Second, cb.config.Timeout)
		assert.Equal(t, 5, cb.config.HalfOpenRequests)
	})
}

func TestCircuitBreakerAllow(t *testing.T) {
	t.Run("closed state allows requests", func(t *testing.T) {
		cb := NewCircuitBreaker("test", DefaultCircuitBreakerConfig())

		err := cb.Allow()
		assert.NoError(t, err)
	})

	t.Run("open state blocks requests", func(t *testing.T) {
		cb := NewCircuitBreaker("test", CircuitBreakerConfig{
			FailureThreshold: 1,
			Timeout:          1 * time.Hour, // Long timeout to ensure it stays open
		})

		cb.RecordFailure()
		assert.Equal(t, StateOpen, cb.State())

		err := cb.Allow()
		assert.Equal(t, ErrCircuitOpen, err)
	})

	t.Run("half-open state allows requests", func(t *testing.T) {
		cb := NewCircuitBreaker("test", CircuitBreakerConfig{
			FailureThreshold: 1,
			Timeout:          1 * time.Millisecond, // Short timeout
		})

		cb.RecordFailure()
		assert.Equal(t, StateOpen, cb.State())

		// Wait for timeout to transition to half-open
		time.Sleep(5 * time.Millisecond)

		err := cb.Allow()
		assert.NoError(t, err)
		assert.Equal(t, StateHalfOpen, cb.State())
	})
}

func TestCircuitBreakerStateTransitions(t *testing.T) {
	t.Run("closed to open after threshold failures", func(t *testing.T) {
		cb := NewCircuitBreaker("test", CircuitBreakerConfig{
			FailureThreshold: 3,
			Timeout:          1 * time.Hour,
		})

		assert.Equal(t, StateClosed, cb.State())

		cb.RecordFailure()
		assert.Equal(t, StateClosed, cb.State())

		cb.RecordFailure()
		assert.Equal(t, StateClosed, cb.State())

		cb.RecordFailure()
		assert.Equal(t, StateOpen, cb.State())
	})

	t.Run("open to half-open after timeout", func(t *testing.T) {
		cb := NewCircuitBreaker("test", CircuitBreakerConfig{
			FailureThreshold: 1,
			Timeout:          1 * time.Millisecond,
		})

		cb.RecordFailure()
		assert.Equal(t, StateOpen, cb.State())

		time.Sleep(5 * time.Millisecond)

		err := cb.Allow()
		assert.NoError(t, err)
		assert.Equal(t, StateHalfOpen, cb.State())
	})

	t.Run("half-open to closed after successful requests", func(t *testing.T) {
		cb := NewCircuitBreaker("test", CircuitBreakerConfig{
			FailureThreshold: 1,
			Timeout:          1 * time.Millisecond,
			HalfOpenRequests: 2,
		})

		cb.RecordFailure()
		time.Sleep(5 * time.Millisecond)

		cb.Allow() // Transitions to half-open
		assert.Equal(t, StateHalfOpen, cb.State())

		cb.RecordSuccess()
		assert.Equal(t, StateHalfOpen, cb.State())

		cb.RecordSuccess()
		assert.Equal(t, StateClosed, cb.State())
	})

	t.Run("half-open to open on failure", func(t *testing.T) {
		cb := NewCircuitBreaker("test", CircuitBreakerConfig{
			FailureThreshold: 1,
			Timeout:          1 * time.Millisecond,
			HalfOpenRequests: 5,
		})

		cb.RecordFailure()
		time.Sleep(5 * time.Millisecond)

		cb.Allow() // Transitions to half-open
		assert.Equal(t, StateHalfOpen, cb.State())

		cb.RecordFailure()
		assert.Equal(t, StateOpen, cb.State())
	})
}

func TestCircuitBreakerExecute(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		cb := NewCircuitBreaker("test", DefaultCircuitBreakerConfig())

		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})

		assert.NoError(t, err)
	})

	t.Run("failed execution records failure", func(t *testing.T) {
		cb := NewCircuitBreaker("test", CircuitBreakerConfig{
			FailureThreshold: 2,
		})

		expectedErr := errors.New("test error")
		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			return expectedErr
		})

		assert.Equal(t, expectedErr, err)

		stats := cb.Stats()
		assert.Equal(t, 1, stats.Failures)
	})

	t.Run("blocks when circuit is open", func(t *testing.T) {
		cb := NewCircuitBreaker("test", CircuitBreakerConfig{
			FailureThreshold: 1,
			Timeout:          1 * time.Hour,
		})

		cb.RecordFailure()

		executed := false
		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			executed = true
			return nil
		})

		assert.Equal(t, ErrCircuitOpen, err)
		assert.False(t, executed)
	})
}

func TestCircuitBreakerRecordSuccess(t *testing.T) {
	t.Run("resets failure count in closed state", func(t *testing.T) {
		cb := NewCircuitBreaker("test", CircuitBreakerConfig{
			FailureThreshold: 5,
		})

		cb.RecordFailure()
		cb.RecordFailure()
		assert.Equal(t, 2, int(cb.failures.Load()))

		cb.RecordSuccess()
		assert.Equal(t, 0, int(cb.failures.Load()))
	})
}

func TestCircuitBreakerStats(t *testing.T) {
	cb := NewCircuitBreaker("test", DefaultCircuitBreakerConfig())

	cb.RecordFailure()
	cb.RecordFailure()

	stats := cb.Stats()

	assert.Equal(t, "test", stats.Name)
	assert.Equal(t, StateClosed, stats.State)
	assert.Equal(t, 2, stats.Failures)
	assert.False(t, stats.LastFailure.IsZero())
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := NewCircuitBreaker("test", CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:          1 * time.Hour,
	})

	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())

	cb.Reset()

	assert.Equal(t, StateClosed, cb.State())
	assert.Equal(t, 0, int(cb.failures.Load()))
}

func TestCircuitBreakerOnStateChange(t *testing.T) {
	var stateChanges []struct {
		from, to CircuitState
	}

	cb := NewCircuitBreaker("test", CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:          1 * time.Millisecond,
		HalfOpenRequests: 1,
		OnStateChange: func(from, to CircuitState) {
			stateChanges = append(stateChanges, struct{ from, to CircuitState }{from, to})
		},
	})

	cb.RecordFailure()
	time.Sleep(5 * time.Millisecond)
	cb.Allow()
	cb.RecordSuccess()

	require.Len(t, stateChanges, 3)
	assert.Equal(t, StateClosed, stateChanges[0].from)
	assert.Equal(t, StateOpen, stateChanges[0].to)
	assert.Equal(t, StateOpen, stateChanges[1].from)
	assert.Equal(t, StateHalfOpen, stateChanges[1].to)
	assert.Equal(t, StateHalfOpen, stateChanges[2].from)
	assert.Equal(t, StateClosed, stateChanges[2].to)
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	cb := NewCircuitBreaker("test", CircuitBreakerConfig{
		FailureThreshold: 100,
		Timeout:          1 * time.Hour,
	})

	var wg sync.WaitGroup
	iterations := 1000

	// Concurrent failures
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.RecordFailure()
		}()
	}

	wg.Wait()

	// Should transition to open
	assert.Equal(t, StateOpen, cb.State())
}

func TestCircuitBreakerRegistry(t *testing.T) {
	t.Run("get creates new circuit breaker", func(t *testing.T) {
		registry := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())

		cb := registry.Get("service1")
		assert.NotNil(t, cb)
		assert.Equal(t, "service1", cb.Name())
	})

	t.Run("get returns same circuit breaker", func(t *testing.T) {
		registry := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())

		cb1 := registry.Get("service1")
		cb2 := registry.Get("service1")

		assert.Same(t, cb1, cb2)
	})

	t.Run("register with custom config", func(t *testing.T) {
		registry := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())

		customConfig := CircuitBreakerConfig{
			FailureThreshold: 10,
		}
		cb := registry.Register("service1", customConfig)

		assert.Equal(t, 10, cb.config.FailureThreshold)
	})

	t.Run("all returns all circuit breakers", func(t *testing.T) {
		registry := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())

		registry.Get("service1")
		registry.Get("service2")
		registry.Get("service3")

		all := registry.All()
		assert.Len(t, all, 3)
	})

	t.Run("stats returns all stats", func(t *testing.T) {
		registry := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())

		registry.Get("service1").RecordFailure()
		registry.Get("service2").RecordFailure()

		stats := registry.Stats()
		assert.Len(t, stats, 2)
	})

	t.Run("concurrent access", func(t *testing.T) {
		registry := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())

		var wg sync.WaitGroup
		var count atomic.Int32

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				name := "service"
				if i%10 == 0 {
					name = "other-service"
				}
				cb := registry.Get(name)
				if cb != nil {
					count.Add(1)
				}
			}(i)
		}

		wg.Wait()
		assert.Equal(t, int32(100), count.Load())
	})
}

func TestCircuitStateString(t *testing.T) {
	assert.Equal(t, "closed", StateClosed.String())
	assert.Equal(t, "open", StateOpen.String())
	assert.Equal(t, "half-open", StateHalfOpen.String())
	assert.Equal(t, "unknown", CircuitState(99).String())
}

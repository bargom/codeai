package shutdown

import (
	"context"
	"errors"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, 30*time.Second, cfg.OverallTimeout)
	assert.Equal(t, 10*time.Second, cfg.PerHookTimeout)
	assert.Equal(t, 10*time.Second, cfg.DrainTimeout)
	assert.Equal(t, 5*time.Second, cfg.SlowHookThreshold)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name     string
		input    Config
		expected Config
	}{
		{
			name:     "zero values get defaults",
			input:    Config{},
			expected: DefaultConfig(),
		},
		{
			name: "negative values get defaults",
			input: Config{
				OverallTimeout:    -1,
				PerHookTimeout:    -1,
				DrainTimeout:      -1,
				SlowHookThreshold: -1,
			},
			expected: DefaultConfig(),
		},
		{
			name: "valid values preserved",
			input: Config{
				OverallTimeout:    60 * time.Second,
				PerHookTimeout:    20 * time.Second,
				DrainTimeout:      15 * time.Second,
				SlowHookThreshold: 10 * time.Second,
			},
			expected: Config{
				OverallTimeout:    60 * time.Second,
				PerHookTimeout:    20 * time.Second,
				DrainTimeout:      15 * time.Second,
				SlowHookThreshold: 10 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.Validate()
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestRegistry(t *testing.T) {
	t.Run("register and count", func(t *testing.T) {
		r := NewRegistry()
		assert.Equal(t, 0, r.Count())

		r.Register("hook1", 10, func(ctx context.Context) error { return nil })
		assert.Equal(t, 1, r.Count())

		r.Register("hook2", 20, func(ctx context.Context) error { return nil })
		assert.Equal(t, 2, r.Count())
	})

	t.Run("hooks returns copy", func(t *testing.T) {
		r := NewRegistry()
		r.Register("hook1", 10, func(ctx context.Context) error { return nil })

		hooks := r.Hooks()
		assert.Len(t, hooks, 1)

		// Modifying returned slice shouldn't affect registry
		hooks = append(hooks, Hook{Name: "hook2"})
		assert.Len(t, r.Hooks(), 1)
	})

	t.Run("clear", func(t *testing.T) {
		r := NewRegistry()
		r.Register("hook1", 10, func(ctx context.Context) error { return nil })
		r.Register("hook2", 20, func(ctx context.Context) error { return nil })

		r.Clear()
		assert.Equal(t, 0, r.Count())
	})
}

func TestManager(t *testing.T) {
	t.Run("hook execution order by priority", func(t *testing.T) {
		m := NewManagerWithDefaults()

		var order []string
		ch := make(chan string, 3)

		m.Register("low", 10, func(ctx context.Context) error {
			ch <- "low"
			return nil
		})
		m.Register("high", 90, func(ctx context.Context) error {
			ch <- "high"
			return nil
		})
		m.Register("medium", 50, func(ctx context.Context) error {
			ch <- "medium"
			return nil
		})

		m.Shutdown()

		close(ch)
		for s := range ch {
			order = append(order, s)
		}

		// High priority should execute first
		assert.Equal(t, "high", order[0])
		assert.Equal(t, "medium", order[1])
		assert.Equal(t, "low", order[2])
	})

	t.Run("hooks with same priority execute concurrently", func(t *testing.T) {
		m := NewManagerWithDefaults()

		var started atomic.Int32
		var finished atomic.Int32
		barrier := make(chan struct{})

		for i := 0; i < 3; i++ {
			m.Register("hook", 50, func(ctx context.Context) error {
				started.Add(1)
				<-barrier // Wait at barrier
				finished.Add(1)
				return nil
			})
		}

		done := make(chan struct{})
		go func() {
			m.Shutdown()
			close(done)
		}()

		// Wait for all hooks to start
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, int32(3), started.Load())
		assert.Equal(t, int32(0), finished.Load())

		// Release barrier
		close(barrier)

		<-done
		assert.Equal(t, int32(3), finished.Load())
	})

	t.Run("shutdown only once", func(t *testing.T) {
		m := NewManagerWithDefaults()

		var count atomic.Int32
		m.Register("counter", 50, func(ctx context.Context) error {
			count.Add(1)
			return nil
		})

		// Call shutdown multiple times
		m.Shutdown()
		m.Shutdown()
		m.Shutdown()

		assert.Equal(t, int32(1), count.Load())
	})

	t.Run("state transitions", func(t *testing.T) {
		m := NewManagerWithDefaults()
		assert.Equal(t, StateRunning, m.State())
		assert.False(t, m.IsShuttingDown())
		assert.False(t, m.IsShutdown())

		done := make(chan struct{})
		m.Register("blocker", 50, func(ctx context.Context) error {
			<-done
			return nil
		})

		go m.Shutdown()

		// Wait for shutdown to start
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, StateShuttingDown, m.State())
		assert.True(t, m.IsShuttingDown())
		assert.False(t, m.IsShutdown())

		close(done)
		m.Wait()

		assert.Equal(t, StateShutdown, m.State())
		assert.False(t, m.IsShuttingDown())
		assert.True(t, m.IsShutdown())
	})

	t.Run("error collection", func(t *testing.T) {
		m := NewManagerWithDefaults()

		expectedErr := errors.New("test error")
		m.Register("failing", 50, func(ctx context.Context) error {
			return expectedErr
		})
		m.Register("success", 50, func(ctx context.Context) error {
			return nil
		})

		m.Shutdown()

		errs := m.Errors()
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "test error")
	})

	t.Run("hook count", func(t *testing.T) {
		m := NewManagerWithDefaults()
		assert.Equal(t, 0, m.HookCount())

		m.Register("hook1", 50, func(ctx context.Context) error { return nil })
		assert.Equal(t, 1, m.HookCount())

		m.Register("hook2", 50, func(ctx context.Context) error { return nil })
		assert.Equal(t, 2, m.HookCount())
	})
}

func TestTimeout(t *testing.T) {
	t.Run("WithTimeout success", func(t *testing.T) {
		ctx := context.Background()
		err := WithTimeout(ctx, time.Second, "test", func(ctx context.Context) error {
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("WithTimeout timeout", func(t *testing.T) {
		ctx := context.Background()
		err := WithTimeout(ctx, 10*time.Millisecond, "test", func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})

		assert.Error(t, err)
		assert.True(t, IsTimeout(err))

		var timeoutErr *TimeoutError
		assert.True(t, errors.As(err, &timeoutErr))
		assert.Equal(t, "test", timeoutErr.Operation)
	})

	t.Run("WithTimeoutAndPanicRecovery panic", func(t *testing.T) {
		ctx := context.Background()
		err := WithTimeoutAndPanicRecovery(ctx, time.Second, "test", func(ctx context.Context) error {
			panic("test panic")
		})

		assert.Error(t, err)
		assert.True(t, IsPanic(err))

		var panicErr *PanicError
		assert.True(t, errors.As(err, &panicErr))
		assert.Equal(t, "test", panicErr.Operation)
		assert.Equal(t, "test panic", panicErr.Value)
	})

	t.Run("WithTimeoutAndPanicRecovery success", func(t *testing.T) {
		ctx := context.Background()
		err := WithTimeoutAndPanicRecovery(ctx, time.Second, "test", func(ctx context.Context) error {
			return nil
		})
		assert.NoError(t, err)
	})
}

func TestManagerPerHookTimeout(t *testing.T) {
	cfg := Config{
		OverallTimeout: 5 * time.Second,
		PerHookTimeout: 50 * time.Millisecond,
		DrainTimeout:   1 * time.Second,
	}
	cfg.Validate()

	m := NewManager(cfg, nil)

	m.Register("slow-hook", 50, func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	start := time.Now()
	m.Shutdown()
	elapsed := time.Since(start)

	// Should timeout quickly, not wait for the full 200ms
	assert.Less(t, elapsed, 150*time.Millisecond)

	errs := m.Errors()
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "timed out")
}

func TestManagerOverallTimeout(t *testing.T) {
	cfg := Config{
		OverallTimeout: 100 * time.Millisecond,
		PerHookTimeout: 1 * time.Second,
		DrainTimeout:   1 * time.Second,
	}
	cfg.Validate()

	m := NewManager(cfg, nil)

	// Register multiple hooks at different priorities
	m.Register("hook1", 90, func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	m.Register("hook2", 80, func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	m.Register("hook3", 70, func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	start := time.Now()
	m.Shutdown()
	elapsed := time.Since(start)

	// Should respect overall timeout
	assert.Less(t, elapsed, 200*time.Millisecond)
}

func TestManagerPanicRecovery(t *testing.T) {
	m := NewManagerWithDefaults()

	m.Register("panicking", 50, func(ctx context.Context) error {
		panic("test panic")
	})
	m.Register("normal", 40, func(ctx context.Context) error {
		return nil
	})

	// Should not panic
	m.Shutdown()

	errs := m.Errors()
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "panicked")
}

func TestSignalHandler(t *testing.T) {
	t.Run("default signals", func(t *testing.T) {
		h := NewSignalHandler()
		signals := h.Signals()
		assert.Len(t, signals, 3)
	})

	t.Run("custom signals", func(t *testing.T) {
		// Passing specific signals
		h := NewSignalHandler(syscall.SIGTERM, syscall.SIGINT)
		signals := h.Signals()
		assert.Len(t, signals, 2)
	})

	t.Run("stop", func(t *testing.T) {
		h := NewSignalHandler()
		ch := h.Listen()
		h.Stop()

		// Channel should be closed
		_, ok := <-ch
		assert.False(t, ok)
	})
}

func TestDrainer(t *testing.T) {
	t.Run("add and done", func(t *testing.T) {
		d := NewDrainer()
		assert.Equal(t, int64(0), d.Count())

		assert.True(t, d.Add())
		assert.Equal(t, int64(1), d.Count())

		assert.True(t, d.Add())
		assert.Equal(t, int64(2), d.Count())

		d.Done()
		assert.Equal(t, int64(1), d.Count())

		d.Done()
		assert.Equal(t, int64(0), d.Count())
	})

	t.Run("add rejected after drain started", func(t *testing.T) {
		d := NewDrainer()
		assert.True(t, d.Add())
		d.Done()

		d.StartDrain()
		assert.False(t, d.Add())
		assert.True(t, d.IsDraining())
	})

	t.Run("wait completes when empty", func(t *testing.T) {
		d := NewDrainer()
		d.StartDrain()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		err := d.Wait(ctx)
		assert.NoError(t, err)
	})

	t.Run("wait blocks until done", func(t *testing.T) {
		d := NewDrainer()
		d.Add()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		done := make(chan error)
		go func() {
			done <- d.Wait(ctx)
		}()

		// Should timeout
		err := <-done
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("wait completes when operations finish", func(t *testing.T) {
		d := NewDrainer()
		d.Add()

		go func() {
			time.Sleep(50 * time.Millisecond)
			d.Done()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		err := d.Wait(ctx)
		assert.NoError(t, err)
	})

	t.Run("WaitWithTimeout", func(t *testing.T) {
		d := NewDrainer()
		d.Add()

		err := d.WaitWithTimeout(50 * time.Millisecond)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

func TestGroupByPriority(t *testing.T) {
	tests := []struct {
		name     string
		hooks    []Hook
		expected int // number of groups
	}{
		{
			name:     "empty",
			hooks:    []Hook{},
			expected: 0,
		},
		{
			name: "single",
			hooks: []Hook{
				{Name: "h1", Priority: 50},
			},
			expected: 1,
		},
		{
			name: "same priority",
			hooks: []Hook{
				{Name: "h1", Priority: 50},
				{Name: "h2", Priority: 50},
				{Name: "h3", Priority: 50},
			},
			expected: 1,
		},
		{
			name: "different priorities",
			hooks: []Hook{
				{Name: "h1", Priority: 90},
				{Name: "h2", Priority: 80},
				{Name: "h3", Priority: 70},
			},
			expected: 3,
		},
		{
			name: "mixed priorities",
			hooks: []Hook{
				{Name: "h1", Priority: 90},
				{Name: "h2", Priority: 90},
				{Name: "h3", Priority: 80},
				{Name: "h4", Priority: 70},
				{Name: "h5", Priority: 70},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := groupByPriority(tt.hooks)
			assert.Len(t, groups, tt.expected)
		})
	}
}

func TestState(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateRunning, "running"},
		{StateShuttingDown, "shutting_down"},
		{StateShutdown, "shutdown"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

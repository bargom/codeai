// Package integration provides resilience patterns for external service communication.
package integration

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bargom/codeai/pkg/metrics"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int32

const (
	// StateClosed allows requests to pass through.
	StateClosed CircuitState = iota
	// StateOpen blocks all requests.
	StateOpen
	// StateHalfOpen allows limited requests to test recovery.
	StateHalfOpen
)

// String returns the string representation of the circuit state.
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ToMetricsState converts the CircuitState to a metrics state.
func (s CircuitState) ToMetricsState() metrics.CircuitBreakerState {
	switch s {
	case StateClosed:
		return metrics.CircuitBreakerClosed
	case StateOpen:
		return metrics.CircuitBreakerOpen
	case StateHalfOpen:
		return metrics.CircuitBreakerHalfOpen
	default:
		return metrics.CircuitBreakerClosed
	}
}

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreakerConfig configures a circuit breaker.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures before opening the circuit.
	// Default: 5
	FailureThreshold int

	// Timeout is the duration the circuit stays open before transitioning to half-open.
	// Default: 60s
	Timeout time.Duration

	// HalfOpenRequests is the number of successful requests required in half-open state
	// to transition back to closed.
	// Default: 3
	HalfOpenRequests int

	// OnStateChange is called when the circuit state changes.
	OnStateChange func(from, to CircuitState)
}

// DefaultCircuitBreakerConfig returns the default circuit breaker configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		Timeout:          60 * time.Second,
		HalfOpenRequests: 3,
	}
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	name   string
	config CircuitBreakerConfig
	logger *slog.Logger

	state             atomic.Int32
	failures          atomic.Int32
	halfOpenSuccesses atomic.Int32
	lastFailure       atomic.Int64
	lastStateChange   atomic.Int64

	mu sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker with the given name and configuration.
func NewCircuitBreaker(name string, config CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.Timeout <= 0 {
		config.Timeout = 60 * time.Second
	}
	if config.HalfOpenRequests <= 0 {
		config.HalfOpenRequests = 3
	}

	cb := &CircuitBreaker{
		name:   name,
		config: config,
		logger: slog.Default().With("component", "circuit_breaker", "service", name),
	}
	cb.state.Store(int32(StateClosed))
	cb.lastStateChange.Store(time.Now().UnixNano())

	return cb
}

// Name returns the name of the circuit breaker.
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitState {
	return CircuitState(cb.state.Load())
}

// Allow checks if a request is allowed to proceed.
// Returns nil if allowed, ErrCircuitOpen if the circuit is open.
func (cb *CircuitBreaker) Allow() error {
	state := cb.State()

	switch state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if timeout has elapsed to transition to half-open
		lastFailure := time.Unix(0, cb.lastFailure.Load())
		if time.Since(lastFailure) >= cb.config.Timeout {
			cb.transitionTo(StateHalfOpen)
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		return nil
	}

	return nil
}

// Execute runs the given function with circuit breaker protection.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if err := cb.Allow(); err != nil {
		return err
	}

	err := fn(ctx)
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	state := cb.State()

	switch state {
	case StateClosed:
		cb.failures.Store(0)

	case StateHalfOpen:
		successes := cb.halfOpenSuccesses.Add(1)
		if int(successes) >= cb.config.HalfOpenRequests {
			cb.transitionTo(StateClosed)
		}
	}
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	now := time.Now()
	cb.lastFailure.Store(now.UnixNano())

	state := cb.State()

	switch state {
	case StateClosed:
		failures := cb.failures.Add(1)
		if int(failures) >= cb.config.FailureThreshold {
			cb.transitionTo(StateOpen)
		}

	case StateHalfOpen:
		// Any failure in half-open state reopens the circuit
		cb.transitionTo(StateOpen)
	}
}

// transitionTo transitions the circuit breaker to a new state.
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := CircuitState(cb.state.Load())
	if oldState == newState {
		return
	}

	cb.state.Store(int32(newState))
	cb.lastStateChange.Store(time.Now().UnixNano())

	// Reset counters based on new state
	switch newState {
	case StateClosed:
		cb.failures.Store(0)
		cb.halfOpenSuccesses.Store(0)
	case StateHalfOpen:
		cb.halfOpenSuccesses.Store(0)
	case StateOpen:
		// Keep failure count
	}

	cb.logger.Info("circuit breaker state changed",
		"from", oldState.String(),
		"to", newState.String(),
	)

	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(oldState, newState)
	}

	// Update metrics
	if reg := metrics.Global(); reg != nil {
		reg.Integration().SetCircuitBreakerState(cb.name, newState.ToMetricsState())
	}
}

// Stats returns the current statistics of the circuit breaker.
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	return CircuitBreakerStats{
		Name:              cb.name,
		State:             cb.State(),
		Failures:          int(cb.failures.Load()),
		HalfOpenSuccesses: int(cb.halfOpenSuccesses.Load()),
		LastFailure:       time.Unix(0, cb.lastFailure.Load()),
		LastStateChange:   time.Unix(0, cb.lastStateChange.Load()),
	}
}

// Reset resets the circuit breaker to its initial closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state.Store(int32(StateClosed))
	cb.failures.Store(0)
	cb.halfOpenSuccesses.Store(0)
	cb.lastFailure.Store(0)
	cb.lastStateChange.Store(time.Now().UnixNano())

	cb.logger.Info("circuit breaker reset")

	if reg := metrics.Global(); reg != nil {
		reg.Integration().SetCircuitBreakerState(cb.name, metrics.CircuitBreakerClosed)
	}
}

// CircuitBreakerStats contains the current statistics of a circuit breaker.
type CircuitBreakerStats struct {
	Name              string
	State             CircuitState
	Failures          int
	HalfOpenSuccesses int
	LastFailure       time.Time
	LastStateChange   time.Time
}

// CircuitBreakerRegistry manages multiple circuit breakers.
type CircuitBreakerRegistry struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
	config   CircuitBreakerConfig
}

// NewCircuitBreakerRegistry creates a new circuit breaker registry.
func NewCircuitBreakerRegistry(defaultConfig CircuitBreakerConfig) *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		breakers: make(map[string]*CircuitBreaker),
		config:   defaultConfig,
	}
}

// Get returns the circuit breaker for the given service name, creating one if it doesn't exist.
func (r *CircuitBreakerRegistry) Get(name string) *CircuitBreaker {
	r.mu.RLock()
	cb, exists := r.breakers[name]
	r.mu.RUnlock()

	if exists {
		return cb
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists = r.breakers[name]; exists {
		return cb
	}

	cb = NewCircuitBreaker(name, r.config)
	r.breakers[name] = cb
	return cb
}

// Register registers a circuit breaker with a custom configuration.
func (r *CircuitBreakerRegistry) Register(name string, config CircuitBreakerConfig) *CircuitBreaker {
	r.mu.Lock()
	defer r.mu.Unlock()

	cb := NewCircuitBreaker(name, config)
	r.breakers[name] = cb
	return cb
}

// All returns all registered circuit breakers.
func (r *CircuitBreakerRegistry) All() map[string]*CircuitBreaker {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*CircuitBreaker, len(r.breakers))
	for k, v := range r.breakers {
		result[k] = v
	}
	return result
}

// Stats returns statistics for all circuit breakers.
func (r *CircuitBreakerRegistry) Stats() []CircuitBreakerStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make([]CircuitBreakerStats, 0, len(r.breakers))
	for _, cb := range r.breakers {
		stats = append(stats, cb.Stats())
	}
	return stats
}

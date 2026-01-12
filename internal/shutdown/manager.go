package shutdown

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"sync"
	"time"
)

// State represents the current state of the shutdown manager.
type State int

const (
	// StateRunning indicates the manager is running normally.
	StateRunning State = iota

	// StateShuttingDown indicates shutdown is in progress.
	StateShuttingDown

	// StateShutdown indicates shutdown is complete.
	StateShutdown
)

func (s State) String() string {
	switch s {
	case StateRunning:
		return "running"
	case StateShuttingDown:
		return "shutting_down"
	case StateShutdown:
		return "shutdown"
	default:
		return "unknown"
	}
}

// Manager coordinates graceful shutdown of all registered components.
type Manager struct {
	config        Config
	registry      *Registry
	signalHandler *SignalHandler
	logger        *slog.Logger

	state        State
	stateMu      sync.RWMutex
	shutdownOnce sync.Once
	done         chan struct{}
	errors       []error
	errorsMu     sync.Mutex
}

// NewManager creates a new shutdown manager with the given configuration.
func NewManager(cfg Config, logger *slog.Logger) *Manager {
	cfg.Validate()

	if logger == nil {
		logger = slog.Default()
	}

	return &Manager{
		config:        cfg,
		registry:      NewRegistry(),
		signalHandler: NewSignalHandler(),
		logger:        logger.With("component", "shutdown"),
		state:         StateRunning,
		done:          make(chan struct{}),
	}
}

// NewManagerWithDefaults creates a shutdown manager with default configuration.
func NewManagerWithDefaults() *Manager {
	return NewManager(DefaultConfig(), nil)
}

// Register adds a shutdown hook with the given name, priority, and function.
// Higher priority hooks are executed first.
func (m *Manager) Register(name string, priority int, fn HookFunc) {
	m.registry.Register(name, priority, fn)
	m.logger.Debug("registered shutdown hook", "name", name, "priority", priority)
}

// RegisterHook adds a Hook struct to the registry.
func (m *Manager) RegisterHook(hook Hook) {
	m.registry.RegisterHook(hook)
	m.logger.Debug("registered shutdown hook", "name", hook.Name, "priority", hook.Priority)
}

// ListenForSignals starts listening for OS shutdown signals and triggers
// shutdown when a signal is received.
// Returns a channel that is closed when shutdown is complete.
func (m *Manager) ListenForSignals() <-chan struct{} {
	sigChan := m.signalHandler.Listen()

	go func() {
		sig, ok := <-sigChan
		if ok {
			m.logger.Info("received shutdown signal", "signal", sig)
			m.Shutdown()
		}
	}()

	return m.done
}

// Shutdown initiates graceful shutdown. It is safe to call multiple times.
// The first call triggers the shutdown, subsequent calls are no-ops.
func (m *Manager) Shutdown() {
	m.shutdownOnce.Do(func() {
		m.setStateLocked(StateShuttingDown)
		m.logger.Info("starting graceful shutdown",
			"timeout", m.config.OverallTimeout,
			"hook_count", m.registry.Count())

		ctx, cancel := context.WithTimeout(context.Background(), m.config.OverallTimeout)
		defer cancel()

		m.executeHooks(ctx)

		m.setStateLocked(StateShutdown)
		m.logger.Info("graceful shutdown complete", "errors", len(m.errors))

		// Stop signal handler
		m.signalHandler.Stop()

		close(m.done)
	})
}

// ShutdownWithContext initiates graceful shutdown with an external context.
func (m *Manager) ShutdownWithContext(ctx context.Context) {
	m.shutdownOnce.Do(func() {
		m.setStateLocked(StateShuttingDown)
		m.logger.Info("starting graceful shutdown with context",
			"hook_count", m.registry.Count())

		m.executeHooks(ctx)

		m.setStateLocked(StateShutdown)
		m.logger.Info("graceful shutdown complete", "errors", len(m.errors))

		// Stop signal handler
		m.signalHandler.Stop()

		close(m.done)
	})
}

// executeHooks runs all registered hooks in priority order.
func (m *Manager) executeHooks(ctx context.Context) {
	hooks := m.registry.Hooks()

	// Sort hooks by priority (descending - higher priorities first)
	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].Priority > hooks[j].Priority
	})

	// Group hooks by priority for parallel execution within groups
	groups := groupByPriority(hooks)

	for _, group := range groups {
		m.executeHookGroup(ctx, group)

		// Check if context is done after each group
		if ctx.Err() != nil {
			m.logger.Warn("shutdown timeout exceeded, remaining hooks skipped",
				"remaining_groups", len(groups)-1)
			m.addError(fmt.Errorf("overall shutdown timeout exceeded"))
			break
		}
	}
}

// executeHookGroup executes all hooks in a group concurrently.
func (m *Manager) executeHookGroup(ctx context.Context, hooks []Hook) {
	if len(hooks) == 0 {
		return
	}

	m.logger.Debug("executing hook group",
		"priority", hooks[0].Priority,
		"count", len(hooks))

	var wg sync.WaitGroup

	for _, hook := range hooks {
		wg.Add(1)
		go func(h Hook) {
			defer wg.Done()
			m.executeHook(ctx, h)
		}(hook)
	}

	wg.Wait()
}

// executeHook executes a single hook with timeout and panic recovery.
func (m *Manager) executeHook(ctx context.Context, hook Hook) {
	start := time.Now()

	m.logger.Info("executing shutdown hook",
		"name", hook.Name,
		"priority", hook.Priority)

	err := WithTimeoutAndPanicRecovery(ctx, m.config.PerHookTimeout, hook.Name, hook.Fn)

	duration := time.Since(start)

	if duration > m.config.SlowHookThreshold {
		m.logger.Warn("slow shutdown hook",
			"name", hook.Name,
			"duration", duration,
			"threshold", m.config.SlowHookThreshold)
	}

	if err != nil {
		m.logger.Error("shutdown hook failed",
			"name", hook.Name,
			"error", err,
			"duration", duration)
		m.addError(fmt.Errorf("hook %s: %w", hook.Name, err))
	} else {
		m.logger.Info("shutdown hook completed",
			"name", hook.Name,
			"duration", duration)
	}
}

// groupByPriority groups hooks by their priority.
func groupByPriority(hooks []Hook) [][]Hook {
	if len(hooks) == 0 {
		return nil
	}

	groups := make([][]Hook, 0)
	currentGroup := []Hook{hooks[0]}
	currentPriority := hooks[0].Priority

	for i := 1; i < len(hooks); i++ {
		if hooks[i].Priority == currentPriority {
			currentGroup = append(currentGroup, hooks[i])
		} else {
			groups = append(groups, currentGroup)
			currentGroup = []Hook{hooks[i]}
			currentPriority = hooks[i].Priority
		}
	}
	groups = append(groups, currentGroup)

	return groups
}

// addError adds an error to the error list.
func (m *Manager) addError(err error) {
	m.errorsMu.Lock()
	defer m.errorsMu.Unlock()
	m.errors = append(m.errors, err)
}

// Errors returns all errors that occurred during shutdown.
func (m *Manager) Errors() []error {
	m.errorsMu.Lock()
	defer m.errorsMu.Unlock()
	result := make([]error, len(m.errors))
	copy(result, m.errors)
	return result
}

// State returns the current state of the manager.
func (m *Manager) State() State {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return m.state
}

// IsShuttingDown returns true if shutdown is in progress.
func (m *Manager) IsShuttingDown() bool {
	return m.State() == StateShuttingDown
}

// IsShutdown returns true if shutdown is complete.
func (m *Manager) IsShutdown() bool {
	return m.State() == StateShutdown
}

// Done returns a channel that is closed when shutdown is complete.
func (m *Manager) Done() <-chan struct{} {
	return m.done
}

// Wait blocks until shutdown is complete.
func (m *Manager) Wait() {
	<-m.done
}

// Config returns the manager's configuration.
func (m *Manager) Config() Config {
	return m.config
}

// HookCount returns the number of registered hooks.
func (m *Manager) HookCount() int {
	return m.registry.Count()
}

func (m *Manager) setStateLocked(state State) {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	m.state = state
}

// ForceExit terminates the process immediately with the given exit code.
// This should only be used when graceful shutdown fails completely.
func ForceExit(code int) {
	os.Exit(code)
}

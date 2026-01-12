package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hibiken/asynq"
)

// Manager manages the Asynq client and server.
type Manager struct {
	client    *asynq.Client
	server    *asynq.Server
	scheduler *asynq.Scheduler
	inspector *asynq.Inspector
	config    Config

	mux     *asynq.ServeMux
	mu      sync.RWMutex
	running bool
}

// NewManager creates a new queue manager.
func NewManager(cfg Config) (*Manager, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}

	client := asynq.NewClient(redisOpt)
	inspector := asynq.NewInspector(redisOpt)

	serverCfg := asynq.Config{
		Concurrency: cfg.Concurrency,
		Queues:      cfg.Queues,
		RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
			// Exponential backoff with jitter
			delay := time.Duration(1<<uint(n)) * time.Second
			if delay > 10*time.Minute {
				delay = 10 * time.Minute
			}
			return delay
		},
		ShutdownTimeout: cfg.ShutdownTimeout,
	}

	server := asynq.NewServer(redisOpt, serverCfg)

	scheduler := asynq.NewScheduler(redisOpt, nil)

	return &Manager{
		client:    client,
		server:    server,
		scheduler: scheduler,
		inspector: inspector,
		config:    cfg,
		mux:       asynq.NewServeMux(),
	}, nil
}

// RegisterHandler registers a task handler for the given task type.
func (m *Manager) RegisterHandler(taskType string, handler asynq.HandlerFunc) {
	m.mux.HandleFunc(taskType, handler)
}

// RegisterHandlerFunc registers a handler function for the given task type.
func (m *Manager) RegisterHandlerFunc(taskType string, handler func(context.Context, *asynq.Task) error) {
	m.mux.HandleFunc(taskType, handler)
}

// SetMux sets the handler mux directly.
func (m *Manager) SetMux(mux *asynq.ServeMux) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mux = mux
}

// EnqueueTask enqueues a task for immediate processing.
func (m *Manager) EnqueueTask(ctx context.Context, task *Task) (*asynq.TaskInfo, error) {
	asynqTask := asynq.NewTask(task.Type, task.Payload)

	opts := []asynq.Option{
		asynq.Queue(task.Queue),
		asynq.MaxRetry(task.MaxRetry),
	}

	if task.Timeout > 0 {
		opts = append(opts, asynq.Timeout(task.Timeout))
	}
	if !task.Deadline.IsZero() {
		opts = append(opts, asynq.Deadline(task.Deadline))
	}
	if task.Retention > 0 {
		opts = append(opts, asynq.Retention(task.Retention))
	}
	if task.UniqueKey != "" && task.UniqueTTL > 0 {
		opts = append(opts, asynq.Unique(task.UniqueTTL))
	}
	if task.Group != "" {
		opts = append(opts, asynq.Group(task.Group))
	}

	info, err := m.client.EnqueueContext(ctx, asynqTask, opts...)
	if err != nil {
		return nil, fmt.Errorf("enqueue task: %w", err)
	}
	return info, nil
}

// ScheduleTask schedules a task for future execution.
func (m *Manager) ScheduleTask(ctx context.Context, task *Task, processAt time.Time) (*asynq.TaskInfo, error) {
	asynqTask := asynq.NewTask(task.Type, task.Payload)

	opts := []asynq.Option{
		asynq.Queue(task.Queue),
		asynq.MaxRetry(task.MaxRetry),
		asynq.ProcessAt(processAt),
	}

	if task.Timeout > 0 {
		opts = append(opts, asynq.Timeout(task.Timeout))
	}
	if task.Retention > 0 {
		opts = append(opts, asynq.Retention(task.Retention))
	}

	info, err := m.client.EnqueueContext(ctx, asynqTask, opts...)
	if err != nil {
		return nil, fmt.Errorf("schedule task: %w", err)
	}
	return info, nil
}

// EnqueueIn enqueues a task to be processed after a delay.
func (m *Manager) EnqueueIn(ctx context.Context, task *Task, delay time.Duration) (*asynq.TaskInfo, error) {
	asynqTask := asynq.NewTask(task.Type, task.Payload)

	opts := []asynq.Option{
		asynq.Queue(task.Queue),
		asynq.MaxRetry(task.MaxRetry),
		asynq.ProcessIn(delay),
	}

	if task.Timeout > 0 {
		opts = append(opts, asynq.Timeout(task.Timeout))
	}
	if task.Retention > 0 {
		opts = append(opts, asynq.Retention(task.Retention))
	}

	info, err := m.client.EnqueueContext(ctx, asynqTask, opts...)
	if err != nil {
		return nil, fmt.Errorf("enqueue in: %w", err)
	}
	return info, nil
}

// EnqueueRecurringTask registers a recurring task with a cron expression.
func (m *Manager) EnqueueRecurringTask(task *Task, cronSpec string, entryID string) (string, error) {
	asynqTask := asynq.NewTask(task.Type, task.Payload)

	opts := []asynq.Option{
		asynq.Queue(task.Queue),
		asynq.MaxRetry(task.MaxRetry),
	}

	if task.Timeout > 0 {
		opts = append(opts, asynq.Timeout(task.Timeout))
	}
	if task.Retention > 0 {
		opts = append(opts, asynq.Retention(task.Retention))
	}

	id, err := m.scheduler.Register(cronSpec, asynqTask, opts...)
	if err != nil {
		return "", fmt.Errorf("register recurring task: %w", err)
	}
	return id, nil
}

// UnregisterRecurringTask removes a recurring task.
func (m *Manager) UnregisterRecurringTask(entryID string) error {
	if err := m.scheduler.Unregister(entryID); err != nil {
		return fmt.Errorf("unregister recurring task: %w", err)
	}
	return nil
}

// CancelTask cancels a pending task.
func (m *Manager) CancelTask(taskID string) error {
	if err := m.inspector.CancelProcessing(taskID); err != nil {
		return fmt.Errorf("cancel task: %w", err)
	}
	return nil
}

// DeleteTask deletes a task from a queue.
func (m *Manager) DeleteTask(queue string, taskID string) error {
	if err := m.inspector.DeleteTask(queue, taskID); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}

// ArchiveTask archives a task.
func (m *Manager) ArchiveTask(queue string, taskID string) error {
	if err := m.inspector.ArchiveTask(queue, taskID); err != nil {
		return fmt.Errorf("archive task: %w", err)
	}
	return nil
}

// GetTaskInfo retrieves information about a task.
func (m *Manager) GetTaskInfo(queue string, taskID string) (*asynq.TaskInfo, error) {
	info, err := m.inspector.GetTaskInfo(queue, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task info: %w", err)
	}
	return info, nil
}

// GetQueueInfo retrieves information about a queue.
func (m *Manager) GetQueueInfo(queue string) (*asynq.QueueInfo, error) {
	info, err := m.inspector.GetQueueInfo(queue)
	if err != nil {
		return nil, fmt.Errorf("get queue info: %w", err)
	}
	return info, nil
}

// ListQueues returns all queue names.
func (m *Manager) ListQueues() ([]string, error) {
	queues, err := m.inspector.Queues()
	if err != nil {
		return nil, fmt.Errorf("list queues: %w", err)
	}
	return queues, nil
}

// Start starts the queue server and scheduler.
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	// Start the scheduler in a goroutine
	go func() {
		if err := m.scheduler.Run(); err != nil {
			// Log error
		}
	}()

	// Start the server in a goroutine
	go func() {
		if err := m.server.Run(m.mux); err != nil {
			// Log error
		}
	}()

	m.running = true
	return nil
}

// Stop gracefully stops the queue server and scheduler.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	// Shutdown scheduler
	m.scheduler.Shutdown()

	// Shutdown server
	m.server.Shutdown()

	// Close client
	if err := m.client.Close(); err != nil {
		return fmt.Errorf("close client: %w", err)
	}

	// Close inspector
	if err := m.inspector.Close(); err != nil {
		return fmt.Errorf("close inspector: %w", err)
	}

	m.running = false
	return nil
}

// IsRunning returns whether the manager is running.
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// Client returns the underlying Asynq client.
func (m *Manager) Client() *asynq.Client {
	return m.client
}

// Inspector returns the Asynq inspector for queue introspection.
func (m *Manager) Inspector() *asynq.Inspector {
	return m.inspector
}

// Scheduler returns the Asynq scheduler for cron jobs.
func (m *Manager) Scheduler() *asynq.Scheduler {
	return m.scheduler
}

// Package scheduler provides DSL-to-Asynq job loading and scheduling.
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/robfig/cron/v3"

	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/scheduler/tasks"
)

// DSLJobConfig holds configuration for a DSL-loaded job.
type DSLJobConfig struct {
	Name        string
	Schedule    string // Cron expression
	Task        string
	Queue       string
	RetryPolicy *DSLRetryPolicy
}

// DSLRetryPolicy defines retry behavior for DSL jobs.
type DSLRetryPolicy struct {
	MaxAttempts       int
	InitialInterval   time.Duration
	BackoffMultiplier float64
}

// DSLJobPayload is the payload sent to Asynq for DSL jobs.
type DSLJobPayload struct {
	JobName   string            `json:"jobName"`
	Task      string            `json:"task"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// DSLJobResult holds the result of a DSL job execution.
type DSLJobResult struct {
	JobName     string          `json:"jobName"`
	Task        string          `json:"task"`
	Status      string          `json:"status"`
	Output      json.RawMessage `json:"output,omitempty"`
	Error       string          `json:"error,omitempty"`
	StartedAt   time.Time       `json:"startedAt"`
	CompletedAt time.Time       `json:"completedAt"`
	Duration    time.Duration   `json:"duration"`
}

// LoadJobFromAST loads an Asynq job configuration from an AST JobDecl.
func LoadJobFromAST(decl *ast.JobDecl) (*DSLJobConfig, error) {
	if decl == nil {
		return nil, fmt.Errorf("job declaration is nil")
	}

	config := &DSLJobConfig{
		Name:     decl.Name,
		Schedule: decl.Schedule,
		Task:     decl.Task,
		Queue:    decl.Queue,
	}

	// Default queue if not specified
	if config.Queue == "" {
		config.Queue = "default"
	}

	// Validate cron expression if schedule is provided
	if config.Schedule != "" {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		if _, err := parser.Parse(config.Schedule); err != nil {
			return nil, fmt.Errorf("invalid cron schedule %q: %w", config.Schedule, err)
		}
	}

	// Convert retry policy
	if decl.Retry != nil {
		policy, err := convertJobRetryPolicy(decl.Retry)
		if err != nil {
			return nil, fmt.Errorf("invalid retry policy: %w", err)
		}
		config.RetryPolicy = policy
	} else {
		config.RetryPolicy = defaultJobRetryPolicy()
	}

	return config, nil
}

// convertJobRetryPolicy converts an AST RetryPolicyDecl to a DSLRetryPolicy.
func convertJobRetryPolicy(decl *ast.RetryPolicyDecl) (*DSLRetryPolicy, error) {
	policy := &DSLRetryPolicy{
		MaxAttempts: decl.MaxAttempts,
	}

	if decl.InitialInterval != "" {
		interval, err := time.ParseDuration(decl.InitialInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid initial_interval: %w", err)
		}
		policy.InitialInterval = interval
	} else {
		policy.InitialInterval = 10 * time.Second
	}

	if decl.BackoffMultiplier > 0 {
		policy.BackoffMultiplier = decl.BackoffMultiplier
	} else {
		policy.BackoffMultiplier = 2.0
	}

	return policy, nil
}

// defaultJobRetryPolicy returns a sensible default retry policy.
func defaultJobRetryPolicy() *DSLRetryPolicy {
	return &DSLRetryPolicy{
		MaxAttempts:       3,
		InitialInterval:   10 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// DSLJobRegistry manages loaded DSL jobs.
type DSLJobRegistry struct {
	jobs map[string]*DSLJobConfig
}

// NewDSLJobRegistry creates a new job registry.
func NewDSLJobRegistry() *DSLJobRegistry {
	return &DSLJobRegistry{
		jobs: make(map[string]*DSLJobConfig),
	}
}

// Register adds a job configuration to the registry.
func (r *DSLJobRegistry) Register(config *DSLJobConfig) error {
	if config == nil {
		return fmt.Errorf("job config is nil")
	}
	if config.Name == "" {
		return fmt.Errorf("job name is required")
	}
	r.jobs[config.Name] = config
	return nil
}

// Get retrieves a job configuration by name.
func (r *DSLJobRegistry) Get(name string) (*DSLJobConfig, bool) {
	config, ok := r.jobs[name]
	return config, ok
}

// List returns all registered job names.
func (r *DSLJobRegistry) List() []string {
	names := make([]string, 0, len(r.jobs))
	for name := range r.jobs {
		names = append(names, name)
	}
	return names
}

// GetScheduledJobs returns all jobs that have a schedule.
func (r *DSLJobRegistry) GetScheduledJobs() []*DSLJobConfig {
	var scheduled []*DSLJobConfig
	for _, config := range r.jobs {
		if config.Schedule != "" {
			scheduled = append(scheduled, config)
		}
	}
	return scheduled
}

// LoadJobs loads multiple job declarations into the registry.
func (r *DSLJobRegistry) LoadJobs(decls []*ast.JobDecl) error {
	for _, decl := range decls {
		config, err := LoadJobFromAST(decl)
		if err != nil {
			return fmt.Errorf("failed to load job %q: %w", decl.Name, err)
		}
		if err := r.Register(config); err != nil {
			return err
		}
	}
	return nil
}

// DSLJobScheduler manages scheduling and execution of DSL jobs.
type DSLJobScheduler struct {
	registry  *DSLJobRegistry
	client    *asynq.Client
	scheduler *asynq.Scheduler
}

// NewDSLJobScheduler creates a new job scheduler.
func NewDSLJobScheduler(registry *DSLJobRegistry, redisAddr string) (*DSLJobScheduler, error) {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	scheduler := asynq.NewScheduler(asynq.RedisClientOpt{Addr: redisAddr}, nil)

	return &DSLJobScheduler{
		registry:  registry,
		client:    client,
		scheduler: scheduler,
	}, nil
}

// RegisterScheduledJobs registers all scheduled jobs with the Asynq scheduler.
func (s *DSLJobScheduler) RegisterScheduledJobs() error {
	scheduledJobs := s.registry.GetScheduledJobs()

	for _, config := range scheduledJobs {
		task, err := s.createTask(config)
		if err != nil {
			return fmt.Errorf("failed to create task for job %q: %w", config.Name, err)
		}

		entryID, err := s.scheduler.Register(config.Schedule, task,
			asynq.Queue(config.Queue),
			asynq.MaxRetry(config.RetryPolicy.MaxAttempts),
		)
		if err != nil {
			return fmt.Errorf("failed to register job %q: %w", config.Name, err)
		}

		_ = entryID // Could be used for tracking
	}

	return nil
}

// EnqueueJob enqueues a job for immediate execution.
func (s *DSLJobScheduler) EnqueueJob(ctx context.Context, jobName string, metadata map[string]string) error {
	config, ok := s.registry.Get(jobName)
	if !ok {
		return fmt.Errorf("job %q not found", jobName)
	}

	task, err := s.createTaskWithMetadata(config, metadata)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	_, err = s.client.EnqueueContext(ctx, task,
		asynq.Queue(config.Queue),
		asynq.MaxRetry(config.RetryPolicy.MaxAttempts),
	)
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	return nil
}

// ScheduleJob schedules a job to run at a specific time.
func (s *DSLJobScheduler) ScheduleJob(ctx context.Context, jobName string, runAt time.Time, metadata map[string]string) error {
	config, ok := s.registry.Get(jobName)
	if !ok {
		return fmt.Errorf("job %q not found", jobName)
	}

	task, err := s.createTaskWithMetadata(config, metadata)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	_, err = s.client.EnqueueContext(ctx, task,
		asynq.Queue(config.Queue),
		asynq.MaxRetry(config.RetryPolicy.MaxAttempts),
		asynq.ProcessAt(runAt),
	)
	if err != nil {
		return fmt.Errorf("failed to schedule job: %w", err)
	}

	return nil
}

// createTask creates an Asynq task from a job configuration.
func (s *DSLJobScheduler) createTask(config *DSLJobConfig) (*asynq.Task, error) {
	return s.createTaskWithMetadata(config, nil)
}

// createTaskWithMetadata creates an Asynq task with metadata.
func (s *DSLJobScheduler) createTaskWithMetadata(config *DSLJobConfig, metadata map[string]string) (*asynq.Task, error) {
	payload := DSLJobPayload{
		JobName:   config.Name,
		Task:      config.Task,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Use the task type from the job configuration
	return asynq.NewTask(config.Task, payloadBytes), nil
}

// Start starts the scheduler.
func (s *DSLJobScheduler) Start() error {
	return s.scheduler.Start()
}

// Stop stops the scheduler and closes the client.
func (s *DSLJobScheduler) Stop() error {
	s.scheduler.Shutdown()
	return s.client.Close()
}

// DSLJobHandler handles execution of DSL jobs.
type DSLJobHandler struct {
	registry *DSLJobRegistry
	handlers map[string]TaskHandler
}

// TaskHandler is a function that handles a specific task type.
type TaskHandler func(ctx context.Context, payload DSLJobPayload) (tasks.TaskResult, error)

// NewDSLJobHandler creates a new job handler.
func NewDSLJobHandler(registry *DSLJobRegistry) *DSLJobHandler {
	return &DSLJobHandler{
		registry: registry,
		handlers: make(map[string]TaskHandler),
	}
}

// RegisterHandler registers a handler for a specific task type.
func (h *DSLJobHandler) RegisterHandler(taskType string, handler TaskHandler) {
	h.handlers[taskType] = handler
}

// ProcessTask processes an Asynq task.
func (h *DSLJobHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload DSLJobPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	handler, ok := h.handlers[payload.Task]
	if !ok {
		return fmt.Errorf("no handler registered for task type %q", payload.Task)
	}

	result, err := handler(ctx, payload)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("task failed: %s", result.Error)
	}

	return nil
}

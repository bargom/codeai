package engine

import (
	"context"
	"fmt"
	"sync"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// WorkflowFunc represents a workflow function that can be registered.
type WorkflowFunc interface{}

// ActivityFunc represents an activity function that can be registered.
type ActivityFunc interface{}

// Engine orchestrates workflow execution using Temporal.
type Engine struct {
	client    client.Client
	worker    worker.Worker
	config    Config
	mu        sync.RWMutex
	running   bool
	workflows []WorkflowFunc
	activities []ActivityFunc
}

// NewEngine creates a new workflow engine with the given configuration.
func NewEngine(cfg Config) (*Engine, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &Engine{
		config:     cfg,
		workflows:  make([]WorkflowFunc, 0),
		activities: make([]ActivityFunc, 0),
	}, nil
}

// RegisterWorkflow registers a workflow function with the engine.
func (e *Engine) RegisterWorkflow(wf WorkflowFunc) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.workflows = append(e.workflows, wf)
}

// RegisterActivity registers an activity function with the engine.
func (e *Engine) RegisterActivity(act ActivityFunc) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.activities = append(e.activities, act)
}

// Start initializes the Temporal client and worker, then starts processing.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return ErrEngineAlreadyStarted
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  e.config.TemporalHostPort,
		Namespace: e.config.Namespace,
	})
	if err != nil {
		return fmt.Errorf("creating temporal client: %w", err)
	}
	e.client = c

	// Create worker
	workerOptions := worker.Options{
		MaxConcurrentWorkflowTaskExecutionSize:  e.config.MaxConcurrentWorkflows,
		MaxConcurrentActivityExecutionSize:      e.config.MaxConcurrentActivities,
		Identity:                                e.config.WorkerID,
	}

	e.worker = worker.New(e.client, e.config.TaskQueue, workerOptions)

	// Register workflows and activities
	for _, wf := range e.workflows {
		e.worker.RegisterWorkflow(wf)
	}
	for _, act := range e.activities {
		e.worker.RegisterActivity(act)
	}

	// Start worker in background
	if err := e.worker.Start(); err != nil {
		e.client.Close()
		return fmt.Errorf("starting worker: %w", err)
	}

	e.running = true
	return nil
}

// Stop gracefully shuts down the engine.
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return ErrEngineNotStarted
	}

	e.worker.Stop()
	e.client.Close()
	e.running = false

	return nil
}

// ExecuteWorkflow starts a new workflow execution.
func (e *Engine) ExecuteWorkflow(ctx context.Context, workflowID string, workflow interface{}, input interface{}) (client.WorkflowRun, error) {
	e.mu.RLock()
	if !e.running {
		e.mu.RUnlock()
		return nil, ErrEngineNotStarted
	}
	c := e.client
	taskQueue := e.config.TaskQueue
	timeout := e.config.DefaultTimeout
	e.mu.RUnlock()

	options := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                taskQueue,
		WorkflowExecutionTimeout: timeout,
	}

	run, err := c.ExecuteWorkflow(ctx, options, workflow, input)
	if err != nil {
		return nil, fmt.Errorf("executing workflow: %w", err)
	}

	return run, nil
}

// GetWorkflowResult retrieves the result of a workflow execution.
func (e *Engine) GetWorkflowResult(ctx context.Context, workflowID, runID string, result interface{}) error {
	e.mu.RLock()
	if !e.running {
		e.mu.RUnlock()
		return ErrEngineNotStarted
	}
	c := e.client
	e.mu.RUnlock()

	run := c.GetWorkflow(ctx, workflowID, runID)
	if err := run.Get(ctx, result); err != nil {
		return fmt.Errorf("getting workflow result: %w", err)
	}

	return nil
}

// CancelWorkflow cancels a running workflow execution.
func (e *Engine) CancelWorkflow(ctx context.Context, workflowID, runID string) error {
	e.mu.RLock()
	if !e.running {
		e.mu.RUnlock()
		return ErrEngineNotStarted
	}
	c := e.client
	e.mu.RUnlock()

	if err := c.CancelWorkflow(ctx, workflowID, runID); err != nil {
		return fmt.Errorf("canceling workflow: %w", err)
	}

	return nil
}

// TerminateWorkflow forcefully terminates a workflow execution.
func (e *Engine) TerminateWorkflow(ctx context.Context, workflowID, runID, reason string) error {
	e.mu.RLock()
	if !e.running {
		e.mu.RUnlock()
		return ErrEngineNotStarted
	}
	c := e.client
	e.mu.RUnlock()

	if err := c.TerminateWorkflow(ctx, workflowID, runID, reason); err != nil {
		return fmt.Errorf("terminating workflow: %w", err)
	}

	return nil
}

// GetWorkflowHistory retrieves the history of a workflow execution.
func (e *Engine) GetWorkflowHistory(ctx context.Context, workflowID, runID string) (client.HistoryEventIterator, error) {
	e.mu.RLock()
	if !e.running {
		e.mu.RUnlock()
		return nil, ErrEngineNotStarted
	}
	c := e.client
	e.mu.RUnlock()

	iter := c.GetWorkflowHistory(ctx, workflowID, runID, false, 0)
	return iter, nil
}

// Client returns the underlying Temporal client.
func (e *Engine) Client() client.Client {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.client
}

// IsRunning returns true if the engine is currently running.
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// SendSignal sends a signal to a running workflow.
func (e *Engine) SendSignal(ctx context.Context, workflowID, signalName string, signalData interface{}) error {
	e.mu.RLock()
	if !e.running {
		e.mu.RUnlock()
		return ErrEngineNotStarted
	}
	c := e.client
	e.mu.RUnlock()

	if err := c.SignalWorkflow(ctx, workflowID, "", signalName, signalData); err != nil {
		return fmt.Errorf("signaling workflow: %w", err)
	}

	return nil
}

// QueryWorkflow queries a workflow for its current state.
func (e *Engine) QueryWorkflow(ctx context.Context, workflowID, runID, queryType string, args ...interface{}) (interface{}, error) {
	e.mu.RLock()
	if !e.running {
		e.mu.RUnlock()
		return nil, ErrEngineNotStarted
	}
	c := e.client
	e.mu.RUnlock()

	result, err := c.QueryWorkflow(ctx, workflowID, runID, queryType, args...)
	if err != nil {
		return nil, fmt.Errorf("querying workflow: %w", err)
	}

	var value interface{}
	if err := result.Get(&value); err != nil {
		return nil, fmt.Errorf("decoding query result: %w", err)
	}

	return value, nil
}

// WorkflowExecutionInfo contains information about a workflow execution.
type WorkflowExecutionInfo struct {
	WorkflowID string
	RunID      string
	Type       string
	Status     string
}

// DescribeWorkflow returns information about a workflow execution.
func (e *Engine) DescribeWorkflow(ctx context.Context, workflowID, runID string) (*WorkflowExecutionInfo, error) {
	e.mu.RLock()
	if !e.running {
		e.mu.RUnlock()
		return nil, ErrEngineNotStarted
	}
	c := e.client
	e.mu.RUnlock()

	desc, err := c.DescribeWorkflowExecution(ctx, workflowID, runID)
	if err != nil {
		return nil, fmt.Errorf("describing workflow: %w", err)
	}

	return &WorkflowExecutionInfo{
		WorkflowID: desc.WorkflowExecutionInfo.Execution.WorkflowId,
		RunID:      desc.WorkflowExecutionInfo.Execution.RunId,
		Type:       desc.WorkflowExecutionInfo.Type.Name,
		Status:     desc.WorkflowExecutionInfo.Status.String(),
	}, nil
}

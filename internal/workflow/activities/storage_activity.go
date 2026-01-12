package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/bargom/codeai/internal/workflow/definitions"
)

// StorageClient defines the interface for storing workflow results.
type StorageClient interface {
	SavePipelineResult(ctx context.Context, workflowID string, result json.RawMessage) error
	SaveTestSuiteResult(ctx context.Context, suiteID string, result json.RawMessage) error
}

// StorageActivities holds storage activity implementations.
type StorageActivities struct {
	client StorageClient
}

// NewStorageActivities creates a new StorageActivities instance.
func NewStorageActivities(client StorageClient) *StorageActivities {
	return &StorageActivities{client: client}
}

// StorePipelineResult stores the result of a pipeline workflow.
func (a *StorageActivities) StorePipelineResult(ctx context.Context, req definitions.StorePipelineResultRequest) (definitions.StorageResult, error) {
	info := activity.GetInfo(ctx)
	activity.RecordHeartbeat(ctx, fmt.Sprintf("storing pipeline result for %s (attempt %d)", req.WorkflowID, info.Attempt))

	data, err := json.Marshal(req.Output)
	if err != nil {
		return definitions.StorageResult{
			Success: false,
			Error:   fmt.Sprintf("failed to marshal result: %v", err),
		}, err
	}

	if err := a.client.SavePipelineResult(ctx, req.WorkflowID, data); err != nil {
		return definitions.StorageResult{
			Success: false,
			Error:   fmt.Sprintf("failed to save result: %v", err),
		}, err
	}

	return definitions.StorageResult{Success: true}, nil
}

// StoreTestSuiteResult stores the result of a test suite workflow.
func (a *StorageActivities) StoreTestSuiteResult(ctx context.Context, req definitions.StoreTestSuiteResultRequest) (definitions.StorageResult, error) {
	info := activity.GetInfo(ctx)
	activity.RecordHeartbeat(ctx, fmt.Sprintf("storing test suite result for %s (attempt %d)", req.SuiteID, info.Attempt))

	data, err := json.Marshal(req.Output)
	if err != nil {
		return definitions.StorageResult{
			Success: false,
			Error:   fmt.Sprintf("failed to marshal result: %v", err),
		}, err
	}

	if err := a.client.SaveTestSuiteResult(ctx, req.SuiteID, data); err != nil {
		return definitions.StorageResult{
			Success: false,
			Error:   fmt.Sprintf("failed to save result: %v", err),
		}, err
	}

	return definitions.StorageResult{Success: true}, nil
}

// InMemoryStorageClient is an in-memory implementation of StorageClient for testing.
type InMemoryStorageClient struct {
	pipelineResults  map[string]json.RawMessage
	testSuiteResults map[string]json.RawMessage
}

// NewInMemoryStorageClient creates a new in-memory storage client.
func NewInMemoryStorageClient() *InMemoryStorageClient {
	return &InMemoryStorageClient{
		pipelineResults:  make(map[string]json.RawMessage),
		testSuiteResults: make(map[string]json.RawMessage),
	}
}

// SavePipelineResult saves a pipeline result in memory.
func (c *InMemoryStorageClient) SavePipelineResult(ctx context.Context, workflowID string, result json.RawMessage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	c.pipelineResults[workflowID] = result
	return nil
}

// SaveTestSuiteResult saves a test suite result in memory.
func (c *InMemoryStorageClient) SaveTestSuiteResult(ctx context.Context, suiteID string, result json.RawMessage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	c.testSuiteResults[suiteID] = result
	return nil
}

// GetPipelineResult retrieves a pipeline result from memory.
func (c *InMemoryStorageClient) GetPipelineResult(workflowID string) (json.RawMessage, bool) {
	result, ok := c.pipelineResults[workflowID]
	return result, ok
}

// GetTestSuiteResult retrieves a test suite result from memory.
func (c *InMemoryStorageClient) GetTestSuiteResult(suiteID string) (json.RawMessage, bool) {
	result, ok := c.testSuiteResults[suiteID]
	return result, ok
}

// DatabaseStorageClient wraps a repository for persistent storage.
type DatabaseStorageClient struct {
	repo WorkflowRepository
}

// WorkflowRepository defines the interface for workflow data persistence.
type WorkflowRepository interface {
	SaveExecution(ctx context.Context, workflowID string, status string, output json.RawMessage) error
}

// NewDatabaseStorageClient creates a new database storage client.
func NewDatabaseStorageClient(repo WorkflowRepository) *DatabaseStorageClient {
	return &DatabaseStorageClient{repo: repo}
}

// SavePipelineResult saves a pipeline result to the database.
func (c *DatabaseStorageClient) SavePipelineResult(ctx context.Context, workflowID string, result json.RawMessage) error {
	return c.repo.SaveExecution(ctx, workflowID, string(definitions.StatusCompleted), result)
}

// SaveTestSuiteResult saves a test suite result to the database.
func (c *DatabaseStorageClient) SaveTestSuiteResult(ctx context.Context, suiteID string, result json.RawMessage) error {
	return c.repo.SaveExecution(ctx, suiteID, string(definitions.StatusCompleted), result)
}

// LoggingStorageClient wraps another client and logs operations.
type LoggingStorageClient struct {
	inner StorageClient
}

// NewLoggingStorageClient creates a new logging storage client.
func NewLoggingStorageClient(inner StorageClient) *LoggingStorageClient {
	return &LoggingStorageClient{inner: inner}
}

// SavePipelineResult saves a pipeline result and logs the operation.
func (c *LoggingStorageClient) SavePipelineResult(ctx context.Context, workflowID string, result json.RawMessage) error {
	start := time.Now()
	err := c.inner.SavePipelineResult(ctx, workflowID, result)
	_ = time.Since(start) // duration for logging
	return err
}

// SaveTestSuiteResult saves a test suite result and logs the operation.
func (c *LoggingStorageClient) SaveTestSuiteResult(ctx context.Context, suiteID string, result json.RawMessage) error {
	start := time.Now()
	err := c.inner.SaveTestSuiteResult(ctx, suiteID, result)
	_ = time.Since(start) // duration for logging
	return err
}

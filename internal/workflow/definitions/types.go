// Package definitions contains workflow and activity type definitions.
package definitions

import (
	"encoding/json"
	"time"
)

// Status represents the status of a workflow or task.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

// RetryConfig defines retry behavior for workflows and activities.
type RetryConfig struct {
	MaxAttempts       int           `json:"maxAttempts"`
	InitialInterval   time.Duration `json:"initialInterval"`
	BackoffCoefficient float64       `json:"backoffCoefficient"`
	MaximumInterval   time.Duration `json:"maximumInterval"`
}

// DefaultRetryConfig returns sensible retry defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:        3,
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    60 * time.Second,
	}
}

// AgentConfig defines configuration for an AI agent in a pipeline.
type AgentConfig struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Config     map[string]string `json:"config,omitempty"`
	Timeout    time.Duration     `json:"timeout,omitempty"`
	RetryPolicy RetryConfig       `json:"retryPolicy,omitempty"`
}

// AgentResult holds the result from a single agent execution.
type AgentResult struct {
	AgentName  string          `json:"agentName"`
	Status     Status          `json:"status"`
	Output     json.RawMessage `json:"output,omitempty"`
	Error      string          `json:"error,omitempty"`
	Duration   time.Duration   `json:"duration"`
	StartedAt  time.Time       `json:"startedAt"`
	FinishedAt time.Time       `json:"finishedAt"`
}

// PipelineInput is the input for AIAgentPipelineWorkflow.
type PipelineInput struct {
	WorkflowID  string        `json:"workflowId"`
	Agents      []AgentConfig `json:"agents"`
	Timeout     time.Duration `json:"timeout,omitempty"`
	RetryPolicy RetryConfig   `json:"retryPolicy,omitempty"`
	Parallel    bool          `json:"parallel,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// PipelineOutput is the output from AIAgentPipelineWorkflow.
type PipelineOutput struct {
	WorkflowID   string        `json:"workflowId"`
	Status       Status        `json:"status"`
	Results      []AgentResult `json:"results"`
	StartedAt    time.Time     `json:"startedAt"`
	CompletedAt  time.Time     `json:"completedAt"`
	TotalDuration time.Duration `json:"totalDuration"`
	Error        string        `json:"error,omitempty"`
}

// TestCase defines a single test case in a test suite.
type TestCase struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Input       json.RawMessage   `json:"input"`
	Expected    json.RawMessage   `json:"expected,omitempty"`
	Timeout     time.Duration     `json:"timeout,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
}

// TestCaseResult holds the result of a single test case execution.
type TestCaseResult struct {
	TestCaseID  string          `json:"testCaseId"`
	Name        string          `json:"name"`
	Status      Status          `json:"status"`
	Output      json.RawMessage `json:"output,omitempty"`
	Error       string          `json:"error,omitempty"`
	Duration    time.Duration   `json:"duration"`
	Attempts    int             `json:"attempts"`
	StartedAt   time.Time       `json:"startedAt"`
	FinishedAt  time.Time       `json:"finishedAt"`
}

// TestSuiteInput is the input for TestSuiteWorkflow.
type TestSuiteInput struct {
	SuiteID     string        `json:"suiteId"`
	Name        string        `json:"name"`
	TestCases   []TestCase    `json:"testCases"`
	Timeout     time.Duration `json:"timeout,omitempty"`
	RetryPolicy RetryConfig   `json:"retryPolicy,omitempty"`
	Parallel    bool          `json:"parallel,omitempty"`
	StopOnFailure bool        `json:"stopOnFailure,omitempty"`
}

// TestSuiteOutput is the output from TestSuiteWorkflow.
type TestSuiteOutput struct {
	SuiteID      string            `json:"suiteId"`
	Name         string            `json:"name"`
	Status       Status            `json:"status"`
	Results      []TestCaseResult  `json:"results"`
	TotalTests   int               `json:"totalTests"`
	PassedTests  int               `json:"passedTests"`
	FailedTests  int               `json:"failedTests"`
	SkippedTests int               `json:"skippedTests"`
	StartedAt    time.Time         `json:"startedAt"`
	CompletedAt  time.Time         `json:"completedAt"`
	TotalDuration time.Duration    `json:"totalDuration"`
	Error        string            `json:"error,omitempty"`
}

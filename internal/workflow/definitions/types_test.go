package definitions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	assert.Equal(t, 3, cfg.MaxAttempts)
	assert.Equal(t, time.Second, cfg.InitialInterval)
	assert.Equal(t, 2.0, cfg.BackoffCoefficient)
	assert.Equal(t, 60*time.Second, cfg.MaximumInterval)
}

func TestStatusConstants(t *testing.T) {
	assert.Equal(t, Status("pending"), StatusPending)
	assert.Equal(t, Status("running"), StatusRunning)
	assert.Equal(t, Status("completed"), StatusCompleted)
	assert.Equal(t, Status("failed"), StatusFailed)
	assert.Equal(t, Status("canceled"), StatusCanceled)
}

func TestAgentConfig(t *testing.T) {
	config := AgentConfig{
		Name:    "test-agent",
		Type:    "code-analysis",
		Config:  map[string]string{"key": "value"},
		Timeout: 10 * time.Minute,
		RetryPolicy: RetryConfig{
			MaxAttempts: 5,
		},
	}

	assert.Equal(t, "test-agent", config.Name)
	assert.Equal(t, "code-analysis", config.Type)
	assert.Equal(t, "value", config.Config["key"])
	assert.Equal(t, 10*time.Minute, config.Timeout)
	assert.Equal(t, 5, config.RetryPolicy.MaxAttempts)
}

func TestPipelineInput(t *testing.T) {
	input := PipelineInput{
		WorkflowID: "wf-123",
		Agents: []AgentConfig{
			{Name: "agent1", Type: "code-analysis"},
			{Name: "agent2", Type: "test-generator"},
		},
		Timeout:  30 * time.Minute,
		Parallel: true,
		Metadata: map[string]string{"env": "test"},
	}

	assert.Equal(t, "wf-123", input.WorkflowID)
	assert.Len(t, input.Agents, 2)
	assert.True(t, input.Parallel)
	assert.Equal(t, "test", input.Metadata["env"])
}

func TestTestSuiteInput(t *testing.T) {
	input := TestSuiteInput{
		SuiteID: "suite-123",
		Name:    "Integration Tests",
		TestCases: []TestCase{
			{ID: "tc-1", Name: "Test 1"},
			{ID: "tc-2", Name: "Test 2"},
		},
		Parallel:      true,
		StopOnFailure: false,
	}

	assert.Equal(t, "suite-123", input.SuiteID)
	assert.Equal(t, "Integration Tests", input.Name)
	assert.Len(t, input.TestCases, 2)
	assert.True(t, input.Parallel)
	assert.False(t, input.StopOnFailure)
}

func TestPipelineOutput(t *testing.T) {
	now := time.Now()
	output := PipelineOutput{
		WorkflowID: "wf-123",
		Status:     StatusCompleted,
		Results: []AgentResult{
			{AgentName: "agent1", Status: StatusCompleted},
		},
		StartedAt:     now,
		CompletedAt:   now.Add(5 * time.Minute),
		TotalDuration: 5 * time.Minute,
	}

	assert.Equal(t, "wf-123", output.WorkflowID)
	assert.Equal(t, StatusCompleted, output.Status)
	assert.Len(t, output.Results, 1)
}

func TestTestSuiteOutput(t *testing.T) {
	now := time.Now()
	output := TestSuiteOutput{
		SuiteID:      "suite-123",
		Name:         "Integration Tests",
		Status:       StatusCompleted,
		TotalTests:   10,
		PassedTests:  8,
		FailedTests:  2,
		SkippedTests: 0,
		StartedAt:    now,
		CompletedAt:  now.Add(10 * time.Minute),
	}

	assert.Equal(t, "suite-123", output.SuiteID)
	assert.Equal(t, 10, output.TotalTests)
	assert.Equal(t, 8, output.PassedTests)
	assert.Equal(t, 2, output.FailedTests)
}

package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/bargom/codeai/internal/workflow/definitions"
)

// AgentExecutor defines the interface for executing AI agents.
type AgentExecutor interface {
	Execute(ctx context.Context, agentType string, config map[string]string) (json.RawMessage, error)
}

// AgentActivities holds agent execution activity implementations.
type AgentActivities struct {
	executor AgentExecutor
}

// NewAgentActivities creates a new AgentActivities instance.
func NewAgentActivities(executor AgentExecutor) *AgentActivities {
	return &AgentActivities{executor: executor}
}

// ExecuteAgent executes a single AI agent.
func (a *AgentActivities) ExecuteAgent(ctx context.Context, req definitions.ExecuteAgentRequest) (definitions.ExecuteAgentResponse, error) {
	info := activity.GetInfo(ctx)
	agent := req.Agent

	result := definitions.AgentResult{
		AgentName: agent.Name,
		Status:    definitions.StatusRunning,
		StartedAt: time.Now(),
	}

	// Log activity start
	activity.RecordHeartbeat(ctx, fmt.Sprintf("starting agent %s (attempt %d)", agent.Name, info.Attempt))

	// Execute the agent
	output, err := a.executor.Execute(ctx, agent.Type, agent.Config)
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)

	if err != nil {
		result.Status = definitions.StatusFailed
		result.Error = err.Error()
		return definitions.ExecuteAgentResponse{Result: result}, err
	}

	result.Status = definitions.StatusCompleted
	result.Output = output

	return definitions.ExecuteAgentResponse{Result: result}, nil
}

// DefaultAgentExecutor is a default implementation of AgentExecutor.
type DefaultAgentExecutor struct{}

// Execute implements AgentExecutor for the default executor.
func (e *DefaultAgentExecutor) Execute(ctx context.Context, agentType string, config map[string]string) (json.RawMessage, error) {
	switch agentType {
	case "code-analysis":
		return e.executeCodeAnalysis(ctx, config)
	case "test-generator":
		return e.executeTestGenerator(ctx, config)
	case "documentation":
		return e.executeDocumentation(ctx, config)
	default:
		return nil, fmt.Errorf("unknown agent type: %s", agentType)
	}
}

func (e *DefaultAgentExecutor) executeCodeAnalysis(ctx context.Context, config map[string]string) (json.RawMessage, error) {
	result := map[string]interface{}{
		"analysis": "code analysis completed",
		"type":     "code-analysis",
	}
	return json.Marshal(result)
}

func (e *DefaultAgentExecutor) executeTestGenerator(ctx context.Context, config map[string]string) (json.RawMessage, error) {
	result := map[string]interface{}{
		"testsGenerated": 0,
		"type":           "test-generator",
	}
	return json.Marshal(result)
}

func (e *DefaultAgentExecutor) executeDocumentation(ctx context.Context, config map[string]string) (json.RawMessage, error) {
	result := map[string]interface{}{
		"documentation": "documentation generated",
		"type":          "documentation",
	}
	return json.Marshal(result)
}

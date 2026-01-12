// Package workflow provides HTTP handlers for workflow operations.
package workflow

import (
	"encoding/json"
	"time"

	"github.com/bargom/codeai/internal/workflow/definitions"
	"github.com/bargom/codeai/internal/workflow/repository"
)

// StartWorkflowRequest represents a request to start a new workflow.
type StartWorkflowRequest struct {
	WorkflowID   string            `json:"workflowId" validate:"required"`
	WorkflowType string            `json:"workflowType" validate:"required,oneof=ai-pipeline test-suite"`
	Input        json.RawMessage   `json:"input" validate:"required"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// StartWorkflowResponse represents the response after starting a workflow.
type StartWorkflowResponse struct {
	WorkflowID string `json:"workflowId"`
	RunID      string `json:"runId"`
	Status     string `json:"status"`
}

// WorkflowStatusResponse represents the status of a workflow.
type WorkflowStatusResponse struct {
	WorkflowID  string            `json:"workflowId"`
	WorkflowType string           `json:"workflowType"`
	RunID       string            `json:"runId,omitempty"`
	Status      string            `json:"status"`
	Input       json.RawMessage   `json:"input,omitempty"`
	Output      json.RawMessage   `json:"output,omitempty"`
	Error       string            `json:"error,omitempty"`
	StartedAt   time.Time         `json:"startedAt"`
	CompletedAt *time.Time        `json:"completedAt,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CancelWorkflowRequest represents a request to cancel a workflow.
type CancelWorkflowRequest struct {
	Reason string `json:"reason,omitempty"`
}

// CancelWorkflowResponse represents the response after canceling a workflow.
type CancelWorkflowResponse struct {
	WorkflowID string `json:"workflowId"`
	Status     string `json:"status"`
}

// HistoryEvent represents a single event in workflow history.
type HistoryEvent struct {
	EventID   int64     `json:"eventId"`
	EventType string    `json:"eventType"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details,omitempty"`
}

// WorkflowHistoryResponse represents the history of a workflow.
type WorkflowHistoryResponse struct {
	WorkflowID string         `json:"workflowId"`
	RunID      string         `json:"runId"`
	Events     []HistoryEvent `json:"events"`
}

// ListWorkflowsRequest represents parameters for listing workflows.
type ListWorkflowsRequest struct {
	WorkflowType string  `json:"workflowType,omitempty"`
	Status       string  `json:"status,omitempty"`
	Limit        int     `json:"limit,omitempty"`
	Offset       int     `json:"offset,omitempty"`
}

// ListWorkflowsResponse represents a list of workflows.
type ListWorkflowsResponse struct {
	Workflows []WorkflowStatusResponse `json:"workflows"`
	Total     int                      `json:"total"`
	Limit     int                      `json:"limit"`
	Offset    int                      `json:"offset"`
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

// Converters for internal types

// ToWorkflowStatusResponse converts a repository.WorkflowExecution to WorkflowStatusResponse.
func ToWorkflowStatusResponse(exec *repository.WorkflowExecution) WorkflowStatusResponse {
	return WorkflowStatusResponse{
		WorkflowID:   exec.WorkflowID,
		WorkflowType: exec.WorkflowType,
		RunID:        exec.RunID,
		Status:       string(exec.Status),
		Input:        exec.Input,
		Output:       exec.Output,
		Error:        exec.Error,
		StartedAt:    exec.StartedAt,
		CompletedAt:  exec.CompletedAt,
		Metadata:     exec.Metadata,
	}
}

// ToPipelineInput converts a StartWorkflowRequest to definitions.PipelineInput.
func ToPipelineInput(req StartWorkflowRequest) (definitions.PipelineInput, error) {
	var input definitions.PipelineInput
	if err := json.Unmarshal(req.Input, &input); err != nil {
		return input, err
	}
	input.WorkflowID = req.WorkflowID
	return input, nil
}

// ToTestSuiteInput converts a StartWorkflowRequest to definitions.TestSuiteInput.
func ToTestSuiteInput(req StartWorkflowRequest) (definitions.TestSuiteInput, error) {
	var input definitions.TestSuiteInput
	if err := json.Unmarshal(req.Input, &input); err != nil {
		return input, err
	}
	input.SuiteID = req.WorkflowID
	return input, nil
}

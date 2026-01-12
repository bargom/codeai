// Package activities implements Temporal activities for workflow execution.
package activities

import (
	"context"
	"fmt"

	"github.com/bargom/codeai/internal/workflow/definitions"
)

// ValidationActivities holds validation activity implementations.
type ValidationActivities struct{}

// NewValidationActivities creates a new ValidationActivities instance.
func NewValidationActivities() *ValidationActivities {
	return &ValidationActivities{}
}

// ValidateInput validates pipeline input.
func (a *ValidationActivities) ValidateInput(ctx context.Context, req definitions.ValidateInputRequest) (definitions.ValidationResult, error) {
	input := req.Input

	if input.WorkflowID == "" {
		return definitions.ValidationResult{
			Valid: false,
			Error: "workflow ID is required",
		}, nil
	}

	if len(input.Agents) == 0 {
		return definitions.ValidationResult{
			Valid: false,
			Error: "at least one agent is required",
		}, nil
	}

	for i, agent := range input.Agents {
		if agent.Name == "" {
			return definitions.ValidationResult{
				Valid: false,
				Error: fmt.Sprintf("agent at index %d: name is required", i),
			}, nil
		}
		if agent.Type == "" {
			return definitions.ValidationResult{
				Valid: false,
				Error: fmt.Sprintf("agent %s: type is required", agent.Name),
			}, nil
		}
	}

	return definitions.ValidationResult{Valid: true}, nil
}

// ValidateTestSuite validates test suite input.
func (a *ValidationActivities) ValidateTestSuite(ctx context.Context, req definitions.ValidateTestSuiteRequest) (definitions.ValidationResult, error) {
	input := req.Input

	if input.SuiteID == "" {
		return definitions.ValidationResult{
			Valid: false,
			Error: "suite ID is required",
		}, nil
	}

	if input.Name == "" {
		return definitions.ValidationResult{
			Valid: false,
			Error: "suite name is required",
		}, nil
	}

	if len(input.TestCases) == 0 {
		return definitions.ValidationResult{
			Valid: false,
			Error: "at least one test case is required",
		}, nil
	}

	seenIDs := make(map[string]bool)
	for i, tc := range input.TestCases {
		if tc.ID == "" {
			return definitions.ValidationResult{
				Valid: false,
				Error: fmt.Sprintf("test case at index %d: ID is required", i),
			}, nil
		}
		if seenIDs[tc.ID] {
			return definitions.ValidationResult{
				Valid: false,
				Error: fmt.Sprintf("duplicate test case ID: %s", tc.ID),
			}, nil
		}
		seenIDs[tc.ID] = true

		if tc.Name == "" {
			return definitions.ValidationResult{
				Valid: false,
				Error: fmt.Sprintf("test case %s: name is required", tc.ID),
			}, nil
		}
	}

	return definitions.ValidationResult{Valid: true}, nil
}

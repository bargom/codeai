package activities

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bargom/codeai/internal/workflow/definitions"
)

func TestValidationActivities_ValidateInput(t *testing.T) {
	va := NewValidationActivities()
	ctx := context.Background()

	tests := []struct {
		name      string
		input     definitions.PipelineInput
		wantValid bool
		wantError string
	}{
		{
			name: "valid input",
			input: definitions.PipelineInput{
				WorkflowID: "wf-123",
				Agents: []definitions.AgentConfig{
					{Name: "agent1", Type: "code-analysis"},
				},
			},
			wantValid: true,
		},
		{
			name: "missing workflow ID",
			input: definitions.PipelineInput{
				Agents: []definitions.AgentConfig{
					{Name: "agent1", Type: "code-analysis"},
				},
			},
			wantValid: false,
			wantError: "workflow ID is required",
		},
		{
			name: "no agents",
			input: definitions.PipelineInput{
				WorkflowID: "wf-123",
				Agents:     []definitions.AgentConfig{},
			},
			wantValid: false,
			wantError: "at least one agent is required",
		},
		{
			name: "agent missing name",
			input: definitions.PipelineInput{
				WorkflowID: "wf-123",
				Agents: []definitions.AgentConfig{
					{Name: "", Type: "code-analysis"},
				},
			},
			wantValid: false,
			wantError: "name is required",
		},
		{
			name: "agent missing type",
			input: definitions.PipelineInput{
				WorkflowID: "wf-123",
				Agents: []definitions.AgentConfig{
					{Name: "agent1", Type: ""},
				},
			},
			wantValid: false,
			wantError: "type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := va.ValidateInput(ctx, definitions.ValidateInputRequest{
				Input: tt.input,
			})
			require.NoError(t, err)

			assert.Equal(t, tt.wantValid, result.Valid)
			if !tt.wantValid {
				assert.Contains(t, result.Error, tt.wantError)
			}
		})
	}
}

func TestValidationActivities_ValidateTestSuite(t *testing.T) {
	va := NewValidationActivities()
	ctx := context.Background()

	tests := []struct {
		name      string
		input     definitions.TestSuiteInput
		wantValid bool
		wantError string
	}{
		{
			name: "valid input",
			input: definitions.TestSuiteInput{
				SuiteID: "suite-123",
				Name:    "Test Suite",
				TestCases: []definitions.TestCase{
					{ID: "tc-1", Name: "Test Case 1"},
				},
			},
			wantValid: true,
		},
		{
			name: "missing suite ID",
			input: definitions.TestSuiteInput{
				Name: "Test Suite",
				TestCases: []definitions.TestCase{
					{ID: "tc-1", Name: "Test Case 1"},
				},
			},
			wantValid: false,
			wantError: "suite ID is required",
		},
		{
			name: "missing suite name",
			input: definitions.TestSuiteInput{
				SuiteID: "suite-123",
				TestCases: []definitions.TestCase{
					{ID: "tc-1", Name: "Test Case 1"},
				},
			},
			wantValid: false,
			wantError: "suite name is required",
		},
		{
			name: "no test cases",
			input: definitions.TestSuiteInput{
				SuiteID:   "suite-123",
				Name:      "Test Suite",
				TestCases: []definitions.TestCase{},
			},
			wantValid: false,
			wantError: "at least one test case is required",
		},
		{
			name: "test case missing ID",
			input: definitions.TestSuiteInput{
				SuiteID: "suite-123",
				Name:    "Test Suite",
				TestCases: []definitions.TestCase{
					{ID: "", Name: "Test Case 1"},
				},
			},
			wantValid: false,
			wantError: "ID is required",
		},
		{
			name: "test case missing name",
			input: definitions.TestSuiteInput{
				SuiteID: "suite-123",
				Name:    "Test Suite",
				TestCases: []definitions.TestCase{
					{ID: "tc-1", Name: ""},
				},
			},
			wantValid: false,
			wantError: "name is required",
		},
		{
			name: "duplicate test case ID",
			input: definitions.TestSuiteInput{
				SuiteID: "suite-123",
				Name:    "Test Suite",
				TestCases: []definitions.TestCase{
					{ID: "tc-1", Name: "Test Case 1"},
					{ID: "tc-1", Name: "Test Case 2"},
				},
			},
			wantValid: false,
			wantError: "duplicate test case ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := va.ValidateTestSuite(ctx, definitions.ValidateTestSuiteRequest{
				Input: tt.input,
			})
			require.NoError(t, err)

			assert.Equal(t, tt.wantValid, result.Valid)
			if !tt.wantValid {
				assert.Contains(t, result.Error, tt.wantError)
			}
		})
	}
}

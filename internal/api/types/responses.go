// Package types defines API request and response types.
package types

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/bargom/codeai/internal/database/models"
)

// DeploymentResponse represents a deployment in API responses.
type DeploymentResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ConfigID  *string   `json:"config_id,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DeploymentFromModel converts a database model to an API response.
func DeploymentFromModel(d *models.Deployment) *DeploymentResponse {
	resp := &DeploymentResponse{
		ID:        d.ID,
		Name:      d.Name,
		Status:    d.Status,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
	if d.ConfigID.Valid {
		resp.ConfigID = &d.ConfigID.String
	}
	return resp
}

// DeploymentsFromModels converts a slice of database models to API responses.
func DeploymentsFromModels(deployments []*models.Deployment) []*DeploymentResponse {
	responses := make([]*DeploymentResponse, len(deployments))
	for i, d := range deployments {
		responses[i] = DeploymentFromModel(d)
	}
	return responses
}

// ConfigResponse represents a config in API responses.
type ConfigResponse struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Content          string          `json:"content"`
	ASTJSON          json.RawMessage `json:"ast_json,omitempty"`
	ValidationErrors json.RawMessage `json:"validation_errors,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

// ConfigFromModel converts a database model to an API response.
func ConfigFromModel(c *models.Config) *ConfigResponse {
	return &ConfigResponse{
		ID:               c.ID,
		Name:             c.Name,
		Content:          c.Content,
		ASTJSON:          c.ASTJSON,
		ValidationErrors: c.ValidationErrors,
		CreatedAt:        c.CreatedAt,
	}
}

// ConfigsFromModels converts a slice of database models to API responses.
func ConfigsFromModels(configs []*models.Config) []*ConfigResponse {
	responses := make([]*ConfigResponse, len(configs))
	for i, c := range configs {
		responses[i] = ConfigFromModel(c)
	}
	return responses
}

// ExecutionResponse represents an execution in API responses.
type ExecutionResponse struct {
	ID           string     `json:"id"`
	DeploymentID string     `json:"deployment_id"`
	Command      string     `json:"command"`
	Output       *string    `json:"output,omitempty"`
	ExitCode     *int32     `json:"exit_code,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// ExecutionFromModel converts a database model to an API response.
func ExecutionFromModel(e *models.Execution) *ExecutionResponse {
	resp := &ExecutionResponse{
		ID:           e.ID,
		DeploymentID: e.DeploymentID,
		Command:      e.Command,
		StartedAt:    e.StartedAt,
	}
	if e.Output.Valid {
		resp.Output = &e.Output.String
	}
	if e.ExitCode.Valid {
		resp.ExitCode = &e.ExitCode.Int32
	}
	if e.CompletedAt.Valid {
		resp.CompletedAt = &e.CompletedAt.Time
	}
	return resp
}

// ExecutionsFromModels converts a slice of database models to API responses.
func ExecutionsFromModels(executions []*models.Execution) []*ExecutionResponse {
	responses := make([]*ExecutionResponse, len(executions))
	for i, e := range executions {
		responses[i] = ExecutionFromModel(e)
	}
	return responses
}

// ErrorResponse represents an error in API responses.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

// ValidationErrorResponse represents validation errors in API responses.
type ValidationErrorResponse struct {
	Error  string `json:"error"`
	Errors []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

// ValidationResult represents the result of config validation.
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// ListResponse represents a paginated list response.
type ListResponse[T any] struct {
	Data   []T `json:"data"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total,omitempty"`
}

// NewListResponse creates a new list response.
func NewListResponse[T any](data []T, limit, offset int) *ListResponse[T] {
	return &ListResponse[T]{
		Data:   data,
		Limit:  limit,
		Offset: offset,
	}
}

// NullStringToPtr converts sql.NullString to *string.
func NullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// NullInt32ToPtr converts sql.NullInt32 to *int32.
func NullInt32ToPtr(ni sql.NullInt32) *int32 {
	if ni.Valid {
		return &ni.Int32
	}
	return nil
}

// NullTimeToPtr converts sql.NullTime to *time.Time.
func NullTimeToPtr(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

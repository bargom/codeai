// Package types defines API request and response types.
package types

// CreateDeploymentRequest represents a request to create a deployment.
type CreateDeploymentRequest struct {
	Name     string `json:"name" validate:"required,min=1,max=255"`
	ConfigID string `json:"config_id" validate:"omitempty,uuid"`
}

// UpdateDeploymentRequest represents a request to update a deployment.
type UpdateDeploymentRequest struct {
	Name     string `json:"name" validate:"omitempty,min=1,max=255"`
	ConfigID string `json:"config_id" validate:"omitempty,uuid"`
	Status   string `json:"status" validate:"omitempty,oneof=pending running stopped failed complete"`
}

// ExecuteDeploymentRequest represents a request to execute a deployment.
type ExecuteDeploymentRequest struct {
	Variables map[string]interface{} `json:"variables"`
}

// CreateConfigRequest represents a request to create a config.
type CreateConfigRequest struct {
	Name    string `json:"name" validate:"required,min=1,max=255"`
	Content string `json:"content" validate:"required"`
}

// UpdateConfigRequest represents a request to update a config.
type UpdateConfigRequest struct {
	Name    string `json:"name" validate:"omitempty,min=1,max=255"`
	Content string `json:"content" validate:"omitempty"`
}

// ValidateConfigRequest represents a request to validate a config.
type ValidateConfigRequest struct {
	Content string `json:"content" validate:"required"`
}

// ListParams represents common pagination parameters.
type ListParams struct {
	Limit  int `json:"limit" validate:"omitempty,min=1,max=100"`
	Offset int `json:"offset" validate:"omitempty,min=0"`
}

// DefaultLimit is the default number of items per page.
const DefaultLimit = 20

// DefaultMaxLimit is the maximum allowed limit.
const DefaultMaxLimit = 100

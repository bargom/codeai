// Package repository implements the repository pattern for data access.
package repository

import (
	"context"

	"github.com/bargom/codeai/internal/database/models"
)

// DeploymentRepo defines the interface for deployment persistence operations.
// Both PostgreSQL and MongoDB implementations satisfy this interface.
type DeploymentRepo interface {
	// Create inserts a new deployment.
	Create(ctx context.Context, d *models.Deployment) error
	// GetByID retrieves a deployment by its ID.
	GetByID(ctx context.Context, id string) (*models.Deployment, error)
	// GetByName retrieves a deployment by its name.
	GetByName(ctx context.Context, name string) (*models.Deployment, error)
	// List retrieves all deployments with pagination.
	List(ctx context.Context, limit, offset int) ([]*models.Deployment, error)
	// Update updates an existing deployment.
	Update(ctx context.Context, d *models.Deployment) error
	// Delete removes a deployment.
	Delete(ctx context.Context, id string) error
}

// ConfigRepo defines the interface for config persistence operations.
// Both PostgreSQL and MongoDB implementations satisfy this interface.
type ConfigRepo interface {
	// Create inserts a new config.
	Create(ctx context.Context, c *models.Config) error
	// GetByID retrieves a config by its ID.
	GetByID(ctx context.Context, id string) (*models.Config, error)
	// GetByName retrieves a config by its name.
	GetByName(ctx context.Context, name string) (*models.Config, error)
	// List retrieves all configs with pagination.
	List(ctx context.Context, limit, offset int) ([]*models.Config, error)
	// Update updates an existing config.
	Update(ctx context.Context, c *models.Config) error
	// Delete removes a config.
	Delete(ctx context.Context, id string) error
}

// ExecutionRepo defines the interface for execution persistence operations.
// Both PostgreSQL and MongoDB implementations satisfy this interface.
type ExecutionRepo interface {
	// Create inserts a new execution.
	Create(ctx context.Context, e *models.Execution) error
	// GetByID retrieves an execution by its ID.
	GetByID(ctx context.Context, id string) (*models.Execution, error)
	// List retrieves all executions with pagination.
	List(ctx context.Context, limit, offset int) ([]*models.Execution, error)
	// ListByDeployment retrieves executions for a specific deployment.
	ListByDeployment(ctx context.Context, deploymentID string, limit, offset int) ([]*models.Execution, error)
	// Update updates an existing execution.
	Update(ctx context.Context, e *models.Execution) error
	// Delete removes an execution.
	Delete(ctx context.Context, id string) error
}

// Repositories holds all repository interfaces for dependency injection.
type Repositories struct {
	Deployments DeploymentRepo
	Configs     ConfigRepo
	Executions  ExecutionRepo
}

// NewRepositories creates a Repositories instance from PostgreSQL repositories.
func NewRepositories(deployments *DeploymentRepository, configs *ConfigRepository, executions *ExecutionRepository) *Repositories {
	return &Repositories{
		Deployments: deployments,
		Configs:     configs,
		Executions:  executions,
	}
}

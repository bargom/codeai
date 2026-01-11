// Package models defines domain models for the database layer.
package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Deployment represents a deployment configuration.
type Deployment struct {
	ID        string
	Name      string
	ConfigID  sql.NullString
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DeploymentStatus represents valid deployment statuses.
type DeploymentStatus string

const (
	DeploymentStatusPending  DeploymentStatus = "pending"
	DeploymentStatusRunning  DeploymentStatus = "running"
	DeploymentStatusStopped  DeploymentStatus = "stopped"
	DeploymentStatusFailed   DeploymentStatus = "failed"
	DeploymentStatusComplete DeploymentStatus = "complete"
)

// Config represents a DSL configuration.
type Config struct {
	ID               string
	Name             string
	Content          string
	ASTJSON          json.RawMessage
	ValidationErrors json.RawMessage
	CreatedAt        time.Time
}

// Execution represents a command execution history entry.
type Execution struct {
	ID           string
	DeploymentID string
	Command      string
	Output       sql.NullString
	ExitCode     sql.NullInt32
	StartedAt    time.Time
	CompletedAt  sql.NullTime
}

// NewDeployment creates a new Deployment with sensible defaults.
func NewDeployment(name string) *Deployment {
	now := time.Now().UTC()
	return &Deployment{
		Name:      name,
		Status:    string(DeploymentStatusPending),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewConfig creates a new Config with sensible defaults.
func NewConfig(name, content string) *Config {
	return &Config{
		Name:      name,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}
}

// NewExecution creates a new Execution with sensible defaults.
func NewExecution(deploymentID, command string) *Execution {
	return &Execution{
		DeploymentID: deploymentID,
		Command:      command,
		StartedAt:    time.Now().UTC(),
	}
}

// SetCompleted marks the execution as completed with output and exit code.
func (e *Execution) SetCompleted(output string, exitCode int) {
	e.Output = sql.NullString{String: output, Valid: true}
	e.ExitCode = sql.NullInt32{Int32: int32(exitCode), Valid: true}
	e.CompletedAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
}

// IsCompleted returns true if the execution has completed.
func (e *Execution) IsCompleted() bool {
	return e.CompletedAt.Valid
}

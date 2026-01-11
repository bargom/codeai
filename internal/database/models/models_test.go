package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDeployment(t *testing.T) {
	d := NewDeployment("test-deployment")

	assert.Equal(t, "test-deployment", d.Name)
	assert.Equal(t, string(DeploymentStatusPending), d.Status)
	assert.False(t, d.CreatedAt.IsZero())
	assert.False(t, d.UpdatedAt.IsZero())
	assert.False(t, d.ConfigID.Valid)
}

func TestNewConfig(t *testing.T) {
	c := NewConfig("test-config", "deployment test {}")

	assert.Equal(t, "test-config", c.Name)
	assert.Equal(t, "deployment test {}", c.Content)
	assert.False(t, c.CreatedAt.IsZero())
	assert.Nil(t, c.ASTJSON)
	assert.Nil(t, c.ValidationErrors)
}

func TestNewExecution(t *testing.T) {
	e := NewExecution("deployment-id", "echo hello")

	assert.Equal(t, "deployment-id", e.DeploymentID)
	assert.Equal(t, "echo hello", e.Command)
	assert.False(t, e.StartedAt.IsZero())
	assert.False(t, e.Output.Valid)
	assert.False(t, e.ExitCode.Valid)
	assert.False(t, e.CompletedAt.Valid)
}

func TestExecution_SetCompleted(t *testing.T) {
	e := NewExecution("deployment-id", "echo hello")

	e.SetCompleted("hello\n", 0)

	assert.True(t, e.Output.Valid)
	assert.Equal(t, "hello\n", e.Output.String)
	assert.True(t, e.ExitCode.Valid)
	assert.Equal(t, int32(0), e.ExitCode.Int32)
	assert.True(t, e.CompletedAt.Valid)
}

func TestExecution_IsCompleted(t *testing.T) {
	e := NewExecution("deployment-id", "echo hello")

	assert.False(t, e.IsCompleted())

	e.SetCompleted("output", 0)

	assert.True(t, e.IsCompleted())
}

func TestDeploymentStatus(t *testing.T) {
	assert.Equal(t, DeploymentStatus("pending"), DeploymentStatusPending)
	assert.Equal(t, DeploymentStatus("running"), DeploymentStatusRunning)
	assert.Equal(t, DeploymentStatus("stopped"), DeploymentStatusStopped)
	assert.Equal(t, DeploymentStatus("failed"), DeploymentStatusFailed)
	assert.Equal(t, DeploymentStatus("complete"), DeploymentStatusComplete)
}

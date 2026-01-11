package types_test

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/api/types"
	"github.com/bargom/codeai/internal/database/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploymentFromModel(t *testing.T) {
	now := time.Now().UTC()

	t.Run("converts deployment without config_id", func(t *testing.T) {
		d := &models.Deployment{
			ID:        "123",
			Name:      "test-deployment",
			Status:    "pending",
			CreatedAt: now,
			UpdatedAt: now,
		}

		resp := types.DeploymentFromModel(d)

		assert.Equal(t, "123", resp.ID)
		assert.Equal(t, "test-deployment", resp.Name)
		assert.Equal(t, "pending", resp.Status)
		assert.Nil(t, resp.ConfigID)
		assert.Equal(t, now, resp.CreatedAt)
		assert.Equal(t, now, resp.UpdatedAt)
	})

	t.Run("converts deployment with config_id", func(t *testing.T) {
		d := &models.Deployment{
			ID:        "123",
			Name:      "test-deployment",
			ConfigID:  sql.NullString{String: "config-456", Valid: true},
			Status:    "running",
			CreatedAt: now,
			UpdatedAt: now,
		}

		resp := types.DeploymentFromModel(d)

		assert.NotNil(t, resp.ConfigID)
		assert.Equal(t, "config-456", *resp.ConfigID)
	})
}

func TestDeploymentsFromModels(t *testing.T) {
	deployments := []*models.Deployment{
		{ID: "1", Name: "deploy-1"},
		{ID: "2", Name: "deploy-2"},
	}

	responses := types.DeploymentsFromModels(deployments)

	assert.Len(t, responses, 2)
	assert.Equal(t, "1", responses[0].ID)
	assert.Equal(t, "2", responses[1].ID)
}

func TestConfigFromModel(t *testing.T) {
	now := time.Now().UTC()

	c := &models.Config{
		ID:               "config-123",
		Name:             "test-config",
		Content:          "deployment test {}",
		ASTJSON:          json.RawMessage(`{"nodes":[]}`),
		ValidationErrors: json.RawMessage(`[]`),
		CreatedAt:        now,
	}

	resp := types.ConfigFromModel(c)

	assert.Equal(t, "config-123", resp.ID)
	assert.Equal(t, "test-config", resp.Name)
	assert.Equal(t, "deployment test {}", resp.Content)
	assert.Equal(t, json.RawMessage(`{"nodes":[]}`), resp.ASTJSON)
	assert.Equal(t, json.RawMessage(`[]`), resp.ValidationErrors)
	assert.Equal(t, now, resp.CreatedAt)
}

func TestConfigsFromModels(t *testing.T) {
	configs := []*models.Config{
		{ID: "1", Name: "config-1"},
		{ID: "2", Name: "config-2"},
	}

	responses := types.ConfigsFromModels(configs)

	assert.Len(t, responses, 2)
	assert.Equal(t, "1", responses[0].ID)
	assert.Equal(t, "2", responses[1].ID)
}

func TestExecutionFromModel(t *testing.T) {
	now := time.Now().UTC()
	completedAt := now.Add(time.Minute)

	t.Run("converts incomplete execution", func(t *testing.T) {
		e := &models.Execution{
			ID:           "exec-123",
			DeploymentID: "deploy-456",
			Command:      "echo hello",
			StartedAt:    now,
		}

		resp := types.ExecutionFromModel(e)

		assert.Equal(t, "exec-123", resp.ID)
		assert.Equal(t, "deploy-456", resp.DeploymentID)
		assert.Equal(t, "echo hello", resp.Command)
		assert.Nil(t, resp.Output)
		assert.Nil(t, resp.ExitCode)
		assert.Nil(t, resp.CompletedAt)
	})

	t.Run("converts completed execution", func(t *testing.T) {
		e := &models.Execution{
			ID:           "exec-123",
			DeploymentID: "deploy-456",
			Command:      "echo hello",
			Output:       sql.NullString{String: "hello\n", Valid: true},
			ExitCode:     sql.NullInt32{Int32: 0, Valid: true},
			StartedAt:    now,
			CompletedAt:  sql.NullTime{Time: completedAt, Valid: true},
		}

		resp := types.ExecutionFromModel(e)

		assert.NotNil(t, resp.Output)
		assert.Equal(t, "hello\n", *resp.Output)
		assert.NotNil(t, resp.ExitCode)
		assert.Equal(t, int32(0), *resp.ExitCode)
		assert.NotNil(t, resp.CompletedAt)
		assert.Equal(t, completedAt, *resp.CompletedAt)
	})
}

func TestExecutionsFromModels(t *testing.T) {
	executions := []*models.Execution{
		{ID: "1", DeploymentID: "d1"},
		{ID: "2", DeploymentID: "d2"},
	}

	responses := types.ExecutionsFromModels(executions)

	assert.Len(t, responses, 2)
	assert.Equal(t, "1", responses[0].ID)
	assert.Equal(t, "2", responses[1].ID)
}

func TestNewListResponse(t *testing.T) {
	data := []string{"a", "b", "c"}

	resp := types.NewListResponse(data, 10, 5)

	assert.Equal(t, data, resp.Data)
	assert.Equal(t, 10, resp.Limit)
	assert.Equal(t, 5, resp.Offset)
}

func TestNullHelpers(t *testing.T) {
	t.Run("NullStringToPtr with valid string", func(t *testing.T) {
		ns := sql.NullString{String: "hello", Valid: true}
		ptr := types.NullStringToPtr(ns)
		require.NotNil(t, ptr)
		assert.Equal(t, "hello", *ptr)
	})

	t.Run("NullStringToPtr with invalid string", func(t *testing.T) {
		ns := sql.NullString{Valid: false}
		ptr := types.NullStringToPtr(ns)
		assert.Nil(t, ptr)
	})

	t.Run("NullInt32ToPtr with valid int", func(t *testing.T) {
		ni := sql.NullInt32{Int32: 42, Valid: true}
		ptr := types.NullInt32ToPtr(ni)
		require.NotNil(t, ptr)
		assert.Equal(t, int32(42), *ptr)
	})

	t.Run("NullInt32ToPtr with invalid int", func(t *testing.T) {
		ni := sql.NullInt32{Valid: false}
		ptr := types.NullInt32ToPtr(ni)
		assert.Nil(t, ptr)
	})

	t.Run("NullTimeToPtr with valid time", func(t *testing.T) {
		now := time.Now()
		nt := sql.NullTime{Time: now, Valid: true}
		ptr := types.NullTimeToPtr(nt)
		require.NotNil(t, ptr)
		assert.Equal(t, now, *ptr)
	})

	t.Run("NullTimeToPtr with invalid time", func(t *testing.T) {
		nt := sql.NullTime{Valid: false}
		ptr := types.NullTimeToPtr(nt)
		assert.Nil(t, ptr)
	})
}

func TestErrorResponse(t *testing.T) {
	resp := types.ErrorResponse{
		Error: "something went wrong",
		Details: map[string]string{
			"field": "is required",
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded types.ErrorResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "something went wrong", decoded.Error)
	assert.Equal(t, "is required", decoded.Details["field"])
}

func TestHealthResponse(t *testing.T) {
	resp := types.HealthResponse{
		Status:    "healthy",
		Timestamp: "2024-01-11T12:00:00Z",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded types.HealthResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "healthy", decoded.Status)
	assert.Equal(t, "2024-01-11T12:00:00Z", decoded.Timestamp)
}

func TestValidationResult(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		resp := types.ValidationResult{
			Valid:  true,
			Errors: []string{},
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"valid":true`)
	})

	t.Run("invalid result with errors", func(t *testing.T) {
		resp := types.ValidationResult{
			Valid:  false,
			Errors: []string{"error 1", "error 2"},
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"valid":false`)
		assert.Contains(t, string(data), `"error 1"`)
	})
}

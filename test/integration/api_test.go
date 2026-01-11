//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/bargom/codeai/internal/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIHealthEndpoint tests the health check endpoint.
func TestAPIHealthEndpoint(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	t.Run("health check returns ok", func(t *testing.T) {
		resp, err := http.Get(suite.Server.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health types.HealthResponse
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)
		assert.Equal(t, "healthy", health.Status)
	})
}

// TestAPIConfigEndpoints tests config CRUD endpoints.
func TestAPIConfigEndpoints(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	var createdConfigID string

	t.Run("create config", func(t *testing.T) {
		reqBody := types.CreateConfigRequest{
			Name:    "api-test-config",
			Content: `var x = 1`,
		}
		body, _ := json.Marshal(reqBody)

		resp, err := http.Post(suite.Server.URL+"/configs", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var config types.ConfigResponse
		err = json.NewDecoder(resp.Body).Decode(&config)
		require.NoError(t, err)
		assert.NotEmpty(t, config.ID)
		assert.Equal(t, "api-test-config", config.Name)
		createdConfigID = config.ID
	})

	t.Run("get config", func(t *testing.T) {
		require.NotEmpty(t, createdConfigID)

		resp, err := http.Get(suite.Server.URL + "/configs/" + createdConfigID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var config types.ConfigResponse
		err = json.NewDecoder(resp.Body).Decode(&config)
		require.NoError(t, err)
		assert.Equal(t, createdConfigID, config.ID)
	})

	t.Run("list configs", func(t *testing.T) {
		resp, err := http.Get(suite.Server.URL + "/configs")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp types.ListResponse[*types.ConfigResponse]
		err = json.NewDecoder(resp.Body).Decode(&listResp)
		require.NoError(t, err)
		assert.NotEmpty(t, listResp.Data)
	})

	t.Run("update config", func(t *testing.T) {
		require.NotEmpty(t, createdConfigID)

		reqBody := types.UpdateConfigRequest{
			Name:    "updated-config",
			Content: `var y = 2`,
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, suite.Server.URL+"/configs/"+createdConfigID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("delete config", func(t *testing.T) {
		require.NotEmpty(t, createdConfigID)

		req, _ := http.NewRequest(http.MethodDelete, suite.Server.URL+"/configs/"+createdConfigID, nil)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify deleted
		resp2, err := http.Get(suite.Server.URL + "/configs/" + createdConfigID)
		require.NoError(t, err)
		defer resp2.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
	})
}

// TestAPIDeploymentEndpoints tests deployment CRUD endpoints.
func TestAPIDeploymentEndpoints(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	// First create a config
	cfg := suite.CreateTestConfig(t, "deploy-test-config", `var x = 1`)
	var createdDeploymentID string

	t.Run("create deployment", func(t *testing.T) {
		reqBody := types.CreateDeploymentRequest{
			Name:     "api-test-deployment",
			ConfigID: cfg.ID,
		}
		body, _ := json.Marshal(reqBody)

		resp, err := http.Post(suite.Server.URL+"/deployments", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var deployment types.DeploymentResponse
		err = json.NewDecoder(resp.Body).Decode(&deployment)
		require.NoError(t, err)
		assert.NotEmpty(t, deployment.ID)
		assert.Equal(t, "api-test-deployment", deployment.Name)
		createdDeploymentID = deployment.ID
	})

	t.Run("get deployment", func(t *testing.T) {
		require.NotEmpty(t, createdDeploymentID)

		resp, err := http.Get(suite.Server.URL + "/deployments/" + createdDeploymentID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var deployment types.DeploymentResponse
		err = json.NewDecoder(resp.Body).Decode(&deployment)
		require.NoError(t, err)
		assert.Equal(t, createdDeploymentID, deployment.ID)
	})

	t.Run("list deployments", func(t *testing.T) {
		resp, err := http.Get(suite.Server.URL + "/deployments")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp types.ListResponse[*types.DeploymentResponse]
		err = json.NewDecoder(resp.Body).Decode(&listResp)
		require.NoError(t, err)
		assert.NotEmpty(t, listResp.Data)
	})

	t.Run("update deployment", func(t *testing.T) {
		require.NotEmpty(t, createdDeploymentID)

		reqBody := types.UpdateDeploymentRequest{
			Name:   "updated-deployment",
			Status: "running",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, suite.Server.URL+"/deployments/"+createdDeploymentID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("delete deployment", func(t *testing.T) {
		require.NotEmpty(t, createdDeploymentID)

		req, _ := http.NewRequest(http.MethodDelete, suite.Server.URL+"/deployments/"+createdDeploymentID, nil)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

// TestAPIExecutionEndpoints tests execution endpoints.
func TestAPIExecutionEndpoints(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	// Create deployment first
	deploy := suite.CreateTestDeployment(t, "exec-test-deploy", nil)
	// Create an execution
	exec := suite.CreateTestExecution(t, deploy.ID, "echo hello")

	t.Run("get execution", func(t *testing.T) {
		resp, err := http.Get(suite.Server.URL + "/executions/" + exec.ID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var execution types.ExecutionResponse
		err = json.NewDecoder(resp.Body).Decode(&execution)
		require.NoError(t, err)
		assert.Equal(t, exec.ID, execution.ID)
	})

	t.Run("list executions", func(t *testing.T) {
		resp, err := http.Get(suite.Server.URL + "/executions")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("list deployment executions", func(t *testing.T) {
		resp, err := http.Get(suite.Server.URL + "/deployments/" + deploy.ID + "/executions")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestAPIValidation tests request validation.
func TestAPIValidation(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	t.Run("create config with empty name fails", func(t *testing.T) {
		reqBody := types.CreateConfigRequest{
			Name:    "",
			Content: `var x = 1`,
		}
		body, _ := json.Marshal(reqBody)

		resp, err := http.Post(suite.Server.URL+"/configs", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("create deployment with empty name fails", func(t *testing.T) {
		reqBody := types.CreateDeploymentRequest{
			Name: "",
		}
		body, _ := json.Marshal(reqBody)

		resp, err := http.Post(suite.Server.URL+"/deployments", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestAPIPagination tests pagination parameters.
func TestAPIPagination(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	// Create multiple configs
	for i := 0; i < 5; i++ {
		suite.CreateTestConfig(t, "pagination-config-"+string(rune('a'+i)), `var x = 1`)
	}

	t.Run("list with limit", func(t *testing.T) {
		resp, err := http.Get(suite.Server.URL + "/configs?limit=2")
		require.NoError(t, err)
		defer resp.Body.Close()

		var listResp types.ListResponse[*types.ConfigResponse]
		err = json.NewDecoder(resp.Body).Decode(&listResp)
		require.NoError(t, err)
		assert.Len(t, listResp.Data, 2)
		assert.Equal(t, 2, listResp.Limit)
	})

	t.Run("list with offset", func(t *testing.T) {
		resp, err := http.Get(suite.Server.URL + "/configs?limit=2&offset=2")
		require.NoError(t, err)
		defer resp.Body.Close()

		var listResp types.ListResponse[*types.ConfigResponse]
		err = json.NewDecoder(resp.Body).Decode(&listResp)
		require.NoError(t, err)
		assert.Equal(t, 2, listResp.Offset)
	})
}

// TestAPINotFound tests 404 responses.
func TestAPINotFound(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	t.Run("get nonexistent config", func(t *testing.T) {
		resp, err := http.Get(suite.Server.URL + "/configs/nonexistent-id")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("get nonexistent deployment", func(t *testing.T) {
		resp, err := http.Get(suite.Server.URL + "/deployments/nonexistent-id")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("get nonexistent execution", func(t *testing.T) {
		resp, err := http.Get(suite.Server.URL + "/executions/nonexistent-id")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestAPIContentType tests content type header.
func TestAPIContentType(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	t.Run("responses have json content type", func(t *testing.T) {
		resp, err := http.Get(suite.Server.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	})
}

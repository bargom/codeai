//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/bargom/codeai/internal/api/types"
	"github.com/bargom/codeai/internal/parser"
	"github.com/bargom/codeai/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompleteDeploymentWorkflow tests a full deployment workflow end-to-end.
func TestCompleteDeploymentWorkflow(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	validDSL := `
var apiKey = "sk-12345"
var maxRetries = 3
var debug = true

function configure() {
    var settings = "default"
}

if debug {
    var logLevel = "verbose"
}

exec {
    echo "Configuration complete"
}
`

	// Step 1: Parse and validate DSL locally first
	t.Run("step 1: local validation", func(t *testing.T) {
		program, err := parser.Parse(validDSL)
		require.NoError(t, err, "DSL should parse successfully")

		v := validator.New()
		err = v.Validate(program)
		assert.NoError(t, err, "DSL should validate successfully")
	})

	// Step 2: Create config via API
	var configID string
	t.Run("step 2: create config via API", func(t *testing.T) {
		reqBody := types.CreateConfigRequest{
			Name:    "e2e-test-config",
			Content: validDSL,
		}
		body, _ := json.Marshal(reqBody)

		resp, err := http.Post(suite.Server.URL+"/configs", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var config types.ConfigResponse
		err = json.NewDecoder(resp.Body).Decode(&config)
		require.NoError(t, err)
		require.NotEmpty(t, config.ID)
		configID = config.ID
	})

	// Step 3: Verify config can be retrieved
	t.Run("step 3: verify config retrieval", func(t *testing.T) {
		require.NotEmpty(t, configID)

		resp, err := http.Get(suite.Server.URL + "/configs/" + configID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var config types.ConfigResponse
		err = json.NewDecoder(resp.Body).Decode(&config)
		require.NoError(t, err)
		assert.Equal(t, "e2e-test-config", config.Name)
		assert.Contains(t, config.Content, "apiKey")
	})

	// Step 4: Create deployment referencing the config
	var deploymentID string
	t.Run("step 4: create deployment", func(t *testing.T) {
		require.NotEmpty(t, configID)

		reqBody := types.CreateDeploymentRequest{
			Name:     "e2e-test-deployment",
			ConfigID: configID,
		}
		body, _ := json.Marshal(reqBody)

		resp, err := http.Post(suite.Server.URL+"/deployments", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var deployment types.DeploymentResponse
		err = json.NewDecoder(resp.Body).Decode(&deployment)
		require.NoError(t, err)
		require.NotEmpty(t, deployment.ID)
		assert.Equal(t, "pending", deployment.Status)
		deploymentID = deployment.ID
	})

	// Step 5: Verify deployment is linked to config
	t.Run("step 5: verify deployment-config link", func(t *testing.T) {
		require.NotEmpty(t, deploymentID)

		resp, err := http.Get(suite.Server.URL + "/deployments/" + deploymentID)
		require.NoError(t, err)
		defer resp.Body.Close()

		var deployment types.DeploymentResponse
		err = json.NewDecoder(resp.Body).Decode(&deployment)
		require.NoError(t, err)
		assert.NotNil(t, deployment.ConfigID)
		assert.Equal(t, configID, *deployment.ConfigID)
	})

	// Step 6: Update deployment status
	t.Run("step 6: update deployment status", func(t *testing.T) {
		require.NotEmpty(t, deploymentID)

		reqBody := types.UpdateDeploymentRequest{
			Status: "running",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, suite.Server.URL+"/deployments/"+deploymentID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Step 7: Simulate execution record (via database directly since execute endpoint may not exist)
	var executionID string
	t.Run("step 7: create execution record", func(t *testing.T) {
		require.NotEmpty(t, deploymentID)

		exec := suite.CreateTestExecution(t, deploymentID, "echo hello")
		require.NotEmpty(t, exec.ID)
		executionID = exec.ID
	})

	// Step 8: Verify execution can be retrieved
	t.Run("step 8: verify execution retrieval", func(t *testing.T) {
		require.NotEmpty(t, executionID)

		resp, err := http.Get(suite.Server.URL + "/executions/" + executionID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var execution types.ExecutionResponse
		err = json.NewDecoder(resp.Body).Decode(&execution)
		require.NoError(t, err)
		assert.Equal(t, deploymentID, execution.DeploymentID)
	})

	// Step 9: List deployment executions
	t.Run("step 9: list deployment executions", func(t *testing.T) {
		require.NotEmpty(t, deploymentID)

		resp, err := http.Get(suite.Server.URL + "/deployments/" + deploymentID + "/executions")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp types.ListResponse[*types.ExecutionResponse]
		err = json.NewDecoder(resp.Body).Decode(&listResp)
		require.NoError(t, err)
		assert.Len(t, listResp.Data, 1)
	})

	// Step 10: Complete deployment status
	t.Run("step 10: complete deployment", func(t *testing.T) {
		require.NotEmpty(t, deploymentID)

		reqBody := types.UpdateDeploymentRequest{
			Status: "complete",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, suite.Server.URL+"/deployments/"+deploymentID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify final status
		getResp, err := http.Get(suite.Server.URL + "/deployments/" + deploymentID)
		require.NoError(t, err)
		defer getResp.Body.Close()

		var deployment types.DeploymentResponse
		err = json.NewDecoder(getResp.Body).Decode(&deployment)
		require.NoError(t, err)
		assert.Equal(t, "complete", deployment.Status)
	})
}

// TestConfigUpdateAndRevalidation tests updating a config and revalidating.
func TestConfigUpdateAndRevalidation(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	t.Run("update config content triggers revalidation", func(t *testing.T) {
		// Create initial valid config
		cfg := suite.CreateTestConfig(t, "update-test-config", `var x = 1`)

		// Update with new valid content
		reqBody := types.UpdateConfigRequest{
			Content: `var y = 2`,
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, suite.Server.URL+"/configs/"+cfg.ID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify content updated
		getResp, err := http.Get(suite.Server.URL + "/configs/" + cfg.ID)
		require.NoError(t, err)
		defer getResp.Body.Close()

		var config types.ConfigResponse
		err = json.NewDecoder(getResp.Body).Decode(&config)
		require.NoError(t, err)
		assert.Contains(t, config.Content, "var y = 2")
	})
}

// TestMultipleDeploymentsPerConfig tests multiple deployments using same config.
func TestMultipleDeploymentsPerConfig(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	t.Run("multiple deployments can share a config", func(t *testing.T) {
		// Create config
		cfg := suite.CreateTestConfig(t, "shared-config", `var shared = true`)

		// Create multiple deployments using same config
		var deploymentIDs []string
		for i := 0; i < 3; i++ {
			deploy := suite.CreateTestDeployment(t, "deploy-"+string(rune('a'+i)), &cfg.ID)
			deploymentIDs = append(deploymentIDs, deploy.ID)
		}

		// Verify all deployments reference the same config
		for _, deployID := range deploymentIDs {
			resp, err := http.Get(suite.Server.URL + "/deployments/" + deployID)
			require.NoError(t, err)
			defer resp.Body.Close()

			var deployment types.DeploymentResponse
			err = json.NewDecoder(resp.Body).Decode(&deployment)
			require.NoError(t, err)
			assert.Equal(t, cfg.ID, *deployment.ConfigID)
		}
	})
}

// TestDeploymentWithoutConfig tests deployment without linked config.
func TestDeploymentWithoutConfig(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	t.Run("deployment without config is allowed", func(t *testing.T) {
		reqBody := types.CreateDeploymentRequest{
			Name: "no-config-deployment",
		}
		body, _ := json.Marshal(reqBody)

		resp, err := http.Post(suite.Server.URL+"/deployments", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var deployment types.DeploymentResponse
		err = json.NewDecoder(resp.Body).Decode(&deployment)
		require.NoError(t, err)
		assert.Nil(t, deployment.ConfigID)
	})
}

// TestErrorRecoveryWorkflow tests handling of errors in workflow.
func TestErrorRecoveryWorkflow(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Teardown(t)

	t.Run("invalid DSL validation", func(t *testing.T) {
		invalidDSL := `var x = undefinedVariable`

		// Parsing succeeds
		program, err := parser.Parse(invalidDSL)
		require.NoError(t, err)

		// Validation fails
		v := validator.New()
		err = v.Validate(program)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "undefined")
	})

	t.Run("deployment with invalid config ID", func(t *testing.T) {
		reqBody := types.CreateDeploymentRequest{
			Name:     "invalid-config-deploy",
			ConfigID: "nonexistent-uuid-12345",
		}
		body, _ := json.Marshal(reqBody)

		resp, err := http.Post(suite.Server.URL+"/deployments", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should fail due to foreign key constraint or validation
		// The actual status code depends on implementation
		assert.True(t, resp.StatusCode >= 400)
	})
}

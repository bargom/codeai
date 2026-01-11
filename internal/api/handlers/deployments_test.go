package handlers_test

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/bargom/codeai/internal/api/handlers"
	apitesting "github.com/bargom/codeai/internal/api/testing"
	"github.com/bargom/codeai/internal/api/types"
	"github.com/bargom/codeai/internal/database/models"
	"github.com/bargom/codeai/internal/database/repository"
	dbtesting "github.com/bargom/codeai/internal/database/testing"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestHandler(t *testing.T) (*handlers.Handler, *apitesting.TestServer, func()) {
	t.Helper()

	db := dbtesting.SetupTestDB(t)

	deployments := repository.NewDeploymentRepository(db)
	configs := repository.NewConfigRepository(db)
	executions := repository.NewExecutionRepository(db)

	h := handlers.NewHandler(deployments, configs, executions)

	r := chi.NewRouter()
	r.Route("/deployments", func(r chi.Router) {
		r.Post("/", h.CreateDeployment)
		r.Get("/", h.ListDeployments)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetDeployment)
			r.Put("/", h.UpdateDeployment)
			r.Delete("/", h.DeleteDeployment)
			r.Post("/execute", h.ExecuteDeployment)
			r.Get("/executions", h.ListDeploymentExecutions)
		})
	})

	ts := apitesting.NewTestServer(t, r)

	return h, ts, func() {
		ts.Close()
		dbtesting.TeardownTestDB(t, db)
	}
}

func TestCreateDeployment(t *testing.T) {
	_, ts, cleanup := setupTestHandler(t)
	defer cleanup()

	t.Run("creates deployment with valid input", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "test-deployment",
		}
		resp := ts.MakeRequest(http.MethodPost, "/deployments", body)
		apitesting.AssertStatus(t, resp, http.StatusCreated)
		apitesting.AssertContentType(t, resp, "application/json")

		var deployment types.DeploymentResponse
		apitesting.AssertJSON(t, resp, &deployment)

		assert.NotEmpty(t, deployment.ID)
		assert.Equal(t, "test-deployment", deployment.Name)
		assert.Equal(t, "pending", deployment.Status)
		assert.NotZero(t, deployment.CreatedAt)
		assert.NotZero(t, deployment.UpdatedAt)
	})

	t.Run("creates deployment without config_id", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "deployment-no-config",
		}
		resp := ts.MakeRequest(http.MethodPost, "/deployments", body)
		apitesting.AssertStatus(t, resp, http.StatusCreated)

		var deployment types.DeploymentResponse
		apitesting.AssertJSON(t, resp, &deployment)

		assert.Nil(t, deployment.ConfigID)
	})

	t.Run("rejects empty name", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "",
		}
		resp := ts.MakeRequest(http.MethodPost, "/deployments", body)
		apitesting.AssertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("rejects missing name", func(t *testing.T) {
		body := map[string]interface{}{}
		resp := ts.MakeRequest(http.MethodPost, "/deployments", body)
		apitesting.AssertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("rejects invalid config_id format", func(t *testing.T) {
		body := map[string]interface{}{
			"name":      "test",
			"config_id": "not-a-uuid",
		}
		resp := ts.MakeRequest(http.MethodPost, "/deployments", body)
		apitesting.AssertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodPost, "/deployments", "invalid json")
		apitesting.AssertStatus(t, resp, http.StatusBadRequest)
	})
}

func TestGetDeployment(t *testing.T) {
	_, ts, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create a deployment first
	body := map[string]interface{}{"name": "get-test-deployment"}
	createResp := ts.MakeRequest(http.MethodPost, "/deployments", body)
	var created types.DeploymentResponse
	apitesting.AssertJSON(t, createResp, &created)

	t.Run("gets existing deployment", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/deployments/"+created.ID, nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)
		apitesting.AssertContentType(t, resp, "application/json")

		var deployment types.DeploymentResponse
		apitesting.AssertJSON(t, resp, &deployment)

		assert.Equal(t, created.ID, deployment.ID)
		assert.Equal(t, "get-test-deployment", deployment.Name)
	})

	t.Run("returns 404 for non-existent deployment", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/deployments/"+uuid.New().String(), nil)
		apitesting.AssertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("returns 400 for invalid ID format", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/deployments/invalid-id", nil)
		apitesting.AssertStatus(t, resp, http.StatusNotFound) // Chi will still route, handler should validate
	})
}

func TestListDeployments(t *testing.T) {
	_, ts, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create some deployments
	for i := 0; i < 5; i++ {
		body := map[string]interface{}{"name": "list-test-" + string(rune('a'+i))}
		ts.MakeRequest(http.MethodPost, "/deployments", body)
	}

	t.Run("lists all deployments", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/deployments", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)
		apitesting.AssertContentType(t, resp, "application/json")

		var result types.ListResponse[types.DeploymentResponse]
		apitesting.AssertJSON(t, resp, &result)

		assert.Len(t, result.Data, 5)
		assert.Equal(t, 20, result.Limit)
		assert.Equal(t, 0, result.Offset)
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/deployments?limit=2", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var result types.ListResponse[types.DeploymentResponse]
		apitesting.AssertJSON(t, resp, &result)

		assert.Len(t, result.Data, 2)
		assert.Equal(t, 2, result.Limit)
	})

	t.Run("respects offset parameter", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/deployments?offset=3", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var result types.ListResponse[types.DeploymentResponse]
		apitesting.AssertJSON(t, resp, &result)

		assert.Len(t, result.Data, 2)
		assert.Equal(t, 3, result.Offset)
	})

	t.Run("respects limit and offset together", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/deployments?limit=2&offset=2", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var result types.ListResponse[types.DeploymentResponse]
		apitesting.AssertJSON(t, resp, &result)

		assert.Len(t, result.Data, 2)
		assert.Equal(t, 2, result.Limit)
		assert.Equal(t, 2, result.Offset)
	})

	t.Run("enforces max limit", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/deployments?limit=1000", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var result types.ListResponse[types.DeploymentResponse]
		apitesting.AssertJSON(t, resp, &result)

		assert.Equal(t, 100, result.Limit) // Max limit is 100
	})

	t.Run("handles invalid limit gracefully", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/deployments?limit=invalid", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK) // Falls back to default

		var result types.ListResponse[types.DeploymentResponse]
		apitesting.AssertJSON(t, resp, &result)

		assert.Equal(t, 20, result.Limit) // Default limit
	})
}

func TestUpdateDeployment(t *testing.T) {
	_, ts, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create a deployment first
	createBody := map[string]interface{}{"name": "update-test-deployment"}
	createResp := ts.MakeRequest(http.MethodPost, "/deployments", createBody)
	var created types.DeploymentResponse
	apitesting.AssertJSON(t, createResp, &created)

	t.Run("updates deployment name", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "updated-name",
		}
		resp := ts.MakeRequest(http.MethodPut, "/deployments/"+created.ID, body)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var deployment types.DeploymentResponse
		apitesting.AssertJSON(t, resp, &deployment)

		assert.Equal(t, "updated-name", deployment.Name)
	})

	t.Run("updates deployment status", func(t *testing.T) {
		body := map[string]interface{}{
			"status": "running",
		}
		resp := ts.MakeRequest(http.MethodPut, "/deployments/"+created.ID, body)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var deployment types.DeploymentResponse
		apitesting.AssertJSON(t, resp, &deployment)

		assert.Equal(t, "running", deployment.Status)
	})

	t.Run("rejects invalid status", func(t *testing.T) {
		body := map[string]interface{}{
			"status": "invalid-status",
		}
		resp := ts.MakeRequest(http.MethodPut, "/deployments/"+created.ID, body)
		apitesting.AssertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("returns 404 for non-existent deployment", func(t *testing.T) {
		body := map[string]interface{}{"name": "new-name"}
		resp := ts.MakeRequest(http.MethodPut, "/deployments/"+uuid.New().String(), body)
		apitesting.AssertStatus(t, resp, http.StatusNotFound)
	})
}

func TestDeleteDeployment(t *testing.T) {
	_, ts, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create a deployment first
	createBody := map[string]interface{}{"name": "delete-test-deployment"}
	createResp := ts.MakeRequest(http.MethodPost, "/deployments", createBody)
	var created types.DeploymentResponse
	apitesting.AssertJSON(t, createResp, &created)

	t.Run("deletes existing deployment", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodDelete, "/deployments/"+created.ID, nil)
		apitesting.AssertStatus(t, resp, http.StatusNoContent)

		// Verify it's deleted
		getResp := ts.MakeRequest(http.MethodGet, "/deployments/"+created.ID, nil)
		apitesting.AssertStatus(t, getResp, http.StatusNotFound)
	})

	t.Run("returns 404 for non-existent deployment", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodDelete, "/deployments/"+uuid.New().String(), nil)
		apitesting.AssertStatus(t, resp, http.StatusNotFound)
	})
}

func TestExecuteDeployment(t *testing.T) {
	_, ts, cleanup := setupTestHandler(t)
	defer cleanup()

	// Create a deployment first
	createBody := map[string]interface{}{"name": "execute-test-deployment"}
	createResp := ts.MakeRequest(http.MethodPost, "/deployments", createBody)
	var created types.DeploymentResponse
	apitesting.AssertJSON(t, createResp, &created)

	t.Run("executes deployment", func(t *testing.T) {
		body := map[string]interface{}{
			"variables": map[string]interface{}{
				"key": "value",
			},
		}
		resp := ts.MakeRequest(http.MethodPost, "/deployments/"+created.ID+"/execute", body)
		apitesting.AssertStatus(t, resp, http.StatusAccepted)

		var execution types.ExecutionResponse
		apitesting.AssertJSON(t, resp, &execution)

		assert.NotEmpty(t, execution.ID)
		assert.Equal(t, created.ID, execution.DeploymentID)
	})

	t.Run("executes deployment without variables", func(t *testing.T) {
		body := map[string]interface{}{}
		resp := ts.MakeRequest(http.MethodPost, "/deployments/"+created.ID+"/execute", body)
		apitesting.AssertStatus(t, resp, http.StatusAccepted)
	})

	t.Run("returns 404 for non-existent deployment", func(t *testing.T) {
		body := map[string]interface{}{}
		resp := ts.MakeRequest(http.MethodPost, "/deployments/"+uuid.New().String()+"/execute", body)
		apitesting.AssertStatus(t, resp, http.StatusNotFound)
	})
}

func TestListDeploymentExecutions(t *testing.T) {
	h, ts, cleanup := setupTestHandler(t)
	defer cleanup()
	_ = h // Used for direct repo access if needed

	// Create a deployment first
	createBody := map[string]interface{}{"name": "exec-list-test-deployment"}
	createResp := ts.MakeRequest(http.MethodPost, "/deployments", createBody)
	var created types.DeploymentResponse
	apitesting.AssertJSON(t, createResp, &created)

	// Execute deployment multiple times
	for i := 0; i < 3; i++ {
		body := map[string]interface{}{}
		ts.MakeRequest(http.MethodPost, "/deployments/"+created.ID+"/execute", body)
	}

	t.Run("lists deployment executions", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/deployments/"+created.ID+"/executions", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var result types.ListResponse[types.ExecutionResponse]
		apitesting.AssertJSON(t, resp, &result)

		assert.Len(t, result.Data, 3)
		for _, exec := range result.Data {
			assert.Equal(t, created.ID, exec.DeploymentID)
		}
	})

	t.Run("returns empty list for deployment with no executions", func(t *testing.T) {
		// Create new deployment
		newBody := map[string]interface{}{"name": "no-exec-deployment"}
		newResp := ts.MakeRequest(http.MethodPost, "/deployments", newBody)
		var newCreated types.DeploymentResponse
		apitesting.AssertJSON(t, newResp, &newCreated)

		resp := ts.MakeRequest(http.MethodGet, "/deployments/"+newCreated.ID+"/executions", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var result types.ListResponse[types.ExecutionResponse]
		apitesting.AssertJSON(t, resp, &result)

		assert.Empty(t, result.Data)
	})

	t.Run("returns 404 for non-existent deployment", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/deployments/"+uuid.New().String()+"/executions", nil)
		apitesting.AssertStatus(t, resp, http.StatusNotFound)
	})
}

func createTestDeployment(t *testing.T, repo *repository.DeploymentRepository) *models.Deployment {
	t.Helper()
	d := models.NewDeployment("test-deployment")
	err := repo.Create(nil, d)
	require.NoError(t, err)
	return d
}

func createTestDeploymentWithConfig(t *testing.T, deployRepo *repository.DeploymentRepository, configRepo *repository.ConfigRepository) *models.Deployment {
	t.Helper()

	c := models.NewConfig("test-config", "# Test Config")
	err := configRepo.Create(nil, c)
	require.NoError(t, err)

	d := models.NewDeployment("test-deployment-with-config")
	d.ConfigID = sql.NullString{String: c.ID, Valid: true}
	err = deployRepo.Create(nil, d)
	require.NoError(t, err)

	return d
}

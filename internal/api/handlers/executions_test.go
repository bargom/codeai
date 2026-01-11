package handlers_test

import (
	"context"
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

func setupExecutionTestHandler(t *testing.T) (*handlers.Handler, *apitesting.TestServer, *repository.DeploymentRepository, *repository.ExecutionRepository, func()) {
	t.Helper()

	db := dbtesting.SetupTestDB(t)

	deployments := repository.NewDeploymentRepository(db)
	configs := repository.NewConfigRepository(db)
	executions := repository.NewExecutionRepository(db)

	h := handlers.NewHandler(deployments, configs, executions)

	r := chi.NewRouter()
	r.Route("/executions", func(r chi.Router) {
		r.Get("/", h.ListExecutions)
		r.Get("/{id}", h.GetExecution)
	})

	ts := apitesting.NewTestServer(t, r)

	return h, ts, deployments, executions, func() {
		ts.Close()
		dbtesting.TeardownTestDB(t, db)
	}
}

func TestListExecutions(t *testing.T) {
	_, ts, deploymentRepo, executionRepo, cleanup := setupExecutionTestHandler(t)
	defer cleanup()

	// Create a deployment first
	deployment := models.NewDeployment("test-deployment")
	err := deploymentRepo.Create(context.Background(), deployment)
	require.NoError(t, err)

	// Create some executions
	for i := 0; i < 5; i++ {
		exec := models.NewExecution(deployment.ID, "test command")
		err := executionRepo.Create(context.Background(), exec)
		require.NoError(t, err)
	}

	t.Run("lists all executions", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/executions", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)
		apitesting.AssertContentType(t, resp, "application/json")

		var result types.ListResponse[types.ExecutionResponse]
		apitesting.AssertJSON(t, resp, &result)

		assert.Len(t, result.Data, 5)
		assert.Equal(t, 20, result.Limit)
		assert.Equal(t, 0, result.Offset)
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/executions?limit=2", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var result types.ListResponse[types.ExecutionResponse]
		apitesting.AssertJSON(t, resp, &result)

		assert.Len(t, result.Data, 2)
		assert.Equal(t, 2, result.Limit)
	})

	t.Run("respects offset parameter", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/executions?offset=3", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var result types.ListResponse[types.ExecutionResponse]
		apitesting.AssertJSON(t, resp, &result)

		assert.Len(t, result.Data, 2)
		assert.Equal(t, 3, result.Offset)
	})

	t.Run("respects limit and offset together", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/executions?limit=2&offset=2", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var result types.ListResponse[types.ExecutionResponse]
		apitesting.AssertJSON(t, resp, &result)

		assert.Len(t, result.Data, 2)
		assert.Equal(t, 2, result.Limit)
		assert.Equal(t, 2, result.Offset)
	})
}

func TestGetExecution(t *testing.T) {
	_, ts, deploymentRepo, executionRepo, cleanup := setupExecutionTestHandler(t)
	defer cleanup()

	// Create a deployment first
	deployment := models.NewDeployment("test-deployment")
	err := deploymentRepo.Create(context.Background(), deployment)
	require.NoError(t, err)

	// Create an execution
	execution := models.NewExecution(deployment.ID, "echo hello")
	err = executionRepo.Create(context.Background(), execution)
	require.NoError(t, err)

	t.Run("gets existing execution", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/executions/"+execution.ID, nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)
		apitesting.AssertContentType(t, resp, "application/json")

		var result types.ExecutionResponse
		apitesting.AssertJSON(t, resp, &result)

		assert.Equal(t, execution.ID, result.ID)
		assert.Equal(t, deployment.ID, result.DeploymentID)
		assert.Equal(t, "echo hello", result.Command)
	})

	t.Run("returns 404 for non-existent execution", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/executions/"+uuid.New().String(), nil)
		apitesting.AssertStatus(t, resp, http.StatusNotFound)
	})
}

func TestGetExecutionWithCompletedStatus(t *testing.T) {
	_, ts, deploymentRepo, executionRepo, cleanup := setupExecutionTestHandler(t)
	defer cleanup()

	// Create a deployment first
	deployment := models.NewDeployment("test-deployment")
	err := deploymentRepo.Create(context.Background(), deployment)
	require.NoError(t, err)

	// Create a completed execution
	execution := models.NewExecution(deployment.ID, "echo hello")
	execution.SetCompleted("hello\n", 0)
	err = executionRepo.Create(context.Background(), execution)
	require.NoError(t, err)

	t.Run("gets completed execution with output", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/executions/"+execution.ID, nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var result types.ExecutionResponse
		apitesting.AssertJSON(t, resp, &result)

		assert.Equal(t, execution.ID, result.ID)
		assert.NotNil(t, result.Output)
		assert.Equal(t, "hello\n", *result.Output)
		assert.NotNil(t, result.ExitCode)
		assert.Equal(t, int32(0), *result.ExitCode)
		assert.NotNil(t, result.CompletedAt)
	})
}

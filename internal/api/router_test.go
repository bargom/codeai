package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bargom/codeai/internal/api"
	"github.com/bargom/codeai/internal/api/handlers"
	"github.com/bargom/codeai/internal/database/repository"
	dbtesting "github.com/bargom/codeai/internal/database/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRouter(t *testing.T) (*httptest.Server, func()) {
	t.Helper()

	db := dbtesting.SetupTestDB(t)

	deployments := repository.NewDeploymentRepository(db)
	configs := repository.NewConfigRepository(db)
	executions := repository.NewExecutionRepository(db)

	h := handlers.NewHandler(deployments, configs, executions)
	router := api.NewRouter(h)

	ts := httptest.NewServer(router)

	return ts, func() {
		ts.Close()
		dbtesting.TeardownTestDB(t, db)
	}
}

func TestRouterHealthEndpoint(t *testing.T) {
	ts, cleanup := setupRouter(t)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
}

func TestRouterDeploymentEndpoints(t *testing.T) {
	ts, cleanup := setupRouter(t)
	defer cleanup()

	t.Run("GET /deployments returns 200", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/deployments")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestRouterConfigEndpoints(t *testing.T) {
	ts, cleanup := setupRouter(t)
	defer cleanup()

	t.Run("GET /configs returns 200", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/configs")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestRouterExecutionEndpoints(t *testing.T) {
	ts, cleanup := setupRouter(t)
	defer cleanup()

	t.Run("GET /executions returns 200", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/executions")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestRouterNotFoundHandler(t *testing.T) {
	ts, cleanup := setupRouter(t)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/nonexistent")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

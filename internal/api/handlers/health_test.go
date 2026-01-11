package handlers_test

import (
	"net/http"
	"testing"

	"github.com/bargom/codeai/internal/api/handlers"
	apitesting "github.com/bargom/codeai/internal/api/testing"
	"github.com/bargom/codeai/internal/api/types"
	"github.com/bargom/codeai/internal/database/repository"
	dbtesting "github.com/bargom/codeai/internal/database/testing"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func setupHealthTestHandler(t *testing.T) (*handlers.Handler, *apitesting.TestServer, func()) {
	t.Helper()

	db := dbtesting.SetupTestDB(t)

	deployments := repository.NewDeploymentRepository(db)
	configs := repository.NewConfigRepository(db)
	executions := repository.NewExecutionRepository(db)

	h := handlers.NewHandler(deployments, configs, executions)

	r := chi.NewRouter()
	r.Get("/health", h.Health)

	ts := apitesting.NewTestServer(t, r)

	return h, ts, func() {
		ts.Close()
		dbtesting.TeardownTestDB(t, db)
	}
}

func TestHealth(t *testing.T) {
	_, ts, cleanup := setupHealthTestHandler(t)
	defer cleanup()

	t.Run("returns healthy status", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/health", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)
		apitesting.AssertContentType(t, resp, "application/json")

		var health types.HealthResponse
		apitesting.AssertJSON(t, resp, &health)

		assert.Equal(t, "healthy", health.Status)
		assert.NotEmpty(t, health.Timestamp)
	})
}

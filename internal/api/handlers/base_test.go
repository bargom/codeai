package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bargom/codeai/internal/api/handlers"
	"github.com/bargom/codeai/internal/database/repository"
	dbtesting "github.com/bargom/codeai/internal/database/testing"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestHandlerDecodeJSONErrors(t *testing.T) {
	db := dbtesting.SetupTestDB(t)
	defer dbtesting.TeardownTestDB(t, db)

	deployments := repository.NewDeploymentRepository(db)
	configs := repository.NewConfigRepository(db)
	executions := repository.NewExecutionRepository(db)

	h := handlers.NewHandler(deployments, configs, executions)

	r := chi.NewRouter()
	r.Post("/deployments", h.CreateDeployment)

	t.Run("rejects nil body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/deployments", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects malformed JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/deployments", bytes.NewReader([]byte(`{invalid}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestValidationErrors(t *testing.T) {
	db := dbtesting.SetupTestDB(t)
	defer dbtesting.TeardownTestDB(t, db)

	deployments := repository.NewDeploymentRepository(db)
	configs := repository.NewConfigRepository(db)
	executions := repository.NewExecutionRepository(db)

	h := handlers.NewHandler(deployments, configs, executions)

	r := chi.NewRouter()
	r.Post("/deployments", h.CreateDeployment)
	r.Put("/deployments/{id}", h.UpdateDeployment)
	r.Post("/configs", h.CreateConfig)

	t.Run("validates required field", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/deployments", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "validation failed")
	})

	t.Run("validates min length", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/deployments", bytes.NewReader([]byte(`{"name": ""}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validates uuid format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/deployments", bytes.NewReader([]byte(`{"name": "test", "config_id": "not-a-uuid"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validates oneof constraint", func(t *testing.T) {
		// First create a deployment to update
		createReq := httptest.NewRequest(http.MethodPost, "/deployments", bytes.NewReader([]byte(`{"name": "test-deploy"}`)))
		createReq.Header.Set("Content-Type", "application/json")
		createW := httptest.NewRecorder()
		r.ServeHTTP(createW, createReq)

		// Now update with invalid status
		req := httptest.NewRequest(http.MethodPut, "/deployments/some-id", bytes.NewReader([]byte(`{"status": "invalid-status"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validates max length", func(t *testing.T) {
		longName := make([]byte, 300)
		for i := range longName {
			longName[i] = 'a'
		}
		body := `{"name": "` + string(longName) + `"}`
		req := httptest.NewRequest(http.MethodPost, "/deployments", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

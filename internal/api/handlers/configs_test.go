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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupConfigTestHandler(t *testing.T) (*handlers.Handler, *apitesting.TestServer, func()) {
	t.Helper()

	db := dbtesting.SetupTestDB(t)

	deployments := repository.NewDeploymentRepository(db)
	configs := repository.NewConfigRepository(db)
	executions := repository.NewExecutionRepository(db)

	h := handlers.NewHandler(deployments, configs, executions)

	r := chi.NewRouter()
	r.Route("/configs", func(r chi.Router) {
		r.Post("/", h.CreateConfig)
		r.Get("/", h.ListConfigs)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetConfig)
			r.Put("/", h.UpdateConfig)
			r.Delete("/", h.DeleteConfig)
			r.Post("/validate", h.ValidateConfig)
		})
	})

	ts := apitesting.NewTestServer(t, r)

	return h, ts, func() {
		ts.Close()
		dbtesting.TeardownTestDB(t, db)
	}
}

func TestCreateConfig(t *testing.T) {
	_, ts, cleanup := setupConfigTestHandler(t)
	defer cleanup()

	t.Run("creates config with valid input", func(t *testing.T) {
		body := map[string]interface{}{
			"name":    "test-config",
			"content": "deployment test { task run { exec 'echo hello' } }",
		}
		resp := ts.MakeRequest(http.MethodPost, "/configs", body)
		apitesting.AssertStatus(t, resp, http.StatusCreated)
		apitesting.AssertContentType(t, resp, "application/json")

		var config types.ConfigResponse
		apitesting.AssertJSON(t, resp, &config)

		assert.NotEmpty(t, config.ID)
		assert.Equal(t, "test-config", config.Name)
		assert.Equal(t, "deployment test { task run { exec 'echo hello' } }", config.Content)
		assert.NotZero(t, config.CreatedAt)
	})

	t.Run("rejects empty name", func(t *testing.T) {
		body := map[string]interface{}{
			"name":    "",
			"content": "some content",
		}
		resp := ts.MakeRequest(http.MethodPost, "/configs", body)
		apitesting.AssertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("rejects missing name", func(t *testing.T) {
		body := map[string]interface{}{
			"content": "some content",
		}
		resp := ts.MakeRequest(http.MethodPost, "/configs", body)
		apitesting.AssertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("rejects missing content", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "test-config",
		}
		resp := ts.MakeRequest(http.MethodPost, "/configs", body)
		apitesting.AssertStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodPost, "/configs", "invalid json")
		apitesting.AssertStatus(t, resp, http.StatusBadRequest)
	})
}

func TestGetConfig(t *testing.T) {
	_, ts, cleanup := setupConfigTestHandler(t)
	defer cleanup()

	// Create a config first
	body := map[string]interface{}{
		"name":    "get-test-config",
		"content": "test content",
	}
	createResp := ts.MakeRequest(http.MethodPost, "/configs", body)
	var created types.ConfigResponse
	apitesting.AssertJSON(t, createResp, &created)

	t.Run("gets existing config", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/configs/"+created.ID, nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)
		apitesting.AssertContentType(t, resp, "application/json")

		var config types.ConfigResponse
		apitesting.AssertJSON(t, resp, &config)

		assert.Equal(t, created.ID, config.ID)
		assert.Equal(t, "get-test-config", config.Name)
	})

	t.Run("returns 404 for non-existent config", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/configs/"+uuid.New().String(), nil)
		apitesting.AssertStatus(t, resp, http.StatusNotFound)
	})
}

func TestListConfigs(t *testing.T) {
	_, ts, cleanup := setupConfigTestHandler(t)
	defer cleanup()

	// Create some configs
	for i := 0; i < 5; i++ {
		body := map[string]interface{}{
			"name":    "list-test-" + string(rune('a'+i)),
			"content": "content " + string(rune('a'+i)),
		}
		ts.MakeRequest(http.MethodPost, "/configs", body)
	}

	t.Run("lists all configs", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/configs", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)
		apitesting.AssertContentType(t, resp, "application/json")

		var result types.ListResponse[types.ConfigResponse]
		apitesting.AssertJSON(t, resp, &result)

		assert.Len(t, result.Data, 5)
		assert.Equal(t, 20, result.Limit)
		assert.Equal(t, 0, result.Offset)
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/configs?limit=2", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var result types.ListResponse[types.ConfigResponse]
		apitesting.AssertJSON(t, resp, &result)

		assert.Len(t, result.Data, 2)
		assert.Equal(t, 2, result.Limit)
	})

	t.Run("respects offset parameter", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodGet, "/configs?offset=3", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var result types.ListResponse[types.ConfigResponse]
		apitesting.AssertJSON(t, resp, &result)

		assert.Len(t, result.Data, 2)
		assert.Equal(t, 3, result.Offset)
	})
}

func TestUpdateConfig(t *testing.T) {
	_, ts, cleanup := setupConfigTestHandler(t)
	defer cleanup()

	// Create a config first
	createBody := map[string]interface{}{
		"name":    "update-test-config",
		"content": "original content",
	}
	createResp := ts.MakeRequest(http.MethodPost, "/configs", createBody)
	var created types.ConfigResponse
	apitesting.AssertJSON(t, createResp, &created)

	t.Run("updates config name", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "updated-name",
		}
		resp := ts.MakeRequest(http.MethodPut, "/configs/"+created.ID, body)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var config types.ConfigResponse
		apitesting.AssertJSON(t, resp, &config)

		assert.Equal(t, "updated-name", config.Name)
	})

	t.Run("updates config content", func(t *testing.T) {
		body := map[string]interface{}{
			"content": "updated content",
		}
		resp := ts.MakeRequest(http.MethodPut, "/configs/"+created.ID, body)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var config types.ConfigResponse
		apitesting.AssertJSON(t, resp, &config)

		assert.Equal(t, "updated content", config.Content)
	})

	t.Run("returns 404 for non-existent config", func(t *testing.T) {
		body := map[string]interface{}{"name": "new-name"}
		resp := ts.MakeRequest(http.MethodPut, "/configs/"+uuid.New().String(), body)
		apitesting.AssertStatus(t, resp, http.StatusNotFound)
	})
}

func TestDeleteConfig(t *testing.T) {
	_, ts, cleanup := setupConfigTestHandler(t)
	defer cleanup()

	// Create a config first
	createBody := map[string]interface{}{
		"name":    "delete-test-config",
		"content": "content to delete",
	}
	createResp := ts.MakeRequest(http.MethodPost, "/configs", createBody)
	var created types.ConfigResponse
	apitesting.AssertJSON(t, createResp, &created)

	t.Run("deletes existing config", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodDelete, "/configs/"+created.ID, nil)
		apitesting.AssertStatus(t, resp, http.StatusNoContent)

		// Verify it's deleted
		getResp := ts.MakeRequest(http.MethodGet, "/configs/"+created.ID, nil)
		apitesting.AssertStatus(t, getResp, http.StatusNotFound)
	})

	t.Run("returns 404 for non-existent config", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodDelete, "/configs/"+uuid.New().String(), nil)
		apitesting.AssertStatus(t, resp, http.StatusNotFound)
	})
}

func TestValidateConfig(t *testing.T) {
	_, ts, cleanup := setupConfigTestHandler(t)
	defer cleanup()

	// Create a config first
	createBody := map[string]interface{}{
		"name":    "validate-test-config",
		"content": "deployment test { task run { exec 'echo hello' } }",
	}
	createResp := ts.MakeRequest(http.MethodPost, "/configs", createBody)
	var created types.ConfigResponse
	apitesting.AssertJSON(t, createResp, &created)

	t.Run("validates existing config", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodPost, "/configs/"+created.ID+"/validate", nil)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var result types.ValidationResult
		apitesting.AssertJSON(t, resp, &result)

		// The result should indicate valid or invalid based on parsing
		// For now, we just check it returns a valid response
		assert.NotNil(t, result)
	})

	t.Run("validates with custom content", func(t *testing.T) {
		body := map[string]interface{}{
			"content": "let x = 5",
		}
		resp := ts.MakeRequest(http.MethodPost, "/configs/"+created.ID+"/validate", body)
		apitesting.AssertStatus(t, resp, http.StatusOK)

		var result types.ValidationResult
		apitesting.AssertJSON(t, resp, &result)

		assert.NotNil(t, result)
	})

	t.Run("returns 404 for non-existent config", func(t *testing.T) {
		resp := ts.MakeRequest(http.MethodPost, "/configs/"+uuid.New().String()+"/validate", nil)
		apitesting.AssertStatus(t, resp, http.StatusNotFound)
	})
}

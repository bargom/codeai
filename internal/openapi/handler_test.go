package openapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func createTestSpec() *OpenAPI {
	return &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       "Test API",
			Description: "A test API",
			Version:     "1.0.0",
		},
		Servers: []Server{
			{URL: "https://api.example.com", Description: "Production"},
		},
		Paths: map[string]PathItem{
			"/users": {
				Get: &Operation{
					Summary:     "List users",
					OperationID: "listUsers",
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
		},
		Components: Components{
			Schemas: map[string]*Schema{
				"User": {
					Type: "object",
					Properties: map[string]*Schema{
						"id":   {Type: "string"},
						"name": {Type: "string"},
					},
				},
			},
			SecuritySchemes: map[string]*SecurityScheme{
				"bearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
				},
			},
		},
	}
}

func TestHandlerServeJSON(t *testing.T) {
	spec := createTestSpec()
	handler := NewHandler(spec)

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()

	handler.ServeJSON(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	// Verify it's valid JSON
	var parsed OpenAPI
	err := json.NewDecoder(resp.Body).Decode(&parsed)
	require.NoError(t, err)

	assert.Equal(t, "3.0.0", parsed.OpenAPI)
	assert.Equal(t, "Test API", parsed.Info.Title)
	assert.Contains(t, parsed.Paths, "/users")
}

func TestHandlerServeYAML(t *testing.T) {
	spec := createTestSpec()
	handler := NewHandler(spec)

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	w := httptest.NewRecorder()

	handler.ServeYAML(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/x-yaml", resp.Header.Get("Content-Type"))

	// Verify it's valid YAML
	var parsed OpenAPI
	err := yaml.NewDecoder(resp.Body).Decode(&parsed)
	require.NoError(t, err)

	assert.Equal(t, "3.0.0", parsed.OpenAPI)
	assert.Equal(t, "Test API", parsed.Info.Title)
}

func TestHandlerServeSwaggerUI(t *testing.T) {
	spec := createTestSpec()
	handler := NewHandler(spec)

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	w := httptest.NewRecorder()

	handler.ServeSwaggerUI(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")

	body := w.Body.String()
	assert.Contains(t, body, "swagger-ui")
	assert.Contains(t, body, "/openapi.json")
	assert.Contains(t, body, "Test API")
}

func TestHandlerServeReDoc(t *testing.T) {
	spec := createTestSpec()
	handler := NewHandler(spec)

	req := httptest.NewRequest(http.MethodGet, "/redoc", nil)
	w := httptest.NewRecorder()

	handler.ServeReDoc(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")

	body := w.Body.String()
	assert.Contains(t, body, "redoc")
	assert.Contains(t, body, "/openapi.json")
}

func TestHandlerRegisterRoutes(t *testing.T) {
	spec := createTestSpec()
	handler := NewHandler(spec)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	// Test JSON endpoint
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Test YAML endpoint
	req = httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/x-yaml", w.Header().Get("Content-Type"))

	// Test Swagger UI endpoint
	req = httptest.NewRequest(http.MethodGet, "/docs", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")

	// Test ReDoc endpoint
	req = httptest.NewRequest(http.MethodGet, "/redoc", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
}

func TestHandlerUpdateSpec(t *testing.T) {
	spec := createTestSpec()
	handler := NewHandler(spec)

	// First request to build cache
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()
	handler.ServeJSON(w, req)

	var parsed1 OpenAPI
	json.NewDecoder(w.Body).Decode(&parsed1)
	assert.Equal(t, "Test API", parsed1.Info.Title)

	// Update spec
	newSpec := createTestSpec()
	newSpec.Info.Title = "Updated API"
	handler.UpdateSpec(newSpec)

	// Second request should return updated spec
	req = httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w = httptest.NewRecorder()
	handler.ServeJSON(w, req)

	var parsed2 OpenAPI
	json.NewDecoder(w.Body).Decode(&parsed2)
	assert.Equal(t, "Updated API", parsed2.Info.Title)
}

func TestHandlerGetSpec(t *testing.T) {
	spec := createTestSpec()
	handler := NewHandler(spec)

	retrieved := handler.GetSpec()
	assert.Equal(t, spec, retrieved)
}

func TestHandlerCaching(t *testing.T) {
	spec := createTestSpec()
	handler := NewHandler(spec)

	// Make multiple requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
		w := httptest.NewRecorder()
		handler.ServeJSON(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// The cache should be built once and reused
	assert.NotNil(t, handler.specJSON)
}

func TestGenerateSwaggerUIHTML(t *testing.T) {
	html := generateSwaggerUIHTML("/openapi.json", "My API")

	assert.Contains(t, html, "<title>My API</title>")
	assert.Contains(t, html, "swagger-ui")
	assert.Contains(t, html, `url: "/openapi.json"`)
	assert.Contains(t, html, "SwaggerUIBundle")
}

func TestGenerateSwaggerUIHTMLDefaultTitle(t *testing.T) {
	html := generateSwaggerUIHTML("/openapi.json", "")

	assert.Contains(t, html, "<title>API Documentation</title>")
}

func TestGenerateReDocHTML(t *testing.T) {
	html := generateReDocHTML("/openapi.json", "My API")

	assert.Contains(t, html, "<title>My API</title>")
	assert.Contains(t, html, "redoc")
	assert.Contains(t, html, `spec-url="/openapi.json"`)
}

func TestGenerateReDocHTMLDefaultTitle(t *testing.T) {
	html := generateReDocHTML("/openapi.json", "")

	assert.Contains(t, html, "<title>API Documentation</title>")
}

func TestHandlerCacheControl(t *testing.T) {
	spec := createTestSpec()
	handler := NewHandler(spec)

	// Test JSON endpoint
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()
	handler.ServeJSON(w, req)
	assert.Contains(t, w.Header().Get("Cache-Control"), "max-age")

	// Test YAML endpoint
	req = httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	w = httptest.NewRecorder()
	handler.ServeYAML(w, req)
	assert.Contains(t, w.Header().Get("Cache-Control"), "max-age")
}

func TestHandlerMiddleware(t *testing.T) {
	spec := createTestSpec()
	handler := NewHandler(spec)

	// Create a simple handler chain with the middleware
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := handler.Middleware(innerHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())
}

func TestHandlerJSONValidation(t *testing.T) {
	spec := createTestSpec()
	handler := NewHandler(spec)

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()
	handler.ServeJSON(w, req)

	// Parse and validate the JSON
	var parsed OpenAPI
	err := json.NewDecoder(w.Body).Decode(&parsed)
	require.NoError(t, err)

	// Validate the spec structure
	result := ValidateSpec(&parsed)
	assert.True(t, result.Valid, "Validation errors: %v", result.Errors)
}

func TestHandlerYAMLValidation(t *testing.T) {
	spec := createTestSpec()
	handler := NewHandler(spec)

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	w := httptest.NewRecorder()
	handler.ServeYAML(w, req)

	// Parse and validate the YAML
	var parsed OpenAPI
	err := yaml.NewDecoder(w.Body).Decode(&parsed)
	require.NoError(t, err)

	// Validate the spec structure
	result := ValidateSpec(&parsed)
	assert.True(t, result.Valid, "Validation errors: %v", result.Errors)
}

func TestHandlerWithEmptySpec(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Empty API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{},
		Components: Components{
			Schemas:         map[string]*Schema{},
			SecuritySchemes: map[string]*SecurityScheme{},
		},
	}
	handler := NewHandler(spec)

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()
	handler.ServeJSON(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var parsed OpenAPI
	err := json.NewDecoder(w.Body).Decode(&parsed)
	require.NoError(t, err)
	assert.Equal(t, "Empty API", parsed.Info.Title)
}

func TestHandlerJSONContainsAllFields(t *testing.T) {
	spec := createTestSpec()
	handler := NewHandler(spec)

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()
	handler.ServeJSON(w, req)

	body := w.Body.String()

	// Check that all expected fields are present
	assert.True(t, strings.Contains(body, `"openapi"`))
	assert.True(t, strings.Contains(body, `"info"`))
	assert.True(t, strings.Contains(body, `"paths"`))
	assert.True(t, strings.Contains(body, `"components"`))
	assert.True(t, strings.Contains(body, `"servers"`))
}

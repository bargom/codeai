// Package validation provides input validation with detailed error messages.
package validation

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ValidationMiddleware Tests - Query Parameters
// =============================================================================

func TestValidationMiddleware_QueryParams(t *testing.T) {
	queryParams := []ParamDef{
		{Name: "limit", Type: "integer", Required: true},
		{Name: "offset", Type: "integer", Required: false},
	}

	handler := ValidationMiddleware(nil, queryParams)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	t.Run("valid query params", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?limit=10&offset=5", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "OK", rec.Body.String())
	})

	t.Run("missing required query param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?offset=5", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var resp map[string]any
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, "validation failed", resp["error"])
		assert.NotNil(t, resp["details"])
	})

	t.Run("optional param not provided", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?limit=10", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

// =============================================================================
// ValidationMiddleware Tests - Body Parameters
// =============================================================================

func TestValidationMiddleware_BodyParams(t *testing.T) {
	bodyParams := []ParamDef{
		{Name: "name", Type: "string", Required: true},
		{Name: "email", Type: "email", Required: true},
		{Name: "age", Type: "integer", Required: false},
	}

	handler := ValidationMiddleware(bodyParams, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	t.Run("valid body", func(t *testing.T) {
		body := `{"name": "John", "email": "john@example.com", "age": 30}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(body))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("missing required field", func(t *testing.T) {
		body := `{"name": "John"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(body))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var resp map[string]any
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, "validation failed", resp["error"])
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		body := `{invalid json}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(body))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var resp map[string]any
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, "invalid JSON body", resp["error"])
	})

	t.Run("empty body with body params", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// Empty body should pass through (no body to validate)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("body preserved for downstream handler", func(t *testing.T) {
		var receivedBody map[string]any

		innerHandler := ValidationMiddleware(bodyParams, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&receivedBody)
			w.WriteHeader(http.StatusOK)
		}))

		body := `{"name": "John", "email": "john@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(body))
		rec := httptest.NewRecorder()

		innerHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "John", receivedBody["name"])
		assert.Equal(t, "john@example.com", receivedBody["email"])
	})
}

// =============================================================================
// ValidationMiddleware Tests - Combined Query and Body
// =============================================================================

func TestValidationMiddleware_Combined(t *testing.T) {
	queryParams := []ParamDef{
		{Name: "version", Type: "string", Required: true},
	}
	bodyParams := []ParamDef{
		{Name: "name", Type: "string", Required: true},
	}

	handler := ValidationMiddleware(bodyParams, queryParams)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("both valid", func(t *testing.T) {
		body := `{"name": "John"}`
		req := httptest.NewRequest(http.MethodPost, "/?version=v1", strings.NewReader(body))
		req.ContentLength = int64(len(body))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("query fails first", func(t *testing.T) {
		body := `{"name": "John"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.ContentLength = int64(len(body))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("query valid body invalid", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/?version=v1", strings.NewReader(body))
		req.ContentLength = int64(len(body))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// =============================================================================
// ValidationMiddleware Tests - No Params
// =============================================================================

func TestValidationMiddleware_NoParams(t *testing.T) {
	handler := ValidationMiddleware(nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// =============================================================================
// convertQueryValue Tests
// =============================================================================

func TestConvertQueryValue(t *testing.T) {
	t.Run("integer valid", func(t *testing.T) {
		v := convertQueryValue("42", "integer")
		assert.Equal(t, float64(42), v)
	})

	t.Run("integer invalid", func(t *testing.T) {
		v := convertQueryValue("not-a-number", "integer")
		assert.Equal(t, "not-a-number", v)
	})

	t.Run("number valid", func(t *testing.T) {
		v := convertQueryValue("42.5", "number")
		assert.Equal(t, 42.5, v)
	})

	t.Run("decimal valid", func(t *testing.T) {
		v := convertQueryValue("42.5", "decimal")
		assert.Equal(t, 42.5, v)
	})

	t.Run("number invalid", func(t *testing.T) {
		v := convertQueryValue("not-a-number", "number")
		assert.Equal(t, "not-a-number", v)
	})

	t.Run("boolean true", func(t *testing.T) {
		v := convertQueryValue("true", "boolean")
		assert.Equal(t, true, v)
	})

	t.Run("boolean false", func(t *testing.T) {
		v := convertQueryValue("false", "boolean")
		assert.Equal(t, false, v)
	})

	t.Run("boolean 1", func(t *testing.T) {
		v := convertQueryValue("1", "boolean")
		assert.Equal(t, true, v)
	})

	t.Run("boolean 0", func(t *testing.T) {
		v := convertQueryValue("0", "boolean")
		assert.Equal(t, false, v)
	})

	t.Run("boolean invalid", func(t *testing.T) {
		v := convertQueryValue("not-a-bool", "boolean")
		assert.Equal(t, "not-a-bool", v)
	})

	t.Run("string passthrough", func(t *testing.T) {
		v := convertQueryValue("hello", "string")
		assert.Equal(t, "hello", v)
	})

	t.Run("unknown type passthrough", func(t *testing.T) {
		v := convertQueryValue("hello", "custom")
		assert.Equal(t, "hello", v)
	})
}

// =============================================================================
// ValidateBody Helper Tests
// =============================================================================

func TestValidateBody(t *testing.T) {
	bodyParams := []ParamDef{
		{Name: "name", Type: "string", Required: true},
	}

	handler := ValidateBody(bodyParams)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("valid body", func(t *testing.T) {
		body := `{"name": "John"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.ContentLength = int64(len(body))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.ContentLength = int64(len(body))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// =============================================================================
// ValidateQuery Helper Tests
// =============================================================================

func TestValidateQuery(t *testing.T) {
	queryParams := []ParamDef{
		{Name: "page", Type: "integer", Required: true},
	}

	handler := ValidateQuery(queryParams)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("valid query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?page=1", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// =============================================================================
// Error Response Format Tests
// =============================================================================

func TestValidationMiddleware_ErrorResponseFormat(t *testing.T) {
	bodyParams := []ParamDef{
		{Name: "name", Type: "string", Required: true},
		{Name: "email", Type: "email", Required: true},
	}

	handler := ValidationMiddleware(bodyParams, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := `{"name": 123, "email": "invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.ContentLength = int64(len(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, "validation failed", resp["error"])
	details, ok := resp["details"].([]any)
	require.True(t, ok)
	assert.Len(t, details, 2)

	// Check details structure
	for _, d := range details {
		detail := d.(map[string]any)
		assert.NotEmpty(t, detail["field"])
		assert.NotEmpty(t, detail["rule"])
		assert.NotEmpty(t, detail["message"])
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestValidationMiddleware_EdgeCases(t *testing.T) {
	t.Run("empty query params list", func(t *testing.T) {
		handler := ValidationMiddleware(nil, []ParamDef{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/?random=value", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("empty body params list", func(t *testing.T) {
		handler := ValidationMiddleware([]ParamDef{}, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		body := `{"random": "value"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.ContentLength = int64(len(body))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("zero content length", func(t *testing.T) {
		bodyParams := []ParamDef{
			{Name: "name", Type: "string", Required: true},
		}
		handler := ValidationMiddleware(bodyParams, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte{}))
		req.ContentLength = 0
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// Zero content length should skip body validation
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("content type conversion in query", func(t *testing.T) {
		queryParams := []ParamDef{
			{Name: "active", Type: "boolean", Required: true},
			{Name: "count", Type: "integer", Required: true},
			{Name: "price", Type: "number", Required: true},
		}

		handler := ValidationMiddleware(nil, queryParams)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/?active=true&count=5&price=19.99", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

// =============================================================================
// writeError Tests (via middleware)
// =============================================================================

func TestWriteError(t *testing.T) {
	bodyParams := []ParamDef{
		{Name: "name", Type: "string", Required: true},
	}

	handler := ValidationMiddleware(bodyParams, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Invalid JSON triggers writeError
	body := `{not json`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.ContentLength = int64(len(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp map[string]string
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid JSON body", resp["error"])
}

// =============================================================================
// Query Parameter Type Validation After Conversion
// =============================================================================

func TestValidationMiddleware_QueryTypeValidation(t *testing.T) {
	min := float64(1)
	max := float64(100)
	queryParams := []ParamDef{
		{Name: "limit", Type: "integer", Required: true, Min: &min, Max: &max},
	}

	handler := ValidationMiddleware(nil, queryParams)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("valid integer within range", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?limit=50", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("integer below min", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?limit=0", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("integer above max", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?limit=150", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("non-integer string", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?limit=abc", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// =============================================================================
// Body Read Error Test
// =============================================================================

type errorReader struct{}

func (e errorReader) Read(p []byte) (n int, err error) {
	return 0, assert.AnError
}

func TestValidationMiddleware_BodyReadError(t *testing.T) {
	bodyParams := []ParamDef{
		{Name: "name", Type: "string", Required: true},
	}

	handler := ValidationMiddleware(bodyParams, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", errorReader{})
	req.ContentLength = 100 // Non-zero to trigger body reading
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp map[string]string
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "failed to read request body", resp["error"])
}

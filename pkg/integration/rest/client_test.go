package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/bargom/codeai/pkg/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig(baseURL string) integration.Config {
	return integration.NewConfigBuilder("test-service").
		BaseURL(baseURL).
		Timeout(5 * time.Second).
		MaxRetries(2).
		RetryDelay(10 * time.Millisecond).
		CircuitBreakerThreshold(3).
		CircuitBreakerTimeout(100 * time.Millisecond).
		EnableMetrics(false).
		EnableLogging(false).
		Build()
}

func TestNew(t *testing.T) {
	t.Run("creates client with valid config", func(t *testing.T) {
		config := testConfig("https://api.example.com")
		client, err := New(config)

		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, config.ServiceName, client.Config().ServiceName)
	})

	t.Run("returns error for invalid config", func(t *testing.T) {
		config := integration.Config{} // Missing required fields
		client, err := New(config)

		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestClient_Get(t *testing.T) {
	t.Run("successful GET request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/users/123", r.URL.Path)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"name": "John"})
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		resp, err := client.Get(context.Background(), "/users/123")

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]string
		err = resp.UnmarshalJSON(&result)
		require.NoError(t, err)
		assert.Equal(t, "John", result["name"])
	})

	t.Run("GET with query parameters", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test", r.URL.Query().Get("search"))
			assert.Equal(t, "10", r.URL.Query().Get("limit"))

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		query := url.Values{}
		query.Set("search", "test")
		query.Set("limit", "10")

		resp, err := client.Get(context.Background(), "/search", WithQuery(query))

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("GET with custom headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		resp, err := client.Get(context.Background(), "/endpoint",
			WithHeader("X-Custom-Header", "custom-value"))

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestClient_Post(t *testing.T) {
	t.Run("successful POST request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "John", body["name"])

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{"id": 1, "name": "John"})
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		resp, err := client.Post(context.Background(), "/users", map[string]string{"name": "John"})

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})
}

func TestClient_Put(t *testing.T) {
	t.Run("successful PUT request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPut, r.Method)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		resp, err := client.Put(context.Background(), "/users/123", map[string]string{"name": "Jane"})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestClient_Patch(t *testing.T) {
	t.Run("successful PATCH request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPatch, r.Method)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		resp, err := client.Patch(context.Background(), "/users/123", map[string]string{"name": "Jane"})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestClient_Delete(t *testing.T) {
	t.Run("successful DELETE request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)

			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		resp, err := client.Delete(context.Background(), "/users/123")

		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

func TestClient_ErrorHandling(t *testing.T) {
	t.Run("returns error for 4xx responses", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "not found"}`))
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		resp, err := client.Get(context.Background(), "/users/999", WithSkipRetry())

		assert.Error(t, err)
		// Response with error body is returned along with the error
		if resp != nil {
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		}
		// Verify error contains status code information
		httpErr, ok := err.(*integration.HTTPError)
		require.True(t, ok)
		assert.Equal(t, http.StatusNotFound, httpErr.StatusCode)
	})

	t.Run("returns error for 5xx responses", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		_, err = client.Get(context.Background(), "/error", WithSkipRetry())

		assert.Error(t, err)
		assert.Equal(t, 1, callCount)
	})
}

func TestClient_Authentication(t *testing.T) {
	t.Run("bearer token authentication", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := integration.NewConfigBuilder("test").
			BaseURL(server.URL).
			BearerAuth("test-token").
			EnableMetrics(false).
			EnableLogging(false).
			Build()

		client, err := New(config)
		require.NoError(t, err)

		resp, err := client.Get(context.Background(), "/protected")

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("API key authentication", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-api-key", r.Header.Get("X-API-Key"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := integration.NewConfigBuilder("test").
			BaseURL(server.URL).
			APIKeyAuth("test-api-key", "X-API-Key").
			EnableMetrics(false).
			EnableLogging(false).
			Build()

		client, err := New(config)
		require.NoError(t, err)

		resp, err := client.Get(context.Background(), "/protected")

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("basic authentication", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "user", username)
			assert.Equal(t, "pass", password)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := integration.NewConfigBuilder("test").
			BaseURL(server.URL).
			BasicAuth("user", "pass").
			EnableMetrics(false).
			EnableLogging(false).
			Build()

		client, err := New(config)
		require.NoError(t, err)

		resp, err := client.Get(context.Background(), "/protected")

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestClient_CircuitBreaker(t *testing.T) {
	t.Run("opens circuit after failures", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		config := integration.NewConfigBuilder("test").
			BaseURL(server.URL).
			CircuitBreakerThreshold(2).
			CircuitBreakerTimeout(100 * time.Millisecond).
			MaxRetries(1). // Disable retries for this test
			EnableMetrics(false).
			EnableLogging(false).
			Build()

		client, err := New(config)
		require.NoError(t, err)

		// First two calls should fail and open the circuit
		_, err = client.Get(context.Background(), "/error")
		assert.Error(t, err)

		_, err = client.Get(context.Background(), "/error")
		assert.Error(t, err)

		// Circuit should be open now
		assert.Equal(t, integration.StateOpen, client.CircuitBreaker().State())

		// Third call should be blocked by circuit breaker
		_, err = client.Get(context.Background(), "/error")
		assert.Equal(t, integration.ErrCircuitOpen, err)

		// Server should only have been called twice
		assert.Equal(t, 2, callCount)
	})

	t.Run("skip circuit breaker option", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := integration.NewConfigBuilder("test").
			BaseURL(server.URL).
			CircuitBreakerThreshold(1).
			EnableMetrics(false).
			EnableLogging(false).
			Build()

		client, err := New(config)
		require.NoError(t, err)

		resp, err := client.Get(context.Background(), "/endpoint", WithSkipCircuit())

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestClient_Retry(t *testing.T) {
	t.Run("retries on 5xx errors", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := integration.NewConfigBuilder("test").
			BaseURL(server.URL).
			MaxRetries(3).
			RetryDelay(1 * time.Millisecond).
			CircuitBreakerThreshold(10). // High threshold to avoid opening
			EnableMetrics(false).
			EnableLogging(false).
			Build()

		client, err := New(config)
		require.NoError(t, err)

		resp, err := client.Get(context.Background(), "/flaky")

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, 3, callCount)
	})

	t.Run("does not retry on 4xx errors", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		config := integration.NewConfigBuilder("test").
			BaseURL(server.URL).
			MaxRetries(3).
			CircuitBreakerThreshold(10).
			EnableMetrics(false).
			EnableLogging(false).
			Build()

		client, err := New(config)
		require.NoError(t, err)

		_, err = client.Get(context.Background(), "/bad-request")

		assert.Error(t, err)
		assert.Equal(t, 1, callCount) // Should not retry
	})

	t.Run("skip retry option", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		config := integration.NewConfigBuilder("test").
			BaseURL(server.URL).
			MaxRetries(3).
			CircuitBreakerThreshold(10).
			EnableMetrics(false).
			EnableLogging(false).
			Build()

		client, err := New(config)
		require.NoError(t, err)

		_, err = client.Get(context.Background(), "/error", WithSkipRetry())

		assert.Error(t, err)
		assert.Equal(t, 1, callCount) // Should not retry
	})
}

func TestClient_Timeout(t *testing.T) {
	t.Run("respects custom timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := integration.NewConfigBuilder("test").
			BaseURL(server.URL).
			Timeout(50 * time.Millisecond).
			MaxRetries(1).
			CircuitBreakerThreshold(10).
			EnableMetrics(false).
			EnableLogging(false).
			Build()

		client, err := New(config)
		require.NoError(t, err)

		_, err = client.Get(context.Background(), "/slow", WithTimeout(10*time.Millisecond))

		assert.Error(t, err)
	})
}

func TestClient_Headers(t *testing.T) {
	t.Run("sets default headers from config", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "custom-user-agent", r.Header.Get("User-Agent"))
			assert.Equal(t, "value1", r.Header.Get("X-Custom-1"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := integration.NewConfigBuilder("test").
			BaseURL(server.URL).
			UserAgent("custom-user-agent").
			Header("X-Custom-1", "value1").
			EnableMetrics(false).
			EnableLogging(false).
			Build()

		client, err := New(config)
		require.NoError(t, err)

		resp, err := client.Get(context.Background(), "/endpoint")

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestResponse(t *testing.T) {
	t.Run("IsSuccess returns true for 2xx", func(t *testing.T) {
		resp := &Response{StatusCode: 200}
		assert.True(t, resp.IsSuccess())

		resp.StatusCode = 201
		assert.True(t, resp.IsSuccess())

		resp.StatusCode = 299
		assert.True(t, resp.IsSuccess())
	})

	t.Run("IsClientError returns true for 4xx", func(t *testing.T) {
		resp := &Response{StatusCode: 400}
		assert.True(t, resp.IsClientError())

		resp.StatusCode = 404
		assert.True(t, resp.IsClientError())

		resp.StatusCode = 499
		assert.True(t, resp.IsClientError())
	})

	t.Run("IsServerError returns true for 5xx", func(t *testing.T) {
		resp := &Response{StatusCode: 500}
		assert.True(t, resp.IsServerError())

		resp.StatusCode = 503
		assert.True(t, resp.IsServerError())
	})

	t.Run("UnmarshalJSON works correctly", func(t *testing.T) {
		resp := &Response{
			Body: []byte(`{"name": "John", "age": 30}`),
		}

		var result struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		err := resp.UnmarshalJSON(&result)
		require.NoError(t, err)
		assert.Equal(t, "John", result.Name)
		assert.Equal(t, 30, result.Age)
	})

	t.Run("UnmarshalJSON handles empty body", func(t *testing.T) {
		resp := &Response{Body: nil}

		var result map[string]string
		err := resp.UnmarshalJSON(&result)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestMiddleware(t *testing.T) {
	t.Run("middleware receives request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "middleware-value", r.Header.Get("X-Middleware"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := testConfig(server.URL)
		client, err := New(config)
		require.NoError(t, err)

		// Add custom middleware
		client.Use(NewHeaderMiddleware(map[string]string{"X-Middleware": "middleware-value"}))

		resp, err := client.Get(context.Background(), "/endpoint")

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

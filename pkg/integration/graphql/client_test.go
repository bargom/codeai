package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bargom/codeai/pkg/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig(baseURL string) integration.Config {
	return integration.NewConfigBuilder("test-graphql").
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
		config := testConfig("https://api.example.com/graphql")
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

func TestClient_Query(t *testing.T) {
	t.Run("successful query", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var req Request
			json.NewDecoder(r.Body).Decode(&req)
			assert.Contains(t, req.Query, "query")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"user": map[string]interface{}{
						"id":   "123",
						"name": "John",
					},
				},
			})
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		var result struct {
			User struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"user"`
		}

		err = client.Query(context.Background(), `query { user(id: "123") { id name } }`, nil, &result)

		require.NoError(t, err)
		assert.Equal(t, "123", result.User.ID)
		assert.Equal(t, "John", result.User.Name)
	})

	t.Run("query with variables", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req Request
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "123", req.Variables["id"])

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"user": map[string]interface{}{
						"id": "123",
					},
				},
			})
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		var result struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		}

		vars := map[string]interface{}{
			"id": "123",
		}

		err = client.Query(context.Background(), `query GetUser($id: ID!) { user(id: $id) { id } }`, vars, &result)

		require.NoError(t, err)
		assert.Equal(t, "123", result.User.ID)
	})

	t.Run("query returns GraphQL errors", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": nil,
				"errors": []map[string]interface{}{
					{
						"message": "User not found",
						"path":    []interface{}{"user"},
					},
				},
			})
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		var result struct{}
		err = client.Query(context.Background(), `query { user(id: "999") { id } }`, nil, &result)

		assert.Error(t, err)
		graphQLErr, ok := err.(*GraphQLError)
		require.True(t, ok)
		assert.Equal(t, "User not found", graphQLErr.Error())
	})
}

func TestClient_Mutate(t *testing.T) {
	t.Run("successful mutation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req Request
			json.NewDecoder(r.Body).Decode(&req)
			assert.Contains(t, req.Query, "mutation")

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"createUser": map[string]interface{}{
						"id":   "456",
						"name": "Jane",
					},
				},
			})
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		var result struct {
			CreateUser struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"createUser"`
		}

		vars := map[string]interface{}{
			"name": "Jane",
		}

		err = client.Mutate(context.Background(), `mutation CreateUser($name: String!) { createUser(name: $name) { id name } }`, vars, &result)

		require.NoError(t, err)
		assert.Equal(t, "456", result.CreateUser.ID)
		assert.Equal(t, "Jane", result.CreateUser.Name)
	})
}

func TestClient_Execute(t *testing.T) {
	t.Run("returns response with data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"hello": "world",
				},
			})
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		resp, err := client.Execute(context.Background(), &Request{
			Query: `query { hello }`,
		}, nil)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.False(t, resp.HasError())
	})

	t.Run("returns response with errors", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": nil,
				"errors": []map[string]interface{}{
					{"message": "Error 1"},
					{"message": "Error 2"},
				},
			})
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		resp, err := client.Execute(context.Background(), &Request{
			Query: `query { invalid }`,
		}, nil)

		require.NoError(t, err) // HTTP request succeeded
		assert.True(t, resp.HasError())
		assert.Equal(t, 2, len(resp.Errors))
		assert.Equal(t, "Error 1", resp.FirstError().Message)
	})

	t.Run("returns error for HTTP failures", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client, err := New(testConfig(server.URL))
		require.NoError(t, err)

		_, err = client.Execute(context.Background(), &Request{
			Query: `query { hello }`,
		}, &ExecuteOptions{SkipRetry: true})

		assert.Error(t, err)
	})
}

func TestClient_Authentication(t *testing.T) {
	t.Run("bearer token authentication", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"data": nil})
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

		_, err = client.Execute(context.Background(), &Request{Query: `query { test }`}, nil)

		require.NoError(t, err)
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
			MaxRetries(1).
			EnableMetrics(false).
			EnableLogging(false).
			Build()

		client, err := New(config)
		require.NoError(t, err)

		// First two calls should fail and open the circuit
		_, err = client.Execute(context.Background(), &Request{Query: `query { test }`}, nil)
		assert.Error(t, err)

		_, err = client.Execute(context.Background(), &Request{Query: `query { test }`}, nil)
		assert.Error(t, err)

		// Circuit should be open now
		assert.Equal(t, integration.StateOpen, client.CircuitBreaker().State())

		// Third call should be blocked by circuit breaker
		_, err = client.Execute(context.Background(), &Request{Query: `query { test }`}, nil)
		assert.Equal(t, integration.ErrCircuitOpen, err)

		// Server should only have been called twice
		assert.Equal(t, 2, callCount)
	})
}

func TestResponse(t *testing.T) {
	t.Run("HasError returns true when errors present", func(t *testing.T) {
		resp := &Response{
			Errors: []Error{{Message: "test error"}},
		}
		assert.True(t, resp.HasError())
	})

	t.Run("HasError returns false when no errors", func(t *testing.T) {
		resp := &Response{}
		assert.False(t, resp.HasError())
	})

	t.Run("FirstError returns first error", func(t *testing.T) {
		resp := &Response{
			Errors: []Error{
				{Message: "first"},
				{Message: "second"},
			},
		}
		assert.Equal(t, "first", resp.FirstError().Message)
	})

	t.Run("FirstError returns nil when no errors", func(t *testing.T) {
		resp := &Response{}
		assert.Nil(t, resp.FirstError())
	})

	t.Run("UnmarshalData works correctly", func(t *testing.T) {
		resp := &Response{
			Data: json.RawMessage(`{"name": "John"}`),
		}

		var result struct {
			Name string `json:"name"`
		}

		err := resp.UnmarshalData(&result)
		require.NoError(t, err)
		assert.Equal(t, "John", result.Name)
	})
}

func TestError(t *testing.T) {
	t.Run("Error implements error interface", func(t *testing.T) {
		err := Error{Message: "test error"}
		assert.Equal(t, "test error", err.Error())
	})
}

func TestGraphQLError(t *testing.T) {
	t.Run("Error returns first error message", func(t *testing.T) {
		err := &GraphQLError{
			Errors: []Error{
				{Message: "first error"},
				{Message: "second error"},
			},
		}
		assert.Equal(t, "first error", err.Error())
	})

	t.Run("Error returns default message when empty", func(t *testing.T) {
		err := &GraphQLError{}
		assert.Equal(t, "unknown graphql error", err.Error())
	})
}

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bargom/codeai/pkg/integration"
	"github.com/bargom/codeai/pkg/integration/graphql"
	"github.com/bargom/codeai/pkg/integration/rest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_RESTClient_WithCircuitBreaker(t *testing.T) {
	var requestCount int32

	// Create a server that fails initially then succeeds
	// First 2 requests fail (to open the circuit), then all subsequent requests succeed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	config := integration.NewConfigBuilder("test-service").
		BaseURL(server.URL).
		Timeout(5 * time.Second).
		MaxRetries(1). // Disable retries to test circuit breaker
		CircuitBreakerThreshold(2).
		CircuitBreakerTimeout(50 * time.Millisecond).
		CircuitBreakerHalfOpenRequests(1).
		EnableMetrics(false).
		EnableLogging(false).
		Build()

	client, err := rest.New(config)
	require.NoError(t, err)

	// First 2 calls fail and open the circuit
	_, err = client.Get(context.Background(), "/endpoint")
	assert.Error(t, err)

	_, err = client.Get(context.Background(), "/endpoint")
	assert.Error(t, err)

	// Circuit is now open
	assert.Equal(t, integration.StateOpen, client.CircuitBreaker().State())

	// This call should be blocked
	_, err = client.Get(context.Background(), "/endpoint")
	assert.Equal(t, integration.ErrCircuitOpen, err)

	// Wait for circuit to half-open
	time.Sleep(60 * time.Millisecond)

	// Circuit should transition to half-open on next attempt
	_, err = client.Get(context.Background(), "/endpoint")
	// This might still fail if we haven't reached the success threshold
	// The 4th request should succeed (count > 3)

	// After successful request, circuit should close
	time.Sleep(10 * time.Millisecond)

	resp, err := client.Get(context.Background(), "/endpoint")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIntegration_RESTClient_WithRetry(t *testing.T) {
	var requestCount int32

	// Create a server that fails twice then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	config := integration.NewConfigBuilder("test-service").
		BaseURL(server.URL).
		Timeout(5 * time.Second).
		MaxRetries(3).
		RetryDelay(10 * time.Millisecond).
		CircuitBreakerThreshold(10). // High threshold to avoid opening
		EnableMetrics(false).
		EnableLogging(false).
		Build()

	client, err := rest.New(config)
	require.NoError(t, err)

	resp, err := client.Get(context.Background(), "/endpoint")

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount))
}

func TestIntegration_RESTClient_WithTimeout(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := integration.NewConfigBuilder("test-service").
		BaseURL(server.URL).
		Timeout(50 * time.Millisecond).
		MaxRetries(1). // Disable retries
		CircuitBreakerThreshold(10).
		EnableMetrics(false).
		EnableLogging(false).
		Build()

	client, err := rest.New(config)
	require.NoError(t, err)

	_, err = client.Get(context.Background(), "/slow")

	assert.Error(t, err)
	// Should be a timeout error
	assert.Contains(t, err.Error(), "timed out")
}

func TestIntegration_GraphQLClient_FullWorkflow(t *testing.T) {
	// Create a mock GraphQL server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphql.Request
		json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")

		if req.OperationName == "GetUser" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"user": map[string]interface{}{
						"id":    req.Variables["id"],
						"name":  "Test User",
						"email": "test@example.com",
					},
				},
			})
			return
		}

		if req.OperationName == "CreateUser" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"createUser": map[string]interface{}{
						"id":   "new-id",
						"name": req.Variables["name"],
					},
				},
			})
			return
		}

		// Default response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": nil,
			"errors": []map[string]interface{}{
				{"message": "Unknown operation"},
			},
		})
	}))
	defer server.Close()

	config := integration.NewConfigBuilder("graphql-service").
		BaseURL(server.URL).
		Timeout(5 * time.Second).
		EnableMetrics(false).
		EnableLogging(false).
		Build()

	client, err := graphql.New(config)
	require.NoError(t, err)

	// Test query
	var getUserResult struct {
		User struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"user"`
	}

	err = client.QueryWithOperation(
		context.Background(),
		`query GetUser($id: ID!) { user(id: $id) { id name email } }`,
		"GetUser",
		map[string]interface{}{"id": "123"},
		&getUserResult,
	)

	require.NoError(t, err)
	assert.Equal(t, "123", getUserResult.User.ID)
	assert.Equal(t, "Test User", getUserResult.User.Name)

	// Test mutation
	var createUserResult struct {
		CreateUser struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"createUser"`
	}

	err = client.QueryWithOperation(
		context.Background(),
		`mutation CreateUser($name: String!) { createUser(name: $name) { id name } }`,
		"CreateUser",
		map[string]interface{}{"name": "New User"},
		&createUserResult,
	)

	require.NoError(t, err)
	assert.Equal(t, "new-id", createUserResult.CreateUser.ID)
	assert.Equal(t, "New User", createUserResult.CreateUser.Name)
}

func TestIntegration_QueryBuilder(t *testing.T) {
	query := graphql.NewQuery().
		Name("GetUsers").
		Variable("limit", "Int", 10).
		Variable("filter", "UserFilter").
		Field("users").
			VarArg("limit", "limit").
			VarArg("filter", "filter").
			Select("id", "name", "email").
			SubField("profile").
				Select("avatar", "bio").
				Done().
			Done().
		Build()

	assert.Contains(t, query, "query GetUsers")
	assert.Contains(t, query, "$limit: Int = 10")
	assert.Contains(t, query, "$filter: UserFilter")
	assert.Contains(t, query, "users(limit: $limit, filter: $filter)")
	assert.Contains(t, query, "id")
	assert.Contains(t, query, "name")
	assert.Contains(t, query, "email")
	assert.Contains(t, query, "profile")
	assert.Contains(t, query, "avatar")
	assert.Contains(t, query, "bio")
}

func TestIntegration_CircuitBreakerRegistry(t *testing.T) {
	registry := integration.NewCircuitBreakerRegistry(integration.CircuitBreakerConfig{
		FailureThreshold: 3,
		Timeout:          100 * time.Millisecond,
		HalfOpenRequests: 2,
	})

	// Get creates circuit breakers on demand
	cb1 := registry.Get("service-a")
	cb2 := registry.Get("service-b")
	cb3 := registry.Get("service-a") // Same as cb1

	assert.Same(t, cb1, cb3)
	assert.NotSame(t, cb1, cb2)

	// Test circuit breaker behavior
	ctx := context.Background()

	// Record failures for service-a
	for i := 0; i < 3; i++ {
		cb1.Execute(ctx, func(ctx context.Context) error {
			return &integration.HTTPError{StatusCode: 500, Message: "error"}
		})
	}

	// Service-a should be open
	assert.Equal(t, integration.StateOpen, cb1.State())
	// Service-b should still be closed
	assert.Equal(t, integration.StateClosed, cb2.State())

	// Get all stats
	stats := registry.Stats()
	assert.Len(t, stats, 2)
}

func TestIntegration_ConfigBuilder(t *testing.T) {
	config := integration.NewConfigBuilder("my-service").
		BaseURL("https://api.example.com").
		Timeout(30 * time.Second).
		ConnectTimeout(10 * time.Second).
		MaxRetries(5).
		RetryDelay(100 * time.Millisecond).
		MaxRetryDelay(10 * time.Second).
		RetryMultiplier(2.0).
		RetryJitter(0.1).
		CircuitBreakerThreshold(10).
		CircuitBreakerTimeout(60 * time.Second).
		CircuitBreakerHalfOpenRequests(3).
		BearerAuth("my-token").
		Header("X-Custom", "value").
		UserAgent("MyApp/1.0").
		EnableMetrics(true).
		EnableLogging(true).
		RedactFields("password", "secret").
		Build()

	err := config.Validate()
	require.NoError(t, err)

	assert.Equal(t, "my-service", config.ServiceName)
	assert.Equal(t, "https://api.example.com", config.BaseURL)
	assert.Equal(t, 30*time.Second, config.Timeout.Default)
	assert.Equal(t, 10*time.Second, config.Timeout.Connect)
	assert.Equal(t, 5, config.Retry.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, config.Retry.BaseDelay)
	assert.Equal(t, 10*time.Second, config.Retry.MaxDelay)
	assert.Equal(t, 2.0, config.Retry.Multiplier)
	assert.Equal(t, 0.1, config.Retry.Jitter)
	assert.Equal(t, 10, config.CircuitBreaker.FailureThreshold)
	assert.Equal(t, 60*time.Second, config.CircuitBreaker.Timeout)
	assert.Equal(t, 3, config.CircuitBreaker.HalfOpenRequests)
	assert.Equal(t, integration.AuthBearer, config.Auth.Type)
	assert.Equal(t, "my-token", config.Auth.Token)
	assert.Equal(t, "value", config.Headers["X-Custom"])
	assert.Equal(t, "MyApp/1.0", config.UserAgent)
	assert.True(t, config.EnableMetrics)
	assert.True(t, config.EnableLogging)
	assert.Contains(t, config.RedactFields, "password")
	assert.Contains(t, config.RedactFields, "secret")
}

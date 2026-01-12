//go:build integration

// Package scenarios provides end-to-end integration tests for Phase 4 components.
package scenarios

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bargom/codeai/internal/health"
	"github.com/bargom/codeai/internal/health/checks"
	"github.com/bargom/codeai/internal/shutdown"
	"github.com/bargom/codeai/pkg/integration"
	"github.com/bargom/codeai/pkg/logging"
	"github.com/bargom/codeai/pkg/metrics"
)

// Phase4TestServer represents a test HTTP server with all Phase 4 features enabled.
type Phase4TestServer struct {
	Server          *httptest.Server
	Logger          *logging.Logger
	MetricsRegistry *metrics.Registry
	HealthRegistry  *health.Registry
	ShutdownManager *shutdown.Manager

	logBuffer *bytes.Buffer
	mux       *http.ServeMux
}

// NewPhase4TestServer creates a test server with all Phase 4 components.
func NewPhase4TestServer(t *testing.T) *Phase4TestServer {
	t.Helper()

	logBuffer := &bytes.Buffer{}
	logger := logging.NewWithWriter(logging.Config{
		Level:  "debug",
		Format: "json",
	}, logBuffer)

	metricsRegistry := metrics.NewRegistry(metrics.DefaultConfig())
	healthRegistry := health.NewRegistry("test-1.0.0")
	shutdownManager := shutdown.NewManagerWithDefaults()

	mux := http.NewServeMux()

	// Register health endpoints
	healthHandler := health.NewHandler(healthRegistry)
	healthHandler.RegisterRoutes(mux)

	// Register metrics endpoint
	mux.Handle("/metrics", metricsRegistry.Handler())

	// Test endpoint with logging and metrics
	httpMiddleware := logging.NewHTTPMiddleware(logger.Logger)
	mux.Handle("/api/test", httpMiddleware.Handler(
		metrics.HTTPMiddleware(metricsRegistry)(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			}),
		),
	))

	// Slow endpoint for testing graceful shutdown
	mux.Handle("/api/slow", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"completed": true}`))
		case <-r.Context().Done():
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error": "request cancelled"}`))
		}
	}))

	server := httptest.NewServer(mux)

	return &Phase4TestServer{
		Server:          server,
		Logger:          logger,
		MetricsRegistry: metricsRegistry,
		HealthRegistry:  healthRegistry,
		ShutdownManager: shutdownManager,
		logBuffer:       logBuffer,
		mux:             mux,
	}
}

// Close shuts down the test server.
func (s *Phase4TestServer) Close() {
	s.Server.Close()
}

// GetLogs returns the captured log output.
func (s *Phase4TestServer) GetLogs() string {
	return s.logBuffer.String()
}

// ResetLogs clears the log buffer.
func (s *Phase4TestServer) ResetLogs() {
	s.logBuffer.Reset()
}

func TestPhase4_HTTPServerWithAllFeatures(t *testing.T) {
	server := NewPhase4TestServer(t)
	defer server.Close()

	t.Run("request creates logs with request ID", func(t *testing.T) {
		server.ResetLogs()

		resp, err := http.Get(server.Server.URL + "/api/test")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify response
		var result map[string]string
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "ok", result["status"])

		// Verify logs contain request info
		logs := server.GetLogs()
		assert.Contains(t, logs, "/api/test")
	})

	t.Run("request ID is returned in response header", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.Server.URL+"/api/test", nil)
		require.NoError(t, err)

		// Set a custom request ID
		customID := "test-request-123"
		req.Header.Set("X-Request-ID", customID)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Response should echo the request ID
		assert.Equal(t, customID, resp.Header.Get("X-Request-ID"))
	})

	t.Run("metrics are recorded for requests", func(t *testing.T) {
		// Make several requests
		for i := 0; i < 5; i++ {
			resp, err := http.Get(server.Server.URL + "/api/test")
			require.NoError(t, err)
			resp.Body.Close()
		}

		// Check metrics endpoint
		resp, err := http.Get(server.Server.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		metricsOutput := string(body)
		assert.Contains(t, metricsOutput, "http_requests_total")
		assert.Contains(t, metricsOutput, "http_request_duration_seconds")
	})
}

func TestPhase4_HealthChecks(t *testing.T) {
	server := NewPhase4TestServer(t)
	defer server.Close()

	t.Run("health endpoint returns healthy", func(t *testing.T) {
		resp, err := http.Get(server.Server.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result health.Response
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, health.StatusHealthy, result.Status)
	})

	t.Run("liveness endpoint always returns 200", func(t *testing.T) {
		resp, err := http.Get(server.Server.URL + "/health/live")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("readiness endpoint returns 200 when ready", func(t *testing.T) {
		resp, err := http.Get(server.Server.URL + "/health/ready")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("health check with custom checker", func(t *testing.T) {
		// Add a custom health checker
		server.HealthRegistry.Register(checks.NewCustomChecker("custom", func(ctx context.Context) health.CheckResult {
			return health.CheckResult{
				Status:  health.StatusHealthy,
				Message: "Custom check passed",
			}
		}))

		resp, err := http.Get(server.Server.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result health.Response
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, health.StatusHealthy, result.Status)
		assert.Contains(t, result.Checks, "custom")
	})

	t.Run("health endpoint returns unhealthy when check fails", func(t *testing.T) {
		// Create a new registry to avoid affecting other tests
		unhealthyRegistry := health.NewRegistry("test-1.0.0")
		unhealthyRegistry.Register(checks.NewCustomChecker("failing", func(ctx context.Context) health.CheckResult {
			return health.CheckResult{
				Status:  health.StatusUnhealthy,
				Message: "Service unavailable",
			}
		}, checks.WithCustomSeverity(health.SeverityCritical)))

		mux := http.NewServeMux()
		healthHandler := health.NewHandler(unhealthyRegistry)
		healthHandler.RegisterRoutes(mux)
		testServer := httptest.NewServer(mux)
		defer testServer.Close()

		resp, err := http.Get(testServer.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	})
}

func TestPhase4_GracefulShutdown(t *testing.T) {
	t.Run("shutdown manager executes hooks in priority order", func(t *testing.T) {
		manager := shutdown.NewManagerWithDefaults()

		var executionOrder []string
		var mu sync.Mutex

		manager.Register("low", 10, func(ctx context.Context) error {
			mu.Lock()
			executionOrder = append(executionOrder, "low")
			mu.Unlock()
			return nil
		})

		manager.Register("high", 90, func(ctx context.Context) error {
			mu.Lock()
			executionOrder = append(executionOrder, "high")
			mu.Unlock()
			return nil
		})

		manager.Register("medium", 50, func(ctx context.Context) error {
			mu.Lock()
			executionOrder = append(executionOrder, "medium")
			mu.Unlock()
			return nil
		})

		manager.Shutdown()
		manager.Wait()

		// Should execute in priority order (high to low)
		assert.Equal(t, []string{"high", "medium", "low"}, executionOrder)
	})

	t.Run("shutdown manager recovers from panic", func(t *testing.T) {
		manager := shutdown.NewManagerWithDefaults()

		normalExecuted := false

		manager.Register("panicking", 90, func(ctx context.Context) error {
			panic("test panic")
		})

		manager.Register("normal", 50, func(ctx context.Context) error {
			normalExecuted = true
			return nil
		})

		manager.Shutdown()
		manager.Wait()

		// Should have error from panic but continue
		errs := manager.Errors()
		assert.NotEmpty(t, errs)
		assert.True(t, normalExecuted, "Normal hook should still execute after panic")
	})
}

func TestPhase4_IntegrationModule(t *testing.T) {
	t.Run("circuit breaker opens after failures", func(t *testing.T) {
		cb := integration.NewCircuitBreaker("test-service", integration.CircuitBreakerConfig{
			FailureThreshold: 3,
			Timeout:          50 * time.Millisecond,
			HalfOpenRequests: 1,
		})

		// Record failures to trigger open state
		for i := 0; i < 3; i++ {
			cb.RecordFailure()
		}

		assert.Equal(t, integration.StateOpen, cb.State())

		// Requests should be blocked
		err := cb.Allow()
		assert.Error(t, err)

		// After reset timeout, should transition to half-open
		time.Sleep(60 * time.Millisecond)
		err = cb.Allow()
		assert.NoError(t, err)
		assert.Equal(t, integration.StateHalfOpen, cb.State())
	})

	t.Run("circuit breaker closes after success in half-open", func(t *testing.T) {
		cb := integration.NewCircuitBreaker("test-service", integration.CircuitBreakerConfig{
			FailureThreshold: 3,
			Timeout:          50 * time.Millisecond,
			HalfOpenRequests: 2,
		})

		// Open the circuit
		for i := 0; i < 3; i++ {
			cb.RecordFailure()
		}

		// Wait for half-open
		time.Sleep(60 * time.Millisecond)
		cb.Allow()

		// Record successes in half-open
		cb.RecordSuccess()
		cb.RecordSuccess()

		// Should be closed now
		assert.Equal(t, integration.StateClosed, cb.State())
	})

	t.Run("retry with exponential backoff", func(t *testing.T) {
		retryer := integration.NewRetryer(integration.RetryConfig{
			MaxAttempts: 3,
			BaseDelay:   10 * time.Millisecond,
			MaxDelay:    100 * time.Millisecond,
			Multiplier:  2.0,
			Jitter:      0.1,
		})

		attempts := 0
		err := retryer.Do(context.Background(), func(ctx context.Context) error {
			attempts++
			if attempts < 3 {
				return integration.NewHTTPError(503, "service unavailable")
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("retry stops on non-retryable error", func(t *testing.T) {
		retryer := integration.NewRetryer(integration.RetryConfig{
			MaxAttempts: 5,
			BaseDelay:   10 * time.Millisecond,
		})

		attempts := 0
		err := retryer.Do(context.Background(), func(ctx context.Context) error {
			attempts++
			return integration.NewHTTPError(400, "bad request") // Not retryable
		})

		assert.Error(t, err)
		assert.Equal(t, 1, attempts, "Should not retry on non-retryable error")
	})

	t.Run("retry respects context cancellation", func(t *testing.T) {
		retryer := integration.NewRetryer(integration.RetryConfig{
			MaxAttempts: 10,
			BaseDelay:   100 * time.Millisecond,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		attempts := 0
		err := retryer.Do(ctx, func(ctx context.Context) error {
			attempts++
			return integration.NewHTTPError(503, "service unavailable")
		})

		assert.Error(t, err)
		assert.True(t, attempts < 10, "Should have been cancelled before max attempts")
	})
}

func TestPhase4_SensitiveDataRedaction(t *testing.T) {
	t.Run("sensitive data is redacted in logs", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		logger := logging.NewWithWriter(logging.Config{
			Level:  "debug",
			Format: "json",
		}, logBuffer)

		mux := http.NewServeMux()
		httpMiddleware := logging.NewHTTPMiddleware(logger.Logger).WithVerbosity(logging.VerbosityVerbose)
		mux.Handle("/api/login", httpMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})))

		server := httptest.NewServer(mux)
		defer server.Close()

		// Make request with sensitive header
		req, _ := http.NewRequest("POST", server.URL+"/api/login", nil)
		req.Header.Set("Authorization", "Bearer secret-token-12345")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		logs := logBuffer.String()

		// Authorization header should be redacted
		assert.NotContains(t, logs, "secret-token-12345")
		// Verify the REDACTED placeholder is present
		assert.Contains(t, logs, "[REDACTED]")
	})
}

func TestPhase4_MetricsIntegration(t *testing.T) {
	t.Run("database metrics are recorded", func(t *testing.T) {
		registry := metrics.NewRegistry(metrics.DefaultConfig())
		dbMetrics := registry.DB()

		// Simulate database queries
		for i := 0; i < 10; i++ {
			timer := dbMetrics.NewQueryTimer(metrics.OperationSelect, "users")
			time.Sleep(time.Millisecond)
			timer.Done(nil)
		}

		// Check that metrics were recorded
		resp := httptest.NewRecorder()
		registry.Handler().ServeHTTP(resp, httptest.NewRequest("GET", "/metrics", nil))

		body := resp.Body.String()
		assert.Contains(t, body, "db_query_duration_seconds")
	})

	t.Run("workflow metrics are recorded", func(t *testing.T) {
		registry := metrics.NewRegistry(metrics.DefaultConfig())
		wfMetrics := registry.Workflow()

		// Simulate workflow execution
		wfMetrics.IncActiveWorkflows("test-workflow")
		time.Sleep(10 * time.Millisecond)
		wfMetrics.RecordExecution("test-workflow", metrics.WorkflowStatusSuccess, 10*time.Millisecond)
		wfMetrics.DecActiveWorkflows("test-workflow")

		// Check metrics
		resp := httptest.NewRecorder()
		registry.Handler().ServeHTTP(resp, httptest.NewRequest("GET", "/metrics", nil))

		body := resp.Body.String()
		assert.Contains(t, body, "workflow_executions_total")
	})

	t.Run("integration metrics track circuit breaker state", func(t *testing.T) {
		registry := metrics.NewRegistry(metrics.DefaultConfig())
		intMetrics := registry.Integration()

		// Record circuit breaker state changes
		intMetrics.SetCircuitBreakerState("payment-service", metrics.CircuitBreakerClosed)
		intMetrics.SetCircuitBreakerState("payment-service", metrics.CircuitBreakerOpen)
		intMetrics.SetCircuitBreakerState("payment-service", metrics.CircuitBreakerHalfOpen)

		// Check metrics
		resp := httptest.NewRecorder()
		registry.Handler().ServeHTTP(resp, httptest.NewRequest("GET", "/metrics", nil))

		body := resp.Body.String()
		assert.Contains(t, body, "integration_circuit_breaker_state")
	})
}

func TestPhase4_EndToEndScenario(t *testing.T) {
	t.Run("full request lifecycle with all Phase 4 features", func(t *testing.T) {
		// Setup
		logBuffer := &bytes.Buffer{}
		logger := logging.NewWithWriter(logging.Config{
			Level:  "debug",
			Format: "json",
		}, logBuffer)

		metricsRegistry := metrics.NewRegistry(metrics.DefaultConfig())
		healthRegistry := health.NewRegistry("test-1.0.0")
		shutdownManager := shutdown.NewManagerWithDefaults()

		// Add health checks
		healthRegistry.Register(checks.NewCustomChecker("api", func(ctx context.Context) health.CheckResult {
			return health.CheckResult{Status: health.StatusHealthy, Message: "API is healthy"}
		}))

		// Circuit breaker for external service
		cb := integration.NewCircuitBreaker("external-api", integration.CircuitBreakerConfig{
			FailureThreshold: 5,
			Timeout:          time.Second,
		})

		// Setup shutdown hook
		shutdownComplete := make(chan bool, 1)
		shutdownManager.Register("cleanup", 50, func(ctx context.Context) error {
			shutdownComplete <- true
			return nil
		})

		// Create HTTP server
		mux := http.NewServeMux()
		healthHandler := health.NewHandler(healthRegistry)
		healthHandler.RegisterRoutes(mux)
		mux.Handle("/metrics", metricsRegistry.Handler())

		httpMiddleware := logging.NewHTTPMiddleware(logger.Logger)
		intMetrics := metricsRegistry.Integration()

		mux.Handle("/api/process", httpMiddleware.Handler(
			metrics.HTTPMiddleware(metricsRegistry)(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Check circuit breaker
					if err := cb.Allow(); err != nil {
						intMetrics.RecordError("external-api", "process", "circuit_open")
						w.WriteHeader(http.StatusServiceUnavailable)
						json.NewEncoder(w).Encode(map[string]string{"error": "service unavailable"})
						return
					}

					// Simulate external call
					timer := intMetrics.NewCallTimer("external-api", "process")
					time.Sleep(5 * time.Millisecond)
					timer.Success()

					cb.RecordSuccess()

					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]string{"result": "processed"})
				}),
			),
		))

		server := httptest.NewServer(mux)
		defer server.Close()

		// Test lifecycle

		// 1. Health check passes
		resp, err := http.Get(server.URL + "/health")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// 2. Make API requests
		for i := 0; i < 5; i++ {
			resp, err := http.Get(server.URL + "/api/process")
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		}

		// 3. Verify logs contain request information
		logs := logBuffer.String()
		assert.Contains(t, logs, "/api/process")

		// 4. Verify metrics are recorded
		resp, err = http.Get(server.URL + "/metrics")
		require.NoError(t, err)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		metricsOutput := string(body)
		assert.Contains(t, metricsOutput, "http_requests_total")
		assert.Contains(t, metricsOutput, `method="GET"`)

		// 5. Trigger shutdown
		go func() {
			shutdownManager.Shutdown()
		}()

		// 6. Verify shutdown hook executed
		select {
		case <-shutdownComplete:
			// Success
		case <-time.After(2 * time.Second):
			t.Fatal("Shutdown did not complete in time")
		}
	})
}

func TestPhase4_ConcurrentRequests(t *testing.T) {
	server := NewPhase4TestServer(t)
	defer server.Close()

	t.Run("handles concurrent requests correctly", func(t *testing.T) {
		const numRequests = 100
		var wg sync.WaitGroup
		errors := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				req, _ := http.NewRequest("GET", server.Server.URL+"/api/test", nil)
				req.Header.Set("X-Request-ID", fmt.Sprintf("request-%d", i))

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					errors <- err
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					errors <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		var errs []error
		for err := range errors {
			errs = append(errs, err)
		}

		assert.Empty(t, errs, "All concurrent requests should succeed")

		// Verify metrics reflect all requests
		resp, err := http.Get(server.Server.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "http_requests_total")
	})
}

package metrics_test

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bargom/codeai/pkg/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationHTTPServer tests the full integration with an HTTP server.
func TestIntegrationHTTPServer(t *testing.T) {
	// Create a fresh registry for this test
	cfg := metrics.DefaultConfig().
		WithVersion("1.0.0").
		WithEnvironment("test")
	cfg.EnableProcessMetrics = false
	cfg.EnableRuntimeMetrics = false
	reg := metrics.NewRegistry(cfg)

	// Create a test server with metrics middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate some work
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"users": []}`))
	})
	mux.HandleFunc("/api/users/create", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": "123"}`))
	})
	mux.HandleFunc("/api/error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal error"}`))
	})
	mux.Handle("/metrics", reg.Handler())

	// Wrap with metrics middleware
	handler := metrics.HTTPMiddleware(reg)(mux)
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make several requests
	client := server.Client()

	// 5 GET requests to /api/users
	for i := 0; i < 5; i++ {
		resp, err := client.Get(server.URL + "/api/users")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// 3 POST requests to /api/users/create
	for i := 0; i < 3; i++ {
		resp, err := client.Post(server.URL+"/api/users/create", "application/json", nil)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	// 2 GET requests to /api/error
	for i := 0; i < 2; i++ {
		resp, err := client.Get(server.URL + "/api/error")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	}

	// Fetch and parse metrics
	resp, err := client.Get(server.URL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	metricsMap := parsePrometheusMetrics(t, resp)

	// Verify request counts
	assert.Equal(t, 5.0, metricsMap["codeai_http_requests_total{method=\"GET\",path=\"/api/users\",status_code=\"200\"}"])
	assert.Equal(t, 3.0, metricsMap["codeai_http_requests_total{method=\"POST\",path=\"/api/users/create\",status_code=\"201\"}"])
	assert.Equal(t, 2.0, metricsMap["codeai_http_requests_total{method=\"GET\",path=\"/api/error\",status_code=\"500\"}"])

	// Verify histogram counts exist
	assert.Contains(t, metricsMap, "codeai_http_request_duration_seconds_count{method=\"GET\",path=\"/api/users\"}")
	durationCount := metricsMap["codeai_http_request_duration_seconds_count{method=\"GET\",path=\"/api/users\"}"]
	assert.Equal(t, 5.0, durationCount)
}

// TestIntegrationPathNormalization tests that path normalization works correctly.
func TestIntegrationPathNormalization(t *testing.T) {
	cfg := metrics.DefaultConfig()
	cfg.EnableProcessMetrics = false
	cfg.EnableRuntimeMetrics = false
	reg := metrics.NewRegistry(cfg)

	mux := http.NewServeMux()
	// Using wildcard pattern to match any user ID
	mux.HandleFunc("/api/users/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/metrics", reg.Handler())

	handler := metrics.HTTPMiddleware(reg)(mux)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := server.Client()

	// Make requests with different IDs
	testIDs := []string{
		"123",
		"456",
		"789",
		"550e8400-e29b-41d4-a716-446655440000", // UUID
		"507f1f77bcf86cd799439011",              // MongoDB ObjectID
	}

	for _, id := range testIDs {
		resp, err := client.Get(server.URL + "/api/users/" + id)
		require.NoError(t, err)
		resp.Body.Close()
	}

	// Fetch metrics
	resp, err := client.Get(server.URL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	metricsMap := parsePrometheusMetrics(t, resp)

	// All requests should be normalized to the same path pattern
	normalizedCount := metricsMap["codeai_http_requests_total{method=\"GET\",path=\"/api/users/{id}\",status_code=\"200\"}"]
	assert.Equal(t, float64(len(testIDs)), normalizedCount)
}

// TestIntegrationDatabaseMetrics tests database metrics recording.
func TestIntegrationDatabaseMetrics(t *testing.T) {
	cfg := metrics.DefaultConfig()
	cfg.EnableProcessMetrics = false
	cfg.EnableRuntimeMetrics = false
	reg := metrics.NewRegistry(cfg)

	dbMetrics := reg.DB()

	// Simulate various database operations
	for i := 0; i < 10; i++ {
		timer := dbMetrics.NewQueryTimer(metrics.OperationSelect, "users")
		time.Sleep(1 * time.Millisecond)
		timer.Done(nil)
	}

	for i := 0; i < 5; i++ {
		dbMetrics.RecordQuery(metrics.OperationInsert, "orders", 5*time.Millisecond, nil)
	}

	// Simulate some errors
	dbMetrics.RecordQuery(metrics.OperationUpdate, "products", 10*time.Millisecond, assert.AnError)
	dbMetrics.RecordQueryError(metrics.OperationUpdate, "products", "constraint_violation")

	// Update connection stats
	dbMetrics.UpdateConnectionStats(8, 2, 10)

	// Fetch metrics
	handler := reg.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	metricsMap := parsePrometheusMetrics(t, rec.Result())

	// Verify query counts
	assert.Equal(t, 10.0, metricsMap["codeai_db_queries_total{operation=\"SELECT\",status=\"success\",table=\"users\"}"])
	assert.Equal(t, 5.0, metricsMap["codeai_db_queries_total{operation=\"INSERT\",status=\"success\",table=\"orders\"}"])
	assert.Equal(t, 1.0, metricsMap["codeai_db_queries_total{operation=\"UPDATE\",status=\"error\",table=\"products\"}"])

	// Verify error count
	assert.Equal(t, 1.0, metricsMap["codeai_db_query_errors_total{error_type=\"constraint_violation\",operation=\"UPDATE\",table=\"products\"}"])

	// Verify connection stats
	assert.Equal(t, 8.0, metricsMap["codeai_db_connections_active"])
	assert.Equal(t, 2.0, metricsMap["codeai_db_connections_idle"])
	assert.Equal(t, 10.0, metricsMap["codeai_db_connections_max"])
}

// TestIntegrationWorkflowMetrics tests workflow metrics recording.
func TestIntegrationWorkflowMetrics(t *testing.T) {
	cfg := metrics.DefaultConfig()
	cfg.EnableProcessMetrics = false
	cfg.EnableRuntimeMetrics = false
	reg := metrics.NewRegistry(cfg)

	wfMetrics := reg.Workflow()

	// Simulate workflow executions
	for i := 0; i < 5; i++ {
		timer := wfMetrics.NewExecutionTimer("order_processing")
		time.Sleep(5 * time.Millisecond)
		timer.Success()
	}

	for i := 0; i < 2; i++ {
		timer := wfMetrics.NewExecutionTimer("order_processing")
		time.Sleep(2 * time.Millisecond)
		timer.Failure()
	}

	// Simulate workflow steps
	for i := 0; i < 5; i++ {
		stepTimer := wfMetrics.NewStepTimer("order_processing", "validate_payment")
		time.Sleep(1 * time.Millisecond)
		stepTimer.Done()
	}

	// Fetch metrics
	handler := reg.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	metricsMap := parsePrometheusMetrics(t, rec.Result())

	// Verify workflow counts
	assert.Equal(t, 5.0, metricsMap["codeai_workflow_executions_total{status=\"success\",workflow_name=\"order_processing\"}"])
	assert.Equal(t, 2.0, metricsMap["codeai_workflow_executions_total{status=\"failure\",workflow_name=\"order_processing\"}"])

	// Verify active count is 0 (all completed)
	assert.Equal(t, 0.0, metricsMap["codeai_workflow_active_count{workflow_name=\"order_processing\"}"])

	// Verify step histogram exists
	assert.Contains(t, metricsMap, "codeai_workflow_step_duration_seconds_count{step_name=\"validate_payment\",workflow_name=\"order_processing\"}")
}

// TestIntegrationIntegrationMetrics tests external integration metrics.
func TestIntegrationIntegrationMetrics(t *testing.T) {
	cfg := metrics.DefaultConfig()
	cfg.EnableProcessMetrics = false
	cfg.EnableRuntimeMetrics = false
	reg := metrics.NewRegistry(cfg)

	intMetrics := reg.Integration()

	// Simulate successful API calls
	for i := 0; i < 10; i++ {
		timer := intMetrics.NewCallTimer("payment_gateway", "/charge")
		time.Sleep(2 * time.Millisecond)
		timer.Success()
	}

	// Simulate failed calls with retries
	for i := 0; i < 3; i++ {
		timer := intMetrics.NewCallTimer("email_service", "/send")
		timer.Retry()
		timer.Retry()
		time.Sleep(1 * time.Millisecond)
		timer.Error("timeout")
	}

	// Set circuit breaker states
	intMetrics.SetCircuitBreakerState("payment_gateway", metrics.CircuitBreakerClosed)
	intMetrics.SetCircuitBreakerState("email_service", metrics.CircuitBreakerOpen)

	// Fetch metrics
	handler := reg.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	metricsMap := parsePrometheusMetrics(t, rec.Result())

	// Verify API call counts
	assert.Equal(t, 10.0, metricsMap["codeai_integration_calls_total{endpoint=\"/charge\",service_name=\"payment_gateway\",status_code=\"200\"}"])
	assert.Equal(t, 3.0, metricsMap["codeai_integration_calls_total{endpoint=\"/send\",service_name=\"email_service\",status_code=\"500\"}"])

	// Verify retry counts
	assert.Equal(t, 6.0, metricsMap["codeai_integration_retries_total{endpoint=\"/send\",service_name=\"email_service\"}"])

	// Verify error counts
	assert.Equal(t, 3.0, metricsMap["codeai_integration_errors_total{endpoint=\"/send\",error_type=\"timeout\",service_name=\"email_service\"}"])

	// Verify circuit breaker states
	assert.Equal(t, 1.0, metricsMap["codeai_integration_circuit_breaker_state{service_name=\"payment_gateway\",state=\"closed\"}"])
	assert.Equal(t, 0.0, metricsMap["codeai_integration_circuit_breaker_state{service_name=\"payment_gateway\",state=\"open\"}"])
	assert.Equal(t, 1.0, metricsMap["codeai_integration_circuit_breaker_state{service_name=\"email_service\",state=\"open\"}"])
	assert.Equal(t, 0.0, metricsMap["codeai_integration_circuit_breaker_state{service_name=\"email_service\",state=\"closed\"}"])
}

// TestIntegrationMetricsEndpoint tests the /metrics endpoint in isolation.
func TestIntegrationMetricsEndpoint(t *testing.T) {
	cfg := metrics.DefaultConfig()
	cfg.EnableProcessMetrics = false
	cfg.EnableRuntimeMetrics = false
	reg := metrics.NewRegistry(cfg)

	// Record some metrics
	reg.HTTP().RecordRequest("GET", "/test", 200, 0.1, 100, 200)
	reg.DB().RecordQuery(metrics.OperationSelect, "users", 10*time.Millisecond, nil)
	reg.Workflow().RecordExecution("test_workflow", metrics.WorkflowStatusSuccess, 1*time.Second)
	reg.Integration().RecordCall("test_service", "/api", 200, 50*time.Millisecond)

	// Create test server just for metrics
	server := httptest.NewServer(reg.Handler())
	defer server.Close()

	// Fetch metrics
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")

	metricsMap := parsePrometheusMetrics(t, resp)

	// Verify all metric types are present
	assert.Contains(t, metricsMap, "codeai_http_requests_total{method=\"GET\",path=\"/test\",status_code=\"200\"}")
	assert.Contains(t, metricsMap, "codeai_db_queries_total{operation=\"SELECT\",status=\"success\",table=\"users\"}")
	assert.Contains(t, metricsMap, "codeai_workflow_executions_total{status=\"success\",workflow_name=\"test_workflow\"}")
	assert.Contains(t, metricsMap, "codeai_integration_calls_total{endpoint=\"/api\",service_name=\"test_service\",status_code=\"200\"}")
}

// TestIntegrationWithProcessAndRuntimeMetrics tests that process/runtime metrics are included.
func TestIntegrationWithProcessAndRuntimeMetrics(t *testing.T) {
	cfg := metrics.DefaultConfig()
	cfg.EnableProcessMetrics = true
	cfg.EnableRuntimeMetrics = true
	reg := metrics.NewRegistry(cfg)

	handler := reg.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Check for Go runtime metrics
	assert.Contains(t, body, "go_goroutines")
	assert.Contains(t, body, "go_memstats_alloc_bytes")

	// Check for process metrics
	assert.Contains(t, body, "process_cpu_seconds_total")
	assert.Contains(t, body, "process_resident_memory_bytes")
}

// TestIntegrationConcurrentMetricsRecording tests concurrent access to metrics.
func TestIntegrationConcurrentMetricsRecording(t *testing.T) {
	cfg := metrics.DefaultConfig()
	cfg.EnableProcessMetrics = false
	cfg.EnableRuntimeMetrics = false
	reg := metrics.NewRegistry(cfg)

	httpMetrics := reg.HTTP()
	dbMetrics := reg.DB()
	wfMetrics := reg.Workflow()
	intMetrics := reg.Integration()

	// Run concurrent operations
	done := make(chan bool)
	numGoroutines := 10
	numOperations := 100

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				httpMetrics.RecordRequest("GET", "/api/test", 200, 0.01, 100, 200)
				dbMetrics.RecordQuery(metrics.OperationSelect, "users", 5*time.Millisecond, nil)
				wfMetrics.RecordExecution("concurrent_workflow", metrics.WorkflowStatusSuccess, 10*time.Millisecond)
				intMetrics.RecordCall("test_service", "/api", 200, 5*time.Millisecond)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Fetch metrics
	handler := reg.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	metricsMap := parsePrometheusMetrics(t, rec.Result())

	expectedCount := float64(numGoroutines * numOperations)

	// Verify all counts match expected
	assert.Equal(t, expectedCount, metricsMap["codeai_http_requests_total{method=\"GET\",path=\"/api/test\",status_code=\"200\"}"])
	assert.Equal(t, expectedCount, metricsMap["codeai_db_queries_total{operation=\"SELECT\",status=\"success\",table=\"users\"}"])
	assert.Equal(t, expectedCount, metricsMap["codeai_workflow_executions_total{status=\"success\",workflow_name=\"concurrent_workflow\"}"])
	assert.Equal(t, expectedCount, metricsMap["codeai_integration_calls_total{endpoint=\"/api\",service_name=\"test_service\",status_code=\"200\"}"])
}

// parsePrometheusMetrics parses a Prometheus text format response into a map.
func parsePrometheusMetrics(t *testing.T, resp *http.Response) map[string]float64 {
	t.Helper()
	result := make(map[string]float64)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || len(strings.TrimSpace(line)) == 0 {
			continue
		}

		// Parse metric line: metric_name{labels} value
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			metricName := parts[0]
			valueStr := parts[1]

			value, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				continue
			}

			result[metricName] = value
		}
	}

	require.NoError(t, scanner.Err())
	return result
}

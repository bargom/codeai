package metrics

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "codeai", cfg.Namespace)
	assert.True(t, cfg.EnableProcessMetrics)
	assert.True(t, cfg.EnableRuntimeMetrics)
	assert.Equal(t, "unknown", cfg.DefaultLabels["version"])
	assert.Equal(t, "development", cfg.DefaultLabels["environment"])
}

func TestConfigWithMethods(t *testing.T) {
	cfg := DefaultConfig().
		WithVersion("1.0.0").
		WithEnvironment("production").
		WithInstance("node-1")

	assert.Equal(t, "1.0.0", cfg.DefaultLabels["version"])
	assert.Equal(t, "production", cfg.DefaultLabels["environment"])
	assert.Equal(t, "node-1", cfg.DefaultLabels["instance"])
}

func TestNewRegistry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnableProcessMetrics = false
	cfg.EnableRuntimeMetrics = false

	reg := NewRegistry(cfg)

	assert.NotNil(t, reg)
	assert.NotNil(t, reg.PrometheusRegistry())
	assert.Equal(t, cfg.Namespace, reg.Config().Namespace)
}

func TestHTTPMetrics(t *testing.T) {
	reg := newTestRegistry()
	httpMetrics := reg.HTTP()

	t.Run("RecordRequest", func(t *testing.T) {
		httpMetrics.RecordRequest("GET", "/api/users", 200, 0.1, 100, 500)

		// Verify counter was incremented
		counter, err := getCounterValue(reg.httpRequestsTotal, "GET", "/api/users", "200")
		require.NoError(t, err)
		assert.Equal(t, float64(1), counter)
	})

	t.Run("ActiveRequests", func(t *testing.T) {
		httpMetrics.IncActiveRequests("POST", "/api/items")
		httpMetrics.IncActiveRequests("POST", "/api/items")

		gauge, err := getGaugeValue(reg.httpActiveRequests, "POST", "/api/items")
		require.NoError(t, err)
		assert.Equal(t, float64(2), gauge)

		httpMetrics.DecActiveRequests("POST", "/api/items")
		gauge, err = getGaugeValue(reg.httpActiveRequests, "POST", "/api/items")
		require.NoError(t, err)
		assert.Equal(t, float64(1), gauge)
	})
}

func TestHTTPMiddleware(t *testing.T) {
	reg := newTestRegistry()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	middleware := HTTPMiddleware(reg)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Hello, World!", rec.Body.String())

	// Verify metrics were recorded
	counter, err := getCounterValue(reg.httpRequestsTotal, "GET", "/api/test", "200")
	require.NoError(t, err)
	assert.Equal(t, float64(1), counter)
}

func TestHTTPMiddlewareWithSkipPaths(t *testing.T) {
	reg := newTestRegistry()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := HTTPMiddlewareWithOptions(reg, MiddlewareOptions{
		SkipPaths: []string{"/health"},
	})
	wrappedHandler := middleware(handler)

	// Request to skipped path
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)

	// Should not have metrics for skipped path - counter should be 0
	counter, err := getCounterValue(reg.httpRequestsTotal, "GET", "/health", "200")
	if err == nil {
		assert.Equal(t, float64(0), counter)
	}
	// If error, that's also fine - metric wasn't created

	// Request to non-skipped path
	req2 := httptest.NewRequest("GET", "/api/users", nil)
	rec2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec2, req2)

	// Should have metrics for non-skipped path
	counter2, err := getCounterValue(reg.httpRequestsTotal, "GET", "/api/users", "200")
	require.NoError(t, err)
	assert.Equal(t, float64(1), counter2)
}

func TestDefaultPathNormalizer(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users/123", "/users/{id}"},
		{"/users/123/posts", "/users/{id}/posts"},
		{"/users/123e4567-e89b-12d3-a456-426614174000", "/users/{id}"},
		{"/items/507f1f77bcf86cd799439011", "/items/{id}"},
		{"/api/v1/users", "/api/v1/users"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := DefaultPathNormalizer(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDatabaseMetrics(t *testing.T) {
	reg := newTestRegistry()
	dbMetrics := reg.DB()

	t.Run("RecordQuery", func(t *testing.T) {
		dbMetrics.RecordQuery(OperationSelect, "users", 10*time.Millisecond, nil)

		counter, err := getCounterValue(reg.dbQueriesTotal, "SELECT", "users", "success")
		require.NoError(t, err)
		assert.Equal(t, float64(1), counter)
	})

	t.Run("RecordQueryWithError", func(t *testing.T) {
		dbMetrics.RecordQuery(OperationInsert, "orders", 50*time.Millisecond, errors.New("constraint violation"))

		counter, err := getCounterValue(reg.dbQueriesTotal, "INSERT", "orders", "error")
		require.NoError(t, err)
		assert.Equal(t, float64(1), counter)
	})

	t.Run("UpdateConnectionStats", func(t *testing.T) {
		dbMetrics.UpdateConnectionStats(10, 5, 20)

		active := getSimpleGaugeValue(reg.dbConnectionsActive)
		idle := getSimpleGaugeValue(reg.dbConnectionsIdle)
		max := getSimpleGaugeValue(reg.dbConnectionsMax)

		assert.Equal(t, float64(10), active)
		assert.Equal(t, float64(5), idle)
		assert.Equal(t, float64(20), max)
	})

	t.Run("UpdateFromDBStats", func(t *testing.T) {
		stats := sql.DBStats{
			InUse:              15,
			Idle:               3,
			MaxOpenConnections: 25,
		}
		dbMetrics.UpdateFromDBStats(stats)

		active := getSimpleGaugeValue(reg.dbConnectionsActive)
		assert.Equal(t, float64(15), active)
	})
}

func TestDetectOperation(t *testing.T) {
	tests := []struct {
		query    string
		expected Operation
	}{
		{"SELECT * FROM users", OperationSelect},
		{"  SELECT id FROM items", OperationSelect},
		{"INSERT INTO users VALUES (...)", OperationInsert},
		{"UPDATE users SET name = 'test'", OperationUpdate},
		{"DELETE FROM users WHERE id = 1", OperationDelete},
		{"TRUNCATE TABLE users", OperationOther},
		{"", OperationOther},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := DetectOperation(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueryTimer(t *testing.T) {
	reg := newTestRegistry()
	dbMetrics := reg.DB()

	timer := dbMetrics.NewQueryTimer(OperationSelect, "users")
	time.Sleep(5 * time.Millisecond)
	timer.Done(nil)

	counter, err := getCounterValue(reg.dbQueriesTotal, "SELECT", "users", "success")
	require.NoError(t, err)
	assert.Equal(t, float64(1), counter)
}

func TestWorkflowMetrics(t *testing.T) {
	reg := newTestRegistry()
	wfMetrics := reg.Workflow()

	t.Run("RecordExecution", func(t *testing.T) {
		wfMetrics.RecordExecution("order_processing", WorkflowStatusSuccess, 5*time.Second)

		counter, err := getCounterValue(reg.workflowExecutionsTotal, "order_processing", "success")
		require.NoError(t, err)
		assert.Equal(t, float64(1), counter)
	})

	t.Run("ActiveWorkflows", func(t *testing.T) {
		wfMetrics.IncActiveWorkflows("data_sync")
		wfMetrics.IncActiveWorkflows("data_sync")

		gauge, err := getGaugeValue(reg.workflowActiveCount, "data_sync")
		require.NoError(t, err)
		assert.Equal(t, float64(2), gauge)

		wfMetrics.DecActiveWorkflows("data_sync")
		gauge, err = getGaugeValue(reg.workflowActiveCount, "data_sync")
		require.NoError(t, err)
		assert.Equal(t, float64(1), gauge)
	})

	t.Run("ExecutionTimer", func(t *testing.T) {
		timer := wfMetrics.NewExecutionTimer("email_campaign")

		gauge, err := getGaugeValue(reg.workflowActiveCount, "email_campaign")
		require.NoError(t, err)
		assert.Equal(t, float64(1), gauge)

		time.Sleep(5 * time.Millisecond)
		timer.Success()

		gauge, err = getGaugeValue(reg.workflowActiveCount, "email_campaign")
		require.NoError(t, err)
		assert.Equal(t, float64(0), gauge)

		counter, err := getCounterValue(reg.workflowExecutionsTotal, "email_campaign", "success")
		require.NoError(t, err)
		assert.Equal(t, float64(1), counter)
	})
}

func TestIntegrationMetrics(t *testing.T) {
	reg := newTestRegistry()
	intMetrics := reg.Integration()

	t.Run("RecordCall", func(t *testing.T) {
		intMetrics.RecordCall("payment_gateway", "/charge", 200, 100*time.Millisecond)

		counter, err := getCounterValue(reg.integrationCallsTotal, "payment_gateway", "/charge", "200")
		require.NoError(t, err)
		assert.Equal(t, float64(1), counter)
	})

	t.Run("RecordError", func(t *testing.T) {
		intMetrics.RecordError("email_service", "/send", "timeout")

		counter, err := getCounterValue(reg.integrationErrors, "email_service", "/send", "timeout")
		require.NoError(t, err)
		assert.Equal(t, float64(1), counter)
	})

	t.Run("RecordRetry", func(t *testing.T) {
		intMetrics.RecordRetry("inventory_api", "/stock")
		intMetrics.RecordRetry("inventory_api", "/stock")

		counter, err := getCounterValue(reg.integrationRetryCount, "inventory_api", "/stock")
		require.NoError(t, err)
		assert.Equal(t, float64(2), counter)
	})

	t.Run("SetCircuitBreakerState", func(t *testing.T) {
		intMetrics.SetCircuitBreakerState("external_api", CircuitBreakerOpen)

		openGauge, err := getGaugeValue(reg.integrationCircuitState, "external_api", "open")
		require.NoError(t, err)
		assert.Equal(t, float64(1), openGauge)

		closedGauge, err := getGaugeValue(reg.integrationCircuitState, "external_api", "closed")
		require.NoError(t, err)
		assert.Equal(t, float64(0), closedGauge)
	})

	t.Run("CallTimer", func(t *testing.T) {
		timer := intMetrics.NewCallTimer("slack_api", "/post")
		timer.Retry()
		timer.Retry()
		time.Sleep(5 * time.Millisecond)
		timer.Success()

		assert.Equal(t, 2, timer.RetryCount())

		retryCounter, err := getCounterValue(reg.integrationRetryCount, "slack_api", "/post")
		require.NoError(t, err)
		assert.Equal(t, float64(2), retryCounter)

		callCounter, err := getCounterValue(reg.integrationCallsTotal, "slack_api", "/post", "200")
		require.NoError(t, err)
		assert.Equal(t, float64(1), callCounter)
	})
}

func TestCircuitBreakerStateValue(t *testing.T) {
	assert.Equal(t, float64(0), CircuitBreakerClosed.Value())
	assert.Equal(t, float64(1), CircuitBreakerHalfOpen.Value())
	assert.Equal(t, float64(2), CircuitBreakerOpen.Value())
}

func TestClassifyHTTPError(t *testing.T) {
	assert.Equal(t, "client_error", ClassifyHTTPError(400))
	assert.Equal(t, "client_error", ClassifyHTTPError(404))
	assert.Equal(t, "server_error", ClassifyHTTPError(500))
	assert.Equal(t, "server_error", ClassifyHTTPError(503))
	assert.Equal(t, "connection_error", ClassifyHTTPError(0))
	assert.Equal(t, "unknown", ClassifyHTTPError(200))
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{nil, ""},
		{errors.New("connection timeout"), "timeout"},
		{errors.New("connection refused"), "connection_refused"},
		{errors.New("no such host"), "dns_error"},
		{errors.New("context canceled"), "cancelled"},
		{errors.New("random error"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := ClassifyError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandler(t *testing.T) {
	reg := newTestRegistry()

	// Record some metrics
	reg.HTTP().RecordRequest("GET", "/test", 200, 0.1, 100, 200)

	// Get handler and make request
	handler := reg.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	body, _ := io.ReadAll(rec.Body)
	bodyStr := string(body)

	// Verify our metrics appear in output
	assert.Contains(t, bodyStr, "codeai_http_requests_total")
	assert.Contains(t, bodyStr, "codeai_http_request_duration_seconds")
}

func TestHandlerWithAuth(t *testing.T) {
	reg := newTestRegistry()

	authFn := func(r *http.Request) bool {
		return r.Header.Get("Authorization") == "Bearer valid-token"
	}

	handler := reg.HandlerWithAuth(authFn)

	t.Run("Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Authorized", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestMetricsResponseWriter(t *testing.T) {
	t.Run("DefaultStatus", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mrw := newMetricsResponseWriter(rec)
		mrw.Write([]byte("test"))

		assert.Equal(t, http.StatusOK, mrw.status)
		assert.Equal(t, int64(4), mrw.size)
	})

	t.Run("CustomStatus", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mrw := newMetricsResponseWriter(rec)
		mrw.WriteHeader(http.StatusNotFound)
		mrw.Write([]byte("not found"))

		assert.Equal(t, http.StatusNotFound, mrw.status)
		assert.Equal(t, int64(9), mrw.size)
	})

	t.Run("Flush", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mrw := newMetricsResponseWriter(rec)
		mrw.Flush()
		assert.True(t, rec.Flushed)
	})

	t.Run("Unwrap", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mrw := newMetricsResponseWriter(rec)
		assert.Equal(t, rec, mrw.Unwrap())
	})
}

// Helper functions for testing

func newTestRegistry() *Registry {
	cfg := DefaultConfig()
	cfg.EnableProcessMetrics = false
	cfg.EnableRuntimeMetrics = false
	return NewRegistry(cfg)
}

func getCounterValue(cv *prometheus.CounterVec, labels ...string) (float64, error) {
	counter, err := cv.GetMetricWithLabelValues(labels...)
	if err != nil {
		return 0, err
	}

	var metric dto.Metric
	if err := counter.Write(&metric); err != nil {
		return 0, err
	}

	return metric.GetCounter().GetValue(), nil
}

func getGaugeValue(gv *prometheus.GaugeVec, labels ...string) (float64, error) {
	gauge, err := gv.GetMetricWithLabelValues(labels...)
	if err != nil {
		return 0, err
	}

	var metric dto.Metric
	if err := gauge.Write(&metric); err != nil {
		return 0, err
	}

	return metric.GetGauge().GetValue(), nil
}

func getSimpleGaugeValue(g prometheus.Gauge) float64 {
	var metric dto.Metric
	g.Write(&metric)
	return metric.GetGauge().GetValue()
}

func TestDBQueryErrorClassification(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{nil, ""},
		{errors.New("connection reset"), "connection"},
		{errors.New("query timeout exceeded"), "timeout"},
		{errors.New("unique constraint violation"), "constraint_violation"},
		{errors.New("duplicate key value"), "duplicate_key"},
		{errors.New("deadlock detected"), "deadlock"},
		{errors.New("no rows in result set"), "not_found"},
		{errors.New("some random error"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := classifyDBError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMultipleRequests(t *testing.T) {
	reg := newTestRegistry()
	httpMetrics := reg.HTTP()

	// Record multiple requests
	for i := 0; i < 10; i++ {
		httpMetrics.RecordRequest("GET", "/api/items", 200, 0.05, 50, 100)
	}

	counter, err := getCounterValue(reg.httpRequestsTotal, "GET", "/api/items", "200")
	require.NoError(t, err)
	assert.Equal(t, float64(10), counter)
}

func TestLabelCardinality(t *testing.T) {
	reg := newTestRegistry()
	httpMetrics := reg.HTTP()

	// Test that different status codes create separate counters
	httpMetrics.RecordRequest("GET", "/api/test", 200, 0.1, 100, 200)
	httpMetrics.RecordRequest("GET", "/api/test", 404, 0.1, 100, 50)
	httpMetrics.RecordRequest("GET", "/api/test", 500, 0.1, 100, 100)

	counter200, err := getCounterValue(reg.httpRequestsTotal, "GET", "/api/test", "200")
	require.NoError(t, err)
	assert.Equal(t, float64(1), counter200)

	counter404, err := getCounterValue(reg.httpRequestsTotal, "GET", "/api/test", "404")
	require.NoError(t, err)
	assert.Equal(t, float64(1), counter404)

	counter500, err := getCounterValue(reg.httpRequestsTotal, "GET", "/api/test", "500")
	require.NoError(t, err)
	assert.Equal(t, float64(1), counter500)
}

func TestHistogramObservations(t *testing.T) {
	reg := newTestRegistry()

	// Record a request with known duration
	reg.HTTP().RecordRequest("GET", "/test", 200, 0.5, 100, 200)

	// The histogram should have recorded the observation
	// This is a basic check that the histogram is working
	handler := reg.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	assert.Contains(t, body, "codeai_http_request_duration_seconds_bucket")
	assert.Contains(t, body, "codeai_http_request_duration_seconds_sum")
	assert.Contains(t, body, "codeai_http_request_duration_seconds_count")
}

func TestWorkflowStepTimer(t *testing.T) {
	reg := newTestRegistry()
	wfMetrics := reg.Workflow()

	timer := wfMetrics.NewStepTimer("order_workflow", "validate_payment")
	time.Sleep(5 * time.Millisecond)
	timer.Done()

	// Verify histogram was observed
	handler := reg.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	assert.Contains(t, body, "codeai_workflow_step_duration_seconds")
	assert.Contains(t, body, "order_workflow")
	assert.Contains(t, body, "validate_payment")
}

func TestServeHTTP(t *testing.T) {
	reg := newTestRegistry()

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	reg.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, strings.Contains(rec.Body.String(), "codeai_"))
}

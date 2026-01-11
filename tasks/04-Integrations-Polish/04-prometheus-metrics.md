# Task: Prometheus Metrics

## Overview
Implement Prometheus metrics for monitoring runtime performance, request rates, and business metrics.

## Phase
Phase 4: Integrations and Polish

## Priority
Medium - Important for production monitoring.

## Dependencies
- 01-Foundation/06-http-module.md

## Description
Create a metrics module that exposes Prometheus-compatible metrics for HTTP requests, database operations, workflow executions, and custom business metrics.

## Detailed Requirements

### 1. Metrics Module (internal/metrics/metrics.go)

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // HTTP metrics
    HTTPRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codeai_http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status"},
    )

    HTTPRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "codeai_http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
        },
        []string{"method", "path"},
    )

    HTTPRequestSize = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "codeai_http_request_size_bytes",
            Help:    "HTTP request size in bytes",
            Buckets: prometheus.ExponentialBuckets(100, 10, 8),
        },
        []string{"method", "path"},
    )

    HTTPResponseSize = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "codeai_http_response_size_bytes",
            Help:    "HTTP response size in bytes",
            Buckets: prometheus.ExponentialBuckets(100, 10, 8),
        },
        []string{"method", "path"},
    )

    // Database metrics
    DBQueryTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codeai_db_queries_total",
            Help: "Total number of database queries",
        },
        []string{"operation", "entity"},
    )

    DBQueryDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "codeai_db_query_duration_seconds",
            Help:    "Database query duration in seconds",
            Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
        },
        []string{"operation", "entity"},
    )

    DBConnectionsActive = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "codeai_db_connections_active",
            Help: "Number of active database connections",
        },
    )

    DBConnectionsIdle = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "codeai_db_connections_idle",
            Help: "Number of idle database connections",
        },
    )

    // Workflow metrics
    WorkflowExecutionsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codeai_workflow_executions_total",
            Help: "Total number of workflow executions",
        },
        []string{"workflow", "status"},
    )

    WorkflowExecutionDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "codeai_workflow_execution_duration_seconds",
            Help:    "Workflow execution duration in seconds",
            Buckets: []float64{.1, .5, 1, 5, 10, 30, 60, 120, 300},
        },
        []string{"workflow"},
    )

    WorkflowStepDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "codeai_workflow_step_duration_seconds",
            Help:    "Workflow step duration in seconds",
            Buckets: []float64{.01, .05, .1, .5, 1, 5, 10, 30},
        },
        []string{"workflow", "step"},
    )

    // Job metrics
    JobExecutionsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codeai_job_executions_total",
            Help: "Total number of job executions",
        },
        []string{"job", "status"},
    )

    JobQueueSize = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "codeai_job_queue_size",
            Help: "Number of jobs in queue",
        },
        []string{"queue"},
    )

    // Cache metrics
    CacheHitsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codeai_cache_hits_total",
            Help: "Total number of cache hits",
        },
        []string{"cache"},
    )

    CacheMissesTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codeai_cache_misses_total",
            Help: "Total number of cache misses",
        },
        []string{"cache"},
    )

    // Integration metrics
    IntegrationCallsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codeai_integration_calls_total",
            Help: "Total number of integration calls",
        },
        []string{"integration", "operation", "status"},
    )

    IntegrationCallDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "codeai_integration_call_duration_seconds",
            Help:    "Integration call duration in seconds",
            Buckets: []float64{.01, .05, .1, .5, 1, 5, 10, 30},
        },
        []string{"integration", "operation"},
    )

    CircuitBreakerState = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "codeai_circuit_breaker_state",
            Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
        },
        []string{"integration"},
    )

    // Event metrics
    EventsEmittedTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codeai_events_emitted_total",
            Help: "Total number of events emitted",
        },
        []string{"event"},
    )

    // Runtime metrics
    RuntimeInfo = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "codeai_runtime_info",
            Help: "Runtime information",
        },
        []string{"version", "go_version"},
    )
)

// RecordHTTPRequest records metrics for an HTTP request
func RecordHTTPRequest(method, path string, status int, duration float64, reqSize, respSize int64) {
    statusStr := fmt.Sprintf("%d", status)
    HTTPRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
    HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
    HTTPRequestSize.WithLabelValues(method, path).Observe(float64(reqSize))
    HTTPResponseSize.WithLabelValues(method, path).Observe(float64(respSize))
}

// RecordDBQuery records metrics for a database query
func RecordDBQuery(operation, entity string, duration float64, err error) {
    status := "success"
    if err != nil {
        status = "error"
    }
    DBQueryTotal.WithLabelValues(operation, entity).Inc()
    DBQueryDuration.WithLabelValues(operation, entity).Observe(duration)
}

// RecordWorkflowExecution records metrics for a workflow execution
func RecordWorkflowExecution(workflow, status string, duration float64) {
    WorkflowExecutionsTotal.WithLabelValues(workflow, status).Inc()
    WorkflowExecutionDuration.WithLabelValues(workflow).Observe(duration)
}

// RecordWorkflowStep records metrics for a workflow step
func RecordWorkflowStep(workflow, step string, duration float64) {
    WorkflowStepDuration.WithLabelValues(workflow, step).Observe(duration)
}

// RecordCacheAccess records cache hit/miss
func RecordCacheAccess(cache string, hit bool) {
    if hit {
        CacheHitsTotal.WithLabelValues(cache).Inc()
    } else {
        CacheMissesTotal.WithLabelValues(cache).Inc()
    }
}
```

### 2. HTTP Metrics Middleware (internal/metrics/http.go)

```go
package metrics

import (
    "net/http"
    "strconv"
    "time"

    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsMiddleware adds metrics collection to HTTP handlers
func MetricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Wrap response writer
        wrapped := &metricsResponseWriter{
            ResponseWriter: w,
            status:        200,
        }

        // Process request
        next.ServeHTTP(wrapped, r)

        // Record metrics
        duration := time.Since(start).Seconds()
        path := normalizePath(r.URL.Path)

        RecordHTTPRequest(
            r.Method,
            path,
            wrapped.status,
            duration,
            r.ContentLength,
            wrapped.size,
        )
    })
}

type metricsResponseWriter struct {
    http.ResponseWriter
    status int
    size   int64
}

func (w *metricsResponseWriter) WriteHeader(status int) {
    w.status = status
    w.ResponseWriter.WriteHeader(status)
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
    n, err := w.ResponseWriter.Write(b)
    w.size += int64(n)
    return n, err
}

// normalizePath removes IDs from paths for better metric grouping
func normalizePath(path string) string {
    // Replace UUID-like segments with {id}
    // /users/123e4567-e89b-12d3-a456-426614174000 -> /users/{id}
    // This is a simplified version; production would use regex
    return path
}

// Handler returns the Prometheus metrics handler
func Handler() http.Handler {
    return promhttp.Handler()
}
```

### 3. Database Metrics Hook (internal/metrics/database.go)

```go
package metrics

import (
    "context"
    "time"
)

type DBMetricsHook struct{}

func (h *DBMetricsHook) BeforeQuery(ctx context.Context, query string) context.Context {
    return context.WithValue(ctx, "query_start", time.Now())
}

func (h *DBMetricsHook) AfterQuery(ctx context.Context, query, entity string, err error) {
    start, ok := ctx.Value("query_start").(time.Time)
    if !ok {
        return
    }

    duration := time.Since(start).Seconds()
    operation := detectOperation(query)

    RecordDBQuery(operation, entity, duration, err)
}

func detectOperation(query string) string {
    // Simple detection based on first word
    switch {
    case len(query) >= 6 && query[:6] == "SELECT":
        return "select"
    case len(query) >= 6 && query[:6] == "INSERT":
        return "insert"
    case len(query) >= 6 && query[:6] == "UPDATE":
        return "update"
    case len(query) >= 6 && query[:6] == "DELETE":
        return "delete"
    default:
        return "other"
    }
}

// UpdateConnectionMetrics updates database connection pool metrics
func UpdateConnectionMetrics(active, idle int) {
    DBConnectionsActive.Set(float64(active))
    DBConnectionsIdle.Set(float64(idle))
}
```

### 4. Custom Business Metrics (internal/metrics/business.go)

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

// BusinessMetrics allows registering custom business metrics
type BusinessMetrics struct {
    counters   map[string]*prometheus.CounterVec
    gauges     map[string]*prometheus.GaugeVec
    histograms map[string]*prometheus.HistogramVec
}

func NewBusinessMetrics() *BusinessMetrics {
    return &BusinessMetrics{
        counters:   make(map[string]*prometheus.CounterVec),
        gauges:     make(map[string]*prometheus.GaugeVec),
        histograms: make(map[string]*prometheus.HistogramVec),
    }
}

func (m *BusinessMetrics) RegisterCounter(name, help string, labels []string) {
    m.counters[name] = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codeai_business_" + name,
            Help: help,
        },
        labels,
    )
}

func (m *BusinessMetrics) RegisterGauge(name, help string, labels []string) {
    m.gauges[name] = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "codeai_business_" + name,
            Help: help,
        },
        labels,
    )
}

func (m *BusinessMetrics) IncrementCounter(name string, labels map[string]string) {
    if counter, ok := m.counters[name]; ok {
        counter.With(prometheus.Labels(labels)).Inc()
    }
}

func (m *BusinessMetrics) SetGauge(name string, value float64, labels map[string]string) {
    if gauge, ok := m.gauges[name]; ok {
        gauge.With(prometheus.Labels(labels)).Set(value)
    }
}
```

## Acceptance Criteria
- [ ] HTTP request metrics (count, duration, size)
- [ ] Database query metrics
- [ ] Workflow execution metrics
- [ ] Job execution metrics
- [ ] Cache hit/miss metrics
- [ ] Integration call metrics
- [ ] Circuit breaker state metrics
- [ ] /metrics endpoint for Prometheus scraping

## Testing Strategy
- Unit tests for metric recording
- Integration tests with Prometheus client
- Metric label verification

## Files to Create
- `internal/metrics/metrics.go`
- `internal/metrics/http.go`
- `internal/metrics/database.go`
- `internal/metrics/business.go`
- `internal/metrics/metrics_test.go`

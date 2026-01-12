# Observability and Operations Guide

This guide covers the observability stack for CodeAI, including logging, metrics, health checks, tracing, and graceful shutdown procedures.

## Table of Contents

- [Logging](#logging)
- [Metrics](#metrics)
- [Health Checks](#health-checks)
- [Tracing](#tracing)
- [Graceful Shutdown](#graceful-shutdown)
- [Troubleshooting](#troubleshooting)

---

## Logging

CodeAI uses Go's `log/slog` package for structured logging with context propagation and sensitive data redaction.

### Log Levels

| Level | When to Use |
|-------|-------------|
| `debug` | Verbose diagnostic information for development/troubleshooting |
| `info` | Normal operational events (startup, requests, significant state changes) |
| `warn` | Potentially problematic situations that don't affect functionality |
| `error` | Error conditions that affect the current operation |

### Configuration

Logging is configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Minimum log level (debug, info, warn, error) |
| `LOG_FORMAT` | `json` | Output format (json, text) |
| `LOG_OUTPUT` | `stdout` | Output destination (stdout, stderr, or file path) |
| `LOG_ADD_SOURCE` | `false` | Include source file and line number |

```go
// Programmatic configuration
cfg := logging.Config{
    Level:              "info",
    Format:             "json",
    Output:             "stdout",
    AddSource:          false,
    SampleRate:         1.0,  // 0.0-1.0, 1.0 = log all
    SlowQueryThreshold: 100 * time.Millisecond,
}

// Or from environment
cfg := logging.ConfigFromEnv()
```

### Structured Logging Format

All logs are output in JSON format by default:

```json
{
  "time": "2024-01-15T10:30:00.000Z",
  "level": "INFO",
  "msg": "http request",
  "component": "http",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "trace_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/api/workflows",
  "status": 200,
  "duration": "45.2ms"
}
```

### Context Propagation

Request tracing uses the `TraceContext` struct with automatic context propagation:

```go
// Create trace context
tc := logging.NewTraceContext()

// Add to context
ctx := tc.ToContext(ctx)

// Extract from context anywhere in the call chain
tc := logging.FromContext(ctx)

// Get individual values
requestID := logging.GetRequestID(ctx)
traceID := logging.GetTraceID(ctx)
userID := logging.GetUserID(ctx)
```

### HTTP Middleware

The logging middleware automatically:
- Generates/extracts request IDs from `X-Request-ID` header
- Propagates trace IDs via `X-Trace-ID` header
- Logs request details (method, path, status, duration)
- Sets response headers with request/trace IDs

```go
import "github.com/bargom/codeai/pkg/logging"

// Create middleware
mw := logging.NewHTTPMiddleware(logger)

// Configure verbosity
mw = mw.WithVerbosity(logging.VerbosityStandard)

// Apply to router
router.Use(mw.Handler)

// Or use convenience function
router.Use(logging.RequestLogger(logger))
```

**Verbosity Levels:**
- `VerbosityMinimal`: method, path, status, duration only
- `VerbosityStandard`: + remote_addr, user_agent, response_bytes, query
- `VerbosityVerbose`: + request headers (with sensitive data redacted)

### Context-Aware Logger

Create a logger with context values pre-populated:

```go
// Get logger with trace context from HTTP request context
logger := logging.LoggerFromContext(ctx, baseLogger)
logger.Info("processing order", "order_id", orderID)
// Output includes request_id, trace_id, user_id automatically
```

### Sensitive Data Redaction

The logging package automatically redacts sensitive data:

**Sensitive Fields (case-insensitive):**
- `password`, `passwd`, `secret`, `token`
- `api_key`, `apikey`, `authorization`, `auth`
- `credential`, `credentials`
- `credit_card`, `card_number`, `cvv`, `ssn`
- `private_key`, `access_token`, `refresh_token`
- `session_id`, `bearer`

**Automatic Pattern Redaction:**
- JWT tokens (`eyJ...`)
- Bearer tokens
- Email addresses
- Credit card numbers
- AWS access keys
- Password/secret assignments

```go
// Using RedactingHandler for automatic log redaction
handler := logging.NewRedactingHandler(baseHandler, nil)
logger := slog.New(handler)

// Manually redact sensitive data
safe := logging.RedactSensitive(map[string]any{
    "user": "john",
    "password": "secret123",  // becomes [REDACTED]
})

// Add custom sensitive fields
redactor := logging.NewRedactor()
redactor.AddSensitiveField("custom_secret")
redactor.AddSensitivePattern(`MY_TOKEN_[A-Z0-9]+`)
```

### Log Aggregation Setup

**Fluentd Configuration:**

```yaml
<source>
  @type tail
  path /var/log/codeai/*.log
  pos_file /var/log/fluentd/codeai.pos
  tag codeai.*
  <parse>
    @type json
    time_key time
    time_format %Y-%m-%dT%H:%M:%S.%L%z
  </parse>
</source>

<filter codeai.**>
  @type record_transformer
  <record>
    hostname "#{Socket.gethostname}"
    service codeai
  </record>
</filter>

<match codeai.**>
  @type elasticsearch
  host elasticsearch
  port 9200
  index_name codeai-logs
  type_name _doc
</match>
```

**Vector Configuration:**

```toml
[sources.cai_logs]
type = "file"
include = ["/var/log/codeai/*.log"]

[transforms.parse_json]
type = "remap"
inputs = ["codeai_logs"]
source = '''
. = parse_json!(.message)
'''

[sinks.elasticsearch]
type = "elasticsearch"
inputs = ["parse_json"]
endpoints = ["http://elasticsearch:9200"]
index = "codeai-logs-%Y.%m.%d"
```

---

## Metrics

CodeAI uses Prometheus for metrics collection with custom metrics for HTTP, database, workflow, and integration operations.

### Configuration

```go
import "github.com/bargom/codeai/pkg/metrics"

cfg := metrics.Config{
    Namespace:            "codeai",
    EnableProcessMetrics: true,
    EnableRuntimeMetrics: true,
}.WithVersion("1.0.0").WithEnvironment("production")

registry := metrics.NewRegistry(cfg)
```

### Available Prometheus Metrics

#### HTTP Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `codeai_http_requests_total` | Counter | method, path, status_code | Total HTTP requests processed |
| `codeai_http_request_duration_seconds` | Histogram | method, path | HTTP request duration |
| `codeai_http_request_size_bytes` | Histogram | method, path | HTTP request body size |
| `codeai_http_response_size_bytes` | Histogram | method, path | HTTP response body size |
| `codeai_http_active_requests` | Gauge | method, path | Currently active HTTP requests |

#### Database Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `codeai_db_queries_total` | Counter | operation, table, status | Total database queries executed |
| `codeai_db_query_duration_seconds` | Histogram | operation, table | Database query duration |
| `codeai_db_connections_active` | Gauge | - | Active database connections |
| `codeai_db_connections_idle` | Gauge | - | Idle database connections |
| `codeai_db_connections_max` | Gauge | - | Maximum database connections |
| `codeai_db_query_errors_total` | Counter | operation, table, error_type | Database query errors |

#### Workflow Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `codeai_workflow_executions_total` | Counter | workflow_name, status | Total workflow executions |
| `codeai_workflow_execution_duration_seconds` | Histogram | workflow_name | Workflow execution duration |
| `codeai_workflow_active_count` | Gauge | workflow_name | Active workflow executions |
| `codeai_workflow_step_duration_seconds` | Histogram | workflow_name, step_name | Individual step duration |

#### Integration Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `codeai_integration_calls_total` | Counter | service_name, endpoint, status_code | External API calls |
| `codeai_integration_call_duration_seconds` | Histogram | service_name, endpoint | External API call duration |
| `codeai_integration_circuit_breaker_state` | Gauge | service_name, state | Circuit breaker state (0=closed, 1=half-open, 2=open) |
| `codeai_integration_retries_total` | Counter | service_name, endpoint | Retry attempts |
| `codeai_integration_errors_total` | Counter | service_name, endpoint, error_type | Integration errors |

#### Process/Runtime Metrics (when enabled)

- `process_cpu_seconds_total` - Total CPU time
- `process_resident_memory_bytes` - Resident memory size
- `process_open_fds` - Number of open file descriptors
- `go_goroutines` - Number of goroutines
- `go_gc_duration_seconds` - GC pause duration
- `go_memstats_*` - Go memory statistics

### Exposing Metrics

```go
// Register metrics endpoint
registry.RegisterMetricsRoute(router)  // Exposes /metrics

// Or with custom path
router.Handle("/metrics", registry.Handler())

// With authentication
router.Handle("/metrics", registry.HandlerWithAuth(func(r *http.Request) bool {
    token := r.Header.Get("Authorization")
    return validateToken(token)
}))
```

### Custom Metric Creation

```go
// Access the underlying Prometheus registry for custom metrics
promRegistry := metrics.Global().PrometheusRegistry()

// Create custom counter
myCounter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: "codeai",
        Subsystem: "custom",
        Name:      "events_total",
        Help:      "Total custom events",
    },
    []string{"event_type"},
)
promRegistry.MustRegister(myCounter)

// Use the counter
myCounter.WithLabelValues("user_signup").Inc()
```

### HTTP Metrics Middleware

```go
import "github.com/bargom/codeai/pkg/metrics"

// Use middleware to automatically collect HTTP metrics
router.Use(metrics.HTTPMiddleware(registry, metrics.MiddlewareConfig{
    SkipPaths: []string{"/health", "/metrics"},
}))
```

The middleware automatically:
- Normalizes paths (replaces UUIDs, numeric IDs with `{id}`)
- Tracks active requests
- Records request/response sizes
- Measures request duration

### Grafana Dashboard Setup

**Example Prometheus queries for dashboards:**

```promql
# Request Rate (requests/second)
rate(codeai_http_requests_total[5m])

# Request Latency P99
histogram_quantile(0.99, rate(codeai_http_request_duration_seconds_bucket[5m]))

# Error Rate
sum(rate(codeai_http_requests_total{status_code=~"5.."}[5m]))
/ sum(rate(codeai_http_requests_total[5m])) * 100

# Active Connections
sum(codeai_http_active_requests)

# Database Query Latency P95
histogram_quantile(0.95, rate(codeai_db_query_duration_seconds_bucket[5m]))

# Circuit Breaker State
codeai_integration_circuit_breaker_state{state="open"} == 2

# Active Workflows
sum(codeai_workflow_active_count)
```

### Alerting Recommendations

```yaml
# Example Prometheus alerting rules
groups:
  - name: codeai
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate(codeai_http_requests_total{status_code=~"5.."}[5m]))
          / sum(rate(codeai_http_requests_total[5m])) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value | humanizePercentage }}"

      - alert: HighLatency
        expr: |
          histogram_quantile(0.95, rate(codeai_http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High latency detected"
          description: "P95 latency is {{ $value | humanizeDuration }}"

      - alert: DatabaseConnectionPoolExhausted
        expr: |
          codeai_db_connections_active / codeai_db_connections_max > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Database connection pool near exhaustion"

      - alert: CircuitBreakerOpen
        expr: codeai_integration_circuit_breaker_state{state="open"} == 2
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Circuit breaker open for {{ $labels.service_name }}"

      - alert: HighWorkflowFailureRate
        expr: |
          sum(rate(codeai_workflow_executions_total{status="failed"}[5m]))
          / sum(rate(codeai_workflow_executions_total[5m])) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High workflow failure rate"
```

---

## Health Checks

CodeAI provides comprehensive health check endpoints for Kubernetes probes and monitoring.

### Endpoints

| Endpoint | Purpose | Kubernetes Probe |
|----------|---------|------------------|
| `GET /health` | Full health status with all checks | - |
| `GET /health/live` | Liveness probe | `livenessProbe` |
| `GET /health/ready` | Readiness probe | `readinessProbe` |

### Response Format

**Full Health Check (`/health`):**

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00.000Z",
  "version": "1.0.0",
  "uptime": "24h30m15s",
  "checks": {
    "database": {
      "status": "healthy",
      "duration": "2.5ms",
      "details": {
        "max_connections": 100,
        "open_connections": 15,
        "in_use": 5,
        "idle": 10
      }
    },
    "cache": {
      "status": "healthy",
      "duration": "1.2ms"
    }
  }
}
```

**Liveness Check (`/health/live`):**

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00.000Z",
  "version": "1.0.0",
  "uptime": "24h30m15s"
}
```

### HTTP Status Codes

| Status | HTTP Code | Description |
|--------|-----------|-------------|
| `healthy` | 200 | All checks passed |
| `degraded` | 200 | Non-critical checks failed |
| `unhealthy` | 503 | Critical checks failed |

### Health Check Severity

| Severity | Impact |
|----------|--------|
| `critical` | Affects `/health/ready` - returns 503 if failed |
| `warning` | Logged but doesn't affect readiness |

### Registering Health Checks

```go
import (
    "github.com/bargom/codeai/internal/health"
    "github.com/bargom/codeai/internal/health/checks"
)

// Create registry
registry := health.NewRegistry("1.0.0")

// Register database check (critical)
dbChecker := checks.NewDatabaseChecker(db,
    checks.WithDatabaseTimeout(2*time.Second),
    checks.WithDatabaseSeverity(health.SeverityCritical),
)
registry.Register(dbChecker)

// Register custom check
registry.Register(&CustomChecker{})

// Create handler and register routes
handler := health.NewHandler(registry)
handler.RegisterRoutes(mux)
```

### Built-in Health Checks

**Database Check:**
```go
checker := checks.NewDatabaseChecker(db,
    checks.WithDatabaseTimeout(2*time.Second),
    checks.WithDatabaseSeverity(health.SeverityCritical),
)
```

**Custom Check:**
```go
type CustomChecker struct{}

func (c *CustomChecker) Name() string { return "custom" }

func (c *CustomChecker) Severity() health.Severity {
    return health.SeverityWarning
}

func (c *CustomChecker) Check(ctx context.Context) health.CheckResult {
    // Perform check
    if err := checkSomething(ctx); err != nil {
        return health.CheckResult{
            Status:  health.StatusUnhealthy,
            Message: err.Error(),
        }
    }
    return health.CheckResult{
        Status: health.StatusHealthy,
        Details: map[string]any{
            "last_check": time.Now(),
        },
    }
}
```

### Kubernetes Configuration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: codeai
spec:
  template:
    spec:
      containers:
        - name: codeai
          livenessProbe:
            httpGet:
              path: /health/live
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /health/ready
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 3
```

### Readiness vs Liveness

| Probe | Purpose | Failure Behavior |
|-------|---------|------------------|
| **Liveness** | Is the process running? | Kubernetes restarts the container |
| **Readiness** | Can it handle traffic? | Kubernetes removes from load balancer |

**Best Practices:**
- Liveness: Should almost always return healthy (only fail on deadlock)
- Readiness: Check critical dependencies (database, cache)
- Don't include external services in liveness checks
- Use appropriate timeouts (readiness can be slower)

---

## Tracing

CodeAI includes infrastructure for distributed tracing with manual trace context propagation. OpenTelemetry integration is available.

### Trace Context

The `TraceContext` struct carries tracing information through the request lifecycle:

```go
type TraceContext struct {
    RequestID string  // Unique request identifier
    TraceID   string  // Distributed trace ID (shared across services)
    SpanID    string  // Current span identifier
    UserID    string  // Associated user (if authenticated)
}
```

### Context Propagation

```go
// Create new trace context
tc := logging.NewTraceContext()

// Or inherit from parent trace
tc := logging.NewTraceContextWithParent(parentTraceID)

// Add to context
ctx := tc.ToContext(ctx)

// Pass to downstream services via headers
req.Header.Set("X-Request-ID", tc.RequestID)
req.Header.Set("X-Trace-ID", tc.TraceID)

// Extract on receiving side
requestID := req.Header.Get("X-Request-ID")
traceID := req.Header.Get("X-Trace-ID")
```

### HTTP Headers

| Header | Description |
|--------|-------------|
| `X-Request-ID` | Unique identifier for this request |
| `X-Trace-ID` | Distributed trace ID (propagated across services) |

The HTTP middleware automatically:
- Generates request IDs if not provided
- Propagates trace IDs from incoming requests
- Adds IDs to response headers

### Creating Child Spans

```go
// Create a child span for a sub-operation
parentCtx := r.Context()
childTC := logging.FromContext(parentCtx)
childTC.SpanID = logging.GenerateSpanID()

childCtx := childTC.ToContext(parentCtx)
logger := logging.LoggerFromContext(childCtx, baseLogger)

// Log with span context
logger.Info("processing sub-operation")
```

### OpenTelemetry Integration

CodeAI includes OpenTelemetry dependencies for full distributed tracing:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Wrap HTTP handlers with OpenTelemetry
handler := otelhttp.NewHandler(yourHandler, "operation-name")

// Create spans manually
ctx, span := otel.Tracer("codeai").Start(ctx, "operation-name")
defer span.End()

// Add attributes
span.SetAttributes(
    attribute.String("user.id", userID),
    attribute.Int("items.count", len(items)),
)
```

---

## Graceful Shutdown

CodeAI implements comprehensive graceful shutdown with connection draining, priority-based hook execution, and timeout handling.

### Configuration

```go
import "github.com/bargom/codeai/internal/shutdown"

cfg := shutdown.Config{
    OverallTimeout:    30 * time.Second,  // Max time for all hooks
    PerHookTimeout:    10 * time.Second,  // Max time per hook
    DrainTimeout:      10 * time.Second,  // Time to drain connections
    SlowHookThreshold: 5 * time.Second,   // Log warning if hook exceeds this
}

manager := shutdown.NewManager(cfg, logger)
```

### Signal Handling

The shutdown manager listens for OS signals:

| Signal | Behavior |
|--------|----------|
| `SIGTERM` | Initiates graceful shutdown |
| `SIGINT` | Initiates graceful shutdown (Ctrl+C) |
| `SIGQUIT` | Initiates graceful shutdown |

```go
// Start listening for signals
done := manager.ListenForSignals()

// Wait for shutdown to complete
<-done
```

### Registering Shutdown Hooks

Hooks are executed in priority order (higher priority first):

```go
// Register hooks with priority
manager.Register("http-server", 100, func(ctx context.Context) error {
    return server.Shutdown(ctx)
})

manager.Register("database", 50, func(ctx context.Context) error {
    return db.Close()
})

manager.Register("cache", 50, func(ctx context.Context) error {
    return cache.Close()
})

manager.Register("metrics-flush", 10, func(ctx context.Context) error {
    return flushMetrics()
})
```

**Recommended Priority Order:**

| Priority | Component | Description |
|----------|-----------|-------------|
| 100 | HTTP Server | Stop accepting new requests |
| 90 | Workers | Stop background workers |
| 50 | Database | Close database connections |
| 50 | Cache | Close cache connections |
| 10 | Metrics | Flush pending metrics |

### Connection Draining

The `Drainer` tracks in-flight requests and prevents new requests during shutdown:

```go
import "github.com/bargom/codeai/internal/shutdown"

drainer := shutdown.NewDrainer()

// Use as middleware
router.Use(shutdown.DrainMiddleware(drainer))

// During shutdown
drainer.StartDrain()  // New requests get 503

// Wait for in-flight requests
if err := drainer.WaitWithTimeout(10 * time.Second); err != nil {
    log.Warn("some requests did not complete", "remaining", drainer.Count())
}
```

### HTTP Drainer Wrapper

```go
// Wrap your handler
httpDrainer := shutdown.NewHTTPDrainer(handler)

// Use as main handler
server := &http.Server{Handler: httpDrainer}

// During shutdown
httpDrainer.StartDrain()
httpDrainer.Wait(ctx)
```

### Shutdown Ready Middleware

Automatically return 503 on `/health/ready` during shutdown:

```go
router.Use(shutdown.ShutdownReadyMiddleware(manager))
```

### Complete Shutdown Example

```go
func main() {
    logger := slog.Default()

    // Create shutdown manager
    shutdownMgr := shutdown.NewManager(shutdown.DefaultConfig(), logger)

    // Create drainer
    drainer := shutdown.NewDrainer()

    // Setup HTTP server
    server := &http.Server{
        Addr:    ":8080",
        Handler: setupRouter(drainer),
    }

    // Register shutdown hooks
    shutdownMgr.Register("drain-connections", 110, func(ctx context.Context) error {
        drainer.StartDrain()
        return drainer.Wait(ctx)
    })

    shutdownMgr.Register("http-server", 100, func(ctx context.Context) error {
        server.SetKeepAlivesEnabled(false)
        return server.Shutdown(ctx)
    })

    shutdownMgr.Register("database", 50, func(ctx context.Context) error {
        return db.Close()
    })

    // Start signal listener
    done := shutdownMgr.ListenForSignals()

    // Start server
    go func() {
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            logger.Error("server error", "error", err)
        }
    }()

    logger.Info("server started", "addr", ":8080")

    // Wait for shutdown
    <-done

    // Check for errors
    if errs := shutdownMgr.Errors(); len(errs) > 0 {
        logger.Error("shutdown completed with errors", "errors", errs)
        os.Exit(1)
    }

    logger.Info("shutdown complete")
}
```

### Shutdown States

| State | Description |
|-------|-------------|
| `StateRunning` | Normal operation |
| `StateShuttingDown` | Shutdown in progress |
| `StateShutdown` | Shutdown complete |

```go
// Check state
if manager.IsShuttingDown() {
    // Reject new work
}

// Wait for shutdown
manager.Wait()
```

### Cleanup Procedures

During shutdown, the following cleanup happens in order:

1. **Stop accepting new requests** (drainer.StartDrain())
2. **Wait for in-flight requests** (drainer.Wait())
3. **Shutdown HTTP server** (server.Shutdown())
4. **Close worker pools** (stop background jobs)
5. **Close database connections** (db.Close())
6. **Close cache connections** (cache.Close())
7. **Flush metrics** (final metric push)

---

## Troubleshooting

### Common Issues

**High Memory Usage:**
```promql
# Check Go heap usage
go_memstats_heap_inuse_bytes

# Check goroutine count
go_goroutines
```

**Connection Pool Exhaustion:**
```promql
# Database connections
codeai_db_connections_active / codeai_db_connections_max
```

**Slow Requests:**
```promql
# Find slow endpoints
histogram_quantile(0.99,
  rate(codeai_http_request_duration_seconds_bucket[5m])
) by (path)
```

**Integration Failures:**
```promql
# Check circuit breaker states
codeai_integration_circuit_breaker_state{state="open"} == 2

# Retry rate
rate(codeai_integration_retries_total[5m])
```

### Log Analysis

**Find errors in the last hour:**
```bash
jq 'select(.level == "ERROR")' /var/log/codeai/app.log | tail -100
```

**Trace a specific request:**
```bash
jq 'select(.request_id == "550e8400-e29b-41d4-a716-446655440000")' /var/log/codeai/app.log
```

**Find slow queries:**
```bash
jq 'select(.duration > "100ms")' /var/log/codeai/app.log
```

### Health Check Debugging

```bash
# Full health check
curl -s http://localhost:8080/health | jq

# Check readiness
curl -w "%{http_code}" http://localhost:8080/health/ready

# Check liveness
curl -w "%{http_code}" http://localhost:8080/health/live
```

### Metrics Endpoint Debugging

```bash
# Get all metrics
curl http://localhost:8080/metrics

# Filter specific metrics
curl -s http://localhost:8080/metrics | grep codeai_http_requests_total
```

### Shutdown Debugging

If shutdown hangs, check for:
1. Long-running requests (check `codeai_http_active_requests`)
2. Database queries not respecting context cancellation
3. Goroutines not exiting (check `go_goroutines`)

```go
// Enable debug logging during shutdown
cfg := shutdown.Config{
    SlowHookThreshold: 2 * time.Second,  // Lower threshold for warnings
}
```

### Performance Profiling

```go
import _ "net/http/pprof"

// Expose pprof endpoints
mux.Handle("/debug/pprof/", http.DefaultServeMux)
```

```bash
# CPU profile
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

# Memory profile
go tool pprof http://localhost:8080/debug/pprof/heap

# Goroutine dump
curl http://localhost:8080/debug/pprof/goroutine?debug=2
```

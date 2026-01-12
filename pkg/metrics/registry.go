package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Registry manages all Prometheus metrics for CodeAI.
type Registry struct {
	config   Config
	registry *prometheus.Registry

	// HTTP metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec
	httpActiveRequests  *prometheus.GaugeVec

	// Database metrics
	dbQueriesTotal       *prometheus.CounterVec
	dbQueryDuration      *prometheus.HistogramVec
	dbConnectionsActive  prometheus.Gauge
	dbConnectionsIdle    prometheus.Gauge
	dbConnectionsMax     prometheus.Gauge
	dbQueryErrors        *prometheus.CounterVec

	// Workflow metrics
	workflowExecutionsTotal  *prometheus.CounterVec
	workflowExecutionDuration *prometheus.HistogramVec
	workflowActiveCount      *prometheus.GaugeVec
	workflowStepDuration     *prometheus.HistogramVec

	// Integration metrics
	integrationCallsTotal    *prometheus.CounterVec
	integrationCallDuration  *prometheus.HistogramVec
	integrationCircuitState  *prometheus.GaugeVec
	integrationRetryCount    *prometheus.CounterVec
	integrationErrors        *prometheus.CounterVec

	mu sync.RWMutex
}

// Global registry instance
var (
	globalRegistry *Registry
	once           sync.Once
)

// NewRegistry creates a new metrics registry with the given configuration.
func NewRegistry(config Config) *Registry {
	reg := prometheus.NewRegistry()

	r := &Registry{
		config:   config,
		registry: reg,
	}

	r.registerHTTPMetrics()
	r.registerDatabaseMetrics()
	r.registerWorkflowMetrics()
	r.registerIntegrationMetrics()

	// Register process and runtime metrics if enabled
	if config.EnableProcessMetrics {
		reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}
	if config.EnableRuntimeMetrics {
		reg.MustRegister(collectors.NewGoCollector())
	}

	return r
}

// Global returns the global registry instance, initializing it with default config if needed.
func Global() *Registry {
	once.Do(func() {
		globalRegistry = NewRegistry(DefaultConfig())
	})
	return globalRegistry
}

// SetGlobal sets the global registry instance.
func SetGlobal(r *Registry) {
	globalRegistry = r
}

// PrometheusRegistry returns the underlying Prometheus registry.
func (r *Registry) PrometheusRegistry() *prometheus.Registry {
	return r.registry
}

// Config returns the registry configuration.
func (r *Registry) Config() Config {
	return r.config
}

func (r *Registry) registerHTTPMetrics() {
	ns := r.config.Namespace

	r.httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests processed",
		},
		[]string{"method", "path", "status_code"},
	)

	r.httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   r.config.HistogramBuckets.HTTPDuration,
		},
		[]string{"method", "path"},
	)

	r.httpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: "http",
			Name:      "request_size_bytes",
			Help:      "HTTP request size in bytes",
			Buckets:   r.config.HistogramBuckets.HTTPSize,
		},
		[]string{"method", "path"},
	)

	r.httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   r.config.HistogramBuckets.HTTPSize,
		},
		[]string{"method", "path"},
	)

	r.httpActiveRequests = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: "http",
			Name:      "active_requests",
			Help:      "Number of currently active HTTP requests",
		},
		[]string{"method", "path"},
	)

	r.registry.MustRegister(
		r.httpRequestsTotal,
		r.httpRequestDuration,
		r.httpRequestSize,
		r.httpResponseSize,
		r.httpActiveRequests,
	)
}

func (r *Registry) registerDatabaseMetrics() {
	ns := r.config.Namespace

	r.dbQueriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: "db",
			Name:      "queries_total",
			Help:      "Total number of database queries executed",
		},
		[]string{"operation", "table", "status"},
	)

	r.dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: "db",
			Name:      "query_duration_seconds",
			Help:      "Database query duration in seconds",
			Buckets:   r.config.HistogramBuckets.DBDuration,
		},
		[]string{"operation", "table"},
	)

	r.dbConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: "db",
			Name:      "connections_active",
			Help:      "Number of active database connections",
		},
	)

	r.dbConnectionsIdle = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: "db",
			Name:      "connections_idle",
			Help:      "Number of idle database connections",
		},
	)

	r.dbConnectionsMax = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: "db",
			Name:      "connections_max",
			Help:      "Maximum number of database connections",
		},
	)

	r.dbQueryErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: "db",
			Name:      "query_errors_total",
			Help:      "Total number of database query errors",
		},
		[]string{"operation", "table", "error_type"},
	)

	r.registry.MustRegister(
		r.dbQueriesTotal,
		r.dbQueryDuration,
		r.dbConnectionsActive,
		r.dbConnectionsIdle,
		r.dbConnectionsMax,
		r.dbQueryErrors,
	)
}

func (r *Registry) registerWorkflowMetrics() {
	ns := r.config.Namespace

	r.workflowExecutionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: "workflow",
			Name:      "executions_total",
			Help:      "Total number of workflow executions",
		},
		[]string{"workflow_name", "status"},
	)

	r.workflowExecutionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: "workflow",
			Name:      "execution_duration_seconds",
			Help:      "Workflow execution duration in seconds",
			Buckets:   r.config.HistogramBuckets.WorkflowDuration,
		},
		[]string{"workflow_name"},
	)

	r.workflowActiveCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: "workflow",
			Name:      "active_count",
			Help:      "Number of currently active workflow executions",
		},
		[]string{"workflow_name"},
	)

	r.workflowStepDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: "workflow",
			Name:      "step_duration_seconds",
			Help:      "Workflow step duration in seconds",
			Buckets:   []float64{.01, .05, .1, .5, 1, 5, 10, 30, 60},
		},
		[]string{"workflow_name", "step_name"},
	)

	r.registry.MustRegister(
		r.workflowExecutionsTotal,
		r.workflowExecutionDuration,
		r.workflowActiveCount,
		r.workflowStepDuration,
	)
}

func (r *Registry) registerIntegrationMetrics() {
	ns := r.config.Namespace

	r.integrationCallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: "integration",
			Name:      "calls_total",
			Help:      "Total number of external API calls",
		},
		[]string{"service_name", "endpoint", "status_code"},
	)

	r.integrationCallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: "integration",
			Name:      "call_duration_seconds",
			Help:      "External API call duration in seconds",
			Buckets:   r.config.HistogramBuckets.IntegrationDuration,
		},
		[]string{"service_name", "endpoint"},
	)

	r.integrationCircuitState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: "integration",
			Name:      "circuit_breaker_state",
			Help:      "Circuit breaker state (0=closed, 1=half-open, 2=open)",
		},
		[]string{"service_name", "state"},
	)

	r.integrationRetryCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: "integration",
			Name:      "retries_total",
			Help:      "Total number of retry attempts for external API calls",
		},
		[]string{"service_name", "endpoint"},
	)

	r.integrationErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: "integration",
			Name:      "errors_total",
			Help:      "Total number of integration errors",
		},
		[]string{"service_name", "endpoint", "error_type"},
	)

	r.registry.MustRegister(
		r.integrationCallsTotal,
		r.integrationCallDuration,
		r.integrationCircuitState,
		r.integrationRetryCount,
		r.integrationErrors,
	)
}

// Package metrics provides Prometheus metrics collection for the CodeAI application.
package metrics

// Config holds configuration for the metrics module.
type Config struct {
	// Namespace is the prefix for all metrics (default: "codeai")
	Namespace string

	// Subsystem groups related metrics (e.g., "http", "database")
	Subsystem string

	// DefaultLabels are applied to all metrics
	DefaultLabels map[string]string

	// EnableProcessMetrics enables Go process metrics (CPU, memory, goroutines)
	EnableProcessMetrics bool

	// EnableRuntimeMetrics enables Go runtime metrics
	EnableRuntimeMetrics bool

	// HistogramBuckets allows customizing default histogram buckets
	HistogramBuckets HistogramBucketsConfig
}

// HistogramBucketsConfig holds custom bucket configurations for different metric types.
type HistogramBucketsConfig struct {
	// HTTPDuration buckets for HTTP request duration in seconds
	HTTPDuration []float64

	// HTTPSize buckets for HTTP request/response size in bytes
	HTTPSize []float64

	// DBDuration buckets for database query duration in seconds
	DBDuration []float64

	// WorkflowDuration buckets for workflow execution duration in seconds
	WorkflowDuration []float64

	// IntegrationDuration buckets for external API call duration in seconds
	IntegrationDuration []float64
}

// DefaultConfig returns the default metrics configuration.
func DefaultConfig() Config {
	return Config{
		Namespace: "codeai",
		DefaultLabels: map[string]string{
			"version":     "unknown",
			"environment": "development",
			"instance":    "unknown",
		},
		EnableProcessMetrics: true,
		EnableRuntimeMetrics: true,
		HistogramBuckets:     DefaultHistogramBuckets(),
	}
}

// DefaultHistogramBuckets returns the default histogram bucket configurations.
func DefaultHistogramBuckets() HistogramBucketsConfig {
	return HistogramBucketsConfig{
		HTTPDuration:        []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		HTTPSize:            []float64{100, 1000, 10000, 100000, 1000000, 10000000},
		DBDuration:          []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		WorkflowDuration:    []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
		IntegrationDuration: []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
	}
}

// WithVersion sets the version label.
func (c Config) WithVersion(version string) Config {
	c.DefaultLabels["version"] = version
	return c
}

// WithEnvironment sets the environment label.
func (c Config) WithEnvironment(env string) Config {
	c.DefaultLabels["environment"] = env
	return c
}

// WithInstance sets the instance label.
func (c Config) WithInstance(instance string) Config {
	c.DefaultLabels["instance"] = instance
	return c
}

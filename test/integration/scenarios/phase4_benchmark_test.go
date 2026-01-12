//go:build integration

// Package scenarios provides benchmark tests for Phase 4 components.
package scenarios

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/health"
	"github.com/bargom/codeai/internal/health/checks"
	"github.com/bargom/codeai/internal/shutdown"
	"github.com/bargom/codeai/pkg/integration"
	"github.com/bargom/codeai/pkg/logging"
	"github.com/bargom/codeai/pkg/metrics"
)

// BenchmarkLoggingOverhead measures the overhead of structured logging.
func BenchmarkLoggingOverhead(b *testing.B) {
	logBuffer := &bytes.Buffer{}
	logger := logging.NewWithWriter(logging.Config{
		Level:  "info",
		Format: "json",
	}, logBuffer)

	ctx := context.Background()

	b.Run("simple_log", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Logger.InfoContext(ctx, "test message", "key", "value")
		}
	})

	b.Run("log_with_context", func(b *testing.B) {
		ctx := logging.WithRequestID(ctx, "test-request-id")
		ctx = logging.WithTraceID(ctx, "test-trace-id")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Logger.InfoContext(ctx, "test message", "key", "value")
		}
	})

	b.Run("log_with_redaction", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Logger.InfoContext(ctx, "test message",
				"password", "secret123",
				"api_key", "key123",
				"data", "normal value",
			)
		}
	})
}

// BenchmarkMetricsOverhead measures the overhead of Prometheus metrics recording.
func BenchmarkMetricsOverhead(b *testing.B) {
	registry := metrics.NewRegistry(metrics.DefaultConfig())
	httpMetrics := registry.HTTP()
	dbMetrics := registry.DB()
	intMetrics := registry.Integration()

	b.Run("http_request_recording", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			httpMetrics.RecordRequest("GET", "/api/test", 200, 0.001, 100, 500)
		}
	})

	b.Run("db_query_recording", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dbMetrics.RecordQuery(metrics.OperationSelect, "users", 10*time.Millisecond, nil)
		}
	})

	b.Run("integration_call_recording", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			intMetrics.RecordCall("external-api", "process", 200, 50*time.Millisecond)
		}
	})

	b.Run("circuit_breaker_state", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			intMetrics.SetCircuitBreakerState("test-service", metrics.CircuitBreakerClosed)
		}
	})
}

// BenchmarkHealthCheckOverhead measures the overhead of health checks.
func BenchmarkHealthCheckOverhead(b *testing.B) {
	registry := health.NewRegistry("test-1.0.0")

	// Add a simple custom checker
	registry.Register(checks.NewCustomChecker("api", func(ctx context.Context) health.CheckResult {
		return health.CheckResult{
			Status:  health.StatusHealthy,
			Message: "API is healthy",
		}
	}))

	ctx := context.Background()

	b.Run("liveness_check", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = registry.Liveness(ctx)
		}
	})

	b.Run("readiness_check", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = registry.Readiness(ctx)
		}
	})

	b.Run("full_health_check", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = registry.Health(ctx)
		}
	})
}

// BenchmarkCircuitBreakerOverhead measures the overhead of circuit breaker operations.
func BenchmarkCircuitBreakerOverhead(b *testing.B) {
	cb := integration.NewCircuitBreaker("test-service", integration.CircuitBreakerConfig{
		FailureThreshold: 5,
		Timeout:          60 * time.Second,
		HalfOpenRequests: 3,
	})

	b.Run("allow_closed", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cb.Allow()
		}
	})

	b.Run("record_success", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cb.RecordSuccess()
		}
	})

	b.Run("record_failure_no_trip", func(b *testing.B) {
		// Use a high threshold to avoid opening the circuit
		cbHigh := integration.NewCircuitBreaker("test-high", integration.CircuitBreakerConfig{
			FailureThreshold: 1000000,
			Timeout:          60 * time.Second,
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cbHigh.RecordFailure()
		}
	})
}

// BenchmarkShutdownHookRegistration measures the overhead of shutdown hook registration.
func BenchmarkShutdownHookRegistration(b *testing.B) {
	b.Run("register_hook", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			manager := shutdown.NewManagerWithDefaults()
			manager.Register("test", 50, func(ctx context.Context) error {
				return nil
			})
		}
	})
}

// BenchmarkHTTPMiddlewareOverhead measures the overhead of HTTP middleware stack.
func BenchmarkHTTPMiddlewareOverhead(b *testing.B) {
	logBuffer := &bytes.Buffer{}
	logger := logging.NewWithWriter(logging.Config{
		Level:  "warn", // Reduce log output
		Format: "json",
	}, logBuffer)

	registry := metrics.NewRegistry(metrics.DefaultConfig())

	// Create handler with middleware stack
	httpMiddleware := logging.NewHTTPMiddleware(logger.Logger)
	handler := httpMiddleware.Handler(
		metrics.HTTPMiddleware(registry)(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		),
	)

	// Create a simple handler without middleware for comparison
	simpleHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	b.Run("with_middleware", func(b *testing.B) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})

	b.Run("without_middleware", func(b *testing.B) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			simpleHandler.ServeHTTP(w, req)
		}
	})
}

// BenchmarkMetricsEndpoint measures metrics endpoint response time.
func BenchmarkMetricsEndpoint(b *testing.B) {
	registry := metrics.NewRegistry(metrics.DefaultConfig())

	// Record some metrics
	httpMetrics := registry.HTTP()
	for i := 0; i < 100; i++ {
		httpMetrics.RecordRequest("GET", "/api/test", 200, 0.01, 100, 500)
	}

	handler := registry.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)

	b.Run("metrics_endpoint", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			io.Copy(io.Discard, w.Body)
		}
	})
}

// BenchmarkRetryerOverhead measures the overhead of the retry logic (success case).
func BenchmarkRetryerOverhead(b *testing.B) {
	retryer := integration.NewRetryer(integration.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
	})

	ctx := context.Background()

	b.Run("successful_call", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			retryer.Do(ctx, func(ctx context.Context) error {
				return nil
			})
		}
	})
}

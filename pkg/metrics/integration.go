package metrics

import (
	"strconv"
	"time"
)

// IntegrationMetrics provides methods to record external integration metrics.
type IntegrationMetrics struct {
	registry *Registry
}

// Integration returns the integration metrics interface for the registry.
func (r *Registry) Integration() *IntegrationMetrics {
	return &IntegrationMetrics{registry: r}
}

// CircuitBreakerState represents the state of a circuit breaker.
type CircuitBreakerState string

const (
	CircuitBreakerClosed   CircuitBreakerState = "closed"
	CircuitBreakerHalfOpen CircuitBreakerState = "half-open"
	CircuitBreakerOpen     CircuitBreakerState = "open"
)

// CircuitBreakerStateValue returns the numeric value for a circuit breaker state.
func (s CircuitBreakerState) Value() float64 {
	switch s {
	case CircuitBreakerClosed:
		return 0
	case CircuitBreakerHalfOpen:
		return 1
	case CircuitBreakerOpen:
		return 2
	default:
		return -1
	}
}

// RecordCall records metrics for an external API call.
func (i *IntegrationMetrics) RecordCall(serviceName, endpoint string, statusCode int, duration time.Duration) {
	statusStr := strconv.Itoa(statusCode)

	i.registry.integrationCallsTotal.WithLabelValues(
		serviceName,
		endpoint,
		statusStr,
	).Inc()

	i.registry.integrationCallDuration.WithLabelValues(
		serviceName,
		endpoint,
	).Observe(duration.Seconds())
}

// RecordCallWithStatus is a convenience method that accepts a boolean for success/failure.
func (i *IntegrationMetrics) RecordCallWithStatus(serviceName, endpoint string, success bool, duration time.Duration) {
	statusCode := 200
	if !success {
		statusCode = 500
	}
	i.RecordCall(serviceName, endpoint, statusCode, duration)
}

// RecordError records an integration error.
func (i *IntegrationMetrics) RecordError(serviceName, endpoint, errorType string) {
	i.registry.integrationErrors.WithLabelValues(
		serviceName,
		endpoint,
		errorType,
	).Inc()
}

// RecordRetry records a retry attempt.
func (i *IntegrationMetrics) RecordRetry(serviceName, endpoint string) {
	i.registry.integrationRetryCount.WithLabelValues(serviceName, endpoint).Inc()
}

// SetCircuitBreakerState sets the circuit breaker state for a service.
func (i *IntegrationMetrics) SetCircuitBreakerState(serviceName string, state CircuitBreakerState) {
	// Reset all states to 0 first
	for _, s := range []CircuitBreakerState{CircuitBreakerClosed, CircuitBreakerHalfOpen, CircuitBreakerOpen} {
		val := 0.0
		if s == state {
			val = 1.0
		}
		i.registry.integrationCircuitState.WithLabelValues(serviceName, string(s)).Set(val)
	}
}

// IntegrationCallTimer provides a convenient way to time external API calls.
type IntegrationCallTimer struct {
	metrics     *IntegrationMetrics
	serviceName string
	endpoint    string
	start       time.Time
	retryCount  int
}

// NewCallTimer creates a new integration call timer.
func (i *IntegrationMetrics) NewCallTimer(serviceName, endpoint string) *IntegrationCallTimer {
	return &IntegrationCallTimer{
		metrics:     i,
		serviceName: serviceName,
		endpoint:    endpoint,
		start:       time.Now(),
	}
}

// Retry records a retry attempt and resets the timer.
func (t *IntegrationCallTimer) Retry() {
	t.metrics.RecordRetry(t.serviceName, t.endpoint)
	t.retryCount++
	t.start = time.Now()
}

// Done records the call duration and status.
func (t *IntegrationCallTimer) Done(statusCode int) {
	duration := time.Since(t.start)
	t.metrics.RecordCall(t.serviceName, t.endpoint, statusCode, duration)
}

// Success records a successful call.
func (t *IntegrationCallTimer) Success() {
	t.Done(200)
}

// Error records a failed call with error classification.
func (t *IntegrationCallTimer) Error(errorType string) {
	duration := time.Since(t.start)
	t.metrics.RecordCall(t.serviceName, t.endpoint, 500, duration)
	t.metrics.RecordError(t.serviceName, t.endpoint, errorType)
}

// RetryCount returns the number of retry attempts made.
func (t *IntegrationCallTimer) RetryCount() int {
	return t.retryCount
}

// ClassifyHTTPError classifies an HTTP status code into an error type.
func ClassifyHTTPError(statusCode int) string {
	switch {
	case statusCode >= 400 && statusCode < 500:
		return "client_error"
	case statusCode >= 500 && statusCode < 600:
		return "server_error"
	case statusCode == 0:
		return "connection_error"
	default:
		return "unknown"
	}
}

// ClassifyError classifies an error into a type for metrics.
func ClassifyError(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	switch {
	case contains(errStr, "timeout"):
		return "timeout"
	case contains(errStr, "connection refused"):
		return "connection_refused"
	case contains(errStr, "no such host"):
		return "dns_error"
	case contains(errStr, "tls", "certificate"):
		return "tls_error"
	case contains(errStr, "context canceled"):
		return "cancelled"
	default:
		return "unknown"
	}
}

// contains checks if the error string contains any of the substrings.
func contains(errStr string, substrings ...string) bool {
	for _, sub := range substrings {
		if len(errStr) >= len(sub) {
			for i := 0; i <= len(errStr)-len(sub); i++ {
				match := true
				for j := 0; j < len(sub); j++ {
					if errStr[i+j] != sub[j] && errStr[i+j] != sub[j]-32 && errStr[i+j] != sub[j]+32 {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
	}
	return false
}

// CallsTotal returns the counter for total integration calls (for testing).
func (i *IntegrationMetrics) CallsTotal() interface{} {
	return i.registry.integrationCallsTotal
}

// CircuitState returns the gauge for circuit breaker state (for testing).
func (i *IntegrationMetrics) CircuitState() interface{} {
	return i.registry.integrationCircuitState
}

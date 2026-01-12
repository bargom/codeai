package metrics

import (
	"strconv"
)

// HTTPMetrics provides methods to record HTTP-related metrics.
type HTTPMetrics struct {
	registry *Registry
}

// HTTP returns the HTTP metrics interface for the registry.
func (r *Registry) HTTP() *HTTPMetrics {
	return &HTTPMetrics{registry: r}
}

// RecordRequest records all metrics for an HTTP request.
func (h *HTTPMetrics) RecordRequest(method, path string, statusCode int, duration float64, reqSize, respSize int64) {
	statusStr := strconv.Itoa(statusCode)

	h.registry.httpRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
	h.registry.httpRequestDuration.WithLabelValues(method, path).Observe(duration)

	if reqSize >= 0 {
		h.registry.httpRequestSize.WithLabelValues(method, path).Observe(float64(reqSize))
	}
	if respSize >= 0 {
		h.registry.httpResponseSize.WithLabelValues(method, path).Observe(float64(respSize))
	}
}

// IncActiveRequests increments the active request count.
func (h *HTTPMetrics) IncActiveRequests(method, path string) {
	h.registry.httpActiveRequests.WithLabelValues(method, path).Inc()
}

// DecActiveRequests decrements the active request count.
func (h *HTTPMetrics) DecActiveRequests(method, path string) {
	h.registry.httpActiveRequests.WithLabelValues(method, path).Dec()
}

// RequestsTotal returns the counter for total HTTP requests (for testing).
func (h *HTTPMetrics) RequestsTotal() interface{} {
	return h.registry.httpRequestsTotal
}

// RequestDuration returns the histogram for request duration (for testing).
func (h *HTTPMetrics) RequestDuration() interface{} {
	return h.registry.httpRequestDuration
}

// ActiveRequests returns the gauge for active requests (for testing).
func (h *HTTPMetrics) ActiveRequests() interface{} {
	return h.registry.httpActiveRequests
}

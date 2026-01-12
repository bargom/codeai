package health

import (
	"encoding/json"
	"net/http"
)

// Handler provides HTTP handlers for health check endpoints.
type Handler struct {
	registry *Registry
}

// NewHandler creates a new health check handler.
func NewHandler(registry *Registry) *Handler {
	return &Handler{registry: registry}
}

// HealthHandler handles GET /health endpoint.
// Returns overall health status with all check details.
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	resp := h.registry.Health(r.Context())
	h.writeResponse(w, resp)
}

// LivenessHandler handles GET /health/live endpoint.
// Used for Kubernetes liveness probes.
// Returns 200 unless the process is broken.
func (h *Handler) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	resp := h.registry.Liveness(r.Context())
	h.writeResponse(w, resp)
}

// ReadinessHandler handles GET /health/ready endpoint.
// Used for Kubernetes readiness probes.
// Returns 503 if critical dependencies are unavailable.
func (h *Handler) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	resp := h.registry.Readiness(r.Context())
	h.writeResponse(w, resp)
}

// writeResponse writes a health response as JSON.
func (h *Handler) writeResponse(w http.ResponseWriter, resp Response) {
	w.Header().Set("Content-Type", "application/json")

	status := http.StatusOK
	if resp.Status == StatusUnhealthy {
		status = http.StatusServiceUnavailable
	}

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

// RegisterRoutes registers health check routes on an http.ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.HealthHandler)
	mux.HandleFunc("/health/live", h.LivenessHandler)
	mux.HandleFunc("/health/ready", h.ReadinessHandler)
}

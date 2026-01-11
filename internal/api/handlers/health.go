package handlers

import (
	"net/http"
	"time"

	"github.com/bargom/codeai/internal/api/types"
)

// Health handles GET /health.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	health := &types.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	h.respondJSON(w, http.StatusOK, health)
}

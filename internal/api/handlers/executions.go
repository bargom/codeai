package handlers

import (
	"errors"
	"net/http"

	"github.com/bargom/codeai/internal/api/types"
	"github.com/bargom/codeai/internal/database/repository"
	"github.com/go-chi/chi/v5"
)

// GetExecution handles GET /executions/{id}.
func (h *Handler) GetExecution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	execution, err := h.executions.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "execution not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to get execution")
		return
	}

	h.respondJSON(w, http.StatusOK, types.ExecutionFromModel(execution))
}

// ListExecutions handles GET /executions.
func (h *Handler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	limit, offset := getPaginationParams(r)

	executions, err := h.executions.List(r.Context(), limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to list executions")
		return
	}

	responses := types.ExecutionsFromModels(executions)
	h.respondJSON(w, http.StatusOK, types.NewListResponse(responses, limit, offset))
}

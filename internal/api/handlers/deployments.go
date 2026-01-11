package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/bargom/codeai/internal/api/types"
	"github.com/bargom/codeai/internal/database/models"
	"github.com/bargom/codeai/internal/database/repository"
	"github.com/go-chi/chi/v5"
)

// CreateDeployment handles POST /deployments.
func (h *Handler) CreateDeployment(w http.ResponseWriter, r *http.Request) {
	var req types.CreateDeploymentRequest
	if err := h.decodeAndValidate(r, &req); err != nil {
		h.respondValidationError(w, err)
		return
	}

	deployment := models.NewDeployment(req.Name)
	if req.ConfigID != "" {
		deployment.ConfigID = sql.NullString{String: req.ConfigID, Valid: true}
	}

	if err := h.deployments.Create(r.Context(), deployment); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to create deployment")
		return
	}

	h.respondJSON(w, http.StatusCreated, types.DeploymentFromModel(deployment))
}

// GetDeployment handles GET /deployments/{id}.
func (h *Handler) GetDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	deployment, err := h.deployments.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "deployment not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to get deployment")
		return
	}

	h.respondJSON(w, http.StatusOK, types.DeploymentFromModel(deployment))
}

// ListDeployments handles GET /deployments.
func (h *Handler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	limit, offset := getPaginationParams(r)

	deployments, err := h.deployments.List(r.Context(), limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to list deployments")
		return
	}

	responses := types.DeploymentsFromModels(deployments)
	h.respondJSON(w, http.StatusOK, types.NewListResponse(responses, limit, offset))
}

// UpdateDeployment handles PUT /deployments/{id}.
func (h *Handler) UpdateDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req types.UpdateDeploymentRequest
	if err := h.decodeAndValidate(r, &req); err != nil {
		h.respondValidationError(w, err)
		return
	}

	// Get existing deployment
	deployment, err := h.deployments.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "deployment not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to get deployment")
		return
	}

	// Apply updates
	if req.Name != "" {
		deployment.Name = req.Name
	}
	if req.ConfigID != "" {
		deployment.ConfigID = sql.NullString{String: req.ConfigID, Valid: true}
	}
	if req.Status != "" {
		deployment.Status = req.Status
	}

	if err := h.deployments.Update(r.Context(), deployment); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to update deployment")
		return
	}

	h.respondJSON(w, http.StatusOK, types.DeploymentFromModel(deployment))
}

// DeleteDeployment handles DELETE /deployments/{id}.
func (h *Handler) DeleteDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.deployments.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "deployment not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to delete deployment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ExecuteDeployment handles POST /deployments/{id}/execute.
func (h *Handler) ExecuteDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Check deployment exists
	deployment, err := h.deployments.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "deployment not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to get deployment")
		return
	}

	var req types.ExecuteDeploymentRequest
	if err := h.decodeJSON(r, &req); err != nil && !errors.Is(err, errors.New("request body is required")) {
		// Ignore empty body error for execute
		if err.Error() != "EOF" {
			h.respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	// Create execution record
	execution := models.NewExecution(deployment.ID, "execute")
	if err := h.executions.Create(r.Context(), execution); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to create execution")
		return
	}

	// TODO: Actually execute the deployment asynchronously
	// For now, we just create an execution record and return

	h.respondJSON(w, http.StatusAccepted, types.ExecutionFromModel(execution))
}

// ListDeploymentExecutions handles GET /deployments/{id}/executions.
func (h *Handler) ListDeploymentExecutions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Check deployment exists
	if _, err := h.deployments.GetByID(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "deployment not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to get deployment")
		return
	}

	limit, offset := getPaginationParams(r)

	executions, err := h.executions.ListByDeployment(r.Context(), id, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to list executions")
		return
	}

	responses := types.ExecutionsFromModels(executions)
	h.respondJSON(w, http.StatusOK, types.NewListResponse(responses, limit, offset))
}

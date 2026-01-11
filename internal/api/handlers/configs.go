package handlers

import (
	"errors"
	"net/http"

	"github.com/bargom/codeai/internal/api/types"
	"github.com/bargom/codeai/internal/database/models"
	"github.com/bargom/codeai/internal/database/repository"
	"github.com/bargom/codeai/internal/parser"
	"github.com/bargom/codeai/internal/validator"
	"github.com/go-chi/chi/v5"
)

// CreateConfig handles POST /configs.
func (h *Handler) CreateConfig(w http.ResponseWriter, r *http.Request) {
	var req types.CreateConfigRequest
	if err := h.decodeAndValidate(r, &req); err != nil {
		h.respondValidationError(w, err)
		return
	}

	config := models.NewConfig(req.Name, req.Content)

	if err := h.configs.Create(r.Context(), config); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to create config")
		return
	}

	h.respondJSON(w, http.StatusCreated, types.ConfigFromModel(config))
}

// GetConfig handles GET /configs/{id}.
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	config, err := h.configs.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "config not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to get config")
		return
	}

	h.respondJSON(w, http.StatusOK, types.ConfigFromModel(config))
}

// ListConfigs handles GET /configs.
func (h *Handler) ListConfigs(w http.ResponseWriter, r *http.Request) {
	limit, offset := getPaginationParams(r)

	configs, err := h.configs.List(r.Context(), limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to list configs")
		return
	}

	responses := types.ConfigsFromModels(configs)
	h.respondJSON(w, http.StatusOK, types.NewListResponse(responses, limit, offset))
}

// UpdateConfig handles PUT /configs/{id}.
func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req types.UpdateConfigRequest
	if err := h.decodeAndValidate(r, &req); err != nil {
		h.respondValidationError(w, err)
		return
	}

	// Get existing config
	config, err := h.configs.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "config not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to get config")
		return
	}

	// Apply updates
	if req.Name != "" {
		config.Name = req.Name
	}
	if req.Content != "" {
		config.Content = req.Content
	}

	if err := h.configs.Update(r.Context(), config); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to update config")
		return
	}

	h.respondJSON(w, http.StatusOK, types.ConfigFromModel(config))
}

// DeleteConfig handles DELETE /configs/{id}.
func (h *Handler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.configs.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "config not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to delete config")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ValidateConfig handles POST /configs/{id}/validate.
func (h *Handler) ValidateConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Get existing config
	config, err := h.configs.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "config not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to get config")
		return
	}

	// Check if custom content was provided in the request body
	var req types.ValidateConfigRequest
	content := config.Content
	if err := h.decodeJSON(r, &req); err == nil && req.Content != "" {
		content = req.Content
	}

	// Parse and validate the content
	result := validateDSLContent(content)

	h.respondJSON(w, http.StatusOK, result)
}

// validateDSLContent parses and validates DSL content.
func validateDSLContent(content string) *types.ValidationResult {
	result := &types.ValidationResult{
		Valid:  true,
		Errors: []string{},
	}

	// Parse the content
	astResult, err := parser.Parse(content)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, "parse error: "+err.Error())
		return result
	}

	// Validate the AST
	v := validator.New()
	if err := v.Validate(astResult); err != nil {
		result.Valid = false
		// Type assert to get detailed errors if available
		if validationErrs, ok := err.(*validator.ValidationErrors); ok {
			for _, e := range validationErrs.Errors {
				result.Errors = append(result.Errors, e.Error())
			}
		} else {
			result.Errors = append(result.Errors, err.Error())
		}
	}

	return result
}

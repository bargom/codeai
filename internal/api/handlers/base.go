// Package handlers contains HTTP request handlers for the API.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/bargom/codeai/internal/api/types"
	"github.com/bargom/codeai/internal/database/repository"
	"github.com/go-playground/validator/v10"
)

// Handler provides HTTP handlers for the API.
type Handler struct {
	deployments repository.DeploymentRepo
	configs     repository.ConfigRepo
	executions  repository.ExecutionRepo
	validate    *validator.Validate
}

// NewHandler creates a new Handler with the given repositories.
// Accepts any types that implement the repository interfaces.
func NewHandler(
	deployments repository.DeploymentRepo,
	configs repository.ConfigRepo,
	executions repository.ExecutionRepo,
) *Handler {
	return &Handler{
		deployments: deployments,
		configs:     configs,
		executions:  executions,
		validate:    validator.New(),
	}
}

// NewHandlerFromRepositories creates a new Handler from a Repositories struct.
func NewHandlerFromRepositories(repos *repository.Repositories) *Handler {
	return &Handler{
		deployments: repos.Deployments,
		configs:     repos.Configs,
		executions:  repos.Executions,
		validate:    validator.New(),
	}
}

// respondJSON writes a JSON response with the given status code.
func (h *Handler) respondJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			// Log error but can't change response at this point
			return
		}
	}
}

// respondError writes a JSON error response with the given status code.
func (h *Handler) respondError(w http.ResponseWriter, code int, message string) {
	h.respondJSON(w, code, types.ErrorResponse{Error: message})
}

// respondValidationError writes a JSON validation error response.
func (h *Handler) respondValidationError(w http.ResponseWriter, err error) {
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		details := make(map[string]string)
		for _, e := range validationErrs {
			details[e.Field()] = formatValidationError(e)
		}
		h.respondJSON(w, http.StatusBadRequest, types.ErrorResponse{
			Error:   "validation failed",
			Details: details,
		})
		return
	}
	h.respondError(w, http.StatusBadRequest, "invalid input")
}

// formatValidationError formats a validation error into a human-readable message.
func formatValidationError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "is required"
	case "min":
		return "must be at least " + e.Param() + " characters"
	case "max":
		return "must be at most " + e.Param() + " characters"
	case "uuid":
		return "must be a valid UUID"
	case "oneof":
		return "must be one of: " + e.Param()
	default:
		return "is invalid"
	}
}

// decodeJSON decodes a JSON request body into the given value.
func (h *Handler) decodeJSON(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	return json.NewDecoder(r.Body).Decode(v)
}

// validateRequest validates the given request struct.
func (h *Handler) validateRequest(v interface{}) error {
	return h.validate.Struct(v)
}

// decodeAndValidate decodes and validates a JSON request.
func (h *Handler) decodeAndValidate(r *http.Request, v interface{}) error {
	if err := h.decodeJSON(r, v); err != nil {
		return err
	}
	return h.validateRequest(v)
}

// getPaginationParams extracts pagination parameters from the request.
func getPaginationParams(r *http.Request) (limit, offset int) {
	limit = types.DefaultLimit
	offset = 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			if parsed > types.DefaultMaxLimit {
				parsed = types.DefaultMaxLimit
			}
			limit = parsed
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}

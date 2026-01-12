package workflow

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/bargom/codeai/internal/workflow/definitions"
	"github.com/bargom/codeai/internal/workflow/engine"
	"github.com/bargom/codeai/internal/workflow/repository"
)

const (
	defaultLimit    = 20
	defaultMaxLimit = 100
)

// Handler provides HTTP handlers for workflow operations.
type Handler struct {
	engine   *engine.Engine
	repo     repository.WorkflowRepository
	validate *validator.Validate
}

// NewHandler creates a new workflow Handler.
func NewHandler(eng *engine.Engine, repo repository.WorkflowRepository) *Handler {
	return &Handler{
		engine:   eng,
		repo:     repo,
		validate: validator.New(),
	}
}

// RegisterRoutes registers the workflow routes with the given router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/workflows", func(r chi.Router) {
		r.Post("/", h.StartWorkflow)
		r.Get("/", h.ListWorkflows)
		r.Get("/{id}", h.GetWorkflowStatus)
		r.Post("/{id}/cancel", h.CancelWorkflow)
		r.Get("/{id}/history", h.GetWorkflowHistory)
	})
}

// StartWorkflow handles POST /api/v1/workflows
func (h *Handler) StartWorkflow(w http.ResponseWriter, r *http.Request) {
	var req StartWorkflowRequest
	if err := h.decodeAndValidate(r, &req); err != nil {
		h.respondValidationError(w, err)
		return
	}

	// Create execution record
	exec := &repository.WorkflowExecution{
		WorkflowID:   req.WorkflowID,
		WorkflowType: req.WorkflowType,
		Status:       repository.StatusPending,
		Input:        req.Input,
		Metadata:     req.Metadata,
		StartedAt:    time.Now(),
	}

	if err := h.repo.SaveExecution(r.Context(), exec); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to save workflow execution")
		return
	}

	// Start the workflow
	var run interface{}
	var err error

	switch req.WorkflowType {
	case "ai-pipeline":
		input, parseErr := ToPipelineInput(req)
		if parseErr != nil {
			h.respondError(w, http.StatusBadRequest, "invalid pipeline input: "+parseErr.Error())
			return
		}
		run, err = h.engine.ExecuteWorkflow(r.Context(), req.WorkflowID, definitions.AIAgentPipelineWorkflow, input)

	case "test-suite":
		input, parseErr := ToTestSuiteInput(req)
		if parseErr != nil {
			h.respondError(w, http.StatusBadRequest, "invalid test suite input: "+parseErr.Error())
			return
		}
		run, err = h.engine.ExecuteWorkflow(r.Context(), req.WorkflowID, definitions.TestSuiteWorkflow, input)

	default:
		h.respondError(w, http.StatusBadRequest, "unsupported workflow type: "+req.WorkflowType)
		return
	}

	if err != nil {
		_ = h.repo.UpdateStatus(r.Context(), exec.ID, repository.StatusFailed, err.Error())
		h.respondError(w, http.StatusInternalServerError, "failed to start workflow: "+err.Error())
		return
	}

	// Update execution with running status
	_ = h.repo.UpdateStatus(r.Context(), exec.ID, repository.StatusRunning, "")

	resp := StartWorkflowResponse{
		WorkflowID: req.WorkflowID,
		Status:     string(repository.StatusRunning),
	}

	// Try to get RunID from the workflow run if available
	if run != nil {
		// The run type depends on the Temporal client implementation
		resp.RunID = exec.ID // Use execution ID as fallback
	}

	h.respondJSON(w, http.StatusAccepted, resp)
}

// GetWorkflowStatus handles GET /api/v1/workflows/{id}
func (h *Handler) GetWorkflowStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "workflow ID is required")
		return
	}

	exec, err := h.repo.GetExecutionByWorkflowID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "workflow not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to get workflow status")
		return
	}

	h.respondJSON(w, http.StatusOK, ToWorkflowStatusResponse(exec))
}

// CancelWorkflow handles POST /api/v1/workflows/{id}/cancel
func (h *Handler) CancelWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "workflow ID is required")
		return
	}

	var req CancelWorkflowRequest
	_ = h.decodeJSON(r, &req) // Optional body

	exec, err := h.repo.GetExecutionByWorkflowID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "workflow not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to get workflow")
		return
	}

	// Check if workflow can be canceled
	if exec.Status != repository.StatusRunning && exec.Status != repository.StatusPending {
		h.respondError(w, http.StatusBadRequest, "workflow is not running")
		return
	}

	// Cancel via Temporal
	if err := h.engine.CancelWorkflow(r.Context(), id, exec.RunID); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to cancel workflow: "+err.Error())
		return
	}

	// Update status in repository
	reason := req.Reason
	if reason == "" {
		reason = "canceled by user"
	}
	_ = h.repo.UpdateStatus(r.Context(), exec.ID, repository.StatusCanceled, reason)

	h.respondJSON(w, http.StatusOK, CancelWorkflowResponse{
		WorkflowID: id,
		Status:     string(repository.StatusCanceled),
	})
}

// GetWorkflowHistory handles GET /api/v1/workflows/{id}/history
func (h *Handler) GetWorkflowHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "workflow ID is required")
		return
	}

	exec, err := h.repo.GetExecutionByWorkflowID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.respondError(w, http.StatusNotFound, "workflow not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "failed to get workflow")
		return
	}

	// Get history from Temporal
	iter, err := h.engine.GetWorkflowHistory(r.Context(), id, exec.RunID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to get workflow history: "+err.Error())
		return
	}

	var events []HistoryEvent
	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			break
		}
		events = append(events, HistoryEvent{
			EventID:   event.GetEventId(),
			EventType: event.GetEventType().String(),
			Timestamp: event.GetEventTime().AsTime(),
		})
	}

	h.respondJSON(w, http.StatusOK, WorkflowHistoryResponse{
		WorkflowID: id,
		RunID:      exec.RunID,
		Events:     events,
	})
}

// ListWorkflows handles GET /api/v1/workflows
func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	filter := repository.Filter{
		WorkflowType: r.URL.Query().Get("workflowType"),
		Limit:        defaultLimit,
		Offset:       0,
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = repository.Status(status)
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			if parsed > defaultMaxLimit {
				parsed = defaultMaxLimit
			}
			filter.Limit = parsed
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			filter.Offset = parsed
		}
	}

	executions, err := h.repo.ListExecutions(r.Context(), filter)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to list workflows")
		return
	}

	workflows := make([]WorkflowStatusResponse, 0, len(executions))
	for _, exec := range executions {
		workflows = append(workflows, ToWorkflowStatusResponse(&exec))
	}

	h.respondJSON(w, http.StatusOK, ListWorkflowsResponse{
		Workflows: workflows,
		Total:     len(workflows),
		Limit:     filter.Limit,
		Offset:    filter.Offset,
	})
}

// Helper methods

func (h *Handler) respondJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func (h *Handler) respondError(w http.ResponseWriter, code int, message string) {
	h.respondJSON(w, code, ErrorResponse{Error: message})
}

func (h *Handler) respondValidationError(w http.ResponseWriter, err error) {
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		details := make(map[string]string)
		for _, e := range validationErrs {
			details[e.Field()] = formatValidationError(e)
		}
		h.respondJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   "validation failed",
			Details: details,
		})
		return
	}
	h.respondError(w, http.StatusBadRequest, err.Error())
}

func formatValidationError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "is required"
	case "oneof":
		return "must be one of: " + e.Param()
	default:
		return "is invalid"
	}
}

func (h *Handler) decodeJSON(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	return json.NewDecoder(r.Body).Decode(v)
}

func (h *Handler) decodeAndValidate(r *http.Request, v interface{}) error {
	if err := h.decodeJSON(r, v); err != nil {
		return err
	}
	return h.validate.Struct(v)
}

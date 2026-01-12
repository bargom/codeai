// Package compensation provides HTTP handlers for compensation operations.
package compensation

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.temporal.io/sdk/client"

	"github.com/bargom/codeai/internal/workflow/compensation/repository"
)

// CompensateHandler handles compensation-related API requests.
type CompensateHandler struct {
	temporalClient client.Client
	repository     repository.CompensationRepository
}

// NewCompensateHandler creates a new CompensateHandler.
func NewCompensateHandler(temporalClient client.Client, repo repository.CompensationRepository) *CompensateHandler {
	return &CompensateHandler{
		temporalClient: temporalClient,
		repository:     repo,
	}
}

// TriggerCompensationRequest is the request body for triggering compensation.
type TriggerCompensationRequest struct {
	Reason         string            `json:"reason,omitempty"`
	ActivityNames  []string          `json:"activityNames,omitempty"` // If empty, compensate all
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// TriggerCompensationResponse is the response for triggering compensation.
type TriggerCompensationResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

// CompensationHistoryResponse is the response for compensation history.
type CompensationHistoryResponse struct {
	WorkflowID string                        `json:"workflowId"`
	Records    []repository.CompensationRecord `json:"records"`
	Summary    *repository.CompensationSummary  `json:"summary,omitempty"`
}

// HandleTriggerCompensation triggers manual compensation for a workflow.
// POST /api/v1/workflows/:id/compensate
func (h *CompensateHandler) HandleTriggerCompensation(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "id")
	if workflowID == "" {
		writeError(w, http.StatusBadRequest, "workflow ID is required")
		return
	}

	var req TriggerCompensationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && r.ContentLength > 0 {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Signal the workflow to trigger compensation
	signalData := map[string]interface{}{
		"reason":         req.Reason,
		"activityNames":  req.ActivityNames,
		"metadata":       req.Metadata,
		"triggeredAt":    time.Now().UTC().Format(time.RFC3339),
	}

	err := h.temporalClient.SignalWorkflow(r.Context(), workflowID, "", "compensate", signalData)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to trigger compensation: "+err.Error())
		return
	}

	response := TriggerCompensationResponse{
		Status:  "compensation_triggered",
		Message: "Compensation signal sent to workflow",
	}

	writeJSON(w, http.StatusAccepted, response)
}

// HandleGetCompensationHistory retrieves compensation history for a workflow.
// GET /api/v1/workflows/:id/compensation-history
func (h *CompensateHandler) HandleGetCompensationHistory(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "id")
	if workflowID == "" {
		writeError(w, http.StatusBadRequest, "workflow ID is required")
		return
	}

	records, err := h.repository.GetCompensationHistory(r.Context(), workflowID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get compensation history: "+err.Error())
		return
	}

	summary, err := h.repository.GetCompensationSummary(r.Context(), workflowID)
	if err != nil {
		// Log but don't fail - summary is optional
		summary = nil
	}

	response := CompensationHistoryResponse{
		WorkflowID: workflowID,
		Records:    records,
		Summary:    summary,
	}

	writeJSON(w, http.StatusOK, response)
}

// HandleCompensateActivity triggers compensation for a specific activity.
// POST /api/v1/workflows/:id/compensation/:activityName
func (h *CompensateHandler) HandleCompensateActivity(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "id")
	activityName := chi.URLParam(r, "activityName")

	if workflowID == "" {
		writeError(w, http.StatusBadRequest, "workflow ID is required")
		return
	}
	if activityName == "" {
		writeError(w, http.StatusBadRequest, "activity name is required")
		return
	}

	var req TriggerCompensationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && r.ContentLength > 0 {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Signal the workflow to compensate specific activity
	signalData := map[string]interface{}{
		"reason":        req.Reason,
		"activityNames": []string{activityName},
		"metadata":      req.Metadata,
		"triggeredAt":   time.Now().UTC().Format(time.RFC3339),
	}

	err := h.temporalClient.SignalWorkflow(r.Context(), workflowID, "", "compensate", signalData)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to trigger compensation: "+err.Error())
		return
	}

	response := TriggerCompensationResponse{
		Status:  "compensation_triggered",
		Message: "Compensation signal sent for activity: " + activityName,
	}

	writeJSON(w, http.StatusAccepted, response)
}

// HandleListCompensations lists compensation records with filtering.
// GET /api/v1/compensations
func (h *CompensateHandler) HandleListCompensations(w http.ResponseWriter, r *http.Request) {
	filter := repository.ListCompensationsFilter{
		WorkflowID:   r.URL.Query().Get("workflowId"),
		ActivityName: r.URL.Query().Get("activityName"),
		Limit:        50, // Default limit
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = repository.CompensationStatus(status)
	}

	records, err := h.repository.ListCompensations(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list compensations: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"records": records,
		"count":   len(records),
	})
}

// HandleGetCompensationRecord gets a specific compensation record.
// GET /api/v1/compensations/:id
func (h *CompensateHandler) HandleGetCompensationRecord(w http.ResponseWriter, r *http.Request) {
	recordID := chi.URLParam(r, "id")
	if recordID == "" {
		writeError(w, http.StatusBadRequest, "record ID is required")
		return
	}

	record, err := h.repository.GetCompensationRecord(r.Context(), recordID)
	if err != nil {
		writeError(w, http.StatusNotFound, "compensation record not found")
		return
	}

	writeJSON(w, http.StatusOK, record)
}

// RegisterRoutes registers compensation routes on the given router.
func (h *CompensateHandler) RegisterRoutes(r chi.Router) {
	r.Route("/workflows/{id}", func(r chi.Router) {
		r.Post("/compensate", h.HandleTriggerCompensation)
		r.Get("/compensation-history", h.HandleGetCompensationHistory)
		r.Post("/compensation/{activityName}", h.HandleCompensateActivity)
	})

	r.Route("/compensations", func(r chi.Router) {
		r.Get("/", h.HandleListCompensations)
		r.Get("/{id}", h.HandleGetCompensationRecord)
	})
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

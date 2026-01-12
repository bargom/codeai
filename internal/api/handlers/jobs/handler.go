// Package jobs provides HTTP handlers for job-related API endpoints.
package jobs

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/bargom/codeai/internal/scheduler/repository"
	"github.com/bargom/codeai/internal/scheduler/service"
)

// Handler provides HTTP handlers for job operations.
type Handler struct {
	scheduler *service.SchedulerService
	validate  *validator.Validate
}

// NewHandler creates a new job handler.
func NewHandler(scheduler *service.SchedulerService) *Handler {
	return &Handler{
		scheduler: scheduler,
		validate:  validator.New(),
	}
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

// SubmitRequest represents a job submission request.
type SubmitRequest struct {
	TaskType   string         `json:"task_type" validate:"required"`
	Payload    any            `json:"payload"`
	Queue      string         `json:"queue,omitempty"`
	MaxRetries int            `json:"max_retries,omitempty"`
	Timeout    string         `json:"timeout,omitempty"` // e.g., "5m", "1h"
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// ScheduleRequest represents a job scheduling request.
type ScheduleRequest struct {
	SubmitRequest
	ScheduleAt string `json:"schedule_at" validate:"required"` // RFC3339 format
}

// RecurringRequest represents a recurring job creation request.
type RecurringRequest struct {
	SubmitRequest
	CronSpec string `json:"cron_spec" validate:"required"`
}

// JobResponse represents a job response.
type JobResponse struct {
	ID string `json:"id"`
}

// ListResponse represents a paginated list of jobs.
type ListResponse struct {
	Jobs   []repository.Job `json:"jobs"`
	Total  int64            `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// StatsResponse represents queue statistics response.
type StatsResponse struct {
	Queues map[string]service.QueueStats `json:"queues"`
}

// Submit handles POST /api/v1/jobs
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	var req SubmitRequest
	if err := h.decodeAndValidate(r, &req); err != nil {
		h.respondValidationError(w, err)
		return
	}

	// Parse timeout
	var timeout time.Duration
	if req.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(req.Timeout)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid timeout format")
			return
		}
	}

	jobReq := service.JobRequest{
		TaskType:   req.TaskType,
		Payload:    req.Payload,
		Queue:      req.Queue,
		MaxRetries: req.MaxRetries,
		Timeout:    timeout,
		Metadata:   req.Metadata,
	}

	jobID, err := h.scheduler.SubmitJob(r.Context(), jobReq)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, JobResponse{ID: jobID})
}

// Schedule handles POST /api/v1/jobs/schedule
func (h *Handler) Schedule(w http.ResponseWriter, r *http.Request) {
	var req ScheduleRequest
	if err := h.decodeAndValidate(r, &req); err != nil {
		h.respondValidationError(w, err)
		return
	}

	// Parse schedule time
	scheduleAt, err := time.Parse(time.RFC3339, req.ScheduleAt)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid schedule_at format, use RFC3339")
		return
	}

	// Parse timeout
	var timeout time.Duration
	if req.Timeout != "" {
		timeout, err = time.ParseDuration(req.Timeout)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid timeout format")
			return
		}
	}

	jobReq := service.JobRequest{
		TaskType:   req.TaskType,
		Payload:    req.Payload,
		Queue:      req.Queue,
		MaxRetries: req.MaxRetries,
		Timeout:    timeout,
		Metadata:   req.Metadata,
	}

	jobID, err := h.scheduler.ScheduleJob(r.Context(), jobReq, scheduleAt)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, JobResponse{ID: jobID})
}

// CreateRecurring handles POST /api/v1/jobs/recurring
func (h *Handler) CreateRecurring(w http.ResponseWriter, r *http.Request) {
	var req RecurringRequest
	if err := h.decodeAndValidate(r, &req); err != nil {
		h.respondValidationError(w, err)
		return
	}

	// Parse timeout
	var timeout time.Duration
	if req.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(req.Timeout)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid timeout format")
			return
		}
	}

	jobReq := service.JobRequest{
		TaskType:   req.TaskType,
		Payload:    req.Payload,
		Queue:      req.Queue,
		MaxRetries: req.MaxRetries,
		Timeout:    timeout,
		Metadata:   req.Metadata,
	}

	jobID, err := h.scheduler.CreateRecurringJob(r.Context(), jobReq, req.CronSpec)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, JobResponse{ID: jobID})
}

// GetStatus handles GET /api/v1/jobs/{id}
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	if jobID == "" {
		h.respondError(w, http.StatusBadRequest, "job id is required")
		return
	}

	status, err := h.scheduler.GetJobStatus(r.Context(), jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			h.respondError(w, http.StatusNotFound, "job not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, status)
}

// Cancel handles DELETE /api/v1/jobs/{id}
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	if jobID == "" {
		h.respondError(w, http.StatusBadRequest, "job id is required")
		return
	}

	err := h.scheduler.CancelJob(r.Context(), jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			h.respondError(w, http.StatusNotFound, "job not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List handles GET /api/v1/jobs
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := h.getPaginationParams(r)

	filter := repository.JobFilter{
		Limit:  limit,
		Offset: offset,
	}

	// Parse optional filters
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = []repository.JobStatus{repository.JobStatus(status)}
	}

	if taskType := r.URL.Query().Get("task_type"); taskType != "" {
		filter.TaskTypes = []string{taskType}
	}

	if queue := r.URL.Query().Get("queue"); queue != "" {
		filter.Queue = queue
	}

	jobs, total, err := h.scheduler.ListJobs(r.Context(), filter)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, ListResponse{
		Jobs:   jobs,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// GetStats handles GET /api/v1/jobs/stats
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.scheduler.GetQueueStats(r.Context())
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, StatsResponse{Queues: stats})
}

// Helper methods

func (h *Handler) respondJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			return
		}
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
			details[e.Field()] = h.formatValidationError(e)
		}
		h.respondJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   "validation failed",
			Details: details,
		})
		return
	}
	h.respondError(w, http.StatusBadRequest, "invalid input")
}

func (h *Handler) formatValidationError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "is required"
	case "min":
		return "must be at least " + e.Param() + " characters"
	case "max":
		return "must be at most " + e.Param() + " characters"
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

func (h *Handler) validateRequest(v interface{}) error {
	return h.validate.Struct(v)
}

func (h *Handler) decodeAndValidate(r *http.Request, v interface{}) error {
	if err := h.decodeJSON(r, v); err != nil {
		return err
	}
	return h.validateRequest(v)
}

func (h *Handler) getPaginationParams(r *http.Request) (limit, offset int) {
	limit = 20
	offset = 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			if parsed > 100 {
				parsed = 100
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

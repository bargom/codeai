// Package webhooks provides HTTP handlers for webhook management endpoints.
package webhooks

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/bargom/codeai/internal/event/bus"
	"github.com/bargom/codeai/internal/webhook/repository"
	"github.com/bargom/codeai/internal/webhook/service"
)

// Handler provides HTTP handlers for webhook operations.
type Handler struct {
	webhookService *service.WebhookService
	validate       *validator.Validate
}

// NewHandler creates a new webhook handler.
func NewHandler(webhookService *service.WebhookService) *Handler {
	return &Handler{
		webhookService: webhookService,
		validate:       validator.New(),
	}
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

// CreateWebhookRequest represents a request to create a webhook.
type CreateWebhookRequest struct {
	URL      string                 `json:"url" validate:"required,url"`
	Events   []string               `json:"events,omitempty"` // Empty means all events
	Secret   string                 `json:"secret,omitempty"`
	Headers  map[string]string      `json:"headers,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateWebhookRequest represents a request to update a webhook.
type UpdateWebhookRequest struct {
	URL      *string                `json:"url,omitempty" validate:"omitempty,url"`
	Events   []string               `json:"events,omitempty"`
	Secret   *string                `json:"secret,omitempty"`
	Headers  map[string]string      `json:"headers,omitempty"`
	Active   *bool                  `json:"active,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// WebhookResponse represents a webhook in API responses.
type WebhookResponse struct {
	ID           string                 `json:"id"`
	URL          string                 `json:"url"`
	Events       []string               `json:"events"`
	Headers      map[string]string      `json:"headers,omitempty"`
	Active       bool                   `json:"active"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
	LastDelivery *string                `json:"last_delivery,omitempty"`
	FailureCount int                    `json:"failure_count"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ListWebhooksResponse represents a paginated list of webhooks.
type ListWebhooksResponse struct {
	Webhooks []WebhookResponse `json:"webhooks"`
	Total    int               `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

// DeliveryResponse represents a webhook delivery in API responses.
type DeliveryResponse struct {
	ID           string  `json:"id"`
	WebhookID    string  `json:"webhook_id"`
	EventID      string  `json:"event_id"`
	EventType    string  `json:"event_type"`
	URL          string  `json:"url"`
	StatusCode   int     `json:"status_code"`
	Duration     int64   `json:"duration_ms"`
	Attempts     int     `json:"attempts"`
	Success      bool    `json:"success"`
	Error        string  `json:"error,omitempty"`
	DeliveredAt  string  `json:"delivered_at"`
	NextRetryAt  *string `json:"next_retry_at,omitempty"`
}

// ListDeliveriesResponse represents a paginated list of deliveries.
type ListDeliveriesResponse struct {
	Deliveries []DeliveryResponse `json:"deliveries"`
	Total      int                `json:"total"`
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`
}

// Create handles POST /api/v1/webhooks
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateWebhookRequest
	if err := h.decodeAndValidate(r, &req); err != nil {
		h.respondValidationError(w, err)
		return
	}

	// Convert string event types to bus.EventType
	events := make([]bus.EventType, len(req.Events))
	for i, e := range req.Events {
		events[i] = bus.EventType(e)
	}

	registerReq := service.RegisterWebhookRequest{
		URL:      req.URL,
		Events:   events,
		Secret:   req.Secret,
		Headers:  req.Headers,
		Metadata: req.Metadata,
	}

	webhookID, err := h.webhookService.RegisterWebhook(r.Context(), registerReq)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, map[string]string{"id": webhookID})
}

// Get handles GET /api/v1/webhooks/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	webhookID := chi.URLParam(r, "id")
	if webhookID == "" {
		h.respondError(w, http.StatusBadRequest, "webhook id is required")
		return
	}

	webhook, err := h.webhookService.GetWebhook(r.Context(), webhookID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "webhook not found")
		return
	}

	h.respondJSON(w, http.StatusOK, h.toWebhookResponse(webhook))
}

// Update handles PUT /api/v1/webhooks/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	webhookID := chi.URLParam(r, "id")
	if webhookID == "" {
		h.respondError(w, http.StatusBadRequest, "webhook id is required")
		return
	}

	var req UpdateWebhookRequest
	if err := h.decodeAndValidate(r, &req); err != nil {
		h.respondValidationError(w, err)
		return
	}

	// Convert string event types to bus.EventType
	var events []bus.EventType
	if req.Events != nil {
		events = make([]bus.EventType, len(req.Events))
		for i, e := range req.Events {
			events[i] = bus.EventType(e)
		}
	}

	updateReq := service.UpdateWebhookRequest{
		URL:      req.URL,
		Events:   events,
		Secret:   req.Secret,
		Headers:  req.Headers,
		Active:   req.Active,
		Metadata: req.Metadata,
	}

	if err := h.webhookService.UpdateWebhook(r.Context(), webhookID, updateReq); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Fetch updated webhook
	webhook, err := h.webhookService.GetWebhook(r.Context(), webhookID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "webhook not found")
		return
	}

	h.respondJSON(w, http.StatusOK, h.toWebhookResponse(webhook))
}

// Delete handles DELETE /api/v1/webhooks/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	webhookID := chi.URLParam(r, "id")
	if webhookID == "" {
		h.respondError(w, http.StatusBadRequest, "webhook id is required")
		return
	}

	if err := h.webhookService.DeleteWebhook(r.Context(), webhookID); err != nil {
		h.respondError(w, http.StatusNotFound, "webhook not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List handles GET /api/v1/webhooks
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := h.getPaginationParams(r)

	filter := repository.WebhookFilter{
		Limit:  limit,
		Offset: offset,
	}

	// Parse optional active filter
	if activeStr := r.URL.Query().Get("active"); activeStr != "" {
		active := activeStr == "true"
		filter.Active = &active
	}

	webhooks, err := h.webhookService.ListWebhooks(r.Context(), filter)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responses := make([]WebhookResponse, len(webhooks))
	for i, wh := range webhooks {
		responses[i] = h.toWebhookResponse(&wh)
	}

	h.respondJSON(w, http.StatusOK, ListWebhooksResponse{
		Webhooks: responses,
		Total:    len(responses),
		Limit:    limit,
		Offset:   offset,
	})
}

// ListDeliveries handles GET /api/v1/webhooks/{id}/deliveries
func (h *Handler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	webhookID := chi.URLParam(r, "id")
	if webhookID == "" {
		h.respondError(w, http.StatusBadRequest, "webhook id is required")
		return
	}

	limit, offset := h.getPaginationParams(r)

	filter := repository.DeliveryFilter{
		Limit:  limit,
		Offset: offset,
	}

	// Parse optional success filter
	if successStr := r.URL.Query().Get("success"); successStr != "" {
		success := successStr == "true"
		filter.Success = &success
	}

	deliveries, err := h.webhookService.GetDeliveries(r.Context(), webhookID, filter)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responses := make([]DeliveryResponse, len(deliveries))
	for i, d := range deliveries {
		responses[i] = h.toDeliveryResponse(&d)
	}

	h.respondJSON(w, http.StatusOK, ListDeliveriesResponse{
		Deliveries: responses,
		Total:      len(responses),
		Limit:      limit,
		Offset:     offset,
	})
}

// Test handles POST /api/v1/webhooks/{id}/test
func (h *Handler) Test(w http.ResponseWriter, r *http.Request) {
	webhookID := chi.URLParam(r, "id")
	if webhookID == "" {
		h.respondError(w, http.StatusBadRequest, "webhook id is required")
		return
	}

	delivery, err := h.webhookService.SendTestWebhook(r.Context(), webhookID)
	if err != nil {
		// Still return the delivery info even if it failed
		if delivery != nil {
			h.respondJSON(w, http.StatusOK, h.toDeliveryResponse(delivery))
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, h.toDeliveryResponse(delivery))
}

// RetryDelivery handles POST /api/v1/webhooks/deliveries/{id}/retry
func (h *Handler) RetryDelivery(w http.ResponseWriter, r *http.Request) {
	deliveryID := chi.URLParam(r, "id")
	if deliveryID == "" {
		h.respondError(w, http.StatusBadRequest, "delivery id is required")
		return
	}

	if err := h.webhookService.RetryFailedWebhook(r.Context(), deliveryID); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"status": "retry initiated"})
}

// Helper methods

func (h *Handler) toWebhookResponse(wh *repository.WebhookConfig) WebhookResponse {
	events := make([]string, len(wh.Events))
	for i, e := range wh.Events {
		events[i] = string(e)
	}

	resp := WebhookResponse{
		ID:           wh.ID,
		URL:          wh.URL,
		Events:       events,
		Headers:      wh.Headers,
		Active:       wh.Active,
		CreatedAt:    wh.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    wh.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		FailureCount: wh.FailureCount,
		Metadata:     wh.Metadata,
	}

	if wh.LastDelivery != nil {
		s := wh.LastDelivery.Format("2006-01-02T15:04:05Z")
		resp.LastDelivery = &s
	}

	return resp
}

func (h *Handler) toDeliveryResponse(d *repository.WebhookDelivery) DeliveryResponse {
	resp := DeliveryResponse{
		ID:          d.ID,
		WebhookID:   d.WebhookID,
		EventID:     d.EventID,
		EventType:   string(d.EventType),
		URL:         d.URL,
		StatusCode:  d.StatusCode,
		Duration:    d.Duration.Milliseconds(),
		Attempts:    d.Attempts,
		Success:     d.Success,
		Error:       d.Error,
		DeliveredAt: d.DeliveredAt.Format("2006-01-02T15:04:05Z"),
	}

	if d.NextRetryAt != nil {
		s := d.NextRetryAt.Format("2006-01-02T15:04:05Z")
		resp.NextRetryAt = &s
	}

	return resp
}

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
	case "url":
		return "must be a valid URL"
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

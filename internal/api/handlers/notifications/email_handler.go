// Package notifications provides HTTP handlers for notification endpoints.
package notifications

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/bargom/codeai/internal/notification/email"
	"github.com/bargom/codeai/internal/notification/email/repository"
	"github.com/bargom/codeai/internal/notification/email/templates"
	"github.com/bargom/codeai/pkg/integration/brevo"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

// EmailHandler handles email notification endpoints.
type EmailHandler struct {
	emailService *email.EmailService
	validate     *validator.Validate
}

// NewEmailHandler creates a new email handler.
func NewEmailHandler(emailService *email.EmailService) *EmailHandler {
	return &EmailHandler{
		emailService: emailService,
		validate:     validator.New(),
	}
}

// SendEmailRequest is the request body for sending an email.
type SendEmailRequest struct {
	To           []EmailAddress `json:"to" validate:"required,min=1,dive"`
	Cc           []EmailAddress `json:"cc,omitempty" validate:"omitempty,dive"`
	Bcc          []EmailAddress `json:"bcc,omitempty" validate:"omitempty,dive"`
	Subject      string         `json:"subject" validate:"required,max=255"`
	TemplateType string         `json:"templateType" validate:"required"`
	Data         map[string]any `json:"data,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
}

// EmailAddress represents an email recipient.
type EmailAddress struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email" validate:"required,email"`
}

// EmailResponse represents an email log in API responses.
type EmailResponse struct {
	ID           string         `json:"id"`
	MessageID    string         `json:"messageId,omitempty"`
	To           []string       `json:"to"`
	Subject      string         `json:"subject"`
	TemplateType string         `json:"templateType,omitempty"`
	Status       string         `json:"status"`
	SentAt       string         `json:"sentAt"`
	DeliveredAt  *string        `json:"deliveredAt,omitempty"`
	OpenedAt     *string        `json:"openedAt,omitempty"`
	Error        string         `json:"error,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// ListEmailsResponse is the response for listing emails.
type ListEmailsResponse struct {
	Emails []EmailResponse `json:"emails"`
	Total  int             `json:"total"`
}

// TemplateResponse represents a template in API responses.
type TemplateResponse struct {
	Type    string `json:"type"`
	Subject string `json:"subject"`
}

// ListTemplatesResponse is the response for listing templates.
type ListTemplatesResponse struct {
	Templates []TemplateResponse `json:"templates"`
}

// ErrorResponse is the standard error response.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

// SendEmail handles POST /api/v1/notifications/email
func (h *EmailHandler) SendEmail(w http.ResponseWriter, r *http.Request) {
	var req SendEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.respondValidationError(w, err)
		return
	}

	// Convert to brevo.EmailAddress
	to := make([]brevo.EmailAddress, len(req.To))
	for i, addr := range req.To {
		to[i] = brevo.EmailAddress{Name: addr.Name, Email: addr.Email}
	}

	cc := make([]brevo.EmailAddress, len(req.Cc))
	for i, addr := range req.Cc {
		cc[i] = brevo.EmailAddress{Name: addr.Name, Email: addr.Email}
	}

	bcc := make([]brevo.EmailAddress, len(req.Bcc))
	for i, addr := range req.Bcc {
		bcc[i] = brevo.EmailAddress{Name: addr.Name, Email: addr.Email}
	}

	emailReq := email.EmailRequest{
		To:           to,
		Cc:           cc,
		Bcc:          bcc,
		Subject:      req.Subject,
		TemplateType: templates.TemplateType(req.TemplateType),
		Data:         req.Data,
		Tags:         req.Tags,
	}

	if err := h.emailService.SendCustomEmail(r.Context(), emailReq); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to send email: "+err.Error())
		return
	}

	h.respondJSON(w, http.StatusAccepted, map[string]string{
		"message": "email queued successfully",
	})
}

// GetEmailStatus handles GET /api/v1/notifications/email/:id
func (h *EmailHandler) GetEmailStatus(w http.ResponseWriter, r *http.Request) {
	emailID := chi.URLParam(r, "id")
	if emailID == "" {
		h.respondError(w, http.StatusBadRequest, "email ID is required")
		return
	}

	emailLog, err := h.emailService.GetEmailStatus(r.Context(), emailID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "email not found")
		return
	}

	h.respondJSON(w, http.StatusOK, h.toEmailResponse(emailLog))
}

// ListEmails handles GET /api/v1/notifications/email
func (h *EmailHandler) ListEmails(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	filter := repository.EmailFilter{
		Limit:  20,
		Offset: 0,
	}

	if limit := query.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	}

	if offset := query.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	if status := query.Get("status"); status != "" {
		filter.Status = &status
	}

	if to := query.Get("to"); to != "" {
		filter.To = &to
	}

	emails, err := h.emailService.ListEmails(r.Context(), filter)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to list emails: "+err.Error())
		return
	}

	response := ListEmailsResponse{
		Emails: make([]EmailResponse, len(emails)),
		Total:  len(emails),
	}

	for i, e := range emails {
		response.Emails[i] = h.toEmailResponse(&e)
	}

	h.respondJSON(w, http.StatusOK, response)
}

// ListTemplates handles GET /api/v1/notifications/templates
func (h *EmailHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	registry := templates.NewRegistry()
	tmplList := registry.ListTemplates()

	response := ListTemplatesResponse{
		Templates: make([]TemplateResponse, len(tmplList)),
	}

	for i, tmpl := range tmplList {
		response.Templates[i] = TemplateResponse{
			Type:    string(tmpl.Type),
			Subject: tmpl.Subject,
		}
	}

	h.respondJSON(w, http.StatusOK, response)
}

// toEmailResponse converts an EmailLog to EmailResponse.
func (h *EmailHandler) toEmailResponse(log *repository.EmailLog) EmailResponse {
	resp := EmailResponse{
		ID:           log.ID,
		MessageID:    log.MessageID,
		To:           log.To,
		Subject:      log.Subject,
		TemplateType: log.TemplateType,
		Status:       log.Status,
		SentAt:       log.SentAt.Format("2006-01-02T15:04:05Z07:00"),
		Error:        log.Error,
		Metadata:     log.Metadata,
	}

	if log.DeliveredAt != nil {
		deliveredAt := log.DeliveredAt.Format("2006-01-02T15:04:05Z07:00")
		resp.DeliveredAt = &deliveredAt
	}

	if log.OpenedAt != nil {
		openedAt := log.OpenedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.OpenedAt = &openedAt
	}

	return resp
}

// respondJSON writes a JSON response.
func (h *EmailHandler) respondJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// respondError writes a JSON error response.
func (h *EmailHandler) respondError(w http.ResponseWriter, code int, message string) {
	h.respondJSON(w, code, ErrorResponse{Error: message})
}

// respondValidationError writes a validation error response.
func (h *EmailHandler) respondValidationError(w http.ResponseWriter, err error) {
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
	h.respondError(w, http.StatusBadRequest, "invalid input")
}

// formatValidationError formats a validation error.
func formatValidationError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "must have at least " + e.Param() + " items"
	case "max":
		return "must have at most " + e.Param() + " characters"
	default:
		return "is invalid"
	}
}

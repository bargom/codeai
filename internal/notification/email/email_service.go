// Package email provides email notification services using Brevo.
package email

import (
	"context"
	"fmt"
	"time"

	"github.com/bargom/codeai/internal/event"
	"github.com/bargom/codeai/internal/notification/email/repository"
	"github.com/bargom/codeai/internal/notification/email/templates"
	"github.com/bargom/codeai/pkg/integration/brevo"
	"github.com/google/uuid"
)

// EmailService handles sending email notifications.
type EmailService struct {
	brevoClient *brevo.Client
	repository  repository.EmailRepository
	eventBus    event.Dispatcher
	templates   *templates.Registry
}

// NewEmailService creates a new email service.
func NewEmailService(
	client *brevo.Client,
	repo repository.EmailRepository,
	eventBus event.Dispatcher,
) *EmailService {
	return &EmailService{
		brevoClient: client,
		repository:  repo,
		eventBus:    eventBus,
		templates:   templates.NewRegistry(),
	}
}

// EmailRequest represents a request to send a custom email.
type EmailRequest struct {
	To           []brevo.EmailAddress
	Cc           []brevo.EmailAddress
	Bcc          []brevo.EmailAddress
	Subject      string
	TemplateType templates.TemplateType
	Data         map[string]interface{}
	Attachments  []brevo.Attachment
	Tags         []string
}

// SendWorkflowNotification sends a workflow status email.
func (s *EmailService) SendWorkflowNotification(ctx context.Context, workflowID string, status string, recipients []string) error {
	var templateType templates.TemplateType
	switch status {
	case "completed":
		templateType = templates.TemplateWorkflowCompleted
	case "failed":
		templateType = templates.TemplateWorkflowFailed
	default:
		return fmt.Errorf("email: unknown workflow status: %s", status)
	}

	tmpl, err := s.templates.GetTemplate(templateType)
	if err != nil {
		return fmt.Errorf("email: get template: %w", err)
	}

	data := map[string]interface{}{
		"WorkflowID":   workflowID,
		"Status":       status,
		"Timestamp":    time.Now().Format(time.RFC3339),
		"DashboardURL": fmt.Sprintf("/workflows/%s", workflowID),
	}

	return s.sendWithTemplate(ctx, tmpl, data, recipients, fmt.Sprintf("workflow:%s", workflowID))
}

// SendJobCompletionEmail sends a job completion notification.
func (s *EmailService) SendJobCompletionEmail(ctx context.Context, jobID string, success bool, recipients []string) error {
	var templateType templates.TemplateType
	if success {
		templateType = templates.TemplateJobCompleted
	} else {
		templateType = templates.TemplateJobFailed
	}

	tmpl, err := s.templates.GetTemplate(templateType)
	if err != nil {
		return fmt.Errorf("email: get template: %w", err)
	}

	status := "completed"
	if !success {
		status = "failed"
	}

	data := map[string]interface{}{
		"JobID":        jobID,
		"Status":       status,
		"Success":      success,
		"Timestamp":    time.Now().Format(time.RFC3339),
		"DashboardURL": fmt.Sprintf("/jobs/%s", jobID),
	}

	return s.sendWithTemplate(ctx, tmpl, data, recipients, fmt.Sprintf("job:%s", jobID))
}

// SendTestResultsEmail sends test suite results.
func (s *EmailService) SendTestResultsEmail(ctx context.Context, testRunID string, results TestResults, recipients []string) error {
	tmpl, err := s.templates.GetTemplate(templates.TemplateTestResultsSummary)
	if err != nil {
		return fmt.Errorf("email: get template: %w", err)
	}

	data := map[string]interface{}{
		"TestRunID":   testRunID,
		"PassedCount": results.PassedCount,
		"FailedCount": results.FailedCount,
		"SkippedCount": results.SkippedCount,
		"Duration":    results.Duration.String(),
		"ReportURL":   fmt.Sprintf("/tests/%s/report", testRunID),
		"Timestamp":   time.Now().Format(time.RFC3339),
	}

	return s.sendWithTemplate(ctx, tmpl, data, recipients, fmt.Sprintf("test:%s", testRunID))
}

// SendCustomEmail sends an arbitrary transactional email.
func (s *EmailService) SendCustomEmail(ctx context.Context, req EmailRequest) error {
	tmpl, err := s.templates.GetTemplate(req.TemplateType)
	if err != nil {
		return fmt.Errorf("email: get template: %w", err)
	}

	htmlContent, err := s.templates.RenderTemplate(tmpl, req.Data)
	if err != nil {
		return fmt.Errorf("email: render template: %w", err)
	}

	textContent, err := s.templates.RenderTextTemplate(tmpl, req.Data)
	if err != nil {
		// Text content is optional, log and continue
		textContent = ""
	}

	subject, err := s.templates.RenderSubject(tmpl, req.Data)
	if err != nil {
		subject = req.Subject
	}

	email := &brevo.TransactionalEmail{
		To:          req.To,
		Cc:          req.Cc,
		Bcc:         req.Bcc,
		Subject:     subject,
		HTMLContent: htmlContent,
		TextContent: textContent,
		Attachments: req.Attachments,
		Tags:        req.Tags,
	}

	messageID, err := s.brevoClient.SendTransactionalEmail(ctx, email)
	if err != nil {
		return s.logEmailFailure(ctx, email, err)
	}

	return s.logEmailSuccess(ctx, messageID, email, string(req.TemplateType))
}

// TestResults holds test execution results for email notifications.
type TestResults struct {
	PassedCount  int
	FailedCount  int
	SkippedCount int
	Duration     time.Duration
}

// sendWithTemplate renders and sends an email using a template.
func (s *EmailService) sendWithTemplate(ctx context.Context, tmpl *templates.Template, data map[string]interface{}, recipients []string, reference string) error {
	htmlContent, err := s.templates.RenderTemplate(tmpl, data)
	if err != nil {
		return fmt.Errorf("email: render html: %w", err)
	}

	textContent, err := s.templates.RenderTextTemplate(tmpl, data)
	if err != nil {
		// Text content is optional
		textContent = ""
	}

	subject, err := s.templates.RenderSubject(tmpl, data)
	if err != nil {
		return fmt.Errorf("email: render subject: %w", err)
	}

	to := make([]brevo.EmailAddress, len(recipients))
	for i, r := range recipients {
		to[i] = brevo.EmailAddress{Email: r}
	}

	email := &brevo.TransactionalEmail{
		To:          to,
		Subject:     subject,
		HTMLContent: htmlContent,
		TextContent: textContent,
		Tags:        []string{string(tmpl.Type), reference},
	}

	messageID, err := s.brevoClient.SendTransactionalEmail(ctx, email)
	if err != nil {
		return s.logEmailFailure(ctx, email, err)
	}

	return s.logEmailSuccess(ctx, messageID, email, string(tmpl.Type))
}

// logEmailSuccess logs a successful email send.
func (s *EmailService) logEmailSuccess(ctx context.Context, messageID string, email *brevo.TransactionalEmail, templateType string) error {
	if s.repository == nil {
		return nil
	}

	recipients := make([]string, len(email.To))
	for i, addr := range email.To {
		recipients[i] = addr.Email
	}

	log := &repository.EmailLog{
		ID:           uuid.New().String(),
		MessageID:    messageID,
		To:           recipients,
		Subject:      email.Subject,
		TemplateType: templateType,
		Status:       "sent",
		SentAt:       time.Now(),
	}

	return s.repository.SaveEmail(ctx, log)
}

// logEmailFailure logs a failed email send.
func (s *EmailService) logEmailFailure(ctx context.Context, email *brevo.TransactionalEmail, sendErr error) error {
	if s.repository == nil {
		return sendErr
	}

	recipients := make([]string, len(email.To))
	for i, addr := range email.To {
		recipients[i] = addr.Email
	}

	log := &repository.EmailLog{
		ID:      uuid.New().String(),
		To:      recipients,
		Subject: email.Subject,
		Status:  "failed",
		Error:   sendErr.Error(),
		SentAt:  time.Now(),
	}

	_ = s.repository.SaveEmail(ctx, log)
	return sendErr
}

// GetEmailStatus retrieves the status of a sent email.
func (s *EmailService) GetEmailStatus(ctx context.Context, emailID string) (*repository.EmailLog, error) {
	if s.repository == nil {
		return nil, fmt.Errorf("email: repository not configured")
	}
	return s.repository.GetEmail(ctx, emailID)
}

// ListEmails lists sent emails with optional filtering.
func (s *EmailService) ListEmails(ctx context.Context, filter repository.EmailFilter) ([]repository.EmailLog, error) {
	if s.repository == nil {
		return nil, fmt.Errorf("email: repository not configured")
	}
	return s.repository.ListEmails(ctx, filter)
}

// Package subscriber provides event subscribers for email notifications.
package subscriber

import (
	"context"
	"log/slog"

	"github.com/bargom/codeai/internal/event/bus"
	"github.com/bargom/codeai/internal/notification/email"
)

// EmailEventSubscriber subscribes to events and sends email notifications.
type EmailEventSubscriber struct {
	emailService *email.EmailService
	logger       *slog.Logger
	// recipientResolver determines who should receive notifications
	recipientResolver RecipientResolver
}

// RecipientResolver resolves email recipients for events.
type RecipientResolver interface {
	// GetWorkflowRecipients returns recipients for workflow notifications.
	GetWorkflowRecipients(ctx context.Context, workflowID string) ([]string, error)

	// GetJobRecipients returns recipients for job notifications.
	GetJobRecipients(ctx context.Context, jobID string) ([]string, error)

	// GetTestRecipients returns recipients for test notifications.
	GetTestRecipients(ctx context.Context, testRunID string) ([]string, error)
}

// Config holds configuration for the email event subscriber.
type Config struct {
	Logger            *slog.Logger
	RecipientResolver RecipientResolver
}

// NewEmailEventSubscriber creates a new email event subscriber.
func NewEmailEventSubscriber(svc *email.EmailService, cfg Config) *EmailEventSubscriber {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	resolver := cfg.RecipientResolver
	if resolver == nil {
		resolver = &noOpResolver{}
	}

	return &EmailEventSubscriber{
		emailService:      svc,
		logger:            logger,
		recipientResolver: resolver,
	}
}

// Handle processes events and sends email notifications.
func (s *EmailEventSubscriber) Handle(ctx context.Context, event bus.Event) error {
	switch event.Type {
	case bus.EventWorkflowCompleted:
		return s.handleWorkflowCompleted(ctx, event)

	case bus.EventWorkflowFailed:
		return s.handleWorkflowFailed(ctx, event)

	case bus.EventJobCompleted:
		return s.handleJobCompleted(ctx, event)

	case bus.EventJobFailed:
		return s.handleJobFailed(ctx, event)

	case bus.EventTestSuiteCompleted:
		return s.handleTestSuiteCompleted(ctx, event)

	default:
		// Unknown event type, ignore
		return nil
	}
}

// handleWorkflowCompleted sends email for workflow completion.
func (s *EmailEventSubscriber) handleWorkflowCompleted(ctx context.Context, event bus.Event) error {
	workflowID, ok := event.Data["workflowID"].(string)
	if !ok {
		s.logger.Warn("workflow completed event missing workflowID", "eventID", event.ID)
		return nil
	}

	recipients, err := s.recipientResolver.GetWorkflowRecipients(ctx, workflowID)
	if err != nil {
		s.logger.Error("failed to get workflow recipients", "workflowID", workflowID, "error", err)
		return nil
	}

	if len(recipients) == 0 {
		return nil
	}

	if err := s.emailService.SendWorkflowNotification(ctx, workflowID, "completed", recipients); err != nil {
		s.logger.Error("failed to send workflow completed email", "workflowID", workflowID, "error", err)
		return err
	}

	s.logger.Info("sent workflow completed email", "workflowID", workflowID, "recipients", len(recipients))
	return nil
}

// handleWorkflowFailed sends email for workflow failure.
func (s *EmailEventSubscriber) handleWorkflowFailed(ctx context.Context, event bus.Event) error {
	workflowID, ok := event.Data["workflowID"].(string)
	if !ok {
		s.logger.Warn("workflow failed event missing workflowID", "eventID", event.ID)
		return nil
	}

	recipients, err := s.recipientResolver.GetWorkflowRecipients(ctx, workflowID)
	if err != nil {
		s.logger.Error("failed to get workflow recipients", "workflowID", workflowID, "error", err)
		return nil
	}

	if len(recipients) == 0 {
		return nil
	}

	if err := s.emailService.SendWorkflowNotification(ctx, workflowID, "failed", recipients); err != nil {
		s.logger.Error("failed to send workflow failed email", "workflowID", workflowID, "error", err)
		return err
	}

	s.logger.Info("sent workflow failed email", "workflowID", workflowID, "recipients", len(recipients))
	return nil
}

// handleJobCompleted sends email for job completion.
func (s *EmailEventSubscriber) handleJobCompleted(ctx context.Context, event bus.Event) error {
	jobID, ok := event.Data["jobID"].(string)
	if !ok {
		s.logger.Warn("job completed event missing jobID", "eventID", event.ID)
		return nil
	}

	recipients, err := s.recipientResolver.GetJobRecipients(ctx, jobID)
	if err != nil {
		s.logger.Error("failed to get job recipients", "jobID", jobID, "error", err)
		return nil
	}

	if len(recipients) == 0 {
		return nil
	}

	if err := s.emailService.SendJobCompletionEmail(ctx, jobID, true, recipients); err != nil {
		s.logger.Error("failed to send job completed email", "jobID", jobID, "error", err)
		return err
	}

	s.logger.Info("sent job completed email", "jobID", jobID, "recipients", len(recipients))
	return nil
}

// handleJobFailed sends email for job failure.
func (s *EmailEventSubscriber) handleJobFailed(ctx context.Context, event bus.Event) error {
	jobID, ok := event.Data["jobID"].(string)
	if !ok {
		s.logger.Warn("job failed event missing jobID", "eventID", event.ID)
		return nil
	}

	recipients, err := s.recipientResolver.GetJobRecipients(ctx, jobID)
	if err != nil {
		s.logger.Error("failed to get job recipients", "jobID", jobID, "error", err)
		return nil
	}

	if len(recipients) == 0 {
		return nil
	}

	if err := s.emailService.SendJobCompletionEmail(ctx, jobID, false, recipients); err != nil {
		s.logger.Error("failed to send job failed email", "jobID", jobID, "error", err)
		return err
	}

	s.logger.Info("sent job failed email", "jobID", jobID, "recipients", len(recipients))
	return nil
}

// handleTestSuiteCompleted sends email for test suite completion.
func (s *EmailEventSubscriber) handleTestSuiteCompleted(ctx context.Context, event bus.Event) error {
	testRunID, ok := event.Data["testRunID"].(string)
	if !ok {
		s.logger.Warn("test suite completed event missing testRunID", "eventID", event.ID)
		return nil
	}

	recipients, err := s.recipientResolver.GetTestRecipients(ctx, testRunID)
	if err != nil {
		s.logger.Error("failed to get test recipients", "testRunID", testRunID, "error", err)
		return nil
	}

	if len(recipients) == 0 {
		return nil
	}

	// Extract test results from event data
	results := email.TestResults{}
	if passed, ok := event.Data["passedCount"].(int); ok {
		results.PassedCount = passed
	}
	if failed, ok := event.Data["failedCount"].(int); ok {
		results.FailedCount = failed
	}
	if skipped, ok := event.Data["skippedCount"].(int); ok {
		results.SkippedCount = skipped
	}

	if err := s.emailService.SendTestResultsEmail(ctx, testRunID, results, recipients); err != nil {
		s.logger.Error("failed to send test results email", "testRunID", testRunID, "error", err)
		return err
	}

	s.logger.Info("sent test results email", "testRunID", testRunID, "recipients", len(recipients))
	return nil
}

// SubscribeTo registers the subscriber with an event bus.
func (s *EmailEventSubscriber) SubscribeTo(eventBus *bus.EventBus) {
	eventBus.Subscribe(bus.EventWorkflowCompleted, s)
	eventBus.Subscribe(bus.EventWorkflowFailed, s)
	eventBus.Subscribe(bus.EventJobCompleted, s)
	eventBus.Subscribe(bus.EventJobFailed, s)
	eventBus.Subscribe(bus.EventTestSuiteCompleted, s)
}

// noOpResolver is a no-op implementation of RecipientResolver.
type noOpResolver struct{}

func (r *noOpResolver) GetWorkflowRecipients(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (r *noOpResolver) GetJobRecipients(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (r *noOpResolver) GetTestRecipients(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

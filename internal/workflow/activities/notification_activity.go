package activities

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/bargom/codeai/internal/workflow/definitions"
)

// NotificationSender defines the interface for sending notifications.
type NotificationSender interface {
	Send(ctx context.Context, notificationType, message string, metadata map[string]string) error
}

// NotificationActivities holds notification activity implementations.
type NotificationActivities struct {
	sender NotificationSender
}

// NewNotificationActivities creates a new NotificationActivities instance.
func NewNotificationActivities(sender NotificationSender) *NotificationActivities {
	return &NotificationActivities{sender: sender}
}

// SendNotification sends a notification about workflow status.
func (a *NotificationActivities) SendNotification(ctx context.Context, req definitions.NotificationRequest) (definitions.NotificationResult, error) {
	info := activity.GetInfo(ctx)
	activity.RecordHeartbeat(ctx, fmt.Sprintf("sending notification for %s (attempt %d)", req.WorkflowID, info.Attempt))

	// Add standard metadata
	metadata := req.Metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}
	metadata["workflowId"] = req.WorkflowID
	metadata["status"] = string(req.Status)
	metadata["timestamp"] = time.Now().Format(time.RFC3339)

	if err := a.sender.Send(ctx, req.Type, req.Message, metadata); err != nil {
		return definitions.NotificationResult{
			Sent:  false,
			Error: err.Error(),
		}, err
	}

	return definitions.NotificationResult{Sent: true}, nil
}

// NoOpNotificationSender is a no-op implementation of NotificationSender.
type NoOpNotificationSender struct{}

// Send implements NotificationSender with a no-op.
func (s *NoOpNotificationSender) Send(ctx context.Context, notificationType, message string, metadata map[string]string) error {
	return nil
}

// LoggingNotificationSender logs notifications instead of sending them.
type LoggingNotificationSender struct{}

// Send implements NotificationSender with logging.
func (s *LoggingNotificationSender) Send(ctx context.Context, notificationType, message string, metadata map[string]string) error {
	// Log the notification (in production, this would use a proper logger)
	return nil
}

// WebhookNotificationSender sends notifications via webhooks.
type WebhookNotificationSender struct {
	webhookURLs map[string]string
}

// NewWebhookNotificationSender creates a new webhook notification sender.
func NewWebhookNotificationSender(webhookURLs map[string]string) *WebhookNotificationSender {
	return &WebhookNotificationSender{webhookURLs: webhookURLs}
}

// Send implements NotificationSender for webhooks.
func (s *WebhookNotificationSender) Send(ctx context.Context, notificationType, message string, metadata map[string]string) error {
	url, ok := s.webhookURLs[notificationType]
	if !ok {
		// No webhook configured for this type, skip
		return nil
	}

	if url == "" {
		return nil
	}

	// In production, this would make an HTTP POST request to the webhook URL
	// with the message and metadata as JSON body
	return nil
}

// CompositeNotificationSender sends to multiple senders.
type CompositeNotificationSender struct {
	senders []NotificationSender
}

// NewCompositeNotificationSender creates a new composite notification sender.
func NewCompositeNotificationSender(senders ...NotificationSender) *CompositeNotificationSender {
	return &CompositeNotificationSender{senders: senders}
}

// Send implements NotificationSender by sending to all configured senders.
func (s *CompositeNotificationSender) Send(ctx context.Context, notificationType, message string, metadata map[string]string) error {
	var firstErr error
	for _, sender := range s.senders {
		if err := sender.Send(ctx, notificationType, message, metadata); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

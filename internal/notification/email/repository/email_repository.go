// Package repository provides email persistence.
package repository

import (
	"context"
	"time"
)

// EmailRepository defines the interface for email persistence.
type EmailRepository interface {
	// SaveEmail persists an email log entry.
	SaveEmail(ctx context.Context, email *EmailLog) error

	// GetEmail retrieves an email log by ID.
	GetEmail(ctx context.Context, emailID string) (*EmailLog, error)

	// ListEmails retrieves email logs with optional filtering.
	ListEmails(ctx context.Context, filter EmailFilter) ([]EmailLog, error)

	// UpdateStatus updates the status of an email log.
	UpdateStatus(ctx context.Context, emailID string, status string) error
}

// EmailLog represents a logged email.
type EmailLog struct {
	ID           string                 `json:"id"`
	MessageID    string                 `json:"messageId,omitempty"` // Brevo message ID
	To           []string               `json:"to"`
	Subject      string                 `json:"subject"`
	TemplateType string                 `json:"templateType,omitempty"`
	Status       string                 `json:"status"` // sent, delivered, opened, bounced, failed
	SentAt       time.Time              `json:"sentAt"`
	DeliveredAt  *time.Time             `json:"deliveredAt,omitempty"`
	OpenedAt     *time.Time             `json:"openedAt,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// EmailFilter defines filtering options for listing emails.
type EmailFilter struct {
	To         *string
	Status     *string
	StartTime  *time.Time
	EndTime    *time.Time
	Limit      int
	Offset     int
}

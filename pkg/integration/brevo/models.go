package brevo

import "time"

// EmailAddress represents an email address with an optional name.
type EmailAddress struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}

// TransactionalEmail represents a transactional email to be sent.
type TransactionalEmail struct {
	To          []EmailAddress
	Cc          []EmailAddress
	Bcc         []EmailAddress
	Subject     string
	HTMLContent string
	TextContent string
	Sender      *EmailAddress
	ReplyTo     *EmailAddress
	Attachments []Attachment
	Headers     map[string]string
	Tags        []string
}

// TemplateEmail represents an email to be sent using a Brevo template.
type TemplateEmail struct {
	To         []EmailAddress
	TemplateID int64
	Params     map[string]interface{}
	Sender     *EmailAddress
	Tags       []string
}

// Attachment represents an email attachment.
type Attachment struct {
	Name    string // Filename
	Content string // Base64 encoded content
	URL     string // URL to fetch the attachment from (alternative to Content)
}

// EmailStatus represents the status of a sent email.
type EmailStatus struct {
	MessageID string
	Status    string // "sent", "delivered", "opened", "bounced", "failed"
	Events    []EmailEvent
}

// EmailEvent represents a single event in an email's lifecycle.
type EmailEvent struct {
	Event     string
	Timestamp time.Time
}

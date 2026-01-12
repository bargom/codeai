// Package brevo provides a client for the Brevo (Sendinblue) transactional email API.
package brevo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "https://api.brevo.com/v3"
	defaultTimeout = 30 * time.Second
)

// Client is a Brevo API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	config     Config
}

// Config holds the configuration for the Brevo client.
type Config struct {
	APIKey         string
	DefaultSender  EmailAddress
	TimeoutSeconds int
	BaseURL        string // Optional, defaults to Brevo API
}

// NewClient creates a new Brevo client.
func NewClient(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("brevo: API key is required")
	}

	timeout := defaultTimeout
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}

	baseURL := defaultBaseURL
	if cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}

	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
		apiKey:     cfg.APIKey,
		config:     cfg,
	}, nil
}

// SendTransactionalEmail sends a transactional email using the Brevo API.
func (c *Client) SendTransactionalEmail(ctx context.Context, email *TransactionalEmail) (string, error) {
	if email == nil {
		return "", fmt.Errorf("brevo: email is required")
	}

	// Use default sender if not specified
	sender := email.Sender
	if sender == nil {
		sender = &c.config.DefaultSender
	}

	// Build the request payload
	payload := sendEmailRequest{
		Sender:      sender,
		To:          email.To,
		Subject:     email.Subject,
		HTMLContent: email.HTMLContent,
		TextContent: email.TextContent,
	}

	if len(email.Cc) > 0 {
		payload.Cc = email.Cc
	}
	if len(email.Bcc) > 0 {
		payload.Bcc = email.Bcc
	}
	if email.ReplyTo != nil {
		payload.ReplyTo = email.ReplyTo
	}
	if len(email.Headers) > 0 {
		payload.Headers = email.Headers
	}
	if len(email.Tags) > 0 {
		payload.Tags = email.Tags
	}
	if len(email.Attachments) > 0 {
		payload.Attachment = make([]attachmentPayload, len(email.Attachments))
		for i, att := range email.Attachments {
			payload.Attachment[i] = attachmentPayload{
				Name:    att.Name,
				Content: att.Content,
				URL:     att.URL,
			}
		}
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/smtp/email", payload)
	if err != nil {
		return "", fmt.Errorf("brevo: send email failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("brevo: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result sendEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("brevo: decode response: %w", err)
	}

	return result.MessageID, nil
}

// SendTemplateEmail sends an email using a Brevo template.
func (c *Client) SendTemplateEmail(ctx context.Context, email *TemplateEmail) (string, error) {
	if email == nil {
		return "", fmt.Errorf("brevo: email is required")
	}

	// Use default sender if not specified
	sender := email.Sender
	if sender == nil {
		sender = &c.config.DefaultSender
	}

	payload := sendTemplateRequest{
		TemplateID: email.TemplateID,
		To:         email.To,
		Params:     email.Params,
		Sender:     sender,
		Tags:       email.Tags,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/smtp/email", payload)
	if err != nil {
		return "", fmt.Errorf("brevo: send template email failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("brevo: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result sendEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("brevo: decode response: %w", err)
	}

	return result.MessageID, nil
}

// GetEmailStatus retrieves the status of a sent email by message ID.
func (c *Client) GetEmailStatus(ctx context.Context, messageID string) (*EmailStatus, error) {
	if messageID == "" {
		return nil, fmt.Errorf("brevo: message ID is required")
	}

	endpoint := fmt.Sprintf("/smtp/emails/%s", messageID)
	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("brevo: get email status failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("brevo: email not found: %s", messageID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("brevo: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result emailStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("brevo: decode response: %w", err)
	}

	status := &EmailStatus{
		MessageID: result.MessageID,
		Status:    result.Event,
		Events:    make([]EmailEvent, len(result.Events)),
	}

	for i, e := range result.Events {
		status.Events[i] = EmailEvent{
			Event:     e.Event,
			Timestamp: e.Date,
		}
	}

	return status, nil
}

// Ping verifies the API connection by checking the account.
func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.doRequest(ctx, http.MethodGet, "/account", nil)
	if err != nil {
		return fmt.Errorf("brevo: ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("brevo: ping returned status %d", resp.StatusCode)
	}

	return nil
}

// doRequest performs an HTTP request to the Brevo API.
func (c *Client) doRequest(ctx context.Context, method, endpoint string, payload any) (*http.Response, error) {
	url := c.baseURL + endpoint

	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

// Internal request/response types for the Brevo API.

type sendEmailRequest struct {
	Sender      *EmailAddress       `json:"sender"`
	To          []EmailAddress      `json:"to"`
	Cc          []EmailAddress      `json:"cc,omitempty"`
	Bcc         []EmailAddress      `json:"bcc,omitempty"`
	ReplyTo     *EmailAddress       `json:"replyTo,omitempty"`
	Subject     string              `json:"subject"`
	HTMLContent string              `json:"htmlContent,omitempty"`
	TextContent string              `json:"textContent,omitempty"`
	Headers     map[string]string   `json:"headers,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	Attachment  []attachmentPayload `json:"attachment,omitempty"`
}

type sendTemplateRequest struct {
	TemplateID int64                  `json:"templateId"`
	To         []EmailAddress         `json:"to"`
	Params     map[string]interface{} `json:"params,omitempty"`
	Sender     *EmailAddress          `json:"sender,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
}

type attachmentPayload struct {
	Name    string `json:"name,omitempty"`
	Content string `json:"content,omitempty"` // base64 encoded
	URL     string `json:"url,omitempty"`
}

type sendEmailResponse struct {
	MessageID string `json:"messageId"`
}

type emailStatusResponse struct {
	MessageID string `json:"messageId"`
	Event     string `json:"event"`
	Events    []struct {
		Event string    `json:"event"`
		Date  time.Time `json:"date"`
	} `json:"events"`
}

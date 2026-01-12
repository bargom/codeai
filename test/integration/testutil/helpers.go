//go:build integration

// Package testutil provides utility functions for integration tests.
package testutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// WaitForCondition polls until the condition is met or timeout expires.
func WaitForCondition(t *testing.T, timeout time.Duration, interval time.Duration, condition func() bool) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

// AssertEventually retries an assertion until it passes or timeout expires.
func AssertEventually(t *testing.T, timeout time.Duration, assertion func() error) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		lastErr = assertion()
		if lastErr == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.NoError(t, lastErr, "assertion failed after timeout")
}

// RequireEventually requires that a condition becomes true within the timeout.
func RequireEventually(t *testing.T, timeout time.Duration, interval time.Duration, condition func() bool, msgAndArgs ...interface{}) {
	t.Helper()

	if !WaitForCondition(t, timeout, interval, condition) {
		require.Fail(t, "condition not met within timeout", msgAndArgs...)
	}
}

// WebhookTestServer creates a test HTTP server that captures webhook deliveries.
type WebhookTestServer struct {
	Server       *httptest.Server
	Deliveries   []WebhookDelivery
	mu           sync.Mutex
	FailRequests bool
	Delay        time.Duration
}

// WebhookDelivery represents a captured webhook delivery.
type WebhookDelivery struct {
	Body      []byte
	Headers   http.Header
	Timestamp time.Time
}

// NewWebhookTestServer creates a new webhook test server.
func NewWebhookTestServer() *WebhookTestServer {
	wts := &WebhookTestServer{
		Deliveries: make([]WebhookDelivery, 0),
	}

	wts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wts.Delay > 0 {
			time.Sleep(wts.Delay)
		}

		if wts.FailRequests {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		defer r.Body.Close()

		wts.mu.Lock()
		wts.Deliveries = append(wts.Deliveries, WebhookDelivery{
			Body:      body,
			Headers:   r.Header.Clone(),
			Timestamp: time.Now(),
		})
		wts.mu.Unlock()

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))

	return wts
}

// GetDeliveries returns a copy of all received deliveries.
func (wts *WebhookTestServer) GetDeliveries() []WebhookDelivery {
	wts.mu.Lock()
	defer wts.mu.Unlock()

	result := make([]WebhookDelivery, len(wts.Deliveries))
	copy(result, wts.Deliveries)
	return result
}

// DeliveryCount returns the number of received deliveries.
func (wts *WebhookTestServer) DeliveryCount() int {
	wts.mu.Lock()
	defer wts.mu.Unlock()
	return len(wts.Deliveries)
}

// Reset clears all captured deliveries.
func (wts *WebhookTestServer) Reset() {
	wts.mu.Lock()
	defer wts.mu.Unlock()
	wts.Deliveries = make([]WebhookDelivery, 0)
	wts.FailRequests = false
}

// Close shuts down the test server.
func (wts *WebhookTestServer) Close() {
	wts.Server.Close()
}

// URL returns the server URL.
func (wts *WebhookTestServer) URL() string {
	return wts.Server.URL
}

// SetFailRequests makes the server return 500 errors.
func (wts *WebhookTestServer) SetFailRequests(fail bool) {
	wts.mu.Lock()
	defer wts.mu.Unlock()
	wts.FailRequests = fail
}

// EventCollector collects events for testing.
type EventCollector struct {
	events []interface{}
	mu     sync.Mutex
}

// NewEventCollector creates a new event collector.
func NewEventCollector() *EventCollector {
	return &EventCollector{
		events: make([]interface{}, 0),
	}
}

// Collect adds an event to the collection.
func (ec *EventCollector) Collect(event interface{}) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.events = append(ec.events, event)
}

// Events returns a copy of all collected events.
func (ec *EventCollector) Events() []interface{} {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	result := make([]interface{}, len(ec.events))
	copy(result, ec.events)
	return result
}

// Count returns the number of collected events.
func (ec *EventCollector) Count() int {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	return len(ec.events)
}

// Reset clears all collected events.
func (ec *EventCollector) Reset() {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.events = make([]interface{}, 0)
}

// ContextWithTimeout creates a context with the specified timeout.
func ContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// TestContext returns a background context suitable for tests.
func TestContext() context.Context {
	return context.Background()
}

// GenerateTestID generates a unique test ID.
func GenerateTestID(prefix string) string {
	return prefix + "-" + time.Now().Format("20060102150405") + "-" + RandomString(6)
}

// RandomString generates a random alphanumeric string.
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

// MockEmailClient is a mock implementation of the email client for testing.
type MockEmailClient struct {
	mu         sync.Mutex
	SentEmails []MockSentEmail
	ShouldFail bool
	FailAfter  int
	sendCount  int
}

// MockSentEmail represents a captured email.
type MockSentEmail struct {
	To        []string
	Subject   string
	HTML      string
	Text      string
	Timestamp time.Time
}

// NewMockEmailClient creates a new mock email client.
func NewMockEmailClient() *MockEmailClient {
	return &MockEmailClient{
		SentEmails: make([]MockSentEmail, 0),
	}
}

// Send simulates sending an email.
func (m *MockEmailClient) Send(to []string, subject, html, text string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sendCount++
	if m.ShouldFail || (m.FailAfter > 0 && m.sendCount > m.FailAfter) {
		return "", context.DeadlineExceeded
	}

	m.SentEmails = append(m.SentEmails, MockSentEmail{
		To:        to,
		Subject:   subject,
		HTML:      html,
		Text:      text,
		Timestamp: time.Now(),
	})

	return "mock-message-id-" + RandomString(8), nil
}

// GetSentEmails returns all sent emails.
func (m *MockEmailClient) GetSentEmails() []MockSentEmail {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]MockSentEmail, len(m.SentEmails))
	copy(result, m.SentEmails)
	return result
}

// Reset clears all sent emails and resets failure settings.
func (m *MockEmailClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SentEmails = make([]MockSentEmail, 0)
	m.ShouldFail = false
	m.FailAfter = 0
	m.sendCount = 0
}

// Count returns the number of sent emails.
func (m *MockEmailClient) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.SentEmails)
}

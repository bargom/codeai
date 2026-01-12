//go:build integration

package testutil

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/bargom/codeai/internal/event/bus"
	emailrepo "github.com/bargom/codeai/internal/notification/email/repository"
	schedrepo "github.com/bargom/codeai/internal/scheduler/repository"
	webhookrepo "github.com/bargom/codeai/internal/webhook/repository"
)

// FixtureBuilder provides methods for creating test fixtures.
type FixtureBuilder struct{}

// NewFixtureBuilder creates a new fixture builder.
func NewFixtureBuilder() *FixtureBuilder {
	return &FixtureBuilder{}
}

// CreateTestJob creates a test job with default values.
func (f *FixtureBuilder) CreateTestJob(taskType string, payload interface{}) *schedrepo.Job {
	payloadBytes, _ := json.Marshal(payload)
	now := time.Now()

	return &schedrepo.Job{
		ID:         uuid.New().String(),
		TaskType:   taskType,
		Payload:    payloadBytes,
		Status:     schedrepo.JobStatusPending,
		Queue:      "default",
		MaxRetries: 3,
		Timeout:    30 * time.Second,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// CreateScheduledJob creates a job scheduled for future execution.
func (f *FixtureBuilder) CreateScheduledJob(taskType string, payload interface{}, scheduledAt time.Time) *schedrepo.Job {
	job := f.CreateTestJob(taskType, payload)
	job.Status = schedrepo.JobStatusScheduled
	job.ScheduledAt = &scheduledAt
	return job
}

// CreateRecurringJob creates a recurring job with cron expression.
func (f *FixtureBuilder) CreateRecurringJob(taskType string, payload interface{}, cronExpr string) *schedrepo.Job {
	job := f.CreateTestJob(taskType, payload)
	job.Status = schedrepo.JobStatusScheduled
	job.CronExpression = cronExpr
	job.CronEntryID = uuid.New().String()
	return job
}

// CreateCompletedJob creates a job that has completed successfully.
func (f *FixtureBuilder) CreateCompletedJob(taskType string, payload interface{}, result interface{}) *schedrepo.Job {
	job := f.CreateTestJob(taskType, payload)
	job.Status = schedrepo.JobStatusCompleted
	now := time.Now()
	job.CompletedAt = &now
	job.StartedAt = &now
	if result != nil {
		resultBytes, _ := json.Marshal(result)
		job.Result = resultBytes
	}
	return job
}

// CreateFailedJob creates a job that has failed.
func (f *FixtureBuilder) CreateFailedJob(taskType string, payload interface{}, errMsg string) *schedrepo.Job {
	job := f.CreateTestJob(taskType, payload)
	job.Status = schedrepo.JobStatusFailed
	now := time.Now()
	job.FailedAt = &now
	job.Error = errMsg
	job.RetryCount = 3
	return job
}

// CreateTestWebhook creates a test webhook configuration.
func (f *FixtureBuilder) CreateTestWebhook(url string, events ...bus.EventType) *webhookrepo.WebhookConfig {
	now := time.Now()

	return &webhookrepo.WebhookConfig{
		ID:        uuid.New().String(),
		URL:       url,
		Events:    events,
		Secret:    "test-secret-" + uuid.New().String()[:8],
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// CreateTestWebhookWithHeaders creates a webhook with custom headers.
func (f *FixtureBuilder) CreateTestWebhookWithHeaders(url string, headers map[string]string, events ...bus.EventType) *webhookrepo.WebhookConfig {
	webhook := f.CreateTestWebhook(url, events...)
	webhook.Headers = headers
	return webhook
}

// CreateDisabledWebhook creates an inactive webhook.
func (f *FixtureBuilder) CreateDisabledWebhook(url string, events ...bus.EventType) *webhookrepo.WebhookConfig {
	webhook := f.CreateTestWebhook(url, events...)
	webhook.Active = false
	return webhook
}

// CreateTestEvent creates a test event.
func (f *FixtureBuilder) CreateTestEvent(eventType bus.EventType, data map[string]interface{}) bus.Event {
	return bus.Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Source:    "test",
		Timestamp: time.Now(),
		Data:      data,
		Metadata:  map[string]string{"test": "true"},
	}
}

// CreateWorkflowCompletedEvent creates a workflow completed event.
func (f *FixtureBuilder) CreateWorkflowCompletedEvent(workflowID string) bus.Event {
	return f.CreateTestEvent(bus.EventWorkflowCompleted, map[string]interface{}{
		"workflow_id": workflowID,
		"status":      "completed",
	})
}

// CreateJobCompletedEvent creates a job completed event.
func (f *FixtureBuilder) CreateJobCompletedEvent(jobID string) bus.Event {
	return f.CreateTestEvent(bus.EventJobCompleted, map[string]interface{}{
		"job_id": jobID,
		"status": "completed",
	})
}

// CreateTestEmailLog creates a test email log entry.
func (f *FixtureBuilder) CreateTestEmailLog(to []string, subject string, status string) *emailrepo.EmailLog {
	return &emailrepo.EmailLog{
		ID:           uuid.New().String(),
		MessageID:    "msg-" + uuid.New().String()[:8],
		To:           to,
		Subject:      subject,
		TemplateType: "test",
		Status:       status,
		SentAt:       time.Now(),
	}
}

// CreateWebhookDelivery creates a test webhook delivery record.
func (f *FixtureBuilder) CreateWebhookDelivery(webhookID string, eventType bus.EventType, success bool) *webhookrepo.WebhookDelivery {
	return &webhookrepo.WebhookDelivery{
		ID:          uuid.New().String(),
		WebhookID:   webhookID,
		EventID:     uuid.New().String(),
		EventType:   eventType,
		URL:         "https://example.com/webhook",
		RequestBody: []byte(`{"test": true}`),
		StatusCode:  200,
		Success:     success,
		Attempts:    1,
		DeliveredAt: time.Now(),
	}
}

// TestPayloads provides common test payloads.
type TestPayloads struct{}

// SimplePayload returns a simple test payload.
func (TestPayloads) SimplePayload() map[string]interface{} {
	return map[string]interface{}{
		"message": "test message",
		"count":   42,
	}
}

// WorkflowPayload returns a workflow test payload.
func (TestPayloads) WorkflowPayload() map[string]interface{} {
	return map[string]interface{}{
		"workflow_id": uuid.New().String(),
		"input": map[string]interface{}{
			"query":    "test query",
			"maxSteps": 5,
		},
	}
}

// AgentPayload returns an agent execution payload.
func (TestPayloads) AgentPayload() map[string]interface{} {
	return map[string]interface{}{
		"agent_type": "test-agent",
		"input":      "test input",
		"options": map[string]interface{}{
			"verbose": true,
		},
	}
}

// TestRecipients returns test email recipients.
func (TestPayloads) TestRecipients() []string {
	return []string{"test1@example.com", "test2@example.com"}
}

// Payloads is a convenience accessor for test payloads.
var Payloads = TestPayloads{}

// Fixtures is a global fixture builder instance.
var Fixtures = NewFixtureBuilder()

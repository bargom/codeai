package builder

import (
	"errors"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/event/bus"
	"github.com/stretchr/testify/assert"
)

func TestNewEvent(t *testing.T) {
	builder := NewEvent(bus.EventWorkflowStarted)

	event := builder.Build()

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, bus.EventWorkflowStarted, event.Type)
	assert.NotZero(t, event.Timestamp)
	assert.NotNil(t, event.Data)
	assert.NotNil(t, event.Metadata)
}

func TestEventBuilder_WithID(t *testing.T) {
	customID := "custom-id-123"
	event := NewEvent(bus.EventJobStarted).
		WithID(customID).
		Build()

	assert.Equal(t, customID, event.ID)
}

func TestEventBuilder_WithSource(t *testing.T) {
	event := NewEvent(bus.EventJobCompleted).
		WithSource("test-source").
		Build()

	assert.Equal(t, "test-source", event.Source)
}

func TestEventBuilder_WithTimestamp(t *testing.T) {
	customTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	event := NewEvent(bus.EventAgentExecuted).
		WithTimestamp(customTime).
		Build()

	assert.Equal(t, customTime, event.Timestamp)
}

func TestEventBuilder_WithData(t *testing.T) {
	event := NewEvent(bus.EventWorkflowCompleted).
		WithData("key1", "value1").
		WithData("key2", 42).
		WithData("key3", true).
		Build()

	assert.Equal(t, "value1", event.Data["key1"])
	assert.Equal(t, 42, event.Data["key2"])
	assert.Equal(t, true, event.Data["key3"])
}

func TestEventBuilder_WithDataMap(t *testing.T) {
	dataMap := map[string]interface{}{
		"field1": "data1",
		"field2": 100,
	}

	event := NewEvent(bus.EventJobEnqueued).
		WithData("existing", "value").
		WithDataMap(dataMap).
		Build()

	assert.Equal(t, "value", event.Data["existing"])
	assert.Equal(t, "data1", event.Data["field1"])
	assert.Equal(t, 100, event.Data["field2"])
}

func TestEventBuilder_WithMetadata(t *testing.T) {
	event := NewEvent(bus.EventEmailSent).
		WithMetadata("correlation_id", "abc-123").
		WithMetadata("user_id", "user-456").
		Build()

	assert.Equal(t, "abc-123", event.Metadata["correlation_id"])
	assert.Equal(t, "user-456", event.Metadata["user_id"])
}

func TestEventBuilder_WithMetadataMap(t *testing.T) {
	metadataMap := map[string]string{
		"env":     "production",
		"version": "1.0.0",
	}

	event := NewEvent(bus.EventWebhookTriggered).
		WithMetadata("request_id", "req-123").
		WithMetadataMap(metadataMap).
		Build()

	assert.Equal(t, "req-123", event.Metadata["request_id"])
	assert.Equal(t, "production", event.Metadata["env"])
	assert.Equal(t, "1.0.0", event.Metadata["version"])
}

func TestEventBuilder_ChainedCalls(t *testing.T) {
	customTime := time.Now()

	event := NewEvent(bus.EventWorkflowFailed).
		WithID("chain-test").
		WithSource("chain-source").
		WithTimestamp(customTime).
		WithData("error", "something went wrong").
		WithMetadata("attempt", "3").
		Build()

	assert.Equal(t, "chain-test", event.ID)
	assert.Equal(t, bus.EventWorkflowFailed, event.Type)
	assert.Equal(t, "chain-source", event.Source)
	assert.Equal(t, customTime, event.Timestamp)
	assert.Equal(t, "something went wrong", event.Data["error"])
	assert.Equal(t, "3", event.Metadata["attempt"])
}

func TestWorkflowStarted(t *testing.T) {
	event := WorkflowStarted("wf-123").Build()

	assert.Equal(t, bus.EventWorkflowStarted, event.Type)
	assert.Equal(t, "workflow-engine", event.Source)
	assert.Equal(t, "wf-123", event.Data["workflowID"])
}

func TestWorkflowCompleted(t *testing.T) {
	event := WorkflowCompleted("wf-456", 123.45).Build()

	assert.Equal(t, bus.EventWorkflowCompleted, event.Type)
	assert.Equal(t, "workflow-engine", event.Source)
	assert.Equal(t, "wf-456", event.Data["workflowID"])
	assert.Equal(t, 123.45, event.Data["duration"])
}

func TestWorkflowFailed(t *testing.T) {
	err := errors.New("workflow error")
	event := WorkflowFailed("wf-789", err).Build()

	assert.Equal(t, bus.EventWorkflowFailed, event.Type)
	assert.Equal(t, "workflow-engine", event.Source)
	assert.Equal(t, "wf-789", event.Data["workflowID"])
	assert.Equal(t, "workflow error", event.Data["error"])
}

func TestJobEnqueued(t *testing.T) {
	event := JobEnqueued("job-1", "email").Build()

	assert.Equal(t, bus.EventJobEnqueued, event.Type)
	assert.Equal(t, "job-queue", event.Source)
	assert.Equal(t, "job-1", event.Data["jobID"])
	assert.Equal(t, "email", event.Data["jobType"])
}

func TestJobStarted(t *testing.T) {
	event := JobStarted("job-2", "worker-1").Build()

	assert.Equal(t, bus.EventJobStarted, event.Type)
	assert.Equal(t, "job-worker", event.Source)
	assert.Equal(t, "job-2", event.Data["jobID"])
	assert.Equal(t, "worker-1", event.Data["workerID"])
}

func TestJobCompleted(t *testing.T) {
	event := JobCompleted("job-3", 5.5).Build()

	assert.Equal(t, bus.EventJobCompleted, event.Type)
	assert.Equal(t, "job-worker", event.Source)
	assert.Equal(t, "job-3", event.Data["jobID"])
	assert.Equal(t, 5.5, event.Data["duration"])
}

func TestJobFailed(t *testing.T) {
	err := errors.New("job failed")
	event := JobFailed("job-4", err, 3).Build()

	assert.Equal(t, bus.EventJobFailed, event.Type)
	assert.Equal(t, "job-worker", event.Source)
	assert.Equal(t, "job-4", event.Data["jobID"])
	assert.Equal(t, "job failed", event.Data["error"])
	assert.Equal(t, 3, event.Data["attempts"])
}

func TestAgentExecuted(t *testing.T) {
	event := AgentExecuted("agent-1", "analyze").Build()

	assert.Equal(t, bus.EventAgentExecuted, event.Type)
	assert.Equal(t, "agent-runtime", event.Source)
	assert.Equal(t, "agent-1", event.Data["agentID"])
	assert.Equal(t, "analyze", event.Data["action"])
}

func TestTestSuiteCompleted(t *testing.T) {
	event := TestSuiteCompleted("suite-1", 10, 2, 1).Build()

	assert.Equal(t, bus.EventTestSuiteCompleted, event.Type)
	assert.Equal(t, "test-runner", event.Source)
	assert.Equal(t, "suite-1", event.Data["suiteID"])
	assert.Equal(t, 10, event.Data["passed"])
	assert.Equal(t, 2, event.Data["failed"])
	assert.Equal(t, 1, event.Data["skipped"])
}

func TestWebhookTriggered(t *testing.T) {
	event := WebhookTriggered("wh-1", "https://example.com").Build()

	assert.Equal(t, bus.EventWebhookTriggered, event.Type)
	assert.Equal(t, "webhook-service", event.Source)
	assert.Equal(t, "wh-1", event.Data["webhookID"])
	assert.Equal(t, "https://example.com", event.Data["url"])
}

func TestEmailSent(t *testing.T) {
	event := EmailSent("email-1", "user@example.com").Build()

	assert.Equal(t, bus.EventEmailSent, event.Type)
	assert.Equal(t, "email-service", event.Source)
	assert.Equal(t, "email-1", event.Data["emailID"])
	assert.Equal(t, "user@example.com", event.Data["recipient"])
}

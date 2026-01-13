//go:build integration

// Package cli provides CLI integration tests for CodeAI event and integration parsing.
package cli

import (
	"testing"

	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEventParsing tests the event parser functionality.
func TestEventParsing(t *testing.T) {
	t.Run("parse event with schema", func(t *testing.T) {
		input := `
event user_created {
	schema {
		user_id uuid
		email string
		name string
		created_at timestamp
	}
}
`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		require.Len(t, prog.Statements, 1)

		eventDecl, ok := prog.Statements[0].(*ast.EventDecl)
		require.True(t, ok, "expected EventDecl")
		assert.Equal(t, "user_created", eventDecl.Name)
		require.NotNil(t, eventDecl.Schema)
		assert.Len(t, eventDecl.Schema.Fields, 4)

		// Verify field types
		assert.Equal(t, "user_id", eventDecl.Schema.Fields[0].Name)
		assert.Equal(t, "uuid", eventDecl.Schema.Fields[0].FieldType)
	})

	t.Run("parse event without schema", func(t *testing.T) {
		input := `
event order_cancelled {
}
`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		require.Len(t, prog.Statements, 1)

		eventDecl, ok := prog.Statements[0].(*ast.EventDecl)
		require.True(t, ok)
		assert.Equal(t, "order_cancelled", eventDecl.Name)
		assert.Nil(t, eventDecl.Schema)
	})

	t.Run("parse multiple events", func(t *testing.T) {
		input := `
event user_created {
	schema {
		user_id uuid
	}
}

event user_updated {
	schema {
		user_id uuid
		updated_at timestamp
	}
}
`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		assert.Len(t, prog.Statements, 2)
	})
}

// TestEventHandlerParsing tests the event handler parser functionality.
func TestEventHandlerParsing(t *testing.T) {
	t.Run("parse workflow handler sync", func(t *testing.T) {
		input := `on "user.created" do workflow "send_welcome_email"`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		require.Len(t, prog.Statements, 1)

		handler, ok := prog.Statements[0].(*ast.EventHandlerDecl)
		require.True(t, ok)
		assert.Equal(t, "user.created", handler.EventName)
		assert.Equal(t, "workflow", handler.ActionType)
		assert.Equal(t, "send_welcome_email", handler.Target)
		assert.False(t, handler.Async)
	})

	t.Run("parse workflow handler async", func(t *testing.T) {
		input := `on "user.created" do workflow "send_welcome_email" async`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		require.Len(t, prog.Statements, 1)

		handler, ok := prog.Statements[0].(*ast.EventHandlerDecl)
		require.True(t, ok)
		assert.True(t, handler.Async)
	})

	t.Run("parse integration handler", func(t *testing.T) {
		input := `on "order.completed" do integration "crm.update_contact"`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		require.Len(t, prog.Statements, 1)

		handler, ok := prog.Statements[0].(*ast.EventHandlerDecl)
		require.True(t, ok)
		assert.Equal(t, "integration", handler.ActionType)
		assert.Equal(t, "crm.update_contact", handler.Target)
	})

	t.Run("parse emit handler", func(t *testing.T) {
		input := `on "payment.received" do emit "order.completed"`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		require.Len(t, prog.Statements, 1)

		handler, ok := prog.Statements[0].(*ast.EventHandlerDecl)
		require.True(t, ok)
		assert.Equal(t, "emit", handler.ActionType)
		assert.Equal(t, "order.completed", handler.Target)
	})

	t.Run("parse webhook handler", func(t *testing.T) {
		input := `on "order.completed" do webhook "analytics_system"`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		require.Len(t, prog.Statements, 1)

		handler, ok := prog.Statements[0].(*ast.EventHandlerDecl)
		require.True(t, ok)
		assert.Equal(t, "webhook", handler.ActionType)
		assert.Equal(t, "analytics_system", handler.Target)
	})
}

// TestIntegrationParsing tests the integration parser functionality.
func TestIntegrationParsing(t *testing.T) {
	t.Run("parse simple rest integration", func(t *testing.T) {
		input := `
integration stripe {
	type rest
	base_url "https://api.stripe.com/v1"
}
`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		require.Len(t, prog.Statements, 1)

		intg, ok := prog.Statements[0].(*ast.IntegrationDecl)
		require.True(t, ok)
		assert.Equal(t, "stripe", intg.Name)
		assert.Equal(t, ast.IntegrationTypeREST, intg.IntgType)
		assert.Equal(t, "https://api.stripe.com/v1", intg.BaseURL)
	})

	t.Run("parse integration with bearer auth", func(t *testing.T) {
		input := `
integration analytics {
	type rest
	base_url "https://api.analytics.example.com/v2"
	auth bearer {
		token: "$API_TOKEN"
	}
}
`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		require.Len(t, prog.Statements, 1)

		intg, ok := prog.Statements[0].(*ast.IntegrationDecl)
		require.True(t, ok)
		assert.Equal(t, "analytics", intg.Name)
		require.NotNil(t, intg.Auth)
		assert.Equal(t, ast.IntegrationAuthBearer, intg.Auth.AuthType)
	})

	t.Run("parse integration with apikey auth", func(t *testing.T) {
		input := `
integration crm {
	type rest
	base_url "https://api.crm.example.com"
	auth apikey {
		header: "X-API-Key"
		value: "$CRM_API_KEY"
	}
}
`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		require.Len(t, prog.Statements, 1)

		intg, ok := prog.Statements[0].(*ast.IntegrationDecl)
		require.True(t, ok)
		require.NotNil(t, intg.Auth)
		assert.Equal(t, ast.IntegrationAuthAPIKey, intg.Auth.AuthType)
	})

	t.Run("parse integration with circuit breaker", func(t *testing.T) {
		input := `
integration payments {
	type rest
	base_url "https://api.payments.example.com"
	auth bearer {
		token: "$PAYMENT_TOKEN"
	}
	timeout "30s"
	circuit_breaker {
		threshold 5
		timeout "60s"
		max_concurrent 100
	}
}
`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		require.Len(t, prog.Statements, 1)

		intg, ok := prog.Statements[0].(*ast.IntegrationDecl)
		require.True(t, ok)
		assert.Equal(t, "30s", intg.Timeout)
		require.NotNil(t, intg.CircuitBreaker)
		assert.Equal(t, 5, intg.CircuitBreaker.FailureThreshold)
		assert.Equal(t, "60s", intg.CircuitBreaker.Timeout)
		assert.Equal(t, 100, intg.CircuitBreaker.MaxConcurrent)
	})

	t.Run("parse graphql integration", func(t *testing.T) {
		input := `
integration notifications {
	type graphql
	base_url "https://api.notifications.example.com/graphql"
}
`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		require.Len(t, prog.Statements, 1)

		intg, ok := prog.Statements[0].(*ast.IntegrationDecl)
		require.True(t, ok)
		assert.Equal(t, ast.IntegrationTypeGraphQL, intg.IntgType)
	})
}

// TestWebhookParsing tests the webhook parser functionality.
func TestWebhookParsing(t *testing.T) {
	t.Run("parse webhook with headers and retry", func(t *testing.T) {
		input := `
webhook analytics_system {
	event "order.completed"
	url "https://analytics.example.com/events"
	method POST
	headers {
		"Content-Type": "application/json"
		"X-Source": "codeai"
	}
	retry 3 initial_interval "1s" backoff 2.0
}
`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		require.Len(t, prog.Statements, 1)

		webhook, ok := prog.Statements[0].(*ast.WebhookDecl)
		require.True(t, ok)
		assert.Equal(t, "analytics_system", webhook.Name)
		assert.Equal(t, "order.completed", webhook.Event)
		assert.Equal(t, "https://analytics.example.com/events", webhook.URL)
		assert.Equal(t, ast.WebhookMethodPOST, webhook.Method)
		assert.Len(t, webhook.Headers, 2)
		require.NotNil(t, webhook.Retry)
		assert.Equal(t, 3, webhook.Retry.MaxAttempts)
		assert.Equal(t, "1s", webhook.Retry.InitialInterval)
		assert.Equal(t, 2.0, webhook.Retry.BackoffMultiplier)
	})

	t.Run("parse webhook with PUT method", func(t *testing.T) {
		input := `
webhook update_system {
	event "user.updated"
	url "https://update.example.com/events"
	method PUT
}
`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		require.Len(t, prog.Statements, 1)

		webhook, ok := prog.Statements[0].(*ast.WebhookDecl)
		require.True(t, ok)
		assert.Equal(t, ast.WebhookMethodPUT, webhook.Method)
	})

	t.Run("parse minimal webhook", func(t *testing.T) {
		input := `
webhook simple_webhook {
	event "test.event"
	url "https://example.com/webhook"
	method POST
}
`
		prog, err := parser.Parse(input)
		require.NoError(t, err)
		require.Len(t, prog.Statements, 1)

		webhook, ok := prog.Statements[0].(*ast.WebhookDecl)
		require.True(t, ok)
		assert.Equal(t, "simple_webhook", webhook.Name)
		assert.Nil(t, webhook.Retry)
	})
}

// TestComplexEventScenario tests parsing a complex scenario with multiple constructs.
func TestComplexEventScenario(t *testing.T) {
	input := `
event user_created {
	schema {
		user_id uuid
		email string
	}
}

on "user.created" do workflow "send_welcome_email" async
on "user.created" do integration "crm.create_contact"

integration crm {
	type rest
	base_url "https://api.crm.example.com"
	auth apikey {
		header: "X-API-Key"
		value: "$CRM_API_KEY"
	}
}

webhook analytics {
	event "user.created"
	url "https://analytics.example.com/events"
	method POST
	headers {
		"Content-Type": "application/json"
	}
}
`
	prog, err := parser.Parse(input)
	require.NoError(t, err)
	assert.Len(t, prog.Statements, 5)

	// Verify each statement type
	_, ok := prog.Statements[0].(*ast.EventDecl)
	assert.True(t, ok, "expected EventDecl")

	_, ok = prog.Statements[1].(*ast.EventHandlerDecl)
	assert.True(t, ok, "expected EventHandlerDecl")

	_, ok = prog.Statements[2].(*ast.EventHandlerDecl)
	assert.True(t, ok, "expected EventHandlerDecl")

	_, ok = prog.Statements[3].(*ast.IntegrationDecl)
	assert.True(t, ok, "expected IntegrationDecl")

	_, ok = prog.Statements[4].(*ast.WebhookDecl)
	assert.True(t, ok, "expected WebhookDecl")
}

// TestEventParsingFileExample tests parsing the example file.
func TestEventParsingFileExample(t *testing.T) {
	prog, err := parser.ParseFile("../../examples/10-events-integrations/events.cai")
	require.NoError(t, err)

	// Count statement types
	var eventCount, handlerCount, integrationCount, webhookCount int
	for _, stmt := range prog.Statements {
		switch stmt.(type) {
		case *ast.EventDecl:
			eventCount++
		case *ast.EventHandlerDecl:
			handlerCount++
		case *ast.IntegrationDecl:
			integrationCount++
		case *ast.WebhookDecl:
			webhookCount++
		}
	}

	// Verify expected counts from the example file
	assert.Equal(t, 5, eventCount, "expected 5 event declarations")
	assert.Equal(t, 5, handlerCount, "expected 5 event handlers")
	assert.Equal(t, 4, integrationCount, "expected 4 integrations")
	assert.Equal(t, 3, webhookCount, "expected 3 webhooks")
}

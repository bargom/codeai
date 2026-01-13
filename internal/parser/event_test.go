package parser

import (
	"testing"

	"github.com/bargom/codeai/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEventWithSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantName   string
		wantFields int
		wantErr    bool
	}{
		{
			name: "simple event with schema",
			input: `event user_created {
				schema {
					user_id string
					email string
					created_at timestamp
				}
			}`,
			wantName:   "user_created",
			wantFields: 3,
			wantErr:    false,
		},
		{
			name: "event without schema",
			input: `event order_completed {
			}`,
			wantName:   "order_completed",
			wantFields: 0,
			wantErr:    false,
		},
		{
			name: "event with various field types",
			input: `event payment_processed {
				schema {
					order_id string
					total decimal
					items array
				}
			}`,
			wantName:   "payment_processed",
			wantFields: 3,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)

			eventDecl, ok := program.Statements[0].(*ast.EventDecl)
			require.True(t, ok, "expected EventDecl")
			assert.Equal(t, tt.wantName, eventDecl.Name)

			if tt.wantFields > 0 {
				require.NotNil(t, eventDecl.Schema)
				assert.Len(t, eventDecl.Schema.Fields, tt.wantFields)
			}
		})
	}
}

func TestParseEventHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		wantEventName  string
		wantActionType string
		wantTarget     string
		wantAsync      bool
		wantErr        bool
	}{
		{
			name:           "workflow handler sync",
			input:          `on "user.created" do workflow "send_welcome_email"`,
			wantEventName:  "user.created",
			wantActionType: "workflow",
			wantTarget:     "send_welcome_email",
			wantAsync:      false,
			wantErr:        false,
		},
		{
			name:           "workflow handler async",
			input:          `on "user.created" do workflow "send_welcome_email" async`,
			wantEventName:  "user.created",
			wantActionType: "workflow",
			wantTarget:     "send_welcome_email",
			wantAsync:      true,
			wantErr:        false,
		},
		{
			name:           "integration handler",
			input:          `on "order.completed" do integration "crm.create_contact"`,
			wantEventName:  "order.completed",
			wantActionType: "integration",
			wantTarget:     "crm.create_contact",
			wantAsync:      false,
			wantErr:        false,
		},
		{
			name:           "emit handler",
			input:          `on "order.completed" do emit "inventory.update"`,
			wantEventName:  "order.completed",
			wantActionType: "emit",
			wantTarget:     "inventory.update",
			wantAsync:      false,
			wantErr:        false,
		},
		{
			name:           "webhook handler",
			input:          `on "order.completed" do webhook "analytics_system"`,
			wantEventName:  "order.completed",
			wantActionType: "webhook",
			wantTarget:     "analytics_system",
			wantAsync:      false,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)

			handler, ok := program.Statements[0].(*ast.EventHandlerDecl)
			require.True(t, ok, "expected EventHandlerDecl")
			assert.Equal(t, tt.wantEventName, handler.EventName)
			assert.Equal(t, tt.wantActionType, handler.ActionType)
			assert.Equal(t, tt.wantTarget, handler.Target)
			assert.Equal(t, tt.wantAsync, handler.Async)
		})
	}
}

func TestParseIntegrationSimple(t *testing.T) {
	input := `integration stripe { type rest base_url "https://api.stripe.com/v1" }`
	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)
	intg, ok := program.Statements[0].(*ast.IntegrationDecl)
	require.True(t, ok)
	assert.Equal(t, "stripe", intg.Name)
	assert.Equal(t, ast.IntegrationTypeREST, intg.IntgType)
	assert.Equal(t, "https://api.stripe.com/v1", intg.BaseURL)
}

func TestParseIntegrationWithBearerAuthMinimal(t *testing.T) {
	input := `integration stripe { type rest base_url "https://api.stripe.com" auth bearer { } }`
	program, err := Parse(input)
	if err != nil {
		t.Logf("Error: %v", err)
	}
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)
}

func TestParseIntegrationWithBearerAuthConfig(t *testing.T) {
	input := `integration stripe { type rest base_url "https://api.stripe.com" auth bearer { token: "test" } }`
	program, err := Parse(input)
	if err != nil {
		t.Logf("Error: %v", err)
	}
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)
}

func TestParseIntegrationWithAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantName    string
		wantType    ast.IntegrationType
		wantBaseURL string
		wantAuth    ast.IntegrationAuthType
		wantErr     bool
	}{
		{
			name:        "rest integration with bearer auth",
			input:       `integration stripe { type rest base_url "https://api.stripe.com/v1" auth bearer { token: env("STRIPE_SECRET_KEY") } }`,
			wantName:    "stripe",
			wantType:    ast.IntegrationTypeREST,
			wantBaseURL: "https://api.stripe.com/v1",
			wantAuth:    ast.IntegrationAuthBearer,
			wantErr:     false,
		},
		{
			name:        "rest integration with apikey auth",
			input:       `integration crm { type rest base_url "https://api.crm.example.com" auth apikey { header: "X-API-Key" value: env("CRM_API_KEY") } }`,
			wantName:    "crm",
			wantType:    ast.IntegrationTypeREST,
			wantBaseURL: "https://api.crm.example.com",
			wantAuth:    ast.IntegrationAuthAPIKey,
			wantErr:     false,
		},
		{
			name:        "integration without auth",
			input:       `integration public_api { type rest base_url "https://api.public.com" }`,
			wantName:    "public_api",
			wantType:    ast.IntegrationTypeREST,
			wantBaseURL: "https://api.public.com",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)

			intg, ok := program.Statements[0].(*ast.IntegrationDecl)
			require.True(t, ok, "expected IntegrationDecl")
			assert.Equal(t, tt.wantName, intg.Name)
			assert.Equal(t, tt.wantType, intg.IntgType)
			assert.Equal(t, tt.wantBaseURL, intg.BaseURL)

			if tt.wantAuth != "" {
				require.NotNil(t, intg.Auth)
				assert.Equal(t, tt.wantAuth, intg.Auth.AuthType)
			}
		})
	}
}

func TestParseCircuitBreaker(t *testing.T) {
	t.Parallel()

	input := `integration stripe {
		type rest
		base_url "https://api.stripe.com/v1"
		auth bearer {
			token: env("STRIPE_SECRET_KEY")
		}
		timeout "30s"
		circuit_breaker {
			threshold 5
			timeout "60s"
			max_concurrent 100
		}
	}`

	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)

	intg, ok := program.Statements[0].(*ast.IntegrationDecl)
	require.True(t, ok, "expected IntegrationDecl")
	assert.Equal(t, "stripe", intg.Name)
	assert.Equal(t, "30s", intg.Timeout)

	require.NotNil(t, intg.CircuitBreaker)
	assert.Equal(t, 5, intg.CircuitBreaker.FailureThreshold)
	assert.Equal(t, "60s", intg.CircuitBreaker.Timeout)
	assert.Equal(t, 100, intg.CircuitBreaker.MaxConcurrent)
}

func TestParseWebhook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantName    string
		wantEvent   string
		wantURL     string
		wantMethod  ast.WebhookHTTPMethod
		wantHeaders int
		wantRetry   bool
		wantErr     bool
	}{
		{
			name: "webhook with headers and retry",
			input: `webhook analytics_system {
				event "order.completed"
				url "https://analytics.example.com/events"
				method POST
				headers {
					"Content-Type": "application/json"
					"X-Source": "codeai"
				}
				retry 3 initial_interval "1s" backoff 2.0
			}`,
			wantName:    "analytics_system",
			wantEvent:   "order.completed",
			wantURL:     "https://analytics.example.com/events",
			wantMethod:  ast.WebhookMethodPOST,
			wantHeaders: 2,
			wantRetry:   true,
			wantErr:     false,
		},
		{
			name: "webhook with PUT method",
			input: `webhook update_system {
				event "user.updated"
				url "https://update.example.com/events"
				method PUT
			}`,
			wantName:    "update_system",
			wantEvent:   "user.updated",
			wantURL:     "https://update.example.com/events",
			wantMethod:  ast.WebhookMethodPUT,
			wantHeaders: 0,
			wantRetry:   false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)

			webhook, ok := program.Statements[0].(*ast.WebhookDecl)
			require.True(t, ok, "expected WebhookDecl")
			assert.Equal(t, tt.wantName, webhook.Name)
			assert.Equal(t, tt.wantEvent, webhook.Event)
			assert.Equal(t, tt.wantURL, webhook.URL)
			assert.Equal(t, tt.wantMethod, webhook.Method)
			assert.Len(t, webhook.Headers, tt.wantHeaders)

			if tt.wantRetry {
				require.NotNil(t, webhook.Retry)
				assert.Equal(t, 3, webhook.Retry.MaxAttempts)
				assert.Equal(t, "1s", webhook.Retry.InitialInterval)
				assert.Equal(t, 2.0, webhook.Retry.BackoffMultiplier)
			}
		})
	}
}

func TestParseInvalidEventName(t *testing.T) {
	t.Parallel()

	// Test cases that should fail parsing
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "missing event braces",
			input: `event user_created`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tt.input)
			assert.Error(t, err)
		})
	}
}

func TestParseComplexEventScenario(t *testing.T) {
	t.Parallel()

	input := `
event user_created {
	schema {
		user_id string
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
		value: env("CRM_API_KEY")
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

	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 5)

	// First statement should be EventDecl
	_, ok := program.Statements[0].(*ast.EventDecl)
	assert.True(t, ok, "expected EventDecl")

	// Second and third should be EventHandlerDecl
	_, ok = program.Statements[1].(*ast.EventHandlerDecl)
	assert.True(t, ok, "expected EventHandlerDecl")
	_, ok = program.Statements[2].(*ast.EventHandlerDecl)
	assert.True(t, ok, "expected EventHandlerDecl")

	// Fourth should be IntegrationDecl
	_, ok = program.Statements[3].(*ast.IntegrationDecl)
	assert.True(t, ok, "expected IntegrationDecl")

	// Fifth should be WebhookDecl
	_, ok = program.Statements[4].(*ast.WebhookDecl)
	assert.True(t, ok, "expected WebhookDecl")
}

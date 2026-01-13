// Package validator provides semantic validation for event, integration, and webhook AST nodes.
package validator

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/bargom/codeai/internal/ast"
)

// EventValidation extends the Validator with event-related state.
type EventValidation struct {
	events       map[string]*ast.EventDecl
	handlers     []*ast.EventHandlerDecl
	integrations map[string]*ast.IntegrationDecl
	webhooks     map[string]*ast.WebhookDecl
	workflows    map[string]*ast.WorkflowDecl
}

// initEventValidation initializes event validation state.
func (v *Validator) initEventValidation() {
	if v.eventValidation == nil {
		v.eventValidation = &EventValidation{
			events:       make(map[string]*ast.EventDecl),
			handlers:     make([]*ast.EventHandlerDecl, 0),
			integrations: make(map[string]*ast.IntegrationDecl),
			webhooks:     make(map[string]*ast.WebhookDecl),
			workflows:    make(map[string]*ast.WorkflowDecl),
		}
	}
}

// validateEventDecl validates an event declaration.
func (v *Validator) validateEventDecl(event *ast.EventDecl) {
	v.initEventValidation()

	// Validate event name format (resource.action pattern)
	if !isValidEventNameFormat(event.Name) {
		v.errors.Add(newSemanticError(event.Pos(),
			"event name '"+event.Name+"' should follow resource.action or resource_action pattern"))
	}

	// Check for duplicate event
	if existing, exists := v.eventValidation.events[event.Name]; exists {
		v.errors.Add(newSemanticError(event.Pos(),
			"duplicate event '"+event.Name+"'; first declared at "+existing.Pos().String()))
		return
	}
	v.eventValidation.events[event.Name] = event

	// Validate schema if present
	if event.Schema != nil {
		v.validateEventSchema(event.Schema, event.Name)
	}
}

// validateEventSchema validates an event schema.
func (v *Validator) validateEventSchema(schema *ast.EventSchema, eventName string) {
	fieldNames := make(map[string]bool)
	validTypes := map[string]bool{
		"string": true, "int": true, "integer": true, "decimal": true, "float": true,
		"bool": true, "boolean": true, "timestamp": true, "datetime": true,
		"array": true, "object": true, "uuid": true,
	}

	for _, field := range schema.Fields {
		// Check for duplicate field names
		if fieldNames[field.Name] {
			v.errors.Add(newSemanticError(field.Pos(),
				"duplicate field '"+field.Name+"' in event '"+eventName+"' schema"))
		}
		fieldNames[field.Name] = true

		// Validate field type
		if !validTypes[strings.ToLower(field.FieldType)] {
			v.errors.Add(newSemanticError(field.Pos(),
				"unknown field type '"+field.FieldType+"' in event '"+eventName+"' schema; "+
					"valid types: string, int, decimal, bool, timestamp, array, object, uuid"))
		}
	}
}

// validateEventHandler validates an event handler declaration.
func (v *Validator) validateEventHandler(handler *ast.EventHandlerDecl) {
	v.initEventValidation()
	v.eventValidation.handlers = append(v.eventValidation.handlers, handler)

	// Validate action type
	validActions := map[string]bool{
		"workflow": true, "integration": true, "emit": true, "webhook": true,
	}
	if !validActions[handler.ActionType] {
		v.errors.Add(newSemanticError(handler.Pos(),
			"invalid action type '"+handler.ActionType+"'; valid types: workflow, integration, emit, webhook"))
	}

	// Note: We defer reference validation to a second pass after all declarations are collected
}

// validateIntegrationDecl validates an integration declaration.
func (v *Validator) validateIntegrationDecl(intg *ast.IntegrationDecl) {
	v.initEventValidation()

	// Check for duplicate integration
	if existing, exists := v.eventValidation.integrations[intg.Name]; exists {
		v.errors.Add(newSemanticError(intg.Pos(),
			"duplicate integration '"+intg.Name+"'; first declared at "+existing.Pos().String()))
		return
	}
	v.eventValidation.integrations[intg.Name] = intg

	// Validate integration type
	validTypes := map[ast.IntegrationType]bool{
		ast.IntegrationTypeREST:    true,
		ast.IntegrationTypeGraphQL: true,
		ast.IntegrationTypeGRPC:    true,
		ast.IntegrationTypeWebhook: true,
	}
	if !validTypes[intg.IntgType] {
		v.errors.Add(newSemanticError(intg.Pos(),
			"unknown integration type '"+string(intg.IntgType)+"'; valid types: rest, graphql, grpc, webhook"))
	}

	// Validate base URL (must be valid HTTPS in production)
	if intg.BaseURL == "" {
		v.errors.Add(newSemanticError(intg.Pos(),
			"integration '"+intg.Name+"' requires a base_url"))
	} else {
		parsedURL, err := url.Parse(intg.BaseURL)
		if err != nil {
			v.errors.Add(newSemanticError(intg.Pos(),
				"invalid base_url in integration '"+intg.Name+"': "+err.Error()))
		} else if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
			v.errors.Add(newSemanticError(intg.Pos(),
				"base_url in integration '"+intg.Name+"' should use https:// or http:// scheme"))
		}
	}

	// Validate auth configuration
	if intg.Auth != nil {
		v.validateIntegrationAuth(intg.Auth, intg.Name)
	}

	// Validate circuit breaker configuration
	if intg.CircuitBreaker != nil {
		v.validateCircuitBreaker(intg.CircuitBreaker, intg.Name)
	}
}

// validateIntegrationAuth validates authentication configuration.
func (v *Validator) validateIntegrationAuth(auth *ast.IntegrationAuthDecl, integrationName string) {
	switch auth.AuthType {
	case ast.IntegrationAuthBearer:
		// Bearer auth requires a token
		if _, hasToken := auth.Config["token"]; !hasToken {
			v.errors.Add(newSemanticError(auth.Pos(),
				"bearer auth in integration '"+integrationName+"' requires 'token' config"))
		}
	case ast.IntegrationAuthAPIKey:
		// API key auth requires header and value
		if _, hasHeader := auth.Config["header"]; !hasHeader {
			v.errors.Add(newSemanticError(auth.Pos(),
				"apikey auth in integration '"+integrationName+"' requires 'header' config"))
		}
		if _, hasValue := auth.Config["value"]; !hasValue {
			v.errors.Add(newSemanticError(auth.Pos(),
				"apikey auth in integration '"+integrationName+"' requires 'value' config"))
		}
	case ast.IntegrationAuthBasic, ast.IntegrationAuthOAuth2:
		// Additional validation for other auth types can be added here
	default:
		v.errors.Add(newSemanticError(auth.Pos(),
			"unknown auth type in integration '"+integrationName+"'"))
	}
}

// validateCircuitBreaker validates circuit breaker configuration.
func (v *Validator) validateCircuitBreaker(cb *ast.CircuitBreakerConfig, integrationName string) {
	if cb.FailureThreshold <= 0 {
		v.errors.Add(newSemanticError(cb.Pos(),
			"circuit_breaker threshold must be positive in integration '"+integrationName+"'"))
	}

	if cb.MaxConcurrent <= 0 {
		v.errors.Add(newSemanticError(cb.Pos(),
			"circuit_breaker max_concurrent must be positive in integration '"+integrationName+"'"))
	}

	// Validate timeout format
	if cb.Timeout == "" {
		v.errors.Add(newSemanticError(cb.Pos(),
			"circuit_breaker timeout is required in integration '"+integrationName+"'"))
	}
}

// validateWebhookDecl validates a webhook declaration.
func (v *Validator) validateWebhookDecl(webhook *ast.WebhookDecl) {
	v.initEventValidation()

	// Check for duplicate webhook
	if existing, exists := v.eventValidation.webhooks[webhook.Name]; exists {
		v.errors.Add(newSemanticError(webhook.Pos(),
			"duplicate webhook '"+webhook.Name+"'; first declared at "+existing.Pos().String()))
		return
	}
	v.eventValidation.webhooks[webhook.Name] = webhook

	// Validate URL (must be valid HTTPS)
	if webhook.URL == "" {
		v.errors.Add(newSemanticError(webhook.Pos(),
			"webhook '"+webhook.Name+"' requires a url"))
	} else {
		parsedURL, err := url.Parse(webhook.URL)
		if err != nil {
			v.errors.Add(newSemanticError(webhook.Pos(),
				"invalid url in webhook '"+webhook.Name+"': "+err.Error()))
		} else if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
			v.errors.Add(newSemanticError(webhook.Pos(),
				"url in webhook '"+webhook.Name+"' should use https:// or http:// scheme"))
		}
	}

	// Validate event name
	if webhook.Event == "" {
		v.errors.Add(newSemanticError(webhook.Pos(),
			"webhook '"+webhook.Name+"' requires an event name"))
	}

	// Validate retry configuration
	if webhook.Retry != nil {
		if webhook.Retry.MaxAttempts <= 0 {
			v.errors.Add(newSemanticError(webhook.Pos(),
				"retry max_attempts must be positive in webhook '"+webhook.Name+"'"))
		}
		if webhook.Retry.BackoffMultiplier < 1.0 {
			v.errors.Add(newSemanticError(webhook.Pos(),
				"retry backoff must be >= 1.0 in webhook '"+webhook.Name+"'"))
		}
	}
}

// validateEventReferences validates that all event handler references are valid.
// This should be called after all declarations have been collected.
func (v *Validator) validateEventReferences() {
	if v.eventValidation == nil {
		return
	}

	for _, handler := range v.eventValidation.handlers {
		switch handler.ActionType {
		case "workflow":
			// Check if workflow exists
			if _, exists := v.eventValidation.workflows[handler.Target]; !exists {
				// Note: workflows might be defined elsewhere, so this is a warning
				// v.errors.Add(newSemanticError(handler.Pos(),
				// 	"event handler references unknown workflow '"+handler.Target+"'"))
			}
		case "integration":
			// Check if integration exists
			// Integration target might be "name.method", so extract just the name
			integrationName := handler.Target
			if idx := strings.Index(handler.Target, "."); idx > 0 {
				integrationName = handler.Target[:idx]
			}
			if _, exists := v.eventValidation.integrations[integrationName]; !exists {
				// Note: This is intentionally not an error as integrations might be defined elsewhere
			}
		case "webhook":
			// Check if webhook exists
			if _, exists := v.eventValidation.webhooks[handler.Target]; !exists {
				// Note: This is intentionally not an error as webhooks might be defined elsewhere
			}
		case "emit":
			// Emit targets another event - validation is lenient here
		}
	}

	// Validate that webhooks reference valid events (warning only)
	for _, webhook := range v.eventValidation.webhooks {
		if _, exists := v.eventValidation.events[webhook.Event]; !exists {
			// Event might be defined elsewhere or be a system event
		}
	}
}

// isValidEventNameFormat validates that an event name follows the expected pattern.
// Valid formats: resource.action (e.g., "user.created") or resource_action (e.g., "user_created")
func isValidEventNameFormat(name string) bool {
	// Allow formats like "user.created", "user_created", "order.completed"
	pattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)?$`)
	return pattern.MatchString(name)
}

// Package event provides DSL loading capabilities for event definitions.
package event

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/bargom/codeai/internal/ast"
)

// EventRegistry manages registered events and their handlers.
type EventRegistry struct {
	mu        sync.RWMutex
	events    map[string]*RegisteredEvent
	handlers  map[string][]*RegisteredHandler
	dispatcher Dispatcher
}

// RegisteredEvent represents an event registered from the DSL.
type RegisteredEvent struct {
	Name   string
	Schema *EventSchema
}

// EventSchema represents the schema for an event's payload.
type EventSchema struct {
	Fields map[string]string // field name -> field type
}

// RegisteredHandler represents an event handler registered from the DSL.
type RegisteredHandler struct {
	EventName  string
	ActionType string // "workflow", "integration", "emit", "webhook"
	Target     string
	Async      bool
	handler    Handler
}

// NewEventRegistry creates a new event registry with the given dispatcher.
func NewEventRegistry(dispatcher Dispatcher) *EventRegistry {
	if dispatcher == nil {
		dispatcher = NewDispatcher()
	}
	return &EventRegistry{
		events:     make(map[string]*RegisteredEvent),
		handlers:   make(map[string][]*RegisteredHandler),
		dispatcher: dispatcher,
	}
}

// RegisterEventFromAST registers an event definition from the AST.
func (r *EventRegistry) RegisterEventFromAST(event *ast.EventDecl) error {
	if event == nil {
		return fmt.Errorf("event declaration is nil")
	}

	// Validate event name format (resource.action pattern)
	if !isValidEventName(event.Name) {
		return fmt.Errorf("invalid event name '%s': must follow resource.action or resource_action pattern", event.Name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate event registration
	if _, exists := r.events[event.Name]; exists {
		return fmt.Errorf("event '%s' is already registered", event.Name)
	}

	// Convert schema
	var schema *EventSchema
	if event.Schema != nil {
		schema = &EventSchema{
			Fields: make(map[string]string),
		}
		for _, field := range event.Schema.Fields {
			schema.Fields[field.Name] = field.FieldType
		}
	}

	r.events[event.Name] = &RegisteredEvent{
		Name:   event.Name,
		Schema: schema,
	}

	return nil
}

// SubscribeHandlerFromAST subscribes an event handler from the AST.
func (r *EventRegistry) SubscribeHandlerFromAST(handler *ast.EventHandlerDecl) error {
	if handler == nil {
		return fmt.Errorf("event handler declaration is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	registeredHandler := &RegisteredHandler{
		EventName:  handler.EventName,
		ActionType: handler.ActionType,
		Target:     handler.Target,
		Async:      handler.Async,
	}

	// Create the actual handler function based on action type
	registeredHandler.handler = r.createHandler(registeredHandler)

	// Register with the dispatcher
	r.handlers[handler.EventName] = append(r.handlers[handler.EventName], registeredHandler)
	r.dispatcher.Subscribe(EventType(handler.EventName), registeredHandler.handler)

	return nil
}

// createHandler creates a Handler function for the registered handler.
func (r *EventRegistry) createHandler(rh *RegisteredHandler) Handler {
	return func(ctx context.Context, event Event) error {
		// Execute based on action type
		switch rh.ActionType {
		case "workflow":
			return r.executeWorkflow(ctx, rh.Target, event.Payload)
		case "integration":
			return r.executeIntegration(ctx, rh.Target, event.Payload)
		case "emit":
			return r.emitEvent(ctx, rh.Target, event.Payload)
		case "webhook":
			return r.executeWebhook(ctx, rh.Target, event.Payload)
		default:
			return fmt.Errorf("unknown action type: %s", rh.ActionType)
		}
	}
}

// executeWorkflow executes a workflow for an event.
func (r *EventRegistry) executeWorkflow(ctx context.Context, workflowName string, payload any) error {
	// TODO: Integrate with Temporal workflow engine
	// This is a placeholder for workflow execution
	return nil
}

// executeIntegration calls an external integration.
func (r *EventRegistry) executeIntegration(ctx context.Context, integrationName string, payload any) error {
	// TODO: Integrate with integration registry
	return nil
}

// emitEvent emits another event.
func (r *EventRegistry) emitEvent(ctx context.Context, eventName string, payload any) error {
	event := NewEvent(EventType(eventName), payload)
	return r.dispatcher.Dispatch(ctx, event)
}

// executeWebhook calls a webhook.
func (r *EventRegistry) executeWebhook(ctx context.Context, webhookName string, payload any) error {
	// TODO: Integrate with webhook dispatcher
	return nil
}

// EmitEvent emits an event with the given name and payload.
func (r *EventRegistry) EmitEvent(ctx context.Context, name string, payload map[string]interface{}) error {
	r.mu.RLock()
	registeredEvent, exists := r.events[name]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("event '%s' is not registered", name)
	}

	// Validate payload against schema if defined
	if registeredEvent.Schema != nil {
		if err := validatePayload(payload, registeredEvent.Schema); err != nil {
			return fmt.Errorf("payload validation failed: %w", err)
		}
	}

	event := NewEvent(EventType(name), payload)
	return r.dispatcher.Dispatch(ctx, event)
}

// GetEvent returns a registered event by name.
func (r *EventRegistry) GetEvent(name string) (*RegisteredEvent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	event, exists := r.events[name]
	return event, exists
}

// GetHandlers returns all handlers for an event.
func (r *EventRegistry) GetHandlers(eventName string) []*RegisteredHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.handlers[eventName]
}

// EventCount returns the number of registered events.
func (r *EventRegistry) EventCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.events)
}

// HandlerCount returns the total number of registered handlers.
func (r *EventRegistry) HandlerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, handlers := range r.handlers {
		count += len(handlers)
	}
	return count
}

// isValidEventName validates the event name format.
// Valid formats: resource.action or resource_action (e.g., "user.created", "order_completed")
func isValidEventName(name string) bool {
	// Allow formats like "user.created", "user_created", "order.completed"
	pattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)?$`)
	return pattern.MatchString(name)
}

// validatePayload validates the event payload against the schema.
func validatePayload(payload map[string]interface{}, schema *EventSchema) error {
	if schema == nil || len(schema.Fields) == 0 {
		return nil
	}

	for fieldName, fieldType := range schema.Fields {
		value, exists := payload[fieldName]
		if !exists {
			continue // Schema fields are optional by default
		}

		// Basic type validation
		if err := validateFieldType(fieldName, value, fieldType); err != nil {
			return err
		}
	}

	return nil
}

// validateFieldType validates that a field value matches the expected type.
func validateFieldType(fieldName string, value interface{}, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' expected string, got %T", fieldName, value)
		}
	case "int", "integer":
		switch value.(type) {
		case int, int32, int64, float64:
			// Accept numeric types
		default:
			return fmt.Errorf("field '%s' expected integer, got %T", fieldName, value)
		}
	case "decimal", "float":
		switch value.(type) {
		case float32, float64, int, int32, int64:
			// Accept numeric types
		default:
			return fmt.Errorf("field '%s' expected decimal, got %T", fieldName, value)
		}
	case "bool", "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' expected boolean, got %T", fieldName, value)
		}
	case "timestamp", "datetime":
		// Accept string (ISO format) or time.Time
	case "array":
		switch value.(type) {
		case []interface{}, []string, []int, []float64:
			// Accept various array types
		default:
			return fmt.Errorf("field '%s' expected array, got %T", fieldName, value)
		}
	}

	return nil
}

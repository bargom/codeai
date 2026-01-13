package codegen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"github.com/bargom/codeai/internal/event"
	"github.com/bargom/codeai/internal/integration"
)

// ExecutionContextFactory creates execution contexts for handlers.
type ExecutionContextFactory struct {
	generatedCode *GeneratedCode
	transforms    map[string]TransformFunc
	mu            sync.RWMutex
}

// TransformFunc is a function that transforms data.
type TransformFunc func(data interface{}) (interface{}, error)

// NewExecutionContextFactory creates a new factory.
func NewExecutionContextFactory(code *GeneratedCode) *ExecutionContextFactory {
	return &ExecutionContextFactory{
		generatedCode: code,
		transforms:    make(map[string]TransformFunc),
	}
}

// RegisterTransform registers a named transformation function.
func (f *ExecutionContextFactory) RegisterTransform(name string, fn TransformFunc) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.transforms[name] = fn
}

// NewContext creates a new execution context for a request.
func (f *ExecutionContextFactory) NewContext(ctx context.Context, r *http.Request) *ExecutionContext {
	return &ExecutionContext{
		ctx:           ctx,
		request:       r,
		data:          make(map[string]interface{}),
		generatedCode: f.generatedCode,
		transforms:    f.transforms,
		logger:        slog.Default(),
	}
}

// ExecutionContext holds the runtime state for executing handler logic.
type ExecutionContext struct {
	ctx           context.Context
	request       *http.Request
	input         interface{}
	result        interface{}
	data          map[string]interface{}
	claims        map[string]interface{}
	generatedCode *GeneratedCode
	transforms    map[string]TransformFunc
	logger        *slog.Logger
	mu            sync.RWMutex
}

// Context returns the Go context.
func (c *ExecutionContext) Context() context.Context {
	return c.ctx
}

// Request returns the HTTP request.
func (c *ExecutionContext) Request() *http.Request {
	return c.request
}

// SetInput sets the parsed request input.
func (c *ExecutionContext) SetInput(input interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.input = input
}

// Input returns the parsed request input.
func (c *ExecutionContext) Input() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.input
}

// SetResult sets the handler result.
func (c *ExecutionContext) SetResult(result interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.result = result
}

// Result returns the handler result.
func (c *ExecutionContext) Result() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.result
}

// Set stores a value in the context data store.
func (c *ExecutionContext) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// Get retrieves a value from the context data store.
func (c *ExecutionContext) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// First check data store
	if val, ok := c.data[key]; ok {
		return val
	}

	// Then check input
	if c.input != nil {
		if inputMap, ok := c.input.(map[string]interface{}); ok {
			if val, ok := inputMap[key]; ok {
				return val
			}
		}
	}

	return nil
}

// SetClaims sets the JWT claims from authentication.
func (c *ExecutionContext) SetClaims(claims map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.claims = claims
}

// Claims returns the JWT claims.
func (c *ExecutionContext) Claims() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.claims
}

// Data returns all stored data.
func (c *ExecutionContext) Data() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Make a copy
	result := make(map[string]interface{}, len(c.data))
	for k, v := range c.data {
		result[k] = v
	}
	return result
}

// QueryDatabase executes a database query returning multiple results.
func (c *ExecutionContext) QueryDatabase(table string, conditions map[string]interface{}) (interface{}, error) {
	c.logger.Debug("query database", "table", table, "conditions", conditions)

	// TODO: Integrate with actual database layer
	// For now, return mock data for development/testing
	return []map[string]interface{}{}, nil
}

// FindOne executes a database query returning a single result.
func (c *ExecutionContext) FindOne(table string, id interface{}) (interface{}, error) {
	c.logger.Debug("find one", "table", table, "id", id)

	// TODO: Integrate with actual database layer
	return nil, nil
}

// InsertDatabase inserts a record into the database.
func (c *ExecutionContext) InsertDatabase(table string, data interface{}) (interface{}, error) {
	c.logger.Debug("insert database", "table", table)

	// TODO: Integrate with actual database layer
	// Return mock ID for now
	return "mock-id-123", nil
}

// UpdateDatabase updates a record in the database.
func (c *ExecutionContext) UpdateDatabase(table string, id interface{}, data interface{}) error {
	c.logger.Debug("update database", "table", table, "id", id)

	// TODO: Integrate with actual database layer
	return nil
}

// DeleteDatabase deletes a record from the database.
func (c *ExecutionContext) DeleteDatabase(table string, id interface{}) error {
	c.logger.Debug("delete database", "table", table, "id", id)

	// TODO: Integrate with actual database layer
	return nil
}

// Transform applies a named transformation to data.
func (c *ExecutionContext) Transform(name string, data interface{}) (interface{}, error) {
	// Look up registered transform
	if fn, ok := c.transforms[name]; ok {
		return fn(data)
	}

	// Built-in transforms
	switch name {
	case "toJSON":
		return c.transformToJSON(data)
	case "fromJSON":
		return c.transformFromJSON(data)
	case "pick":
		return c.transformPick(data)
	case "omit":
		return c.transformOmit(data)
	default:
		// No transform found, return data as-is
		return data, nil
	}
}

func (c *ExecutionContext) transformToJSON(data interface{}) (interface{}, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

func (c *ExecutionContext) transformFromJSON(data interface{}) (interface{}, error) {
	str, ok := data.(string)
	if !ok {
		return data, nil
	}

	var result interface{}
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ExecutionContext) transformPick(data interface{}) (interface{}, error) {
	// TODO: Implement field picking
	return data, nil
}

func (c *ExecutionContext) transformOmit(data interface{}) (interface{}, error) {
	// TODO: Implement field omitting
	return data, nil
}

// EmitEvent emits an event to the event bus.
func (c *ExecutionContext) EmitEvent(name string, payload interface{}) error {
	c.logger.Debug("emit event", "name", name)

	if c.generatedCode.EventHandlers == nil {
		return nil
	}

	// Convert payload to map if needed
	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		// Try to convert via JSON
		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("converting payload: %w", err)
		}
		if err := json.Unmarshal(jsonBytes, &payloadMap); err != nil {
			// Use original payload wrapped
			payloadMap = map[string]interface{}{"data": payload}
		}
	}

	return c.generatedCode.EventHandlers.EmitEvent(c.ctx, name, payloadMap)
}

// CallIntegration calls an external integration.
func (c *ExecutionContext) CallIntegration(name, method, path string, body interface{}) (interface{}, error) {
	c.logger.Debug("call integration", "name", name, "method", method, "path", path)

	if c.generatedCode.Integrations == nil {
		return nil, fmt.Errorf("no integrations configured")
	}

	client, ok := c.generatedCode.Integrations.GetClient(name)
	if !ok {
		return nil, fmt.Errorf("integration %q not found", name)
	}

	// Prepare body reader
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	// Execute request
	resp, err := client.Do(c.ctx, method, path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("integration request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Parse JSON response
	var result interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			// Return raw string if not JSON
			return string(respBody), nil
		}
	}

	return result, nil
}

// CacheGet retrieves a value from cache.
func (c *ExecutionContext) CacheGet(key string) (interface{}, error) {
	c.logger.Debug("cache get", "key", key)
	// TODO: Integrate with Redis cache
	return nil, nil
}

// CacheSet stores a value in cache.
func (c *ExecutionContext) CacheSet(key string, value interface{}, ttlSeconds int) error {
	c.logger.Debug("cache set", "key", key, "ttl", ttlSeconds)
	// TODO: Integrate with Redis cache
	return nil
}

// StartWorkflow starts a Temporal workflow.
func (c *ExecutionContext) StartWorkflow(name string, input map[string]interface{}) (string, error) {
	c.logger.Debug("start workflow", "name", name)

	if c.generatedCode.Workflows == nil {
		return "", fmt.Errorf("no workflows configured")
	}

	_, ok := c.generatedCode.Workflows.Get(name)
	if !ok {
		return "", fmt.Errorf("workflow %q not found", name)
	}

	// TODO: Integrate with Temporal client
	return "workflow-run-id-mock", nil
}

// Dispatcher interface for event dispatching.
type Dispatcher interface {
	Dispatch(ctx context.Context, event event.Event) error
	Subscribe(eventType event.EventType, handler event.Handler)
	Unsubscribe(eventType event.EventType, handler event.Handler)
}

// NewDispatcher creates a new in-memory event dispatcher.
func NewDispatcher() Dispatcher {
	return &inMemoryDispatcher{
		handlers: make(map[event.EventType][]event.Handler),
	}
}

type inMemoryDispatcher struct {
	mu       sync.RWMutex
	handlers map[event.EventType][]event.Handler
}

func (d *inMemoryDispatcher) Dispatch(ctx context.Context, evt event.Event) error {
	d.mu.RLock()
	handlers := d.handlers[evt.Type]
	d.mu.RUnlock()

	for _, h := range handlers {
		if err := h(ctx, evt); err != nil {
			slog.Error("event handler failed", "event", evt.Type, "error", err)
		}
	}

	return nil
}

func (d *inMemoryDispatcher) Subscribe(eventType event.EventType, handler event.Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[eventType] = append(d.handlers[eventType], handler)
}

func (d *inMemoryDispatcher) Unsubscribe(eventType event.EventType, handler event.Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	// Note: Simple implementation, doesn't remove specific handler
	// In production, would need to compare function pointers
}

// DatabaseAdapter provides an interface for database operations.
type DatabaseAdapter interface {
	Query(ctx context.Context, table string, conditions map[string]interface{}) ([]map[string]interface{}, error)
	FindOne(ctx context.Context, table string, id interface{}) (map[string]interface{}, error)
	Insert(ctx context.Context, table string, data interface{}) (interface{}, error)
	Update(ctx context.Context, table string, id interface{}, data interface{}) error
	Delete(ctx context.Context, table string, id interface{}) error
}

// CacheAdapter provides an interface for cache operations.
type CacheAdapter interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl int) error
	Delete(ctx context.Context, key string) error
}

// IntegrationClient wraps the integration client for execution context.
type IntegrationClient struct {
	client *integration.Client
}

// NewIntegrationClient creates a new integration client wrapper.
func NewIntegrationClient(client *integration.Client) *IntegrationClient {
	return &IntegrationClient{client: client}
}

// Do executes an HTTP request through the integration.
func (c *IntegrationClient) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	return c.client.Do(ctx, method, path, body)
}

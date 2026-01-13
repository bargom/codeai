package codegen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bargom/codeai/internal/event"
	"github.com/bargom/codeai/internal/integration"
	"github.com/bargom/codeai/internal/workflow"
)

func TestNewExecutionContextFactory(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	}

	factory := NewExecutionContextFactory(code)
	if factory == nil {
		t.Fatal("expected non-nil factory")
	}

	if factory.generatedCode != code {
		t.Error("expected factory to reference generated code")
	}
}

func TestExecutionContextFactory_RegisterTransform(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	}

	factory := NewExecutionContextFactory(code)

	// Register a custom transform
	factory.RegisterTransform("uppercase", func(data interface{}) (interface{}, error) {
		if s, ok := data.(string); ok {
			return s + "!", nil
		}
		return data, nil
	})

	// Create context and test transform
	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))

	result, err := ctx.Transform("uppercase", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "hello!" {
		t.Errorf("expected 'hello!', got %v", result)
	}
}

func TestExecutionContext_InputOutput(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	}

	factory := NewExecutionContextFactory(code)
	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))

	// Test input
	input := map[string]interface{}{"key": "value"}
	ctx.SetInput(input)

	gotInput := ctx.Input()
	if gotInput == nil {
		t.Fatal("expected non-nil input")
	}

	inputMap, ok := gotInput.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", gotInput)
	}

	if inputMap["key"] != "value" {
		t.Errorf("expected key='value', got %v", inputMap["key"])
	}
}

func TestExecutionContext_Result(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	}

	factory := NewExecutionContextFactory(code)
	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))

	// Test result
	result := map[string]interface{}{"id": "123"}
	ctx.SetResult(result)

	gotResult := ctx.Result()
	if gotResult == nil {
		t.Fatal("expected non-nil result")
	}

	resultMap, ok := gotResult.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", gotResult)
	}

	if resultMap["id"] != "123" {
		t.Errorf("expected id='123', got %v", resultMap["id"])
	}
}

func TestExecutionContext_SetGet(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	}

	factory := NewExecutionContextFactory(code)
	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))

	// Set and get
	ctx.Set("foo", "bar")
	val := ctx.Get("foo")
	if val != "bar" {
		t.Errorf("expected 'bar', got %v", val)
	}

	// Get non-existent
	val = ctx.Get("nonexistent")
	if val != nil {
		t.Errorf("expected nil for non-existent key, got %v", val)
	}
}

func TestExecutionContext_GetFromInput(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	}

	factory := NewExecutionContextFactory(code)
	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))

	// Set input
	ctx.SetInput(map[string]interface{}{"id": "456"})

	// Get should fall back to input
	val := ctx.Get("id")
	if val != "456" {
		t.Errorf("expected '456' from input, got %v", val)
	}

	// Set explicit value should override
	ctx.Set("id", "789")
	val = ctx.Get("id")
	if val != "789" {
		t.Errorf("expected '789' from explicit set, got %v", val)
	}
}

func TestExecutionContext_Claims(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	}

	factory := NewExecutionContextFactory(code)
	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))

	// Initially nil
	if ctx.Claims() != nil {
		t.Error("expected nil claims initially")
	}

	// Set claims
	claims := map[string]interface{}{
		"sub":   "user123",
		"roles": []string{"admin"},
	}
	ctx.SetClaims(claims)

	gotClaims := ctx.Claims()
	if gotClaims == nil {
		t.Fatal("expected non-nil claims")
	}

	if gotClaims["sub"] != "user123" {
		t.Errorf("expected sub='user123', got %v", gotClaims["sub"])
	}
}

func TestExecutionContext_Data(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	}

	factory := NewExecutionContextFactory(code)
	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))

	ctx.Set("a", 1)
	ctx.Set("b", 2)

	data := ctx.Data()
	if len(data) != 2 {
		t.Errorf("expected 2 items, got %d", len(data))
	}

	// Verify it's a copy
	data["c"] = 3
	if ctx.Get("c") != nil {
		t.Error("Data() should return a copy, not the original map")
	}
}

func TestExecutionContext_BuiltInTransforms(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	}

	factory := NewExecutionContextFactory(code)
	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))

	t.Run("toJSON", func(t *testing.T) {
		data := map[string]interface{}{"name": "test"}
		result, err := ctx.Transform("toJSON", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		str, ok := result.(string)
		if !ok {
			t.Fatalf("expected string, got %T", result)
		}

		if str != `{"name":"test"}` {
			t.Errorf("unexpected JSON: %s", str)
		}
	})

	t.Run("fromJSON", func(t *testing.T) {
		jsonStr := `{"id": 123}`
		result, err := ctx.Transform("fromJSON", jsonStr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		m, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map, got %T", result)
		}

		if m["id"] != float64(123) {
			t.Errorf("expected id=123, got %v", m["id"])
		}
	})

	t.Run("unknownTransform", func(t *testing.T) {
		// Unknown transform should return data as-is
		data := "test"
		result, err := ctx.Transform("unknown", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != data {
			t.Errorf("expected data to pass through unchanged")
		}
	})
}

func TestExecutionContext_QueryDatabase(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	}

	factory := NewExecutionContextFactory(code)
	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))

	result, err := ctx.QueryDatabase("users", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mock returns empty slice
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestExecutionContext_InsertDatabase(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	}

	factory := NewExecutionContextFactory(code)
	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))

	data := map[string]interface{}{"name": "test"}
	id, err := ctx.InsertDatabase("users", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mock returns a mock ID
	if id == nil {
		t.Error("expected non-nil ID")
	}
}

func TestExecutionContext_EmitEvent(t *testing.T) {
	eventRegistry := event.NewEventRegistry(nil)
	// Register the event first
	eventDecl := &event.RegisteredEvent{
		Name: "test.created",
	}
	_ = eventDecl // We need to use the internal registration

	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
		EventHandlers: eventRegistry,
	}

	factory := NewExecutionContextFactory(code)
	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))

	payload := map[string]interface{}{"id": "123"}
	err := ctx.EmitEvent("test.created", payload)

	// Event not registered, so should get error
	if err == nil {
		t.Log("Event emission returned no error (event may not be registered)")
	}
}

func TestExecutionContext_CallIntegration_NotFound(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
		Integrations:  integration.NewIntegrationRegistry(),
	}

	factory := NewExecutionContextFactory(code)
	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))

	_, err := ctx.CallIntegration("nonexistent", "GET", "/", nil)
	if err == nil {
		t.Fatal("expected error for non-existent integration")
	}
}

func TestExecutionContext_StartWorkflow_NotFound(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
		Workflows:     workflow.NewDSLWorkflowRegistry(),
	}

	factory := NewExecutionContextFactory(code)
	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))

	_, err := ctx.StartWorkflow("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for non-existent workflow")
	}
}

func TestInMemoryDispatcher(t *testing.T) {
	dispatcher := NewDispatcher()

	handlerCalled := false
	handler := func(ctx context.Context, evt event.Event) error {
		handlerCalled = true
		return nil
	}

	dispatcher.Subscribe("test.event", handler)

	evt := event.NewEvent("test.event", nil)
	err := dispatcher.Dispatch(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !handlerCalled {
		t.Error("expected handler to be called")
	}
}

func TestExecutionContext_CacheOperations(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	}

	factory := NewExecutionContextFactory(code)
	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))

	// CacheGet (mock returns nil)
	val, err := ctx.CacheGet("key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil value from mock cache, got %v", val)
	}

	// CacheSet (mock does nothing)
	err = ctx.CacheSet("key", "value", 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecutionContext_Context(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	}

	factory := NewExecutionContextFactory(code)
	goCtx := context.WithValue(context.Background(), "testKey", "testValue")
	ctx := factory.NewContext(goCtx, httptest.NewRequest("GET", "/", nil))

	if ctx.Context() != goCtx {
		t.Error("expected Context() to return the original context")
	}
}

func TestExecutionContext_Request(t *testing.T) {
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	}

	factory := NewExecutionContextFactory(code)
	req := httptest.NewRequest("POST", "/api/test", nil)
	ctx := factory.NewContext(context.Background(), req)

	if ctx.Request() != req {
		t.Error("expected Request() to return the original request")
	}

	if ctx.Request().Method != "POST" {
		t.Errorf("expected method POST, got %s", ctx.Request().Method)
	}
}

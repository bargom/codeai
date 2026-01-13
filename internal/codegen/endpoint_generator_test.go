package codegen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bargom/codeai/internal/ast"
)

func TestExtractRequestData_Body(t *testing.T) {
	body := `{"name": "test", "value": 123}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	requestType := &ast.RequestType{
		TypeName: "TestInput",
		Source:   ast.RequestSourceBody,
	}

	data, err := extractRequestData(req, requestType)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data["name"] != "test" {
		t.Errorf("expected name='test', got %v", data["name"])
	}
}

func TestExtractRequestData_Query(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?name=foo&count=5", nil)

	requestType := &ast.RequestType{
		TypeName: "TestQuery",
		Source:   ast.RequestSourceQuery,
	}

	data, err := extractRequestData(req, requestType)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data["name"] != "foo" {
		t.Errorf("expected name='foo', got %v", data["name"])
	}
	if data["count"] != "5" {
		t.Errorf("expected count='5', got %v", data["count"])
	}
}

func TestExtractRequestData_Header(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Custom-Header", "custom-value")
	req.Header.Set("Authorization", "Bearer token123")

	requestType := &ast.RequestType{
		TypeName: "TestHeaders",
		Source:   ast.RequestSourceHeader,
	}

	data, err := extractRequestData(req, requestType)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data["X-Custom-Header"] != "custom-value" {
		t.Errorf("expected X-Custom-Header='custom-value', got %v", data["X-Custom-Header"])
	}
}

func TestExtractRequestData_EmptyBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/test", nil)

	requestType := &ast.RequestType{
		TypeName: "TestInput",
		Source:   ast.RequestSourceBody,
	}

	data, err := extractRequestData(req, requestType)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("expected empty data, got %v", data)
	}
}

func TestExtractRequestData_InvalidJSON(t *testing.T) {
	body := `{invalid json`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	requestType := &ast.RequestType{
		TypeName: "TestInput",
		Source:   ast.RequestSourceBody,
	}

	_, err := extractRequestData(req, requestType)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestExecuteValidate(t *testing.T) {
	factory := NewExecutionContextFactory(&GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	})

	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))
	ctx.SetInput(map[string]interface{}{"name": "test"})

	step := &ast.LogicStep{
		Action: "validate",
		Args:   []string{"request"},
	}

	err := executeValidate(ctx, step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteValidate_NoData(t *testing.T) {
	factory := NewExecutionContextFactory(&GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	})

	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))
	// No input set

	step := &ast.LogicStep{
		Action: "validate",
		Args:   []string{"request"},
	}

	err := executeValidate(ctx, step)
	if err == nil {
		t.Fatal("expected error for nil data")
	}

	if _, ok := err.(*ValidationError); !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestExecuteAuthorize_NoClaimsRequired(t *testing.T) {
	factory := NewExecutionContextFactory(&GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	})

	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))

	step := &ast.LogicStep{
		Action: "authorize",
		Args:   []string{"request"}, // No role specified
	}

	err := executeAuthorize(ctx, step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteAuthorize_RoleRequired_NoClaims(t *testing.T) {
	factory := NewExecutionContextFactory(&GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	})

	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))
	// No claims set

	step := &ast.LogicStep{
		Action: "authorize",
		Args:   []string{"request", "admin"},
	}

	err := executeAuthorize(ctx, step)
	if err == nil {
		t.Fatal("expected error for missing claims")
	}

	authErr, ok := err.(*AuthorizationError)
	if !ok {
		t.Fatalf("expected AuthorizationError, got %T", err)
	}
	if authErr.Message != "no authentication claims found" {
		t.Errorf("unexpected error message: %s", authErr.Message)
	}
}

func TestExecuteAuthorize_HasRequiredRole(t *testing.T) {
	factory := NewExecutionContextFactory(&GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	})

	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))
	ctx.SetClaims(map[string]interface{}{
		"roles": []interface{}{"user", "admin"},
	})

	step := &ast.LogicStep{
		Action: "authorize",
		Args:   []string{"request", "admin"},
	}

	err := executeAuthorize(ctx, step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteAuthorize_MissingRole(t *testing.T) {
	factory := NewExecutionContextFactory(&GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	})

	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))
	ctx.SetClaims(map[string]interface{}{
		"roles": []interface{}{"user"},
	})

	step := &ast.LogicStep{
		Action: "authorize",
		Args:   []string{"request", "admin"},
	}

	err := executeAuthorize(ctx, step)
	if err == nil {
		t.Fatal("expected error for missing role")
	}

	authErr, ok := err.(*AuthorizationError)
	if !ok {
		t.Fatalf("expected AuthorizationError, got %T", err)
	}
	if authErr.RequiredRole != "admin" {
		t.Errorf("expected RequiredRole='admin', got %s", authErr.RequiredRole)
	}
}

func TestExecuteAuthorize_SingleRole(t *testing.T) {
	factory := NewExecutionContextFactory(&GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	})

	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))
	ctx.SetClaims(map[string]interface{}{
		"role": "admin", // Single role, not array
	})

	step := &ast.LogicStep{
		Action: "authorize",
		Args:   []string{"request", "admin"},
	}

	err := executeAuthorize(ctx, step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCondition(t *testing.T) {
	factory := NewExecutionContextFactory(&GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: NewTypeRegistry(),
	})

	ctx := factory.NewContext(context.Background(), httptest.NewRequest("GET", "/", nil))
	ctx.SetInput(map[string]interface{}{"user_id": "123"})

	tests := []struct {
		condition string
		expected  map[string]interface{}
	}{
		{
			condition: "id = request.user_id",
			expected:  map[string]interface{}{"id": "123"},
		},
		{
			condition: "status = \"active\"",
			expected:  map[string]interface{}{"status": "active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.condition, func(t *testing.T) {
			result := parseCondition(tt.condition, ctx)
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("expected %s=%v, got %v", k, v, result[k])
				}
			}
		})
	}
}

func TestDetermineErrorStatusCode(t *testing.T) {
	tests := []struct {
		err      error
		expected int
	}{
		{&ValidationError{}, http.StatusBadRequest},
		{&AuthorizationError{}, http.StatusForbidden},
		{&NotFoundError{}, http.StatusNotFound},
		{context.DeadlineExceeded, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		result := determineErrorStatusCode(tt.err)
		if result != tt.expected {
			t.Errorf("determineErrorStatusCode(%T) = %d, want %d", tt.err, result, tt.expected)
		}
	}
}

func TestErrorTypes(t *testing.T) {
	t.Run("ValidationError", func(t *testing.T) {
		err := &ValidationError{Field: "email", Message: "invalid format"}
		msg := err.Error()
		if !strings.Contains(msg, "email") {
			t.Errorf("expected error to mention field 'email': %s", msg)
		}
	})

	t.Run("AuthorizationError with role", func(t *testing.T) {
		err := &AuthorizationError{Message: "denied", RequiredRole: "admin"}
		msg := err.Error()
		if !strings.Contains(msg, "admin") {
			t.Errorf("expected error to mention role 'admin': %s", msg)
		}
	})

	t.Run("AuthorizationError without role", func(t *testing.T) {
		err := &AuthorizationError{Message: "denied"}
		msg := err.Error()
		if strings.Contains(msg, "required role") {
			t.Errorf("expected error to not mention required role: %s", msg)
		}
	})

	t.Run("NotFoundError with ID", func(t *testing.T) {
		err := &NotFoundError{Resource: "User", ID: "123"}
		msg := err.Error()
		if !strings.Contains(msg, "User") || !strings.Contains(msg, "123") {
			t.Errorf("expected error to mention resource and ID: %s", msg)
		}
	})

	t.Run("NotFoundError without ID", func(t *testing.T) {
		err := &NotFoundError{Resource: "User"}
		msg := err.Error()
		if !strings.Contains(msg, "User") {
			t.Errorf("expected error to mention resource: %s", msg)
		}
	})
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"message": "success"}

	writeJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %s", contentType)
	}

	if !strings.Contains(w.Body.String(), "success") {
		t.Errorf("expected body to contain 'success': %s", w.Body.String())
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	writeError(w, http.StatusBadRequest, "invalid input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Bad Request") {
		t.Errorf("expected body to contain status text: %s", body)
	}
	if !strings.Contains(body, "invalid input") {
		t.Errorf("expected body to contain message: %s", body)
	}
}

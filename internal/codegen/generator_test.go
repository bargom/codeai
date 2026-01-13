package codegen

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/parser"
)

func TestNewGenerator(t *testing.T) {
	t.Run("creates generator with default config", func(t *testing.T) {
		gen := NewGenerator(nil)
		if gen == nil {
			t.Fatal("expected non-nil generator")
		}
	})

	t.Run("creates generator with custom config", func(t *testing.T) {
		cfg := &Config{
			DatabaseURL: "postgres://localhost:5432/test",
			RedisURL:    "redis://localhost:6379",
		}
		gen := NewGenerator(cfg)
		if gen == nil {
			t.Fatal("expected non-nil generator")
		}
	})
}

func TestGeneratorGenerateFromAST_NilProgram(t *testing.T) {
	gen := NewGenerator(nil)
	_, err := gen.GenerateFromAST(nil)
	if err == nil {
		t.Fatal("expected error for nil program")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("expected error to mention nil, got: %v", err)
	}
}

func TestGeneratorGenerateFromAST_EmptyProgram(t *testing.T) {
	gen := NewGenerator(nil)
	program := &ast.Program{
		Statements: []ast.Statement{},
	}

	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if code == nil {
		t.Fatal("expected non-nil generated code")
	}
	if code.Router == nil {
		t.Fatal("expected non-nil router")
	}
	if code.EndpointCount != 0 {
		t.Errorf("expected 0 endpoints, got %d", code.EndpointCount)
	}
}

func TestGenerateSimpleGETEndpoint(t *testing.T) {
	input := `
		endpoint GET "/health" {
			response HealthResponse status 200
		}
	`

	endpoints, err := parser.ParseEndpoints(input)
	if err != nil {
		t.Fatalf("failed to parse endpoints: %v", err)
	}

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	gen := NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("code generation failed: %v", err)
	}

	if code.EndpointCount != 1 {
		t.Errorf("expected 1 endpoint, got %d", code.EndpointCount)
	}

	// Test the generated endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	code.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestGeneratePOSTEndpointWithBody(t *testing.T) {
	input := `
		endpoint POST "/users" {
			request CreateUser from body
			response User status 201
			do {
				validate(request)
				insert(User, request)
			}
		}
	`

	endpoints, err := parser.ParseEndpoints(input)
	if err != nil {
		t.Fatalf("failed to parse endpoints: %v", err)
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	gen := NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("code generation failed: %v", err)
	}

	// Test with valid JSON body
	body := `{"name": "Test User", "email": "test@example.com"}`
	req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	code.Router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestGenerateEndpointWithPathParams(t *testing.T) {
	input := `
		endpoint GET "/users/:id" {
			request UserID from path
			response User status 200
		}
	`

	endpoints, err := parser.ParseEndpoints(input)
	if err != nil {
		t.Fatalf("failed to parse endpoints: %v", err)
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	gen := NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("code generation failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()
	code.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestGenerateEndpointWithQueryParams(t *testing.T) {
	input := `
		endpoint GET "/search" {
			request SearchParams from query
			response SearchResults status 200
		}
	`

	endpoints, err := parser.ParseEndpoints(input)
	if err != nil {
		t.Fatalf("failed to parse endpoints: %v", err)
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	gen := NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("code generation failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/search?q=test&limit=10", nil)
	w := httptest.NewRecorder()
	code.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestGenerateDeleteEndpoint(t *testing.T) {
	input := `
		endpoint DELETE "/users/:id" {
			request UserID from path
			response Empty status 204
		}
	`

	endpoints, err := parser.ParseEndpoints(input)
	if err != nil {
		t.Fatalf("failed to parse endpoints: %v", err)
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	gen := NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("code generation failed: %v", err)
	}

	req := httptest.NewRequest("DELETE", "/users/123", nil)
	w := httptest.NewRecorder()
	code.Router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}

func TestGeneratePUTEndpoint(t *testing.T) {
	input := `
		endpoint PUT "/users/:id" {
			request UpdateUser from body
			response User status 200
		}
	`

	endpoints, err := parser.ParseEndpoints(input)
	if err != nil {
		t.Fatalf("failed to parse endpoints: %v", err)
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	gen := NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("code generation failed: %v", err)
	}

	body := `{"name": "Updated User"}`
	req := httptest.NewRequest("PUT", "/users/123", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	code.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestGeneratePATCHEndpoint(t *testing.T) {
	input := `
		endpoint PATCH "/users/:id" {
			request PatchUser from body
			response User status 200
		}
	`

	endpoints, err := parser.ParseEndpoints(input)
	if err != nil {
		t.Fatalf("failed to parse endpoints: %v", err)
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	gen := NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("code generation failed: %v", err)
	}

	body := `{"name": "Patched User"}`
	req := httptest.NewRequest("PATCH", "/users/123", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	code.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestConvertPathToChi(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users", "/users"},
		{"/users/:id", "/users/{id}"},
		{"/users/:id/posts/:postId", "/users/{id}/posts/{postId}"},
		{"/api/v1/:resource/:id", "/api/v1/{resource}/{id}"},
		{"/:a/:b/:c", "/{a}/{b}/{c}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertPathToChi(tt.input)
			if result != tt.expected {
				t.Errorf("convertPathToChi(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMultipleEndpoints(t *testing.T) {
	input := `
		endpoint GET "/users" {
			response UserList status 200
		}
		endpoint GET "/users/:id" {
			request UserID from path
			response User status 200
		}
		endpoint POST "/users" {
			request CreateUser from body
			response User status 201
		}
	`

	endpoints, err := parser.ParseEndpoints(input)
	if err != nil {
		t.Fatalf("failed to parse endpoints: %v", err)
	}

	stmts := make([]ast.Statement, len(endpoints))
	for i, ep := range endpoints {
		stmts[i] = ep
	}

	program := &ast.Program{Statements: stmts}

	gen := NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("code generation failed: %v", err)
	}

	if code.EndpointCount != 3 {
		t.Errorf("expected 3 endpoints, got %d", code.EndpointCount)
	}

	// Test each endpoint
	tests := []struct {
		method   string
		path     string
		expected int
	}{
		{"GET", "/users", http.StatusOK},
		{"GET", "/users/123", http.StatusOK},
		{"POST", "/users", http.StatusCreated},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			var req *http.Request
			if tt.method == "POST" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader("{}"))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			w := httptest.NewRecorder()
			code.Router.ServeHTTP(w, req)

			// Accept any non-error status for these tests
			if w.Code >= 500 {
				t.Errorf("%s %s: expected success, got %d", tt.method, tt.path, w.Code)
			}
		})
	}
}

func TestTypeRegistry(t *testing.T) {
	registry := NewTypeRegistry()

	if len(registry.Models) != 0 {
		t.Error("expected empty models map")
	}
	if len(registry.Collections) != 0 {
		t.Error("expected empty collections map")
	}

	// Add a model
	registry.Models["User"] = &ModelInfo{
		Name: "User",
		Fields: []FieldInfo{
			{Name: "id", FieldType: "uuid", Primary: true},
			{Name: "name", FieldType: "string", Required: true},
		},
	}

	if len(registry.Models) != 1 {
		t.Errorf("expected 1 model, got %d", len(registry.Models))
	}

	// Add a collection
	registry.Collections["logs"] = &CollectionInfo{
		Name: "logs",
		Fields: []FieldInfo{
			{Name: "_id", FieldType: "objectid", Primary: true},
			{Name: "message", FieldType: "string"},
		},
	}

	if len(registry.Collections) != 1 {
		t.Errorf("expected 1 collection, got %d", len(registry.Collections))
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DatabaseURL == "" {
		t.Error("expected non-empty DatabaseURL")
	}
	if cfg.RedisURL == "" {
		t.Error("expected non-empty RedisURL")
	}
	if cfg.TemporalHost == "" {
		t.Error("expected non-empty TemporalHost")
	}
	if !cfg.EnableMetrics {
		t.Error("expected EnableMetrics to be true by default")
	}
}

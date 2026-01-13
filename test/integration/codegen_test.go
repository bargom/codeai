package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/codegen"
	"github.com/bargom/codeai/internal/parser"
)

// TestEndToEndCodeGeneration tests the full parse -> generate -> serve flow.
func TestEndToEndCodeGeneration(t *testing.T) {
	// Define a simple API
	caiContent := `
		endpoint GET "/health" {
			response Health status 200
		}
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

	// Parse the DSL
	endpoints, err := parser.ParseEndpoints(caiContent)
	if err != nil {
		t.Fatalf("parsing failed: %v", err)
	}

	// Build program
	stmts := make([]ast.Statement, len(endpoints))
	for i, ep := range endpoints {
		stmts[i] = ep
	}
	program := &ast.Program{Statements: stmts}

	// Generate code
	gen := codegen.NewGenerator(&codegen.Config{})
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("code generation failed: %v", err)
	}

	if code.EndpointCount != 4 {
		t.Errorf("expected 4 endpoints, got %d", code.EndpointCount)
	}

	// Start test server
	server := httptest.NewServer(code.Router)
	defer server.Close()

	// Test cases
	tests := []struct {
		method   string
		path     string
		body     string
		expected int
	}{
		{"GET", "/health", "", http.StatusOK},
		{"GET", "/users", "", http.StatusOK},
		{"GET", "/users/123", "", http.StatusOK},
		{"POST", "/users", `{"name":"Test User"}`, http.StatusCreated},
		{"GET", "/nonexistent", "", http.StatusNotFound},
	}

	client := server.Client()

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			var req *http.Request
			var err error

			if tt.body != "" {
				req, err = http.NewRequest(tt.method, server.URL+tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(tt.method, server.URL+tt.path, nil)
			}
			if err != nil {
				t.Fatalf("creating request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expected {
				t.Errorf("expected status %d, got %d", tt.expected, resp.StatusCode)
			}
		})
	}
}

// TestCodeGenWithRequestBody tests endpoints that process request bodies.
func TestCodeGenWithRequestBody(t *testing.T) {
	caiContent := `
		endpoint POST "/api/items" {
			request CreateItem from body
			response Item status 201
		}
	`

	endpoints, err := parser.ParseEndpoints(caiContent)
	if err != nil {
		t.Fatalf("parsing failed: %v", err)
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	gen := codegen.NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("code generation failed: %v", err)
	}

	server := httptest.NewServer(code.Router)
	defer server.Close()

	// Test with valid JSON body
	body := `{"name": "Test Item", "quantity": 5}`
	resp, err := http.Post(server.URL+"/api/items", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	// Response should be JSON
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected JSON content type, got %s", contentType)
	}
}

// TestCodeGenWithQueryParams tests endpoints that use query parameters.
func TestCodeGenWithQueryParams(t *testing.T) {
	caiContent := `
		endpoint GET "/search" {
			request SearchParams from query
			response SearchResults status 200
		}
	`

	endpoints, err := parser.ParseEndpoints(caiContent)
	if err != nil {
		t.Fatalf("parsing failed: %v", err)
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	gen := codegen.NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("code generation failed: %v", err)
	}

	server := httptest.NewServer(code.Router)
	defer server.Close()

	// Test with query parameters
	resp, err := http.Get(server.URL + "/search?q=test&limit=10&offset=0")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// TestCodeGenCORSMiddleware tests that CORS headers can be applied.
func TestCodeGenResponseFormat(t *testing.T) {
	caiContent := `
		endpoint GET "/api/status" {
			response Status status 200
		}
	`

	endpoints, err := parser.ParseEndpoints(caiContent)
	if err != nil {
		t.Fatalf("parsing failed: %v", err)
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	gen := codegen.NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("code generation failed: %v", err)
	}

	server := httptest.NewServer(code.Router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/status")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check that response is valid JSON
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Errorf("response is not valid JSON: %v", err)
	}
}

// TestCodeGenAllHTTPMethods tests all HTTP methods work correctly.
func TestCodeGenAllHTTPMethods(t *testing.T) {
	caiContent := `
		endpoint GET "/resources" {
			response ResourceList status 200
		}
		endpoint POST "/resources" {
			request CreateResource from body
			response Resource status 201
		}
		endpoint PUT "/resources/:id" {
			request UpdateResource from body
			response Resource status 200
		}
		endpoint PATCH "/resources/:id" {
			request PatchResource from body
			response Resource status 200
		}
		endpoint DELETE "/resources/:id" {
			request ResourceID from path
			response Empty status 204
		}
	`

	endpoints, err := parser.ParseEndpoints(caiContent)
	if err != nil {
		t.Fatalf("parsing failed: %v", err)
	}

	stmts := make([]ast.Statement, len(endpoints))
	for i, ep := range endpoints {
		stmts[i] = ep
	}
	program := &ast.Program{Statements: stmts}

	gen := codegen.NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("code generation failed: %v", err)
	}

	server := httptest.NewServer(code.Router)
	defer server.Close()
	client := server.Client()

	methods := []struct {
		method   string
		path     string
		body     string
		expected int
	}{
		{"GET", "/resources", "", http.StatusOK},
		{"POST", "/resources", "{}", http.StatusCreated},
		{"PUT", "/resources/123", "{}", http.StatusOK},
		{"PATCH", "/resources/123", "{}", http.StatusOK},
		{"DELETE", "/resources/123", "", http.StatusNoContent},
	}

	for _, m := range methods {
		t.Run(m.method, func(t *testing.T) {
			var req *http.Request
			var err error

			if m.body != "" {
				req, err = http.NewRequest(m.method, server.URL+m.path, strings.NewReader(m.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(m.method, server.URL+m.path, nil)
			}
			if err != nil {
				t.Fatalf("creating request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != m.expected {
				t.Errorf("%s %s: expected status %d, got %d", m.method, m.path, m.expected, resp.StatusCode)
			}
		})
	}
}

// TestCodeGenMultiplePathParams tests endpoints with multiple path parameters.
func TestCodeGenMultiplePathParams(t *testing.T) {
	caiContent := `
		endpoint GET "/users/:userId/posts/:postId" {
			request PostParams from path
			response Post status 200
		}
	`

	endpoints, err := parser.ParseEndpoints(caiContent)
	if err != nil {
		t.Fatalf("parsing failed: %v", err)
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	gen := codegen.NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("code generation failed: %v", err)
	}

	server := httptest.NewServer(code.Router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/users/42/posts/123")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// TestCodeGenConcurrentRequests tests that the server handles concurrent requests.
func TestCodeGenConcurrentRequests(t *testing.T) {
	caiContent := `
		endpoint GET "/concurrent" {
			response Data status 200
		}
	`

	endpoints, err := parser.ParseEndpoints(caiContent)
	if err != nil {
		t.Fatalf("parsing failed: %v", err)
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	gen := codegen.NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		t.Fatalf("code generation failed: %v", err)
	}

	server := httptest.NewServer(code.Router)
	defer server.Close()

	// Make concurrent requests
	const numRequests = 50
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			resp, err := http.Get(server.URL + "/concurrent")
			if err != nil {
				results <- 0
				return
			}
			defer resp.Body.Close()
			results <- resp.StatusCode
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < numRequests; i++ {
		status := <-results
		if status == http.StatusOK {
			successCount++
		}
	}

	if successCount != numRequests {
		t.Errorf("expected %d successful requests, got %d", numRequests, successCount)
	}
}

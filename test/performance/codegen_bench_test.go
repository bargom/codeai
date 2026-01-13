package performance

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/codegen"
	"github.com/bargom/codeai/internal/parser"
)

// BenchmarkGenerateEndpoint measures the time to generate a single endpoint.
func BenchmarkGenerateEndpoint(b *testing.B) {
	input := `
		endpoint GET "/users/:id" {
			request UserID from path
			response User status 200
		}
	`

	endpoints, err := parser.ParseEndpoints(input)
	if err != nil {
		b.Fatalf("parsing failed: %v", err)
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen := codegen.NewGenerator(nil)
		_, err := gen.GenerateFromAST(program)
		if err != nil {
			b.Fatalf("code generation failed: %v", err)
		}
	}
}

// BenchmarkGenerateMultipleEndpoints measures generating multiple endpoints.
func BenchmarkGenerateMultipleEndpoints(b *testing.B) {
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
		endpoint PUT "/users/:id" {
			request UpdateUser from body
			response User status 200
		}
		endpoint DELETE "/users/:id" {
			request UserID from path
			response Empty status 204
		}
	`

	endpoints, err := parser.ParseEndpoints(input)
	if err != nil {
		b.Fatalf("parsing failed: %v", err)
	}

	stmts := make([]ast.Statement, len(endpoints))
	for i, ep := range endpoints {
		stmts[i] = ep
	}
	program := &ast.Program{Statements: stmts}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen := codegen.NewGenerator(nil)
		_, err := gen.GenerateFromAST(program)
		if err != nil {
			b.Fatalf("code generation failed: %v", err)
		}
	}
}

// BenchmarkHandlerExecution measures the time for a request to be handled.
func BenchmarkHandlerExecution(b *testing.B) {
	input := `
		endpoint GET "/users/:id" {
			request UserID from path
			response User status 200
		}
	`

	endpoints, err := parser.ParseEndpoints(input)
	if err != nil {
		b.Fatalf("parsing failed: %v", err)
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	gen := codegen.NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		b.Fatalf("code generation failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/users/123", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		code.Router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("expected 200, got %d", w.Code)
		}
	}
}

// BenchmarkHandlerExecutionWithBody measures handling POST requests with body.
func BenchmarkHandlerExecutionWithBody(b *testing.B) {
	input := `
		endpoint POST "/users" {
			request CreateUser from body
			response User status 201
		}
	`

	endpoints, err := parser.ParseEndpoints(input)
	if err != nil {
		b.Fatalf("parsing failed: %v", err)
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	gen := codegen.NewGenerator(nil)
	code, err := gen.GenerateFromAST(program)
	if err != nil {
		b.Fatalf("code generation failed: %v", err)
	}

	body := `{"name": "Test User", "email": "test@example.com"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		code.Router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			b.Fatalf("expected 201, got %d", w.Code)
		}
	}
}

// BenchmarkExecutionContext measures execution context operations.
func BenchmarkExecutionContext(b *testing.B) {
	generatedCode := &codegen.GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		ModelRegistry: codegen.NewTypeRegistry(),
	}

	factory := codegen.NewExecutionContextFactory(generatedCode)
	req := httptest.NewRequest("GET", "/", nil)

	b.Run("SetGet", func(b *testing.B) {
		ctx := factory.NewContext(req.Context(), req)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx.Set("key", "value")
			_ = ctx.Get("key")
		}
	})

	b.Run("SetInput", func(b *testing.B) {
		ctx := factory.NewContext(req.Context(), req)
		input := map[string]interface{}{"id": "123", "name": "test"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx.SetInput(input)
			_ = ctx.Input()
		}
	})

	b.Run("Transform", func(b *testing.B) {
		ctx := factory.NewContext(req.Context(), req)
		data := map[string]interface{}{"name": "test"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = ctx.Transform("toJSON", data)
		}
	})
}

// BenchmarkParseAndGenerate measures end-to-end parse+generate time.
func BenchmarkParseAndGenerate(b *testing.B) {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		endpoints, err := parser.ParseEndpoints(input)
		if err != nil {
			b.Fatalf("parsing failed: %v", err)
		}

		stmts := make([]ast.Statement, len(endpoints))
		for j, ep := range endpoints {
			stmts[j] = ep
		}
		program := &ast.Program{Statements: stmts}

		gen := codegen.NewGenerator(nil)
		_, err = gen.GenerateFromAST(program)
		if err != nil {
			b.Fatalf("code generation failed: %v", err)
		}
	}
}

// BenchmarkConvertPathToChi measures path conversion performance.
func BenchmarkConvertPathToChi(b *testing.B) {
	paths := []string{
		"/users",
		"/users/:id",
		"/users/:id/posts/:postId",
		"/api/v1/:resource/:id/:subresource",
	}

	// We need to export the function or use reflection
	// For now, we'll just benchmark the full generation
	input := `
		endpoint GET "/users/:id/posts/:postId" {
			request Params from path
			response Post status 200
		}
	`

	endpoints, err := parser.ParseEndpoints(input)
	if err != nil {
		b.Fatalf("parsing failed: %v", err)
	}

	program := &ast.Program{
		Statements: []ast.Statement{endpoints[0]},
	}

	_ = paths // paths would be used if convertPathToChi was exported

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen := codegen.NewGenerator(nil)
		_, err = gen.GenerateFromAST(program)
		if err != nil {
			b.Fatalf("code generation failed: %v", err)
		}
	}
}

// BenchmarkManyEndpoints measures generation with many endpoints.
func BenchmarkManyEndpoints(b *testing.B) {
	// Generate many endpoints programmatically
	var sb strings.Builder
	for i := 0; i < 50; i++ {
		sb.WriteString(`endpoint GET "/api/resource` + string(rune('a'+i%26)) + `/:id" {
			request ResourceID from path
			response Resource status 200
		}
		`)
	}

	input := sb.String()

	endpoints, err := parser.ParseEndpoints(input)
	if err != nil {
		b.Fatalf("parsing failed: %v", err)
	}

	stmts := make([]ast.Statement, len(endpoints))
	for i, ep := range endpoints {
		stmts[i] = ep
	}
	program := &ast.Program{Statements: stmts}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen := codegen.NewGenerator(nil)
		_, err = gen.GenerateFromAST(program)
		if err != nil {
			b.Fatalf("code generation failed: %v", err)
		}
	}
}

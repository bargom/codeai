package parser

import (
	"testing"

	"github.com/bargom/codeai/internal/ast"
)

// =============================================================================
// Basic Endpoint Parsing Tests
// =============================================================================

func TestParseEndpoint_SimpleGET(t *testing.T) {
	input := `endpoint GET "/users" {
		response UserList status 200
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if endpoint == nil {
		t.Fatal("Expected endpoint, got nil")
	}

	if endpoint.Method != ast.HTTPMethodGET {
		t.Errorf("Expected method GET, got %s", endpoint.Method)
	}

	if endpoint.Path != "/users" {
		t.Errorf("Expected path /users, got %s", endpoint.Path)
	}

	if endpoint.Handler == nil {
		t.Fatal("Expected handler, got nil")
	}

	if endpoint.Handler.Response == nil {
		t.Fatal("Expected response, got nil")
	}

	if endpoint.Handler.Response.TypeName != "UserList" {
		t.Errorf("Expected response type UserList, got %s", endpoint.Handler.Response.TypeName)
	}

	if endpoint.Handler.Response.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", endpoint.Handler.Response.StatusCode)
	}
}

func TestParseEndpoint_POSTWithBody(t *testing.T) {
	input := `endpoint POST "/users" {
		request CreateUser from body
		response User status 201
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if endpoint.Method != ast.HTTPMethodPOST {
		t.Errorf("Expected method POST, got %s", endpoint.Method)
	}

	if endpoint.Handler.Request == nil {
		t.Fatal("Expected request, got nil")
	}

	if endpoint.Handler.Request.TypeName != "CreateUser" {
		t.Errorf("Expected request type CreateUser, got %s", endpoint.Handler.Request.TypeName)
	}

	if endpoint.Handler.Request.Source != ast.RequestSourceBody {
		t.Errorf("Expected source body, got %s", endpoint.Handler.Request.Source)
	}

	if endpoint.Handler.Response.StatusCode != 201 {
		t.Errorf("Expected status 201, got %d", endpoint.Handler.Response.StatusCode)
	}
}

func TestParseEndpoint_GETWithPathParam(t *testing.T) {
	input := `endpoint GET "/users/:id" {
		request UserID from path
		response UserDetail status 200
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if endpoint.Path != "/users/:id" {
		t.Errorf("Expected path /users/:id, got %s", endpoint.Path)
	}

	if endpoint.Handler.Request.Source != ast.RequestSourcePath {
		t.Errorf("Expected source path, got %s", endpoint.Handler.Request.Source)
	}
}

func TestParseEndpoint_GETWithQueryParams(t *testing.T) {
	input := `endpoint GET "/users" {
		request SearchParams from query
		response UserList status 200
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if endpoint.Handler.Request.Source != ast.RequestSourceQuery {
		t.Errorf("Expected source query, got %s", endpoint.Handler.Request.Source)
	}
}

func TestParseEndpoint_GETWithHeader(t *testing.T) {
	input := `endpoint GET "/users/me" {
		request AuthToken from header
		response User status 200
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if endpoint.Handler.Request.Source != ast.RequestSourceHeader {
		t.Errorf("Expected source header, got %s", endpoint.Handler.Request.Source)
	}
}

// =============================================================================
// HTTP Method Tests
// =============================================================================

func TestParseEndpoint_AllHTTPMethods(t *testing.T) {
	tests := []struct {
		method   string
		expected ast.HTTPMethod
	}{
		{"GET", ast.HTTPMethodGET},
		{"POST", ast.HTTPMethodPOST},
		{"PUT", ast.HTTPMethodPUT},
		{"DELETE", ast.HTTPMethodDELETE},
		{"PATCH", ast.HTTPMethodPATCH},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			input := `endpoint ` + tt.method + ` "/test" {
				response Result status 200
			}`

			endpoint, err := ParseEndpoint(input)
			if err != nil {
				t.Fatalf("Failed to parse endpoint: %v", err)
			}

			if endpoint.Method != tt.expected {
				t.Errorf("Expected method %s, got %s", tt.expected, endpoint.Method)
			}
		})
	}
}

// =============================================================================
// Middleware Tests
// =============================================================================

func TestParseEndpoint_WithMiddleware(t *testing.T) {
	input := `endpoint POST "/admin/users" {
		middleware auth
		request CreateUser from body
		response User status 201
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if len(endpoint.Middlewares) != 1 {
		t.Fatalf("Expected 1 middleware, got %d", len(endpoint.Middlewares))
	}

	if endpoint.Middlewares[0].Name != "auth" {
		t.Errorf("Expected middleware auth, got %s", endpoint.Middlewares[0].Name)
	}
}

func TestParseEndpoint_WithMultipleMiddlewares(t *testing.T) {
	input := `endpoint DELETE "/admin/users/:id" {
		middleware auth
		middleware admin
		middleware logging
		request UserID from path
		response Empty status 204
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if len(endpoint.Middlewares) != 3 {
		t.Fatalf("Expected 3 middlewares, got %d", len(endpoint.Middlewares))
	}

	expectedMiddlewares := []string{"auth", "admin", "logging"}
	for i, expected := range expectedMiddlewares {
		if endpoint.Middlewares[i].Name != expected {
			t.Errorf("Expected middleware[%d] %s, got %s", i, expected, endpoint.Middlewares[i].Name)
		}
	}
}

// =============================================================================
// Annotation Tests
// =============================================================================

func TestParseEndpoint_WithAnnotation(t *testing.T) {
	input := `@deprecated
	endpoint GET "/old/users" {
		response UserList status 200
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if len(endpoint.Annotations) != 1 {
		t.Fatalf("Expected 1 annotation, got %d", len(endpoint.Annotations))
	}

	if endpoint.Annotations[0].Name != "deprecated" {
		t.Errorf("Expected annotation deprecated, got %s", endpoint.Annotations[0].Name)
	}
}

func TestParseEndpoint_WithAnnotationValue(t *testing.T) {
	input := `@auth("admin")
	endpoint DELETE "/users/:id" {
		request UserID from path
		response Empty status 204
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if len(endpoint.Annotations) != 1 {
		t.Fatalf("Expected 1 annotation, got %d", len(endpoint.Annotations))
	}

	if endpoint.Annotations[0].Name != "auth" {
		t.Errorf("Expected annotation auth, got %s", endpoint.Annotations[0].Name)
	}

	if endpoint.Annotations[0].Value != "admin" {
		t.Errorf("Expected annotation value admin, got %s", endpoint.Annotations[0].Value)
	}
}

func TestParseEndpoint_WithMultipleAnnotations(t *testing.T) {
	input := `@deprecated
	@auth("admin")
	@rateLimit("100")
	endpoint GET "/api/v1/data" {
		response Data status 200
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if len(endpoint.Annotations) != 3 {
		t.Fatalf("Expected 3 annotations, got %d", len(endpoint.Annotations))
	}

	expectedAnnotations := []struct {
		name  string
		value string
	}{
		{"deprecated", ""},
		{"auth", "admin"},
		{"rateLimit", "100"},
	}

	for i, expected := range expectedAnnotations {
		if endpoint.Annotations[i].Name != expected.name {
			t.Errorf("Expected annotation[%d] name %s, got %s", i, expected.name, endpoint.Annotations[i].Name)
		}
		if endpoint.Annotations[i].Value != expected.value {
			t.Errorf("Expected annotation[%d] value %s, got %s", i, expected.value, endpoint.Annotations[i].Value)
		}
	}
}

// =============================================================================
// Handler Logic Tests
// =============================================================================

func TestParseEndpoint_WithLogicBlock(t *testing.T) {
	input := `endpoint GET "/users/:id" {
		request UserID from path
		response User status 200
		do {
			validate(request)
		}
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if endpoint.Handler.Logic == nil {
		t.Fatal("Expected logic block, got nil")
	}

	if len(endpoint.Handler.Logic.Steps) != 1 {
		t.Fatalf("Expected 1 logic step, got %d", len(endpoint.Handler.Logic.Steps))
	}

	step := endpoint.Handler.Logic.Steps[0]
	if step.Action != "validate" {
		t.Errorf("Expected action validate, got %s", step.Action)
	}

	if len(step.Args) != 1 || step.Args[0] != "request" {
		t.Errorf("Expected args [request], got %v", step.Args)
	}
}

func TestParseEndpoint_WithAssignmentLogic(t *testing.T) {
	input := `endpoint GET "/users/:id" {
		request UserID from path
		response UserDetail status 200
		do {
			user = find(User, id)
		}
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if len(endpoint.Handler.Logic.Steps) != 1 {
		t.Fatalf("Expected 1 logic step, got %d", len(endpoint.Handler.Logic.Steps))
	}

	step := endpoint.Handler.Logic.Steps[0]
	if step.Target != "user" {
		t.Errorf("Expected target user, got %s", step.Target)
	}

	if step.Action != "find" {
		t.Errorf("Expected action find, got %s", step.Action)
	}
}

func TestParseEndpoint_WithMultipleLogicSteps(t *testing.T) {
	input := `endpoint POST "/users" {
		request CreateUser from body
		response User status 201
		do {
			validate(request)
			authorize(request, "admin")
			user = create(User, request)
			send(email, user)
		}
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if len(endpoint.Handler.Logic.Steps) != 4 {
		t.Fatalf("Expected 4 logic steps, got %d", len(endpoint.Handler.Logic.Steps))
	}

	// Check first step
	if endpoint.Handler.Logic.Steps[0].Action != "validate" {
		t.Errorf("Expected action validate, got %s", endpoint.Handler.Logic.Steps[0].Action)
	}

	// Check second step
	if endpoint.Handler.Logic.Steps[1].Action != "authorize" {
		t.Errorf("Expected action authorize, got %s", endpoint.Handler.Logic.Steps[1].Action)
	}
	if len(endpoint.Handler.Logic.Steps[1].Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(endpoint.Handler.Logic.Steps[1].Args))
	}

	// Check third step with assignment
	if endpoint.Handler.Logic.Steps[2].Target != "user" {
		t.Errorf("Expected target user, got %s", endpoint.Handler.Logic.Steps[2].Target)
	}
	if endpoint.Handler.Logic.Steps[2].Action != "create" {
		t.Errorf("Expected action create, got %s", endpoint.Handler.Logic.Steps[2].Action)
	}
}

func TestParseEndpoint_WithLogicWhereClause(t *testing.T) {
	input := `endpoint GET "/users" {
		request SearchParams from query
		response UserList status 200
		do {
			users = find(User) where "active = true"
		}
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	step := endpoint.Handler.Logic.Steps[0]
	if step.Condition != "active = true" {
		t.Errorf("Expected condition 'active = true', got %s", step.Condition)
	}
}

func TestParseEndpoint_WithLogicOptions(t *testing.T) {
	input := `endpoint GET "/users/:id" {
		request UserID from path
		response User status 200
		do {
			user = find(User, id) with { cache: true }
		}
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	step := endpoint.Handler.Logic.Steps[0]
	if len(step.Options) != 1 {
		t.Fatalf("Expected 1 option, got %d", len(step.Options))
	}

	if step.Options[0].Key != "cache" {
		t.Errorf("Expected option key cache, got %s", step.Options[0].Key)
	}
}

// =============================================================================
// Multiple Endpoints Tests
// =============================================================================

func TestParseEndpoints_Multiple(t *testing.T) {
	input := `
	endpoint GET "/users" {
		response UserList status 200
	}

	endpoint POST "/users" {
		request CreateUser from body
		response User status 201
	}

	endpoint GET "/users/:id" {
		request UserID from path
		response UserDetail status 200
	}

	endpoint DELETE "/users/:id" {
		middleware auth
		request UserID from path
		response Empty status 204
	}`

	endpoints, err := ParseEndpoints(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoints: %v", err)
	}

	if len(endpoints) != 4 {
		t.Fatalf("Expected 4 endpoints, got %d", len(endpoints))
	}

	// Verify methods
	expectedMethods := []ast.HTTPMethod{
		ast.HTTPMethodGET,
		ast.HTTPMethodPOST,
		ast.HTTPMethodGET,
		ast.HTTPMethodDELETE,
	}

	for i, expected := range expectedMethods {
		if endpoints[i].Method != expected {
			t.Errorf("Endpoint[%d]: expected method %s, got %s", i, expected, endpoints[i].Method)
		}
	}
}

// =============================================================================
// Complex Endpoint Tests
// =============================================================================

func TestParseEndpoint_ComplexEndpoint(t *testing.T) {
	input := `@auth("admin")
	@rateLimit("50")
	endpoint PUT "/admin/users/:id" {
		middleware auth
		middleware admin
		request UpdateUser from body
		response User status 200
		do {
			validate(request)
			authorize(request, "admin")
			user = find(User, id) where "deleted_at IS NULL"
			updated = update(user, request)
			log(audit, updated)
		}
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	// Check method
	if endpoint.Method != ast.HTTPMethodPUT {
		t.Errorf("Expected method PUT, got %s", endpoint.Method)
	}

	// Check path
	if endpoint.Path != "/admin/users/:id" {
		t.Errorf("Expected path /admin/users/:id, got %s", endpoint.Path)
	}

	// Check annotations
	if len(endpoint.Annotations) != 2 {
		t.Errorf("Expected 2 annotations, got %d", len(endpoint.Annotations))
	}

	// Check middlewares
	if len(endpoint.Middlewares) != 2 {
		t.Errorf("Expected 2 middlewares, got %d", len(endpoint.Middlewares))
	}

	// Check request
	if endpoint.Handler.Request.TypeName != "UpdateUser" {
		t.Errorf("Expected request type UpdateUser, got %s", endpoint.Handler.Request.TypeName)
	}

	// Check response
	if endpoint.Handler.Response.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", endpoint.Handler.Response.StatusCode)
	}

	// Check logic steps
	if len(endpoint.Handler.Logic.Steps) != 5 {
		t.Errorf("Expected 5 logic steps, got %d", len(endpoint.Handler.Logic.Steps))
	}
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func TestParseEndpoint_EmptyBody(t *testing.T) {
	input := `endpoint GET "/health" {}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if endpoint.Handler.Request != nil {
		t.Error("Expected no request, got one")
	}

	if endpoint.Handler.Response != nil {
		t.Error("Expected no response, got one")
	}
}

func TestParseEndpoint_OnlyRequest(t *testing.T) {
	input := `endpoint POST "/data" {
		request DataInput from body
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if endpoint.Handler.Request == nil {
		t.Fatal("Expected request, got nil")
	}

	if endpoint.Handler.Response != nil {
		t.Error("Expected no response, got one")
	}
}

func TestParseEndpoint_OnlyResponse(t *testing.T) {
	input := `endpoint GET "/status" {
		response Status status 200
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if endpoint.Handler.Request != nil {
		t.Error("Expected no request, got one")
	}

	if endpoint.Handler.Response == nil {
		t.Fatal("Expected response, got nil")
	}
}

func TestParseEndpoint_PathWithMultipleParams(t *testing.T) {
	input := `endpoint GET "/orgs/:orgId/users/:userId" {
		response User status 200
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if endpoint.Path != "/orgs/:orgId/users/:userId" {
		t.Errorf("Expected path /orgs/:orgId/users/:userId, got %s", endpoint.Path)
	}
}

func TestParseEndpoint_SpecialStatusCodes(t *testing.T) {
	tests := []struct {
		status int
		input  string
	}{
		{200, `endpoint GET "/ok" { response OK status 200 }`},
		{201, `endpoint POST "/created" { response Created status 201 }`},
		{204, `endpoint DELETE "/deleted" { response Empty status 204 }`},
		{400, `endpoint POST "/bad" { response Error status 400 }`},
		{401, `endpoint GET "/unauthorized" { response Error status 401 }`},
		{403, `endpoint GET "/forbidden" { response Error status 403 }`},
		{404, `endpoint GET "/notfound" { response Error status 404 }`},
		{500, `endpoint GET "/error" { response Error status 500 }`},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.status)), func(t *testing.T) {
			endpoint, err := ParseEndpoint(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse endpoint: %v", err)
			}

			if endpoint.Handler.Response.StatusCode != tt.status {
				t.Errorf("Expected status %d, got %d", tt.status, endpoint.Handler.Response.StatusCode)
			}
		})
	}
}

func TestParseEndpoint_WithComments(t *testing.T) {
	input := `// This is a user endpoint
	endpoint GET "/users" {
		// Return all users
		response UserList status 200
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if endpoint.Method != ast.HTTPMethodGET {
		t.Errorf("Expected method GET, got %s", endpoint.Method)
	}
}

func TestParseEndpoint_InvalidMethod(t *testing.T) {
	input := `endpoint INVALID "/test" {
		response Test status 200
	}`

	_, err := ParseEndpoint(input)
	if err == nil {
		t.Error("Expected error for invalid method, got nil")
	}
}

func TestParseEndpoint_InvalidSource(t *testing.T) {
	input := `endpoint GET "/test" {
		request Test from invalid
		response Test status 200
	}`

	_, err := ParseEndpoint(input)
	if err == nil {
		t.Error("Expected error for invalid source, got nil")
	}
}

func TestParseEndpoints_Empty(t *testing.T) {
	input := ""

	endpoints, err := ParseEndpoints(input)
	if err != nil {
		t.Fatalf("Failed to parse empty input: %v", err)
	}

	if len(endpoints) != 0 {
		t.Errorf("Expected 0 endpoints, got %d", len(endpoints))
	}
}

// =============================================================================
// String Representation Tests
// =============================================================================

func TestEndpointDecl_String(t *testing.T) {
	input := `endpoint GET "/users" {
		response UserList status 200
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	str := endpoint.String()
	if str != `EndpointDecl{Method: "GET", Path: "/users"}` {
		t.Errorf("Unexpected string representation: %s", str)
	}
}

func TestEndpointDecl_Type(t *testing.T) {
	input := `endpoint GET "/users" {
		response UserList status 200
	}`

	endpoint, err := ParseEndpoint(input)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	if endpoint.Type() != ast.NodeEndpointDecl {
		t.Errorf("Expected NodeEndpointDecl, got %v", endpoint.Type())
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkParseEndpoint_Simple(b *testing.B) {
	input := `endpoint GET "/users" {
		response UserList status 200
	}`

	for i := 0; i < b.N; i++ {
		_, err := ParseEndpoint(input)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParseEndpoint_Complex(b *testing.B) {
	input := `@auth("admin")
	@rateLimit("50")
	endpoint PUT "/admin/users/:id" {
		middleware auth
		middleware admin
		request UpdateUser from body
		response User status 200
		do {
			validate(request)
			authorize(request, "admin")
			user = find(User, id)
			updated = update(user, request)
			log(audit, updated)
		}
	}`

	for i := 0; i < b.N; i++ {
		_, err := ParseEndpoint(input)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParseEndpoints_Multiple(b *testing.B) {
	input := `
	endpoint GET "/users" { response UserList status 200 }
	endpoint POST "/users" { request CreateUser from body response User status 201 }
	endpoint GET "/users/:id" { request UserID from path response UserDetail status 200 }
	endpoint PUT "/users/:id" { request UpdateUser from body response User status 200 }
	endpoint DELETE "/users/:id" { request UserID from path response Empty status 204 }
	`

	for i := 0; i < b.N; i++ {
		_, err := ParseEndpoints(input)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

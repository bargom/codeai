//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/parser"
	"github.com/bargom/codeai/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseCompleteApplicationWithEndpoints tests parsing a complete .cai file with endpoints.
func TestParseCompleteApplicationWithEndpoints(t *testing.T) {
	// Find the with_endpoints.cai file
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Navigate up to find examples directory
	for {
		examplesPath := filepath.Join(cwd, "examples", "with_endpoints.cai")
		if _, err := os.Stat(examplesPath); err == nil {
			// Parse the file
			program, err := parser.ParseFile(examplesPath)
			require.NoError(t, err, "Failed to parse with_endpoints.cai")
			require.NotNil(t, program, "Parsed program should not be nil")

			// Verify we have statements
			assert.Greater(t, len(program.Statements), 0, "Program should have statements")

			// Count endpoint statements
			endpointCount := 0
			for _, stmt := range program.Statements {
				if stmt.Type() == ast.NodeEndpointDecl {
					endpointCount++
				}
			}

			// We should have multiple endpoints
			assert.Greater(t, endpointCount, 5, "Should have at least 6 endpoints")
			t.Logf("Found %d endpoints in the file", endpointCount)
			return
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Skip("Could not find examples/with_endpoints.cai in parent directories")
			return
		}
		cwd = parent
	}
}

// TestValidateCompleteApplicationWithEndpoints tests validation of endpoints with models and middleware.
func TestValidateCompleteApplicationWithEndpoints(t *testing.T) {
	// Find the with_endpoints.cai file
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Navigate up to find examples directory
	for {
		examplesPath := filepath.Join(cwd, "examples", "with_endpoints.cai")
		if _, err := os.Stat(examplesPath); err == nil {
			// Parse the file
			program, err := parser.ParseFile(examplesPath)
			require.NoError(t, err, "Failed to parse with_endpoints.cai")

			// Validate the parsed program
			v := validator.New()
			err = v.Validate(program)
			require.NoError(t, err, "Validation should pass for complete application with endpoints")
			return
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Skip("Could not find examples/with_endpoints.cai in parent directories")
			return
		}
		cwd = parent
	}
}

// TestEndpointWithModelsAndMiddleware tests integration between endpoints, models, and middleware.
func TestEndpointWithModelsAndMiddleware(t *testing.T) {
	source := `
database postgres {
    model User {
        id: string, primary
        email: string, required
        name: string, required
    }
}

middleware auth {
    type authentication
    config {
        method: jwt
        secret: env("JWT_SECRET")
    }
}

endpoint GET "/users/:id" {
  middleware auth
  request User from path
  response User status 200

  do {
    validate(id)
    query("users", id)
  }
}
`

	// Parse the source
	program, err := parser.Parse(source)
	require.NoError(t, err, "Should parse endpoint with models and middleware")

	// Count different node types
	modelCount := 0
	middlewareCount := 0
	endpointCount := 0

	for _, stmt := range program.Statements {
		switch stmt.Type() {
		case ast.NodeModelDecl:
			modelCount++
		case ast.NodeMiddlewareDecl:
			middlewareCount++
		case ast.NodeEndpointDecl:
			endpointCount++
		}
	}

	assert.Equal(t, 1, modelCount, "Should have 1 model")
	assert.Equal(t, 1, middlewareCount, "Should have 1 middleware")
	assert.Equal(t, 1, endpointCount, "Should have 1 endpoint")

	// Validate the program - should pass since all references are defined
	v := validator.New()
	err = v.Validate(program)
	assert.NoError(t, err, "Validation should pass when all types and middleware are defined")
}

// TestMultipleEndpointsIntegration tests parsing multiple endpoints with different HTTP methods.
func TestMultipleEndpointsIntegration(t *testing.T) {
	source := `
database postgres {
    model User {
        id: string, primary
        name: string, required
    }

    model CreateUserRequest {
        name: string, required
    }
}

middleware auth {
    type authentication
    config {
        method: jwt
    }
}

endpoint GET "/users" {
  response User status 200
}

endpoint POST "/users" {
  request CreateUserRequest from body
  response User status 201
  middleware auth
}

endpoint PUT "/users/:id" {
  request User from body
  response User status 200
  middleware auth
}

endpoint DELETE "/users/:id" {
  request User from path
  response User status 204
  middleware auth
}
`

	// Parse the source
	program, err := parser.Parse(source)
	require.NoError(t, err, "Should parse multiple endpoints")

	// Extract endpoints
	endpoints := []*ast.EndpointDecl{}
	for _, stmt := range program.Statements {
		if endpoint, ok := stmt.(*ast.EndpointDecl); ok {
			endpoints = append(endpoints, endpoint)
		}
	}

	require.Len(t, endpoints, 4, "Should have 4 endpoints")

	// Verify HTTP methods
	methods := []ast.HTTPMethod{
		endpoints[0].Method,
		endpoints[1].Method,
		endpoints[2].Method,
		endpoints[3].Method,
	}

	assert.Contains(t, methods, ast.HTTPMethodGET)
	assert.Contains(t, methods, ast.HTTPMethodPOST)
	assert.Contains(t, methods, ast.HTTPMethodPUT)
	assert.Contains(t, methods, ast.HTTPMethodDELETE)

	// Verify paths
	paths := []string{
		endpoints[0].Path,
		endpoints[1].Path,
		endpoints[2].Path,
		endpoints[3].Path,
	}

	assert.Contains(t, paths, "/users")
	assert.Contains(t, paths, "/users/:id")

	// Validate the program
	v := validator.New()
	err = v.Validate(program)
	assert.NoError(t, err, "Multiple endpoints should validate successfully")
}

// TestEndpointValidationErrors tests validation error handling for endpoints.
func TestEndpointValidationErrors(t *testing.T) {
	testCases := []struct {
		name        string
		source      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "undefined request type",
			source: `
endpoint GET "/users" {
  request UndefinedType from body
  response User status 200
}`,
			expectError: true,
			errorMsg:    "undefined",
		},
		{
			name: "undefined response type",
			source: `
endpoint GET "/users" {
  request User from body
  response UndefinedResponse status 200
}`,
			expectError: true,
			errorMsg:    "undefined",
		},
		{
			name: "undefined middleware",
			source: `
database postgres {
    model User {
        id: string, primary
    }
}

endpoint GET "/users" {
  middleware undefinedMiddleware
  response User status 200
}`,
			expectError: true,
			errorMsg:    "undefined",
		},
		{
			name: "valid endpoint with defined types",
			source: `
database postgres {
    model User {
        id: string, primary
    }

    model CreateUser {
        name: string, required
    }
}

middleware auth {
    type authentication
    config {
        method: jwt
    }
}

endpoint POST "/users" {
  middleware auth
  request CreateUser from body
  response User status 201
}`,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			program, parseErr := parser.Parse(tc.source)
			require.NoError(t, parseErr, "Parse should succeed")

			v := validator.New()
			validateErr := v.Validate(program)

			if tc.expectError {
				assert.Error(t, validateErr, "Should have validation error")
				if tc.errorMsg != "" {
					assert.Contains(t, validateErr.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, validateErr, "Should not have validation error")
			}
		})
	}
}

// TestEndpointRequestSources tests different request sources (body, path, query, header).
func TestEndpointRequestSources(t *testing.T) {
	source := `
database postgres {
    model User {
        id: string, primary
        email: string, required
    }

    model SearchParams {
        query: string
    }

    model AuthHeader {
        token: string
    }
}

endpoint POST "/users" {
  request User from body
  response User status 201
}

endpoint GET "/users/:id" {
  request User from path
  response User status 200
}

endpoint GET "/search" {
  request SearchParams from query
  response User status 200
}

endpoint GET "/profile" {
  request AuthHeader from header
  response User status 200
}
`

	// Parse the source
	program, err := parser.Parse(source)
	require.NoError(t, err, "Should parse endpoints with different request sources")

	// Extract endpoints and verify request sources
	endpoints := []*ast.EndpointDecl{}
	for _, stmt := range program.Statements {
		if endpoint, ok := stmt.(*ast.EndpointDecl); ok {
			endpoints = append(endpoints, endpoint)
		}
	}

	require.Len(t, endpoints, 4, "Should have 4 endpoints")

	// Verify request sources
	sources := []ast.RequestSource{
		endpoints[0].Handler.Request.Source,
		endpoints[1].Handler.Request.Source,
		endpoints[2].Handler.Request.Source,
		endpoints[3].Handler.Request.Source,
	}

	assert.Contains(t, sources, ast.RequestSourceBody)
	assert.Contains(t, sources, ast.RequestSourcePath)
	assert.Contains(t, sources, ast.RequestSourceQuery)
	assert.Contains(t, sources, ast.RequestSourceHeader)

	// Validate the program
	v := validator.New()
	err = v.Validate(program)
	assert.NoError(t, err, "Endpoints with different request sources should validate")
}

// TestEndpointStatusCodes tests various HTTP status codes.
func TestEndpointStatusCodes(t *testing.T) {
	source := `
database postgres {
    model User {
        id: string, primary
    }

    model Error {
        message: string
    }
}

endpoint GET "/users" {
  response User status 200
}

endpoint POST "/users" {
  request User from body
  response User status 201
}

endpoint DELETE "/users/:id" {
  response User status 204
}

endpoint GET "/not-found" {
  response Error status 404
}

endpoint GET "/error" {
  response Error status 500
}
`

	// Parse the source
	program, err := parser.Parse(source)
	require.NoError(t, err, "Should parse endpoints with different status codes")

	// Extract endpoints and verify status codes
	endpoints := []*ast.EndpointDecl{}
	for _, stmt := range program.Statements {
		if endpoint, ok := stmt.(*ast.EndpointDecl); ok {
			endpoints = append(endpoints, endpoint)
		}
	}

	require.Len(t, endpoints, 5, "Should have 5 endpoints")

	// Verify status codes
	statusCodes := []int{
		endpoints[0].Handler.Response.StatusCode,
		endpoints[1].Handler.Response.StatusCode,
		endpoints[2].Handler.Response.StatusCode,
		endpoints[3].Handler.Response.StatusCode,
		endpoints[4].Handler.Response.StatusCode,
	}

	assert.Contains(t, statusCodes, 200)
	assert.Contains(t, statusCodes, 201)
	assert.Contains(t, statusCodes, 204)
	assert.Contains(t, statusCodes, 404)
	assert.Contains(t, statusCodes, 500)

	// Validate the program
	v := validator.New()
	err = v.Validate(program)
	assert.NoError(t, err, "Endpoints with different status codes should validate")
}

// TestCompleteEndpointPipeline tests the complete pipeline from parse to validate for complex endpoints.
func TestCompleteEndpointPipeline(t *testing.T) {
	source := `
database postgres {
    model User {
        id: string, primary
        email: string, required
        name: string, required
    }

    model CreateUser {
        email: string, required
        name: string, required
    }
}

middleware auth {
    type authentication
    config {
        method: jwt
        secret: env("JWT_SECRET")
    }
}

middleware rate_limit {
  type: rate_limiter
  requests: 100
  window: "1m"
}

@auth("admin")
endpoint POST "/admin/users" {
  middleware auth
  middleware rate_limit
  request CreateUser from body
  response User status 201

  do {
    validate(request)
    authorize(request, "admin")
    user = create(User, request)
    emit("user.created")
  }
}
`

	// Step 1: Parse .cai file
	program, err := parser.Parse(source)
	require.NoError(t, err, "Should parse complex endpoint")
	require.NotNil(t, program, "Program should not be nil")

	// Step 2: Validate AST
	v := validator.New()
	err = v.Validate(program)
	require.NoError(t, err, "Should validate complex endpoint")

	// Step 3: Verify endpoint definitions
	endpoints := []*ast.EndpointDecl{}
	for _, stmt := range program.Statements {
		if endpoint, ok := stmt.(*ast.EndpointDecl); ok {
			endpoints = append(endpoints, endpoint)
		}
	}

	require.Len(t, endpoints, 1, "Should have 1 endpoint")
	endpoint := endpoints[0]

	// Step 4: Verify type references
	assert.Equal(t, "CreateUser", endpoint.Handler.Request.TypeName)
	assert.Equal(t, "User", endpoint.Handler.Response.TypeName)

	// Step 5: Verify middleware references
	require.Len(t, endpoint.Middlewares, 2)
	assert.Equal(t, "auth", endpoint.Middlewares[0].Name)
	assert.Equal(t, "rate_limit", endpoint.Middlewares[1].Name)

	// Step 6: Verify annotations
	require.Len(t, endpoint.Annotations, 1)
	assert.Equal(t, "auth", endpoint.Annotations[0].Name)
	assert.Equal(t, "admin", endpoint.Annotations[0].Value)

	// Step 7: Verify logic steps
	require.NotNil(t, endpoint.Handler.Logic)
	require.Len(t, endpoint.Handler.Logic.Steps, 4)
	assert.Equal(t, "validate", endpoint.Handler.Logic.Steps[0].Action)
	assert.Equal(t, "authorize", endpoint.Handler.Logic.Steps[1].Action)
	assert.Equal(t, "create", endpoint.Handler.Logic.Steps[2].Action)
	assert.Equal(t, "emit", endpoint.Handler.Logic.Steps[3].Action)

	t.Logf("Successfully validated complete endpoint pipeline")
}
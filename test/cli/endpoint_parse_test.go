//go:build integration

// Package cli provides CLI integration tests for CodeAI endpoint parsing.
package cli

import (
	"testing"

	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndpointParsing tests the endpoint parser functionality.
func TestEndpointParsing(t *testing.T) {
	t.Run("parse simple GET endpoint", func(t *testing.T) {
		input := `endpoint GET "/users" {
			response UserList status 200
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		assert.Equal(t, ast.HTTPMethodGET, endpoint.Method)
		assert.Equal(t, "/users", endpoint.Path)
		assert.NotNil(t, endpoint.Handler)
		assert.NotNil(t, endpoint.Handler.Response)
		assert.Equal(t, "UserList", endpoint.Handler.Response.TypeName)
		assert.Equal(t, 200, endpoint.Handler.Response.StatusCode)
	})

	t.Run("parse POST endpoint with body", func(t *testing.T) {
		input := `endpoint POST "/users" {
			request CreateUser from body
			response User status 201
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		assert.Equal(t, ast.HTTPMethodPOST, endpoint.Method)
		assert.NotNil(t, endpoint.Handler.Request)
		assert.Equal(t, "CreateUser", endpoint.Handler.Request.TypeName)
		assert.Equal(t, ast.RequestSourceBody, endpoint.Handler.Request.Source)
		assert.Equal(t, 201, endpoint.Handler.Response.StatusCode)
	})

	t.Run("parse PUT endpoint with path param", func(t *testing.T) {
		input := `endpoint PUT "/users/:id" {
			request UpdateUser from body
			response User status 200
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		assert.Equal(t, ast.HTTPMethodPUT, endpoint.Method)
		assert.Equal(t, "/users/:id", endpoint.Path)
	})

	t.Run("parse DELETE endpoint", func(t *testing.T) {
		input := `endpoint DELETE "/users/:id" {
			request UserID from path
			response Empty status 204
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		assert.Equal(t, ast.HTTPMethodDELETE, endpoint.Method)
		assert.Equal(t, ast.RequestSourcePath, endpoint.Handler.Request.Source)
		assert.Equal(t, 204, endpoint.Handler.Response.StatusCode)
	})

	t.Run("parse PATCH endpoint", func(t *testing.T) {
		input := `endpoint PATCH "/users/:id" {
			request PatchUser from body
			response User status 200
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		assert.Equal(t, ast.HTTPMethodPATCH, endpoint.Method)
	})

	t.Run("parse GET with query params", func(t *testing.T) {
		input := `endpoint GET "/users" {
			request SearchParams from query
			response UserList status 200
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		assert.Equal(t, ast.RequestSourceQuery, endpoint.Handler.Request.Source)
	})

	t.Run("parse GET with header", func(t *testing.T) {
		input := `endpoint GET "/users/me" {
			request AuthToken from header
			response User status 200
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		assert.Equal(t, ast.RequestSourceHeader, endpoint.Handler.Request.Source)
	})
}

// TestEndpointMiddlewares tests middleware parsing for endpoints.
func TestEndpointMiddlewares(t *testing.T) {
	t.Run("parse endpoint with single middleware", func(t *testing.T) {
		input := `endpoint POST "/admin/users" {
			middleware auth
			request CreateUser from body
			response User status 201
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		require.Len(t, endpoint.Middlewares, 1)
		assert.Equal(t, "auth", endpoint.Middlewares[0].Name)
	})

	t.Run("parse endpoint with multiple middlewares", func(t *testing.T) {
		input := `endpoint DELETE "/admin/users/:id" {
			middleware auth
			middleware admin
			middleware logging
			request UserID from path
			response Empty status 204
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		require.Len(t, endpoint.Middlewares, 3)
		assert.Equal(t, "auth", endpoint.Middlewares[0].Name)
		assert.Equal(t, "admin", endpoint.Middlewares[1].Name)
		assert.Equal(t, "logging", endpoint.Middlewares[2].Name)
	})
}

// TestEndpointAnnotations tests annotation parsing for endpoints.
func TestEndpointAnnotations(t *testing.T) {
	t.Run("parse endpoint with @deprecated annotation", func(t *testing.T) {
		input := `@deprecated
		endpoint GET "/old/users" {
			response UserList status 200
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		require.Len(t, endpoint.Annotations, 1)
		assert.Equal(t, "deprecated", endpoint.Annotations[0].Name)
		assert.Empty(t, endpoint.Annotations[0].Value)
	})

	t.Run("parse endpoint with @auth annotation with value", func(t *testing.T) {
		input := `@auth("admin")
		endpoint DELETE "/users/:id" {
			request UserID from path
			response Empty status 204
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		require.Len(t, endpoint.Annotations, 1)
		assert.Equal(t, "auth", endpoint.Annotations[0].Name)
		assert.Equal(t, "admin", endpoint.Annotations[0].Value)
	})

	t.Run("parse endpoint with multiple annotations", func(t *testing.T) {
		input := `@deprecated
		@auth("admin")
		@rateLimit("100")
		endpoint GET "/api/v1/data" {
			response Data status 200
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		require.Len(t, endpoint.Annotations, 3)
		assert.Equal(t, "deprecated", endpoint.Annotations[0].Name)
		assert.Equal(t, "auth", endpoint.Annotations[1].Name)
		assert.Equal(t, "admin", endpoint.Annotations[1].Value)
		assert.Equal(t, "rateLimit", endpoint.Annotations[2].Name)
		assert.Equal(t, "100", endpoint.Annotations[2].Value)
	})
}

// TestEndpointLogic tests handler logic parsing for endpoints.
func TestEndpointLogic(t *testing.T) {
	t.Run("parse endpoint with logic block", func(t *testing.T) {
		input := `endpoint GET "/users/:id" {
			request UserID from path
			response User status 200
			do {
				validate(request)
			}
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		require.NotNil(t, endpoint.Handler.Logic)
		require.Len(t, endpoint.Handler.Logic.Steps, 1)
		assert.Equal(t, "validate", endpoint.Handler.Logic.Steps[0].Action)
	})

	t.Run("parse endpoint with assignment logic", func(t *testing.T) {
		input := `endpoint GET "/users/:id" {
			request UserID from path
			response UserDetail status 200
			do {
				user = find(User, id)
			}
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		require.Len(t, endpoint.Handler.Logic.Steps, 1)
		assert.Equal(t, "user", endpoint.Handler.Logic.Steps[0].Target)
		assert.Equal(t, "find", endpoint.Handler.Logic.Steps[0].Action)
	})

	t.Run("parse endpoint with multiple logic steps", func(t *testing.T) {
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

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		require.Len(t, endpoint.Handler.Logic.Steps, 4)
		assert.Equal(t, "validate", endpoint.Handler.Logic.Steps[0].Action)
		assert.Equal(t, "authorize", endpoint.Handler.Logic.Steps[1].Action)
		assert.Equal(t, "create", endpoint.Handler.Logic.Steps[2].Action)
		assert.Equal(t, "user", endpoint.Handler.Logic.Steps[2].Target)
		assert.Equal(t, "send", endpoint.Handler.Logic.Steps[3].Action)
	})

	t.Run("parse endpoint with where clause in logic", func(t *testing.T) {
		input := `endpoint GET "/users" {
			request SearchParams from query
			response UserList status 200
			do {
				users = find(User) where "active = true"
			}
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		require.Len(t, endpoint.Handler.Logic.Steps, 1)
		assert.Equal(t, "active = true", endpoint.Handler.Logic.Steps[0].Condition)
	})

	t.Run("parse endpoint with options in logic", func(t *testing.T) {
		input := `endpoint GET "/users/:id" {
			request UserID from path
			response User status 200
			do {
				user = find(User, id) with { cache: true }
			}
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		require.Len(t, endpoint.Handler.Logic.Steps[0].Options, 1)
		assert.Equal(t, "cache", endpoint.Handler.Logic.Steps[0].Options[0].Key)
	})
}

// TestMultipleEndpoints tests parsing multiple endpoints.
func TestMultipleEndpoints(t *testing.T) {
	t.Run("parse multiple endpoints", func(t *testing.T) {
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

		endpoints, err := parser.ParseEndpoints(input)
		require.NoError(t, err)
		require.Len(t, endpoints, 4)

		// Verify methods
		assert.Equal(t, ast.HTTPMethodGET, endpoints[0].Method)
		assert.Equal(t, ast.HTTPMethodPOST, endpoints[1].Method)
		assert.Equal(t, ast.HTTPMethodGET, endpoints[2].Method)
		assert.Equal(t, ast.HTTPMethodDELETE, endpoints[3].Method)

		// Verify paths
		assert.Equal(t, "/users", endpoints[0].Path)
		assert.Equal(t, "/users", endpoints[1].Path)
		assert.Equal(t, "/users/:id", endpoints[2].Path)
		assert.Equal(t, "/users/:id", endpoints[3].Path)
	})
}

// TestComplexEndpoint tests parsing a complex endpoint with all features.
func TestComplexEndpoint(t *testing.T) {
	t.Run("parse complex endpoint", func(t *testing.T) {
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

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)

		// Verify method and path
		assert.Equal(t, ast.HTTPMethodPUT, endpoint.Method)
		assert.Equal(t, "/admin/users/:id", endpoint.Path)

		// Verify annotations
		require.Len(t, endpoint.Annotations, 2)
		assert.Equal(t, "auth", endpoint.Annotations[0].Name)
		assert.Equal(t, "admin", endpoint.Annotations[0].Value)
		assert.Equal(t, "rateLimit", endpoint.Annotations[1].Name)
		assert.Equal(t, "50", endpoint.Annotations[1].Value)

		// Verify middlewares
		require.Len(t, endpoint.Middlewares, 2)
		assert.Equal(t, "auth", endpoint.Middlewares[0].Name)
		assert.Equal(t, "admin", endpoint.Middlewares[1].Name)

		// Verify request
		require.NotNil(t, endpoint.Handler.Request)
		assert.Equal(t, "UpdateUser", endpoint.Handler.Request.TypeName)
		assert.Equal(t, ast.RequestSourceBody, endpoint.Handler.Request.Source)

		// Verify response
		require.NotNil(t, endpoint.Handler.Response)
		assert.Equal(t, "User", endpoint.Handler.Response.TypeName)
		assert.Equal(t, 200, endpoint.Handler.Response.StatusCode)

		// Verify logic steps
		require.NotNil(t, endpoint.Handler.Logic)
		require.Len(t, endpoint.Handler.Logic.Steps, 5)
		assert.Equal(t, "validate", endpoint.Handler.Logic.Steps[0].Action)
		assert.Equal(t, "authorize", endpoint.Handler.Logic.Steps[1].Action)
		assert.Equal(t, "find", endpoint.Handler.Logic.Steps[2].Action)
		assert.Equal(t, "user", endpoint.Handler.Logic.Steps[2].Target)
		assert.Equal(t, "deleted_at IS NULL", endpoint.Handler.Logic.Steps[2].Condition)
		assert.Equal(t, "update", endpoint.Handler.Logic.Steps[3].Action)
		assert.Equal(t, "log", endpoint.Handler.Logic.Steps[4].Action)
	})
}

// TestEndpointEdgeCases tests edge cases in endpoint parsing.
func TestEndpointEdgeCases(t *testing.T) {
	t.Run("parse empty endpoint body", func(t *testing.T) {
		input := `endpoint GET "/health" {}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		assert.Nil(t, endpoint.Handler.Request)
		assert.Nil(t, endpoint.Handler.Response)
	})

	t.Run("parse endpoint with only request", func(t *testing.T) {
		input := `endpoint POST "/data" {
			request DataInput from body
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		assert.NotNil(t, endpoint.Handler.Request)
		assert.Nil(t, endpoint.Handler.Response)
	})

	t.Run("parse endpoint with only response", func(t *testing.T) {
		input := `endpoint GET "/status" {
			response Status status 200
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		assert.Nil(t, endpoint.Handler.Request)
		assert.NotNil(t, endpoint.Handler.Response)
	})

	t.Run("parse endpoint with multiple path params", func(t *testing.T) {
		input := `endpoint GET "/orgs/:orgId/users/:userId" {
			response User status 200
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		assert.Equal(t, "/orgs/:orgId/users/:userId", endpoint.Path)
	})

	t.Run("parse endpoint with comments", func(t *testing.T) {
		input := `// This is a user endpoint
		endpoint GET "/users" {
			// Return all users
			response UserList status 200
		}`

		endpoint, err := parser.ParseEndpoint(input)
		require.NoError(t, err)
		assert.Equal(t, ast.HTTPMethodGET, endpoint.Method)
	})

	t.Run("parse empty endpoints returns empty slice", func(t *testing.T) {
		input := ""

		endpoints, err := parser.ParseEndpoints(input)
		require.NoError(t, err)
		assert.Empty(t, endpoints)
	})
}

// TestEndpointStatusCodes tests various status codes.
func TestEndpointStatusCodes(t *testing.T) {
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
		t.Run("status "+string(rune(tt.status)), func(t *testing.T) {
			endpoint, err := parser.ParseEndpoint(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.status, endpoint.Handler.Response.StatusCode)
		})
	}
}

// TestEndpointParsingErrors tests error handling in endpoint parsing.
func TestEndpointParsingErrors(t *testing.T) {
	t.Run("invalid HTTP method", func(t *testing.T) {
		input := `endpoint INVALID "/test" {
			response Test status 200
		}`

		_, err := parser.ParseEndpoint(input)
		assert.Error(t, err)
	})

	t.Run("invalid request source", func(t *testing.T) {
		input := `endpoint GET "/test" {
			request Test from invalid
			response Test status 200
		}`

		_, err := parser.ParseEndpoint(input)
		assert.Error(t, err)
	})

	t.Run("missing closing brace", func(t *testing.T) {
		input := `endpoint GET "/test" {
			response Test status 200`

		_, err := parser.ParseEndpoint(input)
		assert.Error(t, err)
	})

	t.Run("missing path", func(t *testing.T) {
		input := `endpoint GET {
			response Test status 200
		}`

		_, err := parser.ParseEndpoint(input)
		assert.Error(t, err)
	})
}

// TestEndpointStringRepresentation tests the String() method.
func TestEndpointStringRepresentation(t *testing.T) {
	input := `endpoint GET "/users" {
		response UserList status 200
	}`

	endpoint, err := parser.ParseEndpoint(input)
	require.NoError(t, err)

	str := endpoint.String()
	assert.Contains(t, str, "GET")
	assert.Contains(t, str, "/users")
}

// TestEndpointNodeType tests the Type() method.
func TestEndpointNodeType(t *testing.T) {
	input := `endpoint GET "/users" {
		response UserList status 200
	}`

	endpoint, err := parser.ParseEndpoint(input)
	require.NoError(t, err)

	assert.Equal(t, ast.NodeEndpointDecl, endpoint.Type())
}

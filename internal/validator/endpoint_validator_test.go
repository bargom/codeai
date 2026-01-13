// Package validator provides endpoint validation tests for CodeAI AST.
package validator

import (
	"testing"

	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Basic Endpoint Validation Tests
// =============================================================================

func TestEndpointValidator_ValidEndpoints(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "simple GET endpoint",
			input: `endpoint GET "/users" {
				response UserList status 200
			}`,
		},
		{
			name: "POST endpoint with body",
			input: `endpoint POST "/users" {
				request CreateUser from body
				response User status 201
			}`,
		},
		{
			name: "PUT endpoint with path param",
			input: `endpoint PUT "/users/:id" {
				request UpdateUser from body
				response User status 200
			}`,
		},
		{
			name: "DELETE endpoint",
			input: `endpoint DELETE "/users/:id" {
				request UserID from path
				response Empty status 204
			}`,
		},
		{
			name: "PATCH endpoint",
			input: `endpoint PATCH "/users/:id" {
				request PatchUser from body
				response User status 200
			}`,
		},
		{
			name: "GET with query params",
			input: `endpoint GET "/users" {
				request SearchParams from query
				response UserList status 200
			}`,
		},
		{
			name: "GET with header",
			input: `endpoint GET "/users/me" {
				request AuthToken from header
				response User status 200
			}`,
		},
		{
			name: "endpoint with middleware",
			input: `endpoint POST "/admin/users" {
				middleware auth
				request CreateUser from body
				response User status 201
			}`,
		},
		{
			name: "endpoint with multiple middlewares",
			input: `endpoint DELETE "/admin/users/:id" {
				middleware auth
				middleware admin
				request UserID from path
				response Empty status 204
			}`,
		},
		{
			name: "endpoint with annotation",
			input: `@deprecated
			endpoint GET "/old/users" {
				response UserList status 200
			}`,
		},
		{
			name: "endpoint with auth annotation",
			input: `@auth("admin")
			endpoint DELETE "/users/:id" {
				request UserID from path
				response Empty status 204
			}`,
		},
		{
			name: "endpoint with multiple annotations",
			input: `@deprecated
			@auth("admin")
			endpoint GET "/api/v1/data" {
				response Data status 200
			}`,
		},
		{
			name: "endpoint with logic block",
			input: `endpoint GET "/users/:id" {
				request UserID from path
				response User status 200
				do {
					validate(request)
				}
			}`,
		},
		{
			name: "endpoint with multiple logic steps",
			input: `endpoint POST "/users" {
				request CreateUser from body
				response User status 201
				do {
					validate(request)
					authorize(request, "admin")
					user = create(User, request)
				}
			}`,
		},
		{
			name: "endpoint with path containing multiple params",
			input: `endpoint GET "/orgs/:orgId/users/:userId" {
				response User status 200
			}`,
		},
		{
			name: "health check endpoint (empty body)",
			input: `endpoint GET "/health" {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoints, err := parser.ParseEndpoints(tt.input)
			require.NoError(t, err, "parse error")

			v := NewEndpointValidator()
			err = v.ValidateEndpoints(endpoints)
			assert.NoError(t, err, "validation should pass")
		})
	}
}

// =============================================================================
// HTTP Method Validation Tests
// =============================================================================

func TestEndpointValidator_HTTPMethodValidation(t *testing.T) {
	tests := []struct {
		name   string
		method ast.HTTPMethod
		valid  bool
	}{
		{"GET is valid", ast.HTTPMethodGET, true},
		{"POST is valid", ast.HTTPMethodPOST, true},
		{"PUT is valid", ast.HTTPMethodPUT, true},
		{"DELETE is valid", ast.HTTPMethodDELETE, true},
		{"PATCH is valid", ast.HTTPMethodPATCH, true},
		{"invalid method", ast.HTTPMethod("INVALID"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &ast.EndpointDecl{
				Method: tt.method,
				Path:   "/test",
				Handler: &ast.Handler{
					Response: &ast.ResponseType{
						TypeName:   "Test",
						StatusCode: 200,
					},
				},
			}

			v := NewEndpointValidator()
			err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid HTTP method")
			}
		})
	}
}

// =============================================================================
// Path Validation Tests
// =============================================================================

func TestEndpointValidator_PathValidation(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid path",
			path:        "/users",
			expectError: false,
		},
		{
			name:        "valid path with param",
			path:        "/users/:id",
			expectError: false,
		},
		{
			name:        "valid nested path",
			path:        "/api/v1/users/:id/posts",
			expectError: false,
		},
		{
			name:        "path without leading slash",
			path:        "users",
			expectError: true,
			errorMsg:    "must start with '/'",
		},
		{
			name:        "path with double slashes",
			path:        "/users//id",
			expectError: true,
			errorMsg:    "double slashes",
		},
		{
			name:        "path with duplicate param",
			path:        "/users/:id/posts/:id",
			expectError: true,
			errorMsg:    "duplicate path parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &ast.EndpointDecl{
				Method: ast.HTTPMethodGET,
				Path:   tt.path,
				Handler: &ast.Handler{
					Response: &ast.ResponseType{
						TypeName:   "Test",
						StatusCode: 200,
					},
				},
			}

			v := NewEndpointValidator()
			err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Duplicate Endpoint Tests
// =============================================================================

func TestEndpointValidator_DuplicateEndpoint(t *testing.T) {
	input := `
	endpoint GET "/users" {
		response UserList status 200
	}
	endpoint GET "/users" {
		response UserList status 200
	}`

	endpoints, err := parser.ParseEndpoints(input)
	require.NoError(t, err)

	v := NewEndpointValidator()
	err = v.ValidateEndpoints(endpoints)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate endpoint")
}

func TestEndpointValidator_SamePathDifferentMethods(t *testing.T) {
	input := `
	endpoint GET "/users" {
		response UserList status 200
	}
	endpoint POST "/users" {
		request CreateUser from body
		response User status 201
	}`

	endpoints, err := parser.ParseEndpoints(input)
	require.NoError(t, err)

	v := NewEndpointValidator()
	err = v.ValidateEndpoints(endpoints)
	assert.NoError(t, err, "same path with different methods should be valid")
}

// =============================================================================
// Status Code Validation Tests
// =============================================================================

func TestEndpointValidator_StatusCodeValidation(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		expectError bool
		errorMsg    string
	}{
		{"valid 200", 200, false, ""},
		{"valid 201", 201, false, ""},
		{"valid 204", 204, false, ""},
		{"valid 400", 400, false, ""},
		{"valid 401", 401, false, ""},
		{"valid 403", 403, false, ""},
		{"valid 404", 404, false, ""},
		{"valid 500", 500, false, ""},
		{"valid 100", 100, false, ""},
		{"valid 599", 599, false, ""},
		{"invalid 99", 99, true, "invalid HTTP status code"},
		{"invalid 600", 600, true, "invalid HTTP status code"},
		{"invalid negative", -1, true, "invalid HTTP status code"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &ast.EndpointDecl{
				Method: ast.HTTPMethodGET,
				Path:   "/test",
				Handler: &ast.Handler{
					Response: &ast.ResponseType{
						TypeName:   "Test",
						StatusCode: tt.status,
					},
				},
			}

			v := NewEndpointValidator()
			err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Request Source Validation Tests
// =============================================================================

func TestEndpointValidator_RequestSourceValidation(t *testing.T) {
	tests := []struct {
		name        string
		source      ast.RequestSource
		method      ast.HTTPMethod
		expectError bool
		errorMsg    string
	}{
		{"body with POST is valid", ast.RequestSourceBody, ast.HTTPMethodPOST, false, ""},
		{"body with PUT is valid", ast.RequestSourceBody, ast.HTTPMethodPUT, false, ""},
		{"body with PATCH is valid", ast.RequestSourceBody, ast.HTTPMethodPATCH, false, ""},
		{"query with GET is valid", ast.RequestSourceQuery, ast.HTTPMethodGET, false, ""},
		{"path with GET is valid", ast.RequestSourcePath, ast.HTTPMethodGET, false, ""},
		{"header with GET is valid", ast.RequestSourceHeader, ast.HTTPMethodGET, false, ""},
		{"body with GET is invalid", ast.RequestSourceBody, ast.HTTPMethodGET, true, "should not have a body"},
		{"body with DELETE is invalid", ast.RequestSourceBody, ast.HTTPMethodDELETE, true, "should not have a body"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &ast.EndpointDecl{
				Method: tt.method,
				Path:   "/test",
				Handler: &ast.Handler{
					Request: &ast.RequestType{
						TypeName: "TestRequest",
						Source:   tt.source,
					},
					Response: &ast.ResponseType{
						TypeName:   "TestResponse",
						StatusCode: 200,
					},
				},
			}

			v := NewEndpointValidator()
			err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Type Name Validation Tests
// =============================================================================

func TestEndpointValidator_TypeNameValidation(t *testing.T) {
	tests := []struct {
		name        string
		typeName    string
		expectError bool
	}{
		{"PascalCase is valid", "UserList", false},
		{"single word PascalCase", "User", false},
		{"with numbers", "User123", false},
		{"lowercase is invalid", "userlist", true},
		{"camelCase is invalid", "userList", true},
		{"snake_case is invalid", "user_list", true},
		{"starts with number", "123User", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &ast.EndpointDecl{
				Method: ast.HTTPMethodGET,
				Path:   "/test",
				Handler: &ast.Handler{
					Response: &ast.ResponseType{
						TypeName:   tt.typeName,
						StatusCode: 200,
					},
				},
			}

			v := NewEndpointValidator()
			err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid response type name")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Middleware Validation Tests
// =============================================================================

func TestEndpointValidator_MiddlewareValidation(t *testing.T) {
	t.Run("duplicate middleware", func(t *testing.T) {
		endpoint := &ast.EndpointDecl{
			Method: ast.HTTPMethodGET,
			Path:   "/test",
			Handler: &ast.Handler{
				Response: &ast.ResponseType{
					TypeName:   "Test",
					StatusCode: 200,
				},
			},
			Middlewares: []*ast.MiddlewareRef{
				{Name: "auth"},
				{Name: "auth"},
			},
		}

		v := NewEndpointValidator()
		err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate middleware")
	})

	t.Run("unknown middleware with strict validation", func(t *testing.T) {
		endpoint := &ast.EndpointDecl{
			Method: ast.HTTPMethodGET,
			Path:   "/test",
			Handler: &ast.Handler{
				Response: &ast.ResponseType{
					TypeName:   "Test",
					StatusCode: 200,
				},
			},
			Middlewares: []*ast.MiddlewareRef{
				{Name: "unknown_middleware"},
			},
		}

		v := NewEndpointValidator()
		v.RegisterMiddleware("auth") // Register known middleware
		err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown middleware")
	})

	t.Run("known middleware with strict validation", func(t *testing.T) {
		endpoint := &ast.EndpointDecl{
			Method: ast.HTTPMethodGET,
			Path:   "/test",
			Handler: &ast.Handler{
				Response: &ast.ResponseType{
					TypeName:   "Test",
					StatusCode: 200,
				},
			},
			Middlewares: []*ast.MiddlewareRef{
				{Name: "auth"},
			},
		}

		v := NewEndpointValidator()
		v.RegisterMiddleware("auth")
		err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})
		assert.NoError(t, err)
	})
}

// =============================================================================
// Annotation Validation Tests
// =============================================================================

func TestEndpointValidator_AnnotationValidation(t *testing.T) {
	t.Run("duplicate annotation", func(t *testing.T) {
		endpoint := &ast.EndpointDecl{
			Method: ast.HTTPMethodGET,
			Path:   "/test",
			Handler: &ast.Handler{
				Response: &ast.ResponseType{
					TypeName:   "Test",
					StatusCode: 200,
				},
			},
			Annotations: []*ast.Annotation{
				{Name: "deprecated"},
				{Name: "deprecated"},
			},
		}

		v := NewEndpointValidator()
		err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate annotation")
	})

	t.Run("auth annotation requires value", func(t *testing.T) {
		endpoint := &ast.EndpointDecl{
			Method: ast.HTTPMethodGET,
			Path:   "/test",
			Handler: &ast.Handler{
				Response: &ast.ResponseType{
					TypeName:   "Test",
					StatusCode: 200,
				},
			},
			Annotations: []*ast.Annotation{
				{Name: "auth", Value: ""},
			},
		}

		v := NewEndpointValidator()
		err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "@auth annotation requires a value")
	})

	t.Run("rate_limit annotation requires numeric value", func(t *testing.T) {
		endpoint := &ast.EndpointDecl{
			Method: ast.HTTPMethodGET,
			Path:   "/test",
			Handler: &ast.Handler{
				Response: &ast.ResponseType{
					TypeName:   "Test",
					StatusCode: 200,
				},
			},
			Annotations: []*ast.Annotation{
				{Name: "rate_limit", Value: "not_a_number"},
			},
		}

		v := NewEndpointValidator()
		err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "@rate_limit annotation value must be numeric")
	})

	t.Run("tag annotation can repeat", func(t *testing.T) {
		endpoint := &ast.EndpointDecl{
			Method: ast.HTTPMethodGET,
			Path:   "/test",
			Handler: &ast.Handler{
				Response: &ast.ResponseType{
					TypeName:   "Test",
					StatusCode: 200,
				},
			},
			Annotations: []*ast.Annotation{
				{Name: "tag", Value: "users"},
				{Name: "tag", Value: "admin"},
			},
		}

		v := NewEndpointValidator()
		err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})
		assert.NoError(t, err, "tag annotation should be repeatable")
	})
}

// =============================================================================
// Handler Logic Validation Tests
// =============================================================================

func TestEndpointValidator_HandlerLogicValidation(t *testing.T) {
	t.Run("valid logic steps", func(t *testing.T) {
		input := `endpoint POST "/users" {
			request CreateUser from body
			response User status 201
			do {
				validate(request)
				user = create(User, request)
			}
		}`

		endpoints, err := parser.ParseEndpoints(input)
		require.NoError(t, err)

		v := NewEndpointValidator()
		err = v.ValidateEndpoints(endpoints)
		assert.NoError(t, err)
	})

	t.Run("unknown action with strict validation", func(t *testing.T) {
		endpoint := &ast.EndpointDecl{
			Method: ast.HTTPMethodGET,
			Path:   "/test",
			Handler: &ast.Handler{
				Response: &ast.ResponseType{
					TypeName:   "Test",
					StatusCode: 200,
				},
				Logic: &ast.HandlerLogic{
					Steps: []*ast.LogicStep{
						{Action: "unknown_action"},
					},
				},
			},
		}

		v := NewEndpointValidator()
		v.RegisterAction("validate") // Register known action
		err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown action")
	})
}

// =============================================================================
// Type Registration Tests
// =============================================================================

func TestEndpointValidator_TypeRegistration(t *testing.T) {
	t.Run("unknown request type with strict validation", func(t *testing.T) {
		endpoint := &ast.EndpointDecl{
			Method: ast.HTTPMethodPOST,
			Path:   "/test",
			Handler: &ast.Handler{
				Request: &ast.RequestType{
					TypeName: "UnknownType",
					Source:   ast.RequestSourceBody,
				},
				Response: &ast.ResponseType{
					TypeName:   "Test",
					StatusCode: 200,
				},
			},
		}

		v := NewEndpointValidator()
		v.RegisterType("Test")
		v.RegisterType("KnownType")
		err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown request type")
	})

	t.Run("unknown response type with strict validation", func(t *testing.T) {
		endpoint := &ast.EndpointDecl{
			Method: ast.HTTPMethodGET,
			Path:   "/test",
			Handler: &ast.Handler{
				Response: &ast.ResponseType{
					TypeName:   "UnknownType",
					StatusCode: 200,
				},
			},
		}

		v := NewEndpointValidator()
		v.RegisterType("KnownType")
		err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown response type")
	})
}

// =============================================================================
// Status Code Method Consistency Tests
// =============================================================================

func TestEndpointValidator_StatusCodeMethodConsistency(t *testing.T) {
	t.Run("DELETE with 201 status is unusual", func(t *testing.T) {
		endpoint := &ast.EndpointDecl{
			Method: ast.HTTPMethodDELETE,
			Path:   "/test/:id",
			Handler: &ast.Handler{
				Request: &ast.RequestType{
					TypeName: "ID",
					Source:   ast.RequestSourcePath,
				},
				Response: &ast.ResponseType{
					TypeName:   "Created",
					StatusCode: 201,
				},
			},
		}

		v := NewEndpointValidator()
		err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "201 Created which is unusual")
	})
}

// =============================================================================
// Nil Handler Tests
// =============================================================================

func TestEndpointValidator_NilHandler(t *testing.T) {
	endpoint := &ast.EndpointDecl{
		Method:  ast.HTTPMethodGET,
		Path:    "/test",
		Handler: nil,
	}

	v := NewEndpointValidator()
	err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})
	assert.NoError(t, err, "nil handler should not cause error")
}

func TestEndpointValidator_NilEndpoint(t *testing.T) {
	v := NewEndpointValidator()
	err := v.ValidateEndpoints([]*ast.EndpointDecl{nil})
	assert.NoError(t, err, "nil endpoint should not cause error")
}

func TestEndpointValidator_EmptySlice(t *testing.T) {
	v := NewEndpointValidator()
	err := v.ValidateEndpoints([]*ast.EndpointDecl{})
	assert.NoError(t, err, "empty slice should not cause error")
}

// =============================================================================
// Complex Endpoint Tests
// =============================================================================

func TestEndpointValidator_ComplexEndpoint(t *testing.T) {
	input := `@auth("admin")
	@rate_limit("50")
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

	endpoints, err := parser.ParseEndpoints(input)
	require.NoError(t, err)
	require.Len(t, endpoints, 1)

	v := NewEndpointValidator()
	err = v.ValidateEndpoints(endpoints)
	assert.NoError(t, err, "complex endpoint should validate successfully")
}

// =============================================================================
// Multiple Endpoints Tests
// =============================================================================

func TestEndpointValidator_MultipleEndpoints(t *testing.T) {
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

	endpoint PUT "/users/:id" {
		request UpdateUser from body
		response User status 200
	}

	endpoint DELETE "/users/:id" {
		request UserID from path
		response Empty status 204
	}`

	endpoints, err := parser.ParseEndpoints(input)
	require.NoError(t, err)
	require.Len(t, endpoints, 5)

	v := NewEndpointValidator()
	err = v.ValidateEndpoints(endpoints)
	assert.NoError(t, err, "multiple valid endpoints should validate successfully")
}

// =============================================================================
// Version Annotation Tests
// =============================================================================

func TestEndpointValidator_VersionAnnotation(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectError bool
	}{
		{"v1 is valid", "v1", false},
		{"v2 is valid", "v2", false},
		{"v1.0 is valid", "v1.0", false},
		{"v2.1.3 is valid", "v2.1.3", false},
		{"1 without v is valid", "1", false},
		{"1.0 without v is valid", "1.0", false},
		{"invalid version", "latest", true},
		{"invalid alpha", "v1-alpha", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &ast.EndpointDecl{
				Method: ast.HTTPMethodGET,
				Path:   "/test",
				Handler: &ast.Handler{
					Response: &ast.ResponseType{
						TypeName:   "Test",
						StatusCode: 200,
					},
				},
				Annotations: []*ast.Annotation{
					{Name: "version", Value: tt.version},
				},
			}

			v := NewEndpointValidator()
			err := v.ValidateEndpoints([]*ast.EndpointDecl{endpoint})

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "@version")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkEndpointValidator_Single(b *testing.B) {
	input := `endpoint GET "/users" {
		response UserList status 200
	}`

	endpoints, _ := parser.ParseEndpoints(input)

	for i := 0; i < b.N; i++ {
		v := NewEndpointValidator()
		_ = v.ValidateEndpoints(endpoints)
	}
}

func BenchmarkEndpointValidator_Complex(b *testing.B) {
	input := `@auth("admin")
	@rate_limit("50")
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

	endpoints, _ := parser.ParseEndpoints(input)

	for i := 0; i < b.N; i++ {
		v := NewEndpointValidator()
		_ = v.ValidateEndpoints(endpoints)
	}
}

func BenchmarkEndpointValidator_Multiple(b *testing.B) {
	input := `
	endpoint GET "/users" { response UserList status 200 }
	endpoint POST "/users" { request CreateUser from body response User status 201 }
	endpoint GET "/users/:id" { request UserID from path response UserDetail status 200 }
	endpoint PUT "/users/:id" { request UpdateUser from body response User status 200 }
	endpoint DELETE "/users/:id" { request UserID from path response Empty status 204 }
	`

	endpoints, _ := parser.ParseEndpoints(input)

	for i := 0; i < b.N; i++ {
		v := NewEndpointValidator()
		_ = v.ValidateEndpoints(endpoints)
	}
}

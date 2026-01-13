package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bargom/codeai/internal/ast"
)

// Tests for endpoint integration in main parser
func TestMainParser_SimpleGETEndpoint(t *testing.T) {
	input := `
config {
	database_type: "postgres"
}

endpoint GET "/users" {
	response UserList status 200
}
`

	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 2)

	// Check config parsed correctly
	configDecl := program.Statements[0].(*ast.ConfigDecl)
	assert.Equal(t, ast.DatabaseTypePostgres, configDecl.DatabaseType)

	// Check endpoint parsed correctly
	endpointDecl := program.Statements[1].(*ast.EndpointDecl)
	assert.Equal(t, ast.HTTPMethodGET, endpointDecl.Method)
	assert.Equal(t, "/users", endpointDecl.Path)
	assert.NotNil(t, endpointDecl.Handler)
	assert.NotNil(t, endpointDecl.Handler.Response)
	assert.Equal(t, "UserList", endpointDecl.Handler.Response.TypeName)
	assert.Equal(t, 200, endpointDecl.Handler.Response.StatusCode)
}

func TestMainParser_POSTWithMiddleware(t *testing.T) {
	input := `
auth jwt_auth {
	method jwt
	jwks_url "https://auth.example.com/.well-known/jwks.json"
}

middleware rate_limit {
	type rate_limiter
}

endpoint POST "/users" {
	middleware rate_limit
	request CreateUserRequest from body
	response User status 201

	do {
		validate(request)
		insert("users", request)
	}
}
`

	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 3)

	// Check auth
	authDecl := program.Statements[0].(*ast.AuthDecl)
	assert.Equal(t, "jwt_auth", authDecl.Name)

	// Check middleware
	middlewareDecl := program.Statements[1].(*ast.MiddlewareDecl)
	assert.Equal(t, "rate_limit", middlewareDecl.Name)

	// Check endpoint
	endpointDecl := program.Statements[2].(*ast.EndpointDecl)
	assert.Equal(t, ast.HTTPMethodPOST, endpointDecl.Method)
	assert.Equal(t, "/users", endpointDecl.Path)
	assert.Len(t, endpointDecl.Middlewares, 1)
	assert.Equal(t, "rate_limit", endpointDecl.Middlewares[0].Name)

	// Check request
	assert.NotNil(t, endpointDecl.Handler.Request)
	assert.Equal(t, "CreateUserRequest", endpointDecl.Handler.Request.TypeName)
	assert.Equal(t, ast.RequestSourceBody, endpointDecl.Handler.Request.Source)

	// Check response
	assert.NotNil(t, endpointDecl.Handler.Response)
	assert.Equal(t, "User", endpointDecl.Handler.Response.TypeName)
	assert.Equal(t, 201, endpointDecl.Handler.Response.StatusCode)

	// Check logic
	assert.NotNil(t, endpointDecl.Handler.Logic)
	assert.Len(t, endpointDecl.Handler.Logic.Steps, 2)
	assert.Equal(t, "validate", endpointDecl.Handler.Logic.Steps[0].Action)
	assert.Equal(t, "insert", endpointDecl.Handler.Logic.Steps[1].Action)
}

func TestMainParser_EndpointWithLogic(t *testing.T) {
	input := `
endpoint PUT "/users/:id" {
	request UpdateUserRequest from body
	response User status 200

	do {
		validate(request)
		update("users", request)
	}
}
`

	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)

	endpointDecl := program.Statements[0].(*ast.EndpointDecl)
	assert.Equal(t, ast.HTTPMethodPUT, endpointDecl.Method)
	assert.Equal(t, "/users/:id", endpointDecl.Path)

	// Check logic
	assert.NotNil(t, endpointDecl.Handler.Logic)
	assert.Len(t, endpointDecl.Handler.Logic.Steps, 2)

	// First step: validate(request)
	step1 := endpointDecl.Handler.Logic.Steps[0]
	assert.Equal(t, "validate", step1.Action)

	// Second step: update("users", request)
	step2 := endpointDecl.Handler.Logic.Steps[1]
	assert.Equal(t, "update", step2.Action)
}

func TestMainParser_AllHTTPMethods(t *testing.T) {
	methods := []struct {
		method string
		ast    ast.HTTPMethod
	}{
		{"GET", ast.HTTPMethodGET},
		{"POST", ast.HTTPMethodPOST},
		{"PUT", ast.HTTPMethodPUT},
		{"DELETE", ast.HTTPMethodDELETE},
		{"PATCH", ast.HTTPMethodPATCH},
	}

	for _, tc := range methods {
		t.Run(tc.method, func(t *testing.T) {
			input := `endpoint ` + tc.method + ` "/test" {
	response TestResponse status 200
}`

			program, err := Parse(input)
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)

			endpointDecl := program.Statements[0].(*ast.EndpointDecl)
			assert.Equal(t, tc.ast, endpointDecl.Method)
		})
	}
}

func TestMainParser_MultipleMixedStatements(t *testing.T) {
	input := `
config {
	database_type: "mongodb"
	mongodb_uri: "mongodb://localhost:27017"
	mongodb_database: "myapp"
}

database mongodb {
	collection User {
		_id: objectid, primary
		email: string, required
		status: string, required
	}
}

auth api_auth {
	method apikey
}

endpoint GET "/users/:id" {
	request User from path
	response UserDetail status 200
}

endpoint POST "/users" {
	middleware auth
	request CreateUserRequest from body
	response User status 201

	do {
		validate(request)
		insert("users", request)
	}
}
`

	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 5)

	// Verify all statement types are parsed correctly
	assert.IsType(t, &ast.ConfigDecl{}, program.Statements[0])
	assert.IsType(t, &ast.DatabaseBlock{}, program.Statements[1])
	assert.IsType(t, &ast.AuthDecl{}, program.Statements[2])
	assert.IsType(t, &ast.EndpointDecl{}, program.Statements[3])
	assert.IsType(t, &ast.EndpointDecl{}, program.Statements[4])

	// Verify endpoints work correctly alongside other constructs
	endpoint1 := program.Statements[3].(*ast.EndpointDecl)
	assert.Equal(t, ast.HTTPMethodGET, endpoint1.Method)
	assert.Equal(t, "/users/:id", endpoint1.Path)

	endpoint2 := program.Statements[4].(*ast.EndpointDecl)
	assert.Equal(t, ast.HTTPMethodPOST, endpoint2.Method)
	assert.Equal(t, "/users", endpoint2.Path)
	assert.Len(t, endpoint2.Middlewares, 1)
	assert.Equal(t, "auth", endpoint2.Middlewares[0].Name)
}

func TestMainParser_EndpointWithAnnotations(t *testing.T) {
	input := `
@deprecated
@auth("admin")
endpoint DELETE "/admin/users/:id" {
	request DeleteUserRequest from path
	response EmptyResponse status 204
}
`

	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)

	endpointDecl := program.Statements[0].(*ast.EndpointDecl)
	assert.Equal(t, ast.HTTPMethodDELETE, endpointDecl.Method)
	assert.Len(t, endpointDecl.Annotations, 2)

	assert.Equal(t, "deprecated", endpointDecl.Annotations[0].Name)
	assert.Equal(t, "", endpointDecl.Annotations[0].Value)

	assert.Equal(t, "auth", endpointDecl.Annotations[1].Name)
	assert.Equal(t, "admin", endpointDecl.Annotations[1].Value)
}
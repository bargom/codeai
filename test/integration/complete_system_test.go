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

// TestCompleteAppParsing tests parsing the complete e-commerce example application.
func TestCompleteAppParsing(t *testing.T) {
	// Find the complete_app.cai file
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Navigate up to find examples directory
	for {
		examplesPath := filepath.Join(cwd, "examples", "complete_app.cai")
		if _, err := os.Stat(examplesPath); err == nil {
			// Parse the file
			program, err := parser.ParseFile(examplesPath)
			require.NoError(t, err, "Failed to parse complete_app.cai")
			require.NotNil(t, program, "Parsed program should not be nil")

			// Verify we have statements
			assert.Greater(t, len(program.Statements), 0, "Program should have statements")
			return
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Skip("Could not find examples/complete_app.cai in parent directories")
			return
		}
		cwd = parent
	}
}

// TestCompleteAppValidation tests validating the complete e-commerce example application.
func TestCompleteAppValidation(t *testing.T) {
	// Find the complete_app.cai file
	cwd, err := os.Getwd()
	require.NoError(t, err)

	for {
		examplesPath := filepath.Join(cwd, "examples", "complete_app.cai")
		if _, err := os.Stat(examplesPath); err == nil {
			// Parse the file
			program, err := parser.ParseFile(examplesPath)
			require.NoError(t, err, "Failed to parse complete_app.cai")

			// Validate the AST
			v := validator.New()
			err = v.Validate(program)
			assert.Empty(t, err, "Validation should have no errors")
			return
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Skip("Could not find examples/complete_app.cai in parent directories")
			return
		}
		cwd = parent
	}
}

// TestAllExamplesParsing tests that all example files parse correctly.
func TestAllExamplesParsing(t *testing.T) {
	// Examples that should parse successfully
	passingExamples := []string{
		"06-mongodb-collections/mongodb-collections.cai",
		"07-mixed-databases/mixed-databases.cai",
		"08-with-auth/with_auth.cai",
		"10-events-integrations/events.cai",
		"complete_app.cai",
		"with_endpoints.cai",
	}

	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Find examples directory
	var examplesDir string
	for {
		testPath := filepath.Join(cwd, "examples")
		if _, err := os.Stat(testPath); err == nil {
			examplesDir = testPath
			break
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Skip("Could not find examples directory")
			return
		}
		cwd = parent
	}

	for _, example := range passingExamples {
		t.Run(example, func(t *testing.T) {
			fullPath := filepath.Join(examplesDir, example)
			program, err := parser.ParseFile(fullPath)
			assert.NoError(t, err, "Should parse without error")
			assert.NotNil(t, program, "Program should not be nil")
		})
	}
}

// TestDatabaseModels tests parsing and validating database models.
func TestDatabaseModels(t *testing.T) {
	source := `
config {
    database_type: "postgres"
}

database postgres {
    model User {
        id: uuid, primary, auto
        email: string, required, unique
        name: string, required
        role: string, default("user")
        created_at: timestamp, auto

        index: [email]
        index: [role]
    }

    model Post {
        id: uuid, primary, auto
        user_id: ref(User), required
        title: string, required
        content: text, optional
        published: boolean, default(false)
        created_at: timestamp, auto

        index: [user_id]
        index: [published, created_at]
    }
}
`
	program, err := parser.Parse(source)
	require.NoError(t, err, "Should parse database models")
	require.NotNil(t, program)

	v := validator.New()
	err = v.Validate(program)
	assert.Empty(t, err, "Should validate without errors")
}

// TestAuthProviders tests parsing auth providers.
func TestAuthProviders(t *testing.T) {
	source := `
auth jwt_provider {
    method jwt
    jwks_url "https://auth.example.com/.well-known/jwks.json"
    issuer "https://auth.example.com"
    audience "api.example.com"
}

auth google_oauth {
    method oauth2
}

auth api_key_auth {
    method apikey
}

auth basic_auth {
    method basic
}
`
	program, err := parser.Parse(source)
	require.NoError(t, err, "Should parse auth providers")
	require.NotNil(t, program)

	v := validator.New()
	err = v.Validate(program)
	assert.Empty(t, err, "Should validate without errors")
}

// TestRoleDefinitions tests parsing role definitions.
func TestRoleDefinitions(t *testing.T) {
	source := `
role admin {
    permissions ["users:*", "products:*", "orders:*"]
}

role manager {
    permissions ["products:read", "products:write", "orders:read"]
}

role customer {
    permissions ["products:read", "orders:create", "orders:read"]
}

role guest {
    permissions []
}
`
	program, err := parser.Parse(source)
	require.NoError(t, err, "Should parse role definitions")
	require.NotNil(t, program)

	v := validator.New()
	err = v.Validate(program)
	assert.Empty(t, err, "Should validate without errors")
}

// TestMiddlewareDefinitions tests parsing middleware definitions.
func TestMiddlewareDefinitions(t *testing.T) {
	source := `
auth jwt_provider {
    method jwt
    jwks_url "https://auth.example.com/.well-known/jwks.json"
    issuer "https://auth.example.com"
    audience "api.example.com"
}

middleware auth_required {
    type authentication
    config {
        provider: jwt_provider
        required: true
    }
}

middleware rate_limit {
    type rate_limiting
    config {
        requests: 100
        window: "1m"
        strategy: sliding_window
    }
}

middleware cache {
    type cache
    config {
        ttl: 300
        vary_by: "Authorization"
    }
}
`
	program, err := parser.Parse(source)
	require.NoError(t, err, "Should parse middleware definitions")
	require.NotNil(t, program)

	v := validator.New()
	err = v.Validate(program)
	assert.Empty(t, err, "Should validate without errors")
}

// TestEventDefinitions tests parsing event definitions.
func TestEventDefinitions(t *testing.T) {
	source := `
event user_created {
    schema {
        user_id uuid
        email string
        name string
        created_at timestamp
    }
}

event order_created {
    schema {
        order_id uuid
        user_id uuid
        total decimal
        created_at timestamp
    }
}

on "user.created" do workflow "send_welcome_email" async
on "order.created" do webhook "notify_shipping"
`
	program, err := parser.Parse(source)
	require.NoError(t, err, "Should parse event definitions")
	require.NotNil(t, program)

	v := validator.New()
	err = v.Validate(program)
	assert.Empty(t, err, "Should validate without errors")
}

// TestIntegrationDefinitions tests parsing integration definitions.
func TestIntegrationDefinitions(t *testing.T) {
	source := `
integration stripe {
    type rest
    base_url "https://api.stripe.com/v1"
    auth bearer {
        token: "$STRIPE_SECRET_KEY"
    }
    timeout "30s"
    circuit_breaker {
        threshold 5
        timeout "60s"
        max_concurrent 100
    }
}

integration analytics {
    type graphql
    base_url "https://api.analytics.example.com/graphql"
    auth bearer {
        token: "$ANALYTICS_API_KEY"
    }
    timeout "10s"
}
`
	program, err := parser.Parse(source)
	require.NoError(t, err, "Should parse integration definitions")
	require.NotNil(t, program)

	v := validator.New()
	err = v.Validate(program)
	assert.Empty(t, err, "Should validate without errors")
}

// TestWebhookDefinitions tests parsing webhook definitions.
func TestWebhookDefinitions(t *testing.T) {
	source := `
webhook order_notification {
    event "order.created"
    url "https://api.example.com/webhooks/orders"
    method POST
    headers {
        "Content-Type": "application/json"
        "X-Source": "codeai"
    }
    retry 5 initial_interval "1s" backoff 2.0
}

webhook slack_alert {
    event "error.critical"
    url "https://hooks.slack.com/services/XXX"
    method POST
    headers {
        "Content-Type": "application/json"
    }
    retry 3 initial_interval "500ms" backoff 1.5
}
`
	program, err := parser.Parse(source)
	require.NoError(t, err, "Should parse webhook definitions")
	require.NotNil(t, program)

	v := validator.New()
	err = v.Validate(program)
	assert.Empty(t, err, "Should validate without errors")
}

// TestCompleteEcommerceScenario tests a complete e-commerce scenario with all features.
func TestCompleteEcommerceScenario(t *testing.T) {
	source := `
// Configuration
config {
    database_type: "postgres"
}

// Database
database postgres {
    model User {
        id: uuid, primary, auto
        email: string, required, unique
        name: string, required
        role: string, default("customer")
        created_at: timestamp, auto

        index: [email]
    }

    model Product {
        id: uuid, primary, auto
        name: string, required
        price: decimal, required
        stock: int, default(0)
        created_at: timestamp, auto

        index: [name]
    }

    model Order {
        id: uuid, primary, auto
        user_id: ref(User), required
        total: decimal, required
        status: string, default("pending")
        created_at: timestamp, auto

        index: [user_id]
        index: [status]
    }
}

// Auth
auth jwt_provider {
    method jwt
    jwks_url "https://auth.example.com/.well-known/jwks.json"
    issuer "https://auth.example.com"
    audience "api.example.com"
}

// Roles
role admin {
    permissions ["users:*", "products:*", "orders:*"]
}

role customer {
    permissions ["products:read", "orders:create", "orders:read"]
}

// Middleware
middleware auth_required {
    type authentication
    config {
        provider: jwt_provider
        required: true
    }
}

middleware rate_limit {
    type rate_limiting
    config {
        requests: 100
        window: "1m"
        strategy: sliding_window
    }
}

// Events
event user_created {
    schema {
        user_id uuid
        email string
        created_at timestamp
    }
}

event order_created {
    schema {
        order_id uuid
        user_id uuid
        total decimal
        created_at timestamp
    }
}

// Event Handlers
on "user.created" do workflow "send_welcome_email" async
on "order.created" do webhook "notify_shipping"

// Integration
integration stripe {
    type rest
    base_url "https://api.stripe.com/v1"
    auth bearer {
        token: "$STRIPE_SECRET_KEY"
    }
    timeout "30s"
    circuit_breaker {
        threshold 5
        timeout "60s"
        max_concurrent 100
    }
}

// Webhooks
webhook notify_shipping {
    event "order.created"
    url "https://shipping.example.com/webhooks/orders"
    method POST
    headers {
        "Content-Type": "application/json"
    }
    retry 5 initial_interval "1s" backoff 2.0
}
`
	program, err := parser.Parse(source)
	require.NoError(t, err, "Should parse complete e-commerce scenario")
	require.NotNil(t, program)
	assert.Greater(t, len(program.Statements), 0, "Should have statements")

	v := validator.New()
	err = v.Validate(program)
	assert.Empty(t, err, "Should validate without errors")
}

// TestMongoDBCollections tests parsing MongoDB-specific features.
func TestMongoDBCollections(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	for {
		examplesPath := filepath.Join(cwd, "examples", "06-mongodb-collections", "mongodb-collections.cai")
		if _, err := os.Stat(examplesPath); err == nil {
			program, err := parser.ParseFile(examplesPath)
			require.NoError(t, err, "Should parse MongoDB collections example")
			require.NotNil(t, program)

			v := validator.New()
			err = v.Validate(program)
			assert.Empty(t, err, "Should validate without errors")
			return
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Skip("Could not find MongoDB example")
			return
		}
		cwd = parent
	}
}

// TestMixedDatabases tests parsing mixed PostgreSQL and MongoDB features.
func TestMixedDatabases(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	for {
		examplesPath := filepath.Join(cwd, "examples", "07-mixed-databases", "mixed-databases.cai")
		if _, err := os.Stat(examplesPath); err == nil {
			program, err := parser.ParseFile(examplesPath)
			require.NoError(t, err, "Should parse mixed databases example")
			require.NotNil(t, program)

			v := validator.New()
			err = v.Validate(program)
			assert.Empty(t, err, "Should validate without errors")
			return
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Skip("Could not find mixed databases example")
			return
		}
		cwd = parent
	}
}

// TestEndpointsParsing tests parsing the endpoint example file.
func TestEndpointsParsing(t *testing.T) {
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

			// Count different statement types
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

			t.Logf("Found %d models, %d middleware, %d endpoints", modelCount, middlewareCount, endpointCount)
			assert.Greater(t, modelCount, 5, "Should have multiple models")
			assert.Greater(t, middlewareCount, 3, "Should have multiple middleware")
			assert.Greater(t, endpointCount, 8, "Should have multiple endpoints")
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

// TestEndpointsValidation tests validating the endpoint example file.
func TestEndpointsValidation(t *testing.T) {
	// Find the with_endpoints.cai file
	cwd, err := os.Getwd()
	require.NoError(t, err)

	for {
		examplesPath := filepath.Join(cwd, "examples", "with_endpoints.cai")
		if _, err := os.Stat(examplesPath); err == nil {
			// Parse the file
			program, err := parser.ParseFile(examplesPath)
			require.NoError(t, err, "Failed to parse with_endpoints.cai")

			// Validate the AST
			v := validator.New()
			err = v.Validate(program)
			assert.NoError(t, err, "Validation should pass for endpoints example")
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

// TestCompleteEndpointSystemTest tests the complete endpoint system integration.
func TestCompleteEndpointSystemTest(t *testing.T) {
	source := `
model User {
  id: string
  email: string
  name: string
  created_at: timestamp
}

model CreateUserRequest {
  email: string
  name: string
}

model UserList {
  users: array
  total: int
}

middleware auth {
  type: jwt
  secret: env("JWT_SECRET")
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
  request CreateUserRequest from body
  response User status 201

  do {
    validate request
    authorize request "admin"
    user = create User with request
    emit "user.created"
  }
}

endpoint GET "/users" {
  middleware auth
  response UserList status 200

  do {
    authorize request "user"
    users = query User where "active = true"
    paginate users 20 0
  }
}

endpoint GET "/users/:id" {
  middleware auth
  request User from path
  response User status 200

  do {
    validate id
    user = query User where id
    authorize user request
  }
}

endpoint PUT "/users/:id" {
  middleware auth
  request CreateUserRequest from body
  response User status 200

  do {
    validate id
    validate request
    authorize request "user"
    user = update User where id with request
    emit "user.updated"
  }
}

endpoint DELETE "/users/:id" {
  middleware auth
  request User from path
  response User status 204

  do {
    validate id
    authorize request "admin"
    delete User where id
    emit "user.deleted"
  }
}
`

	// Step 1: Parse complete .cai file with endpoints
	program, err := parser.Parse(source)
	require.NoError(t, err, "Should parse complete application with endpoints")
	require.NotNil(t, program, "Program should not be nil")

	// Step 2: Validate AST
	v := validator.New()
	err = v.Validate(program)
	require.NoError(t, err, "Should validate complete application with endpoints")

	// Step 3: Verify endpoint definitions
	endpoints := []*ast.EndpointDecl{}
	models := []*ast.ModelDecl{}
	middleware := []*ast.MiddlewareDecl{}

	for _, stmt := range program.Statements {
		switch s := stmt.(type) {
		case *ast.EndpointDecl:
			endpoints = append(endpoints, s)
		case *ast.ModelDecl:
			models = append(models, s)
		case *ast.MiddlewareDecl:
			middleware = append(middleware, s)
		}
	}

	// Step 4: Verify type references are correctly resolved
	require.Len(t, models, 3, "Should have 3 models")
	require.Len(t, middleware, 2, "Should have 2 middleware")
	require.Len(t, endpoints, 5, "Should have 5 endpoints")

	// Step 5: Verify middleware references in endpoints
	for _, endpoint := range endpoints {
		if len(endpoint.Middlewares) > 0 {
			for _, mw := range endpoint.Middlewares {
				// Middleware names should be defined in the middleware list
				found := false
				for _, def := range middleware {
					if def.Name == mw.Name {
						found = true
						break
					}
				}
				assert.True(t, found, "Middleware %s should be defined", mw.Name)
			}
		}
	}

	// Step 6: Verify request/response type references
	for _, endpoint := range endpoints {
		if endpoint.Handler.Request != nil {
			// Request type should be defined in models
			found := false
			for _, model := range models {
				if model.Name == endpoint.Handler.Request.TypeName {
					found = true
					break
				}
			}
			assert.True(t, found, "Request type %s should be defined", endpoint.Handler.Request.TypeName)
		}

		if endpoint.Handler.Response != nil {
			// Response type should be defined in models
			found := false
			for _, model := range models {
				if model.Name == endpoint.Handler.Response.TypeName {
					found = true
					break
				}
			}
			assert.True(t, found, "Response type %s should be defined", endpoint.Handler.Response.TypeName)
		}
	}

	t.Logf("Successfully validated complete endpoint system integration")
}

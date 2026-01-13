package parser

import (
	"testing"

	"github.com/bargom/codeai/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Auth Parsing Tests
// =============================================================================

func TestParseAuthDecl(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantName   string
		wantMethod ast.AuthMethod
		wantErr    bool
	}{
		{
			name: "auth with jwt method",
			input: `auth jwt_provider {
				method jwt
			}`,
			wantName:   "jwt_provider",
			wantMethod: ast.AuthMethodJWT,
			wantErr:    false,
		},
		{
			name: "auth with oauth2 method",
			input: `auth google_auth {
				method oauth2
			}`,
			wantName:   "google_auth",
			wantMethod: ast.AuthMethodOAuth2,
			wantErr:    false,
		},
		{
			name: "auth with apikey method",
			input: `auth api_auth {
				method apikey
			}`,
			wantName:   "api_auth",
			wantMethod: ast.AuthMethodAPIKey,
			wantErr:    false,
		},
		{
			name: "auth with basic method",
			input: `auth basic_auth {
				method basic
			}`,
			wantName:   "basic_auth",
			wantMethod: ast.AuthMethodBasic,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)
			authDecl, ok := program.Statements[0].(*ast.AuthDecl)
			require.True(t, ok, "expected AuthDecl")
			assert.Equal(t, tt.wantName, authDecl.Name)
			assert.Equal(t, tt.wantMethod, authDecl.Method)
		})
	}
}

func TestParseAuthWithJWKS(t *testing.T) {
	t.Parallel()

	input := `auth jwt_provider {
		method jwt
		jwks_url "https://auth.example.com/.well-known/jwks.json"
		issuer "https://auth.example.com"
		audience "api.example.com"
	}`

	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)

	authDecl, ok := program.Statements[0].(*ast.AuthDecl)
	require.True(t, ok, "expected AuthDecl")
	assert.Equal(t, "jwt_provider", authDecl.Name)
	assert.Equal(t, ast.AuthMethodJWT, authDecl.Method)

	require.NotNil(t, authDecl.JWKS, "expected JWKS config")
	assert.Equal(t, "https://auth.example.com/.well-known/jwks.json", authDecl.JWKS.URL)
	assert.Equal(t, "https://auth.example.com", authDecl.JWKS.Issuer)
	assert.Equal(t, "api.example.com", authDecl.JWKS.Audience)
}

// =============================================================================
// Role Parsing Tests
// =============================================================================

func TestParseRoleDecl(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		input           string
		wantName        string
		wantPermissions []string
		wantErr         bool
	}{
		{
			name: "role with multiple permissions",
			input: `role admin {
				permissions ["users:read", "users:write", "users:delete"]
			}`,
			wantName:        "admin",
			wantPermissions: []string{"users:read", "users:write", "users:delete"},
			wantErr:         false,
		},
		{
			name: "role with single permission",
			input: `role viewer {
				permissions ["users:read"]
			}`,
			wantName:        "viewer",
			wantPermissions: []string{"users:read"},
			wantErr:         false,
		},
		{
			name: "role with empty permissions",
			input: `role guest {
				permissions []
			}`,
			wantName:        "guest",
			wantPermissions: []string{},
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)
			roleDecl, ok := program.Statements[0].(*ast.RoleDecl)
			require.True(t, ok, "expected RoleDecl")
			assert.Equal(t, tt.wantName, roleDecl.Name)
			assert.Equal(t, tt.wantPermissions, roleDecl.Permissions)
		})
	}
}

func TestParseMultipleRoles(t *testing.T) {
	t.Parallel()

	input := `
role admin {
	permissions ["users:read", "users:write", "users:delete"]
}

role editor {
	permissions ["users:read", "users:write"]
}

role viewer {
	permissions ["users:read"]
}
`

	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 3)

	// Check each role
	for i, expected := range []struct {
		name        string
		permissions []string
	}{
		{"admin", []string{"users:read", "users:write", "users:delete"}},
		{"editor", []string{"users:read", "users:write"}},
		{"viewer", []string{"users:read"}},
	} {
		roleDecl, ok := program.Statements[i].(*ast.RoleDecl)
		require.True(t, ok, "expected RoleDecl at index %d", i)
		assert.Equal(t, expected.name, roleDecl.Name)
		assert.Equal(t, expected.permissions, roleDecl.Permissions)
	}
}

// =============================================================================
// Middleware Parsing Tests
// =============================================================================

func TestParseMiddlewareDecl(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		wantName       string
		wantType       string
		wantConfigKeys []string
		wantErr        bool
	}{
		{
			name: "middleware with type only",
			input: `middleware auth_check {
				type authentication
			}`,
			wantName:       "auth_check",
			wantType:       "authentication",
			wantConfigKeys: nil,
			wantErr:        false,
		},
		{
			name: "middleware with config",
			input: `middleware rate_limit {
				type rate_limiting
				config {
					requests: 100
					window: "1m"
				}
			}`,
			wantName:       "rate_limit",
			wantType:       "rate_limiting",
			wantConfigKeys: []string{"requests", "window"},
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)
			middlewareDecl, ok := program.Statements[0].(*ast.MiddlewareDecl)
			require.True(t, ok, "expected MiddlewareDecl")
			assert.Equal(t, tt.wantName, middlewareDecl.Name)
			assert.Equal(t, tt.wantType, middlewareDecl.MiddlewareType)

			if tt.wantConfigKeys != nil {
				for _, key := range tt.wantConfigKeys {
					_, exists := middlewareDecl.Config[key]
					assert.True(t, exists, "expected config key %q", key)
				}
			}
		})
	}
}

func TestParseRateLimitMiddleware(t *testing.T) {
	t.Parallel()

	input := `middleware rate_limit {
		type rate_limiting
		config {
			requests: 100
			window: "1m"
			strategy: sliding_window
		}
	}`

	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)

	middlewareDecl, ok := program.Statements[0].(*ast.MiddlewareDecl)
	require.True(t, ok, "expected MiddlewareDecl")
	assert.Equal(t, "rate_limit", middlewareDecl.Name)
	assert.Equal(t, "rate_limiting", middlewareDecl.MiddlewareType)

	// Check config values
	requests, ok := middlewareDecl.Config["requests"].(*ast.NumberLiteral)
	require.True(t, ok, "expected NumberLiteral for requests")
	assert.Equal(t, 100.0, requests.Value)

	window, ok := middlewareDecl.Config["window"].(*ast.StringLiteral)
	require.True(t, ok, "expected StringLiteral for window")
	assert.Equal(t, "1m", window.Value)

	strategy, ok := middlewareDecl.Config["strategy"].(*ast.Identifier)
	require.True(t, ok, "expected Identifier for strategy")
	assert.Equal(t, "sliding_window", strategy.Name)
}

// =============================================================================
// Combined Auth & Middleware Tests
// =============================================================================

func TestParseAuthWithMiddleware(t *testing.T) {
	t.Parallel()

	input := `
auth jwt_provider {
	method jwt
	jwks_url "https://auth.example.com/.well-known/jwks.json"
	issuer "https://auth.example.com"
	audience "api.example.com"
}

role admin {
	permissions ["users:read", "users:write", "users:delete"]
}

role viewer {
	permissions ["users:read"]
}

middleware auth_check {
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
	}
}
`

	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 5)

	// Check auth declaration
	authDecl, ok := program.Statements[0].(*ast.AuthDecl)
	require.True(t, ok, "expected AuthDecl")
	assert.Equal(t, "jwt_provider", authDecl.Name)
	assert.Equal(t, ast.AuthMethodJWT, authDecl.Method)

	// Check first role
	role1, ok := program.Statements[1].(*ast.RoleDecl)
	require.True(t, ok, "expected RoleDecl")
	assert.Equal(t, "admin", role1.Name)

	// Check second role
	role2, ok := program.Statements[2].(*ast.RoleDecl)
	require.True(t, ok, "expected RoleDecl")
	assert.Equal(t, "viewer", role2.Name)

	// Check auth middleware
	authMiddleware, ok := program.Statements[3].(*ast.MiddlewareDecl)
	require.True(t, ok, "expected MiddlewareDecl")
	assert.Equal(t, "auth_check", authMiddleware.Name)
	assert.Equal(t, "authentication", authMiddleware.MiddlewareType)

	// Check rate limit middleware
	rateLimitMiddleware, ok := program.Statements[4].(*ast.MiddlewareDecl)
	require.True(t, ok, "expected MiddlewareDecl")
	assert.Equal(t, "rate_limit", rateLimitMiddleware.Name)
}

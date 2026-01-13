package validator

import (
	"testing"

	"github.com/bargom/codeai/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMiddleware(t *testing.T) {
	tests := []struct {
		name        string
		middleware  *ast.MiddlewareDecl
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid authentication middleware",
			middleware: &ast.MiddlewareDecl{
				Name:           "auth",
				MiddlewareType: "authentication",
			},
			expectError: false,
		},
		{
			name: "valid rate limiting middleware",
			middleware: &ast.MiddlewareDecl{
				Name:           "rate_limit",
				MiddlewareType: "rate_limiting",
			},
			expectError: false,
		},
		{
			name:        "nil middleware",
			middleware:  nil,
			expectError: true,
			errorMsg:    "middleware is nil",
		},
		{
			name: "empty name",
			middleware: &ast.MiddlewareDecl{
				MiddlewareType: "authentication",
			},
			expectError: true,
			errorMsg:    "middleware name cannot be empty",
		},
		{
			name: "empty type",
			middleware: &ast.MiddlewareDecl{
				Name: "auth",
			},
			expectError: true,
			errorMsg:    "middleware auth: type cannot be empty",
		},
		{
			name: "unknown type",
			middleware: &ast.MiddlewareDecl{
				Name:           "unknown",
				MiddlewareType: "invalid_type",
			},
			expectError: true,
			errorMsg:    "middleware unknown: unknown type 'invalid_type'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMiddleware(tt.middleware)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAuth(t *testing.T) {
	tests := []struct {
		name        string
		auth        *ast.AuthDecl
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid JWT auth",
			auth: &ast.AuthDecl{
				Name:   "jwt_auth",
				Method: ast.AuthMethodJWT,
				JWKS: &ast.JWKSConfig{
					URL:      "https://example.com/.well-known/jwks.json",
					Issuer:   "https://example.com",
					Audience: "api",
				},
			},
			expectError: false,
		},
		{
			name: "valid OAuth2 auth",
			auth: &ast.AuthDecl{
				Name:   "oauth2_auth",
				Method: ast.AuthMethodOAuth2,
			},
			expectError: false,
		},
		{
			name:        "nil auth",
			auth:        nil,
			expectError: true,
			errorMsg:    "auth is nil",
		},
		{
			name: "empty name",
			auth: &ast.AuthDecl{
				Method: ast.AuthMethodJWT,
			},
			expectError: true,
			errorMsg:    "auth provider name cannot be empty",
		},
		{
			name: "JWT with invalid JWKS",
			auth: &ast.AuthDecl{
				Name:   "jwt_auth",
				Method: ast.AuthMethodJWT,
				JWKS: &ast.JWKSConfig{
					URL: "http://insecure.com/jwks.json", // not HTTPS
				},
			},
			expectError: true,
			errorMsg:    "jwks_url must use HTTPS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAuth(tt.auth)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateJWKS(t *testing.T) {
	tests := []struct {
		name        string
		jwks        *ast.JWKSConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid JWKS config",
			jwks: &ast.JWKSConfig{
				URL:      "https://example.com/.well-known/jwks.json",
				Issuer:   "https://example.com",
				Audience: "api",
			},
			expectError: false,
		},
		{
			name:        "nil config",
			jwks:        nil,
			expectError: true,
			errorMsg:    "jwks config is nil",
		},
		{
			name: "non-HTTPS URL",
			jwks: &ast.JWKSConfig{
				URL:      "http://example.com/jwks.json",
				Issuer:   "https://example.com",
				Audience: "api",
			},
			expectError: true,
			errorMsg:    "jwks_url must use HTTPS",
		},
		{
			name: "invalid URL format",
			jwks: &ast.JWKSConfig{
				URL:      "not-a-url",
				Issuer:   "https://example.com",
				Audience: "api",
			},
			expectError: true,
			errorMsg:    "missing scheme",
		},
		{
			name: "empty issuer",
			jwks: &ast.JWKSConfig{
				URL:      "https://example.com/.well-known/jwks.json",
				Issuer:   "",
				Audience: "api",
			},
			expectError: true,
			errorMsg:    "jwks issuer cannot be empty",
		},
		{
			name: "empty audience",
			jwks: &ast.JWKSConfig{
				URL:      "https://example.com/.well-known/jwks.json",
				Issuer:   "https://example.com",
				Audience: "",
			},
			expectError: true,
			errorMsg:    "jwks audience cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJWKS(tt.jwks)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRole(t *testing.T) {
	tests := []struct {
		name        string
		role        *ast.RoleDecl
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid role",
			role: &ast.RoleDecl{
				Name:        "admin",
				Permissions: []string{"users:read", "users:write", "posts:delete"},
			},
			expectError: false,
		},
		{
			name:        "nil role",
			role:        nil,
			expectError: true,
			errorMsg:    "role is nil",
		},
		{
			name: "empty name",
			role: &ast.RoleDecl{
				Permissions: []string{"users:read"},
			},
			expectError: true,
			errorMsg:    "role name cannot be empty",
		},
		{
			name: "no permissions",
			role: &ast.RoleDecl{
				Name:        "empty_role",
				Permissions: []string{},
			},
			expectError: true,
			errorMsg:    "role empty_role: must have at least one permission",
		},
		{
			name: "invalid permission format",
			role: &ast.RoleDecl{
				Name:        "invalid_role",
				Permissions: []string{"invalid_permission"},
			},
			expectError: true,
			errorMsg:    "invalid permission format 'invalid_permission' (must be 'resource:action')",
		},
		{
			name: "mixed valid and invalid permissions",
			role: &ast.RoleDecl{
				Name:        "mixed_role",
				Permissions: []string{"users:read", "invalid", "posts:write"},
			},
			expectError: true,
			errorMsg:    "invalid permission format 'invalid'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRole(tt.role)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMiddlewares(t *testing.T) {
	tests := []struct {
		name        string
		middlewares []*ast.MiddlewareDecl
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid middlewares",
			middlewares: []*ast.MiddlewareDecl{
				{
					Name:           "auth",
					MiddlewareType: "authentication",
				},
				{
					Name:           "rate_limit",
					MiddlewareType: "rate_limiting",
				},
			},
			expectError: false,
		},
		{
			name:        "empty list",
			middlewares: []*ast.MiddlewareDecl{},
			expectError: false,
		},
		{
			name: "duplicate middleware names",
			middlewares: []*ast.MiddlewareDecl{
				{
					Name:           "auth",
					MiddlewareType: "authentication",
				},
				{
					Name:           "auth",
					MiddlewareType: "authorization",
				},
			},
			expectError: true,
			errorMsg:    "duplicate middleware: auth",
		},
		{
			name: "invalid middleware in collection",
			middlewares: []*ast.MiddlewareDecl{
				{
					Name:           "auth",
					MiddlewareType: "authentication",
				},
				{
					Name:           "invalid",
					MiddlewareType: "unknown_type",
				},
			},
			expectError: true,
			errorMsg:    "middleware invalid: unknown type 'unknown_type'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMiddlewares(tt.middlewares)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRoles(t *testing.T) {
	tests := []struct {
		name        string
		roles       []*ast.RoleDecl
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid roles",
			roles: []*ast.RoleDecl{
				{
					Name:        "admin",
					Permissions: []string{"users:read", "users:write"},
				},
				{
					Name:        "user",
					Permissions: []string{"posts:read"},
				},
			},
			expectError: false,
		},
		{
			name:        "empty list",
			roles:       []*ast.RoleDecl{},
			expectError: false,
		},
		{
			name: "duplicate role names",
			roles: []*ast.RoleDecl{
				{
					Name:        "admin",
					Permissions: []string{"users:read"},
				},
				{
					Name:        "admin",
					Permissions: []string{"posts:read"},
				},
			},
			expectError: true,
			errorMsg:    "duplicate role: admin",
		},
		{
			name: "invalid role in collection",
			roles: []*ast.RoleDecl{
				{
					Name:        "admin",
					Permissions: []string{"users:read"},
				},
				{
					Name:        "invalid_role",
					Permissions: []string{"invalid_permission"},
				},
			},
			expectError: true,
			errorMsg:    "invalid permission format 'invalid_permission'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRoles(tt.roles)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
package validator

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/bargom/codeai/internal/ast"
)

var permissionPattern = regexp.MustCompile(`^[a-z_]+:[a-z_]+$`)

// ValidateMiddleware validates a middleware definition
func ValidateMiddleware(mw *ast.MiddlewareDecl) error {
	if mw == nil {
		return fmt.Errorf("middleware is nil")
	}

	if mw.Name == "" {
		return fmt.Errorf("middleware name cannot be empty")
	}

	if mw.MiddlewareType == "" {
		return fmt.Errorf("middleware %s: type cannot be empty", mw.Name)
	}

	// Validate known middleware types
	validTypes := map[string]bool{
		"authentication": true,
		"authorization":  true,
		"rate_limiting":  true,
		"cors":           true,
		"logging":        true,
		"compression":    true,
		"cache":          true,
		"custom":         true,
	}

	if !validTypes[mw.MiddlewareType] {
		return fmt.Errorf("middleware %s: unknown type '%s'", mw.Name, mw.MiddlewareType)
	}

	return nil
}

// ValidateAuth validates an auth provider definition
func ValidateAuth(auth *ast.AuthDecl) error {
	if auth == nil {
		return fmt.Errorf("auth is nil")
	}

	if auth.Name == "" {
		return fmt.Errorf("auth provider name cannot be empty")
	}

	// Validate auth method
	validMethods := map[string]bool{
		"jwt":     true,
		"oauth2":  true,
		"apikey":  true,
		"basic":   true,
	}

	methodStr := string(auth.Method)
	if !validMethods[methodStr] {
		return fmt.Errorf("auth %s: invalid method '%s'", auth.Name, methodStr)
	}

	// Validate JWKS config if present
	if auth.JWKS != nil {
		if err := ValidateJWKS(auth.JWKS); err != nil {
			return fmt.Errorf("auth %s: %w", auth.Name, err)
		}
	}

	return nil
}

// ValidateJWKS validates JWKS configuration
func ValidateJWKS(jwks *ast.JWKSConfig) error {
	if jwks == nil {
		return fmt.Errorf("jwks config is nil")
	}

	// Validate URL format first
	parsedURL, err := url.Parse(jwks.URL)
	if err != nil {
		return fmt.Errorf("invalid jwks_url: %w", err)
	}

	// Check if URL has a scheme (more rigorous validation)
	if parsedURL.Scheme == "" {
		return fmt.Errorf("invalid jwks_url: missing scheme in %s", jwks.URL)
	}

	// Validate JWKS URL is HTTPS
	if !strings.HasPrefix(jwks.URL, "https://") {
		return fmt.Errorf("jwks_url must use HTTPS: %s", jwks.URL)
	}

	if jwks.Issuer == "" {
		return fmt.Errorf("jwks issuer cannot be empty")
	}

	if jwks.Audience == "" {
		return fmt.Errorf("jwks audience cannot be empty")
	}

	return nil
}

// ValidateRole validates a role definition
func ValidateRole(role *ast.RoleDecl) error {
	if role == nil {
		return fmt.Errorf("role is nil")
	}

	if role.Name == "" {
		return fmt.Errorf("role name cannot be empty")
	}

	if len(role.Permissions) == 0 {
		return fmt.Errorf("role %s: must have at least one permission", role.Name)
	}

	// Validate permission format
	for _, perm := range role.Permissions {
		if !permissionPattern.MatchString(perm) {
			return fmt.Errorf("role %s: invalid permission format '%s' (must be 'resource:action')", role.Name, perm)
		}
	}

	return nil
}

// ValidateMiddlewares validates all middleware definitions are unique
func ValidateMiddlewares(middlewares []*ast.MiddlewareDecl) error {
	seen := make(map[string]bool)

	for _, mw := range middlewares {
		if err := ValidateMiddleware(mw); err != nil {
			return err
		}

		if seen[mw.Name] {
			return fmt.Errorf("duplicate middleware: %s", mw.Name)
		}
		seen[mw.Name] = true
	}

	return nil
}

// ValidateRoles validates all role definitions are unique
func ValidateRoles(roles []*ast.RoleDecl) error {
	seen := make(map[string]bool)

	for _, role := range roles {
		if err := ValidateRole(role); err != nil {
			return err
		}

		if seen[role.Name] {
			return fmt.Errorf("duplicate role: %s", role.Name)
		}
		seen[role.Name] = true
	}

	return nil
}
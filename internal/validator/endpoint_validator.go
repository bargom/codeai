// Package validator provides endpoint validation for CodeAI AST.
package validator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/bargom/codeai/internal/ast"
)

// EndpointValidator performs semantic validation on endpoint declarations.
type EndpointValidator struct {
	errors      *ValidationErrors
	endpoints   map[string]*ast.EndpointDecl // key: method + path
	middlewares map[string]bool              // known middleware names
	types       map[string]bool              // known type names
	actions     map[string]bool              // known action names for logic steps
}

// NewEndpointValidator creates a new EndpointValidator instance.
func NewEndpointValidator() *EndpointValidator {
	return &EndpointValidator{
		errors:      &ValidationErrors{},
		endpoints:   make(map[string]*ast.EndpointDecl),
		middlewares: make(map[string]bool),
		types:       make(map[string]bool),
		actions:     make(map[string]bool),
	}
}

// RegisterMiddleware registers a middleware as known/valid.
func (v *EndpointValidator) RegisterMiddleware(name string) {
	v.middlewares[name] = true
}

// RegisterType registers a type as known/valid (for request/response type validation).
func (v *EndpointValidator) RegisterType(name string) {
	v.types[name] = true
}

// RegisterAction registers an action as known/valid (for logic step validation).
func (v *EndpointValidator) RegisterAction(name string) {
	v.actions[name] = true
}

// ValidateEndpoints validates a slice of endpoint declarations.
func (v *EndpointValidator) ValidateEndpoints(decls []*ast.EndpointDecl) error {
	for _, decl := range decls {
		v.validateEndpoint(decl)
	}

	if v.errors.HasErrors() {
		return v.errors
	}
	return nil
}

// validateEndpoint validates a single endpoint declaration.
func (v *EndpointValidator) validateEndpoint(decl *ast.EndpointDecl) {
	if decl == nil {
		return
	}

	// Build unique key for endpoint (method + path)
	key := string(decl.Method) + " " + decl.Path

	// Check for duplicate endpoints
	if existing, exists := v.endpoints[key]; exists {
		v.errors.Add(newSemanticError(decl.Pos(),
			fmt.Sprintf("duplicate endpoint %q; first declared at %s", key, existing.Pos().String())))
		return
	}
	v.endpoints[key] = decl

	// Validate HTTP method
	v.validateHTTPMethod(decl)

	// Validate path
	v.validatePath(decl)

	// Validate middlewares
	v.validateMiddlewares(decl)

	// Validate annotations
	v.validateAnnotations(decl)

	// Validate handler
	v.validateHandler(decl)
}

// validateHTTPMethod validates the HTTP method.
func (v *EndpointValidator) validateHTTPMethod(decl *ast.EndpointDecl) {
	validMethods := map[ast.HTTPMethod]bool{
		ast.HTTPMethodGET:    true,
		ast.HTTPMethodPOST:   true,
		ast.HTTPMethodPUT:    true,
		ast.HTTPMethodDELETE: true,
		ast.HTTPMethodPATCH:  true,
	}

	if !validMethods[decl.Method] {
		v.errors.Add(newSemanticError(decl.Pos(),
			fmt.Sprintf("invalid HTTP method %q; valid methods: GET, POST, PUT, DELETE, PATCH", decl.Method)))
	}
}

// validatePath validates the endpoint path.
func (v *EndpointValidator) validatePath(decl *ast.EndpointDecl) {
	path := decl.Path

	// Path must start with /
	if !strings.HasPrefix(path, "/") {
		v.errors.Add(newSemanticError(decl.Pos(),
			fmt.Sprintf("endpoint path %q must start with '/'", path)))
	}

	// Validate path parameters format
	pathParams := extractPathParams(path)
	seenParams := make(map[string]bool)
	for _, param := range pathParams {
		// Check for duplicate path parameters
		if seenParams[param] {
			v.errors.Add(newSemanticError(decl.Pos(),
				fmt.Sprintf("duplicate path parameter :%s in endpoint path %q", param, path)))
		}
		seenParams[param] = true

		// Validate parameter name format
		if !isValidIdentifier(param) {
			v.errors.Add(newSemanticError(decl.Pos(),
				fmt.Sprintf("invalid path parameter name :%s in endpoint path %q", param, path)))
		}
	}

	// Validate path format (no double slashes, no trailing slash for non-root)
	if strings.Contains(path, "//") {
		v.errors.Add(newSemanticError(decl.Pos(),
			fmt.Sprintf("endpoint path %q contains invalid double slashes", path)))
	}

	// Warning for trailing slash (except root)
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		// This is a soft warning - not an error
	}
}

// extractPathParams extracts parameter names from a path (e.g., "/users/:id" -> ["id"])
var pathParamPattern = regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)

func extractPathParams(path string) []string {
	matches := pathParamPattern.FindAllStringSubmatch(path, -1)
	params := make([]string, len(matches))
	for i, match := range matches {
		params[i] = match[1]
	}
	return params
}

// validateMiddlewares validates endpoint middlewares.
func (v *EndpointValidator) validateMiddlewares(decl *ast.EndpointDecl) {
	seenMiddlewares := make(map[string]bool)

	for _, mw := range decl.Middlewares {
		// Check for duplicate middleware usage
		if seenMiddlewares[mw.Name] {
			v.errors.Add(newSemanticError(decl.Pos(),
				fmt.Sprintf("duplicate middleware %q in endpoint %s %q", mw.Name, decl.Method, decl.Path)))
		}
		seenMiddlewares[mw.Name] = true

		// Validate middleware name format
		if !isValidIdentifier(mw.Name) {
			v.errors.Add(newSemanticError(decl.Pos(),
				fmt.Sprintf("invalid middleware name %q", mw.Name)))
		}

		// Check if middleware is registered (if strict validation is enabled)
		if len(v.middlewares) > 0 && !v.middlewares[mw.Name] {
			v.errors.Add(newSemanticError(decl.Pos(),
				fmt.Sprintf("unknown middleware %q in endpoint %s %q", mw.Name, decl.Method, decl.Path)))
		}
	}
}

// validateAnnotations validates endpoint annotations.
func (v *EndpointValidator) validateAnnotations(decl *ast.EndpointDecl) {
	seenAnnotations := make(map[string]bool)
	validAnnotations := map[string]bool{
		"deprecated":  true,
		"auth":        true,
		"rate_limit":  true,
		"cache":       true,
		"description": true,
		"summary":     true,
		"tag":         true,
		"version":     true,
		"internal":    true,
		"experimental": true,
	}

	for _, ann := range decl.Annotations {
		// Check for duplicate annotations (except for some that can repeat)
		repeatableAnnotations := map[string]bool{"tag": true}
		if seenAnnotations[ann.Name] && !repeatableAnnotations[ann.Name] {
			v.errors.Add(newSemanticError(decl.Pos(),
				fmt.Sprintf("duplicate annotation @%s in endpoint %s %q", ann.Name, decl.Method, decl.Path)))
		}
		seenAnnotations[ann.Name] = true

		// Validate annotation name format
		if !isValidIdentifier(ann.Name) {
			v.errors.Add(newSemanticError(decl.Pos(),
				fmt.Sprintf("invalid annotation name @%s", ann.Name)))
		}

		// Check if annotation is known (soft validation - unknown annotations are warnings)
		if !validAnnotations[ann.Name] {
			// This could be a custom annotation, so just a soft warning
		}

		// Validate annotation-specific requirements
		v.validateAnnotationValue(ann, decl)
	}
}

// validateAnnotationValue validates annotation-specific value requirements.
func (v *EndpointValidator) validateAnnotationValue(ann *ast.Annotation, decl *ast.EndpointDecl) {
	switch ann.Name {
	case "auth":
		// @auth requires a value (the auth provider or role)
		if ann.Value == "" {
			v.errors.Add(newSemanticError(decl.Pos(),
				fmt.Sprintf("@auth annotation requires a value in endpoint %s %q", decl.Method, decl.Path)))
		}
	case "rate_limit":
		// @rate_limit should have a numeric value
		if ann.Value != "" {
			if _, err := strconv.Atoi(ann.Value); err != nil {
				v.errors.Add(newSemanticError(decl.Pos(),
					fmt.Sprintf("@rate_limit annotation value must be numeric in endpoint %s %q", decl.Method, decl.Path)))
			}
		}
	case "version":
		// @version should follow semver pattern
		if ann.Value != "" && !isValidVersion(ann.Value) {
			v.errors.Add(newSemanticError(decl.Pos(),
				fmt.Sprintf("@version annotation value %q should follow semver format (e.g., 'v1', 'v2.0')", ann.Value)))
		}
	}
}

// isValidVersion checks if a string is a valid version format.
var versionPattern = regexp.MustCompile(`^v?\d+(\.\d+)*$`)

func isValidVersion(version string) bool {
	return versionPattern.MatchString(version)
}

// validateHandler validates the endpoint handler.
func (v *EndpointValidator) validateHandler(decl *ast.EndpointDecl) {
	if decl.Handler == nil {
		return
	}

	handler := decl.Handler

	// Validate request type
	if handler.Request != nil {
		v.validateRequestType(handler.Request, decl)
	}

	// Validate response type
	if handler.Response != nil {
		v.validateResponseType(handler.Response, decl)
	}

	// Validate logic block
	if handler.Logic != nil {
		v.validateHandlerLogic(handler.Logic, decl)
	}

	// Validate consistency between request source and HTTP method
	if handler.Request != nil {
		v.validateRequestSourceMethodConsistency(handler.Request, decl)
	}
}

// validateRequestType validates the request type specification.
func (v *EndpointValidator) validateRequestType(req *ast.RequestType, decl *ast.EndpointDecl) {
	// Validate type name format
	if !isValidTypeName(req.TypeName) {
		v.errors.Add(newSemanticError(decl.Pos(),
			fmt.Sprintf("invalid request type name %q in endpoint %s %q", req.TypeName, decl.Method, decl.Path)))
	}

	// Check if type is registered (if strict validation is enabled)
	if len(v.types) > 0 && !v.types[req.TypeName] {
		v.errors.Add(newSemanticError(decl.Pos(),
			fmt.Sprintf("unknown request type %q in endpoint %s %q", req.TypeName, decl.Method, decl.Path)))
	}

	// Validate request source
	validSources := map[ast.RequestSource]bool{
		ast.RequestSourceBody:   true,
		ast.RequestSourceQuery:  true,
		ast.RequestSourcePath:   true,
		ast.RequestSourceHeader: true,
	}
	if !validSources[req.Source] {
		v.errors.Add(newSemanticError(decl.Pos(),
			fmt.Sprintf("invalid request source %q; valid sources: body, query, path, header", req.Source)))
	}
}

// validateResponseType validates the response type specification.
func (v *EndpointValidator) validateResponseType(resp *ast.ResponseType, decl *ast.EndpointDecl) {
	// Validate type name format
	if !isValidTypeName(resp.TypeName) {
		v.errors.Add(newSemanticError(decl.Pos(),
			fmt.Sprintf("invalid response type name %q in endpoint %s %q", resp.TypeName, decl.Method, decl.Path)))
	}

	// Check if type is registered (if strict validation is enabled)
	if len(v.types) > 0 && !v.types[resp.TypeName] {
		v.errors.Add(newSemanticError(decl.Pos(),
			fmt.Sprintf("unknown response type %q in endpoint %s %q", resp.TypeName, decl.Method, decl.Path)))
	}

	// Validate status code
	if resp.StatusCode < 100 || resp.StatusCode > 599 {
		v.errors.Add(newSemanticError(decl.Pos(),
			fmt.Sprintf("invalid HTTP status code %d; must be between 100 and 599", resp.StatusCode)))
	}

	// Validate status code appropriateness
	v.validateStatusCodeForMethod(resp.StatusCode, decl)
}

// validateStatusCodeForMethod validates that the status code is appropriate for the HTTP method.
func (v *EndpointValidator) validateStatusCodeForMethod(status int, decl *ast.EndpointDecl) {
	// Common patterns - not errors, but unusual
	switch decl.Method {
	case ast.HTTPMethodPOST:
		// POST typically returns 201 Created, but 200 is also valid
		if status == 204 {
			// 204 No Content is unusual for POST but valid
		}
	case ast.HTTPMethodDELETE:
		// DELETE typically returns 204 No Content or 200 OK
		if status == 201 {
			v.errors.Add(newSemanticError(decl.Pos(),
				fmt.Sprintf("DELETE endpoint %q returns 201 Created which is unusual", decl.Path)))
		}
	}
}

// validateHandlerLogic validates the handler logic block.
func (v *EndpointValidator) validateHandlerLogic(logic *ast.HandlerLogic, decl *ast.EndpointDecl) {
	if logic == nil || len(logic.Steps) == 0 {
		return
	}

	targets := make(map[string]bool)

	for _, step := range logic.Steps {
		v.validateLogicStep(step, decl, targets)
	}
}

// validateLogicStep validates a single logic step.
func (v *EndpointValidator) validateLogicStep(step *ast.LogicStep, decl *ast.EndpointDecl, targets map[string]bool) {
	// Validate action name
	if step.Action == "" {
		v.errors.Add(newSemanticError(decl.Pos(),
			fmt.Sprintf("logic step in endpoint %s %q has empty action", decl.Method, decl.Path)))
		return
	}

	if !isValidIdentifier(step.Action) {
		v.errors.Add(newSemanticError(decl.Pos(),
			fmt.Sprintf("invalid action name %q in endpoint %s %q", step.Action, decl.Method, decl.Path)))
	}

	// Check if action is registered (if strict validation is enabled)
	if len(v.actions) > 0 && !v.actions[step.Action] {
		v.errors.Add(newSemanticError(decl.Pos(),
			fmt.Sprintf("unknown action %q in endpoint %s %q", step.Action, decl.Method, decl.Path)))
	}

	// Validate target variable
	if step.Target != "" {
		if !isValidIdentifier(step.Target) {
			v.errors.Add(newSemanticError(decl.Pos(),
				fmt.Sprintf("invalid target variable name %q in endpoint %s %q", step.Target, decl.Method, decl.Path)))
		}
		// Track target for potential duplicate detection
		if targets[step.Target] {
			// Reassignment is valid, but might be a mistake
		}
		targets[step.Target] = true
	}

	// Validate arguments
	for _, arg := range step.Args {
		// Arguments can be identifiers or string literals
		// String literals start with quote - skip validation for those
		if !strings.HasPrefix(arg, "\"") && !isValidIdentifier(arg) && arg != "request" && arg != "response" {
			v.errors.Add(newSemanticError(decl.Pos(),
				fmt.Sprintf("invalid argument %q in logic step %q", arg, step.Action)))
		}
	}

	// Validate options
	for _, opt := range step.Options {
		if !isValidIdentifier(opt.Key) {
			v.errors.Add(newSemanticError(decl.Pos(),
				fmt.Sprintf("invalid option key %q in logic step %q", opt.Key, step.Action)))
		}
	}
}

// validateRequestSourceMethodConsistency validates that request source makes sense for the HTTP method.
func (v *EndpointValidator) validateRequestSourceMethodConsistency(req *ast.RequestType, decl *ast.EndpointDecl) {
	switch decl.Method {
	case ast.HTTPMethodGET, ast.HTTPMethodDELETE:
		// GET and DELETE should not have body
		if req.Source == ast.RequestSourceBody {
			v.errors.Add(newSemanticError(decl.Pos(),
				fmt.Sprintf("%s endpoint %q has request from body; GET/DELETE requests should not have a body", decl.Method, decl.Path)))
		}
	}
}

// isValidTypeName checks if a string is a valid type name (PascalCase).
var typeNamePattern = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

func isValidTypeName(name string) bool {
	return typeNamePattern.MatchString(name)
}


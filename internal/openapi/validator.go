package openapi

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidationError represents a validation error.
type ValidationError struct {
	Path    string
	Message string
}

func (e ValidationError) Error() string {
	if e.Path == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// ValidationResult contains the result of validating an OpenAPI spec.
type ValidationResult struct {
	Valid    bool
	Errors   []ValidationError
	Warnings []ValidationError
}

// Validator validates OpenAPI specifications.
type Validator struct {
	// StrictMode enables strict validation
	StrictMode bool
}

// NewValidator creates a new validator.
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates an OpenAPI specification.
func (v *Validator) Validate(spec *OpenAPI) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Validate required fields
	v.validateOpenAPIVersion(spec, result)
	v.validateInfo(spec, result)
	v.validatePaths(spec, result)
	v.validateComponents(spec, result)
	v.validateTags(spec, result)
	v.validateSecurity(spec, result)

	return result
}

func (v *Validator) addError(result *ValidationResult, path, message string) {
	result.Valid = false
	result.Errors = append(result.Errors, ValidationError{Path: path, Message: message})
}

func (v *Validator) addWarning(result *ValidationResult, path, message string) {
	result.Warnings = append(result.Warnings, ValidationError{Path: path, Message: message})
}

func (v *Validator) validateOpenAPIVersion(spec *OpenAPI, result *ValidationResult) {
	if spec.OpenAPI == "" {
		v.addError(result, "openapi", "openapi version is required")
		return
	}

	// Valid versions: 3.0.0, 3.0.1, 3.0.2, 3.0.3, 3.1.0
	validVersions := map[string]bool{
		"3.0.0": true,
		"3.0.1": true,
		"3.0.2": true,
		"3.0.3": true,
		"3.1.0": true,
	}

	if !validVersions[spec.OpenAPI] {
		v.addError(result, "openapi", fmt.Sprintf("invalid OpenAPI version: %s", spec.OpenAPI))
	}
}

func (v *Validator) validateInfo(spec *OpenAPI, result *ValidationResult) {
	if spec.Info.Title == "" {
		v.addError(result, "info.title", "title is required")
	}

	if spec.Info.Version == "" {
		v.addError(result, "info.version", "version is required")
	}

	// Validate contact URL if present
	if spec.Info.Contact != nil && spec.Info.Contact.URL != "" {
		if !isValidURL(spec.Info.Contact.URL) {
			v.addWarning(result, "info.contact.url", "invalid URL format")
		}
	}

	// Validate contact email if present
	if spec.Info.Contact != nil && spec.Info.Contact.Email != "" {
		if !isValidEmail(spec.Info.Contact.Email) {
			v.addWarning(result, "info.contact.email", "invalid email format")
		}
	}

	// Validate license URL if present
	if spec.Info.License != nil {
		if spec.Info.License.Name == "" {
			v.addError(result, "info.license.name", "license name is required when license is present")
		}
		if spec.Info.License.URL != "" && !isValidURL(spec.Info.License.URL) {
			v.addWarning(result, "info.license.url", "invalid URL format")
		}
	}
}

func (v *Validator) validatePaths(spec *OpenAPI, result *ValidationResult) {
	if len(spec.Paths) == 0 {
		v.addWarning(result, "paths", "no paths defined")
		return
	}

	for path, item := range spec.Paths {
		v.validatePath(path, item, result)
	}
}

func (v *Validator) validatePath(path string, item PathItem, result *ValidationResult) {
	// Path must start with /
	if !strings.HasPrefix(path, "/") {
		v.addError(result, fmt.Sprintf("paths.%s", path), "path must start with /")
	}

	// Validate path parameters are properly formatted
	pathParams := extractPathParams(path)

	// Count operations
	operationCount := 0

	if item.Get != nil {
		operationCount++
		v.validateOperation(path, "get", item.Get, pathParams, result)
	}
	if item.Put != nil {
		operationCount++
		v.validateOperation(path, "put", item.Put, pathParams, result)
	}
	if item.Post != nil {
		operationCount++
		v.validateOperation(path, "post", item.Post, pathParams, result)
	}
	if item.Delete != nil {
		operationCount++
		v.validateOperation(path, "delete", item.Delete, pathParams, result)
	}
	if item.Patch != nil {
		operationCount++
		v.validateOperation(path, "patch", item.Patch, pathParams, result)
	}
	if item.Head != nil {
		operationCount++
		v.validateOperation(path, "head", item.Head, pathParams, result)
	}
	if item.Options != nil {
		operationCount++
		v.validateOperation(path, "options", item.Options, pathParams, result)
	}

	if operationCount == 0 && item.Ref == "" {
		v.addWarning(result, fmt.Sprintf("paths.%s", path), "path has no operations")
	}
}

func (v *Validator) validateOperation(path, method string, op *Operation, pathParams []string, result *ValidationResult) {
	basePath := fmt.Sprintf("paths.%s.%s", path, method)

	// Validate responses
	if len(op.Responses) == 0 {
		v.addError(result, basePath+".responses", "responses is required")
	}

	// Validate all path parameters are defined
	definedParams := make(map[string]bool)
	for _, param := range op.Parameters {
		if param.In == "path" {
			definedParams[param.Name] = true
		}
	}

	for _, pathParam := range pathParams {
		if !definedParams[pathParam] {
			v.addWarning(result, basePath+".parameters",
				fmt.Sprintf("path parameter '%s' is not defined", pathParam))
		}
	}

	// Validate parameters
	for i, param := range op.Parameters {
		v.validateParameter(fmt.Sprintf("%s.parameters[%d]", basePath, i), param, result)
	}

	// Validate request body
	if op.RequestBody != nil {
		v.validateRequestBody(basePath+".requestBody", op.RequestBody, result)
	}

	// Validate responses
	for code, resp := range op.Responses {
		v.validateResponse(fmt.Sprintf("%s.responses.%s", basePath, code), code, resp, result)
	}

	// Validate operation ID is unique (if strict mode)
	if v.StrictMode && op.OperationID == "" {
		v.addWarning(result, basePath+".operationId", "operationId should be defined")
	}
}

func (v *Validator) validateParameter(path string, param Parameter, result *ValidationResult) {
	if param.Ref != "" {
		// Reference parameter, skip validation
		return
	}

	if param.Name == "" {
		v.addError(result, path+".name", "parameter name is required")
	}

	if param.In == "" {
		v.addError(result, path+".in", "parameter location (in) is required")
	} else {
		validLocations := map[string]bool{
			"query":  true,
			"header": true,
			"path":   true,
			"cookie": true,
		}
		if !validLocations[param.In] {
			v.addError(result, path+".in", fmt.Sprintf("invalid parameter location: %s", param.In))
		}
	}

	// Path parameters must be required
	if param.In == "path" && !param.Required {
		v.addError(result, path+".required", "path parameters must be required")
	}
}

func (v *Validator) validateRequestBody(path string, body *RequestBody, result *ValidationResult) {
	if body.Ref != "" {
		// Reference, skip validation
		return
	}

	if len(body.Content) == 0 {
		v.addError(result, path+".content", "request body content is required")
	}

	for mediaType, content := range body.Content {
		v.validateMediaType(fmt.Sprintf("%s.content.%s", path, mediaType), content, result)
	}
}

func (v *Validator) validateResponse(path, code string, resp Response, result *ValidationResult) {
	if resp.Ref != "" {
		// Reference, skip validation
		return
	}

	// Validate status code
	if code != "default" {
		if !isValidStatusCode(code) {
			v.addError(result, path, fmt.Sprintf("invalid status code: %s", code))
		}
	}

	if resp.Description == "" {
		v.addError(result, path+".description", "response description is required")
	}

	for mediaType, content := range resp.Content {
		v.validateMediaType(fmt.Sprintf("%s.content.%s", path, mediaType), content, result)
	}
}

func (v *Validator) validateMediaType(path string, mt MediaType, result *ValidationResult) {
	// Schema is optional but should be present in most cases
	if mt.Schema == nil && v.StrictMode {
		v.addWarning(result, path+".schema", "schema should be defined")
	}

	if mt.Schema != nil {
		v.validateSchema(path+".schema", mt.Schema, result)
	}
}

func (v *Validator) validateSchema(path string, schema *Schema, result *ValidationResult) {
	if schema.Ref != "" {
		// Validate reference format
		if !strings.HasPrefix(schema.Ref, "#/") {
			v.addWarning(result, path+".$ref", "reference should start with #/")
		}
		return
	}

	// Type validation
	if schema.Type != "" {
		validTypes := map[string]bool{
			"string":  true,
			"number":  true,
			"integer": true,
			"boolean": true,
			"array":   true,
			"object":  true,
		}
		if !validTypes[schema.Type] {
			v.addError(result, path+".type", fmt.Sprintf("invalid type: %s", schema.Type))
		}
	}

	// Array items validation
	if schema.Type == "array" && schema.Items == nil {
		v.addError(result, path+".items", "items is required for array type")
	}

	// Validate nested schemas
	if schema.Items != nil {
		v.validateSchema(path+".items", schema.Items, result)
	}

	for propName, propSchema := range schema.Properties {
		v.validateSchema(fmt.Sprintf("%s.properties.%s", path, propName), propSchema, result)
	}

	if schema.AdditionalProperties != nil {
		v.validateSchema(path+".additionalProperties", schema.AdditionalProperties, result)
	}
}

func (v *Validator) validateComponents(spec *OpenAPI, result *ValidationResult) {
	// Validate schemas
	for name, schema := range spec.Components.Schemas {
		v.validateSchema(fmt.Sprintf("components.schemas.%s", name), schema, result)
	}

	// Validate security schemes
	for name, scheme := range spec.Components.SecuritySchemes {
		v.validateSecurityScheme(fmt.Sprintf("components.securitySchemes.%s", name), scheme, result)
	}
}

func (v *Validator) validateSecurityScheme(path string, scheme *SecurityScheme, result *ValidationResult) {
	if scheme.Ref != "" {
		return
	}

	if scheme.Type == "" {
		v.addError(result, path+".type", "security scheme type is required")
		return
	}

	validTypes := map[string]bool{
		"apiKey":        true,
		"http":          true,
		"oauth2":        true,
		"openIdConnect": true,
	}

	if !validTypes[scheme.Type] {
		v.addError(result, path+".type", fmt.Sprintf("invalid security scheme type: %s", scheme.Type))
	}

	switch scheme.Type {
	case "apiKey":
		if scheme.Name == "" {
			v.addError(result, path+".name", "name is required for apiKey type")
		}
		if scheme.In == "" {
			v.addError(result, path+".in", "in is required for apiKey type")
		} else if scheme.In != "query" && scheme.In != "header" && scheme.In != "cookie" {
			v.addError(result, path+".in", "in must be query, header, or cookie for apiKey type")
		}
	case "http":
		if scheme.Scheme == "" {
			v.addError(result, path+".scheme", "scheme is required for http type")
		}
	case "oauth2":
		if scheme.Flows == nil {
			v.addError(result, path+".flows", "flows is required for oauth2 type")
		}
	case "openIdConnect":
		if scheme.OpenIDConnectURL == "" {
			v.addError(result, path+".openIdConnectUrl", "openIdConnectUrl is required for openIdConnect type")
		}
	}
}

func (v *Validator) validateTags(spec *OpenAPI, result *ValidationResult) {
	// Check for duplicate tags
	seen := make(map[string]bool)
	for _, tag := range spec.Tags {
		if tag.Name == "" {
			v.addError(result, "tags", "tag name is required")
			continue
		}
		if seen[tag.Name] {
			v.addWarning(result, "tags", fmt.Sprintf("duplicate tag: %s", tag.Name))
		}
		seen[tag.Name] = true
	}
}

func (v *Validator) validateSecurity(spec *OpenAPI, result *ValidationResult) {
	// Validate that referenced security schemes exist
	for _, req := range spec.Security {
		for name := range req {
			if _, exists := spec.Components.SecuritySchemes[name]; !exists {
				v.addError(result, "security", fmt.Sprintf("security scheme '%s' is not defined", name))
			}
		}
	}
}

// Helper functions

func extractPathParams(path string) []string {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(path, -1)
	params := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}
	return params
}

func isValidURL(url string) bool {
	// Simple URL validation
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func isValidEmail(email string) bool {
	// Simple email validation
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func isValidStatusCode(code string) bool {
	// Valid HTTP status codes: 1XX, 2XX, 3XX, 4XX, 5XX or specific codes
	if len(code) == 3 {
		if code[0] >= '1' && code[0] <= '5' {
			return (code[1] >= '0' && code[1] <= '9') && (code[2] >= '0' && code[2] <= '9')
		}
	}
	return false
}

// ValidateSpec is a convenience function to validate an OpenAPI spec.
func ValidateSpec(spec *OpenAPI) *ValidationResult {
	v := NewValidator()
	return v.Validate(spec)
}

// ValidateSpecStrict is a convenience function to validate with strict mode.
func ValidateSpecStrict(spec *OpenAPI) *ValidationResult {
	v := NewValidator()
	v.StrictMode = true
	return v.Validate(spec)
}

package openapi

import (
	"fmt"
	"strings"

	"github.com/bargom/codeai/internal/ast"
)

// Mapper converts CodeAI AST to OpenAPI structures.
type Mapper struct {
	// Spec is the OpenAPI specification being built
	Spec *OpenAPI
	// SchemaGen generates schemas from types
	SchemaGen *SchemaGenerator
}

// NewMapper creates a new AST to OpenAPI mapper.
func NewMapper() *Mapper {
	return &Mapper{
		Spec: &OpenAPI{
			OpenAPI: "3.0.0",
			Paths:   make(map[string]PathItem),
			Components: Components{
				Schemas:         make(map[string]*Schema),
				SecuritySchemes: make(map[string]*SecurityScheme),
			},
		},
		SchemaGen: NewSchemaGenerator(),
	}
}

// MapProgram maps a CodeAI Program AST to OpenAPI paths and schemas.
func (m *Mapper) MapProgram(program *ast.Program) error {
	for _, stmt := range program.Statements {
		if err := m.mapStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (m *Mapper) mapStatement(stmt ast.Statement) error {
	switch s := stmt.(type) {
	case *ast.FunctionDecl:
		return m.mapFunction(s)
	case *ast.VarDecl:
		return m.mapVariable(s)
	case *ast.CollectionDecl:
		return m.mapCollection(s)
	case *ast.EndpointDecl:
		return m.mapEndpoint(s)
	case *ast.ModelDecl:
		return m.mapModel(s)
	case *ast.AuthDecl:
		return m.mapAuth(s)
	default:
		// Skip other statement types
		return nil
	}
}

// mapFunction maps a function declaration to an OpenAPI operation.
// Functions with HTTP-related names are treated as endpoints.
func (m *Mapper) mapFunction(fn *ast.FunctionDecl) error {
	// Check if this looks like an HTTP handler
	method, path := parseHandlerName(fn.Name)
	if method == "" {
		return nil // Not an HTTP handler
	}

	// Create the operation
	op := &Operation{
		OperationID: fn.Name,
		Summary:     generateSummary(method, path),
		Responses: map[string]Response{
			"200": {Description: "Success"},
			"400": {Description: "Bad Request"},
			"500": {Description: "Internal Server Error"},
		},
		// Add CodeAI extension
		XCodeAIHandler: fn.Name,
		XCodeAISource:  fn.Pos().String(),
	}

	// Map function parameters to OpenAPI parameters
	for _, param := range fn.Params {
		p := m.mapParameter(param, path)
		op.Parameters = append(op.Parameters, p)
	}

	// Add or update the path item
	pathItem, ok := m.Spec.Paths[path]
	if !ok {
		pathItem = PathItem{}
	}

	switch method {
	case "GET":
		pathItem.Get = op
	case "POST":
		pathItem.Post = op
	case "PUT":
		pathItem.Put = op
	case "DELETE":
		pathItem.Delete = op
	case "PATCH":
		pathItem.Patch = op
	case "HEAD":
		pathItem.Head = op
	case "OPTIONS":
		pathItem.Options = op
	}

	m.Spec.Paths[path] = pathItem
	return nil
}

// mapVariable maps a variable declaration, potentially to a schema.
func (m *Mapper) mapVariable(v *ast.VarDecl) error {
	// Check if this looks like a type/schema definition
	// In CodeAI DSL, we might have entity-like variable declarations
	return nil
}

// mapCollection maps a MongoDB collection declaration to an OpenAPI schema.
func (m *Mapper) mapCollection(coll *ast.CollectionDecl) error {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
		// Add MongoDB-specific extension
		XMongoCollection: coll.Name,
	}

	var required []string

	for _, field := range coll.Fields {
		fieldSchema := m.mapMongoField(field)
		schema.Properties[field.Name] = fieldSchema

		// Required fields are those not marked as optional
		if !isOptionalField(field) {
			required = append(required, field.Name)
		}
	}

	if len(required) > 0 {
		schema.Required = required
	}

	// Add description with database info
	if coll.Description != "" {
		schema.Description = coll.Description
	} else {
		schema.Description = fmt.Sprintf("MongoDB collection: %s", coll.Name)
	}

	// Add the schema to components
	m.Spec.Components.Schemas[coll.Name] = schema

	return nil
}

// isOptionalField checks if a MongoDB field has an "optional" modifier.
func isOptionalField(field *ast.MongoFieldDecl) bool {
	for _, mod := range field.Modifiers {
		if strings.ToLower(mod.Name) == "optional" {
			return true
		}
	}
	return false
}

// mapMongoField converts a MongoDB field declaration to an OpenAPI schema.
func (m *Mapper) mapMongoField(field *ast.MongoFieldDecl) *Schema {
	schema := m.mapMongoType(field.FieldType)

	// Check for nullable modifier
	for _, mod := range field.Modifiers {
		if strings.ToLower(mod.Name) == "nullable" {
			schema.Nullable = true
		}
	}

	return schema
}

// mapMongoType converts a MongoDB type reference to an OpenAPI schema.
func (m *Mapper) mapMongoType(typeRef *ast.MongoTypeRef) *Schema {
	if typeRef == nil {
		return &Schema{Type: "object"}
	}

	// Handle embedded documents
	if typeRef.EmbeddedDoc != nil {
		return m.mapEmbeddedDoc(typeRef.EmbeddedDoc)
	}

	typeName := strings.ToLower(typeRef.Name)

	// Handle MongoDB-specific types
	switch typeName {
	case "objectid":
		return &Schema{
			Type:        "string",
			Format:      "objectid",
			Description: "MongoDB ObjectID",
			Pattern:     "^[a-fA-F0-9]{24}$",
		}
	case "string":
		return &Schema{Type: "string"}
	case "int", "integer":
		return &Schema{Type: "integer"}
	case "int32":
		return &Schema{Type: "integer", Format: "int32"}
	case "int64":
		return &Schema{Type: "integer", Format: "int64"}
	case "double", "float", "number":
		return &Schema{Type: "number"}
	case "decimal", "decimal128":
		return &Schema{Type: "string", Format: "decimal"}
	case "bool", "boolean":
		return &Schema{Type: "boolean"}
	case "date", "datetime", "timestamp":
		return &Schema{Type: "string", Format: "date-time"}
	case "binary", "bindata":
		return &Schema{Type: "string", Format: "binary"}
	case "array":
		schema := &Schema{Type: "array"}
		// Check if there's a type parameter for array elements
		if len(typeRef.Params) > 0 {
			elemType := &ast.MongoTypeRef{Name: typeRef.Params[0]}
			schema.Items = m.mapMongoType(elemType)
		}
		return schema
	case "object", "document":
		return &Schema{Type: "object"}
	default:
		// Reference to another schema
		return &Schema{Ref: "#/components/schemas/" + typeName}
	}
}

// mapEmbeddedDoc maps an embedded document to an OpenAPI schema.
func (m *Mapper) mapEmbeddedDoc(doc *ast.EmbeddedDocDecl) *Schema {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	var required []string
	for _, field := range doc.Fields {
		schema.Properties[field.Name] = m.mapMongoField(field)
		if !isOptionalField(field) {
			required = append(required, field.Name)
		}
	}

	if len(required) > 0 {
		schema.Required = required
	}

	return schema
}

// mapParameter converts an AST parameter to an OpenAPI parameter.
func (m *Mapper) mapParameter(param ast.Parameter, path string) Parameter {
	// Determine if this is a path parameter
	isPathParam := strings.Contains(path, "{"+param.Name+"}")

	p := Parameter{
		Name:     param.Name,
		Required: isPathParam, // Path params are always required
		Schema:   m.inferParamSchema(param),
	}

	if isPathParam {
		p.In = "path"
	} else {
		p.In = "query"
	}

	return p
}

// inferParamSchema infers the schema for a parameter.
func (m *Mapper) inferParamSchema(param ast.Parameter) *Schema {
	// Check if there's a type annotation
	if param.Type != "" {
		return SchemaFromType(param.Type)
	}

	// Try to infer from default value
	if param.Default != nil {
		return m.schemaFromExpression(param.Default)
	}

	// Default to string
	return &Schema{Type: "string"}
}

// schemaFromExpression infers a schema from an AST expression.
func (m *Mapper) schemaFromExpression(expr ast.Expression) *Schema {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return &Schema{Type: "string", Example: e.Value}
	case *ast.NumberLiteral:
		if e.Value == float64(int(e.Value)) {
			return &Schema{Type: "integer", Example: e.Value}
		}
		return &Schema{Type: "number", Example: e.Value}
	case *ast.BoolLiteral:
		return &Schema{Type: "boolean", Example: e.Value}
	case *ast.ArrayLiteral:
		schema := &Schema{Type: "array"}
		if len(e.Elements) > 0 {
			schema.Items = m.schemaFromExpression(e.Elements[0])
		}
		return schema
	default:
		return &Schema{Type: "object"}
	}
}

// parseHandlerName attempts to extract HTTP method and path from a function name.
// Conventions:
// - getUsers -> GET /users
// - getUserById -> GET /users/{id}
// - createUser -> POST /users
// - updateUser -> PUT /users/{id}
// - deleteUser -> DELETE /users/{id}
// - handle_get_users -> GET /users
func parseHandlerName(name string) (method, path string) {
	lowerName := strings.ToLower(name)

	// Check for common prefixes
	prefixes := map[string]string{
		"get":    "GET",
		"list":   "GET",
		"fetch":  "GET",
		"create": "POST",
		"add":    "POST",
		"new":    "POST",
		"post":   "POST",
		"update": "PUT",
		"put":    "PUT",
		"edit":   "PUT",
		"patch":  "PATCH",
		"delete": "DELETE",
		"remove": "DELETE",
		"handle": "", // Special case
	}

	var foundPrefix string
	for prefix, httpMethod := range prefixes {
		if strings.HasPrefix(lowerName, prefix) {
			foundPrefix = prefix
			method = httpMethod
			break
		}
	}

	if foundPrefix == "" {
		return "", "" // Not an HTTP handler
	}

	// Handle "handle_method_resource" pattern
	if foundPrefix == "handle" {
		parts := strings.SplitN(name, "_", 3)
		if len(parts) >= 2 {
			method = strings.ToUpper(parts[1])
			if len(parts) >= 3 {
				path = "/" + toSnakeCase(parts[2])
			}
			return method, path
		}
		return "", ""
	}

	// Extract resource name after prefix
	resourcePart := name[len(foundPrefix):]
	resourcePart = strings.TrimPrefix(resourcePart, "_")

	// Check for "ById" or "ByID" suffix
	hasIdParam := false
	if strings.HasSuffix(strings.ToLower(resourcePart), "byid") {
		resourcePart = resourcePart[:len(resourcePart)-4]
		hasIdParam = true
	}

	// Convert to path
	path = "/" + toSnakeCase(resourcePart)
	if hasIdParam {
		path += "/{id}"
	}

	// Pluralize simple resources
	if !hasIdParam && (method == "GET" || method == "POST") {
		// Don't pluralize if already plural or complex
	}

	return method, path
}

// toSnakeCase converts camelCase or PascalCase to snake_case.
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// generateSummary generates a human-readable summary from method and path.
func generateSummary(method, path string) string {
	// Extract resource name from path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return method + " endpoint"
	}

	resource := parts[0]
	resource = strings.ReplaceAll(resource, "_", " ")
	resource = strings.Title(resource)

	hasId := strings.Contains(path, "{")

	switch method {
	case "GET":
		if hasId {
			return fmt.Sprintf("Get %s by ID", singularize(resource))
		}
		return fmt.Sprintf("List %s", resource)
	case "POST":
		return fmt.Sprintf("Create %s", singularize(resource))
	case "PUT":
		return fmt.Sprintf("Update %s", singularize(resource))
	case "PATCH":
		return fmt.Sprintf("Partially update %s", singularize(resource))
	case "DELETE":
		return fmt.Sprintf("Delete %s", singularize(resource))
	default:
		return fmt.Sprintf("%s %s", method, resource)
	}
}

// singularize attempts to convert a plural word to singular.
func singularize(word string) string {
	word = strings.TrimSpace(word)
	if word == "" {
		return word
	}

	// Simple rules
	if strings.HasSuffix(word, "ies") {
		return word[:len(word)-3] + "y"
	}
	if strings.HasSuffix(word, "es") {
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") {
		return word[:len(word)-1]
	}
	return word
}

// AddRoute adds a route to the OpenAPI spec.
func (m *Mapper) AddRoute(method, path string, op *Operation) {
	pathItem, ok := m.Spec.Paths[path]
	if !ok {
		pathItem = PathItem{}
	}

	switch strings.ToUpper(method) {
	case "GET":
		pathItem.Get = op
	case "POST":
		pathItem.Post = op
	case "PUT":
		pathItem.Put = op
	case "DELETE":
		pathItem.Delete = op
	case "PATCH":
		pathItem.Patch = op
	case "HEAD":
		pathItem.Head = op
	case "OPTIONS":
		pathItem.Options = op
	}

	m.Spec.Paths[path] = pathItem
}

// AddSchema adds a schema to the components.
func (m *Mapper) AddSchema(name string, schema *Schema) {
	m.Spec.Components.Schemas[name] = schema
}

// AddSecurityScheme adds a security scheme to the components.
func (m *Mapper) AddSecurityScheme(name string, scheme *SecurityScheme) {
	m.Spec.Components.SecuritySchemes[name] = scheme
}

// AddTag adds a tag to the spec.
func (m *Mapper) AddTag(tag Tag) {
	// Check if tag already exists
	for _, t := range m.Spec.Tags {
		if t.Name == tag.Name {
			return
		}
	}
	m.Spec.Tags = append(m.Spec.Tags, tag)
}

// GetSpec returns the built OpenAPI specification.
func (m *Mapper) GetSpec() *OpenAPI {
	return m.Spec
}

// =============================================================================
// Endpoint Mapping
// =============================================================================

// mapEndpoint maps a CodeAI EndpointDecl to an OpenAPI PathItem and Operation.
func (m *Mapper) mapEndpoint(ep *ast.EndpointDecl) error {
	// Convert path from :param to {param} format
	path := convertPathParams(ep.Path)

	// Create the operation
	op := &Operation{
		OperationID: generateEndpointOperationID(string(ep.Method), ep.Path),
		Summary:     generateEndpointSummary(string(ep.Method), ep.Path),
		Responses:   make(map[string]Response),
		// Add CodeAI extensions
		XCodeAISource: ep.Pos().String(),
	}

	// Extract tags from path
	tags := extractTagsFromPath(ep.Path)
	if len(tags) > 0 {
		op.Tags = tags
	}

	// Map middlewares to extension
	if len(ep.Middlewares) > 0 {
		middlewareNames := make([]string, len(ep.Middlewares))
		for i, mw := range ep.Middlewares {
			middlewareNames[i] = mw.Name
		}
		op.XCodeAIMiddleware = middlewareNames
	}

	// Map annotations
	for _, ann := range ep.Annotations {
		m.applyAnnotation(op, ann)
	}

	// Map handler request/response
	if ep.Handler != nil {
		m.mapHandler(op, ep.Handler, ep.Path)
	}

	// Add default responses if none defined
	if len(op.Responses) == 0 {
		op.Responses["200"] = Response{Description: "Success"}
	}
	// Add common error responses
	if _, exists := op.Responses["400"]; !exists {
		op.Responses["400"] = Response{Description: "Bad Request"}
	}
	if _, exists := op.Responses["401"]; !exists {
		op.Responses["401"] = Response{Description: "Unauthorized"}
	}
	if _, exists := op.Responses["500"]; !exists {
		op.Responses["500"] = Response{Description: "Internal Server Error"}
	}

	// Add or update the path item
	pathItem, ok := m.Spec.Paths[path]
	if !ok {
		pathItem = PathItem{}
	}

	switch ep.Method {
	case ast.HTTPMethodGET:
		pathItem.Get = op
	case ast.HTTPMethodPOST:
		pathItem.Post = op
	case ast.HTTPMethodPUT:
		pathItem.Put = op
	case ast.HTTPMethodDELETE:
		pathItem.Delete = op
	case ast.HTTPMethodPATCH:
		pathItem.Patch = op
	}

	m.Spec.Paths[path] = pathItem
	return nil
}

// mapHandler maps handler request/response to OpenAPI parameters, request body, and responses.
func (m *Mapper) mapHandler(op *Operation, handler *ast.Handler, path string) {
	// Map request
	if handler.Request != nil {
		m.mapRequest(op, handler.Request, path)
	}

	// Map response
	if handler.Response != nil {
		m.mapResponse(op, handler.Response)
	}
}

// mapRequest maps a RequestType to OpenAPI parameters or request body.
func (m *Mapper) mapRequest(op *Operation, req *ast.RequestType, path string) {
	switch req.Source {
	case ast.RequestSourcePath:
		// Extract path parameters from the path
		params := extractPathParamsFromPath(path)
		for _, paramName := range params {
			op.Parameters = append(op.Parameters, Parameter{
				Name:     paramName,
				In:       "path",
				Required: true,
				Schema:   &Schema{Type: "string"},
			})
		}
	case ast.RequestSourceQuery:
		// Reference the type as query parameters
		op.Parameters = append(op.Parameters, Parameter{
			Name:   "query",
			In:     "query",
			Schema: GenerateRef(req.TypeName),
		})
	case ast.RequestSourceBody:
		// Create request body
		op.RequestBody = &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/json": {
					Schema: GenerateRef(req.TypeName),
				},
			},
		}
	case ast.RequestSourceHeader:
		op.Parameters = append(op.Parameters, Parameter{
			Name:   req.TypeName,
			In:     "header",
			Schema: &Schema{Type: "string"},
		})
	}
}

// mapResponse maps a ResponseType to OpenAPI responses.
func (m *Mapper) mapResponse(op *Operation, resp *ast.ResponseType) {
	statusCode := fmt.Sprintf("%d", resp.StatusCode)
	r := Response{
		Description: getStatusDescription(resp.StatusCode),
	}

	if resp.TypeName != "" {
		r.Content = map[string]MediaType{
			"application/json": {
				Schema: GenerateRef(resp.TypeName),
			},
		}
	}

	op.Responses[statusCode] = r
}

// applyAnnotation applies an annotation to an operation.
func (m *Mapper) applyAnnotation(op *Operation, ann *ast.Annotation) {
	switch strings.ToLower(ann.Name) {
	case "deprecated":
		op.Deprecated = true
	case "auth":
		// Add security requirement
		if ann.Value != "" {
			op.Security = append(op.Security, SecurityRequirement{ann.Value: {}})
		}
	case "summary":
		op.Summary = ann.Value
	case "description":
		op.Description = ann.Value
	case "tag", "tags":
		op.Tags = append(op.Tags, ann.Value)
	}
}

// =============================================================================
// Model Mapping
// =============================================================================

// mapModel maps a CodeAI ModelDecl to an OpenAPI schema.
func (m *Mapper) mapModel(model *ast.ModelDecl) error {
	schema := &Schema{
		Type:        "object",
		Description: model.Description,
		Properties:  make(map[string]*Schema),
	}

	var required []string

	for _, field := range model.Fields {
		fieldSchema := m.mapFieldType(field.FieldType)

		// Apply modifiers
		for _, mod := range field.Modifiers {
			m.applyFieldModifier(fieldSchema, mod)
			// Check if required
			if strings.ToLower(mod.Name) == "required" {
				required = append(required, field.Name)
			}
		}

		schema.Properties[field.Name] = fieldSchema
	}

	if len(required) > 0 {
		schema.Required = required
	}

	// Add extension for database type
	schema.XDatabaseType = "postgres"

	m.Spec.Components.Schemas[model.Name] = schema
	return nil
}

// mapFieldType converts a CodeAI TypeRef to an OpenAPI schema.
func (m *Mapper) mapFieldType(typeRef *ast.TypeRef) *Schema {
	if typeRef == nil {
		return &Schema{Type: "object"}
	}

	typeName := strings.ToLower(typeRef.Name)

	// Handle parameterized types first
	if len(typeRef.Params) > 0 {
		switch typeName {
		case "list", "array":
			return &Schema{
				Type:  "array",
				Items: m.mapFieldType(typeRef.Params[0]),
			}
		case "ref":
			// Reference to another model
			if len(typeRef.Params) > 0 {
				return GenerateRef(typeRef.Params[0].Name)
			}
		case "optional":
			schema := m.mapFieldType(typeRef.Params[0])
			schema.Nullable = true
			return schema
		}
	}

	// Handle basic types
	switch typeName {
	case "string", "text":
		return &Schema{Type: "string"}
	case "int", "integer":
		return &Schema{Type: "integer", Format: "int32"}
	case "int64", "bigint":
		return &Schema{Type: "integer", Format: "int64"}
	case "float", "float32":
		return &Schema{Type: "number", Format: "float"}
	case "float64", "double", "decimal":
		return &Schema{Type: "number", Format: "double"}
	case "bool", "boolean":
		return &Schema{Type: "boolean"}
	case "uuid":
		return &Schema{Type: "string", Format: "uuid"}
	case "timestamp", "datetime":
		return &Schema{Type: "string", Format: "date-time"}
	case "date":
		return &Schema{Type: "string", Format: "date"}
	case "time":
		return &Schema{Type: "string", Format: "time"}
	case "email":
		return &Schema{Type: "string", Format: "email"}
	case "url", "uri":
		return &Schema{Type: "string", Format: "uri"}
	case "json", "jsonb":
		return &Schema{Type: "object"}
	case "binary", "bytes":
		return &Schema{Type: "string", Format: "binary"}
	default:
		// Assume it's a reference to another schema
		return GenerateRef(typeName)
	}
}

// applyFieldModifier applies a modifier to a schema.
func (m *Mapper) applyFieldModifier(schema *Schema, mod *ast.Modifier) {
	switch strings.ToLower(mod.Name) {
	case "primary":
		schema.Description = appendDescription(schema.Description, "Primary key")
	case "unique":
		schema.Description = appendDescription(schema.Description, "Unique")
	case "auto":
		schema.ReadOnly = true
	case "nullable", "optional":
		schema.Nullable = true
	case "default":
		if mod.Value != nil {
			schema.Default = extractExpressionValue(mod.Value)
		}
	}
}

// =============================================================================
// Auth Mapping
// =============================================================================

// mapAuth maps a CodeAI AuthDecl to an OpenAPI security scheme.
func (m *Mapper) mapAuth(auth *ast.AuthDecl) error {
	scheme := &SecurityScheme{
		Description: fmt.Sprintf("%s authentication", auth.Name),
	}

	switch auth.Method {
	case ast.AuthMethodJWT:
		scheme.Type = "http"
		scheme.Scheme = "bearer"
		scheme.BearerFormat = "JWT"
		if auth.JWKS != nil {
			scheme.Description = fmt.Sprintf("JWT authentication (JWKS: %s)", auth.JWKS.URL)
		}
	case ast.AuthMethodOAuth2:
		scheme.Type = "oauth2"
		scheme.Flows = m.mapOAuth2Flows(auth)
	case ast.AuthMethodAPIKey:
		scheme.Type = "apiKey"
		scheme.In = "header"
		scheme.Name = "X-API-Key"
		// Check config for custom header name
		if headerName, ok := auth.Config["header"]; ok {
			if str, ok := headerName.(*ast.StringLiteral); ok {
				scheme.Name = str.Value
			}
		}
	case ast.AuthMethodBasic:
		scheme.Type = "http"
		scheme.Scheme = "basic"
	}

	m.Spec.Components.SecuritySchemes[auth.Name] = scheme
	return nil
}

// mapOAuth2Flows maps OAuth2 configuration to OAuthFlows.
func (m *Mapper) mapOAuth2Flows(auth *ast.AuthDecl) *OAuthFlows {
	flows := &OAuthFlows{}

	// Check for authorization_url and token_url in config
	var authURL, tokenURL string
	if url, ok := auth.Config["authorization_url"]; ok {
		if str, ok := url.(*ast.StringLiteral); ok {
			authURL = str.Value
		}
	}
	if url, ok := auth.Config["token_url"]; ok {
		if str, ok := url.(*ast.StringLiteral); ok {
			tokenURL = str.Value
		}
	}

	// Determine flow type based on available URLs
	if authURL != "" && tokenURL != "" {
		flows.AuthorizationCode = &OAuthFlow{
			AuthorizationURL: authURL,
			TokenURL:         tokenURL,
			Scopes:           make(map[string]string),
		}
	} else if tokenURL != "" {
		flows.ClientCredentials = &OAuthFlow{
			TokenURL: tokenURL,
			Scopes:   make(map[string]string),
		}
	}

	return flows
}

// =============================================================================
// Helper Functions
// =============================================================================

// convertPathParams converts path parameters from :param to {param} format.
func convertPathParams(path string) string {
	// Replace :paramName with {paramName}
	result := path
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + strings.TrimPrefix(part, ":") + "}"
		}
	}
	result = strings.Join(parts, "/")
	return result
}

// extractPathParamsFromPath extracts parameter names from a path.
func extractPathParamsFromPath(path string) []string {
	var params []string
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			params = append(params, strings.TrimPrefix(part, ":"))
		} else if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			params = append(params, strings.Trim(part, "{}"))
		}
	}
	return params
}

// extractTagsFromPath extracts tags from the first path segment.
func extractTagsFromPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 0 && parts[0] != "" && !strings.HasPrefix(parts[0], ":") && !strings.HasPrefix(parts[0], "{") {
		return []string{parts[0]}
	}
	return nil
}

// generateEndpointOperationID generates an operation ID from method and path.
func generateEndpointOperationID(method, path string) string {
	// Convert path to camelCase identifier
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var result strings.Builder

	result.WriteString(strings.ToLower(method))

	for _, part := range parts {
		// Skip path parameters
		if strings.HasPrefix(part, ":") || strings.HasPrefix(part, "{") {
			result.WriteString("ById")
			continue
		}
		result.WriteString(strings.Title(part))
	}

	return result.String()
}

// generateEndpointSummary generates a human-readable summary.
func generateEndpointSummary(method, path string) string {
	return generateSummary(method, convertPathParams(path))
}

// getStatusDescription returns a description for an HTTP status code.
func getStatusDescription(code int) string {
	descriptions := map[int]string{
		200: "OK",
		201: "Created",
		204: "No Content",
		400: "Bad Request",
		401: "Unauthorized",
		403: "Forbidden",
		404: "Not Found",
		409: "Conflict",
		422: "Unprocessable Entity",
		500: "Internal Server Error",
	}
	if desc, ok := descriptions[code]; ok {
		return desc
	}
	return fmt.Sprintf("Response %d", code)
}

// appendDescription appends text to an existing description.
func appendDescription(existing, addition string) string {
	if existing == "" {
		return addition
	}
	return existing + ". " + addition
}

// extractExpressionValue extracts a value from an AST expression.
func extractExpressionValue(expr ast.Expression) interface{} {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return e.Value
	case *ast.NumberLiteral:
		return e.Value
	case *ast.BoolLiteral:
		return e.Value
	default:
		return nil
	}
}

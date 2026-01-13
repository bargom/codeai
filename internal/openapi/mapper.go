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

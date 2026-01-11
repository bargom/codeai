# Task: OpenAPI Specification Generation

## Overview
Generate OpenAPI 3.0 specifications from CodeAI endpoint and entity definitions.

## Phase
Phase 4: Integrations and Polish

## Priority
Medium - Important for API documentation.

## Dependencies
- 01-Foundation/02-parser-grammar.md
- 01-Foundation/04-validator.md

## Description
Create an OpenAPI generator that produces complete API documentation from CodeAI source files, including schemas, endpoints, authentication, and examples.

## Detailed Requirements

### 1. OpenAPI Types (internal/openapi/types.go)

```go
package openapi

type OpenAPI struct {
    OpenAPI    string                `json:"openapi" yaml:"openapi"`
    Info       Info                  `json:"info" yaml:"info"`
    Servers    []Server              `json:"servers,omitempty" yaml:"servers,omitempty"`
    Paths      map[string]PathItem   `json:"paths" yaml:"paths"`
    Components Components            `json:"components,omitempty" yaml:"components,omitempty"`
    Security   []SecurityRequirement `json:"security,omitempty" yaml:"security,omitempty"`
    Tags       []Tag                 `json:"tags,omitempty" yaml:"tags,omitempty"`
}

type Info struct {
    Title       string  `json:"title" yaml:"title"`
    Description string  `json:"description,omitempty" yaml:"description,omitempty"`
    Version     string  `json:"version" yaml:"version"`
    Contact     *Contact `json:"contact,omitempty" yaml:"contact,omitempty"`
    License     *License `json:"license,omitempty" yaml:"license,omitempty"`
}

type Contact struct {
    Name  string `json:"name,omitempty" yaml:"name,omitempty"`
    URL   string `json:"url,omitempty" yaml:"url,omitempty"`
    Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

type License struct {
    Name string `json:"name" yaml:"name"`
    URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

type Server struct {
    URL         string `json:"url" yaml:"url"`
    Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type PathItem struct {
    Get    *Operation `json:"get,omitempty" yaml:"get,omitempty"`
    Post   *Operation `json:"post,omitempty" yaml:"post,omitempty"`
    Put    *Operation `json:"put,omitempty" yaml:"put,omitempty"`
    Delete *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
    Patch  *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
}

type Operation struct {
    Tags        []string              `json:"tags,omitempty" yaml:"tags,omitempty"`
    Summary     string                `json:"summary,omitempty" yaml:"summary,omitempty"`
    Description string                `json:"description,omitempty" yaml:"description,omitempty"`
    OperationID string                `json:"operationId,omitempty" yaml:"operationId,omitempty"`
    Parameters  []Parameter           `json:"parameters,omitempty" yaml:"parameters,omitempty"`
    RequestBody *RequestBody          `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
    Responses   map[string]Response   `json:"responses" yaml:"responses"`
    Security    []SecurityRequirement `json:"security,omitempty" yaml:"security,omitempty"`
    Deprecated  bool                  `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
}

type Parameter struct {
    Name        string  `json:"name" yaml:"name"`
    In          string  `json:"in" yaml:"in"` // query, path, header, cookie
    Description string  `json:"description,omitempty" yaml:"description,omitempty"`
    Required    bool    `json:"required,omitempty" yaml:"required,omitempty"`
    Schema      *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type RequestBody struct {
    Description string               `json:"description,omitempty" yaml:"description,omitempty"`
    Content     map[string]MediaType `json:"content" yaml:"content"`
    Required    bool                 `json:"required,omitempty" yaml:"required,omitempty"`
}

type Response struct {
    Description string               `json:"description" yaml:"description"`
    Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

type MediaType struct {
    Schema  *Schema  `json:"schema,omitempty" yaml:"schema,omitempty"`
    Example any      `json:"example,omitempty" yaml:"example,omitempty"`
}

type Schema struct {
    Ref         string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
    Type        string             `json:"type,omitempty" yaml:"type,omitempty"`
    Format      string             `json:"format,omitempty" yaml:"format,omitempty"`
    Description string             `json:"description,omitempty" yaml:"description,omitempty"`
    Properties  map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
    Items       *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
    Required    []string           `json:"required,omitempty" yaml:"required,omitempty"`
    Enum        []string           `json:"enum,omitempty" yaml:"enum,omitempty"`
    Minimum     *float64           `json:"minimum,omitempty" yaml:"minimum,omitempty"`
    Maximum     *float64           `json:"maximum,omitempty" yaml:"maximum,omitempty"`
    MinLength   *int               `json:"minLength,omitempty" yaml:"minLength,omitempty"`
    MaxLength   *int               `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
    Default     any                `json:"default,omitempty" yaml:"default,omitempty"`
    Nullable    bool               `json:"nullable,omitempty" yaml:"nullable,omitempty"`
}

type Components struct {
    Schemas         map[string]*Schema         `json:"schemas,omitempty" yaml:"schemas,omitempty"`
    SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
}

type SecurityScheme struct {
    Type         string `json:"type" yaml:"type"`
    Scheme       string `json:"scheme,omitempty" yaml:"scheme,omitempty"`
    BearerFormat string `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"`
    In           string `json:"in,omitempty" yaml:"in,omitempty"`
    Name         string `json:"name,omitempty" yaml:"name,omitempty"`
}

type SecurityRequirement map[string][]string

type Tag struct {
    Name        string `json:"name" yaml:"name"`
    Description string `json:"description,omitempty" yaml:"description,omitempty"`
}
```

### 2. Generator (internal/openapi/generator.go)

```go
package openapi

import (
    "fmt"
    "strings"

    "github.com/codeai/codeai/internal/parser"
)

type Generator struct {
    config GeneratorConfig
}

type GeneratorConfig struct {
    Title       string
    Description string
    Version     string
    Server      string
    Contact     *Contact
    License     *License
}

func NewGenerator(config *GeneratorConfig) *Generator {
    if config == nil {
        config = &GeneratorConfig{}
    }
    return &Generator{config: *config}
}

func (g *Generator) Generate(ast *parser.AST) (*OpenAPI, error) {
    spec := &OpenAPI{
        OpenAPI: "3.0.3",
        Info: Info{
            Title:       g.getTitle(ast),
            Description: g.config.Description,
            Version:     g.getVersion(ast),
            Contact:     g.config.Contact,
            License:     g.config.License,
        },
        Paths:      make(map[string]PathItem),
        Components: Components{
            Schemas:         make(map[string]*Schema),
            SecuritySchemes: make(map[string]*SecurityScheme),
        },
    }

    // Add server if configured
    if g.config.Server != "" {
        spec.Servers = []Server{{URL: g.config.Server}}
    }

    // Generate schemas from entities
    for name, entity := range ast.Entities {
        spec.Components.Schemas[name] = g.entityToSchema(entity)
    }

    // Generate paths from endpoints
    for _, endpoint := range ast.Endpoints {
        g.addEndpoint(spec, endpoint)
    }

    // Add security schemes
    if ast.Config != nil && ast.Config.Auth != nil {
        g.addSecuritySchemes(spec, ast.Config.Auth)
    }

    // Generate tags from entity names
    spec.Tags = g.generateTags(ast)

    return spec, nil
}

func (g *Generator) entityToSchema(entity *parser.Entity) *Schema {
    schema := &Schema{
        Type:        "object",
        Description: entity.Description,
        Properties:  make(map[string]*Schema),
    }

    var required []string

    for _, field := range entity.Fields {
        fieldSchema := g.fieldToSchema(field)
        schema.Properties[field.Name] = fieldSchema

        if field.Required {
            required = append(required, field.Name)
        }
    }

    if len(required) > 0 {
        schema.Required = required
    }

    return schema
}

func (g *Generator) fieldToSchema(field *parser.Field) *Schema {
    schema := &Schema{}

    switch t := field.Type.(type) {
    case *parser.PrimitiveType:
        switch t.Name {
        case "string":
            schema.Type = "string"
        case "text":
            schema.Type = "string"
        case "integer":
            schema.Type = "integer"
            schema.Format = "int64"
        case "decimal":
            schema.Type = "number"
            schema.Format = "double"
        case "boolean":
            schema.Type = "boolean"
        case "uuid":
            schema.Type = "string"
            schema.Format = "uuid"
        case "timestamp":
            schema.Type = "string"
            schema.Format = "date-time"
        case "date":
            schema.Type = "string"
            schema.Format = "date"
        case "json":
            schema.Type = "object"
        }

    case *parser.ListType:
        schema.Type = "array"
        schema.Items = g.typeToSchema(t.ElementType)

    case *parser.RefType:
        schema.Ref = "#/components/schemas/" + t.EntityName

    case *parser.EnumType:
        schema.Type = "string"
        schema.Enum = t.Values
    }

    // Apply validators
    for _, v := range field.Validators {
        switch v.Type {
        case "min":
            if f, ok := v.Value.(float64); ok {
                schema.Minimum = &f
            }
        case "max":
            if f, ok := v.Value.(float64); ok {
                schema.Maximum = &f
            }
        case "minLength":
            if i, ok := v.Value.(int); ok {
                schema.MinLength = &i
            }
        case "maxLength":
            if i, ok := v.Value.(int); ok {
                schema.MaxLength = &i
            }
        }
    }

    if field.Default != nil {
        schema.Default = field.Default
    }

    if !field.Required {
        schema.Nullable = true
    }

    return schema
}

func (g *Generator) typeToSchema(t parser.Type) *Schema {
    switch typ := t.(type) {
    case *parser.PrimitiveType:
        return g.fieldToSchema(&parser.Field{Type: typ})
    case *parser.RefType:
        return &Schema{Ref: "#/components/schemas/" + typ.EntityName}
    default:
        return &Schema{Type: "object"}
    }
}

func (g *Generator) addEndpoint(spec *OpenAPI, endpoint *parser.Endpoint) {
    path := g.convertPath(endpoint.Path)

    pathItem, ok := spec.Paths[path]
    if !ok {
        pathItem = PathItem{}
    }

    operation := g.endpointToOperation(endpoint)

    switch endpoint.Method {
    case "GET":
        pathItem.Get = operation
    case "POST":
        pathItem.Post = operation
    case "PUT":
        pathItem.Put = operation
    case "DELETE":
        pathItem.Delete = operation
    case "PATCH":
        pathItem.Patch = operation
    }

    spec.Paths[path] = pathItem
}

func (g *Generator) endpointToOperation(endpoint *parser.Endpoint) *Operation {
    op := &Operation{
        Summary:     endpoint.Description,
        OperationID: g.generateOperationID(endpoint),
        Parameters:  []Parameter{},
        Responses:   make(map[string]Response),
    }

    // Add tag based on first path segment
    tag := g.getTag(endpoint.Path)
    if tag != "" {
        op.Tags = []string{tag}
    }

    // Add path parameters
    for _, param := range endpoint.PathParams {
        op.Parameters = append(op.Parameters, Parameter{
            Name:     param.Name,
            In:       "path",
            Required: true,
            Schema:   g.paramToSchema(param),
        })
    }

    // Add query parameters
    for _, param := range endpoint.QueryParams {
        op.Parameters = append(op.Parameters, Parameter{
            Name:     param.Name,
            In:       "query",
            Required: param.Required,
            Schema:   g.paramToSchema(param),
        })
    }

    // Add request body
    if len(endpoint.Body) > 0 {
        op.RequestBody = g.bodyToRequestBody(endpoint.Body)
    }

    // Add responses
    op.Responses["200"] = g.generateSuccessResponse(endpoint)
    op.Responses["400"] = Response{Description: "Bad Request"}
    op.Responses["401"] = Response{Description: "Unauthorized"}
    op.Responses["404"] = Response{Description: "Not Found"}
    op.Responses["500"] = Response{Description: "Internal Server Error"}

    // Add security
    if endpoint.Auth == parser.AuthRequired {
        op.Security = []SecurityRequirement{{"bearerAuth": {}}}
    }

    return op
}

func (g *Generator) bodyToRequestBody(params []*parser.Param) *RequestBody {
    schema := &Schema{
        Type:       "object",
        Properties: make(map[string]*Schema),
    }

    var required []string

    for _, param := range params {
        schema.Properties[param.Name] = g.paramToSchema(param)
        if param.Required {
            required = append(required, param.Name)
        }
    }

    schema.Required = required

    return &RequestBody{
        Required: true,
        Content: map[string]MediaType{
            "application/json": {Schema: schema},
        },
    }
}

func (g *Generator) generateSuccessResponse(endpoint *parser.Endpoint) Response {
    resp := Response{Description: "Success"}

    if endpoint.Returns != nil {
        var schema *Schema

        switch endpoint.Returns.Type {
        case "single":
            schema = &Schema{Ref: "#/components/schemas/" + endpoint.Returns.Entity}
        case "list":
            schema = &Schema{
                Type:  "array",
                Items: &Schema{Ref: "#/components/schemas/" + endpoint.Returns.Entity},
            }
        case "paginated":
            schema = &Schema{
                Type: "object",
                Properties: map[string]*Schema{
                    "data": {
                        Type:  "array",
                        Items: &Schema{Ref: "#/components/schemas/" + endpoint.Returns.Entity},
                    },
                    "pagination": {
                        Type: "object",
                        Properties: map[string]*Schema{
                            "total":    {Type: "integer"},
                            "page":     {Type: "integer"},
                            "limit":    {Type: "integer"},
                            "has_next": {Type: "boolean"},
                        },
                    },
                },
            }
        }

        if schema != nil {
            resp.Content = map[string]MediaType{
                "application/json": {Schema: schema},
            }
        }
    }

    return resp
}

func (g *Generator) convertPath(path string) string {
    // Convert CodeAI path to OpenAPI path
    // e.g., /users/{id} stays the same
    return path
}

func (g *Generator) generateOperationID(endpoint *parser.Endpoint) string {
    parts := strings.Split(endpoint.Path, "/")
    var name string

    for _, p := range parts {
        if p == "" || strings.HasPrefix(p, "{") {
            continue
        }
        name += strings.Title(p)
    }

    return strings.ToLower(endpoint.Method) + name
}

func (g *Generator) getTag(path string) string {
    parts := strings.Split(path, "/")
    for _, p := range parts {
        if p != "" && !strings.HasPrefix(p, "{") {
            return p
        }
    }
    return ""
}

func (g *Generator) getTitle(ast *parser.AST) string {
    if g.config.Title != "" {
        return g.config.Title
    }
    if ast.Config != nil && ast.Config.Name != "" {
        return ast.Config.Name + " API"
    }
    return "API"
}

func (g *Generator) getVersion(ast *parser.AST) string {
    if g.config.Version != "" {
        return g.config.Version
    }
    if ast.Config != nil && ast.Config.Version != "" {
        return ast.Config.Version
    }
    return "1.0.0"
}

func (g *Generator) paramToSchema(param *parser.Param) *Schema {
    // Convert param type to schema
    return &Schema{Type: "string"}
}

func (g *Generator) addSecuritySchemes(spec *OpenAPI, auth *parser.AuthConfig) {
    spec.Components.SecuritySchemes["bearerAuth"] = &SecurityScheme{
        Type:         "http",
        Scheme:       "bearer",
        BearerFormat: "JWT",
    }
}

func (g *Generator) generateTags(ast *parser.AST) []Tag {
    tagMap := make(map[string]bool)
    var tags []Tag

    for _, endpoint := range ast.Endpoints {
        tag := g.getTag(endpoint.Path)
        if tag != "" && !tagMap[tag] {
            tagMap[tag] = true
            tags = append(tags, Tag{Name: tag})
        }
    }

    return tags
}
```

## Acceptance Criteria
- [ ] Generate valid OpenAPI 3.0 spec
- [ ] Include all entities as schemas
- [ ] Include all endpoints with parameters
- [ ] Support authentication schemes
- [ ] Generate from AST
- [ ] Output YAML and JSON

## Testing Strategy
- Unit tests for schema generation
- Validate output against OpenAPI spec
- Integration tests with full CodeAI files

## Files to Create
- `internal/openapi/types.go`
- `internal/openapi/generator.go`
- `internal/openapi/generator_test.go`

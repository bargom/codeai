package openapi

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/parser"
	"gopkg.in/yaml.v3"
)

// Generator generates OpenAPI specifications from CodeAI AST.
type Generator struct {
	config    *Config
	mapper    *Mapper
	schemaGen *SchemaGenerator
	annParser *AnnotationParser
}

// NewGenerator creates a new OpenAPI generator with the given configuration.
func NewGenerator(config *Config) *Generator {
	if config == nil {
		config = DefaultConfig()
	}
	return &Generator{
		config:    config,
		mapper:    NewMapper(),
		schemaGen: NewSchemaGenerator(),
		annParser: NewAnnotationParser(),
	}
}

// GenerateFromFile generates an OpenAPI specification from a CodeAI source file.
func (g *Generator) GenerateFromFile(path string) (*OpenAPI, error) {
	// Parse the file
	program, err := parser.ParseFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	return g.GenerateFromAST(program)
}

// GenerateFromSource generates an OpenAPI specification from CodeAI source code.
func (g *Generator) GenerateFromSource(source string) (*OpenAPI, error) {
	// Parse the source
	program, err := parser.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
	}

	return g.GenerateFromAST(program)
}

// GenerateFromAST generates an OpenAPI specification from a parsed AST.
func (g *Generator) GenerateFromAST(program *ast.Program) (*OpenAPI, error) {
	// Map the AST to OpenAPI
	if err := g.mapper.MapProgram(program); err != nil {
		return nil, fmt.Errorf("failed to map AST: %w", err)
	}

	// Get the spec
	spec := g.mapper.GetSpec()

	// Apply configuration
	g.applyConfig(spec)

	// Generate tags from paths if none exist
	if len(spec.Tags) == 0 {
		g.generateTags(spec)
	}

	return spec, nil
}

// GenerateFromTypes generates an OpenAPI specification from Go types.
// This is useful for generating specs from existing Go struct definitions.
func (g *Generator) GenerateFromTypes(types ...interface{}) (*OpenAPI, error) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    g.config.ToInfo(),
		Servers: g.config.ToServers(),
		Paths:   make(map[string]PathItem),
		Components: Components{
			Schemas:         make(map[string]*Schema),
			SecuritySchemes: make(map[string]*SecurityScheme),
		},
	}

	for _, t := range types {
		schema := g.schemaGen.GenerateSchemaFromValue(t)
		typeName := reflect.TypeOf(t).Name()
		if typeName == "" {
			typeName = fmt.Sprintf("Type%d", len(spec.Components.Schemas))
		}
		spec.Components.Schemas[typeName] = schema
	}

	// Add any referenced schemas from the generator
	for name, schema := range g.schemaGen.Definitions {
		if _, exists := spec.Components.Schemas[name]; !exists {
			spec.Components.Schemas[name] = schema
		}
	}

	return spec, nil
}

// applyConfig applies the configuration to the spec.
func (g *Generator) applyConfig(spec *OpenAPI) {
	spec.Info = g.config.ToInfo()
	spec.Servers = g.config.ToServers()

	// Add default security if configured
	if len(g.config.DefaultSecurity) > 0 {
		for _, secName := range g.config.DefaultSecurity {
			spec.Security = append(spec.Security, SecurityRequirement{secName: {}})
		}
	}
}

// generateTags generates tags from the paths in the spec.
func (g *Generator) generateTags(spec *OpenAPI) {
	tagMap := make(map[string]bool)

	for path := range spec.Paths {
		// Extract first path segment as tag
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) > 0 && parts[0] != "" {
			tagName := parts[0]
			if !tagMap[tagName] {
				tagMap[tagName] = true
				spec.Tags = append(spec.Tags, Tag{
					Name:        tagName,
					Description: fmt.Sprintf("Operations for %s", tagName),
				})
			}
		}
	}
}

// WriteJSON writes the spec to a writer in JSON format.
func (g *Generator) WriteJSON(w io.Writer, spec *OpenAPI) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(spec)
}

// WriteYAML writes the spec to a writer in YAML format.
func (g *Generator) WriteYAML(w io.Writer, spec *OpenAPI) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	defer encoder.Close()
	return encoder.Encode(spec)
}

// WriteToFile writes the spec to a file in the configured format.
func (g *Generator) WriteToFile(path string, spec *OpenAPI) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	format := g.config.OutputFormat
	if format == "" {
		// Infer from file extension
		if strings.HasSuffix(path, ".json") {
			format = "json"
		} else {
			format = "yaml"
		}
	}

	switch format {
	case "json":
		return g.WriteJSON(file, spec)
	case "yaml", "yml":
		return g.WriteYAML(file, spec)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// ToJSON converts the spec to a JSON string.
func ToJSON(spec *OpenAPI) (string, error) {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToYAML converts the spec to a YAML string.
func ToYAML(spec *OpenAPI) (string, error) {
	data, err := yaml.Marshal(spec)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// AddRoute adds a route definition to the generator's mapper.
func (g *Generator) AddRoute(method, path, summary, description string) *Operation {
	op := &Operation{
		Summary:     summary,
		Description: description,
		OperationID: generateOperationID(method, path),
		Responses: map[string]Response{
			"200": {Description: "Success"},
		},
	}

	g.mapper.AddRoute(method, path, op)
	return op
}

// AddSchema adds a schema definition to the generator.
func (g *Generator) AddSchema(name string, schema *Schema) {
	g.mapper.AddSchema(name, schema)
}

// AddSchemaFromType adds a schema generated from a Go type.
func (g *Generator) AddSchemaFromType(name string, t interface{}) {
	schema := g.schemaGen.GenerateSchemaFromValue(t)
	g.mapper.AddSchema(name, schema)
}

// AddSecurityScheme adds a security scheme definition.
func (g *Generator) AddSecurityScheme(name string, scheme *SecurityScheme) {
	g.mapper.AddSecurityScheme(name, scheme)
}

// AddBearerAuth adds a bearer token authentication scheme.
func (g *Generator) AddBearerAuth(name string) {
	g.mapper.AddSecurityScheme(name, &SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
	})
}

// AddAPIKeyAuth adds an API key authentication scheme.
func (g *Generator) AddAPIKeyAuth(name, headerName string) {
	g.mapper.AddSecurityScheme(name, &SecurityScheme{
		Type: "apiKey",
		In:   "header",
		Name: headerName,
	})
}

// AddTag adds a tag definition.
func (g *Generator) AddTag(name, description string) {
	g.mapper.AddTag(Tag{
		Name:        name,
		Description: description,
	})
}

// GetSpec returns the current OpenAPI specification.
func (g *Generator) GetSpec() *OpenAPI {
	spec := g.mapper.GetSpec()
	g.applyConfig(spec)
	return spec
}

// generateOperationID generates a unique operation ID from method and path.
func generateOperationID(method, path string) string {
	// Convert path to camelCase
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var result strings.Builder

	result.WriteString(strings.ToLower(method))

	for _, part := range parts {
		// Skip path parameters
		if strings.HasPrefix(part, "{") {
			result.WriteString("ById")
			continue
		}
		result.WriteString(strings.Title(part))
	}

	return result.String()
}

// Merge merges another OpenAPI spec into this one.
func Merge(base, addition *OpenAPI) *OpenAPI {
	if base == nil {
		return addition
	}
	if addition == nil {
		return base
	}

	// Merge paths
	for path, item := range addition.Paths {
		if existing, ok := base.Paths[path]; ok {
			// Merge operations
			if item.Get != nil && existing.Get == nil {
				existing.Get = item.Get
			}
			if item.Post != nil && existing.Post == nil {
				existing.Post = item.Post
			}
			if item.Put != nil && existing.Put == nil {
				existing.Put = item.Put
			}
			if item.Delete != nil && existing.Delete == nil {
				existing.Delete = item.Delete
			}
			if item.Patch != nil && existing.Patch == nil {
				existing.Patch = item.Patch
			}
			base.Paths[path] = existing
		} else {
			base.Paths[path] = item
		}
	}

	// Merge schemas
	for name, schema := range addition.Components.Schemas {
		if _, exists := base.Components.Schemas[name]; !exists {
			base.Components.Schemas[name] = schema
		}
	}

	// Merge security schemes
	for name, scheme := range addition.Components.SecuritySchemes {
		if _, exists := base.Components.SecuritySchemes[name]; !exists {
			base.Components.SecuritySchemes[name] = scheme
		}
	}

	// Merge tags
	tagMap := make(map[string]bool)
	for _, tag := range base.Tags {
		tagMap[tag.Name] = true
	}
	for _, tag := range addition.Tags {
		if !tagMap[tag.Name] {
			base.Tags = append(base.Tags, tag)
		}
	}

	return base
}

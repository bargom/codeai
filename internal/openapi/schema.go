package openapi

import (
	"reflect"
	"strconv"
	"strings"
	"time"
)

// SchemaGenerator generates JSON Schemas from Go types.
type SchemaGenerator struct {
	// Definitions stores schema definitions for complex types
	Definitions map[string]*Schema
	// seen tracks types during generation to handle circular references
	seen map[reflect.Type]bool
}

// NewSchemaGenerator creates a new schema generator.
func NewSchemaGenerator() *SchemaGenerator {
	return &SchemaGenerator{
		Definitions: make(map[string]*Schema),
		seen:        make(map[reflect.Type]bool),
	}
}

// GenerateSchema generates an OpenAPI schema from a Go type.
func (g *SchemaGenerator) GenerateSchema(t reflect.Type) *Schema {
	return g.generateSchema(t, nil)
}

// GenerateSchemaFromValue generates an OpenAPI schema from a Go value.
func (g *SchemaGenerator) GenerateSchemaFromValue(v interface{}) *Schema {
	if v == nil {
		return &Schema{Type: "object"}
	}
	return g.GenerateSchema(reflect.TypeOf(v))
}

func (g *SchemaGenerator) generateSchema(t reflect.Type, field *reflect.StructField) *Schema {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		schema := g.generateSchema(t.Elem(), field)
		schema.Nullable = true
		return schema
	}

	// Handle known types first
	if schema := g.handleKnownType(t); schema != nil {
		g.applyFieldConstraints(schema, field)
		return schema
	}

	switch t.Kind() {
	case reflect.Bool:
		return g.applyFieldConstraints(&Schema{Type: "boolean"}, field)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return g.applyFieldConstraints(&Schema{Type: "integer", Format: "int32"}, field)

	case reflect.Int64:
		return g.applyFieldConstraints(&Schema{Type: "integer", Format: "int64"}, field)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return g.applyFieldConstraints(&Schema{Type: "integer", Format: "int32"}, field)

	case reflect.Uint64:
		return g.applyFieldConstraints(&Schema{Type: "integer", Format: "int64"}, field)

	case reflect.Float32:
		return g.applyFieldConstraints(&Schema{Type: "number", Format: "float"}, field)

	case reflect.Float64:
		return g.applyFieldConstraints(&Schema{Type: "number", Format: "double"}, field)

	case reflect.String:
		return g.applyFieldConstraints(&Schema{Type: "string"}, field)

	case reflect.Slice, reflect.Array:
		return g.generateArraySchema(t, field)

	case reflect.Map:
		return g.generateMapSchema(t, field)

	case reflect.Struct:
		return g.generateStructSchema(t, field)

	case reflect.Interface:
		return &Schema{Type: "object"}

	default:
		return &Schema{Type: "object"}
	}
}

func (g *SchemaGenerator) handleKnownType(t reflect.Type) *Schema {
	// Handle time.Time
	if t == reflect.TypeOf(time.Time{}) {
		return &Schema{Type: "string", Format: "date-time"}
	}

	// Handle json.RawMessage
	if t.String() == "json.RawMessage" {
		return &Schema{Type: "object"}
	}

	// Handle sql.NullString and similar
	if strings.HasPrefix(t.String(), "sql.Null") {
		switch t.String() {
		case "sql.NullString":
			return &Schema{Type: "string", Nullable: true}
		case "sql.NullInt64":
			return &Schema{Type: "integer", Format: "int64", Nullable: true}
		case "sql.NullInt32":
			return &Schema{Type: "integer", Format: "int32", Nullable: true}
		case "sql.NullFloat64":
			return &Schema{Type: "number", Format: "double", Nullable: true}
		case "sql.NullBool":
			return &Schema{Type: "boolean", Nullable: true}
		case "sql.NullTime":
			return &Schema{Type: "string", Format: "date-time", Nullable: true}
		}
	}

	return nil
}

func (g *SchemaGenerator) generateArraySchema(t reflect.Type, field *reflect.StructField) *Schema {
	schema := &Schema{
		Type:  "array",
		Items: g.generateSchema(t.Elem(), nil),
	}
	return g.applyFieldConstraints(schema, field)
}

func (g *SchemaGenerator) generateMapSchema(t reflect.Type, field *reflect.StructField) *Schema {
	schema := &Schema{
		Type:                 "object",
		AdditionalProperties: g.generateSchema(t.Elem(), nil),
	}
	return g.applyFieldConstraints(schema, field)
}

func (g *SchemaGenerator) generateStructSchema(t reflect.Type, parentField *reflect.StructField) *Schema {
	// Check for circular reference
	if g.seen[t] {
		// Return a reference instead
		name := t.Name()
		if name == "" {
			name = "AnonymousStruct"
		}
		return &Schema{Ref: "#/components/schemas/" + name}
	}

	// Mark as seen
	g.seen[t] = true
	defer func() { g.seen[t] = false }()

	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		// Parse JSON tag to get field name
		name := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				name = parts[0]
			}
			// Check for omitempty
			for _, part := range parts[1:] {
				if part == "omitempty" {
					// Field is optional
					break
				}
			}
		}

		// Generate schema for field
		fieldSchema := g.generateSchema(field.Type, &field)

		// Check if field is required from validate tag
		validateTag := field.Tag.Get("validate")
		if strings.Contains(validateTag, "required") {
			required = append(required, name)
		}

		schema.Properties[name] = fieldSchema
	}

	if len(required) > 0 {
		schema.Required = required
	}

	// Store in definitions if it has a name
	if t.Name() != "" {
		g.Definitions[t.Name()] = schema
	}

	return g.applyFieldConstraints(schema, parentField)
}

func (g *SchemaGenerator) applyFieldConstraints(schema *Schema, field *reflect.StructField) *Schema {
	if field == nil {
		return schema
	}

	// Apply validation constraints from struct tags
	validateTag := field.Tag.Get("validate")
	if validateTag != "" {
		g.applyValidateTag(schema, validateTag)
	}

	// Apply binding constraints (common in Gin framework)
	bindingTag := field.Tag.Get("binding")
	if bindingTag != "" {
		g.applyValidateTag(schema, bindingTag)
	}

	// Check for description in doc tag
	if desc := field.Tag.Get("description"); desc != "" {
		schema.Description = desc
	}

	// Check for example in example tag
	if example := field.Tag.Get("example"); example != "" {
		schema.Example = example
	}

	// Check for enum values
	if enumTag := field.Tag.Get("enum"); enumTag != "" {
		values := strings.Split(enumTag, ",")
		schema.Enum = make([]any, len(values))
		for i, v := range values {
			schema.Enum[i] = strings.TrimSpace(v)
		}
	}

	return schema
}

func (g *SchemaGenerator) applyValidateTag(schema *Schema, tag string) {
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Handle key=value format
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			key, value := kv[0], kv[1]

			switch key {
			case "min":
				if schema.Type == "string" {
					if v, err := strconv.Atoi(value); err == nil {
						schema.MinLength = &v
					}
				} else if schema.Type == "integer" || schema.Type == "number" {
					if v, err := strconv.ParseFloat(value, 64); err == nil {
						schema.Minimum = &v
					}
				} else if schema.Type == "array" {
					if v, err := strconv.Atoi(value); err == nil {
						schema.MinItems = &v
					}
				}
			case "max":
				if schema.Type == "string" {
					if v, err := strconv.Atoi(value); err == nil {
						schema.MaxLength = &v
					}
				} else if schema.Type == "integer" || schema.Type == "number" {
					if v, err := strconv.ParseFloat(value, 64); err == nil {
						schema.Maximum = &v
					}
				} else if schema.Type == "array" {
					if v, err := strconv.Atoi(value); err == nil {
						schema.MaxItems = &v
					}
				}
			case "len":
				if v, err := strconv.Atoi(value); err == nil {
					schema.MinLength = &v
					schema.MaxLength = &v
				}
			case "oneof":
				values := strings.Fields(value)
				schema.Enum = make([]any, len(values))
				for i, v := range values {
					schema.Enum[i] = v
				}
			case "gt":
				if v, err := strconv.ParseFloat(value, 64); err == nil {
					schema.ExclusiveMinimum = &v
				}
			case "gte":
				if v, err := strconv.ParseFloat(value, 64); err == nil {
					schema.Minimum = &v
				}
			case "lt":
				if v, err := strconv.ParseFloat(value, 64); err == nil {
					schema.ExclusiveMaximum = &v
				}
			case "lte":
				if v, err := strconv.ParseFloat(value, 64); err == nil {
					schema.Maximum = &v
				}
			}
		} else {
			// Handle flag-style validators
			switch part {
			case "required":
				// Handled at struct level
			case "email":
				schema.Format = "email"
			case "url":
				schema.Format = "uri"
			case "uuid", "uuid4":
				schema.Format = "uuid"
			case "alpha":
				schema.Pattern = "^[a-zA-Z]+$"
			case "alphanum":
				schema.Pattern = "^[a-zA-Z0-9]+$"
			case "numeric":
				schema.Pattern = "^[0-9]+$"
			case "ip", "ipv4":
				schema.Format = "ipv4"
			case "ipv6":
				schema.Format = "ipv6"
			case "datetime":
				schema.Format = "date-time"
			case "date":
				schema.Format = "date"
			}
		}
	}
}

// GenerateRef generates a reference schema to a named type.
func GenerateRef(name string) *Schema {
	return &Schema{Ref: "#/components/schemas/" + name}
}

// SchemaFromType creates a schema from a type name.
func SchemaFromType(typeName string) *Schema {
	switch typeName {
	case "string", "text":
		return &Schema{Type: "string"}
	case "integer", "int", "int32", "int64":
		return &Schema{Type: "integer", Format: "int64"}
	case "number", "float", "float32", "float64", "decimal":
		return &Schema{Type: "number", Format: "double"}
	case "boolean", "bool":
		return &Schema{Type: "boolean"}
	case "uuid":
		return &Schema{Type: "string", Format: "uuid"}
	case "timestamp", "datetime":
		return &Schema{Type: "string", Format: "date-time"}
	case "date":
		return &Schema{Type: "string", Format: "date"}
	case "time":
		return &Schema{Type: "string", Format: "time"}
	case "json", "object":
		return &Schema{Type: "object"}
	case "array":
		return &Schema{Type: "array"}
	default:
		// Assume it's a reference to another schema
		return GenerateRef(typeName)
	}
}

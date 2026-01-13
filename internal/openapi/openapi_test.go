package openapi

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/bargom/codeai/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Config Tests
// =============================================================================

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "API", config.Title)
	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, "yaml", config.OutputFormat)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name:    "missing title",
			config:  &Config{Version: "1.0.0"},
			wantErr: true,
		},
		{
			name:    "missing version",
			config:  &Config{Title: "API"},
			wantErr: true,
		},
		{
			name:    "invalid output format",
			config:  &Config{Title: "API", Version: "1.0.0", OutputFormat: "xml"},
			wantErr: true,
		},
		{
			name:    "valid json format",
			config:  &Config{Title: "API", Version: "1.0.0", OutputFormat: "json"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigToInfo(t *testing.T) {
	config := &Config{
		Title:        "My API",
		Description:  "A test API",
		Version:      "2.0.0",
		ContactName:  "John Doe",
		ContactEmail: "john@example.com",
		LicenseName:  "MIT",
		LicenseURL:   "https://opensource.org/licenses/MIT",
	}

	info := config.ToInfo()

	assert.Equal(t, "My API", info.Title)
	assert.Equal(t, "A test API", info.Description)
	assert.Equal(t, "2.0.0", info.Version)
	require.NotNil(t, info.Contact)
	assert.Equal(t, "John Doe", info.Contact.Name)
	assert.Equal(t, "john@example.com", info.Contact.Email)
	require.NotNil(t, info.License)
	assert.Equal(t, "MIT", info.License.Name)
	assert.Equal(t, "https://opensource.org/licenses/MIT", info.License.URL)
}

func TestConfigToServers(t *testing.T) {
	config := &Config{
		Servers: []ServerConfig{
			{URL: "https://api.example.com", Description: "Production"},
			{URL: "https://staging.example.com", Description: "Staging"},
		},
	}

	servers := config.ToServers()

	assert.Len(t, servers, 2)
	assert.Equal(t, "https://api.example.com", servers[0].URL)
	assert.Equal(t, "Production", servers[0].Description)
}

// =============================================================================
// Schema Generation Tests
// =============================================================================

func TestSchemaGeneratorPrimitiveTypes(t *testing.T) {
	gen := NewSchemaGenerator()

	tests := []struct {
		name       string
		value      interface{}
		wantType   string
		wantFormat string
	}{
		{"string", "hello", "string", ""},
		{"int", int(42), "integer", "int32"},
		{"int64", int64(42), "integer", "int64"},
		{"float32", float32(3.14), "number", "float"},
		{"float64", float64(3.14), "number", "double"},
		{"bool", true, "boolean", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := gen.GenerateSchemaFromValue(tt.value)
			assert.Equal(t, tt.wantType, schema.Type)
			assert.Equal(t, tt.wantFormat, schema.Format)
		})
	}
}

func TestSchemaGeneratorSlice(t *testing.T) {
	gen := NewSchemaGenerator()

	schema := gen.GenerateSchemaFromValue([]string{"a", "b", "c"})

	assert.Equal(t, "array", schema.Type)
	require.NotNil(t, schema.Items)
	assert.Equal(t, "string", schema.Items.Type)
}

func TestSchemaGeneratorMap(t *testing.T) {
	gen := NewSchemaGenerator()

	schema := gen.GenerateSchemaFromValue(map[string]int{"a": 1})

	assert.Equal(t, "object", schema.Type)
	require.NotNil(t, schema.AdditionalProperties)
	assert.Equal(t, "integer", schema.AdditionalProperties.Type)
}

func TestSchemaGeneratorTime(t *testing.T) {
	gen := NewSchemaGenerator()

	schema := gen.GenerateSchemaFromValue(time.Now())

	assert.Equal(t, "string", schema.Type)
	assert.Equal(t, "date-time", schema.Format)
}

func TestSchemaGeneratorStruct(t *testing.T) {
	type TestStruct struct {
		Name     string `json:"name" validate:"required"`
		Age      int    `json:"age" validate:"min=0,max=150"`
		Email    string `json:"email" validate:"email"`
		Optional string `json:"optional,omitempty"`
	}

	gen := NewSchemaGenerator()
	schema := gen.GenerateSchemaFromValue(TestStruct{})

	assert.Equal(t, "object", schema.Type)
	require.NotNil(t, schema.Properties)
	assert.Contains(t, schema.Properties, "name")
	assert.Contains(t, schema.Properties, "age")
	assert.Contains(t, schema.Properties, "email")
	assert.Contains(t, schema.Properties, "optional")
	assert.Contains(t, schema.Required, "name")

	// Check email format
	assert.Equal(t, "email", schema.Properties["email"].Format)

	// Check age constraints
	ageSchema := schema.Properties["age"]
	require.NotNil(t, ageSchema.Minimum)
	require.NotNil(t, ageSchema.Maximum)
	assert.Equal(t, float64(0), *ageSchema.Minimum)
	assert.Equal(t, float64(150), *ageSchema.Maximum)
}

func TestSchemaGeneratorPointer(t *testing.T) {
	gen := NewSchemaGenerator()

	var ptr *string
	schema := gen.GenerateSchema(reflect.TypeOf(ptr))

	assert.Equal(t, "string", schema.Type)
	assert.True(t, schema.Nullable)
}

func TestSchemaFromType(t *testing.T) {
	tests := []struct {
		typeName   string
		wantType   string
		wantFormat string
	}{
		{"string", "string", ""},
		{"integer", "integer", "int64"},
		{"number", "number", "double"},
		{"boolean", "boolean", ""},
		{"uuid", "string", "uuid"},
		{"timestamp", "string", "date-time"},
		{"date", "string", "date"},
		{"json", "object", ""},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			schema := SchemaFromType(tt.typeName)
			assert.Equal(t, tt.wantType, schema.Type)
			assert.Equal(t, tt.wantFormat, schema.Format)
		})
	}
}

func TestSchemaFromTypeRef(t *testing.T) {
	schema := SchemaFromType("CustomModel")

	assert.Equal(t, "#/components/schemas/CustomModel", schema.Ref)
}

// =============================================================================
// Annotation Parser Tests
// =============================================================================

func TestAnnotationParserSummary(t *testing.T) {
	parser := NewAnnotationParser()

	annotations := parser.ParseComment("@Summary Get user by ID")

	require.Len(t, annotations, 1)
	assert.Equal(t, "summary", annotations[0].Type)
	assert.Equal(t, "Get user by ID", annotations[0].Values[0])
}

func TestAnnotationParserDescription(t *testing.T) {
	parser := NewAnnotationParser()

	annotations := parser.ParseComment("@Description This endpoint retrieves a user")

	require.Len(t, annotations, 1)
	assert.Equal(t, "description", annotations[0].Type)
	assert.Equal(t, "This endpoint retrieves a user", annotations[0].Values[0])
}

func TestAnnotationParserTags(t *testing.T) {
	parser := NewAnnotationParser()

	annotations := parser.ParseComment("@Tags users, admin")

	require.Len(t, annotations, 1)
	assert.Equal(t, "tags", annotations[0].Type)
	assert.Equal(t, "users, admin", annotations[0].Values[0])
}

func TestAnnotationParserParam(t *testing.T) {
	parser := NewAnnotationParser()

	annotations := parser.ParseComment("@Param id path string true \"User ID\"")

	require.Len(t, annotations, 1)
	assert.Equal(t, "param", annotations[0].Type)
	assert.Equal(t, "id", annotations[0].Name)
	assert.Equal(t, "path", annotations[0].Values[0])
	assert.Equal(t, "string", annotations[0].Values[1])
	assert.Equal(t, "true", annotations[0].Values[2])
}

func TestAnnotationParserSuccess(t *testing.T) {
	parser := NewAnnotationParser()

	annotations := parser.ParseComment("@Success 200 {object} User")

	require.Len(t, annotations, 1)
	assert.Equal(t, "success", annotations[0].Type)
	assert.Equal(t, "200", annotations[0].Name)
}

func TestAnnotationParserRouter(t *testing.T) {
	parser := NewAnnotationParser()

	annotations := parser.ParseComment("@Router /users/{id} [get]")

	require.Len(t, annotations, 1)
	assert.Equal(t, "router", annotations[0].Type)
	assert.Equal(t, "/users/{id} [get]", annotations[0].Values[0])
}

func TestAnnotationParserMultiple(t *testing.T) {
	parser := NewAnnotationParser()

	comments := []string{
		"@Summary Get user by ID",
		"@Description Retrieves a specific user",
		"@Tags users",
		"@Param id path string true \"User ID\"",
		"@Success 200 {object} User",
		"@Failure 404 {object} Error",
		"@Router /users/{id} [get]",
	}

	annotations := parser.ParseComments(comments)

	assert.Len(t, annotations, 7)
}

func TestExtractOperationMeta(t *testing.T) {
	parser := NewAnnotationParser()

	comments := []string{
		"@Summary Get user by ID",
		"@Description Retrieves a specific user from the database",
		"@Tags users",
		"@ID getUserById",
		"@Accept json",
		"@Produce json",
		"@Param id path string true \"User ID\"",
		"@Success 200 {object} User",
		"@Failure 404 {object} Error \"Not found\"",
		"@Router /users/{id} [get]",
	}

	annotations := parser.ParseComments(comments)
	meta := ExtractOperationMeta(annotations)

	assert.Equal(t, "Get user by ID", meta.Summary)
	assert.Equal(t, "Retrieves a specific user from the database", meta.Description)
	assert.Contains(t, meta.Tags, "users")
	assert.Equal(t, "getUserById", meta.OperationID)
	assert.Contains(t, meta.Accept, "application/json")
	assert.Contains(t, meta.Produce, "application/json")
	require.NotNil(t, meta.Router)
	assert.Equal(t, "/users/{id}", meta.Router.Path)
	assert.Equal(t, "GET", meta.Router.Method)
	assert.Len(t, meta.Params, 1)
	assert.Len(t, meta.Responses, 2)
}

func TestNormalizeMediaType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"json", "application/json"},
		{"xml", "application/xml"},
		{"plain", "text/plain"},
		{"html", "text/html"},
		{"form", "application/x-www-form-urlencoded"},
		{"mpfd", "multipart/form-data"},
		{"application/json", "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeMediaType(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

// =============================================================================
// Mapper Tests
// =============================================================================

func TestMapperAddRoute(t *testing.T) {
	mapper := NewMapper()

	op := &Operation{
		Summary:   "Get users",
		Responses: map[string]Response{"200": {Description: "Success"}},
	}

	mapper.AddRoute("GET", "/users", op)

	pathItem := mapper.Spec.Paths["/users"]
	require.NotNil(t, pathItem.Get)
	assert.Equal(t, "Get users", pathItem.Get.Summary)
}

func TestMapperAddMultipleRoutes(t *testing.T) {
	mapper := NewMapper()

	mapper.AddRoute("GET", "/users", &Operation{Summary: "List users", Responses: map[string]Response{"200": {Description: "Success"}}})
	mapper.AddRoute("POST", "/users", &Operation{Summary: "Create user", Responses: map[string]Response{"201": {Description: "Created"}}})
	mapper.AddRoute("GET", "/users/{id}", &Operation{Summary: "Get user", Responses: map[string]Response{"200": {Description: "Success"}}})

	assert.Len(t, mapper.Spec.Paths, 2)

	usersPath := mapper.Spec.Paths["/users"]
	require.NotNil(t, usersPath.Get)
	require.NotNil(t, usersPath.Post)

	userByIdPath := mapper.Spec.Paths["/users/{id}"]
	require.NotNil(t, userByIdPath.Get)
}

func TestMapperAddSchema(t *testing.T) {
	mapper := NewMapper()

	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"id":   {Type: "string"},
			"name": {Type: "string"},
		},
	}

	mapper.AddSchema("User", schema)

	assert.Contains(t, mapper.Spec.Components.Schemas, "User")
	assert.Equal(t, schema, mapper.Spec.Components.Schemas["User"])
}

func TestMapperAddSecurityScheme(t *testing.T) {
	mapper := NewMapper()

	scheme := &SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
	}

	mapper.AddSecurityScheme("bearerAuth", scheme)

	assert.Contains(t, mapper.Spec.Components.SecuritySchemes, "bearerAuth")
}

func TestMapperAddTag(t *testing.T) {
	mapper := NewMapper()

	mapper.AddTag(Tag{Name: "users", Description: "User operations"})
	mapper.AddTag(Tag{Name: "users", Description: "Duplicate"}) // Should be ignored

	assert.Len(t, mapper.Spec.Tags, 1)
	assert.Equal(t, "users", mapper.Spec.Tags[0].Name)
}

func TestParseHandlerName(t *testing.T) {
	tests := []struct {
		name       string
		wantMethod string
		wantPath   string
	}{
		{"getUsers", "GET", "/users"},
		{"getUserById", "GET", "/user/{id}"},
		{"createUser", "POST", "/user"},
		{"updateUser", "PUT", "/user"},
		{"deleteUser", "DELETE", "/user"},
		{"listProducts", "GET", "/products"},
		{"addProduct", "POST", "/product"},
		{"handle_get_items", "GET", "/items"},
		{"notAnEndpoint", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, path := parseHandlerName(tt.name)
			assert.Equal(t, tt.wantMethod, method)
			assert.Equal(t, tt.wantPath, path)
		})
	}
}

func TestGenerateSummary(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   string
	}{
		{"GET", "/users", "List Users"},
		{"GET", "/users/{id}", "Get User by ID"},
		{"POST", "/users", "Create User"},
		{"PUT", "/users/{id}", "Update User"},
		{"DELETE", "/users/{id}", "Delete User"},
	}

	for _, tt := range tests {
		t.Run(tt.method+"_"+tt.path, func(t *testing.T) {
			result := generateSummary(tt.method, tt.path)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestSingularize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"users", "user"},
		{"categories", "category"},
		{"boxes", "box"},
		{"user", "user"},
		{"boss", "boss"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := singularize(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

// =============================================================================
// Generator Tests
// =============================================================================

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator(nil)
	assert.NotNil(t, gen)
}

func TestGeneratorGenerateFromAST(t *testing.T) {
	config := &Config{
		Title:   "Test API",
		Version: "1.0.0",
	}
	gen := NewGenerator(config)

	program := &ast.Program{
		Statements: []ast.Statement{},
	}

	spec, err := gen.GenerateFromAST(program)
	require.NoError(t, err)
	require.NotNil(t, spec)

	assert.Equal(t, "3.0.0", spec.OpenAPI)
	assert.Equal(t, "Test API", spec.Info.Title)
	assert.Equal(t, "1.0.0", spec.Info.Version)
}

func TestGeneratorAddRoute(t *testing.T) {
	gen := NewGenerator(DefaultConfig())

	op := gen.AddRoute("GET", "/users", "List users", "Get all users")

	require.NotNil(t, op)
	spec := gen.GetSpec()
	assert.Contains(t, spec.Paths, "/users")
}

func TestGeneratorAddBearerAuth(t *testing.T) {
	gen := NewGenerator(DefaultConfig())

	gen.AddBearerAuth("bearerAuth")

	spec := gen.GetSpec()
	scheme := spec.Components.SecuritySchemes["bearerAuth"]
	require.NotNil(t, scheme)
	assert.Equal(t, "http", scheme.Type)
	assert.Equal(t, "bearer", scheme.Scheme)
	assert.Equal(t, "JWT", scheme.BearerFormat)
}

func TestGeneratorAddAPIKeyAuth(t *testing.T) {
	gen := NewGenerator(DefaultConfig())

	gen.AddAPIKeyAuth("apiKey", "X-API-Key")

	spec := gen.GetSpec()
	scheme := spec.Components.SecuritySchemes["apiKey"]
	require.NotNil(t, scheme)
	assert.Equal(t, "apiKey", scheme.Type)
	assert.Equal(t, "header", scheme.In)
	assert.Equal(t, "X-API-Key", scheme.Name)
}

func TestToJSON(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Paths:   map[string]PathItem{},
	}

	jsonStr, err := ToJSON(spec)
	require.NoError(t, err)
	assert.Contains(t, jsonStr, `"openapi": "3.0.0"`)
	assert.Contains(t, jsonStr, `"title": "Test"`)
}

func TestToYAML(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Paths:   map[string]PathItem{},
	}

	yamlStr, err := ToYAML(spec)
	require.NoError(t, err)
	assert.Contains(t, yamlStr, "openapi: 3.0.0")
	assert.Contains(t, yamlStr, "title: Test")
}

func TestMerge(t *testing.T) {
	base := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Base API", Version: "1.0.0"},
		Paths: map[string]PathItem{
			"/users": {Get: &Operation{Summary: "Get users"}},
		},
		Components: Components{
			Schemas: map[string]*Schema{
				"User": {Type: "object"},
			},
			SecuritySchemes: map[string]*SecurityScheme{},
		},
		Tags: []Tag{{Name: "users"}},
	}

	addition := &OpenAPI{
		Paths: map[string]PathItem{
			"/users":    {Post: &Operation{Summary: "Create user"}},
			"/products": {Get: &Operation{Summary: "Get products"}},
		},
		Components: Components{
			Schemas: map[string]*Schema{
				"Product": {Type: "object"},
			},
			SecuritySchemes: map[string]*SecurityScheme{},
		},
		Tags: []Tag{{Name: "products"}},
	}

	merged := Merge(base, addition)

	// Check paths merged
	assert.Len(t, merged.Paths, 2)
	usersPath := merged.Paths["/users"]
	assert.NotNil(t, usersPath.Get)
	assert.NotNil(t, usersPath.Post)

	// Check schemas merged
	assert.Contains(t, merged.Components.Schemas, "User")
	assert.Contains(t, merged.Components.Schemas, "Product")

	// Check tags merged
	assert.Len(t, merged.Tags, 2)
}

// =============================================================================
// Validator Tests
// =============================================================================

func TestValidatorValidSpec(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Test API", Version: "1.0.0"},
		Paths: map[string]PathItem{
			"/users": {
				Get: &Operation{
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
		},
		Components: Components{
			Schemas:         map[string]*Schema{},
			SecuritySchemes: map[string]*SecurityScheme{},
		},
	}

	result := ValidateSpec(spec)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidatorInvalidVersion(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "2.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Paths:   map[string]PathItem{},
	}

	result := ValidateSpec(spec)

	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestValidatorMissingTitle(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Version: "1.0.0"},
		Paths:   map[string]PathItem{},
	}

	result := ValidateSpec(spec)

	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestValidatorMissingVersion(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Test"},
		Paths:   map[string]PathItem{},
	}

	result := ValidateSpec(spec)

	assert.False(t, result.Valid)
}

func TestValidatorPathNotStartingWithSlash(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]PathItem{
			"users": {},
		},
	}

	result := ValidateSpec(spec)

	assert.False(t, result.Valid)
}

func TestValidatorMissingResponses(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]PathItem{
			"/users": {
				Get: &Operation{},
			},
		},
	}

	result := ValidateSpec(spec)

	assert.False(t, result.Valid)
}

func TestValidatorPathParameterNotRequired(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]PathItem{
			"/users/{id}": {
				Get: &Operation{
					Parameters: []Parameter{
						{Name: "id", In: "path", Required: false},
					},
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
		},
	}

	result := ValidateSpec(spec)

	assert.False(t, result.Valid)
}

func TestValidatorInvalidParameterLocation(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]PathItem{
			"/users": {
				Get: &Operation{
					Parameters: []Parameter{
						{Name: "sort", In: "invalid"},
					},
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
		},
	}

	result := ValidateSpec(spec)

	assert.False(t, result.Valid)
}

func TestValidatorInvalidSecuritySchemeType(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Paths:   map[string]PathItem{},
		Components: Components{
			SecuritySchemes: map[string]*SecurityScheme{
				"invalid": {Type: "invalid"},
			},
		},
	}

	result := ValidateSpec(spec)

	assert.False(t, result.Valid)
}

func TestValidatorAPIKeyMissingName(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Paths:   map[string]PathItem{},
		Components: Components{
			SecuritySchemes: map[string]*SecurityScheme{
				"apiKey": {Type: "apiKey", In: "header"},
			},
		},
	}

	result := ValidateSpec(spec)

	assert.False(t, result.Valid)
}

func TestValidatorHTTPMissingScheme(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Paths:   map[string]PathItem{},
		Components: Components{
			SecuritySchemes: map[string]*SecurityScheme{
				"bearer": {Type: "http"},
			},
		},
	}

	result := ValidateSpec(spec)

	assert.False(t, result.Valid)
}

func TestValidatorArrayMissingItems(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Paths:   map[string]PathItem{},
		Components: Components{
			Schemas: map[string]*Schema{
				"InvalidArray": {Type: "array"},
			},
		},
	}

	result := ValidateSpec(spec)

	assert.False(t, result.Valid)
}

func TestValidatorStrictMode(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]PathItem{
			"/users": {
				Get: &Operation{
					// Missing operationId
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
		},
	}

	result := ValidateSpecStrict(spec)

	// Should be valid but have warnings
	assert.True(t, result.Valid)
	assert.NotEmpty(t, result.Warnings)
}

func TestValidatorSecurityNotDefined(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI:  "3.0.0",
		Info:     Info{Title: "Test", Version: "1.0.0"},
		Paths:    map[string]PathItem{},
		Security: []SecurityRequirement{{"undefined": {}}},
		Components: Components{
			SecuritySchemes: map[string]*SecurityScheme{},
		},
	}

	result := ValidateSpec(spec)

	assert.False(t, result.Valid)
}

// =============================================================================
// Types Tests
// =============================================================================

func TestOpenAPITypesJSONMarshal(t *testing.T) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       "Test API",
			Description: "A test API",
			Version:     "1.0.0",
			Contact: &Contact{
				Name:  "Test",
				Email: "test@example.com",
			},
		},
		Servers: []Server{
			{URL: "https://api.example.com", Description: "Production"},
		},
		Paths: map[string]PathItem{
			"/users": {
				Get: &Operation{
					Summary:     "List users",
					OperationID: "listUsers",
					Tags:        []string{"users"},
					Responses: map[string]Response{
						"200": {
							Description: "Success",
							Content: map[string]MediaType{
								"application/json": {
									Schema: &Schema{
										Type: "array",
										Items: &Schema{
											Ref: "#/components/schemas/User",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Components: Components{
			Schemas: map[string]*Schema{
				"User": {
					Type: "object",
					Properties: map[string]*Schema{
						"id":   {Type: "string", Format: "uuid"},
						"name": {Type: "string"},
					},
					Required: []string{"id", "name"},
				},
			},
			SecuritySchemes: map[string]*SecurityScheme{
				"bearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
				},
			},
		},
	}

	data, err := json.Marshal(spec)
	require.NoError(t, err)

	// Verify we can unmarshal it back
	var parsed OpenAPI
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, spec.OpenAPI, parsed.OpenAPI)
	assert.Equal(t, spec.Info.Title, parsed.Info.Title)
}

// =============================================================================
// Endpoint Mapping Tests
// =============================================================================

func TestMapEndpoint(t *testing.T) {
	mapper := NewMapper()

	endpoint := &ast.EndpointDecl{
		Method: ast.HTTPMethodGET,
		Path:   "/users/:id",
		Handler: &ast.Handler{
			Request: &ast.RequestType{
				TypeName: "UserID",
				Source:   ast.RequestSourcePath,
			},
			Response: &ast.ResponseType{
				TypeName:   "User",
				StatusCode: 200,
			},
		},
	}

	err := mapper.mapEndpoint(endpoint)
	require.NoError(t, err)

	// Check path was converted
	assert.Contains(t, mapper.Spec.Paths, "/users/{id}")

	pathItem := mapper.Spec.Paths["/users/{id}"]
	require.NotNil(t, pathItem.Get)

	// Check operation
	op := pathItem.Get
	assert.NotEmpty(t, op.OperationID)
	assert.NotEmpty(t, op.Summary)
	assert.Contains(t, op.Responses, "200")
}

func TestMapEndpointWithRequestBody(t *testing.T) {
	mapper := NewMapper()

	endpoint := &ast.EndpointDecl{
		Method: ast.HTTPMethodPOST,
		Path:   "/users",
		Handler: &ast.Handler{
			Request: &ast.RequestType{
				TypeName: "CreateUserRequest",
				Source:   ast.RequestSourceBody,
			},
			Response: &ast.ResponseType{
				TypeName:   "User",
				StatusCode: 201,
			},
		},
	}

	err := mapper.mapEndpoint(endpoint)
	require.NoError(t, err)

	pathItem := mapper.Spec.Paths["/users"]
	require.NotNil(t, pathItem.Post)

	op := pathItem.Post
	require.NotNil(t, op.RequestBody)
	assert.True(t, op.RequestBody.Required)
	assert.Contains(t, op.RequestBody.Content, "application/json")
	assert.Contains(t, op.Responses, "201")
}

func TestMapEndpointWithAnnotations(t *testing.T) {
	mapper := NewMapper()

	endpoint := &ast.EndpointDecl{
		Method: ast.HTTPMethodDELETE,
		Path:   "/users/:id",
		Annotations: []*ast.Annotation{
			{Name: "deprecated"},
			{Name: "auth", Value: "admin"},
			{Name: "summary", Value: "Delete a user"},
		},
		Handler: &ast.Handler{
			Response: &ast.ResponseType{
				StatusCode: 204,
			},
		},
	}

	err := mapper.mapEndpoint(endpoint)
	require.NoError(t, err)

	pathItem := mapper.Spec.Paths["/users/{id}"]
	require.NotNil(t, pathItem.Delete)

	op := pathItem.Delete
	assert.True(t, op.Deprecated)
	assert.Equal(t, "Delete a user", op.Summary)
	require.Len(t, op.Security, 1)
	assert.Contains(t, op.Security[0], "admin")
}

func TestMapEndpointWithMiddlewares(t *testing.T) {
	mapper := NewMapper()

	endpoint := &ast.EndpointDecl{
		Method: ast.HTTPMethodGET,
		Path:   "/admin/users",
		Middlewares: []*ast.MiddlewareRef{
			{Name: "auth"},
			{Name: "rateLimit"},
		},
		Handler: &ast.Handler{
			Response: &ast.ResponseType{
				TypeName:   "UserList",
				StatusCode: 200,
			},
		},
	}

	err := mapper.mapEndpoint(endpoint)
	require.NoError(t, err)

	pathItem := mapper.Spec.Paths["/admin/users"]
	require.NotNil(t, pathItem.Get)

	op := pathItem.Get
	assert.Equal(t, []string{"auth", "rateLimit"}, op.XCodeAIMiddleware)
}

// =============================================================================
// Model Mapping Tests
// =============================================================================

func TestMapModel(t *testing.T) {
	mapper := NewMapper()

	model := &ast.ModelDecl{
		Name:        "User",
		Description: "A user in the system",
		Fields: []*ast.FieldDecl{
			{
				Name:      "id",
				FieldType: &ast.TypeRef{Name: "uuid"},
				Modifiers: []*ast.Modifier{
					{Name: "primary"},
					{Name: "required"},
				},
			},
			{
				Name:      "email",
				FieldType: &ast.TypeRef{Name: "email"},
				Modifiers: []*ast.Modifier{
					{Name: "unique"},
					{Name: "required"},
				},
			},
			{
				Name:      "name",
				FieldType: &ast.TypeRef{Name: "string"},
			},
			{
				Name:      "age",
				FieldType: &ast.TypeRef{Name: "int"},
				Modifiers: []*ast.Modifier{
					{Name: "nullable"},
				},
			},
		},
	}

	err := mapper.mapModel(model)
	require.NoError(t, err)

	schema, exists := mapper.Spec.Components.Schemas["User"]
	require.True(t, exists)
	assert.Equal(t, "object", schema.Type)
	assert.Equal(t, "A user in the system", schema.Description)

	// Check properties
	assert.Contains(t, schema.Properties, "id")
	assert.Contains(t, schema.Properties, "email")
	assert.Contains(t, schema.Properties, "name")
	assert.Contains(t, schema.Properties, "age")

	// Check types
	assert.Equal(t, "string", schema.Properties["id"].Type)
	assert.Equal(t, "uuid", schema.Properties["id"].Format)
	assert.Equal(t, "string", schema.Properties["email"].Type)
	assert.Equal(t, "email", schema.Properties["email"].Format)
	assert.Equal(t, "string", schema.Properties["name"].Type)
	assert.Equal(t, "integer", schema.Properties["age"].Type)
	assert.True(t, schema.Properties["age"].Nullable)

	// Check required fields
	assert.Contains(t, schema.Required, "id")
	assert.Contains(t, schema.Required, "email")
	assert.NotContains(t, schema.Required, "name")
}

func TestMapModelWithArrayField(t *testing.T) {
	mapper := NewMapper()

	model := &ast.ModelDecl{
		Name: "Post",
		Fields: []*ast.FieldDecl{
			{
				Name: "tags",
				FieldType: &ast.TypeRef{
					Name:   "list",
					Params: []*ast.TypeRef{{Name: "string"}},
				},
			},
		},
	}

	err := mapper.mapModel(model)
	require.NoError(t, err)

	schema := mapper.Spec.Components.Schemas["Post"]
	require.NotNil(t, schema.Properties["tags"])

	tagsSchema := schema.Properties["tags"]
	assert.Equal(t, "array", tagsSchema.Type)
	require.NotNil(t, tagsSchema.Items)
	assert.Equal(t, "string", tagsSchema.Items.Type)
}

func TestMapModelWithReference(t *testing.T) {
	mapper := NewMapper()

	model := &ast.ModelDecl{
		Name: "Post",
		Fields: []*ast.FieldDecl{
			{
				Name: "author",
				FieldType: &ast.TypeRef{
					Name:   "ref",
					Params: []*ast.TypeRef{{Name: "User"}},
				},
			},
		},
	}

	err := mapper.mapModel(model)
	require.NoError(t, err)

	schema := mapper.Spec.Components.Schemas["Post"]
	require.NotNil(t, schema.Properties["author"])

	authorSchema := schema.Properties["author"]
	assert.Equal(t, "#/components/schemas/User", authorSchema.Ref)
}

// =============================================================================
// Auth Mapping Tests
// =============================================================================

func TestMapAuthJWT(t *testing.T) {
	mapper := NewMapper()

	auth := &ast.AuthDecl{
		Name:   "jwt_auth",
		Method: ast.AuthMethodJWT,
		JWKS: &ast.JWKSConfig{
			URL:      "https://auth.example.com/.well-known/jwks.json",
			Issuer:   "https://auth.example.com",
			Audience: "api.example.com",
		},
	}

	err := mapper.mapAuth(auth)
	require.NoError(t, err)

	scheme, exists := mapper.Spec.Components.SecuritySchemes["jwt_auth"]
	require.True(t, exists)
	assert.Equal(t, "http", scheme.Type)
	assert.Equal(t, "bearer", scheme.Scheme)
	assert.Equal(t, "JWT", scheme.BearerFormat)
	assert.Contains(t, scheme.Description, "JWKS")
}

func TestMapAuthAPIKey(t *testing.T) {
	mapper := NewMapper()

	auth := &ast.AuthDecl{
		Name:   "api_key",
		Method: ast.AuthMethodAPIKey,
		Config: map[string]ast.Expression{
			"header": &ast.StringLiteral{Value: "X-Custom-API-Key"},
		},
	}

	err := mapper.mapAuth(auth)
	require.NoError(t, err)

	scheme, exists := mapper.Spec.Components.SecuritySchemes["api_key"]
	require.True(t, exists)
	assert.Equal(t, "apiKey", scheme.Type)
	assert.Equal(t, "header", scheme.In)
	assert.Equal(t, "X-Custom-API-Key", scheme.Name)
}

func TestMapAuthBasic(t *testing.T) {
	mapper := NewMapper()

	auth := &ast.AuthDecl{
		Name:   "basic_auth",
		Method: ast.AuthMethodBasic,
	}

	err := mapper.mapAuth(auth)
	require.NoError(t, err)

	scheme, exists := mapper.Spec.Components.SecuritySchemes["basic_auth"]
	require.True(t, exists)
	assert.Equal(t, "http", scheme.Type)
	assert.Equal(t, "basic", scheme.Scheme)
}

func TestMapAuthOAuth2(t *testing.T) {
	mapper := NewMapper()

	auth := &ast.AuthDecl{
		Name:   "oauth2",
		Method: ast.AuthMethodOAuth2,
		Config: map[string]ast.Expression{
			"authorization_url": &ast.StringLiteral{Value: "https://auth.example.com/authorize"},
			"token_url":         &ast.StringLiteral{Value: "https://auth.example.com/token"},
		},
	}

	err := mapper.mapAuth(auth)
	require.NoError(t, err)

	scheme, exists := mapper.Spec.Components.SecuritySchemes["oauth2"]
	require.True(t, exists)
	assert.Equal(t, "oauth2", scheme.Type)
	require.NotNil(t, scheme.Flows)
	require.NotNil(t, scheme.Flows.AuthorizationCode)
	assert.Equal(t, "https://auth.example.com/authorize", scheme.Flows.AuthorizationCode.AuthorizationURL)
	assert.Equal(t, "https://auth.example.com/token", scheme.Flows.AuthorizationCode.TokenURL)
}

// =============================================================================
// Path Conversion Tests
// =============================================================================

func TestConvertPathParams(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users/:id", "/users/{id}"},
		{"/users/:userId/posts/:postId", "/users/{userId}/posts/{postId}"},
		{"/users", "/users"},
		{"/:version/api/:resource", "/{version}/api/{resource}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertPathParams(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPathParamsFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{"/users/:id", []string{"id"}},
		{"/users/:userId/posts/:postId", []string{"userId", "postId"}},
		{"/users/{id}", []string{"id"}},
		{"/users", nil},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := extractPathParamsFromPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractTagsFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{"/users", []string{"users"}},
		{"/users/:id", []string{"users"}},
		{"/api/v1/users", []string{"api"}},
		{"/:id", nil},
		{"/", nil},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := extractTagsFromPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStatusDescription(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{200, "OK"},
		{201, "Created"},
		{204, "No Content"},
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{404, "Not Found"},
		{500, "Internal Server Error"},
		{418, "Response 418"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := getStatusDescription(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Full AST Integration Test
// =============================================================================

func TestGenerateFromASTWithEndpointsModelsAuth(t *testing.T) {
	config := &Config{
		Title:   "Test API",
		Version: "1.0.0",
	}
	gen := NewGenerator(config)

	// Create a program with models, auth, and endpoints
	program := &ast.Program{
		Statements: []ast.Statement{
			// Auth declaration
			&ast.AuthDecl{
				Name:   "bearerAuth",
				Method: ast.AuthMethodJWT,
			},
			// Model declarations
			&ast.ModelDecl{
				Name: "User",
				Fields: []*ast.FieldDecl{
					{Name: "id", FieldType: &ast.TypeRef{Name: "uuid"}, Modifiers: []*ast.Modifier{{Name: "primary"}, {Name: "required"}}},
					{Name: "email", FieldType: &ast.TypeRef{Name: "email"}, Modifiers: []*ast.Modifier{{Name: "required"}}},
					{Name: "name", FieldType: &ast.TypeRef{Name: "string"}},
				},
			},
			&ast.ModelDecl{
				Name: "CreateUserRequest",
				Fields: []*ast.FieldDecl{
					{Name: "email", FieldType: &ast.TypeRef{Name: "email"}, Modifiers: []*ast.Modifier{{Name: "required"}}},
					{Name: "name", FieldType: &ast.TypeRef{Name: "string"}, Modifiers: []*ast.Modifier{{Name: "required"}}},
				},
			},
			// Endpoint declarations
			&ast.EndpointDecl{
				Method: ast.HTTPMethodGET,
				Path:   "/users",
				Handler: &ast.Handler{
					Response: &ast.ResponseType{TypeName: "User", StatusCode: 200},
				},
			},
			&ast.EndpointDecl{
				Method: ast.HTTPMethodPOST,
				Path:   "/users",
				Handler: &ast.Handler{
					Request:  &ast.RequestType{TypeName: "CreateUserRequest", Source: ast.RequestSourceBody},
					Response: &ast.ResponseType{TypeName: "User", StatusCode: 201},
				},
			},
			&ast.EndpointDecl{
				Method: ast.HTTPMethodGET,
				Path:   "/users/:id",
				Handler: &ast.Handler{
					Request:  &ast.RequestType{TypeName: "UserID", Source: ast.RequestSourcePath},
					Response: &ast.ResponseType{TypeName: "User", StatusCode: 200},
				},
			},
		},
	}

	spec, err := gen.GenerateFromAST(program)
	require.NoError(t, err)
	require.NotNil(t, spec)

	// Validate the spec
	result := ValidateSpec(spec)
	assert.True(t, result.Valid, "Validation errors: %v", result.Errors)

	// Check security schemes
	assert.Contains(t, spec.Components.SecuritySchemes, "bearerAuth")

	// Check schemas
	assert.Contains(t, spec.Components.Schemas, "User")
	assert.Contains(t, spec.Components.Schemas, "CreateUserRequest")

	// Check paths
	assert.Contains(t, spec.Paths, "/users")
	assert.Contains(t, spec.Paths, "/users/{id}")

	// Check operations
	usersPath := spec.Paths["/users"]
	assert.NotNil(t, usersPath.Get)
	assert.NotNil(t, usersPath.Post)
	assert.NotNil(t, usersPath.Post.RequestBody)

	userByIdPath := spec.Paths["/users/{id}"]
	assert.NotNil(t, userByIdPath.Get)
	assert.Len(t, userByIdPath.Get.Parameters, 1)
	assert.Equal(t, "id", userByIdPath.Get.Parameters[0].Name)
	assert.Equal(t, "path", userByIdPath.Get.Parameters[0].In)
}

package openapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// getTestDataPath returns the path to the testdata directory.
func getTestDataPath(t *testing.T) string {
	// Find testdata relative to this test file
	_, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Join("testdata")
}

// =============================================================================
// Integration Tests for Full Pipeline
// =============================================================================

func TestIntegrationGenerateFromCodeAIFile(t *testing.T) {
	testdataPath := getTestDataPath(t)
	sampleFile := filepath.Join(testdataPath, "sample.cai")

	// Skip if file doesn't exist
	if _, err := os.Stat(sampleFile); os.IsNotExist(err) {
		t.Skip("testdata/sample.cai not found")
	}

	config := &Config{
		Title:       "Sample API",
		Description: "Generated from sample.cai",
		Version:     "1.0.0",
	}

	gen := NewGenerator(config)
	spec, err := gen.GenerateFromFile(sampleFile)
	require.NoError(t, err)
	require.NotNil(t, spec)

	// Verify basic structure
	assert.Equal(t, "3.0.0", spec.OpenAPI)
	assert.Equal(t, "Sample API", spec.Info.Title)
	assert.Equal(t, "1.0.0", spec.Info.Version)

	// Should have generated paths from handler functions
	assert.NotEmpty(t, spec.Paths)
}

func TestIntegrationLoadConfig(t *testing.T) {
	testdataPath := getTestDataPath(t)
	configFile := filepath.Join(testdataPath, "openapi-config.yaml")

	// Skip if file doesn't exist
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Skip("testdata/openapi-config.yaml not found")
	}

	config, err := LoadConfig(configFile)
	require.NoError(t, err)

	assert.Equal(t, "Sample API", config.Title)
	assert.Equal(t, "2.0.0", config.Version)
	assert.Equal(t, "API Support", config.ContactName)
	assert.Equal(t, "MIT", config.LicenseName)
	assert.Len(t, config.Servers, 2)
}

func TestIntegrationValidateValidSpec(t *testing.T) {
	testdataPath := getTestDataPath(t)
	specFile := filepath.Join(testdataPath, "valid-spec.yaml")

	// Skip if file doesn't exist
	if _, err := os.Stat(specFile); os.IsNotExist(err) {
		t.Skip("testdata/valid-spec.yaml not found")
	}

	data, err := os.ReadFile(specFile)
	require.NoError(t, err)

	var spec OpenAPI
	err = yaml.Unmarshal(data, &spec)
	require.NoError(t, err)

	result := ValidateSpec(&spec)

	assert.True(t, result.Valid, "Expected valid spec, got errors: %v", result.Errors)
	assert.Empty(t, result.Errors)
}

func TestIntegrationValidateInvalidSpec(t *testing.T) {
	testdataPath := getTestDataPath(t)
	specFile := filepath.Join(testdataPath, "invalid-spec.yaml")

	// Skip if file doesn't exist
	if _, err := os.Stat(specFile); os.IsNotExist(err) {
		t.Skip("testdata/invalid-spec.yaml not found")
	}

	data, err := os.ReadFile(specFile)
	require.NoError(t, err)

	var spec OpenAPI
	err = yaml.Unmarshal(data, &spec)
	require.NoError(t, err)

	result := ValidateSpec(&spec)

	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestIntegrationGenerateJSONOutput(t *testing.T) {
	config := &Config{
		Title:        "JSON Test API",
		Version:      "1.0.0",
		OutputFormat: "json",
	}

	gen := NewGenerator(config)

	gen.AddRoute("GET", "/items", "List items", "Get all items")
	gen.AddRoute("POST", "/items", "Create item", "Create a new item")
	gen.AddBearerAuth("bearerAuth")

	spec := gen.GetSpec()

	// Convert to JSON
	jsonStr, err := ToJSON(spec)
	require.NoError(t, err)

	// Verify it's valid JSON
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "3.0.0", parsed["openapi"])
}

func TestIntegrationGenerateYAMLOutput(t *testing.T) {
	config := &Config{
		Title:        "YAML Test API",
		Version:      "1.0.0",
		OutputFormat: "yaml",
	}

	gen := NewGenerator(config)

	gen.AddRoute("GET", "/items", "List items", "Get all items")
	gen.AddRoute("POST", "/items", "Create item", "Create a new item")
	gen.AddBearerAuth("bearerAuth")

	spec := gen.GetSpec()

	// Convert to YAML
	yamlStr, err := ToYAML(spec)
	require.NoError(t, err)

	// Verify it's valid YAML
	var parsed map[string]interface{}
	err = yaml.Unmarshal([]byte(yamlStr), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "3.0.0", parsed["openapi"])
}

func TestIntegrationCompleteAPISpec(t *testing.T) {
	// Create a complete API spec programmatically
	config := &Config{
		Title:        "Complete Test API",
		Description:  "A comprehensive test API",
		Version:      "1.0.0",
		ContactName:  "Test",
		ContactEmail: "test@example.com",
		LicenseName:  "MIT",
		Servers: []ServerConfig{
			{URL: "https://api.example.com", Description: "Production"},
		},
	}

	gen := NewGenerator(config)

	// Add schemas
	gen.AddSchema("User", &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"id":    {Type: "string", Format: "uuid"},
			"name":  {Type: "string"},
			"email": {Type: "string", Format: "email"},
		},
		Required: []string{"id", "name", "email"},
	})

	gen.AddSchema("Error", &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"code":    {Type: "integer"},
			"message": {Type: "string"},
		},
		Required: []string{"code", "message"},
	})

	// Add routes with full operations
	mapper := gen.mapper

	listUsersOp := &Operation{
		Summary:     "List users",
		Description: "Get a list of all users",
		OperationID: "listUsers",
		Tags:        []string{"users"},
		Parameters: []Parameter{
			{Name: "limit", In: "query", Schema: &Schema{Type: "integer"}},
			{Name: "offset", In: "query", Schema: &Schema{Type: "integer"}},
		},
		Responses: map[string]Response{
			"200": {
				Description: "Successful response",
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{
							Type:  "array",
							Items: &Schema{Ref: "#/components/schemas/User"},
						},
					},
				},
			},
		},
	}
	mapper.AddRoute("GET", "/users", listUsersOp)

	getUserOp := &Operation{
		Summary:     "Get user by ID",
		OperationID: "getUserById",
		Tags:        []string{"users"},
		Parameters: []Parameter{
			{Name: "id", In: "path", Required: true, Schema: &Schema{Type: "string", Format: "uuid"}},
		},
		Responses: map[string]Response{
			"200": {
				Description: "Successful response",
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Ref: "#/components/schemas/User"},
					},
				},
			},
			"404": {
				Description: "User not found",
			},
		},
	}
	mapper.AddRoute("GET", "/users/{id}", getUserOp)

	// Add security
	gen.AddBearerAuth("bearerAuth")

	// Add tags
	gen.AddTag("users", "User management operations")

	// Get spec
	spec := gen.GetSpec()

	// Validate
	result := ValidateSpec(spec)

	assert.True(t, result.Valid, "Spec validation failed: %v", result.Errors)

	// Verify all components
	assert.Equal(t, "3.0.0", spec.OpenAPI)
	assert.Equal(t, "Complete Test API", spec.Info.Title)
	assert.NotNil(t, spec.Info.Contact)
	assert.NotNil(t, spec.Info.License)
	assert.Len(t, spec.Servers, 1)
	assert.Len(t, spec.Paths, 2)
	assert.Contains(t, spec.Components.Schemas, "User")
	assert.Contains(t, spec.Components.Schemas, "Error")
	assert.Contains(t, spec.Components.SecuritySchemes, "bearerAuth")
}

func TestIntegrationSchemaFromGoTypes(t *testing.T) {
	type Address struct {
		Street  string `json:"street" validate:"required"`
		City    string `json:"city" validate:"required"`
		Country string `json:"country" validate:"required"`
		ZipCode string `json:"zip_code"`
	}

	type User struct {
		ID        string   `json:"id" validate:"required,uuid"`
		Name      string   `json:"name" validate:"required,min=1,max=255"`
		Email     string   `json:"email" validate:"required,email"`
		Age       int      `json:"age" validate:"min=0,max=150"`
		Active    bool     `json:"active"`
		Roles     []string `json:"roles"`
		Address   *Address `json:"address,omitempty"`
		CreatedAt string   `json:"created_at"`
	}

	gen := NewGenerator(&Config{
		Title:   "Schema Test API",
		Version: "1.0.0",
	})

	// Generate schemas from types
	spec, err := gen.GenerateFromTypes(User{}, Address{})
	require.NoError(t, err)

	// Verify User schema
	userSchema, exists := spec.Components.Schemas["User"]
	require.True(t, exists)
	assert.Equal(t, "object", userSchema.Type)
	assert.Contains(t, userSchema.Properties, "id")
	assert.Contains(t, userSchema.Properties, "name")
	assert.Contains(t, userSchema.Properties, "email")
	assert.Contains(t, userSchema.Properties, "age")
	assert.Contains(t, userSchema.Properties, "roles")
	assert.Contains(t, userSchema.Properties, "address")

	// Verify required fields
	assert.Contains(t, userSchema.Required, "id")
	assert.Contains(t, userSchema.Required, "name")
	assert.Contains(t, userSchema.Required, "email")

	// Verify constraints
	nameSchema := userSchema.Properties["name"]
	require.NotNil(t, nameSchema.MinLength)
	require.NotNil(t, nameSchema.MaxLength)
	assert.Equal(t, 1, *nameSchema.MinLength)
	assert.Equal(t, 255, *nameSchema.MaxLength)

	emailSchema := userSchema.Properties["email"]
	assert.Equal(t, "email", emailSchema.Format)

	// Verify roles is array
	rolesSchema := userSchema.Properties["roles"]
	assert.Equal(t, "array", rolesSchema.Type)
	require.NotNil(t, rolesSchema.Items)
	assert.Equal(t, "string", rolesSchema.Items.Type)

	// Verify Address schema exists
	addressSchema, exists := spec.Components.Schemas["Address"]
	require.True(t, exists)
	assert.Equal(t, "object", addressSchema.Type)
}

func TestIntegrationMergeSpecs(t *testing.T) {
	// Create first spec (users)
	gen1 := NewGenerator(&Config{Title: "Users API", Version: "1.0.0"})
	gen1.AddRoute("GET", "/users", "List users", "")
	gen1.AddRoute("POST", "/users", "Create user", "")
	gen1.AddSchema("User", &Schema{Type: "object"})
	spec1 := gen1.GetSpec()

	// Create second spec (products)
	gen2 := NewGenerator(&Config{Title: "Products API", Version: "1.0.0"})
	gen2.AddRoute("GET", "/products", "List products", "")
	gen2.AddRoute("POST", "/products", "Create product", "")
	gen2.AddSchema("Product", &Schema{Type: "object"})
	spec2 := gen2.GetSpec()

	// Merge
	merged := Merge(spec1, spec2)

	// Verify merged has all paths
	assert.Contains(t, merged.Paths, "/users")
	assert.Contains(t, merged.Paths, "/products")

	// Verify merged has all schemas
	assert.Contains(t, merged.Components.Schemas, "User")
	assert.Contains(t, merged.Components.Schemas, "Product")
}

func TestIntegrationRoundTrip(t *testing.T) {
	// Create spec
	original := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       "Round Trip Test",
			Description: "Testing JSON/YAML round trip",
			Version:     "1.0.0",
		},
		Paths: map[string]PathItem{
			"/test": {
				Get: &Operation{
					Summary: "Test endpoint",
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
		},
		Components: Components{
			Schemas: map[string]*Schema{
				"Test": {
					Type: "object",
					Properties: map[string]*Schema{
						"field": {Type: "string"},
					},
				},
			},
			SecuritySchemes: map[string]*SecurityScheme{},
		},
	}

	// Convert to JSON and back
	jsonStr, err := ToJSON(original)
	require.NoError(t, err)

	var fromJSON OpenAPI
	err = json.Unmarshal([]byte(jsonStr), &fromJSON)
	require.NoError(t, err)

	assert.Equal(t, original.OpenAPI, fromJSON.OpenAPI)
	assert.Equal(t, original.Info.Title, fromJSON.Info.Title)
	assert.Contains(t, fromJSON.Paths, "/test")

	// Convert to YAML and back
	yamlStr, err := ToYAML(original)
	require.NoError(t, err)

	var fromYAML OpenAPI
	err = yaml.Unmarshal([]byte(yamlStr), &fromYAML)
	require.NoError(t, err)

	assert.Equal(t, original.OpenAPI, fromYAML.OpenAPI)
	assert.Equal(t, original.Info.Title, fromYAML.Info.Title)
	assert.Contains(t, fromYAML.Paths, "/test")
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkSchemaGeneration(b *testing.B) {
	type ComplexStruct struct {
		ID        string            `json:"id"`
		Name      string            `json:"name"`
		Email     string            `json:"email"`
		Age       int               `json:"age"`
		Active    bool              `json:"active"`
		Tags      []string          `json:"tags"`
		Metadata  map[string]string `json:"metadata"`
		CreatedAt string            `json:"created_at"`
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen := NewSchemaGenerator()
		gen.GenerateSchemaFromValue(ComplexStruct{})
	}
}

func BenchmarkValidation(b *testing.B) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]PathItem{
			"/users": {
				Get: &Operation{
					Summary: "List users",
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
			"/users/{id}": {
				Get: &Operation{
					Parameters: []Parameter{
						{Name: "id", In: "path", Required: true},
					},
					Responses: map[string]Response{
						"200": {Description: "Success"},
					},
				},
			},
		},
		Components: Components{
			Schemas: map[string]*Schema{
				"User": {Type: "object"},
			},
			SecuritySchemes: map[string]*SecurityScheme{},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateSpec(spec)
	}
}

func BenchmarkJSONSerialization(b *testing.B) {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]PathItem{
			"/users": {
				Get: &Operation{
					Summary:   "List users",
					Responses: map[string]Response{"200": {Description: "Success"}},
				},
				Post: &Operation{
					Summary:   "Create user",
					Responses: map[string]Response{"201": {Description: "Created"}},
				},
			},
		},
		Components: Components{
			Schemas: map[string]*Schema{
				"User": {
					Type: "object",
					Properties: map[string]*Schema{
						"id":   {Type: "string"},
						"name": {Type: "string"},
					},
				},
			},
			SecuritySchemes: map[string]*SecurityScheme{},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToJSON(spec)
	}
}

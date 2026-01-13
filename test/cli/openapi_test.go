//go:build integration

// Package cli provides CLI integration tests for CodeAI.
package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/bargom/codeai/cmd/codeai/cmd"
	clitest "github.com/bargom/codeai/cmd/codeai/testing"
	"github.com/bargom/codeai/internal/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestCLIOpenAPIGenerate tests the openapi generate command.
func TestCLIOpenAPIGenerate(t *testing.T) {
	t.Run("generate from valid DSL file", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
model User {
    id uuid primary
    email email unique
    name string
}

endpoint GET "/users" {
    response User status 200
}

endpoint POST "/users" {
    request CreateUserRequest from body
    response User status 201
}
`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "openapi", "generate", tmpfile)

		require.NoError(t, err)

		// Check output is valid YAML (default format)
		var spec openapi.OpenAPI
		err = yaml.Unmarshal([]byte(output), &spec)
		require.NoError(t, err, "Output should be valid YAML")

		assert.Equal(t, "3.0.0", spec.OpenAPI)
		assert.Contains(t, spec.Paths, "/users")
	})

	t.Run("generate JSON format", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
model User {
    id uuid primary
    email email unique
}

endpoint GET "/users/:id" {
    request UserID from path
    response User status 200
}
`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "openapi", "generate", tmpfile, "--format", "json")

		require.NoError(t, err)

		// Check output is valid JSON
		var spec openapi.OpenAPI
		err = json.Unmarshal([]byte(output), &spec)
		require.NoError(t, err, "Output should be valid JSON")

		assert.Equal(t, "3.0.0", spec.OpenAPI)
		assert.Contains(t, spec.Paths, "/users/{id}")
	})

	t.Run("generate with custom title", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
endpoint GET "/health" {
    response HealthStatus status 200
}
`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "openapi", "generate", tmpfile, "--title", "My Custom API")

		require.NoError(t, err)

		var spec openapi.OpenAPI
		err = yaml.Unmarshal([]byte(output), &spec)
		require.NoError(t, err)

		assert.Equal(t, "My Custom API", spec.Info.Title)
	})

	t.Run("generate with custom version", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
endpoint GET "/health" {
    response HealthStatus status 200
}
`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "openapi", "generate", tmpfile, "--version", "2.5.0")

		require.NoError(t, err)

		var spec openapi.OpenAPI
		err = yaml.Unmarshal([]byte(output), &spec)
		require.NoError(t, err)

		assert.Equal(t, "2.5.0", spec.Info.Version)
	})

	t.Run("generate nonexistent file fails", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "openapi", "generate", "/nonexistent/file.cai")

		assert.Error(t, err)
	})

	t.Run("generate with endpoints and models", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
model Product {
    id uuid primary
    name string required
    price float
    description string nullable
}

model CreateProductRequest {
    name string required
    price float required
}

endpoint GET "/products" {
    response Product status 200
}

endpoint GET "/products/:id" {
    request ProductID from path
    response Product status 200
}

endpoint POST "/products" {
    request CreateProductRequest from body
    response Product status 201
}

endpoint PUT "/products/:id" {
    request Product from body
    response Product status 200
}

endpoint DELETE "/products/:id" {
    response NoContent status 204
}
`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "openapi", "generate", tmpfile, "--format", "json")

		require.NoError(t, err)

		var spec openapi.OpenAPI
		err = json.Unmarshal([]byte(output), &spec)
		require.NoError(t, err)

		// Check all paths
		assert.Contains(t, spec.Paths, "/products")
		assert.Contains(t, spec.Paths, "/products/{id}")

		// Check methods
		productsPath := spec.Paths["/products"]
		assert.NotNil(t, productsPath.Get)
		assert.NotNil(t, productsPath.Post)

		productByIdPath := spec.Paths["/products/{id}"]
		assert.NotNil(t, productByIdPath.Get)
		assert.NotNil(t, productByIdPath.Put)
		assert.NotNil(t, productByIdPath.Delete)

		// Check schemas
		assert.Contains(t, spec.Components.Schemas, "Product")
		assert.Contains(t, spec.Components.Schemas, "CreateProductRequest")
	})

	t.Run("generate with auth", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
auth jwt_auth jwt {
    jwks "https://auth.example.com/.well-known/jwks.json"
    issuer "https://auth.example.com"
    audience "api.example.com"
}

endpoint GET "/secure" @auth(jwt_auth) {
    response SecureData status 200
}
`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "openapi", "generate", tmpfile, "--format", "json")

		require.NoError(t, err)

		var spec openapi.OpenAPI
		err = json.Unmarshal([]byte(output), &spec)
		require.NoError(t, err)

		// Check security scheme
		assert.Contains(t, spec.Components.SecuritySchemes, "jwt_auth")
		scheme := spec.Components.SecuritySchemes["jwt_auth"]
		assert.Equal(t, "http", scheme.Type)
		assert.Equal(t, "bearer", scheme.Scheme)
	})
}

// TestCLIOpenAPIValidate tests the openapi validate command.
func TestCLIOpenAPIValidate(t *testing.T) {
	t.Run("validate valid spec", func(t *testing.T) {
		validSpec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users:
    get:
      summary: List users
      responses:
        "200":
          description: Success
`
		tmpfile := clitest.CreateTempFileWithExt(t, ".yaml", validSpec)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "openapi", "validate", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, strings.ToLower(output), "valid")
	})

	t.Run("validate invalid spec missing info", func(t *testing.T) {
		invalidSpec := `openapi: "3.0.0"
paths:
  /users:
    get:
      summary: List users
      responses:
        "200":
          description: Success
`
		tmpfile := clitest.CreateTempFileWithExt(t, ".yaml", invalidSpec)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "openapi", "validate", tmpfile)

		assert.Error(t, err)
	})

	t.Run("validate invalid spec missing responses", func(t *testing.T) {
		invalidSpec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users:
    get:
      summary: List users
`
		tmpfile := clitest.CreateTempFileWithExt(t, ".yaml", invalidSpec)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "openapi", "validate", tmpfile)

		assert.Error(t, err)
	})

	t.Run("validate JSON spec", func(t *testing.T) {
		validSpec := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Test API",
    "version": "1.0.0"
  },
  "paths": {
    "/users": {
      "get": {
        "summary": "List users",
        "responses": {
          "200": {
            "description": "Success"
          }
        }
      }
    }
  }
}`
		tmpfile := clitest.CreateTempFileWithExt(t, ".json", validSpec)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "openapi", "validate", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, strings.ToLower(output), "valid")
	})

	t.Run("validate nonexistent file fails", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "openapi", "validate", "/nonexistent/spec.yaml")

		assert.Error(t, err)
	})

	t.Run("validate strict mode", func(t *testing.T) {
		// Spec with missing operationId (should warn in strict mode)
		spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users:
    get:
      summary: List users
      responses:
        "200":
          description: Success
`
		tmpfile := clitest.CreateTempFileWithExt(t, ".yaml", spec)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "openapi", "validate", tmpfile, "--strict")

		require.NoError(t, err)
		// In strict mode, we should still validate but may show warnings
		assert.Contains(t, strings.ToLower(output), "valid")
	})
}

// TestCLIOpenAPIGenerateAndValidateRoundTrip tests generating then validating.
func TestCLIOpenAPIGenerateAndValidateRoundTrip(t *testing.T) {
	t.Run("generated spec passes validation", func(t *testing.T) {
		dslFile := clitest.CreateTempFile(t, `
model User {
    id uuid primary
    email email unique
    name string
}

endpoint GET "/users" {
    response User status 200
}

endpoint GET "/users/:id" {
    request UserID from path
    response User status 200
}

endpoint POST "/users" {
    request CreateUserRequest from body
    response User status 201
}
`)
		defer os.Remove(dslFile)

		// Generate the spec
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "openapi", "generate", dslFile, "--format", "yaml")
		require.NoError(t, err)

		// Write the generated spec to a file
		specFile := clitest.CreateTempFileWithExt(t, ".yaml", output)
		defer os.Remove(specFile)

		// Validate the generated spec
		rootCmd = cmd.NewRootCmd()
		validationOutput, err := clitest.ExecuteCommand(rootCmd, "openapi", "validate", specFile)

		require.NoError(t, err)
		assert.Contains(t, strings.ToLower(validationOutput), "valid")
	})
}

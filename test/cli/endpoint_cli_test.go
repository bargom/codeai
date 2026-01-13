//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLIParseEndpoints tests the CLI parse command with endpoint files.
func TestCLIParseEndpoints(t *testing.T) {
	// Build the binary if it doesn't exist
	binaryPath := ensureBinaryExists(t)

	// Find the with_endpoints.cai file
	examplesFile := findExamplesFile(t, "with_endpoints.cai")

	t.Run("parse endpoints example file", func(t *testing.T) {
		// Test: ./bin/codeai parse examples/with_endpoints.cai
		cmd := exec.Command(binaryPath, "parse", examplesFile)
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "Parse command should succeed: %s", string(output))

		// Should contain parsed output (JSON format by default)
		outputStr := string(output)
		assert.Contains(t, outputStr, "endpoint", "Output should mention endpoints")

		t.Logf("Parse output: %s", outputStr)
	})

	t.Run("parse endpoints with verbose output", func(t *testing.T) {
		// Test: ./bin/codeai parse --verbose examples/with_endpoints.cai
		cmd := exec.Command(binaryPath, "parse", "--verbose", examplesFile)
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "Parse with verbose should succeed: %s", string(output))

		// Should contain verbose information
		outputStr := string(output)
		assert.Contains(t, outputStr, "Parsing file:", "Should show verbose parsing info")

		t.Logf("Verbose parse output: %s", outputStr)
	})

	t.Run("parse endpoints with JSON output", func(t *testing.T) {
		// Test: ./bin/codeai parse --output json examples/with_endpoints.cai
		cmd := exec.Command(binaryPath, "parse", "--output", "json", examplesFile)
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "Parse with JSON output should succeed: %s", string(output))

		// Should contain valid JSON
		outputStr := string(output)
		assert.Contains(t, outputStr, "{", "Should contain JSON")
		assert.Contains(t, outputStr, "}", "Should contain valid JSON")

		t.Logf("JSON parse output length: %d", len(outputStr))
	})
}

// TestCLIValidateEndpoints tests the CLI validate command with endpoint files.
func TestCLIValidateEndpoints(t *testing.T) {
	// Build the binary if it doesn't exist
	binaryPath := ensureBinaryExists(t)

	// Find the with_endpoints.cai file
	examplesFile := findExamplesFile(t, "with_endpoints.cai")

	t.Run("validate endpoints example file", func(t *testing.T) {
		// Test: ./bin/codeai validate examples/with_endpoints.cai
		cmd := exec.Command(binaryPath, "validate", examplesFile)
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "Validate command should succeed: %s", string(output))

		// Should report validation success
		outputStr := string(output)
		// Look for success indicators or lack of error messages
		assert.NotContains(t, outputStr, "error", "Should not contain validation errors")

		t.Logf("Validate output: %s", outputStr)
	})

	t.Run("validate endpoints with verbose output", func(t *testing.T) {
		// Test: ./bin/codeai validate --verbose examples/with_endpoints.cai
		cmd := exec.Command(binaryPath, "validate", "--verbose", examplesFile)
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "Validate with verbose should succeed: %s", string(output))

		// Should contain verbose validation information
		outputStr := string(output)
		assert.Contains(t, outputStr, "Validating", "Should show verbose validation info")

		t.Logf("Verbose validate output: %s", outputStr)
	})
}

// TestCLIParseInvalidEndpoint tests CLI behavior with invalid endpoint files.
func TestCLIParseInvalidEndpoint(t *testing.T) {
	// Build the binary if it doesn't exist
	binaryPath := ensureBinaryExists(t)

	// Create a temporary invalid endpoint file
	invalidContent := `
endpoint INVALID_METHOD "/test" {
  response Test status 200
}
`
	tmpFile := createTempFile(t, "invalid_endpoint.cai", invalidContent)
	defer os.Remove(tmpFile)

	t.Run("parse invalid endpoint should fail", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "parse", tmpFile)
		output, err := cmd.CombinedOutput()

		// Should fail with error
		assert.Error(t, err, "Parse should fail for invalid endpoint")

		outputStr := string(output)
		assert.Contains(t, outputStr, "parse error", "Should report parse error")

		t.Logf("Invalid parse output: %s", outputStr)
	})
}

// TestCLIValidateInvalidEndpoint tests CLI validation with type errors.
func TestCLIValidateInvalidEndpoint(t *testing.T) {
	// Build the binary if it doesn't exist
	binaryPath := ensureBinaryExists(t)

	// Create a temporary file with undefined types
	invalidContent := `
endpoint GET "/test" {
  request UndefinedType from body
  response AnotherUndefinedType status 200
}
`
	tmpFile := createTempFile(t, "undefined_types.cai", invalidContent)
	defer os.Remove(tmpFile)

	t.Run("validate with undefined types should fail", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "validate", tmpFile)
		output, err := cmd.CombinedOutput()

		// Should fail with validation error
		assert.Error(t, err, "Validate should fail for undefined types")

		outputStr := string(output)
		assert.Contains(t, outputStr, "undefined", "Should report undefined type error")

		t.Logf("Invalid validate output: %s", outputStr)
	})
}

// TestCLIEndpointCounting tests that CLI properly counts and reports endpoint information.
func TestCLIEndpointCounting(t *testing.T) {
	// Build the binary if it doesn't exist
	binaryPath := ensureBinaryExists(t)

	// Create a file with known number of endpoints
	endpointContent := `
model User { id: string }
model CreateUser { name: string }

middleware auth { type: jwt }

endpoint GET "/users" {
  response User status 200
}

endpoint POST "/users" {
  request CreateUser from body
  response User status 201
  middleware auth
}

endpoint GET "/users/:id" {
  request User from path
  response User status 200
  middleware auth
}
`
	tmpFile := createTempFile(t, "multiple_endpoints.cai", endpointContent)
	defer os.Remove(tmpFile)

	t.Run("parse should handle multiple endpoints", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "parse", tmpFile)
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "Parse should succeed for multiple endpoints: %s", string(output))

		outputStr := string(output)
		// Should contain information about all endpoints
		assert.Contains(t, outputStr, "GET", "Should contain GET endpoints")
		assert.Contains(t, outputStr, "POST", "Should contain POST endpoints")

		t.Logf("Multiple endpoints parse output length: %d", len(outputStr))
	})
}

// Helper functions

// ensureBinaryExists builds the binary if it doesn't exist and returns its path.
func ensureBinaryExists(t *testing.T) string {
	t.Helper()

	// Find project root
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Navigate up to find project root (where go.mod is)
	projectRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatal("Could not find project root (go.mod)")
		}
		projectRoot = parent
	}

	binaryPath := filepath.Join(projectRoot, "bin", "codeai")

	// Check if binary exists
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath
	}

	// Build the binary
	t.Logf("Building binary at %s", binaryPath)
	buildCmd := exec.Command("make", "build")
	buildCmd.Dir = projectRoot
	buildOutput, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build binary: %s", string(buildOutput))

	// Verify binary was created
	require.FileExists(t, binaryPath, "Binary should exist after build")

	return binaryPath
}

// findExamplesFile finds an example file and returns its full path.
func findExamplesFile(t *testing.T, filename string) string {
	t.Helper()

	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Navigate up to find examples directory
	for {
		examplesPath := filepath.Join(cwd, "examples", filename)
		if _, err := os.Stat(examplesPath); err == nil {
			return examplesPath
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Fatalf("Could not find examples/%s in parent directories", filename)
		}
		cwd = parent
	}
}

// createTempFile creates a temporary file with given content.
func createTempFile(t *testing.T, name, content string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", name)
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}
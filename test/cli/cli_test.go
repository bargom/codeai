//go:build integration

// Package cli provides CLI integration tests for CodeAI.
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bargom/codeai/cmd/codeai/cmd"
	clitest "github.com/bargom/codeai/cmd/codeai/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLIParseWorkflow tests the parse command workflow.
func TestCLIParseWorkflow(t *testing.T) {
	t.Run("parse valid DSL file", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
var greeting = "Hello"
var name = "World"
`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "greeting")
		assert.Contains(t, output, "name")
	})

	t.Run("parse file with functions", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
function greet(name) {
    var message = name
}
`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "greet")
	})

	t.Run("parse file with exec block", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
exec {
    echo "hello"
}
`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "Command")
	})

	t.Run("parse nonexistent file fails", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "parse", "/nonexistent/file.cai")

		assert.Error(t, err)
	})

	t.Run("parse invalid syntax fails", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = `)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		assert.Error(t, err)
	})
}

// TestCLIValidateWorkflow tests the validate command workflow.
func TestCLIValidateWorkflow(t *testing.T) {
	t.Run("validate valid DSL file", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
var greeting = "Hello"
var name = "World"
var message = greeting
`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
	})

	t.Run("validate with undefined variable fails", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = undefinedVar`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "undefined")
	})

	t.Run("validate with duplicate declaration fails", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
var x = 1
var x = 2
`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})
}

// TestCLIDeployWorkflow tests the deploy command workflow.
func TestCLIDeployWorkflow(t *testing.T) {
	t.Run("deploy create dry-run", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var config = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "create", "--dry-run", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
		assert.Contains(t, output, "Dry run")
	})

	t.Run("deploy create dry-run with name", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var config = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "create", "--name", "my-deployment", "--dry-run", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "my-deployment")
	})
}

// TestCLIConfigWorkflow tests the config command workflow.
func TestCLIConfigWorkflow(t *testing.T) {
	t.Run("config create dry-run", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var config = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "config", "create", "--dry-run", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
	})

	t.Run("config create dry-run with name", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var config = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "config", "create", "--name", "my-config", "--dry-run", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "my-config")
	})
}

// TestCLIVersionCommand tests the version command.
func TestCLIVersionCommand(t *testing.T) {
	t.Run("version command works", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "version")

		require.NoError(t, err)
		assert.NotEmpty(t, output)
	})
}

// TestCLIHelpOutput tests help output for all commands.
func TestCLIHelpOutput(t *testing.T) {
	commands := [][]string{
		{"--help"},
		{"parse", "--help"},
		{"validate", "--help"},
		{"deploy", "--help"},
		{"deploy", "create", "--help"},
		{"deploy", "list", "--help"},
		{"config", "--help"},
		{"config", "create", "--help"},
		{"server", "--help"},
	}

	for _, cmdArgs := range commands {
		name := strings.Join(cmdArgs, " ")
		t.Run("help for "+name, func(t *testing.T) {
			rootCmd := cmd.NewRootCmd()
			output, err := clitest.ExecuteCommand(rootCmd, cmdArgs...)

			// Help should not error
			assert.NoError(t, err)
			// Help output should contain Usage
			assert.Contains(t, output, "Usage")
		})
	}
}

// TestCLIComplexDSLParsing tests parsing complex DSL files.
func TestCLIComplexDSLParsing(t *testing.T) {
	complexDSL := `
// Configuration
var apiKey = "sk-12345"
var maxRetries = 3
var debug = true

// Data structures
var users = ["alice", "bob", "charlie"]
var config = ["host", "port"]

// Functions
function greet(name) {
    var message = "Hello"
}

function processUser(user) {
    var status = "active"
}

// Control flow
if debug {
    var logLevel = "verbose"
} else {
    var logLevel = "info"
}

// Iteration
for user in users {
    var processed = user
}

// Exec blocks
exec {
    echo "Starting processing"
}
`

	t.Run("parse complex DSL", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, complexDSL)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "apiKey")
		assert.Contains(t, output, "greet")
		assert.Contains(t, output, "Command")
	})

	t.Run("validate complex DSL", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, complexDSL)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
	})
}

// TestCLIOutputFormats tests different output formats.
func TestCLIOutputFormats(t *testing.T) {
	t.Run("parse with JSON output", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = 1`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", "--output", "json", tmpfile)

		require.NoError(t, err)
		// JSON output should contain valid JSON markers
		assert.Contains(t, output, "{")
	})

	t.Run("validate with JSON output", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = 1`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", "--output", "json", tmpfile)

		require.NoError(t, err)
		// Should produce some output
		assert.NotEmpty(t, output)
	})
}

// TestCLIFromFixtures tests CLI against fixture files.
func TestCLIFromFixtures(t *testing.T) {
	fixturesDir := "../../test/fixtures"

	t.Run("parse valid fixtures", func(t *testing.T) {
		files, err := filepath.Glob(filepath.Join(fixturesDir, "*.cai"))
		if err != nil || len(files) == 0 {
			t.Skip("No fixture files found")
		}

		for _, file := range files {
			basename := filepath.Base(file)
			// Skip invalid fixtures for parse tests - they should still parse
			t.Run("parse_"+basename, func(t *testing.T) {
				rootCmd := cmd.NewRootCmd()
				_, err := clitest.ExecuteCommand(rootCmd, "parse", file)
				// All fixtures should parse successfully (syntax is valid)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("validate fixtures", func(t *testing.T) {
		files, err := filepath.Glob(filepath.Join(fixturesDir, "*.cai"))
		if err != nil || len(files) == 0 {
			t.Skip("No fixture files found")
		}

		for _, file := range files {
			basename := filepath.Base(file)
			t.Run("validate_"+basename, func(t *testing.T) {
				rootCmd := cmd.NewRootCmd()
				_, err := clitest.ExecuteCommand(rootCmd, "validate", file)

				// Invalid fixtures (prefixed with "invalid_") should fail validation
				if strings.HasPrefix(basename, "invalid_") {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

// TestCLIErrorMessages tests that error messages are helpful.
func TestCLIErrorMessages(t *testing.T) {
	t.Run("undefined variable error is clear", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = undefinedVar`)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "undefined")
	})

	t.Run("duplicate declaration error is clear", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, "var x = 1\nvar x = 2")
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})

	t.Run("parse error is clear", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = `)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parse error")
	})
}

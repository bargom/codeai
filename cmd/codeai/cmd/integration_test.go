package cmd

import (
	"os"
	"testing"

	clitest "github.com/bargom/codeai/cmd/codeai/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests that test multiple commands working together

func TestParseAndValidateWorkflow(t *testing.T) {
	t.Run("parse then validate same file", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
var greeting = "Hello"
var name = "World"
var items = [greeting, name]
for item in items {
	var current = item
}
`)
		defer os.Remove(tmpfile)

		// First parse
		rootCmd := NewRootCmd()
		parseOutput, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)
		require.NoError(t, err)
		assert.Contains(t, parseOutput, "greeting")

		// Then validate (using new rootCmd to avoid state issues)
		rootCmd2 := NewRootCmd()
		validateOutput, err := clitest.ExecuteCommand(rootCmd2, "validate", tmpfile)
		require.NoError(t, err)
		assert.Contains(t, validateOutput, "valid")
	})
}

func TestDeployCreateDryRun(t *testing.T) {
	t.Run("dry-run validates without creating", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var config = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "create", "--dry-run", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
		assert.Contains(t, output, "Dry run")
	})

	t.Run("dry-run shows config name", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var config = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "create", "--name", "test-deploy", "--dry-run", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "test-deploy")
	})
}

func TestConfigCreateDryRun(t *testing.T) {
	t.Run("dry-run validates without creating", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var config = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "config", "create", "--dry-run", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
	})

	t.Run("dry-run shows config name", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var config = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "config", "create", "--name", "my-config", "--dry-run", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "my-config")
	})
}

func TestDeploySubcommandFlags(t *testing.T) {
	t.Run("deploy create has all flags", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "create", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "name")
		assert.Contains(t, output, "dry-run")
		assert.Contains(t, output, "api-url")
	})

	t.Run("deploy list has all flags", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "list", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "limit")
		assert.Contains(t, output, "api-url")
	})

	t.Run("deploy get has all flags", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "get", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "api-url")
	})

	t.Run("deploy delete has force flag", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "delete", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "force")
	})

	t.Run("deploy execute has wait flag", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "execute", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "wait")
	})
}

func TestConfigSubcommandFlags(t *testing.T) {
	t.Run("config create has all flags", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "config", "create", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "name")
		assert.Contains(t, output, "dry-run")
	})

	t.Run("config list has limit flag", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "config", "list", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "limit")
	})
}

func TestServerSubcommandFlags(t *testing.T) {
	t.Run("server start has all db flags", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "server", "start", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "db-host")
		assert.Contains(t, output, "db-port")
		assert.Contains(t, output, "db-name")
		assert.Contains(t, output, "db-user")
		assert.Contains(t, output, "db-password")
		assert.Contains(t, output, "db-sslmode")
	})

	t.Run("server migrate has all db flags", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "server", "migrate", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "db-host")
		assert.Contains(t, output, "dry-run")
	})
}

func TestComplexDSLFiles(t *testing.T) {
	t.Run("parses complex DSL with all features", func(t *testing.T) {
		complexDSL := `
// Configuration
var apiKey = "sk-12345"
var maxRetries = 3
var debug = true

// Data
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
		tmpfile := clitest.CreateTempFile(t, complexDSL)
		defer os.Remove(tmpfile)

		// Parse
		rootCmd := NewRootCmd()
		parseOutput, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)
		require.NoError(t, err)
		assert.Contains(t, parseOutput, "apiKey")
		assert.Contains(t, parseOutput, "greet")
		assert.Contains(t, parseOutput, "Command")

		// Validate
		rootCmd2 := NewRootCmd()
		validateOutput, err := clitest.ExecuteCommand(rootCmd2, "validate", tmpfile)
		require.NoError(t, err)
		assert.Contains(t, validateOutput, "valid")
	})
}

func TestErrorMessages(t *testing.T) {
	t.Run("helpful error for undefined variable", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = undefinedVar`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "undefined")
	})

	t.Run("helpful error for duplicate declaration", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
var x = "first"
var x = "second"
`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})

	t.Run("helpful error for syntax error", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = `)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parse error")
	})
}

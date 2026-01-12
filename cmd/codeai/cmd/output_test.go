package cmd

import (
	"os"
	"testing"

	clitest "github.com/bargom/codeai/cmd/codeai/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputFormats(t *testing.T) {
	t.Run("parse with json output", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", "-o", "json", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "{")
		assert.Contains(t, output, "}")
	})

	t.Run("parse with plain output", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", "-o", "plain", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "x")
	})

	t.Run("parse with table output", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", "-o", "table", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "x")
	})

	t.Run("version with json output", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "version", "-o", "json")

		require.NoError(t, err)
		assert.Contains(t, output, "version")
		assert.Contains(t, output, "{")
	})

	t.Run("config validate with json output", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "config", "validate", "-o", "json", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
		assert.Contains(t, output, "{")
	})

	t.Run("deploy execute with json output", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "execute", "-o", "json", "test-id")

		require.NoError(t, err)
		assert.Contains(t, output, "status")
		assert.Contains(t, output, "{")
	})
}

func TestVerboseFlag(t *testing.T) {
	t.Run("parse with verbose flag", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", "-v", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "Parsing")
	})

	t.Run("validate with verbose flag", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", "-v", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "Validating")
		assert.Contains(t, output, "OK")
	})
}

func TestPrintVerbose(t *testing.T) {
	// Note: printVerbose uses the global verbose variable, so we test
	// through actual command execution with --verbose flag
	t.Run("verbose flag enables verbose output", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", "--verbose", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "Parsing file")
	})
}

func TestPrintErrorViaCommand(t *testing.T) {
	// Test error printing through actual command execution
	t.Run("parse shows error for missing file", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "parse", "/nonexistent/file.cai")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no such file")
	})

	t.Run("validate shows error for invalid file", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = undefinedVar`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation error")
	})
}

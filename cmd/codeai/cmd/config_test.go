package cmd

import (
	"os"
	"testing"

	clitest "github.com/bargom/codeai/cmd/codeai/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigCommand(t *testing.T) {
	t.Run("has subcommands", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "config", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "create")
		assert.Contains(t, output, "list")
		assert.Contains(t, output, "get")
		assert.Contains(t, output, "validate")
	})
}

func TestConfigCreateCommand(t *testing.T) {
	t.Run("requires file argument", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "config", "create")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "accepts 1 arg")
	})

	t.Run("validates file exists", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "config", "create", "nonexistent.cai")

		assert.Error(t, err)
	})

	t.Run("validates file syntax", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = `)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "config", "create", tmpfile)

		assert.Error(t, err)
	})

	t.Run("accepts valid file with dry-run", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "config", "create", "--dry-run", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, "valid")
	})

	t.Run("accepts name flag", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "config", "create", "--name", "my-config", "--dry-run", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, "my-config")
	})
}

func TestConfigListCommand(t *testing.T) {
	t.Run("runs without arguments", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "config", "list", "--api-url", "")

		if err != nil {
			assert.Contains(t, err.Error(), "API")
		} else {
			assert.Contains(t, output, "config")
		}
	})

	t.Run("accepts limit flag", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "config", "list", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "limit")
	})
}

func TestConfigGetCommand(t *testing.T) {
	t.Run("requires ID argument", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "config", "get")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "accepts 1 arg")
	})

	t.Run("accepts ID argument", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "config", "get", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "id")
	})

	t.Run("shows not found message", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "config", "get", "test-id")

		require.NoError(t, err)
		assert.Contains(t, output, "not found")
	})
}

func TestConfigValidateCommand(t *testing.T) {
	t.Run("requires file argument", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "config", "validate")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "accepts 1 arg")
	})

	t.Run("validates file exists", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "config", "validate", "nonexistent.cai")

		assert.Error(t, err)
	})

	t.Run("validates valid file", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "config", "validate", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, "valid")
	})

	t.Run("detects invalid file", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = undefinedVar`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "config", "validate", tmpfile)

		assert.Error(t, err)
	})
}

func TestConfigCommandHelp(t *testing.T) {
	rootCmd := NewRootCmd()
	output, err := clitest.ExecuteCommand(rootCmd, "config", "--help")

	require.NoError(t, err)
	assert.Contains(t, output, "config")
}

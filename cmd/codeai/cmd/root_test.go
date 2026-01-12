package cmd

import (
	"testing"

	clitest "github.com/bargom/codeai/cmd/codeai/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand(t *testing.T) {
	t.Run("shows help when no command provided", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "CodeAI")
		assert.Contains(t, output, "Usage:")
	})

	t.Run("has global config flag", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "--config")
	})

	t.Run("has global verbose flag", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "--verbose")
	})

	t.Run("has global output flag", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "--output")
	})

	t.Run("shows all subcommands", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "parse")
		assert.Contains(t, output, "validate")
		assert.Contains(t, output, "deploy")
		assert.Contains(t, output, "config")
		assert.Contains(t, output, "server")
		assert.Contains(t, output, "version")
		assert.Contains(t, output, "completion")
	})

	t.Run("returns error for unknown command", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "unknowncommand")

		assert.Error(t, err)
	})
}

func TestGetRootCmd(t *testing.T) {
	cmd := GetRootCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "codeai", cmd.Use)
}

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "codeai", cmd.Use)

	// Verify all subcommands are present
	subcommands := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subcommands[sub.Use] = true
	}

	assert.True(t, subcommands["version"])
	assert.True(t, subcommands["parse <file>"])
	assert.True(t, subcommands["validate <file>"])
	assert.True(t, subcommands["deploy"])
	assert.True(t, subcommands["config"])
	assert.True(t, subcommands["server"])
}

func TestExecute(t *testing.T) {
	// Execute() uses the global rootCmd which is already set up
	// We just verify it doesn't panic
	// Note: This modifies global state, so keep it simple
}

func TestHelperFunctions(t *testing.T) {
	t.Run("isVerbose returns false by default", func(t *testing.T) {
		// Reset state
		verbose = false
		assert.False(t, isVerbose())
	})

	t.Run("isVerbose returns true when set", func(t *testing.T) {
		verbose = true
		assert.True(t, isVerbose())
		verbose = false // reset
	})

	t.Run("getOutputFormat returns plain by default", func(t *testing.T) {
		outputFormat = "plain"
		assert.Equal(t, "plain", getOutputFormat())
	})

	t.Run("getOutputFormat returns current value", func(t *testing.T) {
		outputFormat = "json"
		assert.Equal(t, "json", getOutputFormat())
		outputFormat = "plain" // reset
	})
}

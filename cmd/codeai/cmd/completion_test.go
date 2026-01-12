package cmd

import (
	"testing"

	clitest "github.com/bargom/codeai/cmd/codeai/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionCommand(t *testing.T) {
	t.Run("has shell subcommands", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "completion", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "bash")
		assert.Contains(t, output, "zsh")
		assert.Contains(t, output, "fish")
		assert.Contains(t, output, "powershell")
	})

	t.Run("generates bash completion", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "completion", "bash")

		require.NoError(t, err)
		assert.Contains(t, output, "bash")
		assert.Contains(t, output, "codeai")
	})

	t.Run("generates zsh completion", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "completion", "zsh")

		require.NoError(t, err)
		assert.Contains(t, output, "#compdef")
	})

	t.Run("generates fish completion", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "completion", "fish")

		require.NoError(t, err)
		assert.Contains(t, output, "complete")
		assert.Contains(t, output, "codeai")
	})

	t.Run("generates powershell completion", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "completion", "powershell")

		require.NoError(t, err)
		assert.Contains(t, output, "Register")
	})
}

func TestCompletionCommandHelp(t *testing.T) {
	rootCmd := NewRootCmd()
	output, err := clitest.ExecuteCommand(rootCmd, "completion", "--help")

	require.NoError(t, err)
	assert.Contains(t, output, "Generate")
	assert.Contains(t, output, "shell")
}

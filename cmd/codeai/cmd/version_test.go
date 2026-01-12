package cmd

import (
	"testing"

	clitest "github.com/bargom/codeai/cmd/codeai/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCommand(t *testing.T) {
	t.Run("prints version information", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "version")

		require.NoError(t, err)
		assert.Contains(t, output, "CodeAI")
		assert.Contains(t, output, "v") // Version prefix
	})

	t.Run("prints build date", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "version")

		require.NoError(t, err)
		assert.Contains(t, output, "Build")
	})

	t.Run("prints git commit", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "version")

		require.NoError(t, err)
		assert.Contains(t, output, "Commit")
	})

	t.Run("JSON output format", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "version", "--output", "json")

		require.NoError(t, err)
		assert.Contains(t, output, "version")
		assert.Contains(t, output, "{")
		assert.Contains(t, output, "}")
	})

	t.Run("does not accept arguments", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "version", "extra")

		// Version command should not accept extra arguments
		assert.Error(t, err)
	})
}

func TestVersionCommandHelp(t *testing.T) {
	rootCmd := NewRootCmd()
	output, err := clitest.ExecuteCommand(rootCmd, "version", "--help")

	require.NoError(t, err)
	assert.Contains(t, output, "version")
}

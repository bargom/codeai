package cmd

import (
	"testing"

	clitest "github.com/bargom/codeai/cmd/codeai/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerCommand(t *testing.T) {
	t.Run("has subcommands", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "server", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "start")
		assert.Contains(t, output, "migrate")
	})
}

func TestServerStartCommand(t *testing.T) {
	t.Run("has port flag", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "server", "start", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "port")
	})

	t.Run("has host flag", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "server", "start", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "host")
	})

	t.Run("accepts custom port", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "server", "start", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "8080") // default port
	})
}

func TestServerMigrateCommand(t *testing.T) {
	t.Run("has database connection flags", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "server", "migrate", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "db")
	})

	t.Run("has dry-run flag", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "server", "migrate", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "dry-run")
	})

	t.Run("dry-run shows pending migrations", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "server", "migrate", "--dry-run")

		require.NoError(t, err)
		assert.Contains(t, output, "migration")
	})
}

func TestServerCommandHelp(t *testing.T) {
	rootCmd := NewRootCmd()
	output, err := clitest.ExecuteCommand(rootCmd, "server", "--help")

	require.NoError(t, err)
	assert.Contains(t, output, "server")
	assert.Contains(t, output, "Usage:")
}

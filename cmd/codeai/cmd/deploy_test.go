package cmd

import (
	"os"
	"testing"

	clitest "github.com/bargom/codeai/cmd/codeai/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployCommand(t *testing.T) {
	t.Run("has subcommands", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "create")
		assert.Contains(t, output, "list")
		assert.Contains(t, output, "get")
		assert.Contains(t, output, "delete")
		assert.Contains(t, output, "execute")
	})
}

func TestDeployCreateCommand(t *testing.T) {
	t.Run("requires file argument", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "deploy", "create")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "accepts 1 arg")
	})

	t.Run("validates file exists", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "deploy", "create", "nonexistent.cai")

		assert.Error(t, err)
	})

	t.Run("validates file syntax before creating", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = `)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "deploy", "create", tmpfile)

		assert.Error(t, err)
	})

	t.Run("accepts valid file", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		// This will fail without API server, but should parse successfully
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "create", "--dry-run", tmpfile)

		// dry-run should succeed
		assert.NoError(t, err)
		assert.Contains(t, output, "valid")
	})

	t.Run("accepts name flag", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "create", "--name", "my-deploy", "--dry-run", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, "my-deploy")
	})
}

func TestDeployListCommand(t *testing.T) {
	t.Run("runs without arguments", func(t *testing.T) {
		rootCmd := NewRootCmd()
		// Will error without API server but should show proper error
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "list", "--api-url", "")

		// Without server, expect graceful error message
		if err != nil {
			assert.Contains(t, err.Error(), "API")
		} else {
			assert.Contains(t, output, "deployment") // Shows empty list or header
		}
	})

	t.Run("accepts limit flag", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "list", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "limit")
	})

	t.Run("lists with verbose output", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "list", "--verbose")

		require.NoError(t, err)
		assert.Contains(t, output, "Listing")
	})
}

func TestDeployGetCommand(t *testing.T) {
	t.Run("requires ID argument", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "deploy", "get")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "accepts 1 arg")
	})

	t.Run("accepts ID argument", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "get", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "id")
	})

	t.Run("shows not found message", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "get", "test-id")

		require.NoError(t, err)
		assert.Contains(t, output, "not found")
	})
}

func TestDeployDeleteCommand(t *testing.T) {
	t.Run("requires ID argument", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "deploy", "delete")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "accepts 1 arg")
	})

	t.Run("has force flag", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "delete", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "force")
	})

	t.Run("prompts without force flag", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "delete", "test-id")

		require.NoError(t, err)
		assert.Contains(t, output, "Are you sure")
	})

	t.Run("deletes with force flag", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "delete", "--force", "test-id")

		require.NoError(t, err)
		assert.Contains(t, output, "deleted")
	})
}

func TestDeployExecuteCommand(t *testing.T) {
	t.Run("requires ID argument", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "deploy", "execute")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "accepts 1 arg")
	})

	t.Run("has wait flag", func(t *testing.T) {
		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "deploy", "execute", "--help")

		require.NoError(t, err)
		assert.Contains(t, output, "wait")
	})
}

func TestDeployCommandHelp(t *testing.T) {
	rootCmd := NewRootCmd()
	output, err := clitest.ExecuteCommand(rootCmd, "deploy", "--help")

	require.NoError(t, err)
	assert.Contains(t, output, "deploy")
	assert.Contains(t, output, "Usage:")
}

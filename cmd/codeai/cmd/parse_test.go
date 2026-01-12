package cmd

import (
	"os"
	"strings"
	"testing"

	clitest "github.com/bargom/codeai/cmd/codeai/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCommand(t *testing.T) {
	t.Run("parses valid file with variable declaration", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "hello"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, "Name")
		assert.Contains(t, output, "x")
	})

	t.Run("parses file with multiple statements", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
var name = "CodeAI"
var count = 42
var items = ["a", "b", "c"]
`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, "Statements")
		assert.Contains(t, output, "name")
		assert.Contains(t, output, "count")
		assert.Contains(t, output, "items")
	})

	t.Run("parses file with function declaration", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
function greet(name) {
	var msg = "Hello"
}
`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, "Params")
		assert.Contains(t, output, "greet")
	})

	t.Run("parses file with exec block", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
exec {
	echo "Hello World"
}
`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, "Command")
		assert.Contains(t, output, "Hello World")
	})

	t.Run("handles missing file error", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "parse", "nonexistent.cai")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no such file")
	})

	t.Run("handles syntax error gracefully", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = `)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		assert.Error(t, err)
	})

	t.Run("requires file argument", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "parse")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "accepts 1 arg")
	})

	t.Run("outputs JSON format when requested", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", "--output", "json", tmpfile)

		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(strings.TrimSpace(output), "{") || strings.HasPrefix(strings.TrimSpace(output), "["))
	})

	t.Run("verbose mode shows additional info", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", "--verbose", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, tmpfile) // Should show filename
	})

	t.Run("parses if statement", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
var flag = true
if flag {
	var result = "yes"
}
`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, "Condition")
		assert.Contains(t, output, "Then")
	})

	t.Run("parses for loop", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
var items = ["a", "b"]
for item in items {
	var x = item
}
`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, "Variable")
		assert.Contains(t, output, "Iterable")
	})
}

func TestParseCommandHelp(t *testing.T) {
	rootCmd := NewRootCmd()
	output, err := clitest.ExecuteCommand(rootCmd, "parse", "--help")

	require.NoError(t, err)
	assert.Contains(t, output, "Parse")
	assert.Contains(t, output, "Usage:")
}

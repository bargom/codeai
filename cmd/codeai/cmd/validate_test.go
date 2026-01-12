package cmd

import (
	"os"
	"testing"

	clitest "github.com/bargom/codeai/cmd/codeai/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCommand(t *testing.T) {
	t.Run("validates correct file", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "hello"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, "valid")
	})

	t.Run("validates file with function", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
function greet(name) {
	var msg = name
}
`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, "valid")
	})

	t.Run("detects undefined variable", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
var x = undefinedVar
`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "undefined")
	})

	t.Run("detects duplicate declaration", func(t *testing.T) {
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

	t.Run("detects wrong argument count for function", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
function add(a, b) {
	var sum = a
}
var result = add("one")
`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "argument")
	})

	t.Run("handles missing file", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate", "nonexistent.cai")

		assert.Error(t, err)
	})

	t.Run("handles syntax error", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = `)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		assert.Error(t, err)
	})

	t.Run("requires file argument", func(t *testing.T) {
		rootCmd := NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "accepts 1 arg")
	})

	t.Run("verbose mode shows additional info", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `var x = "test"`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", "--verbose", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, tmpfile)
	})

	t.Run("validates for loop correctly", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
var items = ["a", "b", "c"]
for item in items {
	var current = item
}
`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, "valid")
	})

	t.Run("validates if-else statement", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, `
var flag = true
if flag {
	var a = "yes"
} else {
	var b = "no"
}
`)
		defer os.Remove(tmpfile)

		rootCmd := NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		assert.NoError(t, err)
		assert.Contains(t, output, "valid")
	})
}

func TestValidateCommandHelp(t *testing.T) {
	rootCmd := NewRootCmd()
	output, err := clitest.ExecuteCommand(rootCmd, "validate", "--help")

	require.NoError(t, err)
	assert.Contains(t, output, "Validate")
	assert.Contains(t, output, "Usage:")
}

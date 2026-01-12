// Package testing provides test utilities for CLI commands.
package testing

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ExecuteCommand runs a cobra command with the given arguments and returns the output.
func ExecuteCommand(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err := root.Execute()
	return buf.String(), err
}

// ExecuteCommandWithErr runs a cobra command and captures stdout and stderr separately.
func ExecuteCommandWithErr(root *cobra.Command, args ...string) (stdout string, stderr string, err error) {
	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)
	root.SetOut(stdoutBuf)
	root.SetErr(stderrBuf)
	root.SetArgs(args)

	err = root.Execute()
	return stdoutBuf.String(), stderrBuf.String(), err
}

// CaptureOutput captures stdout and stderr while executing the given function.
func CaptureOutput(f func()) (stdout string, stderr string) {
	// Save original stdout/stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Create pipes
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Run the function
	f()

	// Close writers and restore
	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Read captured output
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutBuf.ReadFrom(rOut)
	stderrBuf.ReadFrom(rErr)

	return stdoutBuf.String(), stderrBuf.String()
}

// CreateTempFile creates a temporary file with the given content and returns its path.
// The caller is responsible for removing the file.
func CreateTempFile(t *testing.T, content string) string {
	t.Helper()

	f, err := os.CreateTemp("", "codeai-test-*.cai")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		t.Fatalf("failed to write to temp file: %v", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		t.Fatalf("failed to close temp file: %v", err)
	}

	return f.Name()
}

// CreateTempFileWithExt creates a temporary file with given extension and content.
func CreateTempFileWithExt(t *testing.T, ext, content string) string {
	t.Helper()

	f, err := os.CreateTemp("", "codeai-test-*"+ext)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		t.Fatalf("failed to write to temp file: %v", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		t.Fatalf("failed to close temp file: %v", err)
	}

	return f.Name()
}

// CreateTempDir creates a temporary directory for tests.
func CreateTempDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "codeai-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	return dir
}

// ResetCommand resets a cobra command for reuse in tests.
func ResetCommand(cmd *cobra.Command) {
	cmd.SetArgs([]string{})
	cmd.SetOut(nil)
	cmd.SetErr(nil)

	// Reset flags to defaults
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue)
	})
}

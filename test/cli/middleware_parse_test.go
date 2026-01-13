//go:build integration

package cli_test

import (
    "os"
    "os/exec"
    "strings"
    "testing"
)

func TestParseMiddlewareAndAuth(t *testing.T) {
    // Build the CLI if not already built
    buildCmd := exec.Command("make", "build")
    buildCmd.Dir = "../.."
    if err := buildCmd.Run(); err != nil {
        t.Fatalf("Failed to build CLI: %v", err)
    }

    // Run parse command on with_auth.cai example
    cmd := exec.Command("../../bin/codeai", "parse", "../../examples/with_auth.cai")
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("Parse command failed: %v\nOutput: %s", err, string(output))
    }

    outputStr := string(output)

    // Verify middleware are parsed (check for MiddlewareType in JSON)
    if !strings.Contains(outputStr, "MiddlewareType") {
        t.Error("Expected output to contain MiddlewareType")
    }

    // Verify auth provider
    if !strings.Contains(outputStr, "jwt_provider") {
        t.Error("Expected output to mention jwt_provider")
    }

    // Verify roles
    if !strings.Contains(outputStr, "admin") {
        t.Error("Expected output to mention admin role")
    }

    if !strings.Contains(outputStr, "viewer") {
        t.Error("Expected output to mention viewer role")
    }

    // Verify specific middleware types
    if !strings.Contains(outputStr, "authentication") {
        t.Error("Expected output to mention authentication middleware")
    }

    if !strings.Contains(outputStr, "rate_limiting") {
        t.Error("Expected output to mention rate_limiting middleware")
    }
}

func TestParseInvalidAuth(t *testing.T) {
    // Create temporary file with invalid auth
    tmpfile, err := os.CreateTemp("", "invalid_auth_*.cai")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpfile.Name())

    content := `
    auth bad_provider {
      method invalid_method
      jwks_url "http://insecure.com/jwks.json"
      issuer "test"
      audience "test"
    }
    `

    if _, err := tmpfile.Write([]byte(content)); err != nil {
        t.Fatal(err)
    }
    tmpfile.Close()

    // Build the CLI if not already built
    buildCmd := exec.Command("make", "build")
    buildCmd.Dir = "../.."
    if err := buildCmd.Run(); err != nil {
        t.Fatalf("Failed to build CLI: %v", err)
    }

    // Run parse command - should fail
    cmd := exec.Command("../../bin/codeai", "parse", tmpfile.Name())
    output, err := cmd.CombinedOutput()

    // Expect error due to invalid method
    if err == nil {
        t.Errorf("Expected parse to fail for invalid auth method")
    }

    outputStr := string(output)
    if !strings.Contains(outputStr, "invalid") && !strings.Contains(outputStr, "error") {
        t.Errorf("Expected error message in output, got: %s", outputStr)
    }
}

func TestValidateMiddleware(t *testing.T) {
    // Build the CLI if not already built
    buildCmd := exec.Command("make", "build")
    buildCmd.Dir = "../.."
    if err := buildCmd.Run(); err != nil {
        t.Fatalf("Failed to build CLI: %v", err)
    }

    // Run validate command on with_auth.cai example
    cmd := exec.Command("../../bin/codeai", "validate", "../../examples/with_auth.cai")
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("Validate command failed: %v\nOutput: %s", err, string(output))
    }

    outputStr := string(output)

    // Should indicate successful validation
    if !strings.Contains(outputStr, "valid") && !strings.Contains(outputStr, "success") {
        t.Errorf("Expected validation success message, got: %s", outputStr)
    }
}
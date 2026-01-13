//go:build integration

// Package cli provides CLI integration tests for CodeAI.
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bargom/codeai/cmd/codeai/cmd"
	clitest "github.com/bargom/codeai/cmd/codeai/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompleteAppParseWorkflow tests parsing the complete app example via CLI.
func TestCompleteAppParseWorkflow(t *testing.T) {
	// Find the complete_app.cai file
	cwd, err := os.Getwd()
	require.NoError(t, err)

	var examplesPath string
	for {
		testPath := filepath.Join(cwd, "examples", "complete_app.cai")
		if _, err := os.Stat(testPath); err == nil {
			examplesPath = testPath
			break
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Skip("Could not find examples/complete_app.cai")
			return
		}
		cwd = parent
	}

	rootCmd := cmd.NewRootCmd()
	output, err := clitest.ExecuteCommand(rootCmd, "parse", examplesPath)

	require.NoError(t, err, "Parse command should succeed")

	// Verify key elements are in the output
	assert.Contains(t, output, "User", "Should contain User model")
	assert.Contains(t, output, "Product", "Should contain Product model")
	assert.Contains(t, output, "Order", "Should contain Order model")
	assert.Contains(t, output, "jwt_provider", "Should contain jwt_provider auth")
	assert.Contains(t, output, "admin", "Should contain admin role")
	assert.Contains(t, output, "stripe", "Should contain stripe integration")
}

// TestCompleteAppValidateWorkflow tests validating the complete app example via CLI.
func TestCompleteAppValidateWorkflow(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	var examplesPath string
	for {
		testPath := filepath.Join(cwd, "examples", "complete_app.cai")
		if _, err := os.Stat(testPath); err == nil {
			examplesPath = testPath
			break
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Skip("Could not find examples/complete_app.cai")
			return
		}
		cwd = parent
	}

	rootCmd := cmd.NewRootCmd()
	output, err := clitest.ExecuteCommand(rootCmd, "validate", examplesPath)

	require.NoError(t, err, "Validate command should succeed")
	assert.Contains(t, output, "valid", "Output should indicate file is valid")
}

// TestDatabaseModelsCLI tests parsing database models via CLI.
func TestDatabaseModelsCLI(t *testing.T) {
	source := `
config {
    database_type: "postgres"
}

database postgres {
    model User {
        id: uuid, primary, auto
        email: string, required, unique
        name: string, required
        role: string, default("user")
        created_at: timestamp, auto

        index: [email]
    }

    model Post {
        id: uuid, primary, auto
        user_id: ref(User), required
        title: string, required
        content: text, optional
        created_at: timestamp, auto

        index: [user_id]
    }
}
`
	tmpfile := clitest.CreateTempFile(t, source)
	defer os.Remove(tmpfile)

	t.Run("parse database models", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "User")
		assert.Contains(t, output, "Post")
		assert.Contains(t, output, "email")
		assert.Contains(t, output, "user_id")
	})

	t.Run("validate database models", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
	})
}

// TestAuthProvidersCLI tests parsing auth providers via CLI.
func TestAuthProvidersCLI(t *testing.T) {
	source := `
auth jwt_provider {
    method jwt
    jwks_url "https://auth.example.com/.well-known/jwks.json"
    issuer "https://auth.example.com"
    audience "api.example.com"
}

auth oauth2_provider {
    method oauth2
}

auth apikey_provider {
    method apikey
}
`
	tmpfile := clitest.CreateTempFile(t, source)
	defer os.Remove(tmpfile)

	t.Run("parse auth providers", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "jwt_provider")
		assert.Contains(t, output, "oauth2_provider")
		assert.Contains(t, output, "apikey_provider")
		assert.Contains(t, output, "jwt")
		assert.Contains(t, output, "oauth2")
	})

	t.Run("validate auth providers", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
	})
}

// TestRoleDefinitionsCLI tests parsing role definitions via CLI.
func TestRoleDefinitionsCLI(t *testing.T) {
	source := `
role admin {
    permissions ["users:*", "products:*", "orders:*"]
}

role manager {
    permissions ["products:read", "products:write"]
}

role customer {
    permissions ["products:read", "orders:create"]
}
`
	tmpfile := clitest.CreateTempFile(t, source)
	defer os.Remove(tmpfile)

	t.Run("parse role definitions", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "admin")
		assert.Contains(t, output, "manager")
		assert.Contains(t, output, "customer")
		assert.Contains(t, output, "Permissions") // JSON field is capitalized
	})

	t.Run("validate role definitions", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
	})
}

// TestMiddlewareCLI tests parsing middleware via CLI.
func TestMiddlewareCLI(t *testing.T) {
	source := `
auth jwt_provider {
    method jwt
    jwks_url "https://auth.example.com/.well-known/jwks.json"
    issuer "https://auth.example.com"
    audience "api.example.com"
}

middleware auth_required {
    type authentication
    config {
        provider: jwt_provider
        required: true
    }
}

middleware rate_limit {
    type rate_limiting
    config {
        requests: 100
        window: "1m"
        strategy: sliding_window
    }
}
`
	tmpfile := clitest.CreateTempFile(t, source)
	defer os.Remove(tmpfile)

	t.Run("parse middleware", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "auth_required")
		assert.Contains(t, output, "rate_limit")
		assert.Contains(t, output, "authentication")
		assert.Contains(t, output, "rate_limiting")
	})

	t.Run("validate middleware", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
	})
}

// TestEventsCLI tests parsing events via CLI.
func TestEventsCLI(t *testing.T) {
	source := `
event user_created {
    schema {
        user_id uuid
        email string
        created_at timestamp
    }
}

event order_created {
    schema {
        order_id uuid
        user_id uuid
        total decimal
    }
}

on "user.created" do workflow "send_welcome_email" async
on "order.created" do webhook "notify_shipping"
`
	tmpfile := clitest.CreateTempFile(t, source)
	defer os.Remove(tmpfile)

	t.Run("parse events", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "user_created")
		assert.Contains(t, output, "order_created")
		assert.Contains(t, output, "Schema") // JSON field is capitalized
	})

	t.Run("validate events", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
	})
}

// TestIntegrationsCLI tests parsing integrations via CLI.
func TestIntegrationsCLI(t *testing.T) {
	source := `
integration stripe {
    type rest
    base_url "https://api.stripe.com/v1"
    auth bearer {
        token: "$STRIPE_SECRET_KEY"
    }
    timeout "30s"
    circuit_breaker {
        threshold 5
        timeout "60s"
        max_concurrent 100
    }
}

integration analytics {
    type graphql
    base_url "https://api.analytics.example.com/graphql"
    auth bearer {
        token: "$ANALYTICS_API_KEY"
    }
    timeout "10s"
}
`
	tmpfile := clitest.CreateTempFile(t, source)
	defer os.Remove(tmpfile)

	t.Run("parse integrations", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "stripe")
		assert.Contains(t, output, "analytics")
		assert.Contains(t, output, "rest")
		assert.Contains(t, output, "graphql")
		assert.Contains(t, output, "CircuitBreaker") // JSON field is capitalized
	})

	t.Run("validate integrations", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
	})
}

// TestWebhooksCLI tests parsing webhooks via CLI.
func TestWebhooksCLI(t *testing.T) {
	source := `
webhook order_notification {
    event "order.created"
    url "https://api.example.com/webhooks/orders"
    method POST
    headers {
        "Content-Type": "application/json"
        "X-Source": "codeai"
    }
    retry 5 initial_interval "1s" backoff 2.0
}

webhook slack_alert {
    event "error.critical"
    url "https://hooks.slack.com/services/XXX"
    method POST
    headers {
        "Content-Type": "application/json"
    }
    retry 3 initial_interval "500ms" backoff 1.5
}
`
	tmpfile := clitest.CreateTempFile(t, source)
	defer os.Remove(tmpfile)

	t.Run("parse webhooks", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "order_notification")
		assert.Contains(t, output, "slack_alert")
		assert.Contains(t, output, "order.created")
		assert.Contains(t, output, "POST")
	})

	t.Run("validate webhooks", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
	})
}

// TestAllExamplesValidateCLI tests validating all passing example files via CLI.
func TestAllExamplesValidateCLI(t *testing.T) {
	passingExamples := []string{
		"06-mongodb-collections/mongodb-collections.cai",
		"07-mixed-databases/mixed-databases.cai",
		"08-with-auth/with_auth.cai",
		"10-events-integrations/events.cai",
		"complete_app.cai",
	}

	cwd, err := os.Getwd()
	require.NoError(t, err)

	var examplesDir string
	for {
		testPath := filepath.Join(cwd, "examples")
		if _, err := os.Stat(testPath); err == nil {
			examplesDir = testPath
			break
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			t.Skip("Could not find examples directory")
			return
		}
		cwd = parent
	}

	for _, example := range passingExamples {
		t.Run(example, func(t *testing.T) {
			fullPath := filepath.Join(examplesDir, example)

			// First parse
			rootCmd := cmd.NewRootCmd()
			parseOutput, parseErr := clitest.ExecuteCommand(rootCmd, "parse", fullPath)
			require.NoError(t, parseErr, "Parse should succeed for %s", example)
			assert.NotEmpty(t, parseOutput, "Parse output should not be empty")

			// Then validate
			rootCmd = cmd.NewRootCmd()
			validateOutput, validateErr := clitest.ExecuteCommand(rootCmd, "validate", fullPath)
			require.NoError(t, validateErr, "Validate should succeed for %s", example)
			assert.Contains(t, validateOutput, "valid", "Should report file as valid")
		})
	}
}

// TestErrorHandlingCLI tests CLI error handling.
func TestErrorHandlingCLI(t *testing.T) {
	t.Run("parse nonexistent file", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "parse", "/nonexistent/file.cai")
		assert.Error(t, err)
	})

	t.Run("validate nonexistent file", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate", "/nonexistent/file.cai")
		assert.Error(t, err)
	})

	t.Run("parse invalid syntax", func(t *testing.T) {
		source := `config { invalid syntax here }`
		tmpfile := clitest.CreateTempFile(t, source)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)
		assert.Error(t, err)
	})

	t.Run("parse missing argument", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "parse")
		assert.Error(t, err)
	})

	t.Run("validate missing argument", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate")
		assert.Error(t, err)
	})
}

// TestHelpCommands tests help output for commands.
func TestHelpCommands(t *testing.T) {
	t.Run("root help", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "--help")
		require.NoError(t, err)
		assert.Contains(t, output, "codeai")
		assert.Contains(t, output, "parse")
		assert.Contains(t, output, "validate")
	})

	t.Run("parse help", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", "--help")
		require.NoError(t, err)
		assert.Contains(t, output, "parse")
		assert.Contains(t, strings.ToLower(output), "dsl")
	})

	t.Run("validate help", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", "--help")
		require.NoError(t, err)
		assert.Contains(t, output, "validate")
	})
}

// TestVersionCommand tests version output.
func TestVersionCommand(t *testing.T) {
	rootCmd := cmd.NewRootCmd()
	output, err := clitest.ExecuteCommand(rootCmd, "version")
	require.NoError(t, err)
	assert.NotEmpty(t, output, "Version output should not be empty")
}

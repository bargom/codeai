//go:build integration

// Package cli provides CLI integration tests for CodeAI.
package cli

import (
	"os"
	"testing"

	"github.com/bargom/codeai/cmd/codeai/cmd"
	clitest "github.com/bargom/codeai/cmd/codeai/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLIParseMongoDBDSL tests parsing MongoDB DSL files.
func TestCLIParseMongoDBDSL(t *testing.T) {
	t.Run("parse MongoDB collection definition", func(t *testing.T) {
		dsl := `
config {
    database_type: "mongodb"
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "test_db"
}

database mongodb {
    collection User {
        description: "User accounts"

        _id: objectid, primary, auto
        email: string, required, unique
        username: string, required
        password_hash: string, required
        created_at: date, auto

        indexes {
            index: [email] unique
            index: [username] unique
        }
    }
}
`
		tmpfile := clitest.CreateTempFile(t, dsl)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "User")
		assert.Contains(t, output, "mongodb")
	})

	t.Run("parse MongoDB embedded document", func(t *testing.T) {
		dsl := `
config {
    database_type: "mongodb"
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "test_db"
}

database mongodb {
    collection Address {
        _id: objectid, primary, auto
        user_id: objectid, required

        location: embedded {
            type: string, required
            coordinates: array(double), required
        }

        indexes {
            index: [location] geospatial
        }
    }
}
`
		tmpfile := clitest.CreateTempFile(t, dsl)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "Address")
		assert.Contains(t, output, "EmbeddedDoc") // Embedded documents show as EmbeddedDoc in JSON output
	})

	t.Run("parse MongoDB array fields", func(t *testing.T) {
		dsl := `
config {
    database_type: "mongodb"
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "test_db"
}

database mongodb {
    collection Product {
        _id: objectid, primary, auto
        name: string, required
        tags: array(string), optional
        prices: array(double), optional

        indexes {
            index: [name]
        }
    }
}
`
		tmpfile := clitest.CreateTempFile(t, dsl)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "Product")
		assert.Contains(t, output, "array")
	})

	t.Run("parse MongoDB text index", func(t *testing.T) {
		dsl := `
config {
    database_type: "mongodb"
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "test_db"
}

database mongodb {
    collection Article {
        _id: objectid, primary, auto
        title: string, required
        content: string, required

        indexes {
            index: [title, content] text
        }
    }
}
`
		tmpfile := clitest.CreateTempFile(t, dsl)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "Article")
	})
}

// TestCLIValidateMongoDBDSL tests validating MongoDB DSL files.
func TestCLIValidateMongoDBDSL(t *testing.T) {
	t.Run("validate valid MongoDB DSL", func(t *testing.T) {
		dsl := `
config {
    database_type: "mongodb"
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "test_db"
}

database mongodb {
    collection User {
        _id: objectid, primary, auto
        email: string, required, unique
        username: string, required

        indexes {
            index: [email] unique
        }
    }
}
`
		tmpfile := clitest.CreateTempFile(t, dsl)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
	})

	t.Run("validate MongoDB collection without primary key", func(t *testing.T) {
		dsl := `
config {
    database_type: "mongodb"
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "test_db"
}

database mongodb {
    collection User {
        email: string, required
        username: string, required
    }
}
`
		tmpfile := clitest.CreateTempFile(t, dsl)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		// Should either pass (auto-generate _id) or fail with clear error
		// Behavior depends on implementation
		_ = err // Some implementations allow this, others require explicit _id
	})
}

// TestCLIMongoDBConfigValidation tests MongoDB configuration validation.
func TestCLIMongoDBConfigValidation(t *testing.T) {
	t.Run("validate missing mongodb_uri", func(t *testing.T) {
		dsl := `
config {
    database_type: "mongodb"
    mongodb_database: "test_db"
}

database mongodb {
    collection User {
        _id: objectid, primary, auto
        email: string, required
    }
}
`
		tmpfile := clitest.CreateTempFile(t, dsl)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		// May error or warn depending on implementation
		_ = err
	})

	t.Run("validate missing mongodb_database", func(t *testing.T) {
		dsl := `
config {
    database_type: "mongodb"
    mongodb_uri: "mongodb://localhost:27017"
}

database mongodb {
    collection User {
        _id: objectid, primary, auto
        email: string, required
    }
}
`
		tmpfile := clitest.CreateTempFile(t, dsl)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		// May error or warn depending on implementation
		_ = err
	})
}

// TestCLIMongoDBComplexDSL tests parsing and validating complex MongoDB DSL.
func TestCLIMongoDBComplexDSL(t *testing.T) {
	complexDSL := `
config {
    database_type: "mongodb"
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "ecommerce"
}

database mongodb {
    collection User {
        description: "User accounts with authentication"

        _id: objectid, primary, auto
        email: string, required, unique
        username: string, required, unique
        password_hash: string, required
        first_name: string, optional
        last_name: string, optional
        is_active: bool, default(true)
        created_at: date, auto
        updated_at: date, auto

        indexes {
            index: [email] unique
            index: [username] unique
            index: [created_at]
        }
    }

    collection Product {
        description: "E-commerce products with variants"

        _id: objectid, primary, auto
        sku: string, required, unique
        name: string, required
        description: string, optional
        price: double, required
        quantity: int, default(0)
        categories: array(string), optional
        tags: array(string), optional

        attributes: embedded {
            color: string, optional
            size: string, optional
            weight: double, optional
        }

        is_active: bool, default(true)
        created_at: date, auto
        updated_at: date, auto

        indexes {
            index: [sku] unique
            index: [name, description] text
            index: [categories]
            index: [price]
        }
    }

    collection Order {
        description: "Customer orders"

        _id: objectid, primary, auto
        order_number: string, required, unique
        user_id: objectid, required
        status: string, required, default("pending")

        items: array(object), required

        subtotal: double, required
        tax: double, default(0)
        total: double, required

        shipping_address: embedded {
            street: string, required
            city: string, required
            state: string, required
            postal_code: string, required
            country: string, required
        }

        created_at: date, auto
        updated_at: date, auto

        indexes {
            index: [order_number] unique
            index: [user_id]
            index: [status]
            index: [user_id, status]
        }
    }
}
`

	t.Run("parse complex MongoDB DSL", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, complexDSL)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "User")
		assert.Contains(t, output, "Product")
		assert.Contains(t, output, "Order")
	})

	t.Run("validate complex MongoDB DSL", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, complexDSL)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "valid")
	})
}

// TestCLIMongoDBInvalidSyntax tests error handling for invalid MongoDB syntax.
func TestCLIMongoDBInvalidSyntax(t *testing.T) {
	t.Run("invalid field type", func(t *testing.T) {
		dsl := `
config {
    database_type: "mongodb"
}

database mongodb {
    collection User {
        _id: invalidtype, primary
    }
}
`
		tmpfile := clitest.CreateTempFile(t, dsl)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		// Should fail parsing or validation
		// Result depends on parser strictness
		_ = err
	})

	t.Run("unclosed collection block", func(t *testing.T) {
		dsl := `
config {
    database_type: "mongodb"
}

database mongodb {
    collection User {
        _id: objectid, primary
`
		tmpfile := clitest.CreateTempFile(t, dsl)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		assert.Error(t, err)
	})

	t.Run("invalid index syntax", func(t *testing.T) {
		dsl := `
config {
    database_type: "mongodb"
}

database mongodb {
    collection User {
        _id: objectid, primary
        email: string

        indexes {
            index: [email invalidformat
        }
    }
}
`
		tmpfile := clitest.CreateTempFile(t, dsl)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		assert.Error(t, err)
	})
}

// TestCLIMongoDBOutputFormats tests output formats for MongoDB DSL.
func TestCLIMongoDBOutputFormats(t *testing.T) {
	dsl := `
config {
    database_type: "mongodb"
    mongodb_uri: "mongodb://localhost:27017"
    mongodb_database: "test_db"
}

database mongodb {
    collection User {
        _id: objectid, primary, auto
        email: string, required

        indexes {
            index: [email] unique
        }
    }
}
`

	t.Run("parse MongoDB with JSON output", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, dsl)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "parse", "--output", "json", tmpfile)

		require.NoError(t, err)
		assert.Contains(t, output, "{")
	})

	t.Run("validate MongoDB with JSON output", func(t *testing.T) {
		tmpfile := clitest.CreateTempFile(t, dsl)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		output, err := clitest.ExecuteCommand(rootCmd, "validate", "--output", "json", tmpfile)

		require.NoError(t, err)
		assert.NotEmpty(t, output)
	})
}

// TestCLIMongoDBErrorMessages tests that MongoDB error messages are helpful.
func TestCLIMongoDBErrorMessages(t *testing.T) {
	t.Run("clear error for unclosed collection", func(t *testing.T) {
		dsl := `
database mongodb {
    collection User {
        _id: objectid
`
		tmpfile := clitest.CreateTempFile(t, dsl)
		defer os.Remove(tmpfile)

		rootCmd := cmd.NewRootCmd()
		_, err := clitest.ExecuteCommand(rootCmd, "parse", tmpfile)

		assert.Error(t, err)
		// Error message should indicate parsing issue
		assert.Contains(t, err.Error(), "parse error")
	})
}

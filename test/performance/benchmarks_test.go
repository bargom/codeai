// Package performance provides performance benchmarks for CodeAI.
package performance

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/bargom/codeai/internal/database"
	"github.com/bargom/codeai/internal/database/models"
	"github.com/bargom/codeai/internal/database/repository"
	"github.com/bargom/codeai/internal/parser"
	"github.com/bargom/codeai/internal/validator"
	_ "modernc.org/sqlite"
)

// generateDSL generates a DSL program with the specified number of statements.
func generateDSL(statements int) string {
	var sb strings.Builder

	// Add variable declarations
	for i := 0; i < statements/4; i++ {
		sb.WriteString(fmt.Sprintf("var var%d = %d\n", i, i))
	}

	// Add function declarations
	for i := 0; i < statements/4; i++ {
		sb.WriteString(fmt.Sprintf("function func%d(p%d) {\n    var local%d = p%d\n}\n", i, i, i, i))
	}

	// Add if statements
	for i := 0; i < statements/4; i++ {
		sb.WriteString(fmt.Sprintf("var flag%d = true\nif flag%d {\n    var inner%d = 1\n}\n", i, i, i))
	}

	// Add for loops
	for i := 0; i < statements/4; i++ {
		sb.WriteString(fmt.Sprintf("var arr%d = [1, 2, 3]\nfor item%d in arr%d {\n    var x%d = item%d\n}\n", i, i, i, i, i))
	}

	return sb.String()
}

// BenchmarkParseSmall benchmarks parsing a small DSL file.
func BenchmarkParseSmall(b *testing.B) {
	content := generateDSL(10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseMedium benchmarks parsing a medium DSL file.
func BenchmarkParseMedium(b *testing.B) {
	content := generateDSL(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseLarge benchmarks parsing a large DSL file.
func BenchmarkParseLarge(b *testing.B) {
	content := generateDSL(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidateSmall benchmarks validating a small DSL file.
func BenchmarkValidateSmall(b *testing.B) {
	content := generateDSL(10)
	program, err := parser.Parse(content)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := validator.New()
		err := v.Validate(program)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidateMedium benchmarks validating a medium DSL file.
func BenchmarkValidateMedium(b *testing.B) {
	content := generateDSL(100)
	program, err := parser.Parse(content)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := validator.New()
		err := v.Validate(program)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidateLarge benchmarks validating a large DSL file.
func BenchmarkValidateLarge(b *testing.B) {
	content := generateDSL(1000)
	program, err := parser.Parse(content)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := validator.New()
		err := v.Validate(program)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseAndValidate benchmarks the combined parse+validate workflow.
func BenchmarkParseAndValidate(b *testing.B) {
	content := generateDSL(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program, err := parser.Parse(content)
		if err != nil {
			b.Fatal(err)
		}
		v := validator.New()
		err = v.Validate(program)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// setupTestDB creates an in-memory SQLite database for benchmarks.
func setupTestDB(b *testing.B) *sql.DB {
	b.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		b.Fatalf("failed to enable foreign keys: %v", err)
	}

	migrator := database.NewMigrator(db)
	if err := migrator.MigrateUp(); err != nil {
		b.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

// BenchmarkConfigCreate benchmarks config creation.
func BenchmarkConfigCreate(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := repository.NewConfigRepository(db)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := models.NewConfig(fmt.Sprintf("config-%d", i), "var x = 1")
		err := repo.Create(ctx, cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConfigRead benchmarks config reading.
func BenchmarkConfigRead(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := repository.NewConfigRepository(db)
	ctx := context.Background()

	// Create a config to read
	cfg := models.NewConfig("benchmark-config", "var x = 1")
	if err := repo.Create(ctx, cfg); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetByID(ctx, cfg.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConfigList benchmarks config listing.
func BenchmarkConfigList(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := repository.NewConfigRepository(db)
	ctx := context.Background()

	// Create multiple configs
	for i := 0; i < 100; i++ {
		cfg := models.NewConfig(fmt.Sprintf("config-%d", i), "var x = 1")
		if err := repo.Create(ctx, cfg); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.List(ctx, 20, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDeploymentCreate benchmarks deployment creation.
func BenchmarkDeploymentCreate(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	repo := repository.NewDeploymentRepository(db)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		deploy := models.NewDeployment(fmt.Sprintf("deploy-%d", i))
		err := repo.Create(ctx, deploy)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkExecutionCreate benchmarks execution creation.
func BenchmarkExecutionCreate(b *testing.B) {
	db := setupTestDB(b)
	defer db.Close()

	deployRepo := repository.NewDeploymentRepository(db)
	execRepo := repository.NewExecutionRepository(db)
	ctx := context.Background()

	// Create a deployment first
	deploy := models.NewDeployment("benchmark-deploy")
	if err := deployRepo.Create(ctx, deploy); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exec := models.NewExecution(deploy.ID, fmt.Sprintf("echo %d", i))
		err := execRepo.Create(ctx, exec)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Note: BenchmarkConcurrentConfigCreates is intentionally omitted because
// SQLite in-memory databases don't support concurrent writes well.
// For production database benchmarks, use PostgreSQL with testcontainers.

// BenchmarkParseAllocation reports allocations for parsing.
func BenchmarkParseAllocation(b *testing.B) {
	content := generateDSL(100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(content)
	}
}

// BenchmarkValidateAllocation reports allocations for validation.
func BenchmarkValidateAllocation(b *testing.B) {
	content := generateDSL(100)
	program, _ := parser.Parse(content)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := validator.New()
		_ = v.Validate(program)
	}
}

# Task 007: CLI Implementation

## Overview
Implement the CodeAI command-line interface using Cobra, providing commands for running applications, validating syntax, managing migrations, and generating OpenAPI specifications.

## Phase
Phase 1: Foundation

## Priority
Critical - CLI is the primary user interface.

## Dependencies
- Task 001: Project Structure Setup
- Task 002: Participle Grammar Implementation
- Task 004: Validator Implementation
- Task 005: PostgreSQL Database Module
- Task 006: HTTP Module with Chi Router

## Description
Create a comprehensive CLI tool using Cobra that allows developers to run CodeAI applications, validate source files, manage database migrations, and generate documentation.

## Detailed Requirements

### 1. Main Entry Point (cmd/codeai/main.go)

```go
package main

import (
    "fmt"
    "os"

    "github.com/codeai/codeai/cmd/codeai/commands"
)

var (
    version   = "dev"
    buildTime = "unknown"
    commit    = "unknown"
)

func main() {
    commands.SetVersionInfo(version, buildTime, commit)

    if err := commands.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

### 2. Root Command (cmd/codeai/commands/root.go)

```go
package commands

import (
    "os"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var (
    cfgFile string
    verbose bool
    version string
    buildTime string
    commit string
)

var rootCmd = &cobra.Command{
    Use:   "codeai",
    Short: "CodeAI - LLM-Native Programming Language Runtime",
    Long: `CodeAI is a domain-specific programming language designed for LLM code generation.

It provides high-level business primitives for building backend applications
that compile to a high-performance Go runtime.

Documentation: https://codeai.dev/docs
GitHub: https://github.com/codeai/codeai`,
}

func Execute() error {
    return rootCmd.Execute()
}

func SetVersionInfo(v, bt, c string) {
    version = v
    buildTime = bt
    commit = c
}

func init() {
    cobra.OnInitialize(initConfig)

    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./cai.yaml)")
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

    // Bind flags to viper
    viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

func initConfig() {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        viper.SetConfigName("codeai")
        viper.SetConfigType("yaml")
        viper.AddConfigPath(".")
        viper.AddConfigPath("$HOME/.cai")
    }

    viper.AutomaticEnv()
    viper.SetEnvPrefix("CODEAI")

    if err := viper.ReadInConfig(); err == nil {
        if verbose {
            fmt.Println("Using config file:", viper.ConfigFileUsed())
        }
    }
}
```

### 3. Run Command (cmd/codeai/commands/run.go)

```go
package commands

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "log/slog"

    "github.com/codeai/codeai/internal/runtime"
)

var runCmd = &cobra.Command{
    Use:   "run [directory]",
    Short: "Run a CodeAI application",
    Long: `Run a CodeAI application from the specified directory.

The directory should contain one or more .cai files defining
your application's entities, endpoints, workflows, and other components.

Example:
  codeai run ./myapp
  codeai run ./myapp --port 3000
  codeai run . --env production`,
    Args: cobra.MaximumNArgs(1),
    RunE: runRun,
}

func init() {
    rootCmd.AddCommand(runCmd)

    runCmd.Flags().IntP("port", "p", 8080, "HTTP server port")
    runCmd.Flags().String("host", "0.0.0.0", "HTTP server host")
    runCmd.Flags().StringP("env", "e", "development", "Environment (development, staging, production)")
    runCmd.Flags().Bool("migrate", true, "Run database migrations on startup")
    runCmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")
    runCmd.Flags().String("log-format", "text", "Log format (text, json)")

    viper.BindPFlag("port", runCmd.Flags().Lookup("port"))
    viper.BindPFlag("host", runCmd.Flags().Lookup("host"))
    viper.BindPFlag("env", runCmd.Flags().Lookup("env"))
    viper.BindPFlag("migrate", runCmd.Flags().Lookup("migrate"))
    viper.BindPFlag("log_level", runCmd.Flags().Lookup("log-level"))
    viper.BindPFlag("log_format", runCmd.Flags().Lookup("log-format"))
}

func runRun(cmd *cobra.Command, args []string) error {
    // Get source directory
    sourceDir := "."
    if len(args) > 0 {
        sourceDir = args[0]
    }

    // Setup logging
    setupLogging(viper.GetString("log_level"), viper.GetString("log_format"))

    slog.Info("starting CodeAI",
        "version", version,
        "source_dir", sourceDir,
        "environment", viper.GetString("env"),
    )

    // Create engine
    engine, err := runtime.NewEngine(sourceDir)
    if err != nil {
        return fmt.Errorf("failed to create engine: %w", err)
    }

    // Configure engine
    engine.Configure(&runtime.Config{
        Environment: viper.GetString("env"),
        HTTP: runtime.HTTPConfig{
            Host: viper.GetString("host"),
            Port: viper.GetInt("port"),
        },
        Database: runtime.DatabaseConfig{
            ConnectionString: os.Getenv("DATABASE_URL"),
            Migrate:          viper.GetBool("migrate"),
        },
    })

    // Create context with cancellation
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle shutdown signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        sig := <-sigChan
        slog.Info("received shutdown signal", "signal", sig)
        cancel()
    }()

    // Run engine
    if err := engine.Run(ctx); err != nil {
        return fmt.Errorf("engine error: %w", err)
    }

    slog.Info("shutdown complete")
    return nil
}

func setupLogging(level, format string) {
    var logLevel slog.Level
    switch level {
    case "debug":
        logLevel = slog.LevelDebug
    case "warn":
        logLevel = slog.LevelWarn
    case "error":
        logLevel = slog.LevelError
    default:
        logLevel = slog.LevelInfo
    }

    var handler slog.Handler
    opts := &slog.HandlerOptions{Level: logLevel}

    if format == "json" {
        handler = slog.NewJSONHandler(os.Stdout, opts)
    } else {
        handler = slog.NewTextHandler(os.Stdout, opts)
    }

    slog.SetDefault(slog.New(handler))
}
```

### 4. Validate Command (cmd/codeai/commands/validate.go)

```go
package commands

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"

    "github.com/codeai/codeai/internal/parser"
    "github.com/codeai/codeai/internal/validator"
)

var validateCmd = &cobra.Command{
    Use:   "validate [directory]",
    Short: "Validate CodeAI source files",
    Long: `Validate CodeAI source files without running the application.

This command parses and validates all .cai files in the specified
directory, checking for syntax errors, type mismatches, undefined
references, and other issues.

Example:
  codeai validate ./myapp
  codeai validate . --format json`,
    Args: cobra.MaximumNArgs(1),
    RunE: runValidate,
}

func init() {
    rootCmd.AddCommand(validateCmd)

    validateCmd.Flags().String("format", "text", "Output format (text, json)")
    validateCmd.Flags().Bool("strict", false, "Treat warnings as errors")
}

func runValidate(cmd *cobra.Command, args []string) error {
    sourceDir := "."
    if len(args) > 0 {
        sourceDir = args[0]
    }

    format, _ := cmd.Flags().GetString("format")
    strict, _ := cmd.Flags().GetBool("strict")

    // Parse
    p := parser.New()
    program, err := p.ParseDirectory(sourceDir)
    if err != nil {
        if format == "json" {
            printJSONError(err)
        } else {
            printTextError(err)
        }
        return err
    }

    // Validate
    v := validator.NewValidator()
    ast, err := v.Validate(program)

    // Get warnings
    warnings := v.Warnings()

    if err != nil {
        if format == "json" {
            printJSONValidationResult(err.(*validator.ValidationErrors), warnings)
        } else {
            printTextValidationResult(err.(*validator.ValidationErrors), warnings)
        }
        return err
    }

    if strict && len(warnings) > 0 {
        if format == "json" {
            printJSONValidationResult(nil, warnings)
        } else {
            printTextValidationResult(nil, warnings)
        }
        return fmt.Errorf("validation failed with %d warnings in strict mode", len(warnings))
    }

    // Success
    if format == "json" {
        fmt.Println(`{"valid": true, "entities":`, len(ast.Entities), `, "endpoints":`, len(ast.Endpoints), `}`)
    } else {
        fmt.Println("✓ Validation successful")
        fmt.Printf("  Entities:     %d\n", len(ast.Entities))
        fmt.Printf("  Endpoints:    %d\n", len(ast.Endpoints))
        fmt.Printf("  Workflows:    %d\n", len(ast.Workflows))
        fmt.Printf("  Jobs:         %d\n", len(ast.Jobs))
        fmt.Printf("  Integrations: %d\n", len(ast.Integrations))
        fmt.Printf("  Events:       %d\n", len(ast.Events))

        if len(warnings) > 0 {
            fmt.Printf("\n⚠ %d warnings:\n", len(warnings))
            for _, w := range warnings {
                fmt.Printf("  - %s: %s\n", w.Position, w.Message)
            }
        }
    }

    return nil
}

func printTextError(err error) {
    fmt.Fprintf(os.Stderr, "✗ Parse error:\n")
    if pe, ok := err.(*parser.ParseError); ok {
        fmt.Fprintf(os.Stderr, "  %s\n", pe.Pretty())
    } else {
        fmt.Fprintf(os.Stderr, "  %s\n", err)
    }
}

func printTextValidationResult(errs *validator.ValidationErrors, warnings []validator.ValidationWarning) {
    if errs != nil {
        fmt.Fprintf(os.Stderr, "✗ Validation failed with %d errors:\n\n", len(errs.Errors))
        for i, e := range errs.Errors {
            fmt.Fprintf(os.Stderr, "%d. %s\n", i+1, e.Pretty())
        }
    }

    if len(warnings) > 0 {
        fmt.Fprintf(os.Stderr, "\n⚠ %d warnings:\n", len(warnings))
        for _, w := range warnings {
            fmt.Fprintf(os.Stderr, "  - %s: %s\n", w.Position, w.Message)
        }
    }
}

func printJSONError(err error) {
    // Implementation for JSON error output
}

func printJSONValidationResult(errs *validator.ValidationErrors, warnings []validator.ValidationWarning) {
    // Implementation for JSON validation result output
}
```

### 5. Migrate Command (cmd/codeai/commands/migrate.go)

```go
package commands

import (
    "context"
    "fmt"
    "os"

    "github.com/spf13/cobra"

    "github.com/codeai/codeai/internal/parser"
    "github.com/codeai/codeai/internal/validator"
    "github.com/codeai/codeai/internal/modules/database"
)

var migrateCmd = &cobra.Command{
    Use:   "migrate [directory]",
    Short: "Run database migrations",
    Long: `Run database migrations for a CodeAI application.

This command generates and applies database migrations based on
entity definitions in your CodeAI source files.

Example:
  codeai migrate ./myapp
  codeai migrate . --dry-run
  codeai migrate . --status`,
    Args: cobra.MaximumNArgs(1),
    RunE: runMigrate,
}

func init() {
    rootCmd.AddCommand(migrateCmd)

    migrateCmd.Flags().Bool("dry-run", false, "Show migrations without applying them")
    migrateCmd.Flags().Bool("status", false, "Show migration status")
    migrateCmd.Flags().String("database-url", "", "Database connection string (overrides DATABASE_URL)")
}

func runMigrate(cmd *cobra.Command, args []string) error {
    sourceDir := "."
    if len(args) > 0 {
        sourceDir = args[0]
    }

    dryRun, _ := cmd.Flags().GetBool("dry-run")
    status, _ := cmd.Flags().GetBool("status")
    dbURL, _ := cmd.Flags().GetString("database-url")

    if dbURL == "" {
        dbURL = os.Getenv("DATABASE_URL")
    }
    if dbURL == "" {
        return fmt.Errorf("DATABASE_URL is required (set via environment or --database-url)")
    }

    // Parse and validate
    p := parser.New()
    program, err := p.ParseDirectory(sourceDir)
    if err != nil {
        return fmt.Errorf("parse error: %w", err)
    }

    v := validator.NewValidator()
    ast, err := v.Validate(program)
    if err != nil {
        return fmt.Errorf("validation error: %w", err)
    }

    // Connect to database
    db, err := database.NewPostgresModule(&database.Config{
        ConnectionString: dbURL,
    })
    if err != nil {
        return fmt.Errorf("database connection error: %w", err)
    }

    ctx := context.Background()
    if err := db.Start(ctx); err != nil {
        return fmt.Errorf("database start error: %w", err)
    }
    defer db.Stop(ctx)

    // Register entities
    for _, entity := range ast.Entities {
        if err := db.RegisterEntity(entity); err != nil {
            return fmt.Errorf("register entity error: %w", err)
        }
    }

    // Show status
    if status {
        migrations, err := db.MigrationStatus(ctx)
        if err != nil {
            return fmt.Errorf("migration status error: %w", err)
        }

        if len(migrations) == 0 {
            fmt.Println("No migrations have been applied yet.")
        } else {
            fmt.Printf("Applied migrations (%d):\n", len(migrations))
            for _, m := range migrations {
                fmt.Printf("  %d. %s (applied %s)\n", m.ID, m.Name, m.AppliedAt.Format("2006-01-02 15:04:05"))
            }
        }
        return nil
    }

    // Dry run
    if dryRun {
        fmt.Println("Dry run - migrations that would be applied:")
        // Show pending migrations
        return nil
    }

    // Run migrations
    fmt.Println("Running migrations...")
    if err := db.Migrate(ctx); err != nil {
        return fmt.Errorf("migration error: %w", err)
    }

    fmt.Println("✓ Migrations completed successfully")
    return nil
}
```

### 6. OpenAPI Command (cmd/codeai/commands/openapi.go)

```go
package commands

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"

    "github.com/codeai/codeai/internal/parser"
    "github.com/codeai/codeai/internal/validator"
    "github.com/codeai/codeai/internal/openapi"
)

var openapiCmd = &cobra.Command{
    Use:   "openapi [directory]",
    Short: "Generate OpenAPI specification",
    Long: `Generate an OpenAPI 3.0 specification from CodeAI source files.

This command creates an OpenAPI spec describing all endpoints,
request/response schemas, and authentication requirements.

Example:
  codeai openapi ./myapp
  codeai openapi . --output api.yaml
  codeai openapi . --format json --output api.json`,
    Args: cobra.MaximumNArgs(1),
    RunE: runOpenAPI,
}

func init() {
    rootCmd.AddCommand(openapiCmd)

    openapiCmd.Flags().StringP("output", "o", "", "Output file (defaults to stdout)")
    openapiCmd.Flags().StringP("format", "f", "yaml", "Output format (yaml, json)")
    openapiCmd.Flags().String("title", "", "API title (defaults to app name)")
    openapiCmd.Flags().String("version", "1.0.0", "API version")
    openapiCmd.Flags().String("server", "", "Server URL to include")
}

func runOpenAPI(cmd *cobra.Command, args []string) error {
    sourceDir := "."
    if len(args) > 0 {
        sourceDir = args[0]
    }

    output, _ := cmd.Flags().GetString("output")
    format, _ := cmd.Flags().GetString("format")
    title, _ := cmd.Flags().GetString("title")
    apiVersion, _ := cmd.Flags().GetString("version")
    server, _ := cmd.Flags().GetString("server")

    // Parse and validate
    p := parser.New()
    program, err := p.ParseDirectory(sourceDir)
    if err != nil {
        return fmt.Errorf("parse error: %w", err)
    }

    v := validator.NewValidator()
    ast, err := v.Validate(program)
    if err != nil {
        return fmt.Errorf("validation error: %w", err)
    }

    // Generate OpenAPI spec
    gen := openapi.NewGenerator(&openapi.Config{
        Title:   title,
        Version: apiVersion,
        Server:  server,
    })

    spec, err := gen.Generate(ast)
    if err != nil {
        return fmt.Errorf("openapi generation error: %w", err)
    }

    // Serialize
    var data []byte
    if format == "json" {
        data, err = json.MarshalIndent(spec, "", "  ")
    } else {
        data, err = yaml.Marshal(spec)
    }
    if err != nil {
        return fmt.Errorf("serialization error: %w", err)
    }

    // Output
    if output == "" {
        fmt.Println(string(data))
    } else {
        if err := os.WriteFile(output, data, 0644); err != nil {
            return fmt.Errorf("write error: %w", err)
        }
        fmt.Printf("✓ OpenAPI specification written to %s\n", output)
    }

    return nil
}
```

### 7. Version Command (cmd/codeai/commands/version.go)

```go
package commands

import (
    "fmt"
    "runtime"

    "github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Print version information",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("CodeAI %s\n", version)
        fmt.Printf("  Build time: %s\n", buildTime)
        fmt.Printf("  Commit:     %s\n", commit)
        fmt.Printf("  Go version: %s\n", runtime.Version())
        fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
    },
}

func init() {
    rootCmd.AddCommand(versionCmd)
}
```

### 8. Init Command (cmd/codeai/commands/init.go)

```go
package commands

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
    Use:   "init [name]",
    Short: "Initialize a new CodeAI project",
    Long: `Initialize a new CodeAI project with example files.

This command creates a new directory with a basic CodeAI
application structure including example entities and endpoints.

Example:
  codeai init my-app
  codeai init my-app --template api`,
    Args: cobra.MaximumNArgs(1),
    RunE: runInit,
}

func init() {
    rootCmd.AddCommand(initCmd)

    initCmd.Flags().String("template", "basic", "Project template (basic, api, full)")
}

func runInit(cmd *cobra.Command, args []string) error {
    name := "my-app"
    if len(args) > 0 {
        name = args[0]
    }

    template, _ := cmd.Flags().GetString("template")

    // Create directory
    if err := os.MkdirAll(name, 0755); err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }

    // Create app.cai
    appContent := generateAppTemplate(name, template)
    appPath := filepath.Join(name, "app.cai")
    if err := os.WriteFile(appPath, []byte(appContent), 0644); err != nil {
        return fmt.Errorf("failed to create app.cai: %w", err)
    }

    // Create .env.example
    envContent := generateEnvTemplate()
    envPath := filepath.Join(name, ".env.example")
    if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
        return fmt.Errorf("failed to create .env.example: %w", err)
    }

    fmt.Printf("✓ Created new CodeAI project: %s\n\n", name)
    fmt.Println("Next steps:")
    fmt.Printf("  1. cd %s\n", name)
    fmt.Println("  2. cp .env.example .env")
    fmt.Println("  3. Edit .env with your database credentials")
    fmt.Println("  4. codeai run .")

    return nil
}

func generateAppTemplate(name, template string) string {
    return fmt.Sprintf(`# %s
# Generated by CodeAI

config {
    name: "%s"
    version: "1.0.0"

    database: postgres {
        pool_size: 20
    }

    auth: jwt {
        issuer: env(JWT_ISSUER)
    }
}

entity User {
    description: "Application user"

    id: uuid, primary, auto
    email: string, required, unique
    name: string, required
    created_at: timestamp, auto
    updated_at: timestamp, auto_update
}

endpoint GET /users {
    description: "List all users"
    auth: required

    query {
        page: integer, default(1)
        limit: integer, default(20), max(100)
    }

    returns: paginated(User)
}

endpoint GET /users/{id} {
    description: "Get a user by ID"
    auth: required

    path {
        id: uuid, required
    }

    returns: User
    error 404: "User not found"
}

endpoint POST /users {
    description: "Create a new user"
    auth: required
    roles: [admin]

    body {
        email: string, required
        name: string, required
    }

    returns: User
}
`, name, name)
}

func generateEnvTemplate() string {
    return `# Database
DATABASE_URL=postgres://user:password@localhost:5432/myapp?sslmode=disable

# Authentication
JWT_ISSUER=https://auth.example.com
JWT_SECRET=your-secret-key

# Server
CODEAI_PORT=8080
CODEAI_HOST=0.0.0.0
CODEAI_LOG_LEVEL=info
`
}
```

## Acceptance Criteria
- [ ] `codeai run` starts the application
- [ ] `codeai validate` checks syntax and semantics
- [ ] `codeai migrate` manages database migrations
- [ ] `codeai openapi` generates OpenAPI spec
- [ ] `codeai init` creates new projects
- [ ] `codeai version` shows version info
- [ ] Graceful shutdown on SIGINT/SIGTERM
- [ ] Environment variable support
- [ ] Config file support (.yaml)
- [ ] Verbose mode for debugging

## Testing Strategy
- Command execution tests
- Flag parsing tests
- Error handling tests
- Integration tests with actual CodeAI files

## Files to Create/Modify
- `cmd/codeai/main.go`
- `cmd/codeai/commands/root.go`
- `cmd/codeai/commands/run.go`
- `cmd/codeai/commands/validate.go`
- `cmd/codeai/commands/migrate.go`
- `cmd/codeai/commands/openapi.go`
- `cmd/codeai/commands/version.go`
- `cmd/codeai/commands/init.go`

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/bargom/codeai/internal/api"
	"github.com/bargom/codeai/internal/api/handlers"
	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/codegen"
	"github.com/bargom/codeai/internal/database"
	"github.com/bargom/codeai/internal/database/repository"
	"github.com/bargom/codeai/internal/parser"
	"github.com/bargom/codeai/internal/validator"
	"github.com/spf13/cobra"
)

var (
	// serverPort is the port to listen on
	serverPort int
	// serverHost is the host to bind to
	serverHost string
	// dbType is the database type (postgres or mongodb)
	dbType string
	// dbHost is the database host (PostgreSQL)
	dbHost string
	// dbPort is the database port (PostgreSQL)
	dbPort int
	// dbName is the database name (PostgreSQL)
	dbName string
	// dbUser is the database user (PostgreSQL)
	dbUser string
	// dbPassword is the database password (PostgreSQL)
	dbPassword string
	// dbSSLMode is the database SSL mode (PostgreSQL)
	dbSSLMode string
	// mongodbURI is the MongoDB connection URI
	mongodbURI string
	// mongodbDatabase is the MongoDB database name
	mongodbDatabase string
	// caiFile is the path to the .cai configuration file
	caiFile string
	// migrateDryRun shows pending migrations without applying
	migrateDryRun bool
)

// newServerCmd creates the server command with subcommands.
func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Server management commands",
		Long:  `Commands for managing the CodeAI HTTP API server and database.`,
	}

	// Add subcommands
	cmd.AddCommand(newServerStartCmd())
	cmd.AddCommand(newServerMigrateCmd())

	return cmd
}

// newServerStartCmd creates the server start subcommand.
func newServerStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start HTTP API server",
		Long: `Start the CodeAI HTTP API server.

The server provides REST endpoints for managing deployments,
configurations, and executions.`,
		Example: `  codeai server start
  codeai server start --port 3000
  codeai server start --host 0.0.0.0 --port 8080`,
		RunE: runServerStart,
	}

	cmd.Flags().IntVarP(&serverPort, "port", "p", 8080, "port to listen on")
	cmd.Flags().StringVar(&serverHost, "host", "localhost", "host to bind to")
	cmd.Flags().StringVarP(&caiFile, "file", "f", "", "path to .cai file (auto-detects app.cai or *.cai in current dir)")
	cmd.Flags().StringVar(&dbType, "db-type", "", "database type (postgres or mongodb), overrides .cai config")
	// PostgreSQL flags
	cmd.Flags().StringVar(&dbHost, "db-host", "", "PostgreSQL host, overrides .cai config")
	cmd.Flags().IntVar(&dbPort, "db-port", 0, "PostgreSQL port, overrides .cai config")
	cmd.Flags().StringVar(&dbName, "db-name", "", "PostgreSQL database name, overrides .cai config")
	cmd.Flags().StringVar(&dbUser, "db-user", "", "PostgreSQL user, overrides .cai config")
	cmd.Flags().StringVar(&dbPassword, "db-password", "", "PostgreSQL password, overrides .cai config")
	cmd.Flags().StringVar(&dbSSLMode, "db-sslmode", "", "PostgreSQL SSL mode, overrides .cai config")
	// MongoDB flags
	cmd.Flags().StringVar(&mongodbURI, "mongodb-uri", "", "MongoDB connection URI, overrides .cai config")
	cmd.Flags().StringVar(&mongodbDatabase, "mongodb-database", "", "MongoDB database name, overrides .cai config")

	return cmd
}

func runServerStart(cmd *cobra.Command, args []string) error {
	addr := fmt.Sprintf("%s:%d", serverHost, serverPort)

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Starting server on %s\n", addr)
	}

	// Try to find and parse .cai file for config
	var configFromFile *ast.ConfigDecl
	var program *ast.Program
	caiFilePath := findCaiFile(caiFile)
	if caiFilePath != "" {
		var err error
		program, err = parser.ParseFile(caiFilePath)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", caiFilePath, err)
		}
		configFromFile = extractConfig(program)
		if configFromFile != nil && verbose {
			fmt.Fprintf(cmd.OutOrStdout(), "Loaded config from %s\n", caiFilePath)
		}
	}

	// Build database configuration - file config as base, CLI flags override
	dbConfig := buildDatabaseConfig(configFromFile)

	switch dbConfig.Type {
	case database.DatabaseTypeMongoDB:
		if verbose {
			fmt.Fprintf(cmd.OutOrStdout(), "Connecting to MongoDB at %s/%s\n", dbConfig.MongoDB.URI, dbConfig.MongoDB.Database)
		}
	default:
		if verbose {
			fmt.Fprintf(cmd.OutOrStdout(), "Connecting to PostgreSQL at %s:%d/%s\n", dbConfig.Postgres.Host, dbConfig.Postgres.Port, dbConfig.Postgres.Database)
		}
	}

	// Connect to database
	conn, err := database.NewConnection(dbConfig)
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Connected to %s\n", dbConfig.Type)

	// Create repositories (PostgreSQL only for now)
	var configRepo repository.ConfigRepo
	var deploymentRepo repository.DeploymentRepo
	var executionRepo repository.ExecutionRepo

	if pgConn, ok := conn.(*database.PostgresConnection); ok {
		configRepo = repository.NewConfigRepository(pgConn.DB)
		deploymentRepo = repository.NewDeploymentRepository(pgConn.DB)
		executionRepo = repository.NewExecutionRepository(pgConn.DB)
	} else {
		// For MongoDB, repositories are not yet implemented
		fmt.Fprintln(cmd.OutOrStdout(), "Note: MongoDB repositories not yet implemented, running in limited mode")
	}

	// Create handler with repositories
	handler := handlers.NewHandler(deploymentRepo, configRepo, executionRepo)

	// Create router - use codegen if program has endpoints
	var router http.Handler
	if program != nil && hasEndpoints(program) {
		// Validate AST first
		v := validator.New()
		if err := v.Validate(program); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		// Generate code from AST
		gen := codegen.NewGenerator(&codegen.Config{
			DatabaseURL: buildDatabaseURL(dbConfig),
		})

		generatedCode, err := gen.GenerateFromAST(program)
		if err != nil {
			return fmt.Errorf("code generation failed: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Generated %d endpoints from %s\n", generatedCode.EndpointCount, caiFilePath)
		router = generatedCode.Router
	} else {
		// Use default API router
		router = api.NewRouter(handler)
	}

	// Create server
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Handle graceful shutdown
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Fprintln(cmd.OutOrStdout(), "\nShutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Server forced to shutdown: %v\n", err)
		}

		close(done)
	}()

	fmt.Fprintf(cmd.OutOrStdout(), "Server listening on %s\n", addr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	<-done
	fmt.Fprintln(cmd.OutOrStdout(), "Server stopped")

	return nil
}

// newServerMigrateCmd creates the server migrate subcommand.
func newServerMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		Long: `Run database migrations to set up or update the schema.

Use --dry-run to see what migrations would be applied without
actually running them.`,
		Example: `  codeai server migrate
  codeai server migrate --dry-run
  codeai server migrate --db-host localhost --db-name codeai`,
		RunE: runServerMigrate,
	}

	cmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "show pending migrations without applying")
	cmd.Flags().StringVar(&dbHost, "db-host", "localhost", "database host")
	cmd.Flags().IntVar(&dbPort, "db-port", 5432, "database port")
	cmd.Flags().StringVar(&dbName, "db-name", "codeai", "database name")
	cmd.Flags().StringVar(&dbUser, "db-user", "postgres", "database user")
	cmd.Flags().StringVar(&dbPassword, "db-password", "", "database password")
	cmd.Flags().StringVar(&dbSSLMode, "db-sslmode", "disable", "database SSL mode")

	return cmd
}

func runServerMigrate(cmd *cobra.Command, args []string) error {
	if migrateDryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "Dry run: showing pending migrations")
		fmt.Fprintln(cmd.OutOrStdout(), "")
		fmt.Fprintln(cmd.OutOrStdout(), "Pending migrations:")
		fmt.Fprintln(cmd.OutOrStdout(), "  - 001_create_configs_table.sql")
		fmt.Fprintln(cmd.OutOrStdout(), "  - 002_create_deployments_table.sql")
		fmt.Fprintln(cmd.OutOrStdout(), "  - 003_create_executions_table.sql")
		fmt.Fprintln(cmd.OutOrStdout(), "")
		fmt.Fprintln(cmd.OutOrStdout(), "Use 'codeai server migrate' without --dry-run to apply")
		return nil
	}

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Connecting to database %s:%d/%s\n", dbHost, dbPort, dbName)
	}

	// Connect to database
	dbCfg := database.Config{
		Host:     dbHost,
		Port:     dbPort,
		Database: dbName,
		User:     dbUser,
		Password: dbPassword,
		SSLMode:  dbSSLMode,
	}

	db, err := database.Connect(dbCfg)
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer database.Close(db)

	// TODO: Implement actual migrations
	// For now, we'll create the tables directly

	migrations := []string{
		`CREATE TABLE IF NOT EXISTS configs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			ast_json JSONB,
			validation_errors JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS deployments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			config_id UUID REFERENCES configs(id),
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS executions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			deployment_id UUID REFERENCES deployments(id),
			command TEXT,
			output TEXT,
			exit_code INTEGER,
			started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			completed_at TIMESTAMP WITH TIME ZONE
		)`,
	}

	for i, migration := range migrations {
		if verbose {
			fmt.Fprintf(cmd.OutOrStdout(), "Running migration %d...\n", i+1)
		}

		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Migrations completed successfully")

	return nil
}

// findCaiFile finds a .cai file to use for configuration.
// Priority: explicit path > app.cai > first *.cai in current dir
func findCaiFile(explicit string) string {
	if explicit != "" {
		if _, err := os.Stat(explicit); err == nil {
			return explicit
		}
		return ""
	}

	// Try app.cai first
	if _, err := os.Stat("app.cai"); err == nil {
		return "app.cai"
	}

	// Try to find any .cai file
	matches, err := filepath.Glob("*.cai")
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

// extractConfig extracts the ConfigDecl from a parsed program.
func extractConfig(program *ast.Program) *ast.ConfigDecl {
	if program == nil {
		return nil
	}
	for _, stmt := range program.Statements {
		if cfg, ok := stmt.(*ast.ConfigDecl); ok {
			return cfg
		}
	}
	return nil
}

// buildDatabaseConfig builds a DatabaseConfig from file config and CLI flags.
// CLI flags override file config values.
func buildDatabaseConfig(fileConfig *ast.ConfigDecl) database.DatabaseConfig {
	// Start with defaults
	cfg := database.DatabaseConfig{
		Type: database.DatabaseTypePostgres,
		Postgres: &database.PostgresConfig{
			Host:    "localhost",
			Port:    5432,
			SSLMode: "disable",
		},
		MongoDB: &database.MongoDBConfig{
			URI:      "mongodb://localhost:27017",
			Database: "codeai",
		},
	}

	// Apply file config if present
	if fileConfig != nil {
		if fileConfig.DatabaseType != "" {
			cfg.Type = database.ParseDatabaseType(string(fileConfig.DatabaseType))
		}
		if fileConfig.MongoDBURI != "" {
			cfg.MongoDB.URI = fileConfig.MongoDBURI
		}
		if fileConfig.MongoDBName != "" {
			cfg.MongoDB.Database = fileConfig.MongoDBName
		}
	}

	// CLI flags override file config
	if dbType != "" {
		cfg.Type = database.ParseDatabaseType(dbType)
	}

	// PostgreSQL overrides
	if dbHost != "" {
		cfg.Postgres.Host = dbHost
	}
	if dbPort != 0 {
		cfg.Postgres.Port = dbPort
	}
	if dbName != "" {
		cfg.Postgres.Database = dbName
	}
	if dbUser != "" {
		cfg.Postgres.User = dbUser
	}
	if dbPassword != "" {
		cfg.Postgres.Password = dbPassword
	}
	if dbSSLMode != "" {
		cfg.Postgres.SSLMode = dbSSLMode
	}

	// MongoDB overrides
	if mongodbURI != "" {
		cfg.MongoDB.URI = mongodbURI
	}
	if mongodbDatabase != "" {
		cfg.MongoDB.Database = mongodbDatabase
	}

	return cfg
}

// hasEndpoints checks if the program has endpoint declarations.
func hasEndpoints(program *ast.Program) bool {
	for _, stmt := range program.Statements {
		if _, ok := stmt.(*ast.EndpointDecl); ok {
			return true
		}
	}
	return false
}

// buildDatabaseURL constructs a database URL from config.
func buildDatabaseURL(cfg database.DatabaseConfig) string {
	switch cfg.Type {
	case database.DatabaseTypeMongoDB:
		return cfg.MongoDB.URI
	default:
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Postgres.User,
			cfg.Postgres.Password,
			cfg.Postgres.Host,
			cfg.Postgres.Port,
			cfg.Postgres.Database,
			cfg.Postgres.SSLMode,
		)
	}
}

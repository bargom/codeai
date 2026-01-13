package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bargom/codeai/internal/api"
	"github.com/bargom/codeai/internal/api/handlers"
	"github.com/bargom/codeai/internal/database"
	"github.com/bargom/codeai/internal/database/repository"
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
	cmd.Flags().StringVar(&dbType, "db-type", "postgres", "database type (postgres or mongodb)")
	// PostgreSQL flags
	cmd.Flags().StringVar(&dbHost, "db-host", "localhost", "PostgreSQL host")
	cmd.Flags().IntVar(&dbPort, "db-port", 5432, "PostgreSQL port")
	cmd.Flags().StringVar(&dbName, "db-name", "codeai", "PostgreSQL database name")
	cmd.Flags().StringVar(&dbUser, "db-user", "postgres", "PostgreSQL user")
	cmd.Flags().StringVar(&dbPassword, "db-password", "", "PostgreSQL password")
	cmd.Flags().StringVar(&dbSSLMode, "db-sslmode", "disable", "PostgreSQL SSL mode")
	// MongoDB flags
	cmd.Flags().StringVar(&mongodbURI, "mongodb-uri", "mongodb://localhost:27017", "MongoDB connection URI")
	cmd.Flags().StringVar(&mongodbDatabase, "mongodb-database", "codeai", "MongoDB database name")

	return cmd
}

func runServerStart(cmd *cobra.Command, args []string) error {
	addr := fmt.Sprintf("%s:%d", serverHost, serverPort)

	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Starting server on %s\n", addr)
	}

	// Build database configuration based on type
	dbConfig := database.DatabaseConfig{
		Type: database.ParseDatabaseType(dbType),
	}

	switch dbConfig.Type {
	case database.DatabaseTypeMongoDB:
		dbConfig.MongoDB = &database.MongoDBConfig{
			URI:      mongodbURI,
			Database: mongodbDatabase,
		}
		if verbose {
			fmt.Fprintf(cmd.OutOrStdout(), "Connecting to MongoDB at %s/%s\n", mongodbURI, mongodbDatabase)
		}
	default:
		dbConfig.Postgres = &database.PostgresConfig{
			Host:     dbHost,
			Port:     dbPort,
			Database: dbName,
			User:     dbUser,
			Password: dbPassword,
			SSLMode:  dbSSLMode,
		}
		if verbose {
			fmt.Fprintf(cmd.OutOrStdout(), "Connecting to PostgreSQL at %s:%d/%s\n", dbHost, dbPort, dbName)
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

	// Create router
	router := api.NewRouter(handler)

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

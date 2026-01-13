// Package codegen provides runtime code generation from parsed AST.
// It generates HTTP handlers, middleware chains, and integration clients
// at server startup from .cai specification files.
package codegen

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bargom/codeai/internal/auth"
	"github.com/bargom/codeai/internal/event"
	"github.com/bargom/codeai/internal/integration"
	"github.com/bargom/codeai/internal/workflow"
)

// GeneratedCode holds all the runtime artifacts generated from AST.
type GeneratedCode struct {
	// Router is the configured Chi router with all endpoints
	Router chi.Router

	// Middlewares are named middleware functions for reuse
	Middlewares map[string]func(http.Handler) http.Handler

	// Integrations holds external API clients
	Integrations *integration.IntegrationRegistry

	// Workflows holds Temporal workflow configurations
	Workflows *workflow.DSLWorkflowRegistry

	// EventHandlers holds the event registry with handlers
	EventHandlers *event.EventRegistry

	// AuthLoader holds authentication and authorization configuration
	AuthLoader *auth.DSLLoader

	// EndpointCount tracks the number of generated endpoints
	EndpointCount int

	// ModelRegistry holds model/collection type information
	ModelRegistry *TypeRegistry
}

// TypeRegistry holds type information for models and collections.
type TypeRegistry struct {
	// Models holds PostgreSQL model definitions
	Models map[string]*ModelInfo

	// Collections holds MongoDB collection definitions
	Collections map[string]*CollectionInfo
}

// NewTypeRegistry creates a new type registry.
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		Models:      make(map[string]*ModelInfo),
		Collections: make(map[string]*CollectionInfo),
	}
}

// ModelInfo holds metadata about a PostgreSQL model.
type ModelInfo struct {
	Name        string
	Description string
	Fields      []FieldInfo
	Indexes     []IndexInfo
}

// CollectionInfo holds metadata about a MongoDB collection.
type CollectionInfo struct {
	Name        string
	Description string
	Fields      []FieldInfo
	Indexes     []IndexInfo
}

// FieldInfo holds metadata about a field.
type FieldInfo struct {
	Name      string
	FieldType string
	Required  bool
	Unique    bool
	Primary   bool
	Default   interface{}
}

// IndexInfo holds metadata about an index.
type IndexInfo struct {
	Fields []string
	Unique bool
	Kind   string // "", "text", "geospatial"
}

// Config holds configuration for the code generator.
type Config struct {
	// DatabaseURL is the PostgreSQL connection string
	DatabaseURL string

	// RedisURL is the Redis connection string for caching
	RedisURL string

	// TemporalHost is the Temporal server address
	TemporalHost string

	// EnableMetrics enables Prometheus metrics
	EnableMetrics bool

	// EnableTracing enables distributed tracing
	EnableTracing bool

	// LogLevel sets the logging verbosity
	LogLevel string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		DatabaseURL:   "postgres://localhost:5432/codeai",
		RedisURL:      "redis://localhost:6379",
		TemporalHost:  "localhost:7233",
		EnableMetrics: true,
		EnableTracing: false,
		LogLevel:      "info",
	}
}

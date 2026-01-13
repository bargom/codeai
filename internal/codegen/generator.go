package codegen

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/auth"
	"github.com/bargom/codeai/internal/event"
	"github.com/bargom/codeai/internal/integration"
	"github.com/bargom/codeai/internal/workflow"
)

// Generator is the main code generator interface.
type Generator interface {
	// GenerateFromAST generates runtime code from a parsed AST program.
	GenerateFromAST(program *ast.Program) (*GeneratedCode, error)
}

// generator implements the Generator interface.
type generator struct {
	config *Config
	logger *slog.Logger
}

// NewGenerator creates a new code generator with the given configuration.
func NewGenerator(config *Config) Generator {
	if config == nil {
		config = DefaultConfig()
	}
	return &generator{
		config: config,
		logger: slog.Default(),
	}
}

// GenerateFromAST generates all runtime artifacts from the parsed AST.
func (g *generator) GenerateFromAST(program *ast.Program) (*GeneratedCode, error) {
	if program == nil {
		return nil, fmt.Errorf("program is nil")
	}

	g.logger.Info("starting code generation from AST")

	// Initialize registries
	code := &GeneratedCode{
		Middlewares:   make(map[string]func(http.Handler) http.Handler),
		Integrations:  integration.NewIntegrationRegistry(),
		Workflows:     workflow.NewDSLWorkflowRegistry(),
		EventHandlers: event.NewEventRegistry(nil),
		AuthLoader:    auth.NewDSLLoader(),
		ModelRegistry: NewTypeRegistry(),
	}

	// First pass: load configurations (auth, middleware, models, etc.)
	if err := g.loadConfigurations(program, code); err != nil {
		return nil, fmt.Errorf("loading configurations: %w", err)
	}

	// Second pass: load integrations
	if err := g.loadIntegrations(program, code); err != nil {
		return nil, fmt.Errorf("loading integrations: %w", err)
	}

	// Third pass: load workflows
	if err := g.loadWorkflows(program, code); err != nil {
		return nil, fmt.Errorf("loading workflows: %w", err)
	}

	// Fourth pass: load events and handlers
	if err := g.loadEvents(program, code); err != nil {
		return nil, fmt.Errorf("loading events: %w", err)
	}

	// Fifth pass: generate router with endpoints
	router, endpointCount, err := g.generateRouter(program, code)
	if err != nil {
		return nil, fmt.Errorf("generating router: %w", err)
	}
	code.Router = router
	code.EndpointCount = endpointCount

	g.logger.Info("code generation complete",
		"endpoints", endpointCount,
		"integrations", code.Integrations.Count(),
		"workflows", len(code.Workflows.List()),
		"events", code.EventHandlers.EventCount(),
	)

	return code, nil
}

// loadConfigurations loads auth, roles, middlewares, and models from AST.
func (g *generator) loadConfigurations(program *ast.Program, code *GeneratedCode) error {
	// Load auth, roles, and middleware configurations
	if err := code.AuthLoader.LoadProgram(program); err != nil {
		return fmt.Errorf("loading auth configurations: %w", err)
	}

	// Load models and collections
	for _, stmt := range program.Statements {
		switch decl := stmt.(type) {
		case *ast.DatabaseBlock:
			if err := g.loadDatabaseBlock(decl, code); err != nil {
				return err
			}
		case *ast.ModelDecl:
			g.loadModel(decl, code)
		case *ast.CollectionDecl:
			g.loadCollection(decl, code)
		}
	}

	return nil
}

// loadDatabaseBlock processes a database block declaration.
func (g *generator) loadDatabaseBlock(block *ast.DatabaseBlock, code *GeneratedCode) error {
	for _, stmt := range block.Statements {
		switch decl := stmt.(type) {
		case *ast.ModelDecl:
			g.loadModel(decl, code)
		case *ast.CollectionDecl:
			g.loadCollection(decl, code)
		}
	}
	return nil
}

// loadModel registers a PostgreSQL model in the type registry.
func (g *generator) loadModel(model *ast.ModelDecl, code *GeneratedCode) {
	info := &ModelInfo{
		Name:        model.Name,
		Description: model.Description,
		Fields:      make([]FieldInfo, 0, len(model.Fields)),
		Indexes:     make([]IndexInfo, 0, len(model.Indexes)),
	}

	for _, field := range model.Fields {
		fieldInfo := FieldInfo{
			Name:      field.Name,
			FieldType: field.FieldType.String(),
		}

		// Extract modifiers
		for _, mod := range field.Modifiers {
			switch mod.Name {
			case "required":
				fieldInfo.Required = true
			case "unique":
				fieldInfo.Unique = true
			case "primary":
				fieldInfo.Primary = true
			case "default":
				if mod.Value != nil {
					fieldInfo.Default = extractExprValue(mod.Value)
				}
			}
		}

		info.Fields = append(info.Fields, fieldInfo)
	}

	for _, idx := range model.Indexes {
		info.Indexes = append(info.Indexes, IndexInfo{
			Fields: idx.Fields,
			Unique: idx.Unique,
		})
	}

	code.ModelRegistry.Models[model.Name] = info
}

// loadCollection registers a MongoDB collection in the type registry.
func (g *generator) loadCollection(coll *ast.CollectionDecl, code *GeneratedCode) {
	info := &CollectionInfo{
		Name:        coll.Name,
		Description: coll.Description,
		Fields:      make([]FieldInfo, 0, len(coll.Fields)),
		Indexes:     make([]IndexInfo, 0, len(coll.Indexes)),
	}

	for _, field := range coll.Fields {
		fieldInfo := FieldInfo{
			Name:      field.Name,
			FieldType: field.FieldType.String(),
		}

		for _, mod := range field.Modifiers {
			switch mod.Name {
			case "required":
				fieldInfo.Required = true
			case "unique":
				fieldInfo.Unique = true
			case "primary":
				fieldInfo.Primary = true
			case "default":
				if mod.Value != nil {
					fieldInfo.Default = extractExprValue(mod.Value)
				}
			}
		}

		info.Fields = append(info.Fields, fieldInfo)
	}

	for _, idx := range coll.Indexes {
		info.Indexes = append(info.Indexes, IndexInfo{
			Fields: idx.Fields,
			Unique: idx.Unique,
			Kind:   idx.IndexKind,
		})
	}

	code.ModelRegistry.Collections[coll.Name] = info
}

// loadIntegrations loads external API integration configurations.
func (g *generator) loadIntegrations(program *ast.Program, code *GeneratedCode) error {
	for _, stmt := range program.Statements {
		if intg, ok := stmt.(*ast.IntegrationDecl); ok {
			if _, err := code.Integrations.LoadIntegrationFromAST(intg); err != nil {
				return fmt.Errorf("loading integration %q: %w", intg.Name, err)
			}
			g.logger.Debug("loaded integration", "name", intg.Name, "type", intg.IntgType)
		}
	}
	return nil
}

// loadWorkflows loads Temporal workflow configurations.
func (g *generator) loadWorkflows(program *ast.Program, code *GeneratedCode) error {
	var workflows []*ast.WorkflowDecl
	for _, stmt := range program.Statements {
		if wf, ok := stmt.(*ast.WorkflowDecl); ok {
			workflows = append(workflows, wf)
		}
	}

	if len(workflows) > 0 {
		if err := code.Workflows.LoadWorkflows(workflows); err != nil {
			return err
		}
		g.logger.Debug("loaded workflows", "count", len(workflows))
	}

	return nil
}

// loadEvents loads event definitions and handlers.
func (g *generator) loadEvents(program *ast.Program, code *GeneratedCode) error {
	for _, stmt := range program.Statements {
		switch decl := stmt.(type) {
		case *ast.EventDecl:
			if err := code.EventHandlers.RegisterEventFromAST(decl); err != nil {
				return fmt.Errorf("registering event %q: %w", decl.Name, err)
			}
			g.logger.Debug("registered event", "name", decl.Name)
		case *ast.EventHandlerDecl:
			if err := code.EventHandlers.SubscribeHandlerFromAST(decl); err != nil {
				return fmt.Errorf("subscribing handler for %q: %w", decl.EventName, err)
			}
			g.logger.Debug("subscribed handler", "event", decl.EventName, "target", decl.Target)
		}
	}
	return nil
}

// generateRouter creates a Chi router with all endpoint handlers.
func (g *generator) generateRouter(program *ast.Program, code *GeneratedCode) (chi.Router, int, error) {
	r := chi.NewRouter()

	// Add base middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * 1000000000)) // 60 seconds

	// Create execution context factory
	execCtxFactory := NewExecutionContextFactory(code)
	execCtxFactory.dbConnection = g.config.DBConnection

	// Generate endpoint handlers
	endpointCount := 0
	for _, stmt := range program.Statements {
		if ep, ok := stmt.(*ast.EndpointDecl); ok {
			if err := g.registerEndpoint(r, ep, code, execCtxFactory); err != nil {
				return nil, 0, fmt.Errorf("registering endpoint %s %s: %w", ep.Method, ep.Path, err)
			}
			endpointCount++
		}
	}

	return r, endpointCount, nil
}

// registerEndpoint registers a single endpoint in the router.
func (g *generator) registerEndpoint(r chi.Router, ep *ast.EndpointDecl, code *GeneratedCode, factory *ExecutionContextFactory) error {
	// Build middleware chain for this endpoint
	middlewareChain := g.buildMiddlewareChain(ep.Middlewares, code)

	// Generate the handler
	handler := GenerateEndpointHandler(ep, factory)

	// Wrap handler with middleware
	var finalHandler http.Handler = handler
	for i := len(middlewareChain) - 1; i >= 0; i-- {
		finalHandler = middlewareChain[i](finalHandler)
	}

	// Convert path from :param to {param} for chi
	chiPath := convertPathToChi(ep.Path)

	// Register route
	switch ep.Method {
	case ast.HTTPMethodGET:
		r.Get(chiPath, finalHandler.ServeHTTP)
	case ast.HTTPMethodPOST:
		r.Post(chiPath, finalHandler.ServeHTTP)
	case ast.HTTPMethodPUT:
		r.Put(chiPath, finalHandler.ServeHTTP)
	case ast.HTTPMethodDELETE:
		r.Delete(chiPath, finalHandler.ServeHTTP)
	case ast.HTTPMethodPATCH:
		r.Patch(chiPath, finalHandler.ServeHTTP)
	default:
		return fmt.Errorf("unsupported HTTP method: %s", ep.Method)
	}

	g.logger.Debug("registered endpoint", "method", ep.Method, "path", chiPath)
	return nil
}

// buildMiddlewareChain builds the middleware chain for an endpoint.
func (g *generator) buildMiddlewareChain(refs []*ast.MiddlewareRef, code *GeneratedCode) []func(http.Handler) http.Handler {
	chain := make([]func(http.Handler) http.Handler, 0, len(refs))

	for _, ref := range refs {
		// Check if we have a pre-generated middleware
		if mw, ok := code.Middlewares[ref.Name]; ok {
			chain = append(chain, mw)
			continue
		}

		// Try to create middleware from auth loader
		loadedMW, ok := code.AuthLoader.GetMiddleware(ref.Name)
		if !ok {
			g.logger.Warn("unknown middleware", "name", ref.Name)
			continue
		}

		// Generate middleware based on type
		mw := generateMiddlewareFunc(loadedMW, code.AuthLoader)
		if mw != nil {
			chain = append(chain, mw)
			code.Middlewares[ref.Name] = mw
		}
	}

	return chain
}

// convertPathToChi converts :param style paths to {param} for chi router.
func convertPathToChi(path string) string {
	result := make([]byte, 0, len(path))
	i := 0
	for i < len(path) {
		if path[i] == ':' {
			// Start of param
			result = append(result, '{')
			i++
			// Read param name
			for i < len(path) && path[i] != '/' {
				result = append(result, path[i])
				i++
			}
			result = append(result, '}')
		} else {
			result = append(result, path[i])
			i++
		}
	}
	return string(result)
}

// extractExprValue extracts a Go value from an AST expression.
func extractExprValue(expr ast.Expression) interface{} {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return e.Value
	case *ast.NumberLiteral:
		return e.Value
	case *ast.BoolLiteral:
		return e.Value
	case *ast.Identifier:
		return e.Name
	default:
		return nil
	}
}

// generateMiddlewareFunc creates an HTTP middleware from loaded configuration.
func generateMiddlewareFunc(mw *auth.LoadedMiddleware, authLoader *auth.DSLLoader) func(http.Handler) http.Handler {
	switch mw.MiddlewareType {
	case "authentication":
		return generateAuthMiddlewareFunc(mw, authLoader)
	case "rate_limiting":
		return generateRateLimitMiddleware(mw)
	case "cors":
		return generateCORSMiddleware(mw)
	case "logging":
		return generateLoggingMiddleware(mw)
	default:
		return nil
	}
}

// generateAuthMiddlewareFunc creates JWT authentication middleware.
func generateAuthMiddlewareFunc(mw *auth.LoadedMiddleware, authLoader *auth.DSLLoader) func(http.Handler) http.Handler {
	authMW, err := authLoader.CreateMiddlewareChain(mw.Name)
	if err != nil {
		slog.Warn("failed to create auth middleware", "name", mw.Name, "error", err)
		return nil
	}

	// Determine auth requirement from config
	requirement := auth.AuthRequired
	if !mw.Required {
		requirement = auth.AuthOptional
	}

	return authMW.Authenticate(requirement)
}

// generateRateLimitMiddleware creates rate limiting middleware.
func generateRateLimitMiddleware(mw *auth.LoadedMiddleware) func(http.Handler) http.Handler {
	// Extract rate limit config
	limit := 100 // default
	if v, ok := mw.Config["limit"].(float64); ok {
		limit = int(v)
	}
	window := 60 // default: 60 seconds
	if v, ok := mw.Config["window"].(float64); ok {
		window = int(v)
	}

	_ = limit  // TODO: Implement actual rate limiting
	_ = window // TODO: Implement actual rate limiting

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Placeholder: actual rate limiting would use Redis
			next.ServeHTTP(w, r)
		})
	}
}

// generateCORSMiddleware creates CORS middleware.
func generateCORSMiddleware(mw *auth.LoadedMiddleware) func(http.Handler) http.Handler {
	origins := "*"
	if v, ok := mw.Config["origins"].(string); ok {
		origins = v
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origins)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// generateLoggingMiddleware creates request logging middleware.
func generateLoggingMiddleware(mw *auth.LoadedMiddleware) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote", r.RemoteAddr,
			)
			next.ServeHTTP(w, r)
		})
	}
}

package codegen

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/bargom/codeai/internal/ast"
)

// GenerateEndpointHandler generates an HTTP handler from an endpoint declaration.
func GenerateEndpointHandler(ep *ast.EndpointDecl, factory *ExecutionContextFactory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Create execution context
		execCtx := factory.NewContext(ctx, r)

		// 1. Extract request data based on source
		if ep.Handler != nil && ep.Handler.Request != nil {
			requestData, err := extractRequestData(r, ep.Handler.Request)
			if err != nil {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
				return
			}
			execCtx.SetInput(requestData)
		}

		// 2. Execute logic steps
		if ep.Handler != nil && ep.Handler.Logic != nil {
			result, err := executeLogicSteps(execCtx, ep.Handler.Logic.Steps)
			if err != nil {
				// Determine appropriate status code based on error type
				statusCode := determineErrorStatusCode(err)
				writeError(w, statusCode, err.Error())
				return
			}
			execCtx.SetResult(result)
		}

		// 3. Write response
		statusCode := http.StatusOK
		if ep.Handler != nil && ep.Handler.Response != nil {
			statusCode = ep.Handler.Response.StatusCode
		}

		result := execCtx.Result()
		if result == nil {
			result = map[string]interface{}{"status": "ok"}
		}

		writeJSON(w, statusCode, result)
	}
}

// extractRequestData extracts request data based on the source specification.
func extractRequestData(r *http.Request, req *ast.RequestType) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	switch req.Source {
	case ast.RequestSourceBody:
		if r.Body != nil {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, fmt.Errorf("reading request body: %w", err)
			}
			if len(body) > 0 {
				if err := json.Unmarshal(body, &data); err != nil {
					return nil, fmt.Errorf("parsing JSON body: %w", err)
				}
			}
		}

	case ast.RequestSourceQuery:
		for key, values := range r.URL.Query() {
			if len(values) == 1 {
				data[key] = values[0]
			} else {
				data[key] = values
			}
		}

	case ast.RequestSourcePath:
		// Extract path parameters from chi router context
		rctx := chi.RouteContext(r.Context())
		if rctx != nil {
			for i, key := range rctx.URLParams.Keys {
				data[key] = rctx.URLParams.Values[i]
			}
		}

	case ast.RequestSourceHeader:
		for key, values := range r.Header {
			if len(values) == 1 {
				data[key] = values[0]
			} else {
				data[key] = values
			}
		}
	}

	// Also extract path params for all sources (common need)
	if req.Source != ast.RequestSourcePath {
		rctx := chi.RouteContext(r.Context())
		if rctx != nil {
			for i, key := range rctx.URLParams.Keys {
				if _, exists := data[key]; !exists {
					data[key] = rctx.URLParams.Values[i]
				}
			}
		}
	}

	return data, nil
}

// executeLogicSteps executes the handler logic steps sequentially.
func executeLogicSteps(ctx *ExecutionContext, steps []*ast.LogicStep) (interface{}, error) {
	for _, step := range steps {
		if err := executeLogicStep(ctx, step); err != nil {
			return nil, err
		}
	}
	return ctx.Result(), nil
}

// executeLogicStep executes a single logic step.
func executeLogicStep(ctx *ExecutionContext, step *ast.LogicStep) error {
	logger := slog.Default()
	logger.Debug("executing step", "action", step.Action, "target", step.Target)

	switch step.Action {
	case "validate":
		return executeValidate(ctx, step)

	case "authorize":
		return executeAuthorize(ctx, step)

	case "db.find", "find":
		return executeDBFind(ctx, step)

	case "db.findOne", "findOne":
		return executeDBFindOne(ctx, step)

	case "db.insert", "insert":
		return executeDBInsert(ctx, step)

	case "db.update", "update":
		return executeDBUpdate(ctx, step)

	case "db.delete", "delete":
		return executeDBDelete(ctx, step)

	case "transform":
		return executeTransform(ctx, step)

	case "emit":
		return executeEmit(ctx, step)

	case "call":
		return executeIntegrationCall(ctx, step)

	case "cache.get":
		return executeCacheGet(ctx, step)

	case "cache.set":
		return executeCacheSet(ctx, step)

	default:
		// Unknown action - might be a custom function
		logger.Warn("unknown action", "action", step.Action)
		return nil
	}
}

// executeValidate validates input data.
func executeValidate(ctx *ExecutionContext, step *ast.LogicStep) error {
	// Get the target to validate
	var target interface{}
	if len(step.Args) > 0 {
		argName := step.Args[0]
		if argName == "request" || argName == "input" {
			target = ctx.Input()
		} else {
			target = ctx.Get(argName)
		}
	} else {
		target = ctx.Input()
	}

	if target == nil {
		return &ValidationError{Field: "request", Message: "no data to validate"}
	}

	// For now, just verify the data exists
	// TODO: Add schema-based validation
	return nil
}

// executeAuthorize checks authorization.
func executeAuthorize(ctx *ExecutionContext, step *ast.LogicStep) error {
	// Extract required role/permission from args
	if len(step.Args) < 2 {
		return nil // No specific role required
	}

	requiredRole := step.Args[1]
	claims := ctx.Claims()

	if claims == nil {
		return &AuthorizationError{Message: "no authentication claims found"}
	}

	// Check if user has the required role
	userRoles, ok := claims["roles"].([]interface{})
	if !ok {
		// Try single role
		if role, ok := claims["role"].(string); ok {
			if role == requiredRole {
				return nil
			}
		}
		return &AuthorizationError{
			Message:      "insufficient permissions",
			RequiredRole: requiredRole,
		}
	}

	for _, role := range userRoles {
		if roleStr, ok := role.(string); ok && roleStr == requiredRole {
			return nil
		}
	}

	return &AuthorizationError{
		Message:      "insufficient permissions",
		RequiredRole: requiredRole,
	}
}

// executeDBFind executes a database query returning multiple results.
func executeDBFind(ctx *ExecutionContext, step *ast.LogicStep) error {
	tableName := ""
	if len(step.Args) > 0 {
		tableName = step.Args[0]
	}

	// Build query from options/conditions
	conditions := make(map[string]interface{})
	if step.Condition != "" {
		// Parse simple "field = value" conditions
		conditions = parseCondition(step.Condition, ctx)
	}

	// Execute query through execution context
	result, err := ctx.QueryDatabase(tableName, conditions)
	if err != nil {
		return fmt.Errorf("database query failed: %w", err)
	}

	// Store result
	if step.Target != "" {
		ctx.Set(step.Target, result)
	} else {
		ctx.SetResult(result)
	}

	return nil
}

// executeDBFindOne executes a database query returning a single result.
func executeDBFindOne(ctx *ExecutionContext, step *ast.LogicStep) error {
	tableName := ""
	if len(step.Args) > 0 {
		tableName = step.Args[0]
	}

	// Get ID or conditions
	var id interface{}
	if len(step.Args) > 1 {
		idArg := step.Args[1]
		// Check if it's a reference to input
		if strings.HasPrefix(idArg, "request.") || strings.HasPrefix(idArg, "input.") {
			field := strings.TrimPrefix(strings.TrimPrefix(idArg, "request."), "input.")
			if input, ok := ctx.Input().(map[string]interface{}); ok {
				id = input[field]
			}
		} else {
			id = ctx.Get(idArg)
			if id == nil {
				id = idArg
			}
		}
	}

	// Execute query
	result, err := ctx.FindOne(tableName, id)
	if err != nil {
		return fmt.Errorf("database query failed: %w", err)
	}

	if result == nil {
		return &NotFoundError{Resource: tableName}
	}

	// Store result
	if step.Target != "" {
		ctx.Set(step.Target, result)
	} else {
		ctx.SetResult(result)
	}

	return nil
}

// executeDBInsert executes a database insert.
func executeDBInsert(ctx *ExecutionContext, step *ast.LogicStep) error {
	tableName := ""
	if len(step.Args) > 0 {
		tableName = step.Args[0]
	}

	// Get data to insert
	var data interface{}
	if len(step.Args) > 1 {
		dataArg := step.Args[1]
		if dataArg == "request" || dataArg == "input" {
			data = ctx.Input()
		} else {
			data = ctx.Get(dataArg)
		}
	} else {
		data = ctx.Input()
	}

	// Execute insert
	id, err := ctx.InsertDatabase(tableName, data)
	if err != nil {
		return fmt.Errorf("database insert failed: %w", err)
	}

	// Store result
	result := map[string]interface{}{"id": id}
	if step.Target != "" {
		ctx.Set(step.Target, result)
	} else {
		ctx.SetResult(result)
	}

	return nil
}

// executeDBUpdate executes a database update.
func executeDBUpdate(ctx *ExecutionContext, step *ast.LogicStep) error {
	tableName := ""
	if len(step.Args) > 0 {
		tableName = step.Args[0]
	}

	// Get ID and data
	var id, data interface{}
	if len(step.Args) > 1 {
		idArg := step.Args[1]
		if strings.HasPrefix(idArg, "request.") {
			field := strings.TrimPrefix(idArg, "request.")
			if input, ok := ctx.Input().(map[string]interface{}); ok {
				id = input[field]
			}
		} else {
			id = ctx.Get(idArg)
			if id == nil {
				id = idArg
			}
		}
	}

	if len(step.Args) > 2 {
		dataArg := step.Args[2]
		if dataArg == "request" || dataArg == "input" {
			data = ctx.Input()
		} else {
			data = ctx.Get(dataArg)
		}
	} else {
		data = ctx.Input()
	}

	// Execute update
	err := ctx.UpdateDatabase(tableName, id, data)
	if err != nil {
		return fmt.Errorf("database update failed: %w", err)
	}

	// Store success result
	if step.Target != "" {
		ctx.Set(step.Target, map[string]interface{}{"updated": true})
	}

	return nil
}

// executeDBDelete executes a database delete.
func executeDBDelete(ctx *ExecutionContext, step *ast.LogicStep) error {
	tableName := ""
	if len(step.Args) > 0 {
		tableName = step.Args[0]
	}

	// Get ID
	var id interface{}
	if len(step.Args) > 1 {
		idArg := step.Args[1]
		if strings.HasPrefix(idArg, "request.") {
			field := strings.TrimPrefix(idArg, "request.")
			if input, ok := ctx.Input().(map[string]interface{}); ok {
				id = input[field]
			}
		} else {
			id = ctx.Get(idArg)
			if id == nil {
				id = idArg
			}
		}
	}

	// Execute delete
	err := ctx.DeleteDatabase(tableName, id)
	if err != nil {
		return fmt.Errorf("database delete failed: %w", err)
	}

	// Store success result
	if step.Target != "" {
		ctx.Set(step.Target, map[string]interface{}{"deleted": true})
	}

	return nil
}

// executeTransform applies a transformation to data.
func executeTransform(ctx *ExecutionContext, step *ast.LogicStep) error {
	transformName := ""
	if len(step.Args) > 0 {
		transformName = step.Args[0]
	}

	// Get data to transform
	var data interface{}
	if len(step.Args) > 1 {
		dataArg := step.Args[1]
		data = ctx.Get(dataArg)
	} else {
		data = ctx.Result()
	}

	// Apply transformation
	result, err := ctx.Transform(transformName, data)
	if err != nil {
		return fmt.Errorf("transformation failed: %w", err)
	}

	// Store result
	if step.Target != "" {
		ctx.Set(step.Target, result)
	} else {
		ctx.SetResult(result)
	}

	return nil
}

// executeEmit emits an event.
func executeEmit(ctx *ExecutionContext, step *ast.LogicStep) error {
	eventName := ""
	if len(step.Args) > 0 {
		eventName = step.Args[0]
	}

	// Get payload
	var payload interface{}
	if len(step.Args) > 1 {
		payloadArg := step.Args[1]
		payload = ctx.Get(payloadArg)
	} else {
		payload = ctx.Result()
	}

	// Emit event
	return ctx.EmitEvent(eventName, payload)
}

// executeIntegrationCall calls an external integration.
func executeIntegrationCall(ctx *ExecutionContext, step *ast.LogicStep) error {
	integrationName := ""
	if len(step.Args) > 0 {
		integrationName = step.Args[0]
	}

	method := "GET"
	if len(step.Args) > 1 {
		method = step.Args[1]
	}

	path := "/"
	if len(step.Args) > 2 {
		path = step.Args[2]
	}

	// Get request body if POST/PUT
	var body interface{}
	if len(step.Args) > 3 {
		bodyArg := step.Args[3]
		body = ctx.Get(bodyArg)
	}

	// Call integration
	result, err := ctx.CallIntegration(integrationName, method, path, body)
	if err != nil {
		return fmt.Errorf("integration call failed: %w", err)
	}

	// Store result
	if step.Target != "" {
		ctx.Set(step.Target, result)
	} else {
		ctx.SetResult(result)
	}

	return nil
}

// executeCacheGet retrieves a value from cache.
func executeCacheGet(ctx *ExecutionContext, step *ast.LogicStep) error {
	key := ""
	if len(step.Args) > 0 {
		key = step.Args[0]
	}

	value, err := ctx.CacheGet(key)
	if err != nil {
		return fmt.Errorf("cache get failed: %w", err)
	}

	// Store result
	if step.Target != "" {
		ctx.Set(step.Target, value)
	}

	return nil
}

// executeCacheSet stores a value in cache.
func executeCacheSet(ctx *ExecutionContext, step *ast.LogicStep) error {
	key := ""
	if len(step.Args) > 0 {
		key = step.Args[0]
	}

	var value interface{}
	if len(step.Args) > 1 {
		valueArg := step.Args[1]
		value = ctx.Get(valueArg)
	}

	ttl := 0
	if len(step.Args) > 2 {
		// Parse TTL
		// For now, just use default
	}

	return ctx.CacheSet(key, value, ttl)
}

// parseCondition parses a simple condition string.
func parseCondition(condition string, ctx *ExecutionContext) map[string]interface{} {
	result := make(map[string]interface{})

	// Parse simple "field = value" or "field = $var" format
	parts := strings.Split(condition, "=")
	if len(parts) == 2 {
		field := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Check if value is a variable reference
		if strings.HasPrefix(value, "$") {
			varName := strings.TrimPrefix(value, "$")
			result[field] = ctx.Get(varName)
		} else if strings.HasPrefix(value, "request.") || strings.HasPrefix(value, "input.") {
			fieldName := strings.TrimPrefix(strings.TrimPrefix(value, "request."), "input.")
			if input, ok := ctx.Input().(map[string]interface{}); ok {
				result[field] = input[fieldName]
			}
		} else {
			// Literal value
			result[field] = strings.Trim(value, "\"'")
		}
	}

	return result
}

// determineErrorStatusCode determines the HTTP status code for an error.
func determineErrorStatusCode(err error) int {
	switch err.(type) {
	case *ValidationError:
		return http.StatusBadRequest
	case *AuthorizationError:
		return http.StatusForbidden
	case *NotFoundError:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]interface{}{
		"error":   http.StatusText(statusCode),
		"message": message,
	})
}

// Error types for handler logic.

// ValidationError represents a validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

// AuthorizationError represents an authorization failure.
type AuthorizationError struct {
	Message      string
	RequiredRole string
}

func (e *AuthorizationError) Error() string {
	if e.RequiredRole != "" {
		return fmt.Sprintf("authorization failed: %s (required role: %s)", e.Message, e.RequiredRole)
	}
	return fmt.Sprintf("authorization failed: %s", e.Message)
}

// NotFoundError represents a resource not found.
type NotFoundError struct {
	Resource string
	ID       interface{}
}

func (e *NotFoundError) Error() string {
	if e.ID != nil {
		return fmt.Sprintf("%s not found: %v", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

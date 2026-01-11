# Task 006: HTTP Module with Chi Router

## Overview
Implement the HTTP module that handles API routing, request validation, response serialization, middleware, and automatic endpoint generation from CodeAI endpoint declarations.

## Phase
Phase 1: Foundation

## Priority
Critical - HTTP module is required for API functionality.

## Dependencies
- Task 001: Project Structure Setup
- Task 003: AST Node Types and Transformation
- Task 005: PostgreSQL Database Module

## Description
Create the HTTP module using Chi router that automatically registers endpoints from CodeAI declarations, handles request validation, response formatting, and provides standard middleware for logging, recovery, and CORS.

## Detailed Requirements

### 1. Module Interface (internal/modules/http/module.go)

```go
package http

import (
    "context"
    "net/http"

    "github.com/codeai/codeai/internal/parser"
    "github.com/codeai/codeai/internal/runtime"
)

// Module is the interface for HTTP operations
type Module interface {
    // Lifecycle
    Name() string
    Initialize(config *Config) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() runtime.HealthStatus

    // Route registration
    RegisterEndpoint(endpoint *parser.Endpoint, handler HandlerFunc) error

    // Middleware
    Use(middleware Middleware)

    // Server access
    Router() http.Handler
}

// Config for HTTP module
type Config struct {
    Host           string
    Port           int
    ReadTimeout    time.Duration
    WriteTimeout   time.Duration
    IdleTimeout    time.Duration
    MaxHeaderBytes int
    CORS           *CORSConfig
}

type CORSConfig struct {
    Enabled          bool
    AllowedOrigins   []string
    AllowedMethods   []string
    AllowedHeaders   []string
    ExposedHeaders   []string
    AllowCredentials bool
    MaxAge           int
}

// HandlerFunc is the signature for endpoint handlers
type HandlerFunc func(ctx *runtime.ExecutionContext, req *Request) (*Response, error)

// Middleware is the signature for middleware functions
type Middleware func(http.Handler) http.Handler

// Request represents a parsed HTTP request
type Request struct {
    Method      string
    Path        string
    PathParams  map[string]string
    QueryParams map[string]any
    Body        map[string]any
    Headers     http.Header
    RawBody     []byte
}

// Response represents an HTTP response
type Response struct {
    Status  int
    Body    any
    Headers map[string]string
}

// Paginated wraps paginated results
type Paginated struct {
    Data       any   `json:"data"`
    Total      int64 `json:"total"`
    Page       int   `json:"page"`
    PerPage    int   `json:"per_page"`
    TotalPages int   `json:"total_pages"`
}

// Error response types
type ErrorResponse struct {
    Error   string         `json:"error"`
    Code    string         `json:"code,omitempty"`
    Details map[string]any `json:"details,omitempty"`
}
```

### 2. Server Implementation (internal/modules/http/server.go)

```go
package http

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/go-chi/cors"
    "log/slog"

    "github.com/codeai/codeai/internal/parser"
    "github.com/codeai/codeai/internal/runtime"
)

type Server struct {
    router     *chi.Mux
    server     *http.Server
    config     *Config
    endpoints  []*parser.Endpoint
    handlers   map[string]HandlerFunc
    auth       runtime.AuthModule
    db         runtime.DatabaseModule
    logger     *slog.Logger
}

func NewServer(config *Config) *Server {
    r := chi.NewRouter()

    return &Server{
        router:   r,
        config:   config,
        handlers: make(map[string]HandlerFunc),
        logger:   slog.Default().With("module", "http"),
    }
}

func (s *Server) Name() string {
    return "http"
}

func (s *Server) Initialize(config *Config) error {
    s.config = config
    s.setupMiddleware()
    return nil
}

func (s *Server) setupMiddleware() {
    // Request ID
    s.router.Use(middleware.RequestID)

    // Real IP
    s.router.Use(middleware.RealIP)

    // Logger
    s.router.Use(s.loggingMiddleware)

    // Recoverer
    s.router.Use(s.recoveryMiddleware)

    // Timeout
    if s.config.ReadTimeout > 0 {
        s.router.Use(middleware.Timeout(s.config.ReadTimeout))
    }

    // CORS
    if s.config.CORS != nil && s.config.CORS.Enabled {
        s.router.Use(cors.Handler(cors.Options{
            AllowedOrigins:   s.config.CORS.AllowedOrigins,
            AllowedMethods:   s.config.CORS.AllowedMethods,
            AllowedHeaders:   s.config.CORS.AllowedHeaders,
            ExposedHeaders:   s.config.CORS.ExposedHeaders,
            AllowCredentials: s.config.CORS.AllowCredentials,
            MaxAge:           s.config.CORS.MaxAge,
        }))
    }

    // Health check endpoint
    s.router.Get("/health", s.healthHandler)

    // Ready check endpoint
    s.router.Get("/ready", s.readyHandler)
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

        defer func() {
            s.logger.Info("request",
                "method", r.Method,
                "path", r.URL.Path,
                "status", ww.Status(),
                "bytes", ww.BytesWritten(),
                "duration", time.Since(start),
                "request_id", middleware.GetReqID(r.Context()),
            )
        }()

        next.ServeHTTP(ww, r)
    })
}

func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                s.logger.Error("panic recovered",
                    "error", err,
                    "request_id", middleware.GetReqID(r.Context()),
                )
                s.writeError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
            }
        }()
        next.ServeHTTP(w, r)
    })
}

func (s *Server) Start(ctx context.Context) error {
    addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

    s.server = &http.Server{
        Addr:           addr,
        Handler:        s.router,
        ReadTimeout:    s.config.ReadTimeout,
        WriteTimeout:   s.config.WriteTimeout,
        IdleTimeout:    s.config.IdleTimeout,
        MaxHeaderBytes: s.config.MaxHeaderBytes,
    }

    s.logger.Info("starting HTTP server", "addr", addr)

    go func() {
        if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            s.logger.Error("server error", "error", err)
        }
    }()

    return nil
}

func (s *Server) Stop(ctx context.Context) error {
    s.logger.Info("shutting down HTTP server")
    return s.server.Shutdown(ctx)
}

func (s *Server) Router() http.Handler {
    return s.router
}

func (s *Server) Health() runtime.HealthStatus {
    return runtime.HealthStatus{
        Status: "healthy",
        Details: map[string]any{
            "endpoints": len(s.endpoints),
        },
    }
}
```

### 3. Endpoint Registration (internal/modules/http/router.go)

```go
package http

import (
    "fmt"
    "strings"

    "github.com/go-chi/chi/v5"

    "github.com/codeai/codeai/internal/parser"
)

// RegisterEndpoint registers a CodeAI endpoint with the router
func (s *Server) RegisterEndpoint(endpoint *parser.Endpoint, handler HandlerFunc) error {
    // Convert CodeAI path to Chi path pattern
    chiPath := s.convertPath(endpoint.Path)

    // Create the handler
    httpHandler := s.createHandler(endpoint, handler)

    // Register with Chi
    switch endpoint.Method {
    case "GET":
        s.router.Get(chiPath, httpHandler)
    case "POST":
        s.router.Post(chiPath, httpHandler)
    case "PUT":
        s.router.Put(chiPath, httpHandler)
    case "DELETE":
        s.router.Delete(chiPath, httpHandler)
    case "PATCH":
        s.router.Patch(chiPath, httpHandler)
    default:
        return fmt.Errorf("unsupported HTTP method: %s", endpoint.Method)
    }

    s.endpoints = append(s.endpoints, endpoint)
    s.logger.Info("registered endpoint", "method", endpoint.Method, "path", chiPath)

    return nil
}

// convertPath converts CodeAI path syntax to Chi path syntax
// CodeAI: /products/{id}  ->  Chi: /products/{id}
// CodeAI: /users/{user_id}/posts/{post_id}  ->  Chi: /users/{user_id}/posts/{post_id}
func (s *Server) convertPath(path string) string {
    // CodeAI and Chi use the same {param} syntax
    return path
}

func (s *Server) createHandler(endpoint *parser.Endpoint, handler HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Create execution context
        execCtx := s.createExecutionContext(ctx, r)

        // Authentication
        if endpoint.Auth == parser.AuthRequired {
            user, err := s.auth.ValidateToken(ctx, s.extractToken(r))
            if err != nil {
                s.writeError(w, http.StatusUnauthorized, "authentication required", "AUTH_REQUIRED")
                return
            }
            execCtx.SetUser(user)

            // Role check
            if len(endpoint.Roles) > 0 {
                hasRole := false
                for _, role := range endpoint.Roles {
                    if user.HasRole(role) {
                        hasRole = true
                        break
                    }
                }
                if !hasRole {
                    s.writeError(w, http.StatusForbidden, "insufficient permissions", "FORBIDDEN")
                    return
                }
            }
        } else if endpoint.Auth == parser.AuthOptional {
            if token := s.extractToken(r); token != "" {
                if user, err := s.auth.ValidateToken(ctx, token); err == nil {
                    execCtx.SetUser(user)
                }
            }
        }

        // Parse request
        req, err := s.parseRequest(r, endpoint)
        if err != nil {
            s.writeError(w, http.StatusBadRequest, err.Error(), "INVALID_REQUEST")
            return
        }

        // Validate request
        if err := s.validateRequest(req, endpoint); err != nil {
            s.writeError(w, http.StatusBadRequest, err.Error(), "VALIDATION_ERROR")
            return
        }

        // Call handler
        resp, err := handler(execCtx, req)
        if err != nil {
            s.handleError(w, err)
            return
        }

        // Write response
        s.writeResponse(w, resp)
    }
}

func (s *Server) extractToken(r *http.Request) string {
    // Check Authorization header
    auth := r.Header.Get("Authorization")
    if strings.HasPrefix(auth, "Bearer ") {
        return strings.TrimPrefix(auth, "Bearer ")
    }
    return ""
}

func (s *Server) createExecutionContext(ctx context.Context, r *http.Request) *runtime.ExecutionContext {
    return runtime.NewExecutionContext(ctx).
        WithRequestID(middleware.GetReqID(r.Context())).
        WithDatabase(s.db).
        WithLogger(s.logger)
}
```

### 4. Request Parsing and Validation (internal/modules/http/handlers.go)

```go
package http

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strconv"
    "strings"

    "github.com/go-chi/chi/v5"

    "github.com/codeai/codeai/internal/parser"
)

// parseRequest parses an HTTP request according to endpoint definition
func (s *Server) parseRequest(r *http.Request, endpoint *parser.Endpoint) (*Request, error) {
    req := &Request{
        Method:      r.Method,
        Path:        r.URL.Path,
        PathParams:  make(map[string]string),
        QueryParams: make(map[string]any),
        Body:        make(map[string]any),
        Headers:     r.Header,
    }

    // Parse path parameters
    for _, param := range endpoint.PathParams {
        value := chi.URLParam(r, param.Name)
        req.PathParams[param.Name] = value
    }

    // Parse query parameters
    for _, param := range endpoint.QueryParams {
        value := r.URL.Query().Get(param.Name)
        if value == "" {
            if param.Default != nil {
                req.QueryParams[param.Name] = s.evaluateDefault(param.Default)
            }
            continue
        }

        parsed, err := s.parseValue(value, param.Type)
        if err != nil {
            return nil, fmt.Errorf("invalid query parameter '%s': %w", param.Name, err)
        }
        req.QueryParams[param.Name] = parsed
    }

    // Parse body
    if r.Body != nil && r.ContentLength > 0 {
        body, err := io.ReadAll(r.Body)
        if err != nil {
            return nil, fmt.Errorf("failed to read body: %w", err)
        }
        req.RawBody = body

        if len(body) > 0 {
            if err := json.Unmarshal(body, &req.Body); err != nil {
                return nil, fmt.Errorf("invalid JSON body: %w", err)
            }
        }
    }

    return req, nil
}

// validateRequest validates a request against endpoint definition
func (s *Server) validateRequest(req *Request, endpoint *parser.Endpoint) error {
    var errors []string

    // Validate path parameters
    for _, param := range endpoint.PathParams {
        value, ok := req.PathParams[param.Name]
        if !ok || value == "" {
            if param.Required {
                errors = append(errors, fmt.Sprintf("path parameter '%s' is required", param.Name))
            }
            continue
        }

        if err := s.validateValue(value, param); err != nil {
            errors = append(errors, err.Error())
        }
    }

    // Validate query parameters
    for _, param := range endpoint.QueryParams {
        value, ok := req.QueryParams[param.Name]
        if !ok {
            if param.Required {
                errors = append(errors, fmt.Sprintf("query parameter '%s' is required", param.Name))
            }
            continue
        }

        if err := s.validateValue(value, param); err != nil {
            errors = append(errors, err.Error())
        }
    }

    // Validate body parameters
    for _, param := range endpoint.Body {
        value, ok := req.Body[param.Name]
        if !ok {
            if param.Required {
                errors = append(errors, fmt.Sprintf("body field '%s' is required", param.Name))
            }
            continue
        }

        if err := s.validateValue(value, param); err != nil {
            errors = append(errors, err.Error())
        }
    }

    if len(errors) > 0 {
        return &ValidationError{Errors: errors}
    }

    return nil
}

func (s *Server) validateValue(value any, param *parser.Param) error {
    for _, v := range param.Validators {
        switch v.Type {
        case "min":
            minVal := v.Value.(float64)
            switch val := value.(type) {
            case int:
                if float64(val) < minVal {
                    return fmt.Errorf("%s must be at least %v", param.Name, minVal)
                }
            case float64:
                if val < minVal {
                    return fmt.Errorf("%s must be at least %v", param.Name, minVal)
                }
            case string:
                if len(val) < int(minVal) {
                    return fmt.Errorf("%s must be at least %v characters", param.Name, minVal)
                }
            }

        case "max":
            maxVal := v.Value.(float64)
            switch val := value.(type) {
            case int:
                if float64(val) > maxVal {
                    return fmt.Errorf("%s must be at most %v", param.Name, maxVal)
                }
            case float64:
                if val > maxVal {
                    return fmt.Errorf("%s must be at most %v", param.Name, maxVal)
                }
            case string:
                if len(val) > int(maxVal) {
                    return fmt.Errorf("%s must be at most %v characters", param.Name, maxVal)
                }
            }

        case "pattern":
            pattern := v.Value.(string)
            val, ok := value.(string)
            if !ok {
                return fmt.Errorf("%s must be a string for pattern validation", param.Name)
            }
            matched, _ := regexp.MatchString(pattern, val)
            if !matched {
                return fmt.Errorf("%s does not match required pattern", param.Name)
            }
        }
    }

    return nil
}

func (s *Server) parseValue(value string, typ parser.Type) (any, error) {
    switch typ.TypeName() {
    case "string", "text":
        return value, nil
    case "integer":
        return strconv.ParseInt(value, 10, 64)
    case "decimal":
        return strconv.ParseFloat(value, 64)
    case "boolean":
        return strconv.ParseBool(value)
    case "uuid":
        // Validate UUID format
        if !isValidUUID(value) {
            return nil, fmt.Errorf("invalid UUID format")
        }
        return value, nil
    default:
        return value, nil
    }
}
```

### 5. Response Handling

```go
// writeResponse writes a successful response
func (s *Server) writeResponse(w http.ResponseWriter, resp *Response) {
    // Set headers
    for k, v := range resp.Headers {
        w.Header().Set(k, v)
    }

    // Default to JSON
    w.Header().Set("Content-Type", "application/json")

    // Set status
    status := resp.Status
    if status == 0 {
        status = http.StatusOK
    }
    w.WriteHeader(status)

    // Write body
    if resp.Body != nil {
        if err := json.NewEncoder(w).Encode(resp.Body); err != nil {
            s.logger.Error("failed to encode response", "error", err)
        }
    }
}

// writeError writes an error response
func (s *Server) writeError(w http.ResponseWriter, status int, message, code string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)

    json.NewEncoder(w).Encode(ErrorResponse{
        Error: message,
        Code:  code,
    })
}

// handleError handles errors from handlers
func (s *Server) handleError(w http.ResponseWriter, err error) {
    switch e := err.(type) {
    case *NotFoundError:
        s.writeError(w, http.StatusNotFound, e.Error(), "NOT_FOUND")
    case *ValidationError:
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(ErrorResponse{
            Error:   "validation failed",
            Code:    "VALIDATION_ERROR",
            Details: map[string]any{"errors": e.Errors},
        })
    case *UnauthorizedError:
        s.writeError(w, http.StatusUnauthorized, e.Error(), "UNAUTHORIZED")
    case *ForbiddenError:
        s.writeError(w, http.StatusForbidden, e.Error(), "FORBIDDEN")
    default:
        s.logger.Error("handler error", "error", err)
        s.writeError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
    }
}

// Health and ready handlers
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
    // Check if all dependencies are ready
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}
```

### 6. Error Types (internal/modules/http/errors.go)

```go
package http

type ValidationError struct {
    Errors []string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed: %s", strings.Join(e.Errors, ", "))
}

type NotFoundError struct {
    Resource string
    ID       string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s with ID '%s' not found", e.Resource, e.ID)
}

type UnauthorizedError struct {
    Message string
}

func (e *UnauthorizedError) Error() string {
    if e.Message != "" {
        return e.Message
    }
    return "unauthorized"
}

type ForbiddenError struct {
    Message string
}

func (e *ForbiddenError) Error() string {
    if e.Message != "" {
        return e.Message
    }
    return "forbidden"
}
```

## Acceptance Criteria
- [ ] Chi router properly configured
- [ ] Endpoints registered from CodeAI declarations
- [ ] Request parsing for path, query, and body parameters
- [ ] Request validation with proper error messages
- [ ] JSON response serialization
- [ ] CORS middleware configurable
- [ ] Logging middleware with request ID
- [ ] Panic recovery middleware
- [ ] Health and ready endpoints
- [ ] Graceful shutdown

## Testing Strategy
- Unit tests for request parsing
- Unit tests for validation
- Integration tests with test server
- Load testing for concurrent requests

## Files to Create/Modify
- `internal/modules/http/module.go`
- `internal/modules/http/server.go`
- `internal/modules/http/router.go`
- `internal/modules/http/handlers.go`
- `internal/modules/http/middleware.go`
- `internal/modules/http/errors.go`
- `internal/modules/http/server_test.go`

# Task 004: Validator Implementation

## Overview
Implement the validation layer that performs type checking, reference resolution, and semantic validation of parsed CodeAI programs before execution.

## Phase
Phase 1: Foundation

## Priority
Critical - Validation ensures program correctness before runtime.

## Dependencies
- Task 001: Project Structure Setup
- Task 002: Participle Grammar Implementation
- Task 003: AST Node Types and Transformation

## Description
Create a comprehensive validator that transforms parsed grammar structures into validated AST nodes, resolving all references, checking types, and providing detailed error messages suitable for LLM feedback.

## Detailed Requirements

### 1. Validator Core (internal/validator/validator.go)

```go
package validator

import (
    "github.com/codeai/codeai/internal/parser"
)

// Validator performs semantic analysis on parsed programs
type Validator struct {
    errors   []ValidationError
    warnings []ValidationWarning

    // Symbol tables
    entities     map[string]*parser.Entity
    endpoints    []*parser.Endpoint
    workflows    map[string]*parser.Workflow
    jobs         map[string]*parser.Job
    integrations map[string]*parser.Integration
    events       map[string]*parser.Event
    functions    map[string]*parser.Function

    // Type registry
    types map[string]parser.Type

    // Current context for nested validation
    currentEntity   *parser.Entity
    currentEndpoint *parser.Endpoint
    currentWorkflow *parser.Workflow
    currentScope    *Scope
}

func NewValidator() *Validator {
    v := &Validator{
        entities:     make(map[string]*parser.Entity),
        workflows:    make(map[string]*parser.Workflow),
        jobs:         make(map[string]*parser.Job),
        integrations: make(map[string]*parser.Integration),
        events:       make(map[string]*parser.Event),
        functions:    make(map[string]*parser.Function),
        types:        make(map[string]parser.Type),
    }
    v.registerBuiltinTypes()
    v.registerBuiltinFunctions()
    return v
}

func (v *Validator) Validate(program *parser.Program) (*parser.AST, error) {
    // Phase 1: Collect all declarations (forward references)
    for _, decl := range program.Declarations {
        v.collectDeclaration(decl)
    }

    // Phase 2: Resolve references and validate
    for _, decl := range program.Declarations {
        v.validateDeclaration(decl)
    }

    // Phase 3: Check for semantic errors
    v.checkSemanticRules()

    if len(v.errors) > 0 {
        return nil, &ValidationErrors{Errors: v.errors}
    }

    return v.buildAST(), nil
}
```

### 2. Error Types (internal/validator/errors.go)

```go
package validator

import (
    "fmt"
    "strings"

    "github.com/codeai/codeai/internal/parser"
)

type ValidationError struct {
    Position   parser.Position
    Code       string
    Message    string
    Suggestion string
    Context    map[string]any
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Position, e.Message)
}

func (e ValidationError) Pretty() string {
    var b strings.Builder

    b.WriteString(fmt.Sprintf("Error [%s] at %s\n", e.Code, e.Position))
    b.WriteString(fmt.Sprintf("  %s\n", e.Message))

    if e.Suggestion != "" {
        b.WriteString(fmt.Sprintf("\n  Suggestion: %s\n", e.Suggestion))
    }

    return b.String()
}

// LLMFeedback returns JSON-formatted error for LLM correction
func (e ValidationError) LLMFeedback() map[string]any {
    return map[string]any{
        "type":       e.Code,
        "location":   e.Position.String(),
        "message":    e.Message,
        "suggestion": e.Suggestion,
        "context":    e.Context,
    }
}

type ValidationWarning struct {
    Position parser.Position
    Code     string
    Message  string
}

type ValidationErrors struct {
    Errors []ValidationError
}

func (e *ValidationErrors) Error() string {
    var msgs []string
    for _, err := range e.Errors {
        msgs = append(msgs, err.Error())
    }
    return strings.Join(msgs, "\n")
}

// Error codes
const (
    ErrUndefinedEntity      = "undefined_entity"
    ErrUndefinedField       = "undefined_field"
    ErrUndefinedType        = "undefined_type"
    ErrUndefinedFunction    = "undefined_function"
    ErrUndefinedEvent       = "undefined_event"
    ErrUndefinedIntegration = "undefined_integration"
    ErrDuplicateEntity      = "duplicate_entity"
    ErrDuplicateField       = "duplicate_field"
    ErrDuplicateEndpoint    = "duplicate_endpoint"
    ErrTypeMismatch         = "type_mismatch"
    ErrMissingRequired      = "missing_required"
    ErrInvalidModifier      = "invalid_modifier"
    ErrInvalidReference     = "invalid_reference"
    ErrCircularReference    = "circular_reference"
    ErrInvalidExpression    = "invalid_expression"
    ErrInvalidAuth          = "invalid_auth"
)

// Suggestions for common errors
var suggestions = map[string]string{
    ErrUndefinedEntity:   "Check that the entity is declared in a .codeai file",
    ErrUndefinedType:     "Valid types: string, text, integer, decimal, boolean, timestamp, uuid, json, list(T), ref(Entity), enum(a,b,c)",
    ErrTypeMismatch:      "Check that the value type matches the expected type",
    ErrMissingRequired:   "Add the missing required field or parameter",
    ErrDuplicateEntity:   "Entity names must be unique across all files",
}
```

### 3. Reference Resolver (internal/validator/resolver.go)

```go
package validator

import "github.com/codeai/codeai/internal/parser"

// Scope represents a variable scope for expression validation
type Scope struct {
    parent    *Scope
    variables map[string]parser.Type
}

func NewScope(parent *Scope) *Scope {
    return &Scope{
        parent:    parent,
        variables: make(map[string]parser.Type),
    }
}

func (s *Scope) Define(name string, typ parser.Type) {
    s.variables[name] = typ
}

func (s *Scope) Lookup(name string) (parser.Type, bool) {
    if typ, ok := s.variables[name]; ok {
        return typ, true
    }
    if s.parent != nil {
        return s.parent.Lookup(name)
    }
    return nil, false
}

// ResolveEntityRef resolves an entity reference
func (v *Validator) ResolveEntityRef(name string, pos parser.Position) *parser.Entity {
    entity, ok := v.entities[name]
    if !ok {
        v.addError(ValidationError{
            Position: pos,
            Code:     ErrUndefinedEntity,
            Message:  fmt.Sprintf("Entity '%s' is not defined", name),
            Context: map[string]any{
                "defined_entities": v.entityNames(),
            },
        })
        return nil
    }
    return entity
}

// ResolveType resolves a type reference
func (v *Validator) ResolveType(typeRef *parser.TypeRef, pos parser.Position) parser.Type {
    switch typeRef.Name {
    case "string", "text", "integer", "boolean", "timestamp", "date", "time", "uuid", "json":
        return &parser.PrimitiveType{Name: typeRef.Name}

    case "decimal":
        precision, scale := 10, 2
        if len(typeRef.Params) >= 1 {
            precision = int(typeRef.Params[0].(*parser.LiteralExpr).Value.(float64))
        }
        if len(typeRef.Params) >= 2 {
            scale = int(typeRef.Params[1].(*parser.LiteralExpr).Value.(float64))
        }
        return &parser.PrimitiveType{Name: "decimal", Precision: precision, Scale: scale}

    case "list":
        if len(typeRef.Params) != 1 {
            v.addError(ValidationError{
                Position: pos,
                Code:     ErrInvalidType,
                Message:  "list type requires exactly one type parameter",
            })
            return nil
        }
        elemType := v.ResolveType(typeRef.Params[0], pos)
        return &parser.ListType{ElementType: elemType}

    case "ref":
        if len(typeRef.Params) != 1 {
            v.addError(ValidationError{
                Position: pos,
                Code:     ErrInvalidType,
                Message:  "ref type requires exactly one entity parameter",
            })
            return nil
        }
        entityName := typeRef.Params[0].Name
        return &parser.RefType{EntityName: entityName}

    case "enum":
        var values []string
        for _, p := range typeRef.Params {
            values = append(values, p.Name)
        }
        return &parser.EnumType{Values: values}

    default:
        // Check if it's a custom type or entity name
        if entity, ok := v.entities[typeRef.Name]; ok {
            return &parser.RefType{EntityName: entity.Name, Entity: entity}
        }

        v.addError(ValidationError{
            Position:   pos,
            Code:       ErrUndefinedType,
            Message:    fmt.Sprintf("Unknown type '%s'", typeRef.Name),
            Suggestion: suggestions[ErrUndefinedType],
        })
        return nil
    }
}

// ResolveExpression validates and resolves types in an expression
func (v *Validator) ResolveExpression(expr parser.Expression, scope *Scope) parser.Type {
    switch e := expr.(type) {
    case *parser.LiteralExpr:
        return v.resolveLiteralType(e)

    case *parser.IdentifierExpr:
        return v.resolveIdentifier(e, scope)

    case *parser.BinaryExpr:
        return v.resolveBinaryExpr(e, scope)

    case *parser.CallExpr:
        return v.resolveCallExpr(e, scope)

    case *parser.ListExpr:
        return v.resolveListExpr(e, scope)

    case *parser.ConditionalExpr:
        return v.resolveConditionalExpr(e, scope)

    case *parser.LambdaExpr:
        return v.resolveLambdaExpr(e, scope)

    default:
        return nil
    }
}

func (v *Validator) resolveLiteralType(e *parser.LiteralExpr) parser.Type {
    switch e.Value.(type) {
    case string:
        return &parser.PrimitiveType{Name: "string"}
    case float64:
        if float64(int(e.Value.(float64))) == e.Value.(float64) {
            return &parser.PrimitiveType{Name: "integer"}
        }
        return &parser.PrimitiveType{Name: "decimal"}
    case bool:
        return &parser.PrimitiveType{Name: "boolean"}
    case nil:
        return &parser.PrimitiveType{Name: "null"}
    default:
        return nil
    }
}

func (v *Validator) resolveIdentifier(e *parser.IdentifierExpr, scope *Scope) parser.Type {
    // First part of identifier
    firstPart := e.Parts[0]

    // Check scope variables
    if typ, ok := scope.Lookup(firstPart); ok {
        return v.resolveFieldChain(typ, e.Parts[1:], e.Pos)
    }

    // Check entities
    if entity, ok := v.entities[firstPart]; ok {
        e.Ref = entity
        return v.resolveFieldChain(&parser.RefType{Entity: entity}, e.Parts[1:], e.Pos)
    }

    // Check built-in variables (query, path, body, trigger, steps, config)
    switch firstPart {
    case "query", "path", "body":
        return v.resolveRequestField(firstPart, e.Parts[1:], e.Pos)
    case "trigger":
        return v.resolveTriggerField(e.Parts[1:], e.Pos)
    case "steps":
        return v.resolveStepResult(e.Parts[1:], e.Pos)
    case "config":
        return v.resolveConfigField(e.Parts[1:], e.Pos)
    }

    v.addError(ValidationError{
        Position: e.Pos,
        Code:     ErrUndefinedField,
        Message:  fmt.Sprintf("Undefined identifier '%s'", firstPart),
    })
    return nil
}
```

### 4. Entity Validation

```go
func (v *Validator) validateEntity(decl *parser.EntityDecl) *parser.Entity {
    // Check for duplicate entity names
    if _, exists := v.entities[decl.Name]; exists {
        v.addError(ValidationError{
            Position: decl.Pos,
            Code:     ErrDuplicateEntity,
            Message:  fmt.Sprintf("Entity '%s' is already defined", decl.Name),
        })
        return nil
    }

    entity := &parser.Entity{
        Position:    parser.Position(decl.Pos),
        Name:        decl.Name,
        Description: stringVal(decl.Description),
    }

    fieldNames := make(map[string]bool)

    for _, fieldDecl := range decl.Fields {
        // Check for duplicate field names
        if fieldNames[fieldDecl.Name] {
            v.addError(ValidationError{
                Position: fieldDecl.Pos,
                Code:     ErrDuplicateField,
                Message:  fmt.Sprintf("Field '%s' is already defined in entity '%s'", fieldDecl.Name, decl.Name),
            })
            continue
        }
        fieldNames[fieldDecl.Name] = true

        field := v.validateField(fieldDecl, entity)
        if field != nil {
            entity.Fields = append(entity.Fields, field)

            // Track special fields
            if field.Primary {
                if entity.PrimaryKey != nil {
                    v.addError(ValidationError{
                        Position: fieldDecl.Pos,
                        Code:     ErrInvalidModifier,
                        Message:  "Entity can only have one primary key",
                    })
                } else {
                    entity.PrimaryKey = field
                }
            }

            if field.SoftDelete {
                entity.SoftDelete = field
            }

            if field.Auto || field.AutoUpdate {
                entity.Timestamps = append(entity.Timestamps, field)
            }

            if field.Reference != nil {
                entity.References = append(entity.References, field)
            }
        }
    }

    // Validate indexes
    for _, indexDecl := range decl.Indexes {
        index := v.validateIndex(indexDecl, entity)
        if index != nil {
            entity.Indexes = append(entity.Indexes, index)
        }
    }

    // Entity must have a primary key
    if entity.PrimaryKey == nil {
        v.addError(ValidationError{
            Position:   decl.Pos,
            Code:       ErrMissingRequired,
            Message:    fmt.Sprintf("Entity '%s' must have a primary key field", decl.Name),
            Suggestion: "Add a field with 'primary' modifier, e.g., 'id: uuid, primary, auto'",
        })
    }

    return entity
}

func (v *Validator) validateField(decl *parser.FieldDecl, entity *parser.Entity) *parser.Field {
    field := &parser.Field{
        Position: parser.Position(decl.Pos),
        Name:     decl.Name,
    }

    // Resolve type
    field.Type = v.ResolveType(decl.Type, decl.Pos)

    // Process modifiers
    for _, mod := range decl.Modifiers {
        switch mod.Name {
        case "primary":
            field.Primary = true
        case "auto":
            field.Auto = true
        case "auto_update":
            field.AutoUpdate = true
        case "required":
            field.Required = true
        case "optional":
            field.Required = false
        case "unique":
            field.Unique = true
        case "searchable":
            field.Searchable = true
        case "soft_delete":
            field.SoftDelete = true
        case "default":
            if mod.Value == nil {
                v.addError(ValidationError{
                    Position: mod.Pos,
                    Code:     ErrInvalidModifier,
                    Message:  "default modifier requires a value",
                })
            } else {
                field.Default = mod.Value
            }
        case "min", "max", "pattern":
            field.Validators = append(field.Validators, parser.Validator{
                Type:  mod.Name,
                Value: mod.Value,
            })
        default:
            v.addError(ValidationError{
                Position:   mod.Pos,
                Code:       ErrInvalidModifier,
                Message:    fmt.Sprintf("Unknown modifier '%s'", mod.Name),
                Suggestion: "Valid modifiers: primary, auto, auto_update, required, optional, unique, searchable, soft_delete, default(value)",
            })
        }
    }

    // Handle reference types
    if refType, ok := field.Type.(*parser.RefType); ok {
        field.Reference = &parser.EntityRef{
            EntityName: refType.EntityName,
        }
    }

    return field
}
```

### 5. Endpoint Validation

```go
func (v *Validator) validateEndpoint(decl *parser.EndpointDecl) *parser.Endpoint {
    endpoint := &parser.Endpoint{
        Position:    parser.Position(decl.Pos),
        Method:      decl.Method,
        Path:        decl.Path,
        Description: stringVal(decl.Description),
    }

    // Convert to Chi route pattern
    endpoint.RoutePattern = v.convertToRoutePattern(decl.Path)

    // Check for duplicate endpoints
    for _, existing := range v.endpoints {
        if existing.Method == endpoint.Method && existing.RoutePattern == endpoint.RoutePattern {
            v.addError(ValidationError{
                Position: decl.Pos,
                Code:     ErrDuplicateEndpoint,
                Message:  fmt.Sprintf("Endpoint %s %s is already defined", endpoint.Method, endpoint.Path),
            })
        }
    }

    // Validate auth
    if decl.Auth != nil {
        endpoint.Auth = parser.AuthRequirement(decl.Auth.Type)
    } else {
        endpoint.Auth = parser.AuthPublic // Default
    }

    // Validate roles
    if decl.Roles != nil {
        endpoint.Roles = decl.Roles.Roles
        // Roles require auth
        if endpoint.Auth == parser.AuthPublic {
            v.addWarning(ValidationWarning{
                Position: decl.Pos,
                Code:     "roles_without_auth",
                Message:  "Roles specified but endpoint is public; roles will be ignored",
            })
        }
    }

    // Create scope for parameter validation
    scope := NewScope(nil)

    // Validate path parameters
    pathParams := v.extractPathParams(decl.Path)
    if decl.PathParams != nil {
        for _, param := range decl.PathParams.Params {
            p := v.validateParam(param)
            if p != nil {
                endpoint.PathParams = append(endpoint.PathParams, p)
                scope.Define(param.Name, p.Type)

                // Check path param is in path
                if !contains(pathParams, param.Name) {
                    v.addWarning(ValidationWarning{
                        Position: param.Pos,
                        Message:  fmt.Sprintf("Path parameter '%s' is not in the path", param.Name),
                    })
                }
            }
        }
    }

    // Validate query parameters
    if decl.QueryParams != nil {
        for _, param := range decl.QueryParams.Params {
            p := v.validateParam(param)
            if p != nil {
                endpoint.QueryParams = append(endpoint.QueryParams, p)
                scope.Define("query."+param.Name, p.Type)
            }
        }
    }

    // Validate body (only for POST, PUT, PATCH)
    if decl.Body != nil {
        if decl.Method == "GET" || decl.Method == "DELETE" {
            v.addWarning(ValidationWarning{
                Position: decl.Pos,
                Message:  fmt.Sprintf("%s requests should not have a body", decl.Method),
            })
        }
        for _, param := range decl.Body.Params {
            p := v.validateParam(param)
            if p != nil {
                endpoint.Body = append(endpoint.Body, p)
                scope.Define("body."+param.Name, p.Type)
            }
        }
    }

    // Validate returns
    if decl.Returns != nil {
        endpoint.Returns = v.validateReturnType(decl.Returns)
    }

    // Validate filter expressions
    if decl.Filter != nil {
        endpoint.Filter = v.validateFilter(decl.Filter, scope)
    }

    // Validate sort
    if decl.Sort != nil {
        endpoint.Sort = v.validateSort(decl.Sort)
    }

    // Validate cache
    if decl.Cache != nil {
        endpoint.Cache = &parser.CacheDirective{
            TTL: parseDuration(decl.Cache),
        }
    }

    // Validate on_success action
    if decl.OnSuccess != nil {
        endpoint.OnSuccess = v.validateAction(decl.OnSuccess, scope)
    }

    return endpoint
}
```

### 6. Semantic Checks

```go
func (v *Validator) checkSemanticRules() {
    // Check all entity references are resolved
    for _, entity := range v.entities {
        for _, field := range entity.Fields {
            if field.Reference != nil {
                refEntity := v.ResolveEntityRef(field.Reference.EntityName, field.Position)
                if refEntity != nil {
                    field.Reference.Entity = refEntity
                }
            }
        }
    }

    // Check workflow triggers exist
    for _, workflow := range v.workflows {
        if _, ok := v.events[workflow.Trigger]; !ok {
            v.addError(ValidationError{
                Position: workflow.Position,
                Code:     ErrUndefinedEvent,
                Message:  fmt.Sprintf("Workflow trigger event '%s' is not defined", workflow.Trigger),
                Context: map[string]any{
                    "defined_events": v.eventNames(),
                },
            })
        }
    }

    // Check integration calls reference valid operations
    for _, workflow := range v.workflows {
        for _, step := range workflow.Steps {
            if step.Call != nil {
                v.validateIntegrationCall(step.Call, step.Position)
            }
        }
    }

    // Check for circular entity references
    v.checkCircularReferences()
}

func (v *Validator) checkCircularReferences() {
    visited := make(map[string]bool)
    path := make(map[string]bool)

    var visit func(name string) bool
    visit = func(name string) bool {
        if path[name] {
            return true // Circular reference found
        }
        if visited[name] {
            return false
        }

        visited[name] = true
        path[name] = true

        entity := v.entities[name]
        for _, field := range entity.References {
            if visit(field.Reference.EntityName) {
                v.addError(ValidationError{
                    Position: field.Position,
                    Code:     ErrCircularReference,
                    Message:  fmt.Sprintf("Circular reference detected: %s -> %s", name, field.Reference.EntityName),
                })
            }
        }

        path[name] = false
        return false
    }

    for name := range v.entities {
        visit(name)
    }
}
```

## Acceptance Criteria
- [ ] Validator resolves all entity references
- [ ] Type checking works for all expressions
- [ ] Duplicate detection for entities, fields, endpoints
- [ ] Circular reference detection
- [ ] Clear error messages with suggestions
- [ ] LLM-friendly error output format
- [ ] Warning system for non-fatal issues

## Testing Strategy
- Unit tests for each validation function
- Integration tests with valid/invalid CodeAI files
- Error message quality tests

## Files to Create/Modify
- `internal/validator/validator.go`
- `internal/validator/resolver.go`
- `internal/validator/errors.go`
- `internal/validator/validator_test.go`

# Task 003: AST Node Types and Transformation

## Overview
Define the Abstract Syntax Tree (AST) node types that represent the semantic structure of CodeAI programs after parsing, including type resolution and reference tracking.

## Phase
Phase 1: Foundation

## Priority
Critical - AST is the interface between parser and runtime.

## Dependencies
- Task 001: Project Structure Setup
- Task 002: Participle Grammar Implementation

## Description
Create well-typed AST structures that the runtime can efficiently process. These types differ from the parser grammar types in that they are fully resolved, validated, and optimized for execution.

## Detailed Requirements

### 1. Core AST Types (internal/parser/ast.go)

```go
package parser

import "github.com/alecthomas/participle/v2/lexer"

// AST represents a complete, validated CodeAI program
type AST struct {
    Config       *Config
    Entities     map[string]*Entity
    Endpoints    []*Endpoint
    Workflows    map[string]*Workflow
    Jobs         map[string]*Job
    Integrations map[string]*Integration
    Events       map[string]*Event
    Functions    map[string]*Function

    // Metadata
    SourceFiles  []string
    Errors       []Error
}

// Position represents source code location
type Position struct {
    Filename string
    Line     int
    Column   int
    Offset   int
}

func (p Position) String() string {
    return fmt.Sprintf("%s:%d:%d", p.Filename, p.Line, p.Column)
}
```

### 2. Config AST

```go
// Config represents application configuration
type Config struct {
    Position    Position
    Name        string
    Version     string
    Database    *DatabaseConfig
    Cache       *CacheConfig
    Auth        *AuthConfig
    CORS        *CORSConfig
    Custom      map[string]any
}

type DatabaseConfig struct {
    Position         Position
    Type             string // postgres, mongodb
    ConnectionString string
    PoolSize         int
    MinPoolSize      int
    MaxConnLifetime  time.Duration
    MaxConnIdleTime  time.Duration
}

type CacheConfig struct {
    Position Position
    Type     string // redis, memory
    URL      string
    TTL      time.Duration
}

type AuthConfig struct {
    Position Position
    Type     string // jwt, api_key
    Issuer   string
    Audience string
    Secret   string // Environment variable reference
}

type CORSConfig struct {
    Position    Position
    Enabled     bool
    Origins     []string
    Methods     []string
    Headers     []string
    Credentials bool
}
```

### 3. Entity AST

```go
// Entity represents a data model
type Entity struct {
    Position    Position
    Name        string
    Description string
    Fields      []*Field
    Indexes     []*Index

    // Computed fields
    PrimaryKey  *Field
    SoftDelete  *Field
    Timestamps  []*Field
    References  []*Field
}

// Field represents an entity field
type Field struct {
    Position    Position
    Name        string
    Type        Type
    Required    bool
    Unique      bool
    Primary     bool
    Auto        bool
    AutoUpdate  bool
    Searchable  bool
    SoftDelete  bool
    Default     Expression
    Validators  []Validator

    // For reference fields
    Reference   *EntityRef
}

// EntityRef represents a foreign key reference
type EntityRef struct {
    EntityName string
    Entity     *Entity // Resolved reference
    OnDelete   string  // cascade, set_null, restrict
    OnUpdate   string
}

// Index represents a database index
type Index struct {
    Position Position
    Fields   []string
    Unique   bool
    Name     string // Generated or specified
}

// Validator represents a field validation rule
type Validator struct {
    Type  string // min, max, pattern, custom
    Value any
}
```

### 4. Type System

```go
// Type is the interface for all types
type Type interface {
    TypeName() string
    SQLType(dialect string) string
    GoType() string
    ZeroValue() any
    Validate(value any) error
}

// PrimitiveType represents basic types
type PrimitiveType struct {
    Name      string
    Precision int // For decimal
    Scale     int // For decimal
}

func (t *PrimitiveType) TypeName() string { return t.Name }

func (t *PrimitiveType) SQLType(dialect string) string {
    switch t.Name {
    case "uuid":
        return "UUID"
    case "string":
        return "VARCHAR(255)"
    case "text":
        return "TEXT"
    case "integer":
        return "BIGINT"
    case "decimal":
        return fmt.Sprintf("DECIMAL(%d,%d)", t.Precision, t.Scale)
    case "boolean":
        return "BOOLEAN"
    case "timestamp":
        return "TIMESTAMPTZ"
    case "date":
        return "DATE"
    case "time":
        return "TIME"
    case "json":
        if dialect == "postgres" {
            return "JSONB"
        }
        return "JSON"
    default:
        return "TEXT"
    }
}

// ListType represents array/list types
type ListType struct {
    ElementType Type
}

func (t *ListType) TypeName() string {
    return fmt.Sprintf("list(%s)", t.ElementType.TypeName())
}

// RefType represents foreign key references
type RefType struct {
    EntityName string
    Entity     *Entity // Resolved
}

func (t *RefType) TypeName() string {
    return fmt.Sprintf("ref(%s)", t.EntityName)
}

// EnumType represents enumerated values
type EnumType struct {
    Values []string
}

func (t *EnumType) TypeName() string {
    return fmt.Sprintf("enum(%s)", strings.Join(t.Values, ","))
}
```

### 5. Endpoint AST

```go
// Endpoint represents an HTTP API endpoint
type Endpoint struct {
    Position     Position
    Method       string // GET, POST, PUT, DELETE, PATCH
    Path         string
    Description  string

    // Authentication
    Auth         AuthRequirement
    Roles        []string

    // Parameters
    PathParams   []*Param
    QueryParams  []*Param
    Body         []*Param

    // Response
    Returns      *ReturnType
    Errors       []*ErrorResponse

    // Behavior
    Filter       *FilterExpr
    Sort         *SortExpr
    Cache        *CacheDirective
    Validate     []*ValidationRule
    OnSuccess    *Action

    // Computed
    RoutePattern string // Chi-compatible route pattern
    Entity       *Entity // Primary entity if applicable
}

type AuthRequirement string

const (
    AuthRequired AuthRequirement = "required"
    AuthOptional AuthRequirement = "optional"
    AuthPublic   AuthRequirement = "public"
)

type Param struct {
    Position   Position
    Name       string
    Type       Type
    Required   bool
    Default    Expression
    Validators []Validator
}

type ReturnType struct {
    Type       Type
    Paginated  bool
    EntityName string
}

type ErrorResponse struct {
    Status  int
    Message string
}

type CacheDirective struct {
    TTL time.Duration
    Key string
}

type FilterExpr struct {
    Conditions []*FilterCondition
}

type FilterCondition struct {
    Field     string
    Operator  string
    Value     Expression
    Condition string // "if provided"
}

type SortExpr struct {
    Field     string
    Direction string // asc, desc
}
```

### 6. Workflow AST

```go
// Workflow represents a multi-step process
type Workflow struct {
    Position    Position
    Name        string
    Description string
    Trigger     string // Event name
    TriggerEvent *Event // Resolved
    Steps       []*Step
    OnComplete  *Action
    OnFail      *Action
    Timeout     time.Duration
}

type Step struct {
    Position    Position
    Name        string
    Type        StepType

    // Conditional execution
    Condition   Expression
    ForEach     *ForEachExpr

    // Action
    Action      *Action
    Call        *IntegrationCall

    // Error handling
    Timeout     time.Duration
    Retry       *RetryConfig
    Compensate  *Action
    OnSuccess   *Action
    OnFail      *Action
}

type StepType string

const (
    StepTypeAction    StepType = "action"
    StepTypeCondition StepType = "condition"
    StepTypeLoop      StepType = "loop"
    StepTypeParallel  StepType = "parallel"
    StepTypeCall      StepType = "call"
)

type ForEachExpr struct {
    Variable string
    Source   Expression
}

type IntegrationCall struct {
    Integration   string
    Operation     string
    Params        map[string]Expression
    // Resolved
    IntegrationRef *Integration
    OperationRef   *Operation
}

type RetryConfig struct {
    Count    int
    Strategy string // fixed, exponential_backoff
    Delay    time.Duration
    MaxDelay time.Duration
}
```

### 7. Job AST

```go
// Job represents a scheduled background task
type Job struct {
    Position    Position
    Name        string
    Description string
    Schedule    Schedule
    Timezone    string
    Steps       []*Step
    Action      *Action // Simple single-action job
    Timeout     time.Duration
    Retry       int
    OnFail      *Action
}

type Schedule struct {
    Cron     string        // Cron expression
    Interval time.Duration // For "every X" syntax
}

func (s Schedule) String() string {
    if s.Cron != "" {
        return s.Cron
    }
    return fmt.Sprintf("every %s", s.Interval)
}
```

### 8. Integration AST

```go
// Integration represents an external service connection
type Integration struct {
    Position       Position
    Name           string
    Description    string
    Type           string // rest, grpc, graphql
    BaseURL        Expression
    Auth           *IntegrationAuth
    Headers        map[string]string
    Timeout        time.Duration
    Retry          *RetryConfig
    CircuitBreaker *CircuitBreakerConfig
    Operations     map[string]*Operation
}

type IntegrationAuth struct {
    Type   string // bearer, basic, api_key
    Token  Expression
    Header string // For api_key
}

type Operation struct {
    Position Position
    Name     string
    Method   string
    Path     string
    Body     []*Param
    Returns  []*Param
}

type CircuitBreakerConfig struct {
    Threshold  int
    Window     time.Duration
    ResetAfter time.Duration
}
```

### 9. Event AST

```go
// Event represents a message type
type Event struct {
    Position    Position
    Name        string
    Description string
    Payload     []*Param
    Trigger     *EventTrigger
    PublishTo   []*Publisher
}

type EventTrigger struct {
    Condition Expression
    Entity    string
    Field     string
}

type Publisher struct {
    Type   string // kafka, webhook, slack, email
    Target string
    Config map[string]string
}
```

### 10. Function AST

```go
// Function represents a reusable function
type Function struct {
    Position    Position
    Name        string
    Description string
    Params      []*Param
    ReturnType  Type
    Body        []Statement
}

type Statement interface {
    statementNode()
}

type LetStatement struct {
    Position Position
    Name     string
    Value    Expression
}

type ReturnStatement struct {
    Position Position
    Value    Expression
}

type IfStatement struct {
    Position  Position
    Condition Expression
    Then      []Statement
    Else      []Statement
}
```

### 11. Expression AST

```go
// Expression is the interface for all expressions
type Expression interface {
    expressionNode()
    Position() Position
    Type() Type // Resolved type
}

type BinaryExpr struct {
    Pos      Position
    Left     Expression
    Operator string
    Right    Expression
    ExprType Type
}

type UnaryExpr struct {
    Pos      Position
    Operator string
    Operand  Expression
    ExprType Type
}

type LiteralExpr struct {
    Pos      Position
    Value    any
    ExprType Type
}

type IdentifierExpr struct {
    Pos      Position
    Parts    []string
    ExprType Type
    // Resolved reference
    Ref      any
}

type CallExpr struct {
    Pos      Position
    Function string
    Args     []Expression
    ExprType Type
    // Resolved function
    FuncRef  *Function
}

type ListExpr struct {
    Pos      Position
    Elements []Expression
    ExprType Type
}

type LambdaExpr struct {
    Pos      Position
    Params   []string
    Body     Expression
    ExprType Type
}

type ConditionalExpr struct {
    Pos       Position
    Condition Expression
    Then      Expression
    Else      Expression
    ExprType  Type
}

type NullCoalesceExpr struct {
    Pos      Position
    Left     Expression
    Right    Expression
    ExprType Type
}
```

### 12. Action AST

```go
// Action represents a side effect
type Action struct {
    Position Position
    Type     ActionType

    // For update actions
    Target   string
    Field    string
    Value    Expression

    // For emit actions
    Event    string
    Payload  map[string]Expression

    // For send actions
    Channel  string
    Template string
    Data     map[string]Expression

    // For alert actions
    Team     string
    Message  Expression

    // Multiple actions
    Actions  []*Action
}

type ActionType string

const (
    ActionUpdate   ActionType = "update"
    ActionDelete   ActionType = "delete"
    ActionEmit     ActionType = "emit"
    ActionSend     ActionType = "send"
    ActionAlert    ActionType = "alert"
    ActionRollback ActionType = "rollback"
    ActionMultiple ActionType = "multiple"
)
```

## Acceptance Criteria
- [ ] All AST types are defined with proper documentation
- [ ] Type interface with SQLType(), GoType(), ZeroValue() methods
- [ ] Position tracking on all nodes
- [ ] Resolved references (Entity -> *Entity, etc.)
- [ ] JSON serialization works for debugging
- [ ] String() methods for human-readable output

## Implementation Steps
1. Define core types (Position, AST)
2. Define config types
3. Define entity and type system
4. Define endpoint types
5. Define workflow and step types
6. Define job types
7. Define integration types
8. Define event types
9. Define function types
10. Define expression types
11. Define action types
12. Add helper methods and constructors

## Testing Strategy
- Unit tests for type methods
- Serialization round-trip tests
- Position tracking tests

## Files to Create/Modify
- `internal/parser/ast.go`
- `internal/parser/types.go`
- `internal/parser/expressions.go`
- `internal/parser/ast_test.go`

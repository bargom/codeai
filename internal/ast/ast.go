// Package ast defines the Abstract Syntax Tree for CodeAI specifications.
// It provides node types for all language constructs including declarations,
// statements, and expressions.
package ast

import (
	"fmt"
	"strings"
)

// Node is the interface implemented by all AST nodes.
// Every node tracks its source position and provides type information.
type Node interface {
	// Pos returns the source position of the node
	Pos() Position
	// Type returns the node type enum value
	Type() NodeType
	// String returns a human-readable representation for debugging
	String() string
}

// Statement is a marker interface for statement nodes.
// Statements are executable constructs that don't produce values.
type Statement interface {
	Node
	stmtNode()
}

// Expression is a marker interface for expression nodes.
// Expressions are constructs that evaluate to values.
type Expression interface {
	Node
	exprNode()
}

// =============================================================================
// Program Node
// =============================================================================

// Program is the root node of the AST, containing all top-level statements.
type Program struct {
	pos        Position
	Statements []Statement
}

func (p *Program) Pos() Position  { return p.pos }
func (p *Program) Type() NodeType { return NodeProgram }
func (p *Program) String() string {
	var b strings.Builder
	b.WriteString("Program{\n")
	for _, stmt := range p.Statements {
		b.WriteString("  ")
		b.WriteString(stmt.String())
		b.WriteString("\n")
	}
	b.WriteString("}")
	return b.String()
}

// Application represents a structured view of the parsed CodeAI application.
// It organizes statements by type for easier access and validation.
type Application struct {
	pos         Position
	Config      *ConfigDecl       // Configuration block
	Database    *DatabaseBlock    // Database definition
	Endpoints   []*EndpointDecl   // API endpoints
	Middlewares []*MiddlewareDecl // Middleware definitions
	Auths       []*AuthDecl       // Authentication providers
	Roles       []*RoleDecl       // Role definitions
	Events      []*EventDecl      // Event definitions
	Handlers    []*EventHandlerDecl // Event handlers
	Integrations []*IntegrationDecl // External integrations
	Webhooks    []*WebhookDecl    // Webhook configurations
	Workflows   []*WorkflowDecl   // Temporal workflows
	Jobs        []*JobDecl        // Asynq jobs
	Variables   []*VarDecl        // Variable declarations
	Functions   []*FunctionDecl   // Function definitions
}

func (a *Application) Pos() Position  { return a.pos }
func (a *Application) Type() NodeType { return NodeProgram }
func (a *Application) String() string {
	return fmt.Sprintf("Application{Middlewares: %d, Auths: %d, Roles: %d, Endpoints: %d}",
		len(a.Middlewares), len(a.Auths), len(a.Roles), len(a.Endpoints))
}

// ToApplication converts a Program to a structured Application view.
// It organizes all statements by their type for easier access.
func (p *Program) ToApplication() *Application {
	app := &Application{
		pos: p.pos,
	}

	for _, stmt := range p.Statements {
		switch s := stmt.(type) {
		case *ConfigDecl:
			app.Config = s
		case *DatabaseBlock:
			app.Database = s
		case *EndpointDecl:
			app.Endpoints = append(app.Endpoints, s)
		case *MiddlewareDecl:
			app.Middlewares = append(app.Middlewares, s)
		case *AuthDecl:
			app.Auths = append(app.Auths, s)
		case *RoleDecl:
			app.Roles = append(app.Roles, s)
		case *EventDecl:
			app.Events = append(app.Events, s)
		case *EventHandlerDecl:
			app.Handlers = append(app.Handlers, s)
		case *IntegrationDecl:
			app.Integrations = append(app.Integrations, s)
		case *WebhookDecl:
			app.Webhooks = append(app.Webhooks, s)
		case *WorkflowDecl:
			app.Workflows = append(app.Workflows, s)
		case *JobDecl:
			app.Jobs = append(app.Jobs, s)
		case *VarDecl:
			app.Variables = append(app.Variables, s)
		case *FunctionDecl:
			app.Functions = append(app.Functions, s)
		}
	}

	return app
}

// =============================================================================
// Statement Nodes
// =============================================================================

// VarDecl represents a variable declaration: `let x = value`
type VarDecl struct {
	pos   Position
	Name  string
	Value Expression
}

func (v *VarDecl) Pos() Position  { return v.pos }
func (v *VarDecl) Type() NodeType { return NodeVarDecl }
func (v *VarDecl) stmtNode()      {}
func (v *VarDecl) String() string {
	return fmt.Sprintf("VarDecl{Name: %q, Value: %s}", v.Name, v.Value.String())
}

// Assignment represents an assignment statement: `x = value`
type Assignment struct {
	pos   Position
	Name  string
	Value Expression
}

func (a *Assignment) Pos() Position  { return a.pos }
func (a *Assignment) Type() NodeType { return NodeAssignment }
func (a *Assignment) stmtNode()      {}
func (a *Assignment) String() string {
	return fmt.Sprintf("Assignment{Name: %q, Value: %s}", a.Name, a.Value.String())
}

// IfStmt represents an if/else conditional statement.
type IfStmt struct {
	pos       Position
	Condition Expression
	Then      *Block
	Else      *Block // may be nil
}

func (i *IfStmt) Pos() Position  { return i.pos }
func (i *IfStmt) Type() NodeType { return NodeIfStmt }
func (i *IfStmt) stmtNode()      {}
func (i *IfStmt) String() string {
	var b strings.Builder
	b.WriteString("IfStmt{Condition: ")
	b.WriteString(i.Condition.String())
	b.WriteString(", Then: ")
	b.WriteString(i.Then.String())
	if i.Else != nil {
		b.WriteString(", Else: ")
		b.WriteString(i.Else.String())
	}
	b.WriteString("}")
	return b.String()
}

// ForLoop represents a for-in loop: `for item in items { ... }`
type ForLoop struct {
	pos      Position
	Variable string
	Iterable Expression
	Body     *Block
}

func (f *ForLoop) Pos() Position  { return f.pos }
func (f *ForLoop) Type() NodeType { return NodeForLoop }
func (f *ForLoop) stmtNode()      {}
func (f *ForLoop) String() string {
	return fmt.Sprintf("ForLoop{Variable: %q, Iterable: %s, Body: %s}",
		f.Variable, f.Iterable.String(), f.Body.String())
}

// FunctionDecl represents a function declaration.
type FunctionDecl struct {
	pos    Position
	Name   string
	Params []Parameter
	Body   *Block
}

func (f *FunctionDecl) Pos() Position  { return f.pos }
func (f *FunctionDecl) Type() NodeType { return NodeFunctionDecl }
func (f *FunctionDecl) stmtNode()      {}
func (f *FunctionDecl) String() string {
	params := make([]string, len(f.Params))
	for i, p := range f.Params {
		params[i] = p.Name
	}
	return fmt.Sprintf("FunctionDecl{Name: %q, Params: [%s], Body: %s}",
		f.Name, strings.Join(params, ", "), f.Body.String())
}

// Parameter represents a function parameter.
type Parameter struct {
	Name    string
	Type    string
	Default Expression // may be nil
}

// ExecBlock represents a shell command execution block.
type ExecBlock struct {
	pos     Position
	Command string
}

func (e *ExecBlock) Pos() Position  { return e.pos }
func (e *ExecBlock) Type() NodeType { return NodeExecBlock }
func (e *ExecBlock) stmtNode()      {}
func (e *ExecBlock) String() string {
	return fmt.Sprintf("ExecBlock{Command: %q}", e.Command)
}

// Block represents a block of statements enclosed in braces.
type Block struct {
	pos        Position
	Statements []Statement
}

func (b *Block) Pos() Position  { return b.pos }
func (b *Block) Type() NodeType { return NodeBlock }
func (b *Block) stmtNode()      {}
func (b *Block) String() string {
	var sb strings.Builder
	sb.WriteString("Block{")
	for i, stmt := range b.Statements {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(stmt.String())
	}
	sb.WriteString("}")
	return sb.String()
}

// ReturnStmt represents a return statement.
type ReturnStmt struct {
	pos   Position
	Value Expression // may be nil for bare return
}

func (r *ReturnStmt) Pos() Position  { return r.pos }
func (r *ReturnStmt) Type() NodeType { return NodeReturnStmt }
func (r *ReturnStmt) stmtNode()      {}
func (r *ReturnStmt) String() string {
	if r.Value == nil {
		return "ReturnStmt{}"
	}
	return fmt.Sprintf("ReturnStmt{Value: %s}", r.Value.String())
}

// =============================================================================
// Expression Nodes
// =============================================================================

// StringLiteral represents a string literal value.
type StringLiteral struct {
	pos   Position
	Value string
}

func (s *StringLiteral) Pos() Position  { return s.pos }
func (s *StringLiteral) Type() NodeType { return NodeStringLiteral }
func (s *StringLiteral) exprNode()      {}
func (s *StringLiteral) String() string {
	return fmt.Sprintf("StringLiteral{%q}", s.Value)
}

// NumberLiteral represents a numeric literal value (integer or float).
type NumberLiteral struct {
	pos   Position
	Value float64
}

func (n *NumberLiteral) Pos() Position  { return n.pos }
func (n *NumberLiteral) Type() NodeType { return NodeNumberLiteral }
func (n *NumberLiteral) exprNode()      {}
func (n *NumberLiteral) String() string {
	return fmt.Sprintf("NumberLiteral{%v}", n.Value)
}

// BoolLiteral represents a boolean literal (true or false).
type BoolLiteral struct {
	pos   Position
	Value bool
}

func (b *BoolLiteral) Pos() Position  { return b.pos }
func (b *BoolLiteral) Type() NodeType { return NodeBoolLiteral }
func (b *BoolLiteral) exprNode()      {}
func (b *BoolLiteral) String() string {
	return fmt.Sprintf("BoolLiteral{%v}", b.Value)
}

// Identifier represents a variable or function name reference.
type Identifier struct {
	pos  Position
	Name string
}

func (i *Identifier) Pos() Position  { return i.pos }
func (i *Identifier) Type() NodeType { return NodeIdentifier }
func (i *Identifier) exprNode()      {}
func (i *Identifier) String() string {
	return fmt.Sprintf("Identifier{%q}", i.Name)
}

// FunctionCall represents a function invocation with arguments.
type FunctionCall struct {
	pos  Position
	Name string
	Args []Expression
}

func (f *FunctionCall) Pos() Position  { return f.pos }
func (f *FunctionCall) Type() NodeType { return NodeFunctionCall }
func (f *FunctionCall) exprNode()      {}
func (f *FunctionCall) String() string {
	args := make([]string, len(f.Args))
	for i, arg := range f.Args {
		args[i] = arg.String()
	}
	return fmt.Sprintf("FunctionCall{Name: %q, Args: [%s]}", f.Name, strings.Join(args, ", "))
}

// ArrayLiteral represents an array literal: [a, b, c]
type ArrayLiteral struct {
	pos      Position
	Elements []Expression
}

func (a *ArrayLiteral) Pos() Position  { return a.pos }
func (a *ArrayLiteral) Type() NodeType { return NodeArrayLiteral }
func (a *ArrayLiteral) exprNode()      {}
func (a *ArrayLiteral) String() string {
	elems := make([]string, len(a.Elements))
	for i, elem := range a.Elements {
		elems[i] = elem.String()
	}
	return fmt.Sprintf("ArrayLiteral{[%s]}", strings.Join(elems, ", "))
}

// BinaryExpr represents a binary operation: a + b, x == y, etc.
type BinaryExpr struct {
	pos      Position
	Left     Expression
	Operator string
	Right    Expression
}

func (b *BinaryExpr) Pos() Position  { return b.pos }
func (b *BinaryExpr) Type() NodeType { return NodeBinaryExpr }
func (b *BinaryExpr) exprNode()      {}
func (b *BinaryExpr) String() string {
	return fmt.Sprintf("BinaryExpr{Left: %s, Op: %q, Right: %s}",
		b.Left.String(), b.Operator, b.Right.String())
}

// UnaryExpr represents a unary operation: -x, not flag
type UnaryExpr struct {
	pos      Position
	Operator string
	Operand  Expression
}

func (u *UnaryExpr) Pos() Position  { return u.pos }
func (u *UnaryExpr) Type() NodeType { return NodeUnaryExpr }
func (u *UnaryExpr) exprNode()      {}
func (u *UnaryExpr) String() string {
	return fmt.Sprintf("UnaryExpr{Op: %q, Operand: %s}", u.Operator, u.Operand.String())
}

// =============================================================================
// Configuration Nodes
// =============================================================================

// DatabaseType represents the supported database backends.
type DatabaseType string

const (
	DatabaseTypePostgres DatabaseType = "postgres"
	DatabaseTypeMongoDB  DatabaseType = "mongodb"
)

// ConfigDecl represents a config block declaration.
// Example: config { database_type: "mongodb" }
type ConfigDecl struct {
	pos          Position
	DatabaseType DatabaseType // "postgres" or "mongodb"
	MongoDBURI   string       // MongoDB connection URI
	MongoDBName  string       // MongoDB database name
	Properties   map[string]Expression
}

func (c *ConfigDecl) Pos() Position  { return c.pos }
func (c *ConfigDecl) Type() NodeType { return NodeConfigDecl }
func (c *ConfigDecl) stmtNode()      {}
func (c *ConfigDecl) String() string {
	return fmt.Sprintf("ConfigDecl{DatabaseType: %q, MongoDBURI: %q, MongoDBName: %q}",
		c.DatabaseType, c.MongoDBURI, c.MongoDBName)
}

// DatabaseBlock represents a database definition block.
// Example: database postgres { table users { ... } }
// Example: database mongodb { collection users { ... } }
type DatabaseBlock struct {
	pos        Position
	DBType     DatabaseType // "postgres" or "mongodb"
	Name       string       // optional name for the database
	Statements []Statement
}

func (d *DatabaseBlock) Pos() Position  { return d.pos }
func (d *DatabaseBlock) Type() NodeType { return NodeDatabaseBlock }
func (d *DatabaseBlock) stmtNode()      {}
func (d *DatabaseBlock) String() string {
	return fmt.Sprintf("DatabaseBlock{DBType: %q, Name: %q}", d.DBType, d.Name)
}

// =============================================================================
// PostgreSQL Model Nodes
// =============================================================================

// ModelDecl represents a PostgreSQL model declaration.
// Example: model User { id: uuid, primary, auto ... }
type ModelDecl struct {
	pos         Position
	Name        string
	Description string
	Fields      []*FieldDecl
	Indexes     []*IndexDecl
}

func (m *ModelDecl) Pos() Position  { return m.pos }
func (m *ModelDecl) Type() NodeType { return NodeModelDecl }
func (m *ModelDecl) stmtNode()      {}
func (m *ModelDecl) String() string {
	return fmt.Sprintf("ModelDecl{Name: %q, Fields: %d, Indexes: %d}",
		m.Name, len(m.Fields), len(m.Indexes))
}

// FieldDecl represents a field in a PostgreSQL model.
type FieldDecl struct {
	pos       Position
	Name      string
	FieldType *TypeRef
	Modifiers []*Modifier
}

func (f *FieldDecl) Pos() Position  { return f.pos }
func (f *FieldDecl) Type() NodeType { return NodeFieldDecl }
func (f *FieldDecl) stmtNode()      {}
func (f *FieldDecl) String() string {
	return fmt.Sprintf("FieldDecl{Name: %q, Type: %s}", f.Name, f.FieldType.String())
}

// TypeRef represents a type reference (e.g., string, uuid, ref(User), list(string)).
type TypeRef struct {
	pos    Position
	Name   string
	Params []*TypeRef
}

func (t *TypeRef) Pos() Position  { return t.pos }
func (t *TypeRef) Type() NodeType { return NodeTypeRef }
func (t *TypeRef) String() string {
	if len(t.Params) == 0 {
		return t.Name
	}
	params := make([]string, len(t.Params))
	for i, p := range t.Params {
		params[i] = p.String()
	}
	return fmt.Sprintf("%s(%s)", t.Name, strings.Join(params, ", "))
}

// Modifier represents a field modifier (e.g., required, unique, default(value)).
type Modifier struct {
	pos   Position
	Name  string
	Value Expression // may be nil
}

func (m *Modifier) Pos() Position  { return m.pos }
func (m *Modifier) Type() NodeType { return NodeModifier }
func (m *Modifier) String() string {
	if m.Value != nil {
		return fmt.Sprintf("%s(%s)", m.Name, m.Value.String())
	}
	return m.Name
}

// IndexDecl represents a PostgreSQL index declaration.
type IndexDecl struct {
	pos    Position
	Fields []string
	Unique bool
}

func (i *IndexDecl) Pos() Position  { return i.pos }
func (i *IndexDecl) Type() NodeType { return NodeIndexDecl }
func (i *IndexDecl) stmtNode()      {}
func (i *IndexDecl) String() string {
	return fmt.Sprintf("IndexDecl{Fields: %v, Unique: %t}", i.Fields, i.Unique)
}

// =============================================================================
// MongoDB Collection Nodes
// =============================================================================

// CollectionDecl represents a MongoDB collection declaration.
// Example: collection User { _id: objectid, primary ... }
type CollectionDecl struct {
	pos         Position
	Name        string
	Description string
	Fields      []*MongoFieldDecl
	Indexes     []*MongoIndexDecl
}

func (c *CollectionDecl) Pos() Position  { return c.pos }
func (c *CollectionDecl) Type() NodeType { return NodeCollectionDecl }
func (c *CollectionDecl) stmtNode()      {}
func (c *CollectionDecl) String() string {
	return fmt.Sprintf("CollectionDecl{Name: %q, Fields: %d, Indexes: %d}",
		c.Name, len(c.Fields), len(c.Indexes))
}

// MongoFieldDecl represents a field in a MongoDB collection.
type MongoFieldDecl struct {
	pos         Position
	Name        string
	FieldType   *MongoTypeRef
	Modifiers   []*Modifier
}

func (f *MongoFieldDecl) Pos() Position  { return f.pos }
func (f *MongoFieldDecl) Type() NodeType { return NodeMongoFieldDecl }
func (f *MongoFieldDecl) stmtNode()      {}
func (f *MongoFieldDecl) String() string {
	return fmt.Sprintf("MongoFieldDecl{Name: %q, Type: %s}", f.Name, f.FieldType.String())
}

// MongoTypeRef represents a MongoDB-specific type reference.
// Supports: objectid, string, int, double, bool, date, binary, array(T), embedded { ... }
type MongoTypeRef struct {
	pos         Position
	Name        string            // e.g., "objectid", "string", "array"
	Params      []string          // e.g., for array(string) -> ["string"]
	EmbeddedDoc *EmbeddedDocDecl  // for embedded document types
}

func (t *MongoTypeRef) Pos() Position  { return t.pos }
func (t *MongoTypeRef) Type() NodeType { return NodeMongoTypeRef }
func (t *MongoTypeRef) String() string {
	if t.EmbeddedDoc != nil {
		return fmt.Sprintf("embedded{%d fields}", len(t.EmbeddedDoc.Fields))
	}
	if len(t.Params) == 0 {
		return t.Name
	}
	return fmt.Sprintf("%s(%s)", t.Name, strings.Join(t.Params, ", "))
}

// EmbeddedDocDecl represents an embedded document type in MongoDB.
type EmbeddedDocDecl struct {
	pos    Position
	Fields []*MongoFieldDecl
}

func (e *EmbeddedDocDecl) Pos() Position  { return e.pos }
func (e *EmbeddedDocDecl) Type() NodeType { return NodeEmbeddedDocDecl }
func (e *EmbeddedDocDecl) String() string {
	return fmt.Sprintf("EmbeddedDocDecl{Fields: %d}", len(e.Fields))
}

// MongoIndexDecl represents a MongoDB index declaration.
// Supports: single, compound, text, geospatial indexes
type MongoIndexDecl struct {
	pos       Position
	Fields    []string
	Unique    bool
	IndexKind string // "", "text", "geospatial"
}

func (i *MongoIndexDecl) Pos() Position  { return i.pos }
func (i *MongoIndexDecl) Type() NodeType { return NodeMongoIndexDecl }
func (i *MongoIndexDecl) stmtNode()      {}
func (i *MongoIndexDecl) String() string {
	kind := "single"
	if len(i.Fields) > 1 {
		kind = "compound"
	}
	if i.IndexKind != "" {
		kind = i.IndexKind
	}
	return fmt.Sprintf("MongoIndexDecl{Type: %q, Fields: %v, Unique: %t}", kind, i.Fields, i.Unique)
}
// =============================================================================
// Authentication & Authorization Nodes
// =============================================================================

// AuthMethod represents the supported authentication methods.
type AuthMethod string

const (
	AuthMethodJWT    AuthMethod = "jwt"
	AuthMethodOAuth2 AuthMethod = "oauth2"
	AuthMethodAPIKey AuthMethod = "apikey"
	AuthMethodBasic  AuthMethod = "basic"
)

// AuthDecl represents an authentication provider declaration.
type AuthDecl struct {
	pos    Position
	Name   string
	Method AuthMethod
	JWKS   *JWKSConfig
	Config map[string]Expression
}

func (a *AuthDecl) Pos() Position  { return a.pos }
func (a *AuthDecl) Type() NodeType { return NodeAuthDecl }
func (a *AuthDecl) stmtNode()      {}
func (a *AuthDecl) String() string {
	return fmt.Sprintf("AuthDecl{Name: %q, Method: %q}", a.Name, a.Method)
}

// JWKSConfig holds JWKS-specific configuration for JWT authentication.
type JWKSConfig struct {
	pos      Position
	URL      string
	Issuer   string
	Audience string
}

func (j *JWKSConfig) Pos() Position  { return j.pos }
func (j *JWKSConfig) Type() NodeType { return NodeJWKSConfig }
func (j *JWKSConfig) String() string {
	return fmt.Sprintf("JWKSConfig{URL: %q}", j.URL)
}

// RoleDecl represents a role definition with permissions.
type RoleDecl struct {
	pos         Position
	Name        string
	Permissions []string
}

func (r *RoleDecl) Pos() Position  { return r.pos }
func (r *RoleDecl) Type() NodeType { return NodeRoleDecl }
func (r *RoleDecl) stmtNode()      {}
func (r *RoleDecl) String() string {
	return fmt.Sprintf("RoleDecl{Name: %q}", r.Name)
}

// =============================================================================
// Middleware Nodes
// =============================================================================

// MiddlewareDecl represents a middleware definition.
type MiddlewareDecl struct {
	pos            Position
	Name           string
	MiddlewareType string
	Config         map[string]Expression
}

func (m *MiddlewareDecl) Pos() Position  { return m.pos }
func (m *MiddlewareDecl) Type() NodeType { return NodeMiddlewareDecl }
func (m *MiddlewareDecl) stmtNode()      {}
func (m *MiddlewareDecl) String() string {
	return fmt.Sprintf("MiddlewareDecl{Name: %q, Type: %q}", m.Name, m.MiddlewareType)
}

// MiddlewareRef represents a reference to a middleware.
type MiddlewareRef struct {
	pos  Position
	Name string
}

func (m *MiddlewareRef) Pos() Position  { return m.pos }
func (m *MiddlewareRef) Type() NodeType { return NodeMiddlewareRef }
func (m *MiddlewareRef) String() string {
	return fmt.Sprintf("MiddlewareRef{Name: %q}", m.Name)
}

// RateLimitMiddleware represents a rate limiting middleware configuration.
type RateLimitMiddleware struct {
	pos       Position
	Requests  int
	Window    string
	Strategy  string
}

func (r *RateLimitMiddleware) Pos() Position  { return r.pos }
func (r *RateLimitMiddleware) Type() NodeType { return NodeRateLimitMiddleware }
func (r *RateLimitMiddleware) String() string {
	return fmt.Sprintf("RateLimitMiddleware{Requests: %d, Window: %q, Strategy: %q}", r.Requests, r.Window, r.Strategy)
}

// =============================================================================
// Workflow and Job Nodes
// =============================================================================

// TriggerType represents the type of workflow trigger.
type TriggerType string

const (
	TriggerTypeEvent    TriggerType = "event"
	TriggerTypeSchedule TriggerType = "schedule"
	TriggerTypeManual   TriggerType = "manual"
)

// WorkflowDecl represents a Temporal workflow declaration.
// Example: workflow order_fulfillment { trigger event "order.created" ... }
type WorkflowDecl struct {
	pos     Position
	Name    string
	Trigger *Trigger
	Steps   []*WorkflowStep
	Retry   *RetryPolicyDecl
	Timeout string
}

func (w *WorkflowDecl) Pos() Position  { return w.pos }
func (w *WorkflowDecl) Type() NodeType { return NodeWorkflowDecl }
func (w *WorkflowDecl) stmtNode()      {}
func (w *WorkflowDecl) String() string {
	return fmt.Sprintf("WorkflowDecl{Name: %q, Trigger: %s, Steps: %d}",
		w.Name, w.Trigger.String(), len(w.Steps))
}

// Trigger represents a workflow trigger configuration.
type Trigger struct {
	pos      Position
	TrigType TriggerType
	Value    string
}

func (t *Trigger) Pos() Position  { return t.pos }
func (t *Trigger) Type() NodeType { return NodeTrigger }
func (t *Trigger) String() string {
	return fmt.Sprintf("Trigger{Type: %q, Value: %q}", t.TrigType, t.Value)
}

// WorkflowStep represents a step in a workflow.
type WorkflowStep struct {
	pos       Position
	Name      string
	Activity  string
	Input     []*InputMapping
	Condition string
	Parallel  bool
	Steps     []*WorkflowStep // For parallel blocks containing nested steps
}

func (s *WorkflowStep) Pos() Position  { return s.pos }
func (s *WorkflowStep) Type() NodeType { return NodeWorkflowStep }
func (s *WorkflowStep) String() string {
	if s.Parallel {
		return fmt.Sprintf("WorkflowStep{Parallel: %d steps}", len(s.Steps))
	}
	return fmt.Sprintf("WorkflowStep{Name: %q, Activity: %q}", s.Name, s.Activity)
}

// InputMapping represents a key-value mapping for step input.
type InputMapping struct {
	pos   Position
	Key   string
	Value string
}

func (i *InputMapping) Pos() Position  { return i.pos }
func (i *InputMapping) Type() NodeType { return NodeInputMapping }
func (i *InputMapping) String() string {
	return fmt.Sprintf("InputMapping{%q: %q}", i.Key, i.Value)
}

// RetryPolicyDecl represents retry configuration for workflows and jobs.
type RetryPolicyDecl struct {
	pos               Position
	MaxAttempts       int
	InitialInterval   string
	BackoffMultiplier float64
}

func (r *RetryPolicyDecl) Pos() Position  { return r.pos }
func (r *RetryPolicyDecl) Type() NodeType { return NodeRetryPolicy }
func (r *RetryPolicyDecl) String() string {
	return fmt.Sprintf("RetryPolicyDecl{MaxAttempts: %d, InitialInterval: %q, Backoff: %.1f}",
		r.MaxAttempts, r.InitialInterval, r.BackoffMultiplier)
}

// JobDecl represents an Asynq job declaration.
// Example: job cleanup_logs { schedule "0 0 * * 0" task "maintenance.cleanup" ... }
type JobDecl struct {
	pos      Position
	Name     string
	Schedule string
	Task     string
	Queue    string
	Retry    *RetryPolicyDecl
}

func (j *JobDecl) Pos() Position  { return j.pos }
func (j *JobDecl) Type() NodeType { return NodeJobDecl }
func (j *JobDecl) stmtNode()      {}
func (j *JobDecl) String() string {
	return fmt.Sprintf("JobDecl{Name: %q, Schedule: %q, Task: %q, Queue: %q}",
		j.Name, j.Schedule, j.Task, j.Queue)
}

// =============================================================================
// Endpoint Nodes
// =============================================================================

// HTTPMethod represents the HTTP method for an endpoint.
type HTTPMethod string

const (
	HTTPMethodGET    HTTPMethod = "GET"
	HTTPMethodPOST   HTTPMethod = "POST"
	HTTPMethodPUT    HTTPMethod = "PUT"
	HTTPMethodDELETE HTTPMethod = "DELETE"
	HTTPMethodPATCH  HTTPMethod = "PATCH"
)

// RequestSource represents where request data comes from.
type RequestSource string

const (
	RequestSourceBody   RequestSource = "body"
	RequestSourceQuery  RequestSource = "query"
	RequestSourcePath   RequestSource = "path"
	RequestSourceHeader RequestSource = "header"
)

// EndpointDecl represents an endpoint declaration.
// Example: endpoint GET "/users/:id" { middleware auth require_role "admin" request User from path response UserDetail status 200 }
type EndpointDecl struct {
	pos         Position
	Method      HTTPMethod
	Path        string
	Handler     *Handler
	Middlewares []*MiddlewareRef
	RequireRole string
	Annotations []*Annotation
}

func (e *EndpointDecl) Pos() Position  { return e.pos }
func (e *EndpointDecl) Type() NodeType { return NodeEndpointDecl }
func (e *EndpointDecl) stmtNode()      {}
func (e *EndpointDecl) String() string {
	return fmt.Sprintf("EndpointDecl{Method: %q, Path: %q}", e.Method, e.Path)
}

// Handler represents the handler block inside an endpoint.
type Handler struct {
	pos      Position
	Request  *RequestType
	Response *ResponseType
	Logic    *HandlerLogic
}

func (h *Handler) Pos() Position  { return h.pos }
func (h *Handler) Type() NodeType { return NodeHandler }
func (h *Handler) String() string {
	return "Handler{}"
}

// RequestType defines the request type and source for an endpoint.
// Example: request User from body
type RequestType struct {
	pos      Position
	TypeName string
	Source   RequestSource
}

func (r *RequestType) Pos() Position  { return r.pos }
func (r *RequestType) Type() NodeType { return NodeRequestType }
func (r *RequestType) String() string {
	return fmt.Sprintf("RequestType{Type: %q, Source: %q}", r.TypeName, r.Source)
}

// ResponseType defines the response type and status code for an endpoint.
// Example: response UserDetail status 200
type ResponseType struct {
	pos        Position
	TypeName   string
	StatusCode int
}

func (r *ResponseType) Pos() Position  { return r.pos }
func (r *ResponseType) Type() NodeType { return NodeResponseType }
func (r *ResponseType) String() string {
	return fmt.Sprintf("ResponseType{Type: %q, Status: %d}", r.TypeName, r.StatusCode)
}

// HandlerLogic represents the logic block inside an endpoint handler.
// Example: do { validate(request) ... }
type HandlerLogic struct {
	pos   Position
	Steps []*LogicStep
}

func (l *HandlerLogic) Pos() Position  { return l.pos }
func (l *HandlerLogic) Type() NodeType { return NodeHandlerLogic }
func (l *HandlerLogic) String() string {
	return fmt.Sprintf("HandlerLogic{Steps: %d}", len(l.Steps))
}

// LogicStep represents a single step in handler logic.
// Example: validate(request), authorize(request, "admin"), user = db.find(User, request.id)
type LogicStep struct {
	pos        Position
	Action     string
	Target     string // optional: for assignment
	Args       []string
	Condition  string // optional: for 'where' clauses
	Options    []*Option
}

func (s *LogicStep) Pos() Position  { return s.pos }
func (s *LogicStep) Type() NodeType { return NodeLogicStep }
func (s *LogicStep) String() string {
	if s.Target != "" {
		return fmt.Sprintf("LogicStep{Target: %q, Action: %q}", s.Target, s.Action)
	}
	return fmt.Sprintf("LogicStep{Action: %q}", s.Action)
}

// Option represents a key-value option in a logic step.
// Example: with { cache: true, ttl: 300 }
type Option struct {
	pos   Position
	Key   string
	Value Expression
}

func (o *Option) Pos() Position  { return o.pos }
func (o *Option) Type() NodeType { return NodeOption }
func (o *Option) String() string {
	return fmt.Sprintf("Option{Key: %q}", o.Key)
}

// Annotation represents a metadata annotation on an endpoint.
// Example: @deprecated, @auth("admin")
type Annotation struct {
	pos   Position
	Name  string
	Value string
}

func (a *Annotation) Pos() Position  { return a.pos }
func (a *Annotation) Type() NodeType { return NodeAnnotation }
func (a *Annotation) String() string {
	if a.Value != "" {
		return fmt.Sprintf("Annotation{@%s(%q)}", a.Name, a.Value)
	}
	return fmt.Sprintf("Annotation{@%s}", a.Name)
}

// =============================================================================
// Event Nodes
// =============================================================================

// EventDecl represents an event definition.
// Example: event user.created { schema { user_id string ... } }
type EventDecl struct {
	pos      Position
	Name     string              // Event name (e.g., "user.created")
	Schema   *EventSchema        // Event payload schema
	Handlers []*EventHandlerDecl // Handlers registered for this event
}

func (e *EventDecl) Pos() Position  { return e.pos }
func (e *EventDecl) Type() NodeType { return NodeEventDecl }
func (e *EventDecl) stmtNode()      {}
func (e *EventDecl) String() string {
	return fmt.Sprintf("EventDecl{Name: %q, Handlers: %d}", e.Name, len(e.Handlers))
}

// EventSchema represents the schema definition for an event payload.
type EventSchema struct {
	pos    Position
	Fields []*EventSchemaField
}

func (s *EventSchema) Pos() Position  { return s.pos }
func (s *EventSchema) Type() NodeType { return NodeEventSchema }
func (s *EventSchema) String() string {
	return fmt.Sprintf("EventSchema{Fields: %d}", len(s.Fields))
}

// EventSchemaField represents a field in an event schema.
type EventSchemaField struct {
	pos       Position
	Name      string
	FieldType string // e.g., "string", "timestamp", "decimal", "array"
}

func (f *EventSchemaField) Pos() Position  { return f.pos }
func (f *EventSchemaField) Type() NodeType { return NodeEventSchemaField }
func (f *EventSchemaField) String() string {
	return fmt.Sprintf("EventSchemaField{Name: %q, Type: %q}", f.Name, f.FieldType)
}

// EventHandlerDecl represents an event handler declaration.
// Example: on "user.created" do workflow "send_welcome_email" async
type EventHandlerDecl struct {
	pos        Position
	EventName  string // Event to listen for
	ActionType string // "workflow", "integration", "emit", "webhook"
	Target     string // Target name (workflow name, integration name, etc.)
	Async      bool   // Whether handler runs asynchronously
}

func (h *EventHandlerDecl) Pos() Position  { return h.pos }
func (h *EventHandlerDecl) Type() NodeType { return NodeEventHandler }
func (h *EventHandlerDecl) stmtNode()      {}
func (h *EventHandlerDecl) String() string {
	asyncStr := ""
	if h.Async {
		asyncStr = " async"
	}
	return fmt.Sprintf("EventHandlerDecl{Event: %q, Action: %q, Target: %q%s}",
		h.EventName, h.ActionType, h.Target, asyncStr)
}

// =============================================================================
// Integration Nodes
// =============================================================================

// IntegrationType represents the type of external integration.
type IntegrationType string

const (
	IntegrationTypeREST    IntegrationType = "rest"
	IntegrationTypeGraphQL IntegrationType = "graphql"
	IntegrationTypeGRPC    IntegrationType = "grpc"
	IntegrationTypeWebhook IntegrationType = "webhook"
)

// IntegrationAuthType represents the authentication method for integrations.
type IntegrationAuthType string

const (
	IntegrationAuthBearer IntegrationAuthType = "bearer"
	IntegrationAuthBasic  IntegrationAuthType = "basic"
	IntegrationAuthAPIKey IntegrationAuthType = "apikey"
	IntegrationAuthOAuth2 IntegrationAuthType = "oauth2"
)

// IntegrationDecl represents an external API integration declaration.
// Example: integration stripe { type rest base_url "..." auth bearer { ... } }
type IntegrationDecl struct {
	pos            Position
	Name           string                 // Integration name
	IntgType       IntegrationType        // rest, graphql, grpc, webhook
	BaseURL        string                 // Base URL for the API
	Auth           *IntegrationAuthDecl   // Authentication config
	Timeout        string                 // Request timeout (e.g., "30s")
	CircuitBreaker *CircuitBreakerConfig  // Circuit breaker configuration
}

func (i *IntegrationDecl) Pos() Position  { return i.pos }
func (i *IntegrationDecl) Type() NodeType { return NodeIntegrationDecl }
func (i *IntegrationDecl) stmtNode()      {}
func (i *IntegrationDecl) String() string {
	return fmt.Sprintf("IntegrationDecl{Name: %q, Type: %q, BaseURL: %q}",
		i.Name, i.IntgType, i.BaseURL)
}

// IntegrationAuthDecl represents authentication configuration for an integration.
// Example: auth bearer { token env("API_KEY") }
type IntegrationAuthDecl struct {
	pos      Position
	AuthType IntegrationAuthType   // bearer, basic, apikey, oauth2
	Config   map[string]Expression // Auth-specific config (token, header, value, etc.)
}

func (a *IntegrationAuthDecl) Pos() Position  { return a.pos }
func (a *IntegrationAuthDecl) Type() NodeType { return NodeIntegrationAuth }
func (a *IntegrationAuthDecl) String() string {
	return fmt.Sprintf("IntegrationAuthDecl{Type: %q}", a.AuthType)
}

// CircuitBreakerConfig represents circuit breaker configuration.
// Example: circuit_breaker { threshold 5 timeout "60s" max_concurrent 100 }
type CircuitBreakerConfig struct {
	pos              Position
	FailureThreshold int    // Number of failures before opening
	Timeout          string // How long to stay open before half-open
	MaxConcurrent    int    // Maximum concurrent requests
}

func (c *CircuitBreakerConfig) Pos() Position  { return c.pos }
func (c *CircuitBreakerConfig) Type() NodeType { return NodeCircuitBreakerConfig }
func (c *CircuitBreakerConfig) String() string {
	return fmt.Sprintf("CircuitBreakerConfig{Threshold: %d, Timeout: %q, MaxConcurrent: %d}",
		c.FailureThreshold, c.Timeout, c.MaxConcurrent)
}

// =============================================================================
// Webhook Nodes
// =============================================================================

// WebhookHTTPMethod represents the HTTP method for a webhook.
type WebhookHTTPMethod string

const (
	WebhookMethodPOST WebhookHTTPMethod = "POST"
	WebhookMethodPUT  WebhookHTTPMethod = "PUT"
)

// WebhookDecl represents a webhook configuration.
// Example: webhook analytics { event "order.completed" url "..." method POST ... }
type WebhookDecl struct {
	pos     Position
	Name    string            // Webhook name
	Event   string            // Event that triggers this webhook
	URL     string            // Target URL
	Method  WebhookHTTPMethod // HTTP method (POST, PUT)
	Headers []*WebhookHeader  // Custom headers
	Retry   *RetryPolicyDecl  // Retry configuration
}

func (w *WebhookDecl) Pos() Position  { return w.pos }
func (w *WebhookDecl) Type() NodeType { return NodeWebhookDecl }
func (w *WebhookDecl) stmtNode()      {}
func (w *WebhookDecl) String() string {
	return fmt.Sprintf("WebhookDecl{Name: %q, Event: %q, URL: %q, Method: %q}",
		w.Name, w.Event, w.URL, w.Method)
}

// WebhookHeader represents a custom header for a webhook.
type WebhookHeader struct {
	pos   Position
	Key   string
	Value string
}

func (h *WebhookHeader) Pos() Position  { return h.pos }
func (h *WebhookHeader) Type() NodeType { return NodeWebhookHeader }
func (h *WebhookHeader) String() string {
	return fmt.Sprintf("WebhookHeader{%q: %q}", h.Key, h.Value)
}

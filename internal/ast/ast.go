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
	Type       DatabaseType // "postgres" or "mongodb"
	Name       string       // optional name for the database
	Statements []Statement
}

func (d *DatabaseBlock) Pos() Position  { return d.pos }
func (d *DatabaseBlock) Type() NodeType { return NodeDatabaseBlock }
func (d *DatabaseBlock) stmtNode()      {}
func (d *DatabaseBlock) String() string {
	return fmt.Sprintf("DatabaseBlock{Type: %q, Name: %q}", d.Type, d.Name)
}

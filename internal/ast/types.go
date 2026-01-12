// Package ast defines the Abstract Syntax Tree for CodeAI specifications.
package ast

import "fmt"

// Position represents a source code position within a file.
// It tracks the filename, line number, column number, and byte offset.
type Position struct {
	// Filename is the name of the source file
	Filename string
	// Line is the 1-indexed line number
	Line int
	// Column is the 1-indexed column number
	Column int
	// Offset is the byte offset from the start of the file
	Offset int
}

// String returns a human-readable representation of the position
// in the format "filename:line:column".
func (p Position) String() string {
	return fmt.Sprintf("%s:%d:%d", p.Filename, p.Line, p.Column)
}

// IsValid returns true if the position has been set (non-zero line).
func (p Position) IsValid() bool {
	return p.Line > 0
}

// NodeType represents the type of an AST node.
type NodeType int

// Node type constants for all AST node kinds.
const (
	NodeProgram NodeType = iota
	NodeVarDecl
	NodeAssignment
	NodeIfStmt
	NodeForLoop
	NodeFunctionDecl
	NodeExecBlock
	NodeStringLiteral
	NodeNumberLiteral
	NodeBoolLiteral
	NodeIdentifier
	NodeFunctionCall
	NodeArrayLiteral
	NodeBlock
	NodeReturnStmt
	NodeBinaryExpr
	NodeUnaryExpr
	NodeConfigDecl
	NodeDatabaseBlock
)

// nodeTypeNames maps NodeType values to their string representations.
var nodeTypeNames = map[NodeType]string{
	NodeProgram:       "Program",
	NodeVarDecl:       "VarDecl",
	NodeAssignment:    "Assignment",
	NodeIfStmt:        "IfStmt",
	NodeForLoop:       "ForLoop",
	NodeFunctionDecl:  "FunctionDecl",
	NodeExecBlock:     "ExecBlock",
	NodeStringLiteral: "StringLiteral",
	NodeNumberLiteral: "NumberLiteral",
	NodeBoolLiteral:   "BoolLiteral",
	NodeIdentifier:    "Identifier",
	NodeFunctionCall:  "FunctionCall",
	NodeArrayLiteral:  "ArrayLiteral",
	NodeBlock:         "Block",
	NodeReturnStmt:    "ReturnStmt",
	NodeBinaryExpr:    "BinaryExpr",
	NodeUnaryExpr:     "UnaryExpr",
	NodeConfigDecl:    "ConfigDecl",
	NodeDatabaseBlock: "DatabaseBlock",
}

// String returns the string representation of the NodeType.
func (nt NodeType) String() string {
	if name, ok := nodeTypeNames[nt]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", nt)
}

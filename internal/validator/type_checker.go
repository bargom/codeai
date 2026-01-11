package validator

import (
	"github.com/bargom/codeai/internal/ast"
)

// Type represents the type of a value in the CodeAI language.
type Type int

const (
	// TypeUnknown represents an undetermined type.
	TypeUnknown Type = iota
	// TypeString represents string values.
	TypeString
	// TypeNumber represents numeric values.
	TypeNumber
	// TypeBool represents boolean values.
	TypeBool
	// TypeArray represents array values.
	TypeArray
	// TypeFunction represents function values.
	TypeFunction
	// TypeVoid represents no value (for statements).
	TypeVoid
)

// typeNames maps Type to string representations.
var typeNames = map[Type]string{
	TypeUnknown:  "unknown",
	TypeString:   "string",
	TypeNumber:   "number",
	TypeBool:     "bool",
	TypeArray:    "array",
	TypeFunction: "function",
	TypeVoid:     "void",
}

// String returns the string representation of Type.
func (t Type) String() string {
	if name, ok := typeNames[t]; ok {
		return name
	}
	return "unknown"
}

// TypeChecker performs type inference and checking on AST nodes.
type TypeChecker struct {
	symbols *SymbolTable
}

// NewTypeChecker creates a new type checker with the given symbol table.
func NewTypeChecker(symbols *SymbolTable) *TypeChecker {
	return &TypeChecker{
		symbols: symbols,
	}
}

// InferType determines the type of an expression.
func (tc *TypeChecker) InferType(expr ast.Expression) Type {
	if expr == nil {
		return TypeUnknown
	}

	switch e := expr.(type) {
	case *ast.StringLiteral:
		return TypeString

	case *ast.NumberLiteral:
		return TypeNumber

	case *ast.BoolLiteral:
		return TypeBool

	case *ast.ArrayLiteral:
		return TypeArray

	case *ast.Identifier:
		if sym, ok := tc.symbols.Lookup(e.Name); ok {
			return sym.Type
		}
		return TypeUnknown

	case *ast.FunctionCall:
		// Function calls have unknown return type for now
		// Could be extended to track return types
		return TypeUnknown

	case *ast.BinaryExpr:
		return tc.inferBinaryType(e)

	case *ast.UnaryExpr:
		return tc.inferUnaryType(e)

	default:
		return TypeUnknown
	}
}

// inferBinaryType determines the result type of a binary expression.
func (tc *TypeChecker) inferBinaryType(expr *ast.BinaryExpr) Type {
	leftType := tc.InferType(expr.Left)
	rightType := tc.InferType(expr.Right)

	switch expr.Operator {
	// Comparison operators return bool
	case "==", "!=", "<", ">", "<=", ">=":
		return TypeBool

	// Logical operators return bool
	case "and", "or":
		return TypeBool

	// Arithmetic operators
	case "+":
		// String concatenation or numeric addition
		if leftType == TypeString || rightType == TypeString {
			return TypeString
		}
		return TypeNumber

	case "-", "*", "/", "%":
		return TypeNumber

	default:
		return TypeUnknown
	}
}

// inferUnaryType determines the result type of a unary expression.
func (tc *TypeChecker) inferUnaryType(expr *ast.UnaryExpr) Type {
	switch expr.Operator {
	case "not", "!":
		return TypeBool
	case "-":
		return TypeNumber
	default:
		return TypeUnknown
	}
}

// IsIterable checks if a type can be used in a for-in loop.
func (tc *TypeChecker) IsIterable(typ Type) bool {
	// Only arrays are iterable in CodeAI
	return typ == TypeArray
}

// CheckCompatible checks if two types are compatible for an operation.
func (tc *TypeChecker) CheckCompatible(left, right Type) bool {
	// Same types are always compatible
	if left == right {
		return true
	}

	// Unknown types are compatible with anything (allows more lenient checking)
	if left == TypeUnknown || right == TypeUnknown {
		return true
	}

	return false
}

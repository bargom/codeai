// Package validator provides semantic validation for CodeAI AST.
package validator

import (
	"fmt"
	"strings"

	"github.com/bargom/codeai/internal/ast"
)

// ErrorType categorizes validation errors for structured handling.
type ErrorType int

const (
	// ErrorScope indicates a scope-related error (undefined/duplicate variables).
	ErrorScope ErrorType = iota
	// ErrorType indicates a type-related error.
	ErrorTypeCheck
	// ErrorFunction indicates a function-related error (wrong args, undefined).
	ErrorFunction
	// ErrorSemantic indicates a general semantic error.
	ErrorSemantic
)

// errorTypeNames maps ErrorType to human-readable names.
var errorTypeNames = map[ErrorType]string{
	ErrorScope:     "ScopeError",
	ErrorTypeCheck: "TypeError",
	ErrorFunction:  "FunctionError",
	ErrorSemantic:  "SemanticError",
}

// String returns the string representation of ErrorType.
func (et ErrorType) String() string {
	if name, ok := errorTypeNames[et]; ok {
		return name
	}
	return fmt.Sprintf("UnknownError(%d)", et)
}

// ValidationError represents a single semantic validation error.
type ValidationError struct {
	Position ast.Position
	Message  string
	Type     ErrorType
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Position.IsValid() {
		return fmt.Sprintf("%s: %s: %s", e.Position.String(), e.Type.String(), e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Type.String(), e.Message)
}

// ValidationErrors aggregates multiple validation errors.
type ValidationErrors struct {
	Errors []*ValidationError
}

// Error implements the error interface, formatting all errors.
func (ve *ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return "no validation errors"
	}

	if len(ve.Errors) == 1 {
		return ve.Errors[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d validation errors:\n", len(ve.Errors)))
	for i, err := range ve.Errors {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return sb.String()
}

// Add appends a validation error to the collection.
func (ve *ValidationErrors) Add(err *ValidationError) {
	ve.Errors = append(ve.Errors, err)
}

// HasErrors returns true if there are any validation errors.
func (ve *ValidationErrors) HasErrors() bool {
	return len(ve.Errors) > 0
}

// Unwrap returns the underlying errors for errors.Is/As compatibility.
func (ve *ValidationErrors) Unwrap() []error {
	errs := make([]error, len(ve.Errors))
	for i, e := range ve.Errors {
		errs[i] = e
	}
	return errs
}

// Helper functions to create specific error types

// newScopeError creates a scope-related validation error.
func newScopeError(pos ast.Position, message string) *ValidationError {
	return &ValidationError{
		Position: pos,
		Message:  message,
		Type:     ErrorScope,
	}
}

// newTypeError creates a type-related validation error.
func newTypeError(pos ast.Position, message string) *ValidationError {
	return &ValidationError{
		Position: pos,
		Message:  message,
		Type:     ErrorTypeCheck,
	}
}

// newFunctionError creates a function-related validation error.
func newFunctionError(pos ast.Position, message string) *ValidationError {
	return &ValidationError{
		Position: pos,
		Message:  message,
		Type:     ErrorFunction,
	}
}

// newSemanticError creates a general semantic validation error.
func newSemanticError(pos ast.Position, message string) *ValidationError {
	return &ValidationError{
		Position: pos,
		Message:  message,
		Type:     ErrorSemantic,
	}
}

// Specific error constructors for common validation failures

func errUndefinedVariable(pos ast.Position, name string) *ValidationError {
	return newScopeError(pos, fmt.Sprintf("undefined variable '%s'", name))
}

func errDuplicateDeclaration(pos ast.Position, name string) *ValidationError {
	return newScopeError(pos, fmt.Sprintf("duplicate declaration '%s'", name))
}

func errDuplicateParameter(pos ast.Position, name string) *ValidationError {
	return newScopeError(pos, fmt.Sprintf("duplicate parameter '%s'", name))
}

func errUndefinedFunction(pos ast.Position, name string) *ValidationError {
	return newFunctionError(pos, fmt.Sprintf("undefined function '%s'", name))
}

func errWrongArgCount(pos ast.Position, name string, expected, got int) *ValidationError {
	return newFunctionError(pos, fmt.Sprintf("wrong number of arguments for '%s': expected %d, got %d", name, expected, got))
}

func errNotAFunction(pos ast.Position, name string) *ValidationError {
	return newFunctionError(pos, fmt.Sprintf("'%s' is not a function", name))
}

func errCannotIterate(pos ast.Position, typeName string) *ValidationError {
	return newTypeError(pos, fmt.Sprintf("cannot iterate over non-array type '%s'", typeName))
}

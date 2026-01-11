package query

import (
	"fmt"
	"strings"
)

// ErrorType categorizes query errors for structured handling.
type ErrorType int

const (
	// ErrorLexer indicates a lexical error (invalid token).
	ErrorLexer ErrorType = iota
	// ErrorParser indicates a syntax error during parsing.
	ErrorParser
	// ErrorSemantic indicates a semantic error (e.g., unknown entity).
	ErrorSemantic
	// ErrorCompiler indicates an error during SQL compilation.
	ErrorCompiler
	// ErrorExecution indicates an error during query execution.
	ErrorExecution
)

// errorTypeNames maps ErrorType to human-readable names.
var errorTypeNames = map[ErrorType]string{
	ErrorLexer:     "LexerError",
	ErrorParser:    "ParserError",
	ErrorSemantic:  "SemanticError",
	ErrorCompiler:  "CompilerError",
	ErrorExecution: "ExecutionError",
}

// String returns the string representation of ErrorType.
func (et ErrorType) String() string {
	if name, ok := errorTypeNames[et]; ok {
		return name
	}
	return fmt.Sprintf("UnknownError(%d)", et)
}

// Position represents a location in the query string.
type Position struct {
	Offset int
	Line   int
	Column int
}

// String returns a human-readable position string.
func (p Position) String() string {
	return fmt.Sprintf("line %d, column %d", p.Line, p.Column)
}

// IsValid returns true if the position is valid.
func (p Position) IsValid() bool {
	return p.Line > 0
}

// QueryError represents a single query error.
type QueryError struct {
	Position Position
	Message  string
	Type     ErrorType
	Token    *Token // The token that caused the error, if applicable
}

// Error implements the error interface.
func (e *QueryError) Error() string {
	if e.Position.IsValid() {
		return fmt.Sprintf("%s at %s: %s", e.Type.String(), e.Position.String(), e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Type.String(), e.Message)
}

// Unwrap returns nil as QueryError doesn't wrap other errors.
func (e *QueryError) Unwrap() error {
	return nil
}

// QueryErrors aggregates multiple query errors.
type QueryErrors struct {
	Errors []*QueryError
}

// Error implements the error interface, formatting all errors.
func (qe *QueryErrors) Error() string {
	if len(qe.Errors) == 0 {
		return "no query errors"
	}

	if len(qe.Errors) == 1 {
		return qe.Errors[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d query errors:\n", len(qe.Errors)))
	for i, err := range qe.Errors {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return sb.String()
}

// Add appends a query error to the collection.
func (qe *QueryErrors) Add(err *QueryError) {
	qe.Errors = append(qe.Errors, err)
}

// HasErrors returns true if there are any query errors.
func (qe *QueryErrors) HasErrors() bool {
	return len(qe.Errors) > 0
}

// Unwrap returns the underlying errors for errors.Is/As compatibility.
func (qe *QueryErrors) Unwrap() []error {
	errs := make([]error, len(qe.Errors))
	for i, e := range qe.Errors {
		errs[i] = e
	}
	return errs
}

// NewQueryErrors creates a new empty QueryErrors.
func NewQueryErrors() *QueryErrors {
	return &QueryErrors{Errors: make([]*QueryError, 0)}
}

// Helper functions to create specific error types

// NewLexerError creates a lexer error.
func NewLexerError(pos Position, message string) *QueryError {
	return &QueryError{
		Position: pos,
		Message:  message,
		Type:     ErrorLexer,
	}
}

// NewLexerErrorWithToken creates a lexer error with the associated token.
func NewLexerErrorWithToken(tok Token, message string) *QueryError {
	return &QueryError{
		Position: Position{Offset: tok.Pos, Line: tok.Line, Column: tok.Column},
		Message:  message,
		Type:     ErrorLexer,
		Token:    &tok,
	}
}

// NewParserError creates a parser error.
func NewParserError(pos Position, message string) *QueryError {
	return &QueryError{
		Position: pos,
		Message:  message,
		Type:     ErrorParser,
	}
}

// NewParserErrorWithToken creates a parser error with the associated token.
func NewParserErrorWithToken(tok Token, message string) *QueryError {
	return &QueryError{
		Position: Position{Offset: tok.Pos, Line: tok.Line, Column: tok.Column},
		Message:  message,
		Type:     ErrorParser,
		Token:    &tok,
	}
}

// NewSemanticError creates a semantic error.
func NewSemanticError(pos Position, message string) *QueryError {
	return &QueryError{
		Position: pos,
		Message:  message,
		Type:     ErrorSemantic,
	}
}

// NewCompilerError creates a compiler error.
func NewCompilerError(message string) *QueryError {
	return &QueryError{
		Message: message,
		Type:    ErrorCompiler,
	}
}

// NewCompilerErrorWithPosition creates a compiler error with position.
func NewCompilerErrorWithPosition(pos Position, message string) *QueryError {
	return &QueryError{
		Position: pos,
		Message:  message,
		Type:     ErrorCompiler,
	}
}

// NewExecutionError creates an execution error.
func NewExecutionError(message string) *QueryError {
	return &QueryError{
		Message: message,
		Type:    ErrorExecution,
	}
}

// Specific error constructors for common parsing failures

// ErrUnexpectedToken creates an error for an unexpected token.
func ErrUnexpectedToken(tok Token, expected string) *QueryError {
	msg := fmt.Sprintf("unexpected token %q", tok.Value)
	if expected != "" {
		msg = fmt.Sprintf("expected %s, got %q", expected, tok.Value)
	}
	return NewParserErrorWithToken(tok, msg)
}

// ErrUnexpectedEOF creates an error for unexpected end of input.
func ErrUnexpectedEOF(expected string) *QueryError {
	msg := "unexpected end of input"
	if expected != "" {
		msg = fmt.Sprintf("expected %s, got end of input", expected)
	}
	return &QueryError{
		Message: msg,
		Type:    ErrorParser,
	}
}

// ErrInvalidOperator creates an error for an invalid operator.
func ErrInvalidOperator(tok Token) *QueryError {
	return NewParserErrorWithToken(tok, fmt.Sprintf("invalid operator %q", tok.Value))
}

// ErrInvalidValue creates an error for an invalid value.
func ErrInvalidValue(tok Token, expected string) *QueryError {
	msg := fmt.Sprintf("invalid value %q", tok.Value)
	if expected != "" {
		msg = fmt.Sprintf("expected %s, got %q", expected, tok.Value)
	}
	return NewParserErrorWithToken(tok, msg)
}

// ErrUnknownEntity creates an error for an unknown entity.
func ErrUnknownEntity(name string) *QueryError {
	return NewCompilerError(fmt.Sprintf("unknown entity %q", name))
}

// ErrUnknownField creates an error for an unknown field.
func ErrUnknownField(entity, field string) *QueryError {
	return NewCompilerError(fmt.Sprintf("unknown field %q on entity %q", field, entity))
}

// ErrTypeMismatch creates an error for type mismatch.
func ErrTypeMismatch(field, expectedType, gotType string) *QueryError {
	return NewCompilerError(fmt.Sprintf("type mismatch for field %q: expected %s, got %s", field, expectedType, gotType))
}

// ErrInvalidQueryType creates an error for invalid query type.
func ErrInvalidQueryType(queryType QueryType) *QueryError {
	return NewCompilerError(fmt.Sprintf("unsupported query type: %s", queryType))
}

// ErrMissingEntity creates an error for missing entity in query.
func ErrMissingEntity() *QueryError {
	return NewParserError(Position{}, "query must specify an entity")
}

// ErrInvalidLimit creates an error for invalid LIMIT value.
func ErrInvalidLimit(value string) *QueryError {
	return NewParserError(Position{}, fmt.Sprintf("invalid LIMIT value: %s", value))
}

// ErrInvalidOffset creates an error for invalid OFFSET value.
func ErrInvalidOffset(value string) *QueryError {
	return NewParserError(Position{}, fmt.Sprintf("invalid OFFSET value: %s", value))
}

// ErrNestedGroupDepth creates an error for too deeply nested groups.
func ErrNestedGroupDepth(maxDepth int) *QueryError {
	return NewParserError(Position{}, fmt.Sprintf("maximum nesting depth of %d exceeded", maxDepth))
}

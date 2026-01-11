package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorType_String(t *testing.T) {
	tests := []struct {
		et       ErrorType
		expected string
	}{
		{ErrorLexer, "LexerError"},
		{ErrorParser, "ParserError"},
		{ErrorSemantic, "SemanticError"},
		{ErrorCompiler, "CompilerError"},
		{ErrorExecution, "ExecutionError"},
		{ErrorType(999), "UnknownError(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.et.String())
		})
	}
}

func TestPosition_String(t *testing.T) {
	pos := Position{Offset: 10, Line: 2, Column: 5}
	assert.Equal(t, "line 2, column 5", pos.String())
}

func TestPosition_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		pos      Position
		expected bool
	}{
		{"valid", Position{Line: 1, Column: 1}, true},
		{"invalid line 0", Position{Line: 0, Column: 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pos.IsValid())
		})
	}
}

func TestQueryError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *QueryError
		expected string
	}{
		{
			name: "with valid position",
			err: &QueryError{
				Position: Position{Line: 1, Column: 5},
				Message:  "unexpected token",
				Type:     ErrorParser,
			},
			expected: "ParserError at line 1, column 5: unexpected token",
		},
		{
			name: "without position",
			err: &QueryError{
				Position: Position{},
				Message:  "unknown entity",
				Type:     ErrorCompiler,
			},
			expected: "CompilerError: unknown entity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestQueryError_Unwrap(t *testing.T) {
	err := &QueryError{Message: "test"}
	assert.Nil(t, err.Unwrap())
}

func TestQueryErrors_Error(t *testing.T) {
	tests := []struct {
		name     string
		errors   *QueryErrors
		contains string
	}{
		{
			name:     "no errors",
			errors:   NewQueryErrors(),
			contains: "no query errors",
		},
		{
			name: "single error",
			errors: &QueryErrors{
				Errors: []*QueryError{
					{Message: "test error", Type: ErrorParser},
				},
			},
			contains: "test error",
		},
		{
			name: "multiple errors",
			errors: &QueryErrors{
				Errors: []*QueryError{
					{Message: "error 1", Type: ErrorParser},
					{Message: "error 2", Type: ErrorCompiler},
				},
			},
			contains: "2 query errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, tt.errors.Error(), tt.contains)
		})
	}
}

func TestQueryErrors_Add(t *testing.T) {
	errs := NewQueryErrors()
	assert.False(t, errs.HasErrors())

	errs.Add(&QueryError{Message: "test"})
	assert.True(t, errs.HasErrors())
	assert.Len(t, errs.Errors, 1)
}

func TestQueryErrors_Unwrap(t *testing.T) {
	errs := &QueryErrors{
		Errors: []*QueryError{
			{Message: "error 1"},
			{Message: "error 2"},
		},
	}

	unwrapped := errs.Unwrap()
	assert.Len(t, unwrapped, 2)
}

func TestNewLexerError(t *testing.T) {
	pos := Position{Line: 1, Column: 5}
	err := NewLexerError(pos, "unexpected character")

	assert.Equal(t, ErrorLexer, err.Type)
	assert.Equal(t, "unexpected character", err.Message)
	assert.Equal(t, pos, err.Position)
}

func TestNewLexerErrorWithToken(t *testing.T) {
	tok := Token{Type: TokenError, Value: "test", Pos: 10, Line: 2, Column: 5}
	err := NewLexerErrorWithToken(tok, "invalid token")

	assert.Equal(t, ErrorLexer, err.Type)
	assert.Equal(t, "invalid token", err.Message)
	assert.Equal(t, 2, err.Position.Line)
	assert.Equal(t, 5, err.Position.Column)
	assert.NotNil(t, err.Token)
}

func TestNewParserError(t *testing.T) {
	pos := Position{Line: 3, Column: 10}
	err := NewParserError(pos, "syntax error")

	assert.Equal(t, ErrorParser, err.Type)
	assert.Equal(t, "syntax error", err.Message)
}

func TestNewParserErrorWithToken(t *testing.T) {
	tok := Token{Type: TokenIdent, Value: "foo", Pos: 5, Line: 1, Column: 6}
	err := NewParserErrorWithToken(tok, "unexpected identifier")

	assert.Equal(t, ErrorParser, err.Type)
	assert.NotNil(t, err.Token)
}

func TestNewSemanticError(t *testing.T) {
	pos := Position{Line: 1, Column: 1}
	err := NewSemanticError(pos, "undefined variable")

	assert.Equal(t, ErrorSemantic, err.Type)
}

func TestNewCompilerError(t *testing.T) {
	err := NewCompilerError("unknown entity")
	assert.Equal(t, ErrorCompiler, err.Type)
	assert.Equal(t, "unknown entity", err.Message)
}

func TestNewCompilerErrorWithPosition(t *testing.T) {
	pos := Position{Line: 5, Column: 3}
	err := NewCompilerErrorWithPosition(pos, "type mismatch")

	assert.Equal(t, ErrorCompiler, err.Type)
	assert.Equal(t, pos, err.Position)
}

func TestNewExecutionError(t *testing.T) {
	err := NewExecutionError("connection failed")
	assert.Equal(t, ErrorExecution, err.Type)
}

func TestErrUnexpectedToken(t *testing.T) {
	tok := Token{Type: TokenIdent, Value: "foo", Line: 1, Column: 5}

	tests := []struct {
		name     string
		expected string
		contains string
	}{
		{
			name:     "with expected",
			expected: "SELECT",
			contains: "expected SELECT",
		},
		{
			name:     "without expected",
			expected: "",
			contains: "unexpected token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ErrUnexpectedToken(tok, tt.expected)
			assert.Contains(t, err.Error(), tt.contains)
		})
	}
}

func TestErrUnexpectedEOF(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		contains string
	}{
		{
			name:     "with expected",
			expected: "entity name",
			contains: "expected entity name",
		},
		{
			name:     "without expected",
			expected: "",
			contains: "unexpected end of input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ErrUnexpectedEOF(tt.expected)
			assert.Contains(t, err.Error(), tt.contains)
		})
	}
}

func TestErrInvalidOperator(t *testing.T) {
	tok := Token{Type: TokenIdent, Value: "%%", Line: 1, Column: 10}
	err := ErrInvalidOperator(tok)
	assert.Contains(t, err.Error(), "invalid operator")
}

func TestErrInvalidValue(t *testing.T) {
	tok := Token{Type: TokenString, Value: "abc", Line: 1, Column: 5}

	tests := []struct {
		name     string
		expected string
		contains string
	}{
		{
			name:     "with expected type",
			expected: "integer",
			contains: "expected integer",
		},
		{
			name:     "without expected type",
			expected: "",
			contains: "invalid value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ErrInvalidValue(tok, tt.expected)
			assert.Contains(t, err.Error(), tt.contains)
		})
	}
}

func TestErrUnknownEntity(t *testing.T) {
	err := ErrUnknownEntity("foobar")
	assert.Contains(t, err.Error(), "unknown entity")
	assert.Contains(t, err.Error(), "foobar")
}

func TestErrUnknownField(t *testing.T) {
	err := ErrUnknownField("users", "unknown_field")
	assert.Contains(t, err.Error(), "unknown field")
	assert.Contains(t, err.Error(), "users")
	assert.Contains(t, err.Error(), "unknown_field")
}

func TestErrTypeMismatch(t *testing.T) {
	err := ErrTypeMismatch("age", "integer", "string")
	assert.Contains(t, err.Error(), "type mismatch")
	assert.Contains(t, err.Error(), "age")
}

func TestErrInvalidQueryType(t *testing.T) {
	err := ErrInvalidQueryType(QueryType(999))
	assert.Contains(t, err.Error(), "unsupported query type")
}

func TestErrMissingEntity(t *testing.T) {
	err := ErrMissingEntity()
	assert.Contains(t, err.Error(), "must specify an entity")
}

func TestErrInvalidLimit(t *testing.T) {
	err := ErrInvalidLimit("abc")
	assert.Contains(t, err.Error(), "invalid LIMIT")
}

func TestErrInvalidOffset(t *testing.T) {
	err := ErrInvalidOffset("-5")
	assert.Contains(t, err.Error(), "invalid OFFSET")
}

func TestErrNestedGroupDepth(t *testing.T) {
	err := ErrNestedGroupDepth(10)
	assert.Contains(t, err.Error(), "maximum nesting depth")
	assert.Contains(t, err.Error(), "10")
}

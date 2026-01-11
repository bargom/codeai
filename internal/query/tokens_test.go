package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLexer_SimpleTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:     "punctuation",
			input:    "( ) [ ] , . :",
			expected: []TokenType{TokenLParen, TokenRParen, TokenLBracket, TokenRBracket, TokenComma, TokenDot, TokenColon, TokenEOF},
		},
		{
			name:     "operators",
			input:    "= != > >= < <= <>",
			expected: []TokenType{TokenEquals, TokenNotEquals, TokenGreater, TokenGreaterEq, TokenLess, TokenLessEq, TokenNotEquals, TokenEOF},
		},
		{
			name:     "tilde",
			input:    "~fuzzy",
			expected: []TokenType{TokenTilde, TokenIdent, TokenEOF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			types := make([]TokenType, len(tokens))
			for i, tok := range tokens {
				types[i] = tok.Type
			}
			assert.Equal(t, tt.expected, types)
		})
	}
}

func TestLexer_Keywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"select", TokenSelect},
		{"SELECT", TokenSelect},
		{"count", TokenCount},
		{"sum", TokenSum},
		{"avg", TokenAvg},
		{"min", TokenMin},
		{"max", TokenMax},
		{"update", TokenUpdate},
		{"delete", TokenDelete},
		{"from", TokenFrom},
		{"where", TokenWhere},
		{"and", TokenAnd},
		{"or", TokenOr},
		{"not", TokenNot},
		{"in", TokenIn},
		{"is", TokenIs},
		{"null", TokenNull},
		{"order", TokenOrder},
		{"by", TokenBy},
		{"asc", TokenAsc},
		{"desc", TokenDesc},
		{"limit", TokenLimit},
		{"offset", TokenOffset},
		{"with", TokenWith},
		{"include", TokenInclude},
		{"set", TokenSet},
		{"true", TokenTrue},
		{"false", TokenFalse},
		{"like", TokenLike},
		{"ilike", TokenILike},
		{"between", TokenBetween},
		{"having", TokenHaving},
		{"contains", TokenContains},
		{"startswith", TokenStartsWith},
		{"endswith", TokenEndsWith},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)
			require.Len(t, tokens, 2) // keyword + EOF
			assert.Equal(t, tt.expected, tokens[0].Type)
		})
	}
}

func TestLexer_Strings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"double quotes", `"hello world"`, "hello world"},
		{"single quotes", `'hello world'`, "hello world"},
		{"escape newline", `"hello\nworld"`, "hello\nworld"},
		{"escape tab", `"hello\tworld"`, "hello\tworld"},
		{"escape quote", `"hello\"world"`, `hello"world`},
		{"escape backslash", `"hello\\world"`, `hello\world`},
		{"empty string", `""`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)
			require.Len(t, tokens, 2)
			assert.Equal(t, TokenString, tokens[0].Type)
			assert.Equal(t, tt.expected, tokens[0].Value)
		})
	}
}

func TestLexer_UnterminatedString(t *testing.T) {
	lexer := NewLexer(`"hello`)
	tokens, err := lexer.Tokenize()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unterminated string")
	require.Len(t, tokens, 1)
	assert.Equal(t, TokenError, tokens[0].Type)
}

func TestLexer_Numbers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"integer", "42", "42"},
		{"negative integer", "-42", "-42"},
		{"decimal", "3.14", "3.14"},
		{"negative decimal", "-3.14", "-3.14"},
		{"exponent", "1e10", "1e10"},
		{"negative exponent", "1e-10", "1e-10"},
		{"decimal with exponent", "1.5e10", "1.5e10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)
			require.Len(t, tokens, 2)
			assert.Equal(t, TokenNumber, tokens[0].Type)
			assert.Equal(t, tt.expected, tokens[0].Value)
		})
	}
}

func TestLexer_Identifiers(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo", "foo"},
		{"foo_bar", "foo_bar"},
		{"foo123", "foo123"},
		{"_private", "_private"},
		{"CamelCase", "CamelCase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)
			require.Len(t, tokens, 2)
			assert.Equal(t, TokenIdent, tokens[0].Type)
			assert.Equal(t, tt.expected, tokens[0].Value)
		})
	}
}

func TestLexer_Parameters(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"@userId", "userId"},
		{"$name", "name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)
			require.Len(t, tokens, 2)
			assert.Equal(t, TokenParam, tokens[0].Type)
			assert.Equal(t, tt.expected, tokens[0].Value)
		})
	}
}

func TestLexer_InvalidParameter(t *testing.T) {
	lexer := NewLexer("@")
	tokens, err := lexer.Tokenize()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected parameter name")
	require.Len(t, tokens, 1)
	assert.Equal(t, TokenError, tokens[0].Type)
}

func TestLexer_FullQuery(t *testing.T) {
	input := `select users where status = "active" and age >= 18 order by created_at desc limit 10`

	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	expectedTypes := []TokenType{
		TokenSelect, TokenIdent, TokenWhere, TokenIdent, TokenEquals, TokenString,
		TokenAnd, TokenIdent, TokenGreaterEq, TokenNumber, TokenOrder, TokenBy,
		TokenIdent, TokenDesc, TokenLimit, TokenNumber, TokenEOF,
	}

	types := make([]TokenType, len(tokens))
	for i, tok := range tokens {
		types[i] = tok.Type
	}
	assert.Equal(t, expectedTypes, types)
}

func TestLexer_Whitespace(t *testing.T) {
	// Test various whitespace handling
	input := "select\tusers\nwhere\r\nstatus = 'active'"

	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	// Should skip all whitespace
	expectedTypes := []TokenType{TokenSelect, TokenIdent, TokenWhere, TokenIdent, TokenEquals, TokenString, TokenEOF}
	types := make([]TokenType, len(tokens))
	for i, tok := range tokens {
		types[i] = tok.Type
	}
	assert.Equal(t, expectedTypes, types)
}

func TestLexer_Position(t *testing.T) {
	input := "select\nusers"

	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)
	require.Len(t, tokens, 3)

	// First token: line 1
	assert.Equal(t, 1, tokens[0].Line)

	// Second token: line 2
	assert.Equal(t, 2, tokens[1].Line)
}

func TestLexer_UnexpectedCharacter(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"single exclamation", "!"},
		{"at sign only", "@ "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			require.Error(t, err)
			var hasError bool
			for _, tok := range tokens {
				if tok.Type == TokenError {
					hasError = true
					break
				}
			}
			assert.True(t, hasError)
		})
	}
}

func TestToken_String(t *testing.T) {
	tests := []struct {
		token    Token
		expected string
	}{
		{Token{Type: TokenEOF}, "EOF"},
		{Token{Type: TokenError, Value: "test error"}, "Error(test error)"},
		{Token{Type: TokenSelect, Value: "select"}, "SELECT(\"select\")"},
		{Token{Type: TokenIdent, Value: "foo"}, "IDENT(\"foo\")"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.token.String())
		})
	}
}

func TestLexer_ComplexQuery(t *testing.T) {
	input := `select id, name, email from users where (status = "active" or role = "admin") and created_at >= "2024-01-01" order by name asc limit 50 offset 100`

	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	// Verify we got all expected tokens
	assert.True(t, len(tokens) > 20)
	assert.Equal(t, TokenEOF, tokens[len(tokens)-1].Type)

	// Verify no error tokens
	for _, tok := range tokens {
		assert.NotEqual(t, TokenError, tok.Type, "unexpected error token: %s", tok.Value)
	}
}

func TestLexer_ArraySyntax(t *testing.T) {
	input := `status in ["active", "pending", "approved"]`

	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	expectedTypes := []TokenType{
		TokenIdent, TokenIn, TokenLBracket,
		TokenString, TokenComma, TokenString, TokenComma, TokenString,
		TokenRBracket, TokenEOF,
	}

	types := make([]TokenType, len(tokens))
	for i, tok := range tokens {
		types[i] = tok.Type
	}
	assert.Equal(t, expectedTypes, types)
}

func TestLexer_FieldColonValue(t *testing.T) {
	input := `status:active priority:1`

	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	expectedTypes := []TokenType{
		TokenIdent, TokenColon, TokenIdent,
		TokenIdent, TokenColon, TokenNumber,
		TokenEOF,
	}

	types := make([]TokenType, len(tokens))
	for i, tok := range tokens {
		types[i] = tok.Type
	}
	assert.Equal(t, expectedTypes, types)
}

func TestLexer_FuzzySearch(t *testing.T) {
	input := `~fuzzy "exact phrase"`

	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	expectedTypes := []TokenType{TokenTilde, TokenIdent, TokenString, TokenEOF}

	types := make([]TokenType, len(tokens))
	for i, tok := range tokens {
		types[i] = tok.Type
	}
	assert.Equal(t, expectedTypes, types)
}

func TestLexer_Comparison(t *testing.T) {
	input := `age > 18 count <= 100 created_at >= "2024-01-01"`

	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	// Check for comparison operators
	hasGreater := false
	hasLessEq := false
	hasGreaterEq := false

	for _, tok := range tokens {
		switch tok.Type {
		case TokenGreater:
			hasGreater = true
		case TokenLessEq:
			hasLessEq = true
		case TokenGreaterEq:
			hasGreaterEq = true
		}
	}

	assert.True(t, hasGreater)
	assert.True(t, hasLessEq)
	assert.True(t, hasGreaterEq)
}

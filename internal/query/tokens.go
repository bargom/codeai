// Package query provides a query language parser and SQL compiler for CodeAI.
package query

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents the type of a lexical token.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenError

	// Keywords
	TokenSelect
	TokenCount
	TokenSum
	TokenAvg
	TokenMin
	TokenMax
	TokenUpdate
	TokenDelete
	TokenFrom
	TokenWhere
	TokenAnd
	TokenOr
	TokenNot
	TokenIn
	TokenIs
	TokenNull
	TokenOrder
	TokenBy
	TokenAsc
	TokenDesc
	TokenLimit
	TokenOffset
	TokenWith
	TokenInclude
	TokenSet
	TokenTrue
	TokenFalse
	TokenLike
	TokenILike
	TokenBetween
	TokenHaving
	TokenGroup
	TokenGroupBy

	// Operators
	TokenEquals
	TokenNotEquals
	TokenGreater
	TokenGreaterEq
	TokenLess
	TokenLessEq
	TokenContains
	TokenStartsWith
	TokenEndsWith

	// Literals
	TokenIdent
	TokenString
	TokenNumber
	TokenParam

	// Punctuation
	TokenLParen
	TokenRParen
	TokenLBracket
	TokenRBracket
	TokenComma
	TokenDot
	TokenColon
	TokenTilde
)

// Token represents a lexical token.
type Token struct {
	Type   TokenType
	Value  string
	Pos    int
	Line   int
	Column int
}

// String returns a string representation of the token.
func (t Token) String() string {
	if t.Type == TokenEOF {
		return "EOF"
	}
	if t.Type == TokenError {
		return fmt.Sprintf("Error(%s)", t.Value)
	}
	return fmt.Sprintf("%s(%q)", tokenTypeNames[t.Type], t.Value)
}

var tokenTypeNames = map[TokenType]string{
	TokenEOF:        "EOF",
	TokenError:      "Error",
	TokenSelect:     "SELECT",
	TokenCount:      "COUNT",
	TokenSum:        "SUM",
	TokenAvg:        "AVG",
	TokenMin:        "MIN",
	TokenMax:        "MAX",
	TokenUpdate:     "UPDATE",
	TokenDelete:     "DELETE",
	TokenFrom:       "FROM",
	TokenWhere:      "WHERE",
	TokenAnd:        "AND",
	TokenOr:         "OR",
	TokenNot:        "NOT",
	TokenIn:         "IN",
	TokenIs:         "IS",
	TokenNull:       "NULL",
	TokenOrder:      "ORDER",
	TokenBy:         "BY",
	TokenAsc:        "ASC",
	TokenDesc:       "DESC",
	TokenLimit:      "LIMIT",
	TokenOffset:     "OFFSET",
	TokenWith:       "WITH",
	TokenInclude:    "INCLUDE",
	TokenSet:        "SET",
	TokenTrue:       "TRUE",
	TokenFalse:      "FALSE",
	TokenLike:       "LIKE",
	TokenILike:      "ILIKE",
	TokenBetween:    "BETWEEN",
	TokenHaving:     "HAVING",
	TokenGroup:      "GROUP",
	TokenGroupBy:    "GROUP BY",
	TokenEquals:     "=",
	TokenNotEquals:  "!=",
	TokenGreater:    ">",
	TokenGreaterEq:  ">=",
	TokenLess:       "<",
	TokenLessEq:     "<=",
	TokenContains:   "CONTAINS",
	TokenStartsWith: "STARTSWITH",
	TokenEndsWith:   "ENDSWITH",
	TokenIdent:      "IDENT",
	TokenString:     "STRING",
	TokenNumber:     "NUMBER",
	TokenParam:      "PARAM",
	TokenLParen:     "(",
	TokenRParen:     ")",
	TokenLBracket:   "[",
	TokenRBracket:   "]",
	TokenComma:      ",",
	TokenDot:        ".",
	TokenColon:      ":",
	TokenTilde:      "~",
}

// keywords maps keyword strings to token types.
var keywords = map[string]TokenType{
	"select":     TokenSelect,
	"count":      TokenCount,
	"sum":        TokenSum,
	"avg":        TokenAvg,
	"min":        TokenMin,
	"max":        TokenMax,
	"update":     TokenUpdate,
	"delete":     TokenDelete,
	"from":       TokenFrom,
	"where":      TokenWhere,
	"and":        TokenAnd,
	"or":         TokenOr,
	"not":        TokenNot,
	"in":         TokenIn,
	"is":         TokenIs,
	"null":       TokenNull,
	"order":      TokenOrder,
	"by":         TokenBy,
	"asc":        TokenAsc,
	"desc":       TokenDesc,
	"limit":      TokenLimit,
	"offset":     TokenOffset,
	"with":       TokenWith,
	"include":    TokenInclude,
	"set":        TokenSet,
	"true":       TokenTrue,
	"false":      TokenFalse,
	"like":       TokenLike,
	"ilike":      TokenILike,
	"between":    TokenBetween,
	"having":     TokenHaving,
	"group":      TokenGroup,
	"contains":   TokenContains,
	"startswith": TokenStartsWith,
	"endswith":   TokenEndsWith,
}

// Lexer tokenizes a query string.
type Lexer struct {
	input  string
	pos    int
	line   int
	column int
	tokens []Token
}

// NewLexer creates a new Lexer for the given input.
func NewLexer(input string) *Lexer {
	return &Lexer{
		input:  input,
		pos:    0,
		line:   1,
		column: 1,
		tokens: make([]Token, 0),
	}
}

// Tokenize scans the input and returns all tokens.
func (l *Lexer) Tokenize() ([]Token, error) {
	for {
		tok := l.nextToken()
		l.tokens = append(l.tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
		if tok.Type == TokenError {
			return l.tokens, fmt.Errorf("lexer error at line %d, column %d: %s", tok.Line, tok.Column, tok.Value)
		}
	}
	return l.tokens, nil
}

// nextToken returns the next token from the input.
func (l *Lexer) nextToken() Token {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Pos: l.pos, Line: l.line, Column: l.column}
	}

	startPos := l.pos
	startLine := l.line
	startColumn := l.column
	ch := l.input[l.pos]

	// Single-character tokens
	switch ch {
	case '(':
		l.advance()
		return Token{Type: TokenLParen, Value: "(", Pos: startPos, Line: startLine, Column: startColumn}
	case ')':
		l.advance()
		return Token{Type: TokenRParen, Value: ")", Pos: startPos, Line: startLine, Column: startColumn}
	case '[':
		l.advance()
		return Token{Type: TokenLBracket, Value: "[", Pos: startPos, Line: startLine, Column: startColumn}
	case ']':
		l.advance()
		return Token{Type: TokenRBracket, Value: "]", Pos: startPos, Line: startLine, Column: startColumn}
	case ',':
		l.advance()
		return Token{Type: TokenComma, Value: ",", Pos: startPos, Line: startLine, Column: startColumn}
	case '.':
		l.advance()
		return Token{Type: TokenDot, Value: ".", Pos: startPos, Line: startLine, Column: startColumn}
	case ':':
		l.advance()
		return Token{Type: TokenColon, Value: ":", Pos: startPos, Line: startLine, Column: startColumn}
	case '~':
		l.advance()
		return Token{Type: TokenTilde, Value: "~", Pos: startPos, Line: startLine, Column: startColumn}
	case '=':
		l.advance()
		return Token{Type: TokenEquals, Value: "=", Pos: startPos, Line: startLine, Column: startColumn}
	case '!':
		l.advance()
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.advance()
			return Token{Type: TokenNotEquals, Value: "!=", Pos: startPos, Line: startLine, Column: startColumn}
		}
		return Token{Type: TokenError, Value: "unexpected character '!'", Pos: startPos, Line: startLine, Column: startColumn}
	case '>':
		l.advance()
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.advance()
			return Token{Type: TokenGreaterEq, Value: ">=", Pos: startPos, Line: startLine, Column: startColumn}
		}
		return Token{Type: TokenGreater, Value: ">", Pos: startPos, Line: startLine, Column: startColumn}
	case '<':
		l.advance()
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.advance()
			return Token{Type: TokenLessEq, Value: "<=", Pos: startPos, Line: startLine, Column: startColumn}
		}
		if l.pos < len(l.input) && l.input[l.pos] == '>' {
			l.advance()
			return Token{Type: TokenNotEquals, Value: "<>", Pos: startPos, Line: startLine, Column: startColumn}
		}
		return Token{Type: TokenLess, Value: "<", Pos: startPos, Line: startLine, Column: startColumn}
	case '"':
		return l.readString('"')
	case '\'':
		return l.readString('\'')
	case '@':
		return l.readParam()
	case '$':
		return l.readParam()
	}

	// Numbers
	if isDigit(ch) || (ch == '-' && l.pos+1 < len(l.input) && isDigit(l.input[l.pos+1])) {
		return l.readNumber()
	}

	// Identifiers and keywords
	if isIdentStart(ch) {
		return l.readIdentifier()
	}

	l.advance()
	return Token{Type: TokenError, Value: fmt.Sprintf("unexpected character '%c'", ch), Pos: startPos, Line: startLine, Column: startColumn}
}

// skipWhitespace advances past any whitespace characters.
func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.advance()
		} else if ch == '\n' {
			l.advance()
			l.line++
			l.column = 1
		} else {
			break
		}
	}
}

// advance moves to the next character.
func (l *Lexer) advance() {
	if l.pos < len(l.input) {
		l.pos++
		l.column++
	}
}

// readString reads a quoted string literal.
func (l *Lexer) readString(quote byte) Token {
	startPos := l.pos
	startLine := l.line
	startColumn := l.column
	l.advance() // consume opening quote

	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == quote {
			l.advance() // consume closing quote
			return Token{Type: TokenString, Value: sb.String(), Pos: startPos, Line: startLine, Column: startColumn}
		}
		if ch == '\\' && l.pos+1 < len(l.input) {
			l.advance()
			nextCh := l.input[l.pos]
			switch nextCh {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			case '\'':
				sb.WriteByte('\'')
			default:
				sb.WriteByte(nextCh)
			}
			l.advance()
			continue
		}
		if ch == '\n' {
			l.line++
			l.column = 0
		}
		sb.WriteByte(ch)
		l.advance()
	}

	return Token{Type: TokenError, Value: "unterminated string", Pos: startPos, Line: startLine, Column: startColumn}
}

// readNumber reads a numeric literal.
func (l *Lexer) readNumber() Token {
	startPos := l.pos
	startLine := l.line
	startColumn := l.column

	var sb strings.Builder
	if l.input[l.pos] == '-' {
		sb.WriteByte('-')
		l.advance()
	}

	for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
		sb.WriteByte(l.input[l.pos])
		l.advance()
	}

	// Check for decimal point
	if l.pos < len(l.input) && l.input[l.pos] == '.' {
		if l.pos+1 < len(l.input) && isDigit(l.input[l.pos+1]) {
			sb.WriteByte('.')
			l.advance()
			for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
				sb.WriteByte(l.input[l.pos])
				l.advance()
			}
		}
	}

	// Check for exponent
	if l.pos < len(l.input) && (l.input[l.pos] == 'e' || l.input[l.pos] == 'E') {
		sb.WriteByte(l.input[l.pos])
		l.advance()
		if l.pos < len(l.input) && (l.input[l.pos] == '+' || l.input[l.pos] == '-') {
			sb.WriteByte(l.input[l.pos])
			l.advance()
		}
		for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
			sb.WriteByte(l.input[l.pos])
			l.advance()
		}
	}

	return Token{Type: TokenNumber, Value: sb.String(), Pos: startPos, Line: startLine, Column: startColumn}
}

// readIdentifier reads an identifier or keyword.
func (l *Lexer) readIdentifier() Token {
	startPos := l.pos
	startLine := l.line
	startColumn := l.column

	var sb strings.Builder
	for l.pos < len(l.input) && isIdentChar(l.input[l.pos]) {
		sb.WriteByte(l.input[l.pos])
		l.advance()
	}

	value := sb.String()
	lower := strings.ToLower(value)

	// Check for keywords
	if tokType, ok := keywords[lower]; ok {
		return Token{Type: tokType, Value: value, Pos: startPos, Line: startLine, Column: startColumn}
	}

	return Token{Type: TokenIdent, Value: value, Pos: startPos, Line: startLine, Column: startColumn}
}

// readParam reads a parameter token (@name or $name).
func (l *Lexer) readParam() Token {
	startPos := l.pos
	startLine := l.line
	startColumn := l.column
	l.advance() // consume @ or $

	if l.pos >= len(l.input) || !isIdentStart(l.input[l.pos]) {
		return Token{Type: TokenError, Value: "expected parameter name", Pos: startPos, Line: startLine, Column: startColumn}
	}

	var sb strings.Builder
	for l.pos < len(l.input) && isIdentChar(l.input[l.pos]) {
		sb.WriteByte(l.input[l.pos])
		l.advance()
	}

	return Token{Type: TokenParam, Value: sb.String(), Pos: startPos, Line: startLine, Column: startColumn}
}

// isDigit checks if a character is a digit.
func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// isIdentStart checks if a character can start an identifier.
func isIdentStart(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

// isIdentChar checks if a character can be part of an identifier.
func isIdentChar(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_'
}

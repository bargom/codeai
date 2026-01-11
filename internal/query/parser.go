package query

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser parses query tokens into a Query AST.
type Parser struct {
	tokens   []Token
	pos      int
	current  Token
	maxDepth int
	depth    int
}

// NewParser creates a new parser for the given tokens.
func NewParser(tokens []Token) *Parser {
	p := &Parser{
		tokens:   tokens,
		pos:      0,
		maxDepth: 10, // Maximum nesting depth for parentheses
	}
	if len(tokens) > 0 {
		p.current = tokens[0]
	}
	return p
}

// isIdentifierOrKeyword checks if the current token can be used as an identifier.
// This allows keywords like "count", "status", "group" to be used as field names.
func (p *Parser) isIdentifierOrKeyword() bool {
	switch p.current.Type {
	case TokenIdent:
		return true
	// Keywords that can also be field names
	case TokenCount, TokenSum, TokenAvg, TokenMin, TokenMax,
		TokenFrom, TokenWhere, TokenAnd, TokenOr, TokenNot,
		TokenIn, TokenIs, TokenNull, TokenOrder, TokenBy,
		TokenAsc, TokenDesc, TokenLimit, TokenOffset,
		TokenWith, TokenInclude, TokenSet, TokenTrue, TokenFalse,
		TokenLike, TokenILike, TokenBetween, TokenHaving, TokenGroup,
		TokenContains, TokenStartsWith, TokenEndsWith,
		TokenSelect, TokenUpdate, TokenDelete:
		return true
	default:
		return false
	}
}

// getIdentifierValue returns the value of the current token if it can be used as an identifier.
func (p *Parser) getIdentifierValue() string {
	return p.current.Value
}

// Parse parses a query string and returns the Query AST.
func Parse(input string) (*Query, error) {
	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}

	parser := NewParser(tokens)
	return parser.Parse()
}

// ParseSimple parses a simple filter expression like "status:active priority:1"
func ParseSimple(input string) (*Query, error) {
	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}

	parser := NewParser(tokens)
	return parser.parseSimpleFilters()
}

// Parse parses the tokens and returns a Query AST.
func (p *Parser) Parse() (*Query, error) {
	if len(p.tokens) == 0 || p.current.Type == TokenEOF {
		return nil, ErrUnexpectedEOF("query")
	}

	switch p.current.Type {
	case TokenSelect:
		return p.parseSelect()
	case TokenCount:
		return p.parseAggregate(QueryCount)
	case TokenSum:
		return p.parseAggregate(QuerySum)
	case TokenAvg:
		return p.parseAggregate(QueryAvg)
	case TokenMin:
		return p.parseAggregate(QueryMin)
	case TokenMax:
		return p.parseAggregate(QueryMax)
	case TokenUpdate:
		return p.parseUpdate()
	case TokenDelete:
		return p.parseDelete()
	default:
		// Try parsing as simple filter expression
		return p.parseSimpleFilters()
	}
}

// parseSelect parses a SELECT query.
func (p *Parser) parseSelect() (*Query, error) {
	q := &Query{Type: QuerySelect}
	p.advance() // consume 'select'

	// Parse optional fields list or *
	if p.current.Type == TokenIdent && p.current.Value != "*" {
		// Check if this is the entity name or field list
		if p.peek().Type == TokenComma || p.peek().Type == TokenFrom {
			// It's a field list
			fields, err := p.parseIdentList()
			if err != nil {
				return nil, err
			}
			q.Fields = fields
		}
	}

	// Parse FROM clause
	if p.current.Type == TokenFrom {
		p.advance()
	}

	// Parse entity name
	if p.current.Type != TokenIdent {
		return nil, ErrUnexpectedToken(p.current, "entity name")
	}
	q.Entity = p.current.Value
	p.advance()

	// Parse optional clauses
	if err := p.parseClauses(q); err != nil {
		return nil, err
	}

	return q, nil
}

// parseAggregate parses an aggregate query (COUNT, SUM, AVG, MIN, MAX).
func (p *Parser) parseAggregate(qType QueryType) (*Query, error) {
	q := &Query{Type: qType}
	p.advance() // consume aggregate keyword

	// Parse optional field for SUM, AVG, MIN, MAX
	if qType != QueryCount {
		if p.current.Type == TokenLParen {
			p.advance()
			if p.current.Type != TokenIdent {
				return nil, ErrUnexpectedToken(p.current, "field name")
			}
			q.AggField = p.current.Value
			p.advance()
			if p.current.Type != TokenRParen {
				return nil, ErrUnexpectedToken(p.current, ")")
			}
			p.advance()
		}
	}

	// Parse entity name
	if p.current.Type == TokenFrom {
		p.advance()
	}
	if p.current.Type != TokenIdent {
		return nil, ErrUnexpectedToken(p.current, "entity name")
	}
	q.Entity = p.current.Value
	p.advance()

	// Parse optional WHERE clause
	if err := p.parseClauses(q); err != nil {
		return nil, err
	}

	return q, nil
}

// parseUpdate parses an UPDATE query.
func (p *Parser) parseUpdate() (*Query, error) {
	q := &Query{Type: QueryUpdate}
	p.advance() // consume 'update'

	// Parse entity name
	if p.current.Type != TokenIdent {
		return nil, ErrUnexpectedToken(p.current, "entity name")
	}
	q.Entity = p.current.Value
	p.advance()

	// Parse SET clause
	if p.current.Type != TokenSet {
		return nil, ErrUnexpectedToken(p.current, "SET")
	}
	p.advance()

	updates, err := p.parseUpdateSets()
	if err != nil {
		return nil, err
	}
	q.Updates = updates

	// Parse optional WHERE clause
	if err := p.parseClauses(q); err != nil {
		return nil, err
	}

	return q, nil
}

// parseDelete parses a DELETE query.
func (p *Parser) parseDelete() (*Query, error) {
	q := &Query{Type: QueryDelete}
	p.advance() // consume 'delete'

	// Parse FROM if present
	if p.current.Type == TokenFrom {
		p.advance()
	}

	// Parse entity name
	if p.current.Type != TokenIdent {
		return nil, ErrUnexpectedToken(p.current, "entity name")
	}
	q.Entity = p.current.Value
	p.advance()

	// Parse optional WHERE clause
	if err := p.parseClauses(q); err != nil {
		return nil, err
	}

	return q, nil
}

// parseSimpleFilters parses a simple filter expression like "status:active priority:1"
func (p *Parser) parseSimpleFilters() (*Query, error) {
	q := &Query{Type: QuerySelect}
	conditions := make([]Condition, 0)

	for p.current.Type != TokenEOF {
		cond, err := p.parseSimpleFilter()
		if err != nil {
			return nil, err
		}
		if cond != nil {
			conditions = append(conditions, *cond)
		}
	}

	if len(conditions) > 0 {
		q.Where = &WhereClause{
			Conditions: conditions,
			Operator:   LogicalAnd,
		}
	}

	return q, nil
}

// parseSimpleFilter parses a single simple filter like "status:active"
func (p *Parser) parseSimpleFilter() (*Condition, error) {
	// Handle quoted phrases: "exact phrase"
	if p.current.Type == TokenString {
		value := p.current.Value
		p.advance()
		return &Condition{
			Operator: OpExact,
			Value:    value,
		}, nil
	}

	// Handle fuzzy search: ~fuzzy
	if p.current.Type == TokenTilde {
		p.advance()
		if !p.isIdentifierOrKeyword() && p.current.Type != TokenString {
			return nil, ErrUnexpectedToken(p.current, "search term")
		}
		value := p.current.Value
		p.advance()
		return &Condition{
			Operator: OpFuzzy,
			Value:    value,
		}, nil
	}

	// Handle NOT
	not := false
	if p.current.Type == TokenNot {
		not = true
		p.advance()
	}

	// Handle parenthesized expressions
	if p.current.Type == TokenLParen {
		p.depth++
		if p.depth > p.maxDepth {
			return nil, ErrNestedGroupDepth(p.maxDepth)
		}
		p.advance()

		where, err := p.parseWhereExpr()
		if err != nil {
			return nil, err
		}

		if p.current.Type != TokenRParen {
			return nil, ErrUnexpectedToken(p.current, ")")
		}
		p.advance()
		p.depth--

		return &Condition{
			Not:    not,
			Nested: where,
		}, nil
	}

	// Handle field filters: field:value, field=value, field>value, etc.
	if p.isIdentifierOrKeyword() {
		field := p.getIdentifierValue()
		p.advance()

		// Check for operator
		var op CompareOp
		switch p.current.Type {
		case TokenColon:
			op = OpEquals
			p.advance()
		case TokenEquals:
			op = OpEquals
			p.advance()
		case TokenNotEquals:
			op = OpNotEquals
			p.advance()
		case TokenGreater:
			op = OpGreaterThan
			p.advance()
		case TokenGreaterEq:
			op = OpGreaterThanOrEqual
			p.advance()
		case TokenLess:
			op = OpLessThan
			p.advance()
		case TokenLessEq:
			op = OpLessThanOrEqual
			p.advance()
		default:
			// Just an identifier by itself - treat as entity name or search term
			return &Condition{
				Field:    field,
				Operator: OpEquals,
				Value:    true,
			}, nil
		}

		// Parse value
		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}

		return &Condition{
			Field:    field,
			Operator: op,
			Value:    value,
			Not:      not,
		}, nil
	}

	// Handle logical operators between conditions
	if p.current.Type == TokenAnd || p.current.Type == TokenOr {
		p.advance()
		return nil, nil // Let the caller handle this
	}

	return nil, ErrUnexpectedToken(p.current, "filter expression")
}

// parseClauses parses optional query clauses (WHERE, ORDER BY, LIMIT, etc.)
func (p *Parser) parseClauses(q *Query) error {
	for p.current.Type != TokenEOF {
		switch p.current.Type {
		case TokenWith, TokenInclude:
			p.advance()
			includes, err := p.parseIdentList()
			if err != nil {
				return err
			}
			q.Include = includes

		case TokenWhere:
			p.advance()
			where, err := p.parseWhereExpr()
			if err != nil {
				return err
			}
			q.Where = where

		case TokenOrder:
			p.advance()
			if p.current.Type == TokenBy {
				p.advance()
			}
			orderBy, err := p.parseOrderBy()
			if err != nil {
				return err
			}
			q.OrderBy = orderBy

		case TokenGroup:
			p.advance()
			if p.current.Type == TokenBy {
				p.advance()
			}
			groupBy, err := p.parseIdentList()
			if err != nil {
				return err
			}
			q.GroupBy = groupBy

		case TokenGroupBy:
			p.advance()
			groupBy, err := p.parseIdentList()
			if err != nil {
				return err
			}
			q.GroupBy = groupBy

		case TokenHaving:
			p.advance()
			having, err := p.parseWhereExpr()
			if err != nil {
				return err
			}
			q.Having = having

		case TokenLimit:
			p.advance()
			limit, err := p.parseInt()
			if err != nil {
				return err
			}
			q.Limit = &limit

		case TokenOffset:
			p.advance()
			offset, err := p.parseInt()
			if err != nil {
				return err
			}
			q.Offset = &offset

		default:
			return ErrUnexpectedToken(p.current, "clause keyword")
		}
	}
	return nil
}

// parseWhereExpr parses a WHERE expression with AND/OR logic.
func (p *Parser) parseWhereExpr() (*WhereClause, error) {
	clause := &WhereClause{Operator: LogicalAnd}

	cond, err := p.parseCondition()
	if err != nil {
		return nil, err
	}
	clause.Conditions = append(clause.Conditions, cond)

	for {
		if p.current.Type == TokenAnd {
			p.advance()
			cond, err := p.parseCondition()
			if err != nil {
				return nil, err
			}
			clause.Conditions = append(clause.Conditions, cond)
			continue
		}
		if p.current.Type == TokenOr {
			p.advance()
			// If we already have AND conditions, we need to restructure
			if clause.Operator == LogicalAnd && len(clause.Conditions) > 0 {
				clause.Operator = LogicalOr
			}
			cond, err := p.parseCondition()
			if err != nil {
				return nil, err
			}
			clause.Conditions = append(clause.Conditions, cond)
			continue
		}
		break
	}

	return clause, nil
}

// parseCondition parses a single condition.
func (p *Parser) parseCondition() (Condition, error) {
	cond := Condition{}

	// Handle NOT
	if p.current.Type == TokenNot {
		cond.Not = true
		p.advance()
	}

	// Handle parenthesized expression
	if p.current.Type == TokenLParen {
		p.depth++
		if p.depth > p.maxDepth {
			return cond, ErrNestedGroupDepth(p.maxDepth)
		}
		p.advance()

		nested, err := p.parseWhereExpr()
		if err != nil {
			return cond, err
		}

		if p.current.Type != TokenRParen {
			return cond, ErrUnexpectedToken(p.current, ")")
		}
		p.advance()
		p.depth--

		cond.Nested = nested
		return cond, nil
	}

	// Parse field name
	if !p.isIdentifierOrKeyword() {
		return cond, ErrUnexpectedToken(p.current, "field name")
	}
	cond.Field = p.getIdentifierValue()
	p.advance()

	// Parse operator
	switch p.current.Type {
	case TokenEquals:
		cond.Operator = OpEquals
	case TokenNotEquals:
		cond.Operator = OpNotEquals
	case TokenGreater:
		cond.Operator = OpGreaterThan
	case TokenGreaterEq:
		cond.Operator = OpGreaterThanOrEqual
	case TokenLess:
		cond.Operator = OpLessThan
	case TokenLessEq:
		cond.Operator = OpLessThanOrEqual
	case TokenContains:
		cond.Operator = OpContains
	case TokenStartsWith:
		cond.Operator = OpStartsWith
	case TokenEndsWith:
		cond.Operator = OpEndsWith
	case TokenLike:
		cond.Operator = OpLike
	case TokenILike:
		cond.Operator = OpILike
	case TokenIn:
		cond.Operator = OpIn
	case TokenIs:
		p.advance()
		if p.current.Type == TokenNull {
			cond.Operator = OpIsNull
			p.advance()
			return cond, nil
		} else if p.current.Type == TokenNot {
			p.advance()
			if p.current.Type != TokenNull {
				return cond, ErrUnexpectedToken(p.current, "NULL")
			}
			cond.Operator = OpIsNotNull
			p.advance()
			return cond, nil
		}
		return cond, ErrUnexpectedToken(p.current, "NULL or NOT NULL")
	case TokenBetween:
		cond.Operator = OpBetween
		p.advance()
		low, err := p.parseValue()
		if err != nil {
			return cond, err
		}
		if p.current.Type != TokenAnd {
			return cond, ErrUnexpectedToken(p.current, "AND")
		}
		p.advance()
		high, err := p.parseValue()
		if err != nil {
			return cond, err
		}
		cond.Value = BetweenValue{Low: low, High: high}
		return cond, nil
	default:
		return cond, ErrUnexpectedToken(p.current, "comparison operator")
	}
	p.advance()

	// Parse value
	value, err := p.parseValue()
	if err != nil {
		return cond, err
	}
	cond.Value = value

	return cond, nil
}

// parseValue parses a value (string, number, bool, array, or parameter).
func (p *Parser) parseValue() (interface{}, error) {
	switch p.current.Type {
	case TokenString:
		val := p.current.Value
		p.advance()
		return val, nil
	case TokenNumber:
		val := p.current.Value
		p.advance()
		if strings.Contains(val, ".") || strings.Contains(val, "e") || strings.Contains(val, "E") {
			return strconv.ParseFloat(val, 64)
		}
		return strconv.ParseInt(val, 10, 64)
	case TokenTrue:
		p.advance()
		return true, nil
	case TokenFalse:
		p.advance()
		return false, nil
	case TokenNull:
		p.advance()
		return nil, nil
	case TokenParam:
		val := p.current.Value
		p.advance()
		return &Parameter{Name: val}, nil
	case TokenLBracket:
		return p.parseList()
	case TokenIdent:
		// Allow identifiers as values (e.g., for field:value syntax)
		val := p.current.Value
		p.advance()
		// Check for comma-separated values (e.g., tags:frontend,backend)
		if p.current.Type == TokenComma {
			values := []interface{}{val}
			for p.current.Type == TokenComma {
				p.advance()
				if p.current.Type != TokenIdent && p.current.Type != TokenString && p.current.Type != TokenNumber {
					break
				}
				values = append(values, p.current.Value)
				p.advance()
			}
			return values, nil
		}
		return val, nil
	default:
		return nil, ErrUnexpectedToken(p.current, "value")
	}
}

// parseList parses an array value [a, b, c].
func (p *Parser) parseList() ([]interface{}, error) {
	if p.current.Type != TokenLBracket {
		return nil, ErrUnexpectedToken(p.current, "[")
	}
	p.advance()

	values := make([]interface{}, 0)

	for p.current.Type != TokenRBracket && p.current.Type != TokenEOF {
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		values = append(values, val)

		if p.current.Type == TokenComma {
			p.advance()
		} else if p.current.Type != TokenRBracket {
			return nil, ErrUnexpectedToken(p.current, ", or ]")
		}
	}

	if p.current.Type != TokenRBracket {
		return nil, ErrUnexpectedToken(p.current, "]")
	}
	p.advance()

	return values, nil
}

// parseIdentList parses a comma-separated list of identifiers.
func (p *Parser) parseIdentList() ([]string, error) {
	idents := make([]string, 0)

	for {
		if !p.isIdentifierOrKeyword() {
			if len(idents) == 0 {
				return nil, ErrUnexpectedToken(p.current, "identifier")
			}
			break
		}
		idents = append(idents, p.getIdentifierValue())
		p.advance()

		if p.current.Type == TokenComma {
			p.advance()
		} else {
			break
		}
	}

	return idents, nil
}

// parseOrderBy parses ORDER BY clauses.
func (p *Parser) parseOrderBy() ([]OrderClause, error) {
	orders := make([]OrderClause, 0)

	for {
		if !p.isIdentifierOrKeyword() {
			if len(orders) == 0 {
				return nil, ErrUnexpectedToken(p.current, "field name")
			}
			break
		}

		order := OrderClause{
			Field:     p.getIdentifierValue(),
			Direction: OrderAsc,
		}
		p.advance()

		if p.current.Type == TokenAsc {
			p.advance()
		} else if p.current.Type == TokenDesc {
			order.Direction = OrderDesc
			p.advance()
		}

		orders = append(orders, order)

		if p.current.Type == TokenComma {
			p.advance()
		} else {
			break
		}
	}

	return orders, nil
}

// parseUpdateSets parses SET clauses for UPDATE queries.
func (p *Parser) parseUpdateSets() ([]UpdateSet, error) {
	updates := make([]UpdateSet, 0)

	for {
		if !p.isIdentifierOrKeyword() {
			if len(updates) == 0 {
				return nil, ErrUnexpectedToken(p.current, "field name")
			}
			break
		}

		update := UpdateSet{
			Field: p.getIdentifierValue(),
			Op:    UpdateSetValue,
		}
		p.advance()

		if p.current.Type != TokenEquals {
			return nil, ErrUnexpectedToken(p.current, "=")
		}
		p.advance()

		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		update.Value = val

		updates = append(updates, update)

		if p.current.Type == TokenComma {
			p.advance()
		} else {
			break
		}
	}

	return updates, nil
}

// parseInt parses an integer value.
func (p *Parser) parseInt() (int, error) {
	if p.current.Type != TokenNumber {
		return 0, ErrUnexpectedToken(p.current, "integer")
	}
	val, err := strconv.Atoi(p.current.Value)
	if err != nil {
		return 0, ErrInvalidValue(p.current, "integer")
	}
	p.advance()
	return val, nil
}

// advance moves to the next token.
func (p *Parser) advance() {
	p.pos++
	if p.pos < len(p.tokens) {
		p.current = p.tokens[p.pos]
	} else {
		p.current = Token{Type: TokenEOF}
	}
}

// peek looks at the next token without consuming it.
func (p *Parser) peek() Token {
	if p.pos+1 < len(p.tokens) {
		return p.tokens[p.pos+1]
	}
	return Token{Type: TokenEOF}
}

// expect checks if the current token is of the expected type and advances.
func (p *Parser) expect(tokenType TokenType) error {
	if p.current.Type != tokenType {
		return ErrUnexpectedToken(p.current, fmt.Sprintf("%s", tokenTypeNames[tokenType]))
	}
	p.advance()
	return nil
}

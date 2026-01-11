# Task: Query Language Parser and Executor

## Overview
Implement CodeAI's built-in query language for safe, LLM-friendly data operations that compile to parameterized SQL.

## Phase
Phase 2: Core Features

## Priority
High - Core feature for data access.

## Dependencies
- 01-Foundation/05-database-module.md
- 01-Foundation/02-parser-grammar.md

## Description
Create a query language parser and executor that translates CodeAI queries to safe, parameterized SQL while providing a simpler, more LLM-friendly syntax.

## Detailed Requirements

### 1. Query AST (internal/query/ast.go)

```go
package query

type Query struct {
    Type       QueryType
    Entity     string
    Fields     []string      // SELECT fields, empty = all
    Where      *WhereClause
    OrderBy    []OrderClause
    Limit      *int
    Offset     *int
    Include    []string      // Relations to load
    GroupBy    []string
    Having     *WhereClause
}

type QueryType int

const (
    QuerySelect QueryType = iota
    QueryCount
    QuerySum
    QueryAvg
    QueryMin
    QueryMax
    QueryUpdate
    QueryDelete
)

type WhereClause struct {
    Conditions []Condition
    Operator   LogicalOp // AND, OR
}

type Condition struct {
    Field    string
    Operator CompareOp
    Value    any
    SubQuery *Query
}

type CompareOp int

const (
    OpEquals CompareOp = iota
    OpNotEquals
    OpGreaterThan
    OpGreaterThanOrEqual
    OpLessThan
    OpLessThanOrEqual
    OpContains
    OpStartsWith
    OpEndsWith
    OpIn
    OpNotIn
    OpIsNull
    OpIsNotNull
    OpIncludes // For arrays
)

type LogicalOp int

const (
    LogicalAnd LogicalOp = iota
    LogicalOr
)

type OrderClause struct {
    Field     string
    Direction OrderDirection
}

type OrderDirection int

const (
    OrderAsc OrderDirection = iota
    OrderDesc
)

type UpdateSet struct {
    Field string
    Value any
    Op    UpdateOp // set, increment, decrement
}

type UpdateOp int

const (
    UpdateSet UpdateOp = iota
    UpdateIncrement
    UpdateDecrement
)
```

### 2. Query Parser (internal/query/parser.go)

```go
package query

import (
    "fmt"
    "strconv"
    "strings"
)

type Parser struct {
    tokens  []Token
    pos     int
    current Token
}

func Parse(input string) (*Query, error) {
    lexer := NewLexer(input)
    tokens, err := lexer.Tokenize()
    if err != nil {
        return nil, err
    }

    parser := &Parser{tokens: tokens}
    return parser.parse()
}

func (p *Parser) parse() (*Query, error) {
    p.advance()

    switch p.current.Type {
    case TokenSelect:
        return p.parseSelect()
    case TokenCount:
        return p.parseAggregate(QueryCount)
    case TokenSum:
        return p.parseAggregate(QuerySum)
    case TokenAvg:
        return p.parseAggregate(QueryAvg)
    case TokenUpdate:
        return p.parseUpdate()
    case TokenDelete:
        return p.parseDelete()
    default:
        return nil, fmt.Errorf("unexpected token: %s", p.current.Value)
    }
}

func (p *Parser) parseSelect() (*Query, error) {
    q := &Query{Type: QuerySelect}
    p.advance() // consume 'select'

    // Parse entity name
    if p.current.Type != TokenIdent {
        return nil, fmt.Errorf("expected entity name")
    }
    q.Entity = p.current.Value
    p.advance()

    // Parse optional clauses
    for p.current.Type != TokenEOF {
        switch p.current.Type {
        case TokenWith:
            p.advance()
            includes, err := p.parseIdentList()
            if err != nil {
                return nil, err
            }
            q.Include = includes

        case TokenWhere:
            p.advance()
            where, err := p.parseWhere()
            if err != nil {
                return nil, err
            }
            q.Where = where

        case TokenOrder:
            p.advance()
            p.expect(TokenBy)
            orderBy, err := p.parseOrderBy()
            if err != nil {
                return nil, err
            }
            q.OrderBy = orderBy

        case TokenLimit:
            p.advance()
            limit, err := p.parseInt()
            if err != nil {
                return nil, err
            }
            q.Limit = &limit

        case TokenOffset:
            p.advance()
            offset, err := p.parseInt()
            if err != nil {
                return nil, err
            }
            q.Offset = &offset

        default:
            return nil, fmt.Errorf("unexpected token: %s", p.current.Value)
        }
    }

    return q, nil
}

func (p *Parser) parseWhere() (*WhereClause, error) {
    clause := &WhereClause{Operator: LogicalAnd}

    for {
        cond, err := p.parseCondition()
        if err != nil {
            return nil, err
        }
        clause.Conditions = append(clause.Conditions, cond)

        if p.current.Type == TokenAnd {
            p.advance()
            continue
        }
        if p.current.Type == TokenOr {
            p.advance()
            clause.Operator = LogicalOr
            continue
        }
        break
    }

    return clause, nil
}

func (p *Parser) parseCondition() (Condition, error) {
    cond := Condition{}

    if p.current.Type != TokenIdent {
        return cond, fmt.Errorf("expected field name")
    }
    cond.Field = p.current.Value
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
            p.expect(TokenNull)
            cond.Operator = OpIsNotNull
            p.advance()
            return cond, nil
        }
    default:
        return cond, fmt.Errorf("expected comparison operator")
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

func (p *Parser) parseValue() (any, error) {
    switch p.current.Type {
    case TokenString:
        val := p.current.Value
        p.advance()
        return val, nil
    case TokenNumber:
        val := p.current.Value
        p.advance()
        if strings.Contains(val, ".") {
            return strconv.ParseFloat(val, 64)
        }
        return strconv.ParseInt(val, 10, 64)
    case TokenTrue:
        p.advance()
        return true, nil
    case TokenFalse:
        p.advance()
        return false, nil
    case TokenParam:
        val := p.current.Value
        p.advance()
        return &Parameter{Name: val}, nil
    case TokenLBracket:
        return p.parseList()
    default:
        return nil, fmt.Errorf("expected value, got %s", p.current.Value)
    }
}

type Parameter struct {
    Name string
}
```

### 3. SQL Compiler (internal/query/compiler.go)

```go
package query

import (
    "fmt"
    "strings"
)

type SQLCompiler struct {
    entities map[string]*EntityMeta
    params   []any
    paramIdx int
}

type CompiledQuery struct {
    SQL    string
    Params []any
}

func NewSQLCompiler(entities map[string]*EntityMeta) *SQLCompiler {
    return &SQLCompiler{
        entities: entities,
        params:   make([]any, 0),
    }
}

func (c *SQLCompiler) Compile(q *Query) (*CompiledQuery, error) {
    c.params = nil
    c.paramIdx = 0

    var sql string
    var err error

    switch q.Type {
    case QuerySelect:
        sql, err = c.compileSelect(q)
    case QueryCount:
        sql, err = c.compileCount(q)
    case QueryUpdate:
        sql, err = c.compileUpdate(q)
    case QueryDelete:
        sql, err = c.compileDelete(q)
    default:
        err = fmt.Errorf("unsupported query type")
    }

    if err != nil {
        return nil, err
    }

    return &CompiledQuery{SQL: sql, Params: c.params}, nil
}

func (c *SQLCompiler) compileSelect(q *Query) (string, error) {
    entity := c.entities[q.Entity]
    if entity == nil {
        return "", fmt.Errorf("unknown entity: %s", q.Entity)
    }

    var b strings.Builder

    // SELECT
    b.WriteString("SELECT ")
    if len(q.Fields) == 0 {
        b.WriteString("*")
    } else {
        b.WriteString(strings.Join(q.Fields, ", "))
    }

    // FROM
    b.WriteString(" FROM ")
    b.WriteString(entity.TableName)

    // WHERE
    if q.Where != nil {
        whereSQL, err := c.compileWhere(q.Where)
        if err != nil {
            return "", err
        }
        b.WriteString(" WHERE ")
        b.WriteString(whereSQL)
    }

    // Handle soft delete
    if entity.SoftDelete != "" {
        if q.Where != nil {
            b.WriteString(" AND ")
        } else {
            b.WriteString(" WHERE ")
        }
        b.WriteString(entity.SoftDelete)
        b.WriteString(" IS NULL")
    }

    // ORDER BY
    if len(q.OrderBy) > 0 {
        b.WriteString(" ORDER BY ")
        var orders []string
        for _, o := range q.OrderBy {
            dir := "ASC"
            if o.Direction == OrderDesc {
                dir = "DESC"
            }
            orders = append(orders, fmt.Sprintf("%s %s", o.Field, dir))
        }
        b.WriteString(strings.Join(orders, ", "))
    }

    // LIMIT
    if q.Limit != nil {
        b.WriteString(fmt.Sprintf(" LIMIT %d", *q.Limit))
    }

    // OFFSET
    if q.Offset != nil {
        b.WriteString(fmt.Sprintf(" OFFSET %d", *q.Offset))
    }

    return b.String(), nil
}

func (c *SQLCompiler) compileWhere(where *WhereClause) (string, error) {
    var conditions []string
    op := " AND "
    if where.Operator == LogicalOr {
        op = " OR "
    }

    for _, cond := range where.Conditions {
        sql, err := c.compileCondition(cond)
        if err != nil {
            return "", err
        }
        conditions = append(conditions, sql)
    }

    return strings.Join(conditions, op), nil
}

func (c *SQLCompiler) compileCondition(cond Condition) (string, error) {
    c.paramIdx++
    placeholder := fmt.Sprintf("$%d", c.paramIdx)

    switch cond.Operator {
    case OpEquals:
        c.params = append(c.params, cond.Value)
        return fmt.Sprintf("%s = %s", cond.Field, placeholder), nil

    case OpNotEquals:
        c.params = append(c.params, cond.Value)
        return fmt.Sprintf("%s != %s", cond.Field, placeholder), nil

    case OpGreaterThan:
        c.params = append(c.params, cond.Value)
        return fmt.Sprintf("%s > %s", cond.Field, placeholder), nil

    case OpGreaterThanOrEqual:
        c.params = append(c.params, cond.Value)
        return fmt.Sprintf("%s >= %s", cond.Field, placeholder), nil

    case OpLessThan:
        c.params = append(c.params, cond.Value)
        return fmt.Sprintf("%s < %s", cond.Field, placeholder), nil

    case OpLessThanOrEqual:
        c.params = append(c.params, cond.Value)
        return fmt.Sprintf("%s <= %s", cond.Field, placeholder), nil

    case OpContains:
        c.params = append(c.params, "%"+cond.Value.(string)+"%")
        return fmt.Sprintf("%s ILIKE %s", cond.Field, placeholder), nil

    case OpIn:
        values := cond.Value.([]any)
        placeholders := make([]string, len(values))
        for i, v := range values {
            c.paramIdx++
            placeholders[i] = fmt.Sprintf("$%d", c.paramIdx)
            c.params = append(c.params, v)
        }
        c.paramIdx-- // Adjust since we started with +1
        return fmt.Sprintf("%s IN (%s)", cond.Field, strings.Join(placeholders, ", ")), nil

    case OpIsNull:
        c.paramIdx-- // No param needed
        return fmt.Sprintf("%s IS NULL", cond.Field), nil

    case OpIsNotNull:
        c.paramIdx-- // No param needed
        return fmt.Sprintf("%s IS NOT NULL", cond.Field), nil

    default:
        return "", fmt.Errorf("unsupported operator: %d", cond.Operator)
    }
}

func (c *SQLCompiler) compileUpdate(q *Query) (string, error) {
    // Implementation for UPDATE queries
    return "", nil
}

func (c *SQLCompiler) compileDelete(q *Query) (string, error) {
    // Implementation for DELETE queries
    return "", nil
}

func (c *SQLCompiler) compileCount(q *Query) (string, error) {
    entity := c.entities[q.Entity]
    if entity == nil {
        return "", fmt.Errorf("unknown entity: %s", q.Entity)
    }

    var b strings.Builder
    b.WriteString("SELECT COUNT(*) FROM ")
    b.WriteString(entity.TableName)

    if q.Where != nil {
        whereSQL, err := c.compileWhere(q.Where)
        if err != nil {
            return "", err
        }
        b.WriteString(" WHERE ")
        b.WriteString(whereSQL)
    }

    return b.String(), nil
}
```

### 4. Query Executor (internal/query/executor.go)

```go
package query

import (
    "context"

    "github.com/codeai/codeai/internal/modules/database"
)

type Executor struct {
    db       database.Module
    compiler *SQLCompiler
}

func NewExecutor(db database.Module, entities map[string]*EntityMeta) *Executor {
    return &Executor{
        db:       db,
        compiler: NewSQLCompiler(entities),
    }
}

func (e *Executor) Execute(ctx context.Context, q *Query) ([]map[string]any, error) {
    compiled, err := e.compiler.Compile(q)
    if err != nil {
        return nil, err
    }

    return e.db.Query(ctx, compiled.SQL, compiled.Params...)
}

func (e *Executor) ExecuteCount(ctx context.Context, q *Query) (int64, error) {
    compiled, err := e.compiler.Compile(q)
    if err != nil {
        return 0, err
    }

    row, err := e.db.QueryOne(ctx, compiled.SQL, compiled.Params...)
    if err != nil {
        return 0, err
    }

    return row["count"].(int64), nil
}
```

## Acceptance Criteria
- [ ] Parse select, count, update, delete queries
- [ ] Compile to parameterized SQL
- [ ] Support all comparison operators
- [ ] Handle soft deletes automatically
- [ ] Support ordering, limit, offset
- [ ] Include related entities
- [ ] Prevent SQL injection

## Testing Strategy
- Unit tests for parser
- Unit tests for SQL compiler
- Integration tests with database

## Files to Create
- `internal/query/ast.go`
- `internal/query/lexer.go`
- `internal/query/parser.go`
- `internal/query/compiler.go`
- `internal/query/executor.go`
- `internal/query/parser_test.go`

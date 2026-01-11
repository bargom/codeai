# Task: Pagination and Filtering Support

## Overview
Implement consistent pagination and filtering for list endpoints with cursor-based and offset-based pagination options.

## Phase
Phase 2: Core Features

## Priority
Medium - Required for production APIs.

## Dependencies
- 02-Core-Features/04-query-language.md
- 01-Foundation/06-http-module.md

## Description
Create a pagination system supporting both offset and cursor-based pagination, with declarative filtering that integrates with CodeAI endpoint definitions.

## Detailed Requirements

### 1. Pagination Types (internal/pagination/types.go)

```go
package pagination

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
)

type PaginationType string

const (
    PaginationOffset PaginationType = "offset"
    PaginationCursor PaginationType = "cursor"
)

type PageRequest struct {
    Type   PaginationType
    Page   int    // For offset pagination
    Limit  int
    Cursor string // For cursor pagination
    After  string // Alias for cursor
    Before string // For backward pagination
}

type PageResponse[T any] struct {
    Data       []T       `json:"data"`
    Pagination PageInfo  `json:"pagination"`
}

type PageInfo struct {
    Total       *int64  `json:"total,omitempty"`
    Page        *int    `json:"page,omitempty"`
    Limit       int     `json:"limit"`
    HasNext     bool    `json:"has_next"`
    HasPrevious bool    `json:"has_previous"`
    NextCursor  string  `json:"next_cursor,omitempty"`
    PrevCursor  string  `json:"prev_cursor,omitempty"`
}

type Cursor struct {
    ID        string `json:"id"`
    Field     string `json:"field,omitempty"`
    Value     any    `json:"value,omitempty"`
    Direction string `json:"dir,omitempty"`
}

func EncodeCursor(c Cursor) string {
    data, _ := json.Marshal(c)
    return base64.RawURLEncoding.EncodeToString(data)
}

func DecodeCursor(s string) (*Cursor, error) {
    data, err := base64.RawURLEncoding.DecodeString(s)
    if err != nil {
        return nil, fmt.Errorf("invalid cursor")
    }

    var c Cursor
    if err := json.Unmarshal(data, &c); err != nil {
        return nil, fmt.Errorf("invalid cursor")
    }

    return &c, nil
}

func DefaultPageRequest() PageRequest {
    return PageRequest{
        Type:  PaginationOffset,
        Page:  1,
        Limit: 20,
    }
}
```

### 2. Paginator (internal/pagination/paginator.go)

```go
package pagination

import (
    "context"
    "fmt"

    "github.com/codeai/codeai/internal/query"
)

type Paginator struct {
    executor *query.Executor
    entity   string
    baseQuery *query.Query
}

func NewPaginator(executor *query.Executor, entity string) *Paginator {
    return &Paginator{
        executor: executor,
        entity:   entity,
        baseQuery: &query.Query{
            Type:   query.QuerySelect,
            Entity: entity,
        },
    }
}

func (p *Paginator) WithWhere(where *query.WhereClause) *Paginator {
    p.baseQuery.Where = where
    return p
}

func (p *Paginator) WithOrderBy(orderBy []query.OrderClause) *Paginator {
    p.baseQuery.OrderBy = orderBy
    return p
}

func (p *Paginator) WithInclude(include []string) *Paginator {
    p.baseQuery.Include = include
    return p
}

func (p *Paginator) Execute(ctx context.Context, req PageRequest) (*PageResponse[map[string]any], error) {
    switch req.Type {
    case PaginationOffset:
        return p.executeOffset(ctx, req)
    case PaginationCursor:
        return p.executeCursor(ctx, req)
    default:
        return p.executeOffset(ctx, req)
    }
}

func (p *Paginator) executeOffset(ctx context.Context, req PageRequest) (*PageResponse[map[string]any], error) {
    // Clone query
    q := *p.baseQuery
    q.Limit = &req.Limit
    offset := (req.Page - 1) * req.Limit
    q.Offset = &offset

    // Fetch one extra to determine if there's a next page
    fetchLimit := req.Limit + 1
    q.Limit = &fetchLimit

    // Execute query
    results, err := p.executor.Execute(ctx, &q)
    if err != nil {
        return nil, err
    }

    hasNext := len(results) > req.Limit
    if hasNext {
        results = results[:req.Limit]
    }

    // Get total count
    countQuery := &query.Query{
        Type:   query.QueryCount,
        Entity: p.entity,
        Where:  p.baseQuery.Where,
    }
    total, err := p.executor.ExecuteCount(ctx, countQuery)
    if err != nil {
        return nil, err
    }

    return &PageResponse[map[string]any]{
        Data: results,
        Pagination: PageInfo{
            Total:       &total,
            Page:        &req.Page,
            Limit:       req.Limit,
            HasNext:     hasNext,
            HasPrevious: req.Page > 1,
        },
    }, nil
}

func (p *Paginator) executeCursor(ctx context.Context, req PageRequest) (*PageResponse[map[string]any], error) {
    q := *p.baseQuery

    // Ensure we have an order by clause
    if len(q.OrderBy) == 0 {
        q.OrderBy = []query.OrderClause{{Field: "id", Direction: query.OrderAsc}}
    }

    // Apply cursor condition
    if req.Cursor != "" || req.After != "" {
        cursor := req.Cursor
        if cursor == "" {
            cursor = req.After
        }

        decoded, err := DecodeCursor(cursor)
        if err != nil {
            return nil, err
        }

        cursorCond := p.buildCursorCondition(decoded, q.OrderBy, false)
        if q.Where == nil {
            q.Where = &query.WhereClause{Operator: query.LogicalAnd}
        }
        q.Where.Conditions = append(q.Where.Conditions, cursorCond)
    }

    if req.Before != "" {
        decoded, err := DecodeCursor(req.Before)
        if err != nil {
            return nil, err
        }

        cursorCond := p.buildCursorCondition(decoded, q.OrderBy, true)
        if q.Where == nil {
            q.Where = &query.WhereClause{Operator: query.LogicalAnd}
        }
        q.Where.Conditions = append(q.Where.Conditions, cursorCond)
    }

    // Fetch one extra
    fetchLimit := req.Limit + 1
    q.Limit = &fetchLimit

    results, err := p.executor.Execute(ctx, &q)
    if err != nil {
        return nil, err
    }

    hasNext := len(results) > req.Limit
    if hasNext {
        results = results[:req.Limit]
    }

    // Build cursors
    var nextCursor, prevCursor string
    if hasNext && len(results) > 0 {
        last := results[len(results)-1]
        nextCursor = p.buildCursor(last, q.OrderBy)
    }
    if len(results) > 0 && (req.Cursor != "" || req.After != "") {
        first := results[0]
        prevCursor = p.buildCursor(first, q.OrderBy)
    }

    return &PageResponse[map[string]any]{
        Data: results,
        Pagination: PageInfo{
            Limit:       req.Limit,
            HasNext:     hasNext,
            HasPrevious: req.Cursor != "" || req.After != "",
            NextCursor:  nextCursor,
            PrevCursor:  prevCursor,
        },
    }, nil
}

func (p *Paginator) buildCursorCondition(cursor *Cursor, orderBy []query.OrderClause, before bool) query.Condition {
    field := "id"
    if len(orderBy) > 0 {
        field = orderBy[0].Field
    }

    op := query.OpGreaterThan
    if before || (len(orderBy) > 0 && orderBy[0].Direction == query.OrderDesc) {
        op = query.OpLessThan
    }

    return query.Condition{
        Field:    field,
        Operator: op,
        Value:    cursor.Value,
    }
}

func (p *Paginator) buildCursor(record map[string]any, orderBy []query.OrderClause) string {
    field := "id"
    if len(orderBy) > 0 {
        field = orderBy[0].Field
    }

    return EncodeCursor(Cursor{
        ID:    fmt.Sprintf("%v", record["id"]),
        Field: field,
        Value: record[field],
    })
}
```

### 3. Filter Builder (internal/pagination/filter.go)

```go
package pagination

import (
    "fmt"
    "strings"

    "github.com/codeai/codeai/internal/query"
)

type FilterDef struct {
    Field      string
    Operator   string // eq, ne, gt, gte, lt, lte, contains, in
    AllowedOps []string
}

type FilterBuilder struct {
    definitions map[string]FilterDef
}

func NewFilterBuilder() *FilterBuilder {
    return &FilterBuilder{
        definitions: make(map[string]FilterDef),
    }
}

func (b *FilterBuilder) Allow(field string, ops ...string) *FilterBuilder {
    if len(ops) == 0 {
        ops = []string{"eq", "ne", "gt", "gte", "lt", "lte", "contains", "in"}
    }
    b.definitions[field] = FilterDef{
        Field:      field,
        AllowedOps: ops,
    }
    return b
}

func (b *FilterBuilder) Build(params map[string]any) (*query.WhereClause, error) {
    where := &query.WhereClause{Operator: query.LogicalAnd}

    for key, value := range params {
        // Parse field[op] syntax: e.g., price[gte]=100
        field, op := parseFilterKey(key)

        def, ok := b.definitions[field]
        if !ok {
            continue // Ignore unknown fields
        }

        if !contains(def.AllowedOps, op) {
            return nil, fmt.Errorf("operator '%s' not allowed for field '%s'", op, field)
        }

        cond, err := buildCondition(field, op, value)
        if err != nil {
            return nil, err
        }

        where.Conditions = append(where.Conditions, cond)
    }

    if len(where.Conditions) == 0 {
        return nil, nil
    }

    return where, nil
}

func parseFilterKey(key string) (field, op string) {
    if idx := strings.Index(key, "["); idx != -1 {
        field = key[:idx]
        op = strings.TrimSuffix(key[idx+1:], "]")
    } else {
        field = key
        op = "eq"
    }
    return
}

func buildCondition(field, op string, value any) (query.Condition, error) {
    cond := query.Condition{Field: field, Value: value}

    switch op {
    case "eq":
        cond.Operator = query.OpEquals
    case "ne":
        cond.Operator = query.OpNotEquals
    case "gt":
        cond.Operator = query.OpGreaterThan
    case "gte":
        cond.Operator = query.OpGreaterThanOrEqual
    case "lt":
        cond.Operator = query.OpLessThan
    case "lte":
        cond.Operator = query.OpLessThanOrEqual
    case "contains":
        cond.Operator = query.OpContains
    case "in":
        cond.Operator = query.OpIn
    default:
        return cond, fmt.Errorf("unknown operator: %s", op)
    }

    return cond, nil
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

### 4. HTTP Integration

```go
// internal/pagination/http.go
package pagination

import (
    "net/http"
    "strconv"
)

func ParsePageRequest(r *http.Request) PageRequest {
    req := DefaultPageRequest()

    q := r.URL.Query()

    if page := q.Get("page"); page != "" {
        if p, err := strconv.Atoi(page); err == nil && p > 0 {
            req.Page = p
        }
    }

    if limit := q.Get("limit"); limit != "" {
        if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
            req.Limit = l
        }
    }

    if cursor := q.Get("cursor"); cursor != "" {
        req.Type = PaginationCursor
        req.Cursor = cursor
    }

    if after := q.Get("after"); after != "" {
        req.Type = PaginationCursor
        req.After = after
    }

    if before := q.Get("before"); before != "" {
        req.Type = PaginationCursor
        req.Before = before
    }

    return req
}

func ParseFilters(r *http.Request, allowed []string) map[string]any {
    filters := make(map[string]any)
    q := r.URL.Query()

    for _, field := range allowed {
        // Check exact match
        if v := q.Get(field); v != "" {
            filters[field] = v
        }

        // Check operator variants
        for _, op := range []string{"eq", "ne", "gt", "gte", "lt", "lte", "contains", "in"} {
            key := field + "[" + op + "]"
            if v := q.Get(key); v != "" {
                filters[key] = v
            }
        }
    }

    return filters
}
```

## Acceptance Criteria
- [ ] Offset-based pagination with page/limit
- [ ] Cursor-based pagination with opaque cursors
- [ ] Total count for offset pagination
- [ ] Filter parsing from query params
- [ ] Filter[operator] syntax support
- [ ] Integration with query executor

## Testing Strategy
- Unit tests for pagination logic
- Unit tests for cursor encoding/decoding
- Integration tests with database

## Files to Create
- `internal/pagination/types.go`
- `internal/pagination/paginator.go`
- `internal/pagination/filter.go`
- `internal/pagination/http.go`
- `internal/pagination/paginator_test.go`

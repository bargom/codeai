package pagination

import (
	"context"
	"fmt"

	"github.com/bargom/codeai/internal/query"
)

// Paginator executes paginated queries against a database.
type Paginator struct {
	executor    *query.Executor
	entity      string
	baseQuery   *query.Query
	includeTotal bool
}

// NewPaginator creates a new Paginator for the given entity.
func NewPaginator(executor *query.Executor, entity string) *Paginator {
	return &Paginator{
		executor: executor,
		entity:   entity,
		baseQuery: &query.Query{
			Type:   query.QuerySelect,
			Entity: entity,
		},
		includeTotal: true,
	}
}

// WithWhere sets the WHERE clause for the paginator.
func (p *Paginator) WithWhere(where *query.WhereClause) *Paginator {
	p.baseQuery.Where = where
	return p
}

// WithOrderBy sets the ORDER BY clauses for the paginator.
func (p *Paginator) WithOrderBy(orderBy []query.OrderClause) *Paginator {
	p.baseQuery.OrderBy = orderBy
	return p
}

// WithInclude sets the relations to include.
func (p *Paginator) WithInclude(include []string) *Paginator {
	p.baseQuery.Include = include
	return p
}

// WithFields sets the fields to select.
func (p *Paginator) WithFields(fields []string) *Paginator {
	p.baseQuery.Fields = fields
	return p
}

// WithTotal enables or disables total count in offset pagination.
func (p *Paginator) WithTotal(include bool) *Paginator {
	p.includeTotal = include
	return p
}

// Execute executes the paginated query based on the request type.
func (p *Paginator) Execute(ctx context.Context, req PageRequest) (*PageResponse[map[string]any], error) {
	req.Validate()

	if req.IsCursorPagination() {
		return p.executeCursor(ctx, req)
	}
	return p.executeOffset(ctx, req)
}

// executeOffset executes offset-based pagination.
func (p *Paginator) executeOffset(ctx context.Context, req PageRequest) (*PageResponse[map[string]any], error) {
	// Clone query
	q := p.cloneQuery()
	offset := req.GetOffset()
	q.Offset = &offset

	// Fetch one extra to determine if there's a next page
	fetchLimit := req.Limit + 1
	q.Limit = &fetchLimit

	// Execute query
	results, err := p.executor.Execute(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("pagination query failed: %w", err)
	}

	hasNext := len(results) > req.Limit
	if hasNext {
		results = results[:req.Limit]
	}

	response := &PageResponse[map[string]any]{
		Data: results,
		Pagination: PageInfo{
			Page:        &req.Page,
			Limit:       req.Limit,
			HasNext:     hasNext,
			HasPrevious: req.Page > 1,
		},
	}

	// Get total count if requested
	if p.includeTotal {
		total, err := p.getTotal(ctx)
		if err != nil {
			return nil, fmt.Errorf("count query failed: %w", err)
		}
		response.Pagination.Total = &total
	}

	return response, nil
}

// executeCursor executes cursor-based pagination.
func (p *Paginator) executeCursor(ctx context.Context, req PageRequest) (*PageResponse[map[string]any], error) {
	q := p.cloneQuery()

	// Ensure we have an order by clause
	if len(q.OrderBy) == 0 {
		q.OrderBy = []query.OrderClause{{Field: "id", Direction: query.OrderAsc}}
	}

	// Apply cursor condition for forward pagination
	cursor := req.GetCursor()
	if cursor != "" {
		decoded, err := DecodeCursor(cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}

		cursorCond := p.buildCursorCondition(decoded, q.OrderBy, false)
		q.Where = p.appendCondition(q.Where, cursorCond)
	}

	// Apply cursor condition for backward pagination
	if req.Before != "" {
		decoded, err := DecodeCursor(req.Before)
		if err != nil {
			return nil, fmt.Errorf("invalid before cursor: %w", err)
		}

		cursorCond := p.buildCursorCondition(decoded, q.OrderBy, true)
		q.Where = p.appendCondition(q.Where, cursorCond)
	}

	// Fetch one extra to determine if there's a next page
	fetchLimit := req.Limit + 1
	q.Limit = &fetchLimit

	results, err := p.executor.Execute(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("pagination query failed: %w", err)
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
	if len(results) > 0 && cursor != "" {
		first := results[0]
		prevCursor = p.buildCursor(first, q.OrderBy)
	}

	return &PageResponse[map[string]any]{
		Data: results,
		Pagination: PageInfo{
			Limit:       req.Limit,
			HasNext:     hasNext,
			HasPrevious: cursor != "",
			NextCursor:  nextCursor,
			PrevCursor:  prevCursor,
		},
	}, nil
}

// cloneQuery creates a shallow copy of the base query.
func (p *Paginator) cloneQuery() *query.Query {
	q := *p.baseQuery
	// Copy slices to avoid mutation
	if len(p.baseQuery.OrderBy) > 0 {
		q.OrderBy = make([]query.OrderClause, len(p.baseQuery.OrderBy))
		copy(q.OrderBy, p.baseQuery.OrderBy)
	}
	if len(p.baseQuery.Fields) > 0 {
		q.Fields = make([]string, len(p.baseQuery.Fields))
		copy(q.Fields, p.baseQuery.Fields)
	}
	if len(p.baseQuery.Include) > 0 {
		q.Include = make([]string, len(p.baseQuery.Include))
		copy(q.Include, p.baseQuery.Include)
	}
	return &q
}

// getTotal returns the total count of records matching the base query.
func (p *Paginator) getTotal(ctx context.Context) (int64, error) {
	countQuery := &query.Query{
		Type:   query.QueryCount,
		Entity: p.entity,
		Where:  p.baseQuery.Where,
	}
	return p.executor.ExecuteCount(ctx, countQuery)
}

// appendCondition appends a condition to the WHERE clause.
func (p *Paginator) appendCondition(where *query.WhereClause, cond query.Condition) *query.WhereClause {
	if where == nil {
		return &query.WhereClause{
			Operator:   query.LogicalAnd,
			Conditions: []query.Condition{cond},
		}
	}
	// Clone the where clause
	newWhere := &query.WhereClause{
		Operator:   where.Operator,
		Conditions: make([]query.Condition, len(where.Conditions), len(where.Conditions)+1),
	}
	copy(newWhere.Conditions, where.Conditions)
	newWhere.Conditions = append(newWhere.Conditions, cond)
	return newWhere
}

// buildCursorCondition creates a condition for cursor-based pagination.
func (p *Paginator) buildCursorCondition(cursor *Cursor, orderBy []query.OrderClause, before bool) query.Condition {
	field := "id"
	if len(orderBy) > 0 {
		field = orderBy[0].Field
	}
	if cursor.Field != "" {
		field = cursor.Field
	}

	// Determine comparison operator based on sort direction and pagination direction
	op := query.OpGreaterThan
	isDesc := len(orderBy) > 0 && orderBy[0].Direction == query.OrderDesc

	// For forward pagination (after):
	//   ASC order: id > cursor (get items after)
	//   DESC order: id < cursor (get items after in descending order)
	// For backward pagination (before):
	//   ASC order: id < cursor (get items before)
	//   DESC order: id > cursor (get items before in descending order)
	if before {
		if isDesc {
			op = query.OpGreaterThan
		} else {
			op = query.OpLessThan
		}
	} else {
		if isDesc {
			op = query.OpLessThan
		} else {
			op = query.OpGreaterThan
		}
	}

	return query.Condition{
		Field:    field,
		Operator: op,
		Value:    cursor.Value,
	}
}

// buildCursor creates an encoded cursor from a record.
func (p *Paginator) buildCursor(record map[string]any, orderBy []query.OrderClause) string {
	field := "id"
	if len(orderBy) > 0 {
		field = orderBy[0].Field
	}

	idVal := ""
	if id, ok := record["id"]; ok {
		idVal = fmt.Sprintf("%v", id)
	}

	return EncodeCursor(Cursor{
		ID:    idVal,
		Field: field,
		Value: record[field],
	})
}

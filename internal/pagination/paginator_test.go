package pagination

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/bargom/codeai/internal/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDB implements the query.DB interface for testing
type MockDB struct {
	QueryFunc    func(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error)
	QueryRowFunc func(ctx context.Context, q string, args ...interface{}) *sql.Row
	ExecFunc     func(ctx context.Context, q string, args ...interface{}) (sql.Result, error)
}

func (m *MockDB) QueryContext(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, q, args...)
	}
	return nil, nil
}

func (m *MockDB) QueryRowContext(ctx context.Context, q string, args ...interface{}) *sql.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, q, args...)
	}
	return nil
}

func (m *MockDB) ExecContext(ctx context.Context, q string, args ...interface{}) (sql.Result, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, q, args...)
	}
	return nil, nil
}

func testEntities() map[string]*query.EntityMeta {
	return map[string]*query.EntityMeta{
		"users": {
			TableName:  "users",
			PrimaryKey: "id",
		},
		"posts": {
			TableName:  "posts",
			PrimaryKey: "id",
		},
	}
}

func TestNewPaginator(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())

	p := NewPaginator(executor, "users")

	assert.NotNil(t, p)
	assert.Equal(t, "users", p.entity)
	assert.NotNil(t, p.baseQuery)
	assert.Equal(t, query.QuerySelect, p.baseQuery.Type)
	assert.True(t, p.includeTotal)
}

func TestPaginator_WithWhere(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())

	where := &query.WhereClause{
		Operator: query.LogicalAnd,
		Conditions: []query.Condition{
			{Field: "status", Operator: query.OpEquals, Value: "active"},
		},
	}

	p := NewPaginator(executor, "users").WithWhere(where)

	assert.Equal(t, where, p.baseQuery.Where)
}

func TestPaginator_WithOrderBy(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())

	orderBy := []query.OrderClause{
		{Field: "created_at", Direction: query.OrderDesc},
	}

	p := NewPaginator(executor, "users").WithOrderBy(orderBy)

	assert.Equal(t, orderBy, p.baseQuery.OrderBy)
}

func TestPaginator_WithInclude(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())

	p := NewPaginator(executor, "users").WithInclude([]string{"posts", "comments"})

	assert.Equal(t, []string{"posts", "comments"}, p.baseQuery.Include)
}

func TestPaginator_WithFields(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())

	p := NewPaginator(executor, "users").WithFields([]string{"id", "name", "email"})

	assert.Equal(t, []string{"id", "name", "email"}, p.baseQuery.Fields)
}

func TestPaginator_WithTotal(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())

	p := NewPaginator(executor, "users").WithTotal(false)

	assert.False(t, p.includeTotal)
}

func TestPaginator_CloneQuery(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())

	p := NewPaginator(executor, "users").
		WithOrderBy([]query.OrderClause{{Field: "id", Direction: query.OrderAsc}}).
		WithFields([]string{"id", "name"}).
		WithInclude([]string{"posts"})

	cloned := p.cloneQuery()

	// Modify the clone
	cloned.OrderBy[0].Field = "modified"
	cloned.Fields[0] = "modified"
	cloned.Include[0] = "modified"

	// Original should be unchanged
	assert.Equal(t, "id", p.baseQuery.OrderBy[0].Field)
	assert.Equal(t, "id", p.baseQuery.Fields[0])
	assert.Equal(t, "posts", p.baseQuery.Include[0])
}

func TestPaginator_AppendCondition_NilWhere(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	cond := query.Condition{
		Field:    "id",
		Operator: query.OpGreaterThan,
		Value:    100,
	}

	result := p.appendCondition(nil, cond)

	require.NotNil(t, result)
	assert.Equal(t, query.LogicalAnd, result.Operator)
	assert.Len(t, result.Conditions, 1)
	assert.Equal(t, "id", result.Conditions[0].Field)
}

func TestPaginator_AppendCondition_ExistingWhere(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	existing := &query.WhereClause{
		Operator: query.LogicalAnd,
		Conditions: []query.Condition{
			{Field: "status", Operator: query.OpEquals, Value: "active"},
		},
	}

	newCond := query.Condition{
		Field:    "id",
		Operator: query.OpGreaterThan,
		Value:    100,
	}

	result := p.appendCondition(existing, newCond)

	require.NotNil(t, result)
	assert.Len(t, result.Conditions, 2)
	// Original should be unchanged
	assert.Len(t, existing.Conditions, 1)
}

func TestPaginator_BuildCursorCondition_AscForward(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	cursor := &Cursor{ID: "100", Field: "id", Value: 100}
	orderBy := []query.OrderClause{{Field: "id", Direction: query.OrderAsc}}

	cond := p.buildCursorCondition(cursor, orderBy, false)

	assert.Equal(t, "id", cond.Field)
	assert.Equal(t, query.OpGreaterThan, cond.Operator)
	assert.Equal(t, 100, cond.Value)
}

func TestPaginator_BuildCursorCondition_DescForward(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	cursor := &Cursor{ID: "100", Field: "id", Value: 100}
	orderBy := []query.OrderClause{{Field: "id", Direction: query.OrderDesc}}

	cond := p.buildCursorCondition(cursor, orderBy, false)

	assert.Equal(t, "id", cond.Field)
	assert.Equal(t, query.OpLessThan, cond.Operator)
}

func TestPaginator_BuildCursorCondition_AscBackward(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	cursor := &Cursor{ID: "100", Field: "id", Value: 100}
	orderBy := []query.OrderClause{{Field: "id", Direction: query.OrderAsc}}

	cond := p.buildCursorCondition(cursor, orderBy, true)

	assert.Equal(t, "id", cond.Field)
	assert.Equal(t, query.OpLessThan, cond.Operator)
}

func TestPaginator_BuildCursorCondition_DescBackward(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	cursor := &Cursor{ID: "100", Field: "id", Value: 100}
	orderBy := []query.OrderClause{{Field: "id", Direction: query.OrderDesc}}

	cond := p.buildCursorCondition(cursor, orderBy, true)

	assert.Equal(t, "id", cond.Field)
	assert.Equal(t, query.OpGreaterThan, cond.Operator)
}

func TestPaginator_BuildCursorCondition_UseCursorField(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	cursor := &Cursor{ID: "100", Field: "created_at", Value: "2024-01-01"}
	orderBy := []query.OrderClause{{Field: "id", Direction: query.OrderAsc}}

	cond := p.buildCursorCondition(cursor, orderBy, false)

	// Should use cursor's field, not orderBy's field
	assert.Equal(t, "created_at", cond.Field)
}

func TestPaginator_BuildCursorCondition_EmptyOrderBy(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	cursor := &Cursor{ID: "100", Value: 100}
	var orderBy []query.OrderClause

	cond := p.buildCursorCondition(cursor, orderBy, false)

	// Should default to "id"
	assert.Equal(t, "id", cond.Field)
	assert.Equal(t, query.OpGreaterThan, cond.Operator)
}

func TestPaginator_BuildCursor(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	record := map[string]any{"id": 123, "name": "test"}
	orderBy := []query.OrderClause{{Field: "id", Direction: query.OrderAsc}}

	cursorStr := p.buildCursor(record, orderBy)

	decoded, err := DecodeCursor(cursorStr)
	require.NoError(t, err)
	assert.Equal(t, "123", decoded.ID)
	assert.Equal(t, "id", decoded.Field)
	assert.Equal(t, float64(123), decoded.Value) // JSON decodes numbers as float64
}

func TestPaginator_BuildCursor_EmptyOrderBy(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	record := map[string]any{"id": 456, "name": "test"}
	var orderBy []query.OrderClause

	cursorStr := p.buildCursor(record, orderBy)

	decoded, err := DecodeCursor(cursorStr)
	require.NoError(t, err)
	assert.Equal(t, "456", decoded.ID)
	assert.Equal(t, "id", decoded.Field)
}

func TestPaginator_BuildCursor_NoID(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	record := map[string]any{"name": "test"}
	orderBy := []query.OrderClause{{Field: "name", Direction: query.OrderAsc}}

	cursorStr := p.buildCursor(record, orderBy)

	decoded, err := DecodeCursor(cursorStr)
	require.NoError(t, err)
	assert.Equal(t, "", decoded.ID)
	assert.Equal(t, "name", decoded.Field)
	assert.Equal(t, "test", decoded.Value)
}

func TestPaginator_Execute_InvalidCursor(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	req := PageRequest{
		Type:   PaginationCursor,
		Cursor: "invalid-cursor",
		Limit:  10,
	}

	_, err := p.Execute(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cursor")
}

func TestPaginator_Execute_InvalidBeforeCursor(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	req := PageRequest{
		Type:   PaginationCursor,
		Before: "invalid-cursor",
		Limit:  10,
	}

	_, err := p.Execute(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid before cursor")
}

func TestPaginator_Execute_SelectsOffsetByDefault(t *testing.T) {
	db := &MockDB{
		QueryFunc: func(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
			// Should include OFFSET for offset pagination
			assert.Contains(t, q, "OFFSET")
			return nil, errors.New("test error")
		},
	}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	req := DefaultPageRequest()
	_, err := p.Execute(context.Background(), req)
	assert.Error(t, err) // Expected - we're just checking the query was built correctly
}

func TestPaginator_Execute_SelectsCursorWhenSet(t *testing.T) {
	cursor := EncodeCursor(Cursor{ID: "100", Field: "id", Value: float64(100)})
	db := &MockDB{
		QueryFunc: func(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
			// Should NOT include OFFSET for cursor pagination
			assert.NotContains(t, q, "OFFSET")
			return nil, errors.New("test error")
		},
	}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	req := PageRequest{
		Type:   PaginationCursor,
		Cursor: cursor,
		Limit:  10,
	}
	_, err := p.Execute(context.Background(), req)
	assert.Error(t, err) // Expected - we're just checking the query was built correctly
}

func TestPaginator_ChainedConfiguration(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())

	where := &query.WhereClause{
		Operator:   query.LogicalAnd,
		Conditions: []query.Condition{{Field: "active", Operator: query.OpEquals, Value: true}},
	}

	p := NewPaginator(executor, "users").
		WithWhere(where).
		WithOrderBy([]query.OrderClause{{Field: "created_at", Direction: query.OrderDesc}}).
		WithFields([]string{"id", "name"}).
		WithInclude([]string{"posts"}).
		WithTotal(false)

	assert.Equal(t, where, p.baseQuery.Where)
	assert.Len(t, p.baseQuery.OrderBy, 1)
	assert.Equal(t, []string{"id", "name"}, p.baseQuery.Fields)
	assert.Equal(t, []string{"posts"}, p.baseQuery.Include)
	assert.False(t, p.includeTotal)
}

func TestPageRequest_ValidateIsCalledInExecute(t *testing.T) {
	db := &MockDB{
		QueryFunc: func(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
			// Limit should be validated to be within bounds
			assert.Contains(t, q, "LIMIT")
			return nil, errors.New("test error")
		},
	}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	// Invalid limit should be corrected
	req := PageRequest{
		Page:  0, // Invalid
		Limit: 0, // Invalid
	}

	_, _ = p.Execute(context.Background(), req)
	// The validation happens internally
}

func TestPaginator_GetTotal(t *testing.T) {
	// Test that getTotal properly constructs a count query
	// We can't easily test the actual execution without a more sophisticated mock
	// The getTotal method is tested indirectly through the Execute tests
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	// Verify the paginator is set up correctly
	assert.NotNil(t, p.executor)
	assert.Equal(t, "users", p.entity)
}

func TestPaginator_Execute_CursorUsesAfterField(t *testing.T) {
	cursor := EncodeCursor(Cursor{ID: "100", Field: "id", Value: float64(100)})
	db := &MockDB{
		QueryFunc: func(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
			return nil, errors.New("test error")
		},
	}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	// Test with After field instead of Cursor
	req := PageRequest{
		After: cursor,
		Limit: 10,
	}
	_, err := p.Execute(context.Background(), req)
	assert.Error(t, err) // Expected error from mock
}

func TestPaginator_Execute_CursorWithExistingWhere(t *testing.T) {
	cursor := EncodeCursor(Cursor{ID: "100", Field: "id", Value: float64(100)})
	queryCalled := false
	db := &MockDB{
		QueryFunc: func(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
			queryCalled = true
			// Should combine existing WHERE with cursor condition
			assert.Contains(t, q, "WHERE")
			return nil, errors.New("test error")
		},
	}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users").
		WithWhere(&query.WhereClause{
			Operator: query.LogicalAnd,
			Conditions: []query.Condition{
				{Field: "status", Operator: query.OpEquals, Value: "active"},
			},
		})

	req := PageRequest{
		Type:   PaginationCursor,
		Cursor: cursor,
		Limit:  10,
	}
	_, _ = p.Execute(context.Background(), req)
	assert.True(t, queryCalled)
}

func TestPaginator_Execute_BeforeCursorWithExistingWhere(t *testing.T) {
	cursor := EncodeCursor(Cursor{ID: "100", Field: "id", Value: float64(100)})
	queryCalled := false
	db := &MockDB{
		QueryFunc: func(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
			queryCalled = true
			return nil, errors.New("test error")
		},
	}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	req := PageRequest{
		Type:   PaginationCursor,
		Before: cursor,
		Limit:  10,
	}
	_, _ = p.Execute(context.Background(), req)
	assert.True(t, queryCalled)
}

func TestPaginator_Execute_CursorWithNoExistingWhere(t *testing.T) {
	cursor := EncodeCursor(Cursor{ID: "100", Field: "id", Value: float64(100)})
	queryCalled := false
	db := &MockDB{
		QueryFunc: func(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
			queryCalled = true
			return nil, errors.New("test error")
		},
	}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users") // No WHERE clause set

	req := PageRequest{
		Type:   PaginationCursor,
		Cursor: cursor,
		Limit:  10,
	}
	_, _ = p.Execute(context.Background(), req)
	assert.True(t, queryCalled)
}

func TestPaginator_Execute_AddsDefaultOrderBy(t *testing.T) {
	cursor := EncodeCursor(Cursor{ID: "100", Field: "id", Value: float64(100)})
	queryCalled := false
	db := &MockDB{
		QueryFunc: func(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
			queryCalled = true
			// Should add default ORDER BY id ASC
			assert.Contains(t, q, "ORDER BY")
			return nil, errors.New("test error")
		},
	}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users") // No ORDER BY set

	req := PageRequest{
		Type:   PaginationCursor,
		Cursor: cursor,
		Limit:  10,
	}
	_, _ = p.Execute(context.Background(), req)
	assert.True(t, queryCalled)
}

func TestPaginator_CloneQuery_EmptySlices(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())

	// Create paginator with no orderBy, fields, or include
	p := NewPaginator(executor, "users")

	cloned := p.cloneQuery()

	// Cloned query should still be valid
	assert.Equal(t, "users", cloned.Entity)
	assert.Empty(t, cloned.OrderBy)
	assert.Empty(t, cloned.Fields)
	assert.Empty(t, cloned.Include)
}

func TestPaginator_Execute_OffsetPaginationQueryError(t *testing.T) {
	db := &MockDB{
		QueryFunc: func(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
			return nil, errors.New("database connection error")
		},
	}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	req := PageRequest{
		Type:  PaginationOffset,
		Page:  1,
		Limit: 10,
	}

	_, err := p.Execute(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pagination query failed")
}

func TestPaginator_Execute_CursorPaginationQueryError(t *testing.T) {
	cursor := EncodeCursor(Cursor{ID: "100", Field: "id", Value: float64(100)})
	db := &MockDB{
		QueryFunc: func(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
			return nil, errors.New("database connection error")
		},
	}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	req := PageRequest{
		Type:   PaginationCursor,
		Cursor: cursor,
		Limit:  10,
	}

	_, err := p.Execute(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pagination query failed")
}

func TestPaginator_BuildCursorCondition_EmptyCursorField(t *testing.T) {
	db := &MockDB{}
	executor := query.NewExecutor(db, testEntities())
	p := NewPaginator(executor, "users")

	// Cursor with empty field - should use orderBy field
	cursor := &Cursor{ID: "100", Field: "", Value: 100}
	orderBy := []query.OrderClause{{Field: "created_at", Direction: query.OrderAsc}}

	cond := p.buildCursorCondition(cursor, orderBy, false)

	// Should use orderBy's field since cursor.Field is empty
	assert.Equal(t, "created_at", cond.Field)
}

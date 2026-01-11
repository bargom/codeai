package query

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDB implements the DB interface for testing
type MockDB struct {
	QueryFunc    func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowFunc func(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecFunc     func(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func (m *MockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, query, args...)
	}
	return nil, nil
}

func (m *MockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, query, args...)
	}
	return nil
}

func (m *MockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, query, args...)
	}
	return nil, nil
}

func TestQueryBuilder_Select(t *testing.T) {
	q := Select("users").
		Fields("id", "name", "email").
		Where("status", OpEquals, "active").
		OrderBy("created_at", OrderDesc).
		Limit(10).
		Offset(20).
		Build()

	assert.Equal(t, QuerySelect, q.Type)
	assert.Equal(t, "users", q.Entity)
	assert.Equal(t, []string{"id", "name", "email"}, q.Fields)
	assert.NotNil(t, q.Where)
	assert.Len(t, q.Where.Conditions, 1)
	assert.Len(t, q.OrderBy, 1)
	assert.Equal(t, OrderDesc, q.OrderBy[0].Direction)
	require.NotNil(t, q.Limit)
	assert.Equal(t, 10, *q.Limit)
	require.NotNil(t, q.Offset)
	assert.Equal(t, 20, *q.Offset)
}

func TestQueryBuilder_Count(t *testing.T) {
	q := Count("users").
		Where("status", OpEquals, "active").
		Build()

	assert.Equal(t, QueryCount, q.Type)
	assert.Equal(t, "users", q.Entity)
	assert.NotNil(t, q.Where)
}

func TestQueryBuilder_Update(t *testing.T) {
	q := Update("users").
		Set("name", "John").
		Set("status", "active").
		Increment("login_count", 1).
		Where("id", OpEquals, 123).
		Build()

	assert.Equal(t, QueryUpdate, q.Type)
	assert.Equal(t, "users", q.Entity)
	assert.Len(t, q.Updates, 3)
	assert.Equal(t, UpdateSetValue, q.Updates[0].Op)
	assert.Equal(t, UpdateIncrement, q.Updates[2].Op)
}

func TestQueryBuilder_Delete(t *testing.T) {
	q := Delete("users").
		Where("id", OpEquals, 123).
		Build()

	assert.Equal(t, QueryDelete, q.Type)
	assert.Equal(t, "users", q.Entity)
	assert.NotNil(t, q.Where)
}

func TestQueryBuilder_Include(t *testing.T) {
	q := Select("users").
		Include("posts", "comments").
		Build()

	assert.Equal(t, []string{"posts", "comments"}, q.Include)
}

func TestQueryBuilder_WhereClause(t *testing.T) {
	where := &WhereClause{
		Conditions: []Condition{
			{Field: "status", Operator: OpEquals, Value: "active"},
			{Field: "age", Operator: OpGreaterThan, Value: 18},
		},
		Operator: LogicalAnd,
	}

	q := Select("users").
		WhereClause(where).
		Build()

	assert.Equal(t, where, q.Where)
}

func TestQueryBuilder_Decrement(t *testing.T) {
	q := Update("products").
		Decrement("stock", 1).
		Where("id", OpEquals, 123).
		Build()

	assert.Len(t, q.Updates, 1)
	assert.Equal(t, UpdateDecrement, q.Updates[0].Op)
}

func TestQueryBuilder_MultipleWhere(t *testing.T) {
	q := Select("users").
		Where("status", OpEquals, "active").
		Where("age", OpGreaterThan, 18).
		Where("verified", OpEquals, true).
		Build()

	assert.Len(t, q.Where.Conditions, 3)
}

func TestExecutor_ExecuteUpdate_WrongType(t *testing.T) {
	exec := NewExecutor(&MockDB{}, testEntities())

	q := &Query{
		Type:   QuerySelect, // Wrong type
		Entity: "users",
	}

	_, err := exec.ExecuteUpdate(context.Background(), q)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected UPDATE query")
}

func TestExecutor_ExecuteDelete_WrongType(t *testing.T) {
	exec := NewExecutor(&MockDB{}, testEntities())

	q := &Query{
		Type:   QuerySelect, // Wrong type
		Entity: "users",
	}

	_, err := exec.ExecuteDelete(context.Background(), q)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected DELETE query")
}

func TestExecutor_CompileError(t *testing.T) {
	exec := NewExecutor(&MockDB{}, testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "unknown_entity",
	}

	_, err := exec.Execute(context.Background(), q)
	require.Error(t, err)
}

func TestExecutor_ExecuteString(t *testing.T) {
	exec := NewExecutor(&MockDB{}, testEntities())

	// Invalid query string should return an error
	_, err := exec.ExecuteString(context.Background(), "")
	require.Error(t, err)
}

func TestExecutor_ExecuteCount_ConvertsSelectToCount(t *testing.T) {
	// This tests that ExecuteCount properly converts a SELECT query to COUNT
	queryCalled := false
	exec := NewExecutor(&MockDB{
		QueryRowFunc: func(ctx context.Context, query string, args ...interface{}) *sql.Row {
			queryCalled = true
			assert.Contains(t, query, "COUNT(*)")
			// Return nil - this will cause a panic in Scan, which is expected
			// The important thing is that we verified the query was correct
			return nil
		},
	}, testEntities())

	q := &Query{
		Type:   QuerySelect, // Will be converted to COUNT
		Entity: "users",
	}

	// We expect this to panic because we're returning nil from QueryRowFunc
	// But we've verified the query was properly converted to COUNT
	defer func() {
		// Recover from the expected panic
		if r := recover(); r != nil {
			// Expected panic - the query was converted correctly
			assert.True(t, queryCalled, "QueryRowFunc should have been called")
		}
	}()
	_, _ = exec.ExecuteCount(context.Background(), q)
}

func TestExecutor_ExecuteOne_AddsLimit(t *testing.T) {
	exec := NewExecutor(&MockDB{
		QueryFunc: func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
			assert.Contains(t, query, "LIMIT 1")
			return nil, sql.ErrNoRows
		},
	}, testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
	}

	// Will error but we've verified the LIMIT was added
	_, _ = exec.ExecuteOne(context.Background(), q)
	assert.NotNil(t, q.Limit)
	assert.Equal(t, 1, *q.Limit)
}

// Test AST type String methods
func TestQueryType_String(t *testing.T) {
	tests := []struct {
		qt       QueryType
		expected string
	}{
		{QuerySelect, "SELECT"},
		{QueryCount, "COUNT"},
		{QuerySum, "SUM"},
		{QueryAvg, "AVG"},
		{QueryMin, "MIN"},
		{QueryMax, "MAX"},
		{QueryUpdate, "UPDATE"},
		{QueryDelete, "DELETE"},
		{QueryType(999), "Unknown(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.qt.String())
		})
	}
}

func TestCompareOp_String(t *testing.T) {
	tests := []struct {
		op       CompareOp
		expected string
	}{
		{OpEquals, "="},
		{OpNotEquals, "!="},
		{OpGreaterThan, ">"},
		{OpGreaterThanOrEqual, ">="},
		{OpLessThan, "<"},
		{OpLessThanOrEqual, "<="},
		{OpContains, "CONTAINS"},
		{OpStartsWith, "STARTSWITH"},
		{OpEndsWith, "ENDSWITH"},
		{OpIn, "IN"},
		{OpNotIn, "NOT IN"},
		{OpIsNull, "IS NULL"},
		{OpIsNotNull, "IS NOT NULL"},
		{OpIncludes, "INCLUDES"},
		{OpLike, "LIKE"},
		{OpILike, "ILIKE"},
		{OpBetween, "BETWEEN"},
		{OpFuzzy, "~"},
		{OpExact, "EXACT"},
		{OpArrayContains, "@>"},
		{CompareOp(999), "Unknown(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.op.String())
		})
	}
}

func TestLogicalOp_String(t *testing.T) {
	tests := []struct {
		op       LogicalOp
		expected string
	}{
		{LogicalAnd, "AND"},
		{LogicalOr, "OR"},
		{LogicalOp(999), "Unknown(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.op.String())
		})
	}
}

func TestOrderDirection_String(t *testing.T) {
	tests := []struct {
		od       OrderDirection
		expected string
	}{
		{OrderAsc, "ASC"},
		{OrderDesc, "DESC"},
		{OrderDirection(999), "Unknown(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.od.String())
		})
	}
}

func TestUpdateOp_String(t *testing.T) {
	tests := []struct {
		op       UpdateOp
		expected string
	}{
		{UpdateSetValue, "SET"},
		{UpdateIncrement, "INCREMENT"},
		{UpdateDecrement, "DECREMENT"},
		{UpdateAppend, "APPEND"},
		{UpdateRemove, "REMOVE"},
		{UpdateOp(999), "Unknown(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.op.String())
		})
	}
}

func TestRelationType_String(t *testing.T) {
	tests := []struct {
		rt       RelationType
		expected string
	}{
		{RelationHasOne, "HAS_ONE"},
		{RelationHasMany, "HAS_MANY"},
		{RelationBelongsTo, "BELONGS_TO"},
		{RelationManyToMany, "MANY_TO_MANY"},
		{RelationType(999), "Unknown(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.rt.String())
		})
	}
}

// Package integration provides integration tests for the CodeAI modules.
package integration

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/bargom/codeai/internal/pagination"
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
	return nil, errors.New("QueryFunc not implemented")
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
	return nil, errors.New("ExecFunc not implemented")
}

// TestQueryParsingAndCompilation tests the full query parsing -> compilation flow.
func TestQueryParsingAndCompilation(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedEntity string
		expectedType   query.QueryType
		hasWhere       bool
		hasOrderBy     bool
		hasLimit       bool
	}{
		{
			name:           "simple select",
			input:          "select users",
			expectedEntity: "users",
			expectedType:   query.QuerySelect,
		},
		{
			name:           "select with where",
			input:          `select users where status = "active"`,
			expectedEntity: "users",
			expectedType:   query.QuerySelect,
			hasWhere:       true,
		},
		{
			name:           "select with order and limit",
			input:          "select users order by created_at desc limit 10",
			expectedEntity: "users",
			expectedType:   query.QuerySelect,
			hasOrderBy:     true,
			hasLimit:       true,
		},
		{
			name:           "count query",
			input:          `count users where status = "active"`,
			expectedEntity: "users",
			expectedType:   query.QueryCount,
			hasWhere:       true,
		},
		{
			name:           "update query",
			input:          `update users set status = "inactive" where id = 123`,
			expectedEntity: "users",
			expectedType:   query.QueryUpdate,
			hasWhere:       true,
		},
		{
			name:           "delete query",
			input:          `delete from users where status = "deleted"`,
			expectedEntity: "users",
			expectedType:   query.QueryDelete,
			hasWhere:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := query.Parse(tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedEntity, q.Entity)
			assert.Equal(t, tt.expectedType, q.Type)

			if tt.hasWhere {
				assert.NotNil(t, q.Where)
			}
			if tt.hasOrderBy {
				assert.NotEmpty(t, q.OrderBy)
			}
			if tt.hasLimit {
				assert.NotNil(t, q.Limit)
			}

			// Test compilation for SELECT queries
			if q.Type == query.QuerySelect {
				entities := map[string]*query.EntityMeta{
					"users": {
						TableName:  "users",
						PrimaryKey: "id",
					},
				}
				compiler := query.NewSQLCompiler(entities)
				compiled, err := compiler.Compile(q)
				require.NoError(t, err)
				assert.NotEmpty(t, compiled.SQL)
			}
		})
	}
}

// TestQueryCompilerWithComplexConditions tests complex WHERE conditions compilation.
func TestQueryCompilerWithComplexConditions(t *testing.T) {
	entities := map[string]*query.EntityMeta{
		"users": {
			TableName:  "users",
			PrimaryKey: "id",
		},
	}
	compiler := query.NewSQLCompiler(entities)

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "multiple AND conditions",
			input: `select users where status = "active" and age > 18 and verified = true`,
		},
		{
			name:  "OR conditions",
			input: `select users where role = "admin" or role = "moderator"`,
		},
		{
			name:  "IN operator",
			input: `select users where status in ["active", "pending"]`,
		},
		{
			name:  "BETWEEN operator",
			input: `select users where age between 18 and 65`,
		},
		{
			name:  "IS NULL",
			input: `select users where deleted_at is null`,
		},
		{
			name:  "IS NOT NULL",
			input: `select users where email is not null`,
		},
		{
			name:  "LIKE operator",
			input: `select users where name like "%john%"`,
		},
		{
			name:  "complex nested conditions",
			input: `select users where (status = "active" or status = "pending") and age >= 21`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := query.Parse(tt.input)
			require.NoError(t, err)

			compiled, err := compiler.Compile(q)
			require.NoError(t, err)
			assert.NotEmpty(t, compiled.SQL)

			// Verify the SQL contains expected components
			assert.Contains(t, compiled.SQL, "SELECT")
			assert.Contains(t, compiled.SQL, "users")
			assert.Contains(t, compiled.SQL, "WHERE")
		})
	}
}

// TestQueryPaginationIntegration tests query parsing with pagination.
func TestQueryPaginationIntegration(t *testing.T) {
	entities := map[string]*query.EntityMeta{
		"users": {
			TableName:  "users",
			PrimaryKey: "id",
		},
		"posts": {
			TableName:  "posts",
			PrimaryKey: "id",
		},
	}

	db := &MockDB{
		QueryFunc: func(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
			// Verify the query contains pagination elements
			if q != "" {
				assert.Contains(t, q, "LIMIT")
			}
			return nil, errors.New("mock error - expected")
		},
	}
	executor := query.NewExecutor(db, entities)

	t.Run("offset pagination setup", func(t *testing.T) {
		p := pagination.NewPaginator(executor, "users").
			WithOrderBy([]query.OrderClause{{Field: "created_at", Direction: query.OrderDesc}}).
			WithTotal(true)

		req := pagination.PageRequest{
			Type:  pagination.PaginationOffset,
			Page:  1,
			Limit: 20,
		}

		// Execute - will fail due to mock but validates query building
		_, err := p.Execute(context.Background(), req)
		assert.Error(t, err) // Expected mock error
	})

	t.Run("cursor pagination setup", func(t *testing.T) {
		cursor := pagination.EncodeCursor(pagination.Cursor{
			ID:    "100",
			Field: "id",
			Value: float64(100),
		})

		p := pagination.NewPaginator(executor, "posts").
			WithOrderBy([]query.OrderClause{{Field: "id", Direction: query.OrderAsc}})

		req := pagination.PageRequest{
			Type:   pagination.PaginationCursor,
			Cursor: cursor,
			Limit:  10,
		}

		// Execute - will fail due to mock but validates cursor handling
		_, err := p.Execute(context.Background(), req)
		assert.Error(t, err) // Expected mock error
	})

	t.Run("paginator with where clause", func(t *testing.T) {
		where := &query.WhereClause{
			Operator: query.LogicalAnd,
			Conditions: []query.Condition{
				{Field: "status", Operator: query.OpEquals, Value: "active"},
			},
		}

		p := pagination.NewPaginator(executor, "users").
			WithWhere(where).
			WithOrderBy([]query.OrderClause{{Field: "name", Direction: query.OrderAsc}}).
			WithFields([]string{"id", "name", "email"})

		req := pagination.DefaultPageRequest()

		// Execute - will fail due to mock but validates where clause handling
		_, err := p.Execute(context.Background(), req)
		assert.Error(t, err) // Expected mock error
	})
}

// TestCursorEncodingDecoding tests cursor serialization roundtrip.
func TestCursorEncodingDecoding(t *testing.T) {
	tests := []struct {
		name   string
		cursor pagination.Cursor
	}{
		{
			name: "simple numeric cursor",
			cursor: pagination.Cursor{
				ID:    "123",
				Field: "id",
				Value: float64(123),
			},
		},
		{
			name: "string value cursor",
			cursor: pagination.Cursor{
				ID:    "abc",
				Field: "name",
				Value: "test-value",
			},
		},
		{
			name: "timestamp cursor",
			cursor: pagination.Cursor{
				ID:    "2024-01-15T10:30:00Z",
				Field: "created_at",
				Value: "2024-01-15T10:30:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := pagination.EncodeCursor(tt.cursor)
			assert.NotEmpty(t, encoded)

			decoded, err := pagination.DecodeCursor(encoded)
			require.NoError(t, err)

			assert.Equal(t, tt.cursor.ID, decoded.ID)
			assert.Equal(t, tt.cursor.Field, decoded.Field)
			assert.Equal(t, tt.cursor.Value, decoded.Value)
		})
	}

	t.Run("invalid cursor returns error", func(t *testing.T) {
		_, err := pagination.DecodeCursor("invalid-base64!")
		assert.Error(t, err)
	})

	t.Run("empty cursor returns error", func(t *testing.T) {
		_, err := pagination.DecodeCursor("")
		assert.Error(t, err)
	})
}

// TestPageRequestValidation tests PageRequest validation.
func TestPageRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request pagination.PageRequest
	}{
		{
			name: "default request is valid",
			request: pagination.PageRequest{
				Type:  pagination.PaginationOffset,
				Page:  1,
				Limit: 20,
			},
		},
		{
			name: "negative page gets corrected",
			request: pagination.PageRequest{
				Type:  pagination.PaginationOffset,
				Page:  -1,
				Limit: 20,
			},
		},
		{
			name: "zero limit gets corrected",
			request: pagination.PageRequest{
				Type:  pagination.PaginationOffset,
				Page:  1,
				Limit: 0,
			},
		},
		{
			name: "large limit gets capped",
			request: pagination.PageRequest{
				Type:  pagination.PaginationOffset,
				Page:  1,
				Limit: 10000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy since Validate modifies in place
			req := tt.request
			req.Validate()

			// Page should be at least 1
			assert.GreaterOrEqual(t, req.Page, 1)

			// Limit should be within bounds
			assert.GreaterOrEqual(t, req.Limit, 1)
			assert.LessOrEqual(t, req.Limit, pagination.MaxLimit)
		})
	}
}

// TestSimpleQueryParsing tests simple filter syntax parsing.
func TestSimpleQueryParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		numConds int
	}{
		{
			name:     "single field filter",
			input:    "status:active",
			numConds: 1,
		},
		{
			name:     "multiple field filters",
			input:    "status:active priority:high",
			numConds: 2,
		},
		{
			name:     "comparison filter",
			input:    "age>18",
			numConds: 1,
		},
		{
			name:     "exact phrase search",
			input:    `"hello world"`,
			numConds: 1,
		},
		{
			name:     "fuzzy search",
			input:    "~search",
			numConds: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := query.ParseSimple(tt.input)
			require.NoError(t, err)

			if tt.numConds > 0 {
				require.NotNil(t, q.Where)
				assert.Len(t, q.Where.Conditions, tt.numConds)
			}
		})
	}
}

// TestQueryExecutorSetup tests the query executor setup.
func TestQueryExecutorSetup(t *testing.T) {
	entities := map[string]*query.EntityMeta{
		"users": {
			TableName:  "users",
			PrimaryKey: "id",
		},
		"posts": {
			TableName:  "posts",
			PrimaryKey: "id",
		},
		"comments": {
			TableName:  "comments",
			PrimaryKey: "id",
		},
	}

	db := &MockDB{}
	executor := query.NewExecutor(db, entities)

	assert.NotNil(t, executor)
}

// TestAggregateQueries tests aggregate query parsing.
func TestAggregateQueries(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		qType    query.QueryType
		aggField string
	}{
		{
			name:  "count",
			input: "count users",
			qType: query.QueryCount,
		},
		{
			name:     "sum",
			input:    "sum(amount) orders",
			qType:    query.QuerySum,
			aggField: "amount",
		},
		{
			name:     "avg",
			input:    "avg(price) products",
			qType:    query.QueryAvg,
			aggField: "price",
		},
		{
			name:     "min",
			input:    "min(created_at) users",
			qType:    query.QueryMin,
			aggField: "created_at",
		},
		{
			name:     "max",
			input:    "max(score) results",
			qType:    query.QueryMax,
			aggField: "score",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := query.Parse(tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.qType, q.Type)
			if tt.aggField != "" {
				assert.Equal(t, tt.aggField, q.AggField)
			}
		})
	}
}

// TestQueryWithIncludes tests query with include/with clauses.
func TestQueryWithIncludes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		includes []string
	}{
		{
			name:     "single include",
			input:    "select users with posts",
			includes: []string{"posts"},
		},
		{
			name:     "multiple includes",
			input:    "select users include posts, comments, likes",
			includes: []string{"posts", "comments", "likes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := query.Parse(tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.includes, q.Include)
		})
	}
}

// TestQueryWithGroupByHaving tests GROUP BY and HAVING clauses.
func TestQueryWithGroupByHaving(t *testing.T) {
	t.Run("group by", func(t *testing.T) {
		input := "select orders group by status"
		q, err := query.Parse(input)
		require.NoError(t, err)
		assert.Equal(t, []string{"status"}, q.GroupBy)
	})

	t.Run("group by with having", func(t *testing.T) {
		input := "select orders group by user_id having count > 5"
		q, err := query.Parse(input)
		require.NoError(t, err)
		assert.NotEmpty(t, q.GroupBy)
		assert.NotNil(t, q.Having)
	})
}

// TestPaginatorChaining tests fluent API chaining on paginator.
func TestPaginatorChaining(t *testing.T) {
	entities := map[string]*query.EntityMeta{
		"users": {TableName: "users", PrimaryKey: "id"},
	}
	db := &MockDB{}
	executor := query.NewExecutor(db, entities)

	where := &query.WhereClause{
		Operator: query.LogicalAnd,
		Conditions: []query.Condition{
			{Field: "active", Operator: query.OpEquals, Value: true},
		},
	}

	p := pagination.NewPaginator(executor, "users").
		WithWhere(where).
		WithOrderBy([]query.OrderClause{
			{Field: "created_at", Direction: query.OrderDesc},
			{Field: "name", Direction: query.OrderAsc},
		}).
		WithFields([]string{"id", "name", "email"}).
		WithInclude([]string{"posts", "comments"}).
		WithTotal(false)

	assert.NotNil(t, p)
}

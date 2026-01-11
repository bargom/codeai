package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testEntities() map[string]*EntityMeta {
	return map[string]*EntityMeta{
		"users": {
			TableName:  "users",
			PrimaryKey: "id",
			SoftDelete: "deleted_at",
			Columns: map[string]string{
				"id":        "id",
				"name":      "name",
				"email":     "email",
				"status":    "status",
				"age":       "age",
				"createdAt": "created_at",
			},
			JSONColumns: map[string]bool{
				"metadata": true,
			},
			TSVColumns: map[string]string{
				"search": "search_vector",
			},
		},
		"posts": {
			TableName:  "posts",
			PrimaryKey: "id",
			Columns: map[string]string{
				"id":      "id",
				"title":   "title",
				"content": "content",
				"userId":  "user_id",
			},
		},
		"orders": {
			TableName:  "orders",
			PrimaryKey: "id",
			Columns: map[string]string{
				"id":     "id",
				"amount": "amount",
				"status": "status",
			},
		},
	}
}

func TestCompiler_SelectAll(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)

	assert.Equal(t, `SELECT * FROM "users" WHERE "deleted_at" IS NULL`, compiled.SQL)
	assert.Empty(t, compiled.Params)
}

func TestCompiler_SelectFields(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Fields: []string{"id", "name", "email"},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)

	assert.Equal(t, `SELECT "id", "name", "email" FROM "users" WHERE "deleted_at" IS NULL`, compiled.SQL)
}

func TestCompiler_SelectWithWhere(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "status", Operator: OpEquals, Value: "active"},
			},
			Operator: LogicalAnd,
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)

	assert.Equal(t, `SELECT * FROM "users" WHERE ("status" = $1) AND ("deleted_at" IS NULL)`, compiled.SQL)
	assert.Equal(t, []interface{}{"active"}, compiled.Params)
}

func TestCompiler_SelectWithMultipleConditions(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "status", Operator: OpEquals, Value: "active"},
				{Field: "age", Operator: OpGreaterThanOrEqual, Value: int64(18)},
			},
			Operator: LogicalAnd,
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)

	assert.Contains(t, compiled.SQL, `"status" = $1`)
	assert.Contains(t, compiled.SQL, `"age" >= $2`)
	assert.Contains(t, compiled.SQL, " AND ")
	assert.Len(t, compiled.Params, 2)
}

func TestCompiler_SelectWithOrConditions(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "status", Operator: OpEquals, Value: "active"},
				{Field: "status", Operator: OpEquals, Value: "pending"},
			},
			Operator: LogicalOr,
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)

	assert.Contains(t, compiled.SQL, " OR ")
}

func TestCompiler_ComparisonOperators(t *testing.T) {
	tests := []struct {
		name     string
		op       CompareOp
		value    interface{}
		expected string
	}{
		{
			name:     "equals",
			op:       OpEquals,
			value:    "test",
			expected: `"age" = $1`,
		},
		{
			name:     "not equals",
			op:       OpNotEquals,
			value:    "test",
			expected: `"age" != $1`,
		},
		{
			name:     "greater than",
			op:       OpGreaterThan,
			value:    18,
			expected: `"age" > $1`,
		},
		{
			name:     "greater than or equal",
			op:       OpGreaterThanOrEqual,
			value:    18,
			expected: `"age" >= $1`,
		},
		{
			name:     "less than",
			op:       OpLessThan,
			value:    100,
			expected: `"age" < $1`,
		},
		{
			name:     "less than or equal",
			op:       OpLessThanOrEqual,
			value:    100,
			expected: `"age" <= $1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewSQLCompiler(testEntities())
			q := &Query{
				Type:   QuerySelect,
				Entity: "users",
				Where: &WhereClause{
					Conditions: []Condition{
						{Field: "age", Operator: tt.op, Value: tt.value},
					},
				},
			}

			compiled, err := compiler.Compile(q)
			require.NoError(t, err)
			assert.Contains(t, compiled.SQL, tt.expected)
		})
	}
}

func TestCompiler_ContainsOperator(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "name", Operator: OpContains, Value: "john"},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)

	assert.Contains(t, compiled.SQL, `"name" ILIKE $1`)
	assert.Equal(t, []interface{}{"%john%"}, compiled.Params)
}

func TestCompiler_StartsWithEndsWith(t *testing.T) {
	tests := []struct {
		name          string
		op            CompareOp
		value         string
		expectedParam string
	}{
		{
			name:          "starts with",
			op:            OpStartsWith,
			value:         "admin",
			expectedParam: "admin%",
		},
		{
			name:          "ends with",
			op:            OpEndsWith,
			value:         "@example.com",
			expectedParam: "%@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewSQLCompiler(testEntities())
			q := &Query{
				Type:   QuerySelect,
				Entity: "users",
				Where: &WhereClause{
					Conditions: []Condition{
						{Field: "email", Operator: tt.op, Value: tt.value},
					},
				},
			}

			compiled, err := compiler.Compile(q)
			require.NoError(t, err)
			assert.Contains(t, compiled.SQL, "ILIKE")
			assert.Equal(t, tt.expectedParam, compiled.Params[0])
		})
	}
}

func TestCompiler_InOperator(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "status", Operator: OpIn, Value: []interface{}{"active", "pending", "approved"}},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)

	assert.Contains(t, compiled.SQL, `"status" IN ($1, $2, $3)`)
	assert.Len(t, compiled.Params, 3)
}

func TestCompiler_InOperatorEmpty(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "status", Operator: OpIn, Value: []interface{}{}},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "FALSE")
}

func TestCompiler_IsNull(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "posts", // No soft delete
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "content", Operator: OpIsNull},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, `"content" IS NULL`)
}

func TestCompiler_IsNotNull(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "posts",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "content", Operator: OpIsNotNull},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, `"content" IS NOT NULL`)
}

func TestCompiler_BetweenOperator(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "age", Operator: OpBetween, Value: BetweenValue{Low: 18, High: 65}},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, `"age" BETWEEN $1 AND $2`)
	assert.Equal(t, 18, compiled.Params[0])
	assert.Equal(t, 65, compiled.Params[1])
}

func TestCompiler_NotCondition(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "status", Operator: OpEquals, Value: "inactive", Not: true},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, `NOT "status" = $1`)
}

func TestCompiler_OrderBy(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		OrderBy: []OrderClause{
			{Field: "createdAt", Direction: OrderDesc},
			{Field: "name", Direction: OrderAsc},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, `ORDER BY "created_at" DESC, "name" ASC`)
}

func TestCompiler_LimitOffset(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	limit := 10
	offset := 20
	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Limit:  &limit,
		Offset: &offset,
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "LIMIT 10")
	assert.Contains(t, compiled.SQL, "OFFSET 20")
}

func TestCompiler_Count(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QueryCount,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "status", Operator: OpEquals, Value: "active"},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Equal(t, `SELECT COUNT(*) FROM "users" WHERE ("status" = $1) AND ("deleted_at" IS NULL)`, compiled.SQL)
}

func TestCompiler_AggregateSum(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:     QuerySum,
		Entity:   "orders",
		AggField: "amount",
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Equal(t, `SELECT SUM("amount") FROM "orders"`, compiled.SQL)
}

func TestCompiler_AggregateAvg(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:     QueryAvg,
		Entity:   "orders",
		AggField: "amount",
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "AVG")
}

func TestCompiler_AggregateMinMax(t *testing.T) {
	tests := []struct {
		qType    QueryType
		expected string
	}{
		{QueryMin, "MIN"},
		{QueryMax, "MAX"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			compiler := NewSQLCompiler(testEntities())
			q := &Query{
				Type:     tt.qType,
				Entity:   "orders",
				AggField: "amount",
			}

			compiled, err := compiler.Compile(q)
			require.NoError(t, err)
			assert.Contains(t, compiled.SQL, tt.expected)
		})
	}
}

func TestCompiler_Update(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QueryUpdate,
		Entity: "users",
		Updates: []UpdateSet{
			{Field: "status", Value: "active", Op: UpdateSetValue},
			{Field: "name", Value: "John", Op: UpdateSetValue},
		},
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "id", Operator: OpEquals, Value: 123},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)

	assert.Contains(t, compiled.SQL, `UPDATE "users"`)
	assert.Contains(t, compiled.SQL, `SET "status" = $1, "name" = $2`)
	assert.Contains(t, compiled.SQL, `WHERE`)
	assert.Len(t, compiled.Params, 3)
}

func TestCompiler_UpdateIncrement(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QueryUpdate,
		Entity: "posts",
		Updates: []UpdateSet{
			{Field: "viewCount", Value: 1, Op: UpdateIncrement},
		},
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "id", Operator: OpEquals, Value: 1},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, `"view_count" = "view_count" + $1`)
}

func TestCompiler_UpdateDecrement(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QueryUpdate,
		Entity: "posts",
		Updates: []UpdateSet{
			{Field: "count", Value: 1, Op: UpdateDecrement},
		},
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "id", Operator: OpEquals, Value: 1},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, `- $1`)
}

func TestCompiler_Delete(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QueryDelete,
		Entity: "posts", // No soft delete
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "id", Operator: OpEquals, Value: 123},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, `DELETE FROM "posts"`)
}

func TestCompiler_SoftDelete(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QueryDelete,
		Entity: "users", // Has soft delete
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "id", Operator: OpEquals, Value: 123},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, `UPDATE "users"`)
	assert.Contains(t, compiled.SQL, `SET "deleted_at" = NOW()`)
}

func TestCompiler_UnknownEntity(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "unknown_entity",
	}

	_, err := compiler.Compile(q)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown entity")
}

func TestCompiler_JSONColumn(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "metadata", Operator: OpEquals, Value: map[string]string{"key": "value"}},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "@>")
	assert.Contains(t, compiled.SQL, "::jsonb")
}

func TestCompiler_NestedConditions(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	// (status = "active" OR status = "pending") AND age >= 18
	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{
					Nested: &WhereClause{
						Conditions: []Condition{
							{Field: "status", Operator: OpEquals, Value: "active"},
							{Field: "status", Operator: OpEquals, Value: "pending"},
						},
						Operator: LogicalOr,
					},
				},
				{Field: "age", Operator: OpGreaterThanOrEqual, Value: 18},
			},
			Operator: LogicalAnd,
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, " OR ")
	assert.Contains(t, compiled.SQL, " AND ")
}

func TestCompiler_NotNestedCondition(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{
					Not: true,
					Nested: &WhereClause{
						Conditions: []Condition{
							{Field: "status", Operator: OpEquals, Value: "deleted"},
						},
					},
				},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "NOT (")
}

func TestCompiler_GroupBy(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:    QuerySelect,
		Entity:  "orders",
		Fields:  []string{"status"},
		GroupBy: []string{"status"},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, `GROUP BY "status"`)
}

func TestCompiler_Having(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:    QuerySelect,
		Entity:  "orders",
		Fields:  []string{"status"},
		GroupBy: []string{"status"},
		Having: &WhereClause{
			Conditions: []Condition{
				{Field: "count", Operator: OpGreaterThan, Value: 10},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "HAVING")
}

func TestCompiler_CaseInsensitiveEntityLookup(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "USERS", // uppercase
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, `FROM "users"`)
}

func TestCompiler_ColumnMapping(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		OrderBy: []OrderClause{
			{Field: "createdAt", Direction: OrderDesc}, // camelCase to snake_case
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, `"created_at"`)
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"createdAt", "created_at"},
		{"userId", "user_id"},
		{"id", "id"},
		{"simplecase", "simplecase"},
		{"htmlParser", "html_parser"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompiler_ArrayContains(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "tags", Operator: OpArrayContains, Value: []string{"admin"}},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "@>")
}

func TestCompiler_UpdateAppendRemove(t *testing.T) {
	tests := []struct {
		name     string
		op       UpdateOp
		expected string
	}{
		{
			name:     "append",
			op:       UpdateAppend,
			expected: "array_append",
		},
		{
			name:     "remove",
			op:       UpdateRemove,
			expected: "array_remove",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewSQLCompiler(testEntities())
			q := &Query{
				Type:   QueryUpdate,
				Entity: "posts",
				Updates: []UpdateSet{
					{Field: "tags", Value: "new-tag", Op: tt.op},
				},
				Where: &WhereClause{
					Conditions: []Condition{
						{Field: "id", Operator: OpEquals, Value: 1},
					},
				},
			}

			compiled, err := compiler.Compile(q)
			require.NoError(t, err)
			assert.Contains(t, compiled.SQL, tt.expected)
		})
	}
}

func TestCompiler_LikeOperator(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "name", Operator: OpLike, Value: "John%"},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, `"name" LIKE $1`)
}

func TestCompiler_NotInOperator(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "status", Operator: OpNotIn, Value: []interface{}{"deleted", "banned"}},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, `"status" NOT IN`)
}

func TestCompiler_NotInOperatorEmpty(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "status", Operator: OpNotIn, Value: []interface{}{}},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "TRUE")
}

func TestCompiler_QuoteIdentWithDot(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())
	result := compiler.quoteIdent("schema.table")
	assert.Equal(t, `"schema"."table"`, result)
}

func TestCompiler_QuoteIdentWithQuotes(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())
	result := compiler.quoteIdent(`table"name`)
	assert.Equal(t, `"table""name"`, result)
}

func TestCompiler_EmptyWhere(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "posts", // No soft delete
		Where:  &WhereClause{},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.NotContains(t, compiled.SQL, "WHERE")
}

func TestCompiler_SingleConditionWhere(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "posts",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "title", Operator: OpEquals, Value: "Test"},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "WHERE")
}

func TestCompiler_InOperatorInvalidValue(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "status", Operator: OpIn, Value: "not-an-array"}, // Invalid
			},
		},
	}

	_, err := compiler.Compile(q)
	require.Error(t, err)
}

func TestCompiler_NotInOperatorInvalidValue(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "status", Operator: OpNotIn, Value: "not-an-array"}, // Invalid
			},
		},
	}

	_, err := compiler.Compile(q)
	require.Error(t, err)
}

func TestCompiler_BetweenInvalidValue(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "age", Operator: OpBetween, Value: "not-between-value"}, // Invalid
			},
		},
	}

	_, err := compiler.Compile(q)
	require.Error(t, err)
}

func TestCompiler_CountWithSoftDelete(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QueryCount,
		Entity: "users", // Has soft delete
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "COUNT(*)")
	assert.Contains(t, compiled.SQL, "deleted_at")
}

func TestCompiler_AggregateWithWhere(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:     QuerySum,
		Entity:   "orders",
		AggField: "amount",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "status", Operator: OpEquals, Value: "completed"},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "SUM")
	assert.Contains(t, compiled.SQL, "WHERE")
}

func TestCompiler_DeleteWithWhere(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QueryDelete,
		Entity: "posts",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "id", Operator: OpEquals, Value: 1},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "DELETE")
	assert.Contains(t, compiled.SQL, "WHERE")
}

func TestCompiler_NotIsNull(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "posts",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "content", Operator: OpIsNull, Not: true},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "IS NOT NULL")
}

func TestCompiler_NotIsNotNull(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "posts",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "content", Operator: OpIsNotNull, Not: true},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "IS NULL")
}

func TestCompiler_ILikeOperator(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "name", Operator: OpILike, Value: "john"},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "ILIKE")
	assert.Equal(t, "%john%", compiled.Params[0])
}

func TestCompiler_IncludesOperator(t *testing.T) {
	compiler := NewSQLCompiler(testEntities())

	q := &Query{
		Type:   QuerySelect,
		Entity: "users",
		Where: &WhereClause{
			Conditions: []Condition{
				{Field: "roles", Operator: OpIncludes, Value: "admin"},
			},
		},
	}

	compiled, err := compiler.Compile(q)
	require.NoError(t, err)
	assert.Contains(t, compiled.SQL, "@>")
}

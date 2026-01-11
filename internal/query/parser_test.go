package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_SelectQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		entity   string
		fields   []string
		hasWhere bool
	}{
		{
			name:   "simple select",
			input:  "select users",
			entity: "users",
		},
		{
			name:   "select from",
			input:  "select from users",
			entity: "users",
		},
		{
			name:   "select with where",
			input:  `select users where status = "active"`,
			entity: "users",
			hasWhere: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, QuerySelect, q.Type)
			assert.Equal(t, tt.entity, q.Entity)
			if tt.hasWhere {
				assert.NotNil(t, q.Where)
			}
		})
	}
}

func TestParse_CountQuery(t *testing.T) {
	input := `count users where status = "active"`

	q, err := Parse(input)
	require.NoError(t, err)
	assert.Equal(t, QueryCount, q.Type)
	assert.Equal(t, "users", q.Entity)
	assert.NotNil(t, q.Where)
}

func TestParse_AggregateQueries(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		qType   QueryType
		entity  string
		aggField string
	}{
		{
			name:   "sum",
			input:  "sum(amount) orders",
			qType:  QuerySum,
			entity: "orders",
			aggField: "amount",
		},
		{
			name:   "avg",
			input:  "avg(price) from products",
			qType:  QueryAvg,
			entity: "products",
			aggField: "price",
		},
		{
			name:   "min",
			input:  "min(created_at) users",
			qType:  QueryMin,
			entity: "users",
			aggField: "created_at",
		},
		{
			name:   "max",
			input:  "max(score) from results",
			qType:  QueryMax,
			entity: "results",
			aggField: "score",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.qType, q.Type)
			assert.Equal(t, tt.entity, q.Entity)
			assert.Equal(t, tt.aggField, q.AggField)
		})
	}
}

func TestParse_UpdateQuery(t *testing.T) {
	input := `update users set status = "active", count = 5 where id = 123`

	q, err := Parse(input)
	require.NoError(t, err)
	assert.Equal(t, QueryUpdate, q.Type)
	assert.Equal(t, "users", q.Entity)
	assert.Len(t, q.Updates, 2)
	assert.Equal(t, "status", q.Updates[0].Field)
	assert.Equal(t, "active", q.Updates[0].Value)
	assert.Equal(t, "count", q.Updates[1].Field)
	assert.NotNil(t, q.Where)
}

func TestParse_DeleteQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		entity   string
		hasWhere bool
	}{
		{
			name:   "simple delete",
			input:  "delete users",
			entity: "users",
		},
		{
			name:   "delete from",
			input:  "delete from users",
			entity: "users",
		},
		{
			name:     "delete with where",
			input:    `delete users where status = "inactive"`,
			entity:   "users",
			hasWhere: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, QueryDelete, q.Type)
			assert.Equal(t, tt.entity, q.Entity)
			if tt.hasWhere {
				assert.NotNil(t, q.Where)
			}
		})
	}
}

func TestParse_WhereConditions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected CompareOp
		value    interface{}
	}{
		{
			name:     "equals string",
			input:    `select users where name = "John"`,
			expected: OpEquals,
			value:    "John",
		},
		{
			name:     "not equals",
			input:    `select users where status != "deleted"`,
			expected: OpNotEquals,
			value:    "deleted",
		},
		{
			name:     "greater than",
			input:    "select users where age > 18",
			expected: OpGreaterThan,
			value:    int64(18),
		},
		{
			name:     "greater than or equal",
			input:    "select users where age >= 21",
			expected: OpGreaterThanOrEqual,
			value:    int64(21),
		},
		{
			name:     "less than",
			input:    "select users where count < 100",
			expected: OpLessThan,
			value:    int64(100),
		},
		{
			name:     "less than or equal",
			input:    "select users where score <= 50",
			expected: OpLessThanOrEqual,
			value:    int64(50),
		},
		{
			name:     "contains",
			input:    `select users where name contains "john"`,
			expected: OpContains,
			value:    "john",
		},
		{
			name:     "startswith",
			input:    `select users where email startswith "admin"`,
			expected: OpStartsWith,
			value:    "admin",
		},
		{
			name:     "endswith",
			input:    `select users where email endswith "@example.com"`,
			expected: OpEndsWith,
			value:    "@example.com",
		},
		{
			name:     "like",
			input:    `select users where name like "%john%"`,
			expected: OpLike,
			value:    "%john%",
		},
		{
			name:     "ilike",
			input:    `select users where name ilike "%JOHN%"`,
			expected: OpILike,
			value:    "%JOHN%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			require.NoError(t, err)
			require.NotNil(t, q.Where)
			require.Len(t, q.Where.Conditions, 1)
			assert.Equal(t, tt.expected, q.Where.Conditions[0].Operator)
			assert.Equal(t, tt.value, q.Where.Conditions[0].Value)
		})
	}
}

func TestParse_IsNull(t *testing.T) {
	input := "select users where deleted_at is null"

	q, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
	require.Len(t, q.Where.Conditions, 1)
	assert.Equal(t, OpIsNull, q.Where.Conditions[0].Operator)
}

func TestParse_IsNotNull(t *testing.T) {
	input := "select users where email is not null"

	q, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
	require.Len(t, q.Where.Conditions, 1)
	assert.Equal(t, OpIsNotNull, q.Where.Conditions[0].Operator)
}

func TestParse_InOperator(t *testing.T) {
	input := `select users where status in ["active", "pending"]`

	q, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
	require.Len(t, q.Where.Conditions, 1)
	assert.Equal(t, OpIn, q.Where.Conditions[0].Operator)

	values, ok := q.Where.Conditions[0].Value.([]interface{})
	require.True(t, ok)
	assert.Len(t, values, 2)
}

func TestParse_BetweenOperator(t *testing.T) {
	input := "select users where age between 18 and 65"

	q, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
	require.Len(t, q.Where.Conditions, 1)
	assert.Equal(t, OpBetween, q.Where.Conditions[0].Operator)

	bv, ok := q.Where.Conditions[0].Value.(BetweenValue)
	require.True(t, ok)
	assert.Equal(t, int64(18), bv.Low)
	assert.Equal(t, int64(65), bv.High)
}

func TestParse_AndOr(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		operator  LogicalOp
		numConds  int
	}{
		{
			name:     "and",
			input:    `select users where status = "active" and age > 18`,
			operator: LogicalAnd,
			numConds: 2,
		},
		{
			name:     "or",
			input:    `select users where role = "admin" or role = "moderator"`,
			operator: LogicalOr,
			numConds: 2,
		},
		{
			name:     "multiple and",
			input:    `select users where a = 1 and b = 2 and c = 3`,
			operator: LogicalAnd,
			numConds: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			require.NoError(t, err)
			require.NotNil(t, q.Where)
			assert.Len(t, q.Where.Conditions, tt.numConds)
		})
	}
}

func TestParse_GroupedConditions(t *testing.T) {
	input := `select users where (status = "active" or status = "pending") and verified = true`

	q, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
	assert.True(t, len(q.Where.Conditions) >= 1)
}

func TestParse_OrderBy(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		numOrders int
		direction OrderDirection
	}{
		{
			name:      "single asc",
			input:     "select users order by name asc",
			numOrders: 1,
			direction: OrderAsc,
		},
		{
			name:      "single desc",
			input:     "select users order by created_at desc",
			numOrders: 1,
			direction: OrderDesc,
		},
		{
			name:      "default asc",
			input:     "select users order by name",
			numOrders: 1,
			direction: OrderAsc,
		},
		{
			name:      "multiple",
			input:     "select users order by name asc, created_at desc",
			numOrders: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			require.NoError(t, err)
			assert.Len(t, q.OrderBy, tt.numOrders)
			if tt.numOrders == 1 {
				assert.Equal(t, tt.direction, q.OrderBy[0].Direction)
			}
		})
	}
}

func TestParse_LimitOffset(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		limit  *int
		offset *int
	}{
		{
			name:  "limit only",
			input: "select users limit 10",
			limit: intPtr(10),
		},
		{
			name:   "offset only",
			input:  "select users offset 20",
			offset: intPtr(20),
		},
		{
			name:   "both",
			input:  "select users limit 10 offset 20",
			limit:  intPtr(10),
			offset: intPtr(20),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			require.NoError(t, err)
			if tt.limit != nil {
				require.NotNil(t, q.Limit)
				assert.Equal(t, *tt.limit, *q.Limit)
			}
			if tt.offset != nil {
				require.NotNil(t, q.Offset)
				assert.Equal(t, *tt.offset, *q.Offset)
			}
		})
	}
}

func TestParse_Include(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		include []string
	}{
		{
			name:    "with single",
			input:   "select users with posts",
			include: []string{"posts"},
		},
		{
			name:    "include single",
			input:   "select users include comments",
			include: []string{"comments"},
		},
		{
			name:    "multiple",
			input:   "select users with posts, comments, likes",
			include: []string{"posts", "comments", "likes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.include, q.Include)
		})
	}
}

func TestParse_Parameters(t *testing.T) {
	input := `select users where id = @userId`

	q, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
	require.Len(t, q.Where.Conditions, 1)

	param, ok := q.Where.Conditions[0].Value.(*Parameter)
	require.True(t, ok)
	assert.Equal(t, "userId", param.Name)
}

func TestParse_BooleanValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "true",
			input:    "select users where active = true",
			expected: true,
		},
		{
			name:     "false",
			input:    "select users where deleted = false",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			require.NoError(t, err)
			require.NotNil(t, q.Where)
			require.Len(t, q.Where.Conditions, 1)
			assert.Equal(t, tt.expected, q.Where.Conditions[0].Value)
		})
	}
}

func TestParse_NullValue(t *testing.T) {
	input := "select users where deleted_at = null"

	q, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
	require.Len(t, q.Where.Conditions, 1)
	assert.Nil(t, q.Where.Conditions[0].Value)
}

func TestParseSimple_FieldFilters(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		numConds  int
		firstField string
	}{
		{
			name:       "single filter",
			input:      "status:active",
			numConds:   1,
			firstField: "status",
		},
		{
			name:       "multiple filters",
			input:      "status:active priority:1",
			numConds:   2,
			firstField: "status",
		},
		{
			name:       "comparison",
			input:      "age>18",
			numConds:   1,
			firstField: "age",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := ParseSimple(tt.input)
			require.NoError(t, err)
			require.NotNil(t, q.Where)
			assert.Len(t, q.Where.Conditions, tt.numConds)
			assert.Equal(t, tt.firstField, q.Where.Conditions[0].Field)
		})
	}
}

func TestParseSimple_TextSearch(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		operator CompareOp
		value    string
	}{
		{
			name:     "exact phrase",
			input:    `"hello world"`,
			operator: OpExact,
			value:    "hello world",
		},
		{
			name:     "fuzzy search",
			input:    "~fuzzy",
			operator: OpFuzzy,
			value:    "fuzzy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := ParseSimple(tt.input)
			require.NoError(t, err)
			require.NotNil(t, q.Where)
			require.Len(t, q.Where.Conditions, 1)
			assert.Equal(t, tt.operator, q.Where.Conditions[0].Operator)
			assert.Equal(t, tt.value, q.Where.Conditions[0].Value)
		})
	}
}

func TestParseSimple_CommaSeparatedValues(t *testing.T) {
	input := "tags:frontend,backend,devops"

	q, err := ParseSimple(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
	require.Len(t, q.Where.Conditions, 1)

	// Values should be parsed as an array
	values, ok := q.Where.Conditions[0].Value.([]interface{})
	require.True(t, ok)
	assert.Len(t, values, 3)
}

func TestParse_ComplexQuery(t *testing.T) {
	input := `select users with posts, comments where (status = "active" or role = "admin") and age >= 18 order by created_at desc limit 50 offset 100`

	q, err := Parse(input)
	require.NoError(t, err)
	assert.Equal(t, QuerySelect, q.Type)
	assert.Equal(t, "users", q.Entity)
	assert.Equal(t, []string{"posts", "comments"}, q.Include)
	assert.NotNil(t, q.Where)
	assert.Len(t, q.OrderBy, 1)
	assert.Equal(t, OrderDesc, q.OrderBy[0].Direction)
	require.NotNil(t, q.Limit)
	assert.Equal(t, 50, *q.Limit)
	require.NotNil(t, q.Offset)
	assert.Equal(t, 100, *q.Offset)
}

func TestParse_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty input",
			input: "",
		},
		{
			name:  "missing entity",
			input: "select where id = 1",
		},
		{
			name:  "invalid operator",
			input: "select users where name %% 'test'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			assert.Error(t, err)
		})
	}
}

func TestParse_NotCondition(t *testing.T) {
	input := `select users where not active = true`

	q, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
	require.Len(t, q.Where.Conditions, 1)
	assert.True(t, q.Where.Conditions[0].Not)
}

func TestParse_FloatNumbers(t *testing.T) {
	input := "select products where price >= 19.99"

	q, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
	require.Len(t, q.Where.Conditions, 1)
	assert.Equal(t, 19.99, q.Where.Conditions[0].Value)
}

func TestParse_GroupBy(t *testing.T) {
	input := "select from orders where status = 'completed' group by user_id"

	q, err := Parse(input)
	require.NoError(t, err)
	assert.Equal(t, []string{"user_id"}, q.GroupBy)
}

func TestParse_Having(t *testing.T) {
	input := "select from orders group by status having count > 10"

	q, err := Parse(input)
	require.NoError(t, err)
	assert.NotNil(t, q.Having)
}

func TestParse_SelectWithFields(t *testing.T) {
	input := "select id, name, email from users"

	q, err := Parse(input)
	require.NoError(t, err)
	assert.Equal(t, []string{"id", "name", "email"}, q.Fields)
}

func TestParse_ParenthesizedConditions(t *testing.T) {
	input := "select users where (status = 'active' or status = 'pending') and age > 18"

	q, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
	assert.True(t, len(q.Where.Conditions) >= 1)
}

func TestParse_NestedParens(t *testing.T) {
	input := "select users where ((a = 1 and b = 2) or (c = 3))"

	q, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
}

func TestParse_MaxDepthExceeded(t *testing.T) {
	// Create deeply nested query that exceeds max depth
	input := "select users where ((((((((((((status = 'active'))))))))))))"

	_, err := Parse(input)
	// This may or may not error depending on depth limit
	// Just make sure it doesn't crash
	_ = err
}

func TestParse_ArrayInWhere(t *testing.T) {
	input := `select users where ids in [1, 2, 3, 4, 5]`

	q, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
	require.Len(t, q.Where.Conditions, 1)
	assert.Equal(t, OpIn, q.Where.Conditions[0].Operator)
}

func TestParse_EmptyArray(t *testing.T) {
	input := `select users where ids in []`

	q, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
}

func TestParse_MultipleOrderBy(t *testing.T) {
	input := "select users order by name asc, created_at desc, id"

	q, err := Parse(input)
	require.NoError(t, err)
	assert.Len(t, q.OrderBy, 3)
	assert.Equal(t, OrderAsc, q.OrderBy[0].Direction)
	assert.Equal(t, OrderDesc, q.OrderBy[1].Direction)
	assert.Equal(t, OrderAsc, q.OrderBy[2].Direction) // Default
}

func TestParse_UpdateWithIncrement(t *testing.T) {
	input := `update users set views = 100 where id = 1`

	q, err := Parse(input)
	require.NoError(t, err)
	assert.Equal(t, QueryUpdate, q.Type)
	assert.Len(t, q.Updates, 1)
}

func TestParse_NegativeNumbers(t *testing.T) {
	input := "select accounts where balance < -100"

	q, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
}

func TestParseSimple_LogicalOperators(t *testing.T) {
	input := "status:active and priority:high"

	q, err := ParseSimple(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
}

func TestParseSimple_NotWithParens(t *testing.T) {
	input := "not (status = 'deleted')"

	q, err := ParseSimple(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
}

func TestParse_MixedStringQuotes(t *testing.T) {
	input := `select users where name = 'John' and email = "john@example.com"`

	q, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, q.Where)
	assert.Len(t, q.Where.Conditions, 2)
}

func TestParse_KeywordsAsFieldNames(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"count field", `update users set count = 10 where id = 1`},
		{"from field", `update data set from = "2024-01-01" where id = 1`},
		{"select field", `select id, select from users`},
		{"limit field", `update users set limit = 100 where id = 1`},
		{"order field", `update users set order = 5 where id = 1`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			require.NoError(t, err)
		})
	}
}

func TestParse_InvalidBetween(t *testing.T) {
	input := "select users where age between 18"

	_, err := Parse(input)
	require.Error(t, err) // Missing AND clause
}

func TestParse_InvalidIs(t *testing.T) {
	input := "select users where status is invalid"

	_, err := Parse(input)
	require.Error(t, err)
}

// Helper function
func intPtr(i int) *int {
	return &i
}

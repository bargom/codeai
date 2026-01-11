package pagination

import (
	"testing"

	"github.com/bargom/codeai/internal/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFilterBuilder(t *testing.T) {
	fb := NewFilterBuilder()
	assert.NotNil(t, fb)
	assert.NotNil(t, fb.definitions)
}

func TestFilterBuilder_Allow(t *testing.T) {
	fb := NewFilterBuilder()

	// Allow with default operators
	fb.Allow("name")
	assert.Contains(t, fb.definitions, "name")
	assert.Equal(t, SupportedOperators, fb.definitions["name"].AllowedOps)

	// Allow with specific operators
	fb.Allow("status", "eq", "ne")
	assert.Contains(t, fb.definitions, "status")
	assert.Equal(t, []string{"eq", "ne"}, fb.definitions["status"].AllowedOps)
}

func TestFilterBuilder_AllowAll(t *testing.T) {
	fb := NewFilterBuilder()

	fb.AllowAll("name", "email", "status")

	assert.Contains(t, fb.definitions, "name")
	assert.Contains(t, fb.definitions, "email")
	assert.Contains(t, fb.definitions, "status")
}

func TestFilterBuilder_Build_EmptyParams(t *testing.T) {
	fb := NewFilterBuilder().Allow("name")

	where, err := fb.Build(map[string]any{})
	require.NoError(t, err)
	assert.Nil(t, where)
}

func TestFilterBuilder_Build_NilParams(t *testing.T) {
	fb := NewFilterBuilder().Allow("name")

	where, err := fb.Build(nil)
	require.NoError(t, err)
	assert.Nil(t, where)
}

func TestFilterBuilder_Build_SimpleEquals(t *testing.T) {
	fb := NewFilterBuilder().Allow("name")

	params := map[string]any{"name": "John"}
	where, err := fb.Build(params)

	require.NoError(t, err)
	require.NotNil(t, where)
	assert.Len(t, where.Conditions, 1)
	assert.Equal(t, "name", where.Conditions[0].Field)
	assert.Equal(t, query.OpEquals, where.Conditions[0].Operator)
	assert.Equal(t, "John", where.Conditions[0].Value)
}

func TestFilterBuilder_Build_WithOperator(t *testing.T) {
	fb := NewFilterBuilder().Allow("age")

	params := map[string]any{"age[gte]": 18}
	where, err := fb.Build(params)

	require.NoError(t, err)
	require.NotNil(t, where)
	assert.Len(t, where.Conditions, 1)
	assert.Equal(t, "age", where.Conditions[0].Field)
	assert.Equal(t, query.OpGreaterThanOrEqual, where.Conditions[0].Operator)
}

func TestFilterBuilder_Build_MultipleConditions(t *testing.T) {
	fb := NewFilterBuilder().AllowAll("name", "status", "age")

	params := map[string]any{
		"name":       "John",
		"status":     "active",
		"age[gte]":   18,
		"age[lte]":   65,
	}
	where, err := fb.Build(params)

	require.NoError(t, err)
	require.NotNil(t, where)
	assert.Len(t, where.Conditions, 4)
	assert.Equal(t, query.LogicalAnd, where.Operator)
}

func TestFilterBuilder_Build_UnknownField(t *testing.T) {
	fb := NewFilterBuilder().Allow("name")

	// Unknown field should be ignored
	params := map[string]any{"unknown": "value", "name": "John"}
	where, err := fb.Build(params)

	require.NoError(t, err)
	require.NotNil(t, where)
	assert.Len(t, where.Conditions, 1)
	assert.Equal(t, "name", where.Conditions[0].Field)
}

func TestFilterBuilder_Build_DisallowedOperator(t *testing.T) {
	fb := NewFilterBuilder().Allow("status", "eq") // Only allow eq

	params := map[string]any{"status[gte]": "active"} // Try to use gte
	_, err := fb.Build(params)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operator 'gte' not allowed")
}

func TestFilterBuilder_Build_AllOperators(t *testing.T) {
	tests := []struct {
		operator string
		expected query.CompareOp
	}{
		{"eq", query.OpEquals},
		{"ne", query.OpNotEquals},
		{"gt", query.OpGreaterThan},
		{"gte", query.OpGreaterThanOrEqual},
		{"lt", query.OpLessThan},
		{"lte", query.OpLessThanOrEqual},
		{"contains", query.OpContains},
		{"startswith", query.OpStartsWith},
		{"endswith", query.OpEndsWith},
	}

	for _, tt := range tests {
		t.Run(tt.operator, func(t *testing.T) {
			fb := NewFilterBuilder().Allow("field")

			params := map[string]any{"field[" + tt.operator + "]": "value"}
			where, err := fb.Build(params)

			require.NoError(t, err)
			require.NotNil(t, where)
			assert.Equal(t, tt.expected, where.Conditions[0].Operator)
		})
	}
}

func TestFilterBuilder_Build_InOperator(t *testing.T) {
	fb := NewFilterBuilder().Allow("status")

	// String value with comma-separated values
	params := map[string]any{"status[in]": "active,pending,completed"}
	where, err := fb.Build(params)

	require.NoError(t, err)
	require.NotNil(t, where)
	assert.Equal(t, query.OpIn, where.Conditions[0].Operator)
	values := where.Conditions[0].Value.([]interface{})
	assert.Len(t, values, 3)
	assert.Equal(t, "active", values[0])
	assert.Equal(t, "pending", values[1])
	assert.Equal(t, "completed", values[2])
}

func TestFilterBuilder_Build_IsNullOperator(t *testing.T) {
	fb := NewFilterBuilder().Allow("deleted_at")

	tests := []struct {
		name        string
		value       any
		expectedOp  query.CompareOp
	}{
		{"true string", "true", query.OpIsNull},
		{"1 string", "1", query.OpIsNull},
		{"false string", "false", query.OpIsNotNull},
		{"0 string", "0", query.OpIsNotNull},
		{"true bool", true, query.OpIsNull},
		{"false bool", false, query.OpIsNotNull},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]any{"deleted_at[isnull]": tt.value}
			where, err := fb.Build(params)

			require.NoError(t, err)
			require.NotNil(t, where)
			assert.Equal(t, tt.expectedOp, where.Conditions[0].Operator)
		})
	}
}

func TestFilterBuilder_Build_IsNull_DefaultValue(t *testing.T) {
	fb := NewFilterBuilder().Allow("deleted_at")

	// Non-string, non-bool value should default to IsNull
	params := map[string]any{"deleted_at[isnull]": 123}
	where, err := fb.Build(params)

	require.NoError(t, err)
	require.NotNil(t, where)
	assert.Equal(t, query.OpIsNull, where.Conditions[0].Operator)
}

func TestFilterBuilder_Build_UnknownOperator(t *testing.T) {
	fb := NewFilterBuilder().Allow("field", "unknown")

	params := map[string]any{"field[unknown]": "value"}
	_, err := fb.Build(params)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown operator")
}

func TestParseFilterKey(t *testing.T) {
	tests := []struct {
		input    string
		field    string
		operator string
	}{
		{"name", "name", "eq"},
		{"name[eq]", "name", "eq"},
		{"age[gte]", "age", "gte"},
		{"status[contains]", "status", "contains"},
		{"field[op]extra", "field", "op"}, // Edge case
		{"field[", "field", ""},            // Missing closing bracket
		{"field[op", "field", "op"},        // Missing closing bracket (gets trimmed)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			field, op := ParseFilterKey(tt.input)
			assert.Equal(t, tt.field, field)
			assert.Equal(t, tt.operator, op)
		})
	}
}

func TestBuildCondition(t *testing.T) {
	tests := []struct {
		field    string
		op       string
		value    any
		expected query.CompareOp
		hasError bool
	}{
		{"name", "eq", "John", query.OpEquals, false},
		{"name", "ne", "John", query.OpNotEquals, false},
		{"age", "gt", 18, query.OpGreaterThan, false},
		{"age", "gte", 18, query.OpGreaterThanOrEqual, false},
		{"age", "lt", 65, query.OpLessThan, false},
		{"age", "lte", 65, query.OpLessThanOrEqual, false},
		{"name", "contains", "oh", query.OpContains, false},
		{"name", "startswith", "J", query.OpStartsWith, false},
		{"name", "endswith", "n", query.OpEndsWith, false},
		{"field", "invalid", "value", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			cond, err := BuildCondition(tt.field, tt.op, tt.value)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.field, cond.Field)
				assert.Equal(t, tt.expected, cond.Operator)
			}
		})
	}
}

func TestBuildCondition_CaseInsensitive(t *testing.T) {
	cond, err := BuildCondition("name", "EQ", "John")
	require.NoError(t, err)
	assert.Equal(t, query.OpEquals, cond.Operator)

	cond, err = BuildCondition("name", "Contains", "test")
	require.NoError(t, err)
	assert.Equal(t, query.OpContains, cond.Operator)
}

func TestContains(t *testing.T) {
	slice := []string{"eq", "ne", "gt"}

	assert.True(t, contains(slice, "eq"))
	assert.True(t, contains(slice, "EQ")) // Case insensitive
	assert.True(t, contains(slice, "Ne"))
	assert.False(t, contains(slice, "unknown"))
	assert.False(t, contains(nil, "eq"))
	assert.False(t, contains([]string{}, "eq"))
}

func TestMergeWhereClauses(t *testing.T) {
	clause1 := &query.WhereClause{
		Conditions: []query.Condition{
			{Field: "name", Operator: query.OpEquals, Value: "John"},
		},
	}
	clause2 := &query.WhereClause{
		Conditions: []query.Condition{
			{Field: "status", Operator: query.OpEquals, Value: "active"},
		},
	}

	merged := MergeWhereClauses(clause1, clause2)

	require.NotNil(t, merged)
	assert.Equal(t, query.LogicalAnd, merged.Operator)
	assert.Len(t, merged.Conditions, 2)
}

func TestMergeWhereClauses_WithNil(t *testing.T) {
	clause := &query.WhereClause{
		Conditions: []query.Condition{
			{Field: "name", Operator: query.OpEquals, Value: "John"},
		},
	}

	merged := MergeWhereClauses(nil, clause, nil)

	require.NotNil(t, merged)
	assert.Len(t, merged.Conditions, 1)
}

func TestMergeWhereClauses_AllNil(t *testing.T) {
	merged := MergeWhereClauses(nil, nil, nil)
	assert.Nil(t, merged)
}

func TestMergeWhereClauses_Empty(t *testing.T) {
	merged := MergeWhereClauses()
	assert.Nil(t, merged)
}

func TestSupportedOperators(t *testing.T) {
	expected := []string{"eq", "ne", "gt", "gte", "lt", "lte", "contains", "in", "startswith", "endswith", "isnull"}
	assert.Equal(t, expected, SupportedOperators)
}

func TestFilterBuilder_Chaining(t *testing.T) {
	fb := NewFilterBuilder().
		Allow("name", "eq", "contains").
		Allow("age", "gt", "lt", "gte", "lte").
		AllowAll("status", "email")

	assert.Contains(t, fb.definitions, "name")
	assert.Contains(t, fb.definitions, "age")
	assert.Contains(t, fb.definitions, "status")
	assert.Contains(t, fb.definitions, "email")
}

func TestFilterBuilder_Build_OnlyAllowedFields(t *testing.T) {
	fb := NewFilterBuilder().Allow("name")

	params := map[string]any{
		"name":     "John",
		"password": "secret", // Should be ignored
		"admin":    true,     // Should be ignored
	}

	where, err := fb.Build(params)

	require.NoError(t, err)
	require.NotNil(t, where)
	assert.Len(t, where.Conditions, 1)
	assert.Equal(t, "name", where.Conditions[0].Field)
}

func TestFilterBuilder_Build_InOperator_Whitespace(t *testing.T) {
	fb := NewFilterBuilder().Allow("status")

	// Values with whitespace
	params := map[string]any{"status[in]": "active , pending , completed"}
	where, err := fb.Build(params)

	require.NoError(t, err)
	require.NotNil(t, where)
	values := where.Conditions[0].Value.([]interface{})
	assert.Equal(t, "active", values[0])
	assert.Equal(t, "pending", values[1])
	assert.Equal(t, "completed", values[2])
}

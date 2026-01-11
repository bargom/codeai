package pagination

import (
	"fmt"
	"strings"

	"github.com/bargom/codeai/internal/query"
)

// FilterDef defines allowed filter operations for a field.
type FilterDef struct {
	Field      string
	AllowedOps []string
}

// FilterBuilder builds query WHERE clauses from filter parameters.
type FilterBuilder struct {
	definitions map[string]FilterDef
}

// SupportedOperators lists all supported filter operators.
var SupportedOperators = []string{"eq", "ne", "gt", "gte", "lt", "lte", "contains", "in", "startswith", "endswith", "isnull"}

// NewFilterBuilder creates a new FilterBuilder.
func NewFilterBuilder() *FilterBuilder {
	return &FilterBuilder{
		definitions: make(map[string]FilterDef),
	}
}

// Allow registers a field with allowed operators.
// If no operators are specified, all operators are allowed.
func (b *FilterBuilder) Allow(field string, ops ...string) *FilterBuilder {
	if len(ops) == 0 {
		ops = SupportedOperators
	}
	b.definitions[field] = FilterDef{
		Field:      field,
		AllowedOps: ops,
	}
	return b
}

// AllowAll registers multiple fields with all operators allowed.
func (b *FilterBuilder) AllowAll(fields ...string) *FilterBuilder {
	for _, field := range fields {
		b.Allow(field)
	}
	return b
}

// Build parses filter parameters and builds a WHERE clause.
// Parameters should be in the format: field=value or field[op]=value
func (b *FilterBuilder) Build(params map[string]any) (*query.WhereClause, error) {
	if len(params) == 0 {
		return nil, nil
	}

	where := &query.WhereClause{Operator: query.LogicalAnd}

	for key, value := range params {
		// Parse field[op] syntax
		field, op := ParseFilterKey(key)

		def, ok := b.definitions[field]
		if !ok {
			continue // Ignore unknown fields
		}

		if !contains(def.AllowedOps, op) {
			return nil, fmt.Errorf("operator '%s' not allowed for field '%s'", op, field)
		}

		cond, err := BuildCondition(field, op, value)
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

// ParseFilterKey parses a filter key in the format field[op] into field and operator.
// If no operator is specified, defaults to "eq".
func ParseFilterKey(key string) (field, op string) {
	if idx := strings.Index(key, "["); idx != -1 {
		endIdx := strings.Index(key, "]")
		if endIdx > idx {
			field = key[:idx]
			op = key[idx+1 : endIdx]
		} else {
			field = key[:idx]
			op = strings.TrimSuffix(key[idx+1:], "]")
		}
	} else {
		field = key
		op = "eq"
	}
	return
}

// BuildCondition creates a query Condition from field, operator, and value.
func BuildCondition(field, op string, value any) (query.Condition, error) {
	cond := query.Condition{Field: field, Value: value}

	switch strings.ToLower(op) {
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
		// Ensure value is a slice for IN operator
		if strVal, ok := value.(string); ok {
			// Parse comma-separated values
			parts := strings.Split(strVal, ",")
			vals := make([]interface{}, len(parts))
			for i, p := range parts {
				vals[i] = strings.TrimSpace(p)
			}
			cond.Value = vals
		}
	case "startswith":
		cond.Operator = query.OpStartsWith
	case "endswith":
		cond.Operator = query.OpEndsWith
	case "isnull":
		// Handle boolean value for isnull
		if boolVal, ok := value.(bool); ok {
			if boolVal {
				cond.Operator = query.OpIsNull
			} else {
				cond.Operator = query.OpIsNotNull
			}
		} else if strVal, ok := value.(string); ok {
			if strVal == "true" || strVal == "1" {
				cond.Operator = query.OpIsNull
			} else {
				cond.Operator = query.OpIsNotNull
			}
		} else {
			cond.Operator = query.OpIsNull
		}
		cond.Value = nil
	default:
		return cond, fmt.Errorf("unknown operator: %s", op)
	}

	return cond, nil
}

// contains checks if a string slice contains a value.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

// MergeWhereClauses combines multiple WHERE clauses with AND logic.
func MergeWhereClauses(clauses ...*query.WhereClause) *query.WhereClause {
	var conditions []query.Condition

	for _, clause := range clauses {
		if clause != nil {
			conditions = append(conditions, clause.Conditions...)
		}
	}

	if len(conditions) == 0 {
		return nil
	}

	return &query.WhereClause{
		Operator:   query.LogicalAnd,
		Conditions: conditions,
	}
}

package query

import (
	"fmt"
	"strings"
)

// SQLCompiler compiles Query AST to parameterized SQL.
type SQLCompiler struct {
	entities map[string]*EntityMeta
	params   []interface{}
	paramIdx int
}

// CompiledQuery represents a compiled SQL query with parameters.
type CompiledQuery struct {
	SQL    string
	Params []interface{}
}

// NewSQLCompiler creates a new SQL compiler with entity metadata.
func NewSQLCompiler(entities map[string]*EntityMeta) *SQLCompiler {
	return &SQLCompiler{
		entities: entities,
		params:   make([]interface{}, 0),
	}
}

// Compile compiles a Query to SQL.
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
	case QuerySum, QueryAvg, QueryMin, QueryMax:
		sql, err = c.compileAggregateFunc(q)
	case QueryUpdate:
		sql, err = c.compileUpdate(q)
	case QueryDelete:
		sql, err = c.compileDelete(q)
	default:
		err = ErrInvalidQueryType(q.Type)
	}

	if err != nil {
		return nil, err
	}

	return &CompiledQuery{SQL: sql, Params: c.params}, nil
}

// compileSelect compiles a SELECT query.
func (c *SQLCompiler) compileSelect(q *Query) (string, error) {
	entity := c.getEntity(q.Entity)
	if entity == nil {
		return "", ErrUnknownEntity(q.Entity)
	}

	var b strings.Builder

	// SELECT
	b.WriteString("SELECT ")
	if len(q.Fields) == 0 {
		b.WriteString("*")
	} else {
		fields := make([]string, len(q.Fields))
		for i, f := range q.Fields {
			fields[i] = c.quoteIdent(c.mapColumn(entity, f))
		}
		b.WriteString(strings.Join(fields, ", "))
	}

	// FROM
	b.WriteString(" FROM ")
	b.WriteString(c.quoteIdent(entity.TableName))

	// WHERE
	whereSQL, err := c.compileWhereWithSoftDelete(q.Where, entity)
	if err != nil {
		return "", err
	}
	if whereSQL != "" {
		b.WriteString(" WHERE ")
		b.WriteString(whereSQL)
	}

	// GROUP BY
	if len(q.GroupBy) > 0 {
		b.WriteString(" GROUP BY ")
		groups := make([]string, len(q.GroupBy))
		for i, g := range q.GroupBy {
			groups[i] = c.quoteIdent(c.mapColumn(entity, g))
		}
		b.WriteString(strings.Join(groups, ", "))
	}

	// HAVING
	if q.Having != nil {
		havingSQL, err := c.compileWhere(q.Having, entity)
		if err != nil {
			return "", err
		}
		b.WriteString(" HAVING ")
		b.WriteString(havingSQL)
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
			orders = append(orders, fmt.Sprintf("%s %s", c.quoteIdent(c.mapColumn(entity, o.Field)), dir))
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

// compileCount compiles a COUNT query.
func (c *SQLCompiler) compileCount(q *Query) (string, error) {
	entity := c.getEntity(q.Entity)
	if entity == nil {
		return "", ErrUnknownEntity(q.Entity)
	}

	var b strings.Builder
	b.WriteString("SELECT COUNT(*) FROM ")
	b.WriteString(c.quoteIdent(entity.TableName))

	whereSQL, err := c.compileWhereWithSoftDelete(q.Where, entity)
	if err != nil {
		return "", err
	}
	if whereSQL != "" {
		b.WriteString(" WHERE ")
		b.WriteString(whereSQL)
	}

	return b.String(), nil
}

// compileAggregateFunc compiles SUM, AVG, MIN, MAX queries.
func (c *SQLCompiler) compileAggregateFunc(q *Query) (string, error) {
	entity := c.getEntity(q.Entity)
	if entity == nil {
		return "", ErrUnknownEntity(q.Entity)
	}

	var funcName string
	switch q.Type {
	case QuerySum:
		funcName = "SUM"
	case QueryAvg:
		funcName = "AVG"
	case QueryMin:
		funcName = "MIN"
	case QueryMax:
		funcName = "MAX"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("SELECT %s(%s) FROM ", funcName, c.quoteIdent(c.mapColumn(entity, q.AggField))))
	b.WriteString(c.quoteIdent(entity.TableName))

	whereSQL, err := c.compileWhereWithSoftDelete(q.Where, entity)
	if err != nil {
		return "", err
	}
	if whereSQL != "" {
		b.WriteString(" WHERE ")
		b.WriteString(whereSQL)
	}

	return b.String(), nil
}

// compileUpdate compiles an UPDATE query.
func (c *SQLCompiler) compileUpdate(q *Query) (string, error) {
	entity := c.getEntity(q.Entity)
	if entity == nil {
		return "", ErrUnknownEntity(q.Entity)
	}

	var b strings.Builder
	b.WriteString("UPDATE ")
	b.WriteString(c.quoteIdent(entity.TableName))
	b.WriteString(" SET ")

	sets := make([]string, 0, len(q.Updates))
	for _, u := range q.Updates {
		col := c.quoteIdent(c.mapColumn(entity, u.Field))
		switch u.Op {
		case UpdateSetValue:
			c.paramIdx++
			c.params = append(c.params, u.Value)
			sets = append(sets, fmt.Sprintf("%s = $%d", col, c.paramIdx))
		case UpdateIncrement:
			c.paramIdx++
			c.params = append(c.params, u.Value)
			sets = append(sets, fmt.Sprintf("%s = %s + $%d", col, col, c.paramIdx))
		case UpdateDecrement:
			c.paramIdx++
			c.params = append(c.params, u.Value)
			sets = append(sets, fmt.Sprintf("%s = %s - $%d", col, col, c.paramIdx))
		case UpdateAppend:
			c.paramIdx++
			c.params = append(c.params, u.Value)
			sets = append(sets, fmt.Sprintf("%s = array_append(%s, $%d)", col, col, c.paramIdx))
		case UpdateRemove:
			c.paramIdx++
			c.params = append(c.params, u.Value)
			sets = append(sets, fmt.Sprintf("%s = array_remove(%s, $%d)", col, col, c.paramIdx))
		}
	}
	b.WriteString(strings.Join(sets, ", "))

	whereSQL, err := c.compileWhereWithSoftDelete(q.Where, entity)
	if err != nil {
		return "", err
	}
	if whereSQL != "" {
		b.WriteString(" WHERE ")
		b.WriteString(whereSQL)
	}

	return b.String(), nil
}

// compileDelete compiles a DELETE query.
func (c *SQLCompiler) compileDelete(q *Query) (string, error) {
	entity := c.getEntity(q.Entity)
	if entity == nil {
		return "", ErrUnknownEntity(q.Entity)
	}

	var b strings.Builder

	// Use soft delete if configured
	if entity.SoftDelete != "" {
		b.WriteString("UPDATE ")
		b.WriteString(c.quoteIdent(entity.TableName))
		b.WriteString(" SET ")
		b.WriteString(c.quoteIdent(entity.SoftDelete))
		b.WriteString(" = NOW()")
	} else {
		b.WriteString("DELETE FROM ")
		b.WriteString(c.quoteIdent(entity.TableName))
	}

	whereSQL, err := c.compileWhereWithSoftDelete(q.Where, entity)
	if err != nil {
		return "", err
	}
	if whereSQL != "" {
		b.WriteString(" WHERE ")
		b.WriteString(whereSQL)
	}

	return b.String(), nil
}

// compileWhereWithSoftDelete compiles WHERE clause and adds soft delete condition.
func (c *SQLCompiler) compileWhereWithSoftDelete(where *WhereClause, entity *EntityMeta) (string, error) {
	var conditions []string

	if where != nil {
		whereSQL, err := c.compileWhere(where, entity)
		if err != nil {
			return "", err
		}
		if whereSQL != "" {
			conditions = append(conditions, whereSQL)
		}
	}

	// Add soft delete condition
	if entity.SoftDelete != "" {
		conditions = append(conditions, fmt.Sprintf("%s IS NULL", c.quoteIdent(entity.SoftDelete)))
	}

	if len(conditions) == 0 {
		return "", nil
	}
	if len(conditions) == 1 {
		return conditions[0], nil
	}
	return "(" + strings.Join(conditions, ") AND (") + ")", nil
}

// compileWhere compiles a WHERE clause.
func (c *SQLCompiler) compileWhere(where *WhereClause, entity *EntityMeta) (string, error) {
	if where == nil || len(where.Conditions) == 0 {
		return "", nil
	}

	var conditions []string
	op := " AND "
	if where.Operator == LogicalOr {
		op = " OR "
	}

	for _, cond := range where.Conditions {
		sql, err := c.compileCondition(cond, entity)
		if err != nil {
			return "", err
		}
		if sql != "" {
			conditions = append(conditions, sql)
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}
	if len(conditions) == 1 {
		return conditions[0], nil
	}

	return "(" + strings.Join(conditions, op) + ")", nil
}

// compileCondition compiles a single condition.
func (c *SQLCompiler) compileCondition(cond Condition, entity *EntityMeta) (string, error) {
	// Handle nested groups
	if cond.Nested != nil {
		sql, err := c.compileWhere(cond.Nested, entity)
		if err != nil {
			return "", err
		}
		if cond.Not {
			return "NOT (" + sql + ")", nil
		}
		return sql, nil
	}

	// Handle text search (no field specified)
	if cond.Field == "" {
		return c.compileTextSearch(cond, entity)
	}

	col := c.quoteIdent(c.mapColumn(entity, cond.Field))
	isJSON := entity.JSONColumns != nil && entity.JSONColumns[cond.Field]
	notPrefix := ""
	if cond.Not {
		notPrefix = "NOT "
	}

	switch cond.Operator {
	case OpEquals:
		c.paramIdx++
		c.params = append(c.params, cond.Value)
		if isJSON {
			return fmt.Sprintf("%s%s @> $%d::jsonb", notPrefix, col, c.paramIdx), nil
		}
		return fmt.Sprintf("%s%s = $%d", notPrefix, col, c.paramIdx), nil

	case OpNotEquals:
		c.paramIdx++
		c.params = append(c.params, cond.Value)
		return fmt.Sprintf("%s%s != $%d", notPrefix, col, c.paramIdx), nil

	case OpGreaterThan:
		c.paramIdx++
		c.params = append(c.params, cond.Value)
		return fmt.Sprintf("%s%s > $%d", notPrefix, col, c.paramIdx), nil

	case OpGreaterThanOrEqual:
		c.paramIdx++
		c.params = append(c.params, cond.Value)
		return fmt.Sprintf("%s%s >= $%d", notPrefix, col, c.paramIdx), nil

	case OpLessThan:
		c.paramIdx++
		c.params = append(c.params, cond.Value)
		return fmt.Sprintf("%s%s < $%d", notPrefix, col, c.paramIdx), nil

	case OpLessThanOrEqual:
		c.paramIdx++
		c.params = append(c.params, cond.Value)
		return fmt.Sprintf("%s%s <= $%d", notPrefix, col, c.paramIdx), nil

	case OpContains, OpILike:
		c.paramIdx++
		c.params = append(c.params, "%"+fmt.Sprint(cond.Value)+"%")
		return fmt.Sprintf("%s%s ILIKE $%d", notPrefix, col, c.paramIdx), nil

	case OpLike:
		c.paramIdx++
		c.params = append(c.params, cond.Value)
		return fmt.Sprintf("%s%s LIKE $%d", notPrefix, col, c.paramIdx), nil

	case OpStartsWith:
		c.paramIdx++
		c.params = append(c.params, fmt.Sprint(cond.Value)+"%")
		return fmt.Sprintf("%s%s ILIKE $%d", notPrefix, col, c.paramIdx), nil

	case OpEndsWith:
		c.paramIdx++
		c.params = append(c.params, "%"+fmt.Sprint(cond.Value))
		return fmt.Sprintf("%s%s ILIKE $%d", notPrefix, col, c.paramIdx), nil

	case OpIn:
		values, ok := cond.Value.([]interface{})
		if !ok {
			return "", NewCompilerError(fmt.Sprintf("IN operator requires array value, got %T", cond.Value))
		}
		if len(values) == 0 {
			if cond.Not {
				return "TRUE", nil // NOT IN () is always true
			}
			return "FALSE", nil // IN () is always false
		}
		placeholders := make([]string, len(values))
		for i, v := range values {
			c.paramIdx++
			placeholders[i] = fmt.Sprintf("$%d", c.paramIdx)
			c.params = append(c.params, v)
		}
		return fmt.Sprintf("%s%s IN (%s)", notPrefix, col, strings.Join(placeholders, ", ")), nil

	case OpNotIn:
		values, ok := cond.Value.([]interface{})
		if !ok {
			return "", NewCompilerError(fmt.Sprintf("NOT IN operator requires array value, got %T", cond.Value))
		}
		if len(values) == 0 {
			return "TRUE", nil
		}
		placeholders := make([]string, len(values))
		for i, v := range values {
			c.paramIdx++
			placeholders[i] = fmt.Sprintf("$%d", c.paramIdx)
			c.params = append(c.params, v)
		}
		return fmt.Sprintf("%s%s NOT IN (%s)", notPrefix, col, strings.Join(placeholders, ", ")), nil

	case OpIsNull:
		if cond.Not {
			return fmt.Sprintf("%s IS NOT NULL", col), nil
		}
		return fmt.Sprintf("%s IS NULL", col), nil

	case OpIsNotNull:
		if cond.Not {
			return fmt.Sprintf("%s IS NULL", col), nil
		}
		return fmt.Sprintf("%s IS NOT NULL", col), nil

	case OpIncludes, OpArrayContains:
		// PostgreSQL array contains
		c.paramIdx++
		c.params = append(c.params, cond.Value)
		return fmt.Sprintf("%s%s @> $%d", notPrefix, col, c.paramIdx), nil

	case OpBetween:
		bv, ok := cond.Value.(BetweenValue)
		if !ok {
			return "", NewCompilerError("BETWEEN operator requires BetweenValue")
		}
		c.paramIdx++
		lowIdx := c.paramIdx
		c.params = append(c.params, bv.Low)
		c.paramIdx++
		highIdx := c.paramIdx
		c.params = append(c.params, bv.High)
		return fmt.Sprintf("%s%s BETWEEN $%d AND $%d", notPrefix, col, lowIdx, highIdx), nil

	default:
		return "", NewCompilerError(fmt.Sprintf("unsupported operator: %s", cond.Operator))
	}
}

// compileTextSearch compiles a full-text search condition.
func (c *SQLCompiler) compileTextSearch(cond Condition, entity *EntityMeta) (string, error) {
	val := fmt.Sprint(cond.Value)

	// Check if entity has a tsvector column
	if entity.TSVColumns != nil && len(entity.TSVColumns) > 0 {
		// Use the first tsvector column
		for _, tsvCol := range entity.TSVColumns {
			c.paramIdx++
			switch cond.Operator {
			case OpFuzzy:
				// Fuzzy search using similarity
				c.params = append(c.params, val)
				return fmt.Sprintf("%s @@ to_tsquery('simple', $%d)", c.quoteIdent(tsvCol), c.paramIdx), nil
			case OpExact:
				// Exact phrase search
				c.params = append(c.params, val)
				return fmt.Sprintf("%s @@ phraseto_tsquery('simple', $%d)", c.quoteIdent(tsvCol), c.paramIdx), nil
			default:
				c.params = append(c.params, val)
				return fmt.Sprintf("%s @@ plainto_tsquery('simple', $%d)", c.quoteIdent(tsvCol), c.paramIdx), nil
			}
		}
	}

	// Fallback: search using ILIKE on common text fields
	return "", NewCompilerError("no text search column configured for entity")
}

// getEntity returns the entity metadata, handling case-insensitive lookup.
func (c *SQLCompiler) getEntity(name string) *EntityMeta {
	if entity, ok := c.entities[name]; ok {
		return entity
	}
	// Try case-insensitive lookup
	lower := strings.ToLower(name)
	for k, v := range c.entities {
		if strings.ToLower(k) == lower {
			return v
		}
	}
	return nil
}

// mapColumn maps a field name to its database column name.
func (c *SQLCompiler) mapColumn(entity *EntityMeta, field string) string {
	if entity.Columns != nil {
		if col, ok := entity.Columns[field]; ok {
			return col
		}
	}
	// Default: snake_case the field name
	return toSnakeCase(field)
}

// quoteIdent quotes an identifier for safe SQL usage.
func (c *SQLCompiler) quoteIdent(ident string) string {
	// Handle dot notation for qualified names
	if strings.Contains(ident, ".") {
		parts := strings.Split(ident, ".")
		for i, p := range parts {
			parts[i] = `"` + strings.ReplaceAll(p, `"`, `""`) + `"`
		}
		return strings.Join(parts, ".")
	}
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

// toSnakeCase converts a camelCase or PascalCase string to snake_case.
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(r + 32) // lowercase
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

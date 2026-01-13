// Package mongodb provides MongoDB database connectivity and repository operations.
package mongodb

import (
	"fmt"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FilterOp represents a MongoDB filter operator.
type FilterOp string

// Filter operators for MongoDB queries.
const (
	OpEq    FilterOp = "eq"
	OpNe    FilterOp = "ne"
	OpGt    FilterOp = "gt"
	OpGte   FilterOp = "gte"
	OpLt    FilterOp = "lt"
	OpLte   FilterOp = "lte"
	OpIn    FilterOp = "in"
	OpNin   FilterOp = "nin"
	OpRegex FilterOp = "regex"
)

// mongoOpMap maps DSL operators to MongoDB operators.
var mongoOpMap = map[FilterOp]string{
	OpEq:    "$eq",
	OpNe:    "$ne",
	OpGt:    "$gt",
	OpGte:   "$gte",
	OpLt:    "$lt",
	OpLte:   "$lte",
	OpIn:    "$in",
	OpNin:   "$nin",
	OpRegex: "$regex",
}

// SortDirection represents sort order.
type SortDirection int

const (
	// SortAsc represents ascending sort order.
	SortAsc SortDirection = 1
	// SortDesc represents descending sort order.
	SortDesc SortDirection = -1
)

// FilterExpr represents a single filter expression.
type FilterExpr struct {
	Field    string
	Operator FilterOp
	Value    interface{}
	Options  string // for regex options like "i" (case-insensitive)
}

// LogicalExpr represents a logical grouping of filters.
type LogicalExpr struct {
	Type    string        // "and", "or", "not"
	Filters []FilterExpr
	Groups  []LogicalExpr
}

// SortExpr represents a sort expression.
type SortExpr struct {
	Field     string
	Direction SortDirection
}

// ArrayExpr represents an array query expression.
type ArrayExpr struct {
	Field    string
	Type     string      // "elemMatch", "all", "size"
	Value    interface{} // for size: int, for all: []interface{}, for elemMatch: bson.M
}

// QueryBuilder provides a fluent API for building MongoDB aggregation pipelines.
type QueryBuilder struct {
	collection string
	filters    []FilterExpr
	logicals   []LogicalExpr
	sorts      []SortExpr
	skip       *int64
	limit      *int64
	projection bson.M
	arrayExprs []ArrayExpr
	errors     []error
}

// NewQueryBuilder creates a new QueryBuilder for the specified collection.
func NewQueryBuilder(collection string) *QueryBuilder {
	return &QueryBuilder{
		collection: collection,
		filters:    make([]FilterExpr, 0),
		logicals:   make([]LogicalExpr, 0),
		sorts:      make([]SortExpr, 0),
		projection: make(bson.M),
		arrayExprs: make([]ArrayExpr, 0),
		errors:     make([]error, 0),
	}
}

// =============================================================================
// Filter Methods
// =============================================================================

// Where adds an equality filter.
func (qb *QueryBuilder) Where(field string, value interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpr{
		Field:    field,
		Operator: OpEq,
		Value:    value,
	})
	return qb
}

// Eq adds an equality filter.
func (qb *QueryBuilder) Eq(field string, value interface{}) *QueryBuilder {
	return qb.Where(field, value)
}

// Ne adds a not-equal filter.
func (qb *QueryBuilder) Ne(field string, value interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpr{
		Field:    field,
		Operator: OpNe,
		Value:    value,
	})
	return qb
}

// Gt adds a greater-than filter.
func (qb *QueryBuilder) Gt(field string, value interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpr{
		Field:    field,
		Operator: OpGt,
		Value:    value,
	})
	return qb
}

// Gte adds a greater-than-or-equal filter.
func (qb *QueryBuilder) Gte(field string, value interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpr{
		Field:    field,
		Operator: OpGte,
		Value:    value,
	})
	return qb
}

// Lt adds a less-than filter.
func (qb *QueryBuilder) Lt(field string, value interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpr{
		Field:    field,
		Operator: OpLt,
		Value:    value,
	})
	return qb
}

// Lte adds a less-than-or-equal filter.
func (qb *QueryBuilder) Lte(field string, value interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpr{
		Field:    field,
		Operator: OpLte,
		Value:    value,
	})
	return qb
}

// In adds an "in" filter for matching any value in a list.
func (qb *QueryBuilder) In(field string, values interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpr{
		Field:    field,
		Operator: OpIn,
		Value:    values,
	})
	return qb
}

// Nin adds a "not in" filter for excluding values in a list.
func (qb *QueryBuilder) Nin(field string, values interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpr{
		Field:    field,
		Operator: OpNin,
		Value:    values,
	})
	return qb
}

// Regex adds a regex filter for pattern matching.
func (qb *QueryBuilder) Regex(field string, pattern string, options string) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpr{
		Field:    field,
		Operator: OpRegex,
		Value:    pattern,
		Options:  options,
	})
	return qb
}

// RegexCaseInsensitive adds a case-insensitive regex filter.
func (qb *QueryBuilder) RegexCaseInsensitive(field string, pattern string) *QueryBuilder {
	return qb.Regex(field, pattern, "i")
}

// =============================================================================
// Logical Operator Methods
// =============================================================================

// And groups multiple filters with AND logic.
func (qb *QueryBuilder) And(filters ...FilterExpr) *QueryBuilder {
	qb.logicals = append(qb.logicals, LogicalExpr{
		Type:    "and",
		Filters: filters,
	})
	return qb
}

// Or groups multiple filters with OR logic.
func (qb *QueryBuilder) Or(filters ...FilterExpr) *QueryBuilder {
	qb.logicals = append(qb.logicals, LogicalExpr{
		Type:    "or",
		Filters: filters,
	})
	return qb
}

// Not negates a filter.
func (qb *QueryBuilder) Not(filter FilterExpr) *QueryBuilder {
	qb.logicals = append(qb.logicals, LogicalExpr{
		Type:    "not",
		Filters: []FilterExpr{filter},
	})
	return qb
}

// OrGroup adds an OR group with nested logical expressions.
func (qb *QueryBuilder) OrGroup(groups ...LogicalExpr) *QueryBuilder {
	qb.logicals = append(qb.logicals, LogicalExpr{
		Type:   "or",
		Groups: groups,
	})
	return qb
}

// AndGroup adds an AND group with nested logical expressions.
func (qb *QueryBuilder) AndGroup(groups ...LogicalExpr) *QueryBuilder {
	qb.logicals = append(qb.logicals, LogicalExpr{
		Type:   "and",
		Groups: groups,
	})
	return qb
}

// =============================================================================
// Sorting Methods
// =============================================================================

// Sort adds a sort expression.
func (qb *QueryBuilder) Sort(field string, direction SortDirection) *QueryBuilder {
	qb.sorts = append(qb.sorts, SortExpr{
		Field:     field,
		Direction: direction,
	})
	return qb
}

// SortAscending adds an ascending sort.
func (qb *QueryBuilder) SortAscending(field string) *QueryBuilder {
	return qb.Sort(field, SortAsc)
}

// SortDescending adds a descending sort.
func (qb *QueryBuilder) SortDescending(field string) *QueryBuilder {
	return qb.Sort(field, SortDesc)
}

// =============================================================================
// Pagination Methods
// =============================================================================

// Skip sets the number of documents to skip.
func (qb *QueryBuilder) Skip(n int64) *QueryBuilder {
	qb.skip = &n
	return qb
}

// Limit sets the maximum number of documents to return.
func (qb *QueryBuilder) Limit(n int64) *QueryBuilder {
	qb.limit = &n
	return qb
}

// =============================================================================
// Field Selection Methods
// =============================================================================

// Select specifies which fields to include in results.
func (qb *QueryBuilder) Select(fields ...string) *QueryBuilder {
	for _, field := range fields {
		qb.projection[field] = 1
	}
	return qb
}

// Exclude specifies which fields to exclude from results.
func (qb *QueryBuilder) Exclude(fields ...string) *QueryBuilder {
	for _, field := range fields {
		qb.projection[field] = 0
	}
	return qb
}

// Project sets a custom projection.
func (qb *QueryBuilder) Project(projection bson.M) *QueryBuilder {
	qb.projection = projection
	return qb
}

// =============================================================================
// Array Query Methods
// =============================================================================

// ElemMatch adds an $elemMatch query for array elements.
func (qb *QueryBuilder) ElemMatch(field string, conditions bson.M) *QueryBuilder {
	qb.arrayExprs = append(qb.arrayExprs, ArrayExpr{
		Field: field,
		Type:  "elemMatch",
		Value: conditions,
	})
	return qb
}

// All adds an $all query for arrays containing all specified elements.
func (qb *QueryBuilder) All(field string, values []interface{}) *QueryBuilder {
	qb.arrayExprs = append(qb.arrayExprs, ArrayExpr{
		Field: field,
		Type:  "all",
		Value: values,
	})
	return qb
}

// Size adds a $size query for arrays of a specific length.
func (qb *QueryBuilder) Size(field string, size int) *QueryBuilder {
	qb.arrayExprs = append(qb.arrayExprs, ArrayExpr{
		Field: field,
		Type:  "size",
		Value: size,
	})
	return qb
}

// =============================================================================
// Pipeline Building
// =============================================================================

// BuildFilter builds the $match filter from all filter expressions.
func (qb *QueryBuilder) BuildFilter() bson.M {
	filter := bson.M{}

	// Add simple filters
	for _, f := range qb.filters {
		qb.addFilterToDoc(filter, f)
	}

	// Add logical expressions
	for _, l := range qb.logicals {
		qb.addLogicalToDoc(filter, l)
	}

	// Add array expressions
	for _, a := range qb.arrayExprs {
		qb.addArrayExprToDoc(filter, a)
	}

	return filter
}

// addFilterToDoc adds a single filter expression to a bson.M document.
func (qb *QueryBuilder) addFilterToDoc(doc bson.M, f FilterExpr) {
	// Handle _id field with ObjectID conversion
	value := f.Value
	if f.Field == "_id" {
		if idStr, ok := value.(string); ok {
			if oid, err := primitive.ObjectIDFromHex(idStr); err == nil {
				value = oid
			}
		}
	}

	mongoOp, ok := mongoOpMap[f.Operator]
	if !ok {
		qb.errors = append(qb.errors, fmt.Errorf("unknown operator: %s", f.Operator))
		return
	}

	switch f.Operator {
	case OpEq:
		// Direct equality (no $eq needed unless merging with other ops)
		if existing, exists := doc[f.Field]; exists {
			// Merge with existing conditions
			if existingMap, ok := existing.(bson.M); ok {
				existingMap["$eq"] = value
			} else {
				doc[f.Field] = bson.M{"$eq": value}
			}
		} else {
			doc[f.Field] = value
		}
	case OpRegex:
		regexDoc := bson.M{"$regex": value}
		if f.Options != "" {
			regexDoc["$options"] = f.Options
		}
		doc[f.Field] = regexDoc
	default:
		if existing, exists := doc[f.Field]; exists {
			// Merge with existing conditions on same field
			if existingMap, ok := existing.(bson.M); ok {
				existingMap[mongoOp] = value
			} else {
				doc[f.Field] = bson.M{mongoOp: value}
			}
		} else {
			doc[f.Field] = bson.M{mongoOp: value}
		}
	}
}

// addLogicalToDoc adds a logical expression to a bson.M document.
func (qb *QueryBuilder) addLogicalToDoc(doc bson.M, l LogicalExpr) {
	conditions := make([]bson.M, 0)

	// Convert filters to conditions
	for _, f := range l.Filters {
		condDoc := bson.M{}
		qb.addFilterToDoc(condDoc, f)
		conditions = append(conditions, condDoc)
	}

	// Convert nested groups to conditions
	for _, g := range l.Groups {
		groupDoc := bson.M{}
		qb.addLogicalToDoc(groupDoc, g)
		conditions = append(conditions, groupDoc)
	}

	switch l.Type {
	case "and":
		if existing, exists := doc["$and"]; exists {
			doc["$and"] = append(existing.([]bson.M), conditions...)
		} else {
			doc["$and"] = conditions
		}
	case "or":
		if existing, exists := doc["$or"]; exists {
			doc["$or"] = append(existing.([]bson.M), conditions...)
		} else {
			doc["$or"] = conditions
		}
	case "not":
		if len(conditions) > 0 {
			// $not only works on field-level operators
			for k, v := range conditions[0] {
				doc[k] = bson.M{"$not": v}
			}
		}
	}
}

// addArrayExprToDoc adds an array expression to a bson.M document.
func (qb *QueryBuilder) addArrayExprToDoc(doc bson.M, a ArrayExpr) {
	switch a.Type {
	case "elemMatch":
		doc[a.Field] = bson.M{"$elemMatch": a.Value}
	case "all":
		doc[a.Field] = bson.M{"$all": a.Value}
	case "size":
		doc[a.Field] = bson.M{"$size": a.Value}
	}
}

// BuildSort builds the $sort stage document.
func (qb *QueryBuilder) BuildSort() bson.D {
	if len(qb.sorts) == 0 {
		return nil
	}

	sort := bson.D{}
	for _, s := range qb.sorts {
		sort = append(sort, bson.E{Key: s.Field, Value: int(s.Direction)})
	}
	return sort
}

// BuildPipeline builds the complete aggregation pipeline.
func (qb *QueryBuilder) BuildPipeline() []bson.M {
	pipeline := make([]bson.M, 0)

	// $match stage
	filter := qb.BuildFilter()
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.M{"$match": filter})
	}

	// $sort stage
	sort := qb.BuildSort()
	if len(sort) > 0 {
		pipeline = append(pipeline, bson.M{"$sort": sort})
	}

	// $skip stage
	if qb.skip != nil && *qb.skip > 0 {
		pipeline = append(pipeline, bson.M{"$skip": *qb.skip})
	}

	// $limit stage
	if qb.limit != nil && *qb.limit > 0 {
		pipeline = append(pipeline, bson.M{"$limit": *qb.limit})
	}

	// $project stage
	if len(qb.projection) > 0 {
		pipeline = append(pipeline, bson.M{"$project": qb.projection})
	}

	return pipeline
}

// Errors returns any errors encountered during query building.
func (qb *QueryBuilder) Errors() []error {
	return qb.errors
}

// HasErrors returns true if there are any errors.
func (qb *QueryBuilder) HasErrors() bool {
	return len(qb.errors) > 0
}

// =============================================================================
// DSL Query Parsing
// =============================================================================

// DSLQuery represents a parsed DSL query.
type DSLQuery struct {
	Name       string
	Collection string
	Filter     map[string]interface{}
	Sort       map[string]string
	Limit      *int64
	Skip       *int64
	Select     []string
}

// ParseDSLFilter parses a DSL filter map into MongoDB filter.
func ParseDSLFilter(dslFilter map[string]interface{}) bson.M {
	return parseDSLFilterRecursive(dslFilter)
}

// parseDSLFilterRecursive recursively parses DSL filter expressions.
func parseDSLFilterRecursive(filter map[string]interface{}) bson.M {
	result := bson.M{}

	for field, value := range filter {
		// Handle logical operators
		if field == "$and" || field == "and" {
			if arr, ok := value.([]interface{}); ok {
				andConditions := make([]bson.M, len(arr))
				for i, item := range arr {
					if itemMap, ok := item.(map[string]interface{}); ok {
						andConditions[i] = parseDSLFilterRecursive(itemMap)
					}
				}
				result["$and"] = andConditions
			}
			continue
		}

		if field == "$or" || field == "or" {
			if arr, ok := value.([]interface{}); ok {
				orConditions := make([]bson.M, len(arr))
				for i, item := range arr {
					if itemMap, ok := item.(map[string]interface{}); ok {
						orConditions[i] = parseDSLFilterRecursive(itemMap)
					}
				}
				result["$or"] = orConditions
			}
			continue
		}

		if field == "$not" || field == "not" {
			if itemMap, ok := value.(map[string]interface{}); ok {
				notFilter := parseDSLFilterRecursive(itemMap)
				// Apply $not to each field condition
				for k, v := range notFilter {
					result[k] = bson.M{"$not": v}
				}
			}
			continue
		}

		// Handle nested operator expressions { field: { op: value } }
		if opMap, ok := value.(map[string]interface{}); ok {
			fieldConditions := bson.M{}
			for op, opValue := range opMap {
				mongoOp := translateDSLOperator(op)
				if mongoOp != "" {
					// Handle ObjectID for _id field
					if field == "_id" {
						if idStr, ok := opValue.(string); ok {
							if oid, err := primitive.ObjectIDFromHex(idStr); err == nil {
								opValue = oid
							}
						}
					}
					fieldConditions[mongoOp] = opValue
				} else {
					// Nested document query (e.g., address.city)
					fieldConditions[op] = opValue
				}
			}
			if len(fieldConditions) > 0 {
				result[field] = fieldConditions
			}
			continue
		}

		// Handle direct equality
		// Handle _id field with ObjectID conversion
		if field == "_id" {
			if idStr, ok := value.(string); ok {
				if oid, err := primitive.ObjectIDFromHex(idStr); err == nil {
					value = oid
				}
			}
		}
		result[field] = value
	}

	return result
}

// translateDSLOperator converts DSL operator names to MongoDB operators.
func translateDSLOperator(op string) string {
	op = strings.ToLower(op)
	switch op {
	case "eq", "$eq":
		return "$eq"
	case "ne", "$ne", "neq":
		return "$ne"
	case "gt", "$gt":
		return "$gt"
	case "gte", "$gte", "ge":
		return "$gte"
	case "lt", "$lt":
		return "$lt"
	case "lte", "$lte", "le":
		return "$lte"
	case "in", "$in":
		return "$in"
	case "nin", "$nin", "notin":
		return "$nin"
	case "regex", "$regex":
		return "$regex"
	case "exists", "$exists":
		return "$exists"
	case "type", "$type":
		return "$type"
	case "elemmatch", "$elemmatch", "elemMatch":
		return "$elemMatch"
	case "all", "$all":
		return "$all"
	case "size", "$size":
		return "$size"
	default:
		return ""
	}
}

// BuildFromDSL creates a QueryBuilder from a DSL query definition.
func BuildFromDSL(dsl *DSLQuery) *QueryBuilder {
	qb := NewQueryBuilder(dsl.Collection)

	// Parse filter
	if dsl.Filter != nil {
		mongoFilter := ParseDSLFilter(dsl.Filter)
		// Add parsed filter conditions directly
		for field, value := range mongoFilter {
			if field == "$and" || field == "$or" {
				// These are already in correct format
				if arr, ok := value.([]bson.M); ok {
					for _, cond := range arr {
						for f, v := range cond {
							qb.filters = append(qb.filters, FilterExpr{
								Field:    f,
								Operator: OpEq,
								Value:    v,
							})
						}
					}
				}
			} else {
				qb.filters = append(qb.filters, FilterExpr{
					Field:    field,
					Operator: OpEq,
					Value:    value,
				})
			}
		}
	}

	// Parse sort
	if dsl.Sort != nil {
		for field, dir := range dsl.Sort {
			direction := SortAsc
			if strings.ToLower(dir) == "desc" || dir == "-1" {
				direction = SortDesc
			}
			qb.Sort(field, direction)
		}
	}

	// Set pagination
	if dsl.Limit != nil {
		qb.Limit(*dsl.Limit)
	}
	if dsl.Skip != nil {
		qb.Skip(*dsl.Skip)
	}

	// Set projection
	if len(dsl.Select) > 0 {
		qb.Select(dsl.Select...)
	}

	return qb
}

// BuildPipelineFromDSL creates an aggregation pipeline directly from a DSL filter map.
func BuildPipelineFromDSL(filter map[string]interface{}, sort map[string]string, limit, skip *int64, project []string) []bson.M {
	pipeline := make([]bson.M, 0)

	// $match stage
	if filter != nil && len(filter) > 0 {
		mongoFilter := ParseDSLFilter(filter)
		pipeline = append(pipeline, bson.M{"$match": mongoFilter})
	}

	// $sort stage
	if sort != nil && len(sort) > 0 {
		sortDoc := bson.D{}
		for field, dir := range sort {
			sortVal := 1
			if strings.ToLower(dir) == "desc" || dir == "-1" {
				sortVal = -1
			}
			sortDoc = append(sortDoc, bson.E{Key: field, Value: sortVal})
		}
		pipeline = append(pipeline, bson.M{"$sort": sortDoc})
	}

	// $skip stage
	if skip != nil && *skip > 0 {
		pipeline = append(pipeline, bson.M{"$skip": *skip})
	}

	// $limit stage
	if limit != nil && *limit > 0 {
		pipeline = append(pipeline, bson.M{"$limit": *limit})
	}

	// $project stage
	if len(project) > 0 {
		projection := bson.M{}
		for _, field := range project {
			projection[field] = 1
		}
		pipeline = append(pipeline, bson.M{"$project": projection})
	}

	return pipeline
}

// =============================================================================
// Utility Functions
// =============================================================================

// EscapeRegexPattern escapes special regex characters in a pattern string.
func EscapeRegexPattern(pattern string) string {
	special := regexp.MustCompile(`([\\.*+?^${}()|[\]])`)
	return special.ReplaceAllString(pattern, `\$1`)
}

// WildcardToRegex converts wildcard patterns (* and ?) to regex.
func WildcardToRegex(pattern string) string {
	escaped := EscapeRegexPattern(pattern)
	escaped = strings.ReplaceAll(escaped, `\*`, ".*")
	escaped = strings.ReplaceAll(escaped, `\?`, ".")
	return "^" + escaped + "$"
}

// ContainsRegex creates a case-insensitive contains regex pattern.
func ContainsRegex(substring string) primitive.Regex {
	return primitive.Regex{
		Pattern: EscapeRegexPattern(substring),
		Options: "i",
	}
}

// StartsWithRegex creates a case-insensitive starts-with regex pattern.
func StartsWithRegex(prefix string) primitive.Regex {
	return primitive.Regex{
		Pattern: "^" + EscapeRegexPattern(prefix),
		Options: "i",
	}
}

// EndsWithRegex creates a case-insensitive ends-with regex pattern.
func EndsWithRegex(suffix string) primitive.Regex {
	return primitive.Regex{
		Pattern: EscapeRegexPattern(suffix) + "$",
		Options: "i",
	}
}

package query

import "fmt"

// QueryType represents the type of query operation.
type QueryType int

const (
	QuerySelect QueryType = iota
	QueryCount
	QuerySum
	QueryAvg
	QueryMin
	QueryMax
	QueryUpdate
	QueryDelete
)

// String returns a string representation of the QueryType.
func (qt QueryType) String() string {
	switch qt {
	case QuerySelect:
		return "SELECT"
	case QueryCount:
		return "COUNT"
	case QuerySum:
		return "SUM"
	case QueryAvg:
		return "AVG"
	case QueryMin:
		return "MIN"
	case QueryMax:
		return "MAX"
	case QueryUpdate:
		return "UPDATE"
	case QueryDelete:
		return "DELETE"
	default:
		return fmt.Sprintf("Unknown(%d)", qt)
	}
}

// Query represents a parsed query.
type Query struct {
	Type       QueryType
	Entity     string
	Fields     []string      // SELECT fields, empty = all
	Where      *WhereClause
	OrderBy    []OrderClause
	Limit      *int
	Offset     *int
	Include    []string      // Relations to load
	GroupBy    []string
	Having     *WhereClause
	Updates    []UpdateSet   // For UPDATE queries
	AggField   string        // Field for aggregate functions (SUM, AVG, etc.)
}

// WhereClause represents a WHERE clause with conditions.
type WhereClause struct {
	Conditions []Condition
	Operator   LogicalOp
	Groups     []*WhereClause // Nested groups for complex expressions
}

// Condition represents a single condition in a WHERE clause.
type Condition struct {
	Field    string
	Operator CompareOp
	Value    interface{}
	Not      bool           // For NOT prefix
	SubQuery *Query         // For subqueries
	Nested   *WhereClause   // For grouped conditions (a OR b) AND c
}

// CompareOp represents a comparison operator.
type CompareOp int

const (
	OpEquals CompareOp = iota
	OpNotEquals
	OpGreaterThan
	OpGreaterThanOrEqual
	OpLessThan
	OpLessThanOrEqual
	OpContains
	OpStartsWith
	OpEndsWith
	OpIn
	OpNotIn
	OpIsNull
	OpIsNotNull
	OpIncludes       // For arrays
	OpLike
	OpILike
	OpBetween
	OpFuzzy          // For fuzzy search (~)
	OpExact          // For exact phrase search
	OpArrayContains  // For PostgreSQL @>
)

// String returns a string representation of the CompareOp.
func (op CompareOp) String() string {
	switch op {
	case OpEquals:
		return "="
	case OpNotEquals:
		return "!="
	case OpGreaterThan:
		return ">"
	case OpGreaterThanOrEqual:
		return ">="
	case OpLessThan:
		return "<"
	case OpLessThanOrEqual:
		return "<="
	case OpContains:
		return "CONTAINS"
	case OpStartsWith:
		return "STARTSWITH"
	case OpEndsWith:
		return "ENDSWITH"
	case OpIn:
		return "IN"
	case OpNotIn:
		return "NOT IN"
	case OpIsNull:
		return "IS NULL"
	case OpIsNotNull:
		return "IS NOT NULL"
	case OpIncludes:
		return "INCLUDES"
	case OpLike:
		return "LIKE"
	case OpILike:
		return "ILIKE"
	case OpBetween:
		return "BETWEEN"
	case OpFuzzy:
		return "~"
	case OpExact:
		return "EXACT"
	case OpArrayContains:
		return "@>"
	default:
		return fmt.Sprintf("Unknown(%d)", op)
	}
}

// LogicalOp represents a logical operator (AND/OR).
type LogicalOp int

const (
	LogicalAnd LogicalOp = iota
	LogicalOr
)

// String returns a string representation of the LogicalOp.
func (op LogicalOp) String() string {
	switch op {
	case LogicalAnd:
		return "AND"
	case LogicalOr:
		return "OR"
	default:
		return fmt.Sprintf("Unknown(%d)", op)
	}
}

// OrderClause represents an ORDER BY clause.
type OrderClause struct {
	Field     string
	Direction OrderDirection
}

// OrderDirection represents the sort direction.
type OrderDirection int

const (
	OrderAsc OrderDirection = iota
	OrderDesc
)

// String returns a string representation of the OrderDirection.
func (od OrderDirection) String() string {
	switch od {
	case OrderAsc:
		return "ASC"
	case OrderDesc:
		return "DESC"
	default:
		return fmt.Sprintf("Unknown(%d)", od)
	}
}

// UpdateSet represents a field update in an UPDATE query.
type UpdateSet struct {
	Field string
	Value interface{}
	Op    UpdateOp
}

// UpdateOp represents the type of update operation.
type UpdateOp int

const (
	UpdateSetValue UpdateOp = iota
	UpdateIncrement
	UpdateDecrement
	UpdateAppend
	UpdateRemove
)

// String returns a string representation of the UpdateOp.
func (op UpdateOp) String() string {
	switch op {
	case UpdateSetValue:
		return "SET"
	case UpdateIncrement:
		return "INCREMENT"
	case UpdateDecrement:
		return "DECREMENT"
	case UpdateAppend:
		return "APPEND"
	case UpdateRemove:
		return "REMOVE"
	default:
		return fmt.Sprintf("Unknown(%d)", op)
	}
}

// Parameter represents a named parameter in a query.
type Parameter struct {
	Name string
}

// BetweenValue represents a range for BETWEEN operator.
type BetweenValue struct {
	Low  interface{}
	High interface{}
}

// EntityMeta contains metadata about an entity for SQL compilation.
type EntityMeta struct {
	TableName   string
	PrimaryKey  string
	SoftDelete  string            // Column name for soft delete (e.g., "deleted_at")
	Columns     map[string]string // Maps field names to column names
	JSONColumns map[string]bool   // Columns that store JSON data
	TSVColumns  map[string]string // Full-text search columns (field -> tsvector column)
	Relations   map[string]*RelationMeta
}

// RelationMeta describes a relation to another entity.
type RelationMeta struct {
	Entity       string
	Type         RelationType
	ForeignKey   string
	JoinTable    string // For many-to-many
	JoinColumn   string
	InverseColumn string
}

// RelationType represents the type of relation.
type RelationType int

const (
	RelationHasOne RelationType = iota
	RelationHasMany
	RelationBelongsTo
	RelationManyToMany
)

// String returns a string representation of the RelationType.
func (rt RelationType) String() string {
	switch rt {
	case RelationHasOne:
		return "HAS_ONE"
	case RelationHasMany:
		return "HAS_MANY"
	case RelationBelongsTo:
		return "BELONGS_TO"
	case RelationManyToMany:
		return "MANY_TO_MANY"
	default:
		return fmt.Sprintf("Unknown(%d)", rt)
	}
}

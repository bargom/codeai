package graphql

import (
	"fmt"
	"reflect"
	"strings"
)

// QueryBuilder provides a fluent interface for building GraphQL queries.
type QueryBuilder struct {
	operationType string
	operationName string
	variables     []variable
	fields        []field
	fragments     []string
}

type variable struct {
	name         string
	variableType string
	defaultValue interface{}
}

type field struct {
	name      string
	alias     string
	args      []argument
	subFields []field
	fragments []string
	directives []directive
}

type argument struct {
	name  string
	value interface{}
}

type directive struct {
	name string
	args []argument
}

// NewQuery creates a new query builder.
func NewQuery() *QueryBuilder {
	return &QueryBuilder{
		operationType: "query",
	}
}

// NewMutation creates a new mutation builder.
func NewMutation() *QueryBuilder {
	return &QueryBuilder{
		operationType: "mutation",
	}
}

// NewSubscription creates a new subscription builder.
func NewSubscription() *QueryBuilder {
	return &QueryBuilder{
		operationType: "subscription",
	}
}

// Name sets the operation name.
func (q *QueryBuilder) Name(name string) *QueryBuilder {
	q.operationName = name
	return q
}

// Variable adds a variable definition.
func (q *QueryBuilder) Variable(name, variableType string, defaultValue ...interface{}) *QueryBuilder {
	v := variable{
		name:         name,
		variableType: variableType,
	}
	if len(defaultValue) > 0 {
		v.defaultValue = defaultValue[0]
	}
	q.variables = append(q.variables, v)
	return q
}

// Field adds a field to the query.
func (q *QueryBuilder) Field(name string) *FieldBuilder {
	f := &FieldBuilder{
		parent: q,
		field:  field{name: name},
	}
	return f
}

// Fields adds multiple simple fields.
func (q *QueryBuilder) Fields(names ...string) *QueryBuilder {
	for _, name := range names {
		q.fields = append(q.fields, field{name: name})
	}
	return q
}

// Fragment adds a fragment reference.
func (q *QueryBuilder) Fragment(name string) *QueryBuilder {
	q.fragments = append(q.fragments, name)
	return q
}

// Build builds the GraphQL query string.
func (q *QueryBuilder) Build() string {
	var sb strings.Builder

	// Operation type and name
	sb.WriteString(q.operationType)
	if q.operationName != "" {
		sb.WriteString(" ")
		sb.WriteString(q.operationName)
	}

	// Variables
	if len(q.variables) > 0 {
		sb.WriteString("(")
		for i, v := range q.variables {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("$")
			sb.WriteString(v.name)
			sb.WriteString(": ")
			sb.WriteString(v.variableType)
			if v.defaultValue != nil {
				sb.WriteString(" = ")
				sb.WriteString(formatValue(v.defaultValue))
			}
		}
		sb.WriteString(")")
	}

	// Fields
	sb.WriteString(" { ")
	for i, f := range q.fields {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(buildField(f))
	}
	for _, frag := range q.fragments {
		sb.WriteString(" ...")
		sb.WriteString(frag)
	}
	sb.WriteString(" }")

	return sb.String()
}

// FieldBuilder builds a field.
type FieldBuilder struct {
	parent *QueryBuilder
	field  field
}

// Alias sets the field alias.
func (f *FieldBuilder) Alias(alias string) *FieldBuilder {
	f.field.alias = alias
	return f
}

// Arg adds an argument to the field.
func (f *FieldBuilder) Arg(name string, value interface{}) *FieldBuilder {
	f.field.args = append(f.field.args, argument{name: name, value: value})
	return f
}

// Args adds multiple arguments to the field.
func (f *FieldBuilder) Args(args map[string]interface{}) *FieldBuilder {
	for name, value := range args {
		f.field.args = append(f.field.args, argument{name: name, value: value})
	}
	return f
}

// VarArg adds a variable argument to the field.
func (f *FieldBuilder) VarArg(name, variableName string) *FieldBuilder {
	f.field.args = append(f.field.args, argument{name: name, value: variableRef(variableName)})
	return f
}

// Select adds sub-fields.
func (f *FieldBuilder) Select(names ...string) *FieldBuilder {
	for _, name := range names {
		f.field.subFields = append(f.field.subFields, field{name: name})
	}
	return f
}

// SubField adds a sub-field builder.
func (f *FieldBuilder) SubField(name string) *SubFieldBuilder {
	return &SubFieldBuilder{
		parent: f,
		field:  field{name: name},
	}
}

// Fragment adds a fragment reference.
func (f *FieldBuilder) Fragment(name string) *FieldBuilder {
	f.field.fragments = append(f.field.fragments, name)
	return f
}

// Include adds @include directive.
func (f *FieldBuilder) Include(variableName string) *FieldBuilder {
	f.field.directives = append(f.field.directives, directive{
		name: "include",
		args: []argument{{name: "if", value: variableRef(variableName)}},
	})
	return f
}

// Skip adds @skip directive.
func (f *FieldBuilder) Skip(variableName string) *FieldBuilder {
	f.field.directives = append(f.field.directives, directive{
		name: "skip",
		args: []argument{{name: "if", value: variableRef(variableName)}},
	})
	return f
}

// Done completes the field and returns the parent query builder.
func (f *FieldBuilder) Done() *QueryBuilder {
	f.parent.fields = append(f.parent.fields, f.field)
	return f.parent
}

// SubFieldBuilder builds a sub-field.
type SubFieldBuilder struct {
	parent *FieldBuilder
	field  field
}

// Alias sets the sub-field alias.
func (sf *SubFieldBuilder) Alias(alias string) *SubFieldBuilder {
	sf.field.alias = alias
	return sf
}

// Arg adds an argument to the sub-field.
func (sf *SubFieldBuilder) Arg(name string, value interface{}) *SubFieldBuilder {
	sf.field.args = append(sf.field.args, argument{name: name, value: value})
	return sf
}

// Select adds nested fields.
func (sf *SubFieldBuilder) Select(names ...string) *SubFieldBuilder {
	for _, name := range names {
		sf.field.subFields = append(sf.field.subFields, field{name: name})
	}
	return sf
}

// Done completes the sub-field and returns the parent field builder.
func (sf *SubFieldBuilder) Done() *FieldBuilder {
	sf.parent.field.subFields = append(sf.parent.field.subFields, sf.field)
	return sf.parent
}

// variableRef is a marker type for variable references.
type variableRef string

// VarRef creates a variable reference for use in query arguments.
// This allows referencing query variables in field arguments.
func VarRef(name string) interface{} {
	return variableRef(name)
}

// buildField builds a field string.
func buildField(f field) string {
	var sb strings.Builder

	// Alias
	if f.alias != "" {
		sb.WriteString(f.alias)
		sb.WriteString(": ")
	}

	// Name
	sb.WriteString(f.name)

	// Arguments
	if len(f.args) > 0 {
		sb.WriteString("(")
		for i, arg := range f.args {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(arg.name)
			sb.WriteString(": ")
			sb.WriteString(formatValue(arg.value))
		}
		sb.WriteString(")")
	}

	// Directives
	for _, dir := range f.directives {
		sb.WriteString(" @")
		sb.WriteString(dir.name)
		if len(dir.args) > 0 {
			sb.WriteString("(")
			for i, arg := range dir.args {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(arg.name)
				sb.WriteString(": ")
				sb.WriteString(formatValue(arg.value))
			}
			sb.WriteString(")")
		}
	}

	// Sub-fields
	if len(f.subFields) > 0 || len(f.fragments) > 0 {
		sb.WriteString(" { ")
		for i, sf := range f.subFields {
			if i > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(buildField(sf))
		}
		for _, frag := range f.fragments {
			sb.WriteString(" ...")
			sb.WriteString(frag)
		}
		sb.WriteString(" }")
	}

	return sb.String()
}

// formatValue formats a value for GraphQL.
func formatValue(v interface{}) string {
	if v == nil {
		return "null"
	}

	// Handle variable references
	if vr, ok := v.(variableRef); ok {
		return "$" + string(vr)
	}

	switch val := v.(type) {
	case string:
		// Escape special characters
		escaped := strings.ReplaceAll(val, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		escaped = strings.ReplaceAll(escaped, "\n", "\\n")
		return fmt.Sprintf("\"%s\"", escaped)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%v", val)
	default:
		// Handle slices
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice {
			var items []string
			for i := 0; i < rv.Len(); i++ {
				items = append(items, formatValue(rv.Index(i).Interface()))
			}
			return "[" + strings.Join(items, ", ") + "]"
		}

		// Handle maps
		if rv.Kind() == reflect.Map {
			var items []string
			iter := rv.MapRange()
			for iter.Next() {
				key := fmt.Sprintf("%v", iter.Key().Interface())
				val := formatValue(iter.Value().Interface())
				items = append(items, fmt.Sprintf("%s: %s", key, val))
			}
			return "{" + strings.Join(items, ", ") + "}"
		}

		// Default: treat as string
		return fmt.Sprintf("\"%v\"", val)
	}
}

// FragmentBuilder builds a GraphQL fragment.
type FragmentBuilder struct {
	name     string
	onType   string
	fields   []field
}

// NewFragment creates a new fragment builder.
func NewFragment(name, onType string) *FragmentBuilder {
	return &FragmentBuilder{
		name:   name,
		onType: onType,
	}
}

// Fields adds fields to the fragment.
func (f *FragmentBuilder) Fields(names ...string) *FragmentBuilder {
	for _, name := range names {
		f.fields = append(f.fields, field{name: name})
	}
	return f
}

// Field adds a field with sub-selection.
func (f *FragmentBuilder) Field(name string, subFields ...string) *FragmentBuilder {
	fd := field{name: name}
	for _, sf := range subFields {
		fd.subFields = append(fd.subFields, field{name: sf})
	}
	f.fields = append(f.fields, fd)
	return f
}

// Build builds the fragment definition.
func (f *FragmentBuilder) Build() string {
	var sb strings.Builder
	sb.WriteString("fragment ")
	sb.WriteString(f.name)
	sb.WriteString(" on ")
	sb.WriteString(f.onType)
	sb.WriteString(" { ")
	for i, field := range f.fields {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(buildField(field))
	}
	sb.WriteString(" }")
	return sb.String()
}

// InlineFragment builds an inline fragment.
type InlineFragment struct {
	onType string
	fields []field
}

// NewInlineFragment creates a new inline fragment.
func NewInlineFragment(onType string) *InlineFragment {
	return &InlineFragment{
		onType: onType,
	}
}

// Fields adds fields to the inline fragment.
func (f *InlineFragment) Fields(names ...string) *InlineFragment {
	for _, name := range names {
		f.fields = append(f.fields, field{name: name})
	}
	return f
}

// Build builds the inline fragment string.
func (f *InlineFragment) Build() string {
	var sb strings.Builder
	sb.WriteString("... on ")
	sb.WriteString(f.onType)
	sb.WriteString(" { ")
	for i, field := range f.fields {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(buildField(field))
	}
	sb.WriteString(" }")
	return sb.String()
}

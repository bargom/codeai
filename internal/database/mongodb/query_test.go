package mongodb

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewQueryBuilder(t *testing.T) {
	qb := NewQueryBuilder("users")
	if qb == nil {
		t.Fatal("expected non-nil QueryBuilder")
	}
	if qb.collection != "users" {
		t.Errorf("expected collection 'users', got '%s'", qb.collection)
	}
}

func TestQueryBuilder_SimpleEquality(t *testing.T) {
	qb := NewQueryBuilder("users").
		Where("status", "active")

	filter := qb.BuildFilter()

	expected := bson.M{"status": "active"}
	if !reflect.DeepEqual(filter, expected) {
		t.Errorf("expected %v, got %v", expected, filter)
	}
}

func TestQueryBuilder_ComparisonOperators(t *testing.T) {
	tests := []struct {
		name     string
		builder  *QueryBuilder
		expected bson.M
	}{
		{
			name:     "greater than",
			builder:  NewQueryBuilder("users").Gt("age", 18),
			expected: bson.M{"age": bson.M{"$gt": 18}},
		},
		{
			name:     "greater than or equal",
			builder:  NewQueryBuilder("users").Gte("age", 18),
			expected: bson.M{"age": bson.M{"$gte": 18}},
		},
		{
			name:     "less than",
			builder:  NewQueryBuilder("users").Lt("age", 65),
			expected: bson.M{"age": bson.M{"$lt": 65}},
		},
		{
			name:     "less than or equal",
			builder:  NewQueryBuilder("users").Lte("age", 65),
			expected: bson.M{"age": bson.M{"$lte": 65}},
		},
		{
			name:     "not equal",
			builder:  NewQueryBuilder("users").Ne("status", "deleted"),
			expected: bson.M{"status": bson.M{"$ne": "deleted"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := tt.builder.BuildFilter()
			if !reflect.DeepEqual(filter, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, filter)
			}
		})
	}
}

func TestQueryBuilder_InOperator(t *testing.T) {
	qb := NewQueryBuilder("users").
		In("tags", []string{"admin", "moderator"})

	filter := qb.BuildFilter()

	expected := bson.M{"tags": bson.M{"$in": []string{"admin", "moderator"}}}
	if !reflect.DeepEqual(filter, expected) {
		t.Errorf("expected %v, got %v", expected, filter)
	}
}

func TestQueryBuilder_NinOperator(t *testing.T) {
	qb := NewQueryBuilder("users").
		Nin("status", []string{"banned", "deleted"})

	filter := qb.BuildFilter()

	expected := bson.M{"status": bson.M{"$nin": []string{"banned", "deleted"}}}
	if !reflect.DeepEqual(filter, expected) {
		t.Errorf("expected %v, got %v", expected, filter)
	}
}

func TestQueryBuilder_RegexOperator(t *testing.T) {
	qb := NewQueryBuilder("users").
		Regex("email", ".*@example\\.com$", "i")

	filter := qb.BuildFilter()

	expected := bson.M{"email": bson.M{"$regex": ".*@example\\.com$", "$options": "i"}}
	if !reflect.DeepEqual(filter, expected) {
		t.Errorf("expected %v, got %v", expected, filter)
	}
}

func TestQueryBuilder_RegexCaseInsensitive(t *testing.T) {
	qb := NewQueryBuilder("users").
		RegexCaseInsensitive("name", "john")

	filter := qb.BuildFilter()

	expected := bson.M{"name": bson.M{"$regex": "john", "$options": "i"}}
	if !reflect.DeepEqual(filter, expected) {
		t.Errorf("expected %v, got %v", expected, filter)
	}
}

func TestQueryBuilder_LogicalAnd(t *testing.T) {
	qb := NewQueryBuilder("users").
		And(
			FilterExpr{Field: "status", Operator: OpEq, Value: "active"},
			FilterExpr{Field: "age", Operator: OpGt, Value: 18},
		)

	filter := qb.BuildFilter()

	// Check that $and exists
	andConditions, ok := filter["$and"].([]bson.M)
	if !ok {
		t.Fatal("expected $and to be []bson.M")
	}
	if len(andConditions) != 2 {
		t.Errorf("expected 2 conditions, got %d", len(andConditions))
	}
}

func TestQueryBuilder_LogicalOr(t *testing.T) {
	qb := NewQueryBuilder("users").
		Or(
			FilterExpr{Field: "role", Operator: OpEq, Value: "admin"},
			FilterExpr{Field: "role", Operator: OpEq, Value: "moderator"},
		)

	filter := qb.BuildFilter()

	// Check that $or exists
	orConditions, ok := filter["$or"].([]bson.M)
	if !ok {
		t.Fatal("expected $or to be []bson.M")
	}
	if len(orConditions) != 2 {
		t.Errorf("expected 2 conditions, got %d", len(orConditions))
	}
}

func TestQueryBuilder_LogicalNot(t *testing.T) {
	qb := NewQueryBuilder("users").
		Not(FilterExpr{Field: "status", Operator: OpEq, Value: "deleted"})

	filter := qb.BuildFilter()

	// Check that $not is applied
	statusFilter, ok := filter["status"].(bson.M)
	if !ok {
		t.Fatal("expected status to have filter")
	}
	if _, hasNot := statusFilter["$not"]; !hasNot {
		t.Error("expected $not operator")
	}
}

func TestQueryBuilder_ArrayElemMatch(t *testing.T) {
	qb := NewQueryBuilder("orders").
		ElemMatch("items", bson.M{"quantity": bson.M{"$gt": 5}, "price": bson.M{"$lt": 100}})

	filter := qb.BuildFilter()

	expected := bson.M{
		"items": bson.M{
			"$elemMatch": bson.M{
				"quantity": bson.M{"$gt": 5},
				"price":    bson.M{"$lt": 100},
			},
		},
	}
	if !reflect.DeepEqual(filter, expected) {
		t.Errorf("expected %v, got %v", expected, filter)
	}
}

func TestQueryBuilder_ArrayAll(t *testing.T) {
	qb := NewQueryBuilder("users").
		All("tags", []interface{}{"golang", "mongodb"})

	filter := qb.BuildFilter()

	expected := bson.M{
		"tags": bson.M{"$all": []interface{}{"golang", "mongodb"}},
	}
	if !reflect.DeepEqual(filter, expected) {
		t.Errorf("expected %v, got %v", expected, filter)
	}
}

func TestQueryBuilder_ArraySize(t *testing.T) {
	qb := NewQueryBuilder("users").
		Size("comments", 5)

	filter := qb.BuildFilter()

	expected := bson.M{"comments": bson.M{"$size": 5}}
	if !reflect.DeepEqual(filter, expected) {
		t.Errorf("expected %v, got %v", expected, filter)
	}
}

func TestQueryBuilder_NestedDocumentQuery(t *testing.T) {
	// MongoDB supports dot notation for nested documents automatically
	qb := NewQueryBuilder("users").
		Where("address.city", "New York").
		Gt("address.zipcode", "10000")

	filter := qb.BuildFilter()

	if filter["address.city"] != "New York" {
		t.Errorf("expected 'New York', got %v", filter["address.city"])
	}
	if zipFilter, ok := filter["address.zipcode"].(bson.M); ok {
		if zipFilter["$gt"] != "10000" {
			t.Errorf("expected $gt '10000', got %v", zipFilter["$gt"])
		}
	} else {
		t.Error("expected address.zipcode to have $gt filter")
	}
}

func TestQueryBuilder_Sorting(t *testing.T) {
	qb := NewQueryBuilder("users").
		SortDescending("created_at").
		SortAscending("name")

	sort := qb.BuildSort()

	if len(sort) != 2 {
		t.Fatalf("expected 2 sort fields, got %d", len(sort))
	}
	if sort[0].Key != "created_at" || sort[0].Value != -1 {
		t.Errorf("expected created_at: -1, got %s: %v", sort[0].Key, sort[0].Value)
	}
	if sort[1].Key != "name" || sort[1].Value != 1 {
		t.Errorf("expected name: 1, got %s: %v", sort[1].Key, sort[1].Value)
	}
}

func TestQueryBuilder_Pagination(t *testing.T) {
	qb := NewQueryBuilder("users").
		Skip(20).
		Limit(10)

	pipeline := qb.BuildPipeline()

	// Find $skip and $limit stages
	var foundSkip, foundLimit bool
	for _, stage := range pipeline {
		if skip, ok := stage["$skip"]; ok {
			if skip != int64(20) {
				t.Errorf("expected $skip 20, got %v", skip)
			}
			foundSkip = true
		}
		if limit, ok := stage["$limit"]; ok {
			if limit != int64(10) {
				t.Errorf("expected $limit 10, got %v", limit)
			}
			foundLimit = true
		}
	}

	if !foundSkip {
		t.Error("expected $skip stage in pipeline")
	}
	if !foundLimit {
		t.Error("expected $limit stage in pipeline")
	}
}

func TestQueryBuilder_FieldSelection(t *testing.T) {
	qb := NewQueryBuilder("users").
		Select("name", "email", "age")

	pipeline := qb.BuildPipeline()

	// Find $project stage
	var projectStage bson.M
	for _, stage := range pipeline {
		if proj, ok := stage["$project"]; ok {
			projectStage = proj.(bson.M)
			break
		}
	}

	if projectStage == nil {
		t.Fatal("expected $project stage in pipeline")
	}

	for _, field := range []string{"name", "email", "age"} {
		if projectStage[field] != 1 {
			t.Errorf("expected field '%s' to be included", field)
		}
	}
}

func TestQueryBuilder_ExcludeFields(t *testing.T) {
	qb := NewQueryBuilder("users").
		Exclude("password", "secret")

	pipeline := qb.BuildPipeline()

	// Find $project stage
	var projectStage bson.M
	for _, stage := range pipeline {
		if proj, ok := stage["$project"]; ok {
			projectStage = proj.(bson.M)
			break
		}
	}

	if projectStage == nil {
		t.Fatal("expected $project stage in pipeline")
	}

	for _, field := range []string{"password", "secret"} {
		if projectStage[field] != 0 {
			t.Errorf("expected field '%s' to be excluded", field)
		}
	}
}

func TestQueryBuilder_CompletePipeline(t *testing.T) {
	qb := NewQueryBuilder("users").
		Where("status", "active").
		Gt("age", 18).
		SortDescending("created_at").
		Skip(10).
		Limit(20).
		Select("name", "email")

	pipeline := qb.BuildPipeline()

	// Should have: $match, $sort, $skip, $limit, $project
	if len(pipeline) != 5 {
		t.Errorf("expected 5 stages, got %d", len(pipeline))
	}

	// Verify stage order
	stageOrder := []string{"$match", "$sort", "$skip", "$limit", "$project"}
	for i, expectedStage := range stageOrder {
		if i >= len(pipeline) {
			t.Errorf("missing stage %s", expectedStage)
			continue
		}
		if _, ok := pipeline[i][expectedStage]; !ok {
			t.Errorf("expected stage %s at position %d", expectedStage, i)
		}
	}
}

func TestQueryBuilder_ComplexQuery(t *testing.T) {
	// Test complex query with multiple conditions
	qb := NewQueryBuilder("users").
		Where("status", "active").
		Gt("age", 18).
		Lt("age", 65).
		In("role", []string{"user", "admin"}).
		RegexCaseInsensitive("email", "@company\\.com$").
		SortDescending("created_at").
		Limit(100)

	pipeline := qb.BuildPipeline()

	if len(pipeline) < 3 {
		t.Errorf("expected at least 3 stages, got %d", len(pipeline))
	}

	// Verify $match stage contains all conditions
	matchStage, ok := pipeline[0]["$match"].(bson.M)
	if !ok {
		t.Fatal("expected $match stage first")
	}

	if matchStage["status"] != "active" {
		t.Error("expected status filter")
	}
}

// =============================================================================
// DSL Parsing Tests
// =============================================================================

func TestParseDSLFilter_SimpleEquality(t *testing.T) {
	dslFilter := map[string]interface{}{
		"status": "active",
		"age":    25,
	}

	result := ParseDSLFilter(dslFilter)

	if result["status"] != "active" {
		t.Errorf("expected status 'active', got %v", result["status"])
	}
	if result["age"] != 25 {
		t.Errorf("expected age 25, got %v", result["age"])
	}
}

func TestParseDSLFilter_ComparisonOperators(t *testing.T) {
	dslFilter := map[string]interface{}{
		"age": map[string]interface{}{
			"gt": 18,
		},
	}

	result := ParseDSLFilter(dslFilter)

	ageFilter, ok := result["age"].(bson.M)
	if !ok {
		t.Fatal("expected age to be bson.M")
	}
	if ageFilter["$gt"] != 18 {
		t.Errorf("expected $gt 18, got %v", ageFilter["$gt"])
	}
}

func TestParseDSLFilter_MultipleOperators(t *testing.T) {
	dslFilter := map[string]interface{}{
		"age": map[string]interface{}{
			"gt":  18,
			"lte": 65,
		},
	}

	result := ParseDSLFilter(dslFilter)

	ageFilter, ok := result["age"].(bson.M)
	if !ok {
		t.Fatal("expected age to be bson.M")
	}
	if ageFilter["$gt"] != 18 {
		t.Errorf("expected $gt 18, got %v", ageFilter["$gt"])
	}
	if ageFilter["$lte"] != 65 {
		t.Errorf("expected $lte 65, got %v", ageFilter["$lte"])
	}
}

func TestParseDSLFilter_InOperator(t *testing.T) {
	dslFilter := map[string]interface{}{
		"tags": map[string]interface{}{
			"in": []interface{}{"admin", "moderator"},
		},
	}

	result := ParseDSLFilter(dslFilter)

	tagsFilter, ok := result["tags"].(bson.M)
	if !ok {
		t.Fatal("expected tags to be bson.M")
	}
	if tagsFilter["$in"] == nil {
		t.Error("expected $in operator")
	}
}

func TestParseDSLFilter_LogicalAnd(t *testing.T) {
	dslFilter := map[string]interface{}{
		"and": []interface{}{
			map[string]interface{}{"status": "active"},
			map[string]interface{}{"age": map[string]interface{}{"gt": 18}},
		},
	}

	result := ParseDSLFilter(dslFilter)

	andConditions, ok := result["$and"].([]bson.M)
	if !ok {
		t.Fatal("expected $and to be []bson.M")
	}
	if len(andConditions) != 2 {
		t.Errorf("expected 2 conditions, got %d", len(andConditions))
	}
}

func TestParseDSLFilter_LogicalOr(t *testing.T) {
	dslFilter := map[string]interface{}{
		"or": []interface{}{
			map[string]interface{}{"role": "admin"},
			map[string]interface{}{"role": "moderator"},
		},
	}

	result := ParseDSLFilter(dslFilter)

	orConditions, ok := result["$or"].([]bson.M)
	if !ok {
		t.Fatal("expected $or to be []bson.M")
	}
	if len(orConditions) != 2 {
		t.Errorf("expected 2 conditions, got %d", len(orConditions))
	}
}

func TestParseDSLFilter_LogicalNot(t *testing.T) {
	dslFilter := map[string]interface{}{
		"not": map[string]interface{}{
			"status": "deleted",
		},
	}

	result := ParseDSLFilter(dslFilter)

	statusFilter, ok := result["status"].(bson.M)
	if !ok {
		t.Fatal("expected status to be bson.M")
	}
	if _, hasNot := statusFilter["$not"]; !hasNot {
		t.Error("expected $not operator")
	}
}

func TestParseDSLFilter_NestedDocument(t *testing.T) {
	dslFilter := map[string]interface{}{
		"address.city": "New York",
		"address.state": map[string]interface{}{
			"in": []interface{}{"NY", "NJ", "CT"},
		},
	}

	result := ParseDSLFilter(dslFilter)

	if result["address.city"] != "New York" {
		t.Errorf("expected 'New York', got %v", result["address.city"])
	}

	stateFilter, ok := result["address.state"].(bson.M)
	if !ok {
		t.Fatal("expected address.state to be bson.M")
	}
	if stateFilter["$in"] == nil {
		t.Error("expected $in operator for address.state")
	}
}

func TestParseDSLFilter_ArrayQueries(t *testing.T) {
	t.Run("elemMatch", func(t *testing.T) {
		filter := map[string]interface{}{
			"items": map[string]interface{}{
				"elemMatch": map[string]interface{}{
					"quantity": map[string]interface{}{"gt": 5},
				},
			},
		}

		result := ParseDSLFilter(filter)

		// Check structure manually due to type differences
		itemsFilter, ok := result["items"].(bson.M)
		if !ok {
			t.Fatal("expected items to be bson.M")
		}
		if itemsFilter["$elemMatch"] == nil {
			t.Error("expected $elemMatch operator")
		}
	})

	t.Run("all", func(t *testing.T) {
		filter := map[string]interface{}{
			"tags": map[string]interface{}{
				"all": []interface{}{"a", "b"},
			},
		}

		result := ParseDSLFilter(filter)

		tagsFilter, ok := result["tags"].(bson.M)
		if !ok {
			t.Fatal("expected tags to be bson.M")
		}
		allValues := tagsFilter["$all"]
		if allValues == nil {
			t.Error("expected $all operator")
		}
	})

	t.Run("size", func(t *testing.T) {
		filter := map[string]interface{}{
			"comments": map[string]interface{}{
				"size": 10,
			},
		}

		result := ParseDSLFilter(filter)

		commentsFilter, ok := result["comments"].(bson.M)
		if !ok {
			t.Fatal("expected comments to be bson.M")
		}
		if commentsFilter["$size"] != 10 {
			t.Errorf("expected $size 10, got %v", commentsFilter["$size"])
		}
	})
}

func TestParseDSLFilter_ObjectID(t *testing.T) {
	validOID := primitive.NewObjectID().Hex()
	dslFilter := map[string]interface{}{
		"_id": validOID,
	}

	result := ParseDSLFilter(dslFilter)

	// Should be converted to ObjectID
	if _, ok := result["_id"].(primitive.ObjectID); !ok {
		t.Error("expected _id to be converted to ObjectID")
	}
}

func TestParseDSLFilter_ObjectIDWithOperator(t *testing.T) {
	validOID := primitive.NewObjectID().Hex()
	dslFilter := map[string]interface{}{
		"_id": map[string]interface{}{
			"ne": validOID,
		},
	}

	result := ParseDSLFilter(dslFilter)

	idFilter, ok := result["_id"].(bson.M)
	if !ok {
		t.Fatal("expected _id to be bson.M")
	}
	if _, ok := idFilter["$ne"].(primitive.ObjectID); !ok {
		t.Error("expected $ne value to be ObjectID")
	}
}

func TestBuildPipelineFromDSL(t *testing.T) {
	filter := map[string]interface{}{
		"status": "active",
		"age":    map[string]interface{}{"gt": 18},
	}
	sort := map[string]string{
		"created_at": "desc",
	}
	limit := int64(10)
	skip := int64(20)
	project := []string{"name", "email"}

	pipeline := BuildPipelineFromDSL(filter, sort, &limit, &skip, project)

	// Should have $match, $sort, $skip, $limit, $project
	if len(pipeline) != 5 {
		t.Errorf("expected 5 stages, got %d", len(pipeline))
	}

	// Verify stage order
	stageOrder := []string{"$match", "$sort", "$skip", "$limit", "$project"}
	for i, expectedStage := range stageOrder {
		if _, ok := pipeline[i][expectedStage]; !ok {
			t.Errorf("expected stage %s at position %d", expectedStage, i)
		}
	}
}

func TestBuildFromDSL(t *testing.T) {
	limit := int64(10)
	dsl := &DSLQuery{
		Name:       "findActiveUsers",
		Collection: "users",
		Filter: map[string]interface{}{
			"status": "active",
			"age":    map[string]interface{}{"gt": 18},
		},
		Sort: map[string]string{
			"created_at": "desc",
		},
		Limit:  &limit,
		Select: []string{"name", "email"},
	}

	qb := BuildFromDSL(dsl)

	pipeline := qb.BuildPipeline()

	if len(pipeline) < 3 {
		t.Errorf("expected at least 3 stages, got %d", len(pipeline))
	}
}

// =============================================================================
// Utility Function Tests
// =============================================================================

func TestEscapeRegexPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello.world", `hello\.world`},
		{"test*pattern", `test\*pattern`},
		{"query?", `query\?`},
		{"[a-z]", `\[a-z\]`},
		{"plain", "plain"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := EscapeRegexPattern(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestWildcardToRegex(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"*.txt", `^.*\.txt$`},
		{"file?.go", `^file.\.go$`},
		{"test", "^test$"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := WildcardToRegex(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestContainsRegex(t *testing.T) {
	regex := ContainsRegex("test")
	if regex.Pattern != "test" {
		t.Errorf("expected pattern 'test', got '%s'", regex.Pattern)
	}
	if regex.Options != "i" {
		t.Errorf("expected options 'i', got '%s'", regex.Options)
	}
}

func TestStartsWithRegex(t *testing.T) {
	regex := StartsWithRegex("prefix")
	if regex.Pattern != "^prefix" {
		t.Errorf("expected pattern '^prefix', got '%s'", regex.Pattern)
	}
}

func TestEndsWithRegex(t *testing.T) {
	regex := EndsWithRegex("suffix")
	if regex.Pattern != "suffix$" {
		t.Errorf("expected pattern 'suffix$', got '%s'", regex.Pattern)
	}
}

func TestQueryBuilder_Errors(t *testing.T) {
	qb := NewQueryBuilder("users")

	if qb.HasErrors() {
		t.Error("expected no errors initially")
	}

	errors := qb.Errors()
	if len(errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errors))
	}
}

func TestQueryBuilder_ChainedOperations(t *testing.T) {
	// Test that chaining returns the same builder
	qb := NewQueryBuilder("users")

	result := qb.
		Where("a", 1).
		Gt("b", 2).
		Lt("c", 3).
		In("d", []int{1, 2, 3}).
		SortAscending("e").
		Limit(10).
		Skip(5).
		Select("f", "g")

	if result != qb {
		t.Error("expected chaining to return same QueryBuilder")
	}
}

func TestTranslateDSLOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"eq", "$eq"},
		{"EQ", "$eq"},
		{"$eq", "$eq"},
		{"ne", "$ne"},
		{"neq", "$ne"},
		{"gt", "$gt"},
		{"gte", "$gte"},
		{"ge", "$gte"},
		{"lt", "$lt"},
		{"lte", "$lte"},
		{"le", "$lte"},
		{"in", "$in"},
		{"nin", "$nin"},
		{"notin", "$nin"},
		{"regex", "$regex"},
		{"exists", "$exists"},
		{"elemmatch", "$elemMatch"},
		{"all", "$all"},
		{"size", "$size"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := translateDSLOperator(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

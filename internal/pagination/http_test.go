package pagination

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createRequest(query string) *http.Request {
	u, _ := url.Parse("http://example.com/api?" + query)
	return &http.Request{URL: u}
}

func TestParsePageRequest_Default(t *testing.T) {
	r := createRequest("")

	req := ParsePageRequest(r)

	assert.Equal(t, PaginationOffset, req.Type)
	assert.Equal(t, 1, req.Page)
	assert.Equal(t, DefaultLimit, req.Limit)
}

func TestParsePageRequest_Page(t *testing.T) {
	tests := []struct {
		query    string
		expected int
	}{
		{"page=1", 1},
		{"page=5", 5},
		{"page=100", 100},
		{"page=0", 1},    // Invalid, defaults to 1
		{"page=-1", 1},   // Invalid, defaults to 1
		{"page=abc", 1},  // Invalid, defaults to 1
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			r := createRequest(tt.query)
			req := ParsePageRequest(r)
			assert.Equal(t, tt.expected, req.Page)
		})
	}
}

func TestParsePageRequest_Limit(t *testing.T) {
	tests := []struct {
		query    string
		expected int
	}{
		{"limit=10", 10},
		{"limit=50", 50},
		{"limit=100", 100},
		{"limit=0", DefaultLimit},   // Below min, defaults
		{"limit=-1", DefaultLimit},  // Below min, defaults
		{"limit=101", DefaultLimit}, // Above max, defaults
		{"limit=200", DefaultLimit}, // Above max, defaults
		{"limit=abc", DefaultLimit}, // Invalid, defaults
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			r := createRequest(tt.query)
			req := ParsePageRequest(r)
			assert.Equal(t, tt.expected, req.Limit)
		})
	}
}

func TestParsePageRequest_PerPage(t *testing.T) {
	// per_page is alias for limit
	r := createRequest("per_page=25")
	req := ParsePageRequest(r)
	assert.Equal(t, 25, req.Limit)
}

func TestParsePageRequest_LimitOverridesPerPage(t *testing.T) {
	r := createRequest("limit=30&per_page=25")
	req := ParsePageRequest(r)
	assert.Equal(t, 30, req.Limit) // limit takes precedence
}

func TestParsePageRequest_Offset(t *testing.T) {
	tests := []struct {
		query        string
		expectedPage int
	}{
		{"offset=0&limit=20", 1},
		{"offset=20&limit=20", 2},
		{"offset=40&limit=20", 3},
		{"offset=25&limit=10", 3}, // 25/10 = 2, +1 = 3
		{"offset=-1", 1},          // Invalid, page stays 1
		{"offset=abc", 1},         // Invalid, page stays 1
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			r := createRequest(tt.query)
			req := ParsePageRequest(r)
			assert.Equal(t, tt.expectedPage, req.Page)
		})
	}
}

func TestParsePageRequest_Cursor(t *testing.T) {
	cursor := EncodeCursor(Cursor{ID: "100", Field: "id", Value: 100})

	r := createRequest("cursor=" + cursor)
	req := ParsePageRequest(r)

	assert.Equal(t, PaginationCursor, req.Type)
	assert.Equal(t, cursor, req.Cursor)
}

func TestParsePageRequest_After(t *testing.T) {
	cursor := EncodeCursor(Cursor{ID: "100", Field: "id", Value: 100})

	r := createRequest("after=" + cursor)
	req := ParsePageRequest(r)

	assert.Equal(t, PaginationCursor, req.Type)
	assert.Equal(t, cursor, req.After)
}

func TestParsePageRequest_Before(t *testing.T) {
	cursor := EncodeCursor(Cursor{ID: "100", Field: "id", Value: 100})

	r := createRequest("before=" + cursor)
	req := ParsePageRequest(r)

	assert.Equal(t, PaginationCursor, req.Type)
	assert.Equal(t, cursor, req.Before)
}

func TestParsePageRequest_CombinedParams(t *testing.T) {
	r := createRequest("page=3&limit=50")
	req := ParsePageRequest(r)

	assert.Equal(t, PaginationOffset, req.Type)
	assert.Equal(t, 3, req.Page)
	assert.Equal(t, 50, req.Limit)
}

func TestParseFilters(t *testing.T) {
	r := createRequest("name=John&status=active&unknown=value")
	allowed := []string{"name", "status"}

	filters := ParseFilters(r, allowed)

	assert.Equal(t, "John", filters["name"])
	assert.Equal(t, "active", filters["status"])
	assert.NotContains(t, filters, "unknown")
}

func TestParseFilters_WithOperators(t *testing.T) {
	r := createRequest("age[gte]=18&age[lte]=65&status=active")
	allowed := []string{"age", "status"}

	filters := ParseFilters(r, allowed)

	assert.Equal(t, "18", filters["age[gte]"])
	assert.Equal(t, "65", filters["age[lte]"])
	assert.Equal(t, "active", filters["status"])
}

func TestParseFilters_EmptyAllowed(t *testing.T) {
	r := createRequest("name=John&status=active")
	allowed := []string{}

	filters := ParseFilters(r, allowed)

	assert.Empty(t, filters)
}

func TestParseFilters_NilAllowed(t *testing.T) {
	r := createRequest("name=John")

	filters := ParseFilters(r, nil)

	assert.Empty(t, filters)
}

func TestParseFilters_MultipleValues(t *testing.T) {
	// URL with multiple values for same key
	r := createRequest("status=active&status=pending")
	allowed := []string{"status"}

	filters := ParseFilters(r, allowed)

	// Should take first value
	assert.Equal(t, "active", filters["status"])
}

func TestParseSort(t *testing.T) {
	tests := []struct {
		query    string
		allowed  []string
		expected []SortField
	}{
		{
			query:   "sort=name",
			allowed: []string{"name"},
			expected: []SortField{
				{Field: "name", Desc: false},
			},
		},
		{
			query:   "sort=-name",
			allowed: []string{"name"},
			expected: []SortField{
				{Field: "name", Desc: true},
			},
		},
		{
			query:   "sort=+name",
			allowed: []string{"name"},
			expected: []SortField{
				{Field: "name", Desc: false},
			},
		},
		{
			query:   "sort=name:asc",
			allowed: []string{"name"},
			expected: []SortField{
				{Field: "name", Desc: false},
			},
		},
		{
			query:   "sort=name:desc",
			allowed: []string{"name"},
			expected: []SortField{
				{Field: "name", Desc: true},
			},
		},
		{
			query:   "sort=name,created_at",
			allowed: []string{"name", "created_at"},
			expected: []SortField{
				{Field: "name", Desc: false},
				{Field: "created_at", Desc: false},
			},
		},
		{
			query:   "sort=-name,created_at:desc",
			allowed: []string{"name", "created_at"},
			expected: []SortField{
				{Field: "name", Desc: true},
				{Field: "created_at", Desc: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			r := createRequest(tt.query)
			sorts := ParseSort(r, tt.allowed)
			assert.Equal(t, tt.expected, sorts)
		})
	}
}

func TestParseSort_OrderBy(t *testing.T) {
	// order_by is alias for sort
	r := createRequest("order_by=name")
	allowed := []string{"name"}

	sorts := ParseSort(r, allowed)

	assert.Len(t, sorts, 1)
	assert.Equal(t, "name", sorts[0].Field)
}

func TestParseSort_Empty(t *testing.T) {
	r := createRequest("")
	allowed := []string{"name"}

	sorts := ParseSort(r, allowed)

	assert.Nil(t, sorts)
}

func TestParseSort_NotAllowed(t *testing.T) {
	r := createRequest("sort=password")
	allowed := []string{"name", "email"}

	sorts := ParseSort(r, allowed)

	assert.Empty(t, sorts)
}

func TestParseSort_PartialAllowed(t *testing.T) {
	r := createRequest("sort=name,password,email")
	allowed := []string{"name", "email"}

	sorts := ParseSort(r, allowed)

	assert.Len(t, sorts, 2)
	assert.Equal(t, "name", sorts[0].Field)
	assert.Equal(t, "email", sorts[1].Field)
}

func TestParseSortField(t *testing.T) {
	tests := []struct {
		input    string
		expected SortField
	}{
		{"name", SortField{Field: "name", Desc: false}},
		{"-name", SortField{Field: "name", Desc: true}},
		{"+name", SortField{Field: "name", Desc: false}},
		{"name:asc", SortField{Field: "name", Desc: false}},
		{"name:desc", SortField{Field: "name", Desc: true}},
		{"name:ASC", SortField{Field: "name", Desc: false}},
		{"name:DESC", SortField{Field: "name", Desc: true}},
		{"created_at:desc", SortField{Field: "created_at", Desc: true}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSortField(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSort_EmptyParts(t *testing.T) {
	r := createRequest("sort=name,,email")
	allowed := []string{"name", "email"}

	sorts := ParseSort(r, allowed)

	assert.Len(t, sorts, 2)
}

func TestParseSort_Whitespace(t *testing.T) {
	r := createRequest("sort=name, email")
	allowed := []string{"name", "email"}

	sorts := ParseSort(r, allowed)

	assert.Len(t, sorts, 2)
	assert.Equal(t, "email", sorts[1].Field)
}

func TestParseQueryParams(t *testing.T) {
	r := createRequest("page=2&limit=25&name=John&status[in]=active,pending&sort=-created_at")
	allowedFilters := []string{"name", "status"}
	allowedSorts := []string{"created_at", "name"}

	pageReq, filters, sorts := ParseQueryParams(r, allowedFilters, allowedSorts)

	assert.Equal(t, 2, pageReq.Page)
	assert.Equal(t, 25, pageReq.Limit)
	assert.Equal(t, "John", filters["name"])
	assert.Equal(t, "active,pending", filters["status[in]"])
	assert.Len(t, sorts, 1)
	assert.Equal(t, "created_at", sorts[0].Field)
	assert.True(t, sorts[0].Desc)
}

func TestSortField_Fields(t *testing.T) {
	sf := SortField{
		Field: "created_at",
		Desc:  true,
	}

	assert.Equal(t, "created_at", sf.Field)
	assert.True(t, sf.Desc)
}

func TestParseFilters_AllOperatorVariants(t *testing.T) {
	r := createRequest(
		"field[eq]=1&field[ne]=2&field[gt]=3&field[gte]=4&field[lt]=5&field[lte]=6&field[contains]=7&field[in]=8",
	)
	allowed := []string{"field"}

	filters := ParseFilters(r, allowed)

	assert.Equal(t, "1", filters["field[eq]"])
	assert.Equal(t, "2", filters["field[ne]"])
	assert.Equal(t, "3", filters["field[gt]"])
	assert.Equal(t, "4", filters["field[gte]"])
	assert.Equal(t, "5", filters["field[lt]"])
	assert.Equal(t, "6", filters["field[lte]"])
	assert.Equal(t, "7", filters["field[contains]"])
	assert.Equal(t, "8", filters["field[in]"])
}

func TestParsePageRequest_Validation(t *testing.T) {
	// Ensure validation is called
	r := createRequest("page=-1&limit=500")
	req := ParsePageRequest(r)

	// Should be validated
	assert.Equal(t, 1, req.Page)          // -1 corrected to 1
	assert.Equal(t, DefaultLimit, req.Limit) // 500 corrected to default
}

func TestParseFilters_EmptyValues(t *testing.T) {
	// Query with empty value key
	u, _ := url.Parse("http://example.com/api")
	u.RawQuery = "name=" // Empty value
	r := &http.Request{URL: u}
	allowed := []string{"name"}

	filters := ParseFilters(r, allowed)

	// Empty value should be included
	assert.Contains(t, filters, "name")
	assert.Equal(t, "", filters["name"])
}

func TestParseFilters_NoValuesInQuery(t *testing.T) {
	// Create URL with no query params
	u, _ := url.Parse("http://example.com/api")
	r := &http.Request{URL: u}
	allowed := []string{"name"}

	filters := ParseFilters(r, allowed)

	assert.Empty(t, filters)
}

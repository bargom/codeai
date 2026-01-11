package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeCursor(t *testing.T) {
	tests := []struct {
		name   string
		cursor Cursor
	}{
		{
			name: "basic cursor with ID",
			cursor: Cursor{
				ID:    "123",
				Field: "id",
				Value: 123,
			},
		},
		{
			name: "cursor with string value",
			cursor: Cursor{
				ID:    "abc",
				Field: "name",
				Value: "test",
			},
		},
		{
			name: "cursor with direction",
			cursor: Cursor{
				ID:        "456",
				Field:     "created_at",
				Value:     "2024-01-01",
				Direction: "asc",
			},
		},
		{
			name: "empty cursor",
			cursor: Cursor{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeCursor(tt.cursor)
			assert.NotEmpty(t, encoded)

			// Should be base64 URL-safe
			assert.NotContains(t, encoded, "+")
			assert.NotContains(t, encoded, "/")
		})
	}
}

func TestDecodeCursor(t *testing.T) {
	tests := []struct {
		name        string
		cursor      Cursor
		expectError bool
	}{
		{
			name: "valid cursor with ID",
			cursor: Cursor{
				ID:    "123",
				Field: "id",
				Value: float64(123), // JSON numbers decode as float64
			},
		},
		{
			name: "valid cursor with string value",
			cursor: Cursor{
				ID:    "abc",
				Field: "name",
				Value: "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeCursor(tt.cursor)
			decoded, err := DecodeCursor(encoded)

			require.NoError(t, err)
			assert.Equal(t, tt.cursor.ID, decoded.ID)
			assert.Equal(t, tt.cursor.Field, decoded.Field)
			assert.Equal(t, tt.cursor.Value, decoded.Value)
		})
	}
}

func TestDecodeCursor_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "invalid base64",
			input: "not-valid-base64!!!",
		},
		{
			name:  "valid base64 but invalid JSON",
			input: "bm90LWpzb24=", // "not-json" in base64
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := DecodeCursor(tt.input)
			assert.Error(t, err)
			assert.Nil(t, decoded)
			assert.ErrorIs(t, err, ErrInvalidCursor)
		})
	}
}

func TestDefaultPageRequest(t *testing.T) {
	req := DefaultPageRequest()

	assert.Equal(t, PaginationOffset, req.Type)
	assert.Equal(t, 1, req.Page)
	assert.Equal(t, DefaultLimit, req.Limit)
	assert.Empty(t, req.Cursor)
	assert.Empty(t, req.After)
	assert.Empty(t, req.Before)
}

func TestPageRequest_Validate(t *testing.T) {
	tests := []struct {
		name          string
		input         PageRequest
		expectedPage  int
		expectedLimit int
	}{
		{
			name: "valid request unchanged",
			input: PageRequest{
				Page:  5,
				Limit: 50,
			},
			expectedPage:  5,
			expectedLimit: 50,
		},
		{
			name: "zero page becomes 1",
			input: PageRequest{
				Page:  0,
				Limit: 20,
			},
			expectedPage:  1,
			expectedLimit: 20,
		},
		{
			name: "negative page becomes 1",
			input: PageRequest{
				Page:  -5,
				Limit: 20,
			},
			expectedPage:  1,
			expectedLimit: 20,
		},
		{
			name: "zero limit becomes default",
			input: PageRequest{
				Page:  1,
				Limit: 0,
			},
			expectedPage:  1,
			expectedLimit: DefaultLimit,
		},
		{
			name: "limit above max becomes max",
			input: PageRequest{
				Page:  1,
				Limit: 200,
			},
			expectedPage:  1,
			expectedLimit: MaxLimit,
		},
		{
			name: "negative limit becomes default",
			input: PageRequest{
				Page:  1,
				Limit: -10,
			},
			expectedPage:  1,
			expectedLimit: DefaultLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.input
			req.Validate()

			assert.Equal(t, tt.expectedPage, req.Page)
			assert.Equal(t, tt.expectedLimit, req.Limit)
		})
	}
}

func TestPageRequest_GetOffset(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		limit          int
		expectedOffset int
	}{
		{
			name:           "first page",
			page:           1,
			limit:          20,
			expectedOffset: 0,
		},
		{
			name:           "second page",
			page:           2,
			limit:          20,
			expectedOffset: 20,
		},
		{
			name:           "third page with different limit",
			page:           3,
			limit:          10,
			expectedOffset: 20,
		},
		{
			name:           "large page",
			page:           100,
			limit:          50,
			expectedOffset: 4950,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := PageRequest{Page: tt.page, Limit: tt.limit}
			assert.Equal(t, tt.expectedOffset, req.GetOffset())
		})
	}
}

func TestPageRequest_GetCursor(t *testing.T) {
	tests := []struct {
		name           string
		cursor         string
		after          string
		expectedCursor string
	}{
		{
			name:           "cursor takes precedence",
			cursor:         "cursor-value",
			after:          "after-value",
			expectedCursor: "cursor-value",
		},
		{
			name:           "after when no cursor",
			cursor:         "",
			after:          "after-value",
			expectedCursor: "after-value",
		},
		{
			name:           "empty when neither set",
			cursor:         "",
			after:          "",
			expectedCursor: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := PageRequest{Cursor: tt.cursor, After: tt.after}
			assert.Equal(t, tt.expectedCursor, req.GetCursor())
		})
	}
}

func TestPageRequest_IsCursorPagination(t *testing.T) {
	tests := []struct {
		name     string
		req      PageRequest
		expected bool
	}{
		{
			name:     "offset type",
			req:      PageRequest{Type: PaginationOffset},
			expected: false,
		},
		{
			name:     "cursor type",
			req:      PageRequest{Type: PaginationCursor},
			expected: true,
		},
		{
			name:     "cursor parameter set",
			req:      PageRequest{Type: PaginationOffset, Cursor: "abc"},
			expected: true,
		},
		{
			name:     "after parameter set",
			req:      PageRequest{Type: PaginationOffset, After: "abc"},
			expected: true,
		},
		{
			name:     "before parameter set",
			req:      PageRequest{Type: PaginationOffset, Before: "abc"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.req.IsCursorPagination())
		})
	}
}

func TestPaginationType_String(t *testing.T) {
	assert.Equal(t, "offset", string(PaginationOffset))
	assert.Equal(t, "cursor", string(PaginationCursor))
}

func TestConstants(t *testing.T) {
	assert.Equal(t, 20, DefaultLimit)
	assert.Equal(t, 100, MaxLimit)
	assert.Equal(t, 1, MinLimit)
}

func TestPageInfo_JSON(t *testing.T) {
	total := int64(100)
	page := 5
	info := PageInfo{
		Total:       &total,
		Page:        &page,
		Limit:       20,
		HasNext:     true,
		HasPrevious: true,
		NextCursor:  "next",
		PrevCursor:  "prev",
	}

	assert.Equal(t, int64(100), *info.Total)
	assert.Equal(t, 5, *info.Page)
	assert.Equal(t, 20, info.Limit)
	assert.True(t, info.HasNext)
	assert.True(t, info.HasPrevious)
	assert.Equal(t, "next", info.NextCursor)
	assert.Equal(t, "prev", info.PrevCursor)
}

func TestPageResponse_Generic(t *testing.T) {
	// Test with string type
	strResp := PageResponse[string]{
		Data: []string{"a", "b", "c"},
		Pagination: PageInfo{
			Limit:   3,
			HasNext: false,
		},
	}
	assert.Len(t, strResp.Data, 3)
	assert.Equal(t, "a", strResp.Data[0])

	// Test with map type
	mapResp := PageResponse[map[string]any]{
		Data: []map[string]any{
			{"id": 1, "name": "test"},
		},
		Pagination: PageInfo{
			Limit:   1,
			HasNext: false,
		},
	}
	assert.Len(t, mapResp.Data, 1)
	assert.Equal(t, 1, mapResp.Data[0]["id"])
}

func TestCursor_Fields(t *testing.T) {
	c := Cursor{
		ID:        "test-id",
		Field:     "created_at",
		Value:     "2024-01-01",
		Direction: "desc",
	}

	assert.Equal(t, "test-id", c.ID)
	assert.Equal(t, "created_at", c.Field)
	assert.Equal(t, "2024-01-01", c.Value)
	assert.Equal(t, "desc", c.Direction)
}

func TestEncodeCursor_RoundTrip(t *testing.T) {
	// Test various value types
	cursors := []Cursor{
		{ID: "1", Field: "id", Value: float64(100)},
		{ID: "2", Field: "name", Value: "test string"},
		{ID: "3", Field: "active", Value: true},
		{ID: "4", Field: "score", Value: 3.14},
		{ID: "5", Field: "nullable", Value: nil},
	}

	for _, original := range cursors {
		encoded := EncodeCursor(original)
		decoded, err := DecodeCursor(encoded)
		require.NoError(t, err)

		assert.Equal(t, original.ID, decoded.ID)
		assert.Equal(t, original.Field, decoded.Field)
		// Value comparison for JSON round-trip
		if original.Value != nil {
			assert.Equal(t, original.Value, decoded.Value)
		}
	}
}

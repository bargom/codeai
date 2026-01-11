// Package pagination provides offset and cursor-based pagination support
// for database queries with filtering integration.
package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// PaginationType represents the type of pagination to use.
type PaginationType string

const (
	// PaginationOffset uses traditional page/offset pagination.
	PaginationOffset PaginationType = "offset"
	// PaginationCursor uses cursor-based (keyset) pagination.
	PaginationCursor PaginationType = "cursor"
)

const (
	// DefaultLimit is the default number of items per page.
	DefaultLimit = 20
	// MaxLimit is the maximum allowed items per page.
	MaxLimit = 100
	// MinLimit is the minimum allowed items per page.
	MinLimit = 1
)

// PageRequest contains pagination parameters from the request.
type PageRequest struct {
	Type   PaginationType
	Page   int    // For offset pagination (1-indexed)
	Limit  int    // Items per page
	Cursor string // For cursor pagination (forward)
	After  string // Alias for cursor (forward pagination)
	Before string // For backward pagination
}

// PageResponse represents a paginated response with items and pagination metadata.
type PageResponse[T any] struct {
	Data       []T      `json:"data"`
	Pagination PageInfo `json:"pagination"`
}

// PageInfo contains pagination metadata for the response.
type PageInfo struct {
	Total       *int64 `json:"total,omitempty"`
	Page        *int   `json:"page,omitempty"`
	Limit       int    `json:"limit"`
	HasNext     bool   `json:"has_next"`
	HasPrevious bool   `json:"has_previous"`
	NextCursor  string `json:"next_cursor,omitempty"`
	PrevCursor  string `json:"prev_cursor,omitempty"`
}

// Cursor represents an encoded cursor for keyset pagination.
type Cursor struct {
	ID        string `json:"id"`
	Field     string `json:"field,omitempty"`
	Value     any    `json:"value,omitempty"`
	Direction string `json:"dir,omitempty"`
}

// ErrInvalidCursor is returned when cursor decoding fails.
var ErrInvalidCursor = errors.New("invalid cursor")

// EncodeCursor encodes a cursor struct to a base64 URL-safe string.
func EncodeCursor(c Cursor) string {
	data, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

// DecodeCursor decodes a base64 URL-safe string to a cursor struct.
func DecodeCursor(s string) (*Cursor, error) {
	if s == "" {
		return nil, ErrInvalidCursor
	}

	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("%w: base64 decode failed", ErrInvalidCursor)
	}

	var c Cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%w: json unmarshal failed", ErrInvalidCursor)
	}

	return &c, nil
}

// DefaultPageRequest returns a PageRequest with sensible defaults.
func DefaultPageRequest() PageRequest {
	return PageRequest{
		Type:  PaginationOffset,
		Page:  1,
		Limit: DefaultLimit,
	}
}

// Validate validates and normalizes the PageRequest.
func (pr *PageRequest) Validate() {
	if pr.Limit < MinLimit {
		pr.Limit = DefaultLimit
	}
	if pr.Limit > MaxLimit {
		pr.Limit = MaxLimit
	}
	if pr.Page < 1 {
		pr.Page = 1
	}
}

// GetOffset calculates the offset for offset-based pagination.
func (pr *PageRequest) GetOffset() int {
	return (pr.Page - 1) * pr.Limit
}

// GetCursor returns the cursor value, checking both Cursor and After fields.
func (pr *PageRequest) GetCursor() string {
	if pr.Cursor != "" {
		return pr.Cursor
	}
	return pr.After
}

// IsCursorPagination returns true if cursor-based pagination is being used.
func (pr *PageRequest) IsCursorPagination() bool {
	return pr.Type == PaginationCursor || pr.Cursor != "" || pr.After != "" || pr.Before != ""
}

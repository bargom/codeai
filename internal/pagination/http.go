package pagination

import (
	"net/http"
	"strconv"
	"strings"
)

// ParsePageRequest extracts pagination parameters from an HTTP request.
func ParsePageRequest(r *http.Request) PageRequest {
	req := DefaultPageRequest()
	q := r.URL.Query()

	// Parse page number
	if page := q.Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			req.Page = p
		}
	}

	// Parse limit
	if limit := q.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l >= MinLimit && l <= MaxLimit {
			req.Limit = l
		}
	}

	// Parse per_page as alias for limit
	if perPage := q.Get("per_page"); perPage != "" && q.Get("limit") == "" {
		if l, err := strconv.Atoi(perPage); err == nil && l >= MinLimit && l <= MaxLimit {
			req.Limit = l
		}
	}

	// Parse offset (alternative to page)
	if offset := q.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			// Convert offset to page
			req.Page = (o / req.Limit) + 1
		}
	}

	// Parse cursor for cursor-based pagination
	if cursor := q.Get("cursor"); cursor != "" {
		req.Type = PaginationCursor
		req.Cursor = cursor
	}

	// Parse after as alias for cursor
	if after := q.Get("after"); after != "" {
		req.Type = PaginationCursor
		req.After = after
	}

	// Parse before for backward pagination
	if before := q.Get("before"); before != "" {
		req.Type = PaginationCursor
		req.Before = before
	}

	req.Validate()
	return req
}

// ParseFilters extracts filter parameters from an HTTP request.
// Only fields in the allowed list are extracted.
// Supports both field=value and field[op]=value formats.
func ParseFilters(r *http.Request, allowed []string) map[string]any {
	filters := make(map[string]any)
	q := r.URL.Query()

	allowedSet := make(map[string]bool)
	for _, f := range allowed {
		allowedSet[f] = true
	}

	for key, values := range q {
		if len(values) == 0 {
			continue
		}

		// Parse field[op] syntax
		field, _ := ParseFilterKey(key)

		// Check if field is allowed
		if !allowedSet[field] {
			continue
		}

		// Use first value
		filters[key] = values[0]
	}

	return filters
}

// ParseSort extracts sort parameters from an HTTP request.
// Supports formats: sort=field, sort=-field (descending), sort=field:asc, sort=field:desc
func ParseSort(r *http.Request, allowed []string) []SortField {
	q := r.URL.Query()
	sortParam := q.Get("sort")
	if sortParam == "" {
		sortParam = q.Get("order_by")
	}
	if sortParam == "" {
		return nil
	}

	allowedSet := make(map[string]bool)
	for _, f := range allowed {
		allowedSet[f] = true
	}

	var sorts []SortField
	parts := strings.Split(sortParam, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		sf := parseSortField(part)
		if allowedSet[sf.Field] {
			sorts = append(sorts, sf)
		}
	}

	return sorts
}

// SortField represents a sort field with direction.
type SortField struct {
	Field string
	Desc  bool
}

// parseSortField parses a single sort field.
// Supports: field, -field, field:asc, field:desc
func parseSortField(s string) SortField {
	sf := SortField{}

	// Check for prefix - (descending)
	if strings.HasPrefix(s, "-") {
		sf.Field = s[1:]
		sf.Desc = true
		return sf
	}

	// Check for prefix + (ascending, explicit)
	if strings.HasPrefix(s, "+") {
		sf.Field = s[1:]
		sf.Desc = false
		return sf
	}

	// Check for suffix :asc or :desc
	if idx := strings.LastIndex(s, ":"); idx != -1 {
		sf.Field = s[:idx]
		direction := strings.ToLower(s[idx+1:])
		sf.Desc = direction == "desc"
		return sf
	}

	// Default: ascending
	sf.Field = s
	sf.Desc = false
	return sf
}

// ParseQueryParams extracts both pagination and filter parameters from an HTTP request.
func ParseQueryParams(r *http.Request, allowedFilters []string, allowedSorts []string) (PageRequest, map[string]any, []SortField) {
	pageReq := ParsePageRequest(r)
	filters := ParseFilters(r, allowedFilters)
	sorts := ParseSort(r, allowedSorts)
	return pageReq, filters, sorts
}

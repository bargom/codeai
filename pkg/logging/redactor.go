package logging

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"sync"
)

const RedactedValue = "[REDACTED]"

// Default sensitive field names (case-insensitive).
var defaultSensitiveFields = map[string]bool{
	"password":        true,
	"passwd":          true,
	"secret":          true,
	"token":           true,
	"api_key":         true,
	"apikey":          true,
	"api-key":         true,
	"authorization":   true,
	"auth":            true,
	"credential":      true,
	"credentials":     true,
	"credit_card":     true,
	"creditcard":      true,
	"card_number":     true,
	"cardnumber":      true,
	"cvv":             true,
	"ssn":             true,
	"social_security": true,
	"private_key":     true,
	"privatekey":      true,
	"access_token":    true,
	"refresh_token":   true,
	"session_id":      true,
	"sessionid":       true,
	"bearer":          true,
}

// Default patterns for sensitive data in strings.
var defaultSensitivePatterns = []*regexp.Regexp{
	// Password in various formats
	regexp.MustCompile(`(?i)password[\"']?\s*[:=]\s*[\"']?[^\s\"',}]+`),
	// Bearer tokens
	regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9\-_\.]+`),
	// API keys (common formats)
	regexp.MustCompile(`(?i)api[_-]?key[\"']?\s*[:=]\s*[\"']?[a-zA-Z0-9\-_]+`),
	// JWT tokens
	regexp.MustCompile(`eyJ[a-zA-Z0-9\-_]+\.eyJ[a-zA-Z0-9\-_]+\.[a-zA-Z0-9\-_]+`),
	// Email addresses
	regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
	// Credit card numbers (common formats with spaces/dashes)
	regexp.MustCompile(`\b\d{4}[\s\-]?\d{4}[\s\-]?\d{4}[\s\-]?\d{4}\b`),
	// AWS access key ID
	regexp.MustCompile(`(?i)AKIA[0-9A-Z]{16}`),
	// Generic secret/token patterns
	regexp.MustCompile(`(?i)secret[\"']?\s*[:=]\s*[\"']?[^\s\"',}]+`),
}

// Redactor handles redaction of sensitive data.
type Redactor struct {
	sensitiveFields   map[string]bool
	sensitivePatterns []*regexp.Regexp
	allowlistFields   map[string]bool
	mu                sync.RWMutex
}

// NewRedactor creates a new Redactor with default settings.
func NewRedactor() *Redactor {
	r := &Redactor{
		sensitiveFields:   make(map[string]bool),
		sensitivePatterns: make([]*regexp.Regexp, 0, len(defaultSensitivePatterns)),
		allowlistFields:   make(map[string]bool),
	}

	// Copy default sensitive fields
	for k, v := range defaultSensitiveFields {
		r.sensitiveFields[k] = v
	}

	// Copy default patterns
	r.sensitivePatterns = append(r.sensitivePatterns, defaultSensitivePatterns...)

	return r
}

// AddSensitiveField adds a field name to the sensitive list.
func (r *Redactor) AddSensitiveField(field string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sensitiveFields[strings.ToLower(field)] = true
}

// AddSensitivePattern adds a regex pattern to detect sensitive data.
func (r *Redactor) AddSensitivePattern(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sensitivePatterns = append(r.sensitivePatterns, re)
	return nil
}

// AddAllowlistField adds a field to the allowlist (won't be redacted even if matching).
func (r *Redactor) AddAllowlistField(field string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.allowlistFields[strings.ToLower(field)] = true
}

// IsSensitiveField checks if a field name is sensitive.
func (r *Redactor) IsSensitiveField(field string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	lower := strings.ToLower(field)

	// Check allowlist first
	if r.allowlistFields[lower] {
		return false
	}

	return r.sensitiveFields[lower]
}

// RedactString redacts sensitive patterns from a string.
func (r *Redactor) RedactString(s string) string {
	r.mu.RLock()
	patterns := r.sensitivePatterns
	r.mu.RUnlock()

	result := s
	for _, pattern := range patterns {
		result = pattern.ReplaceAllString(result, RedactedValue)
	}
	return result
}

// RedactMap redacts sensitive fields from a map recursively.
func (r *Redactor) RedactMap(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}

	result := make(map[string]any, len(data))

	for k, v := range data {
		// Check if field is sensitive
		if r.IsSensitiveField(k) {
			result[k] = RedactedValue
			continue
		}

		// Recursively process values
		switch val := v.(type) {
		case map[string]any:
			result[k] = r.RedactMap(val)
		case string:
			result[k] = r.RedactString(val)
		case []any:
			result[k] = r.redactSlice(val)
		default:
			result[k] = v
		}
	}

	return result
}

// redactSlice redacts sensitive data from a slice.
func (r *Redactor) redactSlice(data []any) []any {
	if data == nil {
		return nil
	}

	result := make([]any, len(data))
	for i, v := range data {
		switch val := v.(type) {
		case map[string]any:
			result[i] = r.RedactMap(val)
		case string:
			result[i] = r.RedactString(val)
		case []any:
			result[i] = r.redactSlice(val)
		default:
			result[i] = v
		}
	}
	return result
}

// SafeAttrs creates slog attributes with sensitive data redacted.
func (r *Redactor) SafeAttrs(data map[string]any) []slog.Attr {
	redacted := r.RedactMap(data)
	attrs := make([]slog.Attr, 0, len(redacted))

	for k, v := range redacted {
		attrs = append(attrs, slog.Any(k, v))
	}

	return attrs
}

// Global default redactor.
var defaultRedactor = NewRedactor()

// RedactSensitive redacts sensitive fields from a map using the default redactor.
func RedactSensitive(data map[string]any) map[string]any {
	return defaultRedactor.RedactMap(data)
}

// RedactString redacts sensitive patterns from a string using the default redactor.
func RedactStringValue(s string) string {
	return defaultRedactor.RedactString(s)
}

// SafeAttrs creates slog attributes with sensitive data redacted using the default redactor.
func SafeAttrs(data map[string]any) []slog.Attr {
	return defaultRedactor.SafeAttrs(data)
}

// IsSensitiveField checks if a field name is sensitive using the default redactor.
func IsSensitiveField(field string) bool {
	return defaultRedactor.IsSensitiveField(field)
}

// RedactingHandler wraps a slog.Handler to redact sensitive data from log records.
type RedactingHandler struct {
	slog.Handler
	redactor *Redactor
}

// NewRedactingHandler creates a new RedactingHandler.
func NewRedactingHandler(handler slog.Handler, redactor *Redactor) *RedactingHandler {
	if redactor == nil {
		redactor = defaultRedactor
	}
	return &RedactingHandler{
		Handler:  handler,
		redactor: redactor,
	}
}

// Handle processes log records and redacts sensitive data.
func (h *RedactingHandler) Handle(ctx context.Context, r slog.Record) error {
	// Create a new record with redacted attributes
	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

	r.Attrs(func(a slog.Attr) bool {
		newRecord.AddAttrs(h.redactAttr(a))
		return true
	})

	return h.Handler.Handle(ctx, newRecord)
}

// redactAttr redacts a single attribute.
func (h *RedactingHandler) redactAttr(a slog.Attr) slog.Attr {
	// Check if the key is sensitive
	if h.redactor.IsSensitiveField(a.Key) {
		return slog.String(a.Key, RedactedValue)
	}

	// Handle different value types
	switch a.Value.Kind() {
	case slog.KindString:
		return slog.String(a.Key, h.redactor.RedactString(a.Value.String()))
	case slog.KindGroup:
		attrs := a.Value.Group()
		newAttrs := make([]slog.Attr, len(attrs))
		for i, attr := range attrs {
			newAttrs[i] = h.redactAttr(attr)
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(newAttrs...)}
	default:
		return a
	}
}

// WithAttrs returns a new RedactingHandler with the given attributes.
func (h *RedactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Redact attributes being added
	redactedAttrs := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		redactedAttrs[i] = h.redactAttr(a)
	}
	return &RedactingHandler{
		Handler:  h.Handler.WithAttrs(redactedAttrs),
		redactor: h.redactor,
	}
}

// WithGroup returns a new RedactingHandler with the given group.
func (h *RedactingHandler) WithGroup(name string) slog.Handler {
	return &RedactingHandler{
		Handler:  h.Handler.WithGroup(name),
		redactor: h.redactor,
	}
}

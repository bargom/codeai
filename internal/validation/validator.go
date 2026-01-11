// Package validation provides input validation with detailed error messages.
package validation

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Validator validates data against parameter definitions.
type Validator struct {
	errors []ValidationError
}

// ValidationError represents a single validation error with details.
type ValidationError struct {
	Field      string `json:"field"`
	Value      any    `json:"value,omitempty"`
	Rule       string `json:"rule"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors aggregates multiple validation errors.
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// Error implements the error interface.
func (e *ValidationErrors) Error() string {
	var msgs []string
	for _, err := range e.Errors {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// HasErrors returns true if there are any validation errors.
func (e *ValidationErrors) HasErrors() bool {
	return len(e.Errors) > 0
}

// ParamDef defines validation rules for a parameter.
type ParamDef struct {
	Name      string
	Type      string
	Required  bool
	Default   any
	Min       *float64
	Max       *float64
	MinLength *int
	MaxLength *int
	Pattern   *regexp.Regexp
	Enum      []string
	Custom    func(any) error
}

// NewValidator creates a new Validator instance.
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates data against the provided parameter definitions.
// Returns nil if validation passes, or ValidationErrors if validation fails.
func (v *Validator) Validate(data map[string]any, params []ParamDef) *ValidationErrors {
	v.errors = nil

	for _, param := range params {
		value, exists := data[param.Name]

		// Handle required fields
		if !exists || value == nil {
			if param.Required {
				v.addError(ValidationError{
					Field:      param.Name,
					Rule:       "required",
					Message:    fmt.Sprintf("%s is required", param.Name),
					Suggestion: fmt.Sprintf("Provide a value for '%s'", param.Name),
				})
			}
			continue
		}

		// Type validation
		if err := v.validateType(param.Name, value, param.Type); err != nil {
			v.addError(*err)
			continue
		}

		// Constraint validation
		v.validateConstraints(param, value)
	}

	if len(v.errors) > 0 {
		return &ValidationErrors{Errors: v.errors}
	}
	return nil
}

// validateType validates that the value matches the expected type.
func (v *Validator) validateType(field string, value any, expectedType string) *ValidationError {
	switch expectedType {
	case "string", "text":
		if _, ok := value.(string); !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Rule:    "type",
				Message: fmt.Sprintf("%s must be a string", field),
			}
		}
	case "integer":
		switch val := value.(type) {
		case float64:
			if val != float64(int64(val)) {
				return &ValidationError{
					Field:   field,
					Value:   value,
					Rule:    "type",
					Message: fmt.Sprintf("%s must be an integer", field),
				}
			}
		case int, int64:
			// OK
		default:
			return &ValidationError{
				Field:   field,
				Value:   value,
				Rule:    "type",
				Message: fmt.Sprintf("%s must be an integer", field),
			}
		}
	case "decimal", "number":
		if _, ok := value.(float64); !ok {
			if _, ok := value.(int); !ok {
				return &ValidationError{
					Field:   field,
					Value:   value,
					Rule:    "type",
					Message: fmt.Sprintf("%s must be a number", field),
				}
			}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Rule:    "type",
				Message: fmt.Sprintf("%s must be a boolean", field),
			}
		}
	case "uuid":
		str, ok := value.(string)
		if !ok || !isValidUUID(str) {
			return &ValidationError{
				Field:      field,
				Value:      value,
				Rule:       "type",
				Message:    fmt.Sprintf("%s must be a valid UUID", field),
				Suggestion: "Use format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
			}
		}
	case "email":
		str, ok := value.(string)
		if !ok || !isValidEmail(str) {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Rule:    "type",
				Message: fmt.Sprintf("%s must be a valid email address", field),
			}
		}
	case "timestamp", "datetime":
		str, ok := value.(string)
		if !ok {
			return &ValidationError{
				Field:      field,
				Value:      value,
				Rule:       "type",
				Message:    fmt.Sprintf("%s must be a valid timestamp", field),
				Suggestion: "Use ISO 8601 format: 2024-01-15T10:30:00Z",
			}
		}
		if !isValidISO8601(str) {
			return &ValidationError{
				Field:      field,
				Value:      value,
				Rule:       "type",
				Message:    fmt.Sprintf("%s must be a valid ISO 8601 timestamp", field),
				Suggestion: "Use ISO 8601 format: 2024-01-15T10:30:00Z",
			}
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Rule:    "type",
				Message: fmt.Sprintf("%s must be an array", field),
			}
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Rule:    "type",
				Message: fmt.Sprintf("%s must be an object", field),
			}
		}
	}
	return nil
}

// validateConstraints validates value constraints (min, max, length, pattern, enum, custom).
func (v *Validator) validateConstraints(param ParamDef, value any) {
	// Min/Max for numbers
	if num, ok := toFloat64(value); ok {
		if param.Min != nil && num < *param.Min {
			v.addError(ValidationError{
				Field:   param.Name,
				Value:   value,
				Rule:    "min",
				Message: fmt.Sprintf("%s must be at least %v", param.Name, *param.Min),
			})
		}
		if param.Max != nil && num > *param.Max {
			v.addError(ValidationError{
				Field:   param.Name,
				Value:   value,
				Rule:    "max",
				Message: fmt.Sprintf("%s must be at most %v", param.Name, *param.Max),
			})
		}
	}

	// Length constraints for strings
	if str, ok := value.(string); ok {
		if param.MinLength != nil && len(str) < *param.MinLength {
			v.addError(ValidationError{
				Field:   param.Name,
				Value:   value,
				Rule:    "minLength",
				Message: fmt.Sprintf("%s must be at least %d characters", param.Name, *param.MinLength),
			})
		}
		if param.MaxLength != nil && len(str) > *param.MaxLength {
			v.addError(ValidationError{
				Field:   param.Name,
				Value:   value,
				Rule:    "maxLength",
				Message: fmt.Sprintf("%s must be at most %d characters", param.Name, *param.MaxLength),
			})
		}
		if param.Pattern != nil && !param.Pattern.MatchString(str) {
			v.addError(ValidationError{
				Field:   param.Name,
				Value:   value,
				Rule:    "pattern",
				Message: fmt.Sprintf("%s does not match required pattern", param.Name),
			})
		}
	}

	// Length constraints for arrays
	if arr, ok := value.([]any); ok {
		if param.MinLength != nil && len(arr) < *param.MinLength {
			v.addError(ValidationError{
				Field:   param.Name,
				Value:   value,
				Rule:    "minLength",
				Message: fmt.Sprintf("%s must have at least %d items", param.Name, *param.MinLength),
			})
		}
		if param.MaxLength != nil && len(arr) > *param.MaxLength {
			v.addError(ValidationError{
				Field:   param.Name,
				Value:   value,
				Rule:    "maxLength",
				Message: fmt.Sprintf("%s must have at most %d items", param.Name, *param.MaxLength),
			})
		}
	}

	// Enum validation
	if len(param.Enum) > 0 {
		str, _ := value.(string)
		valid := false
		for _, e := range param.Enum {
			if e == str {
				valid = true
				break
			}
		}
		if !valid {
			v.addError(ValidationError{
				Field:      param.Name,
				Value:      value,
				Rule:       "enum",
				Message:    fmt.Sprintf("%s must be one of: %s", param.Name, strings.Join(param.Enum, ", ")),
				Suggestion: fmt.Sprintf("Valid values: %s", strings.Join(param.Enum, ", ")),
			})
		}
	}

	// Custom validation
	if param.Custom != nil {
		if err := param.Custom(value); err != nil {
			v.addError(ValidationError{
				Field:   param.Name,
				Value:   value,
				Rule:    "custom",
				Message: err.Error(),
			})
		}
	}
}

// addError adds a validation error to the list.
func (v *Validator) addError(err ValidationError) {
	v.errors = append(v.errors, err)
}

// toFloat64 converts a value to float64 if possible.
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	}
	return 0, false
}

// uuidRegex matches valid UUIDs.
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// emailRegex matches valid email addresses.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// isValidUUID checks if a string is a valid UUID.
func isValidUUID(s string) bool {
	return uuidRegex.MatchString(s)
}

// isValidEmail checks if a string is a valid email address.
func isValidEmail(s string) bool {
	return emailRegex.MatchString(s)
}

// isValidISO8601 checks if a string is a valid ISO 8601 timestamp.
func isValidISO8601(s string) bool {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, format := range formats {
		if _, err := time.Parse(format, s); err == nil {
			return true
		}
	}
	return false
}

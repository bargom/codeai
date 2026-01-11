// Package validation provides input validation with detailed error messages.
package validation

import (
	"errors"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ValidationError Tests
// =============================================================================

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{
		Field:   "name",
		Message: "is required",
	}
	assert.Equal(t, "name: is required", err.Error())
}

// =============================================================================
// ValidationErrors Tests
// =============================================================================

func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name     string
		errors   []ValidationError
		expected string
	}{
		{
			name:     "single error",
			errors:   []ValidationError{{Field: "name", Message: "is required"}},
			expected: "name: is required",
		},
		{
			name: "multiple errors",
			errors: []ValidationError{
				{Field: "name", Message: "is required"},
				{Field: "email", Message: "must be valid"},
			},
			expected: "name: is required; email: must be valid",
		},
		{
			name:     "empty errors",
			errors:   []ValidationError{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := &ValidationErrors{Errors: tt.errors}
			assert.Equal(t, tt.expected, errs.Error())
		})
	}
}

func TestValidationErrors_HasErrors(t *testing.T) {
	assert.False(t, (&ValidationErrors{}).HasErrors())
	assert.False(t, (&ValidationErrors{Errors: []ValidationError{}}).HasErrors())
	assert.True(t, (&ValidationErrors{Errors: []ValidationError{{}}}).HasErrors())
}

// =============================================================================
// NewValidator Tests
// =============================================================================

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	assert.NotNil(t, v)
}

// =============================================================================
// Required Field Tests
// =============================================================================

func TestValidate_Required(t *testing.T) {
	v := NewValidator()

	t.Run("missing required field", func(t *testing.T) {
		params := []ParamDef{{Name: "name", Required: true}}
		errs := v.Validate(map[string]any{}, params)
		require.NotNil(t, errs)
		assert.Len(t, errs.Errors, 1)
		assert.Equal(t, "name", errs.Errors[0].Field)
		assert.Equal(t, "required", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "required")
		assert.NotEmpty(t, errs.Errors[0].Suggestion)
	})

	t.Run("nil value for required field", func(t *testing.T) {
		params := []ParamDef{{Name: "name", Required: true}}
		errs := v.Validate(map[string]any{"name": nil}, params)
		require.NotNil(t, errs)
		assert.Len(t, errs.Errors, 1)
		assert.Equal(t, "required", errs.Errors[0].Rule)
	})

	t.Run("present required field", func(t *testing.T) {
		params := []ParamDef{{Name: "name", Required: true, Type: "string"}}
		errs := v.Validate(map[string]any{"name": "test"}, params)
		assert.Nil(t, errs)
	})

	t.Run("optional field not present", func(t *testing.T) {
		params := []ParamDef{{Name: "name", Required: false}}
		errs := v.Validate(map[string]any{}, params)
		assert.Nil(t, errs)
	})
}

// =============================================================================
// Type Validation Tests - String
// =============================================================================

func TestValidate_TypeString(t *testing.T) {
	v := NewValidator()
	params := []ParamDef{{Name: "field", Type: "string"}}

	t.Run("valid string", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "hello"}, params)
		assert.Nil(t, errs)
	})

	t.Run("invalid string - number", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": 123}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "type", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "string")
	})

	t.Run("text type alias", func(t *testing.T) {
		textParams := []ParamDef{{Name: "field", Type: "text"}}
		errs := v.Validate(map[string]any{"field": "hello"}, textParams)
		assert.Nil(t, errs)
	})
}

// =============================================================================
// Type Validation Tests - Integer
// =============================================================================

func TestValidate_TypeInteger(t *testing.T) {
	v := NewValidator()
	params := []ParamDef{{Name: "field", Type: "integer"}}

	t.Run("valid integer from float64", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": float64(42)}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid integer from int", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": 42}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid integer from int64", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": int64(42)}, params)
		assert.Nil(t, errs)
	})

	t.Run("invalid integer - decimal float", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": 42.5}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "type", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "integer")
	})

	t.Run("invalid integer - string", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "42"}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "type", errs.Errors[0].Rule)
	})
}

// =============================================================================
// Type Validation Tests - Number/Decimal
// =============================================================================

func TestValidate_TypeNumber(t *testing.T) {
	v := NewValidator()

	t.Run("valid float64", func(t *testing.T) {
		params := []ParamDef{{Name: "field", Type: "number"}}
		errs := v.Validate(map[string]any{"field": 42.5}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid int as number", func(t *testing.T) {
		params := []ParamDef{{Name: "field", Type: "number"}}
		errs := v.Validate(map[string]any{"field": 42}, params)
		assert.Nil(t, errs)
	})

	t.Run("decimal type alias", func(t *testing.T) {
		params := []ParamDef{{Name: "field", Type: "decimal"}}
		errs := v.Validate(map[string]any{"field": 42.5}, params)
		assert.Nil(t, errs)
	})

	t.Run("invalid number - string", func(t *testing.T) {
		params := []ParamDef{{Name: "field", Type: "number"}}
		errs := v.Validate(map[string]any{"field": "42.5"}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "type", errs.Errors[0].Rule)
	})
}

// =============================================================================
// Type Validation Tests - Boolean
// =============================================================================

func TestValidate_TypeBoolean(t *testing.T) {
	v := NewValidator()
	params := []ParamDef{{Name: "field", Type: "boolean"}}

	t.Run("valid boolean true", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": true}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid boolean false", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": false}, params)
		assert.Nil(t, errs)
	})

	t.Run("invalid boolean - string", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "true"}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "type", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "boolean")
	})
}

// =============================================================================
// Type Validation Tests - UUID
// =============================================================================

func TestValidate_TypeUUID(t *testing.T) {
	v := NewValidator()
	params := []ParamDef{{Name: "field", Type: "uuid"}}

	t.Run("valid UUID lowercase", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "550e8400-e29b-41d4-a716-446655440000"}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid UUID uppercase", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "550E8400-E29B-41D4-A716-446655440000"}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid UUID mixed case", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "550e8400-E29B-41d4-A716-446655440000"}, params)
		assert.Nil(t, errs)
	})

	t.Run("invalid UUID - wrong format", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "not-a-uuid"}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "type", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "UUID")
		assert.NotEmpty(t, errs.Errors[0].Suggestion)
	})

	t.Run("invalid UUID - not a string", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": 123}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "type", errs.Errors[0].Rule)
	})
}

// =============================================================================
// Type Validation Tests - Email
// =============================================================================

func TestValidate_TypeEmail(t *testing.T) {
	v := NewValidator()
	params := []ParamDef{{Name: "field", Type: "email"}}

	t.Run("valid email", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "test@example.com"}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid email with subdomain", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "test@mail.example.com"}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid email with plus", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "test+tag@example.com"}, params)
		assert.Nil(t, errs)
	})

	t.Run("invalid email - no at symbol", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "notanemail"}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "type", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "email")
	})

	t.Run("invalid email - not a string", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": 123}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "type", errs.Errors[0].Rule)
	})
}

// =============================================================================
// Type Validation Tests - Timestamp/Datetime
// =============================================================================

func TestValidate_TypeTimestamp(t *testing.T) {
	v := NewValidator()
	params := []ParamDef{{Name: "field", Type: "timestamp"}}

	t.Run("valid RFC3339", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "2024-01-15T10:30:00Z"}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid RFC3339 with timezone", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "2024-01-15T10:30:00+05:00"}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid RFC3339Nano", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "2024-01-15T10:30:00.123456789Z"}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid date only", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "2024-01-15"}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid datetime no timezone", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "2024-01-15T10:30:00"}, params)
		assert.Nil(t, errs)
	})

	t.Run("datetime type alias", func(t *testing.T) {
		datetimeParams := []ParamDef{{Name: "field", Type: "datetime"}}
		errs := v.Validate(map[string]any{"field": "2024-01-15T10:30:00Z"}, datetimeParams)
		assert.Nil(t, errs)
	})

	t.Run("invalid timestamp - wrong format", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "15/01/2024"}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "type", errs.Errors[0].Rule)
		assert.NotEmpty(t, errs.Errors[0].Suggestion)
	})

	t.Run("invalid timestamp - not a string", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": 123}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "type", errs.Errors[0].Rule)
	})
}

// =============================================================================
// Type Validation Tests - Array
// =============================================================================

func TestValidate_TypeArray(t *testing.T) {
	v := NewValidator()
	params := []ParamDef{{Name: "field", Type: "array"}}

	t.Run("valid array", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": []any{1, 2, 3}}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid empty array", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": []any{}}, params)
		assert.Nil(t, errs)
	})

	t.Run("invalid array - not an array", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "not an array"}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "type", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "array")
	})
}

// =============================================================================
// Type Validation Tests - Object
// =============================================================================

func TestValidate_TypeObject(t *testing.T) {
	v := NewValidator()
	params := []ParamDef{{Name: "field", Type: "object"}}

	t.Run("valid object", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": map[string]any{"key": "value"}}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid empty object", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": map[string]any{}}, params)
		assert.Nil(t, errs)
	})

	t.Run("invalid object - not an object", func(t *testing.T) {
		errs := v.Validate(map[string]any{"field": "not an object"}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "type", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "object")
	})
}

// =============================================================================
// Constraint Tests - Min/Max for Numbers
// =============================================================================

func TestValidate_MinMax(t *testing.T) {
	v := NewValidator()

	t.Run("valid within range", func(t *testing.T) {
		min, max := float64(1), float64(10)
		params := []ParamDef{{Name: "field", Type: "number", Min: &min, Max: &max}}
		errs := v.Validate(map[string]any{"field": float64(5)}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid at min boundary", func(t *testing.T) {
		min := float64(1)
		params := []ParamDef{{Name: "field", Type: "number", Min: &min}}
		errs := v.Validate(map[string]any{"field": float64(1)}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid at max boundary", func(t *testing.T) {
		max := float64(10)
		params := []ParamDef{{Name: "field", Type: "number", Max: &max}}
		errs := v.Validate(map[string]any{"field": float64(10)}, params)
		assert.Nil(t, errs)
	})

	t.Run("invalid below min", func(t *testing.T) {
		min := float64(5)
		params := []ParamDef{{Name: "field", Type: "number", Min: &min}}
		errs := v.Validate(map[string]any{"field": float64(3)}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "min", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "at least")
	})

	t.Run("invalid above max", func(t *testing.T) {
		max := float64(5)
		params := []ParamDef{{Name: "field", Type: "number", Max: &max}}
		errs := v.Validate(map[string]any{"field": float64(10)}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "max", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "at most")
	})

	t.Run("min/max with int value", func(t *testing.T) {
		min := float64(1)
		params := []ParamDef{{Name: "field", Type: "integer", Min: &min}}
		errs := v.Validate(map[string]any{"field": 5}, params)
		assert.Nil(t, errs)
	})

	t.Run("min/max with int64 value", func(t *testing.T) {
		min := float64(1)
		params := []ParamDef{{Name: "field", Type: "integer", Min: &min}}
		errs := v.Validate(map[string]any{"field": int64(5)}, params)
		assert.Nil(t, errs)
	})
}

// =============================================================================
// Constraint Tests - MinLength/MaxLength for Strings
// =============================================================================

func TestValidate_StringLength(t *testing.T) {
	v := NewValidator()

	t.Run("valid string within length", func(t *testing.T) {
		minLen, maxLen := 2, 10
		params := []ParamDef{{Name: "field", Type: "string", MinLength: &minLen, MaxLength: &maxLen}}
		errs := v.Validate(map[string]any{"field": "hello"}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid at min length", func(t *testing.T) {
		minLen := 5
		params := []ParamDef{{Name: "field", Type: "string", MinLength: &minLen}}
		errs := v.Validate(map[string]any{"field": "hello"}, params)
		assert.Nil(t, errs)
	})

	t.Run("valid at max length", func(t *testing.T) {
		maxLen := 5
		params := []ParamDef{{Name: "field", Type: "string", MaxLength: &maxLen}}
		errs := v.Validate(map[string]any{"field": "hello"}, params)
		assert.Nil(t, errs)
	})

	t.Run("invalid below min length", func(t *testing.T) {
		minLen := 5
		params := []ParamDef{{Name: "field", Type: "string", MinLength: &minLen}}
		errs := v.Validate(map[string]any{"field": "hi"}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "minLength", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "at least")
	})

	t.Run("invalid above max length", func(t *testing.T) {
		maxLen := 3
		params := []ParamDef{{Name: "field", Type: "string", MaxLength: &maxLen}}
		errs := v.Validate(map[string]any{"field": "hello"}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "maxLength", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "at most")
	})
}

// =============================================================================
// Constraint Tests - MinLength/MaxLength for Arrays
// =============================================================================

func TestValidate_ArrayLength(t *testing.T) {
	v := NewValidator()

	t.Run("valid array within length", func(t *testing.T) {
		minLen, maxLen := 2, 5
		params := []ParamDef{{Name: "field", Type: "array", MinLength: &minLen, MaxLength: &maxLen}}
		errs := v.Validate(map[string]any{"field": []any{1, 2, 3}}, params)
		assert.Nil(t, errs)
	})

	t.Run("invalid array below min length", func(t *testing.T) {
		minLen := 3
		params := []ParamDef{{Name: "field", Type: "array", MinLength: &minLen}}
		errs := v.Validate(map[string]any{"field": []any{1}}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "minLength", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "at least")
		assert.Contains(t, errs.Errors[0].Message, "items")
	})

	t.Run("invalid array above max length", func(t *testing.T) {
		maxLen := 2
		params := []ParamDef{{Name: "field", Type: "array", MaxLength: &maxLen}}
		errs := v.Validate(map[string]any{"field": []any{1, 2, 3, 4, 5}}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "maxLength", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "at most")
		assert.Contains(t, errs.Errors[0].Message, "items")
	})
}

// =============================================================================
// Constraint Tests - Pattern
// =============================================================================

func TestValidate_Pattern(t *testing.T) {
	v := NewValidator()

	t.Run("valid pattern match", func(t *testing.T) {
		pattern := regexp.MustCompile(`^[a-z]+$`)
		params := []ParamDef{{Name: "field", Type: "string", Pattern: pattern}}
		errs := v.Validate(map[string]any{"field": "hello"}, params)
		assert.Nil(t, errs)
	})

	t.Run("invalid pattern mismatch", func(t *testing.T) {
		pattern := regexp.MustCompile(`^[a-z]+$`)
		params := []ParamDef{{Name: "field", Type: "string", Pattern: pattern}}
		errs := v.Validate(map[string]any{"field": "Hello123"}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "pattern", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "pattern")
	})
}

// =============================================================================
// Constraint Tests - Enum
// =============================================================================

func TestValidate_Enum(t *testing.T) {
	v := NewValidator()

	t.Run("valid enum value", func(t *testing.T) {
		params := []ParamDef{{Name: "field", Type: "string", Enum: []string{"active", "inactive", "pending"}}}
		errs := v.Validate(map[string]any{"field": "active"}, params)
		assert.Nil(t, errs)
	})

	t.Run("invalid enum value", func(t *testing.T) {
		params := []ParamDef{{Name: "field", Type: "string", Enum: []string{"active", "inactive", "pending"}}}
		errs := v.Validate(map[string]any{"field": "unknown"}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "enum", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "one of")
		assert.NotEmpty(t, errs.Errors[0].Suggestion)
	})

	t.Run("enum with non-string value", func(t *testing.T) {
		params := []ParamDef{{Name: "field", Type: "string", Enum: []string{"a", "b"}}}
		// Type validation passes, but enum validation should still work
		errs := v.Validate(map[string]any{"field": "c"}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "enum", errs.Errors[0].Rule)
	})
}

// =============================================================================
// Constraint Tests - Custom Validation
// =============================================================================

func TestValidate_Custom(t *testing.T) {
	v := NewValidator()

	t.Run("valid custom validation", func(t *testing.T) {
		customFunc := func(v any) error {
			if s, ok := v.(string); ok && len(s) > 0 {
				return nil
			}
			return errors.New("custom validation failed")
		}
		params := []ParamDef{{Name: "field", Type: "string", Custom: customFunc}}
		errs := v.Validate(map[string]any{"field": "valid"}, params)
		assert.Nil(t, errs)
	})

	t.Run("invalid custom validation", func(t *testing.T) {
		customFunc := func(v any) error {
			return errors.New("custom validation failed")
		}
		params := []ParamDef{{Name: "field", Type: "string", Custom: customFunc}}
		errs := v.Validate(map[string]any{"field": "test"}, params)
		require.NotNil(t, errs)
		assert.Equal(t, "custom", errs.Errors[0].Rule)
		assert.Contains(t, errs.Errors[0].Message, "custom validation failed")
	})
}

// =============================================================================
// Multiple Fields and Errors
// =============================================================================

func TestValidate_MultipleFields(t *testing.T) {
	v := NewValidator()

	params := []ParamDef{
		{Name: "name", Type: "string", Required: true},
		{Name: "email", Type: "email", Required: true},
		{Name: "age", Type: "integer", Required: true},
	}

	t.Run("all valid", func(t *testing.T) {
		data := map[string]any{
			"name":  "John",
			"email": "john@example.com",
			"age":   float64(30),
		}
		errs := v.Validate(data, params)
		assert.Nil(t, errs)
	})

	t.Run("multiple errors", func(t *testing.T) {
		data := map[string]any{
			"name":  123,         // wrong type
			"email": "not-email", // invalid email
			// age missing
		}
		errs := v.Validate(data, params)
		require.NotNil(t, errs)
		assert.Len(t, errs.Errors, 3)
	})
}

// =============================================================================
// Type Error Stops Further Validation
// =============================================================================

func TestValidate_TypeErrorStopsConstraints(t *testing.T) {
	v := NewValidator()

	minLen := 10
	params := []ParamDef{{Name: "field", Type: "string", MinLength: &minLen}}

	// Pass a number instead of string - should only get type error, not minLength error
	errs := v.Validate(map[string]any{"field": 123}, params)
	require.NotNil(t, errs)
	assert.Len(t, errs.Errors, 1)
	assert.Equal(t, "type", errs.Errors[0].Rule)
}

// =============================================================================
// Unknown Type
// =============================================================================

func TestValidate_UnknownType(t *testing.T) {
	v := NewValidator()

	// Unknown type should pass through without type validation
	params := []ParamDef{{Name: "field", Type: "custom_type", Required: true}}
	errs := v.Validate(map[string]any{"field": "anything"}, params)
	assert.Nil(t, errs)
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestToFloat64(t *testing.T) {
	t.Run("float64", func(t *testing.T) {
		v, ok := toFloat64(float64(42.5))
		assert.True(t, ok)
		assert.Equal(t, 42.5, v)
	})

	t.Run("int", func(t *testing.T) {
		v, ok := toFloat64(42)
		assert.True(t, ok)
		assert.Equal(t, float64(42), v)
	})

	t.Run("int64", func(t *testing.T) {
		v, ok := toFloat64(int64(42))
		assert.True(t, ok)
		assert.Equal(t, float64(42), v)
	})

	t.Run("string fails", func(t *testing.T) {
		_, ok := toFloat64("42")
		assert.False(t, ok)
	})

	t.Run("nil fails", func(t *testing.T) {
		_, ok := toFloat64(nil)
		assert.False(t, ok)
	})
}

func TestIsValidUUID(t *testing.T) {
	assert.True(t, isValidUUID("550e8400-e29b-41d4-a716-446655440000"))
	assert.True(t, isValidUUID("550E8400-E29B-41D4-A716-446655440000"))
	assert.False(t, isValidUUID("not-a-uuid"))
	assert.False(t, isValidUUID("550e8400e29b41d4a716446655440000")) // no dashes
	assert.False(t, isValidUUID(""))
}

func TestIsValidEmail(t *testing.T) {
	assert.True(t, isValidEmail("test@example.com"))
	assert.True(t, isValidEmail("test.name@example.com"))
	assert.True(t, isValidEmail("test+tag@example.com"))
	assert.False(t, isValidEmail("not-an-email"))
	assert.False(t, isValidEmail("@example.com"))
	assert.False(t, isValidEmail("test@"))
	assert.False(t, isValidEmail(""))
}

func TestIsValidISO8601(t *testing.T) {
	assert.True(t, isValidISO8601("2024-01-15T10:30:00Z"))
	assert.True(t, isValidISO8601("2024-01-15T10:30:00+05:00"))
	assert.True(t, isValidISO8601("2024-01-15T10:30:00.123456789Z"))
	assert.True(t, isValidISO8601("2024-01-15"))
	assert.True(t, isValidISO8601("2024-01-15T10:30:00"))
	assert.False(t, isValidISO8601("15/01/2024"))
	assert.False(t, isValidISO8601("not-a-date"))
	assert.False(t, isValidISO8601(""))
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestValidate_EmptyData(t *testing.T) {
	v := NewValidator()

	t.Run("empty data no params", func(t *testing.T) {
		errs := v.Validate(map[string]any{}, []ParamDef{})
		assert.Nil(t, errs)
	})

	t.Run("nil data no params", func(t *testing.T) {
		errs := v.Validate(nil, []ParamDef{})
		assert.Nil(t, errs)
	})
}

func TestValidate_DefaultValue(t *testing.T) {
	v := NewValidator()

	// Default values are not automatically applied by Validator
	// They're meant to be used by the caller before validation
	params := []ParamDef{{Name: "field", Type: "string", Default: "default"}}
	errs := v.Validate(map[string]any{}, params)
	assert.Nil(t, errs) // No error because field is not required
}

func TestValidate_CombinedConstraints(t *testing.T) {
	v := NewValidator()

	minLen := 3
	maxLen := 10
	min := float64(1)
	max := float64(100)

	t.Run("string with multiple constraints - all valid", func(t *testing.T) {
		pattern := regexp.MustCompile(`^[a-z]+$`)
		params := []ParamDef{{
			Name:      "field",
			Type:      "string",
			MinLength: &minLen,
			MaxLength: &maxLen,
			Pattern:   pattern,
		}}
		errs := v.Validate(map[string]any{"field": "hello"}, params)
		assert.Nil(t, errs)
	})

	t.Run("string with multiple constraints - multiple failures", func(t *testing.T) {
		pattern := regexp.MustCompile(`^[A-Z]+$`)
		params := []ParamDef{{
			Name:      "field",
			Type:      "string",
			MinLength: &minLen,
			MaxLength: &maxLen,
			Pattern:   pattern,
		}}
		errs := v.Validate(map[string]any{"field": "hi"}, params) // too short and wrong pattern
		require.NotNil(t, errs)
		assert.Len(t, errs.Errors, 2) // minLength and pattern
	})

	t.Run("number with both min and max failure", func(t *testing.T) {
		params := []ParamDef{{
			Name: "field",
			Type: "number",
			Min:  &max, // min=100
			Max:  &min, // max=1 (impossible range, just for testing both errors)
		}}
		errs := v.Validate(map[string]any{"field": float64(50)}, params)
		require.NotNil(t, errs)
		assert.Len(t, errs.Errors, 2) // both min and max fail
	})
}

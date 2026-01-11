# Task: Input Validation System

## Overview
Implement comprehensive input validation with detailed error messages suitable for LLM feedback and user-facing API responses.

## Phase
Phase 2: Core Features

## Priority
High - Critical for security and data integrity.

## Dependencies
- Phase 1: Parser and AST types

## Description
Create a validation system that validates request bodies, query parameters, and path parameters against CodeAI schema definitions, providing detailed, actionable error messages.

## Detailed Requirements

### 1. Validator Types (internal/validation/validator.go)

```go
package validation

import (
    "fmt"
    "regexp"
    "strings"
)

type Validator struct {
    errors []ValidationError
}

type ValidationError struct {
    Field      string `json:"field"`
    Value      any    `json:"value,omitempty"`
    Rule       string `json:"rule"`
    Message    string `json:"message"`
    Suggestion string `json:"suggestion,omitempty"`
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ValidationErrors struct {
    Errors []ValidationError `json:"errors"`
}

func (e *ValidationErrors) Error() string {
    var msgs []string
    for _, err := range e.Errors {
        msgs = append(msgs, err.Error())
    }
    return strings.Join(msgs, "; ")
}

type ParamDef struct {
    Name       string
    Type       string
    Required   bool
    Default    any
    Min        *float64
    Max        *float64
    MinLength  *int
    MaxLength  *int
    Pattern    *regexp.Regexp
    Enum       []string
    Custom     func(any) error
}

func NewValidator() *Validator {
    return &Validator{}
}

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
        if _, ok := value.(string); !ok {
            return &ValidationError{
                Field:      field,
                Value:      value,
                Rule:       "type",
                Message:    fmt.Sprintf("%s must be a valid timestamp", field),
                Suggestion: "Use ISO 8601 format: 2024-01-15T10:30:00Z",
            }
        }
    }
    return nil
}

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

func (v *Validator) addError(err ValidationError) {
    v.errors = append(v.errors, err)
}

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

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidUUID(s string) bool {
    return uuidRegex.MatchString(strings.ToLower(s))
}

func isValidEmail(s string) bool {
    return emailRegex.MatchString(s)
}
```

### 2. Request Validation Middleware

```go
// internal/validation/middleware.go
package validation

import (
    "encoding/json"
    "net/http"
)

func ValidationMiddleware(bodyParams, queryParams []ParamDef) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            validator := NewValidator()

            // Validate query params
            if len(queryParams) > 0 {
                queryData := make(map[string]any)
                for _, p := range queryParams {
                    if v := r.URL.Query().Get(p.Name); v != "" {
                        queryData[p.Name] = v
                    }
                }
                if errs := validator.Validate(queryData, queryParams); errs != nil {
                    writeValidationError(w, errs)
                    return
                }
            }

            // Validate body
            if len(bodyParams) > 0 && r.Body != nil {
                var body map[string]any
                if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
                    writeError(w, http.StatusBadRequest, "invalid JSON body")
                    return
                }
                if errs := validator.Validate(body, bodyParams); errs != nil {
                    writeValidationError(w, errs)
                    return
                }
            }

            next.ServeHTTP(w, r)
        })
    }
}

func writeValidationError(w http.ResponseWriter, errs *ValidationErrors) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusBadRequest)
    json.NewEncoder(w).Encode(map[string]any{
        "error":   "validation failed",
        "details": errs.Errors,
    })
}

func writeError(w http.ResponseWriter, status int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}
```

## Acceptance Criteria
- [ ] Type validation for all CodeAI types
- [ ] Constraint validation (min, max, length, pattern)
- [ ] Enum validation
- [ ] Detailed error messages with suggestions
- [ ] LLM-friendly error format
- [ ] Middleware for HTTP requests

## Testing Strategy
- Unit tests for each validation rule
- Integration tests with HTTP requests
- Error message quality tests

## Files to Create
- `internal/validation/validator.go`
- `internal/validation/middleware.go`
- `internal/validation/validator_test.go`

// Package validation provides input validation with detailed error messages.
package validation

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

// ValidationMiddleware creates middleware that validates request query parameters and body.
// It validates query params first, then body params. If either fails, it returns a 400 error.
func ValidationMiddleware(bodyParams, queryParams []ParamDef) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			validator := NewValidator()

			// Validate query params
			if len(queryParams) > 0 {
				queryData := make(map[string]any)
				for _, p := range queryParams {
					if v := r.URL.Query().Get(p.Name); v != "" {
						// Try to convert to appropriate type based on param definition
						queryData[p.Name] = convertQueryValue(v, p.Type)
					}
				}
				if errs := validator.Validate(queryData, queryParams); errs != nil {
					writeValidationError(w, errs)
					return
				}
			}

			// Validate body
			if len(bodyParams) > 0 && r.Body != nil && r.ContentLength != 0 {
				// Read body to allow re-reading later
				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil {
					writeError(w, http.StatusBadRequest, "failed to read request body")
					return
				}
				r.Body.Close()

				// Restore body for downstream handlers
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				if len(bodyBytes) > 0 {
					var body map[string]any
					if err := json.Unmarshal(bodyBytes, &body); err != nil {
						writeError(w, http.StatusBadRequest, "invalid JSON body")
						return
					}
					if errs := validator.Validate(body, bodyParams); errs != nil {
						writeValidationError(w, errs)
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// convertQueryValue converts a query string value to the appropriate type.
func convertQueryValue(value string, paramType string) any {
	switch paramType {
	case "integer":
		if v, err := strconv.ParseInt(value, 10, 64); err == nil {
			return float64(v) // JSON numbers are always float64
		}
		return value // Return as string, validation will catch type error
	case "decimal", "number":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			return v
		}
		return value
	case "boolean":
		if v, err := strconv.ParseBool(value); err == nil {
			return v
		}
		return value
	default:
		return value
	}
}

// writeValidationError writes a validation error response.
func writeValidationError(w http.ResponseWriter, errs *ValidationErrors) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":   "validation failed",
		"details": errs.Errors,
	})
}

// writeError writes a simple error response.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// ValidateBody is a helper middleware that only validates request body.
func ValidateBody(params []ParamDef) func(http.Handler) http.Handler {
	return ValidationMiddleware(params, nil)
}

// ValidateQuery is a helper middleware that only validates query parameters.
func ValidateQuery(params []ParamDef) func(http.Handler) http.Handler {
	return ValidationMiddleware(nil, params)
}

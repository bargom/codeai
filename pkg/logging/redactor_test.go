package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactor_IsSensitiveField(t *testing.T) {
	r := NewRedactor()

	sensitiveFields := []string{
		"password", "Password", "PASSWORD",
		"secret", "Secret",
		"token", "Token",
		"api_key", "API_KEY", "apikey",
		"authorization",
		"credit_card", "creditcard",
		"cvv",
		"ssn",
	}

	for _, field := range sensitiveFields {
		t.Run(field, func(t *testing.T) {
			assert.True(t, r.IsSensitiveField(field), "field %s should be sensitive", field)
		})
	}

	nonSensitiveFields := []string{
		"username", "email", "name", "id", "status",
	}

	for _, field := range nonSensitiveFields {
		t.Run(field, func(t *testing.T) {
			assert.False(t, r.IsSensitiveField(field), "field %s should not be sensitive", field)
		})
	}
}

func TestRedactor_RedactString(t *testing.T) {
	r := NewRedactor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "password in key-value format",
			input:    `password: mysecret123`,
			expected: RedactedValue,
		},
		{
			name:     "bearer token",
			input:    `Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abc123`,
			expected: `Authorization: ` + RedactedValue,
		},
		{
			name:     "JWT token",
			input:    `token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c`,
			expected: `token: ` + RedactedValue,
		},
		{
			name:     "email address",
			input:    `user email: john.doe@example.com`,
			expected: `user email: ` + RedactedValue,
		},
		{
			name:     "credit card number with spaces",
			input:    `card: 4111 1111 1111 1111`,
			expected: `card: ` + RedactedValue,
		},
		{
			name:     "credit card number with dashes",
			input:    `card: 4111-1111-1111-1111`,
			expected: `card: ` + RedactedValue,
		},
		{
			name:     "AWS access key",
			input:    `aws_key: AKIAIOSFODNN7EXAMPLE`,
			expected: `aws_key: ` + RedactedValue,
		},
		{
			name:     "api key in format",
			input:    `api_key=sk_live_abcdef123456`,
			expected: RedactedValue,
		},
		{
			name:     "secret in format",
			input:    `secret: my-super-secret`,
			expected: RedactedValue,
		},
		{
			name:     "no sensitive data",
			input:    `Hello, this is a normal message`,
			expected: `Hello, this is a normal message`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.RedactString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactor_RedactMap(t *testing.T) {
	r := NewRedactor()

	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "sensitive field at top level",
			input: map[string]any{
				"username": "john",
				"password": "secret123",
			},
			expected: map[string]any{
				"username": "john",
				"password": RedactedValue,
			},
		},
		{
			name: "nested sensitive field",
			input: map[string]any{
				"user": map[string]any{
					"name":  "john",
					"token": "abc123",
				},
			},
			expected: map[string]any{
				"user": map[string]any{
					"name":  "john",
					"token": RedactedValue,
				},
			},
		},
		{
			name: "sensitive pattern in string value",
			input: map[string]any{
				"log": "User logged in with password: secret123",
			},
			expected: map[string]any{
				"log": "User logged in with " + RedactedValue,
			},
		},
		{
			name: "multiple sensitive fields",
			input: map[string]any{
				"api_key":       "key123",
				"secret":        "sec456",
				"authorization": "Bearer token",
				"data":          "normal data",
			},
			expected: map[string]any{
				"api_key":       RedactedValue,
				"secret":        RedactedValue,
				"authorization": RedactedValue,
				"data":          "normal data",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.RedactMap(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactor_AddSensitiveField(t *testing.T) {
	r := NewRedactor()

	// Initially not sensitive
	assert.False(t, r.IsSensitiveField("custom_secret"))

	// Add custom field
	r.AddSensitiveField("custom_secret")

	// Now it should be sensitive
	assert.True(t, r.IsSensitiveField("custom_secret"))
	assert.True(t, r.IsSensitiveField("CUSTOM_SECRET"))
}

func TestRedactor_AddSensitivePattern(t *testing.T) {
	r := NewRedactor()

	// Add custom pattern for license keys
	err := r.AddSensitivePattern(`LICENSE-[A-Z0-9]{4}-[A-Z0-9]{4}`)
	assert.NoError(t, err)

	input := "License key: LICENSE-ABCD-1234"
	result := r.RedactString(input)
	assert.Equal(t, "License key: "+RedactedValue, result)
}

func TestRedactor_AddAllowlistField(t *testing.T) {
	r := NewRedactor()

	// "token" is normally sensitive
	assert.True(t, r.IsSensitiveField("token"))

	// Add to allowlist
	r.AddAllowlistField("token")

	// Now it should not be sensitive
	assert.False(t, r.IsSensitiveField("token"))
}

func TestRedactor_RedactSlice(t *testing.T) {
	r := NewRedactor()

	input := map[string]any{
		"users": []any{
			map[string]any{
				"name":     "user1",
				"password": "pass1",
			},
			map[string]any{
				"name":     "user2",
				"password": "pass2",
			},
		},
	}

	expected := map[string]any{
		"users": []any{
			map[string]any{
				"name":     "user1",
				"password": RedactedValue,
			},
			map[string]any{
				"name":     "user2",
				"password": RedactedValue,
			},
		},
	}

	result := r.RedactMap(input)
	assert.Equal(t, expected, result)
}

func TestRedactor_NilInput(t *testing.T) {
	r := NewRedactor()

	assert.Nil(t, r.RedactMap(nil))
}

func TestGlobalRedactFunctions(t *testing.T) {
	// Test RedactSensitive
	input := map[string]any{
		"password": "secret",
		"name":     "john",
	}
	result := RedactSensitive(input)
	assert.Equal(t, RedactedValue, result["password"])
	assert.Equal(t, "john", result["name"])

	// Test RedactStringValue
	str := "password: mysecret"
	redacted := RedactStringValue(str)
	assert.Equal(t, RedactedValue, redacted)

	// Test IsSensitiveField
	assert.True(t, IsSensitiveField("password"))
	assert.False(t, IsSensitiveField("username"))
}

func TestSafeAttrs(t *testing.T) {
	input := map[string]any{
		"password": "secret",
		"name":     "john",
	}

	attrs := SafeAttrs(input)
	assert.Len(t, attrs, 2)

	// Check that password is redacted
	for _, attr := range attrs {
		if attr.Key == "password" {
			assert.Equal(t, RedactedValue, attr.Value.String())
		}
		if attr.Key == "name" {
			assert.Equal(t, "john", attr.Value.String())
		}
	}
}

func TestRedactor_ConcurrencySafe(t *testing.T) {
	r := NewRedactor()

	done := make(chan bool)

	// Multiple goroutines adding fields
	for i := 0; i < 10; i++ {
		go func(n int) {
			r.AddSensitiveField("custom_field_" + string(rune('a'+n)))
			done <- true
		}(i)
	}

	// Multiple goroutines checking fields
	for i := 0; i < 10; i++ {
		go func() {
			r.IsSensitiveField("password")
			done <- true
		}()
	}

	// Multiple goroutines redacting strings
	for i := 0; i < 10; i++ {
		go func() {
			r.RedactString("password: secret")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 30; i++ {
		<-done
	}
}

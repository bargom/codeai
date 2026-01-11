//go:build integration

package integration

import (
	"testing"

	"github.com/bargom/codeai/internal/parser"
	"github.com/bargom/codeai/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseAndValidate tests the parser and validator working together.
func TestParseAndValidate(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		shouldParse bool
		shouldValid bool
		errContains string
	}{
		{
			name:        "valid simple variable",
			source:      `var x = 1`,
			shouldParse: true,
			shouldValid: true,
		},
		{
			name:        "valid string variable",
			source:      `var name = "hello"`,
			shouldParse: true,
			shouldValid: true,
		},
		{
			name:        "valid boolean variable",
			source:      `var flag = true`,
			shouldParse: true,
			shouldValid: true,
		},
		{
			name:        "valid array literal",
			source:      `var items = [1, 2, 3]`,
			shouldParse: true,
			shouldValid: true,
		},
		{
			name:        "valid multiple variables",
			source:      "var x = 1\nvar y = 2\nvar z = 3",
			shouldParse: true,
			shouldValid: true,
		},
		{
			name:        "valid assignment",
			source:      "var x = 1\nx = 2",
			shouldParse: true,
			shouldValid: true,
		},
		{
			name:        "valid if statement",
			source:      "var flag = true\nif flag { var x = 1 }",
			shouldParse: true,
			shouldValid: true,
		},
		{
			name:        "valid if-else statement",
			source:      "var flag = true\nif flag { var x = 1 } else { var y = 2 }",
			shouldParse: true,
			shouldValid: true,
		},
		{
			name:        "valid for loop",
			source:      "var items = [1, 2, 3]\nfor item in items { var x = item }",
			shouldParse: true,
			shouldValid: true,
		},
		{
			name:        "valid function declaration",
			source:      "function greet(name) { var msg = name }",
			shouldParse: true,
			shouldValid: true,
		},
		{
			name:        "valid function call in expression",
			source:      "function greet(name) { var msg = name }\nvar result = greet(\"test\")",
			shouldParse: true,
			shouldValid: true,
		},
		{
			name:        "valid exec block",
			source:      "exec { echo hello }",
			shouldParse: true,
			shouldValid: true,
		},
		{
			name:        "valid complex program",
			source: `
var apiKey = "sk-12345"
var maxRetries = 3
var debug = true
var users = ["alice", "bob"]

function processUser(user) {
    var status = "active"
}

if debug {
    var logLevel = "verbose"
} else {
    var logLevel = "info"
}

for user in users {
    var current = user
}

exec {
    echo "done"
}
`,
			shouldParse: true,
			shouldValid: true,
		},
		// Parse failures
		{
			name:        "parse fail - incomplete variable",
			source:      `var x =`,
			shouldParse: false,
			shouldValid: false,
		},
		{
			name:        "parse fail - missing brace",
			source:      `if true { var x = 1`,
			shouldParse: false,
			shouldValid: false,
		},
		// Validation failures
		{
			name:        "validate fail - undefined variable",
			source:      `var x = undefinedVar`,
			shouldParse: true,
			shouldValid: false,
			errContains: "undefined",
		},
		{
			name:        "validate fail - duplicate declaration",
			source:      "var x = 1\nvar x = 2",
			shouldParse: true,
			shouldValid: false,
			errContains: "duplicate",
		},
		{
			name:        "validate fail - assign to undeclared",
			source:      `y = 1`,
			shouldParse: true,
			shouldValid: false,
			errContains: "undefined",
		},
		{
			name:        "validate fail - undefined function in expression",
			source:      `var result = unknown()`,
			shouldParse: true,
			shouldValid: false,
			errContains: "undefined",
		},
		{
			name:        "validate fail - wrong argument count",
			source:      "function greet(name) { var msg = name }\nvar result = greet()",
			shouldParse: true,
			shouldValid: false,
			errContains: "argument",
		},
		{
			name:        "validate fail - call non-function",
			source:      "var x = 1\nvar result = x()",
			shouldParse: true,
			shouldValid: false,
			errContains: "not a function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			program, parseErr := parser.Parse(tt.source)

			if !tt.shouldParse {
				assert.Error(t, parseErr, "expected parse error")
				return
			}
			require.NoError(t, parseErr, "unexpected parse error")
			require.NotNil(t, program, "program should not be nil")

			// Validate
			v := validator.New()
			validateErr := v.Validate(program)

			if !tt.shouldValid {
				assert.Error(t, validateErr, "expected validation error")
				if tt.errContains != "" {
					assert.Contains(t, validateErr.Error(), tt.errContains)
				}
				return
			}
			assert.NoError(t, validateErr, "unexpected validation error")
		})
	}
}

// TestParserASTStructure verifies the AST structure is correct.
func TestParserASTStructure(t *testing.T) {
	t.Run("variable declaration has correct structure", func(t *testing.T) {
		program, err := parser.Parse(`var greeting = "hello"`)
		require.NoError(t, err)
		require.Len(t, program.Statements, 1)
		// The statement should be parsed correctly
		assert.NotNil(t, program.Statements[0])
	})

	t.Run("function declaration has correct structure", func(t *testing.T) {
		program, err := parser.Parse(`function test(a, b) { var sum = a }`)
		require.NoError(t, err)
		require.Len(t, program.Statements, 1)
		assert.NotNil(t, program.Statements[0])
	})

	t.Run("for loop has correct structure", func(t *testing.T) {
		program, err := parser.Parse("var arr = [1, 2]\nfor x in arr { var y = x }")
		require.NoError(t, err)
		require.Len(t, program.Statements, 2)
	})

	t.Run("nested blocks work correctly", func(t *testing.T) {
		program, err := parser.Parse(`
var flag = true
if flag {
    var items = [1, 2]
    for item in items {
        var x = item
    }
}
`)
		require.NoError(t, err)
		require.Len(t, program.Statements, 2)
	})
}

// TestValidatorScoping tests scope handling.
func TestValidatorScoping(t *testing.T) {
	t.Run("variable visible in same scope", func(t *testing.T) {
		program, _ := parser.Parse("var x = 1\nvar y = x")
		v := validator.New()
		err := v.Validate(program)
		assert.NoError(t, err)
	})

	t.Run("variable not visible outside block", func(t *testing.T) {
		program, _ := parser.Parse(`
var flag = true
if flag {
    var inner = 1
}
var y = inner
`)
		v := validator.New()
		err := v.Validate(program)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "undefined")
	})

	t.Run("loop variable visible in loop body", func(t *testing.T) {
		program, _ := parser.Parse("var arr = [1, 2]\nfor item in arr { var x = item }")
		v := validator.New()
		err := v.Validate(program)
		assert.NoError(t, err)
	})

	t.Run("function parameter visible in body", func(t *testing.T) {
		program, _ := parser.Parse("function test(param) { var x = param }")
		v := validator.New()
		err := v.Validate(program)
		assert.NoError(t, err)
	})

	t.Run("duplicate parameter names fail", func(t *testing.T) {
		program, _ := parser.Parse("function test(a, a) { var x = a }")
		v := validator.New()
		err := v.Validate(program)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})
}

// TestCommentsAndWhitespace tests comment handling.
func TestCommentsAndWhitespace(t *testing.T) {
	t.Run("single line comments are ignored", func(t *testing.T) {
		program, err := parser.Parse(`
// This is a comment
var x = 1  // inline comment
// Another comment
var y = 2
`)
		require.NoError(t, err)
		require.Len(t, program.Statements, 2)
	})

	t.Run("multi-line comments are ignored", func(t *testing.T) {
		program, err := parser.Parse(`
/* This is a
   multi-line comment */
var x = 1
`)
		require.NoError(t, err)
		require.Len(t, program.Statements, 1)
	})
}

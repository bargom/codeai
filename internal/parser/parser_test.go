package parser

import (
	"os"
	"testing"

	"github.com/bargom/codeai/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVarDecl(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantName string
		wantErr  bool
	}{
		{
			name:     "string variable",
			input:    `var x = "hello"`,
			wantName: "x",
			wantErr:  false,
		},
		{
			name:     "number variable",
			input:    `var count = 42`,
			wantName: "count",
			wantErr:  false,
		},
		{
			name:     "variable with underscore",
			input:    `var my_var = "test"`,
			wantName: "my_var",
			wantErr:  false,
		},
		{
			name:    "missing equals",
			input:   `var x "hello"`,
			wantErr: true,
		},
		{
			name:    "missing value",
			input:   `var x =`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)
			varDecl, ok := program.Statements[0].(*ast.VarDecl)
			require.True(t, ok, "expected VarDecl")
			assert.Equal(t, tt.wantName, varDecl.Name)
		})
	}
}

func TestParseAssignment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantName string
		wantErr  bool
	}{
		{
			name:     "simple assignment",
			input:    `x = 42`,
			wantName: "x",
			wantErr:  false,
		},
		{
			name:     "string assignment",
			input:    `name = "Alice"`,
			wantName: "name",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)
			assign, ok := program.Statements[0].(*ast.Assignment)
			require.True(t, ok, "expected Assignment")
			assert.Equal(t, tt.wantName, assign.Name)
		})
	}
}

func TestParseIfStmt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantElse    bool
		wantBodyLen int
		wantErr     bool
	}{
		{
			name:        "simple if",
			input:       `if active { var x = 1 }`,
			wantElse:    false,
			wantBodyLen: 1,
			wantErr:     false,
		},
		{
			name:        "if with else",
			input:       `if active { var x = 1 } else { var x = 2 }`,
			wantElse:    true,
			wantBodyLen: 1,
			wantErr:     false,
		},
		{
			name:        "if with empty body",
			input:       `if active { }`,
			wantElse:    false,
			wantBodyLen: 0,
			wantErr:     false,
		},
		{
			name:    "missing closing brace",
			input:   `if active { var x = 1`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)
			ifStmt, ok := program.Statements[0].(*ast.IfStmt)
			require.True(t, ok, "expected IfStmt")
			assert.Len(t, ifStmt.Then.Statements, tt.wantBodyLen)
			if tt.wantElse {
				assert.NotNil(t, ifStmt.Else)
			} else {
				assert.Nil(t, ifStmt.Else)
			}
		})
	}
}

func TestParseForLoop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		wantVariable string
		wantBodyLen  int
		wantErr      bool
	}{
		{
			name:         "simple for loop",
			input:        `for item in items { var x = item }`,
			wantVariable: "item",
			wantBodyLen:  1,
			wantErr:      false,
		},
		{
			name:         "for loop with exec",
			input:        `for pod in pods { exec { kubectl delete pod $pod } }`,
			wantVariable: "pod",
			wantBodyLen:  1,
			wantErr:      false,
		},
		{
			name:    "missing in keyword",
			input:   `for item items { }`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)
			forLoop, ok := program.Statements[0].(*ast.ForLoop)
			require.True(t, ok, "expected ForLoop")
			assert.Equal(t, tt.wantVariable, forLoop.Variable)
			assert.Len(t, forLoop.Body.Statements, tt.wantBodyLen)
		})
	}
}

func TestParseFunctionDecl(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantName   string
		wantParams []string
		wantErr    bool
	}{
		{
			name:       "function with no params",
			input:      `function greet() { var msg = "Hello" }`,
			wantName:   "greet",
			wantParams: nil,
			wantErr:    false,
		},
		{
			name:       "function with one param",
			input:      `function greet(name) { var msg = "Hello" }`,
			wantName:   "greet",
			wantParams: []string{"name"},
			wantErr:    false,
		},
		{
			name:       "function with multiple params",
			input:      `function add(a, b, c) { var sum = a }`,
			wantName:   "add",
			wantParams: []string{"a", "b", "c"},
			wantErr:    false,
		},
		{
			name:    "missing parentheses",
			input:   `function greet { }`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)
			funcDecl, ok := program.Statements[0].(*ast.FunctionDecl)
			require.True(t, ok, "expected FunctionDecl")
			assert.Equal(t, tt.wantName, funcDecl.Name)
			paramNames := make([]string, len(funcDecl.Params))
			for i, p := range funcDecl.Params {
				paramNames[i] = p.Name
			}
			if tt.wantParams == nil {
				assert.Empty(t, paramNames)
			} else {
				assert.Equal(t, tt.wantParams, paramNames)
			}
		})
	}
}

func TestParseExecBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantCommand string
		wantErr     bool
	}{
		{
			name:        "simple exec",
			input:       `exec { kubectl get pods }`,
			wantCommand: "kubectl get pods",
			wantErr:     false,
		},
		{
			name:        "exec with variables",
			input:       `exec { echo $name }`,
			wantCommand: "echo $name",
			wantErr:     false,
		},
		{
			name:        "exec with pipes",
			input:       `exec { kubectl get pods | grep Running }`,
			wantCommand: "kubectl get pods | grep Running",
			wantErr:     false,
		},
		{
			name:    "missing closing brace",
			input:   `exec { kubectl get pods`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)
			execBlock, ok := program.Statements[0].(*ast.ExecBlock)
			require.True(t, ok, "expected ExecBlock")
			assert.Equal(t, tt.wantCommand, execBlock.Command)
		})
	}
}

func TestParseExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		checkValue func(t *testing.T, expr ast.Expression)
	}{
		{
			name:  "string literal",
			input: `var x = "hello world"`,
			checkValue: func(t *testing.T, expr ast.Expression) {
				strLit, ok := expr.(*ast.StringLiteral)
				require.True(t, ok, "expected StringLiteral")
				assert.Equal(t, "hello world", strLit.Value)
			},
		},
		{
			name:  "integer literal",
			input: `var x = 42`,
			checkValue: func(t *testing.T, expr ast.Expression) {
				numLit, ok := expr.(*ast.NumberLiteral)
				require.True(t, ok, "expected NumberLiteral")
				assert.Equal(t, 42.0, numLit.Value)
			},
		},
		{
			name:  "float literal",
			input: `var x = 3.14`,
			checkValue: func(t *testing.T, expr ast.Expression) {
				numLit, ok := expr.(*ast.NumberLiteral)
				require.True(t, ok, "expected NumberLiteral")
				assert.Equal(t, 3.14, numLit.Value)
			},
		},
		{
			name:  "identifier",
			input: `var x = other`,
			checkValue: func(t *testing.T, expr ast.Expression) {
				ident, ok := expr.(*ast.Identifier)
				require.True(t, ok, "expected Identifier")
				assert.Equal(t, "other", ident.Name)
			},
		},
		{
			name:  "empty array",
			input: `var x = []`,
			checkValue: func(t *testing.T, expr ast.Expression) {
				arr, ok := expr.(*ast.ArrayLiteral)
				require.True(t, ok, "expected ArrayLiteral")
				assert.Empty(t, arr.Elements)
			},
		},
		{
			name:  "array with elements",
			input: `var x = [1, 2, 3]`,
			checkValue: func(t *testing.T, expr ast.Expression) {
				arr, ok := expr.(*ast.ArrayLiteral)
				require.True(t, ok, "expected ArrayLiteral")
				assert.Len(t, arr.Elements, 3)
			},
		},
		{
			name:  "function call no args",
			input: `var x = greet()`,
			checkValue: func(t *testing.T, expr ast.Expression) {
				call, ok := expr.(*ast.FunctionCall)
				require.True(t, ok, "expected FunctionCall")
				assert.Equal(t, "greet", call.Name)
				assert.Empty(t, call.Args)
			},
		},
		{
			name:  "function call with args",
			input: `var x = add(1, 2)`,
			checkValue: func(t *testing.T, expr ast.Expression) {
				call, ok := expr.(*ast.FunctionCall)
				require.True(t, ok, "expected FunctionCall")
				assert.Equal(t, "add", call.Name)
				assert.Len(t, call.Args, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)
			varDecl, ok := program.Statements[0].(*ast.VarDecl)
			require.True(t, ok, "expected VarDecl")
			tt.checkValue(t, varDecl.Value)
		})
	}
}

func TestParseComplexProgram(t *testing.T) {
	t.Parallel()

	input := `
var name = "test"
var count = 10

function greet(who) {
	var msg = "Hello"
	exec { echo $msg $who }
}

if active {
	for item in items {
		var result = greet(item)
	}
} else {
	var x = 0
}
`

	program, err := Parse(input)
	require.NoError(t, err)
	assert.Len(t, program.Statements, 4) // 2 var decls, 1 function, 1 if
}

func TestParseNestedStructures(t *testing.T) {
	t.Parallel()

	input := `
if outer {
	if inner {
		var x = 1
	}
}
`

	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)
	outerIf, ok := program.Statements[0].(*ast.IfStmt)
	require.True(t, ok, "expected IfStmt")
	require.Len(t, outerIf.Then.Statements, 1)
	_, ok = outerIf.Then.Statements[0].(*ast.IfStmt)
	require.True(t, ok, "expected nested IfStmt")
}

func TestParseComments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantStmtLen int
	}{
		{
			name: "single line comment",
			input: `
// This is a comment
var x = 1
`,
			wantStmtLen: 1,
		},
		{
			name: "multi line comment",
			input: `
/* This is a
   multi-line comment */
var x = 1
`,
			wantStmtLen: 1,
		},
		{
			name: "inline comment",
			input: `
var x = 1 // inline comment
var y = 2
`,
			wantStmtLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			require.NoError(t, err)
			assert.Len(t, program.Statements, tt.wantStmtLen)
		})
	}
}

func TestParseErrorMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "unexpected token",
			input: `var = 1`,
		},
		{
			name:  "unclosed string",
			input: `var x = "hello`,
		},
		{
			name:  "unclosed brace",
			input: `if x {`,
		},
		{
			name:  "unclosed bracket",
			input: `var x = [1, 2`,
		},
		{
			name:  "unclosed paren",
			input: `var x = func(1, 2`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tt.input)
			assert.Error(t, err)
			// Error should contain position information
			assert.Contains(t, err.Error(), ":")
		})
	}
}

func TestParseBooleans(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantBool bool
	}{
		{
			name:     "true literal",
			input:    `var x = true`,
			wantBool: true,
		},
		{
			name:     "false literal",
			input:    `var x = false`,
			wantBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)
			varDecl, ok := program.Statements[0].(*ast.VarDecl)
			require.True(t, ok, "expected VarDecl")
			boolLit, ok := varDecl.Value.(*ast.BoolLiteral)
			require.True(t, ok, "expected BoolLiteral")
			assert.Equal(t, tt.wantBool, boolLit.Value)
		})
	}
}

func TestParseEmptyProgram(t *testing.T) {
	t.Parallel()

	program, err := Parse("")
	require.NoError(t, err)
	assert.Empty(t, program.Statements)
}

func TestParseWhitespaceHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "tabs",
			input: "var\tx\t=\t1",
		},
		{
			name:  "newlines",
			input: "var\nx\n=\n1",
		},
		{
			name:  "mixed whitespace",
			input: "  var   x  =   1  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program, err := Parse(tt.input)
			require.NoError(t, err)
			require.Len(t, program.Statements, 1)
		})
	}
}

func TestParseFile(t *testing.T) {
	t.Parallel()

	t.Run("nonexistent file", func(t *testing.T) {
		t.Parallel()
		_, err := ParseFile("/nonexistent/path/file.cai")
		assert.Error(t, err)
	})

	t.Run("valid file", func(t *testing.T) {
		t.Parallel()
		// Create a temp file
		tmpfile, err := os.CreateTemp("", "test*.cai")
		require.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		content := `var x = "hello"
var y = 42`
		_, err = tmpfile.WriteString(content)
		require.NoError(t, err)
		tmpfile.Close()

		program, err := ParseFile(tmpfile.Name())
		require.NoError(t, err)
		assert.Len(t, program.Statements, 2)
	})

	t.Run("invalid file content", func(t *testing.T) {
		t.Parallel()
		tmpfile, err := os.CreateTemp("", "test*.cai")
		require.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		content := `var = "invalid"`
		_, err = tmpfile.WriteString(content)
		require.NoError(t, err)
		tmpfile.Close()

		_, err = ParseFile(tmpfile.Name())
		assert.Error(t, err)
	})
}

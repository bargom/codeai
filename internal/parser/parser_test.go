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

// =============================================================================
// Config Parsing Tests
// =============================================================================

func TestParseConfigDecl(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		wantDBType     ast.DatabaseType
		wantMongoURI   string
		wantMongoDBName string
		wantErr        bool
	}{
		{
			name:       "config with postgres",
			input:      `config { database_type: "postgres" }`,
			wantDBType: ast.DatabaseTypePostgres,
			wantErr:    false,
		},
		{
			name:           "config with mongodb",
			input:          `config { database_type: "mongodb" mongodb_uri: "mongodb://localhost:27017" mongodb_database: "myapp" }`,
			wantDBType:     ast.DatabaseTypeMongoDB,
			wantMongoURI:   "mongodb://localhost:27017",
			wantMongoDBName: "myapp",
			wantErr:        false,
		},
		{
			name:       "config defaults to postgres",
			input:      `config { }`,
			wantDBType: ast.DatabaseTypePostgres,
			wantErr:    false,
		},
		{
			name:    "config missing brace",
			input:   `config { database_type: "postgres"`,
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
			configDecl, ok := program.Statements[0].(*ast.ConfigDecl)
			require.True(t, ok, "expected ConfigDecl")
			assert.Equal(t, tt.wantDBType, configDecl.DatabaseType)
			if tt.wantMongoURI != "" {
				assert.Equal(t, tt.wantMongoURI, configDecl.MongoDBURI)
			}
			if tt.wantMongoDBName != "" {
				assert.Equal(t, tt.wantMongoDBName, configDecl.MongoDBName)
			}
		})
	}
}

func TestParseDatabaseBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantDBType ast.DatabaseType
		wantErr    bool
	}{
		{
			name:       "postgres database block",
			input:      `database postgres { }`,
			wantDBType: ast.DatabaseTypePostgres,
			wantErr:    false,
		},
		{
			name:       "mongodb database block",
			input:      `database mongodb { }`,
			wantDBType: ast.DatabaseTypeMongoDB,
			wantErr:    false,
		},
		{
			name:    "database missing type",
			input:   `database { }`,
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
			dbBlock, ok := program.Statements[0].(*ast.DatabaseBlock)
			require.True(t, ok, "expected DatabaseBlock")
			assert.Equal(t, tt.wantDBType, dbBlock.DBType)
		})
	}
}

func TestParseConfigWithDatabaseBlock(t *testing.T) {
	t.Parallel()

	input := `
config {
	database_type: "mongodb"
	mongodb_uri: "mongodb://localhost:27017"
	mongodb_database: "testdb"
}

database mongodb {
}
`

	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 2)

	configDecl, ok := program.Statements[0].(*ast.ConfigDecl)
	require.True(t, ok, "expected ConfigDecl as first statement")
	assert.Equal(t, ast.DatabaseTypeMongoDB, configDecl.DatabaseType)

	dbBlock, ok := program.Statements[1].(*ast.DatabaseBlock)
	require.True(t, ok, "expected DatabaseBlock as second statement")
	assert.Equal(t, ast.DatabaseTypeMongoDB, dbBlock.DBType)
}

// =============================================================================
// MongoDB Collection Parsing Tests
// =============================================================================

func TestParseMongoDBCollection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantName    string
		wantDesc    string
		wantFields  int
		wantIndexes int
		wantErr     bool
	}{
		{
			name: "basic collection with primitive fields",
			input: `database mongodb {
	collection User {
		_id: objectid, primary, auto
		email: string, required, unique
		name: string, optional
	}
}`,
			wantName:    "User",
			wantFields:  3,
			wantIndexes: 0,
			wantErr:     false,
		},
		{
			name: "collection with description",
			input: `database mongodb {
	collection Product {
		description: "Product catalog items"
		_id: objectid, primary
		name: string, required
	}
}`,
			wantName:    "Product",
			wantDesc:    "Product catalog items",
			wantFields:  2,
			wantIndexes: 0,
			wantErr:     false,
		},
		{
			name: "collection with indexes",
			input: `database mongodb {
	collection User {
		_id: objectid, primary
		email: string, required, unique
		name: string, required
		indexes {
			index: [email] unique
			index: [name]
		}
	}
}`,
			wantName:    "User",
			wantFields:  3,
			wantIndexes: 2,
			wantErr:     false,
		},
		{
			name: "collection with text index",
			input: `database mongodb {
	collection Article {
		_id: objectid, primary
		title: string, required
		content: string, required
		indexes {
			index: [title, content] text
		}
	}
}`,
			wantName:    "Article",
			wantFields:  3,
			wantIndexes: 1,
			wantErr:     false,
		},
		{
			name: "collection with default values",
			input: `database mongodb {
	collection User {
		_id: objectid, primary
		is_active: bool, default(true)
		count: int, default(0)
	}
}`,
			wantName:   "User",
			wantFields: 3,
			wantErr:    false,
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

			dbBlock, ok := program.Statements[0].(*ast.DatabaseBlock)
			require.True(t, ok, "expected DatabaseBlock")
			assert.Equal(t, ast.DatabaseTypeMongoDB, dbBlock.DBType)
			require.Len(t, dbBlock.Statements, 1)

			collection, ok := dbBlock.Statements[0].(*ast.CollectionDecl)
			require.True(t, ok, "expected CollectionDecl")
			assert.Equal(t, tt.wantName, collection.Name)
			if tt.wantDesc != "" {
				assert.Equal(t, tt.wantDesc, collection.Description)
			}
			assert.Len(t, collection.Fields, tt.wantFields)
			assert.Len(t, collection.Indexes, tt.wantIndexes)
		})
	}
}

func TestParseMongoDBEmbeddedDocuments(t *testing.T) {
	t.Parallel()

	input := `database mongodb {
	collection Address {
		_id: objectid, primary
		street: string, required
		location: embedded {
			type: string, required
			coordinates: array(double), required
		}
	}
}`

	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 1)

	dbBlock := program.Statements[0].(*ast.DatabaseBlock)
	require.Len(t, dbBlock.Statements, 1)

	collection := dbBlock.Statements[0].(*ast.CollectionDecl)
	assert.Equal(t, "Address", collection.Name)
	require.Len(t, collection.Fields, 3)

	// Check the embedded document field
	locationField := collection.Fields[2]
	assert.Equal(t, "location", locationField.Name)
	require.NotNil(t, locationField.FieldType.EmbeddedDoc)
	assert.Len(t, locationField.FieldType.EmbeddedDoc.Fields, 2)
}

func TestParseMongoDBArrayTypes(t *testing.T) {
	t.Parallel()

	input := `database mongodb {
	collection Product {
		_id: objectid, primary
		tags: array(string), optional
		prices: array(double), required
	}
}`

	program, err := Parse(input)
	require.NoError(t, err)

	dbBlock := program.Statements[0].(*ast.DatabaseBlock)
	collection := dbBlock.Statements[0].(*ast.CollectionDecl)

	tagsField := collection.Fields[1]
	assert.Equal(t, "tags", tagsField.Name)
	assert.Equal(t, "array", tagsField.FieldType.Name)
	assert.Equal(t, []string{"string"}, tagsField.FieldType.Params)

	pricesField := collection.Fields[2]
	assert.Equal(t, "prices", pricesField.Name)
	assert.Equal(t, "array", pricesField.FieldType.Name)
	assert.Equal(t, []string{"double"}, pricesField.FieldType.Params)
}

func TestParseMongoDBGeospatialIndex(t *testing.T) {
	t.Parallel()

	input := `database mongodb {
	collection Location {
		_id: objectid, primary
		coordinates: array(double), required
		indexes {
			index: [coordinates] geospatial
		}
	}
}`

	program, err := Parse(input)
	require.NoError(t, err)

	dbBlock := program.Statements[0].(*ast.DatabaseBlock)
	collection := dbBlock.Statements[0].(*ast.CollectionDecl)
	require.Len(t, collection.Indexes, 1)

	geoIndex := collection.Indexes[0]
	assert.Equal(t, []string{"coordinates"}, geoIndex.Fields)
	assert.Equal(t, "geospatial", geoIndex.IndexKind)
}

func TestParseMongoDBCompoundIndex(t *testing.T) {
	t.Parallel()

	input := `database mongodb {
	collection Order {
		_id: objectid, primary
		user_id: objectid, required
		status: string, required
		indexes {
			index: [user_id, status]
		}
	}
}`

	program, err := Parse(input)
	require.NoError(t, err)

	dbBlock := program.Statements[0].(*ast.DatabaseBlock)
	collection := dbBlock.Statements[0].(*ast.CollectionDecl)
	require.Len(t, collection.Indexes, 1)

	compoundIndex := collection.Indexes[0]
	assert.Equal(t, []string{"user_id", "status"}, compoundIndex.Fields)
	assert.False(t, compoundIndex.Unique)
}

func TestParseMixedPostgresAndMongoDB(t *testing.T) {
	t.Parallel()

	input := `
database postgres {
	model User {
		id: uuid, primary, auto
		email: string, required, unique
	}
}

database mongodb {
	collection Log {
		_id: objectid, primary
		message: string, required
	}
}
`

	program, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, program.Statements, 2)

	// First is PostgreSQL
	pgBlock := program.Statements[0].(*ast.DatabaseBlock)
	assert.Equal(t, ast.DatabaseTypePostgres, pgBlock.DBType)
	require.Len(t, pgBlock.Statements, 1)
	model, ok := pgBlock.Statements[0].(*ast.ModelDecl)
	require.True(t, ok)
	assert.Equal(t, "User", model.Name)

	// Second is MongoDB
	mongoBlock := program.Statements[1].(*ast.DatabaseBlock)
	assert.Equal(t, ast.DatabaseTypeMongoDB, mongoBlock.DBType)
	require.Len(t, mongoBlock.Statements, 1)
	collection, ok := mongoBlock.Statements[0].(*ast.CollectionDecl)
	require.True(t, ok)
	assert.Equal(t, "Log", collection.Name)
}

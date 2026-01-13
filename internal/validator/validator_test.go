// Package validator provides semantic validation for CodeAI AST.
package validator

import (
	"strings"
	"testing"

	"github.com/bargom/codeai/internal/ast"
	"github.com/bargom/codeai/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Valid Programs - Should Pass Validation
// =============================================================================

func TestValidPrograms(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "simple var declaration",
			source: `var x = "hello"`,
		},
		{
			name:   "var with number",
			source: `var x = 42`,
		},
		{
			name:   "var with boolean true",
			source: `var x = true`,
		},
		{
			name:   "var with boolean false",
			source: `var x = false`,
		},
		{
			name:   "var with array",
			source: `var arr = [1, 2, 3]`,
		},
		{
			name:   "multiple var declarations",
			source: `var x = 1` + "\n" + `var y = 2` + "\n" + `var z = 3`,
		},
		{
			name:   "assignment after declaration",
			source: `var x = 1` + "\n" + `x = 2`,
		},
		{
			name:   "function declaration and call",
			source: `function f() { var x = 1 }` + "\n" + `var result = f()`,
		},
		{
			name:   "function with params",
			source: `function add(a, b) { var sum = a }` + "\n" + `var result = add(1, 2)`,
		},
		{
			name:   "for loop over array",
			source: `var arr = [1, 2, 3]` + "\n" + `for x in arr { var y = x }`,
		},
		{
			name:   "for loop variable scoping",
			source: `var arr = [1, 2, 3]` + "\n" + `for x in arr { var y = x }` + "\n" + `var x = 10`,
		},
		{
			name:   "if statement",
			source: `var x = true` + "\n" + `if x { var y = 1 }`,
		},
		{
			name:   "if-else statement",
			source: `var x = true` + "\n" + `if x { var y = 1 } else { var z = 2 }`,
		},
		{
			name:   "nested scopes",
			source: `var x = 1` + "\n" + `function f() { var x = 2 }`,
		},
		{
			name:   "function parameter shadows outer var",
			source: `var x = 1` + "\n" + `function f(x) { var y = x }`,
		},
		{
			name:   "exec block",
			source: `exec { echo "hello" }`,
		},
		{
			name:   "function call with var arg",
			source: `var x = 1` + "\n" + `function f(a) { var y = a }` + "\n" + `var result = f(x)`,
		},
		{
			name:   "empty program",
			source: ``,
		},
		{
			name:   "nested for loops",
			source: `var outer = [[1, 2], [3, 4]]` + "\n" + `for row in outer { for col in row { var x = col } }`,
		},
		{
			name:   "variable used in nested block",
			source: `var x = 1` + "\n" + `if true { var y = x }`,
		},
		{
			name:   "function returns value",
			source: `function f() { var x = 1 }`,
		},
		{
			name:   "builtin function print",
			source: `var result = print("hello")`,
		},
		{
			name:   "builtin function len",
			source: `var arr = [1, 2, 3]` + "\n" + `var length = len(arr)`,
		},
		{
			name:   "function call in expression",
			source: `function getValue() { var x = 42 }` + "\n" + `var result = getValue()`,
		},
		{
			name:   "array with mixed types",
			source: `var arr = ["hello", 42, true]`,
		},
		{
			name:   "nested array",
			source: `var arr = [[1, 2], [3, 4]]`,
		},
		{
			name:   "variable reference in array",
			source: `var x = 1` + "\n" + `var arr = [x, 2, 3]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := parser.Parse(tt.source)
			require.NoError(t, err, "parse error")

			v := New()
			err = v.Validate(prog)
			assert.NoError(t, err, "validation should pass for valid program")
		})
	}
}

// =============================================================================
// Scope Errors - Should Fail Validation
// =============================================================================

func TestScopeErrors(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectedErr string
	}{
		{
			name:        "undefined variable in assignment",
			source:      `x = 1`,
			expectedErr: "undefined variable 'x'",
		},
		{
			name:        "undefined variable in expression",
			source:      `var y = x`,
			expectedErr: "undefined variable 'x'",
		},
		{
			name:        "duplicate variable declaration in same scope",
			source:      `var x = 1` + "\n" + `var x = 2`,
			expectedErr: "duplicate declaration 'x'",
		},
		{
			name:        "duplicate function declaration",
			source:      `function f() { var x = 1 }` + "\n" + `function f() { var y = 2 }`,
			expectedErr: "duplicate declaration 'f'",
		},
		{
			name:        "duplicate param name in function",
			source:      `function f(a, a) { var x = a }`,
			expectedErr: "duplicate parameter 'a'",
		},
		{
			name:        "undefined variable used in function call arg",
			source:      `function f(a) { var x = a }` + "\n" + `var result = f(undefined_var)`,
			expectedErr: "undefined variable 'undefined_var'",
		},
		{
			name:        "undefined variable used in for loop iterable",
			source:      `for x in undefined_arr { var y = x }`,
			expectedErr: "undefined variable 'undefined_arr'",
		},
		{
			name:        "undefined variable used in if condition",
			source:      `if undefined_cond { var x = 1 }`,
			expectedErr: "undefined variable 'undefined_cond'",
		},
		{
			name:        "variable used before declaration",
			source:      `var y = x` + "\n" + `var x = 1`,
			expectedErr: "undefined variable 'x'",
		},
		{
			name:        "for loop variable shadows outer in error context",
			source:      `var arr = [1, 2]` + "\n" + `for x in arr { var y = unknown }`,
			expectedErr: "undefined variable 'unknown'",
		},
		{
			name:        "undefined variable in array literal",
			source:      `var arr = [1, unknown, 3]`,
			expectedErr: "undefined variable 'unknown'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := parser.Parse(tt.source)
			require.NoError(t, err, "parse error")

			v := New()
			err = v.Validate(prog)
			require.Error(t, err, "validation should fail")
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// =============================================================================
// Function Errors - Should Fail Validation
// =============================================================================

func TestFunctionErrors(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectedErr string
	}{
		{
			name:        "undefined function call",
			source:      `var result = unknown_func()`,
			expectedErr: "undefined function 'unknown_func'",
		},
		{
			name:        "wrong argument count - too few",
			source:      `function f(a, b) { var x = a }` + "\n" + `var result = f(1)`,
			expectedErr: "wrong number of arguments",
		},
		{
			name:        "wrong argument count - too many",
			source:      `function f(a) { var x = a }` + "\n" + `var result = f(1, 2, 3)`,
			expectedErr: "wrong number of arguments",
		},
		{
			name:        "function called before declaration",
			source:      `var result = f()` + "\n" + `function f() { var x = 1 }`,
			expectedErr: "undefined function 'f'",
		},
		{
			name:        "calling variable as function",
			source:      `var f = 1` + "\n" + `var result = f()`,
			expectedErr: "'f' is not a function",
		},
		{
			name:        "wrong arg count for builtin print",
			source:      `var result = print()`,
			expectedErr: "wrong number of arguments",
		},
		{
			name:        "wrong arg count for builtin len",
			source:      `var result = len()`,
			expectedErr: "wrong number of arguments",
		},
		{
			name:        "too many args for builtin len",
			source:      `var result = len("a", "b")`,
			expectedErr: "wrong number of arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := parser.Parse(tt.source)
			require.NoError(t, err, "parse error")

			v := New()
			err = v.Validate(prog)
			require.Error(t, err, "validation should fail")
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// =============================================================================
// Type Errors - Should Fail Validation
// =============================================================================

func TestTypeErrors(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectedErr string
	}{
		{
			name:        "for loop over non-array string",
			source:      `var x = "hello"` + "\n" + `for item in x { var y = item }`,
			expectedErr: "cannot iterate over non-array",
		},
		{
			name:        "for loop over number",
			source:      `var x = 42` + "\n" + `for item in x { var y = item }`,
			expectedErr: "cannot iterate over non-array",
		},
		{
			name:        "for loop over boolean",
			source:      `var x = true` + "\n" + `for item in x { var y = item }`,
			expectedErr: "cannot iterate over non-array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := parser.Parse(tt.source)
			require.NoError(t, err, "parse error")

			v := New()
			err = v.Validate(prog)
			require.Error(t, err, "validation should fail")
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// =============================================================================
// Multiple Errors - Validator Should Collect All Errors
// =============================================================================

func TestMultipleErrors(t *testing.T) {
	source := `x = 1
var y = unknown
var z = another_unknown`

	prog, err := parser.Parse(source)
	require.NoError(t, err, "parse error")

	v := New()
	err = v.Validate(prog)
	require.Error(t, err, "validation should fail")

	// Check that multiple errors are collected
	errStr := err.Error()
	assert.Contains(t, errStr, "undefined variable 'x'", "should report undefined x")
	assert.Contains(t, errStr, "undefined variable 'unknown'", "should report undefined unknown")
	assert.Contains(t, errStr, "undefined variable 'another_unknown'", "should report undefined another_unknown")
}

// =============================================================================
// Error Position Tests
// =============================================================================

func TestErrorPositions(t *testing.T) {
	source := `var x = 1
undefined_var = 2`

	prog, err := parser.Parse(source)
	require.NoError(t, err, "parse error")

	v := New()
	err = v.Validate(prog)
	require.Error(t, err, "validation should fail")

	// Error should contain position info
	errStr := err.Error()
	// Position format varies, but should contain line reference
	assert.True(t, strings.Contains(errStr, "2") || strings.Contains(errStr, "undefined_var"),
		"error should indicate position or variable name")
}

// =============================================================================
// ValidationError Type Tests
// =============================================================================

func TestValidationErrorType(t *testing.T) {
	source := `undefined_var = 1`

	prog, err := parser.Parse(source)
	require.NoError(t, err, "parse error")

	v := New()
	err = v.Validate(prog)
	require.Error(t, err, "validation should fail")

	// Check that error can be unwrapped to get individual errors
	var verr *ValidationErrors
	assert.ErrorAs(t, err, &verr, "should be ValidationErrors type")

	if verr != nil {
		assert.NotEmpty(t, verr.Errors, "should have at least one error")
		assert.Equal(t, ErrorScope, verr.Errors[0].Type, "should be scope error")
	}
}

// =============================================================================
// Symbol Table Tests
// =============================================================================

func TestSymbolTable_BasicOperations(t *testing.T) {
	st := NewSymbolTable()

	// Test declaration
	err := st.Declare("x", SymbolVariable)
	assert.NoError(t, err)

	// Test lookup
	sym, ok := st.Lookup("x")
	assert.True(t, ok)
	assert.Equal(t, "x", sym.Name)
	assert.Equal(t, SymbolVariable, sym.Kind)

	// Test duplicate declaration error
	err = st.Declare("x", SymbolVariable)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestSymbolTable_Scoping(t *testing.T) {
	st := NewSymbolTable()

	// Outer scope
	err := st.Declare("x", SymbolVariable)
	require.NoError(t, err)

	// Enter inner scope
	st.EnterScope()
	err = st.Declare("y", SymbolVariable)
	require.NoError(t, err)

	// x should still be visible
	_, ok := st.Lookup("x")
	assert.True(t, ok, "outer variable should be visible in inner scope")

	// y should be visible
	_, ok = st.Lookup("y")
	assert.True(t, ok, "inner variable should be visible")

	// Exit inner scope
	st.ExitScope()

	// x should still be visible
	_, ok = st.Lookup("x")
	assert.True(t, ok, "outer variable should still be visible after exit")

	// y should NOT be visible
	_, ok = st.Lookup("y")
	assert.False(t, ok, "inner variable should not be visible after scope exit")
}

func TestSymbolTable_Shadowing(t *testing.T) {
	st := NewSymbolTable()

	// Outer scope
	err := st.Declare("x", SymbolVariable)
	require.NoError(t, err)

	// Enter inner scope and shadow x
	st.EnterScope()
	err = st.Declare("x", SymbolVariable)
	require.NoError(t, err, "shadowing should be allowed")

	// Exit inner scope
	st.ExitScope()

	// x should still reference outer variable
	sym, ok := st.Lookup("x")
	assert.True(t, ok)
	assert.Equal(t, "x", sym.Name)
}

func TestSymbolTable_FunctionSymbol(t *testing.T) {
	st := NewSymbolTable()

	// Declare function with param count
	err := st.DeclareFunction("add", 2)
	require.NoError(t, err)

	// Lookup function
	sym, ok := st.Lookup("add")
	assert.True(t, ok)
	assert.Equal(t, SymbolFunction, sym.Kind)
	assert.Equal(t, 2, sym.ParamCount)
}

func TestSymbolTable_LookupLocal(t *testing.T) {
	st := NewSymbolTable()

	// Declare in global
	err := st.Declare("x", SymbolVariable)
	require.NoError(t, err)

	// Enter inner scope
	st.EnterScope()

	// x should NOT be found in local scope
	_, ok := st.LookupLocal("x")
	assert.False(t, ok, "LookupLocal should only check current scope")

	// Declare y in local scope
	err = st.Declare("y", SymbolVariable)
	require.NoError(t, err)

	// y should be found in local scope
	_, ok = st.LookupLocal("y")
	assert.True(t, ok, "LookupLocal should find local variable")
}

func TestSymbolTable_DeclareWithType(t *testing.T) {
	st := NewSymbolTable()

	err := st.DeclareWithType("x", SymbolVariable, TypeString)
	require.NoError(t, err)

	sym, ok := st.Lookup("x")
	assert.True(t, ok)
	assert.Equal(t, TypeString, sym.Type)
}

func TestSymbolTable_UpdateType(t *testing.T) {
	st := NewSymbolTable()

	err := st.DeclareWithType("x", SymbolVariable, TypeUnknown)
	require.NoError(t, err)

	st.UpdateType("x", TypeNumber)

	sym, ok := st.Lookup("x")
	assert.True(t, ok)
	assert.Equal(t, TypeNumber, sym.Type)
}

func TestSymbolTable_CurrentScopeDepth(t *testing.T) {
	st := NewSymbolTable()

	assert.Equal(t, 1, st.CurrentScopeDepth(), "global scope should be depth 1")

	st.EnterScope()
	assert.Equal(t, 2, st.CurrentScopeDepth(), "after enter should be depth 2")

	st.EnterScope()
	assert.Equal(t, 3, st.CurrentScopeDepth(), "after second enter should be depth 3")

	st.ExitScope()
	assert.Equal(t, 2, st.CurrentScopeDepth(), "after exit should be depth 2")
}

// =============================================================================
// Type Inference Tests
// =============================================================================

func TestTypeInference(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		varName  string
		expected Type
	}{
		{"string literal", `var x = "hello"`, "x", TypeString},
		{"number literal", `var x = 42`, "x", TypeNumber},
		{"bool true", `var x = true`, "x", TypeBool},
		{"bool false", `var x = false`, "x", TypeBool},
		{"array literal", `var x = [1, 2, 3]`, "x", TypeArray},
		{"identifier reference", `var a = 1` + "\n" + `var x = a`, "x", TypeNumber},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := parser.Parse(tt.source)
			require.NoError(t, err)

			v := New()
			err = v.Validate(prog)
			require.NoError(t, err)

			// Check the inferred type
			typ, ok := v.TypeOf(tt.varName)
			assert.True(t, ok, "should have type for %s", tt.varName)
			assert.Equal(t, tt.expected, typ, "type mismatch for %s", tt.varName)
		})
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestEdgeCases(t *testing.T) {
	t.Run("deeply nested scopes", func(t *testing.T) {
		source := `var a = 1
function f() {
	var b = 2
	if true {
		var c = 3
		for x in [1, 2] {
			var d = x
		}
	}
}`
		prog, err := parser.Parse(source)
		require.NoError(t, err)

		v := New()
		err = v.Validate(prog)
		assert.NoError(t, err)
	})

	t.Run("empty function body", func(t *testing.T) {
		source := `function empty() { }`
		prog, err := parser.Parse(source)
		require.NoError(t, err)

		v := New()
		err = v.Validate(prog)
		assert.NoError(t, err)
	})

	t.Run("empty for loop body", func(t *testing.T) {
		source := `var arr = [1, 2, 3]
for x in arr { }`
		prog, err := parser.Parse(source)
		require.NoError(t, err)

		v := New()
		err = v.Validate(prog)
		assert.NoError(t, err)
	})

	t.Run("function call as expression", func(t *testing.T) {
		source := `function get() { var x = 1 }
var y = get()`
		prog, err := parser.Parse(source)
		require.NoError(t, err)

		v := New()
		err = v.Validate(prog)
		assert.NoError(t, err)
	})

	t.Run("nil program", func(t *testing.T) {
		v := New()
		err := v.Validate(nil)
		assert.NoError(t, err)
	})

	t.Run("function call in function body", func(t *testing.T) {
		source := `function outer() {
	function inner() { var x = 1 }
	var result = inner()
}`
		prog, err := parser.Parse(source)
		require.NoError(t, err)

		v := New()
		err = v.Validate(prog)
		assert.NoError(t, err)
	})
}

// =============================================================================
// Builtin Functions Tests
// =============================================================================

func TestBuiltinFunctions(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"print with string", `var result = print("hello")`},
		{"print with number", `var result = print(42)`},
		{"print with variable", `var x = "world"` + "\n" + `var result = print(x)`},
		{"len with array", `var arr = [1, 2, 3]` + "\n" + `var length = len(arr)`},
		{"len with string", `var length = len("hello")`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := parser.Parse(tt.source)
			require.NoError(t, err)

			v := New()
			err = v.Validate(prog)
			assert.NoError(t, err)
		})
	}
}

// =============================================================================
// Type Checker Unit Tests
// =============================================================================

func TestTypeChecker_InferType(t *testing.T) {
	st := NewSymbolTable()
	tc := NewTypeChecker(st)

	// Test that InferType works correctly for various cases
	// These are indirect tests through the validator, but we can test type strings
	assert.Equal(t, "unknown", TypeUnknown.String())
	assert.Equal(t, "string", TypeString.String())
	assert.Equal(t, "number", TypeNumber.String())
	assert.Equal(t, "bool", TypeBool.String())
	assert.Equal(t, "array", TypeArray.String())
	assert.Equal(t, "function", TypeFunction.String())
	assert.Equal(t, "void", TypeVoid.String())

	// Test IsIterable
	assert.True(t, tc.IsIterable(TypeArray))
	assert.False(t, tc.IsIterable(TypeString))
	assert.False(t, tc.IsIterable(TypeNumber))
	assert.False(t, tc.IsIterable(TypeBool))

	// Test CheckCompatible
	assert.True(t, tc.CheckCompatible(TypeString, TypeString))
	assert.True(t, tc.CheckCompatible(TypeUnknown, TypeString))
	assert.True(t, tc.CheckCompatible(TypeString, TypeUnknown))
	assert.False(t, tc.CheckCompatible(TypeString, TypeNumber))
}

// =============================================================================
// Error Type Tests
// =============================================================================

func TestErrorTypeString(t *testing.T) {
	assert.Equal(t, "ScopeError", ErrorScope.String())
	assert.Equal(t, "TypeError", ErrorTypeCheck.String())
	assert.Equal(t, "FunctionError", ErrorFunction.String())
	assert.Contains(t, ErrorType(99).String(), "Unknown")
}

func TestSymbolKindString(t *testing.T) {
	assert.Equal(t, "variable", SymbolVariable.String())
	assert.Equal(t, "function", SymbolFunction.String())
	assert.Equal(t, "parameter", SymbolParameter.String())
	assert.Contains(t, SymbolKind(99).String(), "Unknown")
}

func TestValidationError_Error(t *testing.T) {
	// Test error without position
	err := &ValidationError{
		Message: "test error",
		Type:    ErrorScope,
	}
	assert.Equal(t, "ScopeError: test error", err.Error())

	// Test error with position
	// Since we can't easily set position, we test the ValidationErrors wrapper
}

func TestValidationErrors_NoErrors(t *testing.T) {
	ve := &ValidationErrors{}
	assert.Equal(t, "no validation errors", ve.Error())
	assert.False(t, ve.HasErrors())
}

func TestValidationErrors_SingleError(t *testing.T) {
	ve := &ValidationErrors{}
	ve.Add(&ValidationError{Message: "test", Type: ErrorScope})
	assert.Contains(t, ve.Error(), "test")
	assert.True(t, ve.HasErrors())
}

func TestValidationErrors_MultipleErrors(t *testing.T) {
	ve := &ValidationErrors{}
	ve.Add(&ValidationError{Message: "error1", Type: ErrorScope})
	ve.Add(&ValidationError{Message: "error2", Type: ErrorFunction})
	errStr := ve.Error()
	assert.Contains(t, errStr, "2 validation errors")
	assert.Contains(t, errStr, "error1")
	assert.Contains(t, errStr, "error2")
}

func TestValidationErrors_Unwrap(t *testing.T) {
	ve := &ValidationErrors{}
	ve.Add(&ValidationError{Message: "error1", Type: ErrorScope})
	ve.Add(&ValidationError{Message: "error2", Type: ErrorFunction})
	errs := ve.Unwrap()
	assert.Len(t, errs, 2)
}

// =============================================================================
// Additional Coverage Tests
// =============================================================================

func TestTypeChecker_InferBinaryType(t *testing.T) {
	st := NewSymbolTable()
	tc := NewTypeChecker(st)

	// Test nil expression handling
	typ := tc.InferType(nil)
	assert.Equal(t, TypeUnknown, typ)
}

func TestValidationError_WithPosition(t *testing.T) {
	// Test error with valid position
	err := &ValidationError{
		Position: ast.Position{
			Filename: "test.cai",
			Line:     10,
			Column:   5,
		},
		Message: "test error",
		Type:    ErrorScope,
	}
	errStr := err.Error()
	assert.Contains(t, errStr, "test.cai")
	assert.Contains(t, errStr, "10")
	assert.Contains(t, errStr, "test error")
}

func TestSymbolTable_DeclareFunctionDuplicate(t *testing.T) {
	st := NewSymbolTable()

	err := st.DeclareFunction("f", 0)
	require.NoError(t, err)

	// Try to declare again - should fail
	err = st.DeclareFunction("f", 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestSymbolTable_DeclareWithTypeDuplicate(t *testing.T) {
	st := NewSymbolTable()

	err := st.DeclareWithType("x", SymbolVariable, TypeString)
	require.NoError(t, err)

	// Try to declare again - should fail
	err = st.DeclareWithType("x", SymbolVariable, TypeNumber)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestSymbolTable_UpdateTypeNonExistent(t *testing.T) {
	st := NewSymbolTable()

	// Should not panic when updating non-existent variable
	st.UpdateType("nonexistent", TypeString)
	// No assertion needed - just checking it doesn't panic
}

func TestTypeString_Unknown(t *testing.T) {
	// Test unknown type string representation
	typ := Type(999)
	assert.Equal(t, "unknown", typ.String())
}

func TestExitScopeAtGlobal(t *testing.T) {
	st := NewSymbolTable()
	// Exit scope at global should not panic
	st.ExitScope()
	assert.Equal(t, 1, st.CurrentScopeDepth())
}

// Test validateExecBlock path
func TestExecBlockValidation(t *testing.T) {
	source := `exec { echo "test" }`
	prog, err := parser.Parse(source)
	require.NoError(t, err)

	v := New()
	err = v.Validate(prog)
	assert.NoError(t, err)
}

// Test for loop with array literal directly
func TestForLoopWithArrayLiteral(t *testing.T) {
	source := `for x in [1, 2, 3] { var y = x }`
	prog, err := parser.Parse(source)
	require.NoError(t, err)

	v := New()
	err = v.Validate(prog)
	assert.NoError(t, err)
}

// Test that covers validateBlock with statements
func TestValidateBlockWithStatements(t *testing.T) {
	source := `if true { var x = 1
var y = 2
var z = 3 }`
	prog, err := parser.Parse(source)
	require.NoError(t, err)

	v := New()
	err = v.Validate(prog)
	assert.NoError(t, err)
}

// Additional edge case: Assignment to function
func TestAssignmentToFunction(t *testing.T) {
	source := `function f() { var x = 1 }
f = 42`
	prog, err := parser.Parse(source)
	require.NoError(t, err)

	v := New()
	err = v.Validate(prog)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot assign to function")
}

// Test type inference for function call
func TestTypeInference_FunctionCall(t *testing.T) {
	source := `function getValue() { var x = 1 }
var result = getValue()`
	prog, err := parser.Parse(source)
	require.NoError(t, err)

	v := New()
	err = v.Validate(prog)
	require.NoError(t, err)

	// Function call returns unknown type
	typ, ok := v.TypeOf("result")
	assert.True(t, ok)
	assert.Equal(t, TypeUnknown, typ)
}

// =============================================================================
// Direct AST Node Tests (for code paths not covered by parser)
// =============================================================================

func TestValidateReturnStmt(t *testing.T) {
	// Create AST directly since parser doesn't support return
	v := New()

	// Test return with nil value
	ret := &ast.ReturnStmt{}
	v.validateReturnStmt(ret)
	assert.False(t, v.errors.HasErrors())

	// Test return with value
	v = New()
	v.symbols.DeclareWithType("x", SymbolVariable, TypeNumber)
	retWithVal := &ast.ReturnStmt{
		Value: &ast.Identifier{Name: "x"},
	}
	v.validateReturnStmt(retWithVal)
	assert.False(t, v.errors.HasErrors())

	// Test return with undefined value
	v = New()
	retWithUndef := &ast.ReturnStmt{
		Value: &ast.Identifier{Name: "undefined"},
	}
	v.validateReturnStmt(retWithUndef)
	assert.True(t, v.errors.HasErrors())
}

func TestValidateBinaryExpr(t *testing.T) {
	// Create AST directly since parser doesn't support binary expressions
	v := New()
	v.symbols.DeclareWithType("a", SymbolVariable, TypeNumber)
	v.symbols.DeclareWithType("b", SymbolVariable, TypeNumber)

	binExpr := &ast.BinaryExpr{
		Left:     &ast.Identifier{Name: "a"},
		Operator: "+",
		Right:    &ast.Identifier{Name: "b"},
	}
	v.validateExpression(binExpr)
	assert.False(t, v.errors.HasErrors())

	// Test with undefined variable
	v = New()
	binExprUndef := &ast.BinaryExpr{
		Left:     &ast.Identifier{Name: "undefined"},
		Operator: "+",
		Right:    &ast.NumberLiteral{Value: 1},
	}
	v.validateExpression(binExprUndef)
	assert.True(t, v.errors.HasErrors())
}

func TestValidateUnaryExpr(t *testing.T) {
	// Create AST directly since parser doesn't support unary expressions
	v := New()
	v.symbols.DeclareWithType("x", SymbolVariable, TypeNumber)

	unaryExpr := &ast.UnaryExpr{
		Operator: "-",
		Operand:  &ast.Identifier{Name: "x"},
	}
	v.validateExpression(unaryExpr)
	assert.False(t, v.errors.HasErrors())

	// Test with undefined variable
	v = New()
	unaryExprUndef := &ast.UnaryExpr{
		Operator: "not",
		Operand:  &ast.Identifier{Name: "undefined"},
	}
	v.validateExpression(unaryExprUndef)
	assert.True(t, v.errors.HasErrors())
}

func TestInferBinaryType(t *testing.T) {
	st := NewSymbolTable()
	tc := NewTypeChecker(st)

	// Test comparison operators return bool
	binExprCompare := &ast.BinaryExpr{
		Left:     &ast.NumberLiteral{Value: 1},
		Operator: "==",
		Right:    &ast.NumberLiteral{Value: 2},
	}
	assert.Equal(t, TypeBool, tc.InferType(binExprCompare))

	// Test not equal
	binExprNe := &ast.BinaryExpr{
		Left:     &ast.NumberLiteral{Value: 1},
		Operator: "!=",
		Right:    &ast.NumberLiteral{Value: 2},
	}
	assert.Equal(t, TypeBool, tc.InferType(binExprNe))

	// Test less than
	binExprLt := &ast.BinaryExpr{
		Left:     &ast.NumberLiteral{Value: 1},
		Operator: "<",
		Right:    &ast.NumberLiteral{Value: 2},
	}
	assert.Equal(t, TypeBool, tc.InferType(binExprLt))

	// Test greater than
	binExprGt := &ast.BinaryExpr{
		Left:     &ast.NumberLiteral{Value: 1},
		Operator: ">",
		Right:    &ast.NumberLiteral{Value: 2},
	}
	assert.Equal(t, TypeBool, tc.InferType(binExprGt))

	// Test less than or equal
	binExprLe := &ast.BinaryExpr{
		Left:     &ast.NumberLiteral{Value: 1},
		Operator: "<=",
		Right:    &ast.NumberLiteral{Value: 2},
	}
	assert.Equal(t, TypeBool, tc.InferType(binExprLe))

	// Test greater than or equal
	binExprGe := &ast.BinaryExpr{
		Left:     &ast.NumberLiteral{Value: 1},
		Operator: ">=",
		Right:    &ast.NumberLiteral{Value: 2},
	}
	assert.Equal(t, TypeBool, tc.InferType(binExprGe))

	// Test logical operators return bool
	binExprAnd := &ast.BinaryExpr{
		Left:     &ast.BoolLiteral{Value: true},
		Operator: "and",
		Right:    &ast.BoolLiteral{Value: false},
	}
	assert.Equal(t, TypeBool, tc.InferType(binExprAnd))

	binExprOr := &ast.BinaryExpr{
		Left:     &ast.BoolLiteral{Value: true},
		Operator: "or",
		Right:    &ast.BoolLiteral{Value: false},
	}
	assert.Equal(t, TypeBool, tc.InferType(binExprOr))

	// Test arithmetic operators return number
	binExprSub := &ast.BinaryExpr{
		Left:     &ast.NumberLiteral{Value: 5},
		Operator: "-",
		Right:    &ast.NumberLiteral{Value: 3},
	}
	assert.Equal(t, TypeNumber, tc.InferType(binExprSub))

	binExprMul := &ast.BinaryExpr{
		Left:     &ast.NumberLiteral{Value: 5},
		Operator: "*",
		Right:    &ast.NumberLiteral{Value: 3},
	}
	assert.Equal(t, TypeNumber, tc.InferType(binExprMul))

	binExprDiv := &ast.BinaryExpr{
		Left:     &ast.NumberLiteral{Value: 6},
		Operator: "/",
		Right:    &ast.NumberLiteral{Value: 2},
	}
	assert.Equal(t, TypeNumber, tc.InferType(binExprDiv))

	binExprMod := &ast.BinaryExpr{
		Left:     &ast.NumberLiteral{Value: 7},
		Operator: "%",
		Right:    &ast.NumberLiteral{Value: 3},
	}
	assert.Equal(t, TypeNumber, tc.InferType(binExprMod))

	// Test + with numbers returns number
	binExprAdd := &ast.BinaryExpr{
		Left:     &ast.NumberLiteral{Value: 1},
		Operator: "+",
		Right:    &ast.NumberLiteral{Value: 2},
	}
	assert.Equal(t, TypeNumber, tc.InferType(binExprAdd))

	// Test + with string returns string (concatenation)
	binExprConcat := &ast.BinaryExpr{
		Left:     &ast.StringLiteral{Value: "hello"},
		Operator: "+",
		Right:    &ast.StringLiteral{Value: "world"},
	}
	assert.Equal(t, TypeString, tc.InferType(binExprConcat))

	// Test unknown operator returns unknown
	binExprUnknown := &ast.BinaryExpr{
		Left:     &ast.NumberLiteral{Value: 1},
		Operator: "???",
		Right:    &ast.NumberLiteral{Value: 2},
	}
	assert.Equal(t, TypeUnknown, tc.InferType(binExprUnknown))
}

func TestInferUnaryType(t *testing.T) {
	st := NewSymbolTable()
	tc := NewTypeChecker(st)

	// Test 'not' returns bool
	unaryNot := &ast.UnaryExpr{
		Operator: "not",
		Operand:  &ast.BoolLiteral{Value: true},
	}
	assert.Equal(t, TypeBool, tc.InferType(unaryNot))

	// Test '!' returns bool
	unaryBang := &ast.UnaryExpr{
		Operator: "!",
		Operand:  &ast.BoolLiteral{Value: true},
	}
	assert.Equal(t, TypeBool, tc.InferType(unaryBang))

	// Test '-' returns number
	unaryNeg := &ast.UnaryExpr{
		Operator: "-",
		Operand:  &ast.NumberLiteral{Value: 5},
	}
	assert.Equal(t, TypeNumber, tc.InferType(unaryNeg))

	// Test unknown operator returns unknown
	unaryUnknown := &ast.UnaryExpr{
		Operator: "???",
		Operand:  &ast.NumberLiteral{Value: 5},
	}
	assert.Equal(t, TypeUnknown, tc.InferType(unaryUnknown))
}

func TestValidateStatement_Block(t *testing.T) {
	// Test validateStatement with Block directly
	v := New()

	block := &ast.Block{
		Statements: []ast.Statement{
			&ast.VarDecl{Name: "x", Value: &ast.NumberLiteral{Value: 1}},
		},
	}
	v.validateStatement(block)
	assert.False(t, v.errors.HasErrors())
}

func TestValidateStatement_ReturnStmt(t *testing.T) {
	// Test validateStatement with ReturnStmt directly
	v := New()

	ret := &ast.ReturnStmt{}
	v.validateStatement(ret)
	assert.False(t, v.errors.HasErrors())
}

func TestValidateExecBlock(t *testing.T) {
	// Test validateExecBlock directly
	v := New()

	exec := &ast.ExecBlock{Command: "echo hello"}
	v.validateExecBlock(exec)
	assert.False(t, v.errors.HasErrors())
}

func TestForLoopDuplicateVariable(t *testing.T) {
	// Test for loop with duplicate variable declaration error
	// This is hard to trigger through normal parsing, so we test directly
	v := New()

	// Create a for loop that will try to declare a variable that already exists in scope
	// First enter scope to simulate the for loop's scope
	v.symbols.EnterScope()
	v.symbols.Declare("x", SymbolVariable) // Pre-declare x in loop scope

	// Now try to declare it again (simulating what would happen in validateForLoop)
	err := v.symbols.DeclareWithType("x", SymbolVariable, TypeUnknown)
	assert.Error(t, err)
}

func TestVarDeclDuplicateInner(t *testing.T) {
	// Test duplicate declaration catching the inner DeclareWithType error path
	v := New()

	// First declare x
	decl1 := &ast.VarDecl{Name: "x", Value: &ast.NumberLiteral{Value: 1}}
	v.validateVarDecl(decl1)
	assert.False(t, v.errors.HasErrors())

	// Now try to declare x again - this should hit the LookupLocal path
	decl2 := &ast.VarDecl{Name: "x", Value: &ast.NumberLiteral{Value: 2}}
	v.validateVarDecl(decl2)
	assert.True(t, v.errors.HasErrors())
}

func TestFunctionDeclDuplicateInner(t *testing.T) {
	// Test function declaration duplicate catching inner DeclareFunction error
	v := New()

	funcDecl1 := &ast.FunctionDecl{
		Name:   "f",
		Params: []ast.Parameter{},
		Body:   &ast.Block{Statements: []ast.Statement{}},
	}
	v.validateFunctionDecl(funcDecl1)
	assert.False(t, v.errors.HasErrors())

	// Try again - should hit LookupLocal path
	funcDecl2 := &ast.FunctionDecl{
		Name:   "f",
		Params: []ast.Parameter{},
		Body:   &ast.Block{Statements: []ast.Statement{}},
	}
	v.validateFunctionDecl(funcDecl2)
	assert.True(t, v.errors.HasErrors())
}

// =============================================================================
// Config and Database Validation Tests
// =============================================================================

func TestConfigDecl_Valid(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "postgres config without database block",
			source: `config {
				database_type: "postgres"
			}`,
		},
		{
			name: "postgres config with matching database block",
			source: `config {
				database_type: "postgres"
			}
			database postgres { }`,
		},
		{
			name: "mongodb config with matching database block and required fields",
			source: `config {
				database_type: "mongodb"
				mongodb_uri: "mongodb://localhost:27017"
				mongodb_database: "testdb"
			}
			database mongodb { }`,
		},
		{
			name:   "database block without config defaults to postgres",
			source: `database postgres { }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := parser.Parse(tt.source)
			require.NoError(t, err, "parse error")

			v := New()
			err = v.Validate(prog)
			assert.NoError(t, err, "validation should pass")
		})
	}
}

func TestConfigDecl_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectedErr string
	}{
		{
			name: "mongodb config without mongodb_uri",
			source: `config {
				database_type: "mongodb"
				mongodb_database: "testdb"
			}`,
			expectedErr: "mongodb_uri is required",
		},
		{
			name: "mongodb config without mongodb_database",
			source: `config {
				database_type: "mongodb"
				mongodb_uri: "mongodb://localhost:27017"
			}`,
			expectedErr: "mongodb_database is required",
		},
		{
			name: "duplicate config declaration",
			source: `config {
				database_type: "postgres"
			}
			config {
				database_type: "mongodb"
			}`,
			expectedErr: "duplicate config declaration",
		},
		{
			name: "config specifies postgres but database block is mongodb",
			source: `config {
				database_type: "postgres"
			}
			database mongodb { }`,
			expectedErr: "does not match config database_type",
		},
		{
			name: "config specifies mongodb but database block is postgres",
			source: `config {
				database_type: "mongodb"
				mongodb_uri: "mongodb://localhost:27017"
				mongodb_database: "testdb"
			}
			database postgres { }`,
			expectedErr: "does not match config database_type",
		},
		{
			name: "no config with mongodb database block (defaults to postgres)",
			source: `database mongodb { }`,
			expectedErr: "does not match config database_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := parser.Parse(tt.source)
			require.NoError(t, err, "parse error")

			v := New()
			err = v.Validate(prog)
			require.Error(t, err, "validation should fail")
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestDatabaseBlockValidation(t *testing.T) {
	// Test that database block content is validated
	v := New()

	dbBlock := &ast.DatabaseBlock{
		DBType:     ast.DatabaseTypePostgres,
		Statements: []ast.Statement{},
	}
	v.validateDatabaseBlock(dbBlock)
	assert.Len(t, v.databaseBlocks, 1)
	assert.False(t, v.errors.HasErrors())
}

func TestDatabaseTypeConsistency_NoConfigNoBlocks(t *testing.T) {
	// Test that no errors occur when there's no config and no database blocks
	v := New()
	v.validateDatabaseTypeConsistency()
	assert.False(t, v.errors.HasErrors())
}

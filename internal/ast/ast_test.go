package ast

import (
	"testing"
)

// =============================================================================
// Position Tests
// =============================================================================

func TestPosition(t *testing.T) {
	t.Run("creates valid position", func(t *testing.T) {
		pos := Position{
			Filename: "test.cai",
			Line:     10,
			Column:   5,
			Offset:   100,
		}

		if pos.Filename != "test.cai" {
			t.Errorf("expected filename 'test.cai', got '%s'", pos.Filename)
		}
		if pos.Line != 10 {
			t.Errorf("expected line 10, got %d", pos.Line)
		}
		if pos.Column != 5 {
			t.Errorf("expected column 5, got %d", pos.Column)
		}
		if pos.Offset != 100 {
			t.Errorf("expected offset 100, got %d", pos.Offset)
		}
	})

	t.Run("String() output", func(t *testing.T) {
		pos := Position{
			Filename: "test.cai",
			Line:     10,
			Column:   5,
		}

		expected := "test.cai:10:5"
		if pos.String() != expected {
			t.Errorf("expected '%s', got '%s'", expected, pos.String())
		}
	})

	t.Run("IsValid returns true for valid position", func(t *testing.T) {
		pos := Position{
			Filename: "test.cai",
			Line:     1,
			Column:   1,
		}

		if !pos.IsValid() {
			t.Error("expected position to be valid")
		}
	})

	t.Run("IsValid returns false for zero position", func(t *testing.T) {
		pos := Position{}

		if pos.IsValid() {
			t.Error("expected zero position to be invalid")
		}
	})
}

// =============================================================================
// Program Node Tests
// =============================================================================

func TestProgram(t *testing.T) {
	t.Run("creates valid program node", func(t *testing.T) {
		prog := &Program{
			Statements: []Statement{},
		}

		if prog.Type() != NodeProgram {
			t.Errorf("expected NodeProgram, got %v", prog.Type())
		}
	})

	t.Run("String() output", func(t *testing.T) {
		prog := &Program{
			Statements: []Statement{},
		}

		str := prog.String()
		if str == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("Pos() returns correct position", func(t *testing.T) {
		pos := Position{Filename: "main.cai", Line: 1, Column: 1}
		prog := &Program{
			pos:        pos,
			Statements: []Statement{},
		}

		if prog.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, prog.Pos())
		}
	})

	t.Run("with statements", func(t *testing.T) {
		varDecl := &VarDecl{
			Name:  "x",
			Value: &NumberLiteral{Value: 42},
		}
		prog := &Program{
			Statements: []Statement{varDecl},
		}

		if len(prog.Statements) != 1 {
			t.Errorf("expected 1 statement, got %d", len(prog.Statements))
		}
	})
}

// =============================================================================
// VarDecl Tests
// =============================================================================

func TestVarDecl(t *testing.T) {
	t.Run("creates valid node", func(t *testing.T) {
		decl := &VarDecl{
			Name:  "myVar",
			Value: &NumberLiteral{Value: 42},
		}

		if decl.Type() != NodeVarDecl {
			t.Errorf("expected NodeVarDecl, got %v", decl.Type())
		}
		if decl.Name != "myVar" {
			t.Errorf("expected name 'myVar', got '%s'", decl.Name)
		}
	})

	t.Run("String() output", func(t *testing.T) {
		decl := &VarDecl{
			Name:  "myVar",
			Value: &NumberLiteral{Value: 42},
		}

		str := decl.String()
		if str == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "test.cai", Line: 5, Column: 3}
		decl := &VarDecl{
			pos:   pos,
			Name:  "x",
			Value: &NumberLiteral{Value: 1},
		}

		if decl.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, decl.Pos())
		}
	})

	t.Run("with string value", func(t *testing.T) {
		decl := &VarDecl{
			Name:  "greeting",
			Value: &StringLiteral{Value: "hello"},
		}

		if decl.Name != "greeting" {
			t.Errorf("expected name 'greeting', got '%s'", decl.Name)
		}
	})
}

// =============================================================================
// Assignment Tests
// =============================================================================

func TestAssignment(t *testing.T) {
	t.Run("creates valid node", func(t *testing.T) {
		assign := &Assignment{
			Name:  "x",
			Value: &NumberLiteral{Value: 100},
		}

		if assign.Type() != NodeAssignment {
			t.Errorf("expected NodeAssignment, got %v", assign.Type())
		}
	})

	t.Run("String() output", func(t *testing.T) {
		assign := &Assignment{
			Name:  "x",
			Value: &NumberLiteral{Value: 100},
		}

		str := assign.String()
		if str == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "test.cai", Line: 10, Column: 1}
		assign := &Assignment{
			pos:   pos,
			Name:  "count",
			Value: &NumberLiteral{Value: 0},
		}

		if assign.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, assign.Pos())
		}
	})
}

// =============================================================================
// IfStmt Tests
// =============================================================================

func TestIfStmt(t *testing.T) {
	t.Run("creates valid node", func(t *testing.T) {
		ifStmt := &IfStmt{
			Condition: &Identifier{Name: "flag"},
			Then:      &Block{Statements: []Statement{}},
		}

		if ifStmt.Type() != NodeIfStmt {
			t.Errorf("expected NodeIfStmt, got %v", ifStmt.Type())
		}
	})

	t.Run("String() output", func(t *testing.T) {
		ifStmt := &IfStmt{
			Condition: &Identifier{Name: "flag"},
			Then:      &Block{Statements: []Statement{}},
		}

		str := ifStmt.String()
		if str == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("with else block", func(t *testing.T) {
		ifStmt := &IfStmt{
			Condition: &Identifier{Name: "flag"},
			Then:      &Block{Statements: []Statement{}},
			Else:      &Block{Statements: []Statement{}},
		}

		if ifStmt.Else == nil {
			t.Error("expected else block to be non-nil")
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "test.cai", Line: 15, Column: 1}
		ifStmt := &IfStmt{
			pos:       pos,
			Condition: &Identifier{Name: "flag"},
			Then:      &Block{Statements: []Statement{}},
		}

		if ifStmt.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, ifStmt.Pos())
		}
	})
}

// =============================================================================
// ForLoop Tests
// =============================================================================

func TestForLoop(t *testing.T) {
	t.Run("creates valid node", func(t *testing.T) {
		forLoop := &ForLoop{
			Variable: "item",
			Iterable: &Identifier{Name: "items"},
			Body:     &Block{Statements: []Statement{}},
		}

		if forLoop.Type() != NodeForLoop {
			t.Errorf("expected NodeForLoop, got %v", forLoop.Type())
		}
	})

	t.Run("String() output", func(t *testing.T) {
		forLoop := &ForLoop{
			Variable: "item",
			Iterable: &Identifier{Name: "items"},
			Body:     &Block{Statements: []Statement{}},
		}

		str := forLoop.String()
		if str == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "test.cai", Line: 20, Column: 1}
		forLoop := &ForLoop{
			pos:      pos,
			Variable: "i",
			Iterable: &Identifier{Name: "list"},
			Body:     &Block{Statements: []Statement{}},
		}

		if forLoop.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, forLoop.Pos())
		}
	})
}

// =============================================================================
// FunctionDecl Tests
// =============================================================================

func TestFunctionDecl(t *testing.T) {
	t.Run("creates valid node", func(t *testing.T) {
		fn := &FunctionDecl{
			Name:   "add",
			Params: []Parameter{{Name: "a"}, {Name: "b"}},
			Body:   &Block{Statements: []Statement{}},
		}

		if fn.Type() != NodeFunctionDecl {
			t.Errorf("expected NodeFunctionDecl, got %v", fn.Type())
		}
		if fn.Name != "add" {
			t.Errorf("expected name 'add', got '%s'", fn.Name)
		}
	})

	t.Run("String() output", func(t *testing.T) {
		fn := &FunctionDecl{
			Name:   "greet",
			Params: []Parameter{{Name: "name"}},
			Body:   &Block{Statements: []Statement{}},
		}

		str := fn.String()
		if str == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("with no parameters", func(t *testing.T) {
		fn := &FunctionDecl{
			Name:   "noParams",
			Params: []Parameter{},
			Body:   &Block{Statements: []Statement{}},
		}

		if len(fn.Params) != 0 {
			t.Errorf("expected 0 params, got %d", len(fn.Params))
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "funcs.cai", Line: 1, Column: 1}
		fn := &FunctionDecl{
			pos:    pos,
			Name:   "test",
			Params: []Parameter{},
			Body:   &Block{Statements: []Statement{}},
		}

		if fn.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, fn.Pos())
		}
	})
}

// =============================================================================
// ExecBlock Tests
// =============================================================================

func TestExecBlock(t *testing.T) {
	t.Run("creates valid node", func(t *testing.T) {
		exec := &ExecBlock{
			Command: "echo hello",
		}

		if exec.Type() != NodeExecBlock {
			t.Errorf("expected NodeExecBlock, got %v", exec.Type())
		}
	})

	t.Run("String() output", func(t *testing.T) {
		exec := &ExecBlock{
			Command: "ls -la",
		}

		str := exec.String()
		if str == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "script.cai", Line: 5, Column: 1}
		exec := &ExecBlock{
			pos:     pos,
			Command: "date",
		}

		if exec.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, exec.Pos())
		}
	})
}

// =============================================================================
// StringLiteral Tests
// =============================================================================

func TestStringLiteral(t *testing.T) {
	t.Run("creates valid node", func(t *testing.T) {
		str := &StringLiteral{Value: "hello world"}

		if str.Type() != NodeStringLiteral {
			t.Errorf("expected NodeStringLiteral, got %v", str.Type())
		}
		if str.Value != "hello world" {
			t.Errorf("expected 'hello world', got '%s'", str.Value)
		}
	})

	t.Run("String() output", func(t *testing.T) {
		lit := &StringLiteral{Value: "test"}

		result := lit.String()
		if result == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("empty string", func(t *testing.T) {
		str := &StringLiteral{Value: ""}

		if str.Value != "" {
			t.Errorf("expected empty string, got '%s'", str.Value)
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "test.cai", Line: 3, Column: 10}
		str := &StringLiteral{
			pos:   pos,
			Value: "test",
		}

		if str.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, str.Pos())
		}
	})
}

// =============================================================================
// NumberLiteral Tests
// =============================================================================

func TestNumberLiteral(t *testing.T) {
	t.Run("creates valid integer node", func(t *testing.T) {
		num := &NumberLiteral{Value: 42}

		if num.Type() != NodeNumberLiteral {
			t.Errorf("expected NodeNumberLiteral, got %v", num.Type())
		}
		if num.Value != 42 {
			t.Errorf("expected 42, got %v", num.Value)
		}
	})

	t.Run("creates valid float node", func(t *testing.T) {
		num := &NumberLiteral{Value: 3.14}

		if num.Value != 3.14 {
			t.Errorf("expected 3.14, got %v", num.Value)
		}
	})

	t.Run("String() output", func(t *testing.T) {
		num := &NumberLiteral{Value: 100}

		result := num.String()
		if result == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("zero value", func(t *testing.T) {
		num := &NumberLiteral{Value: 0}

		if num.Value != 0 {
			t.Errorf("expected 0, got %v", num.Value)
		}
	})

	t.Run("negative value", func(t *testing.T) {
		num := &NumberLiteral{Value: -42}

		if num.Value != -42 {
			t.Errorf("expected -42, got %v", num.Value)
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "test.cai", Line: 7, Column: 5}
		num := &NumberLiteral{
			pos:   pos,
			Value: 123,
		}

		if num.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, num.Pos())
		}
	})
}

// =============================================================================
// BoolLiteral Tests
// =============================================================================

func TestBoolLiteral(t *testing.T) {
	t.Run("creates true value", func(t *testing.T) {
		b := &BoolLiteral{Value: true}

		if b.Type() != NodeBoolLiteral {
			t.Errorf("expected NodeBoolLiteral, got %v", b.Type())
		}
		if !b.Value {
			t.Error("expected true")
		}
	})

	t.Run("creates false value", func(t *testing.T) {
		b := &BoolLiteral{Value: false}

		if b.Value {
			t.Error("expected false")
		}
	})

	t.Run("String() output", func(t *testing.T) {
		b := &BoolLiteral{Value: true}

		result := b.String()
		if result == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "test.cai", Line: 1, Column: 1}
		b := &BoolLiteral{
			pos:   pos,
			Value: true,
		}

		if b.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, b.Pos())
		}
	})
}

// =============================================================================
// Identifier Tests
// =============================================================================

func TestIdentifier(t *testing.T) {
	t.Run("creates valid node", func(t *testing.T) {
		id := &Identifier{Name: "myVar"}

		if id.Type() != NodeIdentifier {
			t.Errorf("expected NodeIdentifier, got %v", id.Type())
		}
		if id.Name != "myVar" {
			t.Errorf("expected 'myVar', got '%s'", id.Name)
		}
	})

	t.Run("String() output", func(t *testing.T) {
		id := &Identifier{Name: "test"}

		result := id.String()
		if result == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "test.cai", Line: 2, Column: 8}
		id := &Identifier{
			pos:  pos,
			Name: "var1",
		}

		if id.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, id.Pos())
		}
	})
}

// =============================================================================
// FunctionCall Tests
// =============================================================================

func TestFunctionCall(t *testing.T) {
	t.Run("creates valid node", func(t *testing.T) {
		call := &FunctionCall{
			Name: "print",
			Args: []Expression{&StringLiteral{Value: "hello"}},
		}

		if call.Type() != NodeFunctionCall {
			t.Errorf("expected NodeFunctionCall, got %v", call.Type())
		}
		if call.Name != "print" {
			t.Errorf("expected 'print', got '%s'", call.Name)
		}
	})

	t.Run("String() output", func(t *testing.T) {
		call := &FunctionCall{
			Name: "add",
			Args: []Expression{
				&NumberLiteral{Value: 1},
				&NumberLiteral{Value: 2},
			},
		}

		result := call.String()
		if result == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("no arguments", func(t *testing.T) {
		call := &FunctionCall{
			Name: "now",
			Args: []Expression{},
		}

		if len(call.Args) != 0 {
			t.Errorf("expected 0 args, got %d", len(call.Args))
		}
	})

	t.Run("multiple arguments", func(t *testing.T) {
		call := &FunctionCall{
			Name: "max",
			Args: []Expression{
				&NumberLiteral{Value: 1},
				&NumberLiteral{Value: 2},
				&NumberLiteral{Value: 3},
			},
		}

		if len(call.Args) != 3 {
			t.Errorf("expected 3 args, got %d", len(call.Args))
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "test.cai", Line: 10, Column: 5}
		call := &FunctionCall{
			pos:  pos,
			Name: "test",
			Args: []Expression{},
		}

		if call.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, call.Pos())
		}
	})
}

// =============================================================================
// ArrayLiteral Tests
// =============================================================================

func TestArrayLiteral(t *testing.T) {
	t.Run("creates valid node", func(t *testing.T) {
		arr := &ArrayLiteral{
			Elements: []Expression{
				&NumberLiteral{Value: 1},
				&NumberLiteral{Value: 2},
				&NumberLiteral{Value: 3},
			},
		}

		if arr.Type() != NodeArrayLiteral {
			t.Errorf("expected NodeArrayLiteral, got %v", arr.Type())
		}
	})

	t.Run("String() output", func(t *testing.T) {
		arr := &ArrayLiteral{
			Elements: []Expression{
				&StringLiteral{Value: "a"},
				&StringLiteral{Value: "b"},
			},
		}

		result := arr.String()
		if result == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("empty array", func(t *testing.T) {
		arr := &ArrayLiteral{
			Elements: []Expression{},
		}

		if len(arr.Elements) != 0 {
			t.Errorf("expected 0 elements, got %d", len(arr.Elements))
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "test.cai", Line: 5, Column: 1}
		arr := &ArrayLiteral{
			pos:      pos,
			Elements: []Expression{},
		}

		if arr.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, arr.Pos())
		}
	})
}

// =============================================================================
// Block Tests
// =============================================================================

func TestBlock(t *testing.T) {
	t.Run("creates valid node", func(t *testing.T) {
		block := &Block{
			Statements: []Statement{},
		}

		if block.Type() != NodeBlock {
			t.Errorf("expected NodeBlock, got %v", block.Type())
		}
	})

	t.Run("String() output", func(t *testing.T) {
		block := &Block{
			Statements: []Statement{
				&VarDecl{Name: "x", Value: &NumberLiteral{Value: 1}},
			},
		}

		result := block.String()
		if result == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("with multiple statements", func(t *testing.T) {
		block := &Block{
			Statements: []Statement{
				&VarDecl{Name: "x", Value: &NumberLiteral{Value: 1}},
				&Assignment{Name: "x", Value: &NumberLiteral{Value: 2}},
			},
		}

		if len(block.Statements) != 2 {
			t.Errorf("expected 2 statements, got %d", len(block.Statements))
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "test.cai", Line: 3, Column: 1}
		block := &Block{
			pos:        pos,
			Statements: []Statement{},
		}

		if block.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, block.Pos())
		}
	})
}

// =============================================================================
// ReturnStmt Tests
// =============================================================================

func TestReturnStmt(t *testing.T) {
	t.Run("creates valid node with value", func(t *testing.T) {
		ret := &ReturnStmt{
			Value: &NumberLiteral{Value: 42},
		}

		if ret.Type() != NodeReturnStmt {
			t.Errorf("expected NodeReturnStmt, got %v", ret.Type())
		}
	})

	t.Run("creates valid node without value", func(t *testing.T) {
		ret := &ReturnStmt{}

		if ret.Value != nil {
			t.Error("expected nil value")
		}
	})

	t.Run("String() output", func(t *testing.T) {
		ret := &ReturnStmt{
			Value: &StringLiteral{Value: "done"},
		}

		result := ret.String()
		if result == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "test.cai", Line: 20, Column: 5}
		ret := &ReturnStmt{
			pos:   pos,
			Value: &NumberLiteral{Value: 0},
		}

		if ret.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, ret.Pos())
		}
	})
}

// =============================================================================
// BinaryExpr Tests
// =============================================================================

func TestBinaryExpr(t *testing.T) {
	t.Run("creates valid addition node", func(t *testing.T) {
		expr := &BinaryExpr{
			Left:     &NumberLiteral{Value: 1},
			Operator: "+",
			Right:    &NumberLiteral{Value: 2},
		}

		if expr.Type() != NodeBinaryExpr {
			t.Errorf("expected NodeBinaryExpr, got %v", expr.Type())
		}
		if expr.Operator != "+" {
			t.Errorf("expected '+', got '%s'", expr.Operator)
		}
	})

	t.Run("String() output", func(t *testing.T) {
		expr := &BinaryExpr{
			Left:     &Identifier{Name: "a"},
			Operator: "==",
			Right:    &Identifier{Name: "b"},
		}

		result := expr.String()
		if result == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("various operators", func(t *testing.T) {
		operators := []string{"+", "-", "*", "/", "==", "!=", "<", ">", "<=", ">=", "and", "or"}

		for _, op := range operators {
			expr := &BinaryExpr{
				Left:     &NumberLiteral{Value: 1},
				Operator: op,
				Right:    &NumberLiteral{Value: 2},
			}

			if expr.Operator != op {
				t.Errorf("expected operator '%s', got '%s'", op, expr.Operator)
			}
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "test.cai", Line: 5, Column: 10}
		expr := &BinaryExpr{
			pos:      pos,
			Left:     &NumberLiteral{Value: 1},
			Operator: "+",
			Right:    &NumberLiteral{Value: 2},
		}

		if expr.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, expr.Pos())
		}
	})
}

// =============================================================================
// UnaryExpr Tests
// =============================================================================

func TestUnaryExpr(t *testing.T) {
	t.Run("creates valid negation node", func(t *testing.T) {
		expr := &UnaryExpr{
			Operator: "-",
			Operand:  &NumberLiteral{Value: 5},
		}

		if expr.Type() != NodeUnaryExpr {
			t.Errorf("expected NodeUnaryExpr, got %v", expr.Type())
		}
	})

	t.Run("creates valid not node", func(t *testing.T) {
		expr := &UnaryExpr{
			Operator: "not",
			Operand:  &BoolLiteral{Value: true},
		}

		if expr.Operator != "not" {
			t.Errorf("expected 'not', got '%s'", expr.Operator)
		}
	})

	t.Run("String() output", func(t *testing.T) {
		expr := &UnaryExpr{
			Operator: "-",
			Operand:  &Identifier{Name: "x"},
		}

		result := expr.String()
		if result == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("Position tracking", func(t *testing.T) {
		pos := Position{Filename: "test.cai", Line: 8, Column: 3}
		expr := &UnaryExpr{
			pos:      pos,
			Operator: "not",
			Operand:  &BoolLiteral{Value: false},
		}

		if expr.Pos() != pos {
			t.Errorf("expected position %v, got %v", pos, expr.Pos())
		}
	})
}

// =============================================================================
// Parameter Tests
// =============================================================================

func TestParameter(t *testing.T) {
	t.Run("creates valid parameter", func(t *testing.T) {
		param := Parameter{
			Name: "x",
			Type: "integer",
		}

		if param.Name != "x" {
			t.Errorf("expected name 'x', got '%s'", param.Name)
		}
		if param.Type != "integer" {
			t.Errorf("expected type 'integer', got '%s'", param.Type)
		}
	})

	t.Run("with default value", func(t *testing.T) {
		defaultVal := &NumberLiteral{Value: 0}
		param := Parameter{
			Name:    "count",
			Type:    "integer",
			Default: defaultVal,
		}

		if param.Default == nil {
			t.Error("expected default value")
		}
	})
}

// =============================================================================
// NodeType Tests
// =============================================================================

func TestNodeType(t *testing.T) {
	t.Run("String() method", func(t *testing.T) {
		tests := []struct {
			nodeType NodeType
			expected string
		}{
			{NodeProgram, "Program"},
			{NodeVarDecl, "VarDecl"},
			{NodeAssignment, "Assignment"},
			{NodeIfStmt, "IfStmt"},
			{NodeForLoop, "ForLoop"},
			{NodeFunctionDecl, "FunctionDecl"},
			{NodeExecBlock, "ExecBlock"},
			{NodeStringLiteral, "StringLiteral"},
			{NodeNumberLiteral, "NumberLiteral"},
			{NodeBoolLiteral, "BoolLiteral"},
			{NodeIdentifier, "Identifier"},
			{NodeFunctionCall, "FunctionCall"},
			{NodeArrayLiteral, "ArrayLiteral"},
			{NodeBlock, "Block"},
			{NodeReturnStmt, "ReturnStmt"},
			{NodeBinaryExpr, "BinaryExpr"},
			{NodeUnaryExpr, "UnaryExpr"},
		}

		for _, tt := range tests {
			result := tt.nodeType.String()
			if result != tt.expected {
				t.Errorf("NodeType %d: expected '%s', got '%s'", tt.nodeType, tt.expected, result)
			}
		}
	})
}

// =============================================================================
// Walk Function Tests
// =============================================================================

func TestWalk(t *testing.T) {
	t.Run("visits all nodes in program", func(t *testing.T) {
		prog := &Program{
			Statements: []Statement{
				&VarDecl{
					Name:  "x",
					Value: &NumberLiteral{Value: 42},
				},
			},
		}

		visited := []NodeType{}
		visitor := func(node Node) bool {
			visited = append(visited, node.Type())
			return true
		}

		Walk(prog, visitor)

		if len(visited) < 3 {
			t.Errorf("expected at least 3 nodes visited, got %d", len(visited))
		}
	})

	t.Run("stops when visitor returns false", func(t *testing.T) {
		prog := &Program{
			Statements: []Statement{
				&VarDecl{Name: "x", Value: &NumberLiteral{Value: 1}},
				&VarDecl{Name: "y", Value: &NumberLiteral{Value: 2}},
			},
		}

		count := 0
		visitor := func(node Node) bool {
			count++
			return count < 2
		}

		Walk(prog, visitor)

		if count != 2 {
			t.Errorf("expected visitor to stop after 2 nodes, got %d", count)
		}
	})

	t.Run("visits nested blocks", func(t *testing.T) {
		ifStmt := &IfStmt{
			Condition: &BoolLiteral{Value: true},
			Then: &Block{
				Statements: []Statement{
					&VarDecl{Name: "inner", Value: &NumberLiteral{Value: 1}},
				},
			},
		}

		prog := &Program{
			Statements: []Statement{ifStmt},
		}

		foundInner := false
		visitor := func(node Node) bool {
			if v, ok := node.(*VarDecl); ok && v.Name == "inner" {
				foundInner = true
			}
			return true
		}

		Walk(prog, visitor)

		if !foundInner {
			t.Error("expected to find nested VarDecl 'inner'")
		}
	})
}

// =============================================================================
// Print Function Tests
// =============================================================================

func TestPrint(t *testing.T) {
	t.Run("prints program", func(t *testing.T) {
		prog := &Program{
			Statements: []Statement{
				&VarDecl{Name: "x", Value: &NumberLiteral{Value: 42}},
			},
		}

		result := Print(prog)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("prints nested structure", func(t *testing.T) {
		prog := &Program{
			Statements: []Statement{
				&IfStmt{
					Condition: &BoolLiteral{Value: true},
					Then: &Block{
						Statements: []Statement{
							&VarDecl{Name: "y", Value: &StringLiteral{Value: "hello"}},
						},
					},
				},
			},
		}

		result := Print(prog)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})
}

// =============================================================================
// Equal Function Tests
// =============================================================================

func TestEqual(t *testing.T) {
	t.Run("equal number literals", func(t *testing.T) {
		a := &NumberLiteral{Value: 42}
		b := &NumberLiteral{Value: 42}

		if !Equal(a, b) {
			t.Error("expected literals to be equal")
		}
	})

	t.Run("unequal number literals", func(t *testing.T) {
		a := &NumberLiteral{Value: 42}
		b := &NumberLiteral{Value: 43}

		if Equal(a, b) {
			t.Error("expected literals to be unequal")
		}
	})

	t.Run("equal string literals", func(t *testing.T) {
		a := &StringLiteral{Value: "hello"}
		b := &StringLiteral{Value: "hello"}

		if !Equal(a, b) {
			t.Error("expected literals to be equal")
		}
	})

	t.Run("different node types", func(t *testing.T) {
		a := &NumberLiteral{Value: 42}
		b := &StringLiteral{Value: "42"}

		if Equal(a, b) {
			t.Error("expected different types to be unequal")
		}
	})

	t.Run("equal identifiers", func(t *testing.T) {
		a := &Identifier{Name: "x"}
		b := &Identifier{Name: "x"}

		if !Equal(a, b) {
			t.Error("expected identifiers to be equal")
		}
	})

	t.Run("equal var declarations", func(t *testing.T) {
		a := &VarDecl{Name: "x", Value: &NumberLiteral{Value: 1}}
		b := &VarDecl{Name: "x", Value: &NumberLiteral{Value: 1}}

		if !Equal(a, b) {
			t.Error("expected VarDecls to be equal")
		}
	})

	t.Run("unequal var declarations - different names", func(t *testing.T) {
		a := &VarDecl{Name: "x", Value: &NumberLiteral{Value: 1}}
		b := &VarDecl{Name: "y", Value: &NumberLiteral{Value: 1}}

		if Equal(a, b) {
			t.Error("expected VarDecls to be unequal")
		}
	})

	t.Run("nil nodes", func(t *testing.T) {
		if !Equal(nil, nil) {
			t.Error("expected nil == nil to be true")
		}

		a := &NumberLiteral{Value: 1}
		if Equal(a, nil) {
			t.Error("expected non-nil != nil")
		}
		if Equal(nil, a) {
			t.Error("expected nil != non-nil")
		}
	})

	t.Run("equal arrays", func(t *testing.T) {
		a := &ArrayLiteral{
			Elements: []Expression{
				&NumberLiteral{Value: 1},
				&NumberLiteral{Value: 2},
			},
		}
		b := &ArrayLiteral{
			Elements: []Expression{
				&NumberLiteral{Value: 1},
				&NumberLiteral{Value: 2},
			},
		}

		if !Equal(a, b) {
			t.Error("expected arrays to be equal")
		}
	})

	t.Run("unequal arrays - different lengths", func(t *testing.T) {
		a := &ArrayLiteral{
			Elements: []Expression{
				&NumberLiteral{Value: 1},
			},
		}
		b := &ArrayLiteral{
			Elements: []Expression{
				&NumberLiteral{Value: 1},
				&NumberLiteral{Value: 2},
			},
		}

		if Equal(a, b) {
			t.Error("expected arrays to be unequal")
		}
	})

	t.Run("equal function calls", func(t *testing.T) {
		a := &FunctionCall{
			Name: "add",
			Args: []Expression{
				&NumberLiteral{Value: 1},
				&NumberLiteral{Value: 2},
			},
		}
		b := &FunctionCall{
			Name: "add",
			Args: []Expression{
				&NumberLiteral{Value: 1},
				&NumberLiteral{Value: 2},
			},
		}

		if !Equal(a, b) {
			t.Error("expected function calls to be equal")
		}
	})

	t.Run("equal binary expressions", func(t *testing.T) {
		a := &BinaryExpr{
			Left:     &NumberLiteral{Value: 1},
			Operator: "+",
			Right:    &NumberLiteral{Value: 2},
		}
		b := &BinaryExpr{
			Left:     &NumberLiteral{Value: 1},
			Operator: "+",
			Right:    &NumberLiteral{Value: 2},
		}

		if !Equal(a, b) {
			t.Error("expected binary expressions to be equal")
		}
	})
}

// =============================================================================
// Clone Function Tests
// =============================================================================

func TestClone(t *testing.T) {
	t.Run("clones number literal", func(t *testing.T) {
		original := &NumberLiteral{Value: 42}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}

		// Modify original and ensure clone is not affected
		original.Value = 100
		if cloned.(*NumberLiteral).Value != 42 {
			t.Error("clone should be independent of original")
		}
	})

	t.Run("clones string literal", func(t *testing.T) {
		original := &StringLiteral{Value: "hello"}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})

	t.Run("clones var decl", func(t *testing.T) {
		original := &VarDecl{
			Name:  "x",
			Value: &NumberLiteral{Value: 42},
		}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}

		// Modify original and ensure clone is independent
		original.Name = "y"
		if cloned.(*VarDecl).Name != "x" {
			t.Error("clone should be independent of original")
		}
	})

	t.Run("clones array literal deeply", func(t *testing.T) {
		original := &ArrayLiteral{
			Elements: []Expression{
				&NumberLiteral{Value: 1},
				&NumberLiteral{Value: 2},
			},
		}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}

		// Modify original element
		original.Elements[0].(*NumberLiteral).Value = 100
		if cloned.(*ArrayLiteral).Elements[0].(*NumberLiteral).Value != 1 {
			t.Error("deep clone should be independent")
		}
	})

	t.Run("clones nil returns nil", func(t *testing.T) {
		cloned := Clone(nil)
		if cloned != nil {
			t.Error("clone of nil should be nil")
		}
	})

	t.Run("clones program with statements", func(t *testing.T) {
		original := &Program{
			Statements: []Statement{
				&VarDecl{Name: "x", Value: &NumberLiteral{Value: 1}},
				&Assignment{Name: "x", Value: &NumberLiteral{Value: 2}},
			},
		}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned program should equal original")
		}
	})

	t.Run("preserves position", func(t *testing.T) {
		pos := Position{Filename: "test.cai", Line: 10, Column: 5}
		original := &NumberLiteral{
			pos:   pos,
			Value: 42,
		}
		cloned := Clone(original)

		if cloned.Pos() != pos {
			t.Error("clone should preserve position")
		}
	})

	t.Run("clones bool literal", func(t *testing.T) {
		original := &BoolLiteral{Value: true}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})

	t.Run("clones identifier", func(t *testing.T) {
		original := &Identifier{Name: "myVar"}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})

	t.Run("clones function call", func(t *testing.T) {
		original := &FunctionCall{
			Name: "add",
			Args: []Expression{
				&NumberLiteral{Value: 1},
				&NumberLiteral{Value: 2},
			},
		}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})

	t.Run("clones binary expr", func(t *testing.T) {
		original := &BinaryExpr{
			Left:     &NumberLiteral{Value: 1},
			Operator: "+",
			Right:    &NumberLiteral{Value: 2},
		}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})

	t.Run("clones unary expr", func(t *testing.T) {
		original := &UnaryExpr{
			Operator: "-",
			Operand:  &NumberLiteral{Value: 5},
		}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})

	t.Run("clones if stmt", func(t *testing.T) {
		original := &IfStmt{
			Condition: &BoolLiteral{Value: true},
			Then:      &Block{Statements: []Statement{}},
			Else:      &Block{Statements: []Statement{}},
		}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})

	t.Run("clones if stmt without else", func(t *testing.T) {
		original := &IfStmt{
			Condition: &BoolLiteral{Value: true},
			Then:      &Block{Statements: []Statement{}},
		}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})

	t.Run("clones for loop", func(t *testing.T) {
		original := &ForLoop{
			Variable: "item",
			Iterable: &Identifier{Name: "items"},
			Body:     &Block{Statements: []Statement{}},
		}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})

	t.Run("clones function decl", func(t *testing.T) {
		original := &FunctionDecl{
			Name: "test",
			Params: []Parameter{
				{Name: "x", Type: "int"},
			},
			Body: &Block{Statements: []Statement{}},
		}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})

	t.Run("clones function decl with default params", func(t *testing.T) {
		original := &FunctionDecl{
			Name: "test",
			Params: []Parameter{
				{Name: "x", Type: "int", Default: &NumberLiteral{Value: 0}},
			},
			Body: &Block{Statements: []Statement{}},
		}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})

	t.Run("clones exec block", func(t *testing.T) {
		original := &ExecBlock{Command: "ls -la"}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})

	t.Run("clones block", func(t *testing.T) {
		original := &Block{
			Statements: []Statement{
				&VarDecl{Name: "x", Value: &NumberLiteral{Value: 1}},
			},
		}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})

	t.Run("clones return stmt with value", func(t *testing.T) {
		original := &ReturnStmt{Value: &NumberLiteral{Value: 42}}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})

	t.Run("clones return stmt without value", func(t *testing.T) {
		original := &ReturnStmt{}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})

	t.Run("clones assignment", func(t *testing.T) {
		original := &Assignment{Name: "x", Value: &NumberLiteral{Value: 42}}
		cloned := Clone(original)

		if !Equal(original, cloned) {
			t.Error("cloned node should equal original")
		}
	})
}

// =============================================================================
// Additional Walk Tests for Coverage
// =============================================================================

func TestWalkAllNodeTypes(t *testing.T) {
	t.Run("walks assignment", func(t *testing.T) {
		assign := &Assignment{Name: "x", Value: &NumberLiteral{Value: 1}}
		count := 0
		Walk(assign, func(n Node) bool {
			count++
			return true
		})
		if count != 2 { // Assignment + NumberLiteral
			t.Errorf("expected 2 nodes, got %d", count)
		}
	})

	t.Run("walks if with else", func(t *testing.T) {
		ifStmt := &IfStmt{
			Condition: &BoolLiteral{Value: true},
			Then:      &Block{Statements: []Statement{}},
			Else:      &Block{Statements: []Statement{}},
		}
		count := 0
		Walk(ifStmt, func(n Node) bool {
			count++
			return true
		})
		if count != 4 { // IfStmt + BoolLiteral + Then Block + Else Block
			t.Errorf("expected 4 nodes, got %d", count)
		}
	})

	t.Run("walks for loop", func(t *testing.T) {
		forLoop := &ForLoop{
			Variable: "item",
			Iterable: &Identifier{Name: "items"},
			Body:     &Block{Statements: []Statement{}},
		}
		count := 0
		Walk(forLoop, func(n Node) bool {
			count++
			return true
		})
		if count != 3 { // ForLoop + Identifier + Block
			t.Errorf("expected 3 nodes, got %d", count)
		}
	})

	t.Run("walks function decl with default param", func(t *testing.T) {
		fn := &FunctionDecl{
			Name: "test",
			Params: []Parameter{
				{Name: "x", Default: &NumberLiteral{Value: 0}},
			},
			Body: &Block{Statements: []Statement{}},
		}
		count := 0
		Walk(fn, func(n Node) bool {
			count++
			return true
		})
		if count != 3 { // FunctionDecl + NumberLiteral (default) + Block
			t.Errorf("expected 3 nodes, got %d", count)
		}
	})

	t.Run("walks exec block", func(t *testing.T) {
		exec := &ExecBlock{Command: "ls"}
		count := 0
		Walk(exec, func(n Node) bool {
			count++
			return true
		})
		if count != 1 { // Just ExecBlock
			t.Errorf("expected 1 node, got %d", count)
		}
	})

	t.Run("walks return stmt", func(t *testing.T) {
		ret := &ReturnStmt{Value: &NumberLiteral{Value: 42}}
		count := 0
		Walk(ret, func(n Node) bool {
			count++
			return true
		})
		if count != 2 { // ReturnStmt + NumberLiteral
			t.Errorf("expected 2 nodes, got %d", count)
		}
	})

	t.Run("walks nil return", func(t *testing.T) {
		ret := &ReturnStmt{}
		count := 0
		Walk(ret, func(n Node) bool {
			count++
			return true
		})
		if count != 1 { // Just ReturnStmt
			t.Errorf("expected 1 node, got %d", count)
		}
	})

	t.Run("walks string literal", func(t *testing.T) {
		str := &StringLiteral{Value: "hello"}
		count := 0
		Walk(str, func(n Node) bool {
			count++
			return true
		})
		if count != 1 {
			t.Errorf("expected 1 node, got %d", count)
		}
	})

	t.Run("walks number literal", func(t *testing.T) {
		num := &NumberLiteral{Value: 42}
		count := 0
		Walk(num, func(n Node) bool {
			count++
			return true
		})
		if count != 1 {
			t.Errorf("expected 1 node, got %d", count)
		}
	})

	t.Run("walks bool literal", func(t *testing.T) {
		b := &BoolLiteral{Value: true}
		count := 0
		Walk(b, func(n Node) bool {
			count++
			return true
		})
		if count != 1 {
			t.Errorf("expected 1 node, got %d", count)
		}
	})

	t.Run("walks identifier", func(t *testing.T) {
		id := &Identifier{Name: "x"}
		count := 0
		Walk(id, func(n Node) bool {
			count++
			return true
		})
		if count != 1 {
			t.Errorf("expected 1 node, got %d", count)
		}
	})

	t.Run("walks function call", func(t *testing.T) {
		call := &FunctionCall{
			Name: "add",
			Args: []Expression{
				&NumberLiteral{Value: 1},
				&NumberLiteral{Value: 2},
			},
		}
		count := 0
		Walk(call, func(n Node) bool {
			count++
			return true
		})
		if count != 3 { // FunctionCall + 2 args
			t.Errorf("expected 3 nodes, got %d", count)
		}
	})

	t.Run("walks array literal", func(t *testing.T) {
		arr := &ArrayLiteral{
			Elements: []Expression{
				&NumberLiteral{Value: 1},
				&NumberLiteral{Value: 2},
			},
		}
		count := 0
		Walk(arr, func(n Node) bool {
			count++
			return true
		})
		if count != 3 { // ArrayLiteral + 2 elements
			t.Errorf("expected 3 nodes, got %d", count)
		}
	})

	t.Run("walks binary expr", func(t *testing.T) {
		expr := &BinaryExpr{
			Left:     &NumberLiteral{Value: 1},
			Operator: "+",
			Right:    &NumberLiteral{Value: 2},
		}
		count := 0
		Walk(expr, func(n Node) bool {
			count++
			return true
		})
		if count != 3 { // BinaryExpr + Left + Right
			t.Errorf("expected 3 nodes, got %d", count)
		}
	})

	t.Run("walks unary expr", func(t *testing.T) {
		expr := &UnaryExpr{
			Operator: "-",
			Operand:  &NumberLiteral{Value: 5},
		}
		count := 0
		Walk(expr, func(n Node) bool {
			count++
			return true
		})
		if count != 2 { // UnaryExpr + Operand
			t.Errorf("expected 2 nodes, got %d", count)
		}
	})

	t.Run("walks nil", func(t *testing.T) {
		result := Walk(nil, func(n Node) bool {
			return true
		})
		if !result {
			t.Error("Walk on nil should return true")
		}
	})
}

// =============================================================================
// Additional Equal Tests for Coverage
// =============================================================================

func TestEqualAllNodeTypes(t *testing.T) {
	t.Run("equal bool literals", func(t *testing.T) {
		a := &BoolLiteral{Value: true}
		b := &BoolLiteral{Value: true}
		c := &BoolLiteral{Value: false}

		if !Equal(a, b) {
			t.Error("expected equal")
		}
		if Equal(a, c) {
			t.Error("expected not equal")
		}
	})

	t.Run("equal exec blocks", func(t *testing.T) {
		a := &ExecBlock{Command: "ls"}
		b := &ExecBlock{Command: "ls"}
		c := &ExecBlock{Command: "pwd"}

		if !Equal(a, b) {
			t.Error("expected equal")
		}
		if Equal(a, c) {
			t.Error("expected not equal")
		}
	})

	t.Run("equal blocks", func(t *testing.T) {
		a := &Block{Statements: []Statement{}}
		b := &Block{Statements: []Statement{}}

		if !Equal(a, b) {
			t.Error("expected equal")
		}
	})

	t.Run("unequal blocks different length", func(t *testing.T) {
		a := &Block{Statements: []Statement{}}
		b := &Block{Statements: []Statement{&VarDecl{Name: "x", Value: &NumberLiteral{Value: 1}}}}

		if Equal(a, b) {
			t.Error("expected not equal")
		}
	})

	t.Run("equal return stmts with value", func(t *testing.T) {
		a := &ReturnStmt{Value: &NumberLiteral{Value: 42}}
		b := &ReturnStmt{Value: &NumberLiteral{Value: 42}}

		if !Equal(a, b) {
			t.Error("expected equal")
		}
	})

	t.Run("equal return stmts nil", func(t *testing.T) {
		a := &ReturnStmt{}
		b := &ReturnStmt{}

		if !Equal(a, b) {
			t.Error("expected equal")
		}
	})

	t.Run("unequal return stmts one nil", func(t *testing.T) {
		a := &ReturnStmt{Value: &NumberLiteral{Value: 42}}
		b := &ReturnStmt{}

		if Equal(a, b) {
			t.Error("expected not equal")
		}
	})

	t.Run("equal if stmts", func(t *testing.T) {
		a := &IfStmt{
			Condition: &BoolLiteral{Value: true},
			Then:      &Block{Statements: []Statement{}},
		}
		b := &IfStmt{
			Condition: &BoolLiteral{Value: true},
			Then:      &Block{Statements: []Statement{}},
		}

		if !Equal(a, b) {
			t.Error("expected equal")
		}
	})

	t.Run("unequal if stmts different condition", func(t *testing.T) {
		a := &IfStmt{
			Condition: &BoolLiteral{Value: true},
			Then:      &Block{Statements: []Statement{}},
		}
		b := &IfStmt{
			Condition: &BoolLiteral{Value: false},
			Then:      &Block{Statements: []Statement{}},
		}

		if Equal(a, b) {
			t.Error("expected not equal")
		}
	})

	t.Run("unequal if stmts one with else", func(t *testing.T) {
		a := &IfStmt{
			Condition: &BoolLiteral{Value: true},
			Then:      &Block{Statements: []Statement{}},
			Else:      &Block{Statements: []Statement{}},
		}
		b := &IfStmt{
			Condition: &BoolLiteral{Value: true},
			Then:      &Block{Statements: []Statement{}},
		}

		if Equal(a, b) {
			t.Error("expected not equal")
		}
	})

	t.Run("equal for loops", func(t *testing.T) {
		a := &ForLoop{
			Variable: "item",
			Iterable: &Identifier{Name: "items"},
			Body:     &Block{Statements: []Statement{}},
		}
		b := &ForLoop{
			Variable: "item",
			Iterable: &Identifier{Name: "items"},
			Body:     &Block{Statements: []Statement{}},
		}

		if !Equal(a, b) {
			t.Error("expected equal")
		}
	})

	t.Run("equal function decls", func(t *testing.T) {
		a := &FunctionDecl{
			Name:   "test",
			Params: []Parameter{{Name: "x", Type: "int"}},
			Body:   &Block{Statements: []Statement{}},
		}
		b := &FunctionDecl{
			Name:   "test",
			Params: []Parameter{{Name: "x", Type: "int"}},
			Body:   &Block{Statements: []Statement{}},
		}

		if !Equal(a, b) {
			t.Error("expected equal")
		}
	})

	t.Run("unequal function decls different name", func(t *testing.T) {
		a := &FunctionDecl{
			Name:   "test1",
			Params: []Parameter{},
			Body:   &Block{Statements: []Statement{}},
		}
		b := &FunctionDecl{
			Name:   "test2",
			Params: []Parameter{},
			Body:   &Block{Statements: []Statement{}},
		}

		if Equal(a, b) {
			t.Error("expected not equal")
		}
	})

	t.Run("unequal function decls different param count", func(t *testing.T) {
		a := &FunctionDecl{
			Name:   "test",
			Params: []Parameter{{Name: "x"}},
			Body:   &Block{Statements: []Statement{}},
		}
		b := &FunctionDecl{
			Name:   "test",
			Params: []Parameter{},
			Body:   &Block{Statements: []Statement{}},
		}

		if Equal(a, b) {
			t.Error("expected not equal")
		}
	})

	t.Run("unequal function calls different name", func(t *testing.T) {
		a := &FunctionCall{Name: "add", Args: []Expression{}}
		b := &FunctionCall{Name: "sub", Args: []Expression{}}

		if Equal(a, b) {
			t.Error("expected not equal")
		}
	})

	t.Run("unequal function calls different arg count", func(t *testing.T) {
		a := &FunctionCall{Name: "add", Args: []Expression{&NumberLiteral{Value: 1}}}
		b := &FunctionCall{Name: "add", Args: []Expression{}}

		if Equal(a, b) {
			t.Error("expected not equal")
		}
	})

	t.Run("equal unary exprs", func(t *testing.T) {
		a := &UnaryExpr{Operator: "-", Operand: &NumberLiteral{Value: 5}}
		b := &UnaryExpr{Operator: "-", Operand: &NumberLiteral{Value: 5}}

		if !Equal(a, b) {
			t.Error("expected equal")
		}
	})

	t.Run("unequal programs different length", func(t *testing.T) {
		a := &Program{Statements: []Statement{}}
		b := &Program{Statements: []Statement{&VarDecl{Name: "x", Value: &NumberLiteral{Value: 1}}}}

		if Equal(a, b) {
			t.Error("expected not equal")
		}
	})
}

// =============================================================================
// Additional Print Tests for Coverage
// =============================================================================

func TestPrintAllNodeTypes(t *testing.T) {
	t.Run("prints nil", func(t *testing.T) {
		result := Print(nil)
		if result != "<nil>" {
			t.Errorf("expected '<nil>', got %q", result)
		}
	})

	t.Run("prints assignment", func(t *testing.T) {
		node := &Assignment{Name: "x", Value: &NumberLiteral{Value: 42}}
		result := Print(node)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("prints for loop", func(t *testing.T) {
		node := &ForLoop{
			Variable: "item",
			Iterable: &Identifier{Name: "items"},
			Body:     &Block{Statements: []Statement{}},
		}
		result := Print(node)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("prints function decl", func(t *testing.T) {
		node := &FunctionDecl{
			Name:   "test",
			Params: []Parameter{{Name: "x"}, {Name: "y"}},
			Body:   &Block{Statements: []Statement{}},
		}
		result := Print(node)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("prints exec block", func(t *testing.T) {
		node := &ExecBlock{Command: "ls -la"}
		result := Print(node)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("prints return stmt", func(t *testing.T) {
		node := &ReturnStmt{Value: &NumberLiteral{Value: 42}}
		result := Print(node)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("prints return stmt nil", func(t *testing.T) {
		node := &ReturnStmt{}
		result := Print(node)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("prints bool literal true", func(t *testing.T) {
		node := &BoolLiteral{Value: true}
		result := Print(node)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("prints bool literal false", func(t *testing.T) {
		node := &BoolLiteral{Value: false}
		result := Print(node)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("prints identifier", func(t *testing.T) {
		node := &Identifier{Name: "myVar"}
		result := Print(node)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("prints function call", func(t *testing.T) {
		node := &FunctionCall{
			Name: "add",
			Args: []Expression{&NumberLiteral{Value: 1}, &NumberLiteral{Value: 2}},
		}
		result := Print(node)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("prints array literal", func(t *testing.T) {
		node := &ArrayLiteral{
			Elements: []Expression{&NumberLiteral{Value: 1}, &NumberLiteral{Value: 2}},
		}
		result := Print(node)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("prints binary expr", func(t *testing.T) {
		node := &BinaryExpr{
			Left:     &NumberLiteral{Value: 1},
			Operator: "+",
			Right:    &NumberLiteral{Value: 2},
		}
		result := Print(node)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("prints unary expr", func(t *testing.T) {
		node := &UnaryExpr{Operator: "-", Operand: &NumberLiteral{Value: 5}}
		result := Print(node)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("prints if with else", func(t *testing.T) {
		node := &IfStmt{
			Condition: &BoolLiteral{Value: true},
			Then:      &Block{Statements: []Statement{}},
			Else:      &Block{Statements: []Statement{}},
		}
		result := Print(node)
		if result == "" {
			t.Error("expected non-empty output")
		}
	})
}

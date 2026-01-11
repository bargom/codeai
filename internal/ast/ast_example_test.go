package ast_test

import (
	"fmt"

	"github.com/bargom/codeai/internal/ast"
)

// ExamplePosition demonstrates creating and using Position values.
func ExamplePosition() {
	pos := ast.Position{
		Filename: "example.codeai",
		Line:     10,
		Column:   5,
		Offset:   150,
	}

	fmt.Println(pos.String())
	fmt.Println("Valid:", pos.IsValid())
	// Output:
	// example.codeai:10:5
	// Valid: true
}

// ExampleProgram demonstrates creating a Program node with statements.
func ExampleProgram() {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.VarDecl{
				Name:  "greeting",
				Value: &ast.StringLiteral{Value: "hello"},
			},
		},
	}

	fmt.Println("Type:", prog.Type())
	fmt.Println("Statements:", len(prog.Statements))
	// Output:
	// Type: Program
	// Statements: 1
}

// ExampleVarDecl demonstrates creating a variable declaration.
func ExampleVarDecl() {
	decl := &ast.VarDecl{
		Name:  "count",
		Value: &ast.NumberLiteral{Value: 42},
	}

	fmt.Println("Type:", decl.Type())
	fmt.Println("Name:", decl.Name)
	// Output:
	// Type: VarDecl
	// Name: count
}

// ExampleFunctionCall demonstrates creating a function call expression.
func ExampleFunctionCall() {
	call := &ast.FunctionCall{
		Name: "add",
		Args: []ast.Expression{
			&ast.NumberLiteral{Value: 1},
			&ast.NumberLiteral{Value: 2},
		},
	}

	fmt.Println("Type:", call.Type())
	fmt.Println("Function:", call.Name)
	fmt.Println("Arguments:", len(call.Args))
	// Output:
	// Type: FunctionCall
	// Function: add
	// Arguments: 2
}

// ExampleWalk demonstrates traversing an AST to count nodes.
func ExampleWalk() {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.VarDecl{
				Name:  "x",
				Value: &ast.NumberLiteral{Value: 1},
			},
			&ast.VarDecl{
				Name:  "y",
				Value: &ast.NumberLiteral{Value: 2},
			},
		},
	}

	count := 0
	ast.Walk(prog, func(n ast.Node) bool {
		count++
		return true
	})

	fmt.Println("Node count:", count)
	// Output:
	// Node count: 5
}

// ExampleWalk_findIdentifiers demonstrates using Walk to find all identifiers.
func ExampleWalk_findIdentifiers() {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.VarDecl{
				Name:  "result",
				Value: &ast.Identifier{Name: "input"},
			},
		},
	}

	identifiers := []string{}
	ast.Walk(prog, func(n ast.Node) bool {
		if id, ok := n.(*ast.Identifier); ok {
			identifiers = append(identifiers, id.Name)
		}
		return true
	})

	fmt.Println("Identifiers found:", identifiers)
	// Output:
	// Identifiers found: [input]
}

// ExamplePrint demonstrates pretty-printing an AST.
func ExamplePrint() {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.VarDecl{
				Name:  "x",
				Value: &ast.NumberLiteral{Value: 42},
			},
		},
	}

	output := ast.Print(prog)
	fmt.Println(output)
	// Output:
	// Program
	//   VarDecl: x
	//     NumberLiteral: NumberLiteral{42}
}

// ExampleEqual demonstrates comparing AST nodes for equality.
func ExampleEqual() {
	a := &ast.NumberLiteral{Value: 42}
	b := &ast.NumberLiteral{Value: 42}
	c := &ast.NumberLiteral{Value: 100}

	fmt.Println("a == b:", ast.Equal(a, b))
	fmt.Println("a == c:", ast.Equal(a, c))
	// Output:
	// a == b: true
	// a == c: false
}

// ExampleClone demonstrates deep copying an AST node.
func ExampleClone() {
	original := &ast.VarDecl{
		Name:  "x",
		Value: &ast.NumberLiteral{Value: 42},
	}

	cloned := ast.Clone(original).(*ast.VarDecl)

	// Modify the clone
	cloned.Name = "y"

	fmt.Println("Original name:", original.Name)
	fmt.Println("Cloned name:", cloned.Name)
	// Output:
	// Original name: x
	// Cloned name: y
}

// ExampleBinaryExpr demonstrates creating binary expressions.
func ExampleBinaryExpr() {
	// Represents: 1 + 2 * 3
	expr := &ast.BinaryExpr{
		Left: &ast.NumberLiteral{Value: 1},
		Operator: "+",
		Right: &ast.BinaryExpr{
			Left:     &ast.NumberLiteral{Value: 2},
			Operator: "*",
			Right:    &ast.NumberLiteral{Value: 3},
		},
	}

	fmt.Println("Type:", expr.Type())
	fmt.Println("Operator:", expr.Operator)
	// Output:
	// Type: BinaryExpr
	// Operator: +
}

// ExampleArrayLiteral demonstrates creating array literals.
func ExampleArrayLiteral() {
	arr := &ast.ArrayLiteral{
		Elements: []ast.Expression{
			&ast.StringLiteral{Value: "a"},
			&ast.StringLiteral{Value: "b"},
			&ast.StringLiteral{Value: "c"},
		},
	}

	fmt.Println("Type:", arr.Type())
	fmt.Println("Elements:", len(arr.Elements))
	// Output:
	// Type: ArrayLiteral
	// Elements: 3
}

// ExampleIfStmt demonstrates creating if statements.
func ExampleIfStmt() {
	ifStmt := &ast.IfStmt{
		Condition: &ast.BoolLiteral{Value: true},
		Then: &ast.Block{
			Statements: []ast.Statement{
				&ast.ReturnStmt{Value: &ast.NumberLiteral{Value: 1}},
			},
		},
		Else: &ast.Block{
			Statements: []ast.Statement{
				&ast.ReturnStmt{Value: &ast.NumberLiteral{Value: 0}},
			},
		},
	}

	fmt.Println("Type:", ifStmt.Type())
	fmt.Println("Has else:", ifStmt.Else != nil)
	// Output:
	// Type: IfStmt
	// Has else: true
}

// ExampleForLoop demonstrates creating for-in loops.
func ExampleForLoop() {
	forLoop := &ast.ForLoop{
		Variable: "item",
		Iterable: &ast.Identifier{Name: "items"},
		Body: &ast.Block{
			Statements: []ast.Statement{
				&ast.VarDecl{
					Name:  "processed",
					Value: &ast.Identifier{Name: "item"},
				},
			},
		},
	}

	fmt.Println("Type:", forLoop.Type())
	fmt.Println("Variable:", forLoop.Variable)
	// Output:
	// Type: ForLoop
	// Variable: item
}

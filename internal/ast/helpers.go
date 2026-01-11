package ast

import (
	"strings"
)

// Visitor is a function type for AST traversal.
// It receives each node and returns true to continue traversal
// or false to stop.
type Visitor func(node Node) bool

// Walk traverses the AST in depth-first order, calling the visitor
// function for each node. If the visitor returns false, traversal stops
// immediately. Walk returns true if traversal completed normally, false
// if it was stopped early.
func Walk(node Node, visitor Visitor) bool {
	if node == nil {
		return true
	}

	if !visitor(node) {
		return false
	}

	switch n := node.(type) {
	case *Program:
		for _, stmt := range n.Statements {
			if !Walk(stmt, visitor) {
				return false
			}
		}

	case *VarDecl:
		if !Walk(n.Value, visitor) {
			return false
		}

	case *Assignment:
		if !Walk(n.Value, visitor) {
			return false
		}

	case *IfStmt:
		if !Walk(n.Condition, visitor) {
			return false
		}
		if !Walk(n.Then, visitor) {
			return false
		}
		if n.Else != nil {
			if !Walk(n.Else, visitor) {
				return false
			}
		}

	case *ForLoop:
		if !Walk(n.Iterable, visitor) {
			return false
		}
		if !Walk(n.Body, visitor) {
			return false
		}

	case *FunctionDecl:
		for _, p := range n.Params {
			if p.Default != nil {
				if !Walk(p.Default, visitor) {
					return false
				}
			}
		}
		if !Walk(n.Body, visitor) {
			return false
		}

	case *ExecBlock:
		// No children

	case *Block:
		for _, stmt := range n.Statements {
			if !Walk(stmt, visitor) {
				return false
			}
		}

	case *ReturnStmt:
		if n.Value != nil {
			if !Walk(n.Value, visitor) {
				return false
			}
		}

	case *StringLiteral:
		// No children

	case *NumberLiteral:
		// No children

	case *BoolLiteral:
		// No children

	case *Identifier:
		// No children

	case *FunctionCall:
		for _, arg := range n.Args {
			if !Walk(arg, visitor) {
				return false
			}
		}

	case *ArrayLiteral:
		for _, elem := range n.Elements {
			if !Walk(elem, visitor) {
				return false
			}
		}

	case *BinaryExpr:
		if !Walk(n.Left, visitor) {
			return false
		}
		if !Walk(n.Right, visitor) {
			return false
		}

	case *UnaryExpr:
		if !Walk(n.Operand, visitor) {
			return false
		}
	}

	return true
}

// Print returns a pretty-printed string representation of the AST.
func Print(node Node) string {
	if node == nil {
		return "<nil>"
	}

	var b strings.Builder
	printNode(&b, node, 0)
	return b.String()
}

func printNode(b *strings.Builder, node Node, indent int) {
	if node == nil {
		return
	}

	prefix := strings.Repeat("  ", indent)

	switch n := node.(type) {
	case *Program:
		b.WriteString(prefix)
		b.WriteString("Program\n")
		for _, stmt := range n.Statements {
			printNode(b, stmt, indent+1)
		}

	case *VarDecl:
		b.WriteString(prefix)
		b.WriteString("VarDecl: ")
		b.WriteString(n.Name)
		b.WriteString("\n")
		printNode(b, n.Value, indent+1)

	case *Assignment:
		b.WriteString(prefix)
		b.WriteString("Assignment: ")
		b.WriteString(n.Name)
		b.WriteString("\n")
		printNode(b, n.Value, indent+1)

	case *IfStmt:
		b.WriteString(prefix)
		b.WriteString("IfStmt\n")
		b.WriteString(prefix)
		b.WriteString("  Condition:\n")
		printNode(b, n.Condition, indent+2)
		b.WriteString(prefix)
		b.WriteString("  Then:\n")
		printNode(b, n.Then, indent+2)
		if n.Else != nil {
			b.WriteString(prefix)
			b.WriteString("  Else:\n")
			printNode(b, n.Else, indent+2)
		}

	case *ForLoop:
		b.WriteString(prefix)
		b.WriteString("ForLoop: ")
		b.WriteString(n.Variable)
		b.WriteString("\n")
		b.WriteString(prefix)
		b.WriteString("  Iterable:\n")
		printNode(b, n.Iterable, indent+2)
		b.WriteString(prefix)
		b.WriteString("  Body:\n")
		printNode(b, n.Body, indent+2)

	case *FunctionDecl:
		b.WriteString(prefix)
		b.WriteString("FunctionDecl: ")
		b.WriteString(n.Name)
		b.WriteString("(")
		for i, p := range n.Params {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(p.Name)
		}
		b.WriteString(")\n")
		printNode(b, n.Body, indent+1)

	case *ExecBlock:
		b.WriteString(prefix)
		b.WriteString("ExecBlock: ")
		b.WriteString(n.Command)
		b.WriteString("\n")

	case *Block:
		b.WriteString(prefix)
		b.WriteString("Block\n")
		for _, stmt := range n.Statements {
			printNode(b, stmt, indent+1)
		}

	case *ReturnStmt:
		b.WriteString(prefix)
		b.WriteString("ReturnStmt\n")
		if n.Value != nil {
			printNode(b, n.Value, indent+1)
		}

	case *StringLiteral:
		b.WriteString(prefix)
		b.WriteString("StringLiteral: \"")
		b.WriteString(n.Value)
		b.WriteString("\"\n")

	case *NumberLiteral:
		b.WriteString(prefix)
		b.WriteString("NumberLiteral: ")
		b.WriteString(n.String())
		b.WriteString("\n")

	case *BoolLiteral:
		b.WriteString(prefix)
		b.WriteString("BoolLiteral: ")
		if n.Value {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
		b.WriteString("\n")

	case *Identifier:
		b.WriteString(prefix)
		b.WriteString("Identifier: ")
		b.WriteString(n.Name)
		b.WriteString("\n")

	case *FunctionCall:
		b.WriteString(prefix)
		b.WriteString("FunctionCall: ")
		b.WriteString(n.Name)
		b.WriteString("\n")
		for _, arg := range n.Args {
			printNode(b, arg, indent+1)
		}

	case *ArrayLiteral:
		b.WriteString(prefix)
		b.WriteString("ArrayLiteral\n")
		for _, elem := range n.Elements {
			printNode(b, elem, indent+1)
		}

	case *BinaryExpr:
		b.WriteString(prefix)
		b.WriteString("BinaryExpr: ")
		b.WriteString(n.Operator)
		b.WriteString("\n")
		printNode(b, n.Left, indent+1)
		printNode(b, n.Right, indent+1)

	case *UnaryExpr:
		b.WriteString(prefix)
		b.WriteString("UnaryExpr: ")
		b.WriteString(n.Operator)
		b.WriteString("\n")
		printNode(b, n.Operand, indent+1)
	}
}

// Equal performs a deep equality check between two AST nodes.
func Equal(a, b Node) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Type() != b.Type() {
		return false
	}

	switch na := a.(type) {
	case *Program:
		nb := b.(*Program)
		if len(na.Statements) != len(nb.Statements) {
			return false
		}
		for i := range na.Statements {
			if !Equal(na.Statements[i], nb.Statements[i]) {
				return false
			}
		}
		return true

	case *VarDecl:
		nb := b.(*VarDecl)
		return na.Name == nb.Name && Equal(na.Value, nb.Value)

	case *Assignment:
		nb := b.(*Assignment)
		return na.Name == nb.Name && Equal(na.Value, nb.Value)

	case *IfStmt:
		nb := b.(*IfStmt)
		if !Equal(na.Condition, nb.Condition) {
			return false
		}
		if !Equal(na.Then, nb.Then) {
			return false
		}
		if na.Else == nil && nb.Else == nil {
			return true
		}
		if na.Else == nil || nb.Else == nil {
			return false
		}
		return Equal(na.Else, nb.Else)

	case *ForLoop:
		nb := b.(*ForLoop)
		return na.Variable == nb.Variable &&
			Equal(na.Iterable, nb.Iterable) &&
			Equal(na.Body, nb.Body)

	case *FunctionDecl:
		nb := b.(*FunctionDecl)
		if na.Name != nb.Name {
			return false
		}
		if len(na.Params) != len(nb.Params) {
			return false
		}
		for i := range na.Params {
			if na.Params[i].Name != nb.Params[i].Name ||
				na.Params[i].Type != nb.Params[i].Type {
				return false
			}
		}
		return Equal(na.Body, nb.Body)

	case *ExecBlock:
		nb := b.(*ExecBlock)
		return na.Command == nb.Command

	case *Block:
		nb := b.(*Block)
		if len(na.Statements) != len(nb.Statements) {
			return false
		}
		for i := range na.Statements {
			if !Equal(na.Statements[i], nb.Statements[i]) {
				return false
			}
		}
		return true

	case *ReturnStmt:
		nb := b.(*ReturnStmt)
		if na.Value == nil && nb.Value == nil {
			return true
		}
		if na.Value == nil || nb.Value == nil {
			return false
		}
		return Equal(na.Value, nb.Value)

	case *StringLiteral:
		nb := b.(*StringLiteral)
		return na.Value == nb.Value

	case *NumberLiteral:
		nb := b.(*NumberLiteral)
		return na.Value == nb.Value

	case *BoolLiteral:
		nb := b.(*BoolLiteral)
		return na.Value == nb.Value

	case *Identifier:
		nb := b.(*Identifier)
		return na.Name == nb.Name

	case *FunctionCall:
		nb := b.(*FunctionCall)
		if na.Name != nb.Name {
			return false
		}
		if len(na.Args) != len(nb.Args) {
			return false
		}
		for i := range na.Args {
			if !Equal(na.Args[i], nb.Args[i]) {
				return false
			}
		}
		return true

	case *ArrayLiteral:
		nb := b.(*ArrayLiteral)
		if len(na.Elements) != len(nb.Elements) {
			return false
		}
		for i := range na.Elements {
			if !Equal(na.Elements[i], nb.Elements[i]) {
				return false
			}
		}
		return true

	case *BinaryExpr:
		nb := b.(*BinaryExpr)
		return na.Operator == nb.Operator &&
			Equal(na.Left, nb.Left) &&
			Equal(na.Right, nb.Right)

	case *UnaryExpr:
		nb := b.(*UnaryExpr)
		return na.Operator == nb.Operator &&
			Equal(na.Operand, nb.Operand)
	}

	return false
}

// Clone creates a deep copy of the given AST node.
func Clone(node Node) Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *Program:
		stmts := make([]Statement, len(n.Statements))
		for i, stmt := range n.Statements {
			stmts[i] = Clone(stmt).(Statement)
		}
		return &Program{
			pos:        n.pos,
			Statements: stmts,
		}

	case *VarDecl:
		return &VarDecl{
			pos:   n.pos,
			Name:  n.Name,
			Value: Clone(n.Value).(Expression),
		}

	case *Assignment:
		return &Assignment{
			pos:   n.pos,
			Name:  n.Name,
			Value: Clone(n.Value).(Expression),
		}

	case *IfStmt:
		result := &IfStmt{
			pos:       n.pos,
			Condition: Clone(n.Condition).(Expression),
			Then:      Clone(n.Then).(*Block),
		}
		if n.Else != nil {
			result.Else = Clone(n.Else).(*Block)
		}
		return result

	case *ForLoop:
		return &ForLoop{
			pos:      n.pos,
			Variable: n.Variable,
			Iterable: Clone(n.Iterable).(Expression),
			Body:     Clone(n.Body).(*Block),
		}

	case *FunctionDecl:
		params := make([]Parameter, len(n.Params))
		for i, p := range n.Params {
			params[i] = Parameter{
				Name: p.Name,
				Type: p.Type,
			}
			if p.Default != nil {
				params[i].Default = Clone(p.Default).(Expression)
			}
		}
		return &FunctionDecl{
			pos:    n.pos,
			Name:   n.Name,
			Params: params,
			Body:   Clone(n.Body).(*Block),
		}

	case *ExecBlock:
		return &ExecBlock{
			pos:     n.pos,
			Command: n.Command,
		}

	case *Block:
		stmts := make([]Statement, len(n.Statements))
		for i, stmt := range n.Statements {
			stmts[i] = Clone(stmt).(Statement)
		}
		return &Block{
			pos:        n.pos,
			Statements: stmts,
		}

	case *ReturnStmt:
		result := &ReturnStmt{pos: n.pos}
		if n.Value != nil {
			result.Value = Clone(n.Value).(Expression)
		}
		return result

	case *StringLiteral:
		return &StringLiteral{
			pos:   n.pos,
			Value: n.Value,
		}

	case *NumberLiteral:
		return &NumberLiteral{
			pos:   n.pos,
			Value: n.Value,
		}

	case *BoolLiteral:
		return &BoolLiteral{
			pos:   n.pos,
			Value: n.Value,
		}

	case *Identifier:
		return &Identifier{
			pos:  n.pos,
			Name: n.Name,
		}

	case *FunctionCall:
		args := make([]Expression, len(n.Args))
		for i, arg := range n.Args {
			args[i] = Clone(arg).(Expression)
		}
		return &FunctionCall{
			pos:  n.pos,
			Name: n.Name,
			Args: args,
		}

	case *ArrayLiteral:
		elems := make([]Expression, len(n.Elements))
		for i, elem := range n.Elements {
			elems[i] = Clone(elem).(Expression)
		}
		return &ArrayLiteral{
			pos:      n.pos,
			Elements: elems,
		}

	case *BinaryExpr:
		return &BinaryExpr{
			pos:      n.pos,
			Left:     Clone(n.Left).(Expression),
			Operator: n.Operator,
			Right:    Clone(n.Right).(Expression),
		}

	case *UnaryExpr:
		return &UnaryExpr{
			pos:      n.pos,
			Operator: n.Operator,
			Operand:  Clone(n.Operand).(Expression),
		}
	}

	return nil
}

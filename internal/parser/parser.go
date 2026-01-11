// Package parser provides a Participle-based parser for the CodeAI DSL.
package parser

import (
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"

	"github.com/bargom/codeai/internal/ast"
)

// =============================================================================
// Lexer Definition
// =============================================================================

var dslLexer = lexer.MustStateful(lexer.Rules{
	"Root": {
		// Whitespace and comments
		{Name: "whitespace", Pattern: `[\s]+`, Action: nil},
		{Name: "SingleLineComment", Pattern: `//[^\n]*`, Action: nil},
		{Name: "MultiLineComment", Pattern: `/\*([^*]|\*[^/])*\*/`, Action: nil},

		// Exec block - must come before keywords to capture properly
		{Name: "ExecOpen", Pattern: `exec\s*\{`, Action: lexer.Push("Shell")},

		// Keywords
		{Name: "Var", Pattern: `\bvar\b`, Action: nil},
		{Name: "If", Pattern: `\bif\b`, Action: nil},
		{Name: "Else", Pattern: `\belse\b`, Action: nil},
		{Name: "For", Pattern: `\bfor\b`, Action: nil},
		{Name: "In", Pattern: `\bin\b`, Action: nil},
		{Name: "Func", Pattern: `\bfunction\b`, Action: nil},
		{Name: "True", Pattern: `\btrue\b`, Action: nil},
		{Name: "False", Pattern: `\bfalse\b`, Action: nil},

		// Literals
		{Name: "Number", Pattern: `[0-9]+\.?[0-9]*`, Action: nil},
		{Name: "String", Pattern: `"([^"\\]|\\.)*"`, Action: nil},

		// Identifiers
		{Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`, Action: nil},

		// Operators and punctuation
		{Name: "Equals", Pattern: `=`, Action: nil},
		{Name: "LBracket", Pattern: `\[`, Action: nil},
		{Name: "RBracket", Pattern: `\]`, Action: nil},
		{Name: "LParen", Pattern: `\(`, Action: nil},
		{Name: "RParen", Pattern: `\)`, Action: nil},
		{Name: "LBrace", Pattern: `\{`, Action: nil},
		{Name: "RBrace", Pattern: `\}`, Action: nil},
		{Name: "Comma", Pattern: `,`, Action: nil},
	},
	"Shell": {
		// Capture everything until closing brace as ShellCommand
		{Name: "ShellCommand", Pattern: `[^}]+`, Action: nil},
		{Name: "ShellClose", Pattern: `\}`, Action: lexer.Pop()},
	},
})

// =============================================================================
// Participle Grammar Structs (Intermediate Representation)
// =============================================================================

// pProgram is the Participle grammar for a program.
type pProgram struct {
	Pos        lexer.Position
	Statements []*pStatement `@@*`
}

// pStatement is the Participle grammar for a statement.
type pStatement struct {
	Pos          lexer.Position
	VarDecl      *pVarDecl      `  @@`
	IfStmt       *pIfStmt       `| @@`
	ForLoop      *pForLoop      `| @@`
	FuncDecl     *pFuncDecl     `| @@`
	ExecBlock    *pExecBlock    `| @@`
	Assignment   *pAssignment   `| @@`
}

// pVarDecl is the Participle grammar for variable declaration.
type pVarDecl struct {
	Pos   lexer.Position
	Name  string       `Var @Ident`
	Value *pExpression `Equals @@`
}

// pAssignment is the Participle grammar for assignment.
type pAssignment struct {
	Pos   lexer.Position
	Name  string       `@Ident`
	Value *pExpression `Equals @@`
}

// pIfStmt is the Participle grammar for if statement.
type pIfStmt struct {
	Pos       lexer.Position
	Condition *pExpression  `If @@`
	Body      []*pStatement `LBrace @@* RBrace`
	Else      []*pStatement `( Else LBrace @@* RBrace )?`
}

// pForLoop is the Participle grammar for for loop.
type pForLoop struct {
	Pos      lexer.Position
	Variable string        `For @Ident`
	Iterable *pExpression  `In @@`
	Body     []*pStatement `LBrace @@* RBrace`
}

// pFuncDecl is the Participle grammar for function declaration.
type pFuncDecl struct {
	Pos    lexer.Position
	Name   string        `Func @Ident`
	Params []string      `LParen ( @Ident ( Comma @Ident )* )? RParen`
	Body   []*pStatement `LBrace @@* RBrace`
}

// pExecBlock is the Participle grammar for exec block.
type pExecBlock struct {
	Pos     lexer.Position
	Command string `ExecOpen @ShellCommand ShellClose`
}

// pExpression is the Participle grammar for expressions.
type pExpression struct {
	Pos          lexer.Position
	String       *string        `  @String`
	Number       *string        `| @Number`
	True         bool           `| @True`
	False        bool           `| @False`
	Array        *pArrayLiteral `| @@`
	FuncCall     *pFuncCall     `| @@`
	Identifier   *string        `| @Ident`
}

// pArrayLiteral is the Participle grammar for array literals.
type pArrayLiteral struct {
	Pos      lexer.Position
	Elements []*pExpression `LBracket ( @@ ( Comma @@ )* )? RBracket`
}

// pFuncCall is the Participle grammar for function calls.
type pFuncCall struct {
	Pos       lexer.Position
	Name      string         `@Ident`
	Arguments []*pExpression `LParen ( @@ ( Comma @@ )* )? RParen`
}

// =============================================================================
// Parser Instance
// =============================================================================

var parserInstance = participle.MustBuild[pProgram](
	participle.Lexer(dslLexer),
	participle.Elide("whitespace", "SingleLineComment", "MultiLineComment"),
	participle.UseLookahead(3),
)

// =============================================================================
// Public API
// =============================================================================

// Parse parses the input string and returns an AST Program.
func Parse(input string) (*ast.Program, error) {
	parsed, err := parserInstance.ParseString("", input)
	if err != nil {
		return nil, err
	}
	return convertProgram(parsed), nil
}

// ParseFile parses a file and returns an AST Program.
func ParseFile(filename string) (*ast.Program, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	parsed, err := parserInstance.ParseBytes(filename, data)
	if err != nil {
		return nil, err
	}
	return convertProgram(parsed), nil
}

// =============================================================================
// Conversion Helpers (Participle IR -> AST)
// =============================================================================

func convertProgram(p *pProgram) *ast.Program {
	stmts := make([]ast.Statement, 0, len(p.Statements))
	for _, s := range p.Statements {
		stmts = append(stmts, convertStatement(s))
	}
	return createProgram(stmts)
}

func convertStatement(s *pStatement) ast.Statement {
	switch {
	case s.VarDecl != nil:
		return convertVarDecl(s.VarDecl)
	case s.Assignment != nil:
		return convertAssignment(s.Assignment)
	case s.IfStmt != nil:
		return convertIfStmt(s.IfStmt)
	case s.ForLoop != nil:
		return convertForLoop(s.ForLoop)
	case s.FuncDecl != nil:
		return convertFuncDecl(s.FuncDecl)
	case s.ExecBlock != nil:
		return convertExecBlock(s.ExecBlock)
	default:
		return nil
	}
}

func convertVarDecl(v *pVarDecl) *ast.VarDecl {
	return createVarDecl(v.Name, convertExpression(v.Value))
}

func convertAssignment(a *pAssignment) *ast.Assignment {
	return createAssignment(a.Name, convertExpression(a.Value))
}

func convertIfStmt(i *pIfStmt) *ast.IfStmt {
	thenStmts := make([]ast.Statement, 0, len(i.Body))
	for _, s := range i.Body {
		thenStmts = append(thenStmts, convertStatement(s))
	}
	thenBlock := createBlock(thenStmts)

	var elseBlock *ast.Block
	if len(i.Else) > 0 {
		elseStmts := make([]ast.Statement, 0, len(i.Else))
		for _, s := range i.Else {
			elseStmts = append(elseStmts, convertStatement(s))
		}
		elseBlock = createBlock(elseStmts)
	}

	return createIfStmt(convertExpression(i.Condition), thenBlock, elseBlock)
}

func convertForLoop(f *pForLoop) *ast.ForLoop {
	bodyStmts := make([]ast.Statement, 0, len(f.Body))
	for _, s := range f.Body {
		bodyStmts = append(bodyStmts, convertStatement(s))
	}
	body := createBlock(bodyStmts)

	return createForLoop(f.Variable, convertExpression(f.Iterable), body)
}

func convertFuncDecl(f *pFuncDecl) *ast.FunctionDecl {
	params := make([]ast.Parameter, len(f.Params))
	for i, p := range f.Params {
		params[i] = ast.Parameter{Name: p}
	}

	bodyStmts := make([]ast.Statement, 0, len(f.Body))
	for _, s := range f.Body {
		bodyStmts = append(bodyStmts, convertStatement(s))
	}
	body := createBlock(bodyStmts)

	return createFuncDecl(f.Name, params, body)
}

func convertExecBlock(e *pExecBlock) *ast.ExecBlock {
	return createExecBlock(strings.TrimSpace(e.Command))
}

func convertExpression(e *pExpression) ast.Expression {
	switch {
	case e.String != nil:
		// Remove surrounding quotes
		s := *e.String
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			s = s[1 : len(s)-1]
		}
		return createStringLiteral(s)
	case e.Number != nil:
		n, _ := strconv.ParseFloat(*e.Number, 64)
		return createNumberLiteral(n)
	case e.True:
		return createBoolLiteral(true)
	case e.False:
		return createBoolLiteral(false)
	case e.Array != nil:
		return convertArrayLiteral(e.Array)
	case e.FuncCall != nil:
		return convertFuncCall(e.FuncCall)
	case e.Identifier != nil:
		return createIdentifier(*e.Identifier)
	default:
		return nil
	}
}

func convertArrayLiteral(a *pArrayLiteral) *ast.ArrayLiteral {
	elems := make([]ast.Expression, len(a.Elements))
	for i, e := range a.Elements {
		elems[i] = convertExpression(e)
	}
	return createArrayLiteral(elems)
}

func convertFuncCall(f *pFuncCall) *ast.FunctionCall {
	args := make([]ast.Expression, len(f.Arguments))
	for i, a := range f.Arguments {
		args[i] = convertExpression(a)
	}
	return createFuncCallNode(f.Name, args)
}

// =============================================================================
// AST Node Creators
// =============================================================================

func createProgram(stmts []ast.Statement) *ast.Program {
	return &ast.Program{Statements: stmts}
}

func createVarDecl(name string, value ast.Expression) *ast.VarDecl {
	return &ast.VarDecl{Name: name, Value: value}
}

func createAssignment(name string, value ast.Expression) *ast.Assignment {
	return &ast.Assignment{Name: name, Value: value}
}

func createIfStmt(cond ast.Expression, then, elseBlock *ast.Block) *ast.IfStmt {
	return &ast.IfStmt{Condition: cond, Then: then, Else: elseBlock}
}

func createForLoop(variable string, iterable ast.Expression, body *ast.Block) *ast.ForLoop {
	return &ast.ForLoop{Variable: variable, Iterable: iterable, Body: body}
}

func createFuncDecl(name string, params []ast.Parameter, body *ast.Block) *ast.FunctionDecl {
	return &ast.FunctionDecl{Name: name, Params: params, Body: body}
}

func createExecBlock(command string) *ast.ExecBlock {
	return &ast.ExecBlock{Command: command}
}

func createBlock(stmts []ast.Statement) *ast.Block {
	return &ast.Block{Statements: stmts}
}

func createStringLiteral(value string) *ast.StringLiteral {
	return &ast.StringLiteral{Value: value}
}

func createNumberLiteral(value float64) *ast.NumberLiteral {
	return &ast.NumberLiteral{Value: value}
}

func createBoolLiteral(value bool) *ast.BoolLiteral {
	return &ast.BoolLiteral{Value: value}
}

func createIdentifier(name string) *ast.Identifier {
	return &ast.Identifier{Name: name}
}

func createArrayLiteral(elems []ast.Expression) *ast.ArrayLiteral {
	return &ast.ArrayLiteral{Elements: elems}
}

func createFuncCallNode(name string, args []ast.Expression) *ast.FunctionCall {
	return &ast.FunctionCall{Name: name, Args: args}
}

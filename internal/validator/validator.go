// Package validator provides semantic validation for CodeAI AST.
package validator

import (
	"github.com/bargom/codeai/internal/ast"
)

// Validator performs semantic analysis on CodeAI AST.
type Validator struct {
	symbols     *SymbolTable
	typeChecker *TypeChecker
	errors      *ValidationErrors
	types       map[string]Type // Track variable types for testing
}

// New creates a new Validator instance.
func New() *Validator {
	symbols := NewSymbolTable()
	return &Validator{
		symbols:     symbols,
		typeChecker: NewTypeChecker(symbols),
		errors:      &ValidationErrors{},
		types:       make(map[string]Type),
	}
}

// Validate performs semantic validation on the given AST program.
// It returns an error if any validation issues are found.
func (v *Validator) Validate(program *ast.Program) error {
	if program == nil {
		return nil
	}

	// First pass: collect function declarations (for forward references)
	// Note: CodeAI requires declaration before use, so this is optional
	// For now, we'll validate in order

	// Validate all statements
	for _, stmt := range program.Statements {
		v.validateStatement(stmt)
	}

	// Return aggregated errors if any
	if v.errors.HasErrors() {
		return v.errors
	}
	return nil
}

// TypeOf returns the inferred type of a variable by name.
// Used for testing type inference.
func (v *Validator) TypeOf(name string) (Type, bool) {
	typ, ok := v.types[name]
	return typ, ok
}

// validateStatement dispatches validation based on statement type.
func (v *Validator) validateStatement(stmt ast.Statement) {
	if stmt == nil {
		return
	}

	switch s := stmt.(type) {
	case *ast.VarDecl:
		v.validateVarDecl(s)
	case *ast.Assignment:
		v.validateAssignment(s)
	case *ast.IfStmt:
		v.validateIfStmt(s)
	case *ast.ForLoop:
		v.validateForLoop(s)
	case *ast.FunctionDecl:
		v.validateFunctionDecl(s)
	case *ast.ExecBlock:
		v.validateExecBlock(s)
	case *ast.Block:
		v.validateBlock(s)
	case *ast.ReturnStmt:
		v.validateReturnStmt(s)
	}
}

// validateVarDecl validates a variable declaration.
func (v *Validator) validateVarDecl(decl *ast.VarDecl) {
	// First, validate the value expression
	v.validateExpression(decl.Value)

	// Check for duplicate declaration in current scope
	if _, exists := v.symbols.LookupLocal(decl.Name); exists {
		v.errors.Add(errDuplicateDeclaration(decl.Pos(), decl.Name))
		return
	}

	// Infer type from value
	typ := v.typeChecker.InferType(decl.Value)

	// Declare the variable with its type
	if err := v.symbols.DeclareWithType(decl.Name, SymbolVariable, typ); err != nil {
		v.errors.Add(errDuplicateDeclaration(decl.Pos(), decl.Name))
		return
	}

	// Track type for testing
	v.types[decl.Name] = typ
}

// validateAssignment validates an assignment statement.
func (v *Validator) validateAssignment(assign *ast.Assignment) {
	// First, validate the value expression
	v.validateExpression(assign.Value)

	// Check that the variable is declared
	sym, ok := v.symbols.Lookup(assign.Name)
	if !ok {
		v.errors.Add(errUndefinedVariable(assign.Pos(), assign.Name))
		return
	}

	// Verify it's a variable, not a function
	if sym.Kind == SymbolFunction {
		v.errors.Add(newScopeError(assign.Pos(), "cannot assign to function '"+assign.Name+"'"))
		return
	}

	// Update type if we can infer a more specific type
	newType := v.typeChecker.InferType(assign.Value)
	if newType != TypeUnknown {
		v.symbols.UpdateType(assign.Name, newType)
		v.types[assign.Name] = newType
	}
}

// validateIfStmt validates an if statement.
func (v *Validator) validateIfStmt(ifStmt *ast.IfStmt) {
	// Validate condition expression
	v.validateExpression(ifStmt.Condition)

	// Validate then block in new scope
	v.symbols.EnterScope()
	v.validateBlock(ifStmt.Then)
	v.symbols.ExitScope()

	// Validate else block if present
	if ifStmt.Else != nil {
		v.symbols.EnterScope()
		v.validateBlock(ifStmt.Else)
		v.symbols.ExitScope()
	}
}

// validateForLoop validates a for-in loop.
func (v *Validator) validateForLoop(forLoop *ast.ForLoop) {
	// Validate iterable expression
	v.validateExpression(forLoop.Iterable)

	// Check that iterable is actually iterable (array)
	iterableType := v.typeChecker.InferType(forLoop.Iterable)
	if iterableType != TypeUnknown && !v.typeChecker.IsIterable(iterableType) {
		v.errors.Add(errCannotIterate(forLoop.Pos(), iterableType.String()))
	}

	// Enter new scope for loop body
	v.symbols.EnterScope()

	// Declare loop variable in loop scope
	// Loop variable type is element type of array (unknown for now)
	if err := v.symbols.DeclareWithType(forLoop.Variable, SymbolVariable, TypeUnknown); err != nil {
		v.errors.Add(errDuplicateDeclaration(forLoop.Pos(), forLoop.Variable))
	}

	// Validate body
	v.validateBlock(forLoop.Body)

	v.symbols.ExitScope()
}

// validateFunctionDecl validates a function declaration.
func (v *Validator) validateFunctionDecl(funcDecl *ast.FunctionDecl) {
	// Check for duplicate declaration
	if _, exists := v.symbols.LookupLocal(funcDecl.Name); exists {
		v.errors.Add(errDuplicateDeclaration(funcDecl.Pos(), funcDecl.Name))
		return
	}

	// Declare function
	if err := v.symbols.DeclareFunction(funcDecl.Name, len(funcDecl.Params)); err != nil {
		v.errors.Add(errDuplicateDeclaration(funcDecl.Pos(), funcDecl.Name))
		return
	}

	// Enter new scope for function body
	v.symbols.EnterScope()

	// Check for duplicate parameter names and declare params
	paramNames := make(map[string]bool)
	for _, param := range funcDecl.Params {
		if paramNames[param.Name] {
			v.errors.Add(errDuplicateParameter(funcDecl.Pos(), param.Name))
		} else {
			paramNames[param.Name] = true
			if err := v.symbols.DeclareWithType(param.Name, SymbolParameter, TypeUnknown); err != nil {
				v.errors.Add(errDuplicateDeclaration(funcDecl.Pos(), param.Name))
			}
		}
	}

	// Validate function body
	v.validateBlock(funcDecl.Body)

	v.symbols.ExitScope()
}

// validateExecBlock validates an exec block (shell command).
func (v *Validator) validateExecBlock(exec *ast.ExecBlock) {
	// Exec blocks don't have semantic validation
	// Could add checks for dangerous commands if needed
}

// validateBlock validates a block of statements.
func (v *Validator) validateBlock(block *ast.Block) {
	if block == nil {
		return
	}
	for _, stmt := range block.Statements {
		v.validateStatement(stmt)
	}
}

// validateReturnStmt validates a return statement.
func (v *Validator) validateReturnStmt(ret *ast.ReturnStmt) {
	if ret.Value != nil {
		v.validateExpression(ret.Value)
	}
}

// validateExpression validates an expression and checks all referenced variables.
func (v *Validator) validateExpression(expr ast.Expression) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.Identifier:
		v.validateIdentifier(e)

	case *ast.FunctionCall:
		v.validateFunctionCall(e)

	case *ast.ArrayLiteral:
		for _, elem := range e.Elements {
			v.validateExpression(elem)
		}

	case *ast.BinaryExpr:
		v.validateExpression(e.Left)
		v.validateExpression(e.Right)

	case *ast.UnaryExpr:
		v.validateExpression(e.Operand)

	// Literals don't need validation
	case *ast.StringLiteral, *ast.NumberLiteral, *ast.BoolLiteral:
		// No validation needed
	}
}

// validateIdentifier checks that an identifier is declared.
func (v *Validator) validateIdentifier(ident *ast.Identifier) {
	if _, ok := v.symbols.Lookup(ident.Name); !ok {
		v.errors.Add(errUndefinedVariable(ident.Pos(), ident.Name))
	}
}

// validateFunctionCall validates a function call expression.
func (v *Validator) validateFunctionCall(call *ast.FunctionCall) {
	// Validate arguments first
	for _, arg := range call.Args {
		v.validateExpression(arg)
	}

	// Look up the function
	sym, ok := v.symbols.Lookup(call.Name)
	if !ok {
		v.errors.Add(errUndefinedFunction(call.Pos(), call.Name))
		return
	}

	// Verify it's a function
	if sym.Kind != SymbolFunction {
		v.errors.Add(errNotAFunction(call.Pos(), call.Name))
		return
	}

	// Check argument count
	if len(call.Args) != sym.ParamCount {
		v.errors.Add(errWrongArgCount(call.Pos(), call.Name, sym.ParamCount, len(call.Args)))
	}
}

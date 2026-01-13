// Package validator provides semantic validation for CodeAI AST.
package validator

import (
	"github.com/bargom/codeai/internal/ast"
)

// Validator performs semantic analysis on CodeAI AST.
type Validator struct {
	symbols        *SymbolTable
	typeChecker    *TypeChecker
	errors         *ValidationErrors
	types          map[string]Type // Track variable types for testing
	configDecl     *ast.ConfigDecl // Track config declaration for validation
	databaseBlocks []*ast.DatabaseBlock // Track database blocks for validation
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

	// Validate that database_type in config matches declared database blocks
	v.validateDatabaseTypeConsistency()

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
	case *ast.ConfigDecl:
		v.validateConfigDecl(s)
	case *ast.DatabaseBlock:
		v.validateDatabaseBlock(s)
	case *ast.ModelDecl:
		v.validateModelDecl(s)
	case *ast.CollectionDecl:
		v.validateCollectionDecl(s)
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

// validateConfigDecl validates a config block declaration.
func (v *Validator) validateConfigDecl(cfg *ast.ConfigDecl) {
	// Check for duplicate config declaration
	if v.configDecl != nil {
		v.errors.Add(newSemanticError(cfg.Pos(), "duplicate config declaration; only one config block allowed"))
		return
	}
	v.configDecl = cfg

	// Validate database_type value
	if cfg.DatabaseType != ast.DatabaseTypePostgres && cfg.DatabaseType != ast.DatabaseTypeMongoDB {
		v.errors.Add(newSemanticError(cfg.Pos(),
			"invalid database_type: must be 'postgres' or 'mongodb'"))
	}

	// If MongoDB is specified, validate required MongoDB config fields
	if cfg.DatabaseType == ast.DatabaseTypeMongoDB {
		if cfg.MongoDBURI == "" {
			v.errors.Add(newSemanticError(cfg.Pos(),
				"mongodb_uri is required when database_type is 'mongodb'"))
		}
		if cfg.MongoDBName == "" {
			v.errors.Add(newSemanticError(cfg.Pos(),
				"mongodb_database is required when database_type is 'mongodb'"))
		}
	}
}

// validateDatabaseBlock validates a database block declaration.
func (v *Validator) validateDatabaseBlock(db *ast.DatabaseBlock) {
	v.databaseBlocks = append(v.databaseBlocks, db)

	// Validate nested statements
	for _, stmt := range db.Statements {
		v.validateStatement(stmt)
	}
}

// validateDatabaseTypeConsistency validates that database_type in config
// matches the declared database blocks.
func (v *Validator) validateDatabaseTypeConsistency() {
	if v.configDecl == nil && len(v.databaseBlocks) == 0 {
		// No config or database blocks - nothing to validate
		return
	}

	// Default database type is postgres
	configDBType := ast.DatabaseTypePostgres
	if v.configDecl != nil {
		configDBType = v.configDecl.DatabaseType
	}

	// Check that all database blocks match the config database_type
	for _, db := range v.databaseBlocks {
		if db.DBType != configDBType {
			var pos ast.Position
			if v.configDecl != nil {
				pos = v.configDecl.Pos()
			} else {
				pos = db.Pos()
			}
			v.errors.Add(newSemanticError(pos,
				"database block type '"+string(db.DBType)+
					"' does not match config database_type '"+string(configDBType)+
					"'; all database blocks must match the configured database type"))
		}
	}

	// If config specifies mongodb, ensure at least one mongodb block exists
	if v.configDecl != nil && v.configDecl.DatabaseType == ast.DatabaseTypeMongoDB {
		hasMongoBlock := false
		for _, db := range v.databaseBlocks {
			if db.DBType == ast.DatabaseTypeMongoDB {
				hasMongoBlock = true
				break
			}
		}
		if !hasMongoBlock && len(v.databaseBlocks) > 0 {
			v.errors.Add(newSemanticError(v.configDecl.Pos(),
				"config specifies database_type 'mongodb' but no mongodb database block found"))
		}
	}
}

// =============================================================================
// PostgreSQL Model Validation
// =============================================================================

// validateModelDecl validates a PostgreSQL model declaration.
func (v *Validator) validateModelDecl(model *ast.ModelDecl) {
	// Track field names for duplicate checking
	fieldNames := make(map[string]bool)

	for _, field := range model.Fields {
		// Check for duplicate field names
		if fieldNames[field.Name] {
			v.errors.Add(newSemanticError(field.Pos(),
				"duplicate field '"+field.Name+"' in model '"+model.Name+"'"))
		}
		fieldNames[field.Name] = true

		// Validate field type
		v.validateFieldType(field.FieldType, model.Name)

		// Validate modifiers
		v.validateModifiers(field.Modifiers, model.Name, field.Name)
	}

	// Validate indexes
	for _, idx := range model.Indexes {
		for _, fieldName := range idx.Fields {
			if !fieldNames[fieldName] {
				v.errors.Add(newSemanticError(idx.Pos(),
					"index references unknown field '"+fieldName+"' in model '"+model.Name+"'"))
			}
		}
	}
}

// validateFieldType validates a PostgreSQL field type reference.
func (v *Validator) validateFieldType(typeRef *ast.TypeRef, modelName string) {
	if typeRef == nil {
		return
	}

	// Valid PostgreSQL types
	validTypes := map[string]bool{
		"uuid": true, "string": true, "text": true, "integer": true, "int": true,
		"decimal": true, "boolean": true, "bool": true, "timestamp": true,
		"date": true, "time": true, "json": true, "jsonb": true,
		"list": true, "array": true, "ref": true, "enum": true,
	}

	if !validTypes[typeRef.Name] {
		v.errors.Add(newSemanticError(typeRef.Pos(),
			"unknown type '"+typeRef.Name+"' in model '"+modelName+"'; valid types: uuid, string, text, integer, decimal, boolean, timestamp, date, time, json, list, ref, enum"))
	}

	// Validate type parameters
	for _, param := range typeRef.Params {
		v.validateFieldType(param, modelName)
	}
}

// validateModifiers validates field modifiers.
func (v *Validator) validateModifiers(modifiers []*ast.Modifier, modelName, fieldName string) {
	seenModifiers := make(map[string]bool)

	for _, mod := range modifiers {
		// Check for duplicate modifiers
		if seenModifiers[mod.Name] {
			v.errors.Add(newSemanticError(mod.Pos(),
				"duplicate modifier '"+mod.Name+"' on field '"+fieldName+"' in model '"+modelName+"'"))
		}
		seenModifiers[mod.Name] = true

		// Validate modifier values
		if mod.Value != nil {
			v.validateExpression(mod.Value)
		}
	}

	// Check for conflicting modifiers
	if seenModifiers["required"] && seenModifiers["optional"] {
		v.errors.Add(newSemanticError(modifiers[0].Pos(),
			"field '"+fieldName+"' cannot be both required and optional"))
	}
}

// =============================================================================
// MongoDB Collection Validation
// =============================================================================

// validateCollectionDecl validates a MongoDB collection declaration.
func (v *Validator) validateCollectionDecl(coll *ast.CollectionDecl) {
	// Track field names for duplicate checking
	fieldNames := make(map[string]bool)

	for _, field := range coll.Fields {
		// Check for duplicate field names
		if fieldNames[field.Name] {
			v.errors.Add(newSemanticError(field.Pos(),
				"duplicate field '"+field.Name+"' in collection '"+coll.Name+"'"))
		}
		fieldNames[field.Name] = true

		// Validate field type
		v.validateMongoFieldType(field.FieldType, coll.Name, field.Name, 0)

		// Validate modifiers - check for invalid modifiers in MongoDB
		v.validateMongoModifiers(field.Modifiers, coll.Name, field.Name)
	}

	// Validate indexes
	for _, idx := range coll.Indexes {
		for _, fieldName := range idx.Fields {
			if !fieldNames[fieldName] {
				v.errors.Add(newSemanticError(idx.Pos(),
					"index references unknown field '"+fieldName+"' in collection '"+coll.Name+"'"))
			}
		}
	}
}

// validateMongoFieldType validates a MongoDB field type reference.
func (v *Validator) validateMongoFieldType(typeRef *ast.MongoTypeRef, collName, fieldName string, depth int) {
	if typeRef == nil {
		return
	}

	// MongoDB document nesting limit (16 levels is MongoDB's limit, we enforce 10 for safety)
	const maxNestingDepth = 10
	if depth > maxNestingDepth {
		v.errors.Add(newSemanticError(typeRef.Pos(),
			"embedded document nesting exceeds maximum depth of "+
				string(rune('0'+maxNestingDepth))+" in collection '"+collName+"', field '"+fieldName+"'"))
		return
	}

	// Handle embedded documents
	if typeRef.EmbeddedDoc != nil {
		for _, embField := range typeRef.EmbeddedDoc.Fields {
			v.validateMongoFieldType(embField.FieldType, collName, embField.Name, depth+1)
			v.validateMongoModifiers(embField.Modifiers, collName, embField.Name)
		}
		return
	}

	// Valid MongoDB types
	validTypes := map[string]bool{
		"objectid": true, "string": true, "int": true, "int32": true, "int64": true,
		"double": true, "decimal": true, "bool": true, "boolean": true,
		"date": true, "timestamp": true, "binary": true, "regex": true,
		"array": true, "object": true, "null": true, "mixed": true,
	}

	if !validTypes[typeRef.Name] {
		v.errors.Add(newSemanticError(typeRef.Pos(),
			"unknown MongoDB type '"+typeRef.Name+"' in collection '"+collName+"', field '"+fieldName+"'; "+
				"valid types: objectid, string, int, int32, int64, double, decimal, bool, date, timestamp, binary, array, object"))
	}
}

// validateMongoModifiers validates MongoDB field modifiers.
func (v *Validator) validateMongoModifiers(modifiers []*ast.Modifier, collName, fieldName string) {
	seenModifiers := make(map[string]bool)

	// Modifiers that are NOT valid for MongoDB (relational concepts)
	invalidMongoModifiers := map[string]bool{
		"foreign_key": true,
		"references":  true,
		"on_delete":   true,
		"on_update":   true,
		"cascade":     true,
	}

	for _, mod := range modifiers {
		// Check for invalid MongoDB modifiers
		if invalidMongoModifiers[mod.Name] {
			v.errors.Add(newSemanticError(mod.Pos(),
				"modifier '"+mod.Name+"' is not valid for MongoDB collections; "+
					"foreign keys and referential constraints are not supported"))
		}

		// Check for duplicate modifiers
		if seenModifiers[mod.Name] {
			v.errors.Add(newSemanticError(mod.Pos(),
				"duplicate modifier '"+mod.Name+"' on field '"+fieldName+"' in collection '"+collName+"'"))
		}
		seenModifiers[mod.Name] = true

		// Validate modifier values
		if mod.Value != nil {
			v.validateExpression(mod.Value)
		}
	}

	// Check for conflicting modifiers
	if seenModifiers["required"] && seenModifiers["optional"] {
		v.errors.Add(newSemanticError(modifiers[0].Pos(),
			"field '"+fieldName+"' cannot be both required and optional"))
	}
}

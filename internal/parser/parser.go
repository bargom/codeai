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
		{Name: "Config", Pattern: `\bconfig\b`, Action: nil},
		{Name: "Var", Pattern: `\bvar\b`, Action: nil},
		{Name: "If", Pattern: `\bif\b`, Action: nil},
		{Name: "Else", Pattern: `\belse\b`, Action: nil},
		{Name: "For", Pattern: `\bfor\b`, Action: nil},
		{Name: "In", Pattern: `\bin\b`, Action: nil},
		{Name: "Func", Pattern: `\bfunction\b`, Action: nil},
		{Name: "True", Pattern: `\btrue\b`, Action: nil},
		{Name: "False", Pattern: `\bfalse\b`, Action: nil},

		// Database keywords
		{Name: "Database", Pattern: `\bdatabase\b`, Action: nil},
		{Name: "Postgres", Pattern: `\bpostgres\b`, Action: nil},
		{Name: "MongoDB", Pattern: `\bmongodb\b`, Action: nil},
		{Name: "Model", Pattern: `\bmodel\b`, Action: nil},
		{Name: "Collection", Pattern: `\bcollection\b`, Action: nil},
		{Name: "Indexes", Pattern: `\bindexes\b`, Action: nil},
		{Name: "Index", Pattern: `\bindex\b`, Action: nil},
		{Name: "Unique", Pattern: `\bunique\b`, Action: nil},
		{Name: "Text", Pattern: `\btext\b`, Action: nil},
		{Name: "Geospatial", Pattern: `\bgeospatial\b`, Action: nil},
		{Name: "Embedded", Pattern: `\bembedded\b`, Action: nil},
		{Name: "Required", Pattern: `\brequired\b`, Action: nil},
		{Name: "Optional", Pattern: `\boptional\b`, Action: nil},
		{Name: "Primary", Pattern: `\bprimary\b`, Action: nil},
		{Name: "Auto", Pattern: `\bauto\b`, Action: nil},
		{Name: "Default", Pattern: `\bdefault\b`, Action: nil},
		{Name: "Description", Pattern: `\bdescription\b`, Action: nil},

		// Literals
		{Name: "Number", Pattern: `[0-9]+\.?[0-9]*`, Action: nil},
		{Name: "String", Pattern: `"([^"\\]|\\.)*"`, Action: nil},

		// Identifiers
		{Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`, Action: nil},

		// Operators and punctuation
		{Name: "Equals", Pattern: `=`, Action: nil},
		{Name: "Colon", Pattern: `:`, Action: nil},
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
	Pos           lexer.Position
	ConfigDecl    *pConfigDecl    `  @@`
	DatabaseBlock *pDatabaseBlock `| @@`
	VarDecl       *pVarDecl       `| @@`
	IfStmt        *pIfStmt        `| @@`
	ForLoop       *pForLoop       `| @@`
	FuncDecl      *pFuncDecl      `| @@`
	ExecBlock     *pExecBlock     `| @@`
	Assignment    *pAssignment    `| @@`
}

// pVarDecl is the Participle grammar for variable declaration.
// Note: Variable names can be keywords, so we accept both Ident and keyword tokens.
type pVarDecl struct {
	Pos   lexer.Position
	Name  string       `Var @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description)`
	Value *pExpression `Equals @@`
}

// pAssignment is the Participle grammar for assignment.
// Note: Variable names can be keywords, so we accept both Ident and keyword tokens.
type pAssignment struct {
	Pos   lexer.Position
	Name  string       `@(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description)`
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
	Variable string        `For @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description)`
	Iterable *pExpression  `In @@`
	Body     []*pStatement `LBrace @@* RBrace`
}

// pFuncDecl is the Participle grammar for function declaration.
type pFuncDecl struct {
	Pos    lexer.Position
	Name   string        `Func @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description)`
	Params []string      `LParen ( @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description) ( Comma @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description) )* )? RParen`
	Body   []*pStatement `LBrace @@* RBrace`
}

// pExecBlock is the Participle grammar for exec block.
type pExecBlock struct {
	Pos     lexer.Position
	Command string `ExecOpen @ShellCommand ShellClose`
}

// pConfigDecl is the Participle grammar for config block.
// Example: config { database_type: "mongodb" mongodb_uri: "..." }
type pConfigDecl struct {
	Pos        lexer.Position
	Properties []*pConfigProperty `Config LBrace @@* RBrace`
}

// pConfigProperty is a key-value pair in a config block.
type pConfigProperty struct {
	Pos   lexer.Position
	Key   string       `@(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description) Colon`
	Value *pExpression `@@`
}

// pDatabaseBlock is the Participle grammar for database blocks.
// Example: database mongodb { collection ... } or database postgres { model ... }
type pDatabaseBlock struct {
	Pos         lexer.Position
	Type        string             `Database @( Postgres | MongoDB ) LBrace`
	Models      []*pModelDecl      `@@*`
	Collections []*pCollectionDecl `@@* RBrace`
}

// =============================================================================
// PostgreSQL Model Grammar
// =============================================================================

// pModelDecl represents a PostgreSQL model declaration.
// Example: model User { id: uuid, primary, auto }
type pModelDecl struct {
	Pos         lexer.Position
	Name        string        `Model @Ident LBrace`
	Description *string       `(Description Colon @String)?`
	Fields      []*pFieldDecl `@@*`
	Indexes     []*pIndexDecl `@@* RBrace`
}

// pFieldDecl represents a field in a PostgreSQL model.
// Note: Field names can be keywords, so we accept both Ident and keyword tokens.
type pFieldDecl struct {
	Pos       lexer.Position
	Name      string       `@(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description) Colon`
	Type      *pTypeRef    `@@`
	Modifiers []*pModifier `(Comma @@)*`
}

// pTypeRef represents a type reference (e.g., string, uuid, ref(User)).
type pTypeRef struct {
	Pos    lexer.Position
	Name   string      `@Ident`
	Params []*pTypeRef `(LParen @@ (Comma @@)* RParen)?`
}

// pModifier represents a field modifier (e.g., required, unique, default(value)).
type pModifier struct {
	Pos   lexer.Position
	Name  string       `@(Required | Optional | Unique | Primary | Auto | Default | Text | Geospatial | Ident)`
	Value *pExpression `(LParen @@ RParen)?`
}

// pIndexDecl represents a PostgreSQL index declaration.
// Note: Index field names can be keywords, so we accept both Ident and keyword tokens.
type pIndexDecl struct {
	Pos    lexer.Position
	Fields []string `Index Colon LBracket @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description) (Comma @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description))* RBracket`
	Unique bool     `@Unique?`
}

// =============================================================================
// MongoDB Collection Grammar
// =============================================================================

// pCollectionDecl represents a MongoDB collection declaration.
// Example: collection User { _id: objectid, primary }
type pCollectionDecl struct {
	Pos         lexer.Position
	Name        string               `Collection @Ident LBrace`
	Description *string              `(Description Colon @String)?`
	Fields      []*pMongoFieldDecl   `@@*`
	Indexes     *pMongoIndexesBlock  `@@? RBrace`
}

// pMongoFieldDecl represents a field in a MongoDB collection.
// Note: Field names can be keywords, so we accept both Ident and keyword tokens.
type pMongoFieldDecl struct {
	Pos       lexer.Position
	Name      string         `@(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description) Colon`
	Type      *pMongoTypeRef `@@`
	Modifiers []*pModifier   `(Comma @@)*`
}

// pMongoTypeRef represents a MongoDB-specific type reference.
// Supports: objectid, string, int, double, bool, date, binary, array(T), embedded { ... }
type pMongoTypeRef struct {
	Pos         lexer.Position
	EmbeddedDoc *pEmbeddedDoc `  @@`
	Name        string        `| @Ident`
	Params      []string      `(LParen @Ident (Comma @Ident)* RParen)?`
}

// pEmbeddedDoc represents an embedded document type in MongoDB.
type pEmbeddedDoc struct {
	Pos    lexer.Position
	Fields []*pMongoFieldDecl `Embedded LBrace @@* RBrace`
}

// pMongoIndexesBlock represents the indexes block in a MongoDB collection.
type pMongoIndexesBlock struct {
	Pos     lexer.Position
	Indexes []*pMongoIndexDecl `Indexes LBrace @@* RBrace`
}

// pMongoIndexDecl represents a MongoDB index declaration.
// Supports: single, compound, text, geospatial indexes
// Note: Index field names can be keywords, so we accept both Ident and keyword tokens.
type pMongoIndexDecl struct {
	Pos       lexer.Position
	Fields    []string `Index Colon LBracket @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description) (Comma @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description))* RBracket`
	Unique    bool     `@Unique?`
	IndexKind string   `@(Text | Geospatial)?`
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
	Identifier   *string        `| @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description)`
}

// pArrayLiteral is the Participle grammar for array literals.
type pArrayLiteral struct {
	Pos      lexer.Position
	Elements []*pExpression `LBracket ( @@ ( Comma @@ )* )? RBracket`
}

// pFuncCall is the Participle grammar for function calls.
type pFuncCall struct {
	Pos       lexer.Position
	Name      string         `@(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description)`
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
	case s.ConfigDecl != nil:
		return convertConfigDecl(s.ConfigDecl)
	case s.DatabaseBlock != nil:
		return convertDatabaseBlock(s.DatabaseBlock)
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

func convertConfigDecl(c *pConfigDecl) *ast.ConfigDecl {
	props := make(map[string]ast.Expression)
	var dbType ast.DatabaseType = ast.DatabaseTypePostgres // default
	var mongoURI, mongoDBName string

	for _, prop := range c.Properties {
		expr := convertExpression(prop.Value)
		props[prop.Key] = expr

		// Extract well-known properties
		if strLit, ok := expr.(*ast.StringLiteral); ok {
			switch prop.Key {
			case "database_type":
				switch strLit.Value {
				case "postgres":
					dbType = ast.DatabaseTypePostgres
				case "mongodb":
					dbType = ast.DatabaseTypeMongoDB
				}
			case "mongodb_uri":
				mongoURI = strLit.Value
			case "mongodb_database":
				mongoDBName = strLit.Value
			}
		}
	}

	return createConfigDecl(dbType, mongoURI, mongoDBName, props)
}

func convertDatabaseBlock(d *pDatabaseBlock) *ast.DatabaseBlock {
	var dbType ast.DatabaseType
	switch d.Type {
	case "postgres":
		dbType = ast.DatabaseTypePostgres
	case "mongodb":
		dbType = ast.DatabaseTypeMongoDB
	default:
		dbType = ast.DatabaseTypePostgres
	}

	// Collect all models and collections as statements
	stmts := make([]ast.Statement, 0, len(d.Models)+len(d.Collections))
	for _, m := range d.Models {
		stmts = append(stmts, convertModelDecl(m))
	}
	for _, c := range d.Collections {
		stmts = append(stmts, convertCollectionDecl(c))
	}

	return createDatabaseBlock(dbType, stmts)
}

// =============================================================================
// PostgreSQL Model Conversion Functions
// =============================================================================

func convertModelDecl(m *pModelDecl) *ast.ModelDecl {
	fields := make([]*ast.FieldDecl, len(m.Fields))
	for i, f := range m.Fields {
		fields[i] = convertFieldDecl(f)
	}

	indexes := make([]*ast.IndexDecl, len(m.Indexes))
	for i, idx := range m.Indexes {
		indexes[i] = convertIndexDecl(idx)
	}

	desc := ""
	if m.Description != nil {
		desc = unquote(*m.Description)
	}

	return &ast.ModelDecl{
		Name:        m.Name,
		Description: desc,
		Fields:      fields,
		Indexes:     indexes,
	}
}

func convertFieldDecl(f *pFieldDecl) *ast.FieldDecl {
	modifiers := make([]*ast.Modifier, len(f.Modifiers))
	for i, mod := range f.Modifiers {
		modifiers[i] = convertModifier(mod)
	}

	return &ast.FieldDecl{
		Name:      f.Name,
		FieldType: convertTypeRef(f.Type),
		Modifiers: modifiers,
	}
}

func convertTypeRef(t *pTypeRef) *ast.TypeRef {
	if t == nil {
		return nil
	}

	params := make([]*ast.TypeRef, len(t.Params))
	for i, p := range t.Params {
		params[i] = convertTypeRef(p)
	}

	return &ast.TypeRef{
		Name:   t.Name,
		Params: params,
	}
}

func convertModifier(m *pModifier) *ast.Modifier {
	var value ast.Expression
	if m.Value != nil {
		value = convertExpression(m.Value)
	}

	return &ast.Modifier{
		Name:  m.Name,
		Value: value,
	}
}

func convertIndexDecl(idx *pIndexDecl) *ast.IndexDecl {
	return &ast.IndexDecl{
		Fields: idx.Fields,
		Unique: idx.Unique,
	}
}

// =============================================================================
// MongoDB Collection Conversion Functions
// =============================================================================

func convertCollectionDecl(c *pCollectionDecl) *ast.CollectionDecl {
	fields := make([]*ast.MongoFieldDecl, len(c.Fields))
	for i, f := range c.Fields {
		fields[i] = convertMongoFieldDecl(f)
	}

	var indexes []*ast.MongoIndexDecl
	if c.Indexes != nil {
		indexes = make([]*ast.MongoIndexDecl, len(c.Indexes.Indexes))
		for i, idx := range c.Indexes.Indexes {
			indexes[i] = convertMongoIndexDecl(idx)
		}
	}

	desc := ""
	if c.Description != nil {
		desc = unquote(*c.Description)
	}

	return &ast.CollectionDecl{
		Name:        c.Name,
		Description: desc,
		Fields:      fields,
		Indexes:     indexes,
	}
}

func convertMongoFieldDecl(f *pMongoFieldDecl) *ast.MongoFieldDecl {
	modifiers := make([]*ast.Modifier, len(f.Modifiers))
	for i, mod := range f.Modifiers {
		modifiers[i] = convertModifier(mod)
	}

	return &ast.MongoFieldDecl{
		Name:      f.Name,
		FieldType: convertMongoTypeRef(f.Type),
		Modifiers: modifiers,
	}
}

func convertMongoTypeRef(t *pMongoTypeRef) *ast.MongoTypeRef {
	if t == nil {
		return nil
	}

	if t.EmbeddedDoc != nil {
		return &ast.MongoTypeRef{
			EmbeddedDoc: convertEmbeddedDoc(t.EmbeddedDoc),
		}
	}

	return &ast.MongoTypeRef{
		Name:   t.Name,
		Params: t.Params,
	}
}

func convertEmbeddedDoc(e *pEmbeddedDoc) *ast.EmbeddedDocDecl {
	fields := make([]*ast.MongoFieldDecl, len(e.Fields))
	for i, f := range e.Fields {
		fields[i] = convertMongoFieldDecl(f)
	}

	return &ast.EmbeddedDocDecl{
		Fields: fields,
	}
}

func convertMongoIndexDecl(idx *pMongoIndexDecl) *ast.MongoIndexDecl {
	return &ast.MongoIndexDecl{
		Fields:    idx.Fields,
		Unique:    idx.Unique,
		IndexKind: idx.IndexKind,
	}
}

// unquote removes surrounding quotes from a string if present.
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
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

func createConfigDecl(dbType ast.DatabaseType, mongoURI, mongoDBName string, props map[string]ast.Expression) *ast.ConfigDecl {
	return &ast.ConfigDecl{
		DatabaseType: dbType,
		MongoDBURI:   mongoURI,
		MongoDBName:  mongoDBName,
		Properties:   props,
	}
}

func createDatabaseBlock(dbType ast.DatabaseType, stmts []ast.Statement) *ast.DatabaseBlock {
	return &ast.DatabaseBlock{
		DBType:     dbType,
		Statements: stmts,
	}
}

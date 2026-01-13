// Package parser provides endpoint parsing support for the CodeAI DSL.
package parser

import (
	"strconv"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"

	"github.com/bargom/codeai/internal/ast"
)

// =============================================================================
// Endpoint Lexer Definition
// =============================================================================

var endpointLexer = lexer.MustStateful(lexer.Rules{
	"Root": {
		// Whitespace and comments
		{Name: "whitespace", Pattern: `[\s]+`, Action: nil},
		{Name: "SingleLineComment", Pattern: `//[^\n]*`, Action: nil},
		{Name: "MultiLineComment", Pattern: `/\*([^*]|\*[^/])*\*/`, Action: nil},

		// Endpoint keywords
		{Name: "Endpoint", Pattern: `\bendpoint\b`, Action: nil},
		{Name: "GET", Pattern: `\bGET\b`, Action: nil},
		{Name: "POST", Pattern: `\bPOST\b`, Action: nil},
		{Name: "PUT", Pattern: `\bPUT\b`, Action: nil},
		{Name: "DELETE", Pattern: `\bDELETE\b`, Action: nil},
		{Name: "PATCH", Pattern: `\bPATCH\b`, Action: nil},
		{Name: "Request", Pattern: `\brequest\b`, Action: nil},
		{Name: "Response", Pattern: `\bresponse\b`, Action: nil},
		{Name: "From", Pattern: `\bfrom\b`, Action: nil},
		{Name: "Body", Pattern: `\bbody\b`, Action: nil},
		{Name: "Query", Pattern: `\bquery\b`, Action: nil},
		{Name: "Path", Pattern: `\bpath\b`, Action: nil},
		{Name: "Header", Pattern: `\bheader\b`, Action: nil},
		{Name: "Status", Pattern: `\bstatus\b`, Action: nil},
		{Name: "Do", Pattern: `\bdo\b`, Action: nil},
		{Name: "Where", Pattern: `\bwhere\b`, Action: nil},
		{Name: "With", Pattern: `\bwith\b`, Action: nil},
		{Name: "Middleware", Pattern: `\bmiddleware\b`, Action: nil},
		{Name: "RequireRole", Pattern: `\brequire_role\b`, Action: nil},

		// Literals
		{Name: "Number", Pattern: `[0-9]+\.?[0-9]*`, Action: nil},
		{Name: "String", Pattern: `"([^"\\]|\\.)*"`, Action: nil},

		// Identifiers
		{Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`, Action: nil},

		// Operators and punctuation
		{Name: "At", Pattern: `@`, Action: nil},
		{Name: "Equals", Pattern: `=`, Action: nil},
		{Name: "Colon", Pattern: `:`, Action: nil},
		{Name: "Dot", Pattern: `\.`, Action: nil},
		{Name: "LBracket", Pattern: `\[`, Action: nil},
		{Name: "RBracket", Pattern: `\]`, Action: nil},
		{Name: "LParen", Pattern: `\(`, Action: nil},
		{Name: "RParen", Pattern: `\)`, Action: nil},
		{Name: "LBrace", Pattern: `\{`, Action: nil},
		{Name: "RBrace", Pattern: `\}`, Action: nil},
		{Name: "Comma", Pattern: `,`, Action: nil},
	},
})

// =============================================================================
// Endpoint Grammar Structs
// =============================================================================

// pEndpointFile represents a file containing endpoint declarations.
type pEndpointFile struct {
	Pos       lexer.Position
	Endpoints []*pEndpointDecl `parser:"@@*"`
}

// pEndpointDecl is the Participle grammar for endpoint declaration.
// Example: endpoint GET "/users/:id" { middleware auth require_role "admin" request User from path response UserDetail status 200 }
type pEndpointDecl struct {
	Pos         lexer.Position
	Annotations []*pAnnotation    `parser:"@@*"`
	Method      string            `parser:"Endpoint @( GET | POST | PUT | DELETE | PATCH )"`
	Path        string            `parser:"@String LBrace"`
	Middlewares []*pMiddlewareRef `parser:"@@*"`
	RequireRole *string           `parser:"( \"require_role\" @String )?"`
	Request     *pRequestType     `parser:"@@?"`
	Response    *pResponseType    `parser:"@@?"`
	Logic       *pHandlerLogic    `parser:"@@? RBrace"`
}

// pMiddlewareRef is a reference to a middleware.
type pMiddlewareRef struct {
	Pos  lexer.Position
	Name string `parser:"Middleware @Ident"`
}

// pRequestType defines the request type and source for an endpoint.
// Example: request User from body
type pRequestType struct {
	Pos      lexer.Position
	TypeName string `parser:"Request @Ident"`
	Source   string `parser:"From @( Body | Query | Path | Header )"`
}

// pResponseType defines the response type and status code for an endpoint.
// Example: response UserDetail status 200
type pResponseType struct {
	Pos        lexer.Position
	TypeName   string `parser:"Response @Ident"`
	StatusCode string `parser:"Status @Number"`
}

// pHandlerLogic represents the logic block inside an endpoint handler.
// Example: do { validate(request) ... }
type pHandlerLogic struct {
	Pos   lexer.Position
	Steps []*pLogicStep `parser:"Do LBrace @@* RBrace"`
}

// pLogicStep represents a single step in handler logic.
// Example: validate(request), authorize(request, "admin"), user = db.find(User, request.id)
type pLogicStep struct {
	Pos       lexer.Position
	Target    *string    `parser:"( @Ident Equals )?"`
	Action    string     `parser:"@Ident"`
	Args      []string   `parser:"LParen ( @( Ident | String | Request | Response ) ( Comma @( Ident | String | Request | Response ) )* )? RParen"`
	Condition *string    `parser:"( Where @String )?"`
	Options   []*pOption `parser:"@@?"`
}

// pOption represents a key-value option in a logic step.
// Example: with { cache: true, ttl: 300 }
type pOption struct {
	Pos   lexer.Position
	Key   string `parser:"With LBrace @Ident Colon"`
	Value string `parser:"@( Ident | Number | String ) RBrace"`
}

// pAnnotation represents a metadata annotation on an endpoint.
// Example: @deprecated, @auth("admin")
type pAnnotation struct {
	Pos   lexer.Position
	Name  string  `parser:"At @Ident"`
	Value *string `parser:"( LParen @String RParen )?"`
}

// =============================================================================
// Endpoint Parser Instance
// =============================================================================

var endpointParser = participle.MustBuild[pEndpointFile](
	participle.Lexer(endpointLexer),
	participle.Elide("whitespace", "SingleLineComment", "MultiLineComment"),
	participle.UseLookahead(10),
)

// =============================================================================
// Public API
// =============================================================================

// ParseEndpoints parses a string containing endpoint declarations.
func ParseEndpoints(input string) ([]*ast.EndpointDecl, error) {
	parsed, err := endpointParser.ParseString("", input)
	if err != nil {
		return nil, err
	}
	return convertEndpoints(parsed), nil
}

// ParseEndpoint parses a single endpoint declaration string.
func ParseEndpoint(input string) (*ast.EndpointDecl, error) {
	endpoints, err := ParseEndpoints(input)
	if err != nil {
		return nil, err
	}
	if len(endpoints) == 0 {
		return nil, nil
	}
	return endpoints[0], nil
}

// =============================================================================
// Conversion Functions
// =============================================================================

func convertEndpoints(f *pEndpointFile) []*ast.EndpointDecl {
	endpoints := make([]*ast.EndpointDecl, len(f.Endpoints))
	for i, e := range f.Endpoints {
		endpoints[i] = convertEndpointDecl(e)
	}
	return endpoints
}

func convertEndpointDecl(e *pEndpointDecl) *ast.EndpointDecl {
	var method ast.HTTPMethod
	switch e.Method {
	case "GET":
		method = ast.HTTPMethodGET
	case "POST":
		method = ast.HTTPMethodPOST
	case "PUT":
		method = ast.HTTPMethodPUT
	case "DELETE":
		method = ast.HTTPMethodDELETE
	case "PATCH":
		method = ast.HTTPMethodPATCH
	default:
		method = ast.HTTPMethodGET
	}

	// Convert middlewares
	middlewares := make([]*ast.MiddlewareRef, len(e.Middlewares))
	for i, m := range e.Middlewares {
		middlewares[i] = &ast.MiddlewareRef{Name: m.Name}
	}

	// Convert annotations
	annotations := make([]*ast.Annotation, len(e.Annotations))
	for i, a := range e.Annotations {
		annotations[i] = convertAnnotation(a)
	}

	// Build handler
	handler := &ast.Handler{}
	if e.Request != nil {
		handler.Request = convertRequestType(e.Request)
	}
	if e.Response != nil {
		handler.Response = convertResponseType(e.Response)
	}
	if e.Logic != nil {
		handler.Logic = convertHandlerLogic(e.Logic)
	}

	requireRole := ""
	if e.RequireRole != nil {
		requireRole = unquote(*e.RequireRole)
	}

	return &ast.EndpointDecl{
		Method:      method,
		Path:        unquote(e.Path),
		Handler:     handler,
		Middlewares: middlewares,
		RequireRole: requireRole,
		Annotations: annotations,
	}
}

func convertRequestType(r *pRequestType) *ast.RequestType {
	var source ast.RequestSource
	switch r.Source {
	case "body":
		source = ast.RequestSourceBody
	case "query":
		source = ast.RequestSourceQuery
	case "path":
		source = ast.RequestSourcePath
	case "header":
		source = ast.RequestSourceHeader
	default:
		source = ast.RequestSourceBody
	}

	return &ast.RequestType{
		TypeName: r.TypeName,
		Source:   source,
	}
}

func convertResponseType(r *pResponseType) *ast.ResponseType {
	status, _ := strconv.Atoi(r.StatusCode)
	return &ast.ResponseType{
		TypeName:   r.TypeName,
		StatusCode: status,
	}
}

func convertHandlerLogic(l *pHandlerLogic) *ast.HandlerLogic {
	steps := make([]*ast.LogicStep, len(l.Steps))
	for i, s := range l.Steps {
		steps[i] = convertLogicStep(s)
	}
	return &ast.HandlerLogic{
		Steps: steps,
	}
}

func convertLogicStep(s *pLogicStep) *ast.LogicStep {
	// Convert args, unquoting strings
	args := make([]string, len(s.Args))
	for i, arg := range s.Args {
		args[i] = unquote(arg)
	}

	// Convert options
	options := make([]*ast.Option, len(s.Options))
	for i, o := range s.Options {
		options[i] = convertOption(o)
	}

	target := ""
	if s.Target != nil {
		target = *s.Target
	}

	condition := ""
	if s.Condition != nil {
		condition = unquote(*s.Condition)
	}

	return &ast.LogicStep{
		Target:    target,
		Action:    s.Action,
		Args:      args,
		Condition: condition,
		Options:   options,
	}
}

func convertOption(o *pOption) *ast.Option {
	return &ast.Option{
		Key:   o.Key,
		Value: createStringLiteral(unquote(o.Value)),
	}
}

func convertAnnotation(a *pAnnotation) *ast.Annotation {
	value := ""
	if a.Value != nil {
		value = unquote(*a.Value)
	}
	return &ast.Annotation{
		Name:  a.Name,
		Value: value,
	}
}

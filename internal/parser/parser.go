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

		// Auth and middleware keywords
		{Name: "Auth", Pattern: `\bauth\b`, Action: nil},
		{Name: "Role", Pattern: `\brole\b`, Action: nil},
		{Name: "Middleware", Pattern: `\bmiddleware\b`, Action: nil},
		{Name: "Method", Pattern: `\bmethod\b`, Action: nil},
		{Name: "JwksUrl", Pattern: `\bjwks_url\b`, Action: nil},
		{Name: "Issuer", Pattern: `\bissuer\b`, Action: nil},
		{Name: "Audience", Pattern: `\baudience\b`, Action: nil},
		{Name: "Permissions", Pattern: `\bpermissions\b`, Action: nil},
		{Name: "Type", Pattern: `\btype\b`, Action: nil},
		{Name: "Provider", Pattern: `\bprovider\b`, Action: nil},
		{Name: "Jwt", Pattern: `\bjwt\b`, Action: nil},
		{Name: "Oauth2", Pattern: `\boauth2\b`, Action: nil},
		{Name: "Apikey", Pattern: `\bapikey\b`, Action: nil},
		{Name: "Basic", Pattern: `\bbasic\b`, Action: nil},

		// Event keywords
		{Name: "Event", Pattern: `\bevent\b`, Action: nil},
		{Name: "Schema", Pattern: `\bschema\b`, Action: nil},
		{Name: "On", Pattern: `\bon\b`, Action: nil},
		{Name: "Do", Pattern: `\bdo\b`, Action: nil},
		{Name: "Workflow", Pattern: `\bworkflow\b`, Action: nil},
		{Name: "Emit", Pattern: `\bemit\b`, Action: nil},
		{Name: "Async", Pattern: `\basync\b`, Action: nil},

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
		{Name: "Status", Pattern: `\bstatus\b`, Action: nil},
		{Name: "Where", Pattern: `\bwhere\b`, Action: nil},
		{Name: "With", Pattern: `\bwith\b`, Action: nil},

		// Integration keywords
		{Name: "Integration", Pattern: `\bintegration\b`, Action: nil},
		{Name: "BaseUrl", Pattern: `\bbase_url\b`, Action: nil},
		{Name: "Timeout", Pattern: `\btimeout\b`, Action: nil},
		{Name: "CircuitBreaker", Pattern: `\bcircuit_breaker\b`, Action: nil},
		{Name: "Threshold", Pattern: `\bthreshold\b`, Action: nil},
		{Name: "MaxConcurrent", Pattern: `\bmax_concurrent\b`, Action: nil},
		{Name: "Rest", Pattern: `\brest\b`, Action: nil},
		{Name: "Graphql", Pattern: `\bgraphql\b`, Action: nil},
		{Name: "Grpc", Pattern: `\bgrpc\b`, Action: nil},
		{Name: "Bearer", Pattern: `\bbearer\b`, Action: nil},
		{Name: "Token", Pattern: `\btoken\b`, Action: nil},
		{Name: "Header", Pattern: `\bheader\b`, Action: nil},
		{Name: "Value", Pattern: `\bvalue\b`, Action: nil},

		// Webhook keywords
		{Name: "Webhook", Pattern: `\bwebhook\b`, Action: nil},
		{Name: "Url", Pattern: `\burl\b`, Action: nil},
		{Name: "Headers", Pattern: `\bheaders\b`, Action: nil},
		{Name: "Retry", Pattern: `\bretry\b`, Action: nil},
		{Name: "InitialInterval", Pattern: `\binitial_interval\b`, Action: nil},
		{Name: "Backoff", Pattern: `\bbackoff\b`, Action: nil},
		{Name: "POST", Pattern: `\bPOST\b`, Action: nil},
		{Name: "PUT", Pattern: `\bPUT\b`, Action: nil},

		// Literals
		{Name: "Number", Pattern: `[0-9]+\.?[0-9]*`, Action: nil},
		{Name: "String", Pattern: `"([^"\\]|\\.)*"`, Action: nil},

		// Identifiers
		{Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`, Action: nil},

		// Operators and punctuation
		{Name: "At", Pattern: `@`, Action: nil},
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
	Statements []*pStatement `parser:"@@*"`
}

// pStatement is the Participle grammar for a statement.
type pStatement struct {
	Pos             lexer.Position
	ConfigDecl      *pConfigDecl      `parser:"  @@"`
	DatabaseBlock   *pDatabaseBlock   `parser:"| @@"`
	AuthDecl        *pAuthDecl        `parser:"| @@"`
	RoleDecl        *pRoleDecl        `parser:"| @@"`
	MiddlewareDecl  *pMiddlewareDecl  `parser:"| @@"`
	EventDecl       *pEventDecl       `parser:"| @@"`
	EventHandler    *pEventHandler    `parser:"| @@"`
	IntegrationDecl *pIntegrationDecl `parser:"| @@"`
	WebhookDecl     *pWebhookDecl     `parser:"| @@"`
	VarDecl         *pVarDecl         `parser:"| @@"`
	IfStmt          *pIfStmt          `parser:"| @@"`
	ForLoop         *pForLoop         `parser:"| @@"`
	FuncDecl        *pFuncDecl        `parser:"| @@"`
	ExecBlock       *pExecBlock       `parser:"| @@"`
	Assignment      *pAssignment      `parser:"| @@"`
}

// pVarDecl is the Participle grammar for variable declaration.
// Note: Variable names can be keywords, so we accept both Ident and keyword tokens.
type pVarDecl struct {
	Pos   lexer.Position
	Name  string       `parser:"Var @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With | Token | Header | Value | Event | Schema | On | Do | Workflow | Emit | Async | Integration | BaseUrl | Timeout | CircuitBreaker | Threshold | MaxConcurrent | Rest | Graphql | Grpc | Bearer | Webhook | Url | Headers | Retry | InitialInterval | Backoff | POST | PUT)"`
	Value *pExpression `parser:"Equals @@"`
}

// pAssignment is the Participle grammar for assignment.
// Note: Variable names can be keywords, so we accept both Ident and keyword tokens.
type pAssignment struct {
	Pos   lexer.Position
	Name  string       `parser:"@(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With | Token | Header | Value | Event | Schema | On | Do | Workflow | Emit | Async | Integration | BaseUrl | Timeout | CircuitBreaker | Threshold | MaxConcurrent | Rest | Graphql | Grpc | Bearer | Webhook | Url | Headers | Retry | InitialInterval | Backoff | POST | PUT)"`
	Value *pExpression `parser:"Equals @@"`
}

// pIfStmt is the Participle grammar for if statement.
type pIfStmt struct {
	Pos       lexer.Position
	Condition *pExpression  `parser:"If @@"`
	Body      []*pStatement `parser:"LBrace @@* RBrace"`
	Else      []*pStatement `parser:"( Else LBrace @@* RBrace )?"`
}

// pForLoop is the Participle grammar for for loop.
type pForLoop struct {
	Pos      lexer.Position
	Variable string        `parser:"For @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With | Token | Header | Value | Event | Schema | On | Do | Workflow | Emit | Async | Integration | BaseUrl | Timeout | CircuitBreaker | Threshold | MaxConcurrent | Rest | Graphql | Grpc | Bearer | Webhook | Url | Headers | Retry | InitialInterval | Backoff | POST | PUT)"`
	Iterable *pExpression  `parser:"In @@"`
	Body     []*pStatement `parser:"LBrace @@* RBrace"`
}

// pFuncDecl is the Participle grammar for function declaration.
type pFuncDecl struct {
	Pos    lexer.Position
	Name   string        `parser:"Func @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With | Token | Header | Value | Event | Schema | On | Do | Workflow | Emit | Async | Integration | BaseUrl | Timeout | CircuitBreaker | Threshold | MaxConcurrent | Rest | Graphql | Grpc | Bearer | Webhook | Url | Headers | Retry | InitialInterval | Backoff | POST | PUT)"`
	Params []string      `parser:"LParen ( @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With | Token | Header | Value | Event | Schema | On | Do | Workflow | Emit | Async | Integration | BaseUrl | Timeout | CircuitBreaker | Threshold | MaxConcurrent | Rest | Graphql | Grpc | Bearer | Webhook | Url | Headers | Retry | InitialInterval | Backoff | POST | PUT) ( Comma @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With | Token | Header | Value | Event | Schema | On | Do | Workflow | Emit | Async | Integration | BaseUrl | Timeout | CircuitBreaker | Threshold | MaxConcurrent | Rest | Graphql | Grpc | Bearer | Webhook | Url | Headers | Retry | InitialInterval | Backoff | POST | PUT) )* )? RParen"`
	Body   []*pStatement `parser:"LBrace @@* RBrace"`
}

// pExecBlock is the Participle grammar for exec block.
type pExecBlock struct {
	Pos     lexer.Position
	Command string `parser:"ExecOpen @ShellCommand ShellClose"`
}

// pConfigDecl is the Participle grammar for config block.
// Example: config { database_type: "mongodb" mongodb_uri: "..." }
type pConfigDecl struct {
	Pos        lexer.Position
	Properties []*pConfigProperty `parser:"Config LBrace @@* RBrace"`
}

// pConfigProperty is a key-value pair in a config block.
type pConfigProperty struct {
	Pos   lexer.Position
	Key   string       `parser:"@(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With | Token | Header | Value | Event | Schema | On | Do | Workflow | Emit | Async | Integration | BaseUrl | Timeout | CircuitBreaker | Threshold | MaxConcurrent | Rest | Graphql | Grpc | Bearer | Webhook | Url | Headers | Retry | InitialInterval | Backoff | POST | PUT) Colon"`
	Value *pExpression `parser:"@@"`
}

// pDatabaseBlock is the Participle grammar for database blocks.
// Example: database mongodb { collection ... } or database postgres { model ... }
type pDatabaseBlock struct {
	Pos         lexer.Position
	Type        string             `parser:"Database @( Postgres | MongoDB ) LBrace"`
	Models      []*pModelDecl      `parser:"@@*"`
	Collections []*pCollectionDecl `parser:"@@* RBrace"`
}
// =============================================================================
// Authentication & Authorization Grammar
// =============================================================================

// pAuthDecl is the Participle grammar for auth provider declaration.
type pAuthDecl struct {
	Pos      lexer.Position
	Name     string   `parser:"Auth @Ident LBrace"`
	Method   string   `parser:"Method @( Jwt | Oauth2 | Apikey | Basic )"`
	JwksURL  *string  `parser:"( JwksUrl @String )?"`
	Issuer   *string  `parser:"( Issuer @String )?"`
	Audience *string  `parser:"( Audience @String )? RBrace"`
}

// pRoleDecl is the Participle grammar for role declaration.
type pRoleDecl struct {
	Pos         lexer.Position
	Name        string   `parser:"Role @Ident LBrace"`
	Permissions []string `parser:"Permissions LBracket ( @String ( Comma @String )* )? RBracket RBrace"`
}

// pMiddlewareDecl is the Participle grammar for middleware declaration.
type pMiddlewareDecl struct {
	Pos            lexer.Position
	Name           string             `parser:"Middleware @Ident LBrace"`
	MiddlewareType string             `parser:"Type @Ident"`
	Config         []*pConfigProperty `parser:"( Config LBrace @@* RBrace )? RBrace"`
}

// =============================================================================
// Event Grammar
// =============================================================================

// pEventDecl is the Participle grammar for event declaration.
// Example: event user.created { schema { user_id string ... } }
type pEventDecl struct {
	Pos    lexer.Position
	Name   string             `parser:"Event @Ident ((\".\") @Ident)*"`
	Schema *pEventSchema      `parser:"LBrace (Schema @@)?"`
	RBrace string             `parser:"RBrace"`
}

// pEventSchema is the Participle grammar for event schema.
type pEventSchema struct {
	Pos    lexer.Position
	Fields []*pEventSchemaField `parser:"LBrace @@* RBrace"`
}

// pEventSchemaField is the Participle grammar for event schema field.
type pEventSchemaField struct {
	Pos       lexer.Position
	Name      string `parser:"@(Ident | Method | Type | Event | Schema | On | Do | Workflow | Emit | Async | Integration | Auth | Token | Header | Value | Webhook | Url | Headers | Retry | Timeout | Text)"`
	FieldType string `parser:"@(Ident | Text)"`
}

// pEventHandler is the Participle grammar for event handler declaration.
// Example: on "user.created" do workflow "send_welcome_email" async
type pEventHandler struct {
	Pos        lexer.Position
	EventName  string `parser:"On @String"`
	ActionType string `parser:"Do @(Workflow | Integration | Emit | Webhook)"`
	Target     string `parser:"@String"`
	Async      bool   `parser:"@Async?"`
}

// =============================================================================
// Integration Grammar
// =============================================================================

// pIntegrationDecl is the Participle grammar for integration declaration.
// Example: integration stripe { type rest base_url "..." auth bearer { ... } }
type pIntegrationDecl struct {
	Pos            lexer.Position
	Name           string             `parser:"Integration @Ident LBrace"`
	IntgType       string             `parser:"Type @(Rest | Graphql | Grpc | Webhook)"`
	BaseURL        string             `parser:"BaseUrl @String"`
	Auth           *pIntegrationAuth  `parser:"(Auth @@)?"`
	Timeout        *string            `parser:"(Timeout @String)?"`
	CircuitBreaker *pCircuitBreaker   `parser:"@@? RBrace"`
}

// pIntegrationAuth is the Participle grammar for integration auth.
// Example: auth bearer { token env("API_KEY") }
type pIntegrationAuth struct {
	Pos      lexer.Position
	AuthType string             `parser:"@(Bearer | Basic | Apikey | Oauth2)"`
	Config   []*pConfigProperty `parser:"LBrace @@* RBrace"`
}

// pCircuitBreaker is the Participle grammar for circuit breaker config.
// Example: circuit_breaker { threshold 5 timeout "60s" max_concurrent 100 }
type pCircuitBreaker struct {
	Pos           lexer.Position
	Threshold     int     `parser:"CircuitBreaker LBrace Threshold @Number"`
	Timeout       string  `parser:"Timeout @String"`
	MaxConcurrent int     `parser:"MaxConcurrent @Number RBrace"`
}

// =============================================================================
// Webhook Grammar
// =============================================================================

// pWebhookDecl is the Participle grammar for webhook declaration.
// Example: webhook analytics { event "order.completed" url "..." method POST ... }
type pWebhookDecl struct {
	Pos     lexer.Position
	Name    string                 `parser:"Webhook @Ident LBrace"`
	Event   string                 `parser:"Event @String"`
	URL     string                 `parser:"Url @String"`
	Method  string                 `parser:"Method @(POST | PUT)"`
	Headers *pWebhookHeaders       `parser:"@@?"`
	Retry   *pWebhookRetryPolicy   `parser:"@@? RBrace"`
}

// pWebhookHeaders is the Participle grammar for webhook headers.
type pWebhookHeaders struct {
	Pos     lexer.Position
	Headers []*pWebhookHeader `parser:"Headers LBrace @@* RBrace"`
}

// pWebhookHeader is the Participle grammar for a single webhook header.
type pWebhookHeader struct {
	Pos   lexer.Position
	Key   string `parser:"@String"`
	Value string `parser:"Colon @String"`
}

// pWebhookRetryPolicy is the Participle grammar for webhook retry policy.
// Example: retry 3 initial_interval "1s" backoff 2.0
type pWebhookRetryPolicy struct {
	Pos             lexer.Position
	MaxAttempts     int     `parser:"Retry @Number"`
	InitialInterval string  `parser:"InitialInterval @String"`
	Backoff         float64 `parser:"Backoff @Number"`
}

// =============================================================================
// PostgreSQL Model Grammar
// =============================================================================

// pModelDecl represents a PostgreSQL model declaration.
// Example: model User { id: uuid, primary, auto }
type pModelDecl struct {
	Pos         lexer.Position
	Name        string        `parser:"Model @Ident LBrace"`
	Description *string       `parser:"(Description Colon @String)?"`
	Fields      []*pFieldDecl `parser:"@@*"`
	Indexes     []*pIndexDecl `parser:"@@* RBrace"`
}

// pFieldDecl represents a field in a PostgreSQL model.
// Note: Field names can be keywords, so we accept both Ident and keyword tokens.
type pFieldDecl struct {
	Pos       lexer.Position
	Name      string       `parser:"@(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With) Colon"`
	Type      *pTypeRef    `parser:"@@"`
	Modifiers []*pModifier `parser:"(Comma @@)*"`
}

// pTypeRef represents a type reference (e.g., string, uuid, ref(User)).
type pTypeRef struct {
	Pos    lexer.Position
	Name   string      `parser:"@(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With)"`
	Params []*pTypeRef `parser:"(LParen @@ (Comma @@)* RParen)?"`
}

// pModifier represents a field modifier (e.g., required, unique, default(value)).
type pModifier struct {
	Pos   lexer.Position
	Name  string       `parser:"@(Required | Optional | Unique | Primary | Auto | Default | Text | Geospatial | Ident)"`
	Value *pExpression `parser:"(LParen @@ RParen)?"`
}

// pIndexDecl represents a PostgreSQL index declaration.
// Note: Index field names can be keywords, so we accept both Ident and keyword tokens.
type pIndexDecl struct {
	Pos    lexer.Position
	Fields []string `parser:"Index Colon LBracket @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With) (Comma @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With))* RBracket"`
	Unique bool     `parser:"@Unique?"`
}

// =============================================================================
// MongoDB Collection Grammar
// =============================================================================

// pCollectionDecl represents a MongoDB collection declaration.
// Example: collection User { _id: objectid, primary }
type pCollectionDecl struct {
	Pos         lexer.Position
	Name        string               `parser:"Collection @Ident LBrace"`
	Description *string              `parser:"(Description Colon @String)?"`
	Fields      []*pMongoFieldDecl   `parser:"@@*"`
	Indexes     *pMongoIndexesBlock  `parser:"@@? RBrace"`
}

// pMongoFieldDecl represents a field in a MongoDB collection.
// Note: Field names can be keywords, so we accept both Ident and keyword tokens.
type pMongoFieldDecl struct {
	Pos       lexer.Position
	Name      string         `parser:"@(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With) Colon"`
	Type      *pMongoTypeRef `parser:"@@"`
	Modifiers []*pModifier   `parser:"(Comma @@)*"`
}

// pMongoTypeRef represents a MongoDB-specific type reference.
// Supports: objectid, string, int, double, bool, date, binary, array(T), embedded { ... }
type pMongoTypeRef struct {
	Pos         lexer.Position
	EmbeddedDoc *pEmbeddedDoc `parser:"  @@"`
	Name        string        `parser:"| @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With)"`
	Params      []string      `parser:"(LParen @Ident (Comma @Ident)* RParen)?"`
}

// pEmbeddedDoc represents an embedded document type in MongoDB.
type pEmbeddedDoc struct {
	Pos    lexer.Position
	Fields []*pMongoFieldDecl `parser:"Embedded LBrace @@* RBrace"`
}

// pMongoIndexesBlock represents the indexes block in a MongoDB collection.
type pMongoIndexesBlock struct {
	Pos     lexer.Position
	Indexes []*pMongoIndexDecl `parser:"Indexes LBrace @@* RBrace"`
}

// pMongoIndexDecl represents a MongoDB index declaration.
// Supports: single, compound, text, geospatial indexes
// Note: Index field names can be keywords, so we accept both Ident and keyword tokens.
type pMongoIndexDecl struct {
	Pos       lexer.Position
	Fields    []string `parser:"Index Colon LBracket @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With) (Comma @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With))* RBracket"`
	Unique    bool     `parser:"@Unique?"`
	IndexKind string   `parser:"@(Text | Geospatial)?"`
}

// pExpression is the Participle grammar for expressions.
type pExpression struct {
	Pos          lexer.Position
	String       *string        `parser:"  @String"`
	Number       *string        `parser:"| @Number"`
	True         bool           `parser:"| @True"`
	False        bool           `parser:"| @False"`
	Array        *pArrayLiteral `parser:"| @@"`
	FuncCall     *pFuncCall     `parser:"| @@"`
	Identifier   *string        `parser:"| @(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With | Token | Header | Value | Event | Schema | On | Do | Workflow | Emit | Async | Integration | BaseUrl | Timeout | CircuitBreaker | Threshold | MaxConcurrent | Rest | Graphql | Grpc | Bearer | Webhook | Url | Headers | Retry | InitialInterval | Backoff | POST | PUT)"`
}

// pArrayLiteral is the Participle grammar for array literals.
type pArrayLiteral struct {
	Pos      lexer.Position
	Elements []*pExpression `parser:"LBracket ( @@ ( Comma @@ )* )? RBracket"`
}

// pFuncCall is the Participle grammar for function calls.
type pFuncCall struct {
	Pos       lexer.Position
	Name      string         `parser:"@(Ident | Config | Database | Postgres | MongoDB | Model | Collection | Indexes | Index | Unique | Text | Geospatial | Embedded | Required | Optional | Primary | Auto | Default | Description | Auth | Role | Middleware | Method | JwksUrl | Issuer | Audience | Permissions | Type | Provider | Jwt | Oauth2 | Apikey | Basic | Endpoint | GET | POST | PUT | DELETE | PATCH | Request | Response | From | Body | Query | Path | Status | Where | With | Token | Header | Value | Event | Schema | On | Do | Workflow | Emit | Async | Integration | BaseUrl | Timeout | CircuitBreaker | Threshold | MaxConcurrent | Rest | Graphql | Grpc | Bearer | Webhook | Url | Headers | Retry | InitialInterval | Backoff | POST | PUT)"`
	Arguments []*pExpression `parser:"LParen ( @@ ( Comma @@ )* )? RParen"`
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
	// First, extract and parse any endpoint declarations separately
	endpoints, cleanedInput, err := extractAndParseEndpoints(input)
	if err != nil {
		return nil, err
	}

	// Parse the main DSL without endpoints
	parsed, err := parserInstance.ParseString("", cleanedInput)
	if err != nil {
		return nil, err
	}

	// Convert main program
	program := convertProgram(parsed)

	// Add endpoint declarations to the program
	for _, endpoint := range endpoints {
		program.Statements = append(program.Statements, endpoint)
	}

	return program, nil
}

// ParseApplication parses the input string and returns a structured Application.
// This provides organized access to different types of declarations (auth, middleware, etc.)
func ParseApplication(input string) (*ast.Application, error) {
	program, err := Parse(input)
	if err != nil {
		return nil, err
	}
	return program.ToApplication(), nil
}

// ParseFile parses a file and returns an AST Program.
func ParseFile(filename string) (*ast.Program, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	// Use Parse() which handles endpoint extraction
	return Parse(string(data))
}

// ParseApplicationFile parses a file and returns a structured Application.
func ParseApplicationFile(filename string) (*ast.Application, error) {
	program, err := ParseFile(filename)
	if err != nil {
		return nil, err
	}
	return program.ToApplication(), nil
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
	case s.AuthDecl != nil:
		return convertAuthDecl(s.AuthDecl)
	case s.RoleDecl != nil:
		return convertRoleDecl(s.RoleDecl)
	case s.MiddlewareDecl != nil:
		return convertMiddlewareDecl(s.MiddlewareDecl)
	case s.EventDecl != nil:
		return convertEventDecl(s.EventDecl)
	case s.EventHandler != nil:
		return convertEventHandler(s.EventHandler)
	case s.IntegrationDecl != nil:
		return convertIntegrationDecl(s.IntegrationDecl)
	case s.WebhookDecl != nil:
		return convertWebhookDecl(s.WebhookDecl)
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
// Authentication & Authorization Conversion Functions
// =============================================================================

func convertAuthDecl(a *pAuthDecl) *ast.AuthDecl {
	var method ast.AuthMethod
	switch a.Method {
	case "jwt":
		method = ast.AuthMethodJWT
	case "oauth2":
		method = ast.AuthMethodOAuth2
	case "apikey":
		method = ast.AuthMethodAPIKey
	case "basic":
		method = ast.AuthMethodBasic
	default:
		method = ast.AuthMethodJWT
	}

	var jwksConfig *ast.JWKSConfig
	if method == ast.AuthMethodJWT && a.JwksURL != nil {
		jwksConfig = &ast.JWKSConfig{
			URL:      unquote(*a.JwksURL),
			Issuer:   safeUnquote(a.Issuer),
			Audience: safeUnquote(a.Audience),
		}
	}

	return &ast.AuthDecl{
		Name:   a.Name,
		Method: method,
		JWKS:   jwksConfig,
		Config: make(map[string]ast.Expression),
	}
}

func convertRoleDecl(r *pRoleDecl) *ast.RoleDecl {
	permissions := make([]string, len(r.Permissions))
	for i, p := range r.Permissions {
		permissions[i] = unquote(p)
	}
	return &ast.RoleDecl{
		Name:        r.Name,
		Permissions: permissions,
	}
}

func convertMiddlewareDecl(m *pMiddlewareDecl) *ast.MiddlewareDecl {
	config := make(map[string]ast.Expression)
	for _, prop := range m.Config {
		config[prop.Key] = convertExpression(prop.Value)
	}
	return &ast.MiddlewareDecl{
		Name:           m.Name,
		MiddlewareType: m.MiddlewareType,
		Config:         config,
	}
}

func safeUnquote(s *string) string {
	if s == nil {
		return ""
	}
	return unquote(*s)
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

// =============================================================================
// Event Conversion Functions
// =============================================================================

func convertEventDecl(e *pEventDecl) *ast.EventDecl {
	var schema *ast.EventSchema
	if e.Schema != nil {
		schema = convertEventSchema(e.Schema)
	}

	return &ast.EventDecl{
		Name:     e.Name,
		Schema:   schema,
		Handlers: nil, // Handlers are parsed separately as EventHandler statements
	}
}

func convertEventSchema(s *pEventSchema) *ast.EventSchema {
	fields := make([]*ast.EventSchemaField, len(s.Fields))
	for i, f := range s.Fields {
		fields[i] = &ast.EventSchemaField{
			Name:      f.Name,
			FieldType: f.FieldType,
		}
	}
	return &ast.EventSchema{
		Fields: fields,
	}
}

func convertEventHandler(h *pEventHandler) *ast.EventHandlerDecl {
	return &ast.EventHandlerDecl{
		EventName:  unquote(h.EventName),
		ActionType: h.ActionType,
		Target:     unquote(h.Target),
		Async:      h.Async,
	}
}

// =============================================================================
// Integration Conversion Functions
// =============================================================================

func convertIntegrationDecl(i *pIntegrationDecl) *ast.IntegrationDecl {
	var intgType ast.IntegrationType
	switch i.IntgType {
	case "rest":
		intgType = ast.IntegrationTypeREST
	case "graphql":
		intgType = ast.IntegrationTypeGraphQL
	case "grpc":
		intgType = ast.IntegrationTypeGRPC
	case "webhook":
		intgType = ast.IntegrationTypeWebhook
	default:
		intgType = ast.IntegrationTypeREST
	}

	var auth *ast.IntegrationAuthDecl
	if i.Auth != nil {
		auth = convertIntegrationAuth(i.Auth)
	}

	var timeout string
	if i.Timeout != nil {
		timeout = unquote(*i.Timeout)
	}

	var circuitBreaker *ast.CircuitBreakerConfig
	if i.CircuitBreaker != nil {
		circuitBreaker = convertCircuitBreaker(i.CircuitBreaker)
	}

	return &ast.IntegrationDecl{
		Name:           i.Name,
		IntgType:       intgType,
		BaseURL:        unquote(i.BaseURL),
		Auth:           auth,
		Timeout:        timeout,
		CircuitBreaker: circuitBreaker,
	}
}

func convertIntegrationAuth(a *pIntegrationAuth) *ast.IntegrationAuthDecl {
	var authType ast.IntegrationAuthType
	switch a.AuthType {
	case "bearer":
		authType = ast.IntegrationAuthBearer
	case "basic":
		authType = ast.IntegrationAuthBasic
	case "apikey":
		authType = ast.IntegrationAuthAPIKey
	case "oauth2":
		authType = ast.IntegrationAuthOAuth2
	default:
		authType = ast.IntegrationAuthBearer
	}

	config := make(map[string]ast.Expression)
	for _, prop := range a.Config {
		config[prop.Key] = convertExpression(prop.Value)
	}

	return &ast.IntegrationAuthDecl{
		AuthType: authType,
		Config:   config,
	}
}

func convertCircuitBreaker(cb *pCircuitBreaker) *ast.CircuitBreakerConfig {
	return &ast.CircuitBreakerConfig{
		FailureThreshold: cb.Threshold,
		Timeout:          unquote(cb.Timeout),
		MaxConcurrent:    cb.MaxConcurrent,
	}
}

// =============================================================================
// Webhook Conversion Functions
// =============================================================================

func convertWebhookDecl(w *pWebhookDecl) *ast.WebhookDecl {
	var method ast.WebhookHTTPMethod
	switch w.Method {
	case "POST":
		method = ast.WebhookMethodPOST
	case "PUT":
		method = ast.WebhookMethodPUT
	default:
		method = ast.WebhookMethodPOST
	}

	var headers []*ast.WebhookHeader
	if w.Headers != nil {
		headers = make([]*ast.WebhookHeader, len(w.Headers.Headers))
		for i, h := range w.Headers.Headers {
			headers[i] = &ast.WebhookHeader{
				Key:   unquote(h.Key),
				Value: unquote(h.Value),
			}
		}
	}

	var retry *ast.RetryPolicyDecl
	if w.Retry != nil {
		retry = convertWebhookRetryPolicy(w.Retry)
	}

	return &ast.WebhookDecl{
		Name:    w.Name,
		Event:   unquote(w.Event),
		URL:     unquote(w.URL),
		Method:  method,
		Headers: headers,
		Retry:   retry,
	}
}

func convertWebhookRetryPolicy(p *pWebhookRetryPolicy) *ast.RetryPolicyDecl {
	return &ast.RetryPolicyDecl{
		MaxAttempts:       p.MaxAttempts,
		InitialInterval:   unquote(p.InitialInterval),
		BackoffMultiplier: p.Backoff,
	}
}

// =============================================================================
// Endpoint Integration Helper Functions
// =============================================================================

// extractAndParseEndpoints extracts endpoint declarations from the input,
// parses them separately using the endpoint parser, and returns the cleaned input
func extractAndParseEndpoints(input string) ([]*ast.EndpointDecl, string, error) {
	var endpoints []*ast.EndpointDecl
	var cleanedLines []string

	lines := strings.Split(input, "\n")
	i := 0

	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Check if this line starts an endpoint declaration
		if strings.HasPrefix(line, "@") || strings.HasPrefix(line, "endpoint ") {
			// Find the start of the endpoint block
			endpointStart := i
			if strings.HasPrefix(line, "@") {
				// Skip annotation lines to find the actual endpoint line
				for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "endpoint ") {
					i++
				}
				if i >= len(lines) {
					break
				}
			}

			// Find the opening brace
			for i < len(lines) && !strings.Contains(lines[i], "{") {
				i++
			}
			if i >= len(lines) {
				break
			}

			// Count braces to find the end of the block
			braceCount := 1
			i++

			for i < len(lines) && braceCount > 0 {
				for _, ch := range lines[i] {
					if ch == '{' {
						braceCount++
					} else if ch == '}' {
						braceCount--
					}
				}
				if braceCount > 0 {
					i++
				}
			}

			if braceCount == 0 {
				// Extract the endpoint block
				endpointBlock := strings.Join(lines[endpointStart:i+1], "\n")

				// Parse this endpoint using the dedicated endpoint parser
				endpoint, err := ParseEndpoint(endpointBlock)
				if err != nil {
					return nil, "", err
				}
				endpoints = append(endpoints, endpoint)

				// Don't add these lines to cleaned input
			} else {
				// Unclosed braces, treat as regular content
				cleanedLines = append(cleanedLines, lines[endpointStart])
			}
		} else {
			// Regular line, add to cleaned input
			cleanedLines = append(cleanedLines, lines[i])
		}
		i++
	}

	cleanedInput := strings.Join(cleanedLines, "\n")
	cleanedInput = strings.TrimSpace(cleanedInput)

	return endpoints, cleanedInput, nil
}


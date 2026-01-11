# Task 002: Participle Grammar Implementation

## Overview
Implement the complete Participle v2 grammar for parsing CodeAI DSL source files, covering all language constructs: config, entity, endpoint, workflow, job, integration, event, and function declarations.

## Phase
Phase 1: Foundation

## Priority
Critical - Parser is required for all runtime functionality.

## Dependencies
- Task 001: Project Structure Setup

## Description
Create the grammar definitions using Participle v2's struct tag-based approach. The grammar must be fault-tolerant to handle common LLM generation patterns and provide excellent error messages.

## Detailed Requirements

### 1. Grammar File Structure (internal/parser/grammar.go)

```go
package parser

import (
    "github.com/alecthomas/participle/v2"
    "github.com/alecthomas/participle/v2/lexer"
)

// Program is the root node of a CodeAI source file
type Program struct {
    Pos          lexer.Position
    Declarations []*Declaration `@@*`
}

// Declaration represents a top-level declaration
type Declaration struct {
    Pos         lexer.Position
    Config      *ConfigBlock      `  @@`
    Entity      *EntityDecl       `| @@`
    Endpoint    *EndpointDecl     `| @@`
    Workflow    *WorkflowDecl     `| @@`
    Job         *JobDecl          `| @@`
    Integration *IntegrationDecl  `| @@`
    Event       *EventDecl        `| @@`
    Function    *FunctionDecl     `| @@`
}
```

### 2. Config Block Grammar

```go
type ConfigBlock struct {
    Pos      lexer.Position
    Settings []*ConfigSetting `"config" "{" @@* "}"`
}

type ConfigSetting struct {
    Pos   lexer.Position
    Key   string       `@Ident ":"`
    Value *ConfigValue `@@`
}

type ConfigValue struct {
    String  *string        `  @String`
    Number  *float64       `| @Number`
    Bool    *bool          `| @("true" | "false")`
    Ident   *string        `| @Ident`
    Block   *ConfigBlock   `| @@`
    List    []*ConfigValue `| "[" (@@ ("," @@)*)? ","? "]"`
}
```

### 3. Entity Grammar

```go
type EntityDecl struct {
    Pos         lexer.Position
    Name        string        `"entity" @Ident "{"`
    Description *string       `("description" ":" @String)?`
    Fields      []*FieldDecl  `@@*`
    Indexes     []*IndexDecl  `@@* "}"`
}

type FieldDecl struct {
    Pos       lexer.Position
    Name      string       `@Ident ":"`
    Type      *TypeRef     `@@`
    Modifiers []*Modifier  `("," @@)*`
}

type TypeRef struct {
    Pos    lexer.Position
    Name   string     `@Ident`
    Params []*TypeRef `("(" @@ ("," @@)* ")")?`
}

type Modifier struct {
    Pos   lexer.Position
    Name  string      `@Ident`
    Value *Expression `("(" @@ ")")?`
}

type IndexDecl struct {
    Pos     lexer.Position
    Fields  []string `"index" ":" "[" @Ident ("," @Ident)* "]"`
    Unique  bool     `@"unique"?`
}
```

### 4. Endpoint Grammar

```go
type EndpointDecl struct {
    Pos         lexer.Position
    Method      string         `"endpoint" @("GET"|"POST"|"PUT"|"DELETE"|"PATCH")`
    Path        string         `@Path "{"`
    Description *string        `("description" ":" @String)?`
    Auth        *AuthClause    `@@?`
    Roles       *RolesClause   `@@?`
    PathParams  *ParamsBlock   `("path" @@)?`
    QueryParams *ParamsBlock   `("query" @@)?`
    Body        *ParamsBlock   `("body" @@)?`
    Returns     *ReturnClause  `@@?`
    Validate    *ValidateBlock `@@?`
    Filter      *FilterBlock   `@@?`
    Sort        *SortClause    `@@?`
    Cache       *CacheClause   `@@?`
    OnSuccess   *ActionClause  `@@?`
    Error       []*ErrorClause `@@* "}"`
}

type AuthClause struct {
    Pos  lexer.Position
    Type string `"auth" ":" @("required" | "optional" | "public")`
}

type RolesClause struct {
    Pos   lexer.Position
    Roles []string `"roles" ":" "[" @Ident ("," @Ident)* "]"`
}

type ParamsBlock struct {
    Pos    lexer.Position
    Params []*ParamDecl `"{" @@* "}"`
}

type ParamDecl struct {
    Pos       lexer.Position
    Name      string      `@Ident ":"`
    Type      *TypeRef    `@@`
    Modifiers []*Modifier `("," @@)*`
}

type ReturnClause struct {
    Pos  lexer.Position
    Type *TypeRef `"returns" ":" @@`
}
```

### 5. Workflow Grammar

```go
type WorkflowDecl struct {
    Pos         lexer.Position
    Name        string         `"workflow" @Ident "{"`
    Description *string        `("description" ":" @String)?`
    Trigger     string         `"trigger" ":" @Ident`
    Steps       *StepsBlock    `@@`
    OnComplete  *ActionClause  `@@?`
    OnFail      *ActionClause  `@@? "}"`
}

type StepsBlock struct {
    Pos   lexer.Position
    Steps []*StepDecl `"steps" "{" @@* "}"`
}

type StepDecl struct {
    Pos        lexer.Position
    Name       string          `@Ident "{"`
    ForEach    *ForEachClause  `@@?`
    Check      *Expression     `("check" ":" @@)?`
    Action     *Expression     `("action" ":" @@)?`
    Call       *CallExpr       `@@?`
    Timeout    *string         `("timeout" ":" @Duration)?`
    Retry      *RetryClause    `@@?`
    Compensate *ActionClause   `@@?`
    OnSuccess  *ActionClause   `@@?`
    OnFail     *ActionClause   `@@? "}"`
}

type RetryClause struct {
    Pos      lexer.Position
    Count    int    `"retry" ":" @Number "times"`
    Strategy string `("with" @Ident)?`
}
```

### 6. Job Grammar

```go
type JobDecl struct {
    Pos         lexer.Position
    Name        string        `"job" @Ident "{"`
    Description *string       `("description" ":" @String)?`
    Schedule    string        `"schedule" ":" @(String | ScheduleExpr)`
    Timezone    *string       `("timezone" ":" @String)?`
    Steps       *StepsBlock   `@@?`
    Action      *ActionClause `@@?`
    Timeout     *string       `("timeout" ":" @Duration)?`
    Retry       *int          `("retry" ":" @Number "times")?`
    OnFail      *ActionClause `@@? "}"`
}
```

### 7. Integration Grammar

```go
type IntegrationDecl struct {
    Pos            lexer.Position
    Name           string            `"integration" @Ident "{"`
    Description    *string           `("description" ":" @String)?`
    Type           string            `"type" ":" @Ident`
    BaseURL        string            `"base_url" ":" @(String | FuncCall)`
    Auth           *IntegrationAuth  `@@?`
    Headers        *HeadersBlock     `@@?`
    Timeout        *string           `("timeout" ":" @Duration)?`
    Retry          *RetryClause      `@@?`
    CircuitBreaker *CircuitBreaker   `@@?`
    Operations     []*OperationDecl  `@@* "}"`
}

type OperationDecl struct {
    Pos     lexer.Position
    Name    string       `"operation" @Ident "{"`
    Method  string       `"method" ":" @("GET"|"POST"|"PUT"|"DELETE"|"PATCH")`
    Path    string       `"path" ":" @String`
    Body    *ParamsBlock `("body" @@)?`
    Returns *ParamsBlock `("returns" @@)? "}"`
}

type CircuitBreaker struct {
    Pos        lexer.Position
    Threshold  string `"circuit_breaker" ":" "{" "threshold" ":" @ThresholdExpr`
    ResetAfter string `"reset_after" ":" @Duration "}"`
}
```

### 8. Event Grammar

```go
type EventDecl struct {
    Pos         lexer.Position
    Name        string         `"event" @Ident "{"`
    Description *string        `("description" ":" @String)?`
    Payload     *ParamsBlock   `("payload" @@)?`
    Trigger     *TriggerClause `@@?`
    PublishTo   []string       `("publish_to" ":" "[" @PublishTarget ("," @PublishTarget)* "]")? "}"`
}

type TriggerClause struct {
    Pos       lexer.Position
    Condition *Expression `"trigger" ":" "when" @@`
}
```

### 9. Function Grammar

```go
type FunctionDecl struct {
    Pos         lexer.Position
    Name        string        `"function" @Ident`
    Params      []*ParamDecl  `"(" (@@ ("," @@)*)? ")"`
    ReturnType  *TypeRef      `("->" @@)?`
    Body        *FunctionBody `"{" @@ "}"`
}

type FunctionBody struct {
    Pos        lexer.Position
    Description *string       `("description" ":" @String)?`
    Statements  []*Statement  `@@*`
}
```

### 10. Expression Grammar

```go
type Expression struct {
    Pos     lexer.Position
    Or      []*AndExpr `@@ ("or" @@)*`
}

type AndExpr struct {
    Not []*NotExpr `@@ ("and" @@)*`
}

type NotExpr struct {
    Not  bool         `@"not"?`
    Comp *Comparison  `@@`
}

type Comparison struct {
    Left  *AddExpr    `@@`
    Op    string      `(@("==" | "!=" | "<=" | ">=" | "<" | ">" | "contains" | "includes" | "is"))?`
    Right *AddExpr    `@@?`
}

type AddExpr struct {
    Left  *MulExpr  `@@`
    Op    string    `(@("+" | "-"))?`
    Right *AddExpr  `@@?`
}

type MulExpr struct {
    Left  *UnaryExpr `@@`
    Op    string     `(@("*" | "/" | "%"))?`
    Right *MulExpr   `@@?`
}

type UnaryExpr struct {
    Op    string     `@("-" | "+")?`
    Value *Primary   `@@`
}

type Primary struct {
    Pos        lexer.Position
    Number     *float64     `  @Number`
    String     *string      `| @String`
    Bool       *bool        `| @("true" | "false")`
    Null       bool         `| @"null"`
    Ident      *IdentExpr   `| @@`
    FuncCall   *FuncCall    `| @@`
    List       []*Expression `| "[" (@@ ("," @@)*)? "]"`
    SubExpr    *Expression  `| "(" @@ ")"`
    If         *IfExpr      `| @@`
    Lambda     *LambdaExpr  `| @@`
}

type IdentExpr struct {
    Parts []string `@Ident ("." @Ident)*`
}

type FuncCall struct {
    Name string        `@Ident`
    Args []*Expression `"(" (@@ ("," @@)*)? ")"`
}

type IfExpr struct {
    Condition *Expression `"if" @@`
    Then      *Expression `"then" @@`
    Else      *Expression `"else" @@`
}

type LambdaExpr struct {
    Params []string    `@Ident ("," @Ident)* "=>"`
    Body   *Expression `@@`
}
```

### 11. Custom Lexer

```go
// internal/parser/lexer.go
package parser

import (
    "github.com/alecthomas/participle/v2/lexer"
)

var CodeAILexer = lexer.MustSimple([]lexer.SimpleRule{
    {Name: "Comment", Pattern: `#[^\n]*`},
    {Name: "Whitespace", Pattern: `\s+`},
    {Name: "Duration", Pattern: `\d+[smhd]`},
    {Name: "Path", Pattern: `/[a-zA-Z0-9_\-/{}\*]+`},
    {Name: "Number", Pattern: `\d+(\.\d+)?`},
    {Name: "String", Pattern: `"[^"]*"`},
    {Name: "ThresholdExpr", Pattern: `\d+\s+failures?\s+in\s+\d+[smhd]`},
    {Name: "ScheduleExpr", Pattern: `every\s+\d+[smhd]`},
    {Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`},
    {Name: "Punct", Pattern: `[{}()\[\]:,.\->=<+*/%!|&]`},
})
```

## Acceptance Criteria
- [ ] All grammar structures compile without errors
- [ ] Parser can parse config blocks
- [ ] Parser can parse entity declarations with all field types
- [ ] Parser can parse endpoint declarations with all clauses
- [ ] Parser can parse workflow declarations
- [ ] Parser can parse job declarations
- [ ] Parser can parse integration declarations
- [ ] Parser can parse event declarations
- [ ] Parser can parse function declarations
- [ ] Parser handles trailing commas gracefully
- [ ] Parser provides position information for error messages
- [ ] Parser handles comments correctly

## Implementation Steps
1. Create lexer.go with custom token definitions
2. Create grammar.go with all struct definitions
3. Create parser.go with parser initialization
4. Add error handling and recovery
5. Write unit tests for each declaration type
6. Test with example CodeAI files

## Testing Strategy
- Unit tests for each grammar construct
- Integration test with complete example file
- Error case testing (malformed input)
- Fuzzing for robustness

## Files to Create/Modify
- `internal/parser/lexer.go`
- `internal/parser/grammar.go`
- `internal/parser/parser.go`
- `internal/parser/parser_test.go`

## Notes
- Participle v2 uses struct tags for grammar rules
- Position tracking is essential for error messages
- Trailing commas should be optional everywhere
- Keywords should be case-insensitive where sensible

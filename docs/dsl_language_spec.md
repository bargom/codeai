# CodeAI DSL Language Specification

**Version**: 1.0
**Date**: 2026-01-12
**Status**: Core Language Implemented | Business DSL Planned

---

## Table of Contents

1. [Language Overview](#1-language-overview)
2. [Lexical Structure](#2-lexical-structure)
3. [Syntax Reference (EBNF Grammar)](#3-syntax-reference-ebnf-grammar)
4. [Type System](#4-type-system)
5. [Statements](#5-statements)
6. [Expressions](#6-expressions)
7. [Shell Execution](#7-shell-execution)
8. [Planned Business DSL](#8-planned-business-dsl)
9. [Examples](#9-examples)
10. [Error Handling](#10-error-handling)

---

## 1. Language Overview

### 1.1 Purpose

CodeAI DSL is a domain-specific programming language designed for Large Language Model (LLM) code generation. The language prioritizes patterns and syntax that LLMs can generate with high accuracy and consistency.

### 1.2 Design Goals

| Goal | Description |
|------|-------------|
| **LLM-Friendly Syntax** | Declarative constructs that align with LLM token prediction patterns |
| **Fault-Tolerant Parsing** | Flexible whitespace handling, lenient syntax variations |
| **Safe by Default** | No exposed unsafe primitives; runtime handles security |
| **Business-Domain Focus** | First-class constructs for common backend patterns |

### 1.3 Target Audience

- **LLMs**: Primary code generators
- **Developers**: Code review, debugging, and extension
- **DevOps**: Deployment and configuration management

### 1.4 File Extension

CodeAI source files use the `.cai` extension.

### 1.5 Current Implementation Status

The current parser (`internal/parser/parser.go`) implements a **scripting language** with:
- Variable declarations and assignments
- Control flow (if/else, for loops)
- Function declarations
- Shell command execution blocks

The **Business DSL** (entities, endpoints, workflows, jobs, integrations, events) described in the Implementation Plan is planned for future phases.

---

## 2. Lexical Structure

### 2.1 Character Set

CodeAI source files are UTF-8 encoded. Identifiers use ASCII letters, digits, and underscores.

### 2.2 Whitespace

Whitespace (spaces, tabs, newlines) is not significant except as token separators. Indentation is flexible.

```codeai
// All equivalent:
var x = 1
var  x  =  1
var
  x
  =
  1
```

### 2.3 Comments

| Type | Syntax | Description |
|------|--------|-------------|
| Single-line | `// comment` | Extends to end of line |
| Multi-line | `/* comment */` | Can span multiple lines |

```codeai
// This is a single-line comment
var x = 1  // Inline comment

/* This is a
   multi-line comment */
var y = 2
```

### 2.4 Keywords

| Keyword | Purpose |
|---------|---------|
| `var` | Variable declaration |
| `if` | Conditional statement |
| `else` | Alternative branch |
| `for` | Loop construct |
| `in` | Iteration operator |
| `function` | Function declaration |
| `true` | Boolean literal |
| `false` | Boolean literal |
| `exec` | Shell execution block |

### 2.5 Identifiers

Identifiers start with a letter or underscore, followed by letters, digits, or underscores.

```ebnf
Ident = [a-zA-Z_][a-zA-Z0-9_]*
```

**Valid identifiers:**
```
x
my_variable
userName
_private
count2
```

**Invalid identifiers:**
```
2count    // Cannot start with digit
my-var    // Hyphens not allowed
$var      // Special characters not allowed
```

### 2.6 Literals

#### 2.6.1 String Literals

Strings are enclosed in double quotes. Escape sequences are supported.

```ebnf
String = '"' ( [^"\\] | '\\' . )* '"'
```

```codeai
var greeting = "Hello, World!"
var escaped = "Line 1\nLine 2"
var quoted = "He said \"hello\""
```

#### 2.6.2 Number Literals

Numbers can be integers or floating-point.

```ebnf
Number = [0-9]+ ( '.' [0-9]* )?
```

```codeai
var count = 42
var price = 19.99
var percentage = 0.15
```

#### 2.6.3 Boolean Literals

```codeai
var active = true
var disabled = false
```

#### 2.6.4 Array Literals

Arrays are enclosed in square brackets with comma-separated elements.

```codeai
var empty = []
var numbers = [1, 2, 3]
var mixed = ["hello", 42, true]
var nested = [[1, 2], [3, 4]]
```

### 2.7 Operators and Punctuation

| Symbol | Name | Usage |
|--------|------|-------|
| `=` | Equals | Assignment |
| `[` `]` | Brackets | Array literals |
| `(` `)` | Parentheses | Function calls, grouping |
| `{` `}` | Braces | Block delimiters |
| `,` | Comma | List separator |

---

## 3. Syntax Reference (EBNF Grammar)

### 3.1 Complete Grammar

```ebnf
(* Top-level program *)
Program        = { Statement } ;

(* Statements *)
Statement      = VarDecl
               | Assignment
               | IfStmt
               | ForLoop
               | FunctionDecl
               | ExecBlock ;

(* Variable declaration *)
VarDecl        = "var" Ident "=" Expression ;

(* Assignment *)
Assignment     = Ident "=" Expression ;

(* Conditional statement *)
IfStmt         = "if" Expression Block [ "else" Block ] ;

(* For-in loop *)
ForLoop        = "for" Ident "in" Expression Block ;

(* Function declaration *)
FunctionDecl   = "function" Ident "(" [ ParamList ] ")" Block ;
ParamList      = Ident { "," Ident } ;

(* Shell execution block *)
ExecBlock      = "exec" "{" ShellCommand "}" ;
ShellCommand   = { any character except "}" } ;

(* Block of statements *)
Block          = "{" { Statement } "}" ;

(* Expressions *)
Expression     = StringLiteral
               | NumberLiteral
               | BoolLiteral
               | ArrayLiteral
               | FunctionCall
               | Identifier ;

StringLiteral  = '"' { character } '"' ;
NumberLiteral  = digit { digit } [ "." { digit } ] ;
BoolLiteral    = "true" | "false" ;
ArrayLiteral   = "[" [ Expression { "," Expression } ] "]" ;
FunctionCall   = Ident "(" [ Expression { "," Expression } ] ")" ;
Identifier     = Ident ;

(* Lexical elements *)
Ident          = letter { letter | digit | "_" } ;
letter         = "a" ... "z" | "A" ... "Z" | "_" ;
digit          = "0" ... "9" ;
```

### 3.2 Grammar Notes

| Feature | Status | Notes |
|---------|--------|-------|
| Trailing commas | Allowed | Parser is lenient |
| Empty blocks | Allowed | `if x { }` is valid |
| Nested blocks | Supported | Arbitrary nesting depth |
| Forward references | Not supported | Declare before use |

---

## 4. Type System

### 4.1 Primitive Types

CodeAI uses a dynamically-inferred type system with the following primitive types:

| Type | Description | Example |
|------|-------------|---------|
| `string` | Text values | `"hello"` |
| `number` | Numeric values (64-bit float) | `42`, `3.14` |
| `bool` | Boolean values | `true`, `false` |
| `array` | Ordered collection | `[1, 2, 3]` |
| `function` | Callable functions | `function add(a, b) { }` |
| `void` | No value | Statement results |
| `unknown` | Undetermined type | Unresolved references |

### 4.2 Type Inference

Types are inferred from literal values and expressions:

```codeai
var name = "Alice"     // Inferred: string
var count = 42         // Inferred: number
var active = true      // Inferred: bool
var items = [1, 2, 3]  // Inferred: array
```

### 4.3 Type Compatibility

| From Type | To Type | Compatible |
|-----------|---------|------------|
| `string` | `string` | Yes |
| `number` | `number` | Yes |
| `bool` | `bool` | Yes |
| `array` | `array` | Yes |
| `unknown` | Any | Yes (lenient) |
| Different types | Different types | No |

### 4.4 Iterable Types

Only `array` types can be used in `for-in` loops:

```codeai
var items = [1, 2, 3]
for item in items {    // Valid: array is iterable
    var x = item
}

var count = 5
for n in count {       // Error: number is not iterable
    var x = n
}
```

---

## 5. Statements

### 5.1 Variable Declaration

Declares a new variable and initializes it with a value.

**Syntax:**
```ebnf
VarDecl = "var" Ident "=" Expression ;
```

**Examples:**
```codeai
var name = "Alice"
var count = 0
var items = []
var result = compute()
```

**Semantic Rules:**
- Variable must not already be declared in current scope
- Value expression must be valid
- Type is inferred from value

### 5.2 Assignment

Assigns a new value to an existing variable.

**Syntax:**
```ebnf
Assignment = Ident "=" Expression ;
```

**Examples:**
```codeai
count = 10
name = "Bob"
items = [1, 2, 3]
```

**Semantic Rules:**
- Variable must be declared before assignment
- Cannot assign to function names
- Type may be updated based on new value

### 5.3 If Statement

Conditional execution based on a boolean expression.

**Syntax:**
```ebnf
IfStmt = "if" Expression Block [ "else" Block ] ;
```

**Examples:**
```codeai
// Simple if
if active {
    var msg = "Active"
}

// If with else
if count > 0 {
    var result = process()
} else {
    var result = "empty"
}

// Nested if
if outer {
    if inner {
        var x = 1
    }
}
```

**Semantic Rules:**
- Condition should evaluate to boolean (not enforced at parse time)
- Then and else blocks create new scopes
- Variables declared in blocks are not visible outside

### 5.4 For Loop

Iterates over elements of an iterable (array).

**Syntax:**
```ebnf
ForLoop = "for" Ident "in" Expression Block ;
```

**Examples:**
```codeai
var items = [1, 2, 3]
for item in items {
    var doubled = item
}

var pods = get_pods()
for pod in pods {
    exec { kubectl delete pod $pod }
}
```

**Semantic Rules:**
- Loop variable is declared in loop scope
- Iterable must be an array type
- Loop body creates a new scope

### 5.5 Function Declaration

Declares a reusable function with parameters.

**Syntax:**
```ebnf
FunctionDecl = "function" Ident "(" [ ParamList ] ")" Block ;
ParamList    = Ident { "," Ident } ;
```

**Examples:**
```codeai
// No parameters
function greet() {
    var msg = "Hello"
}

// One parameter
function greet(name) {
    var msg = "Hello"
    exec { echo $msg $name }
}

// Multiple parameters
function add(a, b, c) {
    var sum = a
}
```

**Semantic Rules:**
- Function name must not already be declared
- Parameter names must be unique
- Parameters are visible only in function body
- Functions must be declared before being called

### 5.6 Block

A sequence of statements enclosed in braces.

**Syntax:**
```ebnf
Block = "{" { Statement } "}" ;
```

**Examples:**
```codeai
{
    var x = 1
    var y = 2
}

// Empty block
{ }
```

**Semantic Rules:**
- Creates a new scope
- Variables declared inside are not visible outside
- Can contain any statement types

---

## 6. Expressions

### 6.1 Literals

#### String Literal
```codeai
var s = "Hello, World!"
var empty = ""
var escaped = "Line 1\nLine 2"
```

#### Number Literal
```codeai
var integer = 42
var floating = 3.14159
var zero = 0
```

#### Boolean Literal
```codeai
var yes = true
var no = false
```

#### Array Literal
```codeai
var empty = []
var numbers = [1, 2, 3]
var strings = ["a", "b", "c"]
var mixed = [1, "two", true]
```

### 6.2 Identifier

References a declared variable or function.

```codeai
var x = 10
var y = x          // References x

function greet() { }
var fn = greet     // References function
```

**Semantic Rules:**
- Identifier must be declared before use
- Looks up through scope chain (local then enclosing)

### 6.3 Function Call

Invokes a function with arguments.

**Syntax:**
```ebnf
FunctionCall = Ident "(" [ Expression { "," Expression } ] ")" ;
```

**Examples:**
```codeai
// No arguments
var result = compute()

// One argument
var doubled = double(5)

// Multiple arguments
var sum = add(1, 2, 3)
```

**Semantic Rules:**
- Function must be declared before call
- Argument count must match parameter count
- Cannot call non-function values

---

## 7. Shell Execution

### 7.1 Exec Block

Executes shell commands within the CodeAI runtime.

**Syntax:**
```ebnf
ExecBlock = "exec" "{" ShellCommand "}" ;
```

**Examples:**
```codeai
// Simple command
exec { kubectl get pods }

// Command with variable interpolation
var name = "my-pod"
exec { echo $name }

// Command with pipes
exec { kubectl get pods | grep Running }

// Multi-word command
exec { kubectl delete pod my-app-12345 --force }
```

### 7.2 Variable Interpolation

Shell variables use `$name` syntax:

```codeai
var pod_name = "nginx-abc123"
exec { kubectl logs $pod_name }

var namespace = "production"
exec { kubectl get pods -n $namespace }
```

### 7.3 Shell Features Supported

| Feature | Example | Supported |
|---------|---------|-----------|
| Simple commands | `kubectl get pods` | Yes |
| Arguments | `echo hello world` | Yes |
| Pipes | `kubectl get pods \| grep Running` | Yes |
| Variable interpolation | `echo $name` | Yes |
| Nested braces | `kubectl get pod -o jsonpath='{.metadata.name}'` | Limited |
| Redirects | `echo hello > file.txt` | Yes |

---

## 8. Planned Business DSL

The following constructs are specified in the Implementation Plan but **not yet implemented** in the current parser. They represent the roadmap for CodeAI's business-domain capabilities.

### 8.1 Entity Declarations

Define data models that map to database tables.

```codeai
entity Product {
    description: "Represents a product in the inventory"

    id: uuid, primary, auto
    sku: string, required, unique
    name: string, required, searchable
    description: text, optional
    price: decimal(10,2), required
    quantity: integer, default(0)
    category: ref(Category), required
    tags: list(string), optional

    created_at: timestamp, auto
    updated_at: timestamp, auto_update

    index: [category, created_at]
}
```

#### Planned Field Types

| Type | Description |
|------|-------------|
| `uuid` | Universally unique identifier |
| `string` | Variable length text (max 255) |
| `text` | Unlimited length text |
| `integer` | 64-bit signed integer |
| `decimal(p,s)` | Fixed precision decimal |
| `boolean` | True/false value |
| `timestamp` | Date and time with timezone |
| `date` | Date without time |
| `time` | Time without date |
| `json` | Arbitrary JSON data |
| `list(T)` | Array of type T |
| `ref(Entity)` | Foreign key reference |
| `enum(a,b,c)` | Enumerated values |

#### Planned Field Modifiers

| Modifier | Description |
|----------|-------------|
| `primary` | Primary key field |
| `auto` | Auto-generated value |
| `required` | Field cannot be null |
| `optional` | Field can be null |
| `unique` | Value must be unique |
| `searchable` | Create full-text index |
| `default(value)` | Default value |
| `soft_delete` | Enable soft deletion |

### 8.2 Endpoint Declarations

Define HTTP API routes.

```codeai
endpoint GET /products {
    description: "List all products"
    auth: required
    roles: [admin, user]

    query {
        category: uuid, optional
        page: integer, default(1)
        limit: integer, default(20)
    }

    returns: paginated(Product)
}

endpoint POST /products {
    description: "Create a new product"
    auth: required
    roles: [admin]

    body {
        sku: string, required
        name: string, required
        price: decimal, required
    }

    returns: Product
    on_success: emit(ProductCreated)
}
```

### 8.3 Workflow Declarations

Define multi-step processes with state persistence.

```codeai
workflow OrderFulfillment {
    description: "Process order from placement to delivery"
    trigger: OrderPlaced

    steps {
        validate_inventory {
            for_each: trigger.order.items
            check: item.product.quantity >= item.quantity
            on_fail: cancel_order("Insufficient inventory")
        }

        process_payment {
            call: PaymentGateway.charge {
                amount: trigger.order.total
            }
            timeout: 30s
            retry: 3 times with exponential_backoff
            on_fail: rollback
        }

        notify_customer {
            send: email(trigger.order.customer.email) {
                template: "order_shipped"
            }
        }
    }

    on_complete: update(trigger.order.status = "completed")
    on_fail: emit(OrderFailed)
}
```

### 8.4 Job Declarations

Define scheduled or recurring background tasks.

```codeai
job DailyInventoryReport {
    description: "Generate daily inventory report"
    schedule: "0 6 * * *"
    timezone: "UTC"

    steps {
        fetch_data {
            query: select Product where quantity < 10
            as: low_stock_items
        }

        generate_report {
            template: "inventory_report"
            data: { low_stock: low_stock_items }
            format: pdf
        }

        distribute {
            send: email(config.reports.recipients) {
                subject: "Daily Inventory Report"
                attachment: report_file
            }
        }
    }

    retry: 3 times
    timeout: 10m
}

job CleanupExpiredSessions {
    schedule: every 1h
    action: delete Session where expires_at < now()
}
```

### 8.5 Integration Declarations

Define connections to external services.

```codeai
integration PaymentGateway {
    description: "Stripe payment processing"
    type: rest
    base_url: env(STRIPE_API_URL)

    auth: bearer(env(STRIPE_SECRET_KEY))

    timeout: 30s
    retry: 3 times with exponential_backoff
    circuit_breaker: {
        threshold: 5 failures in 1m
        reset_after: 30s
    }

    operation charge {
        method: POST
        path: "/charges"
        body: {
            amount: integer, required
            currency: string, default("usd")
        }
        returns: {
            id: string
            status: string
        }
    }
}
```

### 8.6 Event Declarations

Define messages for event-driven architecture.

```codeai
event ProductCreated {
    description: "Emitted when a new product is added"

    payload {
        product_id: uuid
        sku: string
        name: string
        created_at: timestamp
    }

    publish_to: [kafka("products"), webhook("inventory-updates")]
}

event LowStockAlert {
    payload {
        product_id: uuid
        current_quantity: integer
    }

    trigger: when Product.quantity < 10
    publish_to: [slack("#inventory-alerts")]
}
```

### 8.7 Configuration Block

Application metadata and runtime settings.

```codeai
config {
    name: "inventory-service"
    version: "1.0.0"

    database: postgres {
        pool_size: 20
        timeout: 30s
    }

    cache: redis {
        ttl: 5m
    }

    auth: jwt {
        issuer: env(JWT_ISSUER)
    }
}
```

---

## 9. Examples

### 9.1 Simple Variable Declaration

```codeai
var name = "test"
var count = 10
var active = true
var items = [1, 2, 3]
```

### 9.2 Function with Shell Execution

```codeai
function deploy(namespace, image) {
    var config = "deployment.yaml"
    exec { kubectl apply -f $config -n $namespace }
    exec { kubectl set image deployment/app app=$image }
}
```

### 9.3 Conditional Logic

```codeai
var environment = "production"

if environment {
    var replicas = 3
    exec { kubectl scale deployment/app --replicas=3 }
} else {
    var replicas = 1
    exec { kubectl scale deployment/app --replicas=1 }
}
```

### 9.4 Iterating Over Resources

```codeai
var pods = get_pods()

for pod in pods {
    exec { kubectl describe pod $pod }
}
```

### 9.5 Complex Deployment Script

```codeai
// Configuration
var namespace = "production"
var app_name = "my-service"
var image_tag = "v2.1.0"

// Helper functions
function check_health(service) {
    exec { kubectl rollout status deployment/$service -n $namespace }
}

function notify(message) {
    exec { curl -X POST $SLACK_WEBHOOK -d '{"text": "$message"}' }
}

// Main deployment logic
function deploy() {
    var image = "registry.example.com/$app_name:$image_tag"

    // Pre-deployment checks
    exec { kubectl get namespace $namespace }

    // Apply deployment
    exec { kubectl set image deployment/$app_name app=$image -n $namespace }

    // Wait for rollout
    check_health(app_name)

    // Notify team
    notify("Deployment of $app_name:$image_tag complete")
}

// Execute deployment
var result = deploy()
```

### 9.6 Resource Cleanup Script

```codeai
var namespaces = ["dev", "staging", "temp"]
var cutoff_days = 7

for ns in namespaces {
    // Get old pods
    exec { kubectl get pods -n $ns --sort-by=.metadata.creationTimestamp }

    // Clean up completed jobs
    exec { kubectl delete jobs --field-selector status.successful=1 -n $ns }

    // Clean up old pods
    if cleanup_enabled {
        exec { kubectl delete pods --field-selector=status.phase==Succeeded -n $ns }
    }
}
```

### 9.7 Test Execution Script

```codeai
var test_suites = ["unit", "integration", "e2e"]
var results = []

function run_tests(suite) {
    exec { npm run test:$suite }
}

function generate_report() {
    exec { npm run coverage:report }
}

// Run all test suites
for suite in test_suites {
    var result = run_tests(suite)
}

// Generate final report
generate_report()
```

---

## 10. Error Handling

### 10.1 Parse Errors

The parser provides detailed error messages with position information.

**Format:**
```
<filename>:<line>:<column>: <message>
```

**Examples:**
```
example.cai:5:12: unexpected token
example.cai:10:1: unclosed string literal
example.cai:15:8: unclosed brace
```

### 10.2 Validation Errors

The validator checks semantic correctness after parsing.

| Error Type | Description | Example |
|------------|-------------|---------|
| `DuplicateDeclaration` | Variable/function already declared | `var x = 1; var x = 2` |
| `UndefinedVariable` | Variable not declared | Using `y` without declaration |
| `UndefinedFunction` | Function not declared | Calling `foo()` without definition |
| `NotAFunction` | Calling non-function value | `var x = 1; x()` |
| `WrongArgCount` | Incorrect argument count | `function f(a) {}; f(1, 2)` |
| `CannotIterate` | Non-iterable in for loop | `for x in 5 { }` |
| `DuplicateParameter` | Same parameter name twice | `function f(a, a) {}` |

### 10.3 Error Recovery

The parser implements fault-tolerant recovery for common LLM mistakes:

| Mistake | Recovery |
|---------|----------|
| Extra trailing comma | Ignored |
| Missing trailing semicolon | Not required |
| Inconsistent indentation | Whitespace normalized |
| Extra whitespace | Trimmed |

---

## Appendix A: AST Node Types

| Node Type | Description |
|-----------|-------------|
| `Program` | Root node containing statements |
| `VarDecl` | Variable declaration |
| `Assignment` | Variable assignment |
| `IfStmt` | Conditional statement |
| `ForLoop` | For-in loop |
| `FunctionDecl` | Function declaration |
| `ExecBlock` | Shell execution block |
| `Block` | Statement block |
| `ReturnStmt` | Return statement |
| `StringLiteral` | String value |
| `NumberLiteral` | Numeric value |
| `BoolLiteral` | Boolean value |
| `Identifier` | Variable/function reference |
| `FunctionCall` | Function invocation |
| `ArrayLiteral` | Array value |
| `BinaryExpr` | Binary operation |
| `UnaryExpr` | Unary operation |

---

## Appendix B: Parser Implementation Reference

| File | Purpose |
|------|---------|
| `internal/parser/parser.go` | Participle grammar and parser |
| `internal/parser/parser_test.go` | Parser test cases |
| `internal/ast/ast.go` | AST node definitions |
| `internal/ast/types.go` | Type system definitions |
| `internal/validator/validator.go` | Semantic validation |
| `internal/validator/type_checker.go` | Type inference |
| `internal/validator/symbol_table.go` | Scope management |
| `internal/validator/errors.go` | Error types |

---

## Appendix C: Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2026-01-12 | Initial specification documenting current implementation |

---

*This document describes the CodeAI DSL as currently implemented. For the planned business DSL features (entities, endpoints, workflows, jobs, integrations, events), see the CodeAI Implementation Plan.*

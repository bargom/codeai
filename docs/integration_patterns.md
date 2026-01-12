# Integration Patterns Guide

This guide covers resilience patterns and best practices for integrating with external APIs in CodeAI. All integration components are located in `pkg/integration/`.

## Table of Contents

- [Circuit Breaker Pattern](#circuit-breaker-pattern)
- [Retry Strategies](#retry-strategies)
- [Timeout Handling](#timeout-handling)
- [REST Client](#rest-client)
- [GraphQL Client](#graphql-client)
- [Best Practices](#best-practices)

---

## Circuit Breaker Pattern

The circuit breaker pattern prevents cascading failures by stopping requests to a failing service, giving it time to recover.

### State Transitions

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  ┌──────────┐    failures >= threshold    ┌──────────┐         │
│  │  CLOSED  │ ─────────────────────────▶ │   OPEN   │         │
│  │          │                             │          │         │
│  │ (normal) │                             │ (reject) │         │
│  └──────────┘                             └──────────┘         │
│       ▲                                        │               │
│       │                                        │ timeout       │
│       │ successes >= halfOpenRequests          │ elapsed       │
│       │                                        ▼               │
│       │                                  ┌───────────┐         │
│       └───────────────────────────────── │ HALF-OPEN │         │
│                                          │           │         │
│                  any failure             │  (probe)  │ ◀──┐    │
│                  ┌───────────────────────┴───────────┘    │    │
│                  │                                        │    │
│                  └────────────────────────────────────────┘    │
│                           reopens circuit                      │
└─────────────────────────────────────────────────────────────────┘
```

**States:**
- **Closed**: Normal operation. Requests pass through. Failures are counted.
- **Open**: All requests are rejected immediately with `ErrCircuitOpen`.
- **Half-Open**: Limited requests are allowed to test if the service has recovered.

### Configuration Options

```go
type CircuitBreakerConfig struct {
    // FailureThreshold is the number of failures before opening the circuit.
    // Default: 5
    FailureThreshold int

    // Timeout is the duration the circuit stays open before transitioning to half-open.
    // Default: 60s
    Timeout time.Duration

    // HalfOpenRequests is the number of successful requests required in half-open state
    // to transition back to closed.
    // Default: 3
    HalfOpenRequests int

    // OnStateChange is called when the circuit state changes.
    OnStateChange func(from, to CircuitState)
}
```

### Example Usage

```go
package main

import (
    "context"
    "time"

    "github.com/bargom/codeai/pkg/integration"
)

func main() {
    // Create a circuit breaker with custom configuration
    config := integration.CircuitBreakerConfig{
        FailureThreshold: 3,             // Open after 3 failures
        Timeout:          30 * time.Second, // Stay open for 30s
        HalfOpenRequests: 2,             // Need 2 successes to close
        OnStateChange: func(from, to integration.CircuitState) {
            log.Printf("Circuit state changed: %s -> %s", from, to)
        },
    }

    cb := integration.NewCircuitBreaker("payment-service", config)

    // Execute with circuit breaker protection
    err := cb.Execute(ctx, func(ctx context.Context) error {
        return callPaymentAPI(ctx)
    })

    if err == integration.ErrCircuitOpen {
        // Circuit is open, use fallback
        return useCachedPaymentMethod()
    }
}
```

### Circuit Breaker Registry

For managing multiple circuit breakers:

```go
// Create a registry with default configuration
registry := integration.NewCircuitBreakerRegistry(integration.DefaultCircuitBreakerConfig())

// Get or create circuit breakers by name
paymentCB := registry.Get("payment-service")
inventoryCB := registry.Get("inventory-service")

// Register with custom config
customCB := registry.Register("critical-service", integration.CircuitBreakerConfig{
    FailureThreshold: 2,
    Timeout:          2 * time.Minute,
})

// Get statistics for all circuit breakers
stats := registry.Stats()
for _, s := range stats {
    fmt.Printf("Service: %s, State: %s, Failures: %d\n",
        s.Name, s.State, s.Failures)
}
```

### Fallback Strategies

When the circuit is open:

1. **Return cached data**: Use the last known good response
2. **Return default values**: Provide sensible defaults
3. **Graceful degradation**: Disable non-critical features
4. **Queue for later**: Store requests to retry when service recovers

```go
func GetUserProfile(ctx context.Context, userID string) (*Profile, error) {
    err := cb.Execute(ctx, func(ctx context.Context) error {
        profile, err := fetchProfileFromAPI(ctx, userID)
        if err != nil {
            return err
        }
        cache.Set(userID, profile) // Cache successful responses
        return nil
    })

    if err == integration.ErrCircuitOpen {
        // Fallback to cache
        if cached, ok := cache.Get(userID); ok {
            return cached.(*Profile), nil
        }
        // Return minimal profile
        return &Profile{ID: userID, Status: "unavailable"}, nil
    }

    return profile, err
}
```

---

## Retry Strategies

The retry mechanism handles transient failures with exponential backoff and jitter.

### Configuration Options

```go
type RetryConfig struct {
    // MaxAttempts is the maximum number of attempts (including the first one).
    // Default: 3
    MaxAttempts int

    // BaseDelay is the initial delay between retries.
    // Default: 100ms
    BaseDelay time.Duration

    // MaxDelay is the maximum delay between retries.
    // Default: 30s
    MaxDelay time.Duration

    // Multiplier is the factor by which the delay increases.
    // Default: 2.0
    Multiplier float64

    // Jitter adds randomness to delays to prevent thundering herd.
    // Value between 0 and 1 representing the percentage of jitter (e.g., 0.25 = +/-25%).
    // Default: 0.25
    Jitter float64

    // RetryIf is a function that determines if an error should be retried.
    // If nil, uses the default retryable check.
    RetryIf func(err error) bool

    // OnRetry is called before each retry attempt.
    OnRetry func(attempt int, err error, delay time.Duration)
}
```

### Exponential Backoff with Jitter

The delay between retries follows this pattern:

```
Attempt 1: BaseDelay                    (100ms)
Attempt 2: BaseDelay * Multiplier       (200ms)
Attempt 3: BaseDelay * Multiplier^2     (400ms)
Attempt 4: BaseDelay * Multiplier^3     (800ms)
... capped at MaxDelay

With 25% jitter:
- 100ms becomes 75ms-125ms
- 200ms becomes 150ms-250ms
- etc.
```

Jitter prevents the "thundering herd" problem where many clients retry simultaneously.

### Retryable Errors

By default, these errors are retried:

| Error Type | Examples |
|------------|----------|
| Timeout errors | `context.DeadlineExceeded` |
| Connection errors | Connection refused, reset, broken pipe |
| Network errors | DNS failures, network unreachable |
| HTTP 5xx errors | 500, 502, 503, 504 |
| HTTP 429 | Too Many Requests |

Non-retryable errors (fail immediately):
- HTTP 4xx (except 429)
- Business logic errors
- Authentication failures

### Example Usage

```go
package main

import (
    "context"
    "time"

    "github.com/bargom/codeai/pkg/integration"
)

func main() {
    // Simple retry with defaults
    err := integration.Retry(ctx, func(ctx context.Context) error {
        return callExternalAPI(ctx)
    })

    // Custom retry configuration
    config := integration.RetryConfig{
        MaxAttempts: 5,
        BaseDelay:   200 * time.Millisecond,
        MaxDelay:    10 * time.Second,
        Multiplier:  2.0,
        Jitter:      0.3,
        OnRetry: func(attempt int, err error, delay time.Duration) {
            log.Printf("Retry attempt %d after error: %v (waiting %v)",
                attempt, err, delay)
        },
    }

    err = integration.RetryWithConfig(ctx, config, func(ctx context.Context) error {
        return callExternalAPI(ctx)
    })
}
```

### Custom Retry Conditions

```go
config := integration.RetryConfig{
    MaxAttempts: 3,
    RetryIf: func(err error) bool {
        // Only retry on specific errors
        var httpErr *integration.HTTPError
        if errors.As(err, &httpErr) {
            // Retry on 503 Service Unavailable only
            return httpErr.StatusCode == 503
        }
        // Retry on network errors
        return integration.IsRetryable(err)
    },
}
```

### Retry with Result

For functions that return a value:

```go
retryer := integration.NewRetryer(config).WithService("github-api", "/users")

result, err := integration.DoWithResult(ctx, retryer, func(ctx context.Context) (*User, error) {
    return fetchUser(ctx, userID)
})
```

---

## Timeout Handling

Timeouts prevent requests from hanging indefinitely and ensure predictable response times.

### Timeout Types

```go
type TimeoutConfig struct {
    // Default is the default timeout for operations.
    // Default: 30s
    Default time.Duration

    // Connect is the timeout for establishing connections.
    // Default: 10s
    Connect time.Duration

    // Read is the timeout for reading response.
    // Default: 30s
    Read time.Duration

    // Write is the timeout for writing request.
    // Default: 30s
    Write time.Duration

    // OnTimeout is called when a timeout occurs.
    OnTimeout func(operation string, timeout time.Duration)
}
```

### Request vs Connection Timeouts

| Timeout Type | Purpose | Typical Value |
|--------------|---------|---------------|
| **Connect** | Time to establish TCP connection | 5-10s |
| **Read** | Time to read response body | 30-60s |
| **Write** | Time to send request body | 30s |
| **Default** | Overall operation timeout | 30-60s |

### Context Propagation

Timeouts respect the parent context's deadline:

```go
tm := integration.NewTimeoutManager(config)

// If parent context has deadline of 5s, a 10s timeout will be capped at 5s
ctx, cancel := tm.WithTimeout(parentCtx, 10*time.Second)
defer cancel()

err := doWork(ctx)
```

### Example Usage

```go
package main

import (
    "context"
    "time"

    "github.com/bargom/codeai/pkg/integration"
)

func main() {
    config := integration.TimeoutConfig{
        Default: 30 * time.Second,
        Connect: 5 * time.Second,
        Read:    60 * time.Second,
        OnTimeout: func(operation string, timeout time.Duration) {
            log.Printf("Operation %s timed out after %v", operation, timeout)
            metrics.RecordTimeout(operation)
        },
    }

    tm := integration.NewTimeoutManager(config).WithService("api", "/data")

    // Execute with timeout
    err := tm.Execute(ctx, 10*time.Second, "fetch-data", func(ctx context.Context) error {
        return fetchData(ctx)
    })

    if err == integration.ErrTimeout {
        // Handle timeout specifically
        return getCachedData()
    }
}
```

### Per-Endpoint Timeouts

Different endpoints may need different timeouts:

```go
// Fast endpoint - short timeout
err := tm.Execute(ctx, 2*time.Second, "health-check", checkHealth)

// Slow endpoint - longer timeout
err := tm.Execute(ctx, 60*time.Second, "generate-report", generateReport)

// Using request options with REST client
resp, err := client.Get(ctx, "/quick-check", rest.WithTimeout(2*time.Second))
resp, err := client.Post(ctx, "/long-process", data, rest.WithTimeout(5*time.Minute))
```

### Timeout Context Utilities

```go
// Create a timeout context with tracking
tc := integration.NewTimeoutContext(ctx, 30*time.Second)
defer tc.Cancel()

// Check remaining time
remaining := tc.Remaining()
if remaining < 5*time.Second {
    // Not enough time, use fast path
    return quickResult()
}

// Check if expired
if tc.IsExpired() {
    return ErrTimeout
}

// Extend timeout if needed
tc = tc.Extend(10*time.Second)
```

---

## REST Client

The REST client provides a full-featured HTTP client with built-in resilience patterns.

### Creating a Client

```go
package main

import (
    "time"

    "github.com/bargom/codeai/pkg/integration"
    "github.com/bargom/codeai/pkg/integration/rest"
)

func main() {
    // Using the config builder
    config := integration.NewConfigBuilder("github").
        BaseURL("https://api.github.com").
        Timeout(30 * time.Second).
        MaxRetries(3).
        RetryDelay(100 * time.Millisecond).
        CircuitBreakerThreshold(5).
        BearerAuth(os.Getenv("GITHUB_TOKEN")).
        Header("Accept", "application/vnd.github.v3+json").
        Build()

    client, err := rest.New(config)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Authentication Types

#### Bearer Token

```go
config := integration.NewConfigBuilder("api").
    BaseURL("https://api.example.com").
    BearerAuth("your-token-here").
    Build()
```

#### API Key

```go
config := integration.NewConfigBuilder("api").
    BaseURL("https://api.example.com").
    APIKeyAuth("your-api-key", "X-API-Key"). // header name is optional
    Build()
```

#### Basic Auth

```go
config := integration.NewConfigBuilder("api").
    BaseURL("https://api.example.com").
    BasicAuth("username", "password").
    Build()
```

#### OAuth2

```go
oauth2Config := &integration.OAuth2Config{
    ClientID:     "client-id",
    ClientSecret: "client-secret",
    TokenURL:     "https://auth.example.com/oauth/token",
    Scopes:       []string{"read", "write"},
}

config := integration.NewConfigBuilder("api").
    BaseURL("https://api.example.com").
    OAuth2Auth(oauth2Config).
    Build()
```

### Making Requests

```go
// GET request
resp, err := client.Get(ctx, "/users/123")

// POST request with body
user := &User{Name: "John", Email: "john@example.com"}
resp, err := client.Post(ctx, "/users", user)

// PUT request
resp, err := client.Put(ctx, "/users/123", updatedUser)

// PATCH request
resp, err := client.Patch(ctx, "/users/123", partialUpdate)

// DELETE request
resp, err := client.Delete(ctx, "/users/123")
```

### Request Options

```go
// Add query parameters
params := url.Values{"page": {"1"}, "limit": {"20"}}
resp, err := client.Get(ctx, "/users", rest.WithQuery(params))

// Add headers
resp, err := client.Get(ctx, "/users",
    rest.WithHeader("X-Request-ID", "abc123"),
    rest.WithHeader("X-Custom", "value"),
)

// Set custom timeout
resp, err := client.Get(ctx, "/slow-endpoint", rest.WithTimeout(2*time.Minute))

// Skip retry for this request
resp, err := client.Post(ctx, "/idempotent", data, rest.WithSkipRetry())

// Skip circuit breaker
resp, err := client.Get(ctx, "/health", rest.WithSkipCircuit())
```

### Handling Responses

```go
resp, err := client.Get(ctx, "/users/123")
if err != nil {
    var httpErr *integration.HTTPError
    if errors.As(err, &httpErr) {
        if httpErr.StatusCode == 404 {
            return nil, ErrUserNotFound
        }
        log.Printf("HTTP error: %d - %s", httpErr.StatusCode, httpErr.Body)
    }
    return nil, err
}

// Check response status
if resp.IsSuccess() {
    var user User
    if err := resp.UnmarshalJSON(&user); err != nil {
        return nil, err
    }
    return &user, nil
}

if resp.IsClientError() {
    // Handle 4xx errors
}

if resp.IsServerError() {
    // Handle 5xx errors
}
```

### Error Mapping

Map API errors to domain errors:

```go
func (c *GitHubClient) GetRepo(ctx context.Context, owner, name string) (*Repo, error) {
    resp, err := c.client.Get(ctx, fmt.Sprintf("/repos/%s/%s", owner, name))
    if err != nil {
        return nil, c.mapError(err, "get repo")
    }

    var repo Repo
    return &repo, resp.UnmarshalJSON(&repo)
}

func (c *GitHubClient) mapError(err error, operation string) error {
    if err == integration.ErrCircuitOpen {
        return fmt.Errorf("github service unavailable: %w", ErrServiceUnavailable)
    }

    var httpErr *integration.HTTPError
    if errors.As(err, &httpErr) {
        switch httpErr.StatusCode {
        case 401:
            return ErrUnauthorized
        case 403:
            return ErrForbidden
        case 404:
            return ErrNotFound
        case 422:
            return fmt.Errorf("validation error: %s", httpErr.Body)
        case 429:
            return ErrRateLimited
        }
    }

    return fmt.Errorf("%s failed: %w", operation, err)
}
```

### Complete Example: Third-Party API Integration

```go
package stripe

import (
    "context"
    "fmt"
    "os"
    "time"

    "github.com/bargom/codeai/pkg/integration"
    "github.com/bargom/codeai/pkg/integration/rest"
)

type Client struct {
    client *rest.Client
}

func NewClient() (*Client, error) {
    config := integration.NewConfigBuilder("stripe").
        BaseURL("https://api.stripe.com/v1").
        Timeout(30 * time.Second).
        ConnectTimeout(5 * time.Second).
        MaxRetries(3).
        RetryDelay(500 * time.Millisecond).
        MaxRetryDelay(5 * time.Second).
        RetryJitter(0.2).
        CircuitBreakerThreshold(10).
        CircuitBreakerTimeout(1 * time.Minute).
        BearerAuth(os.Getenv("STRIPE_SECRET_KEY")).
        Header("Stripe-Version", "2023-10-16").
        UserAgent("CodeAI/1.0").
        Build()

    client, err := rest.New(config)
    if err != nil {
        return nil, err
    }

    return &Client{client: client}, nil
}

type Charge struct {
    ID       string `json:"id"`
    Amount   int64  `json:"amount"`
    Currency string `json:"currency"`
    Status   string `json:"status"`
}

type CreateChargeRequest struct {
    Amount   int64  `json:"amount"`
    Currency string `json:"currency"`
    Source   string `json:"source"`
}

func (c *Client) CreateCharge(ctx context.Context, req *CreateChargeRequest) (*Charge, error) {
    resp, err := c.client.Post(ctx, "/charges", req)
    if err != nil {
        return nil, c.handleError(err)
    }

    var charge Charge
    if err := resp.UnmarshalJSON(&charge); err != nil {
        return nil, fmt.Errorf("failed to parse charge: %w", err)
    }

    return &charge, nil
}

func (c *Client) GetCharge(ctx context.Context, id string) (*Charge, error) {
    resp, err := c.client.Get(ctx, "/charges/"+id)
    if err != nil {
        return nil, c.handleError(err)
    }

    var charge Charge
    if err := resp.UnmarshalJSON(&charge); err != nil {
        return nil, fmt.Errorf("failed to parse charge: %w", err)
    }

    return &charge, nil
}

func (c *Client) handleError(err error) error {
    if err == integration.ErrCircuitOpen {
        return ErrStripeUnavailable
    }
    if err == integration.ErrTimeout {
        return ErrStripeTimeout
    }

    var httpErr *integration.HTTPError
    if errors.As(err, &httpErr) {
        switch httpErr.StatusCode {
        case 401:
            return ErrInvalidAPIKey
        case 402:
            return ErrPaymentRequired
        case 429:
            return ErrRateLimited
        }
    }

    return err
}
```

---

## GraphQL Client

The GraphQL client supports queries, mutations, and subscriptions with full resilience patterns.

### Creating a Client

```go
package main

import (
    "time"

    "github.com/bargom/codeai/pkg/integration"
    "github.com/bargom/codeai/pkg/integration/graphql"
)

func main() {
    config := integration.NewConfigBuilder("github-graphql").
        BaseURL("https://api.github.com/graphql").
        Timeout(30 * time.Second).
        MaxRetries(3).
        CircuitBreakerThreshold(5).
        BearerAuth(os.Getenv("GITHUB_TOKEN")).
        Build()

    client, err := graphql.New(config)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Simple Queries

```go
// Query with variables
query := `
    query GetUser($login: String!) {
        user(login: $login) {
            id
            name
            email
            repositories(first: 10) {
                nodes {
                    name
                    stargazerCount
                }
            }
        }
    }
`

var result struct {
    User struct {
        ID    string
        Name  string
        Email string
        Repositories struct {
            Nodes []struct {
                Name           string
                StargazerCount int
            }
        }
    }
}

err := client.Query(ctx, query, map[string]interface{}{
    "login": "octocat",
}, &result)

if err != nil {
    log.Fatal(err)
}

fmt.Printf("User: %s (%s)\n", result.User.Name, result.User.ID)
```

### Mutations

```go
mutation := `
    mutation CreateIssue($input: CreateIssueInput!) {
        createIssue(input: $input) {
            issue {
                id
                number
                title
            }
        }
    }
`

var result struct {
    CreateIssue struct {
        Issue struct {
            ID     string
            Number int
            Title  string
        }
    }
}

err := client.Mutate(ctx, mutation, map[string]interface{}{
    "input": map[string]interface{}{
        "repositoryId": "R_kgDOBxxxxxxx",
        "title":        "Bug report",
        "body":         "Found a bug...",
    },
}, &result)
```

### Query Builder

Build queries programmatically:

```go
// Simple query
query := graphql.NewQuery().
    Name("GetUser").
    Variable("login", "String!", nil).
    Field("user").
        VarArg("login", "login").
        Select("id", "name", "email").
        SubField("repositories").
            Arg("first", 10).
            Select("totalCount").
            SubField("nodes").
                Select("name", "stargazerCount").
            Done().
        Done().
    Done().
    Build()

// Output:
// query GetUser($login: String!) {
//   user(login: $login) {
//     id name email
//     repositories(first: 10) {
//       totalCount
//       nodes { name stargazerCount }
//     }
//   }
// }
```

### Variable Binding

```go
// Define variables in the query
query := graphql.NewQuery().
    Name("SearchRepos").
    Variable("query", "String!").
    Variable("first", "Int", 10). // with default value
    Field("search").
        VarArg("query", "query").
        VarArg("first", "first").
        Arg("type", "REPOSITORY").
        Select("repositoryCount").
        SubField("nodes").
            Fragment("RepoFields").
        Done().
    Done().
    Build()

// Execute with variable values
vars := map[string]interface{}{
    "query": "language:go stars:>100",
    "first": 20,
}

resp, err := client.Execute(ctx, &graphql.Request{
    Query:     query,
    Variables: vars,
}, nil)
```

### Fragments

```go
// Named fragment
fragment := graphql.NewFragment("RepoFields", "Repository").
    Fields("id", "name", "description").
    Field("owner", "login", "avatarUrl").
    Build()

// Output: fragment RepoFields on Repository { id name description owner { login avatarUrl } }

// Use in query
query := graphql.NewQuery().
    Field("repository").
        Arg("owner", "octocat").
        Arg("name", "hello-world").
        Fragment("RepoFields").
    Done().
    Build()

// Full query with fragment
fullQuery := query + "\n" + fragment
```

### Error Handling

```go
resp, err := client.Execute(ctx, req, nil)
if err != nil {
    // Network/HTTP error
    var httpErr *integration.HTTPError
    if errors.As(err, &httpErr) {
        log.Printf("HTTP error: %d", httpErr.StatusCode)
    }
    return err
}

// Check for GraphQL errors
if resp.HasError() {
    firstErr := resp.FirstError()
    log.Printf("GraphQL error: %s at line %d",
        firstErr.Message,
        firstErr.Locations[0].Line)

    // Check error extensions for error codes
    if code, ok := firstErr.Extensions["code"].(string); ok {
        switch code {
        case "FORBIDDEN":
            return ErrForbidden
        case "NOT_FOUND":
            return ErrNotFound
        }
    }

    return &graphql.GraphQLError{Errors: resp.Errors}
}

// Parse successful data
var data MyData
if err := resp.UnmarshalData(&data); err != nil {
    return err
}
```

### Complete Example

```go
package github

import (
    "context"
    "fmt"

    "github.com/bargom/codeai/pkg/integration"
    "github.com/bargom/codeai/pkg/integration/graphql"
)

type Client struct {
    gql *graphql.Client
}

func NewClient(token string) (*Client, error) {
    config := integration.NewConfigBuilder("github-graphql").
        BaseURL("https://api.github.com/graphql").
        Timeout(30 * time.Second).
        MaxRetries(3).
        CircuitBreakerThreshold(5).
        BearerAuth(token).
        Build()

    client, err := graphql.New(config)
    if err != nil {
        return nil, err
    }

    return &Client{gql: client}, nil
}

type Repository struct {
    ID          string
    Name        string
    Description string
    Stars       int
}

func (c *Client) GetRepository(ctx context.Context, owner, name string) (*Repository, error) {
    query := `
        query GetRepo($owner: String!, $name: String!) {
            repository(owner: $owner, name: $name) {
                id
                name
                description
                stargazerCount
            }
        }
    `

    var result struct {
        Repository struct {
            ID             string `json:"id"`
            Name           string `json:"name"`
            Description    string `json:"description"`
            StargazerCount int    `json:"stargazerCount"`
        } `json:"repository"`
    }

    err := c.gql.Query(ctx, query, map[string]interface{}{
        "owner": owner,
        "name":  name,
    }, &result)

    if err != nil {
        return nil, fmt.Errorf("failed to get repository: %w", err)
    }

    return &Repository{
        ID:          result.Repository.ID,
        Name:        result.Repository.Name,
        Description: result.Repository.Description,
        Stars:       result.Repository.StargazerCount,
    }, nil
}
```

---

## Best Practices

### When to Use Circuit Breakers

**Use circuit breakers when:**
- Calling external services that may become unavailable
- The service has known failure modes (timeouts, errors)
- Fallback behavior is acceptable
- You want to protect downstream services from cascading failures

**Don't use circuit breakers for:**
- Database connections (use connection pools instead)
- Message queues (they have their own resilience)
- Internal service calls within the same failure domain

### Retry vs Circuit Breaker Decision Tree

```
                    Is it a transient error?
                            │
               ┌────────────┴────────────┐
               ▼                         ▼
              YES                        NO
               │                         │
    ┌──────────┴──────────┐              │
    ▼                     ▼              ▼
Is the service        Is it a         Don't retry
responsive?          timeout?         (fail fast)
    │                    │
    ▼                    ▼
   YES                  YES
    │                    │
    ▼                    ▼
  Retry              Retry with
(backoff)          shorter timeout
    │                    │
    └────────┬───────────┘
             ▼
    Multiple retries failed?
             │
    ┌────────┴────────┐
    ▼                 ▼
   YES                NO
    │                 │
    ▼                 ▼
Open circuit      Continue
(stop trying)     normally
```

### Monitoring Integration Health

Key metrics to track:

```go
// Request metrics
- integration_requests_total{service, endpoint, status}
- integration_request_duration_seconds{service, endpoint}
- integration_errors_total{service, endpoint, error_type}

// Circuit breaker metrics
- circuit_breaker_state{service} (0=closed, 1=open, 2=half-open)
- circuit_breaker_failures_total{service}
- circuit_breaker_state_changes_total{service, from, to}

// Retry metrics
- integration_retries_total{service, endpoint}
```

### Recommended Default Configuration

```go
// For most external APIs
config := integration.NewConfigBuilder("external-api").
    Timeout(30 * time.Second).
    ConnectTimeout(5 * time.Second).
    MaxRetries(3).
    RetryDelay(100 * time.Millisecond).
    MaxRetryDelay(10 * time.Second).
    RetryJitter(0.25).
    CircuitBreakerThreshold(5).
    CircuitBreakerTimeout(60 * time.Second).
    CircuitBreakerHalfOpenRequests(3).
    Build()

// For critical, high-throughput APIs
config := integration.NewConfigBuilder("payment-api").
    Timeout(10 * time.Second).
    ConnectTimeout(2 * time.Second).
    MaxRetries(2).                        // Fewer retries
    RetryDelay(50 * time.Millisecond).    // Faster initial retry
    CircuitBreakerThreshold(3).           // Open faster
    CircuitBreakerTimeout(30 * time.Second). // Recover faster
    Build()

// For non-critical, slow APIs
config := integration.NewConfigBuilder("analytics-api").
    Timeout(5 * time.Minute).
    MaxRetries(5).
    RetryDelay(1 * time.Second).
    CircuitBreakerThreshold(10).          // More tolerant
    CircuitBreakerTimeout(5 * time.Minute). // Stay open longer
    Build()
```

### Testing Strategies

#### Unit Testing with Mock Client

```go
func TestPaymentService(t *testing.T) {
    // Create a test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch r.URL.Path {
        case "/charges":
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(map[string]string{"id": "ch_123"})
        default:
            w.WriteHeader(http.StatusNotFound)
        }
    }))
    defer server.Close()

    // Create client pointing to test server
    config := integration.NewConfigBuilder("stripe-test").
        BaseURL(server.URL).
        Timeout(5 * time.Second).
        MaxRetries(1).
        Build()

    client, _ := rest.New(config)

    // Test your code
    resp, err := client.Post(ctx, "/charges", chargeRequest)
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
}
```

#### Testing Circuit Breaker Behavior

```go
func TestCircuitBreaker(t *testing.T) {
    failCount := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        failCount++
        if failCount <= 5 {
            w.WriteHeader(http.StatusServiceUnavailable)
            return
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    config := integration.NewConfigBuilder("test").
        BaseURL(server.URL).
        CircuitBreakerThreshold(3).
        CircuitBreakerTimeout(100 * time.Millisecond).
        Build()

    client, _ := rest.New(config)

    // First 3 requests fail
    for i := 0; i < 3; i++ {
        _, err := client.Get(ctx, "/")
        assert.Error(t, err)
    }

    // Circuit should be open now
    _, err := client.Get(ctx, "/")
    assert.ErrorIs(t, err, integration.ErrCircuitOpen)

    // Wait for timeout
    time.Sleep(150 * time.Millisecond)

    // Should work now (half-open -> closed)
    _, err = client.Get(ctx, "/")
    assert.NoError(t, err)
}
```

#### Testing Retry Behavior

```go
func TestRetryOnTransientError(t *testing.T) {
    attempts := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        attempts++
        if attempts < 3 {
            w.WriteHeader(http.StatusServiceUnavailable)
            return
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    config := integration.NewConfigBuilder("test").
        BaseURL(server.URL).
        MaxRetries(5).
        RetryDelay(10 * time.Millisecond).
        Build()

    client, _ := rest.New(config)

    resp, err := client.Get(ctx, "/")

    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
    assert.Equal(t, 3, attempts) // Should have taken 3 attempts
}
```

### Environment-Based Configuration

```go
// Load from environment with prefix
config := integration.ConfigFromEnv("STRIPE")

// Environment variables:
// STRIPE_BASE_URL=https://api.stripe.com/v1
// STRIPE_TIMEOUT=30s
// STRIPE_CONNECT_TIMEOUT=5s
// STRIPE_MAX_RETRIES=3
// STRIPE_RETRY_DELAY=100ms
// STRIPE_CIRCUIT_THRESHOLD=5
// STRIPE_CIRCUIT_TIMEOUT=60s
```

### Logging and Debugging

Enable logging to troubleshoot integration issues:

```go
config := integration.NewConfigBuilder("api").
    BaseURL(baseURL).
    EnableLogging(true).
    RedactFields("password", "api_key", "token", "secret", "authorization").
    Build()
```

Log output includes:
- Request method, URL, headers (redacted)
- Response status, duration
- Retry attempts and delays
- Circuit breaker state changes

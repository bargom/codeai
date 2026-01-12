# Integration Patterns Example

Demonstrates various integration patterns including REST API integration with retry, GraphQL API integration, webhook delivery, circuit breaker, and timeout configurations.

## Overview

This example demonstrates:
- REST API integration with retry and circuit breaker
- GraphQL API integration
- Webhook delivery with reliability
- Caching and fallback strategies
- Rate limiting and batching
- Multi-channel notifications
- Storage integration

## File Structure

```
04-integrations/
├── integrations.cai    # Main DSL file
├── README.md              # This file
└── test.sh                # Sample curl commands for testing
```

## Integration Patterns

### Pattern 1: Circuit Breaker

```
Normal Operation (Circuit Closed)
─────────────────────────────────────────────────────────────
Request → Integration → Success → Response
           ↓
        Failure Count: 0

After Multiple Failures (Circuit Open)
─────────────────────────────────────────────────────────────
Request → Circuit Open → ErrCircuitOpen (immediate fail)
           ↓
        Failures >= Threshold

Recovery Attempt (Circuit Half-Open)
─────────────────────────────────────────────────────────────
                  (after timeout)
Request → Limited Probe → Success → Circuit Closed
                   ↓
                Failure → Circuit Re-opens
```

### Pattern 2: Retry with Exponential Backoff

```
Attempt 1: Immediate
     ↓ (fail)
Attempt 2: Wait 100ms × 2^0 = 100ms
     ↓ (fail)
Attempt 3: Wait 100ms × 2^1 = 200ms
     ↓ (fail)
Attempt 4: Wait 100ms × 2^2 = 400ms
     ↓ (fail)
Give up → Error

With jitter: ±25% randomization prevents thundering herd
```

### Pattern 3: Webhook Delivery with Dead Letter Queue

```
Event → Queue → Delivery Attempt
                      ↓
              ┌───────┴───────┐
              ↓               ↓
           Success         Failure
              ↓               ↓
          Mark Done     Retry (up to 5x)
                              ↓
                        Still Failing?
                              ↓
                    Move to Dead Letter Queue
                              ↓
                    Notify Ops Team
```

## Integrations

### 1. GitHub REST API

A comprehensive REST integration with:
- Bearer token authentication
- Custom headers (API version)
- Circuit breaker (5 failures in 1 minute)
- Rate limiting (5000 requests/hour)
- Response caching

**Operations:**
| Operation | Description | Caching |
|-----------|-------------|---------|
| `get_user` | Get user by username | 5 minutes |
| `list_repos` | List authenticated user's repos | No |
| `create_issue` | Create issue in repo | No |
| `get_rate_limit` | Check rate limit status | No |

**Configuration:**
```codeai
integration GitHubAPI {
    type: rest
    base_url: "https://api.github.com"
    auth: bearer(env(GITHUB_TOKEN))
    timeout: 30s
    retry: 3 times with exponential_backoff
    circuit_breaker: {
        threshold: 5 failures in 1m
        reset_after: 30s
    }
    rate_limit: {
        requests: 5000
        window: 1h
    }
}
```

### 2. Weather API

Demonstrates caching and fallback:
- API key authentication (query param)
- Aggressive caching (weather data)
- **Fallback to cache when circuit is open**

**Configuration:**
```codeai
integration WeatherAPI {
    type: rest
    auth: api_key(env(OPENWEATHER_API_KEY), "appid")
    default_cache_ttl: 15m
    fallback: cache    // Return cached data when unavailable
}
```

### 3. Stripe Payment API

Payment processing with idempotency:
- Auto-generated idempotency keys
- Prevents duplicate charges

**Configuration:**
```codeai
integration StripeAPI {
    type: rest
    idempotency: {
        header: "Idempotency-Key"
        generate: true
    }
}
```

### 4. GitHub GraphQL API

GraphQL integration example:

```codeai
integration GitHubGraphQL {
    type: graphql
    base_url: "https://api.github.com/graphql"

    operation get_user_repos {
        query: """
            query GetUserRepos($login: String!, $first: Int!) {
                user(login: $login) {
                    repositories(first: $first) {
                        nodes { name, stargazerCount }
                    }
                }
            }
        """
        variables: {
            login: string, required
            first: integer, default(10)
        }
    }
}
```

### 5. Webhook Delivery

Reliable webhook delivery:
- HMAC signature for security
- Exponential backoff retry: 1m, 5m, 30m, 2h, 8h
- Dead letter queue for failed deliveries
- Per-endpoint circuit breaker

```codeai
integration WebhookDeliveryService {
    type: webhook
    signature: {
        algorithm: "sha256"
        header: "X-Webhook-Signature"
    }
    retry: 5 times with exponential_backoff
    retry_delays: [1m, 5m, 30m, 2h, 8h]
    dead_letter: {
        enabled: true
        retention: 7d
    }
}
```

### 6. Notification Service

Multi-channel notifications with batching:

```codeai
integration NotificationService {
    batching: {
        enabled: true
        max_batch_size: 100
        max_delay: 5s
    }
}
```

### 7. S3-Compatible Storage

Object storage with AWS Signature V4:

```codeai
integration StorageService {
    type: rest
    auth: aws_signature_v4 {
        access_key: env(AWS_ACCESS_KEY_ID)
        secret_key: env(AWS_SECRET_ACCESS_KEY)
        region: env(AWS_REGION)
        service: "s3"
    }
}
```

## API Endpoints

### Webhook Management

| Method | Path | Description |
|--------|------|-------------|
| GET | `/webhooks` | List webhook endpoints |
| POST | `/webhooks` | Create webhook endpoint |
| PUT | `/webhooks/{id}` | Update webhook endpoint |
| DELETE | `/webhooks/{id}` | Delete webhook endpoint |
| GET | `/webhooks/{id}/deliveries` | List delivery attempts |
| POST | `/webhooks/{id}/test` | Send test event |
| POST | `/webhooks/deliveries/{id}/retry` | Retry failed delivery |

### Integration Status

| Method | Path | Description |
|--------|------|-------------|
| GET | `/integrations/status` | Status of all integrations |
| GET | `/integrations/{name}/health` | Health of specific integration |
| POST | `/integrations/{name}/circuit/reset` | Reset circuit breaker (admin) |

## Step-by-Step Instructions

### 1. Generate the API

```bash
codeai generate examples/04-integrations/integrations.cai
```

### 2. Configure Environment

```bash
# Database
export DATABASE_URL="postgres://localhost:5432/integrations"
export REDIS_URL="redis://localhost:6379"

# GitHub
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"

# Weather
export OPENWEATHER_API_KEY="your-api-key"

# Stripe
export STRIPE_API_URL="https://api.stripe.com/v1"
export STRIPE_SECRET_KEY="sk_test_..."

# Notifications
export NOTIFICATION_SERVICE_URL="https://api.sendgrid.com/v3"
export NOTIFICATION_API_KEY="your-key"

# Storage
export S3_ENDPOINT_URL="https://s3.amazonaws.com"
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"
export AWS_REGION="us-east-1"
```

### 3. Run Migrations

```bash
codeai migrate up
```

### 4. Start the Server

```bash
codeai run
```

## Sample Requests

### Create Webhook Endpoint

```bash
curl -X POST http://localhost:8080/webhooks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "name": "Order Notifications",
    "url": "https://example.com/webhooks/orders",
    "events": ["order.created", "order.shipped", "order.delivered"]
  }'
```

**Response:**
```json
{
  "id": "webhook-uuid",
  "name": "Order Notifications",
  "url": "https://example.com/webhooks/orders",
  "secret": "whsec_auto_generated_secret",
  "events": ["order.created", "order.shipped", "order.delivered"],
  "is_active": true,
  "is_healthy": true
}
```

### Test Webhook

```bash
curl -X POST http://localhost:8080/webhooks/<id>/test \
  -H "Authorization: Bearer <token>"
```

**Response:**
```json
{
  "id": "delivery-uuid",
  "event_type": "webhook.test",
  "status": "delivered",
  "response_status_code": 200,
  "duration_ms": 234
}
```

### Check Integration Status

```bash
curl http://localhost:8080/integrations/status \
  -H "Authorization: Bearer <token>"
```

**Response:**
```json
{
  "integrations": [
    {
      "name": "GitHubAPI",
      "type": "rest",
      "status": "healthy",
      "circuit_state": "closed",
      "error_rate": 0.02
    },
    {
      "name": "StripeAPI",
      "type": "rest",
      "status": "healthy",
      "circuit_state": "closed",
      "error_rate": 0.00
    },
    {
      "name": "WeatherAPI",
      "type": "rest",
      "status": "degraded",
      "circuit_state": "half-open",
      "error_rate": 0.15
    }
  ]
}
```

### Check Specific Integration Health

```bash
curl http://localhost:8080/integrations/GitHubAPI/health \
  -H "Authorization: Bearer <token>"
```

**Response:**
```json
{
  "name": "GitHubAPI",
  "status": "healthy",
  "circuit_state": "closed",
  "latency_p50": 120,
  "latency_p99": 450,
  "success_rate": 0.98,
  "last_error": null
}
```

### Reset Circuit Breaker (Admin)

```bash
curl -X POST http://localhost:8080/integrations/WeatherAPI/circuit/reset \
  -H "Authorization: Bearer <admin-token>"
```

### Retry Failed Webhook Delivery

```bash
curl -X POST http://localhost:8080/webhooks/deliveries/<delivery-id>/retry \
  -H "Authorization: Bearer <token>"
```

## Key Concepts Demonstrated

### 1. Circuit Breaker Configuration

```codeai
circuit_breaker: {
    threshold: 5 failures in 1m    // Open after 5 failures in 1 minute
    reset_after: 30s               // Try again after 30 seconds
}
```

### 2. Retry Configuration

```codeai
retry: 3 times with exponential_backoff
// Or with custom delays:
retry_delays: [1m, 5m, 30m, 2h, 8h]
```

### 3. Rate Limiting

```codeai
rate_limit: {
    requests: 5000
    window: 1h
    header: "X-RateLimit-Remaining"  // Read from response
}
```

### 4. Caching

```codeai
// Default TTL for all operations
default_cache_ttl: 15m

// Per-operation caching
operation get_current {
    cache: 10m
}
```

### 5. Fallback Strategies

```codeai
// Return cached data when circuit is open
fallback: cache

// Or use a default value
fallback: default({
    status: "unavailable"
})
```

### 6. Webhook Signatures

```codeai
signature: {
    algorithm: "sha256"
    header: "X-Webhook-Signature"
    secret_from: endpoint.secret
}
```

### 7. Request Batching

```codeai
batching: {
    enabled: true
    max_batch_size: 100
    max_delay: 5s
}
```

### 8. Idempotency

```codeai
idempotency: {
    header: "Idempotency-Key"
    generate: true
}
```

## Events

| Event | Trigger | Action |
|-------|---------|--------|
| `WebhookDeliveryFailed` | Delivery exhausted all retries | Notify Slack |
| `IntegrationCircuitOpened` | Circuit breaker opened | Notify Slack + webhook |
| `IntegrationCircuitClosed` | Circuit recovered | Log |

## Best Practices

1. **Always use circuit breakers** for external dependencies
2. **Cache aggressively** for data that doesn't change frequently
3. **Use idempotency keys** for payment and mutation operations
4. **Implement dead letter queues** for reliable event delivery
5. **Monitor circuit breaker states** for early warning
6. **Set appropriate timeouts** based on operation type
7. **Use rate limiting** to respect external API limits

## Next Steps

- See [Scheduled Jobs](../05-scheduled-jobs/) for background task patterns
- Check [E-commerce](../03-ecommerce/) for workflow integration

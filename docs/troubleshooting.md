# CodeAI Troubleshooting Guide

This guide helps you diagnose and resolve common issues with CodeAI.

## Table of Contents

- [Common Issues](#common-issues)
  - [Database Connection Failures](#database-connection-failures)
  - [JWT Token Validation Errors](#jwt-token-validation-errors)
  - [Workflow Execution Failures](#workflow-execution-failures)
  - [Cache (Redis) Issues](#cache-redis-issues)
  - [Integration Circuit Breaker Trips](#integration-circuit-breaker-trips)
- [Debugging Techniques](#debugging-techniques)
  - [Enabling Debug Logging](#enabling-debug-logging)
  - [Using pprof for Profiling](#using-pprof-for-profiling)
  - [Database Query Analysis](#database-query-analysis)
  - [Request Tracing](#request-tracing)
  - [Memory Profiling](#memory-profiling)
- [Error Messages Reference](#error-messages-reference)
- [Performance Issues](#performance-issues)
- [Recovery Procedures](#recovery-procedures)
- [Diagnostic Commands](#diagnostic-commands)
- [Support Information](#support-information)

---

## Common Issues

### Database Connection Failures

**Symptoms:**
- Health check returns `unhealthy` for database
- API requests return 503 Service Unavailable
- Logs show `database ping failed` errors

**Q: Why am I seeing "database ping failed" errors?**

A: The database health checker (`internal/health/checks/database.go:65`) performs a ping with a 2-second timeout. Common causes:

1. **Database not running**: Verify PostgreSQL is running
2. **Connection pool exhausted**: Check pool stats
3. **Network issues**: Verify connectivity to database host

**Diagnosis Steps:**
```bash
# Check if PostgreSQL is running
pg_isready -h localhost -p 5432

# Test connection directly
psql -h <host> -p <port> -U <user> -d <database> -c "SELECT 1"

# Check connection pool via health endpoint
curl -s http://localhost:8080/health | jq '.checks.database'
```

**Resolution:**
1. If database is down, restart the PostgreSQL service
2. If pool exhausted, consider increasing `MaxOpenConns` (default: 25)
3. If network issue, check firewall rules and DNS resolution

**Q: How do I check database connection pool statistics?**

A: The health check returns pool details in the response:
```json
{
  "status": "healthy",
  "details": {
    "max_connections": 25,
    "open_connections": 10,
    "in_use": 5,
    "idle": 5
  }
}
```

If `in_use` consistently equals `max_connections`, increase the pool size.

---

### JWT Token Validation Errors

**Symptoms:**
- API returns 401 Unauthorized
- Logs show authentication errors
- JWKS fetch failures

**Q: What does "invalid token" mean?**

A: The token is malformed or has an invalid signature. See `internal/auth/errors.go` for all auth errors.

| Error | Meaning | Resolution |
|-------|---------|------------|
| `ErrInvalidToken` | Token malformed or bad signature | Obtain a new token |
| `ErrExpiredToken` | Token has expired | Refresh or obtain new token |
| `ErrInvalidIssuer` | Token issuer doesn't match | Check auth provider configuration |
| `ErrInvalidAudience` | Token audience mismatch | Verify audience claim in token |
| `ErrMissingToken` | No token provided | Include `Authorization: Bearer <token>` header |
| `ErrKeyNotFound` | Signing key not found in JWKS | Check JWKS endpoint availability |
| `ErrUnsupportedAlgorithm` | Token uses unsupported algorithm | Use HS256 or RS256 |
| `ErrNoSecretConfigured` | HS256 requested but no secret | Set `JWT_SECRET` environment variable |
| `ErrJWKSFetchFailed` | Cannot fetch JWKS | Verify JWKS endpoint URL and network access |

**Diagnosis Steps:**
```bash
# Decode JWT token (without verification) to inspect claims
echo "<token>" | cut -d. -f2 | base64 -d 2>/dev/null | jq .

# Check JWKS endpoint
curl -s https://your-auth-provider/.well-known/jwks.json | jq .

# Verify token expiration
date -r $(echo "<token>" | cut -d. -f2 | base64 -d 2>/dev/null | jq -r .exp)
```

**Resolution:**
1. For expired tokens: Implement token refresh logic or obtain new token
2. For JWKS failures: Verify network access to auth provider
3. For algorithm issues: Ensure your auth provider uses supported algorithms

---

### Workflow Execution Failures

**Symptoms:**
- Workflow status shows `failed`
- Validation errors in workflow definition
- Execution errors during runtime

**Q: What validation errors might I see?**

A: The validator (`internal/validator/errors.go`) categorizes errors:

| Error Type | Examples | Resolution |
|------------|----------|------------|
| `ScopeError` | `undefined variable 'x'`, `duplicate declaration 'y'` | Check variable names and scopes |
| `TypeError` | `cannot iterate over non-array type 'string'` | Fix type mismatches in expressions |
| `FunctionError` | `undefined function 'foo'`, `wrong number of arguments` | Use correct function names and argument counts |

**Diagnosis Steps:**
```bash
# Validate workflow file
codeai validate workflow.cai

# Parse and inspect AST
codeai parse workflow.cai --output json | jq .
```

**Q: My workflow fails at runtime with query errors. What do they mean?**

A: Query errors (`internal/query/errors.go`) indicate issues with the query DSL:

| Error Type | Meaning |
|------------|---------|
| `LexerError` | Invalid token in query |
| `ParserError` | Syntax error during parsing |
| `SemanticError` | Unknown entity or field |
| `CompilerError` | SQL compilation failed |
| `ExecutionError` | Runtime query failure |

**Resolution:**
1. Check query syntax matches the DSL specification
2. Verify entity and field names exist in schema
3. Ensure proper quoting of string values

---

### Cache (Redis) Issues

**Symptoms:**
- Cache health check returns `unhealthy`
- Slow response times
- Fallback to database for cached data

**Q: How do I diagnose Redis connection issues?**

A: The cache health checker (`internal/health/checks/cache.go`) pings Redis with a 1-second timeout.

**Diagnosis Steps:**
```bash
# Test Redis connectivity
redis-cli -h <host> -p <port> ping

# Check Redis memory usage
redis-cli info memory

# Check Redis clients
redis-cli client list
```

**Resolution:**
1. Restart Redis if not responding
2. Check Redis memory limits if OOM
3. Verify network connectivity and firewall rules

---

### Integration Circuit Breaker Trips

**Symptoms:**
- External API calls failing with `ErrCircuitOpen`
- Circuit breaker logs show state transitions
- Integration health degraded

**Q: What do the circuit breaker states mean?**

A: The circuit breaker (`pkg/integration/circuitbreaker.go`) has three states:

| State | Meaning | Behavior |
|-------|---------|----------|
| `closed` | Normal operation | Requests pass through |
| `open` | Too many failures (default: 5) | All requests blocked |
| `half-open` | Testing recovery | Limited requests allowed |

**Q: How do I know when a circuit breaker opens?**

A: Watch for log entries:
```
circuit breaker state changed from=closed to=open
```

**Q: How do I manually reset a circuit breaker?**

A: Call the reset method programmatically or wait for the timeout (default: 60s):
```go
circuitBreaker.Reset()
```

**Configuration Options:**
| Parameter | Default | Description |
|-----------|---------|-------------|
| `FailureThreshold` | 5 | Failures before opening |
| `Timeout` | 60s | Time before half-open transition |
| `HalfOpenRequests` | 3 | Successes needed to close |

---

## Debugging Techniques

### Enabling Debug Logging

**Q: How do I enable debug logging?**

A: Set the `LOG_LEVEL` environment variable:

```bash
LOG_LEVEL=debug codeai server

# Or set in your environment
export LOG_LEVEL=debug
export LOG_FORMAT=json  # or 'text' for human-readable
export LOG_OUTPUT=stdout  # or 'stderr' or file path
export LOG_ADD_SOURCE=true  # adds file:line to logs
```

**Available Log Levels:**
- `debug` - Verbose debugging information
- `info` - Standard operational logs (default)
- `warn` - Warning conditions
- `error` - Error conditions only

### Using pprof for Profiling

**Q: How do I profile CPU usage?**

A: CodeAI exposes pprof endpoints:

```bash
# CPU profile (30 seconds)
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

# View in browser
go tool pprof -http=:8081 http://localhost:8080/debug/pprof/profile?seconds=30
```

**Q: How do I analyze memory usage?**

A: Use the heap profile:

```bash
# Heap profile
go tool pprof http://localhost:8080/debug/pprof/heap

# In pprof, useful commands:
# top 20 - show top memory consumers
# list <function> - show source code
# web - generate SVG visualization
```

### Database Query Analysis

**Q: How do I find slow queries?**

A: Use PostgreSQL's `EXPLAIN ANALYZE`:

```sql
-- Analyze a specific query
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT * FROM workflows WHERE status = 'active';

-- Check for sequential scans
SELECT schemaname, relname, seq_scan, seq_tup_read
FROM pg_stat_user_tables
WHERE seq_scan > 0
ORDER BY seq_tup_read DESC;

-- Find missing indexes
SELECT schemaname, tablename, attname, null_frac, n_distinct
FROM pg_stats
WHERE schemaname = 'public'
ORDER BY null_frac DESC;
```

**Q: What queries are currently running?**

```sql
SELECT pid, age(clock_timestamp(), query_start), usename, query
FROM pg_stat_activity
WHERE state != 'idle' AND query NOT ILIKE '%pg_stat_activity%'
ORDER BY query_start;
```

### Request Tracing

**Q: How do I trace a request through the system?**

A: CodeAI uses structured logging with request IDs. Enable tracing:

```bash
# Set trace ID header in requests
curl -H "X-Request-ID: trace-123" http://localhost:8080/api/v1/workflows

# Search logs for the trace ID
grep "trace-123" /var/log/codeai/*.log
```

The logging middleware (`pkg/logging/middleware.go`) automatically adds:
- `request_id` - Unique request identifier
- `method` - HTTP method
- `path` - Request path
- `status` - Response status
- `duration` - Request duration
- `user_id` - Authenticated user (if available)

### Memory Profiling

**Q: How do I detect memory leaks?**

A: Compare heap profiles over time:

```bash
# Take initial profile
curl -s http://localhost:8080/debug/pprof/heap > heap1.pb.gz

# Wait for some time...

# Take another profile
curl -s http://localhost:8080/debug/pprof/heap > heap2.pb.gz

# Compare
go tool pprof -base heap1.pb.gz heap2.pb.gz
```

**Q: How do I detect goroutine leaks?**

A: Monitor goroutine count:

```bash
# Get goroutine profile
curl -s http://localhost:8080/debug/pprof/goroutine?debug=2

# Get goroutine count over time
watch -n 5 'curl -s http://localhost:8080/debug/pprof/goroutine?debug=0 | head -1'
```

---

## Error Messages Reference

### Authentication Errors (internal/auth/errors.go)

| Error Code | Message | HTTP Status | Resolution |
|------------|---------|-------------|------------|
| AUTH001 | `invalid token` | 401 | Token malformed or signature invalid |
| AUTH002 | `token has expired` | 401 | Refresh or obtain new token |
| AUTH003 | `invalid token issuer` | 401 | Check auth provider configuration |
| AUTH004 | `invalid token audience` | 401 | Verify audience claim |
| AUTH005 | `missing authentication token` | 401 | Include Authorization header |
| AUTH006 | `signing key not found` | 500 | JWKS key ID mismatch |
| AUTH007 | `unsupported signing algorithm` | 400 | Use HS256 or RS256 |
| AUTH008 | `no secret configured for symmetric algorithm` | 500 | Set JWT_SECRET env var |
| AUTH009 | `failed to fetch JWKS` | 500 | Check JWKS endpoint |

### Validation Errors (internal/validator/errors.go)

| Error Type | Pattern | Example |
|------------|---------|---------|
| `ScopeError` | `undefined variable '%s'` | `undefined variable 'count'` |
| `ScopeError` | `duplicate declaration '%s'` | `duplicate declaration 'i'` |
| `ScopeError` | `duplicate parameter '%s'` | `duplicate parameter 'name'` |
| `TypeError` | `cannot iterate over non-array type '%s'` | `cannot iterate over non-array type 'int'` |
| `FunctionError` | `undefined function '%s'` | `undefined function 'process'` |
| `FunctionError` | `wrong number of arguments for '%s': expected %d, got %d` | `wrong number of arguments for 'sum': expected 2, got 1` |
| `FunctionError` | `'%s' is not a function` | `'data' is not a function` |

### Query Errors (internal/query/errors.go)

| Error Type | Pattern | Example |
|------------|---------|---------|
| `LexerError` | Invalid token | `unexpected character '@'` |
| `ParserError` | `expected %s, got '%s'` | `expected identifier, got '123'` |
| `ParserError` | `unexpected end of input` | Query truncated |
| `SemanticError` | `unknown entity '%s'` | `unknown entity 'users'` |
| `CompilerError` | `unknown field '%s' on entity '%s'` | `unknown field 'email' on entity 'workflow'` |
| `CompilerError` | `type mismatch for field '%s'` | `type mismatch for field 'count': expected int, got string` |
| `ExecutionError` | Runtime failure | Database error during execution |

### Integration Errors (pkg/integration/circuitbreaker.go)

| Error | Message | Resolution |
|-------|---------|------------|
| `ErrCircuitOpen` | `circuit breaker is open` | Wait for timeout or fix underlying service |

---

## Performance Issues

### Slow Database Queries

**Q: How do I identify slow queries?**

A: Enable slow query logging and use EXPLAIN:

```sql
-- In PostgreSQL
ALTER SYSTEM SET log_min_duration_statement = '100ms';
SELECT pg_reload_conf();

-- Analyze specific query
EXPLAIN (ANALYZE, COSTS, BUFFERS)
SELECT * FROM executions WHERE workflow_id = 'abc' ORDER BY created_at DESC LIMIT 10;
```

**Common causes and fixes:**

| Issue | Indicator | Resolution |
|-------|-----------|------------|
| Missing index | Sequential scan on large table | Add appropriate index |
| N+1 queries | Many small queries | Use JOINs or batch loading |
| Large result sets | High row count | Add pagination |
| Lock contention | `waiting` in pg_stat_activity | Optimize transaction scope |

### High Latency Endpoints

**Q: How do I find slow endpoints?**

A: The logging middleware tracks request duration:

```bash
# Find slowest requests (assuming JSON logs)
cat /var/log/codeai/app.log | jq 'select(.duration > 1000) | {path, duration, method}'
```

**Common causes:**
1. Database query performance
2. External API calls
3. Large response serialization
4. Memory pressure

### Memory Consumption

**Q: Why is memory usage high?**

A: Check the heap profile:

```bash
go tool pprof -alloc_space http://localhost:8080/debug/pprof/heap

# In pprof:
top 20
```

**Common causes:**
1. Large response bodies kept in memory
2. Goroutine leaks
3. Cache size too large
4. String concatenation in loops

### Goroutine Leak Detection

**Q: How do I find goroutine leaks?**

A: Monitor goroutine count and inspect stacks:

```bash
# Get all goroutine stacks
curl -s http://localhost:8080/debug/pprof/goroutine?debug=2 > goroutines.txt

# Count by function
cat goroutines.txt | grep "^goroutine" | wc -l

# Look for suspicious patterns
grep -A 10 "chan receive" goroutines.txt | head -50
```

**Common leak patterns:**
1. Blocked channel operations without timeout
2. HTTP response bodies not closed
3. Context cancellation not propagated
4. Timer/Ticker not stopped

---

## Recovery Procedures

### Service Restart Procedures

**Q: How do I gracefully restart the service?**

A: Send SIGTERM for graceful shutdown:

```bash
# Find the process
pgrep -f codeai

# Graceful shutdown (allows in-flight requests to complete)
kill -SIGTERM <pid>

# If not responding after 30s, force kill
kill -SIGKILL <pid>
```

**Kubernetes:**
```bash
kubectl rollout restart deployment/codeai
kubectl rollout status deployment/codeai
```

### Database Recovery

**Q: What if the database is corrupted?**

A: Follow PostgreSQL recovery procedures:

```bash
# Check database integrity
pg_isready -h localhost -p 5432

# Check for corruption
psql -c "SELECT * FROM pg_catalog.pg_tables WHERE schemaname = 'public';"

# Restore from backup if needed
pg_restore -d codeai backup.dump
```

**Q: How do I recover from connection pool exhaustion?**

A:
1. Identify long-running queries and kill them
2. Restart the application to reset the pool
3. Investigate and fix the root cause

```sql
-- Find and terminate long queries
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE state = 'active'
  AND query_start < NOW() - INTERVAL '5 minutes';
```

### Workflow Compensation Triggers

**Q: How do I trigger compensation for failed workflows?**

A: Use the API to retry or compensate:

```bash
# Retry a failed execution
curl -X POST http://localhost:8080/api/v1/executions/<id>/retry

# Trigger compensation manually
curl -X POST http://localhost:8080/api/v1/executions/<id>/compensate
```

### Job Queue Cleanup

**Q: How do I clear stuck jobs from the queue?**

A: Access the queue management endpoints:

```bash
# List pending jobs
curl http://localhost:8080/api/v1/admin/jobs?status=pending

# Cancel a specific job
curl -X DELETE http://localhost:8080/api/v1/admin/jobs/<id>

# Purge failed jobs older than 24h
curl -X POST http://localhost:8080/api/v1/admin/jobs/purge?status=failed&older_than=24h
```

---

## Diagnostic Commands

### Health Check Endpoints

```bash
# Overall health (includes all checks)
curl -s http://localhost:8080/health | jq .

# Liveness probe (for Kubernetes)
curl -s http://localhost:8080/health/live

# Readiness probe (checks critical dependencies)
curl -s http://localhost:8080/health/ready

# Example healthy response:
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "uptime": "24h5m30s",
  "checks": {
    "database": {"status": "healthy", "duration": "1.5ms"},
    "cache": {"status": "healthy", "duration": "0.5ms"}
  }
}
```

### Metrics Inspection

```bash
# Prometheus metrics endpoint
curl -s http://localhost:8080/metrics

# Key metrics to watch:
# - http_requests_total
# - http_request_duration_seconds
# - db_connections_in_use
# - circuit_breaker_state
# - workflow_executions_total
```

### Log Aggregation Queries

**Loki/Grafana queries:**
```logql
# All errors in the last hour
{app="codeai"} |= "error" | json | level = "error"

# Slow requests (>1s)
{app="codeai"} | json | duration > 1000

# Authentication failures
{app="codeai"} |= "authentication" |= "failed"

# Circuit breaker state changes
{app="codeai"} |= "circuit breaker state changed"
```

### Database Connection Status

```bash
# Check database from application
curl -s http://localhost:8080/health | jq '.checks.database'

# Direct PostgreSQL check
psql -h <host> -U <user> -d <database> -c "
  SELECT count(*) as connections,
         state
  FROM pg_stat_activity
  GROUP BY state;
"

# Connection pool stats
curl -s http://localhost:8080/health | jq '.checks.database.details'
```

---

## Support Information

### What to Include in Bug Reports

When reporting issues, include:

1. **Environment Information:**
   - CodeAI version: `codeai version`
   - Go version: `go version`
   - OS and architecture: `uname -a`
   - Deployment type (Docker, Kubernetes, bare metal)

2. **Error Details:**
   - Full error message
   - Stack trace if available
   - Request ID (from logs or response headers)

3. **Reproduction Steps:**
   - Minimal workflow file that triggers the issue
   - API request/response examples
   - Configuration (redact sensitive values)

4. **Diagnostic Data:**
   - Health check output: `curl http://localhost:8080/health`
   - Relevant log entries (last 100 lines before error)
   - Resource usage (CPU, memory)

### Collecting Diagnostic Data

```bash
#!/bin/bash
# diagnostic-bundle.sh

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BUNDLE_DIR="codeai-diagnostics-$TIMESTAMP"
mkdir -p "$BUNDLE_DIR"

# Version info
codeai version > "$BUNDLE_DIR/version.txt"

# Health check
curl -s http://localhost:8080/health > "$BUNDLE_DIR/health.json"

# Metrics snapshot
curl -s http://localhost:8080/metrics > "$BUNDLE_DIR/metrics.txt"

# Goroutine dump
curl -s http://localhost:8080/debug/pprof/goroutine?debug=2 > "$BUNDLE_DIR/goroutines.txt"

# Heap profile
curl -s http://localhost:8080/debug/pprof/heap > "$BUNDLE_DIR/heap.pb.gz"

# Recent logs (last 1000 lines)
tail -1000 /var/log/codeai/app.log > "$BUNDLE_DIR/recent-logs.txt" 2>/dev/null || echo "Log file not found"

# System info
echo "OS: $(uname -a)" > "$BUNDLE_DIR/system-info.txt"
echo "Memory: $(free -h 2>/dev/null || vm_stat)" >> "$BUNDLE_DIR/system-info.txt"

# Create tarball
tar -czf "$BUNDLE_DIR.tar.gz" "$BUNDLE_DIR"
rm -rf "$BUNDLE_DIR"

echo "Diagnostic bundle created: $BUNDLE_DIR.tar.gz"
```

### Log Levels for Troubleshooting

| Scenario | Recommended Level | Environment Variables |
|----------|-------------------|----------------------|
| Normal production | `info` | `LOG_LEVEL=info` |
| Investigating issues | `debug` | `LOG_LEVEL=debug` |
| Performance analysis | `info` with source | `LOG_LEVEL=info LOG_ADD_SOURCE=true` |
| Minimal logging | `error` | `LOG_LEVEL=error` |

**Temporary debug logging:**
```bash
# Enable debug for a single request
curl -H "X-Debug: true" http://localhost:8080/api/v1/workflows
```

---

## Quick Reference

### Common Diagnostic Commands

```bash
# Health check
curl -s http://localhost:8080/health | jq .

# Check specific component
curl -s http://localhost:8080/health | jq '.checks.<component>'

# View error logs
tail -f /var/log/codeai/app.log | grep -i error

# CPU profile
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

# Memory profile
go tool pprof http://localhost:8080/debug/pprof/heap

# Goroutine dump
curl http://localhost:8080/debug/pprof/goroutine?debug=2

# Validate workflow
codeai validate workflow.cai
```

### Emergency Contacts

For critical production issues:
1. Check the [GitHub Issues](https://github.com/bargom/codeai/issues) for known issues
2. Review this troubleshooting guide
3. Collect diagnostic bundle before escalating
4. Contact the on-call team with the diagnostic bundle

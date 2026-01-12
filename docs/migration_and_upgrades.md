# CodeAI Migration and Upgrade Guide

This document provides comprehensive guidance for upgrading CodeAI versions, managing database migrations, and handling configuration changes.

## Table of Contents

1. [Version Upgrade Process](#1-version-upgrade-process)
2. [Database Migrations](#2-database-migrations)
3. [Configuration Changes](#3-configuration-changes)
4. [API Versioning](#4-api-versioning)
5. [DSL Syntax Evolution](#5-dsl-syntax-evolution)
6. [Zero-Downtime Deployment](#6-zero-downtime-deployment)
7. [Troubleshooting](#7-troubleshooting)

---

## 1. Version Upgrade Process

### 1.1 Semantic Versioning Strategy

CodeAI follows [Semantic Versioning 2.0.0](https://semver.org/) with the format `MAJOR.MINOR.PATCH`:

| Component | When Incremented | Examples |
|-----------|------------------|----------|
| **MAJOR** | Breaking changes to API, DSL, or database schema | `1.0.0` -> `2.0.0` |
| **MINOR** | New features, backward-compatible additions | `1.0.0` -> `1.1.0` |
| **PATCH** | Bug fixes, security patches, documentation | `1.0.0` -> `1.0.1` |

**Pre-release versions**: `1.0.0-alpha`, `1.0.0-beta.1`, `1.0.0-rc.1`

### 1.2 Determining Breaking vs Non-Breaking Changes

**Breaking Changes (MAJOR version bump)**:
- Removal of API endpoints or fields
- Changes to database schema requiring data migration
- DSL syntax changes that invalidate existing files
- Configuration key renames or removals
- Changes to default behavior

**Non-Breaking Changes (MINOR version bump)**:
- New API endpoints
- New optional fields in requests/responses
- New DSL features with backward-compatible syntax
- New configuration options with sensible defaults
- Performance improvements

### 1.3 Upgrade Checklist

Before upgrading, complete the following checklist:

```bash
# 1. Review release notes
# Check CHANGELOG.md or GitHub releases for breaking changes

# 2. Backup current state
pg_dump codeai > backup_$(date +%Y%m%d_%H%M%S).sql
cp -r /etc/codeai /etc/codeai.backup

# 3. Check current version
./bin/codeai version

# 4. Review migration requirements
# Check if database migrations are needed

# 5. Test in staging environment first
# Always test upgrades in non-production first

# 6. Plan maintenance window (for major upgrades)
# Notify users of potential downtime
```

### 1.4 Standard Upgrade Procedure

**For PATCH upgrades (e.g., 1.0.0 -> 1.0.1)**:

```bash
# 1. Stop the service
systemctl stop codeai

# 2. Replace the binary
cp bin/codeai-new bin/codeai

# 3. Verify the version
./bin/codeai version

# 4. Start the service
systemctl start codeai

# 5. Verify health
curl http://localhost:8080/health
```

**For MINOR upgrades (e.g., 1.0.0 -> 1.1.0)**:

```bash
# 1. Review new features and configuration options
# Check CHANGELOG.md for new features

# 2. Stop the service
systemctl stop codeai

# 3. Run database migrations (if any)
./bin/codeai migrate up

# 4. Update configuration (if new options are desired)
# Review config/development.yaml for new options

# 5. Replace the binary
cp bin/codeai-new bin/codeai

# 6. Start the service
systemctl start codeai

# 7. Verify functionality
curl http://localhost:8080/health
./bin/codeai validate test/fixtures/simple.cai
```

**For MAJOR upgrades (e.g., 1.x.x -> 2.0.0)**:

```bash
# 1. Full backup
pg_dump codeai > backup_pre_v2.sql
tar -czf codeai_backup.tar.gz /etc/codeai /var/lib/codeai

# 2. Review migration guide for this specific version
# Each major version has a dedicated migration guide

# 3. Stop the service
systemctl stop codeai

# 4. Run pre-migration scripts (if provided)
./scripts/pre_migrate_v2.sh

# 5. Run database migrations
./bin/codeai migrate up

# 6. Update configuration
# Apply configuration changes from migration guide

# 7. Replace the binary
cp bin/codeai-new bin/codeai

# 8. Run post-migration scripts (if provided)
./scripts/post_migrate_v2.sh

# 9. Start the service
systemctl start codeai

# 10. Run validation tests
make test-integration
```

### 1.5 Rollback Procedures

**Immediate Rollback (within minutes)**:

```bash
# 1. Stop the service
systemctl stop codeai

# 2. Restore previous binary
cp bin/codeai.backup bin/codeai

# 3. Rollback database migration (if safe)
./bin/codeai migrate down

# 4. Restore configuration
cp -r /etc/codeai.backup/* /etc/codeai/

# 5. Start the service
systemctl start codeai
```

**Full Rollback (database restore)**:

```bash
# 1. Stop the service
systemctl stop codeai

# 2. Restore database from backup
psql -c "DROP DATABASE codeai;"
psql -c "CREATE DATABASE codeai;"
psql codeai < backup_pre_upgrade.sql

# 3. Restore binary
cp bin/codeai.backup bin/codeai

# 4. Restore configuration
cp -r /etc/codeai.backup/* /etc/codeai/

# 5. Start the service
systemctl start codeai
```

---

## 2. Database Migrations

### 2.1 Migration System Overview

CodeAI uses an embedded migration system based on SQL files. Migrations are:
- Stored in `internal/database/migrations/`
- Embedded in the binary using Go's `embed` directive
- Tracked in the `schema_migrations` table
- Executed in version order

### 2.2 Migration File Structure and Naming

Migrations follow a strict naming convention:

```
{VERSION}_{NAME}.{DIRECTION}.sql
```

| Component | Format | Example |
|-----------|--------|---------|
| VERSION | `YYYYMMDDHHMMSS` | `20260111120000` |
| NAME | snake_case description | `initial_schema` |
| DIRECTION | `up` or `down` | `up` |

**Example migration pair**:
```
internal/database/migrations/
├── 20260111120000_initial_schema.up.sql
├── 20260111120000_initial_schema.down.sql
├── 20260115093000_add_user_table.up.sql
└── 20260115093000_add_user_table.down.sql
```

### 2.3 Writing Migrations

**Up Migration (`*.up.sql`)**:

```sql
-- 20260115093000_add_user_table.up.sql
-- Add users table for authentication

-- Create the table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'viewer',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Add foreign key to deployments
ALTER TABLE deployments ADD COLUMN IF NOT EXISTS created_by TEXT REFERENCES users(id);
```

**Down Migration (`*.down.sql`)**:

```sql
-- 20260115093000_add_user_table.down.sql
-- Rollback: Remove users table

-- Remove foreign key first
ALTER TABLE deployments DROP COLUMN IF EXISTS created_by;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_email;

-- Drop table
DROP TABLE IF EXISTS users;
```

**Migration Best Practices**:

1. **Always create both up and down migrations**
2. **Make migrations idempotent** - Use `IF EXISTS`/`IF NOT EXISTS`
3. **Keep migrations atomic** - One logical change per migration
4. **Test rollbacks** - Verify down migrations work correctly
5. **Never modify existing migrations** - Create new ones instead
6. **Document complex changes** - Add comments explaining the change

### 2.4 Testing Migrations

**Testing in development**:

```bash
# Run pending migrations
./bin/codeai migrate up

# Check migration status
./bin/codeai migrate status

# Test rollback
./bin/codeai migrate down

# Re-apply
./bin/codeai migrate up

# Reset and re-run all migrations
./bin/codeai migrate reset
```

**Testing with a fresh database**:

```bash
# Create test database
createdb codeai_test

# Run all migrations
DATABASE_NAME=codeai_test ./bin/codeai migrate up

# Verify schema
psql codeai_test -c "\dt"

# Run integration tests
DATABASE_NAME=codeai_test make test-integration

# Cleanup
dropdb codeai_test
```

### 2.5 Running Migrations in Production

**Pre-production checklist**:

```bash
# 1. Backup database
pg_dump -Fc codeai > codeai_$(date +%Y%m%d).dump

# 2. Check pending migrations
./bin/codeai migrate status

# 3. Review migration SQL
cat internal/database/migrations/NEW_MIGRATION.up.sql

# 4. Test in staging
STAGING_DB_HOST=staging ./bin/codeai migrate up
```

**Production migration**:

```bash
# During maintenance window
./bin/codeai migrate up --verbose

# Verify success
./bin/codeai migrate status
```

**For long-running migrations**:

```bash
# Run migration with extended timeout
./bin/codeai migrate up --timeout 30m

# Or run SQL directly with progress monitoring
psql codeai -f internal/database/migrations/20260115093000_large_migration.up.sql
```

### 2.6 Migration Rollback

**Rollback last migration**:

```bash
./bin/codeai migrate down
```

**Rollback to specific version**:

```bash
# Rollback until reaching version 20260111120000
while [ "$(./bin/codeai migrate status | grep 'latest' | cut -d' ' -f2)" != "20260111120000" ]; do
    ./bin/codeai migrate down
done
```

**Full rollback (use with caution)**:

```bash
# This will rollback ALL migrations
./bin/codeai migrate reset
```

### 2.7 Data Integrity Verification

After running migrations, verify data integrity:

```bash
# Check table structure
psql codeai -c "\d configs"
psql codeai -c "\d deployments"
psql codeai -c "\d executions"

# Check foreign key constraints
psql codeai -c "
SELECT
    tc.table_name, kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu
    ON ccu.constraint_name = tc.constraint_name
WHERE constraint_type = 'FOREIGN KEY';
"

# Check index usage
psql codeai -c "
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;
"

# Verify record counts
psql codeai -c "
SELECT 'configs' as table_name, COUNT(*) as count FROM configs
UNION ALL
SELECT 'deployments', COUNT(*) FROM deployments
UNION ALL
SELECT 'executions', COUNT(*) FROM executions;
"
```

### 2.8 Example Migration Scripts

**Adding a new column with default value**:

```sql
-- up
ALTER TABLE deployments ADD COLUMN IF NOT EXISTS priority INTEGER DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_deployments_priority ON deployments(priority);

-- down
DROP INDEX IF EXISTS idx_deployments_priority;
ALTER TABLE deployments DROP COLUMN IF EXISTS priority;
```

**Renaming a column (two-phase)**:

```sql
-- Phase 1: Add new column, copy data
ALTER TABLE configs ADD COLUMN IF NOT EXISTS dsl_content TEXT;
UPDATE configs SET dsl_content = content WHERE dsl_content IS NULL;

-- Phase 2 (after application is updated): Drop old column
ALTER TABLE configs DROP COLUMN IF EXISTS content;
```

**Data transformation**:

```sql
-- up: Migrate status values from string to enum-like integers
ALTER TABLE deployments ADD COLUMN IF NOT EXISTS status_code INTEGER;
UPDATE deployments SET status_code = CASE status
    WHEN 'pending' THEN 0
    WHEN 'running' THEN 1
    WHEN 'stopped' THEN 2
    WHEN 'failed' THEN 3
    WHEN 'complete' THEN 4
    ELSE -1
END;

-- down: Restore string status
UPDATE deployments SET status = CASE status_code
    WHEN 0 THEN 'pending'
    WHEN 1 THEN 'running'
    WHEN 2 THEN 'stopped'
    WHEN 3 THEN 'failed'
    WHEN 4 THEN 'complete'
    ELSE 'unknown'
END;
ALTER TABLE deployments DROP COLUMN IF EXISTS status_code;
```

---

## 3. Configuration Changes

### 3.1 Configuration File Locations

| Location | Purpose | Priority |
|----------|---------|----------|
| `/etc/codeai/config.yaml` | System-wide configuration | Low |
| `$HOME/.cai.yaml` | User-specific configuration | Medium |
| `./config.yaml` | Project-specific configuration | High |
| Environment variables | Runtime overrides | Highest |

### 3.2 Deprecated Configuration Options

When configuration options are deprecated, they follow this lifecycle:

1. **Deprecation Notice**: Added to release notes and logs warning
2. **Grace Period**: Old option continues to work for 2 minor versions
3. **Removal**: Old option removed in next major version

**Handling deprecated options**:

```yaml
# Old format (deprecated in v1.2.0, removed in v2.0.0)
db_host: localhost
db_port: 5432

# New format
database:
  host: localhost
  port: 5432
```

### 3.3 Configuration Migration Script

When upgrading configuration between versions:

```bash
#!/bin/bash
# migrate_config.sh - Migrate configuration from v1 to v2

OLD_CONFIG=${1:-/etc/codeai/config.yaml}
NEW_CONFIG=${2:-/etc/codeai/config.yaml.new}

# Read old values
DB_HOST=$(grep 'db_host:' "$OLD_CONFIG" | awk '{print $2}')
DB_PORT=$(grep 'db_port:' "$OLD_CONFIG" | awk '{print $2}')
DB_NAME=$(grep 'db_name:' "$OLD_CONFIG" | awk '{print $2}')

# Generate new format
cat > "$NEW_CONFIG" << EOF
# CodeAI Configuration v2.0
# Migrated from $OLD_CONFIG

database:
  host: ${DB_HOST:-localhost}
  port: ${DB_PORT:-5432}
  name: ${DB_NAME:-codeai}
  pool_size: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

server:
  host: localhost
  port: 8080
  timeout: 60s

logging:
  level: info
  format: json
EOF

echo "Configuration migrated to $NEW_CONFIG"
echo "Review the new configuration and replace the old file:"
echo "  mv $NEW_CONFIG $OLD_CONFIG"
```

### 3.4 Environment Variable Changes

**Current environment variables**:

| Variable | Purpose | Default |
|----------|---------|---------|
| `CODEAI_DB_HOST` | Database host | `localhost` |
| `CODEAI_DB_PORT` | Database port | `5432` |
| `CODEAI_DB_NAME` | Database name | `codeai` |
| `CODEAI_DB_USER` | Database user | `postgres` |
| `CODEAI_DB_PASSWORD` | Database password | (empty) |
| `CODEAI_DB_SSLMODE` | SSL mode | `disable` |
| `CODEAI_SERVER_HOST` | Server bind address | `localhost` |
| `CODEAI_SERVER_PORT` | Server port | `8080` |
| `CODEAI_LOG_LEVEL` | Log level | `info` |
| `CODEAI_LOG_FORMAT` | Log format | `json` |
| `CODEAI_REDIS_URL` | Redis connection URL | (empty) |
| `CODEAI_JWT_SECRET` | JWT signing secret | (empty) |
| `CODEAI_JWKS_URL` | JWKS endpoint URL | (empty) |

**Version-specific changes** are documented in release notes.

---

## 4. API Versioning

### 4.1 API Version Strategy

CodeAI uses URL-based API versioning:

```
/api/v1/deployments
/api/v2/deployments
```

**Current API versions**:

| Version | Status | Supported Until |
|---------|--------|-----------------|
| v1 | Current | - |

### 4.2 Backward Compatibility Guarantees

For a given API version, CodeAI guarantees:

1. **Endpoint stability**: Existing endpoints will not be removed
2. **Field stability**: Existing response fields will not be removed
3. **Type stability**: Field types will not change
4. **Additive changes only**: New fields may be added (clients should ignore unknown fields)

### 4.3 Deprecation Timeline

When an API version is deprecated:

1. **Announcement**: 6 months before end-of-life
2. **Warning headers**: `Deprecation` header added to responses
3. **Documentation**: Migration guide published
4. **Sunset**: API version returns 410 Gone

**Checking for deprecation**:

```bash
curl -I http://localhost:8080/api/v1/deployments

# If deprecated, you'll see:
# Deprecation: true
# Sunset: Sat, 01 Jan 2027 00:00:00 GMT
# Link: </api/v2/deployments>; rel="successor-version"
```

### 4.4 Migrating Between API Versions

**Example: v1 to v2 migration**:

```bash
# v1 request
curl -X POST http://localhost:8080/api/v1/deployments \
  -H "Content-Type: application/json" \
  -d '{"name": "prod", "config_id": "abc123"}'

# v2 request (with additional fields)
curl -X POST http://localhost:8080/api/v2/deployments \
  -H "Content-Type: application/json" \
  -d '{
    "name": "prod",
    "config_id": "abc123",
    "labels": {"env": "production"},
    "priority": 10
  }'
```

---

## 5. DSL Syntax Evolution

### 5.1 DSL Version Compatibility

CodeAI DSL files can optionally specify a version directive:

```codeai
// Explicit version declaration
#version 1.0

var appName = "MyApp"
```

If no version is specified, the latest compatible version is assumed.

### 5.2 Syntax Changes by Version

| Version | Changes | Backward Compatible |
|---------|---------|---------------------|
| 1.0 | Initial release | - |

**Planned changes** are documented in the [DSL Language Specification](./dsl_language_spec.md).

### 5.3 Migrating DSL Files

**Automated migration tool**:

```bash
# Check DSL file compatibility
./bin/codeai validate myapp.cai

# Migrate to latest syntax (when available)
./bin/codeai migrate-dsl myapp.cai --output myapp_v2.cai

# Batch migration
find . -name "*.cai" -exec ./bin/codeai migrate-dsl {} \;
```

**Manual migration checklist**:

1. Add version directive if missing
2. Update deprecated keywords
3. Adjust syntax for new features
4. Run validation
5. Test execution

### 5.4 Validation and Linting

**Validating DSL files**:

```bash
# Single file
./bin/codeai validate myapp.cai

# With verbose output
./bin/codeai validate --verbose myapp.cai

# JSON output for CI/CD
./bin/codeai validate --output json myapp.cai
```

**CI/CD integration**:

```yaml
# .github/workflows/validate.yml
name: Validate DSL
on: [push, pull_request]
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - name: Build
        run: make build
      - name: Validate DSL files
        run: |
          for file in $(find . -name "*.cai"); do
            ./bin/codeai validate "$file"
          done
```

---

## 6. Zero-Downtime Deployment

### 6.1 Blue-Green Deployment

Blue-green deployment maintains two identical production environments:

```
                    ┌─────────────────┐
                    │  Load Balancer  │
                    └────────┬────────┘
                             │
              ┌──────────────┴──────────────┐
              ▼                             ▼
    ┌─────────────────┐           ┌─────────────────┐
    │   Blue (v1.0)   │           │  Green (v1.1)   │
    │    [Active]     │           │   [Standby]     │
    └─────────────────┘           └─────────────────┘
              │                             │
              └──────────────┬──────────────┘
                             ▼
                    ┌─────────────────┐
                    │    Database     │
                    │   (Shared)      │
                    └─────────────────┘
```

**Deployment steps**:

```bash
# 1. Deploy new version to Green environment
./deploy.sh green v1.1.0

# 2. Run health checks on Green
curl http://green.internal:8080/health

# 3. Run smoke tests
./smoke_tests.sh green

# 4. Switch traffic to Green
./switch_traffic.sh green

# 5. Monitor for issues
./monitor.sh --duration 10m

# 6. If issues, rollback to Blue
./switch_traffic.sh blue

# 7. If successful, Blue becomes new standby
./deploy.sh blue v1.1.0
```

### 6.2 Rolling Updates

Rolling updates gradually replace instances:

```bash
# Kubernetes rolling update
kubectl set image deployment/codeai \
  codeai=codeai:v1.1.0 \
  --record

# Monitor rollout
kubectl rollout status deployment/codeai

# Rollback if needed
kubectl rollout undo deployment/codeai
```

**Kubernetes deployment configuration**:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: codeai
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    spec:
      containers:
      - name: codeai
        image: codeai:v1.1.0
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 20
```

### 6.3 Database Migration Strategies

**For zero-downtime database migrations**:

1. **Expand-Contract Pattern**:
   - Phase 1 (Expand): Add new column/table, keep old
   - Deploy application that writes to both
   - Phase 2: Migrate data
   - Phase 3 (Contract): Remove old column/table

2. **Example: Renaming a column**:

```sql
-- Phase 1: Add new column
ALTER TABLE configs ADD COLUMN dsl_content TEXT;

-- Application writes to both columns (v1.1)

-- Phase 2: Copy data
UPDATE configs SET dsl_content = content WHERE dsl_content IS NULL;

-- Application reads from dsl_content (v1.2)

-- Phase 3: Drop old column
ALTER TABLE configs DROP COLUMN content;
```

### 6.4 Health Checks

**Liveness probe** (is the application running?):

```bash
curl http://localhost:8080/health
# {"status":"ok","timestamp":"2026-01-12T15:00:00Z"}
```

**Readiness probe** (is the application ready to serve traffic?):

```bash
curl http://localhost:8080/health/ready
# {"status":"ready","database":"connected","cache":"connected"}
```

**Implementing graceful shutdown**:

```go
// The server handles SIGTERM gracefully
// In-flight requests are completed before shutdown
// New requests receive 503 Service Unavailable
```

### 6.5 Feature Flags

For gradual rollout of new features:

```yaml
# config/features.yaml
features:
  new_parser:
    enabled: true
    rollout_percentage: 10
    allowed_users: ["user1", "user2"]

  async_execution:
    enabled: false
```

**Using feature flags in code**:

```go
if features.IsEnabled("new_parser", userID) {
    return newParser.Parse(input)
}
return legacyParser.Parse(input)
```

---

## 7. Troubleshooting

### 7.1 Common Migration Issues

**Issue: Migration fails with foreign key constraint error**

```
ERROR: cannot drop table "configs" because other objects depend on it
```

**Solution**: Drop dependent objects first or use CASCADE (with caution):

```sql
-- Check dependencies
SELECT
    tc.table_name, tc.constraint_name,
    kcu.column_name, ccu.table_name AS foreign_table
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
    ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage ccu
    ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
    AND ccu.table_name = 'configs';
```

**Issue: Migration stuck or taking too long**

```bash
# Check for locks
psql codeai -c "
SELECT pid, now() - pg_stat_activity.query_start AS duration, query, state
FROM pg_stat_activity
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes';
"

# Cancel long-running query (use with caution)
psql codeai -c "SELECT pg_cancel_backend(PID);"
```

**Issue: Out of disk space during migration**

```bash
# Check disk usage
df -h

# Check database size
psql codeai -c "SELECT pg_size_pretty(pg_database_size('codeai'));"

# Vacuum to reclaim space
psql codeai -c "VACUUM FULL;"
```

### 7.2 Rollback Verification

After a rollback, verify the system state:

```bash
# 1. Check service health
curl http://localhost:8080/health

# 2. Check application version
./bin/codeai version

# 3. Check database schema
psql codeai -c "\dt"

# 4. Check migration status
./bin/codeai migrate status

# 5. Run validation tests
make test-integration

# 6. Check logs for errors
journalctl -u codeai --since "1 hour ago" | grep -i error
```

### 7.3 Getting Help

If you encounter issues during migration:

1. Check the logs: `journalctl -u codeai -f`
2. Review the [Architecture Guide](./architecture.md)
3. Check the [Implementation Status](./implementation_status.md)
4. Open an issue on GitHub with:
   - Current version and target version
   - Error messages
   - Migration status output
   - Relevant log entries

---

## Appendix A: Migration Checklist Template

```markdown
# Migration Checklist: v{OLD} -> v{NEW}

## Pre-Migration
- [ ] Review release notes and breaking changes
- [ ] Backup database: `pg_dump codeai > backup.sql`
- [ ] Backup configuration: `cp -r /etc/codeai /etc/codeai.backup`
- [ ] Test migration in staging environment
- [ ] Notify stakeholders of maintenance window
- [ ] Prepare rollback plan

## Migration
- [ ] Stop application: `systemctl stop codeai`
- [ ] Run database migrations: `./bin/codeai migrate up`
- [ ] Update configuration files
- [ ] Replace application binary
- [ ] Start application: `systemctl start codeai`

## Post-Migration
- [ ] Verify health endpoint: `curl /health`
- [ ] Verify API functionality
- [ ] Check logs for errors
- [ ] Run integration tests
- [ ] Monitor for 24 hours
- [ ] Clean up backups (after validation period)

## Rollback (if needed)
- [ ] Stop application
- [ ] Restore database from backup
- [ ] Restore binary from backup
- [ ] Restore configuration from backup
- [ ] Start application
- [ ] Verify rollback successful
```

---

## Appendix B: Version History

| Version | Date | Notable Changes |
|---------|------|-----------------|
| 0.1.0 | 2026-01-11 | Initial release |

---

*This guide is maintained alongside the CodeAI source code. For the latest version, see the repository.*

# CodeAI Deployment Guide

This guide covers the complete deployment process for CodeAI, from building the binary to running in production with monitoring and scaling.

## Table of Contents

1. [Build Process](#build-process)
2. [Configuration Management](#configuration-management)
3. [Database Setup](#database-setup)
4. [Deployment Options](#deployment-options)
5. [Monitoring and Alerting](#monitoring-and-alerting)
6. [Scaling Considerations](#scaling-considerations)
7. [Security Checklist](#security-checklist)

---

## Build Process

### Compiling to Single Binary

CodeAI compiles to a single, statically-linked binary with no external dependencies.

```bash
# Basic build
make build

# Build with optimizations (strips debug info)
go build -ldflags "-s -w" -o bin/codeai ./cmd/codeai
```

### Build Flags and Optimization

Use ldflags to embed version information and optimize binary size:

```bash
# Full production build with version info
go build -ldflags "\
  -s -w \
  -X 'github.com/bargom/codeai/cmd/codeai/cmd.Version=1.0.0' \
  -X 'github.com/bargom/codeai/cmd/codeai/cmd.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)' \
  -X 'github.com/bargom/codeai/cmd/codeai/cmd.GitCommit=$(git rev-parse HEAD)'" \
  -o bin/codeai ./cmd/codeai
```

| Flag | Purpose |
|------|---------|
| `-s` | Omit symbol table |
| `-w` | Omit DWARF debugging information |
| `-X` | Set string variable at build time |

### Cross-Compilation

Build for different platforms without requiring a cross-compiler:

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o bin/codeai-linux-amd64 ./cmd/codeai

# Linux ARM64 (e.g., AWS Graviton, Raspberry Pi 4)
GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o bin/codeai-linux-arm64 ./cmd/codeai

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o bin/codeai-darwin-amd64 ./cmd/codeai

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o bin/codeai-darwin-arm64 ./cmd/codeai

# Windows
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o bin/codeai-windows-amd64.exe ./cmd/codeai
```

### Binary Size Optimization

Additional techniques to reduce binary size:

```bash
# Use UPX compression (optional, may affect startup time)
upx --best bin/codeai

# Build with CGO disabled for pure Go binary
CGO_ENABLED=0 go build -ldflags "-s -w" -o bin/codeai ./cmd/codeai
```

Expected binary sizes:
- **Unoptimized**: ~50MB
- **With `-s -w`**: ~35MB
- **With UPX**: ~12MB

---

## Configuration Management

### Environment Variables Reference

CodeAI uses environment variables for configuration. Command-line flags take precedence over environment variables.

#### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CODEAI_HOST` | `localhost` | Server bind address |
| `CODEAI_PORT` | `8080` | Server listen port |
| `CODEAI_READ_TIMEOUT` | `15s` | HTTP read timeout |
| `CODEAI_WRITE_TIMEOUT` | `15s` | HTTP write timeout |
| `CODEAI_IDLE_TIMEOUT` | `60s` | HTTP idle timeout |

#### Database Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CODEAI_DB_HOST` | `localhost` | PostgreSQL host |
| `CODEAI_DB_PORT` | `5432` | PostgreSQL port |
| `CODEAI_DB_NAME` | `codeai` | Database name |
| `CODEAI_DB_USER` | `postgres` | Database user |
| `CODEAI_DB_PASSWORD` | (none) | Database password |
| `CODEAI_DB_SSLMODE` | `disable` | SSL mode (`disable`, `require`, `verify-ca`, `verify-full`) |
| `CODEAI_DB_MAX_OPEN_CONNS` | `25` | Maximum open connections |
| `CODEAI_DB_MAX_IDLE_CONNS` | `5` | Maximum idle connections |
| `CODEAI_DB_CONN_MAX_LIFETIME` | `5m` | Connection max lifetime |

#### Cache Configuration (Redis)

| Variable | Default | Description |
|----------|---------|-------------|
| `CODEAI_CACHE_TYPE` | `memory` | Cache type (`memory`, `redis`) |
| `CODEAI_REDIS_URL` | (none) | Redis URL (e.g., `redis://localhost:6379`) |
| `CODEAI_REDIS_PASSWORD` | (none) | Redis password |
| `CODEAI_REDIS_DB` | `0` | Redis database number |
| `CODEAI_REDIS_POOL_SIZE` | `10` | Connection pool size |
| `CODEAI_REDIS_CLUSTER_ADDRS` | (none) | Comma-separated cluster addresses |
| `CODEAI_REDIS_CLUSTER_MODE` | `false` | Enable cluster mode |
| `CODEAI_CACHE_PREFIX` | `codeai` | Key prefix for cache entries |
| `CODEAI_CACHE_DEFAULT_TTL` | `5m` | Default cache TTL |

#### Authentication Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CODEAI_JWT_SECRET` | (none) | Secret for HS256 JWT validation |
| `CODEAI_JWT_PUBLIC_KEY` | (none) | PEM-encoded public key for RS256 |
| `CODEAI_JWT_JWKS_URL` | (none) | JWKS endpoint URL |
| `CODEAI_JWT_ISSUER` | (none) | Expected JWT issuer |
| `CODEAI_JWT_AUDIENCE` | (none) | Expected JWT audience |
| `CODEAI_JWT_ROLES_CLAIM` | `roles` | Claim name for roles |

#### Metrics Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CODEAI_METRICS_ENABLED` | `true` | Enable Prometheus metrics |
| `CODEAI_METRICS_NAMESPACE` | `codeai` | Metrics namespace prefix |
| `CODEAI_METRICS_PATH` | `/metrics` | Metrics endpoint path |
| `CODEAI_METRICS_PROCESS` | `true` | Enable process metrics |
| `CODEAI_METRICS_RUNTIME` | `true` | Enable Go runtime metrics |

#### Logging Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CODEAI_LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| `CODEAI_LOG_FORMAT` | `json` | Log format (`json`, `text`) |

### Configuration Files

CodeAI looks for configuration in the following locations (in order of precedence):
1. Command-line flags
2. Environment variables
3. `$HOME/.codeai.yaml`
4. `/etc/codeai/config.yaml`

Example configuration file:

```yaml
# ~/.codeai.yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  host: "postgres.example.com"
  port: 5432
  name: "codeai"
  user: "codeai_user"
  sslmode: "require"

cache:
  type: "redis"
  redis:
    url: "redis://redis.example.com:6379"
    pool_size: 20

auth:
  jwt:
    issuer: "https://auth.example.com"
    audience: "codeai-api"

metrics:
  enabled: true
  namespace: "codeai"

logging:
  level: "info"
  format: "json"
```

### Secrets Management

#### Environment Variables (Simple)

```bash
export CODEAI_DB_PASSWORD="your-secure-password"
export CODEAI_JWT_SECRET="your-jwt-secret-key"
```

#### HashiCorp Vault

```bash
# Using vault agent
vault agent -config=vault-agent.hcl

# Template for secrets
{{ with secret "secret/data/codeai/database" }}
export CODEAI_DB_PASSWORD="{{ .Data.data.password }}"
{{ end }}
```

#### Kubernetes Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: codeai-secrets
type: Opaque
stringData:
  db-password: "your-secure-password"
  jwt-secret: "your-jwt-secret-key"
---
# Reference in deployment
env:
  - name: CODEAI_DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: codeai-secrets
        key: db-password
```

#### AWS Secrets Manager

```bash
# Using AWS CLI
aws secretsmanager get-secret-value --secret-id codeai/database | \
  jq -r '.SecretString' | jq -r '.password'

# In entrypoint script
export CODEAI_DB_PASSWORD=$(aws secretsmanager get-secret-value \
  --secret-id codeai/database --query SecretString --output text | jq -r '.password')
```

### Configuration Validation

CodeAI validates configuration at startup. Invalid configuration prevents the server from starting:

```bash
# Validate configuration without starting
codeai config validate

# Show current configuration (masks secrets)
codeai config show
```

---

## Database Setup

### PostgreSQL Version Requirements

- **Minimum**: PostgreSQL 12
- **Recommended**: PostgreSQL 15+
- **Required Extensions**: None (uses standard SQL)

### Installation

#### Ubuntu/Debian

```bash
sudo apt-get update
sudo apt-get install postgresql-15 postgresql-contrib-15
sudo systemctl enable postgresql
sudo systemctl start postgresql
```

#### macOS (Homebrew)

```bash
brew install postgresql@15
brew services start postgresql@15
```

#### Docker

```bash
docker run -d \
  --name codeai-postgres \
  -e POSTGRES_USER=codeai \
  -e POSTGRES_PASSWORD=your-password \
  -e POSTGRES_DB=codeai \
  -p 5432:5432 \
  -v postgres-data:/var/lib/postgresql/data \
  postgres:15-alpine
```

### Initial Setup

```sql
-- Create database and user
CREATE USER codeai_user WITH PASSWORD 'secure-password';
CREATE DATABASE codeai OWNER codeai_user;
GRANT ALL PRIVILEGES ON DATABASE codeai TO codeai_user;

-- Connect to codeai database
\c codeai

-- Grant schema permissions
GRANT ALL ON SCHEMA public TO codeai_user;
```

### Connection Pooling Configuration

CodeAI uses Go's built-in connection pooling with sensible defaults:

```go
// Current defaults in internal/database/database.go
db.SetMaxOpenConns(25)       // Maximum number of open connections
db.SetMaxIdleConns(5)        // Maximum number of idle connections
db.SetConnMaxLifetime(5 * time.Minute)  // Maximum connection lifetime
db.SetConnMaxIdleTime(1 * time.Minute)  // Maximum idle time
```

**Production recommendations**:

| Workload | `max_open_conns` | `max_idle_conns` | `conn_max_lifetime` |
|----------|------------------|------------------|---------------------|
| Light | 10 | 5 | 5m |
| Medium | 25 | 10 | 5m |
| Heavy | 50 | 20 | 3m |
| Very Heavy | 100 | 30 | 2m |

**PostgreSQL server settings** (postgresql.conf):

```ini
# Recommended settings for CodeAI
max_connections = 200        # Total connections across all clients
shared_buffers = 256MB       # 25% of available RAM
effective_cache_size = 768MB # 75% of available RAM
work_mem = 64MB
maintenance_work_mem = 128MB
```

### Migration Execution

#### Running Migrations

```bash
# Apply all pending migrations
codeai server migrate

# Preview migrations without applying (dry-run)
codeai server migrate --dry-run

# Migrate with custom database connection
codeai server migrate \
  --db-host postgres.example.com \
  --db-port 5432 \
  --db-name codeai \
  --db-user codeai_user \
  --db-password "$DB_PASSWORD" \
  --db-sslmode require
```

#### Migration Files

Migrations are embedded in the binary. Current migrations:

1. `001_create_configs_table.sql` - Stores DSL configuration documents
2. `002_create_deployments_table.sql` - Tracks deployment states
3. `003_create_executions_table.sql` - Records execution history

#### Rollback Procedure

For manual rollback (emergency only):

```sql
-- Drop tables in reverse order (preserves foreign key constraints)
DROP TABLE IF EXISTS executions;
DROP TABLE IF EXISTS deployments;
DROP TABLE IF EXISTS configs;
```

### Backup and Restore Procedures

#### Backup

```bash
# Full database backup
pg_dump -h localhost -U codeai_user -d codeai -F custom -f codeai_backup.dump

# Backup with compression
pg_dump -h localhost -U codeai_user -d codeai -F custom -Z 9 -f codeai_backup.dump.gz

# Automated daily backup script
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/var/backups/codeai"
mkdir -p $BACKUP_DIR
pg_dump -h localhost -U codeai_user -d codeai -F custom -f "$BACKUP_DIR/codeai_$DATE.dump"
# Retain only last 7 days
find $BACKUP_DIR -name "*.dump" -mtime +7 -delete
```

#### Restore

```bash
# Full restore
pg_restore -h localhost -U codeai_user -d codeai -c codeai_backup.dump

# Restore to new database
createdb -h localhost -U postgres codeai_restored
pg_restore -h localhost -U codeai_user -d codeai_restored codeai_backup.dump
```

---

## Deployment Options

### Bare Metal / VM with Systemd

#### Create System User

```bash
sudo useradd -r -s /bin/false codeai
sudo mkdir -p /opt/codeai /var/log/codeai
sudo chown codeai:codeai /opt/codeai /var/log/codeai
```

#### Install Binary

```bash
sudo cp bin/codeai /opt/codeai/codeai
sudo chmod +x /opt/codeai/codeai
```

#### Systemd Service File

Create `/etc/systemd/system/codeai.service`:

```ini
[Unit]
Description=CodeAI API Server
Documentation=https://github.com/bargom/codeai
After=network-online.target postgresql.service
Wants=network-online.target
Requires=postgresql.service

[Service]
Type=simple
User=codeai
Group=codeai
WorkingDirectory=/opt/codeai

# Environment configuration
EnvironmentFile=-/etc/codeai/environment
Environment=CODEAI_HOST=0.0.0.0
Environment=CODEAI_PORT=8080

# Execute
ExecStart=/opt/codeai/codeai server start
ExecReload=/bin/kill -HUP $MAINPID

# Restart policy
Restart=always
RestartSec=5
StartLimitIntervalSec=60
StartLimitBurst=3

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/log/codeai

# Resource limits
LimitNOFILE=65535
LimitNPROC=4096
MemoryMax=2G
CPUQuota=200%

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=codeai

[Install]
WantedBy=multi-user.target
```

#### Environment File

Create `/etc/codeai/environment`:

```bash
CODEAI_DB_HOST=localhost
CODEAI_DB_PORT=5432
CODEAI_DB_NAME=codeai
CODEAI_DB_USER=codeai_user
CODEAI_DB_PASSWORD=your-secure-password
CODEAI_DB_SSLMODE=require

CODEAI_CACHE_TYPE=redis
CODEAI_REDIS_URL=redis://localhost:6379

CODEAI_JWT_ISSUER=https://auth.example.com
CODEAI_JWT_AUDIENCE=codeai-api
```

#### Enable and Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable codeai
sudo systemctl start codeai
sudo systemctl status codeai

# View logs
journalctl -u codeai -f
```

### Docker

#### Dockerfile

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with optimizations
ARG VERSION=dev
ARG BUILD_DATE
ARG GIT_COMMIT

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w \
        -X 'github.com/bargom/codeai/cmd/codeai/cmd.Version=${VERSION}' \
        -X 'github.com/bargom/codeai/cmd/codeai/cmd.BuildDate=${BUILD_DATE}' \
        -X 'github.com/bargom/codeai/cmd/codeai/cmd.GitCommit=${GIT_COMMIT}'" \
    -o /codeai ./cmd/codeai

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 codeai && \
    adduser -u 1000 -G codeai -s /bin/sh -D codeai

WORKDIR /app

COPY --from=builder /codeai /app/codeai

USER codeai

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/codeai"]
CMD ["server", "start", "--host", "0.0.0.0"]
```

#### Build Image

```bash
docker build \
  --build-arg VERSION=$(git describe --tags --always) \
  --build-arg BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --build-arg GIT_COMMIT=$(git rev-parse HEAD) \
  -t codeai:latest .
```

#### docker-compose.yml

```yaml
version: '3.8'

services:
  codeai:
    image: codeai:latest
    build:
      context: .
      args:
        VERSION: ${VERSION:-dev}
        BUILD_DATE: ${BUILD_DATE:-unknown}
        GIT_COMMIT: ${GIT_COMMIT:-unknown}
    ports:
      - "8080:8080"
    environment:
      CODEAI_HOST: "0.0.0.0"
      CODEAI_PORT: "8080"
      CODEAI_DB_HOST: postgres
      CODEAI_DB_PORT: "5432"
      CODEAI_DB_NAME: codeai
      CODEAI_DB_USER: codeai
      CODEAI_DB_PASSWORD_FILE: /run/secrets/db_password
      CODEAI_DB_SSLMODE: disable
      CODEAI_CACHE_TYPE: redis
      CODEAI_REDIS_URL: redis://redis:6379
    secrets:
      - db_password
      - jwt_secret
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 256M
    restart: unless-stopped

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: codeai
      POSTGRES_PASSWORD_FILE: /run/secrets/db_password
      POSTGRES_DB: codeai
    volumes:
      - postgres_data:/var/lib/postgresql/data
    secrets:
      - db_password
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U codeai -d codeai"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

secrets:
  db_password:
    file: ./secrets/db_password.txt
  jwt_secret:
    file: ./secrets/jwt_secret.txt

volumes:
  postgres_data:
  redis_data:

networks:
  default:
    driver: bridge
```

#### Running with Docker Compose

```bash
# Create secrets directory
mkdir -p secrets
echo "your-secure-db-password" > secrets/db_password.txt
echo "your-jwt-secret-key" > secrets/jwt_secret.txt
chmod 600 secrets/*.txt

# Start services
docker-compose up -d

# Run migrations
docker-compose exec codeai /app/codeai server migrate

# View logs
docker-compose logs -f codeai

# Scale horizontally
docker-compose up -d --scale codeai=3
```

### Kubernetes

#### Namespace and ConfigMap

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: codeai
  labels:
    name: codeai
---
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: codeai-config
  namespace: codeai
data:
  CODEAI_HOST: "0.0.0.0"
  CODEAI_PORT: "8080"
  CODEAI_DB_HOST: "postgres-service"
  CODEAI_DB_PORT: "5432"
  CODEAI_DB_NAME: "codeai"
  CODEAI_DB_SSLMODE: "require"
  CODEAI_CACHE_TYPE: "redis"
  CODEAI_REDIS_URL: "redis://redis-service:6379"
  CODEAI_METRICS_ENABLED: "true"
  CODEAI_LOG_LEVEL: "info"
  CODEAI_LOG_FORMAT: "json"
```

#### Secrets

```yaml
# secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: codeai-secrets
  namespace: codeai
type: Opaque
stringData:
  db-password: "your-secure-password"
  db-user: "codeai_user"
  jwt-secret: "your-jwt-secret-key"
```

#### Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: codeai
  namespace: codeai
  labels:
    app: codeai
    version: v1
spec:
  replicas: 3
  selector:
    matchLabels:
      app: codeai
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: codeai
        version: v1
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: codeai
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        runAsGroup: 1000
        fsGroup: 1000
      containers:
        - name: codeai
          image: your-registry/codeai:latest
          imagePullPolicy: Always
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
          envFrom:
            - configMapRef:
                name: codeai-config
          env:
            - name: CODEAI_DB_USER
              valueFrom:
                secretKeyRef:
                  name: codeai-secrets
                  key: db-user
            - name: CODEAI_DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: codeai-secrets
                  key: db-password
            - name: CODEAI_JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: codeai-secrets
                  key: jwt-secret
          resources:
            requests:
              cpu: "100m"
              memory: "256Mi"
            limits:
              cpu: "1000m"
              memory: "1Gi"
          livenessProbe:
            httpGet:
              path: /health
              port: http
            initialDelaySeconds: 10
            periodSeconds: 15
            timeoutSeconds: 5
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /health
              port: http
            initialDelaySeconds: 5
            periodSeconds: 10
            timeoutSeconds: 3
            failureThreshold: 3
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop:
                - ALL
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchLabels:
                    app: codeai
                topologyKey: kubernetes.io/hostname
      topologySpreadConstraints:
        - maxSkew: 1
          topologyKey: topology.kubernetes.io/zone
          whenUnsatisfiable: ScheduleAnyway
          labelSelector:
            matchLabels:
              app: codeai
```

#### Service

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: codeai-service
  namespace: codeai
  labels:
    app: codeai
spec:
  type: ClusterIP
  ports:
    - name: http
      port: 80
      targetPort: http
      protocol: TCP
  selector:
    app: codeai
```

#### Ingress

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: codeai-ingress
  namespace: codeai
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/rate-limit-window: "1m"
spec:
  tls:
    - hosts:
        - api.codeai.example.com
      secretName: codeai-tls
  rules:
    - host: api.codeai.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: codeai-service
                port:
                  number: 80
```

#### Horizontal Pod Autoscaler

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: codeai-hpa
  namespace: codeai
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: codeai
  minReplicas: 3
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 10
          periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
        - type: Percent
          value: 100
          periodSeconds: 15
        - type: Pods
          value: 4
          periodSeconds: 15
      selectPolicy: Max
```

#### Pod Disruption Budget

```yaml
# pdb.yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: codeai-pdb
  namespace: codeai
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: codeai
```

#### Apply All Manifests

```bash
# Apply in order
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f secrets.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f ingress.yaml
kubectl apply -f hpa.yaml
kubectl apply -f pdb.yaml

# Run migrations as a Job
kubectl create job --from=cronjob/codeai-migrate codeai-migrate-manual
```

---

## Monitoring and Alerting

### Prometheus Scraping Configuration

CodeAI exposes Prometheus metrics at `/metrics`. Add the following to your Prometheus configuration:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'codeai'
    scrape_interval: 15s
    scrape_timeout: 10s
    metrics_path: /metrics

    # Static configuration
    static_configs:
      - targets: ['codeai.example.com:8080']
        labels:
          environment: production

    # OR Kubernetes service discovery
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - codeai
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        target_label: __address__
```

### Available Metrics

CodeAI exposes the following metrics:

#### HTTP Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `codeai_http_requests_total` | Counter | Total HTTP requests by method, path, status |
| `codeai_http_request_duration_seconds` | Histogram | Request latency in seconds |
| `codeai_http_request_size_bytes` | Histogram | Request size in bytes |
| `codeai_http_response_size_bytes` | Histogram | Response size in bytes |
| `codeai_http_active_requests` | Gauge | Currently active requests |

#### Database Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `codeai_db_queries_total` | Counter | Total database queries |
| `codeai_db_query_duration_seconds` | Histogram | Query latency |
| `codeai_db_connections_active` | Gauge | Active database connections |
| `codeai_db_connections_idle` | Gauge | Idle database connections |
| `codeai_db_query_errors_total` | Counter | Database errors by type |

#### Workflow Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `codeai_workflow_executions_total` | Counter | Workflow executions by status |
| `codeai_workflow_execution_duration_seconds` | Histogram | Workflow duration |
| `codeai_workflow_active_count` | Gauge | Active workflows |
| `codeai_workflow_step_duration_seconds` | Histogram | Step duration |

#### Integration Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `codeai_integration_calls_total` | Counter | External API calls |
| `codeai_integration_call_duration_seconds` | Histogram | API call latency |
| `codeai_integration_circuit_breaker_state` | Gauge | Circuit breaker state |
| `codeai_integration_retries_total` | Counter | Retry attempts |
| `codeai_integration_errors_total` | Counter | Integration errors |

### Grafana Dashboard

Import the following dashboard JSON or create panels manually:

```json
{
  "dashboard": {
    "title": "CodeAI Overview",
    "uid": "codeai-overview",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "sum(rate(codeai_http_requests_total[5m])) by (status_code)",
            "legendFormat": "{{status_code}}"
          }
        ]
      },
      {
        "title": "Request Latency (p99)",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.99, sum(rate(codeai_http_request_duration_seconds_bucket[5m])) by (le, path))",
            "legendFormat": "{{path}}"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "singlestat",
        "targets": [
          {
            "expr": "sum(rate(codeai_http_requests_total{status_code=~\"5..\"}[5m])) / sum(rate(codeai_http_requests_total[5m])) * 100"
          }
        ]
      },
      {
        "title": "Database Connections",
        "type": "graph",
        "targets": [
          {
            "expr": "codeai_db_connections_active",
            "legendFormat": "active"
          },
          {
            "expr": "codeai_db_connections_idle",
            "legendFormat": "idle"
          }
        ]
      },
      {
        "title": "Active Workflows",
        "type": "graph",
        "targets": [
          {
            "expr": "sum(codeai_workflow_active_count) by (workflow_name)",
            "legendFormat": "{{workflow_name}}"
          }
        ]
      }
    ]
  }
}
```

### Alert Rules

```yaml
# alerting-rules.yaml
groups:
  - name: codeai
    rules:
      # High Error Rate
      - alert: CodeAIHighErrorRate
        expr: |
          sum(rate(codeai_http_requests_total{status_code=~"5.."}[5m]))
          / sum(rate(codeai_http_requests_total[5m])) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate in CodeAI"
          description: "Error rate is {{ $value | humanizePercentage }} over the last 5 minutes"

      # High Latency
      - alert: CodeAIHighLatency
        expr: |
          histogram_quantile(0.99, sum(rate(codeai_http_request_duration_seconds_bucket[5m])) by (le)) > 2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High latency in CodeAI"
          description: "p99 latency is {{ $value | humanizeDuration }}"

      # Database Connection Pool Exhausted
      - alert: CodeAIDBConnectionPoolExhausted
        expr: codeai_db_connections_active >= 20
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Database connection pool nearing limit"
          description: "{{ $value }} connections active out of 25 max"

      # High Memory Usage
      - alert: CodeAIHighMemory
        expr: |
          container_memory_usage_bytes{container="codeai"}
          / container_spec_memory_limit_bytes{container="codeai"} > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage in CodeAI"
          description: "Memory usage is {{ $value | humanizePercentage }}"

      # Instance Down
      - alert: CodeAIDown
        expr: up{job="codeai"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "CodeAI instance is down"
          description: "Instance {{ $labels.instance }} has been down for more than 1 minute"

      # Circuit Breaker Open
      - alert: CodeAICircuitBreakerOpen
        expr: codeai_integration_circuit_breaker_state{state="open"} > 0
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Circuit breaker open for {{ $labels.service_name }}"
          description: "External service {{ $labels.service_name }} is experiencing issues"
```

### Log Aggregation Setup

#### Loki with Promtail

```yaml
# promtail-config.yaml
server:
  http_listen_port: 9080

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: codeai
    kubernetes_sd_configs:
      - role: pod
    pipeline_stages:
      - json:
          expressions:
            level: level
            msg: msg
            time: time
      - labels:
          level:
      - timestamp:
          source: time
          format: RFC3339Nano
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: keep
        regex: codeai
```

#### Fluent Bit for ELK

```yaml
# fluent-bit.conf
[INPUT]
    Name              tail
    Path              /var/log/containers/codeai-*.log
    Parser            docker
    Tag               codeai.*
    Refresh_Interval  5
    Mem_Buf_Limit     5MB

[FILTER]
    Name              parser
    Match             codeai.*
    Key_Name          log
    Parser            json

[OUTPUT]
    Name              es
    Match             codeai.*
    Host              elasticsearch
    Port              9200
    Index             codeai-logs
    Type              _doc
```

---

## Scaling Considerations

### Horizontal Scaling

CodeAI is designed for horizontal scaling with stateless instances.

#### Multiple Instances

All instances share the same PostgreSQL and Redis backends:

```
                    ┌─────────────┐
                    │   Load      │
                    │  Balancer   │
                    └──────┬──────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
     ┌─────┴─────┐   ┌─────┴─────┐   ┌─────┴─────┐
     │  CodeAI   │   │  CodeAI   │   │  CodeAI   │
     │ Instance 1│   │ Instance 2│   │ Instance 3│
     └─────┬─────┘   └─────┬─────┘   └─────┬─────┘
           │               │               │
           └───────────────┼───────────────┘
                           │
              ┌────────────┴────────────┐
              │                         │
        ┌─────┴─────┐             ┌─────┴─────┐
        │ PostgreSQL│             │   Redis   │
        │  Primary  │             │  Cluster  │
        └───────────┘             └───────────┘
```

#### Scaling Guidelines

| Metric | Scale Trigger | Action |
|--------|---------------|--------|
| CPU > 70% | 2+ minutes | Add instance |
| Memory > 80% | 2+ minutes | Add instance |
| Request latency p99 > 1s | 5+ minutes | Add instance |
| Active connections > 80% pool | Any | Add instance |

### Database Connection Limits

With horizontal scaling, connection pooling becomes critical:

```
Total Connections = Instances × MaxOpenConns per Instance
```

**Example**: 10 instances × 25 connections = 250 total connections

Configure PostgreSQL accordingly:

```sql
-- postgresql.conf
max_connections = 300  -- Leave headroom for admin connections
```

Consider using PgBouncer for connection pooling at scale:

```ini
# pgbouncer.ini
[databases]
codeai = host=postgres-primary dbname=codeai

[pgbouncer]
listen_port = 6432
listen_addr = 0.0.0.0
auth_type = md5
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 50
```

### Redis Cluster Mode

For high-throughput caching, use Redis Cluster:

```yaml
# Redis Cluster environment variables
CODEAI_REDIS_CLUSTER_MODE: "true"
CODEAI_REDIS_CLUSTER_ADDRS: "redis-0:6379,redis-1:6379,redis-2:6379"
```

### Workflow Worker Scaling

For heavy workflow processing, scale workers independently:

```yaml
# Separate deployment for workflow workers
apiVersion: apps/v1
kind: Deployment
metadata:
  name: codeai-worker
spec:
  replicas: 5
  template:
    spec:
      containers:
        - name: worker
          image: codeai:latest
          command: ["/app/codeai", "worker", "start"]
          resources:
            requests:
              cpu: "500m"
              memory: "512Mi"
            limits:
              cpu: "2000m"
              memory: "2Gi"
```

---

## Security Checklist

### Pre-Deployment

- [ ] **Database credentials** are stored in secrets management (not environment files)
- [ ] **JWT secrets** are at least 256 bits for HS256 or using RS256 with proper key management
- [ ] **PostgreSQL SSL mode** is set to `require` or `verify-full` in production
- [ ] **Redis** is configured with password authentication
- [ ] **Container runs as non-root** user (uid 1000)
- [ ] **Network policies** restrict pod-to-pod communication
- [ ] **Resource limits** are set to prevent resource exhaustion

### JWT Secret Rotation

```bash
# 1. Generate new secret
NEW_SECRET=$(openssl rand -base64 32)

# 2. Update secret in secrets manager
kubectl create secret generic codeai-secrets \
  --from-literal=jwt-secret="$NEW_SECRET" \
  --dry-run=client -o yaml | kubectl apply -f -

# 3. Rolling restart to pick up new secret
kubectl rollout restart deployment/codeai

# 4. Monitor for authentication errors
kubectl logs -l app=codeai -f | grep -i "auth\|jwt"
```

### Database Credential Rotation

```bash
# 1. Create new user in PostgreSQL
psql -h postgres -U postgres <<EOF
CREATE USER codeai_user_v2 WITH PASSWORD 'new-password';
GRANT ALL PRIVILEGES ON DATABASE codeai TO codeai_user_v2;
GRANT ALL ON ALL TABLES IN SCHEMA public TO codeai_user_v2;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO codeai_user_v2;
EOF

# 2. Update secret
kubectl create secret generic codeai-secrets \
  --from-literal=db-user="codeai_user_v2" \
  --from-literal=db-password="new-password" \
  --dry-run=client -o yaml | kubectl apply -f -

# 3. Rolling restart
kubectl rollout restart deployment/codeai

# 4. Verify connections, then drop old user
psql -h postgres -U postgres -c "DROP USER codeai_user;"
```

### TLS/HTTPS Setup

#### Nginx with Let's Encrypt

```nginx
server {
    listen 443 ssl http2;
    server_name api.codeai.example.com;

    ssl_certificate /etc/letsencrypt/live/api.codeai.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.codeai.example.com/privkey.pem;

    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers on;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;
    add_header X-XSS-Protection "1; mode=block";

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

server {
    listen 80;
    server_name api.codeai.example.com;
    return 301 https://$server_name$request_uri;
}
```

### Rate Limiting

#### Nginx

```nginx
# Define rate limit zones
limit_req_zone $binary_remote_addr zone=api_limit:10m rate=100r/s;
limit_req_zone $binary_remote_addr zone=auth_limit:10m rate=10r/s;

server {
    location /api/ {
        limit_req zone=api_limit burst=50 nodelay;
        proxy_pass http://codeai;
    }

    location /api/auth/ {
        limit_req zone=auth_limit burst=5 nodelay;
        proxy_pass http://codeai;
    }
}
```

#### Kubernetes Ingress

```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/limit-rps: "100"
    nginx.ingress.kubernetes.io/limit-connections: "10"
```

### CORS Configuration

Configure CORS appropriately for your frontend:

```yaml
# Environment variables
CODEAI_CORS_ALLOWED_ORIGINS: "https://app.example.com,https://admin.example.com"
CODEAI_CORS_ALLOWED_METHODS: "GET,POST,PUT,DELETE,OPTIONS"
CODEAI_CORS_ALLOWED_HEADERS: "Authorization,Content-Type"
CODEAI_CORS_MAX_AGE: "86400"
```

### Security Scanning

Regular security scans should include:

```bash
# Container image scanning
trivy image codeai:latest

# Dependency vulnerability check
go list -json -m all | nancy sleuth

# Static analysis
gosec ./...

# Kubernetes manifest scanning
kubesec scan deployment.yaml
```

---

## Quick Reference

### Essential Commands

```bash
# Build
make build

# Run locally
./bin/codeai server start --port 8080

# Run migrations
./bin/codeai server migrate

# Check health
curl http://localhost:8080/health

# View metrics
curl http://localhost:8080/metrics

# Check version
./bin/codeai version --output json
```

### Troubleshooting

| Issue | Check |
|-------|-------|
| Can't connect to database | `CODEAI_DB_*` env vars, PostgreSQL running, firewall rules |
| High latency | Database connection pool, Redis connectivity, resource limits |
| 401 Unauthorized | JWT configuration, token expiration, JWKS endpoint |
| Pod crashlooping | `kubectl logs`, liveness probe, memory limits |
| Metrics not scraped | `/metrics` endpoint accessible, Prometheus config |

### Health Checks

```bash
# API health
curl -s http://localhost:8080/health | jq .

# Database connectivity
./bin/codeai server migrate --dry-run

# Redis connectivity
redis-cli -h redis-host ping

# Full system check
curl -s http://localhost:8080/health | jq -e '.status == "healthy"'
```

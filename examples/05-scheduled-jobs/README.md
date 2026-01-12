# Scheduled Jobs Example

Demonstrates background job processing patterns including cron-based scheduling, job queue configuration, error handling, retries, and job monitoring.

## Overview

This example demonstrates:
- Cron-based scheduled jobs
- Interval-based recurring jobs
- Job queue configuration with priorities
- Error handling and retry strategies
- Report generation and distribution
- Data cleanup and maintenance jobs
- External data synchronization
- Job monitoring and management

## File Structure

```
05-scheduled-jobs/
├── scheduled-jobs.codeai    # Main DSL file
├── README.md                # This file
└── test.sh                  # Sample curl commands for testing
```

## Job Scheduler Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Job Scheduler System                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌───────────────────┐                                                  │
│  │   Job Definitions │                                                  │
│  │   - Cron schedule │                                                  │
│  │   - Interval      │                                                  │
│  │   - One-time      │                                                  │
│  └─────────┬─────────┘                                                  │
│            │                                                             │
│            ▼                                                             │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                         Queue System                               │  │
│  │                                                                    │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐               │  │
│  │  │  Critical   │  │   Default   │  │    Low      │               │  │
│  │  │  (60%)      │  │   (30%)     │  │   (10%)     │               │  │
│  │  │             │  │             │  │             │               │  │
│  │  │ • Webhooks  │  │ • Reports   │  │ • Cleanup   │               │  │
│  │  │ • Payments  │  │ • Sync      │  │ • Archive   │               │  │
│  │  │ • Alerts    │  │ • Alerts    │  │ • Maintain  │               │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘               │  │
│  │                                                                    │  │
│  └────────────────────────────┬──────────────────────────────────────┘  │
│                               │                                          │
│                               ▼                                          │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                        Worker Pool                                 │  │
│  │                                                                    │  │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ... ┌────────┐      │  │
│  │  │Worker 1│ │Worker 2│ │Worker 3│ │Worker 4│     │Worker N│      │  │
│  │  └────────┘ └────────┘ └────────┘ └────────┘     └────────┘      │  │
│  │                                                                    │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Jobs Overview

### Report Generation Jobs

| Job | Schedule | Description |
|-----|----------|-------------|
| `DailySalesReport` | 6:00 AM daily | Daily sales summary |
| `WeeklyInventoryReport` | Monday 8:00 AM | Inventory status |
| `MonthlyBusinessSummary` | 1st of month | Monthly business metrics |

### Cleanup Jobs

| Job | Schedule | Description |
|-----|----------|-------------|
| `CleanupExpiredSessions` | Every hour | Remove expired sessions |
| `CleanupJobExecutions` | 2:00 AM daily | Archive old executions |
| `CleanupExpiredReports` | 3:00 AM daily | Delete expired files |

### Maintenance Jobs

| Job | Schedule | Description |
|-----|----------|-------------|
| `DatabaseMaintenance` | Sunday 4:00 AM | VACUUM, REINDEX |

### Sync and Processing Jobs

| Job | Schedule | Description |
|-----|----------|-------------|
| `SyncExternalInventory` | Every 30 min | Sync from warehouse |
| `ProcessPendingWebhooks` | Every 5 min | Retry failed webhooks |
| `LowStockAlert` | Every 4 hours | Stock level alerts |

## Queue Configuration

| Queue | Priority | Workers | Use Case |
|-------|----------|---------|----------|
| `critical` | 6 (60%) | ~6 | Webhooks, payments, alerts |
| `default` | 3 (30%) | ~3 | Reports, syncs |
| `low` | 1 (10%) | ~1 | Cleanup, maintenance |

## Cron Schedule Syntax

```
┌───────────── minute (0 - 59)
│ ┌───────────── hour (0 - 23)
│ │ ┌───────────── day of month (1 - 31)
│ │ │ ┌───────────── month (1 - 12)
│ │ │ │ ┌───────────── day of week (0 - 6) (Sunday = 0)
│ │ │ │ │
* * * * *
```

### Common Patterns

| Schedule | Cron Expression | Description |
|----------|-----------------|-------------|
| Daily at 6 AM | `0 6 * * *` | Every day at 6:00 AM |
| Every Monday at 8 AM | `0 8 * * 1` | Mondays only |
| Every hour | `0 * * * *` | On the hour |
| Every 15 minutes | `*/15 * * * *` | Quarter-hourly |
| 1st of month | `0 7 1 * *` | Monthly |
| Weekdays at 9 AM | `0 9 * * 1-5` | Mon-Fri |

### Interval Shortcuts

| Syntax | Equivalent |
|--------|------------|
| `every 1h` | Every hour |
| `every 30m` | Every 30 minutes |
| `every 5m` | Every 5 minutes |
| `every 4h` | Every 4 hours |

## Job Definition Structure

```codeai
job JobName {
    description: "What this job does"

    // Schedule (choose one)
    schedule: "0 6 * * *"      // Cron expression
    schedule: every 1h          // Interval
    run_at: "2026-01-15T10:00:00Z"  // One-time

    timezone: "America/New_York"

    // Execution settings
    queue: default              // Queue name
    timeout: 30m                // Max execution time
    retry: 3 times              // Retry on failure

    // Job logic
    steps {
        step_name {
            // Step logic
        }
    }

    // Event handlers
    on_complete: emit(JobCompleted)
    on_fail: emit(JobFailed)
}
```

## Step Types

### Query Data

```codeai
fetch_data {
    query: select Product where quantity < 10
    as: low_stock_items

    on_empty: skip("No data to process")
}
```

### Execute Action

```codeai
cleanup_records {
    action: delete Session where expires_at < now()
    as: delete_result
}
```

### Call External Service

```codeai
sync_inventory {
    call: WarehouseAPI.get_inventory {
        since: last_sync_timestamp()
    }
    as: inventory
    timeout: 60s
    on_fail: abort("Sync failed")
}
```

### Generate Report

```codeai
create_report {
    template: "daily_sales_report"
    data: {
        period: yesterday()
        metrics: calculated_metrics
    }
    format: pdf
    as: report_file
}
```

### Send Notifications

```codeai
notify_team {
    send: email(recipients) {
        template: "daily_report"
        subject: "Daily Report"
        attachment: report_file
    }
}

send_alert {
    send: slack("#alerts") {
        template: "low_stock_alert"
        data: alert_data
    }
}
```

### For-Each Processing

```codeai
process_items {
    for_each: items
    parallel: 10                // Optional parallel processing

    action: process_item(item)
}
```

## Retry Strategies

### Simple Retry

```codeai
retry: 3 times
```

### Exponential Backoff

```codeai
retry: 3 times with exponential_backoff
// Delays: 100ms, 200ms, 400ms
```

### Custom Delays

```codeai
retry: 5 times
retry_delays: [1m, 5m, 30m, 2h, 8h]
```

## API Endpoints

### Job Management

| Method | Path | Description |
|--------|------|-------------|
| GET | `/jobs` | List all jobs |
| GET | `/jobs/{id}` | Get job details |
| POST | `/jobs/{id}/run` | Trigger job manually |
| POST | `/jobs/{id}/pause` | Pause job |
| POST | `/jobs/{id}/resume` | Resume job |
| PUT | `/jobs/{id}/schedule` | Update schedule |

### Job Executions

| Method | Path | Description |
|--------|------|-------------|
| GET | `/jobs/{id}/executions` | List executions |
| GET | `/executions/{id}` | Get execution details |
| POST | `/executions/{id}/cancel` | Cancel running |
| POST | `/executions/{id}/retry` | Retry failed |

### Reports

| Method | Path | Description |
|--------|------|-------------|
| GET | `/reports` | List reports |
| GET | `/reports/{id}` | Get report details |
| GET | `/reports/{id}/download` | Get download URL |

### Queue Status

| Method | Path | Description |
|--------|------|-------------|
| GET | `/queues/status` | Queue statistics |

## Step-by-Step Instructions

### 1. Generate the API

```bash
codeai generate examples/05-scheduled-jobs/scheduled-jobs.codeai
```

### 2. Configure Environment

```bash
# Database
export DATABASE_URL="postgres://localhost:5432/jobs"
export REDIS_URL="redis://localhost:6379"

# External APIs
export WAREHOUSE_API_URL="https://api.warehouse.example.com"
export WAREHOUSE_API_KEY="your-api-key"

# Notifications
export SLACK_WEBHOOK="https://hooks.slack.com/..."
export SMTP_HOST="smtp.example.com"
```

### 3. Run Migrations

```bash
codeai migrate up
```

### 4. Start the Server

```bash
# Start API server
codeai run

# Start job workers (separate process)
codeai worker start
```

## Sample Requests

### List All Jobs

```bash
curl http://localhost:8080/jobs \
  -H "Authorization: Bearer <admin-token>"
```

**Response:**
```json
{
  "data": [
    {
      "id": "job-uuid",
      "name": "DailySalesReport",
      "job_type": "report",
      "cron_expression": "0 6 * * *",
      "is_enabled": true,
      "next_run_at": "2026-01-13T06:00:00Z",
      "last_run_status": "success"
    }
  ]
}
```

### Manually Trigger a Job

```bash
curl -X POST http://localhost:8080/jobs/<job-id>/run \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-token>" \
  -d '{
    "input": {
      "start_date": "2026-01-01",
      "end_date": "2026-01-12"
    }
  }'
```

**Response:**
```json
{
  "id": "execution-uuid",
  "job_id": "job-uuid",
  "execution_id": "exec-2026-01-12-manual-1",
  "status": "pending",
  "scheduled_at": "2026-01-12T15:30:00Z"
}
```

### Pause a Job

```bash
curl -X POST http://localhost:8080/jobs/<job-id>/pause \
  -H "Authorization: Bearer <admin-token>"
```

### Update Job Schedule

```bash
curl -X PUT http://localhost:8080/jobs/<job-id>/schedule \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-token>" \
  -d '{
    "cron_expression": "0 7 * * *",
    "timezone": "America/Los_Angeles"
  }'
```

### Get Queue Status

```bash
curl http://localhost:8080/queues/status \
  -H "Authorization: Bearer <admin-token>"
```

**Response:**
```json
{
  "queues": [
    {
      "name": "critical",
      "pending": 2,
      "active": 1,
      "completed": 150,
      "failed": 3,
      "scheduled": 5
    },
    {
      "name": "default",
      "pending": 10,
      "active": 3,
      "completed": 500,
      "failed": 12,
      "scheduled": 25
    }
  ],
  "workers": {
    "total": 10,
    "active": 4,
    "idle": 6
  }
}
```

### List Job Executions

```bash
curl "http://localhost:8080/jobs/<job-id>/executions?status=failed" \
  -H "Authorization: Bearer <admin-token>"
```

### Retry Failed Execution

```bash
curl -X POST http://localhost:8080/executions/<execution-id>/retry \
  -H "Authorization: Bearer <admin-token>"
```

### Download a Report

```bash
curl http://localhost:8080/reports/<report-id>/download \
  -H "Authorization: Bearer <token>"
```

**Response:**
```json
{
  "download_url": "https://storage.example.com/reports/daily-sales-2026-01-12.pdf?token=...",
  "expires_at": "2026-01-12T16:30:00Z"
}
```

## Key Concepts Demonstrated

1. **Cron Scheduling**: Standard cron expressions with timezone support
2. **Interval Scheduling**: Simple recurring intervals
3. **Queue Priorities**: Critical > Default > Low
4. **Retry Strategies**: Simple, exponential backoff, custom delays
5. **Step-based Jobs**: Multi-step job definitions
6. **Parallel Processing**: Process items concurrently
7. **Error Handling**: Per-step and per-job error handling
8. **Job Monitoring**: Status, executions, queue metrics
9. **Report Generation**: Template-based reports with distribution

## Monitoring

### Key Metrics to Track

- **Job Success Rate**: `job_executions_total{status="completed"} / job_executions_total`
- **Queue Depth**: Pending tasks per queue
- **Execution Duration**: Histogram by job type
- **Retry Rate**: Retry attempts per job
- **Worker Utilization**: Active vs idle workers

### Alerts to Configure

| Alert | Condition | Action |
|-------|-----------|--------|
| Job Failed | 3+ consecutive failures | Notify Slack |
| Queue Backlog | >100 pending tasks | Scale workers |
| Long Running | Duration > 2x timeout | Investigate |
| Worker Starvation | 0 idle workers | Scale workers |

## Next Steps

- See [E-commerce](../03-ecommerce/) for workflow orchestration
- Check [Integrations](../04-integrations/) for external API patterns
- Review [Hello World](../01-hello-world/) for basic patterns

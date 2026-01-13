# Workflow & Job Example

This example demonstrates the CodeAI DSL for defining Temporal workflows and Asynq background jobs.

## Features Demonstrated

### Temporal Workflows

- **Event-triggered workflows**: Automatically start when specific events occur (e.g., `order.created`)
- **Schedule-triggered workflows**: Run on a cron schedule (e.g., daily reports at 6 AM)
- **Manual workflows**: Started programmatically on-demand (e.g., user onboarding)
- **Parallel step execution**: Run multiple activities concurrently
- **Conditional steps**: Execute steps based on previous step outputs
- **Input mappings**: Pass data between workflow steps
- **Retry policies**: Configure automatic retries with exponential backoff
- **Timeouts**: Set maximum execution time for workflows

### Asynq Background Jobs

- **Cron-scheduled jobs**: Define recurring tasks using cron expressions
- **Queue priorities**: Assign jobs to priority queues (critical, default, low_priority)
- **Retry configuration**: Control retry behavior for failed jobs

## Workflow Examples

### Order Fulfillment Workflow

An event-triggered workflow that orchestrates the order fulfillment process:

1. Validates the order
2. Checks inventory availability
3. Processes payment
4. Reserves inventory and notifies warehouse (in parallel)
5. Sends confirmation email

### Daily Sales Report Workflow

A scheduled workflow running at 6 AM daily:

1. Calculates the date range
2. Fetches sales data
3. Aggregates metrics
4. Generates PDF report
5. Uploads to storage and sends email (in parallel)

### User Onboarding Workflow

A manual workflow for onboarding new users:

1. Creates user profile
2. Sets up preferences
3. Sends welcome email, creates tutorial tasks, sets up integrations (in parallel)
4. Activates the account

## Job Examples

| Job | Schedule | Queue | Description |
|-----|----------|-------|-------------|
| cleanup_sessions | Every hour | low_priority | Removes expired user sessions |
| weekly_backup | Sunday 2 AM | critical | Database backup |
| sync_inventory | Every 30 min | default | Syncs with external warehouse |
| send_reminders | Daily 9 AM | default | Sends pending reminder emails |
| archive_logs | 1st of month | low_priority | Archives old log files |
| generate_analytics | Hourly at :15 | default | Updates analytics metrics |
| health_check_services | Every 5 min | critical | Checks external service health |

## Running This Example

```bash
# Parse and validate the DSL file
./bin/codeai parse examples/09-workflows/with_workflows.cai

# Validate semantic correctness
./bin/codeai validate examples/09-workflows/with_workflows.cai
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         CodeAI Runtime                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌───────────────────┐           ┌───────────────────┐          │
│  │  Temporal Worker  │           │   Asynq Worker    │          │
│  │                   │           │                   │          │
│  │  ┌─────────────┐  │           │  ┌─────────────┐  │          │
│  │  │  Workflows  │  │           │  │    Jobs     │  │          │
│  │  └─────────────┘  │           │  └─────────────┘  │          │
│  │                   │           │                   │          │
│  │  ┌─────────────┐  │           │  ┌─────────────┐  │          │
│  │  │ Activities  │  │           │  │   Queues    │  │          │
│  │  └─────────────┘  │           │  └─────────────┘  │          │
│  └─────────────────┬─┘           └─────────────────┬─┘          │
│                    │                               │            │
│                    ▼                               ▼            │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Event Bus                            │    │
│  └─────────────────────────────────────────────────────────┘    │
│                    │                                            │
│                    ▼                                            │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │               PostgreSQL / Redis                        │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## DSL Syntax Reference

### Workflow Declaration

```
workflow <name> {
    trigger event "<event_name>"      // Event-triggered
    trigger schedule "<cron>"         // Cron-scheduled
    trigger manual                    // On-demand

    timeout "<duration>"              // e.g., "30m", "2h"

    steps {
        <step_name> {
            activity "<activity_type>"
            input {
                <key>: "<expression>"
            }
            if "<condition>"          // Optional condition
        }

        parallel {
            <step1> { ... }
            <step2> { ... }
        }
    }

    retry {
        max_attempts <n>
        initial_interval "<duration>"
        backoff_multiplier <float>
    }
}
```

### Job Declaration

```
job <name> {
    schedule "<cron_expression>"
    task "<task_type>"
    queue "<queue_name>"              // Optional, defaults to "default"

    retry {
        max_attempts <n>
        initial_interval "<duration>"
        backoff_multiplier <float>
    }
}
```

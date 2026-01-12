#!/bin/bash
# =============================================================================
# Scheduled Jobs Test Script
# =============================================================================
# Run this script to test scheduled job management endpoints
# Usage: ./test.sh [BASE_URL] [ADMIN_TOKEN]
# =============================================================================

BASE_URL="${1:-http://localhost:8080}"
ADMIN_TOKEN="${2:-admin-jwt-token}"

echo "============================================="
echo "Scheduled Jobs Test Script"
echo "Base URL: $BASE_URL"
echo "============================================="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

test_header() {
    echo -e "${BLUE}>>> $1${NC}"
}

section_header() {
    echo ""
    echo -e "${YELLOW}=============================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}=============================================${NC}"
    echo ""
}

# Store IDs
JOB_ID=""
EXECUTION_ID=""
REPORT_ID=""

# =============================================================================
section_header "1. Queue Status"
# =============================================================================

test_header "Get Queue Status"
curl -s "$BASE_URL/queues/status" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .
echo ""

# =============================================================================
section_header "2. List Scheduled Jobs"
# =============================================================================

test_header "List All Jobs"
JOBS_RESPONSE=$(curl -s "$BASE_URL/jobs" \
  -H "Authorization: Bearer $ADMIN_TOKEN")
echo "$JOBS_RESPONSE" | jq '.data | length, .[0].name, .[0].cron_expression'
JOB_ID=$(echo "$JOBS_RESPONSE" | jq -r '.data[0].id')
echo "First Job ID: $JOB_ID"
echo ""

test_header "Filter Jobs by Type: report"
curl -s "$BASE_URL/jobs?job_type=report" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.data | .[].name'
echo ""

test_header "Filter Jobs by Type: cleanup"
curl -s "$BASE_URL/jobs?job_type=cleanup" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.data | .[].name'
echo ""

test_header "List Enabled Jobs Only"
curl -s "$BASE_URL/jobs?is_enabled=true" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.data | length'
echo ""

# =============================================================================
section_header "3. Job Details"
# =============================================================================

test_header "Get Job Details"
curl -s "$BASE_URL/jobs/$JOB_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .
echo ""

# =============================================================================
section_header "4. Job Execution Management"
# =============================================================================

test_header "Manually Trigger Job"
EXECUTION_RESPONSE=$(curl -s -X POST "$BASE_URL/jobs/$JOB_ID/run" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "input": {
      "override_date": "2026-01-12"
    }
  }')
echo "$EXECUTION_RESPONSE" | jq .
EXECUTION_ID=$(echo "$EXECUTION_RESPONSE" | jq -r '.id')
echo "Execution ID: $EXECUTION_ID"
echo ""

test_header "List Job Executions"
curl -s "$BASE_URL/jobs/$JOB_ID/executions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.data | .[0:3] | .[].status'
echo ""

test_header "Get Execution Details"
curl -s "$BASE_URL/executions/$EXECUTION_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.status, .progress_percent, .duration_ms'
echo ""

# Wait for execution to progress
echo "Waiting for execution to progress..."
sleep 2
echo ""

test_header "Check Execution Progress"
curl -s "$BASE_URL/executions/$EXECUTION_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.status, .progress_percent, .progress_message'
echo ""

# =============================================================================
section_header "5. Job Control Operations"
# =============================================================================

test_header "Pause a Job"
curl -s -X POST "$BASE_URL/jobs/$JOB_ID/pause" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.is_paused'
echo ""

test_header "Verify Job is Paused"
curl -s "$BASE_URL/jobs/$JOB_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.is_paused, .next_run_at'
echo ""

test_header "Resume Job"
curl -s -X POST "$BASE_URL/jobs/$JOB_ID/resume" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.is_paused, .next_run_at'
echo ""

# =============================================================================
section_header "6. Update Job Schedule"
# =============================================================================

test_header "Get Current Schedule"
curl -s "$BASE_URL/jobs/$JOB_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.cron_expression, .timezone'
echo ""

test_header "Update Schedule (change to 7 AM)"
curl -s -X PUT "$BASE_URL/jobs/$JOB_ID/schedule" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "cron_expression": "0 7 * * *",
    "timezone": "America/Los_Angeles"
  }' | jq '.cron_expression, .timezone, .next_run_at'
echo ""

test_header "Revert Schedule (back to 6 AM UTC)"
curl -s -X PUT "$BASE_URL/jobs/$JOB_ID/schedule" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "cron_expression": "0 6 * * *",
    "timezone": "UTC"
  }' | jq '.cron_expression, .timezone'
echo ""

# =============================================================================
section_header "7. Execution Control"
# =============================================================================

# Trigger another execution to test cancel
test_header "Trigger Long-Running Job (for cancel test)"
LONG_EXEC=$(curl -s -X POST "$BASE_URL/jobs/$JOB_ID/run" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN")
LONG_EXEC_ID=$(echo "$LONG_EXEC" | jq -r '.id')
echo "Long Execution ID: $LONG_EXEC_ID"
echo ""

test_header "Cancel Running Execution"
curl -s -X POST "$BASE_URL/executions/$LONG_EXEC_ID/cancel" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.status'
echo ""

# =============================================================================
section_header "8. Failed Execution and Retry"
# =============================================================================

test_header "List Failed Executions"
FAILED_EXECUTIONS=$(curl -s "$BASE_URL/jobs/$JOB_ID/executions?status=failed" \
  -H "Authorization: Bearer $ADMIN_TOKEN")
echo "$FAILED_EXECUTIONS" | jq '.data | length'
FAILED_EXEC_ID=$(echo "$FAILED_EXECUTIONS" | jq -r '.data[0].id // empty')

if [ -n "$FAILED_EXEC_ID" ] && [ "$FAILED_EXEC_ID" != "null" ]; then
    test_header "Retry Failed Execution"
    curl -s -X POST "$BASE_URL/executions/$FAILED_EXEC_ID/retry" \
      -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.status, .attempt'
    echo ""
else
    echo "No failed executions to retry"
fi
echo ""

# =============================================================================
section_header "9. Reports"
# =============================================================================

test_header "List Generated Reports"
REPORTS_RESPONSE=$(curl -s "$BASE_URL/reports" \
  -H "Authorization: Bearer $ADMIN_TOKEN")
echo "$REPORTS_RESPONSE" | jq '.data | .[0:3] | .[].name'
REPORT_ID=$(echo "$REPORTS_RESPONSE" | jq -r '.data[0].id // empty')
echo ""

test_header "Filter Reports by Type: daily_sales"
curl -s "$BASE_URL/reports?report_type=daily_sales" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.data | length'
echo ""

if [ -n "$REPORT_ID" ] && [ "$REPORT_ID" != "null" ]; then
    test_header "Get Report Details"
    curl -s "$BASE_URL/reports/$REPORT_ID" \
      -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.name, .status, .file_format, .record_count'
    echo ""

    test_header "Get Report Download URL"
    curl -s "$BASE_URL/reports/$REPORT_ID/download" \
      -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.download_url, .expires_at'
    echo ""
else
    echo "No reports available yet"
fi

# =============================================================================
section_header "10. Cleanup Jobs Demonstration"
# =============================================================================

test_header "Find Cleanup Jobs"
curl -s "$BASE_URL/jobs?job_type=cleanup" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.data | .[].name'
echo ""

test_header "Get CleanupExpiredSessions Job Details"
CLEANUP_JOB=$(curl -s "$BASE_URL/jobs?job_type=cleanup" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.data[0].id')

if [ -n "$CLEANUP_JOB" ] && [ "$CLEANUP_JOB" != "null" ]; then
    curl -s "$BASE_URL/jobs/$CLEANUP_JOB" \
      -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.name, .cron_expression, .last_run_status, .run_count'
fi
echo ""

# =============================================================================
section_header "11. Queue Statistics Over Time"
# =============================================================================

test_header "Initial Queue Status"
curl -s "$BASE_URL/queues/status" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.queues'
echo ""

# Trigger a few jobs to see queue activity
test_header "Trigger Multiple Jobs (to see queue activity)"
for i in 1 2 3; do
    curl -s -X POST "$BASE_URL/jobs/$JOB_ID/run" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null
    echo "Triggered job $i"
done
echo ""

test_header "Queue Status After Triggering Jobs"
curl -s "$BASE_URL/queues/status" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.queues[] | {name, pending, active}'
echo ""

# =============================================================================
section_header "Test Complete!"
# =============================================================================

echo "Summary:"
echo "  - Listed and filtered scheduled jobs"
echo "  - Manually triggered job executions"
echo "  - Tested pause/resume functionality"
echo "  - Updated job schedules"
echo "  - Tested execution cancel and retry"
echo "  - Viewed generated reports"
echo "  - Checked queue statistics"
echo ""
echo "Job Types Demonstrated:"
echo "  - Report Generation (DailySalesReport, WeeklyInventoryReport)"
echo "  - Cleanup Jobs (CleanupExpiredSessions, CleanupJobExecutions)"
echo "  - Data Sync (SyncExternalInventory)"
echo "  - Alerting (LowStockAlert)"
echo "  - Processing (ProcessPendingWebhooks)"
echo "  - Maintenance (DatabaseMaintenance)"
echo ""
echo "Queue Priority Levels:"
echo "  - critical (60%): Webhooks, payments, time-sensitive"
echo "  - default (30%): Reports, syncs, alerts"
echo "  - low (10%): Cleanup, maintenance, archival"
echo ""
echo "Events Published:"
echo "  - ReportGenerated → Log"
echo "  - ReportGenerationFailed → Slack: #report-alerts"
echo "  - SyncFailed → Slack: #sync-alerts"

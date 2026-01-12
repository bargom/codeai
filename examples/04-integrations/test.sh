#!/bin/bash
# =============================================================================
# Integration Patterns Test Script
# =============================================================================
# Run this script to test integration patterns and webhook management
# Usage: ./test.sh [BASE_URL] [TOKEN]
# =============================================================================

BASE_URL="${1:-http://localhost:8080}"
TOKEN="${2:-your-jwt-token}"

echo "============================================="
echo "Integration Patterns Test Script"
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
WEBHOOK_ID=""
DELIVERY_ID=""

# =============================================================================
section_header "1. Integration Status"
# =============================================================================

test_header "Get All Integration Status"
curl -s "$BASE_URL/integrations/status" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo ""

test_header "Get GitHub API Health"
curl -s "$BASE_URL/integrations/GitHubAPI/health" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo ""

test_header "Get Weather API Health"
curl -s "$BASE_URL/integrations/WeatherAPI/health" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo ""

test_header "Get Stripe API Health"
curl -s "$BASE_URL/integrations/StripeAPI/health" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo ""

# =============================================================================
section_header "2. Webhook Management"
# =============================================================================

test_header "Create Webhook Endpoint"
WEBHOOK_RESPONSE=$(curl -s -X POST "$BASE_URL/webhooks" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Order Notifications",
    "url": "https://webhook.site/test-endpoint",
    "events": ["order.created", "order.shipped", "order.delivered"]
  }')
echo "$WEBHOOK_RESPONSE" | jq .
WEBHOOK_ID=$(echo "$WEBHOOK_RESPONSE" | jq -r '.id')
WEBHOOK_SECRET=$(echo "$WEBHOOK_RESPONSE" | jq -r '.secret')
echo "Webhook ID: $WEBHOOK_ID"
echo "Webhook Secret: ${WEBHOOK_SECRET:0:20}..."
echo ""

test_header "Create Another Webhook (Payment Events)"
curl -s -X POST "$BASE_URL/webhooks" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Payment Notifications",
    "url": "https://webhook.site/payment-endpoint",
    "events": ["payment.completed", "payment.failed", "refund.processed"]
  }' | jq '.id, .name, .events'
echo ""

test_header "List All Webhooks"
curl -s "$BASE_URL/webhooks" \
  -H "Authorization: Bearer $TOKEN" | jq '.data | length, .[0].name, .[0].is_active'
echo ""

test_header "Get Webhook Details"
curl -s "$BASE_URL/webhooks/$WEBHOOK_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.name, .url, .events, .is_healthy'
echo ""

# =============================================================================
section_header "3. Webhook Testing"
# =============================================================================

test_header "Send Test Event to Webhook"
DELIVERY_RESPONSE=$(curl -s -X POST "$BASE_URL/webhooks/$WEBHOOK_ID/test" \
  -H "Authorization: Bearer $TOKEN")
echo "$DELIVERY_RESPONSE" | jq .
DELIVERY_ID=$(echo "$DELIVERY_RESPONSE" | jq -r '.id')
echo "Delivery ID: $DELIVERY_ID"
echo ""

test_header "List Webhook Deliveries"
curl -s "$BASE_URL/webhooks/$WEBHOOK_ID/deliveries" \
  -H "Authorization: Bearer $TOKEN" | jq '.data | length, .[0].status, .[0].event_type'
echo ""

test_header "Get Delivery Details"
curl -s "$BASE_URL/webhooks/$WEBHOOK_ID/deliveries" \
  -H "Authorization: Bearer $TOKEN" | jq '.data[0]'
echo ""

# =============================================================================
section_header "4. Webhook Update Operations"
# =============================================================================

test_header "Update Webhook Events"
curl -s -X PUT "$BASE_URL/webhooks/$WEBHOOK_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "events": ["order.created", "order.shipped", "order.delivered", "order.cancelled"]
  }' | jq '.events'
echo ""

test_header "Deactivate Webhook"
curl -s -X PUT "$BASE_URL/webhooks/$WEBHOOK_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "is_active": false
  }' | jq '.is_active'
echo ""

test_header "Reactivate Webhook"
curl -s -X PUT "$BASE_URL/webhooks/$WEBHOOK_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "is_active": true
  }' | jq '.is_active'
echo ""

# =============================================================================
section_header "5. Failed Delivery and Retry"
# =============================================================================

test_header "Create Webhook with Invalid URL (to simulate failures)"
FAIL_WEBHOOK=$(curl -s -X POST "$BASE_URL/webhooks" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Failing Webhook",
    "url": "https://invalid-endpoint.example.com/webhook",
    "events": ["test.event"]
  }')
FAIL_WEBHOOK_ID=$(echo "$FAIL_WEBHOOK" | jq -r '.id')
echo "Failing Webhook ID: $FAIL_WEBHOOK_ID"
echo ""

test_header "Send Event to Failing Webhook"
FAIL_DELIVERY=$(curl -s -X POST "$BASE_URL/webhooks/$FAIL_WEBHOOK_ID/test" \
  -H "Authorization: Bearer $TOKEN")
echo "$FAIL_DELIVERY" | jq '.status, .error_message'
FAIL_DELIVERY_ID=$(echo "$FAIL_DELIVERY" | jq -r '.id')
echo ""

test_header "Check Failed Delivery Status"
curl -s "$BASE_URL/webhooks/$FAIL_WEBHOOK_ID/deliveries?status=failed" \
  -H "Authorization: Bearer $TOKEN" | jq '.data | .[0].status, .[0].attempt_count, .[0].error_message'
echo ""

test_header "Manually Retry Failed Delivery"
curl -s -X POST "$BASE_URL/webhooks/deliveries/$FAIL_DELIVERY_ID/retry" \
  -H "Authorization: Bearer $TOKEN" | jq '.status, .attempt_count'
echo ""

# =============================================================================
section_header "6. Circuit Breaker Operations"
# =============================================================================

test_header "Check Circuit Breaker States"
curl -s "$BASE_URL/integrations/status" \
  -H "Authorization: Bearer $TOKEN" | jq '.integrations[] | {name, circuit_state}'
echo ""

test_header "Simulate Circuit Breaker Open (if WeatherAPI is failing)"
echo "If WeatherAPI has failures, circuit may be open:"
curl -s "$BASE_URL/integrations/WeatherAPI/health" \
  -H "Authorization: Bearer $TOKEN" | jq '.circuit_state, .error_rate'
echo ""

test_header "Reset Circuit Breaker (Admin)"
curl -s -X POST "$BASE_URL/integrations/WeatherAPI/circuit/reset" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo ""

# =============================================================================
section_header "7. Filter and Pagination"
# =============================================================================

test_header "List Active Webhooks Only"
curl -s "$BASE_URL/webhooks?is_active=true" \
  -H "Authorization: Bearer $TOKEN" | jq '.data | length'
echo ""

test_header "List Webhooks with Pagination"
curl -s "$BASE_URL/webhooks?page=1&limit=2" \
  -H "Authorization: Bearer $TOKEN" | jq '.pagination'
echo ""

test_header "List Pending Deliveries"
curl -s "$BASE_URL/webhooks/$WEBHOOK_ID/deliveries?status=pending" \
  -H "Authorization: Bearer $TOKEN" | jq '.data | length'
echo ""

test_header "List Delivered Deliveries"
curl -s "$BASE_URL/webhooks/$WEBHOOK_ID/deliveries?status=delivered" \
  -H "Authorization: Bearer $TOKEN" | jq '.data | length'
echo ""

# =============================================================================
section_header "8. Cleanup"
# =============================================================================

test_header "Delete Failing Webhook"
curl -s -X DELETE "$BASE_URL/webhooks/$FAIL_WEBHOOK_ID" \
  -H "Authorization: Bearer $TOKEN"
echo "Deleted webhook: $FAIL_WEBHOOK_ID"
echo ""

test_header "Delete Test Webhook"
curl -s -X DELETE "$BASE_URL/webhooks/$WEBHOOK_ID" \
  -H "Authorization: Bearer $TOKEN"
echo "Deleted webhook: $WEBHOOK_ID"
echo ""

# =============================================================================
section_header "Test Complete!"
# =============================================================================

echo "Summary:"
echo "  - Tested integration status endpoints"
echo "  - Created and managed webhook endpoints"
echo "  - Tested webhook delivery and retry"
echo "  - Demonstrated circuit breaker operations"
echo ""
echo "Integration Patterns Demonstrated:"
echo "  1. Circuit Breaker - Prevents cascading failures"
echo "  2. Retry with Backoff - Handles transient failures"
echo "  3. Webhook Reliability - Dead letter queue, retry"
echo "  4. Health Monitoring - Status and metrics endpoints"
echo ""
echo "Events that would be published:"
echo "  - WebhookDeliveryFailed → Slack: #webhook-alerts"
echo "  - IntegrationCircuitOpened → Slack + monitoring webhook"
echo "  - IntegrationCircuitClosed → Logged"

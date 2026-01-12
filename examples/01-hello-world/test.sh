#!/bin/bash
# =============================================================================
# Hello World API Test Script
# =============================================================================
# Run this script to test the Hello World API endpoints
# Usage: ./test.sh [BASE_URL] [TOKEN]
# =============================================================================

BASE_URL="${1:-http://localhost:8080}"
TOKEN="${2:-your-jwt-token-here}"

echo "============================================="
echo "Hello World API Test Script"
echo "Base URL: $BASE_URL"
echo "============================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print test header
test_header() {
    echo -e "${BLUE}>>> $1${NC}"
}

# Function to print result
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ Success${NC}"
    else
        echo -e "${RED}✗ Failed${NC}"
    fi
    echo ""
}

# -----------------------------------------------------------------------------
# Test 1: Create a Task
# -----------------------------------------------------------------------------
test_header "Test 1: Create a Task"
TASK_RESPONSE=$(curl -s -X POST "$BASE_URL/tasks" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "title": "Learn CodeAI DSL",
    "description": "Complete the hello world tutorial and understand basic DSL syntax",
    "priority": 1,
    "due_date": "2026-01-20"
  }')

echo "$TASK_RESPONSE" | jq .
TASK_ID=$(echo "$TASK_RESPONSE" | jq -r '.id')
echo "Created Task ID: $TASK_ID"
print_result $?

# -----------------------------------------------------------------------------
# Test 2: List All Tasks
# -----------------------------------------------------------------------------
test_header "Test 2: List All Tasks"
curl -s "$BASE_URL/tasks" | jq .
print_result $?

# -----------------------------------------------------------------------------
# Test 3: Get Task by ID
# -----------------------------------------------------------------------------
test_header "Test 3: Get Task by ID"
curl -s "$BASE_URL/tasks/$TASK_ID" | jq .
print_result $?

# -----------------------------------------------------------------------------
# Test 4: List Tasks with Filters
# -----------------------------------------------------------------------------
test_header "Test 4: List Tasks with Filters (status=pending)"
curl -s "$BASE_URL/tasks?status=pending" | jq .
print_result $?

# -----------------------------------------------------------------------------
# Test 5: Search Tasks
# -----------------------------------------------------------------------------
test_header "Test 5: Search Tasks (search=CodeAI)"
curl -s "$BASE_URL/tasks?search=CodeAI" | jq .
print_result $?

# -----------------------------------------------------------------------------
# Test 6: Update Task Status
# -----------------------------------------------------------------------------
test_header "Test 6: Update Task Status to in_progress"
curl -s -X PUT "$BASE_URL/tasks/$TASK_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "status": "in_progress",
    "priority": 2
  }' | jq .
print_result $?

# -----------------------------------------------------------------------------
# Test 7: Mark Task Complete
# -----------------------------------------------------------------------------
test_header "Test 7: Mark Task Complete"
curl -s -X POST "$BASE_URL/tasks/$TASK_ID/complete" \
  -H "Authorization: Bearer $TOKEN" | jq .
print_result $?

# -----------------------------------------------------------------------------
# Test 8: Verify Task is Completed
# -----------------------------------------------------------------------------
test_header "Test 8: Verify Task is Completed"
curl -s "$BASE_URL/tasks/$TASK_ID" | jq '.status, .completed, .completed_at'
print_result $?

# -----------------------------------------------------------------------------
# Test 9: Create Another Task
# -----------------------------------------------------------------------------
test_header "Test 9: Create Another Task"
TASK2_RESPONSE=$(curl -s -X POST "$BASE_URL/tasks" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "title": "Review Blog API Example",
    "description": "Look at entity relationships in the blog API example",
    "priority": 3
  }')
echo "$TASK2_RESPONSE" | jq .
TASK2_ID=$(echo "$TASK2_RESPONSE" | jq -r '.id')
print_result $?

# -----------------------------------------------------------------------------
# Test 10: List Tasks with Pagination
# -----------------------------------------------------------------------------
test_header "Test 10: List Tasks with Pagination"
curl -s "$BASE_URL/tasks?page=1&limit=1" | jq .
print_result $?

# -----------------------------------------------------------------------------
# Test 11: Delete a Task
# -----------------------------------------------------------------------------
test_header "Test 11: Delete Task"
curl -s -X DELETE "$BASE_URL/tasks/$TASK2_ID" \
  -H "Authorization: Bearer $TOKEN"
echo "Task deleted"
print_result $?

# -----------------------------------------------------------------------------
# Test 12: Verify Task is Deleted
# -----------------------------------------------------------------------------
test_header "Test 12: Verify Task is Deleted (should return 404)"
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/tasks/$TASK2_ID")
echo "HTTP Status: $HTTP_STATUS"
if [ "$HTTP_STATUS" -eq 404 ]; then
    echo -e "${GREEN}✓ Task correctly returns 404${NC}"
else
    echo -e "${RED}✗ Expected 404, got $HTTP_STATUS${NC}"
fi
echo ""

# -----------------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------------
echo "============================================="
echo "Test Complete!"
echo "============================================="
echo ""
echo "Tasks created:"
echo "  - Task 1 (completed): $TASK_ID"
echo "  - Task 2 (deleted): $TASK2_ID"
echo ""
echo "To clean up, delete the remaining task:"
echo "curl -X DELETE \"$BASE_URL/tasks/$TASK_ID\" -H \"Authorization: Bearer $TOKEN\""

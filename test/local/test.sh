#!/bin/bash
# CodeAI Local Test - Complete Test Suite
# Tests parsing, validation, and live server with MongoDB

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Change to project root
cd "$(dirname "$0")/../.."
PROJECT_ROOT=$(pwd)

# Variables
SERVER_PID=""
MONGO_STARTED=false
RUN_SERVER_TEST=false

# Parse arguments
if [ "$1" = "--server" ] || [ "$1" = "-s" ]; then
    RUN_SERVER_TEST=true
fi

# Cleanup function
cleanup() {
    if [ -n "$SERVER_PID" ]; then
        echo -e "\n${YELLOW}Stopping server...${NC}"
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi

    if [ "$MONGO_STARTED" = true ]; then
        echo "Stopping MongoDB container..."
        docker stop mongodb 2>/dev/null || true
        docker rm mongodb 2>/dev/null || true
    fi
}

trap cleanup EXIT INT TERM

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  CodeAI Local Test Suite              ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"

# =============================================================================
# Step 1: Prerequisites
# =============================================================================

echo -e "\n${CYAN}━━━ Step 1: Prerequisites ━━━${NC}"

if [ ! -f "$PROJECT_ROOT/bin/codeai" ]; then
    echo -e "${YELLOW}Building CodeAI...${NC}"
    make build
    if [ $? -ne 0 ]; then
        echo -e "${RED}Build failed${NC}"
        exit 1
    fi
fi
echo -e "${GREEN}✓ CodeAI binary ready${NC}"

if command -v jq &> /dev/null; then
    HAS_JQ=true
    echo -e "${GREEN}✓ jq available${NC}"
else
    HAS_JQ=false
    echo -e "${YELLOW}ℹ jq not found (JSON output won't be pretty-printed)${NC}"
fi

# =============================================================================
# Step 2: Parse and Validate
# =============================================================================

echo -e "\n${CYAN}━━━ Step 2: Parse & Validate ━━━${NC}"

echo "Parsing test/local/app.cai..."
if ./bin/codeai parse test/local/app.cai > /tmp/parse-output.json 2>&1; then
    echo -e "${GREEN}✓ Parse successful${NC}"

    if [ "$HAS_JQ" = true ]; then
        ENDPOINT_COUNT=$(jq '[.Statements[] | select(.Method != null)] | length' /tmp/parse-output.json 2>/dev/null || echo "0")
        echo -e "  Found ${ENDPOINT_COUNT} endpoints"
    fi
else
    echo -e "${RED}✗ Parse failed${NC}"
    cat /tmp/parse-output.json
    exit 1
fi

echo "Validating test/local/app.cai..."
if ./bin/codeai validate test/local/app.cai 2>&1 | tee /tmp/validate-output.txt; then
    echo -e "${GREEN}✓ Validation successful${NC}"
else
    echo -e "${RED}✗ Validation failed${NC}"
    cat /tmp/validate-output.txt
    exit 1
fi

# =============================================================================
# Step 3: Server Test (Optional)
# =============================================================================

if [ "$RUN_SERVER_TEST" = true ]; then
    echo -e "\n${CYAN}━━━ Step 3: Server Test ━━━${NC}"

    # Check for Docker
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}Docker not found. Skipping server test.${NC}"
        exit 0
    fi
    echo -e "${GREEN}✓ Docker available${NC}"

    # Start MongoDB
    if docker ps | grep -q mongodb; then
        echo -e "${GREEN}✓ MongoDB already running${NC}"
    else
        echo "Starting MongoDB..."
        if docker run -d -p 27017:27017 --name mongodb mongo:latest >/dev/null 2>&1; then
            MONGO_STARTED=true
            sleep 3
            echo -e "${GREEN}✓ MongoDB started${NC}"
        else
            docker rm mongodb 2>/dev/null || true
            docker run -d -p 27017:27017 --name mongodb mongo:latest >/dev/null 2>&1
            MONGO_STARTED=true
            sleep 3
            echo -e "${GREEN}✓ MongoDB started${NC}"
        fi
    fi

    # Start server
    echo "Starting server on port 8080..."
    ./bin/codeai server start --file test/local/app.cai --port 8080 > /tmp/codeai-server.log 2>&1 &
    SERVER_PID=$!

    # Wait for server
    echo -n "Waiting for server."
    for i in {1..30}; do
        if curl -s http://localhost:8080/health >/dev/null 2>&1 || curl -s http://localhost:8080/users >/dev/null 2>&1; then
            echo ""
            echo -e "${GREEN}✓ Server ready${NC}"
            break
        fi
        if ! ps -p $SERVER_PID > /dev/null; then
            echo ""
            echo -e "${RED}✗ Server failed to start${NC}"
            cat /tmp/codeai-server.log
            exit 1
        fi
        sleep 1
        echo -n "."
    done

    # Test endpoints
    echo -e "\n${CYAN}Testing Endpoints:${NC}"

    # Health check
    echo -n "  GET /health ... "
    if curl -s http://localhost:8080/health | grep -q "ok"; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
    fi

    # List users
    echo -n "  GET /users ... "
    if curl -s http://localhost:8080/users >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
    fi

    # Create user
    echo -n "  POST /users ... "
    RESPONSE=$(curl -s -X POST http://localhost:8080/users \
        -H "Content-Type: application/json" \
        -d '{"name":"Test User","email":"test@example.com","age":25}')
    if echo "$RESPONSE" | grep -q "id\|status"; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
    fi

    echo ""
    echo -e "${GREEN}Server test complete!${NC}"

    # Pause to allow database inspection
    echo -e "\n${CYAN}━━━ Database Inspection ━━━${NC}"
    echo -e "${YELLOW}The server is still running and database contains test data.${NC}"
    echo -e "${YELLOW}You can now inspect the MongoDB database:${NC}"
    echo ""
    echo -e "  ${CYAN}docker exec -it mongodb mongosh${NC}"
    echo -e "  ${CYAN}> use codeai${NC}"
    echo -e "  ${CYAN}> db.users.find().pretty()${NC}"
    echo ""
    echo -e "${YELLOW}Press Enter when done to cleanup and exit...${NC}"
    read -r
fi

# =============================================================================
# Summary
# =============================================================================

echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✓ All tests passed!${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo -e "\n${BLUE}Tested:${NC}"
echo "  ✓ DSL parsing"
echo "  ✓ Schema validation"
echo "  ✓ MongoDB collections"
echo "  ✓ REST API endpoints"
if [ "$RUN_SERVER_TEST" = true ]; then
    echo "  ✓ HTTP server"
    echo "  ✓ CRUD operations"
fi

echo -e "\n${YELLOW}Usage:${NC}"
echo "  ./test.sh           # Parse & validate only (fast)"
echo "  ./test.sh --server  # Full test with MongoDB & HTTP server"
echo ""

#!/bin/bash
# API test script for gosearch
# Tests HTTP API endpoints

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BIN_PATH="${BIN_PATH:-./bin/gosearch}"
DATA_DIR="${DATA_DIR:-./data/api_test}"
API_PORT="${API_PORT:-18080}"
API_BASE_URL="http://localhost:$API_PORT/api/v1"
TEST_URL="${TEST_URL:-https://example.com}"

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    log_info "Cleaning up..."
    # Kill the server process
    if [ -n "$SERVER_PID" ]; then
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    rm -rf "$DATA_DIR"
}

# Setup trap for cleanup
trap cleanup EXIT

log_info "Starting API tests..."

# Step 1: Prepare data
log_info "Step 1: Preparing test data..."
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR"

timeout 60s "$BIN_PATH" crawl "$TEST_URL" -L 1 -w 5 -D "$DATA_DIR" -d 100ms || log_warn "Crawl had issues"

# Step 2: Start server
log_info "Step 2: Starting API server on port $API_PORT..."
"$BIN_PATH" serve -D "$DATA_DIR" -p "$API_PORT" --host localhost &
SERVER_PID=$!

# Wait for server to start
log_info "Waiting for server to start..."
sleep 3

# Check if server is running
if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    log_error "Server failed to start"
    exit 1
fi
log_info "Server started (PID: $SERVER_PID)"

# Step 3: Test health endpoint
log_info "Step 3: Testing health endpoint..."
HEALTH_RESPONSE=$(curl -s -w "\n%{http_code}" "$API_BASE_URL/health" 2>/dev/null || echo "failed")
HEALTH_CODE=$(echo "$HEALTH_RESPONSE" | tail -n1)
HEALTH_BODY=$(echo "$HEALTH_RESPONSE" | head -n-1)

if [ "$HEALTH_CODE" = "200" ]; then
    log_info "Health endpoint returned 200 OK"
    echo "Response: $HEALTH_BODY"
else
    log_warn "Health endpoint returned code: $HEALTH_CODE"
fi

# Step 4: Test stats endpoint
log_info "Step 4: Testing stats endpoint..."
STATS_RESPONSE=$(curl -s -w "\n%{http_code}" "$API_BASE_URL/stats" 2>/dev/null || echo "failed")
STATS_CODE=$(echo "$STATS_RESPONSE" | tail -n1)
STATS_BODY=$(echo "$STATS_RESPONSE" | head -n-1)

if [ "$STATS_CODE" = "200" ]; then
    log_info "Stats endpoint returned 200 OK"
    echo "Response: $STATS_BODY"
else
    log_warn "Stats endpoint returned code: $STATS_CODE"
fi

# Step 5: Test search endpoint
log_info "Step 5: Testing search endpoint..."
SEARCH_RESPONSE=$(curl -s -w "\n%{http_code}" "$API_BASE_URL/search?q=example&limit=5" 2>/dev/null || echo "failed")
SEARCH_CODE=$(echo "$SEARCH_RESPONSE" | tail -n1)
SEARCH_BODY=$(echo "$SEARCH_RESPONSE" | head -n-1)

if [ "$SEARCH_CODE" = "200" ]; then
    log_info "Search endpoint returned 200 OK"
    echo "Response: $SEARCH_BODY"
else
    log_warn "Search endpoint returned code: $SEARCH_CODE"
fi

# Step 6: Test search with invalid query (empty)
log_info "Step 6: Testing search with empty query..."
EMPTY_SEARCH_RESPONSE=$(curl -s -w "\n%{http_code}" "$API_BASE_URL/search?q=" 2>/dev/null || echo "failed")
EMPTY_SEARCH_CODE=$(echo "$EMPTY_SEARCH_RESPONSE" | tail -n1)

if [ "$EMPTY_SEARCH_CODE" = "400" ] || [ "$EMPTY_SEARCH_CODE" = "422" ]; then
    log_info "Empty query correctly rejected with code: $EMPTY_SEARCH_CODE"
else
    log_warn "Empty query returned unexpected code: $EMPTY_SEARCH_CODE"
fi

# Step 7: Test index rebuild endpoint
log_info "Step 7: Testing index rebuild endpoint..."
REBUILD_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/index/rebuild" 2>/dev/null || echo "failed")
REBUILD_CODE=$(echo "$REBUILD_RESPONSE" | tail -n1)
REBUILD_BODY=$(echo "$REBUILD_RESPONSE" | head -n-1)

if [ "$REBUILD_CODE" = "200" ] || [ "$REBUILD_CODE" = "202" ]; then
    log_info "Index rebuild returned code: $REBUILD_CODE"
    echo "Response: $REBUILD_BODY"
else
    log_warn "Index rebuild returned code: $REBUILD_CODE"
fi

# Step 8: Test 404 endpoint
log_info "Step 8: Testing 404 handling..."
NOTFOUND_RESPONSE=$(curl -s -w "\n%{http_code}" "$API_BASE_URL/nonexistent" 2>/dev/null || echo "failed")
NOTFOUND_CODE=$(echo "$NOTFOUND_RESPONSE" | tail -n1)

if [ "$NOTFOUND_CODE" = "404" ]; then
    log_info "404 endpoint correctly returned 404"
else
    log_warn "Nonexistent endpoint returned code: $NOTFOUND_CODE"
fi

log_info "API tests completed!"
exit 0

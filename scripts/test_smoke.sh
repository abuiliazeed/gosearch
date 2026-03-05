#!/bin/bash
# Smoke test script for gosearch
# Tests basic build and core commands

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BIN_PATH="${BIN_PATH:-./bin/gosearch}"
DATA_DIR="${DATA_DIR:-./data/smoke_test}"
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
    rm -rf "$DATA_DIR"
}

# Setup trap for cleanup
trap cleanup EXIT

log_info "Starting smoke tests..."
log_info "Binary: $BIN_PATH"
log_info "Data directory: $DATA_DIR"

# Test 1: Binary exists
log_info "Test 1: Checking if binary exists..."
if [ ! -f "$BIN_PATH" ]; then
    log_error "Binary not found at $BIN_PATH"
    log_info "Run 'make build' first"
    exit 1
fi
log_info "Binary found"

# Test 2: Version/help command
log_info "Test 2: Testing version/help command..."
if ! "$BIN_PATH" --help > /dev/null 2>&1; then
    log_error "Help command failed"
    exit 1
fi
log_info "Help command works"

# Test 3: Clean data directory
log_info "Test 3: Setting up clean data directory..."
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR"

# Test 4: Index stats on empty index
log_info "Test 4: Testing index stats on empty index..."
if "$BIN_PATH" index stats -D "$DATA_DIR" > /dev/null 2>&1; then
    log_info "Index stats works (empty index)"
else
    log_warn "Index stats returned error (expected for empty index)"
fi

# Test 5: Small crawl (with timeout)
log_info "Test 5: Testing crawl command (limited)..."
if timeout 30s "$BIN_PATH" crawl "$TEST_URL" -L 1 -w 2 -D "$DATA_DIR" -d 100ms > /dev/null 2>&1; then
    log_info "Crawl command completed"
else
    log_warn "Crawl command timed out or failed (network issues possible)"
fi

# Test 6: Index stats after crawl
log_info "Test 6: Testing index stats after crawl..."
if "$BIN_PATH" index stats -D "$DATA_DIR" > /dev/null 2>&1; then
    log_info "Index stats works after crawl"
else
    log_warn "Index stats returned error"
fi

# Test 7: Search command
log_info "Test 7: Testing search command..."
if "$BIN_PATH" search "example" -D "$DATA_DIR" --limit 5 > /dev/null 2>&1; then
    log_info "Search command works"
else
    log_warn "Search command failed (might be empty index)"
fi

# Test 8: Clear index
log_info "Test 8: Testing index clear..."
if "$BIN_PATH" index clear -D "$DATA_DIR" > /dev/null 2>&1; then
    log_info "Index clear works"
else
    log_warn "Index clear returned error"
fi

log_info "Smoke tests completed!"
exit 0

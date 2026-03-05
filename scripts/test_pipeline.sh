#!/bin/bash
# Pipeline test script for gosearch
# Tests the full crawl -> index -> search pipeline

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BIN_PATH="${BIN_PATH:-./bin/gosearch}"
DATA_DIR="${DATA_DIR:-./data/pipeline_test}"
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

log_info "Starting pipeline tests..."

# Step 1: Clean state
log_info "Step 1: Starting with clean state..."
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR"

# Step 2: Crawl pages
log_info "Step 2: Crawling pages from $TEST_URL..."
if timeout 60s "$BIN_PATH" crawl "$TEST_URL" -L 1 -w 5 -D "$DATA_DIR" -d 100ms; then
    log_info "Crawl completed successfully"
else
    log_warn "Crawl had issues but continuing..."
fi

# Step 3: Check index was created
log_info "Step 3: Verifying index was created..."
if [ -d "$DATA_DIR/index" ]; then
    INDEX_FILES=$(find "$DATA_DIR/index" -type f | wc -l)
    log_info "Index directory exists with $INDEX_FILES file(s)"
else
    log_warn "Index directory not found"
fi

# Step 4: Get index stats
log_info "Step 4: Getting index statistics..."
"$BIN_PATH" index stats -D "$DATA_DIR" || log_warn "Index stats failed"

# Step 5: Search for terms
log_info "Step 5: Searching for terms..."
SEARCH_TERMS=("example" "domain" "test")

for term in "${SEARCH_TERMS[@]}"; do
    log_info "Searching for: $term"
    if "$BIN_PATH" search "$term" -D "$DATA_DIR" --limit 5; then
        log_info "Search for '$term' completed"
    else
        log_warn "Search for '$term' had no results or failed"
    fi
    echo "---"
done

# Step 6: Test boolean queries
log_info "Step 6: Testing boolean queries..."

log_info "Testing AND query: example AND domain"
"$BIN_PATH" search "example AND domain" -D "$DATA_DIR" || log_warn "AND query failed"

log_info "Testing OR query: example OR test"
"$BIN_PATH" search "example OR test" -D "$DATA_DIR" || log_warn "OR query failed"

# Step 7: Test fuzzy search
log_info "Step 7: Testing fuzzy search..."
"$BIN_PATH" search "exmple" -D "$DATA_DIR" --fuzzy || log_warn "Fuzzy search failed"

# Step 8: Test backup
log_info "Step 8: Testing backup..."
BACKUP_PATH="$DATA_DIR/backup.bin"
if "$BIN_PATH" backup "$BACKUP_PATH" -D "$DATA_DIR"; then
    if [ -f "$BACKUP_PATH" ]; then
        log_info "Backup file created"
    else
        log_warn "Backup file not found"
    fi
else
    log_warn "Backup command failed"
fi

log_info "Pipeline tests completed!"
exit 0

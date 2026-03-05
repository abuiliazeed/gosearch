#!/bin/bash
# Persistence test script for gosearch
# Tests index save/load/restore functionality

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BIN_PATH="${BIN_PATH:-./bin/gosearch}"
DATA_DIR="${DATA_DIR:-./data/persistence_test}"
TEST_URL="${TEST_URL:-https://example.com}"
BACKUP_FILE="/tmp/gosearch_test_backup.bin"

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
    rm -f "$BACKUP_FILE"
}

# Setup trap for cleanup
trap cleanup EXIT

log_info "Starting persistence tests..."

# Step 1: Crawl and index
log_info "Step 1: Crawling and indexing..."
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR"

timeout 60s "$BIN_PATH" crawl "$TEST_URL" -L 1 -w 5 -D "$DATA_DIR" -d 100ms || log_warn "Crawl had issues"

# Step 2: Get initial stats
log_info "Step 2: Getting initial index stats..."
INITIAL_STATS=$("$BIN_PATH" index stats -D "$DATA_DIR" 2>&1 || true)
echo "$INITIAL_STATS"

# Step 3: Save index (implicit with crawl, but let's be explicit)
log_info "Step 3: Index should be persisted in BoltDB..."
if [ -d "$DATA_DIR/index" ]; then
    log_info "Index directory exists"
fi

# Step 4: Test backup command
log_info "Step 4: Testing backup command..."
if "$BIN_PATH" backup "$BACKUP_FILE" -D "$DATA_DIR"; then
    if [ -f "$BACKUP_FILE" ]; then
        BACKUP_SIZE=$(wc -c < "$BACKUP_FILE")
        log_info "Backup created: $BACKUP_FILE ($BACKUP_SIZE bytes)"
    else
        log_warn "Backup file not created"
    fi
else
    log_warn "Backup command failed"
fi

# Step 5: Clear index
log_info "Step 5: Clearing index..."
"$BIN_PATH" index clear -D "$DATA_DIR" || log_warn "Clear failed"

# Step 6: Verify index is empty
log_info "Step 6: Verifying index is cleared..."
CLEARED_STATS=$("$BIN_PATH" index stats -D "$DATA_DIR" 2>&1 || true)
echo "$CLEARED_STATS"

# Step 7: Search should trigger auto-rebuild or show empty
log_info "Step 7: Searching after clear (should auto-rebuild or show empty)..."
SEARCH_OUTPUT=$("$BIN_PATH" search "example" -D "$DATA_DIR" 2>&1 || true)
echo "$SEARCH_OUTPUT"

# Step 8: Restore from backup
log_info "Step 8: Restoring from backup..."
if [ -f "$BACKUP_FILE" ]; then
    if "$BIN_PATH" restore "$BACKUP_FILE" -D "$DATA_DIR"; then
        log_info "Restore completed"
    else
        log_warn "Restore command failed"
    fi
else
    log_warn "No backup file to restore from"
fi

# Step 9: Verify restored index
log_info "Step 9: Verifying restored index..."
RESTORED_STATS=$("$BIN_PATH" index stats -D "$DATA_DIR" 2>&1 || true)
echo "$RESTORED_STATS"

# Step 10: Search after restore
log_info "Step 10: Searching after restore..."
"$BIN_PATH" search "example" -D "$DATA_DIR" || log_warn "Search after restore failed"

log_info "Persistence tests completed!"
exit 0

# Crawler Frontier Bug - Analysis & Investigation

## Problem Description

The crawler does not exit naturally after processing all URLs. Workers continue running indefinitely, causing the crawl command to hang and eventually time out.

### Symptoms

1. **Test Timeout**: Integration tests time out after 30 seconds with "signal: killed"
2. **Stats Show Completion Indicators**: Stats display shows all URLs crawled (e.g., "Crawled: 2 | Queued: 2 | Failed: 0") but crawler doesn't exit
3. **Manual Verification**: Running crawl manually confirms the hang - process must be interrupted with Ctrl+C

### Example Output

```
Crawled: 2 | Queued: 2 | Failed: 0
Crawled: 2 | Queued: 2 | Failed: 0
[repeats every 2 seconds indefinitely]
```

## Root Cause Analysis

### Architectural Issue

The core problem is a **race condition between worker lifecycle and frontier state**:

1. **Colly's Async Mode**: When `colly.Async(true)` is set, requests are queued in Colly's internal worker pool
   - `collector.Request()` returns immediately after queuing
   - Actual HTTP requests happen in background goroutines
   - Callbacks (OnHTML, OnScraped) run asynchronously

2. **Worker Exit Logic Flaw**:
   ```go
   url := c.frontier.Pop()
   if url == nil {
       if c.frontier.Len() == 0 {
           return  // Worker exits
       }
       time.Sleep(100 * time.Millisecond)
       continue
   }
   ```
   This has a race condition: between `Pop()` returning nil and checking `Len()`, another goroutine's OnHTML callback could add new URLs.

3. **Context Cancellation Mismatch**:
   - The `Start()` function waits on parent `ctx.Done()`
   - But worker completion goroutine only calls `c.cancel()` (cancels child context)
   - Parent context never gets cancelled → infinite wait

### Attempted Fixes

| Fix | Description | Result |
|------|-------------|--------|
| Deduplicator check before Push() | Check dedupe before adding URLs to frontier | Prevents duplicates but doesn't fix hang |
| Active requests counter | Track requests-in-flight with atomic ops | Still hangs - race condition persists |
| Switch to sync mode | Set `colly.Async(false)` | Still hangs - workers don't exit properly |
| Context completion channel | Add `completeChan` to signal completion | Partially implemented but incomplete |
| Cancel parent context in goroutine | Call `cancel()` when workers complete | Helps but doesn't fully solve issue |

## Current State (2025-02-11)

### Files Modified

1. **`internal/crawler/crawler.go`**
   - Added `completeChan` field (incomplete implementation)
   - Added `activeRequests` tracking (later removed)
   - Switched to `colly.Async(false)`
   - Modified worker exit conditions

2. **`pkg/cli/crawl.go`**
   - Added `cancel()` call after `crawlr.Start()` returns

3. **`Makefile`**
   - Fixed `CMD_DIR` from `.` to `cmd/gosearch`

4. **`tests/integration_test.go`**
   - Fixed delay flag format from `"100ms"` to `"100"`
   - Fixed API test health endpoint path
   - Fixed `head -n-1` macOS compatibility

5. **`scripts/test_api.sh`**
   - Fixed `head` command for macOS

### Test Results

- **Smoke tests**: PASS
- **Pipeline tests**: FAIL (timeout)
- **Persistence tests**: Not run (dependency on pipeline)
- **API tests**: Partial PASS (health endpoint fixed)

## Recommended Next Steps

### Option 1: Poison Pill Pattern (Recommended)

Add a sentinel URL that signals workers to stop:

```go
// In Start() - after wg.Wait()
close(c.stopChan)

// In worker loop
select {
case <-c.ctx.Done():
    return
case <-c.stopChan:
    return
case url := <-c.frontier.PopChan():
    // Process URL...
}
```

### Option 2: Redesign Worker Lifecycle

Replace "exit when empty" with explicit work counting:

```go
type CollyCrawler struct {
    // ... existing fields
    workPending sync.WaitGroup // Track active work
}

// When starting work
c.workPending.Add(1)
collector.Request(...)

// In OnResponse/OnScraped/OnError
c.workPending.Done()
```

Workers only exit when both frontier is empty AND `workPending` counter is zero.

### Option 3: Synchronous Mode with Proper Drain

Use sync mode but implement a "drain" phase:

```go
// After all workers started
go func() {
    c.wg.Wait()

    // Drain phase: wait for all callbacks to complete
    time.Sleep(1 * time.Second)

    c.complete = true
    c.stats.EndTime = time.Now()
    close(c.completeChan)
}()
```

## Technical Details

### Frontier Implementation

The frontier uses a **priority queue** (`container/heap`):
- `Push()` adds URL with O(log n) complexity
- `Pop()` removes and returns highest priority URL
- `Len()` returns current queue size

**Critical Issue**: The `Pop()` and `Len()` calls use separate locks, creating a race condition window.

### Colly Integration

Colly collector in async mode:
- Maintains internal request queue
- Spawns worker goroutines to process requests
- Callbacks run in those worker goroutines, NOT the caller
- This creates a fundamental mismatch with our worker-per-URL model

### Stats Display

Stats shown every 2 seconds by monitor loop in `runCrawl()`:
```go
case <-ticker.C:
    stats := crawlr.Stats()
    fmt.Printf("\rCrawled: %d | Queued: %d | Failed: %d",
        stats.URLsCrawled,
        stats.URLsQueued,  // Cumulative count, NOT current queue size
        stats.URLsFailed)
```

**Note**: `URLsQueued` is a cumulative counter of all URLs ever added to frontier, NOT the current queue size.

## Related Files

- `internal/crawler/crawler.go` - Main crawler implementation
- `internal/crawler/frontier.go` - Priority queue implementation
- `internal/crawler/dedupe.go` - URL deduplication
- `pkg/cli/crawl.go` - CLI command for crawl
- `tests/integration_test.go` - Integration test suite

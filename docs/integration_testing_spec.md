# Feature Spec: Integration Testing

> **Status:** Draft
> **Author:** Claude Code
> **Date:** 2026-02-11
> **Sprint:** Integration Testing (Post-MVP)

---

## User Story

**As a** developer working on gosearch,
**I want** comprehensive integration tests for the full pipeline,
**So that** I can verify all components work correctly together and catch regressions early.

---

## Acceptance Criteria

Each criterion must be **specific and testable**:

1. [ ] **Full Pipeline Test**: Crawl command successfully crawls pages, indexes them, and search returns results
2. [ ] **Persistence Test**: Index saved to BoltDB can be loaded after restart and searches work correctly
3. [ ] **Fuzzy Matching Test**: Search with typos returns relevant results with suggestions
4. [ ] **Boolean Query Test**: AND, OR, NOT queries return correct result sets
5. [ ] **Backup/Restore Test**: Index backup to file can be restored and searches work
6. [ ] **HTTP API Test**: All API endpoints (search, stats, health, index) return valid responses
7. [ ] **Block Detection Test**: Crawler properly detects and backs off from 403/429 responses
8. [ ] **Cookie Persistence Test**: Cookies persist across crawl requests
9. [ ] **Edge Cases**: Empty index, empty query, invalid URL handled gracefully
10. [ ] **Build Verification**: `go build ./...` succeeds without errors
11. [ ] **Smoke Test**: Basic crawl and search commands work without panics

---

## Test Strategy

### Test Level: Minimal (per CLAUDE.md)

| Test Type | Purpose | Implementation |
|-----------|---------|----------------|
| **Build** | Verify code compiles | `go build ./...` |
| **Smoke** | Core commands work | Manual integration tests |
| **Integration** | Component interaction | Go test scripts |

### Test Environments

- **Local**: Developer's machine with Redis running
- **Isolated**: Temporary data directories for each test run
- **Mock URLs**: Use local test server or example.com for crawling

---

## Component Breakdown

```
Integration Tests (tests/)
├── integration_test.go      — Main integration test suite
├── test_helpers.go          — Test utilities and fixtures
└── testdata/               — Test fixtures
    ├── sample_html.html     — Sample page for crawling
    └── expected_results.json — Expected search results

Test Scripts (scripts/)
├── test_pipeline.sh        — Full pipeline test script
├── test_persistence.sh    — Index persistence test script
├── test_api.sh            — HTTP API test script
└── test_smoke.sh          — Basic smoke test script
```

---

## Test Scenarios

### 1. Full Pipeline Test (Crawl → Index → Search)

```bash
# 1. Start with clean state
rm -rf ./data

# 2. Crawl a small set of pages
./bin/gosearch crawl https://example.com -L 1 -w 2

# 3. Verify index was created
./bin/gosearch index stats

# 4. Search for terms
./bin/gosearch search "example"

# 5. Verify results
```

**Expected Results:**
- Crawl completes without errors
- Index stats show documents indexed
- Search returns relevant results
- No panics or crashes

### 2. Persistence Test (Save/Load Index)

```bash
# 1. Crawl and index pages
./bin/gosearch crawl https://example.com -L 1

# 2. Save index
./bin/gosearch index save

# 3. Clear index from memory
./bin/gosearch index clear

# 4. Search should trigger auto-rebuild
./bin/gosearch search "example"

# 5. Verify results match original
```

**Expected Results:**
- Index persists across restarts
- Search after rebuild returns same results
- No data loss

### 3. Fuzzy Matching Test

```bash
# 1. Crawl pages with known terms
./bin/gosearch crawl https://example.com -L 1

# 2. Search with typos
./bin/gosearch search "exmple" --fuzzy

# 3. Verify suggestions
```

**Expected Results:**
- Fuzzy search finds close matches
- Suggestions displayed for typos
- Jaro-Winkler similarity works correctly

### 4. Boolean Query Test

```bash
# 1. Crawl diverse content
./bin/gosearch crawl https://example.com -L 1

# 2. Test AND query
./bin/gosearch search "term1 AND term2"

# 3. Test OR query
./bin/gosearch search "term1 OR term2"

# 4. Test NOT query
./bin/gosearch search "term1 NOT term2"
```

**Expected Results:**
- AND returns only docs with both terms
- OR returns docs with either term
- NOT excludes docs with the term

### 5. Backup/Restore Test

```bash
# 1. Crawl and index
./bin/gosearch crawl https://example.com -L 1

# 2. Backup index
./bin/gosearch backup /tmp/index_backup.bin

# 3. Clear index
./bin/gosearch index clear

# 4. Restore index
./bin/gosearch restore /tmp/index_backup.bin

# 5. Search works again
./bin/gosearch search "example"
```

**Expected Results:**
- Backup file created successfully
- Restore succeeds without errors
- Search after restore returns same results

### 6. HTTP API Test

```bash
# 1. Start server
./bin/gosearch serve &

# 2. Test health endpoint
curl http://localhost:8080/api/v1/health

# 3. Test search endpoint
curl "http://localhost:8080/api/v1/search?q=example"

# 4. Test stats endpoint
curl http://localhost:8080/api/v1/stats

# 5. Test index rebuild endpoint
curl -X POST http://localhost:8080/api/v1/index/rebuild
```

**Expected Results:**
- All endpoints respond with 200 status
- JSON responses are valid
- Search endpoint returns results
- Server handles graceful shutdown

### 7. Block Detection Test

**Requirements:** Mock HTTP server returning 429 status

```bash
# Start mock server that returns 429
./scripts/mock_block_server.sh &

# Crawl the mock server
./bin/gosearch crawl http://localhost:9999 -L 1
```

**Expected Results:**
- Crawler detects 429 status
- Exponential backoff is applied
- Retry attempts are logged
- No infinite loops

### 8. Edge Cases Test

```bash
# 1. Search with empty index
./bin/gosearch index clear
./bin/gosearch search "test"
# Expected: "index is empty" message

# 2. Search with empty query
./bin/gosearch search ""
# Expected: "query cannot be empty" error

# 3. Crawl invalid URL
./bin/gosearch crawl "not-a-url"
# Expected: Invalid URL error

# 4. Backup non-existent index
./bin/gosearch index clear
./bin/gosearch backup /tmp/backup.bin
# Expected: "index is empty" error
```

---

## Files to Create/Modify

### New Files

| File | Purpose |
|------|---------|
| `tests/integration_test.go` | Go integration test suite |
| `tests/test_helpers.go` | Test utilities and fixtures |
| `scripts/test_pipeline.sh` | Full pipeline test script |
| `scripts/test_persistence.sh` | Persistence test script |
| `scripts/test_api.sh` | HTTP API test script |
| `scripts/test_smoke.sh` | Basic smoke test script |
| `scripts/mock_block_server.sh` | Mock server for block detection test |
| `tests/testdata/sample_html.html` | Sample HTML for tests |

### Modified Files

| File | Changes |
|------|---------|
| `Makefile` | Add test targets (test-integration, test-smoke, test-all) |
| `go.mod` | No new dependencies needed |
| `STATE.md` | Update with testing progress |
| `ROADMAP.md` | Mark testing phase as in progress |

---

## Testing Plan

| Test Script | What It Tests | Command |
|-------------|---------------|---------|
| `test_smoke.sh` | Build + basic commands | `make test-smoke` |
| `test_pipeline.sh` | Full crawl → search pipeline | `make test-pipeline` |
| `test_persistence.sh` | Index save/load/restore | `make test-persistence` |
| `test_api.sh` | HTTP API endpoints | `make test-api` |
| `integration_test.go` | All integration scenarios | `go test ./tests/...` |

---

## Performance Considerations

- **Test isolation**: Each test uses temporary data directory
- **Test speed**: Mock URLs for faster tests (no network dependency)
- **Parallel tests**: Tests can run in parallel if isolated
- **Resource cleanup**: Clean up data directories after tests

---

## Error Handling

| Scenario | Expected Behavior |
|----------|------------------|
| Test setup fails | Skip test with clear message |
| External dependency missing (Redis) | Skip dependent tests, log warning |
| Network timeout during crawl | Use shorter timeouts for tests |
| Port already in use (API test) | Use random port for API server |

---

## Open Questions

- [ ] Should we add unit tests for individual modules? (Outside scope for now)
- [ ] Should we use a test HTTP server library or shell scripts for API tests?
- [ ] Should tests be automated in CI/CD pipeline?

---

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| None required | — | Using Go testing + bash scripts |

---

## Success Metrics

- All test scripts pass
- `go build ./...` succeeds
- Manual smoke test passes
- No regressions in existing functionality
- Test execution time < 5 minutes

---

_Review this spec against CLAUDE.md rules before implementation begins._

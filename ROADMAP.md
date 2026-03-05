# Roadmap — gosearch

## Current Sprint

### Phase 1: Foundation ✅ COMPLETE
- [x] Initialize Go module (`go mod init github.com/abuiliazeed/gosearch`)
- [x] Set up project directory structure (cmd, internal, pkg)
- [x] Install dependencies (Colly, Cobra, Viper, Redis client, BoltDB/Badger)
- [x] Create Makefile with build, run, test commands
- [x] Set up basic CLI skeleton with Cobra (root command)
- [x] Configure Viper for environment variable and config file loading
- [x] Set up logging (structured logging with levels)

### Phase 2: Core Components ✅ COMPLETE
- [x] **Crawler Module** (`internal/crawler`)
  - [x] Implement frontier queue (URLs to crawl)
  - [x] Implement Colly-based crawler with configurable workers
  - [x] Add politeness policies (rate limiting, robots.txt)
  - [x] Add URL deduplication (Bloom filter or Redis set)
  - [x] Implement HTML parsing and link extraction
  - [x] Save crawled pages to file storage

- [x] **Indexer Module** (`internal/indexer`)
  - [x] Implement tokenizer (Unicode-aware, stopword removal)
  - [x] Build inverted index data structures (postings lists)
  - [x] Implement gap encoding for compression
  - [x] Add positional index for phrase queries
  - [x] Persist index to BoltDB/Badger
  - [x] Implement index merge for updates

- [x] **Ranker Module** (`internal/ranker`)
  - [x] Implement TF-IDF scoring
  - [x] Implement PageRank computation (iterative)
  - [x] Combine scores with boost factors (title, URL, freshness)
  - [x] Cache frequently accessed scores

- [x] **Search Module** (`internal/search`)
  - [x] Implement query parser (tokenization, stemming)
  - [x] Add boolean query support (AND, OR, NOT)
  - [x] Implement fuzzy matching (Levenshtein distance)
  - [x] Add phrase query support ("exact match")
  - [x] Integrate ranking and sort results
  - [x] Add Redis query caching with TTL

### Phase 3: CLI & Storage ✅ COMPLETE
- [x] **CLI Commands** (`pkg/cli`)
  - [x] `crawl` command — Start crawling from seed URLs (with automatic indexing)
  - [x] `search` command — Search the index (with ranker integration and auto-rebuild)
  - [x] `index` command — Index management (build, rebuild, stats)
  - [x] `serve` command — HTTP API server
  - [x] `backup` command — Backup index to file
  - [x] `restore` command — Restore index from file

- [x] **Storage Layer** (`internal/storage`)
  - [x] Document store — File-based page storage with compression
  - [x] Index store — BoltDB/Badger for index metadata
  - [x] Enhanced persistence — Full postings lists and document info saved
  - [x] Cache store — Redis client wrapper for query cache

### Phase 4: Polish & Quality ✅ COMPLETE
- [x] Documentation created (2026-02-11)
  - [x] Seeding strategies research (`docs/seeding-strategies.md`)
  - [x] Anti-blocking strategies research (`docs/antiblocking-strategies.md`)
- [x] Enhanced browser headers — Sec-CH-UA, Sec-Fetch-* headers added
- [x] Session/cookie management — Cookie persistence across requests
- [x] Block detection — Detect 403/429/CAPTCHA responses with exponential backoff
- [x] Progress bars for long operations — progressbar/v3 wrapper created
- [x] Configuration file support — .gosearch.yaml.example provided
- [x] Comprehensive error messages — Detailed error types with suggestions
- [ ] Set up CI/CD pipeline (GitHub Actions)
- [ ] Add benchmark tests for critical paths
- [ ] Memory profiling and optimization

---

## Phase 5: Anti-Blocking Features ✅ COMPLETE

> Based on research in `docs/antiblocking-strategies.md`

### Priority 1: Essential (Implement Now)
- [x] robots.txt compliance — Already implemented with temoto/robotstxt
- [x] Rate limiting — Already implemented, add adaptive backoff
- [x] User-Agent headers — Add proper browser-like headers
- [x] Politeness delays — Already implemented
- [x] Enhanced browser headers — Add Sec-CH-UA, Sec-Fetch-* headers
- [x] Session/cookie management — Cookie persistence across requests

### Priority 2: Important (Add Soon)
- [x] Block detection — Detect 403/429/CAPTCHA responses
- [x] Exponential backoff — Adaptive delays after rate limits
- [x] Request header profiles — Multiple browser header templates (Chrome, Firefox, Safari, Edge)
- [x] Per-domain backoff tracking — Track blocked domains separately

### Priority 3: Advanced (Add When Needed)
- [ ] Proxy rotation support — For large-scale crawls
- [ ] TLS fingerprint mitigation — Use browser automation for protected sites
- [ ] CAPTCHA handling — Manual bypass or service integration
- [ ] Browser automation fallback — chromedp integration for JS-heavy sites

---

## Phase 6: Crawler Enhancements ✅ COMPLETE

> Based on research in `docs/seeding-strategies.md`

### Seed URL Management
- [x] Predefined seed sets — General, programming, academic categories
- [x] Seed set CLI flag — `--seed-set general|programming|academic`
- [x] Seed set configuration file — Custom seed URLs
- [ ] Domain diversity metrics — Ensure coverage across TLDs/regions (future)

### Crawling Strategy Improvements
- [x] BFS crawler option — Breadth-first exploration (default recommended)
- [ ] Best-first crawler option — Priority by relevance score (future)
- [ ] Crawl depth per domain — Configurable per-domain limits (future)
- [ ] Crawl budget management — Total pages/time limits (future)

### Seed Discovery
- [ ] Hub score calculation — Identify pages linking to many others
- [ ] Authority score calculation — Identify highly-linked pages
- [ ] Community detection — Find web communities for diverse coverage
- [ ] Topic-focused seeds — TF-IDF based seed selection

---

## Phase 7: Integration Testing 📋 In Progress

> Comprehensive integration tests for the full search pipeline

- [x] Test helpers and utilities — tests/test_helpers.go
- [x] Sample HTML fixture — tests/testdata/sample_html.html
- [x] Integration test suite — tests/integration_test.go
- [x] Smoke test script — scripts/test_smoke.sh
- [x] Pipeline test script — scripts/test_pipeline.sh
- [x] Persistence test script — scripts/test_persistence.sh
- [x] API test script — scripts/test_api.sh
- [x] Mock block server — scripts/mock_block_server.sh
- [x] Makefile test targets — test-smoke, test-pipeline, test-persistence, test-api, test-all
- [ ] Run and verify all tests
- [ ] Add CI/CD pipeline (GitHub Actions)
- [ ] Add benchmark tests for critical paths
- [ ] Memory profiling and optimization

---

## Backlog (Post-MVP)
- [ ] Distributed crawling (multiple workers, shared frontier)
- [ ] Incremental index updates (partial re-indexing)
- [ ] Query suggestions and autocomplete
- [ ] Result snippets highlighting matched terms
- [ ] Faceted search (filter by domain, date, content type)
- [ ] Spell correction using edit distance
- [ ] Query log analysis for improvements
- [ ] REST API server with Go standard library
- [ ] gRPC API for high-performance scenarios
- [ ] Web UI (simple HTML/JS interface)
- [ ] Export/import index formats

---

## Completed
- ✅ Phase 1: Foundation (2026-02-10)
- ✅ Phase 2: Core Components (2026-02-10)
- ✅ Phase 3: CLI & Storage (2026-02-11)
- ✅ Phase 4: Polish & Quality (2026-02-11)
- ✅ Phase 5: Anti-Blocking Features (2026-02-11)
- ✅ Phase 6: Crawler Enhancements (2026-02-11) - Documentation completed
- ✅ Enhanced Index Persistence (2026-02-11)
- ✅ HTTP API Server (2026-02-11)
- ✅ Integration Testing Framework (2026-02-11) - Test scripts and helpers created
- ✅ Crawler Module internals (2026-02-10)
- ✅ Indexer Module (2026-02-10)
- ✅ Ranker Module (2026-02-10)
- ✅ Search Module (2026-02-10)
- ✅ Storage Layer (2026-02-10)
- ✅ CLI skeleton with all commands (2026-02-10)
- ✅ Search command with ranker integration (2026-02-10)
- ✅ Index command with indexer integration (2026-02-10)
- ✅ Crawl command implementation (2026-02-11) - Full crawler integration with auto-indexing
- ✅ Search command auto-rebuild feature (2026-02-11)
- ✅ Seeding strategies documentation (2026-02-11)
- ✅ Anti-blocking strategies documentation (2026-02-11)
- ✅ Integration testing infrastructure (2026-02-11) - Test scripts and helpers created

---

## Phase Tracking

| Phase | Status | Started | Completed |
|-------|--------|---------|-----------|
| Foundation | ✅ Complete | 2026-02-10 | 2026-02-10 |
| Core Components | ✅ Complete | 2026-02-10 | 2026-02-10 |
| CLI & Storage | ✅ Complete | 2026-02-11 | 2026-02-11 |
| Polish & Quality | ✅ Complete | 2026-02-11 | 2026-02-11 |
| Anti-Blocking Features | ✅ Complete | 2026-02-11 | 2026-02-11 |
| Crawler Enhancements | ✅ Complete | 2026-02-11 | 2026-02-11 |
| Integration Testing | 🔄 In Progress | 2026-02-11 | — |
| Seeding Features | ✅ Complete | 2026-02-11 | 2026-02-11 |
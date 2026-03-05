# Changelog

All notable changes to gosearch will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- MIT License for open source distribution
- Open source documentation (CONTRIBUTING.md, SECURITY.md, CODE_OF_CONDUCT.md)
- golangci-lint configuration for consistent code quality checks
- Go version alignment (CI now uses 1.24.2 matching go.mod)

## [2.0.0] - 2026-02-13

### Added
- Interactive TUI for terminal-based search and browsing
  - Google-inspired two-column layout
  - Live markdown preview for search results
  - Keyboard navigation (j/k, arrows, n/p for pagination)
  - Preview caching for responsive navigation
- `search-item` command for JSON output of ranked results
- `search-read` command for styled markdown rendering in terminal
- `md-export` domain command for exporting indexed markdown by domain
  - Deterministic filenames with path-based slugs
  - Overwrite behavior for idempotent exports
  - Subdomain inclusion support
- Markdown-only storage format (v2)
  - Crawled pages stored as gzip-compressed JSON envelopes
  - Clean `content_markdown` field (no raw HTML persisted)
  - Schema version marker for automatic migration
- One-time corpus reset for v2 migration on first crawl
- Queue limit enforcement with `--max-queue` flag
- Detailed timing metrics for crawl operations
- Optimized index saving with batched writes

### Changed
- Storage format from HTML+markdown to markdown-only
- Index document tokenization uses plain text derived from markdown
- TUI renders bounded markdown slices for responsive updates

### Fixed
- TUI navigation keeps selected result visible while scrolling
- TUI timeouts and retries on stalled preview loads
- Crawl command performs automatic schema v2 migration

## [1.0.0] - 2026-02-11

### Added
- Complete web crawler with Colly framework
  - Concurrent crawling with configurable worker pools
  - Politeness policies (rate limiting, robots.txt)
  - URL deduplication with Bloom filter
  - Seed set management (general, programming, academic)
  - Crawling strategies (BFS, best-first)
- Custom inverted index implementation
  - Unicode-aware tokenization with stopword removal
  - Variable-byte gap encoding for compression
  - Positional index for phrase queries
  - Boolean operations (AND, OR, NOT)
  - BoltDB persistence for complete index metadata
- Ranking algorithms
  - TF-IDF scoring with cosine similarity
  - BM25 variant
  - PageRank computation
  - Combined scoring with boost factors
- Search functionality
  - Boolean query parsing (AND, OR, NOT)
  - Phrase query support
  - Fuzzy matching with Levenshtein distance
  - Query suggestions for misspelled terms
  - Redis-based query caching with TTL
- CLI commands
  - `crawl` - Web crawling with automatic indexing
  - `search` - Search the index with ranking
  - `index` - Index management (build, rebuild, stats, clear, optimize, validate)
  - `serve` - HTTP API server
  - `backup` - Backup index to file
  - `restore` - Restore index from file
- Storage layer
  - Document store with gzip compression
  - Index store with BoltDB
  - Cache store with Redis
- Anti-blocking features
  - Browser header profiles (Chrome, Firefox, Safari, Edge)
  - Cookie management with jar
  - Block detection with exponential backoff
  - Request header enrichment (Sec-CH-UA, Sec-Fetch-*)
- Progress bars for long operations
- Configuration file support (.gosearch.yaml)
- Comprehensive error messages with suggestions
- Integration testing infrastructure

## [Unreleased] Features Under Development

### Planned
- Distributed crawling (multiple workers, shared frontier)
- Incremental index updates (partial re-indexing)
- Query suggestions and autocomplete
- Result snippets highlighting matched terms
- Faceted search (filter by domain, date, content type)
- gRPC API for high-performance scenarios
- Web UI (simple HTML/JS interface)
- Export/import index formats

---

## Version Naming

- **Major version** (X.0.0): Breaking changes, major features
- **Minor version** (1.X.0): New features, backward compatible
- **Patch version** (1.0.X): Bug fixes, minor improvements

## Upgrade Notes

### From 1.x to 2.0
- Storage format changed to markdown-only
- Run `crawl` once to perform automatic migration
- Old HTML data is cleared on first run

### From pre-1.0 to 1.0
- Initial stable release
- All features from development phase included

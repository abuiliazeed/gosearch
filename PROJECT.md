# gosearch

## Vision

gosearch is a lightweight web search engine built from scratch in Go. It crawls web pages, builds a custom inverted index, and provides fast search capabilities with page ranking, boolean queries, fuzzy matching, and query caching.

## Project Type

Web Search Engine (CLI-based)

---

## Tech Stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| **Language** | Go 1.21+ | Strong concurrency, performance |
| **Crawler** | Colly Framework | Fast, concurrent web scraping |
| **Index** | Custom Inverted Index | Pure Go implementation |
| **Cache** | Redis | Query result caching |
| **Storage** | File + Embedded KV | BoltDB/Badger for metadata |
| **Concurrency** | Configurable Worker Pool | Controlled goroutine limits |
| **CLI** | Cobra + Viper | Command-line interface |
| **Deployment** | Local Binary | Standalone executable |

---

## Architecture Decisions

### 1. Inverted Index Design
- **Postings list**: `map[string][]*Posting` where `Posting` contains DocumentID, position, frequency
- **Compression**: Variable-byte encoding for gaps in docID lists
- **Tokenization**: Unicode-aware text segmentation with stopword removal
- **Positional index**: Supports phrase queries

### 2. Crawler Architecture
```
Seed URLs → Frontier Queue → Worker Pool → HTML Parser → Content Store
                                              ↓
                                        Link Extractor → Frontier
```
- **Frontier**: Priority queue with politeness policies
- **Worker pool**: Configurable concurrent goroutines
- **Politeness**: Respect robots.txt, rate limiting per domain
- **Deduplication**: URL fingerprinting with seen cache

### 3. Ranking Strategy
- **TF-IDF**: Term frequency-inverse document frequency
- **PageRank**: Iterative link analysis (optional)
- **Boosts**: Title/heading matches, URL depth, freshness
- **Fuzzy matching**: Levenshtein distance for typos

### 4. Storage Layers
| Data | Storage | Reason |
|------|---------|--------|
| Web page content | Files (compressed) | Large blobs, sequential access |
| Index metadata | BoltDB/Badger | Fast KV lookups |
| Query cache | Redis | Fast TTL-based caching |
| URL frontier | Redis Queue | Distributed-ready |

### 5. Concurrency Strategy
```
Config → WorkerCount → Goroutine Pool → Semaphore Pattern
                                      ↓
                                Rate Limiter per Domain
```
- Configurable workers (default: 10, max: 100)
- Semaphore for concurrent requests
- Per-domain rate limiting
- Graceful shutdown with context cancellation

---

## File Structure

```
gosearch/
├── cmd/
│   └── gosearch/                # Main CLI application
│       └── main.go
├── internal/                    # Private application code
│   ├── crawler/                 # Web crawler with Colly
│   │   ├── crawler.go           # Main crawler logic
│   │   ├── frontier.go          # URL frontier queue
│   │   ├── politeness.go        # Rate limiting, robots.txt
│   │   └── dedupe.go            # URL deduplication
│   ├── indexer/                 # Inverted index builder
│   │   ├── indexer.go           # Index construction
│   │   ├── tokenizer.go         # Text tokenization
│   │   ├── postings.go          # Postings list operations
│   │   └── compression.go       # Gap encoding
│   ├── search/                  # Query processor
│   │   ├── search.go            # Main search logic
│   │   ├── boolean.go           # AND/OR/NOT parsing
│   │   ├── fuzzy.go             # Fuzzy matching
│   │   └── cache.go             # Redis query cache
│   ├── ranker/                  # Ranking algorithms
│   │   ├── tfidf.go             # TF-IDF scoring
│   │   ├── pagerank.go          # PageRank computation
│   │   └── scorer.go            # Combined scoring
│   └── storage/                 # Data persistence
│       ├── document_store.go    # File-based page storage
│       ├── index_store.go       # Index metadata (BoltDB)
│       └── cache_store.go       # Redis cache client
├── pkg/                         # Public libraries
│   ├── cli/                     # CLI commands (Cobra)
│   │   ├── root.go              # Root command
│   │   ├── crawl.go             # Crawl command
│   │   ├── search.go            # Search command
│   │   └── index.go             # Index management
│   └── config/                  # Configuration management
│       ├── config.go            # Viper config loader
│       └── defaults.go          # Default values
├── docs/                        # Documentation
│   ├── FEATURE_SPEC_TEMPLATE.md
│   ├── REVIEW_CHECKLIST.md
│   └── AGENT_SESSION_TEMPLATE.md
├── scripts/
│   └── pre-deploy.sh            # Quality gate script
├── data/                        # Runtime data (gitignored)
│   ├── index/                   # Index files
│   ├── pages/                   # Crawled page storage
│   └── cache/                   # Cache data
├── .github/
│   └── workflows/
│       └── quality.yml          # CI/CD pipeline
├── PROJECT.md                   # ← This file
├── CLAUDE.md                    # Agent operating instructions
├── ROADMAP.md                   # Sprint tracking
├── STATE.md                     # Living status document
├── go.mod
├── go.sum
├── Makefile                     # Build/run commands
└── README.md                    # User documentation
```

---

## Core Features (MVP)

- Page Ranking: TF-IDF scoring with optional PageRank
- Boolean Queries: AND, OR, NOT query operators
- Fuzzy Matching: Levenshtein distance for typo tolerance
- Query Caching: Redis-based result caching with TTL

---

## Environment Variables

```bash
# GoSearch Configuration
GOSEARCH_MAX_WORKERS=10           # Max concurrent crawler workers
GOSEARCH_MAX_DEPTH=3              # Max crawl depth from seed
GOSEARCH_USER_AGENT="GoSearch/1.0" # User-Agent string

# Redis Cache
REDIS_HOST=localhost:6379
REDIS_PASSWORD=
REDIS_CACHE_TTL=3600              # Query cache TTL in seconds

# Storage
DATA_DIR=./data                   # Base data directory
INDEX_PATH=./data/index          # Index file location
PAGES_PATH=./data/pages          # Page storage location

# Logging
LOG_LEVEL=info                   # debug, info, warn, error
LOG_FORMAT=text                  # text or json
```

---

## AI Agent Setup

**Agent tool:** Claude Code CLI
**Testing level:** Minimal (build + smoke tests)
**Review strictness:** Standard (critical issues must be fixed)
**Documentation level:** Standard (package-level docs, exported functions)

See `CLAUDE.md` for complete agent operating instructions.
See `docs/AGENT_SESSION_TEMPLATE.md` for how to start every coding session.

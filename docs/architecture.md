# gosearch Architecture Documentation

> This document provides a comprehensive overview of the gosearch web search engine architecture, including component design, data flows, and implementation details.

---

## Table of Contents

1. [Overview](#1-overview)
2. [System Architecture](#2-system-architecture)
3. [Directory Structure](#3-directory-structure)
4. [Core Components](#4-core-components)
5. [Data Flows](#5-data-flows)
6. [CLI Commands](#6-cli-commands)
7. [External Dependencies](#7-external-dependencies)
8. [Configuration](#8-configuration)
9. [Design Patterns Used](#9-design-patterns-used)
10. [Key File References](#10-key-file-references)

---

## 1. Overview

### 1.1 Project Description

**gosearch** is a lightweight, distributed web search engine built from scratch in Go. It crawls web pages, builds a custom inverted index, and provides fast search capabilities with features including:

- Distributed web crawling with politeness policies
- Inverted index with positional information for phrase queries
- PageRank-based result ranking
- Boolean query support (AND, OR, NOT)
- Fuzzy matching with Levenshtein distance
- Query result caching with Redis
- RESTful HTTP API

### 1.2 Technology Stack

| Component | Technology |
|-----------|-----------|
| **Language** | Go 1.24+ |
| **CLI Framework** | Cobra |
| **Web Crawler** | Colly |
| **Index Storage** | BoltDB |
| **Document Storage** | File-based (gzip JSON) |
| **Query Cache** | Redis |
| **Logging** | Zap |
| **Configuration** | Viper |

### 1.3 High-Level Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLI Layer (Cobra)                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │  crawl   │  │  search  │  │  index   │  │  serve   │  │ backup   │       │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘       │
└───────┼────────────┼────────────┼────────────┼────────────┼──────────────────┘
        │            │            │            │            │
┌───────┴────────────┴────────────┴────────────┴────────────┴──────────────────┐
│                              Application Layer                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │   Crawler    │  │    Searcher  │  │   Ranker     │  │    Server    │    │
│  │  (Colly+)    │  │  (Parser+)   │  │  (TF-IDF+PR) │  │   (HTTP+)    │    │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘    │
└─────────┼──────────────────┼──────────────────┼──────────────────┼───────────┘
          │                  │                  │                  │
┌─────────┴──────────────────┴──────────────────┴──────────────────┴───────────┐
│                              Core Services Layer                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │   Indexer    │  │ Tokenizer    │  │  Frontier    │  │ Politeness   │    │
│  │ (InvIndex+)  │  │ (Stopwords)  │  │  (PriorityQ) │  │ (robots.txt) │    │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘    │
└─────────┼──────────────────┼──────────────────┼──────────────────┼───────────┘
          │                  │                  │                  │
┌─────────┴──────────────────┴──────────────────┴──────────────────┴───────────┐
│                              Storage Layer                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │DocumentStore │  │  IndexStore  │  │  CacheStore  │  │  Deduplicator │    │
│  │  (Files)     │  │  (BoltDB)    │  │  (Redis)     │  │(BloomFilter)  │    │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. System Architecture

### 2.1 Layered Architecture

gosearch follows a strict layered architecture pattern:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Presentation Layer                                │
│                    CLI Commands (pkg/cli/) | HTTP API (internal/server/)    │
├─────────────────────────────────────────────────────────────────────────────┤
│                           Application Layer                                 │
│     Crawler | Searcher | Ranker | Indexer | Tokenizer | Parser | Fuzzy      │
├─────────────────────────────────────────────────────────────────────────────┤
│                           Domain Layer                                       │
│        InvertedIndex | PostingsList | Document | Query | Results            │
├─────────────────────────────────────────────────────────────────────────────┤
│                           Storage Layer                                      │
│    DocumentStore | IndexStore | CacheStore | Frontier | BloomFilter         │
├─────────────────────────────────────────────────────────────────────────────┤
│                           Infrastructure Layer                               │
│       HTTP Client | File System | BoltDB | Redis | Logging | Config         │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Component Interaction Overview

```
                    ┌─────────────────┐
                    │   CLI / API     │
                    └────────┬────────┘
                             │
            ┌────────────────┼────────────────┐
            ▼                ▼                ▼
    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
    │   Crawler    │  │   Searcher   │  │   Indexer    │
    │              │  │              │  │              │
    │ • Frontier   │  │ • Parser     │  │ • Tokenizer  │
    │ • Workers    │  │ • Boolean    │  │ • Postings   │
    │ • Politeness │  │ • Fuzzy      │  │ • Compression│
    └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
           │                 │                 │
           └─────────────────┼─────────────────┘
                             ▼
                    ┌──────────────┐
                    │   Ranker     │
                    │              │
                    │ • TF-IDF     │
                    │ • PageRank   │
                    │ • Scoring    │
                    └──────┬───────┘
                           ▼
                    ┌──────────────┐
                    │   Storage    │
                    │              │
                    │ • Documents  │
                    │ • Index      │
                    │ • Cache      │
                    └──────────────┘
```

---

## 3. Directory Structure

```
/Users/abuiliazeed/conductor/workspaces/gosearch/montgomery/
├── cmd/
│   └── gosearch/
│       └── main.go                 # Application entry point
├── internal/                       # Private application code
│   ├── crawler/                    # Web crawler module
│   │   ├── blocker.go              # Block detection (403/429/CAPTCHA)
│   │   ├── cookies.go              # Cookie management with jar
│   │   ├── crawler.go              # Main CollyCrawler with worker pool
│   │   ├── dedupe.go               # URL deduplication with Bloom filter
│   │   ├── frontier.go             # Priority queue for URL scheduling
│   │   ├── headers.go              # Browser header profiles
│   │   ├── politeness.go           # Rate limiting and robots.txt compliance
│   │   └── types.go                # Crawler types (Config, URL, Stats, etc.)
│   ├── indexer/                    # Inverted index module
│   │   ├── compression.go          # Variable-byte gap encoding
│   │   ├── index.go                # In-memory index operations
│   │   ├── indexer.go              # Indexer with persistence
│   │   ├── postings.go             # Postings list operations
│   │   ├── tokenizer.go            # Unicode tokenization
│   │   └── types.go                # Index types (Token, Posting, etc.)
│   ├── ranker/                     # Ranking algorithms
│   │   ├── pagerank.go             # Iterative PageRank computation
│   │   ├── scorer.go               # Combined scoring (TF-IDF + PageRank)
│   │   └── tfidf.go                # TF-IDF with BM25 variant
│   ├── search/                     # Query processor
│   │   ├── fuzzy.go                # Levenshtein & Jaro-Winkler similarity
│   │   ├── parser.go               # Query parser with boolean operators
│   │   ├── search.go               # Main Searcher with caching
│   │   └── types.go                # Search config and result types
│   ├── server/                     # HTTP API server
│   │   ├── handlers.go             # API handlers (search, stats, rebuild)
│   │   ├── middleware.go           # Logging, recovery, CORS, JSON
│   │   ├── server.go               # HTTP server with graceful shutdown
│   │   └── types.go                # Server config and API types
│   └── storage/                    # Data persistence layer
│       ├── cache_store.go          # Redis client wrapper
│       ├── document_store.go       # File-based gzipped JSON storage
│       ├── index_store.go          # BoltDB-based index storage
│       └── types.go                # Storage types (Document, IndexMeta, etc.)
├── pkg/                            # Public libraries
│   ├── cli/                        # CLI commands (Cobra)
│   │   ├── backup.go               # Backup command
│   │   ├── crawl.go                # Crawl command
│   │   ├── index.go                # Index subcommands
│   │   ├── restore.go              # Restore command
│   │   ├── root.go                 # Root command
│   │   ├── search.go               # Search command
│   │   └── serve.go                # Serve command
│   ├── config/                     # Configuration management
│   │   ├── config.go               # Viper-based configuration
│   │   └── errors.go               # Configuration errors
│   └── progress/                   # Progress bar utilities
│       └── progress.go             # Progress bar wrapper
├── data/                           # Runtime data (gitignored)
│   ├── index/                      # BoltDB index files
│   └── pages/                      # Crawled page storage (gzip JSON)
├── docs/                           # Documentation
│   ├── PROJECT.md                  # Project overview
│   ├── ROADMAP.md                  # Feature roadmap
│   ├── STATE.md                    # Current state
│   ├── CLAUDE.md                   # Agent instructions
│   └── architecture.md             # This file
├── .gosearch.yaml                  # Configuration file
├── go.mod                          # Go module definition
├── go.sum                          # Go module checksums
├── Makefile                        # Build commands
└── README.md                       # Project README
```

### Directory Purposes

| Directory | Purpose |
|-----------|---------|
| `cmd/` | Entry points for building binaries |
| `internal/` | Private application code (cannot be imported externally) |
| `pkg/` | Public libraries that could be imported by other projects |
| `data/` | Runtime data storage (gitignored) |
| `docs/` | Project documentation |

---

## 4. Core Components

### 4.1 Storage Layer (`internal/storage/`)

The storage layer provides three distinct storage mechanisms optimized for different use cases.

#### 4.1.1 Document Store

**File:** `internal/storage/document_store.go`

The `DocumentStore` provides file-based persistence for crawled web pages using gzipped JSON format.

```go
// Document represents a crawled web page
type Document struct {
    ID        string    `json:"id"`        // SHA256 hash of URL
    URL       string    `json:"url"`       // Original URL
    Title     string    `json:"title"`     // Page title
    Content   string    `json:"content"`   // Extracted text content
    HTML      string    `json:"html"`      // Raw HTML
    Links     []string  `json:"links"`     // Extracted links
    CrawledAt time.Time `json:"crawled_at"` // Timestamp
    Depth     int       `json:"depth"`     // Crawl depth
}
```

**Key Features:**
- SHA256-based URL hashing for deterministic file paths
- Gzip compression for reduced disk usage
- Atomic writes with temporary file + rename pattern
- Thread-safe operations with mutex protection

#### 4.1.2 Index Store

**File:** `internal/storage/index_store.go`

The `IndexStore` uses BoltDB (embedded key-value store) for index metadata.

**BoltDB Buckets:**
| Bucket | Purpose | Key Type | Value Type |
|--------|---------|----------|------------|
| `meta` | Index metadata | `[]byte("meta")` | `IndexMeta` (JSON) |
| `terms` | Term information | term string | `TermInfo` (JSON) |
| `documents` | Document metadata | doc ID | `DocInfo` (JSON) |
| `postings` | Postings lists | term string | `PersistedPostingsList` (JSON) |
| `docinfo` | Additional doc info | doc ID | `PersistedDocInfo` (JSON) |

```go
// IndexMeta represents metadata about the inverted index
type IndexMeta struct {
    TotalDocuments  int       `json:"total_documents"`
    TotalTerms      int       `json:"total_terms"`
    LastUpdated     time.Time `json:"last_updated"`
    IndexSize       int64     `json:"index_size"`
    TotalPostings   int64     `json:"total_postings"`
    AveragePostings float64   `json:"average_postings"`
}
```

#### 4.1.3 Cache Store

**File:** `internal/storage/cache_store.go`

The `CacheStore` provides Redis-backed query result caching.

```go
// CacheEntry represents a cached search result
type CacheEntry struct {
    Query     string      `json:"query"`
    Results   interface{} `json:"results"`
    ExpiresAt time.Time   `json:"expires_at"`
    CreatedAt time.Time   `json:"created_at"`
}
```

**Key Operations:**
- `Get(ctx, key)` - Retrieve cached entry
- `Set(ctx, key, entry, ttl)` - Store entry with TTL
- `Delete(ctx, key)` - Remove specific entry
- `Clear(ctx)` - Flush all cached entries

---

### 4.2 Crawler Module (`internal/crawler/`)

The crawler module implements a distributed web crawler with politeness policies and advanced features.

#### 4.2.1 CollyCrawler

**File:** `internal/crawler/crawler.go`

The main crawler implements a worker pool pattern with semaphore-controlled concurrency.

```go
// Worker pool initialization (lines 97-100)
for i := 0; i < c.config.MaxWorkers; i++ {
    c.wg.Add(1)
    go c.worker(i, errChan)
}
```

**Architecture:**
```
┌─────────────────────────────────────────────────────────────┐
│                      CollyCrawler                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │   Worker 1   │  │   Worker 2   │  │   Worker N   │     │
│  │  (goroutine) │  │  (goroutine) │  │  (goroutine) │     │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘     │
│         │                 │                 │              │
│         └─────────────────┼─────────────────┘              │
│                           ▼                                │
│                    ┌───────────┐                            │
│                    │  Error    │                            │
│                    │  Channel  │                            │
│                    └───────────┘                            │
└─────────────────────────────────────────────────────────────┘
```

**Configuration Options:**
```go
type Config struct {
    MaxWorkers      int           // Concurrent worker count
    MaxDepth        int           // Maximum crawl depth
    Delay           time.Duration // Delay between requests
    AllowedDomains  []string      // Domain whitelist
    DisallowedDomains []string    // Domain blacklist
    UserAgent       string        // Custom User-Agent
    Timeout         time.Duration // Request timeout
}
```

#### 4.2.2 Frontier (Priority Queue)

**File:** `internal/crawler/frontier.go`

The Frontier implements a priority queue using Go's `heap.Interface` for URL scheduling.

```go
// URL represents a URL to be crawled with priority
type URL struct {
    URL      string
    Priority int
    Depth    int
    Index    int // heap index
}

// Frontier is a priority queue for URLs
type Frontier struct {
    urls []*URL
    mu   sync.RWMutex
}
```

**Priority Scheduling:**
- Higher priority URLs are crawled first
- Same priority: FIFO order
- Depth tracking for breadth-first traversal

#### 4.2.3 Politeness Manager

**File:** `internal/crawler/politeness.go`

The PolitenessManager enforces robots.txt compliance and rate limiting.

**Features:**
- robots.txt parsing and caching
- Per-domain rate limiting with token bucket
- Configurable delay between requests
- Crawl-delay respect

```go
// PolitenessManager manages rate limiting and robots.txt compliance
type PolitenessManager struct {
    mu          sync.RWMutex
    delays      map[string]time.Duration  // per-domain crawl-delay
    lastVisit   map[string]time.Time       // last visit timestamp
    robotsCache map[string]*robotstxt.RobotsData
    delay       time.Duration              // default delay
}
```

#### 4.2.4 Deduplicator (Bloom Filter)

**File:** `internal/crawler/dedupe.go`

URL deduplication uses a Bloom filter for memory-efficient duplicate detection.

```go
// Deduplicator uses a Bloom filter for URL deduplication
type Deduplicator struct {
    filter *bloom.BloomFilter
    mu     sync.RWMutex
}
```

**Benefits:**
- O(1) lookup time
- Constant memory usage
- Configurable false-positive rate
- Thread-safe operations

#### 4.2.5 Header Profiles

**File:** `internal/crawler/headers.go`

Pre-configured browser header profiles for mimicking real browsers.

```go
type HeaderProfile struct {
    Name            string
    UserAgent       string
    Accept          string
    AcceptLanguage  string
    AcceptEncoding  string
    Connection      string
    UpgradeInsecure string
}

var Profiles = []HeaderProfile{
    ChromeProfile,
    FirefoxProfile,
    SafariProfile,
    EdgeProfile,
}
```

#### 4.2.6 Block Detector

**File:** `internal/crawler/blocker.go`

Detects and handles server blocking (403, 429, CAPTCHA) with exponential backoff.

```go
type BlockInfo struct {
    Blocked     bool
    StatusCode  int
    DetectedAt  time.Time
    RetryAfter  time.Duration
}
```

---

### 4.3 Indexer Module (`internal/indexer/`)

The indexer module implements a full-text inverted index with positional information.

#### 4.3.1 Inverted Index Structure

**File:** `internal/indexer/types.go` (lines 55-69)

```go
// InvertedIndex is the main index structure mapping terms to postings
type InvertedIndex struct {
    terms     map[string]*PostingsList  // Term -> PostingsList
    docs      map[string]*DocInfo       // DocID -> DocInfo
    totalDocs int
}

// PostingsList represents the list of documents containing a term
type PostingsList struct {
    DocFrequency int       // Number of documents containing this term
    Postings     []Posting // List of postings (sorted by DocID)
}

// Posting represents a single document occurrence for a term
type Posting struct {
    DocID         string  // Document ID
    Positions     []int   // Positions where term appears in document
    TermFrequency int     // Frequency of term in document
}

// DocInfo stores metadata about an indexed document
type DocInfo struct {
    DocID      string
    URL        string
    Title      string
    TokenCount int     // Total tokens in document
    Length     int     // Document length in tokens (unique tokens)
    IndexedAt  time.Time
}
```

**Index Diagram:**
```
Term → PostingsList → [Posting, Posting, ...]
        │
        ├─ DocFrequency: 3
        └─ Postings:
            ├─ [DocID: "abc123", Positions: [1, 5, 10], TF: 3]
            ├─ [DocID: "def456", Positions: [2, 8], TF: 2]
            └─ [DocID: "ghi789", Positions: [0, 3, 7, 12], TF: 4]
```

#### 4.3.2 Tokenizer

**File:** `internal/indexer/tokenizer.go`

Unicode-aware tokenization with stopword removal.

```go
// TokenizerConfig holds configuration for the tokenizer
type TokenizerConfig struct {
    Stopwords   map[string]bool  // Set of stopwords to filter
    MinTokenLen int              // Minimum token length
}
```

**Tokenization Process:**
1. Unicode normalization (NFC)
2. Case folding to lowercase
3. Word boundary detection using Unicode rules
4. Stopword filtering
5. Minimum length filtering

#### 4.3.3 Postings Operations

**File:** `internal/indexer/postings.go`

Boolean operations on postings lists:

| Operation | Function | Description |
|-----------|----------|-------------|
| **AND** | `Intersect(a, b)` | Returns documents containing both terms |
| **OR** | `Union(a, b)` | Returns documents containing either term |
| **NOT** | `Difference(a, b)` | Returns documents in a but not in b |
| **Merge** | `Merge(lists)` | Combines multiple postings lists |

#### 4.3.4 Compression

**File:** `internal/indexer/compression.go`

Variable-byte gap encoding for posting list compression.

```go
// EncodeGaps encodes a sorted list of integers using gap encoding
func EncodeGaps(nums []int) []byte

// DecodeGaps decodes gap-encoded integers
func DecodeGaps(data []byte) []int
```

**Benefits:**
- Smaller storage footprint
- Faster disk I/O
- Decodes on-demand during search

---

### 4.4 Ranker Module (`internal/ranker/`)

The ranker module implements document ranking using multiple scoring signals.

#### 4.4.1 Combined Scorer

**File:** `internal/ranker/scorer.go`

Combines multiple scoring factors:

```go
// Combined score formula (line 106)
score = w_tfidf * tfidf_score + w_pr * pr_score +
        title_boost + url_boost + freshness_boost
```

**Scoring Components:**

| Component | Weight | Description |
|-----------|--------|-------------|
| TF-IDF | 0.5 | Term frequency-inverse document frequency |
| PageRank | 0.3 | Link-based importance |
| Title Boost | +0.1 | Query terms in title |
| URL Boost | +0.05 | Shorter URLs favored |
| Freshness Boost | +0.05 | More recent documents favored |

#### 4.4.2 TF-IDF (BM25 Variant)

**File:** `internal/ranker/tfidf.go`

```go
// BM25 scoring formula
score = IDF * ((TF * (k1 + 1)) / (TF + k1 * (1 - b + b * (docLen / avgDocLen))))

// Default parameters
k1 = 1.2  // Term frequency saturation
b = 0.75  // Length normalization
```

#### 4.4.3 PageRank

**File:** `internal/ranker/pagerank.go`

Iterative PageRank computation:

```go
// PageRank formula
PR(u) = (1 - d) / N + d * Σ(PR(v) / L(v))

// Default damping factor
d = 0.85
```

---

### 4.5 Search Module (`internal/search/`)

The search module processes queries and retrieves ranked results.

#### 4.5.1 Searcher

**File:** `internal/search/search.go`

Main search component with Redis caching:

```go
type Searcher struct {
    indexer  *indexer.Indexer
    ranker   *ranker.Ranker
    parser   *Parser
    cache    *storage.CacheStore
    config   *SearchConfig
}
```

**Search Flow:**
```
Query → Parse → Cache Check → [hit] → Return Results
                        ↓ [miss]
                   BooleanSearch → Ranker → Cache → Return
```

#### 4.5.2 Query Parser

**File:** `internal/search/parser.go`

Supports boolean operators and phrase queries:

```go
type QueryType int

const (
    TermQuery QueryType = iota
    PhraseQuery
    BooleanQuery
    FuzzyQuery
)

type ParsedQuery struct {
    Type     QueryType
    Terms    []string
    Operator string  // "AND", "OR", "NOT"
    Children []*ParsedQuery
}
```

**Query Examples:**
| Query | Type | Result |
|-------|------|--------|
| `golang tutorial` | Term | Documents containing "golang" OR "tutorial" |
| `"golang tutorial"` | Phrase | Documents with exact phrase |
| `golang AND tutorial` | Boolean | Documents with both terms |
| `golang NOT tutorial` | Boolean | Documents with "golang" but not "tutorial" |
| `golang~` | Fuzzy | Fuzzy match with Levenshtein distance |

#### 4.5.3 Fuzzy Matching

**File:** `internal/search/fuzzy.go`

Implements two similarity algorithms:

```go
// Levenshtein distance (edit distance)
func Levenshtein(a, b string) int

// Jaro-Winkler similarity (0-1 scale)
func JaroWinkler(a, b string) float64
```

---

### 4.6 Server Module (`internal/server/`)

The server module provides a RESTful HTTP API.

#### 4.6.1 HTTP Server

**File:** `internal/server/server.go`

Graceful shutdown with context cancellation:

```go
func (s *Server) Start() error {
    // Start HTTP server
    go func() {
        if err := s.http.Serve(); err != nil && err != http.ErrServerClosed {
            s.log.Error("server error", zap.Error(err))
        }
    }()

    // Wait for shutdown signal
    <-s.shutdown

    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    return s.http.Shutdown(ctx)
}
```

#### 4.6.2 API Endpoints

**File:** `internal/server/handlers.go`

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/search` | GET | Search endpoint |
| `/api/v1/stats` | GET | Index statistics |
| `/api/v1/index/rebuild` | POST | Rebuild index |
| `/health` | GET | Health check |

#### 4.6.3 Middleware

**File:** `internal/server/middleware.go`

Middleware chain:

```
Request → Logging → Recovery → CORS → JSON → Handler → Response
```

**Middleware Components:**
- **Logging**: Request/response logging with request ID
- **Recovery**: Panic recovery with 500 response
- **CORS**: Cross-origin resource sharing
- **JSON**: JSON content-type enforcement

---

## 5. Data Flows

### 5.1 Crawl Pipeline Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CRAWL PHASE                                     │
└─────────────────────────────────────────────────────────────────────────────┘

Seed URLs
    │
    ▼
┌─────────────────┐
│   Frontier      │  Priority Queue (heap.Interface)
│   (PriorityQ)   │  - URL scheduling by priority
└────────┬────────┘  - Depth tracking
         │
         ▼
┌─────────────────┐
│  Worker Pool    │  Goroutine workers with semaphore
│  (goroutines)   │  - Acquire from frontier
└────────┬────────┘  - Process URL concurrently
         │
         ▼
┌─────────────────┐
│  Deduplicator   │  Bloom filter
│  (BloomFilter)  │  - O(1) duplicate check
└────────┬────────┘  - Memory-efficient
         │ [not duplicate]
         ▼
┌─────────────────┐
│ PolitenessMgr   │  robots.txt + Rate limiting
│  (Token Bucket) │  - Domain-specific delays
└────────┬────────┘  - robots.txt compliance
         │ [allowed]
         ▼
┌─────────────────┐
│  Colly Collector│  HTTP fetcher
│  (HTTP Client)  │  - Header profiles
└────────┬────────┘  - Cookie management
         │
         ▼
┌─────────────────┐
│  HTML Parser    │  Content extraction
│  (goquery)      │  - Title, Content, Links
└────────┬────────┘  - Link discovery
         │
         ▼
┌─────────────────┐
│ DocumentStore   │  File-based storage
│  (gzip JSON)    │  - SHA256 URL hashing
└─────────────────┘  - Atomic writes

┌─────────────────────────────────────────────────────────────────────────────┐
│                            LINK EXTRACTION                                   │
└─────────────────────────────────────────────────────────────────────────────┘

Extracted Links
    │
    ▼
┌─────────────────┐
│  URL Filter     │  Domain filtering
│  (Allow/Block)  │  - Allowed domains
└────────┬────────┘  - Disallowed domains
         │ [allowed]
         ▼
┌─────────────────┐
│   Frontier      │  Add back to queue
│  (PriorityQ)    │  - Increment depth
└─────────────────┘
```

### 5.2 Index Pipeline Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              INDEX PHASE                                     │
└─────────────────────────────────────────────────────────────────────────────┘

DocumentStore
    │
    │ Load all crawled documents
    ▼
┌─────────────────┐
│   Document      │  For each document:
│   Iterator      │  - Decompress gzip JSON
└────────┬────────┘  - Load Document struct
         │
         ▼
┌─────────────────┐
│   Tokenizer     │  Tokenization
│  (Unicode)      │  - Normalize to lowercase
└────────┬────────┘  - Remove stopwords
         │           - Filter by length
         ▼
┌─────────────────┐
│  Tokens         │  Token{Text, Position}
│  []Token        │  - Position tracking
└────────┬────────┘  - Per-field tokenization
         │           (title, content, url)
         ▼
┌─────────────────┐
│   Indexer       │  Build InvertedIndex
│  .IndexDoc()    │  - Add terms to index
└────────┬────────┘  - Build postings lists
         │           - Update DocInfo
         ▼
┌─────────────────┐
│ InvertedIndex   │  In-memory index
│  {              │  - terms: map[string]*PostingsList
│    terms: {},   │  - docs: map[string]*DocInfo
│    docs: {}     │
│  }              │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  IndexStore     │  BoltDB persistence
│  (BoltDB)       │  - Save to buckets:
└────────┬────────┘  - meta, terms, documents,
         │            postings, docinfo
         ▼
┌─────────────────┐
│   Disk Storage  │  Persistent index
│  /data/index/   │  - Survives restarts
└─────────────────┘
```

### 5.3 Search Pipeline Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              SEARCH PHASE                                    │
└─────────────────────────────────────────────────────────────────────────────┘

User Query
    │
    ▼
┌─────────────────┐
│    Parser       │  Query parsing
│  .Parse()       │  - Tokenize query
└────────┬────────┘  - Detect operators (AND/OR/NOT)
         │           - Identify phrases
         ▼
┌─────────────────┐
│  ParsedQuery    │  Query AST
│  {              │  - Type: Term/Phrase/Boolean/Fuzzy
│    Type: "",    │  - Terms: []
│    Terms: [],   │  - Operator: "AND/OR/NOT"
│    Operator: "" │
│  }              │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Cache Check    │  Redis cache lookup
│  (Redis)        │  - SHA256 query hash
└────────┬────────┘  - Check TTL
         │
         ├─────────[HIT]────────┐
         │                     ▼
         │           ┌─────────────────┐
         │           │  Cached Results  │  Return immediately
         │           │  (from Redis)    │  - Skip search
         │           └─────────────────┘
         │
         └─────────[MISS]───────┐
                                 ▼
                       ┌─────────────────┐
                       │ BooleanSearch   │  Execute search
                       │  .Search()      │  - Lookup postings
                       └────────┬────────┘  - Merge results
                                │  (AND/OR/NOT)
                                ▼
                       ┌─────────────────┐
                       │  Candidate Docs │  Matching documents
                       │  []string       │  - DocIDs from postings
                       └────────┬────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │    Ranker       │  Score documents
                       │  .RankDocs()    │  - TF-IDF scoring
                       └────────┬────────┘  - PageRank boost
                                │  - Title/URL/freshness
                                ▼
                       ┌─────────────────┐
                       │  Ranked Results │  Sorted by score
                       │  []Result       │  - Descending order
                       └────────┬────────┘  - Apply limit/offset
                                │
                                ▼
                       ┌─────────────────┐
                       │  Cache Store    │  Cache results
                       │  (Redis)        │  - Store with TTL
                       └────────┬────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │  Final Results  │  Return to user
                       │  SearchResponse │  - Include scores
                       └─────────────────┘  - Pagination info
```

---

## 6. CLI Commands

### 6.1 Root Command

**File:** `pkg/cli/root.go`

```bash
gosearch [global flags] <command> [command flags] [args...]
```

**Global Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `.gosearch.yaml` | Config file path |
| `--data-dir` | string | `./data` | Data directory |
| `--verbose` | bool | `false` | Verbose output |
| `--log-format` | string | `console` | Log format (console/json) |

### 6.2 Crawl Command

**File:** `pkg/cli/crawl.go`

```bash
gosearch crawl [urls...] [flags]
```

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--depth, -d` | int | `3` | Maximum crawl depth |
| `--workers, -w` | int | `10` | Number of concurrent workers |
| `--delay` | duration | `1s` | Delay between requests |
| `--allow` | strings | `[]` | Allowed domains |
| `--disallow` | strings | `[]` | Disallowed domains |
| `--user-agent` | string | `GoSearch/1.0` | Custom User-Agent |
| `--output` | string | `./data/pages` | Output directory |

**Example:**
```bash
gosearch crawl https://example.com --depth 2 --workers 20 --delay 500ms
```

### 6.3 Search Command

**File:** `pkg/cli/search.go`

```bash
gosearch search [query] [flags]
```

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit` | int | `10` | Maximum results |
| `--offset` | int | `0` | Result offset |
| `--fuzzy` | bool | `false` | Enable fuzzy matching |
| `--explain` | bool | `false` | Explain scoring |
| `--no-cache` | bool | `false` | Bypass cache |
| `--format` | string | `pretty` | Output format |

**Example:**
```bash
gosearch search "golang tutorial" --limit 20 --explain
```

### 6.4 Index Subcommands

**File:** `pkg/cli/index.go`

```bash
gosearch index <subcommand> [flags]
```

**Subcommands:**

| Command | Description |
|---------|-------------|
| `build` | Build index from crawled pages |
| `stats` | Show index statistics |
| `clear` | Clear the index |
| `optimize` | Optimize index (compress, merge) |
| `validate` | Validate index for consistency |

**Examples:**
```bash
gosearch index build
gosearch index stats
gosearch index optimize
```

### 6.5 Serve Command

**File:** `pkg/cli/serve.go`

```bash
gosearch serve [flags]
```

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--host` | string | `0.0.0.0` | Bind host |
| `--port, -p` | int | `8080` | Bind port |
| `--timeout` | duration | `30s` | Request timeout |

**Example:**
```bash
gosearch serve --host localhost --port 8080
```

### 6.6 Backup/Restore Commands

**Files:** `pkg/cli/backup.go`, `pkg/cli/restore.go`

```bash
gosearch backup [output-file]
gosearch restore [input-file]
```

---

## 7. External Dependencies

### 7.1 Direct Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/spf13/cobra` | v1.8.1 | CLI framework |
| `github.com/spf13/viper` | v1.18.2 | Configuration management |
| `github.com/gocolly/colly/v2` | v2.3.0 | Web scraping |
| `github.com/redis/go-redis/v9` | v9.17.3 | Redis client |
| `go.etcd.io/bbolt` | v1.4.3 | Embedded key-value store |
| `go.uber.org/zap` | v1.21.0 | Structured logging |
| `github.com/bits-and-blooms/bloom/v3` | v3.7.1 | Bloom filter |
| `github.com/temoto/robotstxt` | v1.1.2 | robots.txt parser |
| `github.com/schollz/progressbar/v3` | v3.19.0 | Progress bars |

### 7.2 Indirect Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/PuerkitoBio/goquery` | HTML parsing |
| `github.com/antchfx/htmlquery` | HTML XPath queries |
| `github.com/antchfx/xmlquery` | XML XPath queries |
| `github.com/antchfx/xpath` | XPath engine |
| `github.com/gobwas/glob` | Glob pattern matching |
| `github.com/kennygrant/sanitize` | HTML sanitization |
| `github.com/saintfish/chardet` | Character encoding detection |
| `golang.org/x/net` | Network utilities |
| `golang.org/x/text` | Unicode text processing |

---

## 8. Configuration

### 8.1 Configuration File

**Default locations:**
1. `.gosearch.yaml` (project root)
2. `$HOME/.gosearch.yaml`
3. `/etc/gosearch/config.yaml`

**Example configuration:**

```yaml
# .gosearch.yaml
crawler:
  max_workers: 20
  max_depth: 5
  delay: 1s
  timeout: 30s
  allowed_domains:
    - example.com
  disallowed_domains:
    - spam.com
  user_agent: "GoSearch/1.0"

indexer:
  min_token_len: 2
  compression: true

ranker:
  tfidf_weight: 0.5
  pagerank_weight: 0.3
  title_boost: 0.1
  url_boost: 0.05
  freshness_boost: 0.05
  k1: 1.2
  b: 0.75
  damping: 0.85

storage:
  data_dir: "./data"
  document_dir: "./data/pages"
  index_file: "./data/index/index.db"

cache:
  enabled: true
  ttl: 5m
  max_entries: 1000

server:
  host: "0.0.0.0"
  port: 8080
  timeout: 30s
  read_timeout: 10s
  write_timeout: 10s

logging:
  level: "info"
  format: "console"
```

### 8.2 Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GOSEARCH_MAX_WORKERS` | Max concurrent crawler workers | `10` |
| `GOSEARCH_MAX_DEPTH` | Max crawl depth | `3` |
| `GOSEARCH_USER_AGENT` | User-Agent string | `GoSearch/1.0` |
| `GOSEARCH_DATA_DIR` | Base data directory | `./data` |
| `REDIS_HOST` | Redis host | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password | `` |
| `REDIS_DB` | Redis database number | `0` |
| `REDIS_CACHE_TTL` | Cache TTL duration | `5m` |
| `LOG_LEVEL` | Log level | `info` |
| `LOG_FORMAT` | Log format | `console` |

---

## 9. Design Patterns Used

### 9.1 Worker Pool Pattern

**Location:** `internal/crawler/crawler.go`

Multiple goroutines (workers) process URLs from a shared channel/queue with semaphore-controlled concurrency.

```go
// Acquire semaphore
sem := make(chan struct{}, c.config.MaxWorkers)
sem <- struct{}{} // Acquire

// Release after processing
defer func() { <-sem }()
```

### 9.2 Priority Queue (Heap)

**Location:** `internal/crawler/frontier.go`

URL scheduling using Go's `heap.Interface` for priority-based ordering.

```go
type Frontier struct {
    urls []*URL
}

func (f *Frontier) Push(x interface{}) {
    f.urls = append(f.urls, x.(*URL))
}

func (f *Frontier) Pop() interface{} {
    old := f.urls
    n := len(old)
    item := old[n-1]
    f.urls = old[0 : n-1]
    return item
}
```

### 9.3 Bloom Filter

**Location:** `internal/crawler/dedupe.go`

Memory-efficient probabilistic data structure for URL deduplication.

- O(1) lookup time
- Constant memory usage
- Configurable false-positive rate

### 9.4 Inverted Index

**Location:** `internal/indexer/types.go`

Term → Postings List mapping for fast term lookup and boolean operations.

### 9.5 Context Cancellation

**Location:** Throughout codebase

Graceful shutdown via `context.Context` propagation.

```go
select {
case <-ctx.Done():
    return ctx.Err() // Context cancelled
default:
    // Continue processing
}
```

### 9.6 Layered Storage

**Location:** `internal/storage/`

Separation of concerns with specialized storage:
- DocumentStore → Files (content)
- IndexStore → BoltDB (metadata)
- CacheStore → Redis (query cache)

### 9.7 Middleware Pattern

**Location:** `internal/server/middleware.go`

Chaining middleware functions for HTTP request processing.

```go
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Logging logic
        next.ServeHTTP(w, r)
    })
}
```

### 9.8 Strategy Pattern

**Location:** `internal/ranker/`

Pluggable scoring strategies:
- TF-IDF scorer
- PageRank scorer
- Combined scorer

### 9.9 Builder Pattern

**Location:** `pkg/config/config.go`

Fluent configuration building.

```go
config := NewConfig().
    WithCrawler(crawlerConfig).
    WithIndexer(indexerConfig).
    WithRanker(rankerConfig).
    Build()
```

---

## 10. Key File References

### 10.1 Entry Points

| File | Purpose |
|------|---------|
| `cmd/gosearch/main.go` | Application entry point |
| `pkg/cli/root.go` | Root Cobra command |
| `pkg/cli/crawl.go` | Crawl command (lines 29-201) |
| `pkg/cli/search.go` | Search command (lines 36-199) |
| `pkg/cli/index.go` | Index subcommands |
| `pkg/cli/serve.go` | Serve command |

### 10.2 Core Components

| Module | File | Lines | Purpose |
|--------|------|-------|---------|
| **Storage** | `internal/storage/types.go` | 1-75 | Core storage types |
| | `internal/storage/document_store.go` | - | File-based document storage |
| | `internal/storage/index_store.go` | - | BoltDB index storage |
| | `internal/storage/cache_store.go` | - | Redis cache client |
| **Crawler** | `internal/crawler/types.go` | - | Crawler types |
| | `internal/crawler/crawler.go` | 17-537 | Main CollyCrawler |
| | `internal/crawler/frontier.go` | 1-96 | Priority queue |
| | `internal/crawler/politeness.go` | 1-221 | Rate limiting |
| | `internal/crawler/dedupe.go` | - | URL deduplication |
| **Indexer** | `internal/indexer/types.go` | 1-238 | Index types |
| | `internal/indexer/indexer.go` | 1-478 | Indexer with persistence |
| | `internal/indexer/tokenizer.go` | - | Tokenization |
| | `internal/indexer/postings.go` | - | Postings operations |
| | `internal/indexer/compression.go` | - | Gap encoding |
| **Ranker** | `internal/ranker/scorer.go` | 1-308 | Combined scoring |
| | `internal/ranker/tfidf.go` | - | TF-IDF with BM25 |
| | `internal/ranker/pagerank.go` | - | PageRank algorithm |
| **Search** | `internal/search/search.go` | 1-400 | Main Searcher |
| | `internal/search/parser.go` | - | Query parser |
| | `internal/search/fuzzy.go` | - | Fuzzy matching |
| **Server** | `internal/server/server.go` | 1-85 | HTTP server |
| | `internal/server/handlers.go` | - | API handlers |
| | `internal/server/middleware.go` | - | HTTP middleware |

### 10.3 Configuration

| File | Purpose |
|------|---------|
| `pkg/config/config.go` | Viper-based configuration |
| `pkg/config/errors.go` | Configuration errors |
| `.gosearch.yaml` | Default config file |

---

## Appendix A: Data Structures

### Document (storage/types.go)

```go
type Document struct {
    ID        string    `json:"id"`        // SHA256 hash of URL
    URL       string    `json:"url"`       // Original URL
    Title     string    `json:"title"`     // Page title
    Content   string    `json:"content"`   // Extracted text
    HTML      string    `json:"html"`      // Raw HTML
    Links     []string  `json:"links"`     // Outbound links
    CrawledAt time.Time `json:"crawled_at"` // Crawl timestamp
    Depth     int       `json:"depth"`     // Crawl depth
}
```

### InvertedIndex (indexer/types.go)

```go
type InvertedIndex struct {
    terms     map[string]*PostingsList  // Term → PostingsList
    docs      map[string]*DocInfo       // DocID → DocInfo
    totalDocs int
}

type PostingsList struct {
    DocFrequency int       // Number of documents containing term
    Postings     []Posting // Sorted list of postings
}

type Posting struct {
    DocID         string  // Document ID
    Positions     []int   // Term positions in document
    TermFrequency int     // Term frequency in document
}
```

### SearchResponse (search/types.go)

```go
type SearchResponse struct {
    Query      string     `json:"query"`
    Results    []Result   `json:"results"`
    Total      int        `json:"total"`
    Took       int64      `json:"took_ms"`
    Cached     bool       `json:"cached"`
}
```

---

## Appendix B: Build Commands

```bash
# Build the binary
go build -o bin/gosearch ./cmd/gosearch

# Run tests
go test ./...

# Run with race detector
go run -race ./cmd/gosearch

# Format code
go fmt ./...

# Lint (requires golangci-lint)
golangci-lint run

# Vet code
go vet ./...
```

---

**Document Version:** 1.0
**Last Updated:** 2026-02-11
**Project:** gosearch

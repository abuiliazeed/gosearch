# Feature Spec: Indexer Module

> **Status:** In Progress
> **Author:** Claude (Agent)
> **Date:** 2026-02-10
> **Sprint:** Phase 2 - Core Components

---

## User Story

**As a** search engine developer,
**I want to** build an inverted index from crawled documents,
**So that** users can perform fast and efficient searches on web content.

---

## Acceptance Criteria

1. [ ] Tokenizer correctly splits text into tokens with Unicode support
2. [ ] Stopwords are removed during tokenization (configurable list)
3. [ ] Inverted index maps terms to document postings with positions
4. [ ] Postings lists use gap encoding for compression
5. [ ] Positional index enables phrase queries ("exact match")
6. [ ] Index can be persisted to BoltDB for storage
7. [ ] Index can be loaded from BoltDB for searching
8. [ ] Document updates trigger index modifications
9. [ ] Index statistics (total docs, terms, postings) are tracked
10. [ ] Context cancellation is respected during indexing

---

## Data Model

### New Types
```go
package indexer

// Token represents a word with its position in the document.
type Token struct {
    Text     string
    Position int
}

// Posting represents a single document occurrence for a term.
type Posting struct {
    DocID        string   // Document ID
    Positions    []int    // Positions where term appears
    TermFrequency int     // Frequency of term in document
}

// PostingsList represents the list of documents containing a term.
type PostingsList struct {
    DocFrequency int       // Number of documents containing this term
    Postings     []Posting // List of postings
}

// InvertedIndex is the main index structure mapping terms to postings.
type InvertedIndex struct {
    mu           sync.RWMutex
    terms        map[string]*PostingsList // Term -> PostingsList
    docs         map[string]*DocInfo      // DocID -> DocInfo
    totalDocs    int
}

// DocInfo stores metadata about an indexed document.
type DocInfo struct {
    DocID        string
    URL          string
    Title        string
    TokenCount   int
    Length       int // Document length in tokens
    IndexedAt    time.Time
}

// Tokenizer handles text tokenization with stopword removal.
type Tokenizer struct {
    stopwords    map[string]bool
    minTokenLen  int
}

// Indexer manages index building and persistence.
type Indexer struct {
    index        *InvertedIndex
    tokenizer    *Tokenizer
    store        *storage.IndexStore
    logger       *zap.Logger
}
```

### Data Source
- **Where does the data come from?** Crawled documents from storage.DocumentStore
- **How is it fetched?** Direct storage access by Document ID
- **Caching strategy:** In-memory index with BoltDB persistence

---

## Component Breakdown

```
Indexer Package (internal/indexer)
├── types.go             — Type definitions (Token, Posting, PostingsList, InvertedIndex, DocInfo)
├── tokenizer.go         — Text tokenization with Unicode support and stopword removal
├── postings.go          — Postings list operations (add, merge, gap encoding)
├── index.go             — Inverted index management (build, update, query)
├── compression.go       — Variable-byte gap encoding for compression
└── indexer.go           — Main Indexer with persistence logic

CLI Command (pkg/cli/index.go)
├── build command        — Build index from crawled documents
├── stats command        — Show index statistics
└── rebuild command      — Rebuild index from scratch
```

### Public API
- **Exported functions:**
  - `NewIndexer(store *storage.IndexStore, logger *zap.Logger) *Indexer`
  - `IndexDocument(ctx context.Context, doc *storage.Document) error`
  - `GetPostings(term string) (*PostingsList, error)`
  - `GetDocInfo(docID string) (*DocInfo, error)`
  - `Save(ctx context.Context) error`
  - `Load(ctx context.Context) error`

- **Interface definitions:**
  - `Tokenizer` interface for pluggable tokenizers

---

## Function Signatures

```go
// Package indexer

// NewIndexer creates a new Indexer instance.
func NewIndexer(store *storage.IndexStore, logger *zap.Logger) *Indexer

// NewTokenizer creates a new Tokenizer with the given stopwords.
func NewTokenizer(stopwords []string, minTokenLen int) *Tokenizer

// Tokenize splits text into tokens with positions.
func (t *Tokenizer) Tokenize(text string) []Token

// IndexDocument adds a document to the inverted index.
//
// The ctx parameter controls cancellation. The doc parameter contains
// the document to index (title and content are tokenized).
//
// Returns an error if the document cannot be indexed or context is cancelled.
func (i *Indexer) IndexDocument(ctx context.Context, doc *storage.Document) error

// GetPostings returns the postings list for a term.
//
// Returns ErrNotFound if the term is not in the index.
func (i *Indexer) GetPostings(term string) (*PostingsList, error)

// GetDocInfo returns document metadata for a document ID.
func (i *Indexer) GetDocInfo(docID string) (*DocInfo, error)

// Save persists the index to BoltDB.
func (i *Indexer) Save(ctx context.Context) error

// Load loads the index from BoltDB.
func (i *Indexer) Load(ctx context.Context) error

// Stats returns index statistics.
func (i *Indexer) Stats() *IndexStats

// EncodeGaps encodes docID gaps using variable-byte encoding.
func EncodeGaps(gaps []int) []byte

// DecodeGaps decodes variable-byte encoded gaps to docIDs.
func DecodeGaps(data []byte) []int
```

---

## Error Handling

| Error Condition | Error Variable | Message Format | Recovery |
|----------------|----------------|----------------|----------|
| Term not found | `ErrTermNotFound` | `"term not found in index: %s"` | Return nil, wrap error |
| Document not found | `ErrDocNotFound` | `"document not found: %s"` | Return nil, wrap error |
| Invalid document | `ErrInvalidDocument` | `"invalid document: %w"` | Validate before indexing |
| Storage failure | `ErrStorage` | `"storage error: %w"` | Log and return |
| Context cancelled | `context.Canceled` | Propagate | Return immediately |

---

## Edge Cases & Error States

| Scenario | Expected Behavior |
|----------|------------------|
| Empty document | Skip document, log warning |
| Document with no tokens | Add to index with zero token count |
| Term already exists | Merge postings, update doc frequency |
| Duplicate document ID | Update existing document info |
| Context cancelled during indexing | Return context.Canceled immediately |
| Concurrent index access | Use RWMutex for safe access |
| Token shorter than min length | Skip token |
| Stopword encountered | Skip token |

---

## Files to Create/Modify

### New Files
- `internal/indexer/types.go` — Type definitions
- `internal/indexer/tokenizer.go` — Tokenization logic
- `internal/indexer/postings.go` — Postings list operations
- `internal/indexer/compression.go` — Gap encoding/decoding
- `internal/indexer/index.go` — Inverted index management
- `internal/indexer/indexer.go` — Main Indexer with persistence

### Modified Files
- `pkg/cli/index.go` — Connect to indexer module
- `STATE.md` — Update module status

---

## Testing Plan

| Test Type | What to Test | File |
|-----------|-------------|------|
| Unit | Tokenize function with various inputs | `tokenizer_test.go` |
| Unit | Gap encoding/decoding correctness | `compression_test.go` |
| Unit | Postings list add/merge operations | `postings_test.go` |
| Integration | Index building from documents | `indexer_integration_test.go` |
| Benchmark | Tokenization performance | `tokenizer_bench_test.go` |

---

## Performance Considerations

- **Time complexity:**
  - Tokenization: O(n) where n is text length
  - Index lookup: O(1) for term lookup
  - Gap encoding: O(m) where m is postings count

- **Space complexity:**
  - Inverted index: O(total tokens across all documents)
  - Gap compression: ~50% space reduction for docID lists

- **Bottlenecks:**
  - Document parsing (HTML → text)
  - Large postings lists for common terms

- **Optimization:**
  - Use gap encoding for docID compression
  - Cache frequent terms in memory
  - Batch index operations

---

## Open Questions
- [ ] Should we implement stemming (e.g., Porter stemmer)?
- [ ] Should we support n-grams for partial matching?
- [ ] What's the maximum postings list size before we skip indexing?

---

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| go.uber.org/zap | v1.27.1 | Structured logging |
| go.etcd.io/bbolt | v1.4.3 | Index persistence |
| unicode | stdlib | Unicode tokenization |
| strings | stdlib | String manipulation |
| sync | stdlib | Concurrent access control |

---

## Implementation Order

1. **types.go** — Define all structs and interfaces
2. **tokenizer.go** — Implement Tokenizer with Unicode support
3. **postings.go** — Implement postings list operations
4. **compression.go** — Implement gap encoding/decoding
5. **index.go** — Implement InvertedIndex management
6. **indexer.go** — Implement main Indexer with persistence
7. **pkg/cli/index.go** — Wire up CLI commands

---

_Review this spec against the CLAUDE.md rules before implementation begins._

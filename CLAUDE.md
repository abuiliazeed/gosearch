# Agent Operating Instructions — gosearch

> Read this file at the start of every session. It is your operating manual.

---

## Project Context

**Project:** gosearch — A lightweight web search engine built from scratch in Go. It crawls web pages, builds a custom inverted index, and provides fast search capabilities with page ranking, boolean queries, fuzzy matching, and query caching.

**Stack:** Go 1.21+ | Colly (crawler) | Custom Inverted Index | Redis (cache) | BoltDB/Badger (storage) | Cobra (CLI)


---

## Go Code Standards

### Package Organization
- **internal/**: Private application code (cannot be imported by external projects)
- **pkg/**: Public libraries that could be imported by other projects
- **cmd/**: Main application entry points
- One package per directory
- Package names should be short, lowercase, single words

### Naming Conventions (Go Conventions)
- **Packages**: lowercase, single word — `crawler`, `indexer`, `search`
- **Exports**: PascalCase — `IndexDocument`, `SearchResults`
- **Internal**: camelCase — `tokenizeText`, `mergePostings`
- **Constants**: PascalCase or UPPER_SNAKE_CASE — `MaxWorkers` or `MAX_WORKERS`
- **Interfaces**: PascalCase with `-er` suffix — `Tokenizer`, `Searcher`, `Storer`
- **Files**: lowercase_with_underscores.go — `document_store.go`, `tfidf.go`
- **Booleans**: Prefix with `Is`, `Has`, `Should`, `Can` — `IsIndexed`, `HasError`

### File Organization
```go
// 1. Package declaration
package searcher

// 2. Standard library imports
import (
	"context"
	"fmt"
	"log"
)

// 3. Third-party imports
import (
	"github.com/redis/go-redis/v9"
)

// 4. Internal imports
import (
	"gosearch/internal/indexer"
	"gosearch/internal/ranker"
)

// 5. Constants
const DefaultLimit = 10

// 6. Type definitions
type Searcher struct {
	// fields
}

// 7. Interface definitions (if any)
type Tokenizer interface {
	Tokenize(text string) []string
}

// 8. Constructor
func NewSearcher(idx *indexer.Index) *Searcher {
	return &Searcher{idx: idx}
}

// 9. Public methods (exported)
func (s *Searcher) Search(ctx context.Context, query string) ([]Result, error) {
	// ...
}

// 10. Private methods (unexported)
func (s *Searcher) tokenizeQuery(query string) []string {
	// ...
}
```

### Error Handling
- **Always return errors** — never ignore them
- **Wrap errors with context** using `fmt.Errorf("operation failed: %w", err)`
- **Define error variables** for expected errors — `var ErrNotFound = errors.New("document not found")`
- **Use errors.Is** and `errors.As` for error checking

```go
func (s *Searcher) GetDocument(id string) (*Document, error) {
	doc, err := s.store.Get(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("document %s not found: %w", id, err)
		}
		return nil, fmt.Errorf("failed to get document %s: %w", id, err)
	}
	return doc, nil
}
```

### Context Usage
- **First parameter**: Always pass `context.Context` as the first parameter
- **Propagate context**: Always pass it down to called functions
- **Check cancellation**: Always check for context cancellation in long-running operations

```go
func (c *Crawler) Crawl(ctx context.Context, urls []string) error {
	for _, url := range urls {
		select {
		case <-ctx.Done():
			return ctx.Err() // Context cancelled
		default:
			// Continue processing
		}

		if err := c.crawlURL(ctx, url); err != nil {
			return err
		}
	}
	return nil
}
```

### Concurrency
- **Prefer channels over mutexes** for coordination
- **Use sync.WaitGroup** for waiting on multiple goroutines
- **Use errgroup.Group** for goroutines that can return errors
- **Limit concurrency** with worker pools or semaphores

```go
func (c *Crawler) CrawlParallel(ctx context.Context, urls []string) error {
	g, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, c.maxWorkers) // Semaphore

	for _, url := range urls {
		sem <- struct{}{} // Acquire
		url := url        // Create loop variable

		g.Go(func() error {
			defer func() { <-sem }() // Release
			return c.crawlURL(ctx, url)
		})
	}

	return g.Wait()
}
```

### Struct Initialization
- **Use named struct fields** in constructors for clarity
- **Provide constructor functions** — `NewTypeName()`
- **Zero value should be usable** when possible

```go
type Crawler struct {
	client    *http.Client
	maxDepth  int
	userAgent string
}

func NewCrawler(maxDepth int) *Crawler {
	return &Crawler{
		client:    &http.Client{Timeout: 30 * time.Second},
		maxDepth:  maxDepth,
		userAgent: "GoSearch/1.0",
	}
}
```

---

## Documentation Requirements

### Standard Documentation Level
- **Package comments**: Every package should have a doc comment
- **Exported functions**: All exported functions must have comments with `// FunctionName does...`
- **Context parameter**: Document what the context controls (cancellation, timeout)
- **Return errors**: Document what errors can be returned

```go
// Package searcher provides functionality for searching the inverted index.
//
// The Searcher type handles query parsing, boolean operations,
// fuzzy matching, and result ranking.
package searcher

// Search performs a search query against the index.
//
// The ctx parameter controls cancellation and timeout. The query string
// supports boolean operators (AND, OR, NOT) and phrase queries ("exact match").
//
// Returns a slice of Results sorted by relevance score, or an error if
// the query is invalid or the search fails.
func (s *Searcher) Search(ctx context.Context, query string) ([]Result, error) {
	// ...
}
```

---

## Testing Requirements

### Minimal Testing Level

| What | Type | Requirement |
|------|------|-------------|
| Build | Compile | Must compile with `go build` |
| Smoke | Manual | Core command works, no panics |

### Go Test Conventions
- Test file: `filename_test.go` (same package)
- Test function: `TestFunctionName(t *testing.T)`
- Table-driven tests for multiple cases

```go
func TestTokenize(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "simple text",
			text: "hello world",
			want: []string{"hello", "world"},
		},
		{
			name: "empty string",
			text: "",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Tokenize(tt.text)
			if !slices.Equal(got, tt.want) {
				t.Errorf("Tokenize() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

---

## Go Module & Dependency Management

### Adding Dependencies
```bash
go get github.com/package/name
go mod tidy
```

### Vendor (optional)
```bash
go mod vendor
```

### Version Pinning
- Use specific versions in `go.mod`
- Run `go mod tidy` after adding/removing imports

---

## CLI Standards (Cobra)

### Command Structure
```go
// root.go
var rootCmd = &cobra.Command{
	Use:   "gosearch",
	Short: "A lightweight web search engine",
	Long:  `gosearch crawls web pages and provides fast search capabilities.`,
}

// crawl.go
var crawlCmd = &cobra.Command{
	Use:   "crawl [urls...]",
	Short: "Crawl web pages and build the index",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Implementation
		return nil
	},
}

func init() {
	rootCmd.AddCommand(crawlCmd)
	crawlCmd.Flags().IntP("depth", "d", 3, "max crawl depth")
	crawlCmd.Flags().IntP("workers", "w", 10, "number of concurrent workers")
}
```

### Flag Naming
- **Short flags**: Single letter, lowercase — `-d`, `-w`
- **Long flags**: lowercase with hyphens — `--max-depth`
- **Persistent flags**: Available to subcommands

---

## Implementation Order

When building any feature, ALWAYS follow this order:

1. **Types first** — Define all structs and interfaces
2. **Storage layer** — File I/O, database access
3. **Business logic** — Core algorithms, indexing, ranking
4. **CLI commands** — Cobra command setup
5. **Wiring** — Connect components in main()
6. **Error handling** — Add proper error wrapping
7. **Documentation** — Add package and exported function docs
8. **Tests** — Unit tests for core logic

---

## NEVER Do (Hard Rules)

- ❌ Never ignore errors — always handle them
- ❌ Never use `time.Sleep` for synchronization — use channels or context
- ❌ Never close channels from the receiver side
- ❌ Never use goroutines without controlling their lifecycle
- ❌ Never ignore race detector warnings
- ❌ Never use `panic` for expected errors — return errors
- ❌ Never hardcode file paths — use config or constants
- ❌ Never leave `TODO` comments in committed code
- ❌ Never commit API keys or secrets
- ❌ Never skip input validation

## ALWAYS Do (Hard Rules)

- ✅ Always check for context cancellation in loops
- ✅ Always wrap errors with context
- ✅ Always use named struct fields in constructors
- ✅ Always handle errors from defer functions
- ✅ Always close resources (files, connections) with defer
- ✅ Always document exported types and functions
- ✅ Always run `go vet` and `go fmt` before committing
- ✅ Always update STATE.md at the end of every session

---

## Build Commands

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

<!-- mulch:start -->
## Project Expertise (Mulch)
<!-- mulch-onboard-v:1 -->

This project uses [Mulch](https://github.com/jayminwest/mulch) for structured expertise management.

**At the start of every session**, run:
```bash
mulch prime
```

This injects project-specific conventions, patterns, decisions, and other learnings into your context.
Use `mulch prime --files src/foo.ts` to load only records relevant to specific files.

**Before completing your task**, review your work for insights worth preserving — conventions discovered,
patterns applied, failures encountered, or decisions made — and record them:
```bash
mulch record <domain> --type <convention|pattern|failure|decision|reference|guide> --description "..."
```

Link evidence when available: `--evidence-commit <sha>`, `--evidence-bead <id>`

Run `mulch status` to check domain health and entry counts.
Run `mulch --help` for full usage.
Mulch write commands use file locking and atomic writes — multiple agents can safely record to the same domain concurrently.

### Before You Finish

1. Discover what to record:
   ```bash
   mulch learn
   ```
2. Store insights from this work session:
   ```bash
   mulch record <domain> --type <convention|pattern|failure|decision|reference|guide> --description "..."
   ```
3. Validate and commit:
   ```bash
   mulch sync
   ```
<!-- mulch:end -->

<!-- seeds:start -->
## Issue Tracking (Seeds)
<!-- seeds-onboard-v:1 -->

This project uses [Seeds](https://github.com/jayminwest/seeds) for git-native issue tracking.

**At the start of every session**, run:
```
sd prime
```

This injects session context: rules, command reference, and workflows.

**Quick reference:**
- `sd ready` — Find unblocked work
- `sd create --title "..." --type task --priority 2` — Create issue
- `sd update <id> --status in_progress` — Claim work
- `sd close <id>` — Complete work
- `sd dep add <id> <depends-on>` — Add dependency between issues
- `sd sync` — Sync with git (run before pushing)

### Before You Finish
1. Close completed issues: `sd close <id>`
2. File issues for remaining work: `sd create --title "..."`
3. Sync and push: `sd sync && git push`
<!-- seeds:end -->

<!-- canopy:start -->
## Prompt Management (Canopy)
<!-- canopy-onboard-v:1 -->

This project uses [Canopy](https://github.com/jayminwest/canopy) for git-native prompt management.

**At the start of every session**, run:
```
cn prime
```

This injects prompt workflow context: commands, conventions, and common workflows.

**Quick reference:**
- `cn list` — List all prompts
- `cn render <name>` — View rendered prompt (resolves inheritance)
- `cn emit --all` — Render prompts to files
- `cn update <name>` — Update a prompt (creates new version)
- `cn sync` — Stage and commit .canopy/ changes

**Do not manually edit emitted files.** Use `cn update` to modify prompts, then `cn emit` to regenerate.
<!-- canopy:end -->

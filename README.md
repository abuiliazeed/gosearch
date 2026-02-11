# gosearch

A lightweight web search engine built from scratch in Go.

## Features

- **Web Crawling**: Concurrent web crawler with Colly framework
- **Inverted Index**: Custom Go implementation with compression
- **Boolean Search**: Support for AND, OR, NOT operators
- **Fuzzy Matching**: Levenshtein distance for typo tolerance
- **Page Ranking**: TF-IDF scoring with optional PageRank
- **Query Caching**: Redis-based result caching
- **CLI Interface**: Command-line interface with Cobra

## Installation

### Prerequisites

- Go 1.21 or later
- Redis (optional, for query caching)
- BoltDB (embedded, no setup required)

### Build from source

```bash
# Clone the repository
git clone https://github.com/abuiliazeed/gosearch.git
cd gosearch

# Download dependencies
go mod download

# Build the binary
go build -o bin/gosearch ./cmd/gosearch

# Or use make
make build
```

## Usage

### Crawling web pages

```bash
# Crawl a single URL
./bin/gosearch crawl https://example.com

# Crawl with custom settings
./bin/gosearch crawl https://example.com \
  --depth 3 \
  --workers 10 \
  --delay 1000

# Crawl multiple URLs
./bin/gosearch crawl https://example.com https://another.com \
  --workers 20
```

### Searching

```bash
# Simple search
./bin/gosearch search "golang tutorial"

# Boolean query
./bin/gosearch search "golang AND tutorial"

# Phrase query
./bin/gosearch search "\"golang tutorial\""

# Fuzzy search
./bin/gosearch search "golang~1" --fuzzy
```

### Index management

```bash
# Build index from crawled pages
./bin/gosearch index build

# Show index statistics
./bin/gosearch index stats

# Clear the index
./bin/gosearch index clear
```

## Configuration

gosearch can be configured via:

1. **Command-line flags** (highest priority)
2. **Environment variables** (prefixed with `GOSEARCH_`)
3. **Config file** (`~/.gosearch.yaml` or `.gosearch.yaml`)

### Environment variables

```bash
export GOSEARCH_DATA_DIR=./data
export GOSEARCH_MAX_WORKERS=10
export GOSEARCH_REDIS_HOST=localhost:6379
```

### Config file example

```yaml
data-dir: ./data
max-workers: 10
max-depth: 3
log-format: text
log-level: info
redis:
  host: localhost:6379
  password: ""
  cache-ttl: 3600
```

## Architecture

```
gosearch/
├── cmd/gosearch/          # Main application
├── internal/              # Private packages
│   ├── crawler/           # Web crawler
│   ├── indexer/           # Inverted index
│   ├── search/            # Query processor
│   ├── ranker/            # Ranking algorithms
│   └── storage/           # Data persistence
└── pkg/                   # Public packages
    ├── cli/               # CLI commands
    └── config/            # Configuration
```

## Development

### Running tests

```bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Run benchmarks
go test -bench=. -benchmem ./...
```

### Code quality

```bash
# Format code
go fmt ./...

# Run vet
go vet ./...

# Run linter (requires golangci-lint)
golangci-lint run

# Run pre-deploy checks
bash scripts/pre-deploy.sh
```

### Makefile commands

```bash
make help        # Show all available commands
make build       # Build the binary
make run         # Build and run
make test        # Run tests
make race        # Run tests with race detector
make fmt         # Format code
make vet         # Run go vet
make clean       # Clean build artifacts
```

## Contributing

Contributions are welcome! Please see `docs/AGENT_SESSION_TEMPLATE.md` for how to structure your work.

1. Read `PROJECT.md` for architecture decisions
2. Read `CLAUDE.md` for coding standards
3. Create a feature spec using `docs/FEATURE_SPEC_TEMPLATE.md`
4. Follow the implementation order from `CLAUDE.md`
5. Run `scripts/pre-deploy.sh` before committing

## License

MIT License - see LICENSE file for details

## Roadmap

See `ROADMAP.md` for the development roadmap and current progress.

# Contributing to gosearch

Thank you for your interest in contributing to gosearch! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Development Setup

### Prerequisites

- **Go 1.24.2 or later** - [Install Go](https://golang.org/dl/)
- **Docker** (recommended) - For running Redis caching service
- **git** - For version control

### Clone and Build

```bash
# Clone the repository
git clone https://github.com/abuiliazeed/gosearch.git
cd gosearch

# Download dependencies
go mod download

# Build the binary
go build -o bin/gosearch ./cmd/gosearch

# Run tests
go test ./...
```

### Running Redis (Optional)

Redis is used for query result caching. It's optional but recommended for production use.

```bash
# Using Docker
docker run -d -p 6379:6379 redis:alpine

# Or using docker-compose (if available)
docker-compose up -d
```

## How to Contribute

We welcome contributions in the following areas:

### Reporting Bugs

1. Check existing issues to avoid duplicates
2. Use the issue template and provide:
   - Go version (`go version`)
   - Steps to reproduce
   - Expected vs actual behavior
   - Relevant logs or error messages

### Suggesting Features

1. Check existing feature requests
2. Describe the use case clearly
3. Explain why the feature would be useful
4. Consider if it fits the project's scope

### Submitting Code

#### Before You Start

1. Read `PROJECT.md` for architecture context
2. Read `CLAUDE.md` for coding standards and conventions
3. Check `ROADMAP.md` for planned features

#### Making Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Write clean, well-documented code following project conventions
4. Add tests for new functionality
5. Ensure all tests pass (`go test ./...`)
6. Run quality checks (`make fmt`, `make vet`, `make lint`)

#### Commit Guidelines

We use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation only changes
- `style` - Code style changes (formatting, semi-colons, etc)
- `refactor` - Code refactoring
- `test` - Adding or updating tests
- `chore` - Maintenance tasks
- `perf` - Performance improvements

**Examples:**
```
feat(crawler): add support for sitemap parsing
fix(indexer): handle empty documents gracefully
docs(readme): update installation instructions
test(search): add tests for fuzzy matching
```

## Coding Standards

### Go Conventions

Follow the standards outlined in `CLAUDE.md`:

- **Package names**: lowercase, single word
- **Exported names**: PascalCase (`IndexDocument`)
- **Internal names**: camelCase (`tokenizeText`)
- **Constants**: PascalCase or UPPER_SNAKE_CASE
- **Interfaces**: PascalCase with `-er` suffix (`Tokenizer`)
- **Files**: lowercase_with_underscores.go

### File Organization

```go
// 1. Package declaration
package searcher

// 2. Standard library imports
import (
    "context"
    "fmt"
)

// 3. Third-party imports
import (
    "github.com/redis/go-redis/v9"
)

// 4. Internal imports
import (
    "gosearch/internal/indexer"
)

// 5. Constants
const DefaultLimit = 10

// 6. Type definitions
type Searcher struct { ... }

// 7. Interface definitions (if any)
type Tokenizer interface { ... }

// 8. Constructor
func NewSearcher(...) *Searcher { ... }

// 9. Public methods
func (s *Searcher) Search(...) { ... }

// 10. Private methods
func (s *Searcher) tokenizeQuery(...) { ... }
```

### Error Handling

- **Always return errors** - never ignore them
- **Wrap errors with context** using `fmt.Errorf("operation failed: %w", err)`
- **Define error variables** for expected errors
- **Check context cancellation** in long-running operations

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

### Documentation

- **Package comments**: Every package should have a doc comment
- **Exported functions**: All exported functions must have comments
- **Context parameter**: Document what the context controls
- **Return errors**: Document what errors can be returned

```go
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

## Pull Request Process

1. **Update documentation** - If your changes affect user-facing behavior
2. **Write tests** - Ensure adequate test coverage for new code
3. **Run quality gates**:
   ```bash
   make fmt        # Format code
   make vet        # Run go vet
   make lint       # Run linter
   make test       # Run tests
   make race       # Run tests with race detector
   ```
4. **Commit cleanly** - Use conventional commit format
5. **Push to your fork** and create a pull request

### Pull Request Description

Please include:
- Summary of changes
- Motivation and context
- Related issues (fixes #123)
- Screenshots (if applicable)
- Breaking changes (if any)

## Testing

We use table-driven tests for multiple cases:

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

## Development Workflow

### Running Tests

```bash
# Run all tests
make test

# Run with race detector
make race

# Run benchmarks
make bench

# Generate coverage report
make cover
```

### Before Committing

```bash
# Format code
go fmt ./...

# Run vet
go vet ./...

# Run linter (requires golangci-lint)
make lint

# Run pre-deploy script
bash scripts/pre-deploy.sh
```

## Project Structure

```
gosearch/
├── cmd/gosearch/          # Main application
├── internal/              # Private packages
│   ├── crawler/           # Web crawler
│   ├── indexer/           # Inverted index
│   ├── search/            # Query processor
│   ├── ranker/            # Ranking algorithms
│   └── storage/           # Data persistence
├── pkg/                   # Public packages
│   ├── cli/               # CLI commands
│   └── config/            # Configuration
├── docs/                  # Documentation
└── scripts/               # Build and test scripts
```

## Getting Help

- **Documentation**: Check `README.md`, `PROJECT.md`, and `ROADMAP.md`
- **Issues**: Search or create [GitHub Issues](https://github.com/abuiliazeed/gosearch/issues)
- **Discussions**: Use [GitHub Discussions](https://github.com/abuiliazeed/gosearch/discussions) for questions

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).

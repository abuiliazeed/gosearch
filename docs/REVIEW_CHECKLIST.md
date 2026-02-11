# Code Review Checklist — gosearch

> **Review Strictness:** Standard
> Use this checklist when running the Reviewer Agent on completed code.

---

## How to Use

Copy this prompt to your code review agent:

```
You are a senior code reviewer for gosearch (a Go search engine).
Review the following code changes against this checklist.

Classify each issue as:
- 🔴 CRITICAL — Must fix before merge
- 🟡 WARNING — Should fix, can defer with documented reason
- 🔵 SUGGESTION — Nice to have, optional

Rules: Zero CRITICAL issues. All WARNING issues must be acknowledged (fixed or documented).
```

---

## 1. Logic & Correctness

- [ ] Does the code match ALL acceptance criteria from the feature spec?
- [ ] Are all conditionals correct? (no inverted logic, no off-by-one)
- [ ] Are async operations properly handled with context?
- [ ] Are race conditions possible? (shared state, concurrent access)
- [ ] Are all code paths reachable? (no dead code)
- [ ] Do loops terminate correctly? (no infinite loops, correct bounds)
- [ ] Is channel usage correct? (sender closes, receiver checks for closed)

## 2. Go Types & Code Style

- [ ] No exported functions without documentation comments
- [ ] All package names are short, lowercase, single words
- [ ] Error variables defined with `errors.New` for expected errors
- [ ] Errors are wrapped with context using `fmt.Errorf("...: %w", err)`
- [ ] Context is always the first parameter
- [ ] Struct fields are documented if not obvious
- [ ] File naming follows `lowercase_with_underscores.go` convention
- [ ] Import order: stdlib, third-party, internal (grouped with blank lines)

## 3. Error Handling

- [ ] Every error is checked and handled (never ignored)
- [ ] Errors are wrapped with context, not discarded
- [ ] Expected errors use error variables for `errors.Is`
- [ ] Context cancellation is checked in long-running operations
- [ ] Defer functions handle errors appropriately
- [ ] Resource cleanup is deferred (files, connections, channels)
- [ ] Error messages are helpful and actionable

## 4. Concurrency & Goroutines

- [ ] Goroutines are properly managed (no leaks)
- [ ] Context is used for cancellation signals
- [ ] Channels are used correctly (sender closes, receiver range)
- [ ] Mutex usage is appropriate (prefer channels for coordination)
- [ ] Race detector produces no warnings
- [ ] Worker pools limit concurrency appropriately
- [ ] errgroup.Group is used for concurrent operations with errors

## 5. Package Organization

- [ ] Internal code is in `internal/` (cannot be externally imported)
- [ ] Reusable code is in `pkg/` (public API)
- [ ] One package per directory
- [ ] No circular dependencies between packages
- [ ] Package comments explain the package's purpose
- [ ] Exported types and functions have documentation

## 6. Storage & Persistence

- [ ] File operations use defer for closing
- [ ] Database connections are properly closed
- [ ] Transactions are used where appropriate
- [ ] Cache invalidation is handled correctly
- [ ] Backup/restore considerations are addressed
- [ ] Concurrent access to storage is synchronized

## 7. CLI & Commands (if applicable)

- [ ] Command follows Cobra conventions
- [ ] Flags have short and long versions where appropriate
- [ ] Flag names use hyphens (`--max-depth`)
- [ ] Command validates arguments before processing
- [ ] Help text is clear and accurate
- [ ] Persistent flags are used correctly

## 8. Performance & Resources

- [ ] No memory leaks (goroutines, growing slices)
- [ ] Large buffers are pre-allocated (not append in loop)
- [ ] Expensive operations are cached where appropriate
- [ ] No unnecessary allocations in hot paths
- [ ] Streaming is used for large data (not all in memory)
- [ ] Context has appropriate timeouts

## 9. Security

- [ ] Input validation is performed (user input, URLs, parameters)
- [ ] SQL/NoSQL injection prevention (parameterized queries)
- [ ] No hardcoded secrets or API keys
- [ ] File paths are validated (path traversal prevention)
- [ ] Rate limiting is implemented for external requests
- [ ] robots.txt is respected for web crawling

## 10. Testing

- [ ] Core functions have unit tests
- [ ] Tests are in `*_test.go` files
- [ ] Test names are descriptive: `TestFunctionName`
- [ ] Table-driven tests for multiple cases
- [ ] Race detector runs clean: `go test -race`
- [ ] Benchmark tests for performance-critical code

## 11. Documentation

- [ ] Package has a package comment
- [ ] All exported functions have comments
- [ ] All exported types have comments
- [ ] Complex algorithms have inline comments
- [ ] Context parameter is documented (what it controls)
- [ ] Return errors are documented (what errors can occur)

## 12. Code Style & Standards

- [ ] `go fmt` produces no changes
- [ ] `go vet` produces no warnings
- [ ] No unused imports or variables
- [ ] No blank or commented-out code
- [ ] Constants are used for magic numbers
- [ ] Error checks are immediate, not deferred

---

## Review Output Template

```markdown
## Code Review: [Feature Name]

### Summary
[1-2 sentence summary of the review]

### 🔴 CRITICAL Issues
1. [Issue description + file:line + suggested fix]

### 🟡 WARNING Issues
1. [Issue description + file:line + suggested fix]

### 🔵 SUGGESTIONS
1. [Suggestion description + file:line]

### ✅ What Looks Good
- [Positive feedback on well-implemented aspects]

### Verdict
[PASS / PASS WITH CONDITIONS / FAIL]
```

---

## Go-Specific Review Commands

```bash
# Format check
go fmt ./...

# Vet check
go vet ./...

# Race detector
go test -race ./...

# Run all tests
go test ./...

# Benchmark (if performance-critical)
go test -bench=. -benchmem ./...

# Lint (if golangci-lint is installed)
golangci-lint run
```

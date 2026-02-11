# AI Coding Session Prompt — gosearch

> Use this prompt template at the start of each AI coding session.

---

## Required Pre-Session Reading

Read the following files before doing anything:

| File | Purpose |
|------|---------|
| `PROJECT.md` | Architecture decisions, tech stack, file structure |
| `CLAUDE.md` | Go coding standards, naming conventions, rules |
| `ROADMAP.md` | Current sprint status, what needs to be built |
| `STATE.md` | What was done last session, known issues, decisions |

---

## Optional Reference Documentation

| File | When to Read |
|------|--------------|
| `docs/seeding-strategies.md` | When implementing seed URL selection or crawler enhancements |
| `docs/antiblocking-strategies.md` | When implementing anti-blocking features or rate limiting |

---

## Session Template

Copy and fill in this template for each session:

```
Read the following files before doing anything:
- PROJECT.md (architecture decisions, tech stack)
- CLAUDE.md (Go coding standards, rules)
- ROADMAP.md (what needs to be built)
- STATE.md (what was done last session, known issues)

Today's task: [DESCRIBE WHAT YOU'RE BUILDING OR FIXING]

Before writing any code:
1. Create a feature spec (or reference the existing one in docs/)
2. List all files you plan to create or modify
3. Confirm your approach with me

Follow the implementation order from CLAUDE.md:
Types → Storage → Business Logic → CLI Commands → Wiring → Error Handling → Docs → Tests

At the end of the session:
- Update STATE.md with changes made
- Update ROADMAP.md if tasks were completed
- Run `go build ./...` to verify no compilation errors
```

---

## Example Session Prompts

### For New Feature Implementation
```
Today's task: Implement proxy rotation support for the crawler

Before writing any code:
1. Create a feature spec in docs/proxy-rotation-spec.md
2. Files to create/modify:
   - internal/crawler/proxy_rotator.go (new)
   - internal/crawler/crawler.go (modify)
   - pkg/cli/crawl.go (add --proxy flag)
3. Approach: Create ProxyRotator with health tracking, integrate with CollyCrawler

Follow implementation order: Types → Storage → Business Logic → CLI Commands
```

### For Bug Fixes
```
Today's task: Fix context cancellation issue in crawler shutdown

Files to modify:
- internal/crawler/crawler.go (fix shutdown logic)
- pkg/cli/crawl.go (improve signal handling)

Approach: Ensure goroutines properly exit when context is canceled
```

### For Documentation
```
Today's task: Update ROADMAP.md with completed anti-blocking features

Files to modify:
- ROADMAP.md

Approach: Mark Phase 5 Priority 1 items as complete, update phase tracking
```

---

## Quick Reference Commands

```bash
# Build the project
go build ./...

# Run the binary
./gosearch [command]

# Run tests
go test ./...

# Format code
go fmt ./...

# Vet code
go vet ./...

# Lint (requires golangci-lint)
golangci-lint run

# Run with race detector
go run -race ./cmd/gosearch
```

---

## Key Implementation Principles

1. **Follow the implementation order** from CLAUDE.md strictly
2. **Always handle errors** — never ignore them
3. **Use context.Context** as first parameter for long-running operations
4. **Check for context cancellation** in loops
5. **Wrap errors with context** using `fmt.Errorf("operation: %w", err)`
6. **Document exported types and functions**
7. **Run `go build`** before considering work complete
8. **Update STATE.md** at the end of every session

---

## Common Workflows

### Adding a New CLI Command
1. Create `pkg/cli/commandname.go`
2. Add command to `rootCmd` in `init()`
3. Add flags with appropriate types
4. Implement `RunE` function
5. Test with `./gosearch commandname --help`

### Adding a New Internal Module
1. Create `internal/modulename/modulename.go`
2. Define types and interfaces first
3. Implement business logic
4. Add tests in `internal/modulename/modulename_test.go`
5. Wire up in CLI or other modules

### Modifying Existing Module
1. Read the module files first
2. Understand current implementation
3. Make changes following existing patterns
4. Update related tests
5. Build and verify

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Build fails | Check for missing imports, type mismatches, unused variables |
| Tests fail | Read error messages, check test expectations, verify logic |
| Import errors | Run `go mod tidy` to clean up dependencies |
| Context cancellation | Ensure `select` with `ctx.Done()` is in loops |
| Race conditions | Run with `-race` flag, use mutexes or channels properly |

---

*Last updated: 2026-02-11*

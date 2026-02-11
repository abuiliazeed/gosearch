# Agent Session Template — gosearch

> Copy and paste this prompt at the start of every AI coding session.
> Replace the bracketed sections with specifics for today's work.

---

## Session Start Prompt

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

After completing the work:
1. Verify code compiles: `go build ./...`
2. Run tests: `go test ./...`
3. Run race detector: `go test -race ./...`
4. Format code: `go fmt ./...`
5. Vet code: `go vet ./...`
6. Self-review against docs/REVIEW_CHECKLIST.md
7. Update STATE.md with:
   - What was done
   - Current status of features
   - Any new known issues
   - What to do next session
8. Update go.mod if dependencies changed: `go mod tidy`
```

---

## Session End Checklist

Before ending any agent session, verify:

- [ ] All code compiles: `go build ./...`
- [ ] All tests pass: `go test ./...`
- [ ] No race conditions: `go test -race ./...`
- [ ] Code is formatted: `go fmt ./...`
- [ ] Code passes vet: `go vet ./...`
- [ ] Exported functions have documentation
- [ ] Errors are properly wrapped with context
- [ ] Context is propagated correctly
- [ ] STATE.md is updated with today's work
- [ ] Go module is tidy: `go mod tidy`

---

## Bug Fix Session Prompt

```
Read PROJECT.md, CLAUDE.md, and STATE.md.

Bug to fix: [DESCRIBE THE BUG]

Steps to reproduce:
1. [Step 1]
2. [Step 2]
3. [Expected result vs actual result]

Before fixing:
1. Write a failing test that reproduces the bug
2. Identify the root cause (not just the symptom)
3. Explain the fix approach before implementing

After fixing:
1. Confirm the failing test now passes
2. Run full test suite to check for regressions
3. Run race detector: `go test -race ./...`
4. Update STATE.md with the fix
5. Commit: `git commit -m "fix: [description]"`
```

---

## Refactor Session Prompt

```
Read PROJECT.md, CLAUDE.md, and STATE.md.

Refactor target: [WHAT NEEDS REFACTORING AND WHY]

Rules:
1. No behavior changes — all existing tests must still pass
2. Refactor in small, committed steps
3. Run tests after each step
4. If a test fails, revert the last change and try a different approach

After refactoring:
1. All tests pass (zero regressions)
2. No race conditions: `go test -race ./...`
3. Code is cleaner (better organization, fewer lines)
4. Update STATE.md
5. Commit: `git commit -m "refactor: [description]"`
```

---

## New Feature Session Prompt

```
Read PROJECT.md, CLAUDE.md, ROADMAP.md, and STATE.md.

Feature to build: [FEATURE NAME from ROADMAP.md]

Step 1: Planning
1. Create a feature spec using docs/FEATURE_SPEC_TEMPLATE.md
2. Define all Go types and interfaces
3. Plan the package structure (which internal/ pkg/ dirs)

Step 2: Implementation
Follow this order:
1. Types — Define structs and interfaces
2. Storage — File I/O, database access
3. Business Logic — Core algorithms
4. CLI Commands — Cobra command setup
5. Wiring — Connect components in main()
6. Error Handling — Add proper error wrapping
7. Documentation — Package and exported function docs

Step 3: Verification
1. Compile check: `go build ./...`
2. Run tests: `go test ./...`
3. Race detector: `go test -race ./...`
4. Format: `go fmt ./...`
5. Vet: `go vet ./...`

Step 4: Review
1. Self-review against docs/REVIEW_CHECKLIST.md
2. Update STATE.md
3. Commit: `git commit -m "feat: [description]"`
```

---

## Performance Optimization Session Prompt

```
Read PROJECT.md, CLAUDE.md, and STATE.md.

Performance issue: [DESCRIBE THE BOTTLENECK]

Before optimizing:
1. Run benchmarks: `go test -bench=. -benchmem ./...`
2. Profile with pprof: `go test -cpuprofile=cpu.prof -memprofile=mem.prof ./...`
3. Identify the hot path

After optimizing:
1. Verify benchmarks improved
2. Compare before/after profiles
3. Ensure no regressions in correctness
4. Run race detector: `go test -race ./...`
5. Update STATE.md with optimization notes
6. Commit: `git commit -m "perf: [description]"`
```

---

## Dependency Update Session Prompt

```
Read STATE.md and go.mod.

Dependency updates needed:
1. Check for outdated: `go list -u -m all`
2. Review release notes for breaking changes
3. Update dependencies: `go get -u ./...`
4. Tidy module: `go mod tidy`
5. Run tests to verify compatibility
6. Update STATE.md with dependency changes
7. Commit: `git commit -m "chore: update dependencies"`
```

---

## Multi-Agent Pipeline Prompts

### Agent 1: Architect
```
You are the Architect Agent for gosearch.
Read PROJECT.md and CLAUDE.md.

Task: Create a feature spec for [FEATURE NAME].

Use the template in docs/FEATURE_SPEC_TEMPLATE.md.
Fill in every section. Do not leave placeholders.
Pay special attention to:
- Complete Go type definitions with proper documentation
- Package organization (internal/ vs pkg/)
- Function signatures with context as first parameter
- Error handling strategy (error variables, wrapping)
- All edge cases and concurrency considerations

Save the spec to: docs/specs/[feature-name].md
```

### Agent 2: Builder
```
You are the Builder Agent for gosearch.
Read PROJECT.md, CLAUDE.md, ROADMAP.md, and STATE.md.
Read the feature spec: docs/specs/[feature-name].md

Implement the feature following CLAUDE.md rules exactly.
Follow the implementation order: Types → Storage → Business Logic → CLI Commands → Wiring.

Requirements:
- All code must compile with `go build ./...`
- Context is always the first parameter
- Errors are wrapped with context
- Exported functions have documentation
- Do NOT use panic for expected errors
- Do NOT ignore errors
```

### Agent 3: Reviewer
```
You are the Code Reviewer Agent for gosearch.
Read CLAUDE.md and docs/REVIEW_CHECKLIST.md.
Read the feature spec: docs/specs/[feature-name].md

Review all changed files against:
1. The acceptance criteria in the spec
2. Every item in the REVIEW_CHECKLIST.md
3. The CLAUDE.md coding standards

Output your review using the Review Output Template at the bottom of REVIEW_CHECKLIST.md.
Classify issues as 🔴 CRITICAL, 🟡 WARNING, or 🔵 SUGGESTION.

Verdict must be FAIL if any CRITICAL issues exist.
```

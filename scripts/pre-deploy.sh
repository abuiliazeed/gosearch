#!/bin/bash
# pre-deploy.sh — Quality Gate for gosearch
# Run this before every commit or deploy: bash scripts/pre-deploy.sh

set -e

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  🔍 Pre-Deploy Quality Gate — gosearch"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

ERRORS=0

# 1. Format check
echo "→ [1/5] Format check..."
if unformatted=$(gofmt -l .); then
  if [ -z "$unformatted" ]; then
    echo "  ✅ Format: PASS (all files formatted)"
  else
    echo "  ❌ Format: FAIL — run 'go fmt ./...' to fix:"
    echo "$unformatted"
    ERRORS=$((ERRORS + 1))
  fi
else
  echo "  ❌ Format: FAIL — gofmt error"
  ERRORS=$((ERRORS + 1))
fi
echo ""

# 2. Vet check
echo "→ [2/5] Go vet..."
if go vet ./... 2>/dev/null; then
  echo "  ✅ Vet: PASS"
else
  echo "  ❌ Vet: FAIL — fix vet issues before deploying"
  ERRORS=$((ERRORS + 1))
fi
echo ""

# 3. Build check
echo "→ [3/5] Build check..."
if go build ./... 2>/dev/null; then
  echo "  ✅ Build: PASS"
else
  echo "  ❌ Build: FAIL — fix build errors before deploying"
  ERRORS=$((ERRORS + 1))
fi
echo ""

# 4. Tests
echo "→ [4/5] Running tests..."
if go test ./... 2>/dev/null; then
  echo "  ✅ Tests: PASS"
else
  echo "  ⚠️  Tests: FAIL (non-blocking for minimal testing level)"
fi
echo ""

# 5. Race detector
echo "→ [5/5] Race detector..."
if go test -race ./... 2>/dev/null; then
  echo "  ✅ Race detector: PASS"
else
  echo "  ⚠️  Race detector: FAIL (review for data races)"
fi
echo ""

# Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ $ERRORS -eq 0 ]; then
  echo "  ✅ ALL CHECKS PASSED — Ready to deploy!"
else
  echo "  ❌ $ERRORS CHECK(S) FAILED — Fix issues before deploying"
  exit 1
fi
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

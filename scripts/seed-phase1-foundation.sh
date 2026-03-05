#!/bin/bash
# gosearch Seeding Script - Phase 1: Foundation
# Crawl critical Apple + Go documentation

set -e  # Exit on error

GOSEARCH_DIR="$HOME/developer/gosearch"
SEEDS_DIR="$GOSEARCH_DIR/seeds"
MAX_QUEUE=2000
DEPTH=3
WORKERS=20
DELAY=1000

echo "=========================================="
echo "gosearch Seeding - Phase 1: Foundation"
echo "=========================================="
echo ""
echo "This will crawl Apple + Go documentation:"
echo "  - Swift/SwiftUI/Core Data/SpriteKit"
echo "  - Go language + tools"
echo "  - Official Apple developer docs"
echo ""
echo "Settings:"
echo "  - Max URLs: $MAX_QUEUE"
echo "  - Depth: $DEPTH"
echo "  - Workers: $WORKERS"
echo "  - Delay: ${DELAY}ms"
echo ""

# Check if gosearch binary exists
if [ ! -f "$GOSEARCH_DIR/bin/gosearch" ]; then
    echo "❌ Error: gosearch binary not found at $GOSEARCH_DIR/bin/gosearch"
    echo "   Run: cd $GOSEARCH_DIR && go build -o bin/gosearch ./cmd/gosearch"
    exit 1
fi

# Check if seed file exists
SEED_FILE="$SEEDS_DIR/foundation.yml"
if [ ! -f "$SEED_FILE" ]; then
    echo "❌ Error: Seed file not found at $SEED_FILE"
    exit 1
fi

echo "✅ All checks passed"
echo ""
echo "Starting crawl with seed file: $SEED_FILE"
echo ""

# Run the crawl
cd "$GOSEARCH_DIR"
./bin/gosearch crawl \
    --seeds-file "$SEED_FILE" \
    --max-queue "$MAX_QUEUE" \
    -L "$DEPTH" \
    -w "$WORKERS" \
    -d "$DELAY"

# Check exit status
if [ $? -eq 0 ]; then
    echo ""
    echo "=========================================="
    echo "✅ Crawl completed successfully!"
    echo "=========================================="
    echo ""
    echo "Check stats with:"
    echo "  cd $GOSEARCH_DIR && ./bin/gosearch index stats"
    echo ""
    echo "Test search with:"
    echo "  ./bin/gosearch search 'SwiftUI' --limit 5"
    echo "  ./bin/gosearch search 'Core Data' --limit 5"
    echo ""
    echo "Next: Run Phase 2 (AI knowledge) with:"
    echo "  ./bin/gosearch crawl --seeds-file $SEEDS_DIR/ai-knowledge.yml -L 2 -w 15 --max-queue 2000"
else
    echo ""
    echo "=========================================="
    echo "❌ Crawl failed with exit code $?"
    echo "=========================================="
    echo ""
    echo "Check logs for details"
fi

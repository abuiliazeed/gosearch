package search

import (
	"context"
	"os"
	"testing"

	"github.com/abuiliazeed/gosearch/internal/indexer"
	"github.com/abuiliazeed/gosearch/internal/ranker"
	"github.com/abuiliazeed/gosearch/internal/storage"
	"go.uber.org/zap"
)

func TestSearcher_Search(t *testing.T) {
	// Setup dependencies
	tmpDir, _ := os.MkdirTemp("", "searcher_test")
	defer os.RemoveAll(tmpDir)

	storePath := tmpDir + "/index.db"
	store, _ := storage.NewIndexStore(storePath)
	defer store.Close()

	logger := zap.NewNop()
	idx := indexer.NewIndexer(store, logger)

	// Index some documents
	docs := []*storage.Document{
		{ID: "1", Title: "Go Guide", ContentMarkdown: "Go is a statically typed language"},
		{ID: "2", Title: "Python Guide", ContentMarkdown: "Python is a dynamic language"},
		{ID: "3", Title: "Rust Guide", ContentMarkdown: "Rust is a systems language"},
	}

	ctx := context.Background()
	for _, doc := range docs {
		idx.IndexDocument(ctx, doc)
	}

	// Create Scorer
	// We need access to the underlying InvertedIndex for TFIDF
	// idx is *Indexer. GetIndex() returns *Index. GetIndex() on *Index returns *InvertedIndex.
	invertedIndex := idx.GetIndex().GetIndex()
	tfidf := ranker.NewTFIDF(invertedIndex)
	pr := ranker.DefaultPageRank()
	scorer := ranker.NewScorer(tfidf, pr, nil)

	// Create dummy doc store (nil might work if searcher doesn't use it for pure IDs)
	// But usually searcher might fetch titles/snippets.
	// Let's create a real one.

	// NewSearcher takes *Index
	s := NewSearcher(idx.GetIndex(), scorer, nil, nil) // nil cache, default config

	// Test Search
	results, err := s.Search(ctx, "Go")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results.Results) != 1 {
		t.Errorf("expected 1 hit for 'Go', got %d", len(results.Results))
	}
	if results.Results[0].DocID != "1" {
		t.Errorf("expected doc 1, got %s", results.Results[0].DocID)
	}

	// Test Boolean AND
	results, err = s.Search(ctx, "statically AND typed")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results.Results) != 1 {
		t.Errorf("expected 1 hit, got %d", len(results.Results))
	}
}

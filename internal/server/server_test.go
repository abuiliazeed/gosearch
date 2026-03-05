package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"go.uber.org/zap"

	"github.com/abuiliazeed/gosearch/internal/indexer"
	"github.com/abuiliazeed/gosearch/internal/ranker"
	"github.com/abuiliazeed/gosearch/internal/search"
	"github.com/abuiliazeed/gosearch/internal/storage"
)

func TestServer_Health(t *testing.T) {
	// Minimal setup
	// Check server.go HandleHealth implementation via view_file if unsure,
	// but NewHandlers creates it.
	// Let's do full setup via NewServer to be safe and integration-like.

	tmpDir, _ := os.MkdirTemp("", "server_test")
	defer os.RemoveAll(tmpDir)

	storePath := tmpDir + "/index.db"
	store, _ := storage.NewIndexStore(storePath)
	defer store.Close()

	logger := zap.NewNop()
	idx := indexer.NewIndexer(store, logger)

	invertedIndex := idx.GetIndex().GetIndex()
	tfidf := ranker.NewTFIDF(invertedIndex)
	pr := ranker.DefaultPageRank()
	scorer := ranker.NewScorer(tfidf, pr, nil)

	searcher := search.NewSearcher(idx.GetIndex(), scorer, nil, nil)
	docStore, _ := storage.NewDocumentStore(tmpDir) // Needed by NewServer? Yes.

	srv := NewServer(nil, idx, searcher, docStore)

	// Create request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// We need to serve via srv.handlers or srv.server.Handler
	// srv.server.Handler is configured in NewServer
	handler := srv.server.Handler

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["status"] != "ok" {
		t.Errorf("expected status ok")
	}
}

func TestServer_Search(t *testing.T) {
	// Full setup
	tmpDir, _ := os.MkdirTemp("", "server_search_test")
	defer os.RemoveAll(tmpDir)

	storePath := tmpDir + "/index.db"
	store, _ := storage.NewIndexStore(storePath)
	defer store.Close()

	logger := zap.NewNop()
	idx := indexer.NewIndexer(store, logger)

	// Index doc
	ctx := context.Background()
	doc := &storage.Document{ID: "1", Title: "Go Test", ContentMarkdown: "Golang testing"}
	idx.IndexDocument(ctx, doc)

	invertedIndex := idx.GetIndex().GetIndex()
	tfidf := ranker.NewTFIDF(invertedIndex)
	pr := ranker.DefaultPageRank()
	scorer := ranker.NewScorer(tfidf, pr, nil)

	searcher := search.NewSearcher(idx.GetIndex(), scorer, nil, nil)
	docStore, _ := storage.NewDocumentStore(tmpDir)

	srv := NewServer(nil, idx, searcher, docStore)

	// Search request
	req := httptest.NewRequest("GET", "/api/v1/search?q=Golang", nil)
	w := httptest.NewRecorder()

	srv.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Inspect response
	// The response should match SearchResponse JSON structure
	var resp struct {
		Results []interface{} `json:"results"`
		Total   int           `json:"total_count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Total != 1 {
		t.Errorf("expected 1 result, got %d", resp.Total)
	}
}

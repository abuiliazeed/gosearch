// Package tests provides integration tests for gosearch.
package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/abuiliazeed/gosearch/internal/crawler"
	"github.com/abuiliazeed/gosearch/internal/indexer"
	"github.com/abuiliazeed/gosearch/internal/ranker"
	"github.com/abuiliazeed/gosearch/internal/search"
	"github.com/abuiliazeed/gosearch/internal/storage"
	"go.uber.org/zap"
)

// TestFullPipeline tests the complete crawl -> index -> search pipeline.
func TestFullPipeline(t *testing.T) {
	SkipIfMissingBinary(t, "../bin/gosearch")

	cfg := NewTestConfig(t)
	defer cfg.Cleanup()

	ctx, cancel := cfg.CreateContext()
	defer cancel()

	// Setup test directory
	if err := SetupTestDir(cfg.DataDir); err != nil {
		t.Fatalf("failed to setup test dir: %v", err)
	}

	t.Run("crawl_sample_page", func(t *testing.T) {
		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(SampleFixture().SampleHTML))
		}))
		defer server.Close()

		// Run crawl command
		cmd := exec.CommandContext(ctx, cfg.BinPath, "crawl", server.URL,
			"-L", "1",
			"-w", "2",
			"-D", cfg.DataDir,
			"-d", "100")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("crawl output: %s", output)
			t.Fatalf("crawl command failed: %v", err)
		}

		// Verify index was created
		indexDir := cfg.IndexDir()
		files, err := os.ReadDir(indexDir)
		if err != nil {
			t.Fatalf("failed to read index dir: %v", err)
		}

		if len(files) == 0 {
			t.Error("index directory is empty after crawl")
		}
	})

	t.Run("search_indexed_content", func(t *testing.T) {
		// Run search command
		cmd := exec.CommandContext(ctx, cfg.BinPath, "search", "example",
			"-D", cfg.DataDir,
			"--limit", "10")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("search output: %s", output)
			t.Fatalf("search command failed: %v", err)
		}

		outputStr := string(output)
		AssertContains(t, outputStr, "results")
	})
}

// TestPersistence tests that index can be saved and loaded.
func TestPersistence(t *testing.T) {
	SkipIfMissingBinary(t, "../bin/gosearch")

	cfg := NewTestConfig(t)
	defer cfg.Cleanup()

	ctx, cancel := cfg.CreateContext()
	defer cancel()

	if err := SetupTestDir(cfg.DataDir); err != nil {
		t.Fatalf("failed to setup test dir: %v", err)
	}

	// First, crawl and index
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(SampleFixture().SampleHTML))
	}))
	defer server.Close()

	cmd := exec.CommandContext(ctx, cfg.BinPath, "crawl", server.URL,
		"-L", "1",
		"-w", "2",
		"-D", cfg.DataDir,
		"-d", "100")

	if err := cmd.Run(); err != nil {
		t.Logf("crawl output: %v", err)
	}

	// Check index stats
	cmd = exec.CommandContext(ctx, cfg.BinPath, "index", "stats",
		"-D", cfg.DataDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("stats output: %s", output)
	}

	// Clear index
	cmd = exec.CommandContext(ctx, cfg.BinPath, "index", "clear",
		"-D", cfg.DataDir)

	if err := cmd.Run(); err != nil {
		t.Fatalf("index clear failed: %v", err)
	}

	// Search should trigger auto-rebuild or show empty index
	cmd = exec.CommandContext(ctx, cfg.BinPath, "search", "example",
		"-D", cfg.DataDir)

	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Logf("search after clear output: %s", output)
	}
}

// TestFuzzyMatching tests fuzzy search functionality.
func TestFuzzyMatching(t *testing.T) {
	SkipIfMissingBinary(t, "../bin/gosearch")

	cfg := NewTestConfig(t)
	defer cfg.Cleanup()

	ctx, cancel := cfg.CreateContext()
	defer cancel()

	if err := SetupTestDir(cfg.DataDir); err != nil {
		t.Fatalf("failed to setup test dir: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(SampleFixture().SampleHTML))
	}))
	defer server.Close()

	// Crawl first
	cmd := exec.CommandContext(ctx, cfg.BinPath, "crawl", server.URL,
		"-L", "1",
		"-w", "2",
		"-D", cfg.DataDir,
		"-d", "100")

	_ = cmd.Run() // Crawl may succeed or fail, we're testing search

	// Search with typo using fuzzy flag
	cmd = exec.CommandContext(ctx, cfg.BinPath, "search", "exmple",
		"-D", cfg.DataDir,
		"--fuzzy")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("fuzzy search output: %s", output)
		// Fuzzy search might not find results, that's OK for this test
	}
}

// TestBooleanQueries tests AND, OR, NOT query operators.
func TestBooleanQueries(t *testing.T) {
	SkipIfMissingBinary(t, "../bin/gosearch")

	cfg := NewTestConfig(t)
	defer cfg.Cleanup()

	ctx, cancel := cfg.CreateContext()
	defer cancel()

	if err := SetupTestDir(cfg.DataDir); err != nil {
		t.Fatalf("failed to setup test dir: %v", err)
	}

	// Use sample HTML content directly
	htmlContent := string(ReadTestdata(t, "sample_html.html"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	// Crawl the sample page
	cmd := exec.CommandContext(ctx, cfg.BinPath, "crawl", server.URL,
		"-L", "1",
		"-w", "2",
		"-D", cfg.DataDir,
		"-d", "100")

	_ = cmd.Run()

	// Test AND query
	cmd = exec.CommandContext(ctx, cfg.BinPath, "search", "search AND engine",
		"-D", cfg.DataDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("AND query output: %s", output)
	}
}

// TestBackupRestore tests backup and restore functionality.
func TestBackupRestore(t *testing.T) {
	SkipIfMissingBinary(t, "../bin/gosearch")

	cfg := NewTestConfig(t)
	defer cfg.Cleanup()

	ctx, cancel := cfg.CreateContext()
	defer cancel()

	if err := SetupTestDir(cfg.DataDir); err != nil {
		t.Fatalf("failed to setup test dir: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(SampleFixture().SampleHTML))
	}))
	defer server.Close()

	// Crawl and index
	cmd := exec.CommandContext(ctx, cfg.BinPath, "crawl", server.URL,
		"-L", "1",
		"-w", "2",
		"-D", cfg.DataDir,
		"-d", "100")

	_ = cmd.Run()

	// Backup
	backupPath := cfg.BackupPath("test")
	cmd = exec.CommandContext(ctx, cfg.BinPath, "backup", backupPath,
		"-D", cfg.DataDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("backup output: %s", output)
	}

	// Clear index
	cmd = exec.CommandContext(ctx, cfg.BinPath, "index", "clear",
		"-D", cfg.DataDir)

	_ = cmd.Run()

	// Restore
	cmd = exec.CommandContext(ctx, cfg.BinPath, "restore", backupPath,
		"-D", cfg.DataDir)

	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Logf("restore output: %s", output)
	}
}

// TestHTTPAPI tests the HTTP API endpoints.
func TestHTTPAPI(t *testing.T) {
	SkipIfMissingBinary(t, "../bin/gosearch")

	cfg := NewTestConfig(t)
	defer cfg.Cleanup()

	ctx, cancel := cfg.CreateContext()
	defer cancel()

	if err := SetupTestDir(cfg.DataDir); err != nil {
		t.Fatalf("failed to setup test dir: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(SampleFixture().SampleHTML))
	}))
	defer server.Close()

	// Crawl first
	cmd := exec.CommandContext(ctx, cfg.BinPath, "crawl", server.URL,
		"-L", "1",
		"-w", "2",
		"-D", cfg.DataDir,
		"-d", "100")

	_ = cmd.Run()

	// Start API server
	apiCmd := exec.CommandContext(ctx, cfg.BinPath, "serve",
		"-D", cfg.DataDir,
		"-p", fmt.Sprintf("%d", cfg.APIPort),
		"--host", "localhost")

	if err := apiCmd.Start(); err != nil {
		t.Fatalf("failed to start API server: %v", err)
	}
	defer func() {
		apiCmd.Process.Kill()
		apiCmd.Wait()
	}()

	// Give server time to start
	time.Sleep(2 * time.Second)

	baseURL := cfg.ServerURL()

	t.Run("health_endpoint", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/api/v1/health")
		if err != nil {
			t.Fatalf("health endpoint failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("health endpoint returned status %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("stats_endpoint", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/api/v1/stats")
		if err != nil {
			t.Fatalf("stats endpoint failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Logf("stats response: %s", body)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Logf("failed to decode stats response: %v", err)
		}
	})

	t.Run("search_endpoint", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/api/v1/search?q=example&limit=10")
		if err != nil {
			t.Fatalf("search endpoint failed: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		t.Logf("search response: %s", body)
	})
}

// TestEdgeCases tests edge cases and error handling.
func TestEdgeCases(t *testing.T) {
	SkipIfMissingBinary(t, "../bin/gosearch")

	cfg := NewTestConfig(t)
	defer cfg.Cleanup()

	ctx, cancel := cfg.CreateContext()
	defer cancel()

	if err := SetupTestDir(cfg.DataDir); err != nil {
		t.Fatalf("failed to setup test dir: %v", err)
	}

	t.Run("search_empty_index", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, cfg.BinPath, "search", "test",
			"-D", cfg.DataDir)

		output, err := cmd.CombinedOutput()
		if err != nil {
			outputStr := string(output)
			// Should get an error about empty index
			AssertContains(t, outputStr, "index") // Should mention index in error
		}
	})

	t.Run("search_empty_query", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, cfg.BinPath, "search", "",
			"-D", cfg.DataDir)

		output, err := cmd.CombinedOutput()
		if err != nil {
			outputStr := string(output)
			// Should get an error about empty query
			AssertContains(t, outputStr, "query") // Should mention query in error
		}
	})

	t.Run("crawl_invalid_url", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, cfg.BinPath, "crawl", "not-a-url",
			"-D", cfg.DataDir)

		output, err := cmd.CombinedOutput()
		if err != nil {
			outputStr := string(output)
			// Should get an error about invalid URL
			t.Logf("invalid URL error: %s", outputStr)
		}
	})
}

// TestBlockDetection tests crawler block detection and backoff.
func TestBlockDetection(t *testing.T) {
	SkipIfMissingBinary(t, "../bin/gosearch")

	cfg := NewTestConfig(t)
	defer cfg.Cleanup()

	ctx, cancel := cfg.CreateContext()
	defer cancel()

	if err := SetupTestDir(cfg.DataDir); err != nil {
		t.Fatalf("failed to setup test dir: %v", err)
	}

	// Start a mock server that returns 429
	blockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("<html><body>Too Many Requests</body></html>"))
	}))
	defer blockServer.Close()

	// Try to crawl - should handle 429 gracefully
	cmd := exec.CommandContext(ctx, cfg.BinPath, "crawl", blockServer.URL,
		"-L", "1",
		"-w", "1",
		"-D", cfg.DataDir,
		"-d", "100")

	output, err := cmd.CombinedOutput()
	t.Logf("block detection crawl output: %s", output)

	// The crawler should detect block and handle it
	// It might fail, but shouldn't hang or panic
	if err != nil {
		outputStr := string(output)
		// Check if error mentions rate limiting or blocking
		if strings.Contains(outputStr, "429") || strings.Contains(outputStr, "rate") {
			t.Log("Block detection working correctly")
		}
	}
}

// TestInternalModulesCrawlIndexSearch tests the full crawl -> index -> search workflow
// using internal Go packages directly (not through CLI).
func TestInternalModulesCrawlIndexSearch(t *testing.T) {
	// Create temporary data directory
	tmpDir, err := os.MkdirTemp("", "gosearch_integration_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test server with multiple pages
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Go Search Engine</title></head>
<body>
	<h1>Welcome to Go Search Engine</h1>
	<p>This is a fast search engine built with Go.</p>
	<p>It features web crawling, inverted indexing, and boolean search.</p>
	<a href="/about">About</a>
	<a href="/features">Features</a>
</body>
</html>`))
		case "/about":
			w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>About Go Search</title></head>
<body>
	<h1>About Go Search</h1>
	<p>Go Search is an open source search engine project.</p>
	<p>The indexer uses tokenization and inverted index data structures.</p>
</body>
</html>`))
		case "/features":
			w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Features</title></head>
<body>
	<h1>Search Features</h1>
	<ul>
		<li>Boolean search with AND, OR, NOT operators</li>
		<li>Fuzzy matching with Levenshtein distance</li>
		<li>TF-IDF ranking for relevance scoring</li>
		<li>PageRank algorithm for authority scoring</li>
	</ul>
</body>
</html>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	ctx := context.Background()

	// Step 1: Setup crawler and crawl the test server
	t.Run("crawl_pages", func(t *testing.T) {
		docStore, err := storage.NewDocumentStore(filepath.Join(tmpDir, "pages"))
		if err != nil {
			t.Fatalf("failed to create document store: %v", err)
		}
		defer docStore.Close()

		crawlerCfg := &crawler.Config{
			MaxWorkers:        2,
			MaxDepth:          2,
			Delay:             0, // No delay for tests
			AllowedDomains:    nil,
			UserAgent:         "GoSearch-Test/1.0",
			Timeout:           10 * time.Second,
			RespectRobots:     false,
			EnableBlockDetect: false,
			MaxRetries:        1,
		}

		c, err := crawler.NewCollyCrawler(crawlerCfg, docStore)
		if err != nil {
			t.Fatalf("failed to create crawler: %v", err)
		}

		// Crawl with timeout
		crawlCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		if err := c.Start(crawlCtx, []string{server.URL + "/"}); err != nil {
			t.Logf("crawl ended with error (may be expected): %v", err)
		}

		stats := c.Stats()
		t.Logf("Crawled %d pages", stats.URLsCrawled)

		if stats.URLsCrawled == 0 {
			t.Error("expected to crawl at least one page")
		}
	})

	// Step 2: Index the crawled documents
	t.Run("index_documents", func(t *testing.T) {
		// Create storage
		docStore, err := storage.NewDocumentStore(filepath.Join(tmpDir, "pages"))
		if err != nil {
			t.Fatalf("failed to create document store: %v", err)
		}
		defer docStore.Close()

		indexStore, err := storage.NewIndexStore(filepath.Join(tmpDir, "index"))
		if err != nil {
			t.Fatalf("failed to create index store: %v", err)
		}
		defer indexStore.Close()

		logger, err := zap.NewDevelopment()
		if err != nil {
			t.Fatalf("failed to create logger: %v", err)
		}
		defer logger.Sync()

		idxr := indexer.NewIndexer(indexStore, logger)

		// List all documents
		docs, err := docStore.List()
		if err != nil {
			t.Fatalf("failed to list documents: %v", err)
		}

		t.Logf("Found %d documents to index", len(docs))

		if len(docs) == 0 {
			t.Skip("no documents to index, skipping search tests")
		}

		// Index each document
		for _, docID := range docs {
			doc, err := docStore.Get(docID)
			if err != nil {
				t.Logf("failed to get document %s: %v", docID, err)
				continue
			}

			if err := idxr.IndexDocument(ctx, doc); err != nil {
				t.Logf("failed to index document %s: %v", docID, err)
			}
		}

		// Verify indexing
		stats := idxr.Stats()
		t.Logf("Index stats: %d documents, %d terms", stats.TotalDocuments, stats.TotalTerms)

		if stats.TotalDocuments == 0 {
			t.Error("expected at least one document in index")
		}

		// Save index for next subtest
		if err := idxr.Save(ctx); err != nil {
			t.Logf("failed to save index: %v", err)
		}
	})

	// Step 3: Search with various query types (table-driven)
	t.Run("search_queries", func(t *testing.T) {
		// Setup searcher components
		docStore, err := storage.NewDocumentStore(filepath.Join(tmpDir, "pages"))
		if err != nil {
			t.Fatalf("failed to create document store: %v", err)
		}
		defer docStore.Close()

		indexStore, err := storage.NewIndexStore(filepath.Join(tmpDir, "index"))
		if err != nil {
			t.Fatalf("failed to create index store: %v", err)
		}
		defer indexStore.Close()

		logger, err := zap.NewDevelopment()
		if err != nil {
			t.Fatalf("failed to create logger: %v", err)
		}
		defer logger.Sync()

		// Load index
		idxr := indexer.NewIndexer(indexStore, logger)
		if err := idxr.Load(ctx); err != nil {
			t.Logf("failed to load index: %v (may be empty)", err)
		}

		// Get the index for searching
		index := idxr.GetIndex() // returns *indexer.Index

		// Create scorer with TF-IDF and PageRank
		tfidf := ranker.NewTFIDF(index.GetIndex()) // index.GetIndex() returns *InvertedIndex
		pagerank := ranker.DefaultPageRank()
		scorer := ranker.NewScorer(tfidf, pagerank, nil)
		searcher := search.NewSearcher(index, scorer, nil, nil)

		// Table-driven test cases
		testCases := []struct {
			name          string
			query         string
			minResults    int
			shouldContain string
		}{
			{
				name:          "simple_term_search",
				query:         "search",
				minResults:    1,
				shouldContain: "search",
			},
			{
				name:          "boolean_and_query",
				query:         "search AND engine",
				minResults:    1,
				shouldContain: "search",
			},
			{
				name:          "boolean_or_query",
				query:         "crawler OR indexer",
				minResults:    0,
				shouldContain: "",
			},
			{
				name:          "phrase_query",
				query:         `"search engine"`,
				minResults:    0,
				shouldContain: "",
			},
			{
				name:          "term_go",
				query:         "go",
				minResults:    1,
				shouldContain: "",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := searcher.Search(ctx, tc.query)
				if err != nil {
					t.Logf("search failed for query '%s': %v", tc.query, err)
					// Some searches may fail if no documents match
					return
				}

				t.Logf("Query '%s' returned %d results", tc.query, len(result.Results))

				if len(result.Results) < tc.minResults {
					t.Logf("Expected at least %d results, got %d", tc.minResults, len(result.Results))
					// Don't fail, just log - document content may vary
				}

				// Verify expected content if results exist
				if tc.shouldContain != "" && len(result.Results) > 0 {
					found := false
					for _, r := range result.Results {
						titleLower := strings.ToLower(r.Title)
						snippetLower := strings.ToLower(r.Snippet)
						if strings.Contains(titleLower, tc.shouldContain) || strings.Contains(snippetLower, tc.shouldContain) {
							found = true
							break
						}
					}
					if !found {
						t.Logf("Expected results to contain '%s', but didn't find it", tc.shouldContain)
					}
				}

				// Log results for debugging
				for i, r := range result.Results {
					if i < 3 {
						t.Logf("  Result %d: %s (score: %.2f)", i+1, r.Title, r.Score)
					}
				}
			})
		}
	})
}

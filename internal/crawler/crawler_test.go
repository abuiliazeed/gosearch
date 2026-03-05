package crawler

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"

	"github.com/abuiliazeed/gosearch/internal/storage"
)

func TestCollyCrawler_Start(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, `<html>
			<head><title>Test Page</title></head>
			<body>
				<a href="/page2">Link</a>
			</body>
		</html>`)
	}))
	defer server.Close()

	// Setup deps
	tmpDir, _ := os.MkdirTemp("", "crawler_test")
	defer os.RemoveAll(tmpDir)

	// We can't easily mock storage.DocumentStore here because of the circular dependency if we import storage.
	// However, since we are in package crawler, we can't import storage if storage imports crawler.
	// Check if storage imports crawler? NO. storage does NOT import crawler.
	// The potential cycle is: crawler -> storage -> ... -> crawler?
	// Let's re-read the error.

	docStore, _ := storage.NewDocumentStore(tmpDir)

	config := DefaultConfig()
	config.MaxDepth = 1
	config.Delay = 0
	config.MaxWorkers = 1

	c, err := NewCollyCrawler(config, docStore)
	if err != nil {
		t.Fatalf("failed to create crawler: %v", err)
	}

	// Run crawler
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = c.Start(ctx, []string{server.URL})
	if err != nil {
		t.Fatalf("crawler failed: %v", err)
	}

	// Verify stats
	stats := c.Stats()
	if stats.URLsCrawled == 0 {
		t.Error("expected at least 1 URL crawled")
	}
}

func TestCrawler_RespectsMaxDepth(t *testing.T) {
	// Chain of pages
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			fmt.Fprintln(w, `<a href="/depth1">Level 1</a>`)
		case "/depth1":
			fmt.Fprintln(w, `<a href="/depth2">Level 2</a>`)
		case "/depth2":
			fmt.Fprintln(w, `<a href="/depth3">Level 3</a>`)
		default:
			fmt.Fprintln(w, "End")
		}
	}))
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "crawler_depth_test")
	defer os.RemoveAll(tmpDir)
	docStore, _ := storage.NewDocumentStore(tmpDir)

	config := DefaultConfig()
	config.MaxDepth = 1 // Only root (0) and level 1
	config.Delay = 0

	c, _ := NewCollyCrawler(config, docStore)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c.Start(ctx, []string{server.URL})

	// Should have crawled root and depth1, but NOT depth2
	stats := c.Stats()

	// Depending on implementation, depth 0 is root.
	// Depth 1 means root + children.
	// The implementation seems to treat depth as exact levels traversed

	// We expect:
	// 1. Root (depth 0) -> finds link to depth1
	// 2. Depth1 (depth 1) -> finds link to depth2
	// 3. Depth2 (depth 2) -> should be skipped if max depth is 1

	// So we expect 2 pages crawled (Root, Depth1)
	if stats.URLsCrawled > 2 {
		t.Errorf("expected max 2 pages crawled (depth 0 & 1), got %d", stats.URLsCrawled)
	}
}

func TestCrawler_DoesNotAbortOnSingleURLFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			fmt.Fprintln(w, `<a href="/ok">OK</a><a href="/missing">Missing</a>`)
		case "/ok":
			fmt.Fprintln(w, `<html><head><title>OK</title></head><body>good</body></html>`)
		case "/missing":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "crawler_nonfatal_error_test")
	defer os.RemoveAll(tmpDir)
	docStore, _ := storage.NewDocumentStore(tmpDir)

	config := DefaultConfig()
	config.MaxDepth = 1
	config.Delay = 0
	config.MaxWorkers = 1
	config.RespectRobots = false

	c, err := NewCollyCrawler(config, docStore)
	if err != nil {
		t.Fatalf("failed to create crawler: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.Start(ctx, []string{server.URL}); err != nil {
		t.Fatalf("crawler should continue on URL-level failures, got error: %v", err)
	}

	stats := c.Stats()
	if stats.URLsCrawled < 2 {
		t.Fatalf("expected crawler to continue beyond seed, got URLsCrawled=%d", stats.URLsCrawled)
	}
	if stats.URLsFailed < 1 {
		t.Fatalf("expected at least one failed URL for /missing, got URLsFailed=%d", stats.URLsFailed)
	}

	ids, err := docStore.List()
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}
	if len(ids) < 2 {
		t.Fatalf("expected at least 2 saved docs (seed + /ok), got %d", len(ids))
	}
}

func TestCrawler_RespectsMaxQueueSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if r.URL.Path == "/" {
			var b strings.Builder
			b.WriteString("<html><body>")
			for i := 0; i < 200; i++ {
				fmt.Fprintf(&b, `<a href="/p%d">Page %d</a>`, i, i)
			}
			b.WriteString("</body></html>")
			fmt.Fprintln(w, b.String())
			return
		}
		fmt.Fprintln(w, `<html><head><title>Child</title></head><body>ok</body></html>`)
	}))
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "crawler_max_queue_test")
	defer os.RemoveAll(tmpDir)
	docStore, _ := storage.NewDocumentStore(tmpDir)

	config := DefaultConfig()
	config.MaxDepth = 1
	config.MaxQueueSize = 25
	config.Delay = 0
	config.MaxWorkers = 4
	config.RespectRobots = false

	c, err := NewCollyCrawler(config, docStore)
	if err != nil {
		t.Fatalf("failed to create crawler: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.Start(ctx, []string{server.URL}); err != nil {
		t.Fatalf("crawler failed: %v", err)
	}

	stats := c.Stats()
	if stats.URLsQueued > config.MaxQueueSize {
		t.Fatalf("expected URLsQueued <= %d, got %d", config.MaxQueueSize, stats.URLsQueued)
	}
}

func TestCrawler_DecodesGzipContent(t *testing.T) {
	const html = `<html><head><title>Gzip Test Page</title></head><body>Hello from gzip world</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var compressed bytes.Buffer
		gzipWriter := gzip.NewWriter(&compressed)
		_, _ = gzipWriter.Write([]byte(html))
		_ = gzipWriter.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(compressed.Bytes())
	}))
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "crawler_gzip_test")
	defer os.RemoveAll(tmpDir)
	docStore, _ := storage.NewDocumentStore(tmpDir)

	config := DefaultConfig()
	config.MaxDepth = 0
	config.Delay = 0
	config.MaxWorkers = 1
	config.RespectRobots = false

	c, err := NewCollyCrawler(config, docStore)
	if err != nil {
		t.Fatalf("failed to create crawler: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.Start(ctx, []string{server.URL}); err != nil {
		t.Fatalf("crawler failed: %v", err)
	}

	ids, err := docStore.List()
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 document, got %d", len(ids))
	}

	doc, err := docStore.Get(ids[0])
	if err != nil {
		t.Fatalf("failed to load crawled document: %v", err)
	}

	if !strings.Contains(strings.ToLower(doc.Title), "gzip test page") {
		t.Fatalf("expected decoded title, got %q", doc.Title)
	}
	if !strings.Contains(strings.ToLower(doc.ContentMarkdown), "hello from gzip world") {
		t.Fatalf("expected decoded content, got %q", doc.ContentMarkdown)
	}
}

func TestCrawler_DecodesBrotliContent(t *testing.T) {
	const html = `<html><head><title>Brotli Test Page</title></head><body>Elementrix encoded content</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var compressed bytes.Buffer
		brWriter := brotli.NewWriter(&compressed)
		_, _ = brWriter.Write([]byte(html))
		_ = brWriter.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Encoding", "br")
		_, _ = w.Write(compressed.Bytes())
	}))
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "crawler_brotli_test")
	defer os.RemoveAll(tmpDir)
	docStore, _ := storage.NewDocumentStore(tmpDir)

	config := DefaultConfig()
	config.MaxDepth = 0
	config.Delay = 0
	config.MaxWorkers = 1
	config.RespectRobots = false

	c, err := NewCollyCrawler(config, docStore)
	if err != nil {
		t.Fatalf("failed to create crawler: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.Start(ctx, []string{server.URL}); err != nil {
		t.Fatalf("crawler failed: %v", err)
	}

	ids, err := docStore.List()
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 document, got %d", len(ids))
	}

	doc, err := docStore.Get(ids[0])
	if err != nil {
		t.Fatalf("failed to load crawled document: %v", err)
	}

	if !strings.Contains(strings.ToLower(doc.Title), "brotli test page") {
		t.Fatalf("expected decoded title, got %q", doc.Title)
	}
	if !strings.Contains(strings.ToLower(doc.ContentMarkdown), "elementrix encoded content") {
		t.Fatalf("expected decoded content, got %q", doc.ContentMarkdown)
	}
}

func TestCrawler_SkipsBinaryContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte{0x1f, 0x8b, 0x08, 0x00, 0xff, 0x00, 0xaa, 0xbb})
	}))
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "crawler_binary_test")
	defer os.RemoveAll(tmpDir)
	docStore, _ := storage.NewDocumentStore(tmpDir)

	config := DefaultConfig()
	config.MaxDepth = 0
	config.Delay = 0
	config.MaxWorkers = 1
	config.RespectRobots = false

	c, err := NewCollyCrawler(config, docStore)
	if err != nil {
		t.Fatalf("failed to create crawler: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.Start(ctx, []string{server.URL}); err != nil {
		t.Fatalf("crawler failed: %v", err)
	}

	ids, err := docStore.List()
	if err != nil {
		t.Fatalf("failed to list documents: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected 0 indexed documents for binary response, got %d", len(ids))
	}
}

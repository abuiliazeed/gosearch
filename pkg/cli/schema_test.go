package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureSchemaForCrawl_PerformsResetAndWritesMarker(t *testing.T) {
	dataDir := t.TempDir()
	pagesDir := filepath.Join(dataDir, "pages")
	indexDir := filepath.Join(dataDir, "index")
	indexFile := filepath.Join(indexDir, "index.db")

	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		t.Fatalf("failed to create pages dir: %v", err)
	}
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		t.Fatalf("failed to create index dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pagesDir, "stale.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("failed to create stale page file: %v", err)
	}
	if err := os.WriteFile(indexFile, []byte("old-index"), 0o644); err != nil {
		t.Fatalf("failed to create stale index file: %v", err)
	}

	reset, err := ensureSchemaForCrawl(dataDir)
	if err != nil {
		t.Fatalf("ensureSchemaForCrawl failed: %v", err)
	}
	if !reset {
		t.Fatal("expected reset to be performed")
	}

	if _, err := os.Stat(indexFile); !os.IsNotExist(err) {
		t.Fatalf("expected index file to be removed, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(pagesDir, "stale.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected stale page file to be removed, got err=%v", err)
	}

	markerBytes, err := os.ReadFile(schemaVersionPath(dataDir))
	if err != nil {
		t.Fatalf("failed to read schema marker: %v", err)
	}
	if got := strings.TrimSpace(string(markerBytes)); got != schemaVersionV2 {
		t.Fatalf("expected schema marker %q, got %q", schemaVersionV2, got)
	}

	reset, err = ensureSchemaForCrawl(dataDir)
	if err != nil {
		t.Fatalf("second ensureSchemaForCrawl failed: %v", err)
	}
	if reset {
		t.Fatal("expected no reset on already-initialized schema")
	}
}

func TestRequireSchemaVersion(t *testing.T) {
	dataDir := t.TempDir()
	if err := requireSchemaVersion(dataDir); err == nil {
		t.Fatal("expected error when schema marker is missing")
	}

	if err := writeSchemaVersion(dataDir); err != nil {
		t.Fatalf("failed to write schema marker: %v", err)
	}
	if err := requireSchemaVersion(dataDir); err != nil {
		t.Fatalf("expected schema validation to pass, got %v", err)
	}
}

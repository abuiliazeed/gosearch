package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	schemaVersionFileName = "schema_version"
	schemaVersionV2       = "2-markdown-only"
)

func schemaVersionPath(dataDir string) string {
	return filepath.Join(dataDir, schemaVersionFileName)
}

func readSchemaVersion(dataDir string) (string, error) {
	path := schemaVersionPath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read schema marker %s: %w", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func writeSchemaVersion(dataDir string) error {
	path := schemaVersionPath(dataDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create schema marker directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(schemaVersionV2+"\n"), 0o644); err != nil {
		return fmt.Errorf("failed to write schema marker %s: %w", path, err)
	}
	return nil
}

// ensureSchemaForCrawl validates corpus schema and performs a one-time reset for legacy data.
// It returns true when a reset was performed.
func ensureSchemaForCrawl(dataDir string) (bool, error) {
	version, err := readSchemaVersion(dataDir)
	if err != nil {
		return false, err
	}
	if version == schemaVersionV2 {
		return false, nil
	}

	pagesPath := filepath.Join(dataDir, "pages")
	indexDir := filepath.Join(dataDir, "index")
	indexPath := filepath.Join(indexDir, "index.db")

	if err := os.RemoveAll(pagesPath); err != nil {
		return false, fmt.Errorf("failed to clear pages directory %s: %w", pagesPath, err)
	}
	if err := os.Remove(indexPath); err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to remove index file %s: %w", indexPath, err)
	}
	if err := os.MkdirAll(pagesPath, 0o755); err != nil {
		return false, fmt.Errorf("failed to recreate pages directory %s: %w", pagesPath, err)
	}
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return false, fmt.Errorf("failed to ensure index directory %s: %w", indexDir, err)
	}
	if err := writeSchemaVersion(dataDir); err != nil {
		return false, err
	}

	return true, nil
}

func requireSchemaVersion(dataDir string) error {
	version, err := readSchemaVersion(dataDir)
	if err != nil {
		return err
	}
	if version == schemaVersionV2 {
		return nil
	}

	found := "missing"
	if version != "" {
		found = version
	}
	return fmt.Errorf(
		"data schema mismatch: found %q, expected %q. run `gosearch crawl <url>` once to initialize markdown-only v2 storage",
		found,
		schemaVersionV2,
	)
}

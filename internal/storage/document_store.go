package storage

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// DocumentStore handles file-based storage of crawled documents.
// Documents are stored as gzip-compressed JSON files.
type DocumentStore struct {
	baseDir string
}

// NewDocumentStore creates a new DocumentStore with the given base directory.
func NewDocumentStore(baseDir string) (*DocumentStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create document store directory: %w", err)
	}
	return &DocumentStore{baseDir: baseDir}, nil
}

// Save saves a document to the store.
// The document ID is generated as a SHA256 hash of the URL.
func (ds *DocumentStore) Save(doc *Document) error {
	if doc.ID == "" {
		doc.ID = hashURL(doc.URL)
	}

	filePath := ds.filePath(doc.ID)

	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	// Encode document as JSON and write to gzip writer
	if err := json.NewEncoder(gzWriter).Encode(doc); err != nil {
		return fmt.Errorf("failed to encode document: %w", err)
	}

	return nil
}

// Get retrieves a document by ID.
func (ds *DocumentStore) Get(id string) (*Document, error) {
	filePath := ds.filePath(id)

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("document not found: %s", id)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Decode document
	var doc Document
	if err := json.NewDecoder(gzReader).Decode(&doc); err != nil {
		return nil, fmt.Errorf("failed to decode document: %w", err)
	}

	return &doc, nil
}

// GetByURL retrieves a document by URL.
func (ds *DocumentStore) GetByURL(url string) (*Document, error) {
	id := hashURL(url)
	return ds.Get(id)
}

// Exists checks if a document exists by ID.
func (ds *DocumentStore) Exists(id string) bool {
	filePath := ds.filePath(id)
	_, err := os.Stat(filePath)
	return err == nil
}

// ExistsURL checks if a document exists by URL.
func (ds *DocumentStore) ExistsURL(url string) bool {
	id := hashURL(url)
	return ds.Exists(id)
}

// Delete removes a document by ID.
func (ds *DocumentStore) Delete(id string) error {
	filePath := ds.filePath(id)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	return nil
}

// List returns all document IDs in the store.
func (ds *DocumentStore) List() ([]string, error) {
	var ids []string

	err := filepath.Walk(ds.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".gz" {
			relPath, err := filepath.Rel(ds.baseDir, path)
			if err != nil {
				return err
			}
			// Remove .gz extension
			id := relPath[:len(relPath)-3]
			ids = append(ids, id)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	return ids, nil
}

// Count returns the total number of documents in the store.
func (ds *DocumentStore) Count() (int, error) {
	ids, err := ds.List()
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}

// filePath returns the file path for a given document ID.
// IDs are split into subdirectories for better filesystem performance.
func (ds *DocumentStore) filePath(id string) string {
	// Use first 2 characters as subdirectory, rest as filename
	if len(id) < 4 {
		return filepath.Join(ds.baseDir, id+".gz")
	}
	subdir := id[:2]
	filename := id[2:] + ".gz"
	return filepath.Join(ds.baseDir, subdir, filename)
}

// hashURL generates a SHA256 hash of a URL to use as a document ID.
func hashURL(url string) string {
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:])
}

// Close closes the document store and releases any resources.
func (ds *DocumentStore) Close() error {
	// Nothing to close for file-based storage
	return nil
}

// Backup creates a backup of the document store to the given destination.
func (ds *DocumentStore) Backup(dest string) error {
	return filepath.Walk(ds.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(ds.baseDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		return copyFile(path, destPath)
	})
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return nil
}

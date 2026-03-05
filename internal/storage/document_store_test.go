package storage

import (
	"os"
	"testing"
)

func TestDocumentStore_SaveGet(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "doc_store_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ds, err := NewDocumentStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create doc store: %v", err)
	}

	doc := &Document{
		URL:             "http://example.com/1",
		Title:           "Test Doc",
		ContentMarkdown: "Hello world",
	}

	// Save
	if err := ds.Save(doc); err != nil {
		t.Fatalf("failed to save doc: %v", err)
	}

	if doc.ID == "" {
		t.Error("expected doc ID to be generated")
	}

	// Get by ID
	loaded, err := ds.Get(doc.ID)
	if err != nil {
		t.Fatalf("failed to get doc: %v", err)
	}

	if loaded.URL != doc.URL {
		t.Errorf("expected URL %s, got %s", doc.URL, loaded.URL)
	}

	// Get by URL
	loadedByURL, err := ds.GetByURL(doc.URL)
	if err != nil {
		t.Fatalf("failed to get doc by URL: %v", err)
	}
	if loadedByURL.ID != doc.ID {
		t.Errorf("expected ID %s, got %s", doc.ID, loadedByURL.ID)
	}

	// Exists
	if !ds.Exists(doc.ID) {
		t.Error("expected doc to exist")
	}
}

func TestDocumentStore_Delete(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "doc_store_test_delete")
	defer os.RemoveAll(tmpDir)

	ds, _ := NewDocumentStore(tmpDir)

	doc := &Document{URL: "http://example.com/2", Title: "Delete Me"}
	ds.Save(doc)

	if err := ds.Delete(doc.ID); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	if ds.Exists(doc.ID) {
		t.Error("expected doc to be deleted")
	}
}

func TestDocumentStore_List(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "doc_store_test_list")
	defer os.RemoveAll(tmpDir)

	ds, _ := NewDocumentStore(tmpDir)

	docs := []*Document{
		{URL: "http://example.com/1"},
		{URL: "http://example.com/2"},
		{URL: "http://example.com/3"},
	}

	for _, doc := range docs {
		ds.Save(doc)
	}

	count, err := ds.Count()
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 docs, got %d", count)
	}

	ids, err := ds.List()
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(ids) != 3 {
		t.Errorf("expected 3 IDs, got %d", len(ids))
	}
}

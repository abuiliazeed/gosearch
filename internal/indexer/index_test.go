package indexer

import (
	"context"
	"testing"
)

func TestIndex_AddDocument(t *testing.T) {
	idx := NewIndex()
	tokenizer := NewTokenizer(DefaultTokenizerConfig())

	doc := &DocumentInput{
		DocID:   "doc1",
		Title:   "Go Language",
		Content: "Go is a statically typed language.",
	}

	ctx := context.Background()
	if err := idx.IndexDocument(ctx, tokenizer, doc); err != nil {
		t.Fatalf("failed to index document: %v", err)
	}

	if idx.DocumentCount() != 1 {
		t.Errorf("expected 1 document, got %d", idx.DocumentCount())
	}

	// Verify terms
	if !idx.HasTerm("language") {
		t.Error("expected term 'language' to be indexed")
	}
}

func TestIndex_Search(t *testing.T) {
	idx := NewIndex()
	tokenizer := NewTokenizer(DefaultTokenizerConfig())

	docs := []*DocumentInput{
		{DocID: "1", Title: "A", Content: "apple banana"},
		{DocID: "2", Title: "B", Content: "banana cherry"},
		{DocID: "3", Title: "C", Content: "apple cherry date"},
	}

	ctx := context.Background()
	for _, doc := range docs {
		idx.IndexDocument(ctx, tokenizer, doc)
	}

	// Test Single Term
	query := NewBooleanQuery()
	query.AddAnd("apple")

	results, err := idx.BooleanSearch(ctx, query)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if results.TotalCount != 2 {
		t.Errorf("expected 2 results for 'apple', got %d", results.TotalCount)
	}

	// Test AND
	query = NewBooleanQuery()
	query.AddAnd("apple")
	query.AddAnd("banana") // Only doc 1

	results, err = idx.BooleanSearch(ctx, query)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if results.TotalCount != 1 {
		t.Errorf("expected 1 result for 'apple AND banana', got %d", results.TotalCount)
	}

	// Test OR
	query = NewBooleanQuery()
	query.AddOr("banana")
	query.AddOr("date") // Docs 1, 2, 3 (banana in 1,2; date in 3)

	results, err = idx.BooleanSearch(ctx, query)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if results.TotalCount != 3 {
		t.Errorf("expected 3 results for 'banana OR date', got %d", results.TotalCount)
	}

	// Test NOT
	query = NewBooleanQuery()
	query.AddAnd("apple")
	query.AddNot("date") // Doc 1 only (Doc 3 has date)

	results, err = idx.BooleanSearch(ctx, query)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if results.TotalCount != 1 {
		t.Errorf("expected 1 result for 'apple NOT date', got %d", results.TotalCount)
	}
}

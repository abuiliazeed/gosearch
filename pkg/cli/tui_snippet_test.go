package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/abuiliazeed/gosearch/internal/search"
	"github.com/abuiliazeed/gosearch/internal/storage"
)

func TestExtractQueryTerms(t *testing.T) {
	terms := extractQueryTerms(`example AND "test phrase"`)
	if len(terms) == 0 {
		t.Fatal("expected query terms to be extracted")
	}

	joined := strings.Join(terms, " ")
	if !strings.Contains(joined, "example") {
		t.Fatalf("expected example in terms, got %v", terms)
	}
}

func TestSnippetFromMarkdown_QueryAware(t *testing.T) {
	markdown := `
# Example Site

Welcome to Example Site store.

We have hundreds of products.

This section talks about test phrase and skincare bundles for daily use.
`

	snippet := snippetFromMarkdown(markdown, []string{"test", "phrase"}, 140)
	if snippet == "" {
		t.Fatal("expected non-empty snippet")
	}
	if !strings.Contains(strings.ToLower(snippet), "test phrase") {
		t.Fatalf("expected snippet to include query phrase, got %q", snippet)
	}
}

func TestSnippetFromMarkdown_FallbackWhenNoTermMatch(t *testing.T) {
	markdown := `
# Example Site

Welcome to Example Site store.

This is fallback content when no query terms are present.
`

	snippet := snippetFromMarkdown(markdown, []string{"nonexistentterm"}, 100)
	if snippet == "" {
		t.Fatal("expected non-empty fallback snippet")
	}
	if !strings.Contains(strings.ToLower(snippet), "welcome to example site") {
		t.Fatalf("expected fallback to start of content, got %q", snippet)
	}
}

func TestEnrichResultsWithContentSnippets_ReplacesTitleOnlySnippet(t *testing.T) {
	docStore, err := storage.NewDocumentStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create document store: %v", err)
	}
	defer docStore.Close()

	doc := &storage.Document{
		ID:              "doc-test-1",
		URL:             "https://example.com/test",
		Title:           "Example Site - Home",
		ContentMarkdown: "# Example Site\n\nExplore skincare bundles and test phrase collections.\n",
		CrawledAt:       time.Now(),
	}
	if err := docStore.Save(doc); err != nil {
		t.Fatalf("failed to save document: %v", err)
	}

	results := []*search.Result{
		{
			DocID:   "doc-test-1",
			Title:   "Example Site - Home",
			URL:     "https://example.com/test",
			Snippet: "Example Site - Home",
		},
	}

	runtime := &searchRuntime{docStore: docStore}
	enrichResultsWithContentSnippets(runtime, results, "test phrase")

	got := strings.ToLower(results[0].Snippet)
	if !strings.Contains(got, "test phrase") {
		t.Fatalf("expected enriched snippet to include query terms, got %q", results[0].Snippet)
	}
	if strings.EqualFold(results[0].Snippet, results[0].Title) {
		t.Fatalf("expected snippet to differ from title after enrichment, got %q", results[0].Snippet)
	}
}

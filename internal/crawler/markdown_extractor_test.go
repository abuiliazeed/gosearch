package crawler

import (
	"strings"
	"testing"
)

func TestExtractMarkdownPage_CleansScriptsAndBuildsLinks(t *testing.T) {
	html := `
	<html>
	  <head><title>Test Page</title><script>window.bad = true;</script></head>
	  <body>
	    <h1>Welcome</h1>
	    <p>Hello <strong>world</strong></p>
	    <a href="/about">About</a>
	    <a href="https://external.example/path">External</a>
	  </body>
	</html>`

	result := extractMarkdownPage(html, "https://example.com/store")

	if result.Title == "" {
		t.Fatal("expected extracted title")
	}
	if result.ContentMarkdown == "" {
		t.Fatal("expected extracted markdown content")
	}
	if strings.Contains(result.ContentMarkdown, "window.bad") {
		t.Fatalf("expected scripts removed from markdown, got %q", result.ContentMarkdown)
	}
	if !strings.Contains(result.ContentMarkdown, "Welcome") {
		t.Fatalf("expected heading text in markdown, got %q", result.ContentMarkdown)
	}

	if !containsExact(result.Links, "https://example.com/about") {
		t.Fatalf("expected normalized absolute link for /about, got %+v", result.Links)
	}
	if !containsExact(result.Links, "https://external.example/path") {
		t.Fatalf("expected external absolute link, got %+v", result.Links)
	}
}

func TestExtractMarkdownPage_FallbackWhenURLInvalid(t *testing.T) {
	html := `<html><head><title>Fallback Title</title></head><body><p>Fallback body content</p></body></html>`
	result := extractMarkdownPage(html, "::invalid-url")

	if result.Title == "" {
		t.Fatal("expected fallback title")
	}
	if result.ContentMarkdown == "" {
		t.Fatal("expected fallback markdown content")
	}
	if !strings.Contains(result.ContentMarkdown, "Fallback") {
		t.Fatalf("expected fallback content in markdown, got %q", result.ContentMarkdown)
	}
}

func containsExact(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

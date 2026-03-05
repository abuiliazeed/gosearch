package indexer

import (
	"strings"
	"testing"
)

func TestMarkdownToText(t *testing.T) {
	markdown := `# Title

This is **bold** text with [a link](https://example.com).

` + "```go\nfmt.Println(\"hello\")\n```"

	text := MarkdownToText(markdown)
	if text == "" {
		t.Fatal("expected non-empty text")
	}

	expectedFragments := []string{"Title", "This", "bold", "text", "a", "link", "fmt.Println(\"hello\")"}
	for _, fragment := range expectedFragments {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected fragment %q in %q", fragment, text)
		}
	}
}

func TestMarkdownToText_Empty(t *testing.T) {
	if got := MarkdownToText("   \n\t"); got != "" {
		t.Fatalf("expected empty text, got %q", got)
	}
}

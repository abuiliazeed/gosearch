package cli

import (
	"strings"

	"github.com/abuiliazeed/gosearch/internal/indexer"
	"github.com/abuiliazeed/gosearch/internal/search"
)

const defaultResultSnippetLength = 220

func enrichResultsWithContentSnippets(runtime *searchRuntime, results []*search.Result, query string) {
	if runtime == nil || len(results) == 0 {
		return
	}

	terms := extractQueryTerms(query)
	for _, result := range results {
		if result == nil || result.DocID == "" {
			continue
		}

		doc, err := runtime.GetDocument(result.DocID)
		if err != nil || doc == nil {
			continue
		}

		snippet := snippetFromMarkdown(doc.ContentMarkdown, terms, defaultResultSnippetLength)
		if snippet != "" {
			result.Snippet = snippet
		}
	}
}

func extractQueryTerms(query string) []string {
	parser := search.NewParser(search.DefaultConfig())
	parsed := parser.Parse(query)
	rawTerms := parser.ExtractTerms(parsed)

	terms := make([]string, 0, len(rawTerms))
	seen := make(map[string]struct{}, len(rawTerms))
	for _, term := range rawTerms {
		normalized := strings.TrimSpace(strings.ToLower(term))
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		terms = append(terms, normalized)
	}

	return terms
}

func snippetFromMarkdown(markdown string, queryTerms []string, maxLength int) string {
	plainText := indexer.MarkdownToText(markdown)
	return snippetFromText(plainText, queryTerms, maxLength)
}

func snippetFromText(content string, queryTerms []string, maxLength int) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	if maxLength < 80 {
		maxLength = 80
	}

	lowerContent := strings.ToLower(content)
	firstMatch := -1
	matchedLength := 0
	for _, term := range queryTerms {
		if term == "" {
			continue
		}
		pos := strings.Index(lowerContent, term)
		if pos != -1 && (firstMatch == -1 || pos < firstMatch) {
			firstMatch = pos
			matchedLength = len(term)
		}
	}

	if firstMatch == -1 {
		return truncateSnippetWindow(content, 0, maxLength)
	}

	start := firstMatch - 80
	if start < 0 {
		start = 0
	}
	end := start + maxLength
	if end < firstMatch+matchedLength {
		end = firstMatch + matchedLength + 40
	}
	if end > len(content) {
		end = len(content)
	}
	if end-start > maxLength {
		start = end - maxLength
		if start < 0 {
			start = 0
		}
	}

	return truncateSnippetWindow(content, start, end-start)
}

func truncateSnippetWindow(content string, start int, length int) string {
	if start < 0 {
		start = 0
	}
	if length <= 0 {
		length = len(content)
	}

	end := start + length
	if end > len(content) {
		end = len(content)
	}
	if start > len(content) {
		start = len(content)
	}

	snippet := strings.TrimSpace(content[start:end])
	if snippet == "" {
		return ""
	}

	if start > 0 {
		snippet = "..." + strings.TrimLeft(snippet, " ")
	}
	if end < len(content) {
		snippet = strings.TrimRight(snippet, " ") + "..."
	}

	return snippet
}

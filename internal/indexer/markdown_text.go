package indexer

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var markdownWhitespaceRegex = regexp.MustCompile(`\s+`)

// MarkdownToText converts markdown to plain text for tokenization/snippet extraction.
func MarkdownToText(markdown string) string {
	if strings.TrimSpace(markdown) == "" {
		return ""
	}

	source := []byte(markdown)
	doc := goldmark.DefaultParser().Parse(text.NewReader(source))

	var out bytes.Buffer
	appendFragment := func(fragment string) {
		if fragment == "" {
			return
		}
		if out.Len() > 0 {
			out.WriteByte(' ')
		}
		out.WriteString(fragment)
	}

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Text:
			segment := node.Segment
			appendFragment(string((&segment).Value(source)))
		case *ast.CodeBlock:
			for i := 0; i < node.Lines().Len(); i++ {
				segment := node.Lines().At(i)
				appendFragment(string((&segment).Value(source)))
			}
		case *ast.FencedCodeBlock:
			for i := 0; i < node.Lines().Len(); i++ {
				segment := node.Lines().At(i)
				appendFragment(string((&segment).Value(source)))
			}
		}
		return ast.WalkContinue, nil
	})

	return normalizeMarkdownText(out.String())
}

func normalizeMarkdownText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return markdownWhitespaceRegex.ReplaceAllString(value, " ")
}

package crawler

import (
	"fmt"
	neturl "net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	readability "codeberg.org/readeck/go-readability/v2"
	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
)

var markdownBlankRunRegex = regexp.MustCompile(`\n{3,}`)

type extractedPage struct {
	Title           string
	ContentMarkdown string
	Links           []string
}

func extractMarkdownPage(rawHTML string, pageURL string) extractedPage {
	if rawHTML == "" {
		return extractedPage{}
	}
	if !utf8.ValidString(rawHTML) {
		rawHTML = strings.ToValidUTF8(rawHTML, " ")
	}

	parsedURL, _ := neturl.Parse(pageURL)
	title, markdown := readabilityFirstMarkdown(rawHTML, parsedURL)
	if strings.TrimSpace(markdown) == "" {
		fallbackTitle, fallbackBodyHTML := extractFallbackHTML(rawHTML)
		if strings.TrimSpace(title) == "" {
			title = fallbackTitle
		}
		if converted, err := convertHTMLToMarkdown(fallbackBodyHTML, parsedURL); err == nil {
			markdown = converted
		}
		if strings.TrimSpace(markdown) == "" {
			markdown = normalizeWhitespace(stripHTMLTags(fallbackBodyHTML))
		}
	}
	if strings.TrimSpace(title) == "" {
		title = normalizeWhitespace(pageURL)
	}
	if strings.TrimSpace(markdown) == "" {
		markdown = title
	}

	return extractedPage{
		Title:           title,
		ContentMarkdown: normalizeMarkdown(markdown),
		Links:           extractDocumentLinks(rawHTML, parsedURL),
	}
}

func readabilityFirstMarkdown(rawHTML string, pageURL *neturl.URL) (string, string) {
	if pageURL == nil {
		return "", ""
	}

	article, err := readability.FromReader(strings.NewReader(rawHTML), pageURL)
	if err != nil {
		return "", ""
	}

	title := normalizeWhitespace(article.Title())
	if article.Node == nil {
		return title, ""
	}

	var cleanedHTML strings.Builder
	if err := article.RenderHTML(&cleanedHTML); err != nil {
		return title, ""
	}

	markdown, err := convertHTMLToMarkdown(cleanedHTML.String(), pageURL)
	if err != nil {
		return title, ""
	}

	return title, markdown
}

func convertHTMLToMarkdown(cleanHTML string, pageURL *neturl.URL) (string, error) {
	if strings.TrimSpace(cleanHTML) == "" {
		return "", nil
	}

	domain := ""
	if pageURL != nil {
		domain = pageURL.String()
	}

	converter := md.NewConverter(domain, true, &md.Options{
		GetAbsoluteURL: func(_ *goquery.Selection, rawURL string, _ string) string {
			rawURL = strings.TrimSpace(rawURL)
			if rawURL == "" {
				return rawURL
			}
			if pageURL == nil {
				return rawURL
			}
			ref, err := neturl.Parse(rawURL)
			if err != nil {
				return rawURL
			}
			return pageURL.ResolveReference(ref).String()
		},
	})

	markdown, err := converter.ConvertString(cleanHTML)
	if err != nil {
		return "", fmt.Errorf("failed converting html to markdown: %w", err)
	}

	return normalizeMarkdown(markdown), nil
}

func normalizeMarkdown(markdown string) string {
	markdown = strings.ReplaceAll(markdown, "\r\n", "\n")
	markdown = strings.ReplaceAll(markdown, "\r", "\n")

	lines := strings.Split(markdown, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	markdown = strings.Join(lines, "\n")
	markdown = strings.TrimSpace(markdown)
	markdown = markdownBlankRunRegex.ReplaceAllString(markdown, "\n\n")

	return markdown
}

func extractDocumentLinks(rawHTML string, pageURL *neturl.URL) []string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	links := make([]string, 0, 32)

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		rawHref, exists := s.Attr("href")
		if !exists {
			return
		}
		rawHref = strings.TrimSpace(rawHref)
		if rawHref == "" {
			return
		}

		parsedHref, err := neturl.Parse(rawHref)
		if err != nil {
			return
		}

		resolved := parsedHref
		if pageURL != nil {
			resolved = pageURL.ResolveReference(parsedHref)
		}

		if resolved.Scheme != "http" && resolved.Scheme != "https" {
			return
		}

		normalized := NormalizeURL(resolved.String())
		if normalized == "" {
			return
		}

		if _, exists := seen[normalized]; exists {
			return
		}
		seen[normalized] = struct{}{}
		links = append(links, normalized)
	})

	return links
}

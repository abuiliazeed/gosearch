package crawler

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/andybalholm/brotli"
)

var (
	titleTagRegex      = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	bodyTagRegex       = regexp.MustCompile(`(?is)<body[^>]*>(.*?)</body>`)
	scriptStyleRegex   = regexp.MustCompile(`(?is)<(script|style|noscript)[^>]*>.*?</(script|style|noscript)>`)
	allTagsRegex       = regexp.MustCompile(`(?is)<[^>]+>`)
	whitespaceRunRegex = regexp.MustCompile(`\s+`)
)

// decodeResponseBody decodes an HTTP response body using Content-Encoding.
func decodeResponseBody(body []byte, contentEncoding string) ([]byte, error) {
	if len(body) == 0 || contentEncoding == "" {
		return body, nil
	}

	decoded := body
	encodings := parseContentEncodings(contentEncoding)

	// Encodings are applied in order by the server and must be decoded in reverse.
	for i := len(encodings) - 1; i >= 0; i-- {
		encoding := encodings[i]
		var (
			next []byte
			err  error
		)

		switch encoding {
		case "", "identity":
			continue
		case "gzip", "x-gzip":
			next, err = decodeGzipBody(decoded)
		case "deflate":
			next, err = decodeDeflateBody(decoded)
		case "br":
			next, err = decodeBrotliBody(decoded)
		default:
			// Unknown encoding: keep the current body instead of failing hard.
			continue
		}

		if err != nil {
			return body, err
		}
		decoded = next
	}

	return decoded, nil
}

func parseContentEncodings(value string) []string {
	parts := strings.Split(value, ",")
	encodings := make([]string, 0, len(parts))
	for _, part := range parts {
		encoding := strings.ToLower(strings.TrimSpace(part))
		if encoding == "" {
			continue
		}
		encodings = append(encodings, encoding)
	}
	return encodings
}

func decodeGzipBody(body []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	return io.ReadAll(reader)
}

func decodeDeflateBody(body []byte) ([]byte, error) {
	// Deflate can be zlib-wrapped or raw deflate. Try zlib first, then raw deflate.
	zlibReader, err := zlib.NewReader(bytes.NewReader(body))
	if err == nil {
		defer func() { _ = zlibReader.Close() }()
		return io.ReadAll(zlibReader)
	}

	flateReader := flate.NewReader(bytes.NewReader(body))
	defer func() { _ = flateReader.Close() }()
	return io.ReadAll(flateReader)
}

func decodeBrotliBody(body []byte) ([]byte, error) {
	reader := brotli.NewReader(bytes.NewReader(body))
	return io.ReadAll(reader)
}

// isTextLikeContent returns true if the response content type is indexable text.
func isTextLikeContent(contentType string, body []byte) bool {
	mediaType := normalizeMediaType(contentType)
	if mediaType == "" && len(body) > 0 {
		mediaType = normalizeMediaType(http.DetectContentType(body))
	}

	// Be permissive when content type is missing.
	if mediaType == "" {
		return true
	}

	if strings.HasPrefix(mediaType, "text/") {
		return true
	}

	switch mediaType {
	case "application/xhtml+xml",
		"application/xml",
		"text/xml",
		"application/rss+xml",
		"application/atom+xml",
		"application/json",
		"application/ld+json":
		return true
	default:
		return false
	}
}

func normalizeMediaType(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if contentType == "" {
		return ""
	}
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	return contentType
}

// extractFallbackHTML extracts title and script/style-stripped body HTML from raw HTML.
func extractFallbackHTML(html string) (string, string) {
	if html == "" {
		return "", ""
	}
	if !utf8.ValidString(html) {
		html = strings.ToValidUTF8(html, " ")
	}

	title := ""
	if match := titleTagRegex.FindStringSubmatch(html); len(match) > 1 {
		title = normalizeWhitespace(stripHTMLTags(match[1]))
	}

	bodyText := html
	if match := bodyTagRegex.FindStringSubmatch(html); len(match) > 1 {
		bodyText = match[1]
	}
	bodyText = scriptStyleRegex.ReplaceAllString(bodyText, " ")

	return title, bodyText
}

func stripHTMLTags(value string) string {
	return allTagsRegex.ReplaceAllString(value, " ")
}

func normalizeWhitespace(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return whitespaceRunRegex.ReplaceAllString(value, " ")
}

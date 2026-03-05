package crawler

import (
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var (
	quotedURLRegex = regexp.MustCompile(`(?is)(?:"|')((?:https?:)?//[^"'\s<>]+|\\?/[^"'\s<>]+)(?:"|')`)

	nonNavigableExt = map[string]struct{}{
		".js":    {},
		".css":   {},
		".png":   {},
		".jpg":   {},
		".jpeg":  {},
		".gif":   {},
		".webp":  {},
		".svg":   {},
		".ico":   {},
		".woff":  {},
		".woff2": {},
		".ttf":   {},
		".otf":   {},
		".eot":   {},
		".map":   {},
		".pdf":   {},
		".zip":   {},
		".gz":    {},
		".mp4":   {},
		".mov":   {},
		".webm":  {},
		".mp3":   {},
		".wav":   {},
	}

	nonNavigablePathPrefixes = []string{
		"/img/",
		"/images/",
		"/contentimages/",
		"/productimages/",
		"/js/",
		"/css/",
		"/fonts/",
		"/assets/",
		"/static/",
	}
)

// extractEmbeddedLinks finds navigable URLs embedded in script/JSON blobs.
// This helps SPA-heavy pages that do not expose most routes via <a href>.
func extractEmbeddedLinks(rawContent string, pageURL *url.URL) []string {
	if rawContent == "" || pageURL == nil {
		return nil
	}

	matches := quotedURLRegex.FindAllStringSubmatch(rawContent, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	links := make([]string, 0, 64)

	for _, m := range matches {
		if len(m) < 2 {
			continue
		}

		raw := decodeEscapedURL(m[1])
		if raw == "" {
			continue
		}

		resolvedURL, ok := resolveCandidateURL(raw, pageURL)
		if !ok {
			continue
		}

		if !isLikelyNavigableURL(resolvedURL) {
			continue
		}

		normalized := NormalizeURL(resolvedURL.String())
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}

		seen[normalized] = struct{}{}
		links = append(links, normalized)
	}

	return links
}

func decodeEscapedURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	decoded := raw
	if strings.Contains(raw, `\`) {
		if v, err := strconv.Unquote(`"` + raw + `"`); err == nil {
			decoded = v
		}
	}

	decoded = strings.TrimSpace(decoded)
	if decoded == "" {
		return ""
	}

	if strings.HasPrefix(decoded, "\\/") {
		decoded = strings.ReplaceAll(decoded, "\\/", "/")
	}

	return decoded
}

func resolveCandidateURL(candidate string, pageURL *url.URL) (*url.URL, bool) {
	if pageURL == nil {
		return nil, false
	}

	if strings.HasPrefix(candidate, "//") {
		candidate = pageURL.Scheme + ":" + candidate
	}

	ref, err := url.Parse(candidate)
	if err != nil {
		return nil, false
	}

	resolved := pageURL.ResolveReference(ref)
	if resolved == nil {
		return nil, false
	}

	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return nil, false
	}

	return resolved, true
}

func isLikelyNavigableURL(u *url.URL) bool {
	if u == nil {
		return false
	}

	p := strings.TrimSpace(u.Path)
	if p == "" {
		p = "/"
	}

	lowerPath := strings.ToLower(p)
	for _, prefix := range nonNavigablePathPrefixes {
		if strings.HasPrefix(lowerPath, prefix) {
			return false
		}
	}

	ext := strings.ToLower(path.Ext(lowerPath))
	if ext != "" {
		if _, blocked := nonNavigableExt[ext]; blocked {
			return false
		}
	}

	return true
}

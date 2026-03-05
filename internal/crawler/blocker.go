// Package crawler provides web crawling functionality for gosearch.
package crawler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ResponseAnalyzer analyzes HTTP responses to detect blocks and rate limits.
type ResponseAnalyzer struct {
	detector *BlockDetector
}

// NewResponseAnalyzer creates a new response analyzer.
func NewResponseAnalyzer(detector *BlockDetector) *ResponseAnalyzer {
	return &ResponseAnalyzer{
		detector: detector,
	}
}

// AnalyzeResponse analyzes an HTTP response and detects if the crawler was blocked.
// Returns true if the response indicates a block or rate limit.
func (ra *ResponseAnalyzer) AnalyzeResponse(resp *http.Response) (blocked bool, reason string) {
	if resp == nil {
		return false, ""
	}

	statusCode := resp.StatusCode
	parsedURL, _ := url.Parse(resp.Request.URL.String())
	domain := parsedURL.Host

	// Check for obvious block status codes
	switch statusCode {
	case 429:
		ra.detector.MarkRateLimited(domain)
		return true, fmt.Sprintf("rate limited (429) by %s", domain)

	case 403:
		ra.detector.MarkBlocked(domain, statusCode, "access forbidden (403)")
		return true, fmt.Sprintf("blocked (403) by %s", domain)

	case 503:
		// Service unavailable - might be temporary
		ra.detector.MarkRateLimited(domain)
		return true, fmt.Sprintf("service unavailable (503) by %s", domain)
	}

	// Check for CAPTCHA indicators in response headers
	if ra.hasCaptchaHeaders(resp) {
		ra.detector.MarkBlocked(domain, statusCode, "CAPTCHA detected")
		return true, fmt.Sprintf("CAPTCHA detected by %s", domain)
	}

	return false, ""
}

// hasCaptchaHeaders checks for CAPTCHA indicators in response headers.
func (ra *ResponseAnalyzer) hasCaptchaHeaders(resp *http.Response) bool {
	// Check for common CAPTCHA-related headers
	captchaIndicators := []string{
		"x-captcha",
		"x-captcha-verify",
		"cf-challenge",                // Cloudflare
		"x-frame-options: sameorigin", // Sometimes used with CAPTCHA
	}

	for _, indicator := range captchaIndicators {
		if resp.Header.Get(indicator) != "" {
			return true
		}
	}

	// Check for Cloudflare challenge
	server := resp.Header.Get("Server")
	if strings.Contains(strings.ToLower(server), "cloudflare") {
		// Cloudflare protection might be active
		return true
	}

	return false
}

// ShouldRetry returns true if a request should be retried based on the response.
func (ra *ResponseAnalyzer) ShouldRetry(resp *http.Response, attemptNum int) bool {
	if resp == nil {
		return false
	}

	// Don't retry if we've exceeded max retries
	if attemptNum >= 3 {
		return false
	}

	// Retry on rate limit (429)
	if resp.StatusCode == 429 {
		return true
	}

	// Retry on server error (5xx)
	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		return true
	}

	// Don't retry on client errors (4xx) except 429
	if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
		return false
	}

	// Retry on network errors (no response)
	if resp.StatusCode == 0 {
		return true
	}

	return false
}

// GetRetryDelay returns the delay before retry based on the response.
func (ra *ResponseAnalyzer) GetRetryDelay(resp *http.Response, attemptNum int) time.Duration {
	if resp == nil {
		return time.Duration(attemptNum+1) * 5 * time.Second
	}

	// Check for Retry-After header
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		// Try to parse as seconds
		var seconds int
		if _, err := fmt.Sscanf(retryAfter, "%d", &seconds); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}

	// Exponential backoff based on attempt number
	switch resp.StatusCode {
	case 429:
		// Rate limit - longer delay
		return time.Duration(attemptNum+1) * 30 * time.Second

	case 503:
		// Service unavailable - moderate delay
		return time.Duration(attemptNum+1) * 10 * time.Second

	default:
		// Standard exponential backoff
		return time.Duration(attemptNum+1) * 5 * time.Second
	}
}

// GetRetryAfter returns the time after which a blocked domain can be retried.
// Returns zero time if the domain is not blocked.
func (ra *ResponseAnalyzer) GetRetryAfter(domain string) time.Time {
	if ra.detector == nil {
		return time.Time{}
	}

	ra.detector.mu.RLock()
	defer ra.detector.mu.RUnlock()

	// Check blocked domains
	if info, exists := ra.detector.blockedDomains[domain]; exists {
		return info.RetryAfter
	}

	// Check rate-limited domains
	if info, exists := ra.detector.rateLimitedDomains[domain]; exists {
		return info.RetryAfter
	}

	return time.Time{}
}

// IsBlocked returns true if a domain is currently blocked.
func (ra *ResponseAnalyzer) IsBlocked(domain string) bool {
	if ra.detector == nil {
		return false
	}

	return ra.detector.IsBlocked(domain)
}

// IsRateLimited returns true if a domain is currently rate-limited.
func (ra *ResponseAnalyzer) IsRateLimited(domain string) bool {
	if ra.detector == nil {
		return false
	}

	return ra.detector.IsRateLimited(domain)
}

// ExtractDomain extracts the domain from a URL.
func ExtractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return parsed.Host
}

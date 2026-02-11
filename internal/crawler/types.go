// Package crawler provides web crawling functionality for gosearch.
//
// It uses the Colly framework for concurrent web scraping with
// configurable worker pools, politeness policies, and URL deduplication.
package crawler

import (
	"context"
	"sync"
	"time"
)

// Config holds the crawler configuration.
type Config struct {
	MaxWorkers        int           // Maximum number of concurrent workers
	MaxDepth          int           // Maximum crawl depth from seed URLs
	Delay             time.Duration // Delay between requests
	AllowedDomains    []string      // Allowed domains (empty = all)
	DisallowedPaths   []string      // Disallowed URL path prefixes
	UserAgent         string        // User-Agent string
	Timeout           time.Duration // Request timeout
	RespectRobots     bool          // Whether to respect robots.txt
	HeaderProfile     string        // Header profile to use (chrome, firefox, safari)
	EnableCookies     bool          // Enable cookie persistence
	EnableBlockDetect bool          // Enable block detection and backoff
	MaxRetries        int           // Maximum retries for failed requests
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		MaxWorkers:        10,
		MaxDepth:          3,
		Delay:             1000 * time.Millisecond,
		AllowedDomains:    nil,
		DisallowedPaths:   nil,
		UserAgent:         "GoSearch/1.0 (+https://github.com/abuiliazeed/gosearch)",
		Timeout:           30 * time.Second,
		RespectRobots:     true,
		HeaderProfile:     "chrome",
		EnableCookies:     true,
		EnableBlockDetect: true,
		MaxRetries:        3,
	}
}

// URL represents a URL in the frontier queue.
type URL struct {
	URL      string
	Depth    int
	Parent   string // Parent URL that led to this URL
	Priority int    // Lower number = higher priority
}

// CrawlResult represents the result of crawling a single URL.
type CrawlResult struct {
	Success bool
	URL     string
	Error   error
	Links   []string
	Depth   int
	Title   string
	Content string
	HTML    string
}

// Stats represents crawler statistics.
type Stats struct {
	URLsCrawled        int
	URLsQueued         int
	URLsFailed         int
	TotalLinks         int
	StartTime          time.Time
	EndTime            time.Time
	DomainCount        map[string]int
	BlockedDomains     map[string]int // Domains that blocked us
	RateLimitedDomains map[string]int // Domains that rate limited us
}

// Crawler is the main crawler interface.
type Crawler interface {
	// Start begins crawling from the given seed URLs.
	Start(ctx context.Context, seeds []string) error

	// Stop gracefully stops the crawler.
	Stop() error

	// Stats returns the current crawler statistics.
	Stats() *Stats

	// IsComplete returns true if crawling is complete.
	IsComplete() bool
}

// HeaderProfile represents a set of HTTP headers to mimic a specific browser.
type HeaderProfile struct {
	Name            string
	UserAgent       string
	Accept          string
	AcceptLanguage  string
	AcceptEncoding  string
	Connec          string
	SecCHUA         string
	SecCHUAMobile   string
	SecCHUAPlatform string
	SecFetchDest    string
	SecFetchMode    string
	SecFetchSite    string
	SecFetchUser    string
	UpgradeInsecure string
}

// BlockInfo holds information about a domain that blocked or rate-limited us.
type BlockInfo struct {
	Domain      string
	StatusCode  int
	Reason      string
	LastBlocked time.Time
	RetryAfter  time.Time
	BackoffTime time.Duration
}

// BlockDetector tracks blocked and rate-limited domains.
type BlockDetector struct {
	blockedDomains     map[string]*BlockInfo
	rateLimitedDomains map[string]*BlockInfo
	mu                 sync.RWMutex
}

// NewBlockDetector creates a new block detector.
func NewBlockDetector() *BlockDetector {
	return &BlockDetector{
		blockedDomains:     make(map[string]*BlockInfo),
		rateLimitedDomains: make(map[string]*BlockInfo),
	}
}

// IsBlocked returns true if the domain is currently blocked.
func (bd *BlockDetector) IsBlocked(domain string) bool {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	if info, exists := bd.blockedDomains[domain]; exists {
		if time.Now().Before(info.RetryAfter) {
			return true
		}
		// Retry time has passed, remove the block
		delete(bd.blockedDomains, domain)
	}

	return false
}

// IsRateLimited returns true if the domain is currently rate-limited.
func (bd *BlockDetector) IsRateLimited(domain string) bool {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	if info, exists := bd.rateLimitedDomains[domain]; exists {
		if time.Now().Before(info.RetryAfter) {
			return true
		}
		// Retry time has passed, remove the limit
		delete(bd.rateLimitedDomains, domain)
	}

	return false
}

// MarkBlocked marks a domain as blocked until the retry time.
func (bd *BlockDetector) MarkBlocked(domain string, statusCode int, reason string) {
	bd.mu.Lock()
	defer bd.mu.Unlock()

	// Calculate exponential backoff based on previous blocks
	backoff := 5 * time.Minute
	if info, exists := bd.blockedDomains[domain]; exists {
		backoff = info.BackoffTime * 2
		if backoff > 2*time.Hour {
			backoff = 2 * time.Hour
		}
	}

	bd.blockedDomains[domain] = &BlockInfo{
		Domain:      domain,
		StatusCode:  statusCode,
		Reason:      reason,
		LastBlocked: time.Now(),
		RetryAfter:  time.Now().Add(backoff),
		BackoffTime: backoff,
	}
}

// MarkRateLimited marks a domain as rate-limited until the retry time.
func (bd *BlockDetector) MarkRateLimited(domain string) {
	bd.mu.Lock()
	defer bd.mu.Unlock()

	// Calculate exponential backoff based on previous rate limits
	backoff := 1 * time.Minute
	if info, exists := bd.rateLimitedDomains[domain]; exists {
		backoff = info.BackoffTime * 2
		if backoff > 1*time.Hour {
			backoff = 1 * time.Hour
		}
	}

	bd.rateLimitedDomains[domain] = &BlockInfo{
		Domain:      domain,
		StatusCode:  429,
		Reason:      "Rate limited",
		LastBlocked: time.Now(),
		RetryAfter:  time.Now().Add(backoff),
		BackoffTime: backoff,
	}
}

// GetBlockedDomains returns a list of currently blocked domains.
func (bd *BlockDetector) GetBlockedDomains() []string {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	domains := make([]string, 0, len(bd.blockedDomains))
	for domain := range bd.blockedDomains {
		domains = append(domains, domain)
	}
	return domains
}

// GetRateLimitedDomains returns a list of currently rate-limited domains.
func (bd *BlockDetector) GetRateLimitedDomains() []string {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	domains := make([]string, 0, len(bd.rateLimitedDomains))
	for domain := range bd.rateLimitedDomains {
		domains = append(domains, domain)
	}
	return domains
}

// Clear removes all blocks and rate limits.
func (bd *BlockDetector) Clear() {
	bd.mu.Lock()
	defer bd.mu.Unlock()

	bd.blockedDomains = make(map[string]*BlockInfo)
	bd.rateLimitedDomains = make(map[string]*BlockInfo)
}

package crawler

import (
	"context"
	"fmt"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gocolly/colly/v2"

	"github.com/abuiliazeed/gosearch/internal/storage"
)

// CollyCrawler implements the Crawler interface using the Colly framework.
type CollyCrawler struct {
	// All pointer fields first (8 bytes each)
	config           *Config
	frontier         *Frontier
	politeness       *PolitenessManager
	dedupe           *Deduplicator
	docStore         *storage.DocumentStore
	cookieManager    *CookieManager
	blockDetector    *BlockDetector
	responseAnalyzer *ResponseAnalyzer
	stats            *Stats
	savedDocIDs      map[string]struct{}
	ctx              context.Context
	cancel           context.CancelFunc
	// Value types ordered by size (largest first)
	headerProfile HeaderProfile  // ~200 bytes
	mu            sync.RWMutex   // 24 bytes
	wg            sync.WaitGroup // 16 bytes
	cancelOnce    sync.Once      // 8 bytes
	pendingReqs   int32          // 4 bytes
	complete      bool           // 1 byte
}

// NewCollyCrawler creates a new Colly-based crawler.
func NewCollyCrawler(config *Config, docStore *storage.DocumentStore) (*CollyCrawler, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create cookie manager if enabled
	var cookieManager *CookieManager
	var err error
	if config.EnableCookies {
		cookieManager, err = NewCookieManager()
		if err != nil {
			return nil, fmt.Errorf("failed to create cookie manager: %w", err)
		}
	}

	// Create block detector
	blockDetector := NewBlockDetector()

	// Create response analyzer
	responseAnalyzer := NewResponseAnalyzer(blockDetector)

	// Get header profile
	headerProfile := GetHeaderProfile(config.HeaderProfile)

	return &CollyCrawler{
		config:           config,
		frontier:         NewFrontier(),
		politeness:       NewPolitenessManager(config.Delay, config.UserAgent, config.RespectRobots),
		dedupe:           NewDeduplicator(true, 100000), // Use Bloom filter for 100k URLs
		docStore:         docStore,
		cookieManager:    cookieManager,
		blockDetector:    blockDetector,
		responseAnalyzer: responseAnalyzer,
		headerProfile:    headerProfile,
		stats: &Stats{
			DomainCount:        make(map[string]int),
			BlockedDomains:     make(map[string]int),
			RateLimitedDomains: make(map[string]int),
			StartTime:          time.Now(),
		},
		savedDocIDs: make(map[string]struct{}),
	}, nil
}

// Start begins crawling from the given seed URLs.
func (c *CollyCrawler) Start(ctx context.Context, seeds []string) error {
	c.ctx, c.cancel = context.WithCancel(ctx)

	// Add seed URLs to frontier
	for _, seed := range seeds {
		normalized := NormalizeURL(seed)
		c.tryEnqueueURL(&URL{
			URL:      normalized,
			Depth:    0,
			Priority: 0,
		})
	}

	// Create worker pool
	errChan := make(chan error, c.config.MaxWorkers)
	doneChan := make(chan struct{})

	for i := 0; i < c.config.MaxWorkers; i++ {
		c.wg.Add(1)
		go c.worker(i, errChan)
	}

	// Wait for all workers to complete
	go func() {
		c.wg.Wait()
		c.mu.Lock()
		c.complete = true
		c.stats.EndTime = time.Now()
		c.mu.Unlock()
		close(doneChan)
	}()

	// Wait for an error, external cancellation, or worker completion.
	for {
		select {
		case err := <-errChan:
			// Error occurred, cancel and wait for workers
			c.cancelOnce.Do(func() { c.cancel() })
			c.wg.Wait()
			return err
		case <-ctx.Done():
			// Context canceled externally, wait for workers
			c.cancelOnce.Do(func() { c.cancel() })
			c.wg.Wait()
			return nil
		case <-doneChan:
			// All workers completed naturally.
			c.cancelOnce.Do(func() { c.cancel() })
			return nil
		}
	}
}

// worker processes URLs from the frontier.
func (c *CollyCrawler) worker(id int, errChan chan<- error) {
	defer c.wg.Done()

	// Create a new Colly collector for this worker
	collector := c.createCollector(id, errChan)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// Get next URL from frontier
			url := c.frontier.Pop()
			if url == nil {
				// No more URLs, check if we should wait or exit
				// Don't exit while there are pending async requests
				if c.frontier.Len() == 0 && atomic.LoadInt32(&c.pendingReqs) == 0 {
					return
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Note: URL is already marked as seen when added to frontier
			// No need to check deduplicator again here

			// Check if domain is blocked or rate-limited (if block detection is enabled)
			if c.config.EnableBlockDetect {
				parsedURL, err := neturl.Parse(url.URL)
				if err == nil {
					domain := parsedURL.Host

					// Check if blocked
					if c.responseAnalyzer.IsBlocked(domain) {
						c.mu.Lock()
						c.stats.URLsFailed++
						c.mu.Unlock()
						continue
					}

					// Check if rate-limited
					if c.responseAnalyzer.IsRateLimited(domain) {
						c.mu.Lock()
						c.stats.URLsFailed++
						c.mu.Unlock()
						continue
					}
				}
			}

			// Check politeness
			if err := c.politeness.Acquire(url.URL); err != nil {
				c.mu.Lock()
				c.stats.URLsFailed++
				c.mu.Unlock()
				continue
			}

			// Store depth in request context (must be stored as string for Colly)
			ctx := colly.NewContext()
			ctx.Put("depth", fmt.Sprintf("%d", url.Depth))
			ctx.Put("request_finalized", "0")

			// Track pending request
			atomic.AddInt32(&c.pendingReqs, 1)

			// Visit URL with context
			if err := collector.Request("GET", url.URL, nil, ctx, nil); err != nil {
				// Request-level failures are non-fatal to the whole crawl.
				// If callbacks did not finalize this request, decrement pending here.
				if ctx.Get("request_finalized") != "1" {
					atomic.AddInt32(&c.pendingReqs, -1)
					c.mu.Lock()
					c.stats.URLsFailed++
					c.mu.Unlock()
				}
				c.politeness.Release(url.URL)
				continue
			}

			// Release politeness lock
			c.politeness.Release(url.URL)
		}
	}
}

// createCollector creates a new Colly collector with the configured callbacks.
// The errChan parameter is reserved for future error reporting.
func (c *CollyCrawler) createCollector(_ int, errChan chan<- error) *colly.Collector {
	_ = errChan // Reserved for future use
	collector := colly.NewCollector(
		colly.UserAgent(c.headerProfile.UserAgent),
		colly.MaxDepth(c.config.MaxDepth),
		colly.Async(false), // Use sync mode to simplify completion detection
	)

	// Set timeout
	collector.SetRequestTimeout(30 * time.Second)

	// Limit to allowed domains if specified
	if len(c.config.AllowedDomains) > 0 {
		collector.AllowedDomains = c.config.AllowedDomains
	}

	// On request - apply browser headers
	collector.OnRequest(func(r *colly.Request) {
		c.mu.Lock()
		c.stats.URLsCrawled++
		c.mu.Unlock()

		// Apply browser headers to the request
		if r.Method == "GET" || r.Method == "HEAD" {
			// Apply headers from the profile directly to Colly request headers
			r.Headers.Set("User-Agent", c.headerProfile.UserAgent)
			r.Headers.Set("Accept", c.headerProfile.Accept)
			r.Headers.Set("Accept-Language", c.headerProfile.AcceptLanguage)
			r.Headers.Set("Accept-Encoding", c.headerProfile.AcceptEncoding)

			// Sec-CH-UA headers (Client Hints)
			if c.headerProfile.SecCHUA != "" {
				r.Headers.Set("Sec-CH-UA", c.headerProfile.SecCHUA)
			}
			if c.headerProfile.SecCHUAMobile != "" {
				r.Headers.Set("Sec-CH-UA-Mobile", c.headerProfile.SecCHUAMobile)
			}
			if c.headerProfile.SecCHUAPlatform != "" {
				r.Headers.Set("Sec-CH-UA-Platform", c.headerProfile.SecCHUAPlatform)
			}

			// Sec-Fetch headers
			if c.headerProfile.SecFetchDest != "" {
				r.Headers.Set("Sec-Fetch-Dest", c.headerProfile.SecFetchDest)
			}
			if c.headerProfile.SecFetchMode != "" {
				r.Headers.Set("Sec-Fetch-Mode", c.headerProfile.SecFetchMode)
			}
			if c.headerProfile.SecFetchSite != "" {
				r.Headers.Set("Sec-Fetch-Site", c.headerProfile.SecFetchSite)
			}
			if c.headerProfile.SecFetchUser != "" {
				r.Headers.Set("Sec-Fetch-User", c.headerProfile.SecFetchUser)
			}

			// Additional headers for realism
			r.Headers.Set("DNT", "1")
			r.Headers.Set("Connection", "keep-alive")
		}

		// Apply cookies if enabled
		if c.cookieManager != nil {
			cookies := c.cookieManager.GetCookies(r.URL)
			for _, cookie := range cookies {
				r.Headers.Add("Cookie", cookie.String())
			}
		}
	})

	// On response - prepare to parse HTML
	collector.OnResponse(func(r *colly.Response) {
		// Parse URL
		parsedURL, err := neturl.Parse(r.Request.URL.String())
		if err != nil {
			return
		}

		c.mu.Lock()
		c.stats.DomainCount[parsedURL.Host]++
		c.mu.Unlock()

		// Get depth from context
		depth := 0
		if d := r.Ctx.Get("depth"); d != "" {
			// Convert string to int (Colly stores context values as strings)
			if depthInt, err := strconv.Atoi(d); err == nil {
				depth = depthInt
			}
		}

		contentEncoding := ""
		contentType := ""
		if r.Headers != nil {
			contentEncoding = r.Headers.Get("Content-Encoding")
			contentType = r.Headers.Get("Content-Type")
		}

		// Decode content-encoded bodies to avoid indexing compressed bytes.
		decodedBody, err := decodeResponseBody(r.Body, contentEncoding)
		if err == nil {
			r.Body = decodedBody
			if r.Headers != nil {
				r.Headers.Del("Content-Encoding")
			}
		} else {
			decodedBody = r.Body
		}

		// Skip binary/non-text content to avoid polluting the index.
		if !isTextLikeContent(contentType, decodedBody) {
			r.Ctx.Put("skip_save", "1")
			r.Ctx.Put("depth", fmt.Sprintf("%d", depth))
			return
		}

		// Store HTML and depth in context for OnScraped callback.
		r.Ctx.Put("html", string(decodedBody))
		r.Ctx.Put("depth", fmt.Sprintf("%d", depth))

		// SPA-heavy pages often embed routes inside script/JSON payloads.
		// Extract those links to improve crawl coverage beyond <a href>.
		if c.config.MaxDepth > 0 && depth < c.config.MaxDepth {
			for _, embedded := range extractEmbeddedLinks(string(decodedBody), r.Request.URL) {
				if !c.isAllowedDomain(embedded) {
					continue
				}
				if c.isDisallowed(embedded) {
					continue
				}
				c.tryEnqueueURL(&URL{
					URL:      embedded,
					Depth:    depth + 1,
					Parent:   r.Request.URL.String(),
					Priority: depth + 1,
				})
			}
		}
	})

	// On HTML - extract links
	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		absoluteURL := e.Request.AbsoluteURL(link)

		// Parse URL
		parsedURL, err := neturl.Parse(absoluteURL)
		if err != nil {
			return
		}

		// Check if URL is valid
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return
		}

		// Normalize URL
		normalizedURL := NormalizeURL(absoluteURL)

		// Check if disallowed
		if c.isDisallowed(normalizedURL) {
			return
		}
		if !c.isAllowedDomain(normalizedURL) {
			return
		}

		// Get current depth
		currentDepth := 0
		if d := e.Request.Ctx.Get("depth"); d != "" {
			if depthInt, err := strconv.Atoi(d); err == nil {
				currentDepth = depthInt
			}
		}

		// Don't exceed max depth
		// Explicitly handle depth 0 as "seed only"
		if c.config.MaxDepth <= 0 || currentDepth >= c.config.MaxDepth {
			return
		}

		// Add to frontier while atomically enforcing dedupe and queue limit.
		c.tryEnqueueURL(&URL{
			URL:      normalizedURL,
			Depth:    currentDepth + 1,
			Parent:   e.Request.URL.String(),
			Priority: currentDepth + 1,
		})
	})

	// On scraped - save the document after all HTML parsing is complete
	collector.OnScraped(func(r *colly.Response) {
		r.Ctx.Put("request_finalized", "1")

		// Mark request as completed only after all parsing callbacks are done.
		// This avoids early shutdown while links are still being extracted.
		defer atomic.AddInt32(&c.pendingReqs, -1)

		// Skip documents that are not indexable text content.
		if r.Ctx.Get("skip_save") == "1" {
			return
		}

		// Get values from context
		html := r.Ctx.Get("html")
		depthStr := r.Ctx.Get("depth")
		extracted := extractMarkdownPage(html, r.Request.URL.String())

		// Parse depth
		depth := 0
		if depthStr != "" {
			if d, err := strconv.Atoi(depthStr); err == nil {
				depth = d
			}
		}

		// Create document
		doc := &storage.Document{
			URL:             r.Request.URL.String(),
			Title:           extracted.Title,
			ContentMarkdown: extracted.ContentMarkdown,
			Links:           extracted.Links,
			CrawledAt:       time.Now(),
			Depth:           depth,
		}

		// Save to storage
		if err := c.docStore.Save(doc); err != nil {
			c.mu.Lock()
			c.stats.URLsFailed++
			c.mu.Unlock()
			return
		}

		c.mu.Lock()
		c.savedDocIDs[doc.ID] = struct{}{}
		c.mu.Unlock()
	})

	// On error - detect blocks and rate limits
	collector.OnError(func(r *colly.Response, _ error) {
		r.Ctx.Put("request_finalized", "1")

		// Mark request as completed
		atomic.AddInt32(&c.pendingReqs, -1)

		c.mu.Lock()
		c.stats.URLsFailed++
		c.mu.Unlock()

		// Analyze response for blocks if enabled
		if c.config.EnableBlockDetect && r != nil {
			parsedURL, _ := neturl.Parse(r.Request.URL.String())
			domain := parsedURL.Host

			// Create an http.Response from colly.Response for analysis
			httpResp := &http.Response{
				StatusCode: r.StatusCode,
				Header:     make(http.Header),
				Request: &http.Request{
					URL: r.Request.URL,
				},
			}

			// Copy headers from colly response
			if r.Headers != nil {
				httpResp.Header = *r.Headers
			}

			blocked, reason := c.responseAnalyzer.AnalyzeResponse(httpResp)
			if blocked {
				c.mu.Lock()
				if r.StatusCode == 429 {
					c.stats.RateLimitedDomains[domain]++
				} else {
					c.stats.BlockedDomains[domain]++
				}
				c.mu.Unlock()

				// Log the block
				fmt.Printf("⚠️  %s\n", reason)
			}
		}
	})

	// On response headers - check for blocks before processing
	collector.OnResponseHeaders(func(r *colly.Response) {
		if c.config.EnableBlockDetect {
			parsedURL, _ := neturl.Parse(r.Request.URL.String())
			domain := parsedURL.Host

			// Create an http.Response from colly.Response for analysis
			httpResp := &http.Response{
				StatusCode: r.StatusCode,
				Header:     make(http.Header),
				Request: &http.Request{
					URL: r.Request.URL,
				},
			}

			// Copy headers from colly response
			if r.Headers != nil {
				for key, values := range *r.Headers {
					for _, value := range values {
						httpResp.Header.Add(key, value)
					}
				}
			}

			// Check for blocks immediately when headers are received
			blocked, reason := c.responseAnalyzer.AnalyzeResponse(httpResp)
			if blocked {
				c.mu.Lock()
				if r.StatusCode == 429 {
					c.stats.RateLimitedDomains[domain]++
				} else {
					c.stats.BlockedDomains[domain]++
				}
				c.mu.Unlock()

				// Log the block
				fmt.Printf("⚠️  %s\n", reason)
			}
		}
	})

	return collector
}

// isDisallowed checks if a URL matches any disallowed path pattern.
func (c *CollyCrawler) isDisallowed(urlStr string) bool {
	if len(c.config.DisallowedPaths) == 0 {
		return false
	}

	parsedURL, err := neturl.Parse(urlStr)
	if err != nil {
		return false
	}

	for _, pattern := range c.config.DisallowedPaths {
		if parsedURL.Path == pattern || (len(parsedURL.Path) >= len(pattern) && parsedURL.Path[:len(pattern)] == pattern) {
			return true
		}
	}

	return false
}

// isAllowedDomain checks whether URL host is within configured allowed domains.
func (c *CollyCrawler) isAllowedDomain(urlStr string) bool {
	if len(c.config.AllowedDomains) == 0 {
		return true
	}

	parsedURL, err := neturl.Parse(urlStr)
	if err != nil {
		return false
	}

	host := strings.ToLower(parsedURL.Hostname())
	if host == "" {
		return false
	}

	for _, allowed := range c.config.AllowedDomains {
		allowed = strings.ToLower(strings.TrimSpace(allowed))
		if allowed == "" {
			continue
		}
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return true
		}
	}

	return false
}

// Stop gracefully stops the crawler.
func (c *CollyCrawler) Stop() error {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
	return nil
}

// Stats returns the current crawler statistics.
func (c *CollyCrawler) Stats() *Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	statsCopy := *c.stats
	return &statsCopy
}

// IsComplete returns true if crawling is complete.
func (c *CollyCrawler) IsComplete() bool {
	return c.complete
}

// GetStatsString returns a formatted string of statistics.
func (c *CollyCrawler) GetStatsString() string {
	stats := c.Stats()
	duration := time.Since(stats.StartTime)
	if !stats.EndTime.IsZero() {
		duration = stats.EndTime.Sub(stats.StartTime)
	}

	return fmt.Sprintf(
		"URLs crawled: %d\nURLs queued: %d\nURLs failed: %d\nDuration: %s\nDomains: %d",
		stats.URLsCrawled,
		stats.URLsQueued,
		stats.URLsFailed,
		duration.Round(time.Second),
		len(stats.DomainCount),
	)
}

// GetSavedDocIDs returns document IDs that were saved during this crawl run.
func (c *CollyCrawler) GetSavedDocIDs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ids := make([]string, 0, len(c.savedDocIDs))
	for id := range c.savedDocIDs {
		ids = append(ids, id)
	}
	return ids
}

// tryEnqueueURL adds a URL to the frontier if it has not been seen and the
// configured total queue limit has not been reached.
func (c *CollyCrawler) tryEnqueueURL(u *URL) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.config.MaxQueueSize > 0 && c.stats.URLsQueued >= c.config.MaxQueueSize {
		return
	}

	normalized := NormalizeURL(u.URL)
	if c.dedupe.Seen(normalized) {
		return
	}
	c.dedupe.Add(normalized)
	u.URL = normalized

	c.frontier.Push(u)
	c.stats.URLsQueued++
}

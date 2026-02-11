package crawler

import (
	"context"
	"fmt"
	"net/http"
	neturl "net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abuiliazeed/gosearch/internal/storage"
	"github.com/gocolly/colly/v2"
)

// CollyCrawler implements the Crawler interface using the Colly framework.
type CollyCrawler struct {
	config           *Config
	frontier         *Frontier
	politeness       *PolitenessManager
	dedupe           *Deduplicator
	docStore         *storage.DocumentStore
	cookieManager    *CookieManager
	blockDetector    *BlockDetector
	responseAnalyzer *ResponseAnalyzer
	headerProfile    HeaderProfile
	stats            *Stats
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	cancelOnce       sync.Once       // Ensure cancel is only called once
	wg               sync.WaitGroup
	complete         bool
	pendingReqs      int32          // Track pending async requests
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
	}, nil
}

// Start begins crawling from the given seed URLs.
func (c *CollyCrawler) Start(ctx context.Context, seeds []string) error {
	c.ctx, c.cancel = context.WithCancel(ctx)

	// Add seed URLs to frontier
	for _, seed := range seeds {
		normalized := NormalizeURL(seed)
		c.frontier.Push(&URL{
			URL:      normalized,
			Depth:    0,
			Priority: 0,
		})
		c.stats.URLsQueued++
	}

	// Create worker pool
	errChan := make(chan error, c.config.MaxWorkers)

	for i := 0; i < c.config.MaxWorkers; i++ {
		c.wg.Add(1)
		go c.worker(i, errChan)
	}

	// Wait for all workers to complete
	go func() {
		c.wg.Wait()
		c.complete = true
		c.stats.EndTime = time.Now()
	}()

	// Wait for context cancellation or all work done
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-errChan:
			// Error occurred, cancel and wait for workers
			c.cancelOnce.Do(func() { c.cancel() })
			c.wg.Wait()
			return err
		case <-ctx.Done():
			// Context cancelled externally, wait for workers
			c.cancelOnce.Do(func() { c.cancel() })
			c.wg.Wait()
			return nil
		case <-ticker.C:
			// Check if all work is complete
			c.mu.RLock()
			pending := atomic.LoadInt32(&c.pendingReqs)
			frontierLen := c.frontier.Len()
			allDone := c.complete || (frontierLen == 0 && pending == 0)
			c.mu.RUnlock()

			if allDone {
				// All work done, set complete flag and cancel
				c.mu.Lock()
				c.complete = true
				c.stats.EndTime = time.Now()
				c.mu.Unlock()
				// Cancel context and wait for workers to finish
				c.cancelOnce.Do(func() { c.cancel() })
				c.wg.Wait()
				return nil
			}
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

			// Check if URL has been seen
			if c.dedupe.Seen(url.URL) {
				continue
			}
			c.dedupe.Add(url.URL)

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
				if err == ErrDisallowed {
					c.mu.Lock()
					c.stats.URLsFailed++
					c.mu.Unlock()
					continue
				}
				errChan <- fmt.Errorf("politeness error for %s: %w", url.URL, err)
				continue
			}

			// Store depth in request context (must be stored as string for Colly)
			ctx := colly.NewContext()
			ctx.Put("depth", fmt.Sprintf("%d", url.Depth))

			// Track pending request
			atomic.AddInt32(&c.pendingReqs, 1)

			// Visit URL with context
			if err := collector.Request("GET", url.URL, nil, ctx, nil); err != nil {
				// Request failed immediately, callbacks won't fire - decrement counter
				atomic.AddInt32(&c.pendingReqs, -1)
				errChan <- fmt.Errorf("failed to visit %s: %w", url.URL, err)
				c.politeness.Release(url.URL)
				continue
			}

			// Release politeness lock
			c.politeness.Release(url.URL)
		}
	}
}

// createCollector creates a new Colly collector with the configured callbacks.
func (c *CollyCrawler) createCollector(workerID int, errChan chan<- error) *colly.Collector {
	collector := colly.NewCollector(
		colly.UserAgent(c.headerProfile.UserAgent),
		colly.MaxDepth(c.config.MaxDepth),
		colly.Async(true),
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
		// Mark request as completed
		atomic.AddInt32(&c.pendingReqs, -1)

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

		// Store HTML and depth in context for OnScraped callback
		r.Ctx.Put("html", string(r.Body))
		r.Ctx.Put("depth", fmt.Sprintf("%d", depth))
	})

	// On HTML - extract title
	collector.OnHTML("title", func(e *colly.HTMLElement) {
		e.Request.Ctx.Put("title", e.Text)
	})

	// On HTML - extract body text content
	collector.OnHTML("body", func(e *colly.HTMLElement) {
		e.Request.Ctx.Put("content", e.Text)
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

		// Get current depth
		currentDepth := 0
		if d := e.Request.Ctx.Get("depth"); d != "" {
			if depthInt, err := strconv.Atoi(d); err == nil {
				currentDepth = depthInt
			}
		}

		// Don't exceed max depth
		if currentDepth >= c.config.MaxDepth {
			return
		}

		// Add to frontier
		c.frontier.Push(&URL{
			URL:      normalizedURL,
			Depth:    currentDepth + 1,
			Parent:   e.Request.URL.String(),
			Priority: currentDepth + 1,
		})

		c.mu.Lock()
		c.stats.URLsQueued++
		c.mu.Unlock()
	})

	// On scraped - save the document after all HTML parsing is complete
	collector.OnScraped(func(r *colly.Response) {
		// Get values from context
		html := r.Ctx.Get("html")
		title := r.Ctx.Get("title")
		content := r.Ctx.Get("content")
		depthStr := r.Ctx.Get("depth")

		// Parse depth
		depth := 0
		if depthStr != "" {
			if d, err := strconv.Atoi(depthStr); err == nil {
				depth = d
			}
		}

		// Create document
		doc := &storage.Document{
			URL:       r.Request.URL.String(),
			Title:     title,
			HTML:      html,
			Content:   content,
			Links:     []string{},
			CrawledAt: time.Now(),
			Depth:     depth,
		}

		// Save to storage
		if err := c.docStore.Save(doc); err != nil {
			errChan <- fmt.Errorf("failed to save document %s: %w", doc.URL, err)
		}
	})

	// On error - detect blocks and rate limits
	collector.OnError(func(r *colly.Response, err error) {
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

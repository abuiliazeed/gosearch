# Anti-Blocking Strategies for gosearch

> This document researches and outlines strategies to avoid getting blocked by rate limiting and other anti-crawling measures that websites implement.

---

## Understanding Anti-Scraping Measures

Websites implement various techniques to detect and block web crawlers:

| Technique | Detection Method | Common Implementation |
|-----------|-----------------|----------------------|
| **Rate Limiting** | Request frequency per IP | Nginx rate limits, API gateways |
| **User-Agent Detection** | Invalid or missing UA string | Block unknown/missing UAs |
| **IP Blacklisting** | Repeat offenders | Firewall rules, WAF |
| **TLS Fingerprinting** | TLS handshake characteristics | JA3/JA4 fingerprint matching |
| **Browser Fingerprinting** | Canvas, WebGL, fonts | JavaScript challenges |
| **Behavioral Analysis** | Mouse movement, timing | Bot detection services |
| **CAPTCHA** | Suspicious patterns | reCAPTCHA, hCaptcha, Turnstile |
| **Honeypot Links** | Hidden/trap links | Invisible CSS links |
| **robots.txt Enforcement** | Policy violations | Server-side filtering |

---

## Foundational Strategies (Must Implement)

### 1. Respect robots.txt

Always parse and follow robots.txt directives. This is the "gentleman's agreement" of web crawling.

**Implementation:**
```go
// In internal/crawler/politeness.go (already implemented)
import "github.com/temoto/robotstxt"

func (p *PolitenessManager) CheckRobots(url string) error {
    robotsURL := getBaseURL(url) + "/robots.txt"
    robots, err := robotstxt.FromURL(robotsURL, nil)
    if err != nil {
        return err
    }

    group := robots.FindGroup(p.userAgent)
    if !group.Test(url) {
        return ErrDisallowed
    }

    // Respect crawl-delay if specified
    if delay := group.CrawlDelay; delay > 0 {
        p.delay = delay
    }

    return nil
}
```

**Best Practices:**
- Always fetch and parse robots.txt before crawling
- Respect `Crawl-delay` directive if specified
- Respect `Disallow` rules
- Cache robots.txt responses (don't refetch for every request)

**Sources:**
- [A Web Scraper's Guide to Robots.txt](https://www.scrapingbee.com/blog/robots-txt-web-scraping/)
- [Understanding robots.txt: The Beginner's Guide](https://dev.to/ikram_khan/understanding-robotstxt-the-beginners-guide-for-web-scrapers-241i)
- [Politely Scrape Websites by Following Robots.txt](https://proxyserver.com/web-scraping-crawling/politely-scrape-websites-by-following-robots-txt/)

### 2. Rate Limiting & Politeness Delays

Implement per-domain rate limiting to avoid overwhelming servers.

**Implementation:**
```go
// Enhanced rate limiter with adaptive delays
type RateLimiter struct {
    mu            sync.Mutex
    lastRequest   map[string]time.Time
    delays        map[string]time.Duration
    defaultDelay  time.Duration
    backoffFactor float64
}

func (rl *RateLimiter) Acquire(domain string) error {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    last, exists := rl.lastRequest[domain]
    delay := rl.delays[domain]
    if delay == 0 {
        delay = rl.defaultDelay
    }

    if exists && time.Since(last) < delay {
        sleepTime := delay - time.Since(last)
        time.Sleep(sleepTime)
    }

    // Adaptive backoff if we're getting rate limited
    if isRateLimited(domain) {
        delay = time.Duration(float64(delay) * rl.backoffFactor)
        rl.delays[domain] = delay
    }

    rl.lastRequest[domain] = time.Now()
    return nil
}
```

**Recommended Delays:**
- Default: 1-2 seconds between requests
- For small sites: 5-10 seconds
- After 429 response: Exponential backoff (2x, 4x, 8x...)

**Sources:**
- [What is polite crawling? | Firecrawl](https://www.firecrawl.dev/glossary/web-crawling-apis/what-is-polite-crawling)
- [How to crawl the web politely with Scrapy](https://www.zyte.com/blog/how-to-crawl-the-web-politely-with-scrapy/)

### 3. Realistic User-Agent Strings

Use valid, up-to-date User-Agent strings that match your HTTP client behavior.

**Implementation:**
```go
// User agent rotation
type UserAgentRotator struct {
    agents []string
    index  int
}

func NewUserAgentRotator() *UserAgentRotator {
    return &UserAgentRotator{
        agents: []string{
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
            "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
            "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1 Safari/605.1.15",
        },
    }
}

func (r *UserAgentRotator) Next() string {
    r.index = (r.index + 1) % len(r.agents)
    return r.agents[r.index]
}
```

**Key Headers:**
```go
headers := map[string]string{
    "User-Agent":      rotator.Next(),
    "Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
    "Accept-Language": "en-US,en;q=0.9",
    "Accept-Encoding": "gzip, deflate, br",
    "Connection":      "keep-alive",
    "DNT":             "1",
    "Upgrade-Insecure-Requests": "1",
    "Sec-Fetch-Dest":  "document",
    "Sec-Fetch-Mode":  "navigate",
    "Sec-Fetch-Site":  "none",
    "Sec-Fetch-User":  "?1",
    "Sec-CH-UA":       `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
    "Sec-CH-UA-Mobile": "?0",
    "Sec-CH-UA-Platform": `"Windows"`,
}
```

**Sources:**
- [14 Ways for Web Scraping Without Getting Blocked](https://www.zenrows.com/blog/web-scraping-without-getting-blocked)
- [Customizing your scraper to use actual browser headers](https://scrapfly.io/blog/posts/how-to-scrape-without-getting-blocked-tutorial)

---

## Intermediate Strategies (Should Implement)

### 4. Proxy Rotation

Rotate IP addresses to avoid per-IP rate limiting and blacklisting.

**Implementation:**
```go
// Proxy rotator for avoiding IP-based blocking
type ProxyRotator struct {
    proxies   []string
    index     int
    mu        sync.Mutex
    statistics map[string]*ProxyStats
}

type ProxyStats struct {
    SuccessCount  int
    FailureCount  int
    LastUsed      time.Time
    BlockedUntil  time.Time
}

func NewProxyRotator(proxies []string) *ProxyRotator {
    return &ProxyRotator{
        proxies: proxies,
        statistics: make(map[string]*ProxyStats),
    }
}

func (pr *ProxyRotator) Next() (string, error) {
    pr.mu.Lock()
    defer pr.mu.Unlock()

    // Find available proxy
    for i := 0; i < len(pr.proxies); i++ {
        pr.index = (pr.index + 1) % len(pr.proxies)
        proxy := pr.proxies[pr.index]
        stats := pr.statistics[proxy]

        if stats == nil {
            pr.statistics[proxy] = &ProxyStats{}
            return proxy, nil
        }

        if time.Now().After(stats.BlockedUntil) {
            return proxy, nil
        }
    }

    return "", errors.New("no available proxies")
}

func (pr *ProxyRotator) MarkFailure(proxy string, isHardFailure bool) {
    pr.mu.Lock()
    defer pr.mu.Unlock()

    stats := pr.statistics[proxy]
    if stats == nil {
        return
    }

    stats.FailureCount++
    if isHardFailure {
        // Block for 1 hour on hard failures (403, 429)
        stats.BlockedUntil = time.Now().Add(time.Hour)
    }
}
```

**Proxy Sources:**
- Residential proxy services (Bright Data, Smartproxy)
- Datacenter proxies (lower cost, easier to detect)
- Free proxy lists (unreliable, often already blocked)

**Best Practices:**
- Rotate every N requests (not every request)
- Monitor proxy health
- Implement backoff for failing proxies
- Use session-affinity for requests to same domain

**Sources:**
- [Rate-limiting | Apify Academy](https://docs.apify.com/academy/anti-scraping/techniques/rate-limiting)
- [Use Rotating Proxies for Web Scraping Anonymity](https://infatica.io/blog/how-to-crawl-a-website-without-getting-blocked/)
- [Choose proxies and rotate IP sessions](https://dev.to/apify/web-scraping-how-to-crawl-without-getting-blocked-587b)

### 5. Request Header Randomization

Beyond User-Agent, randomize other headers to appear more like different browsers.

**Implementation:**
```go
type HeaderProfile struct {
    UserAgent      string
    Accept         string
    AcceptLanguage string
    AcceptEncoding string
    SecCHUA        string
    SecCHUAPlatform string
}

var headerProfiles = []HeaderProfile{
    {
        UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/131.0.0.0",
        Accept:         "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
        AcceptLanguage: "en-US,en;q=0.9",
        AcceptEncoding: "gzip, deflate, br",
        SecCHUA:        `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
        SecCHUAPlatform: `"Windows"`,
    },
    {
        UserAgent:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/131.0.0.0",
        Accept:         "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
        AcceptLanguage: "en-GB,en-US;q=0.9,en;q=0.8",
        AcceptEncoding: "gzip, deflate, br",
        SecCHUA:        `"Chromium";v="131", "Google Chrome";v="131", "Not_A Brand";v="24"`,
        SecCHUAPlatform: `"macOS"`,
    },
}
```

### 6. Session & Cookie Management

Maintain cookies across requests to preserve session state.

**Implementation:**
```go
type SessionManager struct {
    jar     map[string][]*http.Cookie
    mu      sync.RWMutex
}

func (sm *SessionManager) GetCookies(url string) []*http.Cookie {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    u, _ := url.Parse(url)
    return sm.jar[u.Hostname()]
}

func (sm *SessionManager) SetCookies(url string, cookies []*http.Cookie) {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    u, _ := url.Parse(url)
    sm.jar[u.Hostname()] = cookies
}
```

**Best Practices:**
- Persist cookies across sessions
- Handle cookie expiration
- Respect cookie scope and domain
- Don't share sessions across different purposes

**Sources:**
- [Mastering Cookie Handling in Web Scraping](https://scrape.do/blog/web-scraping-cookies/)
- [Managing Cookies and Sessions in Web Scrapers](https://dataprixa.com/managing-cookies-and-sessions-in-web-scrapers-a-complete-guide/)

---

## Advanced Strategies (For Difficult Targets)

### 7. TLS Fingerprinting Mitigation

Modern anti-bot systems detect crawlers by analyzing TLS handshake characteristics (JA3/JA4 fingerprints).

**The Problem:**
Go's default TLS stack has a unique fingerprint that differs from real browsers.

**Detection Methods:**
- JA3/JA4 fingerprint matching
- TLS cipher suite order
- TLS extensions presence/order
- ALPN protocol list

**Mitigation Strategies:**

1. **Use Browser Automation Tools**
```go
// Use chromedp with UA spoofing
import (
    "github.com/chromedp/chromedp"
    "github.com/chromedp/chromedp/device"
)

options := []chromedp.ExecAllocatorOption{
    chromedp.Flag("disable-blink-features", "AutomationControlled"),
    chromedp.UserAgent(userAgent),
    device.Reset(),
}
```

2. **Use HTTP/2 with Proper Settings**
```go
transport := &http.Transport{
    ForceAttemptHTTP2: true,
    TLSClientConfig: &tls.Config{
        // Match browser cipher suites
        CipherSuites: []uint16{
            tls.TLS_AES_128_GCM_SHA256,
            tls.TLS_AES_256_GCM_SHA384,
            tls.TLS_CHACHA20_POLY1305_SHA256,
        },
        // Use proper ALPN
        NextProtos: []string{"h2", "http/1.1"},
    },
}
```

3. **Use Specialized Libraries**
```go
// Using httpcloak for browser-identical TLS
import "github.com/ Humphrey httpproxy"

// Creates TLS connections that mimic Chrome fingerprints
```

**Sources:**
- [TLS Fingerprinting: How It Works & How to Bypass It](https://www.browserless.io/blog/tls-fingerprinting-explanation-detection-and-bypassing-it-in-playwright-and-puppeteer)
- [httpcloak tutorial: Bypass TLS Fingerprinting](https://roundproxies.com/blog/httpcloak/)
- [How TLS Fingerprint is Used to Block Web Scrapers](https://scrapfly.io/blog/posts/how-to-avoid-web-scraping-blocking-tls)
- [Overcoming TLS Fingerprinting in Web Scraping](https://rayobyte.com/blog/tls-fingerprinting/)

### 8. Browser Fingerprinting Countermeasures

Defend against JavaScript-based fingerprinting (Canvas, WebGL, AudioContext, etc.).

**Detection Methods:**
- Canvas fingerprinting
- WebGL parameters
- Font enumeration
- Screen resolution
- Audio context fingerprint

**Countermeasures:**

1. **Use Headful Browser Mode**
```go
// Run with visible window (slower but less detectable)
options := append(options, chromedp.Flag("headless", false))
```

2. **Use Anti-Detection Libraries**
```go
// Use undetected-chromedriver or similar
// These patch the browser navigator properties
```

3. **Stealth Configuration**
```go
stealthOptions := []chromedp.ExecAllocatorOption{
    // Hide webdriver flag
    chromedp.Flag("exclude-switches", "enable-automation"),
    chromedp.Flag("disable-blink-features", "AutomationControlled"),

    // Override navigator properties
    chromedp.Eval(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`, nil),

    // Add plugins
    chromedp.Eval(`Object.defineProperty(navigator, 'plugins', {get: () => [1, 2, 3, 4, 5]})`, nil),

    // Set language
    chromedp.Eval(`Object.defineProperty(navigator, 'languages', {get: () => ['en-US', 'en']})`, nil),
}
```

**Sources:**
- [How to Bypass CreepJS and Spoof Browser Fingerprinting](https://www.scrapingbee.com/blog/creepjs-browser-fingerprinting/)
- [Browser Fingerprinting Guide: Detection & Bypass Methods](https://www.browserless.io/blog/device-fingerprinting)
- [embeddinglayer/awesome-fingerprinting GitHub](https://github.com/embeddinglayer/awesome-fingerprinting)

### 9. CAPTCHA Handling

When CAPTCHAs are encountered, implement appropriate handling strategies.

**Types of CAPTCHAs:**
- reCAPTCHA v2/v3
- hCaptcha
- Cloudflare Turnstile
- Custom image CAPTCHAs

**Handling Strategies:**

1. **Avoidance (Best Strategy)**
   - Don't trigger CAPTCHAs in the first place
   - Use proper delays and session management
   - Rotate requests across IPs

2. **Third-Party Solving Services**
```go
// Use services like:
// - 2Captcha
// - Anti-Captcha
// - DeathByCaptcha

type CaptchaSolver interface {
    Solve(siteKey, siteURL string) (string, error)
}
```

3. **Manual Bypass**
```go
// Pause and notify user to solve CAPTCHA
// Useful for personal/research crawls
func ManualCaptchaBypass(url string) error {
    fmt.Printf("CAPTCHA detected at %s\n", url)
    fmt.Println("Please solve in browser and press Enter...")
    fmt.Scanln()
    return nil
}
```

**Sources:**
- [7 Ways to Bypass CAPTCHA While Web Scraping](https://www.zenrows.com/blog/bypass-captcha-web-scraping)
- [Google CAPTCHA Bypass Methods for Web Scraping](https://decodo.com/blog/how-to-bypass-google-captcha)

### 10. Behavioral Mimicry

Make your crawler behave more like a human user.

**Techniques:**

1. **Randomize Timing**
```go
// Add random jitter to delays
func randomDelay(base time.Duration) time.Duration {
    jitter := time.Duration(rand.Int63n(int64(base) / 2))
    return base + jitter - (base / 4)
}
```

2. **Scroll and Mouse Movement** (for browser automation)
```go
// Simulate human-like scrolling
chromedp.Evaluate(`window.scrollBy(0, 200)`, nil)
time.Sleep(randomDelay(500 * time.Millisecond))
```

3. **Request Patterns**
```go
// Don't request pages in perfect sequence
// Occasionally revisit pages
// Simulate navigation patterns
```

---

## Detection and Recovery

### Monitor for Block Indicators

```go
type BlockDetector struct {
    indicators []BlockIndicator
}

type BlockIndicator interface {
    IsBlocked(resp *http.Response, body []byte) bool
}

// Common block indicators
type HTTPStatusIndicator struct{}
func (i HTTPStatusIndicator) IsBlocked(resp *http.Response, body []byte) bool {
    return resp.StatusCode == 403 || resp.StatusCode == 429
}

type ContentLengthIndicator struct {
    minLength int
}
func (i ContentLengthIndicator) IsBlocked(resp *http.Response, body []byte) bool {
    return len(body) < i.minLength && resp.StatusCode == 200
}

type CAPTCHAIndicator struct{}
func (i CAPTCHAIndicator) IsBlocked(resp *http.Response, body []byte) bool {
    content := string(body)
    return strings.Contains(content, "captcha") ||
           strings.Contains(content, "challenge-platform") ||
           strings.Contains(content, "cf-challenge")
}
```

### Exponential Backoff Strategy

```go
type BackoffManager struct {
    maxBackoff    time.Duration
    initialDelay  time.Duration
    multiplier    float64
}

func (bm *BackoffManager) ShouldBackoff(domain string, attempt int) (time.Duration, bool) {
    if attempt > 5 {
        return 0, false // Give up
    }

    delay := time.Duration(float64(bm.initialDelay) * math.Pow(bm.multiplier, float64(attempt)))
    if delay > bm.maxBackoff {
        delay = bm.maxBackoff
    }

    return delay, true
}
```

---

## Implementation Priority for gosearch

### Phase 1: Essential (Implement Now)
1. ✅ **robots.txt compliance** - Already implemented
2. ✅ **Rate limiting** - Already implemented, add adaptive backoff
3. ✅ **User-Agent headers** - Add proper browser-like headers
4. ✅ **Politeness delays** - Already implemented

### Phase 2: Important (Add Soon)
1. **Enhanced header profiles** - Add Sec-CH-UA headers
2. **Session management** - Cookie persistence
3. **Block detection** - Detect 403/429/CAPTCHA responses
4. **Exponential backoff** - Adaptive delays after rate limits

### Phase 3: Advanced (Add When Needed)
1. **Proxy rotation** - For large-scale crawls
2. **TLS fingerprint mitigation** - If targeting protected sites
3. **CAPTCHA handling** - Manual bypass or integration
4. **Browser automation fallback** - For JavaScript-heavy sites

---

## Ethical Considerations

### Legal Boundaries

1. **robots.txt is not law** - But violating it may violate ToS
2. **Public data** - Generally legal to scrape public data
3. **Personal data** - GDPR/CCPA considerations
4. **ToS violations** - May result in account termination

### Ethical Guidelines

1. **Don't harm the target site**
   - Respect rate limits
   - Don't overload servers
   - Avoid peak hours when possible

2. **Identify your crawler**
   - Use descriptive User-Agent
   - Provide contact info
   - Respect removal requests

3. **Use data responsibly**
   - Don't republish without permission
   - Attribute sources
   - Respect copyright

**Sources:**
- [Web Scraping: Ethics, Legality, & Robots.txt](https://medium.com/@ridhopujiono.work/web-scraping-2-ethics-legality-robots-txt-how-to-stay-out-of-trouble-39052f7dc63f)
- [Best Practices - Web Scraping @ Pitt](https://pitt.libguides.com/webscraping/bestpractices)

---

## Code Implementation Template

```go
// internal/crawler/antiblocking.go
package crawler

import (
    "context"
    "crypto/tls"
    "math"
    "math/rand"
    "net/http"
    "sync"
    "time"
)

// AntiBlockingConfig configures anti-blocking strategies
type AntiBlockingConfig struct {
    DefaultDelay       time.Duration
    MaxBackoff         time.Duration
    BackoffMultiplier  float64
    RespectRobots      bool
    RotateUserAgent    bool
    UseProxies         bool
}

// AntiBlockingManager implements anti-blocking strategies
type AntiBlockingManager struct {
    config        *AntiBlockingConfig
    rateLimiter   *RateLimiter
    uaRotator     *UserAgentRotator
    backoffMgr    *BackoffManager
    blockDetector *BlockDetector
    sessionMgr    *SessionManager
}

func NewAntiBlockingManager(config *AntiBlockingConfig) *AntiBlockingManager {
    return &AntiBlockingManager{
        config:        config,
        rateLimiter:   NewRateLimiter(config.DefaultDelay),
        uaRotator:     NewUserAgentRotator(),
        backoffMgr:    &BackoffManager{
            initialDelay: config.DefaultDelay,
            maxBackoff:   config.MaxBackoff,
            multiplier:   config.BackoffMultiplier,
        },
        blockDetector: NewBlockDetector(),
        sessionMgr:    NewSessionManager(),
    }
}

// MakeRequest makes a request with anti-blocking measures
func (abm *AntiBlockingManager) MakeRequest(ctx context.Context, url string) (*http.Response, error) {
    var lastErr error
    var attempt int

    for {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }

        // Get domain from URL
        domain := extractDomain(url)

        // Rate limiting
        if err := abm.rateLimiter.Acquire(domain); err != nil {
            return nil, err
        }

        // Create request with proper headers
        req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
        if err != nil {
            return nil, err
        }

        // Set headers
        if abm.config.RotateUserAgent {
            req.Header.Set("User-Agent", abm.uaRotator.Next())
        }
        setBrowserHeaders(req.Header)

        // Add cookies from session
        for _, cookie := range abm.sessionMgr.GetCookies(url) {
            req.AddCookie(cookie)
        }

        // Make request
        resp, err := abm.client.Do(req)
        if err != nil {
            lastErr = err
            attempt++
            if delay, cont := abm.backoffMgr.ShouldBackoff(domain, attempt); cont {
                time.Sleep(delay)
                continue
            }
            return nil, lastErr
        }

        // Check if blocked
        body, _ := io.ReadAll(resp.Body)
        resp.Body.Close()

        if abm.blockDetector.IsBlocked(resp, body) {
            resp.Body.Close()
            lastErr = fmt.Errorf("blocked: %s", resp.Status)

            // Store cookies for retry
            abm.sessionMgr.SetCookies(url, resp.Cookies())

            attempt++
            if delay, cont := abm.backoffMgr.ShouldBackoff(domain, attempt); cont {
                time.Sleep(delay)
                continue
            }
            return nil, lastErr
        }

        // Success - restore response body
        resp.Body = io.NopCloser(bytes.NewReader(body))

        // Store cookies
        abm.sessionMgr.SetCookies(url, resp.Cookies())

        return resp, nil
    }
}

func setBrowserHeaders(h http.Header) {
    h.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
    h.Set("Accept-Language", "en-US,en;q=0.9")
    h.Set("Accept-Encoding", "gzip, deflate, br")
    h.Set("Connection", "keep-alive")
    h.Set("DNT", "1")
    h.Set("Upgrade-Insecure-Requests", "1")
    h.Set("Sec-Fetch-Dest", "document")
    h.Set("Sec-Fetch-Mode", "navigate")
    h.Set("Sec-Fetch-Site", "none")
    h.Set("Sec-Fetch-User", "?1")
    h.Set("Sec-CH-UA", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
    h.Set("Sec-CH-UA-Mobile", "?0")
    h.Set("Sec-CH-UA-Platform", `"Windows"`)
}
```

---

## References

1. [Rate-limiting | Apify Academy](https://docs.apify.com/academy/anti-scraping/techniques/rate-limiting)
2. [5 Tools to Scrape Without Blocking - Scrapfly](https://scrapfly.io/blog/posts/how-to-scrape-without-getting-blocked-tutorial)
3. [How To Crawl A Website Without Getting Blocked - Infatica](https://infatica.io/blog/how-to-crawl-a-website-without-getting-blocked/)
4. [15 Methods to Not Get Blocked Web Scraping - Roundproxies](https://roundproxies.com/blog/web-scraping-without-getting-blocked)
5. [14 Ways for Web Scraping Without Getting Blocked - ZenRows](https://www.zenrows.com/blog/web-scraping-without-getting-blocked)
6. [Web scraping: how to crawl without getting blocked - DEV Community](https://dev.to/apify/web-scraping-how-to-crawl-without-getting-blocked-587b)
7. [TLS Fingerprinting: How It Works & How to Bypass It - Browserless](https://www.browserless.io/blog/tls-fingerprinting-explanation-detection-and-bypassing-it-in-playwright-and-puppeteer)
8. [httpcloak tutorial: Bypass TLS Fingerprinting](https://roundproxies.com/blog/httpcloak/)
9. [How TLS Fingerprint is Used to Block Web Scrapers - Scrapfly](https://scrapfly.io/blog/posts/how-to-avoid-web-scraping-blocking-tls)
10. [Stop Getting Blocked: 10 Common Web-Scraping Mistakes](https://www.firecrawl.dev/blog/web-scraping-mistakes-and-fixes)

---

*Document created: 2026-02-11*
*Last updated: 2026-02-11*

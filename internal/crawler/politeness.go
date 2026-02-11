package crawler

import (
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

// PolitenessManager manages rate limiting and robots.txt compliance.
type PolitenessManager struct {
	delay         time.Duration
	userAgent     string
	limiter       map[string]*limiter
	limiterMu     sync.RWMutex
	robots        map[string]*robotstxt.RobotsData
	robotsMu      sync.RWMutex
	httpClient    *http.Client
	respectRobots bool
}

// limiter implements a rate limiter for a specific domain.
type limiter struct {
	semaphore chan struct{}
	lastVisit time.Time
}

// NewPolitenessManager creates a new politeness manager.
func NewPolitenessManager(delay time.Duration, userAgent string, respectRobots bool) *PolitenessManager {
	return &PolitenessManager{
		delay:     delay,
		userAgent: userAgent,
		limiter:   make(map[string]*limiter),
		robots:    make(map[string]*robotstxt.RobotsData),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		respectRobots: respectRobots,
	}
}

// Acquire waits for permission to crawl a URL.
// It returns an error if the URL is disallowed by robots.txt.
func (pm *PolitenessManager) Acquire(targetURL string) error {
	if !pm.respectRobots {
		// Just apply rate limiting
		return pm.acquireLimiter(targetURL)
	}

	// Check robots.txt
	allowed, err := pm.isAllowed(targetURL)
	if err != nil {
		// On error, allow but log
		return pm.acquireLimiter(targetURL)
	}

	if !allowed {
		return ErrDisallowed
	}

	return pm.acquireLimiter(targetURL)
}

// Release releases permission after crawling is complete.
func (pm *PolitenessManager) Release(targetURL string) {
	pm.releaseLimiter(targetURL)
}

// acquireLimiter acquires permission from the domain's rate limiter.
func (pm *PolitenessManager) acquireLimiter(targetURL string) error {
	domain := extractDomain(targetURL)

	pm.limiterMu.Lock()
	l, exists := pm.limiter[domain]
	if !exists {
		l = &limiter{
			semaphore: make(chan struct{}, 1), // One request at a time per domain
		}
		pm.limiter[domain] = l
	}
	pm.limiterMu.Unlock()

	// Acquire semaphore (blocks if already in use)
	l.semaphore <- struct{}{}

	// Apply delay between requests
	now := time.Now()
	if !l.lastVisit.IsZero() {
		elapsed := now.Sub(l.lastVisit)
		if elapsed < pm.delay {
			time.Sleep(pm.delay - elapsed)
		}
	}
	l.lastVisit = time.Now()

	return nil
}

// releaseLimiter releases the domain's rate limiter.
func (pm *PolitenessManager) releaseLimiter(targetURL string) {
	domain := extractDomain(targetURL)

	pm.limiterMu.RLock()
	l, exists := pm.limiter[domain]
	pm.limiterMu.RUnlock()

	if exists {
		<-l.semaphore
	}
}

// isAllowed checks if the URL is allowed by robots.txt.
func (pm *PolitenessManager) isAllowed(targetURL string) (bool, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return false, err
	}

	// Get or fetch robots.txt for this domain
	robots, err := pm.getRobots(parsedURL.Scheme + "://" + parsedURL.Host)
	if err != nil {
		return true, err
	}

	group := robots.FindGroup(pm.userAgent)
	if group == nil {
		return true, nil
	}

	return group.Test(parsedURL.Path), nil
}

// getRobots retrieves or fetches robots.txt for a domain.
func (pm *PolitenessManager) getRobots(baseURL string) (*robotstxt.RobotsData, error) {
	domain := extractDomain(baseURL)

	pm.robotsMu.RLock()
	robots, exists := pm.robots[domain]
	pm.robotsMu.RUnlock()

	if exists {
		return robots, nil
	}

	// Fetch robots.txt
	resp, err := pm.httpClient.Get(baseURL + "/robots.txt")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse robots.txt
	robots, err = robotstxt.FromResponse(resp)
	if err != nil {
		return nil, err
	}

	// Cache
	pm.robotsMu.Lock()
	pm.robots[domain] = robots
	pm.robotsMu.Unlock()

	return robots, nil
}

// extractDomain extracts the domain from a URL.
func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return parsed.Host
}

// Errors
var (
	// ErrDisallowed is returned when a URL is disallowed by robots.txt.
	ErrDisallowed = errors.New("URL disallowed by robots.txt")
)

// SetDelay updates the delay between requests.
func (pm *PolitenessManager) SetDelay(delay time.Duration) {
	pm.delay = delay
}

// GetDelay returns the current delay between requests.
func (pm *PolitenessManager) GetDelay() time.Duration {
	return pm.delay
}

// Clear clears all rate limiters and robots.txt cache.
func (pm *PolitenessManager) Clear() {
	pm.limiterMu.Lock()
	pm.limiter = make(map[string]*limiter)
	pm.limiterMu.Unlock()

	pm.robotsMu.Lock()
	pm.robots = make(map[string]*robotstxt.RobotsData)
	pm.robotsMu.Unlock()
}

// Stats returns statistics about the politeness manager.
func (pm *PolitenessManager) Stats() map[string]interface{} {
	pm.limiterMu.RLock()
	domains := len(pm.limiter)
	pm.limiterMu.RUnlock()

	pm.robotsMu.RLock()
	robotsCount := len(pm.robots)
	pm.robotsMu.RUnlock()

	return map[string]interface{}{
		"tracked_domains": domains,
		"cached_robots":   robotsCount,
		"delay_ms":        pm.delay.Milliseconds(),
	}
}

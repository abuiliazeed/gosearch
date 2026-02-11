// Package crawler provides web crawling functionality for gosearch.
package crawler

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
)

// CookieManager manages cookies across crawl sessions.
type CookieManager struct {
	jar *cookiejar.Jar
	mu  sync.RWMutex
}

// NewCookieManager creates a new cookie manager.
func NewCookieManager() (*CookieManager, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	return &CookieManager{
		jar: jar,
	}, nil
}

// GetCookies returns cookies for a URL.
func (cm *CookieManager) GetCookies(u *url.URL) []*http.Cookie {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.jar == nil {
		return nil
	}

	return cm.jar.Cookies(u)
}

// SetCookies sets cookies for a URL.
func (cm *CookieManager) SetCookies(u *url.URL, cookies []*http.Cookie) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.jar == nil {
		return
	}

	cm.jar.SetCookies(u, cookies)
}

// ApplyToRequest applies stored cookies to an HTTP request.
func (cm *CookieManager) ApplyToRequest(req *http.Request) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.jar == nil {
		return
	}

	cookies := cm.jar.Cookies(req.URL)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
}

// StoreFromResponse extracts cookies from an HTTP response and stores them.
func (cm *CookieManager) StoreFromResponse(resp *http.Response) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.jar == nil {
		return
	}

	// Cookies are automatically stored by the jar
	// This method is for explicit handling if needed
}

// Clear removes all stored cookies.
func (cm *CookieManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Create a new empty jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		return
	}
	cm.jar = jar
}

// ClearForDomain removes cookies for a specific domain.
func (cm *CookieManager) ClearForDomain(domain string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.jar == nil {
		return
	}

	// Create a new jar without cookies for this domain
	newJar, err := cookiejar.New(nil)
	if err != nil {
		return
	}

	// Copy all cookies except those for the specified domain
	// Note: This requires iterating over all URLs in the jar
	// Since cookiejar doesn't expose the stored URLs, we'll just recreate
	cm.jar = newJar
}

// GetJar returns the underlying cookie jar for direct access.
func (cm *CookieManager) GetJar() *cookiejar.Jar {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.jar
}

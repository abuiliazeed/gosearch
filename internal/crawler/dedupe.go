package crawler

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sync"

	"github.com/bits-and-blooms/bloom/v3"
)

// Deduplicator handles URL deduplication to prevent re-crawling.
type Deduplicator struct {
	seen     map[string]bool
	mu       sync.RWMutex
	bloom    *bloom.BloomFilter
	useBloom bool
}

// NewDeduplicator creates a new URL deduplicator.
// If useBloom is true, uses a Bloom filter for memory-efficient deduplication.
func NewDeduplicator(useBloom bool, estimatedSize uint) *Deduplicator {
	d := &Deduplicator{
		seen:     make(map[string]bool),
		useBloom: useBloom,
	}

	if useBloom {
		// Create bloom filter with 0.1% false positive rate
		d.bloom = bloom.NewWithEstimates(estimatedSize, 0.001)
	}

	return d
}

// Seen checks if a URL has been seen before.
func (d *Deduplicator) Seen(url string) bool {
	if d.useBloom && d.bloom != nil {
		return d.bloom.Test([]byte(url))
	}

	d.mu.RLock()
	_, exists := d.seen[url]
	d.mu.RUnlock()
	return exists
}

// Add marks a URL as seen.
func (d *Deduplicator) Add(url string) {
	if d.useBloom && d.bloom != nil {
		d.bloom.Add([]byte(url))
	}

	d.mu.Lock()
	d.seen[url] = true
	d.mu.Unlock()
}

// Clear clears all seen URLs.
func (d *Deduplicator) Clear() {
	d.mu.Lock()
	d.seen = make(map[string]bool)
	d.mu.Unlock()

	if d.useBloom && d.bloom != nil {
		d.bloom.ClearAll()
	}
}

// Count returns the number of unique URLs seen.
func (d *Deduplicator) Count() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.seen)
}

// URLHash generates a hash for a URL.
func URLHash(url string) string {
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:])
}

// NormalizeURL normalizes a URL for deduplication.
// This includes removing fragments, converting to lowercase, etc.
func NormalizeURL(rawURL string) string {
	// Parse URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// Remove fragment
	u.Fragment = ""

	// Convert to lowercase
	u.Host = toLower(u.Host)

	// Remove trailing slash from path unless it's root
	if u.Path != "/" && len(u.Path) > 0 && u.Path[len(u.Path)-1] == '/' {
		u.Path = u.Path[:len(u.Path)-1]
	}

	// Remove default ports
	if (u.Scheme == "http" && u.Port() == "80") ||
		(u.Scheme == "https" && u.Port() == "443") {
		u.Host = u.Hostname()
	}

	return u.String()
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

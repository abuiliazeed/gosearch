// Package search provides query processing and search functionality for gosearch.
//
// It includes query parsing, boolean operations, fuzzy matching,
// phrase queries, and result ranking with caching.
package search

import (
	"sync"

	"github.com/microcosm-cc/bluemonday"
)

// Sanitizer provides HTML sanitization to prevent XSS attacks.
// It uses bluemonday to strip potentially dangerous HTML/JavaScript
// from user-generated content like titles, snippets, and URLs.
type Sanitizer struct {
	policy *bluemonday.Policy
	once   sync.Once
}

// NewSanitizer creates a new sanitizer with a safe default policy.
// The policy allows basic formatting but removes:
// - <script> tags and event handlers (onclick, onerror, etc.)
// - <iframe>, <object>, <embed> tags
// - style attributes and tags
// - Dangerous HTML5 elements
func NewSanitizer() *Sanitizer {
	return &Sanitizer{}
}

// getPolicy lazily initializes and returns the bluemonday policy.
func (s *Sanitizer) getPolicy() *bluemonday.Policy {
	s.once.Do(func() {
		// Create a policy that strips all HTML - safest option for search results
		// This prevents any XSS while preserving the text content
		s.policy = bluemonday.StrictPolicy()
	})
	return s.policy
}

// Sanitize cleans potentially dangerous HTML/JavaScript from a string.
// It removes all HTML tags, leaving only plain text content.
// This is safe for titles, snippets, and other user-generated content.
func (s *Sanitizer) Sanitize(input string) string {
	if input == "" {
		return input
	}
	return s.getPolicy().Sanitize(input)
}

// SanitizeSlice sanitizes a slice of strings, returning a new slice.
func (s *Sanitizer) SanitizeSlice(input []string) []string {
	if len(input) == 0 {
		return input
	}
	result := make([]string, len(input))
	for i, str := range input {
		result[i] = s.Sanitize(str)
	}
	return result
}

// DefaultSanitizer returns the shared default sanitizer instance.
func DefaultSanitizer() *Sanitizer {
	return defaultSanitizer
}

var defaultSanitizer = NewSanitizer()

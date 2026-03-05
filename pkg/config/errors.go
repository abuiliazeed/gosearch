// Package config provides configuration management and error definitions for gosearch.
//
// It includes error types with helpful suggestions for common issues,
// and configuration loading from files and environment variables.
package config

import (
	"errors"
	"fmt"
)

// Base error types
var (
	// ErrConfigNotFound is returned when the configuration file is not found.
	ErrConfigNotFound = errors.New("configuration file not found")

	// ErrInvalidConfig is returned when the configuration is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrStorageUnavailable is returned when storage is unavailable.
	ErrStorageUnavailable = errors.New("storage unavailable")

	// ErrIndexEmpty is returned when the index is empty.
	ErrIndexEmpty = errors.New("index is empty")

	// ErrDocumentNotFound is returned when a document is not found.
	ErrDocumentNotFound = errors.New("document not found")

	// ErrCrawlerStopped is returned when the crawler is stopped.
	ErrCrawlerStopped = errors.New("crawler stopped")

	// ErrRateLimited is returned when rate limit is hit.
	ErrRateLimited = errors.New("rate limited")

	// ErrBlocked is returned when blocked by a website.
	ErrBlocked = errors.New("blocked by website")
)

// DetailedError wraps an error with additional context and suggestions.
type DetailedError struct {
	Err         error
	Suggestion  string
	Details     string
	Recoverable bool
}

// Error returns the error message.
func (e *DetailedError) Error() string {
	msg := e.Err.Error()
	if e.Details != "" {
		msg += ": " + e.Details
	}
	if e.Suggestion != "" {
		msg += "\nSuggestion: " + e.Suggestion
	}
	return msg
}

// Unwrap returns the underlying error.
func (e *DetailedError) Unwrap() error {
	return e.Err
}

// NewDetailedError creates a new detailed error.
func NewDetailedError(err error, suggestion, details string, recoverable bool) *DetailedError {
	return &DetailedError{
		Err:         err,
		Suggestion:  suggestion,
		Details:     details,
		Recoverable: recoverable,
	}
}

// Common error constructors with suggestions

// NewConfigError creates a configuration error with helpful suggestions.
func NewConfigError(err error) *DetailedError {
	return &DetailedError{
		Err:         err,
		Suggestion:  "Check that the configuration file exists and is valid YAML format. Run 'gosearch --help' for more information.",
		Details:     fmt.Sprintf("Config error: %v", err),
		Recoverable: true,
	}
}

// NewStorageError creates a storage error with helpful suggestions.
func NewStorageError(err error) *DetailedError {
	return &DetailedError{
		Err:         err,
		Suggestion:  "Check that the data directory exists and is writable. Ensure you have the necessary permissions.",
		Details:     fmt.Sprintf("Storage error: %v", err),
		Recoverable: false,
	}
}

// NewIndexError creates an index error with helpful suggestions.
func NewIndexError(err error) *DetailedError {
	return &DetailedError{
		Err:         err,
		Suggestion:  "Try running 'gosearch index rebuild' to rebuild the index from crawled documents.",
		Details:     fmt.Sprintf("Index error: %v", err),
		Recoverable: true,
	}
}

// NewCrawlerError creates a crawler error with helpful suggestions.
func NewCrawlerError(err error) *DetailedError {
	return &DetailedError{
		Err:         err,
		Suggestion:  "Check your internet connection and ensure the URLs are accessible. Try reducing the number of workers.",
		Details:     fmt.Sprintf("Crawler error: %v", err),
		Recoverable: true,
	}
}

// NewRedisError creates a Redis error with helpful suggestions.
func NewRedisError(err error) *DetailedError {
	return &DetailedError{
		Err:         err,
		Suggestion:  "Ensure Redis is running. Start Redis with 'redis-server' or use '--no-cache' flag to disable caching.",
		Details:     fmt.Sprintf("Redis error: %v", err),
		Recoverable: true,
	}
}

// NewRateLimitError creates a rate limit error with helpful suggestions.
func NewRateLimitError(domain string) *DetailedError {
	return &DetailedError{
		Err:         ErrRateLimited,
		Suggestion:  fmt.Sprintf("Reduce crawl delay for %s using '--delay' flag or wait before retrying.", domain),
		Details:     fmt.Sprintf("Rate limited by domain: %s", domain),
		Recoverable: true,
	}
}

// NewBlockedError creates a blocked error with helpful suggestions.
func NewBlockedError(domain string, statusCode int) *DetailedError {
	var suggestion string
	switch statusCode {
	case 403:
		suggestion = fmt.Sprintf("Your IP may be blocked by %s. Try using a different User-Agent or wait before retrying.", domain)
	case 429:
		suggestion = fmt.Sprintf("Too many requests to %s. Increase the delay between requests using '--delay' flag.", domain)
	default:
		suggestion = fmt.Sprintf("Access to %s was blocked. Check if the site allows crawling.", domain)
	}

	return &DetailedError{
		Err:         ErrBlocked,
		Suggestion:  suggestion,
		Details:     fmt.Sprintf("Blocked by %s with status code: %d", domain, statusCode),
		Recoverable: true,
	}
}

// IsRecoverable checks if an error is recoverable.
func IsRecoverable(err error) bool {
	var detailedErr *DetailedError
	if errors.As(err, &detailedErr) {
		return detailedErr.Recoverable
	}
	return false
}

// GetSuggestion extracts the suggestion from an error if available.
func GetSuggestion(err error) string {
	var detailedErr *DetailedError
	if errors.As(err, &detailedErr) {
		return detailedErr.Suggestion
	}
	return ""
}

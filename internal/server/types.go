// Package server provides HTTP API server functionality for gosearch.
//
// It includes a RESTful API for search, index management, and statistics.
package server

import (
	"time"
)

// Config holds the server configuration.
type Config struct {
	Host               string        // Host to bind to
	Port               int           // Port to listen on
	ReadTimeout        time.Duration // Read timeout
	WriteTimeout       time.Duration // Write timeout
	IdleTimeout        time.Duration // Idle timeout
	CORSAllowedOrigins string        // CORS allowed origins
	CORSAllowedMethods string        // CORS allowed methods
	CORSAllowedHeaders string        // CORS allowed headers
}

// DefaultConfig returns the default server configuration.
func DefaultConfig() *Config {
	return &Config{
		Host:         "127.0.0.1",
		Port:         8080,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

// SearchRequest represents a search request.
type SearchRequest struct {
	Query   string `json:"query"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
	Fuzzy   bool   `json:"fuzzy,omitempty"`
	Explain bool   `json:"explain,omitempty"`
}

// SearchResponse represents a search response.
type SearchResponse struct {
	TotalCount int            `json:"total_count"`
	Results    []SearchResult `json:"results"`
	Query      string         `json:"query"`
	DurationMs int64          `json:"duration_ms"`
}

// SearchResult represents a single search result.
type SearchResult struct {
	DocID   string  `json:"doc_id"`
	URL     string  `json:"url"`
	Title   string  `json:"title"`
	Score   float64 `json:"score"`
	Snippet string  `json:"snippet,omitempty"`
}

// StatsResponse represents index statistics response.
type StatsResponse struct {
	TotalDocuments int     `json:"total_documents"`
	TotalTerms     int     `json:"total_terms"`
	TotalPostings  int64   `json:"total_postings"`
	AvgPostings    float64 `json:"average_postings"`
	LastUpdated    string  `json:"last_updated"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
	Version   string    `json:"version"`
}

// IndexRebuildRequest represents an index rebuild request.
type IndexRebuildRequest struct {
	Force bool `json:"force,omitempty"`
}

// IndexRebuildResponse represents an index rebuild response.
type IndexRebuildResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	DocsIndexed  int    `json:"docs_indexed"`
	TermsIndexed int    `json:"terms_indexed"`
	DurationMs   int64  `json:"duration_ms"`
}

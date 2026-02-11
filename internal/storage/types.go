// Package storage provides data persistence layers for gosearch.
//
// It includes file-based document storage, BoltDB index metadata storage,
// and Redis cache for query results.
package storage

import (
	"time"
)

// Document represents a crawled web page.
type Document struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	HTML      string    `json:"html"`
	Links     []string  `json:"links"`
	CrawledAt time.Time `json:"crawled_at"`
	Depth     int       `json:"depth"`
}

// IndexMeta represents metadata about the inverted index.
type IndexMeta struct {
	TotalDocuments  int       `json:"total_documents"`
	TotalTerms      int       `json:"total_terms"`
	LastUpdated     time.Time `json:"last_updated"`
	IndexSize       int64     `json:"index_size"`
	TotalPostings   int64     `json:"total_postings"`
	AveragePostings float64   `json:"average_postings"`
}

// TermInfo represents information about a term in the index.
type TermInfo struct {
	Term             string    `json:"term"`
	DocFrequency     int       `json:"doc_frequency"`
	TotalOccurrences int       `json:"total_occurrences"`
	LastUpdated      time.Time `json:"last_updated"`
}

// CacheEntry represents a cached search result.
type CacheEntry struct {
	Query     string      `json:"query"`
	Results   interface{} `json:"results"`
	ExpiresAt time.Time   `json:"expires_at"`
	CreatedAt time.Time   `json:"created_at"`
}

// PersistedPosting represents a single document occurrence for a term in storage.
// This is the serialized form of indexer.Posting for BoltDB storage.
type PersistedPosting struct {
	DocID         string `json:"doc_id"`
	Positions     []int  `json:"positions"`
	TermFrequency int    `json:"term_frequency"`
}

// PersistedPostingsList represents the list of documents containing a term in storage.
// This is the serialized form of indexer.PostingsList for BoltDB storage.
type PersistedPostingsList struct {
	Term         string             `json:"term"`
	DocFrequency int                `json:"doc_frequency"`
	Postings     []PersistedPosting `json:"postings"`
}

// PersistedDocInfo represents document metadata in storage.
// This is the serialized form of indexer.DocInfo for BoltDB storage.
type PersistedDocInfo struct {
	DocID      string    `json:"doc_id"`
	URL        string    `json:"url"`
	Title      string    `json:"title"`
	TokenCount int       `json:"token_count"`
	Length     int       `json:"length"`
	IndexedAt  time.Time `json:"indexed_at"`
}

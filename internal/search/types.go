// Package search provides query processing and search functionality for gosearch.
//
// It includes query parsing, boolean operations, fuzzy matching,
// phrase queries, and result ranking with caching.
package search

import (
	"errors"
	"time"
)

var (
	// ErrInvalidQuery is returned when a query is invalid.
	ErrInvalidQuery = errors.New("invalid query")

	// ErrSearchFailed is returned when a search operation fails.
	ErrSearchFailed = errors.New("search failed")

	// ErrCacheDisabled is returned when cache is not available.
	ErrCacheDisabled = errors.New("cache is disabled")
)

// SearchConfig holds configuration for the searcher.
type SearchConfig struct {
	// Enable caching of query results
	CacheEnabled bool

	// Cache TTL for query results
	CacheTTL time.Duration

	// Maximum number of results to return
	MaxResults int

	// Enable fuzzy matching
	FuzzyEnabled bool

	// Maximum edit distance for fuzzy matching
	FuzzyDistance int

	// Enable phrase queries
	PhraseEnabled bool

	// Enable boolean operators
	BooleanEnabled bool
}

// DefaultSearchConfig returns the default search configuration.
func DefaultSearchConfig() *SearchConfig {
	return &SearchConfig{
		CacheEnabled:   true,
		CacheTTL:       5 * time.Minute,
		MaxResults:     100,
		FuzzyEnabled:   true,
		FuzzyDistance:  2,
		PhraseEnabled:  true,
		BooleanEnabled: true,
	}
}

// SearchResult represents a single search result.
type SearchResult struct {
	DocID    string
	URL      string
	Title    string
	Score    float64
	Snippet  string
	Position int
}

// SearchResponse represents the response from a search query.
type SearchResponse struct {
	Results    []*SearchResult
	TotalCount int
	Query      string
	Duration   time.Duration
	Cached     bool
}

// QueryType represents the type of query.
type QueryType int

const (
	// QueryTypeTerm is a simple term query.
	QueryTypeTerm QueryType = iota

	// QueryTypePhrase is a phrase query ("exact match").
	QueryTypePhrase

	// QueryTypeBoolean is a boolean query (AND, OR, NOT).
	QueryTypeBoolean

	// QueryTypeFuzzy is a fuzzy query (approximate match).
	QueryTypeFuzzy
)

// String returns the string representation of the query type.
func (qt QueryType) String() string {
	switch qt {
	case QueryTypeTerm:
		return "term"
	case QueryTypePhrase:
		return "phrase"
	case QueryTypeBoolean:
		return "boolean"
	case QueryTypeFuzzy:
		return "fuzzy"
	default:
		return "unknown"
	}
}

// ParsedQuery represents a parsed search query.
type ParsedQuery struct {
	Type      QueryType
	Terms     []string
	Phrase    string
	Boolean   *BooleanQuery
	FuzzyTerm string
	Original  string
}

// BooleanQuery represents a boolean search query.
type BooleanQuery struct {
	AndTerms []string // Terms that must all match
	OrTerms  []string // Terms where any match is acceptable
	NotTerms []string // Terms that must not match
}

// NewBooleanQuery creates a new boolean query.
func NewBooleanQuery() *BooleanQuery {
	return &BooleanQuery{
		AndTerms: make([]string, 0),
		OrTerms:  make([]string, 0),
		NotTerms: make([]string, 0),
	}
}

// AddAnd adds a term to the AND list.
func (bq *BooleanQuery) AddAnd(term string) {
	bq.AndTerms = append(bq.AndTerms, term)
}

// AddOr adds a term to the OR list.
func (bq *BooleanQuery) AddOr(term string) {
	bq.OrTerms = append(bq.OrTerms, term)
}

// AddNot adds a term to the NOT list.
func (bq *BooleanQuery) AddNot(term string) {
	bq.NotTerms = append(bq.NotTerms, term)
}

// IsEmpty returns true if the boolean query has no terms.
func (bq *BooleanQuery) IsEmpty() bool {
	return len(bq.AndTerms) == 0 && len(bq.OrTerms) == 0 && len(bq.NotTerms) == 0
}

// Clone creates a deep copy of the boolean query.
func (bq *BooleanQuery) Clone() *BooleanQuery {
	return &BooleanQuery{
		AndTerms: append([]string{}, bq.AndTerms...),
		OrTerms:  append([]string{}, bq.OrTerms...),
		NotTerms: append([]string{}, bq.NotTerms...),
	}
}

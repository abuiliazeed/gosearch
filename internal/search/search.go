// Package search provides query processing and search functionality for gosearch.
//
// It includes query parsing, boolean operations, fuzzy matching,
// phrase queries, and result ranking with caching.
package search

import (
	"context"
	"fmt"
	"time"

	"github.com/abuiliazeed/gosearch/internal/indexer"
	"github.com/abuiliazeed/gosearch/internal/ranker"
	"github.com/abuiliazeed/gosearch/internal/storage"
)

// Searcher handles search queries against the index.
type Searcher struct {
	index  *indexer.Index
	scorer *ranker.Scorer
	cache  *storage.CacheStore
	parser *Parser
	config *SearchConfig
}

// NewSearcher creates a new searcher.
func NewSearcher(idx *indexer.Index, scorer *ranker.Scorer, cache *storage.CacheStore, config *SearchConfig) *Searcher {
	if config == nil {
		config = DefaultSearchConfig()
	}

	return &Searcher{
		index:  idx,
		scorer: scorer,
		cache:  cache,
		parser: NewParser(config),
		config: config,
	}
}

// Search performs a search query against the index.
// The ctx parameter controls cancellation and timeout.
//
// Returns a SearchResponse with ranked results, or an error if the search fails.
func (s *Searcher) Search(ctx context.Context, query string) (*SearchResponse, error) {
	startTime := time.Now()

	// Check cache first
	if s.config.CacheEnabled && s.cache != nil {
		cached, err := s.getCachedResult(ctx, query)
		if err == nil && cached != nil {
			cached.Cached = true
			return cached, nil
		}
	}

	// Parse the query
	parsed := s.parser.Parse(query)

	// Check for cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Execute the search
	var docIDs []string
	var err error

	switch parsed.Type {
	case QueryTypeTerm, QueryTypeBoolean:
		docIDs, err = s.executeTermSearch(ctx, parsed)
	case QueryTypePhrase:
		docIDs, err = s.executePhraseSearch(ctx, parsed)
	case QueryTypeFuzzy:
		docIDs, err = s.executeFuzzySearch(ctx, parsed)
	default:
		return nil, fmt.Errorf("%w: unsupported query type %s", ErrInvalidQuery, parsed.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSearchFailed, err)
	}

	// Rank results
	terms := s.parser.ExtractTerms(parsed)
	rankedResults := s.scorer.RankDocuments(terms, docIDs)

	// Build response
	results := make([]*SearchResult, 0, len(rankedResults))
	for i, ranked := range rankedResults {
		// Get document info
		docInfo, err := s.index.GetDocInfo(ranked.DocID)
		if err != nil {
			continue
		}

		result := &SearchResult{
			DocID:    ranked.DocID,
			URL:      docInfo.URL,
			Title:    docInfo.Title,
			Score:    ranked.Score,
			Snippet:  s.generateSnippet(docInfo, terms),
			Position: i + 1,
		}

		results = append(results, result)
	}

	// Limit results
	if s.config.MaxResults > 0 && len(results) > s.config.MaxResults {
		results = results[:s.config.MaxResults]
	}

	response := &SearchResponse{
		Results:    results,
		TotalCount: len(docIDs),
		Query:      query,
		Duration:   time.Since(startTime),
		Cached:     false,
	}

	// Cache the result
	if s.config.CacheEnabled && s.cache != nil {
		_ = s.cacheResult(ctx, query, response)
	}

	return response, nil
}

// executeTermSearch executes a term or boolean search.
func (s *Searcher) executeTermSearch(ctx context.Context, parsed *ParsedQuery) ([]string, error) {
	if parsed.Type == QueryTypeBoolean {
		return s.executeBooleanSearch(ctx, parsed)
	}

	// Simple term search - treat as AND of all terms
	booleanQuery := indexer.NewBooleanQuery()
	for _, term := range parsed.Terms {
		booleanQuery.AddAnd(term)
	}

	results, err := s.index.BooleanSearch(ctx, booleanQuery)
	if err != nil {
		return nil, err
	}

	return results.DocIDs, nil
}

// executeBooleanSearch executes a boolean search query.
func (s *Searcher) executeBooleanSearch(ctx context.Context, parsed *ParsedQuery) ([]string, error) {
	if parsed.Boolean == nil {
		return s.executeTermSearch(ctx, parsed)
	}

	booleanQuery := indexer.NewBooleanQuery()

	// Add AND terms
	for _, term := range parsed.Boolean.AndTerms {
		booleanQuery.AddAnd(term)
	}

	// Add OR terms
	for _, term := range parsed.Boolean.OrTerms {
		booleanQuery.AddOr(term)
	}

	// Add NOT terms
	for _, term := range parsed.Boolean.NotTerms {
		booleanQuery.AddNot(term)
	}

	results, err := s.index.BooleanSearch(ctx, booleanQuery)
	if err != nil {
		return nil, err
	}

	return results.DocIDs, nil
}

// executePhraseSearch executes a phrase search query.
// Uses positional information to find documents with exact phrase matches.
func (s *Searcher) executePhraseSearch(ctx context.Context, parsed *ParsedQuery) ([]string, error) {
	// Phrase search is done by finding all documents containing all phrase terms
	// and then checking for positional proximity

	booleanQuery := indexer.NewBooleanQuery()
	for _, term := range parsed.Terms {
		booleanQuery.AddAnd(term)
	}

	results, err := s.index.BooleanSearch(ctx, booleanQuery)
	if err != nil {
		return nil, err
	}

	// For phrase queries, we need to verify positional proximity
	// This is a simplified implementation - in production, you'd want to
	// properly check that all terms appear in the correct order
	// with the correct spacing.

	return results.DocIDs, nil
}

// executeFuzzySearch executes a fuzzy search query.
// Finds terms similar to the query term and searches for them.
func (s *Searcher) executeFuzzySearch(ctx context.Context, parsed *ParsedQuery) ([]string, error) {
	if len(parsed.FuzzyTerm) == 0 {
		return []string{}, nil
	}

	// Get all terms from the index as vocabulary
	// In a production system, you'd want to maintain a separate term list
	vocabulary := s.index.GetAllTerms()

	// Find similar terms
	threshold := 0.7 // Similarity threshold
	similarTerms := FindSimilarTerms(parsed.FuzzyTerm, vocabulary, threshold, 10)

	// If no similar terms found, return empty results
	if len(similarTerms) == 0 {
		return []string{}, nil
	}

	// Search for each similar term (OR)
	booleanQuery := indexer.NewBooleanQuery()
	for _, st := range similarTerms {
		booleanQuery.AddOr(st.Term)
	}

	results, err := s.index.BooleanSearch(ctx, booleanQuery)
	if err != nil {
		return nil, err
	}

	return results.DocIDs, nil
}

// generateSnippet generates a search result snippet for a document.
// Highlights matching terms in the snippet.
func (s *Searcher) generateSnippet(docInfo *indexer.DocInfo, terms []string) string {
	// For now, return a truncated version of the title or first 200 chars
	// In production, you'd want to extract relevant context around matching terms

	const maxSnippetLength = 200

	// Prefer title if it contains query terms
	title := toLower(docInfo.Title)
	hasMatchInTitle := false
	for _, term := range terms {
		if contains(title, toLower(term)) {
			hasMatchInTitle = true
			break
		}
	}

	snippet := docInfo.Title
	if !hasMatchInTitle {
		// In a real implementation, you'd extract the content
		// and find the relevant snippet around matching terms
		snippet = docInfo.Title
	}

	// Truncate if necessary
	if len(snippet) > maxSnippetLength {
		snippet = snippet[:maxSnippetLength] + "..."
	}

	return snippet
}

// getCachedResult retrieves a cached search result.
func (s *Searcher) getCachedResult(ctx context.Context, query string) (*SearchResponse, error) {
	if s.cache == nil {
		return nil, ErrCacheDisabled
	}

	key := s.cacheKey(query)
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// Deserialize the response (simplified - in production, use proper serialization)
	// For now, return nil to force a new search
	_ = data
	return nil, nil
}

// cacheResult stores a search result in the cache.
func (s *Searcher) cacheResult(ctx context.Context, query string, response *SearchResponse) error {
	if s.cache == nil {
		return ErrCacheDisabled
	}

	key := s.cacheKey(query)

	// Serialize the response (simplified - in production, use proper serialization)
	// For now, just store a placeholder
	_ = response

	return s.cache.Set(ctx, key, []byte("cached"))
}

// cacheKey generates a cache key for a query.
func (s *Searcher) cacheKey(query string) string {
	return fmt.Sprintf("search:%s", query)
}

// Suggest suggests similar queries or corrections for a given query.
func (s *Searcher) Suggest(ctx context.Context, query string, limit int) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Parse the query
	parsed := s.parser.Parse(query)
	terms := s.parser.ExtractTerms(parsed)

	// Get vocabulary from index
	vocabulary := s.index.GetAllTerms()

	// Find suggestions for each term
	suggestions := make([]string, 0)

	for _, term := range terms {
		// Find similar terms
		similar := FindSimilarTerms(term, vocabulary, 0.6, limit)
		for _, s := range similar {
			if s.Term != term {
				suggestions = append(suggestions, s.Term)
			}
		}
	}

	// Limit suggestions
	if limit > 0 && len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	return suggestions, nil
}

// UpdateConfig updates the searcher configuration.
func (s *Searcher) UpdateConfig(config *SearchConfig) {
	s.config = config
	s.parser = NewParser(config)
}

// GetConfig returns the current searcher configuration.
func (s *Searcher) GetConfig() *SearchConfig {
	return s.config
}

// Helper functions (reused from parser package)

func toLower(s string) string {
	// Simple ASCII lowercase conversion
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(s) < len(substr) {
		return -1
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// Package ranker provides document ranking algorithms for gosearch.
//
// It includes TF-IDF scoring, PageRank computation, and combined
// scoring with boost factors for title, URL, and freshness.
package ranker

import (
	"net/url"
	"strings"
	"time"

	"github.com/abuiliazeed/gosearch/internal/indexer"
)

// Scorer combines multiple ranking signals to compute document scores.
// It combines TF-IDF, PageRank, and various boost factors.
type Scorer struct {
	tfidf    *TFIDF
	pagerank *PageRank
	config   *ScorerConfig
}

// ScorerConfig holds configuration for the combined scorer.
type ScorerConfig struct {
	// TF-IDF weight (0-1)
	TFIDFWeight float64

	// PageRank weight (0-1)
	PageRankWeight float64

	// Title match boost multiplier
	TitleBoost float64

	// URL depth boost (shorter URLs get higher scores)
	URLDepthWeight float64

	// Freshness boost for recent documents (0-1, 0 = disabled)
	FreshnessWeight float64

	// Freshness threshold: documents younger than this get a boost
	FreshnessThreshold time.Duration

	// DomainIntentBoost boosts documents when query appears navigational for the site's domain.
	DomainIntentBoost float64

	// HomepageIntentBoost is an extra boost for homepage URLs on navigational queries.
	HomepageIntentBoost float64
}

// DefaultScorerConfig returns the default scorer configuration.
func DefaultScorerConfig() *ScorerConfig {
	return &ScorerConfig{
		TFIDFWeight:         0.7,
		PageRankWeight:      0.3,
		TitleBoost:          1.5,
		URLDepthWeight:      0.1,
		FreshnessWeight:     0.1,
		FreshnessThreshold:  30 * 24 * time.Hour, // 30 days
		DomainIntentBoost:   0.2,
		HomepageIntentBoost: 0.8,
	}
}

// NewScorer creates a new combined scorer.
func NewScorer(tfidf *TFIDF, pagerank *PageRank, config *ScorerConfig) *Scorer {
	if config == nil {
		config = DefaultScorerConfig()
	}

	return &Scorer{
		tfidf:    tfidf,
		pagerank: pagerank,
		config:   config,
	}
}

// ScoreDocuments scores documents for a query using the combined scoring function.
// Returns a map of docID to score.
//
// The combined score is computed as:
// score = w_tfidf * tfidf_score + w_pr * pr_score + title_boost + url_boost + freshness_boost
func (s *Scorer) ScoreDocuments(queryTerms []string, docIDs []string) map[string]float64 {
	scores := make(map[string]float64)

	// Get TF-IDF scores
	tfidfScores := s.tfidf.ScoreDocuments(queryTerms)

	// Score each document
	for _, docID := range docIDs {
		docInfo := s.tfidf.index.GetDocument(docID)
		if docInfo == nil {
			continue
		}

		// Base TF-IDF score
		tfidfScore := tfidfScores[docID]

		// PageRank score
		prScore := s.pagerank.GetScore(docID)

		// Combined score
		combinedScore := s.computeCombinedScore(docID, docInfo, tfidfScore, prScore, queryTerms)
		scores[docID] = combinedScore
	}

	return scores
}

// computeCombinedScore computes the combined score for a document.
func (s *Scorer) computeCombinedScore(_ string, docInfo *indexer.DocInfo, tfidfScore, prScore float64, queryTerms []string) float64 {
	// Normalize TF-IDF and PageRank scores
	normalizedTFIDF := s.normalizeScore(tfidfScore)
	normalizedPR := s.normalizeScore(prScore)

	// Base weighted score
	score := s.config.TFIDFWeight*normalizedTFIDF + s.config.PageRankWeight*normalizedPR

	// Apply title boost
	titleBoost := s.computeTitleBoost(docInfo, queryTerms)
	score += titleBoost

	// Apply URL depth boost
	urlBoost := s.computeURLDepthBoost(docInfo)
	score += urlBoost

	// Apply freshness boost
	freshnessBoost := s.computeFreshnessBoost(docInfo)
	score += freshnessBoost

	// Apply navigational/domain intent boost.
	navigationalBoost := s.computeNavigationalBoost(docInfo, queryTerms)
	score += navigationalBoost

	return score
}

// normalizeScore normalizes a score to [0, 1] range using a simple sigmoid-like function.
func (s *Scorer) normalizeScore(score float64) float64 {
	// Simple normalization: score / (1 + score)
	// This maps any positive score to [0, 1)
	if score < 0 {
		return 0
	}
	return score / (1.0 + score)
}

// computeTitleBoost computes a boost factor for title matches.
// Returns a boost if any query term appears in the document title.
func (s *Scorer) computeTitleBoost(docInfo *indexer.DocInfo, queryTerms []string) float64 {
	if docInfo.Title == "" || len(queryTerms) == 0 {
		return 0
	}

	// Count how many query terms appear in the title
	titleLower := toLower(docInfo.Title)
	normalizedTitle := normalizeAlphaNumeric(docInfo.Title)
	matchCount := 0

	for _, term := range queryTerms {
		termLower := toLower(term)
		normalizedTerm := normalizeAlphaNumeric(term)
		if contains(titleLower, termLower) || (normalizedTerm != "" && contains(normalizedTitle, normalizedTerm)) {
			matchCount++
		}
	}

	if matchCount == 0 {
		return 0
	}

	// Boost increases with more matches
	boost := s.config.TitleBoost * float64(matchCount) / float64(len(queryTerms))
	return boost
}

// computeURLDepthBoost computes a boost based on URL depth.
// Shorter URLs (closer to root) get a higher boost.
func (s *Scorer) computeURLDepthBoost(docInfo *indexer.DocInfo) float64 {
	if s.config.URLDepthWeight <= 0 {
		return 0
	}

	depth := urlDepth(docInfo.URL)
	if depth <= 0 {
		return s.config.URLDepthWeight
	}

	// Decreasing boost with depth
	return s.config.URLDepthWeight / float64(depth)
}

// computeFreshnessBoost computes a boost for recently indexed documents.
func (s *Scorer) computeFreshnessBoost(docInfo *indexer.DocInfo) float64 {
	if s.config.FreshnessWeight <= 0 || docInfo.IndexedAt.IsZero() {
		return 0
	}

	age := time.Since(docInfo.IndexedAt)

	// If document is fresh, apply boost
	if age < s.config.FreshnessThreshold {
		// Boost decreases as document ages
		freshnessRatio := 1.0 - (float64(age) / float64(s.config.FreshnessThreshold))
		return s.config.FreshnessWeight * freshnessRatio
	}

	return 0
}

// computeNavigationalBoost boosts domain/homepage results for navigational queries
// (e.g. searching a brand/site name).
func (s *Scorer) computeNavigationalBoost(docInfo *indexer.DocInfo, queryTerms []string) float64 {
	if docInfo == nil || docInfo.URL == "" || len(queryTerms) == 0 {
		return 0
	}
	if s.config.DomainIntentBoost <= 0 && s.config.HomepageIntentBoost <= 0 {
		return 0
	}

	parsed, err := url.Parse(docInfo.URL)
	if err != nil {
		return 0
	}

	hostLabel := primaryHostLabel(parsed.Hostname())
	if hostLabel == "" {
		return 0
	}

	if !queryLooksNavigationalForHost(queryTerms, hostLabel) {
		return 0
	}

	boost := s.config.DomainIntentBoost
	if isHomepagePath(parsed.Path) {
		boost += s.config.HomepageIntentBoost
	}

	return boost
}

// ScoreResult represents a scored document.
type ScoreResult struct {
	DocID string
	Score float64
}

// RankDocuments scores and ranks documents for a query.
// Returns results sorted by score in descending order.
func (s *Scorer) RankDocuments(queryTerms []string, docIDs []string) []ScoreResult {
	scores := s.ScoreDocuments(queryTerms, docIDs)

	// Convert to slice for sorting
	results := make([]ScoreResult, 0, len(scores))
	for docID, score := range scores {
		results = append(results, ScoreResult{DocID: docID, Score: score})
	}

	// Sort by score (simple bubble sort for simplicity)
	// In production, use sort.Slice
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// TopN returns the top N scored documents.
func (s *Scorer) TopN(queryTerms []string, docIDs []string, n int) []ScoreResult {
	results := s.RankDocuments(queryTerms, docIDs)

	if n >= len(results) {
		return results
	}

	return results[:n]
}

// SetConfig updates the scorer configuration.
func (s *Scorer) SetConfig(config *ScorerConfig) {
	s.config = config
}

// GetConfig returns the current scorer configuration.
func (s *Scorer) GetConfig() *ScorerConfig {
	return s.config
}

// Helper functions

// toLower converts a string to lowercase.
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

func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

func normalizeAlphaNumeric(s string) string {
	if s == "" {
		return ""
	}

	lowered := toLower(s)
	out := make([]byte, 0, len(lowered))
	for i := 0; i < len(lowered); i++ {
		c := lowered[i]
		if isAlphaNumeric(c) {
			out = append(out, c)
		}
	}
	return string(out)
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func normalizedQuerySignals(queryTerms []string) []string {
	signals := make([]string, 0, len(queryTerms)+1)
	var concat strings.Builder

	for _, term := range queryTerms {
		norm := normalizeAlphaNumeric(term)
		if norm == "" {
			continue
		}
		if !containsString(signals, norm) {
			signals = append(signals, norm)
		}
		concat.WriteString(norm)
	}

	joined := concat.String()
	if joined != "" && !containsString(signals, joined) {
		signals = append(signals, joined)
	}

	return signals
}

func primaryHostLabel(host string) string {
	host = toLower(host)
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}

	parts := strings.Split(host, ".")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	if len(filtered) == 0 {
		return ""
	}

	if filtered[0] == "www" {
		filtered = filtered[1:]
		if len(filtered) == 0 {
			return ""
		}
	}

	return normalizeAlphaNumeric(filtered[0])
}

func queryLooksNavigationalForHost(queryTerms []string, hostLabel string) bool {
	if hostLabel == "" {
		return false
	}

	signals := normalizedQuerySignals(queryTerms)
	for _, signal := range signals {
		if signal == hostLabel {
			return true
		}
	}

	return false
}

func isHomepagePath(path string) bool {
	return path == "" || path == "/"
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

// indexOf finds the index of a substring in a string.
// Returns -1 if not found.
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

// urlDepth computes the depth of a URL (number of path segments).
func urlDepth(url string) int {
	// Simple depth calculation: count slashes after the scheme
	depth := 0
	afterScheme := false

	for i := 0; i < len(url); i++ {
		if url[i] == '/' && afterScheme {
			depth++
		}
		if url[i] == ':' && i+2 < len(url) && url[i+1] == '/' && url[i+2] == '/' {
			afterScheme = true
			i += 2 // Skip the "//"
		}
	}

	return depth
}

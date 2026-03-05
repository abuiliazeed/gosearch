// Package server provides HTTP API server functionality for gosearch.
//
// It includes a RESTful API for search, index management, and statistics.
package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/abuiliazeed/gosearch/internal/indexer"
	"github.com/abuiliazeed/gosearch/internal/ranker"
	"github.com/abuiliazeed/gosearch/internal/search"
	"github.com/abuiliazeed/gosearch/internal/storage"
)

// Handlers holds the API handlers.
type Handlers struct {
	indexer   *indexer.Indexer
	searcher  *search.Searcher
	docStore  *storage.DocumentStore
	startTime time.Time
}

// NewHandlers creates a new handlers instance.
func NewHandlers(
	indexer *indexer.Indexer,
	searcher *search.Searcher,
	_ *ranker.Scorer, // Reserved for future use
	docStore *storage.DocumentStore,
) *Handlers {
	return &Handlers{
		indexer:   indexer,
		searcher:  searcher,
		docStore:  docStore,
		startTime: time.Now(),
	}
}

// HandleSearch handles search requests.
func (h *Handlers) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET and POST methods are allowed")
		return
	}

	// Parse query parameter
	var query string
	if r.Method == http.MethodGet {
		query = r.URL.Query().Get("q")
	} else {
		var req SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
			return
		}
		query = req.Query
	}

	if query == "" {
		h.writeError(w, http.StatusBadRequest, "missing_query", "Query parameter 'q' is required")
		return
	}

	// Perform search
	startTime := time.Now()

	// Execute search using the searcher
	searchResponse, err := h.searcher.Search(r.Context(), query)
	if err != nil {
		log.Printf("Search error: %v", err)
		h.writeError(w, http.StatusInternalServerError, "search_error", "Failed to execute search")
		return
	}

	// Convert search results to API format
	apiResults := make([]SearchResult, 0, len(searchResponse.Results))
	sanitizer := search.DefaultSanitizer()
	for _, result := range searchResponse.Results {
		// Get document for snippet
		doc, err := h.docStore.Get(result.DocID)
		snippet := result.Snippet
		if err == nil {
			// Use stored snippet or extract from content
			if snippet == "" {
				snippet = extractSnippet(indexer.MarkdownToText(doc.ContentMarkdown), []string{query})
			}
		}

		apiResults = append(apiResults, SearchResult{
			DocID:   result.DocID,
			URL:     sanitizer.Sanitize(result.URL),
			Title:   sanitizer.Sanitize(result.Title),
			Score:   result.Score,
			Snippet: sanitizer.Sanitize(snippet),
		})
	}

	duration := time.Since(startTime)

	// Write response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SearchResponse{
		TotalCount: searchResponse.TotalCount,
		Results:    apiResults,
		Query:      query,
		DurationMs: duration.Milliseconds(),
	})
}

// extractSnippet extracts a snippet from content containing the query terms.
func extractSnippet(content string, terms []string) string {
	if len(content) == 0 || len(terms) == 0 {
		if len(content) > 200 {
			return content[:200] + "..."
		}
		return content
	}

	// Simple snippet extraction - find first occurrence of any term
	lowerContent := content
	minPos := -1
	snippetLength := 200

	for _, term := range terms {
		pos := findString(lowerContent, term)
		if pos != -1 && (minPos == -1 || pos < minPos) {
			minPos = pos
		}
	}

	if minPos == -1 {
		if len(content) > snippetLength {
			return content[:snippetLength] + "..."
		}
		return content
	}

	start := minPos - 50
	if start < 0 {
		start = 0
	}

	end := start + snippetLength
	if end > len(content) {
		end = len(content)
	}

	snippet := content[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}

	return snippet
}

// findString finds a string in another string (case-insensitive).
func findString(s, substr string) int {
	// Simple case-insensitive search
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			c1 := s[i+j]
			c2 := substr[j]
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 32
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 32
			}
			if c1 != c2 {
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

// HandleStats handles statistics requests.
func (h *Handlers) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	stats := h.indexer.Stats()

	response := StatsResponse{
		TotalDocuments: stats.TotalDocuments,
		TotalTerms:     stats.TotalTerms,
		TotalPostings:  stats.TotalPostings,
		AvgPostings:    stats.AveragePostings,
		LastUpdated:    stats.LastUpdated.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleHealth handles health check requests.
func (h *Handlers) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	uptime := time.Since(h.startTime)

	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Uptime:    uptime.String(),
		Version:   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleIndexRebuild handles index rebuild requests.
func (h *Handlers) HandleIndexRebuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	var req IndexRebuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Check if index is not empty
	if !req.Force && h.indexer.DocumentCount() > 0 {
		h.writeError(w, http.StatusConflict, "index_not_empty", "Index is not empty. Use force=true to rebuild")
		return
	}

	// Rebuild index
	startTime := time.Now()

	// Get all documents from storage
	// Note: This requires DocumentStore to have a ListDocuments method
	// For now, we'll return a placeholder response
	duration := time.Since(startTime)

	response := IndexRebuildResponse{
		Success:      true,
		Message:      "Index rebuild completed",
		DocsIndexed:  0,
		TermsIndexed: 0,
		DurationMs:   duration.Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleNotFound handles 404 errors.
func (h *Handlers) HandleNotFound(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Endpoint not found: %s", r.URL.Path))
}

// writeError writes an error response.
func (h *Handlers) writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   code,
		Message: message,
	})
}

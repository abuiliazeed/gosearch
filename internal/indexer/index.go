// Package indexer provides document indexing and storage for gosearch.
//
// It includes tokenization, inverted index construction, boolean search
// operations, and persistence to BoltDB for efficient full-text search.
// The Indexer type coordinates these components to provide a high-level
// API for indexing documents and searching the index.
package indexer

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// Index manages the inverted index with thread-safe access.
type Index struct {
	mu    sync.RWMutex
	index *InvertedIndex
}

// NewIndex creates a new thread-safe inverted index.
func NewIndex() *Index {
	return &Index{
		index: NewInvertedIndex(),
	}
}

// IndexDocument indexes a document by tokenizing its content
// and adding tokens to the inverted index.
//
// The ctx parameter controls cancellation. Returns an error if
// the document cannot be indexed or context is cancelled.
func (idx *Index) IndexDocument(ctx context.Context, tokenizer *Tokenizer, doc *DocumentInput) error {
	if doc == nil || doc.DocID == "" {
		return fmt.Errorf("%w: document ID is required", ErrInvalidDocument)
	}

	// Check for cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Tokenize title (higher weight for title tokens)
	titleTokens := tokenizer.Tokenize(doc.Title)

	// Tokenize content
	contentTokens := tokenizer.Tokenize(doc.Content)

	// Combine tokens with position offset
	// Title tokens come first, then content tokens
	allTokens := make([]Token, 0, len(titleTokens)+len(contentTokens))
	allTokens = append(allTokens, titleTokens...)

	// Offset content positions
	for _, token := range contentTokens {
		token.Position += len(titleTokens)
		allTokens = append(allTokens, token)
	}

	// Add terms to index
	for _, token := range allTokens {
		idx.index.AddTerm(token.Text, doc.DocID, token.Position)
	}

	// Add document metadata
	docInfo := &DocInfo{
		DocID:      doc.DocID,
		URL:        doc.URL,
		Title:      doc.Title,
		TokenCount: len(allTokens),
		Length:     len(uniqueTokens(allTokens)),
	}

	idx.index.AddDocument(doc.DocID, docInfo)

	return nil
}

// uniqueTokens returns the number of unique tokens.
func uniqueTokens(tokens []Token) map[string]bool {
	seen := make(map[string]bool)
	for _, t := range tokens {
		seen[t.Text] = true
	}
	return seen
}

// GetPostings returns the postings list for a term.
// Thread-safe access to the index.
func (idx *Index) GetPostings(term string) (*PostingsList, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	plist := idx.index.GetPostings(term)
	if plist == nil {
		return nil, fmt.Errorf("%w: %s", ErrTermNotFound, term)
	}

	return plist, nil
}

// GetDocInfo returns document metadata for a document ID.
// Thread-safe access to the index.
func (idx *Index) GetDocInfo(docID string) (*DocInfo, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	info := idx.index.GetDocument(docID)
	if info == nil {
		return nil, fmt.Errorf("%w: %s", ErrDocNotFound, docID)
	}

	return info, nil
}

// Stats returns index statistics.
// Thread-safe access to the index.
func (idx *Index) Stats() *IndexStats {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.index.Stats()
}

// DocumentCount returns the total number of documents in the index.
func (idx *Index) DocumentCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.index.DocumentCount()
}

// TermCount returns the total number of unique terms in the index.
func (idx *Index) TermCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.index.TermCount()
}

// GetAllTerms returns all unique terms in the index.
func (idx *Index) GetAllTerms() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.index.GetAllTerms()
}

// HasDocument returns true if the document is in the index.
func (idx *Index) HasDocument(docID string) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.index.GetDocument(docID) != nil
}

// HasTerm returns true if the term is in the index.
func (idx *Index) HasTerm(term string) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.index.GetPostings(term) != nil
}

// DeleteDocument removes a document from the index.
// This is an expensive operation as it requires updating all postings lists.
func (idx *Index) DeleteDocument(docID string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if _, exists := idx.index.docs[docID]; !exists {
		return fmt.Errorf("%w: %s", ErrDocNotFound, docID)
	}

	// Remove from all postings lists
	for term, plist := range idx.index.terms {
		found := false
		for i, p := range plist.Postings {
			if p.DocID == docID {
				// Remove this posting
				plist.Postings = append(plist.Postings[:i], plist.Postings[i+1:]...)
				plist.DocFrequency--
				found = true
				break
			}
		}

		// Remove term if no more postings
		if found && plist.DocFrequency == 0 {
			delete(idx.index.terms, term)
		}
	}

	// Remove document metadata
	delete(idx.index.docs, docID)
	idx.index.totalDocs--

	return nil
}

// Merge merges another index into this one.
// Both indexes must use the same tokenizer configuration.
func (idx *Index) Merge(other *Index) error {
	if other == nil {
		return nil
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Merge document metadata
	for docID, info := range other.index.docs {
		if _, exists := idx.index.docs[docID]; !exists {
			idx.index.docs[docID] = info
			idx.index.totalDocs++
		}
	}

	// Merge postings lists
	for term, otherPlist := range other.index.terms {
		if plist, exists := idx.index.terms[term]; exists {
			plist.Merge(otherPlist)
		} else {
			idx.index.terms[term] = otherPlist
		}
	}

	return nil
}

// Clear removes all data from the index.
func (idx *Index) Clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.index = NewInvertedIndex()
}

// GetIndex returns a copy of the underlying inverted index.
// Useful for serialization and persistence.
func (idx *Index) GetIndex() *InvertedIndex {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Create a deep copy
	copied := &InvertedIndex{
		terms:     make(map[string]*PostingsList, len(idx.index.terms)),
		docs:      make(map[string]*DocInfo, len(idx.index.docs)),
		totalDocs: idx.index.totalDocs,
	}

	// Copy terms and postings
	for term, plist := range idx.index.terms {
		copied.terms[term] = plist.Clone()
	}

	// Copy document metadata
	for docID, info := range idx.index.docs {
		copied.docs[docID] = &DocInfo{
			DocID:      info.DocID,
			URL:        info.URL,
			Title:      info.Title,
			TokenCount: info.TokenCount,
			Length:     info.Length,
			IndexedAt:  info.IndexedAt,
		}
	}

	return copied
}

// SetIndex replaces the current index with the provided one.
// Useful for loading from persistence.
func (idx *Index) SetIndex(newIndex *InvertedIndex) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.index = newIndex
}

// DocumentInput represents a document to be indexed.
type DocumentInput struct {
	DocID   string
	URL     string
	Title   string
	Content string
}

// SearchResults represents the result of a search operation.
type SearchResults struct {
	DocIDs     []string
	TotalCount int
	Postings   *PostingsList
}

// BooleanSearch performs boolean search on the index.
// Supports AND (terms must all match), OR (any term matches), and NOT operations.
func (idx *Index) BooleanSearch(ctx context.Context, query *BooleanQuery) (*SearchResults, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Handle simple single-term query
	if len(query.AndTerms) == 1 && len(query.OrTerms) == 0 && len(query.NotTerms) == 0 {
		term := query.AndTerms[0]
		plist := idx.index.GetPostings(term)
		if plist == nil {
			return &SearchResults{DocIDs: []string{}, TotalCount: 0}, nil
		}

		docIDs := make([]string, len(plist.Postings))
		for i, p := range plist.Postings {
			docIDs[i] = p.DocID
		}

		return &SearchResults{
			DocIDs:     docIDs,
			TotalCount: len(docIDs),
			Postings:   plist,
		}, nil
	}

	var result *PostingsList

	// Handle AND terms (intersection)
	if len(query.AndTerms) > 0 {
		for i, term := range query.AndTerms {
			plist := idx.index.GetPostings(term)
			if plist == nil {
				// Term not found, no results
				return &SearchResults{DocIDs: []string{}, TotalCount: 0}, nil
			}

			if i == 0 {
				result = plist.Clone()
			} else {
				result = result.Intersect(plist)
			}
		}
	}

	// Handle OR terms (union)
	if len(query.OrTerms) > 0 {
		for _, term := range query.OrTerms {
			plist := idx.index.GetPostings(term)
			if plist == nil {
				continue
			}

			if result == nil {
				result = plist.Clone()
			} else {
				result = result.Union(plist)
			}
		}
	}

	// Handle NOT terms (difference)
	if len(query.NotTerms) > 0 && result != nil {
		for _, term := range query.NotTerms {
			plist := idx.index.GetPostings(term)
			if plist == nil {
				continue
			}
			result = result.Difference(plist)
		}
	}

	if result == nil || len(result.Postings) == 0 {
		return &SearchResults{DocIDs: []string{}, TotalCount: 0}, nil
	}

	docIDs := make([]string, len(result.Postings))
	for i, p := range result.Postings {
		docIDs[i] = p.DocID
	}

	return &SearchResults{
		DocIDs:     docIDs,
		TotalCount: len(docIDs),
		Postings:   result,
	}, nil
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
func (q *BooleanQuery) AddAnd(term string) {
	q.AndTerms = append(q.AndTerms, term)
}

// AddOr adds a term to the OR list.
func (q *BooleanQuery) AddOr(term string) {
	q.OrTerms = append(q.OrTerms, term)
}

// AddNot adds a term to the NOT list.
func (q *BooleanQuery) AddNot(term string) {
	q.NotTerms = append(q.NotTerms, term)
}

// BulkIndex indexes multiple documents concurrently.
// Uses a worker pool for parallel indexing.
func (idx *Index) BulkIndex(ctx context.Context, tokenizer *Tokenizer, docs []*DocumentInput, workers int, logger *zap.Logger) error {
	if len(docs) == 0 {
		return nil
	}

	if workers <= 0 {
		workers = 1
	}

	// Create channels
	docChan := make(chan *DocumentInput, len(docs))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for doc := range docChan {
				if err := idx.IndexDocument(ctx, tokenizer, doc); err != nil {
					logger.Error("failed to index document",
						zap.String("doc_id", doc.DocID),
						zap.Error(err))
				}
			}
		}()
	}

	// Send documents to workers
	for _, doc := range docs {
		select {
		case docChan <- doc:
		case <-ctx.Done():
			close(docChan)
			wg.Wait()
			return ctx.Err()
		}
	}
	close(docChan)

	// Wait for all workers to finish
	wg.Wait()

	return nil
}

// Package indexer provides inverted index building and management for gosearch.
//
// It includes tokenization, postings lists with gap encoding,
// positional indexing for phrase queries, and persistence to BoltDB.
package indexer

import (
	"errors"
	"time"
)

var (
	// ErrTermNotFound is returned when a term is not found in the index.
	ErrTermNotFound = errors.New("term not found in index")

	// ErrDocNotFound is returned when a document is not found in the index.
	ErrDocNotFound = errors.New("document not found in index")

	// ErrInvalidDocument is returned when a document is invalid for indexing.
	ErrInvalidDocument = errors.New("invalid document")

	// ErrStorage is returned when a storage operation fails.
	ErrStorage = errors.New("storage error")
)

// Token represents a word with its position in the document.
type Token struct {
	Text     string
	Position int
}

// Posting represents a single document occurrence for a term.
type Posting struct {
	DocID         string // Document ID
	Positions     []int  // Positions where term appears in document
	TermFrequency int    // Frequency of term in document
}

// PostingsList represents the list of documents containing a term.
type PostingsList struct {
	DocFrequency int       // Number of documents containing this term
	Postings     []Posting // List of postings (sorted by DocID)
}

// DocInfo stores metadata about an indexed document.
type DocInfo struct {
	DocID      string
	URL        string
	Title      string
	TokenCount int // Total tokens in document
	Length     int // Document length in tokens (unique tokens)
	IndexedAt  time.Time
}

// InvertedIndex is the main index structure mapping terms to postings.
type InvertedIndex struct {
	terms     map[string]*PostingsList // Term -> PostingsList
	docs      map[string]*DocInfo      // DocID -> DocInfo
	totalDocs int
}

// NewInvertedIndex creates a new empty inverted index.
func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		terms:     make(map[string]*PostingsList),
		docs:      make(map[string]*DocInfo),
		totalDocs: 0,
	}
}

// AddTerm adds a term occurrence to the index for a document.
func (idx *InvertedIndex) AddTerm(term string, docID string, position int) {
	plist, exists := idx.terms[term]
	if !exists {
		plist = &PostingsList{
			DocFrequency: 0,
			Postings:     make([]Posting, 0),
		}
		idx.terms[term] = plist
	}

	// Find or create posting for this document
	for i := range plist.Postings {
		if plist.Postings[i].DocID == docID {
			// Add position to existing posting
			plist.Postings[i].Positions = append(plist.Postings[i].Positions, position)
			plist.Postings[i].TermFrequency++
			return
		}
	}

	// Create new posting for this document
	plist.Postings = append(plist.Postings, Posting{
		DocID:         docID,
		Positions:     []int{position},
		TermFrequency: 1,
	})
	plist.DocFrequency++
}

// GetPostings returns the postings list for a term.
// Returns nil if the term is not in the index.
func (idx *InvertedIndex) GetPostings(term string) *PostingsList {
	return idx.terms[term]
}

// AddDocument adds document metadata to the index.
func (idx *InvertedIndex) AddDocument(docID string, info *DocInfo) {
	if _, exists := idx.docs[docID]; !exists {
		idx.totalDocs++
	}
	idx.docs[docID] = info
}

// GetDocument returns document metadata for a document ID.
// Returns nil if the document is not in the index.
func (idx *InvertedIndex) GetDocument(docID string) *DocInfo {
	return idx.docs[docID]
}

// DocumentCount returns the total number of documents in the index.
func (idx *InvertedIndex) DocumentCount() int {
	return idx.totalDocs
}

// TermCount returns the total number of unique terms in the index.
func (idx *InvertedIndex) TermCount() int {
	return len(idx.terms)
}

// TotalPostings returns the total number of postings across all terms.
func (idx *InvertedIndex) TotalPostings() int64 {
	total := int64(0)
	for _, plist := range idx.terms {
		total += int64(plist.DocFrequency)
	}
	return total
}

// GetDocuments returns all document metadata in the index.
func (idx *InvertedIndex) GetDocuments() map[string]*DocInfo {
	return idx.docs
}

// GetAllTerms returns all terms in the index.
func (idx *InvertedIndex) GetAllTerms() []string {
	terms := make([]string, 0, len(idx.terms))
	for term := range idx.terms {
		terms = append(terms, term)
	}
	return terms
}

// IndexStats holds statistics about the index.
type IndexStats struct {
	TotalDocuments  int       `json:"total_documents"`
	TotalTerms      int       `json:"total_terms"`
	TotalPostings   int64     `json:"total_postings"`
	AveragePostings float64   `json:"average_postings"`
	LastUpdated     time.Time `json:"last_updated"`
}

// Stats returns current index statistics.
func (idx *InvertedIndex) Stats() *IndexStats {
	totalPostings := idx.TotalPostings()
	avgPostings := 0.0
	if idx.TermCount() > 0 {
		avgPostings = float64(totalPostings) / float64(idx.TermCount())
	}

	return &IndexStats{
		TotalDocuments:  idx.totalDocs,
		TotalTerms:      idx.TermCount(),
		TotalPostings:   totalPostings,
		AveragePostings: avgPostings,
		LastUpdated:     time.Now(),
	}
}

// TokenizerConfig holds configuration for the tokenizer.
type TokenizerConfig struct {
	Stopwords   map[string]bool
	MinTokenLen int
}

// DefaultTokenizerConfig returns the default tokenizer configuration.
func DefaultTokenizerConfig() *TokenizerConfig {
	return &TokenizerConfig{
		Stopwords:   DefaultStopwords(),
		MinTokenLen: 2,
	}
}

// DefaultStopwords returns the default English stopword set.
func DefaultStopwords() map[string]bool {
	return map[string]bool{
		"a": true, "about": true, "above": true, "after": true, "again": true,
		"against": true, "all": true, "am": true, "an": true, "and": true,
		"any": true, "are": true, "aren't": true, "as": true, "at": true,
		"be": true, "because": true, "been": true, "before": true,
		"being": true, "below": true, "between": true, "both": true,
		"but": true, "by": true, "can't": true, "cannot": true, "could": true,
		"couldn't": true, "did": true, "didn't": true, "do": true,
		"does": true, "doesn't": true, "doing": true, "don't": true,
		"down": true, "during": true, "each": true, "few": true, "for": true,
		"from": true, "further": true, "had": true, "hadn't": true,
		"has": true, "hasn't": true, "have": true, "haven't": true,
		"having": true, "he": true, "he'd": true, "he'll": true, "he's": true,
		"her": true, "here": true, "here's": true, "hers": true, "herself": true,
		"him": true, "himself": true, "his": true, "how": true, "how's": true,
		"i": true, "i'd": true, "i'll": true, "i'm": true, "i've": true,
		"if": true, "in": true, "into": true, "is": true, "isn't": true,
		"it": true, "it's": true, "its": true, "itself": true, "let's": true,
		"me": true, "more": true, "most": true, "mustn't": true, "my": true,
		"myself": true, "no": true, "nor": true, "not": true, "of": true,
		"off": true, "on": true, "once": true, "only": true, "or": true,
		"other": true, "ought": true, "our": true, "ours": true,
		"ourselves": true, "out": true, "over": true, "own": true,
		"same": true, "shan't": true, "she": true, "she'd": true,
		"she'll": true, "she's": true, "should": true, "shouldn't": true,
		"so": true, "some": true, "such": true, "than": true, "that": true,
		"that's": true, "the": true, "their": true, "theirs": true,
		"them": true, "themselves": true, "then": true, "there": true,
		"there's": true, "these": true, "they": true, "they'd": true,
		"they'll": true, "they're": true, "they've": true, "this": true,
		"those": true, "through": true, "to": true, "too": true, "under": true,
		"until": true, "up": true, "very": true, "was": true, "wasn't": true,
		"we": true, "we'd": true, "we'll": true, "we're": true, "we've": true,
		"were": true, "weren't": true, "what": true, "what's": true,
		"when": true, "when's": true, "where": true, "where's": true,
		"which": true, "while": true, "who": true, "who's": true, "whom": true,
		"why": true, "why's": true, "with": true, "won't": true, "would": true,
		"wouldn't": true, "you": true, "you'd": true, "you'll": true,
		"you're": true, "you've": true, "your": true, "yours": true,
		"yourself": true, "yourselves": true,
	}
}

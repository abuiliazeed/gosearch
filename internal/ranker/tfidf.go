// Package ranker provides document ranking algorithms for gosearch.
//
// It includes TF-IDF scoring, PageRank computation, and combined
// scoring with boost factors for title, URL, and freshness.
package ranker

import (
	"math"

	"github.com/abuiliazeed/gosearch/internal/indexer"
)

// TFIDF computes Term Frequency-Inverse Document Frequency scores.
type TFIDF struct {
	index *indexer.InvertedIndex
}

// NewTFIDF creates a new TF-IDF scorer.
func NewTFIDF(idx *indexer.InvertedIndex) *TFIDF {
	return &TFIDF{index: idx}
}

// TermFrequency computes the term frequency for a term in a document.
// Uses raw term frequency (count of term occurrences).
func (t *TFIDF) TermFrequency(term, docID string) float64 {
	plist := t.index.GetPostings(term)
	if plist == nil {
		return 0
	}

	for _, posting := range plist.Postings {
		if posting.DocID == docID {
			return float64(posting.TermFrequency)
		}
	}

	return 0
}

// DocumentFrequency returns the number of documents containing the term.
func (t *TFIDF) DocumentFrequency(term string) int {
	plist := t.index.GetPostings(term)
	if plist == nil {
		return 0
	}
	return plist.DocFrequency
}

// InverseDocumentFrequency computes IDF for a term.
// IDF = log(totalDocs / docsContainingTerm)
// Uses natural logarithm.
// Returns 0 if the term is not found in any document.
func (t *TFIDF) InverseDocumentFrequency(term string) float64 {
	df := t.DocumentFrequency(term)
	if df == 0 {
		return 0
	}

	totalDocs := t.index.DocumentCount()
	if totalDocs == 0 {
		return 0
	}

	return math.Log(float64(totalDocs) / float64(df))
}

// TFIDFScore computes the TF-IDF score for a term in a document.
// TF-IDF = TF * IDF
func (t *TFIDF) TFIDFScore(term, docID string) float64 {
	tf := t.TermFrequency(term, docID)
	idf := t.InverseDocumentFrequency(term)
	return tf * idf
}

// DocumentVector computes the TF-IDF vector for a document.
// Returns a map of term to TF-IDF score for all terms in the document.
func (t *TFIDF) DocumentVector(docID string) map[string]float64 {
	docInfo := t.index.GetDocument(docID)
	if docInfo == nil {
		return nil
	}

	vector := make(map[string]float64)

	// Iterate through all terms in the index
	for _, term := range t.index.GetAllTerms() {
		// Check if document contains this term
		plist := t.index.GetPostings(term)
		if plist == nil {
			continue
		}

		for _, posting := range plist.Postings {
			if posting.DocID == docID {
				vector[term] = t.TFIDFScore(term, docID)
				break
			}
		}
	}

	return vector
}

// QueryVector computes the TF-IDF vector for a query.
// The query is treated as a document with term frequency = 1 for each query term.
func (t *TFIDF) QueryVector(queryTerms []string) map[string]float64 {
	vector := make(map[string]float64)

	// Count term frequencies in query
	termCounts := make(map[string]int)
	for _, term := range queryTerms {
		termCounts[term]++
	}

	// Compute TF-IDF for each query term
	for term, count := range termCounts {
		tf := float64(count)
		idf := t.InverseDocumentFrequency(term)
		vector[term] = tf * idf
	}

	return vector
}

// CosineSimilarity computes the cosine similarity between two document vectors.
// cos(θ) = (A · B) / (||A|| * ||B||)
// Returns 0 if either vector is empty.
func (t *TFIDF) CosineSimilarity(vecA, vecB map[string]float64) float64 {
	if len(vecA) == 0 || len(vecB) == 0 {
		return 0
	}

	// Compute dot product
	dotProduct := 0.0
	for term, weightA := range vecA {
		weightB, exists := vecB[term]
		if exists {
			dotProduct += weightA * weightB
		}
	}

	// Compute magnitudes
	magnitudeA := t.magnitude(vecA)
	magnitudeB := t.magnitude(vecB)

	if magnitudeA == 0 || magnitudeB == 0 {
		return 0
	}

	return dotProduct / (magnitudeA * magnitudeB)
}

// magnitude computes the Euclidean magnitude of a vector.
func (t *TFIDF) magnitude(vec map[string]float64) float64 {
	sum := 0.0
	for _, weight := range vec {
		sum += weight * weight
	}
	return math.Sqrt(sum)
}

// ScoreDocuments scores documents for a set of query terms using TF-IDF.
// Returns a map of docID to score, sorted by relevance.
// The score is the sum of TF-IDF scores for all query terms in each document.
func (t *TFIDF) ScoreDocuments(queryTerms []string) map[string]float64 {
	scores := make(map[string]float64)

	// For each query term, add TF-IDF score to matching documents
	for _, term := range queryTerms {
		plist := t.index.GetPostings(term)
		if plist == nil {
			continue
		}

		idf := t.InverseDocumentFrequency(term)

		for _, posting := range plist.Postings {
			tf := float64(posting.TermFrequency)
			tfidf := tf * idf
			scores[posting.DocID] += tfidf
		}
	}

	return scores
}

// BM25Score computes the BM25 score for a term in a document.
// BM25 is an improvement over TF-IDF that accounts for document length.
// BM25 = IDF * (TF * (k1 + 1)) / (TF + k1 * (1 - b + b * (|D| / avgDl)))
// where:
//   - k1 controls term frequency saturation (typically 1.2-2.0)
//   - b controls length normalization (typically 0.75)
//   - |D| is the document length
//   - avgDl is the average document length
func (t *TFIDF) BM25Score(term, docID string, k1, b float64, avgDocLength float64) float64 {
	plist := t.index.GetPostings(term)
	if plist == nil {
		return 0
	}

	docInfo := t.index.GetDocument(docID)
	if docInfo == nil {
		return 0
	}

	// Find term frequency in document
	tf := 0.0
	for _, posting := range plist.Postings {
		if posting.DocID == docID {
			tf = float64(posting.TermFrequency)
			break
		}
	}

	if tf == 0 {
		return 0
	}

	// Compute IDF
	idf := t.InverseDocumentFrequency(term)

	// Compute document length normalization
	docLength := float64(docInfo.TokenCount)
	if avgDocLength == 0 {
		avgDocLength = 1.0
	}

	// BM25 formula
	numerator := tf * (k1 + 1)
	denominator := tf + k1*(1-b+b*(docLength/avgDocLength))

	return idf * (numerator / denominator)
}

// ScoreDocumentsBM25 scores documents using BM25 ranking.
// Returns a map of docID to score.
func (t *TFIDF) ScoreDocumentsBM25(queryTerms []string, k1, b float64) map[string]float64 {
	scores := make(map[string]float64)

	// Compute average document length
	totalDocs := t.index.DocumentCount()
	if totalDocs == 0 {
		return scores
	}

	avgDocLength := 0.0
	for _, docInfo := range t.index.GetDocuments() {
		avgDocLength += float64(docInfo.TokenCount)
	}
	avgDocLength /= float64(totalDocs)

	// Score each document for each query term
	for _, term := range queryTerms {
		plist := t.index.GetPostings(term)
		if plist == nil {
			continue
		}

		for _, posting := range plist.Postings {
			bm25 := t.BM25Score(term, posting.DocID, k1, b, avgDocLength)
			scores[posting.DocID] += bm25
		}
	}

	return scores
}

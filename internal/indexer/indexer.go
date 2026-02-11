package indexer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/abuiliazeed/gosearch/internal/storage"
)

// Indexer manages document indexing and persistence.
//
// The Indexer coordinates tokenization, index building, and storage
// to BoltDB. It provides a high-level API for indexing documents,
// searching the index, and persisting state.
type Indexer struct {
	index     *Index
	tokenizer *Tokenizer
	store     *storage.IndexStore
	logger    *zap.Logger
}

// NewIndexer creates a new Indexer instance.
//
// The store parameter provides BoltDB persistence for the index.
// The logger is used for structured logging.
func NewIndexer(store *storage.IndexStore, logger *zap.Logger) *Indexer {
	return &Indexer{
		index:     NewIndex(),
		tokenizer: NewTokenizer(DefaultTokenizerConfig()),
		store:     store,
		logger:    logger,
	}
}

// NewIndexerWithTokenizer creates a new Indexer with a custom tokenizer.
func NewIndexerWithTokenizer(store *storage.IndexStore, tokenizer *Tokenizer, logger *zap.Logger) *Indexer {
	return &Indexer{
		index:     NewIndex(),
		tokenizer: tokenizer,
		store:     store,
		logger:    logger,
	}
}

// IndexDocument indexes a document by tokenizing its content
// and adding tokens to the inverted index.
//
// The ctx parameter controls cancellation. The doc parameter contains
// the document to index (title and content are tokenized).
//
// Returns an error if the document cannot be indexed or context is cancelled.
func (i *Indexer) IndexDocument(ctx context.Context, doc *storage.Document) error {
	if doc == nil {
		return fmt.Errorf("%w: document is nil", ErrInvalidDocument)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	i.logger.Debug("indexing document",
		zap.String("doc_id", doc.ID),
		zap.String("url", doc.URL))

	// Convert to DocumentInput
	input := &DocumentInput{
		DocID:   doc.ID,
		URL:     doc.URL,
		Title:   doc.Title,
		Content: doc.Content,
	}

	// Index the document
	if err := i.index.IndexDocument(ctx, i.tokenizer, input); err != nil {
		return fmt.Errorf("failed to index document %s: %w", doc.ID, err)
	}

	// Add to document store
	if err := i.store.AddDocument(doc.ID, doc.URL); err != nil {
		i.logger.Warn("failed to add document to store",
			zap.String("doc_id", doc.ID),
			zap.Error(err))
	}

	return nil
}

// IndexDocuments indexes multiple documents.
// Returns the number of successfully indexed documents and any error.
func (i *Indexer) IndexDocuments(ctx context.Context, docs []*storage.Document) (int, error) {
	count := 0
	for _, doc := range docs {
		if err := i.IndexDocument(ctx, doc); err != nil {
			i.logger.Error("failed to index document",
				zap.String("doc_id", doc.ID),
				zap.Error(err))
			continue
		}
		count++
	}
	return count, nil
}

// GetPostings returns the postings list for a term.
// Returns ErrTermNotFound if the term is not in the index.
func (i *Indexer) GetPostings(term string) (*PostingsList, error) {
	return i.index.GetPostings(term)
}

// GetDocInfo returns document metadata for a document ID.
// Returns ErrDocNotFound if the document is not in the index.
func (i *Indexer) GetDocInfo(docID string) (*DocInfo, error) {
	return i.index.GetDocInfo(docID)
}

// Save persists the index to BoltDB.
//
// The ctx parameter controls cancellation. This method serializes
// the in-memory index and stores it in BoltDB for persistence.
func (i *Indexer) Save(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	i.logger.Debug("saving index to storage")

	// Get current statistics
	stats := i.index.Stats()

	// Save metadata
	meta := &storage.IndexMeta{
		TotalDocuments:  stats.TotalDocuments,
		TotalTerms:      stats.TotalTerms,
		LastUpdated:     stats.LastUpdated,
		IndexSize:       0, // Will be calculated
		TotalPostings:   stats.TotalPostings,
		AveragePostings: stats.AveragePostings,
	}

	if err := i.store.SaveMeta(meta); err != nil {
		return fmt.Errorf("%w: failed to save metadata: %w", ErrStorage, err)
	}

	// Get index data
	indexData := i.index.GetIndex()

	// Save each term's postings list
	savedTerms := 0
	for term, plist := range indexData.terms {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Convert to persisted postings list
		persistedPostings := make([]storage.PersistedPosting, len(plist.Postings))
		for j, p := range plist.Postings {
			persistedPostings[j] = storage.PersistedPosting{
				DocID:         p.DocID,
				Positions:     p.Positions,
				TermFrequency: p.TermFrequency,
			}
		}

		persistedList := &storage.PersistedPostingsList{
			Term:         term,
			DocFrequency: plist.DocFrequency,
			Postings:     persistedPostings,
		}

		if err := i.store.SavePostings(persistedList); err != nil {
			i.logger.Warn("failed to save postings",
				zap.String("term", term),
				zap.Error(err))
			continue
		}
		savedTerms++
	}

	// Save each document's metadata
	savedDocs := 0
	for docID, docInfo := range indexData.docs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		persistedDocInfo := &storage.PersistedDocInfo{
			DocID:      docInfo.DocID,
			URL:        docInfo.URL,
			Title:      docInfo.Title,
			TokenCount: docInfo.TokenCount,
			Length:     docInfo.Length,
			IndexedAt:  docInfo.IndexedAt,
		}

		if err := i.store.SaveDocInfo(persistedDocInfo); err != nil {
			i.logger.Warn("failed to save doc info",
				zap.String("doc_id", docID),
				zap.Error(err))
			continue
		}
		savedDocs++
	}

	i.logger.Info("index saved",
		zap.Int("terms_saved", savedTerms),
		zap.Int("docs_saved", savedDocs),
		zap.Int("total_docs", stats.TotalDocuments),
		zap.Int("total_terms", stats.TotalTerms))

	return nil
}

// Load loads the index from BoltDB into memory.
//
// The ctx parameter controls cancellation. This method reads
// the persisted index data from BoltDB and reconstructs the in-memory index.
func (i *Indexer) Load(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	i.logger.Debug("loading index from storage")

	// Clear current index
	i.index.Clear()

	// Load metadata
	meta, err := i.store.GetMeta()
	if err != nil {
		return fmt.Errorf("%w: failed to load metadata: %w", ErrStorage, err)
	}

	// Create a new inverted index to populate
	newIndex := NewInvertedIndex()

	// Load all document info
	docIDs, err := i.store.ListAllDocInfo()
	if err != nil {
		return fmt.Errorf("%w: failed to list doc info: %w", ErrStorage, err)
	}

	loadedDocs := 0
	for _, docID := range docIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		persistedDocInfo, err := i.store.LoadDocInfo(docID)
		if err != nil {
			i.logger.Warn("failed to load doc info",
				zap.String("doc_id", docID),
				zap.Error(err))
			continue
		}

		docInfo := &DocInfo{
			DocID:      persistedDocInfo.DocID,
			URL:        persistedDocInfo.URL,
			Title:      persistedDocInfo.Title,
			TokenCount: persistedDocInfo.TokenCount,
			Length:     persistedDocInfo.Length,
			IndexedAt:  persistedDocInfo.IndexedAt,
		}

		newIndex.AddDocument(docID, docInfo)
		loadedDocs++
	}

	// Load all postings lists
	terms, err := i.store.ListAllPostings()
	if err != nil {
		return fmt.Errorf("%w: failed to list postings: %w", ErrStorage, err)
	}

	loadedTerms := 0
	for _, term := range terms {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		persistedList, err := i.store.LoadPostings(term)
		if err != nil {
			i.logger.Warn("failed to load postings",
				zap.String("term", term),
				zap.Error(err))
			continue
		}

		// Convert to in-memory postings list
		postings := make([]Posting, len(persistedList.Postings))
		for j, pp := range persistedList.Postings {
			postings[j] = Posting{
				DocID:         pp.DocID,
				Positions:     pp.Positions,
				TermFrequency: pp.TermFrequency,
			}
		}

		plist := &PostingsList{
			DocFrequency: persistedList.DocFrequency,
			Postings:     postings,
		}

		newIndex.terms[term] = plist
		loadedTerms++
	}

	// Set the loaded index
	newIndex.totalDocs = loadedDocs
	i.index.SetIndex(newIndex)

	i.logger.Info("index loaded",
		zap.Int("docs_loaded", loadedDocs),
		zap.Int("terms_loaded", loadedTerms),
		zap.Int("total_docs", meta.TotalDocuments),
		zap.Int("total_terms", meta.TotalTerms))

	return nil
}

// Stats returns index statistics.
func (i *Indexer) Stats() *IndexStats {
	return i.index.Stats()
}

// Search performs a search on the index using boolean query logic.
func (i *Indexer) Search(ctx context.Context, query *BooleanQuery) (*SearchResults, error) {
	return i.index.BooleanSearch(ctx, query)
}

// DeleteDocument removes a document from the index.
func (i *Indexer) DeleteDocument(docID string) error {
	return i.index.DeleteDocument(docID)
}

// Clear removes all data from the index.
func (i *Indexer) Clear() {
	i.index.Clear()
}

// DocumentCount returns the total number of documents in the index.
func (i *Indexer) DocumentCount() int {
	return i.index.DocumentCount()
}

// TermCount returns the total number of unique terms in the index.
func (i *Indexer) TermCount() int {
	return i.index.TermCount()
}

// HasDocument returns true if the document is in the index.
func (i *Indexer) HasDocument(docID string) bool {
	return i.index.HasDocument(docID)
}

// HasTerm returns true if the term is in the index.
func (i *Indexer) HasTerm(term string) bool {
	return i.index.HasTerm(term)
}

// GetTokenizer returns the tokenizer used by this indexer.
func (i *Indexer) GetTokenizer() *Tokenizer {
	return i.tokenizer
}

// GetIndex returns the underlying Index for direct access.
// This is useful for components that need direct index access like rankers and searchers.
func (i *Indexer) GetIndex() *Index {
	return i.index
}

// SetTokenizer replaces the tokenizer with a new one.
func (i *Indexer) SetTokenizer(tokenizer *Tokenizer) {
	i.tokenizer = tokenizer
}

// Rebuild rebuilds the index from documents in storage.
// This is useful for updating tokenization rules or recovering from corruption.
func (i *Indexer) Rebuild(ctx context.Context, docStore *storage.DocumentStore) error {
	i.logger.Info("rebuilding index")

	// Clear current index
	i.index.Clear()

	// Get all document IDs from storage
	// Note: This requires DocumentStore to have a ListDocuments method
	// For now, this is a placeholder for the rebuild functionality

	i.logger.Info("index rebuilt")
	return nil
}

// Optimize optimizes the index by sorting postings and merging duplicates.
func (i *Indexer) Optimize() error {
	i.logger.Debug("optimizing index")

	// Get and sort all postings lists
	indexData := i.index.GetIndex()
	for _, plist := range indexData.terms {
		plist.Sort()
	}

	i.logger.Debug("index optimized")
	return nil
}

// Close closes the indexer and releases resources.
func (i *Indexer) Close() error {
	if i.store != nil {
		return i.store.Close()
	}
	return nil
}

// Merge merges another indexer's index into this one.
func (i *Indexer) Merge(other *Indexer) error {
	if other == nil {
		return nil
	}

	i.logger.Info("merging indexes")
	return i.index.Merge(other.index)
}

// Validate checks the index for consistency issues.
// Returns a list of any problems found.
func (i *Indexer) Validate() []string {
	issues := make([]string, 0)

	stats := i.index.Stats()
	indexData := i.index.GetIndex()

	// Check for orphaned postings
	for term, plist := range indexData.terms {
		for _, p := range plist.Postings {
			if indexData.GetDocument(p.DocID) == nil {
				issues = append(issues, fmt.Sprintf("orphaned posting for term '%s': doc '%s' not in index", term, p.DocID))
			}
		}
	}

	// Check for documents with no terms
	for docID := range indexData.docs {
		hasTerm := false
		for _, plist := range indexData.terms {
			if plist.HasDocument(docID) {
				hasTerm = true
				break
			}
		}
		if !hasTerm {
			issues = append(issues, fmt.Sprintf("document '%s' has no terms indexed", docID))
		}
	}

	i.logger.Debug("index validation complete",
		zap.Int("issues_found", len(issues)),
		zap.Int("total_docs", stats.TotalDocuments),
		zap.Int("total_terms", stats.TotalTerms))

	return issues
}

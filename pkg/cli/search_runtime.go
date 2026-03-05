package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/abuiliazeed/gosearch/internal/indexer"
	"github.com/abuiliazeed/gosearch/internal/ranker"
	"github.com/abuiliazeed/gosearch/internal/search"
	"github.com/abuiliazeed/gosearch/internal/storage"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type searchRuntime struct {
	logger     *zap.Logger
	indexStore *storage.IndexStore
	docStore   *storage.DocumentStore
	cache      *storage.CacheStore
	index      *indexer.Index
	scorer     *ranker.Scorer
	noCache    bool
}

func newSearchRuntime(noCache bool) (*searchRuntime, error) {
	dataDir := viper.GetString("data-dir")
	if err := requireSchemaVersion(dataDir); err != nil {
		return nil, err
	}

	indexPath := filepath.Join(dataDir, "index", "index.db")
	pagesPath := filepath.Join(dataDir, "pages")
	logger := initLogger()

	indexStore, err := storage.NewIndexStore(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index store: %w", err)
	}

	docStore, err := storage.NewDocumentStore(pagesPath)
	if err != nil {
		_ = indexStore.Close()
		return nil, fmt.Errorf("failed to open document store: %w", err)
	}

	rt := &searchRuntime{
		logger:     logger,
		indexStore: indexStore,
		docStore:   docStore,
		noCache:    noCache,
	}

	idxr := indexer.NewIndexer(indexStore, logger)

	meta, err := indexStore.GetMeta()
	if err == nil && meta.TotalDocuments > 0 {
		if loadErr := idxr.Load(context.Background()); loadErr != nil {
			logger.Warn("failed to load index metadata, will rebuild from documents", zap.Error(loadErr))
		}
	}

	if idxr.DocumentCount() == 0 {
		docIDs, listErr := docStore.List()
		if listErr != nil {
			_ = rt.Close()
			return nil, fmt.Errorf("failed to list documents for index rebuild: %w", listErr)
		}
		if len(docIDs) == 0 {
			_ = rt.Close()
			return nil, fmt.Errorf("index is empty. run crawl/index first")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		count, indexErr := idxr.IndexDocuments(ctx, loadDocuments(docStore, docIDs))
		if indexErr != nil {
			_ = rt.Close()
			return nil, fmt.Errorf("failed to rebuild index from stored documents: %w", indexErr)
		}

		logger.Info("rebuilt index from stored documents", zap.Int("documents", count))
		if saveErr := idxr.Save(context.Background()); saveErr != nil {
			logger.Warn("failed to persist rebuilt index", zap.Error(saveErr))
		}
	}

	idx := idxr.GetIndex()
	rt.index = idx
	rt.scorer = ranker.NewScorer(
		ranker.NewTFIDF(idx.GetIndex()),
		ranker.DefaultPageRank(),
		nil,
	)

	if !noCache {
		redisHost := firstNonEmpty(viper.GetString("redis-host"), viper.GetString("redis.host"))
		redisPassword := firstNonEmpty(viper.GetString("redis-password"), viper.GetString("redis.password"))
		redisDB := viper.GetInt("redis-db")
		if redisDB == 0 {
			redisDB = viper.GetInt("redis.db")
		}

		cache, cacheErr := storage.NewCacheStore(redisHost, redisPassword, redisDB, 5*time.Minute)
		if cacheErr != nil {
			logger.Warn("failed to connect to Redis, caching disabled", zap.Error(cacheErr))
		} else {
			rt.cache = cache
		}
	}

	return rt, nil
}

func (rt *searchRuntime) Search(ctx context.Context, query string, maxResults int) (*search.SearchResponse, error) {
	searcher, query, err := rt.buildSearcher(query, maxResults)
	if err != nil {
		return nil, err
	}

	response, err := searcher.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	return response, nil
}

func (rt *searchRuntime) Suggest(ctx context.Context, query string, limit int) ([]string, error) {
	searchMax := limit
	if searchMax < 10 {
		searchMax = 10
	}
	searcher, query, err := rt.buildSearcher(query, searchMax)
	if err != nil {
		return nil, err
	}

	if limit < 1 {
		limit = 3
	}

	suggestions, err := searcher.Suggest(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate suggestions: %w", err)
	}

	// Deduplicate while preserving order.
	seen := make(map[string]struct{}, len(suggestions))
	cleaned := make([]string, 0, len(suggestions))
	for _, suggestion := range suggestions {
		trimmed := strings.TrimSpace(suggestion)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		cleaned = append(cleaned, trimmed)
		if len(cleaned) >= limit {
			break
		}
	}

	return cleaned, nil
}

func (rt *searchRuntime) GetDocument(docID string) (*storage.Document, error) {
	if rt == nil || rt.docStore == nil {
		return nil, fmt.Errorf("document store is not initialized")
	}
	return rt.docStore.Get(docID)
}

func (rt *searchRuntime) buildSearcher(query string, maxResults int) (*search.Searcher, string, error) {
	if rt == nil || rt.index == nil || rt.scorer == nil {
		return nil, "", fmt.Errorf("search runtime is not initialized")
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return nil, "", fmt.Errorf("query cannot be empty")
	}

	if maxResults < 1 {
		maxResults = 10
	}

	searcher := search.NewSearcher(rt.index, rt.scorer, rt.cache, &search.Config{
		CacheEnabled:   !rt.noCache && rt.cache != nil,
		CacheTTL:       5 * time.Minute,
		MaxResults:     maxResults,
		FuzzyEnabled:   true,
		FuzzyDistance:  2,
		PhraseEnabled:  true,
		BooleanEnabled: true,
	})

	return searcher, query, nil
}

func (rt *searchRuntime) Close() error {
	if rt == nil {
		return nil
	}

	var firstErr error
	recordErr := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if rt.cache != nil {
		rt.cache.Close()
	}
	if rt.docStore != nil {
		recordErr(rt.docStore.Close())
	}
	if rt.indexStore != nil {
		recordErr(rt.indexStore.Close())
	}
	if rt.logger != nil {
		_ = rt.logger.Sync()
	}

	return firstErr
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

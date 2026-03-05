// Package cli provides the command-line interface for gosearch.
//
// This file contains the search command.
package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/abuiliazeed/gosearch/internal/indexer"
	"github.com/abuiliazeed/gosearch/internal/ranker"
	"github.com/abuiliazeed/gosearch/internal/search"
	"github.com/abuiliazeed/gosearch/internal/storage"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search the index",
	Long: `Search the inverted index for documents matching the query.

Requires storage schema 2-markdown-only. If your data directory is not
initialized yet, run a crawl first.

Supports boolean operators:
  AND or &&  - Both terms must match
  OR or ||   - Either term must match
  NOT or !   - Exclude term

Phrase queries use double quotes: "exact phrase match"

Fuzzy matching uses tilde: term~ (finds similar terms)`,
	Example: `  # Simple search
  gosearch search "sourcebeauty"

  # Boolean query
  gosearch search "sourcebeauty AND skincare"

  # Phrase query
  gosearch search "\"full coverage concealer\""

  # Disable cache
  gosearch search "sourcebeauty" --no-cache`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		fuzzyEnabled, _ := cmd.Flags().GetBool("fuzzy")
		explain, _ := cmd.Flags().GetBool("explain")
		noCache, _ := cmd.Flags().GetBool("no-cache")

		// Get data directory
		dataDir := viper.GetString("data-dir")
		if err := requireSchemaVersion(dataDir); err != nil {
			return err
		}
		indexPath := filepath.Join(dataDir, "index", "index.db")

		// Initialize logger
		logger := initLogger()

		// Open index store
		indexStore, err := storage.NewIndexStore(indexPath)
		if err != nil {
			return fmt.Errorf("failed to open index store: %w", err)
		}
		defer indexStore.Close()

		// Create indexer
		indexer := indexer.NewIndexer(indexStore, logger)

		// Try to load existing index metadata
		meta, err := indexStore.GetMeta()
		hasIndex := err == nil && meta.TotalDocuments > 0

		if hasIndex {
			err = indexer.Load(context.Background())
			if err != nil {
				logger.Warn("failed to load index metadata, will rebuild if needed", zap.Error(err))
			}
		}

		// If index is empty, try to rebuild from documents
		if indexer.DocumentCount() == 0 {
			logger.Info("index is empty, rebuilding from crawled documents...")
			docStorePath := filepath.Join(dataDir, "pages")
			docStore, err := storage.NewDocumentStore(docStorePath)
			if err != nil {
				return fmt.Errorf("failed to open document store: %w", err)
			}
			defer docStore.Close()

			docIDs, err := docStore.List()
			if err != nil {
				return fmt.Errorf("failed to list documents: %w", err)
			}

			if len(docIDs) == 0 {
				return fmt.Errorf("no documents found. Please run 'gosearch crawl' first")
			}

			ctx := context.Background()
			docs := loadDocuments(docStore, docIDs)
			count, err := indexer.IndexDocuments(ctx, docs)
			if err != nil {
				return fmt.Errorf("failed to rebuild index: %w", err)
			}
			logger.Info("rebuilt index", zap.Int("documents", count))
		}

		idx := indexer.GetIndex()

		// Create ranker
		tfidf := ranker.NewTFIDF(idx.GetIndex())
		pr := ranker.DefaultPageRank()
		scorer := ranker.NewScorer(tfidf, pr, nil)

		// Create cache store (if not disabled)
		var cache *storage.CacheStore
		if !noCache {
			redisHost := viper.GetString("redis-host")
			redisPassword := viper.GetString("redis-password")
			redisDB := viper.GetInt("redis-db")

			cache, err = storage.NewCacheStore(redisHost, redisPassword, redisDB, 5*time.Minute)
			if err != nil {
				logger.Warn("failed to connect to Redis, caching disabled", zap.Error(err))
				cache = nil
			}
			defer func() {
				if cache != nil {
					cache.Close()
				}
			}()
		}

		// Create searcher config
		searchConfig := &search.Config{
			CacheEnabled:   !noCache && cache != nil,
			CacheTTL:       5 * time.Minute,
			MaxResults:     limit + offset, // Get extra for offset
			FuzzyEnabled:   fuzzyEnabled,
			FuzzyDistance:  2,
			PhraseEnabled:  true,
			BooleanEnabled: true,
		}

		// Create searcher
		searcher := search.NewSearcher(idx, scorer, cache, searchConfig)

		// Execute search
		query := args[0]
		fmt.Printf("Searching for: %s\n\n", query)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response, err := searcher.Search(ctx, query)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		// Apply offset
		results := response.Results
		if offset > 0 && offset < len(results) {
			results = results[offset:]
		}
		if limit > 0 && limit < len(results) {
			results = results[:limit]
		}

		// Display results
		if response.TotalCount == 0 {
			fmt.Println("No results found")
			return nil
		}

		fmt.Printf("Found %d results in %v\n", response.TotalCount, response.Duration.Round(time.Millisecond))
		if response.Cached {
			fmt.Println("(cached)")
		}
		fmt.Printf("\nShowing %d results:\n\n", len(results))

		for i, result := range results {
			fmt.Printf("%d. [%0.2f] %s\n", i+1+offset, result.Score, result.Title)

			if explain {
				fmt.Printf("    DocID: %s\n", result.DocID)
			}

			if result.Snippet != "" {
				fmt.Printf("    %s\n", result.Snippet)
			}

			fmt.Printf("    %s\n\n", result.URL)
		}

		// Show suggestions if fuzzy is enabled
		if fuzzyEnabled && response.TotalCount == 0 {
			fmt.Println("\nDid you mean...")
			suggestions, err := searcher.Suggest(ctx, query, 5)
			if err == nil && len(suggestions) > 0 {
				for _, s := range suggestions {
					fmt.Printf("  - %s\n", s)
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().IntP("limit", "l", 10, "maximum number of results to return")
	searchCmd.Flags().IntP("offset", "o", 0, "offset for pagination")
	searchCmd.Flags().BoolP("fuzzy", "f", true, "enable fuzzy matching")
	searchCmd.Flags().BoolP("explain", "e", false, "show scoring explanation")
	searchCmd.Flags().Bool("no-cache", false, "disable query result caching")
}

// loadDocuments loads documents from the document store by their IDs.
func loadDocuments(docStore *storage.DocumentStore, docIDs []string) []*storage.Document {
	docs := make([]*storage.Document, 0, len(docIDs))
	for _, docID := range docIDs {
		doc, err := docStore.Get(docID)
		if err != nil {
			continue
		}
		docs = append(docs, doc)
	}
	return docs
}

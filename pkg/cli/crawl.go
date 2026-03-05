package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/abuiliazeed/gosearch/internal/crawler"
	"github.com/abuiliazeed/gosearch/internal/indexer"
	"github.com/abuiliazeed/gosearch/internal/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var crawlCmd = &cobra.Command{
	Use:   "crawl [urls...]",
	Short: "Crawl web pages and build the index",
	Long: `Crawl web pages starting from the given seed URLs.
Respects robots.txt, applies rate limiting, stores cleaned markdown content,
and builds the inverted index.

Default indexing mode is incremental: only newly crawled documents are
processed for indexing. Use --full-reindex to rebuild from all stored pages.

Schema behavior:
  - Expected schema: 2-markdown-only
  - If schema marker is missing/legacy, crawl performs a one-time hard reset
    of pages and index before starting.

Use --seed-set to use predefined seed sets (general, programming, academic).
Use --seeds-file to load custom seed sets from a file.
Use --strategy to control crawling order (bfs, best-first).

If no URLs are provided and no seed set is specified, lists available seed sets.`,
	Example: `  # Crawl one site incrementally (default mode)
  gosearch crawl https://sourcebeauty.com -L 2 -w 10 --max-queue 100

  # Full corpus reindex from stored pages
  gosearch crawl https://sourcebeauty.com --full-reindex -L 0 -w 1 --max-queue 1

  # Crawl from predefined seeds
  gosearch crawl --seed-set=programming`,
	Args: cobra.MinimumNArgs(0),
	RunE: runCrawl,
}

func runCrawl(cmd *cobra.Command, args []string) error {
	totalStart := time.Now()
	// Get flags
	maxDepth, _ := cmd.Flags().GetInt("depth")
	maxWorkers, _ := cmd.Flags().GetInt("workers")
	maxQueueSize, _ := cmd.Flags().GetInt("max-queue")
	delay, _ := cmd.Flags().GetInt("delay")
	allowed, _ := cmd.Flags().GetStringSlice("allow")
	disallowed, _ := cmd.Flags().GetStringSlice("disallow")
	seedSet, _ := cmd.Flags().GetString("seed-set")
	seedsFile, _ := cmd.Flags().GetString("seeds-file")
	strategy, _ := cmd.Flags().GetString("strategy")
	fullReindex, _ := cmd.Flags().GetBool("full-reindex")
	dataDir := viper.GetString("data-dir")

	resetPerformed, err := ensureSchemaForCrawl(dataDir)
	if err != nil {
		return err
	}
	if resetPerformed {
		fmt.Println("Detected legacy or missing data schema. Running one-time hard reset for markdown-only v2.")
		fmt.Printf("  Cleared pages: %s\n", filepath.Join(dataDir, "pages"))
		fmt.Printf("  Cleared index: %s\n", filepath.Join(dataDir, "index", "index.db"))
		fmt.Printf("  Schema marker: %s (%s)\n", filepath.Join(dataDir, schemaVersionFileName), schemaVersionV2)
		fmt.Println()
	}

	// Determine seed URLs
	seedURLs := determineSeeds(args, seedSet, seedsFile)
	if len(seedURLs) == 0 {
		// No seeds determined - show help
		return listAvailableSeedSets(cmd)
	}

	// Derive paths from data directory
	indexPath := dataDir + "/index/index.db"
	pagesPath := dataDir + "/pages"

	// Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	// Initialize storage
	docStore, err := storage.NewDocumentStore(pagesPath)
	if err != nil {
		return fmt.Errorf("failed to create document store: %w", err)
	}
	defer docStore.Close()

	indexStore, err := storage.NewIndexStore(indexPath)
	if err != nil {
		return fmt.Errorf("failed to create index store: %w", err)
	}
	defer indexStore.Close()

	// Create indexer
	idxr := indexer.NewIndexer(indexStore, logger)

	// Configure crawler with strategy
	config := &crawler.Config{
		MaxWorkers:      maxWorkers,
		MaxDepth:        maxDepth,
		MaxQueueSize:    maxQueueSize,
		Delay:           time.Duration(delay) * time.Millisecond,
		AllowedDomains:  allowed,
		DisallowedPaths: disallowed,
		UserAgent:       "GoSearch/1.0 (+https://github.com/abuiliazeed/gosearch)",
		Timeout:         30 * time.Second,
		RespectRobots:   true,
		Strategy:        crawler.ParseStrategy(strategy),
	}

	// Create crawler
	crawlr, err := crawler.NewCollyCrawler(config, docStore)
	if err != nil {
		return fmt.Errorf("failed to create crawler: %w", err)
	}

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start crawling in a goroutine
	var crawlErr error
	var crawlWg sync.WaitGroup

	crawlStart := time.Now()
	crawlWg.Add(1)
	go func() {
		defer crawlWg.Done()
		fmt.Printf("Starting crawl...\n")
		fmt.Printf("  Strategy: %s\n", config.Strategy)
		fmt.Printf("  Seed URLs: %v\n", seedURLs)
		fmt.Printf("  Max depth: %d\n", maxDepth)
		if maxQueueSize > 0 {
			fmt.Printf("  Max queue size: %d\n", maxQueueSize)
		} else {
			fmt.Printf("  Max queue size: unlimited\n")
		}
		fmt.Printf("  Max workers: %d\n", maxWorkers)
		fmt.Printf("  Delay: %dms\n", delay)
		fmt.Printf("  Pages path: %s\n", pagesPath)
		fmt.Printf("  Index path: %s\n", indexPath)
		fmt.Println()

		crawlErr = crawlr.Start(ctx, seedURLs)
		// Cancel parent context when crawler completes
		cancel()
	}()

	// Monitor progress
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	done := false
	for !done {
		select {
		case <-sigChan:
			fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
			cancel()
			done = true

		case <-ticker.C:
			stats := crawlr.Stats()
			fmt.Printf("\rCrawled: %d | Queued: %d | Failed: %d",
				stats.URLsCrawled,
				stats.URLsQueued,
				stats.URLsFailed)

		case <-ctx.Done():
			done = true
		}
	}

	// Wait for crawler to finish
	crawlWg.Wait()
	fmt.Printf("\nCrawling took: %v\n", time.Since(crawlStart).Round(time.Millisecond))

	// Index new crawl results by default (incremental mode). Full corpus rebuild is opt-in.
	fmt.Println("\nIndexing documents...")
	indexStart := time.Now()
	indexCtx := context.Background()
	if !fullReindex {
		// In incremental mode, load existing index first so we can merge new pages.
		meta, metaErr := indexStore.GetMeta()
		if metaErr == nil && meta.TotalDocuments > 0 {
			if err := idxr.Load(indexCtx); err != nil {
				return fmt.Errorf("failed to load existing index for incremental crawl: %w", err)
			}
		}
	}

	var docIDs []string
	if fullReindex {
		docIDs, err = docStore.List()
		if err != nil {
			logger.Warn("failed to list documents for full reindex", zap.Error(err))
		}
		fmt.Printf("  Mode: full reindex (%d stored documents)\n", len(docIDs))
	} else {
		docIDs = crawlr.GetSavedDocIDs()
		fmt.Printf("  Mode: incremental (%d newly crawled documents)\n", len(docIDs))
	}

	var indexErrors []error
	indexedCount := 0
	for i, docID := range docIDs {
		doc, err := docStore.Get(docID)
		if err != nil {
			logger.Warn("failed to get document",
				zap.String("doc_id", docID),
				zap.Error(err))
			continue
		}

		// In incremental mode, replace any existing version of this document.
		if !fullReindex && idxr.HasDocument(doc.ID) {
			if err := idxr.DeleteDocument(doc.ID); err != nil {
				logger.Warn("failed to delete existing document before reindex",
					zap.String("doc_id", doc.ID),
					zap.String("url", doc.URL),
					zap.Error(err))
				continue
			}
		}

		if err := idxr.IndexDocument(indexCtx, doc); err != nil {
			indexErrors = append(indexErrors, fmt.Errorf("index error for %s: %w", doc.URL, err))
		} else {
			indexedCount++
		}

		if i%10 == 0 {
			fmt.Printf("\rIndexed: %d/%d", i+1, len(docIDs))
		}
	}
	fmt.Printf("\rIndexed: %d/%d\n", indexedCount, len(docIDs))

	if len(indexErrors) > 0 {
		logger.Warn("indexing completed with errors",
			zap.Int("error_count", len(indexErrors)))
		// Log first few errors for debugging
		for i, err := range indexErrors {
			if i < 5 {
				logger.Error("indexing error", zap.Error(err))
			}
		}
	}
	fmt.Printf("Indexing took: %v\n", time.Since(indexStart).Round(time.Millisecond))

	// Save index (use fresh context since original may be canceled)
	if len(docIDs) > 0 {
		fmt.Println("\nSaving index...")
		saveStart := time.Now()
		saveCtx := context.Background()
		if err := idxr.Save(saveCtx); err != nil {
			logger.Warn("failed to save index", zap.Error(err))
		}
		fmt.Printf("Saving index took: %v\n", time.Since(saveStart).Round(time.Millisecond))
	} else {
		fmt.Println("\nSaving index skipped (no new documents indexed)")
	}

	// Print final statistics
	fmt.Println("\n=== Crawl Complete ===")
	fmt.Printf("%s\n", crawlr.GetStatsString())
	fmt.Printf("\nDocuments indexed: %d\n", idxr.DocumentCount())
	fmt.Printf("Terms in index: %d\n", idxr.TermCount())
	fmt.Printf("Total time: %v\n", time.Since(totalStart).Round(time.Millisecond))

	return crawlErr
}

// determineSeeds determines which seed URLs to use based on flags and args.
func determineSeeds(args []string, seedSet string, seedsFile string) []string {
	// Priority: args > seed-set flag > seeds-file
	if len(args) > 0 {
		return args
	}

	// If seed set is specified, use it
	if seedSet != "" {
		set, err := crawler.GetSeedSet(crawler.SeedSetType(seedSet))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading seed set: %v\n", err)
			return nil
		}
		return set.URLs
	}

	// If seeds file is specified, load it
	if seedsFile != "" {
		config, err := crawler.LoadSeedConfig(seedsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading seed config: %v\n", err)
			return nil
		}

		// Use default set if specified
		if config.Default != "" {
			set, err := crawler.GetDefaultSeedFromConfig(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return nil
			}
			return set.URLs
		}

		// Otherwise, use first set in config
		for _, set := range config.Sets {
			return set.URLs
		}
	}

	// No seeds determined
	return nil
}

// listAvailableSeedSets prints all available predefined seed sets.
func listAvailableSeedSets(_ *cobra.Command) error {
	fmt.Println("Available predefined seed sets:")
	fmt.Println()

	sets := crawler.ListSeedSets()
	for _, set := range sets {
		fmt.Printf("  %s\n", set.Name)
		fmt.Printf("      %s\n", set.Description)
		fmt.Printf("      URLs: %d\n", len(set.URLs))
		fmt.Println()
	}

	fmt.Println("Usage:")
	fmt.Println("  gosearch crawl --seed-set=<name>")
	fmt.Println("  gosearch crawl --seeds-file=<path> --seed-set=<name>")
	fmt.Println()
	fmt.Println("Available seed sets: general, programming, academic")
	fmt.Println()
	fmt.Println("Or provide URLs directly:")
	fmt.Println("  gosearch crawl https://example.com https://github.com")

	return nil
}

func init() {
	rootCmd.AddCommand(crawlCmd)

	crawlCmd.Flags().IntP("depth", "L", 3, "maximum crawl depth from seed URLs")
	crawlCmd.Flags().IntP("workers", "w", 10, "number of concurrent crawler workers")
	crawlCmd.Flags().IntP("max-queue", "q", 0, "maximum number of URLs to queue (0 = unlimited)")
	crawlCmd.Flags().IntP("delay", "d", 1000, "delay between requests in milliseconds")
	crawlCmd.Flags().StringSliceP("allow", "a", nil, "allowed URL prefixes (default: all)")
	crawlCmd.Flags().StringSliceP("disallow", "x", nil, "disallowed URL prefixes")
	crawlCmd.Flags().StringP("seed-set", "s", "", "predefined seed set to use (general, programming, academic)")
	crawlCmd.Flags().StringP("seeds-file", "f", "", "path to custom seed sets configuration file")
	crawlCmd.Flags().StringP("strategy", "S", "bfs", "crawling strategy (bfs, best-first)")
	crawlCmd.Flags().Bool("full-reindex", false, "reindex all stored pages instead of incremental mode")
}

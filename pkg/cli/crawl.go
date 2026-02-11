package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
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
Respects robots.txt, applies rate limiting, and builds the inverted index.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runCrawl,
}

func runCrawl(cmd *cobra.Command, args []string) error {
	// Get flags
	maxDepth, _ := cmd.Flags().GetInt("depth")
	maxWorkers, _ := cmd.Flags().GetInt("workers")
	delay, _ := cmd.Flags().GetInt("delay")
	allowed, _ := cmd.Flags().GetStringSlice("allow")
	disallowed, _ := cmd.Flags().GetStringSlice("disallow")
	dataDir := viper.GetString("data-dir")

	// Derive paths from data directory
	indexPath := dataDir + "/index"
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

	// Configure crawler
	config := &crawler.Config{
		MaxWorkers:      maxWorkers,
		MaxDepth:        maxDepth,
		Delay:           time.Duration(delay) * time.Millisecond,
		AllowedDomains:  allowed,
		DisallowedPaths: disallowed,
		UserAgent:       "GoSearch/1.0 (+https://github.com/abuiliazeed/gosearch)",
		Timeout:         30 * time.Second,
		RespectRobots:   true,
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
	crawlDone := make(chan struct{})

	crawlWg.Add(1)
	go func() {
		defer crawlWg.Done()
		fmt.Printf("Starting crawl...\n")
		fmt.Printf("  Seed URLs: %v\n", args)
		fmt.Printf("  Max depth: %d\n", maxDepth)
		fmt.Printf("  Max workers: %d\n", maxWorkers)
		fmt.Printf("  Delay: %dms\n", delay)
		fmt.Printf("  Pages path: %s\n", pagesPath)
		fmt.Printf("  Index path: %s\n", indexPath)
		fmt.Println()

		crawlErr = crawlr.Start(ctx, args)
		close(crawlDone)
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

	// Index all crawled documents (use fresh context since original may be canceled)
	fmt.Println("\nIndexing documents...")
	indexCtx := context.Background()
	docIDs, err := docStore.List()
	if err != nil {
		logger.Warn("failed to list documents for indexing", zap.Error(err))
	} else {
		var indexErrors []error
		for i, docID := range docIDs {
			select {
			case <-indexCtx.Done():
				break
			default:
			}

			doc, err := docStore.Get(docID)
			if err != nil {
				logger.Warn("failed to get document",
					zap.String("doc_id", docID),
					zap.Error(err))
				continue
			}

			// Extract text content from HTML (simplified - just use title for now)
			if doc.Content == "" {
				doc.Content = doc.Title
			}

			if err := idxr.IndexDocument(indexCtx, doc); err != nil {
				indexErrors = append(indexErrors, fmt.Errorf("index error for %s: %w", doc.URL, err))
			}

			if i%10 == 0 {
				fmt.Printf("\rIndexed: %d/%d", i+1, len(docIDs))
			}
		}
		fmt.Printf("\rIndexed: %d/%d\n", len(docIDs), len(docIDs))

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
	}

	// Save index (use fresh context since original may be canceled)
	fmt.Println("\nSaving index...")
	saveCtx := context.Background()
	if err := idxr.Save(saveCtx); err != nil {
		logger.Warn("failed to save index", zap.Error(err))
	}

	// Print final statistics
	fmt.Println("\n=== Crawl Complete ===")
	fmt.Printf("%s\n", crawlr.GetStatsString())
	fmt.Printf("\nDocuments indexed: %d\n", idxr.DocumentCount())
	fmt.Printf("Terms in index: %d\n", idxr.TermCount())

	return crawlErr
}

func init() {
	rootCmd.AddCommand(crawlCmd)

	crawlCmd.Flags().IntP("depth", "L", 3, "maximum crawl depth from seed URLs")
	crawlCmd.Flags().IntP("workers", "w", 10, "number of concurrent crawler workers")
	crawlCmd.Flags().IntP("delay", "d", 1000, "delay between requests in milliseconds")
	crawlCmd.Flags().StringSliceP("allow", "a", nil, "allowed URL prefixes (default: all)")
	crawlCmd.Flags().StringSliceP("disallow", "x", nil, "disallowed URL prefixes")
}

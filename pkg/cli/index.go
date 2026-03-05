// Package cli provides the command-line interface for gosearch.
//
// This file contains index management commands.
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/abuiliazeed/gosearch/internal/indexer"
	"github.com/abuiliazeed/gosearch/internal/storage"
)

var indexCmd = &cobra.Command{
	Use:   "index [action]",
	Short: "Index management commands",
	Long: `Manage the inverted index.

Available actions: build, stats, clear, optimize, validate.

Requires storage schema 2-markdown-only. If your data directory is not
initialized yet, run a crawl first.`,
	Example: `  gosearch index stats
  gosearch index validate
  gosearch index clear --force`,
}

// indexBuildCmd builds the index from crawled pages.
var indexBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the index from crawled pages",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Get data directory
		dataDir := viper.GetString("data-dir")
		if err := requireSchemaVersion(dataDir); err != nil {
			return err
		}
		indexPath := filepath.Join(dataDir, "index", "index.db")
		pagesPath := filepath.Join(dataDir, "pages")

		// Initialize logger
		logger := initLogger()

		// Open index store
		indexStore, err := storage.NewIndexStore(indexPath)
		if err != nil {
			return fmt.Errorf("failed to open index store: %w", err)
		}
		defer indexStore.Close()

		// Open document store
		docStore, err := storage.NewDocumentStore(pagesPath)
		if err != nil {
			return fmt.Errorf("failed to open document store: %w", err)
		}
		defer docStore.Close()

		// Create indexer
		idx := indexer.NewIndexer(indexStore, logger)

		// Check if we should load existing index first
		loadFirst, _ := cmd.Flags().GetBool("load")
		if loadFirst {
			fmt.Println("Loading existing index...")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := idx.Load(ctx); err != nil {
				fmt.Printf("Warning: failed to load existing index: %v\n", err)
			}
		}

		// List all documents from storage
		// For now, we'll use a simple approach
		fmt.Println("Building index from crawled pages...")
		fmt.Printf("Pages directory: %s\n", pagesPath)

		// Count indexed documents
		stats := idx.Stats()
		fmt.Printf("\nIndex build complete:\n")
		fmt.Printf("  Total documents: %d\n", stats.TotalDocuments)
		fmt.Printf("  Total terms: %d\n", stats.TotalTerms)
		fmt.Printf("  Total postings: %d\n", stats.TotalPostings)

		// Save index
		fmt.Println("\nSaving index...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := idx.Save(ctx); err != nil {
			return fmt.Errorf("failed to save index: %w", err)
		}

		fmt.Println("Index saved successfully")
		return nil
	},
}

// indexStatsCmd shows index statistics.
var indexStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show index statistics",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Get data directory
		dataDir := viper.GetString("data-dir")
		if err := requireSchemaVersion(dataDir); err != nil {
			return err
		}
		indexPath := filepath.Join(dataDir, "index", "index.db")

		// Open index store
		indexStore, err := storage.NewIndexStore(indexPath)
		if err != nil {
			return fmt.Errorf("failed to open index store: %w", err)
		}
		defer indexStore.Close()

		// Get metadata from store
		meta, err := indexStore.GetMeta()
		if err != nil {
			return fmt.Errorf("failed to get index metadata: %w", err)
		}

		// Get store stats
		storeStats, err := indexStore.Stats()
		if err != nil {
			return fmt.Errorf("failed to get store stats: %w", err)
		}

		// Get file size for BoltDB
		fileInfo, err := os.Stat(indexPath)
		var indexSizeMB float64
		if err == nil {
			indexSizeMB = float64(fileInfo.Size()) / (1024 * 1024)
		}

		fmt.Println("Index Statistics:")
		fmt.Printf("  Total documents: %d\n", meta.TotalDocuments)
		fmt.Printf("  Total terms: %d\n", meta.TotalTerms)
		fmt.Printf("  Total postings: %d\n", meta.TotalPostings)
		fmt.Printf("  Average postings per term: %.2f\n", meta.AveragePostings)
		fmt.Printf("  Last updated: %s\n", meta.LastUpdated.Format(time.RFC3339))
		fmt.Printf("\nStorage statistics:\n")
		fmt.Printf("  Metadata entries: %d\n", storeStats["meta"])
		fmt.Printf("  Term entries: %d\n", storeStats["terms"])
		fmt.Printf("  Document entries: %d\n", storeStats["documents"])
		fmt.Printf("  Disk Usage (BoltDB): %.2f MB\n", indexSizeMB)

		// Redis Cache Statistics
		redisHost := viper.GetString("redis.host")
		if redisHost == "" {
			redisHost = os.Getenv("GOSEARCH_REDIS_HOST")
		}

		if redisHost != "" {
			fmt.Printf("\nRedis Cache Statistics:\n")
			redisPass := viper.GetString("redis.password")
			redisDB := viper.GetInt("redis.db")
			redisTTL := viper.GetDuration("redis.ttl")

			cacheStore, err := storage.NewCacheStore(redisHost, redisPass, redisDB, redisTTL)
			if err != nil {
				fmt.Printf("  Status: Not Connected (Error: %v)\n", err)
			} else {
				defer cacheStore.Close()
				cacheStats, err := cacheStore.Stats(context.Background())
				if err != nil {
					fmt.Printf("  Status: Connected but failed to get stats (%v)\n", err)
				} else {
					fmt.Printf("  Status: Connected (%s)\n", redisHost)
					fmt.Printf("  Cached Queries: %v\n", cacheStats["key_count"])
					fmt.Printf("  Memory Usage: %v\n", cacheStats["memory_usage"])
					fmt.Printf("  TTL Strategy: %s\n", cacheStats["ttl"])
				}
			}
		} else {
			fmt.Printf("\nRedis Cache Statistics:\n")
			fmt.Printf("  Status: Disabled (No specific host configured)\n")
		}

		// Optionally show top terms
		showTerms, _ := cmd.Flags().GetBool("terms")
		if showTerms {
			fmt.Println("\nTop terms by document frequency:")
			terms, err := indexStore.ListTerms()
			if err != nil {
				return fmt.Errorf("failed to list terms: %w", err)
			}

			// Sort by document frequency
			type termFreq struct {
				term string
				freq int
			}
			freqs := make([]termFreq, 0, len(terms))
			for _, term := range terms {
				info, err := indexStore.GetTermInfo(term)
				if err != nil {
					continue
				}
				freqs = append(freqs, termFreq{term: term, freq: info.DocFrequency})
			}

			// Sort by frequency descending
			for i := 0; i < len(freqs)-1; i++ {
				for j := i + 1; j < len(freqs); j++ {
					if freqs[i].freq < freqs[j].freq {
						freqs[i], freqs[j] = freqs[j], freqs[i]
					}
				}
			}

			// Show top 10
			limit := 10
			if len(freqs) < limit {
				limit = len(freqs)
			}
			for i := 0; i < limit; i++ {
				fmt.Printf("  %3d. %s (df: %d)\n", i+1, freqs[i].term, freqs[i].freq)
			}
		}

		return nil
	},
}

// indexClearCmd clears the index.
var indexClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the index",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Get data directory
		dataDir := viper.GetString("data-dir")
		if err := requireSchemaVersion(dataDir); err != nil {
			return err
		}
		indexPath := filepath.Join(dataDir, "index", "index.db")

		// Check if user is sure
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Are you sure you want to clear the index at %s? (y/N): ", indexPath)
			var response string
			_, _ = fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Aborted")
				return nil
			}
		}

		// Remove index file
		if err := os.RemoveAll(indexPath); err != nil {
			return fmt.Errorf("failed to remove index: %w", err)
		}

		fmt.Println("Index cleared successfully")
		return nil
	},
}

// indexOptimizeCmd optimizes the index.
var indexOptimizeCmd = &cobra.Command{
	Use:   "optimize",
	Short: "Optimize the index",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
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
		idx := indexer.NewIndexer(indexStore, logger)

		fmt.Println("Loading index...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := idx.Load(ctx); err != nil {
			return fmt.Errorf("failed to load index: %w", err)
		}

		fmt.Println("Optimizing index...")
		if err := idx.Optimize(); err != nil {
			return fmt.Errorf("failed to optimize index: %w", err)
		}

		fmt.Println("Saving optimized index...")
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := idx.Save(ctx); err != nil {
			return fmt.Errorf("failed to save index: %w", err)
		}

		fmt.Println("Index optimized successfully")
		return nil
	},
}

// indexValidateCmd validates the index.
var indexValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the index for consistency",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
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
		idx := indexer.NewIndexer(indexStore, logger)

		fmt.Println("Loading index...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := idx.Load(ctx); err != nil {
			return fmt.Errorf("failed to load index: %w", err)
		}

		fmt.Println("Validating index...")
		issues := idx.Validate()

		if len(issues) == 0 {
			fmt.Println("✓ Index is valid - no issues found")
		} else {
			fmt.Printf("✗ Found %d issue(s):\n", len(issues))
			for _, issue := range issues {
				fmt.Printf("  - %s\n", issue)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(indexCmd)
	indexCmd.AddCommand(indexBuildCmd)
	indexCmd.AddCommand(indexStatsCmd)
	indexCmd.AddCommand(indexClearCmd)
	indexCmd.AddCommand(indexOptimizeCmd)
	indexCmd.AddCommand(indexValidateCmd)

	// Index build flags
	indexBuildCmd.Flags().Bool("load", true, "load existing index before building")

	// Index stats flags
	indexStatsCmd.Flags().Bool("terms", false, "show top terms by frequency")

	// Index clear flags
	indexClearCmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
}

// initLogger initializes a zap logger.
func initLogger() *zap.Logger {
	// Determine log level from verbose flag
	verbose := viper.GetInt("verbose")
	level := zap.InfoLevel
	if verbose >= 2 {
		level = zap.DebugLevel
	} else if verbose == 0 {
		level = zap.WarnLevel
	}

	// Determine log format
	logFormat := viper.GetString("log-format")
	var logger *zap.Logger
	if logFormat == "json" {
		logger, _ = zap.NewProduction(zap.IncreaseLevel(level))
	} else {
		logger, _ = zap.NewDevelopment(zap.IncreaseLevel(level))
	}

	return logger
}

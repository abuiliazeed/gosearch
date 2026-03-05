// Package cli provides the command-line interface for gosearch.
//
// It uses Cobra for command parsing and Viper for configuration management.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/abuiliazeed/gosearch/internal/indexer"
	"github.com/abuiliazeed/gosearch/internal/ranker"
	"github.com/abuiliazeed/gosearch/internal/search"
	"github.com/abuiliazeed/gosearch/internal/server"
	"github.com/abuiliazeed/gosearch/internal/storage"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server",
	Long: `Start the HTTP API server for search and index management.

The server provides RESTful API endpoints for searching the index,
retrieving statistics, and managing the index.

Requires storage schema 2-markdown-only. If your data directory is not
initialized yet, run a crawl first.

API Endpoints:
  GET  /api/v1/search?q=query        - Search the index
  GET  /api/v1/stats                 - Get index statistics
  POST /api/v1/index/rebuild         - Rebuild the index
  GET  /health                       - Health check`,
	Example: `  # Start API server on default host/port
  gosearch serve

  # Custom host/port
  gosearch serve --host 0.0.0.0 --port 8080`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringP("host", "H", "127.0.0.1", "host to bind to")
	serveCmd.Flags().IntP("port", "p", 8080, "port to listen on")
	serveCmd.Flags().Duration("read-timeout", 30*time.Second, "read timeout")
	serveCmd.Flags().Duration("write-timeout", 30*time.Second, "write timeout")
	serveCmd.Flags().Duration("idle-timeout", 120*time.Second, "idle timeout")

	// Bind flags to viper
	_ = viper.BindPFlag("serve.host", serveCmd.Flags().Lookup("host"))
	_ = viper.BindPFlag("serve.port", serveCmd.Flags().Lookup("port"))
	_ = viper.BindPFlag("serve.read-timeout", serveCmd.Flags().Lookup("read-timeout"))
	_ = viper.BindPFlag("serve.write-timeout", serveCmd.Flags().Lookup("write-timeout"))
	_ = viper.BindPFlag("serve.idle-timeout", serveCmd.Flags().Lookup("idle-timeout"))
}

func runServe(cmd *cobra.Command, args []string) error {
	// Get configuration
	dataDir := viper.GetString("data-dir")
	if err := requireSchemaVersion(dataDir); err != nil {
		return err
	}
	host := viper.GetString("serve.host")
	port := viper.GetInt("serve.port")

	// Get CORS configuration
	corsOrigins := viper.GetString("cors-allowed-origins")
	corsMethods := viper.GetString("cors-allowed-methods")
	corsHeaders := viper.GetString("cors-allowed-headers")

	// Initialize components
	fmt.Println("Initializing gosearch server...")

	// Create index store
	indexPath := viper.GetString("index-path")
	if indexPath == "" {
		indexPath = dataDir + "/index"
	} else {
		// indexPath already set via config
		_ = indexPath // Explicitly mark as used to satisfy staticcheck
	}

	// Get timeout values
	readTimeout := viper.GetDuration("serve.read-timeout")
	writeTimeout := viper.GetDuration("serve.write-timeout")
	idleTimeout := viper.GetDuration("serve.idle-timeout")

	fmt.Printf("Server will listen on %s:%d\n", host, port)
	fmt.Println("\nAPI Endpoints:")
	fmt.Printf("  GET  http://%s:%d/api/v1/search?q=query\n", host, port)
	fmt.Printf("  GET  http://%s:%d/api/v1/stats\n", host, port)
	fmt.Printf("  POST http://%s:%d/api/v1/index/rebuild\n", host, port)
	fmt.Printf("  GET  http://%s:%d/health\n", host, port)
	fmt.Println("\nPress Ctrl+C to stop the server")

	return serveMain(host, port, readTimeout, writeTimeout, idleTimeout, dataDir, corsOrigins, corsMethods, corsHeaders)
}

// serveMain is the actual server implementation (to be integrated).
//
//nolint:unused // Intentionally unused - placeholder for future implementation
func serveMain(host string, port int, _, _, _ time.Duration, dataDir string, corsOrigins, corsMethods, corsHeaders string) error {
	// Create server config
	config := &server.Config{
		Host:               host,
		Port:               port,
		CORSAllowedOrigins: corsOrigins,
		CORSAllowedMethods: corsMethods,
		CORSAllowedHeaders: corsHeaders,
	}

	// Initialize components
	indexStore, err := storage.NewIndexStore(dataDir + "/index/index.db")
	if err != nil {
		return fmt.Errorf("failed to create index store: %w", err)
	}
	defer indexStore.Close()

	docStore, err := storage.NewDocumentStore(dataDir + "/pages")
	if err != nil {
		return fmt.Errorf("failed to create document store: %w", err)
	}

	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()

	tokenizer := indexer.NewTokenizer(indexer.DefaultTokenizerConfig())
	indexerInstance := indexer.NewIndexerWithTokenizer(indexStore, tokenizer, logger)

	// Get the inverted index from the indexer
	// Note: indexer.GetIndex() returns *indexer.Index, we need to get the InvertedIndex
	threadSafeIndex := indexerInstance.GetIndex()
	indexData := threadSafeIndex.GetIndex()

	// Create TF-IDF and PageRank for scorer
	tfidf := ranker.NewTFIDF(indexData)
	pagerank := ranker.DefaultPageRank() // Uses default parameters: damping=0.85, iterations=100, tolerance=1e-6
	rankerConfig := ranker.DefaultScorerConfig()
	rankerInstance := ranker.NewScorer(tfidf, pagerank, rankerConfig)

	cacheStore, err := storage.NewCacheStore("localhost:6379", "", 0, 1*time.Hour)
	if err != nil {
		logger.Warn("failed to connect to Redis, caching disabled", zap.Error(err))
		cacheStore = nil
	}

	searchConfig := search.DefaultConfig()
	searcherInstance := search.NewSearcher(threadSafeIndex, rankerInstance, cacheStore, searchConfig)

	// Create server
	srv := server.NewServer(config, indexerInstance, searcherInstance, docStore)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return err
	case <-sigChan:
		fmt.Println("\nShutting down gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}

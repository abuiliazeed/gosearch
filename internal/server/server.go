// Package server provides HTTP API server functionality for gosearch.
//
// It includes a RESTful API for search, index management, and statistics.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/abuiliazeed/gosearch/internal/indexer"
	"github.com/abuiliazeed/gosearch/internal/search"
	"github.com/abuiliazeed/gosearch/internal/storage"
)

// Server represents the HTTP API server.
type Server struct {
	config   *Config
	handlers *Handlers
	server   *http.Server
}

// NewServer creates a new HTTP server.
func NewServer(
	config *Config,
	indexer *indexer.Indexer,
	searcher *search.Searcher,
	docStore *storage.DocumentStore,
) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	handlers := NewHandlers(indexer, searcher, nil, docStore)

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/v1/search", handlers.HandleSearch)
	mux.HandleFunc("/api/v1/stats", handlers.HandleStats)
	mux.HandleFunc("/api/v1/index/rebuild", handlers.HandleIndexRebuild)

	// Health check
	mux.HandleFunc("/health", handlers.HandleHealth)

	// 404 handler
	mux.HandleFunc("/", handlers.HandleNotFound)

	// Apply middleware
	var handler http.Handler = mux
	handler = CORSMiddleware(handler)
	handler = JSONMiddleware(handler)
	handler = LoggingMiddleware(handler)
	handler = RecoveryMiddleware(handler)

	return &Server{
		config:   config,
		handlers: handlers,
		server: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
			Handler:      handler,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
			IdleTimeout:  config.IdleTimeout,
		},
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	log.Printf("Starting server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	return s.server.Shutdown(ctx)
}

// Address returns the server address.
func (s *Server) Address() string {
	return s.server.Addr
}

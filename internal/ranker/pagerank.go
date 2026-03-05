// Package ranker provides document ranking algorithms for gosearch.
//
// It includes TF-IDF scoring, PageRank computation, and combined
// scoring with boost factors for title, URL, and freshness.
package ranker

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/abuiliazeed/gosearch/internal/indexer"
)

// PageRank computes link-based page importance scores.
// Uses the iterative PageRank algorithm to rank documents based on
// the number and quality of incoming links.
type PageRank struct {
	mu          sync.RWMutex
	scores      map[string]float64 // DocID -> PageRank score
	damping     float64            // Damping factor (typically 0.85)
	iterations  int                // Number of iterations for convergence
	tolerance   float64            // Convergence tolerance
	initialized bool
}

// NewPageRank creates a new PageRank scorer.
// damping is the probability of continuing to follow links (typically 0.85).
// iterations is the maximum number of iterations to perform.
// tolerance is the minimum change in scores to continue iterating.
func NewPageRank(damping float64, iterations int, tolerance float64) *PageRank {
	return &PageRank{
		scores:     make(map[string]float64),
		damping:    damping,
		iterations: iterations,
		tolerance:  tolerance,
	}
}

// DefaultPageRank creates a PageRank scorer with default parameters.
// damping=0.85, iterations=100, tolerance=1e-6.
func DefaultPageRank() *PageRank {
	return NewPageRank(0.85, 100, 1e-6)
}

// LinkGraph represents a directed graph of documents and their links.
type LinkGraph struct {
	links   map[string][]string // DocID -> []OutgoingLinkDocIDs
	inLinks map[string][]string // DocID -> []IncomingLinkDocIDs
	nodes   map[string]bool     // All document IDs in the graph
}

// NewLinkGraph creates a new empty link graph.
func NewLinkGraph() *LinkGraph {
	return &LinkGraph{
		links:   make(map[string][]string),
		inLinks: make(map[string][]string),
		nodes:   make(map[string]bool),
	}
}

// AddNode adds a document node to the graph.
func (g *LinkGraph) AddNode(docID string) {
	g.nodes[docID] = true
	// Initialize empty slices if not present
	if _, exists := g.links[docID]; !exists {
		g.links[docID] = []string{}
	}
	if _, exists := g.inLinks[docID]; !exists {
		g.inLinks[docID] = []string{}
	}
}

// AddLink adds a directed link from source to target.
func (g *LinkGraph) AddLink(source, target string) {
	// Add both nodes
	g.AddNode(source)
	g.AddNode(target)

	// Add outgoing link
	g.links[source] = append(g.links[source], target)

	// Add incoming link
	g.inLinks[target] = append(g.inLinks[target], source)
}

// OutLinks returns the outgoing links for a document.
func (g *LinkGraph) OutLinks(docID string) []string {
	if links, exists := g.links[docID]; exists {
		return links
	}
	return []string{}
}

// InLinks returns the incoming links for a document.
func (g *LinkGraph) InLinks(docID string) []string {
	if links, exists := g.inLinks[docID]; exists {
		return links
	}
	return []string{}
}

// NodeCount returns the total number of nodes in the graph.
func (g *LinkGraph) NodeCount() int {
	return len(g.nodes)
}

// AllNodes returns all document IDs in the graph.
func (g *LinkGraph) AllNodes() []string {
	nodes := make([]string, 0, len(g.nodes))
	for node := range g.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// BuildFromIndex builds a link graph from an inverted index.
// Extracts links from document URLs and assumes links exist between
// documents based on URL patterns.
//
// The ctx parameter controls cancellation.
func (g *LinkGraph) BuildFromIndex(ctx context.Context, idx *indexer.InvertedIndex) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Get all documents
	docs := idx.GetDocuments()

	// Build a URL -> DocID mapping
	urlToDocID := make(map[string]string)
	for docID, docInfo := range docs {
		urlToDocID[docInfo.URL] = docID
		g.AddNode(docID)
	}

	// For each document, extract and add links
	for sourceDocID := range docs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// In a real implementation, you would parse the document content
		// to extract actual links and link them to target DocIDs.
		// For now, we store the node in the graph.
		// Use the BuildWithLinks method to provide explicit link data.
		_ = sourceDocID // Reserved for future link extraction implementation
	}

	return nil
}

// BuildWithLinks builds a link graph from explicit link data.
// linkData is a map of source DocID to slice of target DocIDs.
func (g *LinkGraph) BuildWithLinks(ctx context.Context, linkData map[string][]string) error {
	for source, targets := range linkData {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		for _, target := range targets {
			g.AddLink(source, target)
		}
	}
	return nil
}

// Compute computes PageRank scores for the link graph.
// The ctx parameter controls cancellation.
//
// The algorithm iteratively computes:
// PR(u) = (1-d)/N + d * sum(PR(v)/outlinks(v) for v in inlinks(u))
// where d is the damping factor and N is the total number of nodes.
func (pr *PageRank) Compute(ctx context.Context, graph *LinkGraph) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	nodes := graph.AllNodes()
	nodeCount := len(nodes)

	if nodeCount == 0 {
		return fmt.Errorf("cannot compute PageRank: empty graph")
	}

	// Initialize scores: PR = 1/N for all nodes
	pr.scores = make(map[string]float64, nodeCount)
	initialScore := 1.0 / float64(nodeCount)
	for _, node := range nodes {
		pr.scores[node] = initialScore
	}

	// Iterative computation
	for iteration := 0; iteration < pr.iterations; iteration++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		newScores := make(map[string]float64, nodeCount)
		maxChange := 0.0

		// Compute new scores
		for _, node := range nodes {
			score := (1 - pr.damping) / float64(nodeCount)

			// Add contribution from incoming links
			inLinks := graph.InLinks(node)
			for _, inLink := range inLinks {
				outLinks := graph.OutLinks(inLink)
				if len(outLinks) > 0 {
					score += pr.damping * (pr.scores[inLink] / float64(len(outLinks)))
				}
			}

			newScores[node] = score

			// Track maximum change for convergence check
			change := math.Abs(newScores[node] - pr.scores[node])
			if change > maxChange {
				maxChange = change
			}
		}

		// Update scores
		pr.scores = newScores

		// Check for convergence
		if maxChange < pr.tolerance {
			break
		}
	}

	pr.initialized = true
	return nil
}

// GetScore returns the PageRank score for a document.
// Returns 0 if the document has no score.
func (pr *PageRank) GetScore(docID string) float64 {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	if score, exists := pr.scores[docID]; exists {
		return score
	}
	return 0
}

// GetScores returns a copy of all PageRank scores.
func (pr *PageRank) GetScores() map[string]float64 {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	scores := make(map[string]float64, len(pr.scores))
	for docID, score := range pr.scores {
		scores[docID] = score
	}
	return scores
}

// Normalize scales all PageRank scores to sum to 1.
// This is useful after computing scores to ensure proper probability distribution.
func (pr *PageRank) Normalize() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	sum := 0.0
	for _, score := range pr.scores {
		sum += score
	}

	if sum > 0 {
		for docID := range pr.scores {
			pr.scores[docID] /= sum
		}
	}
}

// ScaleBy scales all PageRank scores by a constant factor.
func (pr *PageRank) ScaleBy(factor float64) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	for docID := range pr.scores {
		pr.scores[docID] *= factor
	}
}

// IsInitialized returns true if PageRank scores have been computed.
func (pr *PageRank) IsInitialized() bool {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	return pr.initialized
}

// Reset clears all computed scores and resets the initialized state.
func (pr *PageRank) Reset() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.scores = make(map[string]float64)
	pr.initialized = false
}

// TopN returns the top N documents by PageRank score.
// Returns a slice of DocIDs sorted by score in descending order.
func (pr *PageRank) TopN(n int) []string {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	// Create a sortable slice
	type docScore struct {
		docID string
		score float64
	}

	docScores := make([]docScore, 0, len(pr.scores))
	for docID, score := range pr.scores {
		docScores = append(docScores, docScore{docID, score})
	}

	// Sort by score (simple bubble sort for simplicity)
	// In production, use sort.Slice or a more efficient algorithm
	for i := 0; i < len(docScores)-1; i++ {
		for j := i + 1; j < len(docScores); j++ {
			if docScores[i].score < docScores[j].score {
				docScores[i], docScores[j] = docScores[j], docScores[i]
			}
		}
	}

	// Get top N
	result := make([]string, 0, n)
	for i := 0; i < n && i < len(docScores); i++ {
		result = append(result, docScores[i].docID)
	}

	return result
}

// BoostScores applies a boost factor to documents based on their PageRank scores.
// This can be used to combine PageRank with other scoring methods.
// The formula is: baseScore * (1 + pageRankBoost * pageRankScore)
func (pr *PageRank) BoostScores(baseScores map[string]float64, boostFactor float64) map[string]float64 {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	boostedScores := make(map[string]float64, len(baseScores))

	// Find maximum PageRank score for normalization
	maxPR := 0.0
	for _, score := range pr.scores {
		if score > maxPR {
			maxPR = score
		}
	}

	for docID, baseScore := range baseScores {
		prScore := pr.GetScore(docID)
		if maxPR > 0 {
			// Normalize PageRank to [0, 1] and apply boost
			normalizedPR := prScore / maxPR
			boostedScores[docID] = baseScore * (1 + boostFactor*normalizedPR)
		} else {
			boostedScores[docID] = baseScore
		}
	}

	return boostedScores
}

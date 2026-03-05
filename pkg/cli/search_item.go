package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type searchItemOutput struct {
	Query           string    `json:"query"`
	Rank            int       `json:"rank"`
	TotalResults    int       `json:"total_results"`
	DocID           string    `json:"doc_id"`
	URL             string    `json:"url"`
	Title           string    `json:"title"`
	Score           float64   `json:"score"`
	Snippet         string    `json:"snippet,omitempty"`
	Links           []string  `json:"links"`
	ContentMarkdown string    `json:"content_markdown"`
	CrawledAt       time.Time `json:"crawled_at,omitempty"`
	Depth           int       `json:"depth"`
}

var searchItemCmd = &cobra.Command{
	Use:   "search-item [query]",
	Short: "Return one ranked search result with full stored markdown content as JSON",
	Long: `Searches the index, picks a ranked result, and returns an agent-friendly JSON payload
containing URL, title, links, and full stored markdown content for that result.

Output fields:
  - query: input query string
  - rank: selected 1-based result rank
  - total_results: number of matched documents
  - doc_id: internal document ID
  - url: document URL
  - title: document title
  - score: ranking score
  - snippet: short preview
  - links: extracted outgoing links from the page
  - content_markdown: full cleaned markdown content
  - crawled_at: crawl timestamp
  - depth: crawl depth from seed`,
	Example: `  # Get top ranked item
  gosearch search-item "sourcebeauty" --rank 1 --no-cache

  # Get third ranked item
  gosearch search-item "retinol" --rank 3`,
	Args: cobra.ExactArgs(1),
	RunE: runSearchItem,
}

func runSearchItem(cmd *cobra.Command, args []string) error {
	rankPos, _ := cmd.Flags().GetInt("rank")
	noCache, _ := cmd.Flags().GetBool("no-cache")
	resolved, err := fetchRankedSearchDocument(args[0], rankPos, noCache)
	if err != nil {
		return err
	}

	output := searchItemOutput{
		Query:           resolved.Query,
		Rank:            resolved.Rank,
		TotalResults:    resolved.TotalResults,
		DocID:           resolved.Result.DocID,
		URL:             resolved.Document.URL,
		Title:           resolved.Document.Title,
		Score:           resolved.Result.Score,
		Snippet:         resolved.Result.Snippet,
		Links:           resolved.Document.Links,
		ContentMarkdown: resolved.Document.ContentMarkdown,
		CrawledAt:       resolved.Document.CrawledAt,
		Depth:           resolved.Document.Depth,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode output: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(searchItemCmd)

	searchItemCmd.Flags().IntP("rank", "r", 1, "1-based rank position to fetch from search results")
	searchItemCmd.Flags().Bool("no-cache", false, "disable query result caching")
}

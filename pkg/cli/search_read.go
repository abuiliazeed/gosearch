package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
)

var searchReadCmd = &cobra.Command{
	Use:   "search-read [query]",
	Short: "Render one ranked search result as terminal-friendly markdown",
	Long: `Searches the index, picks a ranked result, and prints a human-readable
terminal view of the document metadata and rendered markdown content.

This command is optimized for comfortable reading in terminal sessions,
while search-item remains optimized for JSON/agent consumption.`,
	Example: `  # Render top result for a query
  gosearch search-read "sourcebeauty" --rank 1 --no-cache

  # Render with specific style and width
  gosearch search-read "retinol" --rank 2 --style dark --width 100`,
	Args: cobra.ExactArgs(1),
	RunE: runSearchRead,
}

func runSearchRead(cmd *cobra.Command, args []string) error {
	rankPos, _ := cmd.Flags().GetInt("rank")
	noCache, _ := cmd.Flags().GetBool("no-cache")
	style, _ := cmd.Flags().GetString("style")
	width, _ := cmd.Flags().GetInt("width")
	noLinks, _ := cmd.Flags().GetBool("no-links")
	out := cmd.OutOrStdout()

	resolved, err := fetchRankedSearchDocument(args[0], rankPos, noCache)
	if err != nil {
		return err
	}

	rendered, err := renderMarkdownForTerminal(resolved.Document.ContentMarkdown, style, width)
	if err != nil {
		return fmt.Errorf("failed to render markdown: %w", err)
	}

	fmt.Fprintf(out, "Query: %s\n", resolved.Query)
	fmt.Fprintf(out, "Result: %d/%d\n", resolved.Rank, resolved.TotalResults)
	fmt.Fprintf(out, "Score: %.2f\n", resolved.Result.Score)
	fmt.Fprintf(out, "Title: %s\n", resolved.Document.Title)
	fmt.Fprintf(out, "URL: %s\n", resolved.Document.URL)
	fmt.Fprintf(out, "Depth: %d\n", resolved.Document.Depth)
	if !resolved.Document.CrawledAt.IsZero() {
		fmt.Fprintf(out, "Crawled At: %s\n", resolved.Document.CrawledAt.Format("2006-01-02 15:04:05 MST"))
	}
	if !noLinks {
		fmt.Fprintf(out, "Links: %d\n", len(resolved.Document.Links))
		for _, link := range resolved.Document.Links {
			fmt.Fprintf(out, "  - %s\n", link)
		}
	}

	fmt.Fprintln(out, strings.Repeat("=", 80))
	fmt.Fprint(out, rendered)
	if !strings.HasSuffix(rendered, "\n") {
		fmt.Fprintln(out)
	}

	return nil
}

func renderMarkdownForTerminal(markdown string, style string, width int) (string, error) {
	if strings.TrimSpace(markdown) == "" {
		return "(no markdown content stored for this document)\n", nil
	}

	opts := make([]glamour.TermRendererOption, 0, 3)
	if width > 0 {
		opts = append(opts, glamour.WithWordWrap(width))
	}
	opts = append(opts, glamour.WithEmoji())

	if strings.TrimSpace(style) == "" || strings.EqualFold(style, "auto") {
		opts = append(opts, glamour.WithAutoStyle())
	} else {
		opts = append(opts, glamour.WithStandardStyle(style))
	}

	renderer, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		return "", err
	}

	return renderer.Render(markdown)
}

func init() {
	rootCmd.AddCommand(searchReadCmd)

	searchReadCmd.Flags().IntP("rank", "r", 1, "1-based rank position to fetch from search results")
	searchReadCmd.Flags().Bool("no-cache", false, "disable query result caching")
	searchReadCmd.Flags().String("style", "auto", "markdown render style (auto, dark, light, dracula, notty, ascii)")
	searchReadCmd.Flags().Int("width", 100, "word-wrap width for rendered output (0 disables wrapping)")
	searchReadCmd.Flags().Bool("no-links", false, "hide extracted outgoing links")
}

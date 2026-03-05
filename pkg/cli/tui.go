package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui [query]",
	Short: "Google-inspired interactive search TUI",
	Long: `Starts a Google-inspired terminal UI for searching indexed content:
  - top search box and clean 3-line result cards
  - page-based navigation with keyboard controls
  - in-terminal markdown preview and optional browser open`,
	Example: `  # Start TUI
  gosearch tui

  # Start TUI with initial query
  gosearch tui "sourcebeauty"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTUI,
}

func runTUI(cmd *cobra.Command, args []string) error {
	noCache, _ := cmd.Flags().GetBool("no-cache")
	limit, _ := cmd.Flags().GetInt("limit")
	style, _ := cmd.Flags().GetString("style")

	var initialQuery string
	if len(args) > 0 {
		initialQuery = strings.TrimSpace(args[0])
	}

	runtime, err := newSearchRuntime(noCache)
	if err != nil {
		return err
	}
	defer runtime.Close()

	model := newTUIModel(runtime, initialQuery, limit, style)
	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(tuiCmd)

	tuiCmd.Flags().IntP("limit", "l", 100, "maximum number of search results to load per query")
	tuiCmd.Flags().Bool("no-cache", false, "disable query result caching")
	tuiCmd.Flags().String("style", "auto", "markdown render style (auto, dark, light, dracula, notty, ascii)")
}

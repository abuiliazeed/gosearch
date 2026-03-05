package cli

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/abuiliazeed/gosearch/internal/search"
	"github.com/abuiliazeed/gosearch/internal/storage"
)

func TestPaginationHelpers(t *testing.T) {
	tests := []struct {
		name       string
		totalItems int
		pageSize   int
		page       int
		wantPages  int
		wantStart  int
		wantEnd    int
	}{
		{name: "empty", totalItems: 0, pageSize: 10, page: 0, wantPages: 1, wantStart: 0, wantEnd: 0},
		{name: "single page partial", totalItems: 7, pageSize: 10, page: 0, wantPages: 1, wantStart: 0, wantEnd: 7},
		{name: "exact multiple", totalItems: 20, pageSize: 10, page: 1, wantPages: 2, wantStart: 10, wantEnd: 20},
		{name: "partial last page", totalItems: 21, pageSize: 10, page: 2, wantPages: 3, wantStart: 20, wantEnd: 21},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			gotPages := totalPages(testCase.totalItems, testCase.pageSize)
			if gotPages != testCase.wantPages {
				t.Fatalf("expected pages %d, got %d", testCase.wantPages, gotPages)
			}

			gotStart, gotEnd := pageBounds(testCase.page, testCase.pageSize, testCase.totalItems)
			if gotStart != testCase.wantStart || gotEnd != testCase.wantEnd {
				t.Fatalf("expected bounds (%d, %d), got (%d, %d)", testCase.wantStart, testCase.wantEnd, gotStart, gotEnd)
			}
		})
	}
}

func TestFocusedInputIgnoresNavigationHotkeys(t *testing.T) {
	model := newTUIModel(nil, "", 100, "auto")
	model.page = 1
	model.queryInput.SetValue("sou")
	model.queryInput.Focus()

	nextModel, _ := model.updateResultsMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	updated := nextModel.(tuiModel)

	if updated.page != 1 {
		t.Fatalf("expected page to stay 1 while input focused, got %d", updated.page)
	}
	if updated.queryInput.Value() != "soun" {
		t.Fatalf("expected query input to continue typing, got %q", updated.queryInput.Value())
	}
}

func TestNewTUIModelWithInitialQueryStartsBlurred(t *testing.T) {
	model := newTUIModel(nil, "example query", 100, "auto")
	if model.queryInput.Focused() {
		t.Fatal("expected input to start blurred when initial query is provided")
	}
}

func TestFocusedInputNavigationMovesResultsWhenPresent(t *testing.T) {
	model := newTUIModel(nil, "", 100, "auto")
	model.results = mockResults(15)
	model.totalCount = 15
	model.pageSize = 10
	model.page = 0
	model.selected = 9
	model.queryInput.SetValue("example query")
	model.queryInput.Focus()

	next, _ := model.updateResultsMode(tea.KeyMsg{Type: tea.KeyDown})
	updated := next.(tuiModel)

	if updated.queryInput.Focused() {
		t.Fatal("expected input to blur when using navigation keys with visible results")
	}
	if updated.page != 1 || updated.selected != 10 {
		t.Fatalf("expected one keypress to move to next page (1,10), got (%d,%d)", updated.page, updated.selected)
	}
}

func TestPageNavigationDoesNotOverflow(t *testing.T) {
	model := newTUIModel(nil, "", 100, "auto")
	model.results = mockResults(15)
	model.totalCount = 15
	model.pageSize = 10
	model.page = 0
	model.selected = 0
	model.queryInput.Blur()

	next, _ := model.updateResultsMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	updated := next.(tuiModel)
	if updated.page != 0 {
		t.Fatalf("expected page to remain at 0, got %d", updated.page)
	}

	next, _ = updated.updateResultsMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	updated = next.(tuiModel)
	if updated.page != 1 {
		t.Fatalf("expected page to move to 1, got %d", updated.page)
	}
	if updated.selected != 10 {
		t.Fatalf("expected selected index 10 on page 2, got %d", updated.selected)
	}

	next, _ = updated.updateResultsMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	updated = next.(tuiModel)
	if updated.page != 1 {
		t.Fatalf("expected page to stay at 1 when already last page, got %d", updated.page)
	}
}

func TestSuggestionKeyAppliesSuggestionAndStartsSearch(t *testing.T) {
	model := newTUIModel(nil, "", 100, "auto")
	model.queryInput.SetValue("exmple query")
	model.queryInput.Blur()
	model.suggestions = []string{"example query"}
	model.results = nil
	model.totalCount = 0
	model.loading = false

	nextModel, _ := model.updateResultsMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	updated := nextModel.(tuiModel)

	if updated.queryInput.Value() != "example query" {
		t.Fatalf("expected suggestion to populate query input, got %q", updated.queryInput.Value())
	}
	if !updated.loading {
		t.Fatal("expected loading=true after accepting suggestion")
	}
	if updated.activeQuery != "example query" {
		t.Fatalf("expected active query example query, got %q", updated.activeQuery)
	}
}

func TestResultsScreenHidesScoreMetadata(t *testing.T) {
	model := newTUIModel(nil, "", 100, "auto")
	model.ready = true
	model.width = 120
	model.height = 30
	model.queryInput.Blur()
	model.activeQuery = "example query"
	model.results = []*search.Result{
		{
			Title:   "Example Site Home",
			URL:     "https://example.com/",
			Snippet: "Shop example products",
			Score:   9.99,
		},
	}
	model.totalCount = 1
	model.searchDuration = 217 * time.Millisecond
	model.selected = 0

	view := model.resultsScreenView()
	if strings.Contains(view, "9.99") || strings.Contains(view, "[9.99]") {
		t.Fatalf("expected score metadata to be hidden, got view:\n%s", view)
	}
	if !strings.Contains(view, "Example Site Home") {
		t.Fatalf("expected title in view, got:\n%s", view)
	}
	if !strings.Contains(view, "https://example.com/") {
		t.Fatalf("expected URL in view, got:\n%s", view)
	}
}

func TestResultsScreenShowsPageIndicator(t *testing.T) {
	model := newTUIModel(nil, "", 100, "auto")
	model.ready = true
	model.width = 120
	model.height = 30
	model.queryInput.Blur()
	model.activeQuery = "example query"
	model.results = mockResults(15)
	model.totalCount = 15
	model.pageSize = 10
	model.page = 0
	model.selected = 0

	view := model.resultsScreenView()
	if !strings.Contains(view, "Page 1 / 2") {
		t.Fatalf("expected page indicator for first page, got:\n%s", view)
	}

	model.page = 1
	model.selected = 10
	view = model.resultsScreenView()
	if !strings.Contains(view, "Page 2 / 2") {
		t.Fatalf("expected page indicator for second page, got:\n%s", view)
	}
}

func TestBackFromPreviewPreservesSelectionContext(t *testing.T) {
	model := newTUIModel(nil, "", 100, "auto")
	model.mode = tuiModeDocument
	model.page = 1
	model.selected = 12

	nextModel, _ := model.updateDocumentMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	updated := nextModel.(tuiModel)

	if updated.mode != tuiModeResults {
		t.Fatalf("expected mode to switch to results, got %v", updated.mode)
	}
	if updated.page != 1 || updated.selected != 12 {
		t.Fatalf("expected page/selection preserved (1,12), got (%d,%d)", updated.page, updated.selected)
	}
}

func TestURLOpenFailureUpdatesStatusWithoutLeavingResults(t *testing.T) {
	model := newTUIModel(nil, "", 100, "auto")
	model.mode = tuiModeResults
	model.loading = true

	nextModel, _ := model.Update(tuiURLOpenedMsg{
		url: "https://example.com",
		err: fmt.Errorf("open failed"),
	})
	updated := nextModel.(tuiModel)

	if updated.mode != tuiModeResults {
		t.Fatalf("expected to remain in results mode, got %v", updated.mode)
	}
	if updated.loading {
		t.Fatal("expected loading=false after url open response")
	}
	if updated.lastError == nil {
		t.Fatal("expected lastError to be set")
	}
	if !strings.Contains(updated.status, "Failed to open URL") {
		t.Fatalf("expected actionable status message, got %q", updated.status)
	}
}

func TestHomeScreenViewShowsCenteredBrandingAndPrompt(t *testing.T) {
	model := newTUIModel(nil, "", 100, "auto")
	model.ready = true
	model.width = 120
	model.height = 36
	model.queryInput.Width = model.desiredQueryInputWidth()
	model.queryInput.Focus()

	view := model.View()
	if !strings.Contains(view, "GGGG") {
		t.Fatalf("expected home logo in initial view, got:\n%s", view)
	}
	if !strings.Contains(view, "Search your indexed web corpus") {
		t.Fatalf("expected home subtitle in initial view, got:\n%s", view)
	}
	if !strings.Contains(view, "Enter search") {
		t.Fatalf("expected home hint in initial view, got:\n%s", view)
	}
}

func TestSelectionMoveUpdatesInlinePreviewTarget(t *testing.T) {
	model := newTUIModel(nil, "", 100, "auto")
	model.ready = true
	model.width = 120
	model.height = 36
	model.queryInput.Blur()
	model.activeQuery = "example query"
	model.results = []*search.Result{
		{DocID: "doc-1", Title: "Result One", URL: "https://example.com/1", Snippet: "first"},
		{DocID: "doc-2", Title: "Result Two", URL: "https://example.com/2", Snippet: "second"},
	}
	model.totalCount = 2
	model.selected = 0

	cmd := model.ensureInlinePreviewForSelection()
	if cmd != nil {
		t.Fatal("expected no async cmd when runtime is nil")
	}
	if model.inlinePreviewResult == nil || model.inlinePreviewResult.DocID != "doc-1" {
		t.Fatalf("expected inline preview target doc-1, got %#v", model.inlinePreviewResult)
	}

	nextModel, _ := model.updateResultsMode(tea.KeyMsg{Type: tea.KeyDown})
	updated := nextModel.(tuiModel)
	if updated.inlinePreviewResult == nil || updated.inlinePreviewResult.DocID != "doc-2" {
		t.Fatalf("expected inline preview target doc-2 after moving down, got %#v", updated.inlinePreviewResult)
	}
}

func TestSearchCompletedBlursInputForNavigation(t *testing.T) {
	model := newTUIModel(nil, "example query", 100, "auto")
	model.queryInput.Focus()

	nextModel, _ := model.Update(tuiSearchCompletedMsg{
		query: "example query",
		response: &search.SearchResponse{
			Results: []*search.Result{
				{DocID: "doc-1", Title: "Result One", URL: "https://example.com/1", Snippet: "snippet"},
			},
			TotalCount: 1,
		},
	})
	updated := nextModel.(tuiModel)
	if updated.queryInput.Focused() {
		t.Fatal("expected query input to blur after search completion")
	}
}

func TestEnsureInlinePreviewSkipsDuplicateLoadForSameSelection(t *testing.T) {
	model := newTUIModel(&searchRuntime{}, "", 100, "auto")
	model.results = []*search.Result{
		{DocID: "doc-1", Title: "Result One", URL: "https://example.com/1", Snippet: "snippet"},
	}
	model.totalCount = 1
	model.selected = 0
	model.page = 0
	model.pageSize = 10
	model.width = 120
	model.height = 30
	model.ready = true

	firstCmd := model.ensureInlinePreviewForSelection()
	if firstCmd == nil {
		t.Fatal("expected first preview load command to be scheduled")
	}
	firstRequestID := model.inlinePreviewRequestID

	secondCmd := model.ensureInlinePreviewForSelection()
	if secondCmd != nil {
		t.Fatal("expected duplicate preview load for same selection to be skipped")
	}
	if model.inlinePreviewRequestID != firstRequestID {
		t.Fatalf("expected request id to remain %d, got %d", firstRequestID, model.inlinePreviewRequestID)
	}
}

func TestMarkdownInlinePreviewContentIsBounded(t *testing.T) {
	var builder strings.Builder
	for i := 0; i < 300; i++ {
		builder.WriteString("line ")
		builder.WriteString(fmt.Sprintf("%d", i))
		builder.WriteString("\n")
	}

	preview := markdownInlinePreviewContent(builder.String())
	if strings.Count(preview, "\n") > 145 {
		t.Fatalf("expected preview content to be trimmed, got too many lines: %d", strings.Count(preview, "\n"))
	}
	if len([]rune(preview)) > 12100 {
		t.Fatalf("expected preview content to be capped in size, got %d runes", len([]rune(preview)))
	}
}

func TestRenderMarkdownForTUIAutoStyleFallback(t *testing.T) {
	rendered, err := renderMarkdownForTUI("# Title\n\nSome text", "auto", 80)
	if err != nil {
		t.Fatalf("expected renderMarkdownForTUI to succeed, got %v", err)
	}
	if strings.TrimSpace(rendered) == "" {
		t.Fatal("expected non-empty rendered markdown")
	}
}

func TestInlinePreviewTimeoutClearsLoadingState(t *testing.T) {
	model := newTUIModel(nil, "", 100, "auto")
	model.inlinePreviewLoading = true
	model.inlinePreviewLoadingDocID = "doc-1"
	model.inlinePreviewLoadingWidth = 80
	model.inlinePreviewRequestID = 9

	nextModel, _ := model.Update(tuiInlinePreviewTimeoutMsg{requestID: 9})
	updated := nextModel.(tuiModel)

	if updated.inlinePreviewLoading {
		t.Fatal("expected inline preview loading to be cleared on timeout")
	}
	if updated.inlinePreviewLoadingDocID != "" {
		t.Fatalf("expected loading doc id to clear, got %q", updated.inlinePreviewLoadingDocID)
	}
	if updated.inlinePreviewError == nil || !strings.Contains(updated.inlinePreviewError.Error(), "timed out") {
		t.Fatalf("expected timeout error message, got %#v", updated.inlinePreviewError)
	}
}

func TestInlinePreviewTimeoutIgnoresStaleRequest(t *testing.T) {
	model := newTUIModel(nil, "", 100, "auto")
	model.inlinePreviewLoading = true
	model.inlinePreviewRequestID = 4

	nextModel, _ := model.Update(tuiInlinePreviewTimeoutMsg{requestID: 3})
	updated := nextModel.(tuiModel)

	if !updated.inlinePreviewLoading {
		t.Fatal("expected stale timeout to be ignored")
	}
}

func TestEnsureInlinePreviewPreloadSchedulesFirst100(t *testing.T) {
	model := newTUIModel(&searchRuntime{}, "", 100, "auto")
	model.ready = true
	model.width = 120
	model.height = 30
	model.activeQuery = "example query"
	results := make([]*search.Result, 0, 120)
	for i := 0; i < 120; i++ {
		results = append(results, &search.Result{
			DocID:   fmt.Sprintf("doc-%03d", i),
			Title:   fmt.Sprintf("Result %d", i),
			URL:     fmt.Sprintf("https://example.com/%d", i),
			Snippet: "snippet",
		})
	}
	model.results = results
	model.totalCount = 120

	cmd := model.ensureInlinePreviewPreload()
	if cmd == nil {
		t.Fatal("expected preload command to be scheduled")
	}
	if !model.inlinePreviewPreloadInFlight {
		t.Fatal("expected preload to be marked in flight")
	}
	if model.inlinePreviewPreloadRequestID == 0 {
		t.Fatal("expected preload request id to be incremented")
	}

	cmd = model.ensureInlinePreviewPreload()
	if cmd != nil {
		t.Fatal("expected duplicate preload scheduling to be skipped while in flight")
	}
}

func TestInlinePreviewPreloadedMessageMergesCacheAndClearsLoading(t *testing.T) {
	model := newTUIModel(nil, "", 100, "auto")
	model.results = []*search.Result{
		{DocID: "doc-1", Title: "Result One", URL: "https://example.com/1", Snippet: "snippet"},
	}
	model.totalCount = 1
	model.page = 0
	model.pageSize = 10
	model.selected = 0
	model.inlinePreviewLoading = true
	model.inlinePreviewLoadingDocID = "doc-1"
	model.inlinePreviewLoadingWidth = 70
	model.inlinePreviewPreloadRequestID = 5

	doc := &storage.Document{ID: "doc-1", URL: "https://example.com/1", Title: "Result One"}
	entry := tuiInlinePreviewCacheEntry{document: doc, rendered: "rendered markdown", renderWidth: 70}

	nextModel, _ := model.Update(tuiInlinePreviewPreloadedMsg{
		requestID: 5,
		entries:   map[string]tuiInlinePreviewCacheEntry{"doc-1": entry},
		loaded:    1,
	})
	updated := nextModel.(tuiModel)

	if updated.inlinePreviewLoading {
		t.Fatal("expected inline preview loading to clear after preloaded message")
	}
	if strings.TrimSpace(updated.inlinePreviewRendered) == "" {
		t.Fatal("expected inline preview rendered content to be populated from preload cache")
	}
	if _, exists := updated.inlinePreviewCache["doc-1"]; !exists {
		t.Fatal("expected preloaded entry to be merged into inline preview cache")
	}
}

func TestMoveSelectionDownCrossesToNextPage(t *testing.T) {
	model := newTUIModel(nil, "", 100, "auto")
	model.queryInput.Blur()
	model.results = mockResults(15)
	model.totalCount = 15
	model.pageSize = 10
	model.page = 0
	model.selected = 9

	nextModel, _ := model.updateResultsMode(tea.KeyMsg{Type: tea.KeyDown})
	updated := nextModel.(tuiModel)

	if updated.page != 1 {
		t.Fatalf("expected page to advance to 1, got %d", updated.page)
	}
	if updated.selected != 10 {
		t.Fatalf("expected selected index 10, got %d", updated.selected)
	}
}

func TestLeftPaneScrollWindowKeepsSelectedVisible(t *testing.T) {
	model := newTUIModel(nil, "", 100, "auto")
	model.queryInput.Blur()
	model.results = mockResults(10)
	model.totalCount = 10
	model.pageSize = 10
	model.page = 0
	model.selected = 9

	view := model.renderResultsListPane(48, 14)
	if !strings.Contains(view, "Result J") {
		t.Fatalf("expected selected lower result to be visible in scrolled pane, got:\n%s", view)
	}
}

func mockResults(count int) []*search.Result {
	results := make([]*search.Result, 0, count)
	for idx := 0; idx < count; idx++ {
		results = append(results, &search.Result{
			Title:   "Result " + string(rune('A'+(idx%26))),
			URL:     "https://example.com/" + strings.ToLower(string(rune('a'+(idx%26)))),
			Snippet: "Example snippet text for result",
			Score:   1.23,
		})
	}
	return results
}

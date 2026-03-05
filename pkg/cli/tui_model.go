package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/abuiliazeed/gosearch/internal/search"
	"github.com/abuiliazeed/gosearch/internal/storage"
)

type tuiMode int

const (
	tuiModeResults tuiMode = iota
	tuiModeDocument
)

type tuiSearchCompletedMsg struct {
	query       string
	response    *search.SearchResponse
	suggestions []string
	preloaded   map[string]tuiInlinePreviewCacheEntry
	preloadDone int
	preloadErr  error
	err         error
}

type tuiDocumentOpenedMsg struct {
	result   search.Result
	document *storage.Document
	rendered string
	err      error
}

type tuiURLOpenedMsg struct {
	url string
	err error
}

type tuiInlinePreviewLoadedMsg struct {
	requestID   int
	result      search.Result
	document    *storage.Document
	rendered    string
	renderWidth int
	err         error
}

type tuiInlinePreviewTimeoutMsg struct {
	requestID int
}

type tuiInlinePreviewPreloadedMsg struct {
	requestID int
	entries   map[string]tuiInlinePreviewCacheEntry
	loaded    int
	err       error
}

type tuiInlinePreviewCacheEntry struct {
	document    *storage.Document
	rendered    string
	renderWidth int
}

type tuiModel struct {
	runtime *searchRuntime
	limit   int
	style   string

	mode   tuiMode
	ready  bool
	width  int
	height int

	queryInput textinput.Model
	loading    bool
	status     string

	results        []*search.Result
	selected       int
	totalCount     int
	searchDuration time.Duration
	cached         bool

	page     int
	pageSize int

	suggestions  []string
	activeQuery  string
	lastSearchAt time.Time
	lastError    error

	currentResult   *search.Result
	currentDocument *storage.Document
	documentView    viewport.Model

	inlinePreviewRequestID    int
	inlinePreviewResult       *search.Result
	inlinePreviewDocument     *storage.Document
	inlinePreviewRendered     string
	inlinePreviewLoading      bool
	inlinePreviewLoadingDocID string
	inlinePreviewLoadingWidth int
	inlinePreviewError        error
	inlinePreviewCache        map[string]tuiInlinePreviewCacheEntry

	inlinePreviewPreloadRequestID int
	inlinePreviewPreloadInFlight  bool
	inlinePreviewPreloadQuery     string
	inlinePreviewPreloadWidth     int
	inlinePreviewPreloadLoaded    int
}

func newTUIModel(runtime *searchRuntime, initialQuery string, limit int, style string) tuiModel {
	if limit < 1 {
		limit = 100
	}

	queryInput := textinput.New()
	queryInput.Prompt = ""
	queryInput.Placeholder = "Search Gosearch"
	queryInput.CharLimit = 500
	queryInput.SetValue(strings.TrimSpace(initialQuery))
	if strings.TrimSpace(initialQuery) == "" {
		queryInput.Focus()
	} else {
		queryInput.Blur()
	}

	model := tuiModel{
		runtime:            runtime,
		limit:              limit,
		style:              strings.TrimSpace(style),
		mode:               tuiModeResults,
		queryInput:         queryInput,
		documentView:       viewport.New(0, 0),
		pageSize:           10,
		inlinePreviewCache: make(map[string]tuiInlinePreviewCacheEntry),
	}

	return model
}

func (m tuiModel) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink}
	initialQuery := strings.TrimSpace(m.queryInput.Value())
	if initialQuery != "" {
		cmds = append(cmds, tuiRunSearchCmd(m.runtime, initialQuery, m.limit, m.style, m.inlinePreviewRenderWidthForCommand()))
	}
	return tea.Batch(cmds...)
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := msg.(type) {
	case tea.WindowSizeMsg:
		m.ready = true
		m.width = message.Width
		m.height = message.Height

		m.queryInput.Width = m.desiredQueryInputWidth()
		m.resizeDocumentViewport()

		return m, tea.Batch(
			m.ensureInlinePreviewForSelection(),
			m.ensureInlinePreviewPreload(),
		)

	case tuiSearchCompletedMsg:
		m.loading = false
		m.lastSearchAt = time.Now()
		m.activeQuery = message.query
		m.queryInput.SetValue(message.query)
		// Keep keyboard navigation responsive after startup queries and completed searches.
		m.queryInput.Blur()

		if message.err != nil {
			m.lastError = message.err
			m.status = "Search failed"
			m.results = nil
			m.totalCount = 0
			m.suggestions = nil
			m.page = 0
			m.selected = 0
			m.resetInlinePreviewState()
			return m, nil
		}

		m.lastError = nil
		m.results = message.response.Results
		m.totalCount = message.response.TotalCount
		m.searchDuration = message.response.Duration
		m.cached = message.response.Cached
		m.suggestions = message.suggestions
		m.page = 0
		m.selected = 0
		m.inlinePreviewPreloadInFlight = false
		m.inlinePreviewPreloadLoaded = message.preloadDone

		if len(message.preloaded) > 0 {
			for docID, entry := range message.preloaded {
				m.inlinePreviewCache[docID] = entry
			}
		}

		if len(m.results) == 0 {
			m.resetInlinePreviewState()
			m.inlinePreviewPreloadQuery = ""
			m.inlinePreviewPreloadWidth = 0
			if len(m.suggestions) > 0 {
				m.status = fmt.Sprintf("No results for %q. Press y to try %q.", message.query, m.suggestions[0])
			} else {
				m.status = fmt.Sprintf("No results for %q.", message.query)
			}
			return m, nil
		} else {
			m.status = fmt.Sprintf("Search complete for %q", message.query)
		}

		m.inlinePreviewPreloadQuery = strings.TrimSpace(message.query)
		m.inlinePreviewPreloadWidth = m.inlinePreviewRenderWidth()
		if selected := m.selectedResult(); selected != nil {
			if entry, exists := m.inlinePreviewCache[selected.DocID]; exists &&
				entry.renderWidth == m.inlinePreviewRenderWidth() &&
				strings.TrimSpace(entry.rendered) != "" {
				m.inlinePreviewDocument = entry.document
				m.inlinePreviewRendered = entry.rendered
				m.inlinePreviewLoading = false
				m.inlinePreviewLoadingDocID = ""
				m.inlinePreviewLoadingWidth = 0
				m.inlinePreviewError = nil
			}
		}

		return m, tea.Batch(
			m.ensureInlinePreviewForSelection(),
			m.ensureInlinePreviewPreload(),
		)

	case tuiDocumentOpenedMsg:
		m.loading = false
		if message.err != nil {
			m.lastError = message.err
			m.status = "Failed to open result preview"
			return m, nil
		}

		resultCopy := message.result
		m.currentResult = &resultCopy
		m.currentDocument = message.document
		m.mode = tuiModeDocument
		m.lastError = nil
		m.status = fmt.Sprintf("Previewing %s", message.document.URL)
		m.resizeDocumentViewport()
		m.documentView.SetContent(message.rendered)
		m.documentView.GotoTop()

		return m, nil

	case tuiURLOpenedMsg:
		m.loading = false
		if message.err != nil {
			m.lastError = message.err
			m.status = "Failed to open URL in browser"
			return m, nil
		}

		m.lastError = nil
		m.status = fmt.Sprintf("Opened in browser: %s", message.url)
		return m, nil

	case tuiInlinePreviewLoadedMsg:
		if message.requestID != m.inlinePreviewRequestID {
			return m, nil
		}

		m.inlinePreviewLoading = false
		m.inlinePreviewLoadingDocID = ""
		m.inlinePreviewLoadingWidth = 0
		if message.err != nil {
			m.inlinePreviewError = message.err
			return m, nil
		}

		m.inlinePreviewError = nil
		resultCopy := message.result
		m.inlinePreviewResult = &resultCopy
		m.inlinePreviewDocument = message.document
		m.inlinePreviewRendered = message.rendered
		m.inlinePreviewCache[resultCopy.DocID] = tuiInlinePreviewCacheEntry{
			document:    message.document,
			rendered:    message.rendered,
			renderWidth: message.renderWidth,
		}
		return m, nil

	case tuiInlinePreviewTimeoutMsg:
		if message.requestID != m.inlinePreviewRequestID || !m.inlinePreviewLoading {
			return m, nil
		}

		m.inlinePreviewLoading = false
		m.inlinePreviewLoadingDocID = ""
		m.inlinePreviewLoadingWidth = 0
		m.inlinePreviewError = fmt.Errorf("preview load timed out, move selection to retry")
		return m, nil

	case tuiInlinePreviewPreloadedMsg:
		if message.requestID != m.inlinePreviewPreloadRequestID {
			return m, nil
		}

		m.inlinePreviewPreloadInFlight = false
		if message.err != nil {
			// Non-fatal: on-demand preview loading still works.
			return m, nil
		}

		for docID, entry := range message.entries {
			m.inlinePreviewCache[docID] = entry
		}
		m.inlinePreviewPreloadLoaded += message.loaded

		selected := m.selectedResult()
		if selected != nil {
			entry, exists := message.entries[selected.DocID]
			if exists {
				m.inlinePreviewDocument = entry.document
				m.inlinePreviewRendered = entry.rendered
				if m.inlinePreviewLoading &&
					m.inlinePreviewLoadingDocID == selected.DocID &&
					m.inlinePreviewLoadingWidth == entry.renderWidth {
					m.inlinePreviewLoading = false
					m.inlinePreviewLoadingDocID = ""
					m.inlinePreviewLoadingWidth = 0
					m.inlinePreviewError = nil
				}
			}
		}

		return m, nil
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	if m.mode == tuiModeDocument {
		return m.updateDocumentMode(msg)
	}

	return m.updateResultsMode(msg)
}

func (m tuiModel) updateResultsMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if m.queryInput.Focused() {
			// If results are visible, navigation keys should work immediately even if input is focused.
			if len(m.results) > 0 && !m.loading {
				switch key.String() {
				case "n":
					m.queryInput.Blur()
					if m.nextPage() {
						return m, m.ensureInlinePreviewForSelection()
					}
					return m, nil
				case "p":
					m.queryInput.Blur()
					if m.previousPage() {
						return m, m.ensureInlinePreviewForSelection()
					}
					return m, nil
				case "up", "k":
					m.queryInput.Blur()
					if m.moveSelection(-1) {
						return m, m.ensureInlinePreviewForSelection()
					}
					return m, nil
				case "down", "j":
					m.queryInput.Blur()
					if m.moveSelection(1) {
						return m, m.ensureInlinePreviewForSelection()
					}
					return m, nil
				}
			}

			switch key.String() {
			case "enter":
				return m.startSearch()
			case "esc":
				m.queryInput.Blur()
				return m, nil
			}

			// When input is focused, every other key should continue editing query text.
			var cmd tea.Cmd
			m.queryInput, cmd = m.queryInput.Update(msg)
			return m, cmd
		}

		if !m.loading {
			switch key.String() {
			case "/":
				m.queryInput.Focus()
				return m, nil
			case "r":
				return m.startSearch()
			case "n":
				if m.nextPage() {
					return m, m.ensureInlinePreviewForSelection()
				}
				return m, nil
			case "p":
				if m.previousPage() {
					return m, m.ensureInlinePreviewForSelection()
				}
				return m, nil
			case "up", "k":
				if m.moveSelection(-1) {
					return m, m.ensureInlinePreviewForSelection()
				}
				return m, nil
			case "down", "j":
				if m.moveSelection(1) {
					return m, m.ensureInlinePreviewForSelection()
				}
				return m, nil
			case "enter":
				return m.openSelectedDocument()
			case "o":
				return m.openSelectedURL()
			case "y":
				return m.applySuggestion()
			}
		}
	}

	var cmd tea.Cmd
	m.queryInput, cmd = m.queryInput.Update(msg)
	return m, cmd
}

func (m tuiModel) updateDocumentMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "b", "esc":
			m.mode = tuiModeResults
			m.status = "Back to results"
			return m, nil
		case "/":
			m.mode = tuiModeResults
			m.queryInput.Focus()
			return m, nil
		case "o":
			if m.currentDocument != nil {
				m.loading = true
				return m, tuiOpenURLCmd(m.currentDocument.URL)
			}
		}
	}

	var cmd tea.Cmd
	m.documentView, cmd = m.documentView.Update(msg)
	return m, cmd
}

func (m tuiModel) startSearch() (tea.Model, tea.Cmd) {
	query := strings.TrimSpace(m.queryInput.Value())
	if query == "" {
		m.lastError = fmt.Errorf("query cannot be empty")
		m.status = "Enter a query to search"
		return m, nil
	}

	m.queryInput.Blur()
	m.mode = tuiModeResults
	m.loading = true
	m.lastError = nil
	m.status = fmt.Sprintf("Searching for %q...", query)
	m.activeQuery = query
	m.page = 0
	m.selected = 0
	m.suggestions = nil
	m.resetInlinePreviewState()
	m.inlinePreviewPreloadInFlight = false
	m.inlinePreviewPreloadQuery = ""
	m.inlinePreviewPreloadWidth = 0
	m.inlinePreviewPreloadLoaded = 0

	return m, tuiRunSearchCmd(m.runtime, query, m.limit, m.style, m.inlinePreviewRenderWidthForCommand())
}

func (m tuiModel) applySuggestion() (tea.Model, tea.Cmd) {
	if len(m.suggestions) == 0 {
		return m, nil
	}

	m.queryInput.SetValue(m.suggestions[0])
	return m.startSearch()
}

func (m tuiModel) openSelectedDocument() (tea.Model, tea.Cmd) {
	if len(m.results) == 0 {
		return m, nil
	}

	m.clampPage()
	start, end := m.currentPageBounds()
	if m.selected < start || m.selected >= end {
		m.selected = start
	}

	selected := m.results[m.selected]
	m.loading = true
	m.lastError = nil
	m.status = fmt.Sprintf("Loading preview for %s...", selected.URL)

	return m, tuiOpenDocumentCmd(m.runtime, selected, m.style, m.renderWidth())
}

func (m tuiModel) openSelectedURL() (tea.Model, tea.Cmd) {
	if len(m.results) == 0 {
		return m, nil
	}

	m.clampPage()
	start, end := m.currentPageBounds()
	if m.selected < start || m.selected >= end {
		m.selected = start
	}

	selected := m.results[m.selected]
	m.loading = true
	m.lastError = nil
	m.status = fmt.Sprintf("Opening %s in browser...", selected.URL)

	return m, tuiOpenURLCmd(selected.URL)
}

func (m *tuiModel) nextPage() bool {
	totalPages := m.totalPages()
	if m.page < totalPages-1 {
		m.page++
		start, _ := m.currentPageBounds()
		m.selected = start
		m.status = fmt.Sprintf("Page %d of %d", m.page+1, totalPages)
		return true
	}
	return false
}

func (m *tuiModel) previousPage() bool {
	if m.page > 0 {
		m.page--
		start, _ := m.currentPageBounds()
		m.selected = start
		m.status = fmt.Sprintf("Page %d of %d", m.page+1, m.totalPages())
		return true
	}
	return false
}

func (m *tuiModel) moveSelection(delta int) bool {
	if len(m.results) == 0 {
		return false
	}

	next := m.selected + delta
	if next < 0 || next >= len(m.results) {
		return false
	}

	previousPage := m.page
	m.selected = next

	// Keep the selected absolute index in range for the current page window.
	for {
		m.clampPage()
		start, end := m.currentPageBounds()
		if m.selected < start && m.page > 0 {
			m.page--
			continue
		}
		if m.selected >= end && m.page < m.totalPages()-1 {
			m.page++
			continue
		}
		break
	}

	if m.page != previousPage {
		m.status = fmt.Sprintf("Page %d of %d", m.page+1, m.totalPages())
	}

	return true
}

func (m *tuiModel) clampPage() {
	maxPage := m.totalPages() - 1
	if m.page < 0 {
		m.page = 0
	}
	if m.page > maxPage {
		m.page = maxPage
	}
}

func (m tuiModel) totalPages() int {
	return totalPages(len(m.results), m.pageSize)
}

func (m tuiModel) currentPageBounds() (int, int) {
	return pageBounds(m.page, m.pageSize, len(m.results))
}

func (m tuiModel) currentPageResults() []*search.Result {
	start, end := m.currentPageBounds()
	if start >= end {
		return nil
	}
	return m.results[start:end]
}

func (m tuiModel) selectedResult() *search.Result {
	if len(m.results) == 0 {
		return nil
	}

	start, end := m.currentPageBounds()
	index := m.selected
	if index < start || index >= end {
		index = start
	}
	return m.results[index]
}

func (m *tuiModel) resetInlinePreviewState() {
	m.inlinePreviewRequestID++
	m.inlinePreviewResult = nil
	m.inlinePreviewDocument = nil
	m.inlinePreviewRendered = ""
	m.inlinePreviewLoading = false
	m.inlinePreviewLoadingDocID = ""
	m.inlinePreviewLoadingWidth = 0
	m.inlinePreviewError = nil
}

func (m tuiModel) inlinePreviewRenderWidth() int {
	leftPaneWidth, rightPaneWidth := m.resultsPaneWidths()
	_ = leftPaneWidth

	width := rightPaneWidth - 6
	if width < 24 {
		width = 24
	}
	return width
}

func (m tuiModel) inlinePreviewRenderWidthForCommand() int {
	if !m.ready {
		return 70
	}
	return m.inlinePreviewRenderWidth()
}

func (m *tuiModel) ensureInlinePreviewForSelection() tea.Cmd {
	if m.mode != tuiModeResults || len(m.results) == 0 {
		return nil
	}

	selected := m.selectedResult()
	if selected == nil {
		return nil
	}

	resultCopy := *selected
	m.inlinePreviewResult = &resultCopy
	m.inlinePreviewError = nil

	renderWidth := m.inlinePreviewRenderWidth()
	cacheEntry, exists := m.inlinePreviewCache[resultCopy.DocID]
	if exists && cacheEntry.renderWidth == renderWidth && cacheEntry.rendered != "" {
		m.inlinePreviewLoading = false
		m.inlinePreviewLoadingDocID = ""
		m.inlinePreviewLoadingWidth = 0
		m.inlinePreviewDocument = cacheEntry.document
		m.inlinePreviewRendered = cacheEntry.rendered
		return nil
	}

	// Avoid spawning duplicate expensive render jobs for the same selection.
	if m.inlinePreviewLoading &&
		m.inlinePreviewLoadingDocID == resultCopy.DocID &&
		m.inlinePreviewLoadingWidth == renderWidth {
		return nil
	}

	if m.runtime == nil {
		m.inlinePreviewLoading = false
		m.inlinePreviewRendered = ""
		return nil
	}

	m.inlinePreviewLoading = true
	m.inlinePreviewLoadingDocID = resultCopy.DocID
	m.inlinePreviewLoadingWidth = renderWidth
	m.inlinePreviewDocument = nil
	// Keep currently rendered preview visible while loading the next one.
	m.inlinePreviewRequestID++

	requestID := m.inlinePreviewRequestID
	return tea.Batch(
		tuiLoadInlinePreviewCmd(m.runtime, selected, m.style, renderWidth, requestID),
		tuiInlinePreviewTimeoutCmd(requestID, 3*time.Second),
	)
}

func (m tuiModel) inlinePreviewPreloadLimit() int {
	if len(m.results) == 0 {
		return 0
	}
	if len(m.results) < 100 {
		return len(m.results)
	}
	return 100
}

func (m *tuiModel) ensureInlinePreviewPreload() tea.Cmd {
	if m.mode != tuiModeResults || !m.ready || len(m.results) == 0 || m.runtime == nil {
		return nil
	}

	query := strings.TrimSpace(m.activeQuery)
	if query == "" {
		return nil
	}

	renderWidth := m.inlinePreviewRenderWidth()
	targetCount := m.inlinePreviewPreloadLimit()
	if targetCount == 0 {
		return nil
	}

	if m.inlinePreviewPreloadInFlight &&
		m.inlinePreviewPreloadQuery == query &&
		m.inlinePreviewPreloadWidth == renderWidth {
		return nil
	}

	missing := make([]search.Result, 0, targetCount)
	alreadyCached := 0
	for i := 0; i < targetCount; i++ {
		result := m.results[i]
		if result == nil || result.DocID == "" {
			continue
		}

		if cached, exists := m.inlinePreviewCache[result.DocID]; exists &&
			cached.renderWidth == renderWidth &&
			strings.TrimSpace(cached.rendered) != "" {
			alreadyCached++
			continue
		}

		resultCopy := *result
		missing = append(missing, resultCopy)
	}

	m.inlinePreviewPreloadQuery = query
	m.inlinePreviewPreloadWidth = renderWidth
	m.inlinePreviewPreloadLoaded = alreadyCached

	if len(missing) == 0 {
		m.inlinePreviewPreloadInFlight = false
		return nil
	}

	m.inlinePreviewPreloadRequestID++
	requestID := m.inlinePreviewPreloadRequestID
	m.inlinePreviewPreloadInFlight = true
	return tuiPreloadInlinePreviewsCmd(m.runtime, missing, m.style, renderWidth, requestID)
}

func (m *tuiModel) resizeDocumentViewport() {
	if !m.ready {
		return
	}

	viewportWidth := m.width - 4
	if viewportWidth < 20 {
		viewportWidth = 20
	}

	viewportHeight := m.height - 9
	if viewportHeight < 3 {
		viewportHeight = 3
	}

	m.documentView.Width = viewportWidth
	m.documentView.Height = viewportHeight
}

func (m tuiModel) renderWidth() int {
	width := m.width - 8
	if width < 40 {
		width = 100
	}
	return width
}

func (m tuiModel) View() string {
	if !m.ready {
		return "Initializing TUI..."
	}

	if m.mode == tuiModeDocument {
		return m.documentScreenView()
	}
	if m.showHomeView() {
		return m.homeScreenView()
	}
	return m.resultsScreenView()
}

func (m tuiModel) showHomeView() bool {
	return strings.TrimSpace(m.activeQuery) == "" &&
		len(m.results) == 0 &&
		!m.loading &&
		m.lastError == nil
}

func (m tuiModel) desiredQueryInputWidth() int {
	width := m.width - 18
	if width < 24 {
		width = 24
	}
	if width > 72 {
		width = 72
	}
	return width
}

func (m tuiModel) homeScreenView() string {
	theme := newTUITheme()

	logo := lipgloss.JoinVertical(
		lipgloss.Center,
		theme.homeAccentBlue.Render("  GGGG   OOO   SSSS EEEEE  AAAAA RRRR   CCCC H   H"),
		theme.homeAccentRed.Render(" G     O   O S     E      A   A R   R C     H   H"),
		theme.homeAccentYellow.Render(" G  GG O   O  SSS  EEEE   AAAAA RRRR  C     HHHHH"),
		theme.homeAccentGreen.Render(" G   G O   O     S E      A   A R  R  C     H   H"),
		theme.homeAccentBlue.Render("  GGGG  OOO  SSSS  EEEEE  A   A R   R  CCCC H   H"),
	)
	logo = theme.homeTitle.Render(logo)

	queryBoxStyle := theme.queryBoxBlurred
	if m.queryInput.Focused() {
		queryBoxStyle = theme.queryBoxFocused
	}
	query := lipgloss.PlaceHorizontal(
		m.width,
		lipgloss.Center,
		queryBoxStyle.Render(m.queryInput.View()),
	)

	helpText := "Type a query and press Enter"
	if m.queryInput.Focused() {
		helpText = "Enter search • Esc leave input • q quit"
	}

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		logo,
		"",
		theme.homeSubtle.Render("Search your indexed web corpus"),
		"",
		query,
		"",
		theme.homeSubtle.Render(helpText),
	)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (m tuiModel) resultsScreenView() string {
	theme := newTUITheme()
	lines := make([]string, 0, 24)

	headerRight := ""
	if m.loading {
		headerRight = "Searching..."
	} else if m.lastError != nil {
		headerRight = m.lastError.Error()
	} else if m.status != "" {
		headerRight = m.status
	}

	header := theme.appTitle.Render("gosearch")
	if headerRight != "" {
		right := theme.status.Render(truncateToWidth(headerRight, maxInt(24, m.width-20)))
		if m.lastError != nil {
			right = theme.errorStatus.Render(truncateToWidth(headerRight, maxInt(24, m.width-20)))
		}
		header = lipgloss.JoinHorizontal(lipgloss.Top, header, "  ", right)
	}
	lines = append(lines, header)

	queryBoxStyle := theme.queryBoxBlurred
	if m.queryInput.Focused() {
		queryBoxStyle = theme.queryBoxFocused
	}
	lines = append(lines, queryBoxStyle.Render(m.queryInput.View()))

	if m.activeQuery != "" && !m.loading && m.lastError == nil {
		about := fmt.Sprintf("About %d results (%.2fs)", m.totalCount, m.searchDuration.Seconds())
		if m.cached {
			about += " • cached"
		}
		lines = append(lines, theme.about.Render(about))
	}

	if m.totalCount > len(m.results) && len(m.results) > 0 {
		hint := fmt.Sprintf("Showing first %d loaded results. Increase --limit to browse more.", len(m.results))
		lines = append(lines, theme.hint.Render(hint))
	}

	if len(m.results) == 0 {
		if m.activeQuery == "" {
			lines = append(lines, theme.hint.Render("Type a query and press Enter."))
		} else if len(m.suggestions) > 0 {
			didYouMean := fmt.Sprintf("Did you mean %q? Press y to search.", m.suggestions[0])
			lines = append(lines, theme.suggestion.Render(didYouMean))
		} else {
			lines = append(lines, theme.hint.Render("No results found."))
		}

		footer := "Page 1 / 1 • / query • Enter search • q quit"
		if m.queryInput.Focused() {
			footer = "Enter search • Esc leave input • q quit"
		}
		lines = append(lines, theme.footer.Render(footer))
		return strings.Join(lines, "\n")
	}

	headerBlock := strings.Join(lines, "\n")
	bodyHeight := m.height - lipgloss.Height(headerBlock) - 2
	if bodyHeight < 8 {
		bodyHeight = 8
	}

	leftPaneWidth, rightPaneWidth := m.resultsPaneWidths()
	leftPane := m.renderResultsListPane(leftPaneWidth, bodyHeight)
	rightPane := m.renderInlinePreviewPane(rightPaneWidth, bodyHeight)
	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, "   ", rightPane)
	lines = append(lines, panes)

	footer := fmt.Sprintf("Page %d / %d • / query • Enter full preview • n/p page • j/k move • o open URL • q quit", m.page+1, m.totalPages())
	if m.queryInput.Focused() {
		footer = "Enter search • Esc leave input • q quit"
	}
	lines = append(lines, theme.footer.Render(footer))

	return strings.Join(lines, "\n")
}

func (m tuiModel) resultsPaneWidths() (int, int) {
	gap := 3
	total := m.width - gap
	if total < 70 {
		return 34, 33
	}

	left := (total * 45) / 100
	if left < 34 {
		left = 34
	}

	right := total - left
	if right < 34 {
		right = 34
		left = total - right
	}
	if left < 34 {
		left = 34
	}

	return left, right
}

func (m tuiModel) renderResultsListPane(width int, height int) string {
	theme := newTUITheme()
	lines := make([]string, 0, 64)

	m.clampPage()
	start, end := m.currentPageBounds()
	lines = append(lines, theme.status.Render(fmt.Sprintf("Results %d-%d of %d", start+1, end, m.totalCount)))
	lines = append(lines, theme.separator.Render(strings.Repeat("─", maxInt(8, width-1))))

	pageResults := m.currentPageResults()
	if len(pageResults) == 0 {
		lines = append(lines, theme.hint.Render("No results in this page."))
		return clipTextBlockToLines(strings.Join(lines, "\n"), height)
	}

	// Each rendered card consumes ~4 lines plus one separator line.
	availableLines := height - len(lines)
	if availableLines < 4 {
		availableLines = 4
	}
	cardBlockLines := 5
	visibleCount := availableLines / cardBlockLines
	if visibleCount < 1 {
		visibleCount = 1
	}
	if visibleCount > len(pageResults) {
		visibleCount = len(pageResults)
	}

	selectedInPage := m.selected - start
	if selectedInPage < 0 {
		selectedInPage = 0
	}
	if selectedInPage >= len(pageResults) {
		selectedInPage = len(pageResults) - 1
	}

	windowStart := selectedInPage - (visibleCount / 2)
	if windowStart < 0 {
		windowStart = 0
	}
	maxWindowStart := len(pageResults) - visibleCount
	if windowStart > maxWindowStart {
		windowStart = maxWindowStart
	}
	windowEnd := windowStart + visibleCount
	if windowEnd > len(pageResults) {
		windowEnd = len(pageResults)
	}

	visibleResults := pageResults[windowStart:windowEnd]
	for i, result := range visibleResults {
		absoluteIndex := start + windowStart + i
		isSelected := absoluteIndex == m.selected

		cardStyle := theme.cardBorder
		titleStyle := theme.title
		if isSelected {
			cardStyle = theme.cardBorderActive
			titleStyle = theme.titleActive
		}

		title := titleStyle.Render(truncateToWidth(result.Title, maxInt(12, width-6)))
		urlLine := theme.url.Render(truncateToWidth(result.URL, maxInt(12, width-6)))
		snippetLines := wrapText(resultSnippet(result), maxInt(12, width-6), 2)
		cardLines := []string{title, urlLine}
		for _, snippetLine := range snippetLines {
			cardLines = append(cardLines, theme.snippet.Render(snippetLine))
		}

		lines = append(lines, cardStyle.Render(strings.Join(cardLines, "\n")))
		if i < len(visibleResults)-1 {
			lines = append(lines, theme.separator.Render(strings.Repeat("─", maxInt(8, width-1))))
		}
	}

	content := strings.Join(lines, "\n")
	return clipTextBlockToLines(content, height)
}

func (m tuiModel) renderInlinePreviewPane(width int, height int) string {
	theme := newTUITheme()
	lines := make([]string, 0, 64)

	selected := m.selectedResult()
	if selected == nil {
		lines = append(lines, theme.docTitle.Render("Preview"))
		lines = append(lines, theme.hint.Render("Select a result to see its content preview."))
		return clipTextBlockToLines(strings.Join(lines, "\n"), height)
	}

	title := strings.TrimSpace(selected.Title)
	if title == "" {
		title = "(untitled)"
	}
	lines = append(lines, theme.docTitle.Render(truncateToWidth(title, maxInt(12, width-2))))
	lines = append(lines, theme.docURL.Render(truncateToWidth(selected.URL, maxInt(12, width-2))))
	lines = append(lines, theme.separator.Render(strings.Repeat("─", maxInt(8, width-1))))

	if m.inlinePreviewLoading {
		lines = append(lines, theme.status.Render("Loading preview..."))
	}
	if m.inlinePreviewError != nil {
		lines = append(lines, theme.errorStatus.Render(truncateToWidth(m.inlinePreviewError.Error(), maxInt(12, width-2))))
		return clipTextBlockToLines(strings.Join(lines, "\n"), height)
	}

	preview := strings.TrimSpace(m.inlinePreviewRendered)
	if preview == "" {
		if m.inlinePreviewLoading {
			return clipTextBlockToLines(strings.Join(lines, "\n"), height)
		}
		lines = append(lines, theme.hint.Render("Preview unavailable for this result."))
		return clipTextBlockToLines(strings.Join(lines, "\n"), height)
	}

	lines = append(lines, clipTextBlockToLines(preview, maxInt(1, height-len(lines))))
	return clipTextBlockToLines(strings.Join(lines, "\n"), height)
}

func (m tuiModel) documentScreenView() string {
	theme := newTUITheme()
	lines := make([]string, 0, 12)

	title := "(untitled)"
	url := ""
	if m.currentDocument != nil {
		if trimmedTitle := strings.TrimSpace(m.currentDocument.Title); trimmedTitle != "" {
			title = trimmedTitle
		}
		url = m.currentDocument.URL
	}

	lines = append(lines, theme.docTitle.Render("Preview"))
	lines = append(lines, theme.docTitle.Render(truncateToWidth(title, maxInt(20, m.width-4))))
	lines = append(lines, theme.docURL.Render(truncateToWidth(url, maxInt(20, m.width-4))))

	if m.loading {
		lines = append(lines, theme.status.Render("Loading preview..."))
	} else if m.lastError != nil {
		lines = append(lines, theme.errorStatus.Render(truncateToWidth(m.lastError.Error(), maxInt(20, m.width-4))))
	} else {
		lines = append(lines, theme.status.Render("up/down/pgup/pgdn scroll • b back • o open URL • / query • q quit"))
	}

	lines = append(lines, theme.separator.Render(strings.Repeat("─", maxInt(8, m.width-2))))
	lines = append(lines, m.documentView.View())

	return strings.Join(lines, "\n")
}

func tuiRunSearchCmd(runtime *searchRuntime, query string, limit int, style string, previewWidth int) tea.Cmd {
	query = strings.TrimSpace(query)

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response, err := runtime.Search(ctx, query, limit)
		if err != nil {
			return tuiSearchCompletedMsg{query: query, err: err}
		}
		enrichResultsWithContentSnippets(runtime, response.Results, query)

		var suggestions []string
		if response.TotalCount == 0 {
			suggestions, _ = runtime.Suggest(ctx, query, 3)
		}

		preloaded, preloadDone, preloadErr := buildInlinePreviewEntries(runtime, response.Results, style, previewWidth, 100)

		return tuiSearchCompletedMsg{
			query:       query,
			response:    response,
			suggestions: suggestions,
			preloaded:   preloaded,
			preloadDone: preloadDone,
			preloadErr:  preloadErr,
		}
	}
}

func tuiOpenDocumentCmd(runtime *searchRuntime, result *search.Result, style string, width int) tea.Cmd {
	resultCopy := *result

	return func() tea.Msg {
		document, err := runtime.GetDocument(resultCopy.DocID)
		if err != nil {
			return tuiDocumentOpenedMsg{err: fmt.Errorf("failed to load document: %w", err)}
		}

		rendered, err := renderMarkdownForTUI(document.ContentMarkdown, style, width)
		if err != nil {
			return tuiDocumentOpenedMsg{err: fmt.Errorf("failed to render markdown: %w", err)}
		}

		return tuiDocumentOpenedMsg{
			result:   resultCopy,
			document: document,
			rendered: rendered,
		}
	}
}

func tuiLoadInlinePreviewCmd(runtime *searchRuntime, result *search.Result, style string, width int, requestID int) tea.Cmd {
	resultCopy := *result

	return func() tea.Msg {
		document, err := runtime.GetDocument(resultCopy.DocID)
		if err != nil {
			return tuiInlinePreviewLoadedMsg{
				requestID: requestID,
				result:    resultCopy,
				err:       fmt.Errorf("failed to load document: %w", err),
			}
		}

		rendered, err := renderMarkdownForTUI(markdownInlinePreviewContent(document.ContentMarkdown), style, width)
		if err != nil {
			return tuiInlinePreviewLoadedMsg{
				requestID: requestID,
				result:    resultCopy,
				err:       fmt.Errorf("failed to render markdown: %w", err),
			}
		}

		return tuiInlinePreviewLoadedMsg{
			requestID:   requestID,
			result:      resultCopy,
			document:    document,
			rendered:    rendered,
			renderWidth: width,
		}
	}
}

func tuiPreloadInlinePreviewsCmd(runtime *searchRuntime, results []search.Result, style string, width int, requestID int) tea.Cmd {
	resultCopies := make([]search.Result, len(results))
	copy(resultCopies, results)

	return func() tea.Msg {
		resultsPtr := make([]*search.Result, 0, len(resultCopies))
		for i := range resultCopies {
			resultsPtr = append(resultsPtr, &resultCopies[i])
		}

		entries, loaded, err := buildInlinePreviewEntries(runtime, resultsPtr, style, width, len(resultsPtr))

		return tuiInlinePreviewPreloadedMsg{
			requestID: requestID,
			entries:   entries,
			loaded:    loaded,
			err:       err,
		}
	}
}

func buildInlinePreviewEntries(runtime *searchRuntime, results []*search.Result, style string, width int, maxCount int) (map[string]tuiInlinePreviewCacheEntry, int, error) {
	entries := make(map[string]tuiInlinePreviewCacheEntry)
	if runtime == nil || len(results) == 0 || maxCount <= 0 {
		return entries, 0, nil
	}
	if width <= 0 {
		width = 70
	}

	loaded := 0
	seen := make(map[string]struct{}, maxCount)
	for _, result := range results {
		if loaded >= maxCount {
			break
		}
		if result == nil {
			continue
		}
		docID := strings.TrimSpace(result.DocID)
		if docID == "" {
			continue
		}
		if _, exists := seen[docID]; exists {
			continue
		}
		seen[docID] = struct{}{}

		document, err := runtime.GetDocument(docID)
		if err != nil || document == nil {
			continue
		}

		rendered, err := renderMarkdownForTUI(markdownInlinePreviewContent(document.ContentMarkdown), style, width)
		if err != nil {
			continue
		}

		entries[docID] = tuiInlinePreviewCacheEntry{
			document:    document,
			rendered:    rendered,
			renderWidth: width,
		}
		loaded++
	}

	return entries, loaded, nil
}

func renderMarkdownForTUI(markdown string, style string, width int) (string, error) {
	// Avoid glamour auto-style terminal probing in TUI; some terminals emit OSC
	// responses that can leak into input handling on startup.
	tuiStyle := strings.TrimSpace(style)
	if tuiStyle == "" || strings.EqualFold(tuiStyle, "auto") {
		tuiStyle = "dark"
	}
	return renderMarkdownForTerminal(markdown, tuiStyle, width)
}

func tuiInlinePreviewTimeoutCmd(requestID int, timeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(timeout)
		return tuiInlinePreviewTimeoutMsg{requestID: requestID}
	}
}

func markdownInlinePreviewContent(markdown string) string {
	const maxLines = 140
	const maxChars = 12000

	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" {
		return ""
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		trimmed = strings.Join(lines, "\n")
	}

	runes := []rune(trimmed)
	if len(runes) > maxChars {
		trimmed = string(runes[:maxChars]) + "\n\n..."
	}

	return trimmed
}

func tuiOpenURLCmd(url string) tea.Cmd {
	url = strings.TrimSpace(url)
	return func() tea.Msg {
		err := openURLInBrowser(url)
		return tuiURLOpenedMsg{
			url: url,
			err: err,
		}
	}
}

func resultSnippet(result *search.Result) string {
	if result == nil {
		return "No preview text available for this result."
	}

	snippet := strings.TrimSpace(result.Snippet)
	if snippet == "" {
		return "No preview text available for this result."
	}

	return snippet
}

func wrapText(value string, maxWidth int, maxLines int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{""}
	}
	if maxWidth < 10 {
		maxWidth = 10
	}
	if maxLines < 1 {
		maxLines = 1
	}

	words := strings.Fields(value)
	if len(words) == 0 {
		return []string{truncateToWidth(value, maxWidth)}
	}

	lines := make([]string, 0, maxLines)
	current := words[0]
	truncated := false

	for _, word := range words[1:] {
		candidate := current + " " + word
		if lipgloss.Width(candidate) <= maxWidth {
			current = candidate
			continue
		}

		lines = append(lines, current)
		if len(lines) >= maxLines {
			truncated = true
			break
		}

		current = word
	}

	if !truncated && current != "" {
		lines = append(lines, current)
	}

	if len(lines) == 0 {
		lines = append(lines, truncateToWidth(value, maxWidth))
	}

	if len(lines) > maxLines {
		lines = lines[:maxLines]
		truncated = true
	}

	if truncated {
		last := lines[len(lines)-1]
		if !strings.HasSuffix(last, "...") {
			last = truncateToWidth(last, maxWidth-3) + "..."
		}
		lines[len(lines)-1] = last
	}

	for idx := range lines {
		lines[idx] = truncateToWidth(lines[idx], maxWidth)
	}

	return lines
}

func totalPages(totalItems int, pageSize int) int {
	if pageSize < 1 {
		pageSize = 10
	}
	if totalItems <= 0 {
		return 1
	}
	pages := totalItems / pageSize
	if totalItems%pageSize != 0 {
		pages++
	}
	if pages < 1 {
		pages = 1
	}
	return pages
}

func pageBounds(page int, pageSize int, totalItems int) (int, int) {
	if pageSize < 1 {
		pageSize = 10
	}
	if totalItems <= 0 {
		return 0, 0
	}

	pages := totalPages(totalItems, pageSize)
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}

	start := page * pageSize
	if start > totalItems {
		start = totalItems
	}

	end := start + pageSize
	if end > totalItems {
		end = totalItems
	}

	return start, end
}

func truncateToWidth(value string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= maxWidth {
		return value
	}

	if maxWidth <= 3 {
		return string(runes[:maxWidth])
	}
	return string(runes[:maxWidth-3]) + "..."
}

func clipTextBlockToLines(value string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}

	lines := strings.Split(value, "\n")
	if len(lines) <= maxLines {
		return value
	}

	clipped := make([]string, 0, maxLines)
	for i := 0; i < maxLines; i++ {
		clipped = append(clipped, lines[i])
	}
	return strings.Join(clipped, "\n")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

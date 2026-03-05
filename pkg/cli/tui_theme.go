package cli

import "github.com/charmbracelet/lipgloss"

type tuiTheme struct {
	appTitle         lipgloss.Style
	status           lipgloss.Style
	errorStatus      lipgloss.Style
	about            lipgloss.Style
	hint             lipgloss.Style
	homeTitle        lipgloss.Style
	homeAccentBlue   lipgloss.Style
	homeAccentRed    lipgloss.Style
	homeAccentYellow lipgloss.Style
	homeAccentGreen  lipgloss.Style
	homeSubtle       lipgloss.Style
	queryBoxFocused  lipgloss.Style
	queryBoxBlurred  lipgloss.Style
	cardBorder       lipgloss.Style
	cardBorderActive lipgloss.Style
	title            lipgloss.Style
	titleActive      lipgloss.Style
	url              lipgloss.Style
	snippet          lipgloss.Style
	separator        lipgloss.Style
	footer           lipgloss.Style
	suggestion       lipgloss.Style
	docTitle         lipgloss.Style
	docURL           lipgloss.Style
}

func newTUITheme() tuiTheme {
	googleBlue := lipgloss.AdaptiveColor{Light: "#1A0DAB", Dark: "#8AB4F8"}
	googleGreen := lipgloss.AdaptiveColor{Light: "#188038", Dark: "#7FB685"}
	textMuted := lipgloss.AdaptiveColor{Light: "#5F6368", Dark: "#9AA0A6"}
	textPrimary := lipgloss.AdaptiveColor{Light: "#202124", Dark: "#E8EAED"}
	errorColor := lipgloss.AdaptiveColor{Light: "#C5221F", Dark: "#F28B82"}
	focusBorder := lipgloss.AdaptiveColor{Light: "#1A73E8", Dark: "#8AB4F8"}
	cardBorder := lipgloss.AdaptiveColor{Light: "#DADCE0", Dark: "#3C4043"}
	activeCard := lipgloss.AdaptiveColor{Light: "#AECBFA", Dark: "#5F89B3"}
	activeCardBackground := lipgloss.AdaptiveColor{Light: "#E8F0FE", Dark: "#1E2A3A"}

	return tuiTheme{
		appTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(googleBlue),
		status: lipgloss.NewStyle().
			Foreground(textMuted),
		errorStatus: lipgloss.NewStyle().
			Bold(true).
			Foreground(errorColor),
		about: lipgloss.NewStyle().
			Foreground(textMuted),
		hint: lipgloss.NewStyle().
			Foreground(textMuted),
		homeTitle: lipgloss.NewStyle().
			Bold(true),
		homeAccentBlue: lipgloss.NewStyle().
			Bold(true).
			Foreground(googleBlue),
		homeAccentRed: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#D93025", Dark: "#F28B82"}),
		homeAccentYellow: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#F9AB00", Dark: "#FDD663"}),
		homeAccentGreen: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#188038", Dark: "#81C995"}),
		homeSubtle: lipgloss.NewStyle().
			Foreground(textMuted),
		queryBoxFocused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(focusBorder).
			Padding(0, 1),
		queryBoxBlurred: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cardBorder).
			Padding(0, 1),
		cardBorder: lipgloss.NewStyle().
			BorderLeft(true).
			BorderForeground(cardBorder).
			PaddingLeft(1),
		cardBorderActive: lipgloss.NewStyle().
			BorderLeft(true).
			BorderForeground(activeCard).
			Background(activeCardBackground).
			PaddingLeft(1),
		title: lipgloss.NewStyle().
			Foreground(googleBlue),
		titleActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(googleBlue),
		url: lipgloss.NewStyle().
			Foreground(googleGreen),
		snippet: lipgloss.NewStyle().
			Foreground(textPrimary),
		separator: lipgloss.NewStyle().
			Foreground(cardBorder),
		footer: lipgloss.NewStyle().
			Foreground(textMuted),
		suggestion: lipgloss.NewStyle().
			Bold(true).
			Foreground(googleBlue),
		docTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(textPrimary),
		docURL: lipgloss.NewStyle().
			Foreground(googleGreen),
	}
}

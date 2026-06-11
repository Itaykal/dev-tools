package tui

import "github.com/charmbracelet/lipgloss"

// Palette mirrors vars/fzf.zsh so the picker feels like the rest of the
// dotfiles, then leans on Lipgloss for a more polished, modern frame.
var (
	accent    = lipgloss.Color("#f0abfc") // fuchsia — the signature color
	selBg     = lipgloss.Color("#3d1a36") // dark purple selection background
	fg        = lipgloss.Color("#bbbbbb")
	fgBright  = lipgloss.Color("#ffffff")
	borderCol = lipgloss.Color("#2a2a2a")
	muted     = lipgloss.Color("#888888")
	dim       = lipgloss.Color("#555555")
)

var (
	// Frame around each pane.
	leftFrame = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderCol).
			Padding(0, 1)

	previewFrame = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderCol).
			Padding(0, 1)

	// Preview pane label (accent, like fzf's preview-label).
	previewLabelStyle = lipgloss.NewStyle().Foreground(accent)

	// matchStyle highlights fuzzy letter matches (fzf's hl color).
	matchStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)

	// sectionStyle heads a group in the help panel.
	sectionStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	helpKeyStyle = lipgloss.NewStyle().Foreground(fgBright)

	// Query line.
	promptStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)

	// Rows.
	rowStyle    = lipgloss.NewStyle().Foreground(fg)
	rowSelStyle = lipgloss.NewStyle().Foreground(fgBright).Background(selBg)
	typeStyle   = lipgloss.NewStyle().Foreground(muted)
	keyStyle    = lipgloss.NewStyle().Foreground(accent)
	statusStyle = lipgloss.NewStyle().Foreground(dim)

	// Footer keybar + helpers.
	footerKey   = lipgloss.NewStyle().Foreground(accent)
	footerLabel = lipgloss.NewStyle().Foreground(muted)
	footerSep   = lipgloss.NewStyle().Foreground(dim)

	// Misc.
	dimStyle    = lipgloss.NewStyle().Foreground(dim)
	mutedStyle  = lipgloss.NewStyle().Foreground(muted)
	accentStyle = lipgloss.NewStyle().Foreground(accent)
)

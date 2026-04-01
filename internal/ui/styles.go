// Package ui provides styles, keybindings, themes, and clipboard
// utilities for the Wharf TUI.
package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Dark, muted palette — clean and professional
var (
	ColorPrimary   = lipgloss.Color("#5FAFFF")  // soft blue — logo, crumbs, selected
	ColorSecondary = lipgloss.Color("#AFAFD7")  // lavender — project titles
	ColorSuccess   = lipgloss.Color("#87D787")  // soft green — running
	ColorWarning   = lipgloss.Color("#D7D75F")  // soft yellow — partial
	ColorDanger    = lipgloss.Color("#FF5F87")  // soft red — stopped, errors
	ColorMuted     = lipgloss.Color("#808080")  // gray — headers, secondary text, menu
	ColorHighlight = lipgloss.Color("#AFAFD7")  // lavender — menu hotkeys
	ColorBorder    = lipgloss.Color("#444444")  // dark gray — separators
)

// Logo bar
var (
	LogoStyle    = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	CrumbStyle   = lipgloss.NewStyle().Foreground(ColorMuted)
	InfoBarStyle = lipgloss.NewStyle().Foreground(ColorMuted)
)

// Menu bar — subtle, not distracting
var (
	MenuKeyStyle  = lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight)
	MenuTextStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	MenuBarStyle  = lipgloss.NewStyle().Padding(0, 1)
)

// Content
var (
	ContentStyle = lipgloss.NewStyle().Padding(0, 1)

	SelectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Background(lipgloss.Color("#264F78")).
				Foreground(lipgloss.Color("#FFFFFF"))

	HeaderRowStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorMuted)

	ProjectTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary).
				Padding(0, 1)
)

// Status
var (
	RunningStyle = lipgloss.NewStyle().Foreground(ColorSuccess)
	PartialStyle = lipgloss.NewStyle().Foreground(ColorWarning)
	StoppedStyle = lipgloss.NewStyle().Foreground(ColorDanger)
)

// General
var (
	ErrorStyle       = lipgloss.NewStyle().Foreground(ColorDanger)
	MutedStyle       = lipgloss.NewStyle().Foreground(ColorMuted)
	FilterInputStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
	BookmarkStyle        = lipgloss.NewStyle().Foreground(ColorWarning)
	SearchHighlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("#3A3A00")).Foreground(lipgloss.Color("#FFFF00"))
	CommandStyle     = lipgloss.NewStyle().Foreground(ColorPrimary)
	StatusLineStyle  = lipgloss.NewStyle().Foreground(ColorMuted).Padding(0, 1)
)

// Separator renders a horizontal line of the given width.
func Separator(width int) string {
	return lipgloss.NewStyle().
		Foreground(ColorBorder).
		Render(strings.Repeat("─", width))
}

// FormatMenuItem formats a k9s-style menu item: <key>text.
func FormatMenuItem(key, text string) string {
	return MenuKeyStyle.Render("<"+key+">") + MenuTextStyle.Render(text)
}

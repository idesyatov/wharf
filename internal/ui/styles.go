package ui

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary   = lipgloss.Color("#5FAFFF")
	ColorSecondary = lipgloss.Color("#AFAFD7")
	ColorSuccess   = lipgloss.Color("#87D787")
	ColorWarning   = lipgloss.Color("#D7D75F")
	ColorDanger    = lipgloss.Color("#FF5F87")
	ColorMuted     = lipgloss.Color("#636363")
)

var (
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(ColorMuted)

	FooterStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(ColorMuted)

	ContentStyle = lipgloss.NewStyle().
			Padding(1, 2)

	SelectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Background(lipgloss.Color("#333333")).
				Foreground(lipgloss.Color("#FFFFFF"))

	HeaderRowStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorMuted)

	RunningStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	PartialStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	StoppedStyle = lipgloss.NewStyle().
			Foreground(ColorDanger)

	ProjectTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary).
				Padding(0, 1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	FilterInputStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary)
)

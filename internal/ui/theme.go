package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

type Theme struct {
	Colors ThemeColors `yaml:"colors"`
}

type ThemeColors struct {
	Primary    string `yaml:"primary"`
	Secondary  string `yaml:"secondary"`
	Success    string `yaml:"success"`
	Warning    string `yaml:"warning"`
	Danger     string `yaml:"danger"`
	Muted      string `yaml:"muted"`
	Highlight  string `yaml:"highlight"`
	Border     string `yaml:"border"`
	SelectedBg string `yaml:"selected_bg"`
	SelectedFg string `yaml:"selected_fg"`
}

var builtinDark = Theme{
	Colors: ThemeColors{
		Primary:    "#5FAFFF",
		Secondary:  "#AFAFD7",
		Success:    "#87D787",
		Warning:    "#D7D75F",
		Danger:     "#FF5F87",
		Muted:      "#808080",
		Highlight:  "#AFAFD7",
		Border:     "#444444",
		SelectedBg: "#264F78",
		SelectedFg: "#FFFFFF",
	},
}

var builtinLight = Theme{
	Colors: ThemeColors{
		Primary:    "#0550AE",
		Secondary:  "#656D76",
		Success:    "#1A7F37",
		Warning:    "#9A6700",
		Danger:     "#CF222E",
		Muted:      "#656D76",
		Highlight:  "#0550AE",
		Border:     "#D0D7DE",
		SelectedBg: "#DDF4FF",
		SelectedFg: "#000000",
	},
}

func LoadTheme(name string) (*Theme, error) {
	switch name {
	case "", "auto", "dark":
		t := builtinDark
		return &t, nil
	case "light":
		t := builtinLight
		return &t, nil
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "wharf", "themes", name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load theme %s: %w", name, err)
	}
	var theme Theme
	if err := yaml.Unmarshal(data, &theme); err != nil {
		return nil, fmt.Errorf("parse theme %s: %w", name, err)
	}
	return &theme, nil
}

func ApplyTheme(theme *Theme) {
	c := theme.Colors
	if c.Primary != "" {
		ColorPrimary = lipgloss.Color(c.Primary)
	}
	if c.Secondary != "" {
		ColorSecondary = lipgloss.Color(c.Secondary)
	}
	if c.Success != "" {
		ColorSuccess = lipgloss.Color(c.Success)
	}
	if c.Warning != "" {
		ColorWarning = lipgloss.Color(c.Warning)
	}
	if c.Danger != "" {
		ColorDanger = lipgloss.Color(c.Danger)
	}
	if c.Muted != "" {
		ColorMuted = lipgloss.Color(c.Muted)
	}
	if c.Highlight != "" {
		ColorHighlight = lipgloss.Color(c.Highlight)
	}
	if c.Border != "" {
		ColorBorder = lipgloss.Color(c.Border)
	}

	// Rebuild styles with new colors
	LogoStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	CrumbStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	InfoBarStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	MenuKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight)
	MenuTextStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	HeaderRowStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorMuted)
	ProjectTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Padding(0, 1)
	RunningStyle = lipgloss.NewStyle().Foreground(ColorSuccess)
	PartialStyle = lipgloss.NewStyle().Foreground(ColorWarning)
	StoppedStyle = lipgloss.NewStyle().Foreground(ColorDanger)
	ErrorStyle = lipgloss.NewStyle().Foreground(ColorDanger)
	MutedStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	FilterInputStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
	BookmarkStyle = lipgloss.NewStyle().Foreground(ColorWarning)
	CommandStyle = lipgloss.NewStyle().Foreground(ColorPrimary)

	selBg := lipgloss.Color("#264F78")
	selFg := lipgloss.Color("#FFFFFF")
	if c.SelectedBg != "" {
		selBg = lipgloss.Color(c.SelectedBg)
	}
	if c.SelectedFg != "" {
		selFg = lipgloss.Color(c.SelectedFg)
	}
	SelectedRowStyle = lipgloss.NewStyle().Bold(true).Background(selBg).Foreground(selFg)
}

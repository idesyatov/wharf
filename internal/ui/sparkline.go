package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Sparkline renders a mini sparkline chart from a slice of values.
func Sparkline(values []float64, maxVal float64) string {
	bars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	if maxVal <= 0 {
		maxVal = 1
	}
	result := make([]rune, len(values))
	for i, v := range values {
		idx := int(v / maxVal * float64(len(bars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(bars) {
			idx = len(bars) - 1
		}
		result[i] = bars[idx]
	}
	return string(result)
}

// ColoredSparkline renders a sparkline with auto-scaled bar height
// and color based on absolute load percentage.
// scaleMax controls bar height (use max of history for trend visibility).
// colorMax controls color thresholds (use absolute limit for load level).
func ColoredSparkline(values []float64, scaleMax float64, colorMax float64) string {
	bars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	if scaleMax <= 0 {
		scaleMax = 1
	}
	if colorMax <= 0 {
		colorMax = 1
	}
	var result string
	for _, v := range values {
		idx := int(v / scaleMax * float64(len(bars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(bars) {
			idx = len(bars) - 1
		}
		char := string(bars[idx])

		pct := v / colorMax * 100
		var style lipgloss.Style
		switch {
		case pct < 30:
			style = lipgloss.NewStyle().Foreground(ColorSuccess)
		case pct < 70:
			style = lipgloss.NewStyle().Foreground(ColorWarning)
		default:
			style = lipgloss.NewStyle().Foreground(ColorDanger)
		}
		result += style.Render(char)
	}
	return result
}

// ProgressBar renders a colored progress bar [████░░░░░░] with percentage.
func ProgressBar(value, max float64, width int) string {
	if max <= 0 {
		max = 1
	}
	pct := value / max
	if pct > 1 {
		pct = 1
	}
	if pct < 0 {
		pct = 0
	}

	filled := int(pct * float64(width))
	empty := width - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	var style lipgloss.Style
	switch {
	case pct < 0.3:
		style = lipgloss.NewStyle().Foreground(ColorSuccess)
	case pct < 0.7:
		style = lipgloss.NewStyle().Foreground(ColorWarning)
	default:
		style = lipgloss.NewStyle().Foreground(ColorDanger)
	}

	return style.Render("["+bar+"]") + fmt.Sprintf(" %.0f%%", pct*100)
}

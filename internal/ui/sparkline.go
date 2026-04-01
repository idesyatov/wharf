package ui

import "github.com/charmbracelet/lipgloss"

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

// ColoredSparkline renders a sparkline where each bar is colored
// by load level: green (<30%), yellow (30-70%), red (>70%).
func ColoredSparkline(values []float64, maxVal float64) string {
	bars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	if maxVal <= 0 {
		maxVal = 1
	}
	var result string
	for _, v := range values {
		idx := int(v / maxVal * float64(len(bars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(bars) {
			idx = len(bars) - 1
		}
		char := string(bars[idx])

		pct := v / maxVal * 100
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

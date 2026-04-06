package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/ui"
)

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func renderSelectedRow(text string, width int) string {
	for lipgloss.Width(text) < width {
		text += " "
	}
	return ui.SelectedRowStyle.Render(text)
}

func padRight(s string, width int) string {
	visible := lipgloss.Width(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

func formatCPU(percent float64) string {
	if percent < 10 {
		return fmt.Sprintf("%.1f%%", percent)
	}
	return fmt.Sprintf("%.0f%%", percent)
}

func formatMemory(bytes uint64) string {
	const (
		ki = 1024
		mi = ki * 1024
		gi = mi * 1024
	)
	switch {
	case bytes >= gi:
		return fmt.Sprintf("%.1fGi", float64(bytes)/float64(gi))
	case bytes >= mi:
		return fmt.Sprintf("%dMi", bytes/mi)
	case bytes >= ki:
		return fmt.Sprintf("%dKi", bytes/ki)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// FormatBytes formats byte counts for display (exported, used by app.go).
func FormatBytes(bytes uint64) string {
	const (
		ki = 1024
		mi = ki * 1024
		gi = mi * 1024
	)
	switch {
	case bytes >= gi:
		return fmt.Sprintf("%.1fGi", float64(bytes)/float64(gi))
	case bytes >= mi:
		return fmt.Sprintf("%dMi", bytes/mi)
	case bytes >= ki:
		return fmt.Sprintf("%dKi", bytes/ki)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case d < 365*24*time.Hour:
		months := int(d.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(d.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

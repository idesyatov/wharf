package views

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/ui"
)

type SwitchToTopProjectMsg struct {
	Project docker.Project
}

type SwitchToTopContainerMsg struct {
	ContainerID   string
	ContainerName string
	Image         string
}

type SwitchBackFromTopMsg struct{}

type TopStatsLoadedMsg struct {
	Stats map[string]docker.Stats
}

type TopView struct {
	projectName   string
	containerID   string
	containerName string
	image         string
	project       docker.Project
	stats         map[string]docker.Stats
	width, height int
	cursor        int
	scroll        int
}

func NewTopViewProject(project docker.Project) TopView {
	return TopView{
		projectName: project.Name,
		project:     project,
	}
}

func NewTopViewContainer(containerID, containerName, image string) TopView {
	return TopView{
		containerID:   containerID,
		containerName: containerName,
		image:         image,
	}
}

func (v TopView) IsProjectMode() bool     { return v.containerID == "" }
func (v TopView) Project() docker.Project { return v.project }
func (v TopView) ContainerID() string { return v.containerID }
func (v TopView) HasStats() bool      { return len(v.stats) > 0 }

func (v TopView) Breadcrumb() string {
	if v.IsProjectMode() {
		return "› " + v.projectName + " [TOP]"
	}
	return "› " + v.containerName + " [TOP]"
}

func (v TopView) SetSize(w, h int) TopView {
	v.width = w
	v.height = h
	return v
}

func (v TopView) UpdateStats(stats map[string]docker.Stats) TopView {
	v.stats = stats
	return v
}

func (v TopView) Update(msg tea.Msg, keys ui.KeyMap) (TopView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case ui.MatchKey(msg, keys.Left):
			return v, func() tea.Msg { return SwitchBackFromTopMsg{} }
		case ui.MatchKey(msg, keys.Down):
			if v.IsProjectMode() {
				v.cursor++
			} else {
				v.scroll++
			}
		case ui.MatchKey(msg, keys.Up):
			if v.IsProjectMode() {
				if v.cursor > 0 {
					v.cursor--
				}
			} else {
				if v.scroll > 0 {
					v.scroll--
				}
			}
		}
	}
	return v, nil
}

func (v TopView) View() string {
	if v.IsProjectMode() {
		return v.renderProject()
	}
	return v.renderContainer()
}

func shouldShowSparkline(values []float64, threshold float64) bool {
	for _, v := range values {
		if v > threshold {
			return true
		}
	}
	return false
}

func (v TopView) renderProject() string {
	type entry struct {
		name string
		id   string
	}

	var containers []entry
	for _, svc := range v.project.Services {
		for _, ct := range svc.Containers {
			containers = append(containers, entry{name: ct.Name, id: ct.ID})
		}
	}

	if len(containers) == 0 {
		return ui.MutedStyle.Render("No containers")
	}

	// Clamp cursor
	if v.cursor >= len(containers) {
		v.cursor = len(containers) - 1
	}

	colName := 24
	colCPU := 8
	colMem := 10
	colNetRx := 10
	colNetTx := 10

	header := ui.HeaderRowStyle.Render(
		fmt.Sprintf("  %-*s %*s %*s %*s %*s",
			colName, "CONTAINER",
			colCPU, "CPU",
			colMem, "MEM",
			colNetRx, "NET RX",
			colNetTx, "NET TX"),
	)

	var rows []string
	rows = append(rows, header)

	var totalCPU float64
	var totalMem uint64

	for i, ct := range containers {
		s := v.stats[ct.id]
		totalCPU += s.CPUPercent
		totalMem += s.MemUsage

		row := fmt.Sprintf("  %-*s %*s %*s %*s %*s",
			colName, truncate(ct.name, colName),
			colCPU, formatCPU(s.CPUPercent),
			colMem, formatMemory(s.MemUsage),
			colNetRx, formatMemory(s.NetRx),
			colNetTx, formatMemory(s.NetTx),
		)

		if i == v.cursor {
			rows = append(rows, renderSelectedRow(row, v.width-2))
		} else {
			rows = append(rows, row)
		}
	}

	// Separator + totals
	sep := ui.MutedStyle.Render("  " + strings.Repeat("─", colName+colCPU+colMem+colNetRx+colNetTx+4))
	rows = append(rows, sep)
	totals := fmt.Sprintf("  %-*s %*s %*s",
		colName, "Total",
		colCPU, formatCPU(totalCPU),
		colMem, formatMemory(totalMem))
	rows = append(rows, ui.MutedStyle.Render(totals))

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func colorByLevel(value, max float64) lipgloss.Style {
	pct := value / max * 100
	switch {
	case pct < 30:
		return lipgloss.NewStyle().Foreground(ui.ColorSuccess)
	case pct < 70:
		return lipgloss.NewStyle().Foreground(ui.ColorWarning)
	default:
		return lipgloss.NewStyle().Foreground(ui.ColorDanger)
	}
}

func (v TopView) renderContainer() string {
	s := v.stats[v.containerID]

	labelStyle := ui.MutedStyle
	valueStyle := lipgloss.NewStyle().Foreground(ui.ColorPrimary)

	cpuStyle := colorByLevel(s.CPUPercent, 100)

	memPct := 0.0
	if s.MemLimit > 0 {
		memPct = float64(s.MemUsage) / float64(s.MemLimit) * 100
	}
	memStyle := colorByLevel(memPct, 100)

	lines := []string{
		fmt.Sprintf("  %s  %s", labelStyle.Render("Container:"), valueStyle.Render(v.containerName)),
		fmt.Sprintf("  %s  %s", labelStyle.Render("Image:    "), valueStyle.Render(v.image)),
		"",
		fmt.Sprintf("  %s  %s", labelStyle.Render("CPU:      "), cpuStyle.Render(formatCPU(s.CPUPercent))),
		"",
		fmt.Sprintf("  %s  %s / %s (%s)", labelStyle.Render("Memory:   "),
			memStyle.Render(formatMemory(s.MemUsage)),
			formatMemory(s.MemLimit),
			memStyle.Render(fmt.Sprintf("%.1f%%", memPct))),
		"",
		fmt.Sprintf("  %s  %s", labelStyle.Render("Net RX:   "), formatMemory(s.NetRx)),
		fmt.Sprintf("  %s  %s", labelStyle.Render("Net TX:   "), formatMemory(s.NetTx)),
	}

	visible := v.height - 2
	if visible < 1 {
		visible = len(lines)
	}
	start := v.scroll
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + visible
	if end > len(lines) {
		end = len(lines)
	}

	return strings.Join(lines[start:end], "\n")
}

func LoadTopStats(client *docker.Client, project docker.Project) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return TopStatsLoadedMsg{}
		}
		ctx := context.Background()
		result := make(map[string]docker.Stats)
		for _, svc := range project.Services {
			for _, ct := range svc.Containers {
				if ct.Status != "running" {
					continue
				}
				s, err := client.ContainerStats(ctx, ct.ID)
				if err == nil {
					result[ct.ID] = s
				}
			}
		}
		return TopStatsLoadedMsg{Stats: result}
	}
}

func LoadTopContainerStats(client *docker.Client, containerID string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return TopStatsLoadedMsg{}
		}
		ctx := context.Background()
		result := make(map[string]docker.Stats)
		s, err := client.ContainerStats(ctx, containerID)
		if err == nil {
			result[containerID] = s
		}
		return TopStatsLoadedMsg{Stats: result}
	}
}

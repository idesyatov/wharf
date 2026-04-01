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

const topHistorySize = 20

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
	cpuHistory    map[string][]float64
	memHistory    map[string][]float64
	width, height int
	cursor        int
	scroll        int
}

func NewTopViewProject(project docker.Project) TopView {
	return TopView{
		projectName: project.Name,
		project:     project,
		cpuHistory:  make(map[string][]float64),
		memHistory:  make(map[string][]float64),
	}
}

func NewTopViewContainer(containerID, containerName, image string) TopView {
	return TopView{
		containerID:   containerID,
		containerName: containerName,
		image:         image,
		cpuHistory:    make(map[string][]float64),
		memHistory:    make(map[string][]float64),
	}
}

func (v TopView) IsProjectMode() bool { return v.containerID == "" }
func (v TopView) Project() docker.Project { return v.project }
func (v TopView) ContainerID() string     { return v.containerID }

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
	for id, s := range stats {
		cpu := v.cpuHistory[id]
		cpu = append(cpu, s.CPUPercent)
		if len(cpu) > topHistorySize {
			cpu = cpu[len(cpu)-topHistorySize:]
		}
		v.cpuHistory[id] = cpu

		mem := v.memHistory[id]
		mem = append(mem, float64(s.MemUsage))
		if len(mem) > topHistorySize {
			mem = mem[len(mem)-topHistorySize:]
		}
		v.memHistory[id] = mem
	}
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

func (v TopView) renderContainer() string {
	s := v.stats[v.containerID]

	labelStyle := ui.MutedStyle
	valueStyle := lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	pad := "            " // alignment padding for sparkline rows

	lines := []string{
		fmt.Sprintf("  %s  %s", labelStyle.Render("Container:"), valueStyle.Render(v.containerName)),
		fmt.Sprintf("  %s  %s", labelStyle.Render("Image:    "), valueStyle.Render(v.image)),
		"",
		fmt.Sprintf("  %s  %s", labelStyle.Render("CPU:      "), formatCPU(s.CPUPercent)),
	}

	if shouldShowSparkline(v.cpuHistory[v.containerID], 1.0) {
		lines = append(lines,
			fmt.Sprintf("  %s%s", pad, ui.MutedStyle.Render(ui.Sparkline(v.cpuHistory[v.containerID], 100))))
	}

	lines = append(lines, "",
		fmt.Sprintf("  %s  %s / %s", labelStyle.Render("Memory:   "), formatMemory(s.MemUsage), formatMemory(s.MemLimit)),
	)

	if shouldShowSparkline(v.memHistory[v.containerID], 1048576) {
		lines = append(lines,
			fmt.Sprintf("  %s%s", pad, ui.MutedStyle.Render(ui.Sparkline(v.memHistory[v.containerID], float64(s.MemLimit)))))
	}

	lines = append(lines, "",
		fmt.Sprintf("  %s  %s", labelStyle.Render("Net RX:   "), formatMemory(s.NetRx)),
		fmt.Sprintf("  %s  %s", labelStyle.Render("Net TX:   "), formatMemory(s.NetTx)),
	)

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

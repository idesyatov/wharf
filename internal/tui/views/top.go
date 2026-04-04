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
	cpuHistory    []float64
	memHistory    []float64
	chartWidth    int
	prevNetRx     uint64
	prevNetTx     uint64
	netRxHistory  []float64
	netTxHistory  []float64
	netChartWidth int
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
	cw := (w - 10) / 2
	if cw < 10 {
		cw = 10
	}
	if cw > 120 {
		cw = 120
	}
	v.chartWidth = cw
	if len(v.cpuHistory) > cw {
		v.cpuHistory = v.cpuHistory[len(v.cpuHistory)-cw:]
	}
	if len(v.memHistory) > cw {
		v.memHistory = v.memHistory[len(v.memHistory)-cw:]
	}
	nw := 2*cw + 4
	if nw < 20 {
		nw = 20
	}
	v.netChartWidth = nw
	if len(v.netRxHistory) > nw {
		v.netRxHistory = v.netRxHistory[len(v.netRxHistory)-nw:]
	}
	if len(v.netTxHistory) > nw {
		v.netTxHistory = v.netTxHistory[len(v.netTxHistory)-nw:]
	}
	return v
}

func (v TopView) UpdateStats(stats map[string]docker.Stats) TopView {
	v.stats = stats
	if !v.IsProjectMode() {
		s := stats[v.containerID]
		v.cpuHistory = appendHistory(v.cpuHistory, s.CPUPercent, v.chartWidth)
		memPct := 0.0
		if s.MemLimit > 0 {
			memPct = float64(s.MemUsage) / float64(s.MemLimit) * 100
		}
		v.memHistory = appendHistory(v.memHistory, memPct, v.chartWidth)

		// Network rate calculation (delta between ticks)
		if v.prevNetRx > 0 {
			rxRate := float64(s.NetRx - v.prevNetRx)
			txRate := float64(s.NetTx - v.prevNetTx)
			if rxRate < 0 {
				rxRate = 0
			}
			if txRate < 0 {
				txRate = 0
			}
			v.netRxHistory = appendHistory(v.netRxHistory, rxRate, v.netChartWidth)
			v.netTxHistory = appendHistory(v.netTxHistory, txRate, v.netChartWidth)
		}
		v.prevNetRx = s.NetRx
		v.prevNetTx = s.NetTx
	}
	return v
}

func appendHistory(h []float64, val float64, maxLen int) []float64 {
	if maxLen <= 0 {
		maxLen = 60
	}
	h = append(h, val)
	if len(h) > maxLen {
		h = h[len(h)-maxLen:]
	}
	return h
}

func maxInSlice(values []float64) float64 {
	m := 0.0
	for _, v := range values {
		if v > m {
			m = v
		}
	}
	return m
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
			}
		case ui.MatchKey(msg, keys.Up):
			if v.IsProjectMode() {
				if v.cursor > 0 {
					v.cursor--
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

func colorChartByZones(chartText string) string {
	lines := strings.Split(chartText, "\n")
	if len(lines) <= 1 {
		return lipgloss.NewStyle().Foreground(ui.ColorSuccess).Render(chartText)
	}
	var colored []string
	for i, line := range lines {
		pct := float64(i) / float64(len(lines)-1)
		var style lipgloss.Style
		switch {
		case pct < 0.33:
			style = lipgloss.NewStyle().Foreground(ui.ColorDanger)
		case pct < 0.66:
			style = lipgloss.NewStyle().Foreground(ui.ColorWarning)
		default:
			style = lipgloss.NewStyle().Foreground(ui.ColorSuccess)
		}
		colored = append(colored, style.Render(line))
	}
	return strings.Join(colored, "\n")
}

func (v TopView) renderContainer() string {
	s := v.stats[v.containerID]

	labelStyle := ui.MutedStyle
	valueStyle := lipgloss.NewStyle().Foreground(ui.ColorPrimary)

	header := lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("  %s  %s", labelStyle.Render("Container:"), valueStyle.Render(v.containerName)),
		fmt.Sprintf("  %s  %s", labelStyle.Render("Image:    "), valueStyle.Render(v.image)),
	)

	if len(v.cpuHistory) == 0 {
		return header + "\n\n" + lipgloss.NewStyle().Foreground(ui.ColorSuccess).Render("  Loading stats...")
	}

	memPct := 0.0
	if s.MemLimit > 0 {
		memPct = float64(s.MemUsage) / float64(s.MemLimit) * 100
	}

	cpuStyle := colorByLevel(s.CPUPercent, 100)
	memStyle := colorByLevel(memPct, 100)

	chartHeight := v.height - 22
	if chartHeight < 3 {
		chartHeight = 3
	}
	if chartHeight > 12 {
		chartHeight = 12
	}

	cw := v.chartWidth

	cpuMax := maxInSlice(v.cpuHistory)
	if cpuMax < 1 {
		cpuMax = 1
	}
	cpuMax = cpuMax * 1.1
	if cpuMax > 100 {
		cpuMax = 100
	}

	memMax := maxInSlice(v.memHistory)
	if memMax < 1 {
		memMax = 1
	}
	memMax = memMax * 1.2

	cpuChartText := ui.BrailleChart(v.cpuHistory, cpuMax, cw, chartHeight)
	memChartText := ui.BrailleChart(v.memHistory, memMax, cw, chartHeight)

	cpuColored := colorChartByZones(cpuChartText)
	memColored := memStyle.Render(memChartText)

	chartBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorBorder).
		Width(cw)

	cpuBox := chartBoxStyle.Render(cpuColored)
	memBox := chartBoxStyle.Render(memColored)

	cpuLabel := cpuStyle.Render(fmt.Sprintf("CPU [%s]", formatCPU(s.CPUPercent)))
	memLabel := memStyle.Render(fmt.Sprintf("MEM [%s / %s · %.1f%%]",
		formatMemory(s.MemUsage), formatMemory(s.MemLimit), memPct))

	cpuFull := cpuLabel + "\n" + cpuBox
	memFull := memLabel + "\n" + memBox

	// Net I/O chart
	var netFull string
	if len(v.netRxHistory) > 0 || len(v.netTxHistory) > 0 {
		rxRate := 0.0
		if len(v.netRxHistory) > 0 {
			rxRate = v.netRxHistory[len(v.netRxHistory)-1]
		}
		txRate := 0.0
		if len(v.netTxHistory) > 0 {
			txRate = v.netTxHistory[len(v.netTxHistory)-1]
		}

		netLabel := ui.MutedStyle.Render(fmt.Sprintf("Net I/O [↓ %s/s  ↑ %s/s]",
			formatMemory(uint64(rxRate)), formatMemory(uint64(txRate))))

		netMiniHeight := 3
		nw := v.netChartWidth

		rxMax := maxInSlice(v.netRxHistory)
		if rxMax < 1 {
			rxMax = 1
		}
		rxMax *= 1.2
		txMax := maxInSlice(v.netTxHistory)
		if txMax < 1 {
			txMax = 1
		}
		txMax *= 1.2

		rxChart := ui.BrailleChart(v.netRxHistory, rxMax, nw, netMiniHeight)
		txChart := ui.BrailleChart(v.netTxHistory, txMax, nw, netMiniHeight)

		rxColored := colorChartByZones(rxChart)
		txColored := colorChartByZones(txChart)

		separator := ui.MutedStyle.Render(strings.Repeat("─", nw))
		rxLabel := lipgloss.NewStyle().Foreground(ui.ColorSuccess).Render("RX")
		txLabel := lipgloss.NewStyle().Foreground(ui.ColorSuccess).Render("TX")

		netContent := rxLabel + "\n" + rxColored + "\n" + separator + "\n" + txLabel + "\n" + txColored

		netBoxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.ColorBorder).
			Width(nw)

		netBox := netBoxStyle.Render(netContent)
		netFull = netLabel + "\n" + netBox
	}

	// Fallback net line (before first delta is calculated)
	netLine := fmt.Sprintf("  Net RX: %s    Net TX: %s",
		formatMemory(s.NetRx), formatMemory(s.NetTx))

	var charts string
	if v.width < 60 {
		charts = cpuFull + "\n\n" + memFull
		if netFull != "" {
			charts += "\n\n" + netFull
		}
	} else {
		charts = lipgloss.JoinHorizontal(lipgloss.Top, cpuFull, "  ", memFull)
		if netFull != "" {
			charts += "\n\n" + netFull
		}
	}

	indent := lipgloss.NewStyle().PaddingLeft(2)
	if netFull != "" {
		return header + "\n\n" + indent.Render(charts)
	}
	return header + "\n\n" + indent.Render(charts) + "\n\n" + netLine
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

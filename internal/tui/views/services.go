package views

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/config"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/ui"
)

type ActionStartMsg struct{ Service docker.Service }
type ActionStopMsg struct{ Service docker.Service }
type ActionRestartMsg struct{ Service docker.Service }
type ActionResultMsg struct {
	Err         error
	Action      string
	ServiceName string
}

type OpenBrowserMsg struct{ URL string }
type SwitchToDetailMsg struct{ Service docker.Service }
type SwitchToLogsMsg struct{ Container docker.Container }
type ExecMsg struct {
	ContainerID   string
	ContainerName string
	Image         string
	Shell         string
}
type ExecDoneMsg struct{ Err error }
type BuildMsg struct {
	ProjectPath string
	ComposePath string
	Service     string
}
type BuildDoneMsg struct {
	Err     error
	Service string
}

type StatsLoadedMsg struct {
	Stats map[string]docker.Stats
}

type HealthLoadedMsg struct {
	Health map[string]string
}

type CustomCommandMsg struct {
	Command string
	Name    string
}
type CustomCommandDoneMsg struct {
	Err  error
	Name string
}

type ServicesView struct {
	project         docker.Project
	cursor          int
	width           int
	height          int
	pendingG        bool
	filterMode      bool
	filterText      string
	stats           map[string]docker.Stats
	health          map[string]string
	sortColumn      int
	sortReverse     bool
	pendingDown     bool
	pendingDownName string
	pendingDownPath string
	customCommands  []config.CustomCommand
}

func NewServicesView(project docker.Project, customCommands []config.CustomCommand) ServicesView {
	return ServicesView{
		project:        project,
		customCommands: customCommands,
	}
}

func (v ServicesView) CustomCommands() []config.CustomCommand { return v.customCommands }

func (v ServicesView) SetSize(w, h int) ServicesView {
	v.width = w
	v.height = h
	return v
}

func (v ServicesView) Project() docker.Project { return v.project }
func (v ServicesView) Breadcrumb() string      { return "› " + v.project.Name }
func (v ServicesView) ProjectName() string     { return v.project.Name }
func (v ServicesView) ProjectPath() string     { return v.project.Path }
func (v ServicesView) FilterMode() bool        { return v.filterMode }
func (v ServicesView) FilterText() string      { return v.filterText }
func (v ServicesView) PendingDown() bool       { return v.pendingDown }
func (v ServicesView) PendingDownName() string { return v.pendingDownName }
func (v ServicesView) HasStats() bool            { return len(v.stats) > 0 }

func (v ServicesView) UpdateStats(stats map[string]docker.Stats) ServicesView {
	v.stats = stats
	return v
}

func (v ServicesView) UpdateHealth(health map[string]string) ServicesView {
	v.health = health
	return v
}

func LoadStats(client *docker.Client, project docker.Project) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return StatsLoadedMsg{}
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
		return StatsLoadedMsg{Stats: result}
	}
}

func LoadHealth(client *docker.Client, project docker.Project) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return HealthLoadedMsg{}
		}
		ctx := context.Background()
		result := make(map[string]string)
		for _, svc := range project.Services {
			for _, ct := range svc.Containers {
				if ct.Status != "running" {
					continue
				}
				result[ct.ID] = client.ContainerHealthStatus(ctx, ct.ID)
			}
		}
		return HealthLoadedMsg{Health: result}
	}
}

func (v ServicesView) UpdateProject(project docker.Project) ServicesView {
	v.project = project
	filtered := v.filtered()
	if v.cursor >= len(filtered) && len(filtered) > 0 {
		v.cursor = len(filtered) - 1
	}
	return v
}

func (v ServicesView) filtered() []docker.Service {
	var src []docker.Service
	if v.filterText == "" {
		src = make([]docker.Service, len(v.project.Services))
		copy(src, v.project.Services)
	} else {
		q := strings.ToLower(v.filterText)
		for _, s := range v.project.Services {
			if strings.Contains(strings.ToLower(s.Name), q) {
				src = append(src, s)
			}
		}
	}
	v.applySortServices(src)
	return src
}

func (v ServicesView) svcCPU(svc docker.Service) float64 {
	if v.stats == nil {
		return -1
	}
	total := 0.0
	found := false
	for _, ct := range svc.Containers {
		if s, ok := v.stats[ct.ID]; ok {
			total += s.CPUPercent
			found = true
		}
	}
	if !found {
		return -1
	}
	return total
}

func (v ServicesView) svcMem(svc docker.Service) int64 {
	if v.stats == nil {
		return -1
	}
	var total uint64
	found := false
	for _, ct := range svc.Containers {
		if s, ok := v.stats[ct.ID]; ok {
			total += s.MemUsage
			found = true
		}
	}
	if !found {
		return -1
	}
	return int64(total)
}

func svcContainerName(svc docker.Service) string {
	if len(svc.Containers) > 0 {
		return svc.Containers[0].Name
	}
	return ""
}

func (v ServicesView) applySortServices(svcs []docker.Service) {
	sort.SliceStable(svcs, func(i, j int) bool {
		var less bool
		switch v.sortColumn {
		case 0: // SERVICE
			less = svcs[i].Name < svcs[j].Name
		case 1: // CONTAINER
			less = svcContainerName(svcs[i]) < svcContainerName(svcs[j])
		case 2: // STATUS
			less = statusOrder(svcs[i].Status) < statusOrder(svcs[j].Status)
		case 3: // CPU
			less = v.svcCPU(svcs[i]) < v.svcCPU(svcs[j])
		case 4: // MEM
			less = v.svcMem(svcs[i]) < v.svcMem(svcs[j])
		case 5: // IMAGE
			less = svcs[i].Image < svcs[j].Image
		default:
			less = svcs[i].Name < svcs[j].Name
		}
		if v.sortReverse {
			return !less
		}
		return less
	})
}

func (v ServicesView) selectedService() (docker.Service, bool) {
	filtered := v.filtered()
	if len(filtered) == 0 {
		return docker.Service{}, false
	}
	return filtered[v.cursor], true
}

func (v ServicesView) Update(msg tea.Msg, keys ui.KeyMap) (ServicesView, tea.Cmd) {
	switch msg := msg.(type) {
	case StatsLoadedMsg:
		v.stats = msg.Stats
		return v, nil

	case HealthLoadedMsg:
		v.health = msg.Health
		return v, nil

	case tea.MouseMsg:
		filtered := v.filtered()
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			row := msg.Y - 5 // info + breadcrumbs + title + header
			if row >= 0 && row < len(filtered) {
				v.cursor = row
			}
		}
		if msg.Button == tea.MouseButtonWheelDown {
			if v.cursor < len(filtered)-1 {
				v.cursor++
			}
		}
		if msg.Button == tea.MouseButtonWheelUp {
			if v.cursor > 0 {
				v.cursor--
			}
		}
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg, keys)
	}

	return v, nil
}

func (v ServicesView) handleFilterInput(msg tea.KeyMsg) (ServicesView, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		v.filterMode = false
		v.cursor = 0
	case tea.KeyEsc:
		v.filterMode = false
		v.filterText = ""
		v.cursor = 0
	case tea.KeyBackspace:
		if len(v.filterText) > 0 {
			v.filterText = v.filterText[:len(v.filterText)-1]
			v.cursor = 0
		}
	default:
		if msg.Type == tea.KeyRunes {
			v.filterText += string(msg.Runes)
			v.cursor = 0
		}
	}
	return v, nil
}

func (v ServicesView) handleKeyMsg(msg tea.KeyMsg, keys ui.KeyMap) (ServicesView, tea.Cmd) {
	if v.filterMode {
		return v.handleFilterInput(msg)
	}

	if v.pendingDown {
		v.pendingDown = false
		if ui.MatchKey(msg, keys.Confirm) {
			name := v.pendingDownName
			path := v.pendingDownPath
			return v, func() tea.Msg {
				return ComposeDownMsg{ProjectPath: path, ProjectName: name}
			}
		}
		return v, nil
	}

	if v.pendingG {
		v.pendingG = false
		if msg.String() == "g" {
			v.cursor = 0
			return v, nil
		}
	}

	filtered := v.filtered()
	return v.handleNavAndActions(msg, keys, filtered)
}

func (v ServicesView) handleNavAndActions(msg tea.KeyMsg, keys ui.KeyMap, filtered []docker.Service) (ServicesView, tea.Cmd) {
	switch {
	case ui.MatchKey(msg, keys.Down):
		if v.cursor < len(filtered)-1 {
			v.cursor++
		}
	case ui.MatchKey(msg, keys.Up):
		if v.cursor > 0 {
			v.cursor--
		}
	case ui.MatchKey(msg, keys.Bottom):
		if len(filtered) > 0 {
			v.cursor = len(filtered) - 1
		}
	case msg.String() == "g":
		v.pendingG = true
	case ui.MatchKey(msg, keys.Search):
		v.filterMode = true
		v.filterText = ""
		return v, nil
	case ui.MatchKey(msg, keys.Help):
		return v, func() tea.Msg { return SwitchToHelpMsg{} }
	case ui.MatchKey(msg, keys.TopView):
		if svc, ok := v.selectedService(); ok && len(svc.Containers) > 0 {
			ct := svc.Containers[0]
			return v, func() tea.Msg {
				return SwitchToTopContainerMsg{
					ContainerID:   ct.ID,
					ContainerName: ct.Name,
					Image:         svc.Image,
				}
			}
		}
	case ui.MatchKey(msg, keys.Events):
		return v, func() tea.Msg { return SwitchToEventsMsg{} }
	case ui.MatchKey(msg, keys.Left):
		return v, func() tea.Msg { return SwitchToProjectsMsg{} }
	case ui.MatchKey(msg, keys.Right):
		if svc, ok := v.selectedService(); ok {
			return v, func() tea.Msg { return SwitchToDetailMsg{Service: svc} }
		}
	default:
		return v.handleServiceActions(msg, keys, filtered)
	}
	return v, nil
}

func (v ServicesView) handleServiceActions(msg tea.KeyMsg, keys ui.KeyMap, filtered []docker.Service) (ServicesView, tea.Cmd) {
	switch {
	case ui.MatchKey(msg, keys.Start):
		if svc, ok := v.selectedService(); ok {
			return v, func() tea.Msg { return ActionStartMsg{Service: svc} }
		}
	case ui.MatchKey(msg, keys.Stop):
		if svc, ok := v.selectedService(); ok {
			return v, func() tea.Msg { return ActionStopMsg{Service: svc} }
		}
	case ui.MatchKey(msg, keys.Restart):
		if svc, ok := v.selectedService(); ok {
			return v, func() tea.Msg { return ActionRestartMsg{Service: svc} }
		}
	case ui.MatchKey(msg, keys.Logs):
		if svc, ok := v.selectedService(); ok && len(svc.Containers) > 0 {
			return v, func() tea.Msg { return SwitchToLogsMsg{Container: svc.Containers[0]} }
		}
	default:
		return v.handleServiceTools(msg, keys, filtered)
	}
	return v, nil
}

func (v ServicesView) handleServiceTools(msg tea.KeyMsg, keys ui.KeyMap, filtered []docker.Service) (ServicesView, tea.Cmd) {
	switch {
	case ui.MatchKey(msg, keys.Compose):
		return v, func() tea.Msg {
			return SwitchToComposeMsg{ProjectName: v.project.Name, ProjectPath: v.project.Path}
		}
	case ui.MatchKey(msg, keys.VolumesKey):
		return v, func() tea.Msg {
			return SwitchToVolumesMsg{ProjectName: v.project.Name}
		}
	case ui.MatchKey(msg, keys.Exec):
		if svc, ok := v.selectedService(); ok && len(svc.Containers) > 0 {
			ct := svc.Containers[0]
			if ct.Status != "running" {
				break
			}
			return v, func() tea.Msg {
				return ExecMsg{ContainerID: ct.ID, ContainerName: ct.Name, Image: svc.Image}
			}
		}
	case ui.MatchKey(msg, keys.NetworksKey):
		return v, func() tea.Msg {
			return SwitchToNetworksMsg{ProjectName: v.project.Name}
		}
	case ui.MatchKey(msg, keys.Build):
		if svc, ok := v.selectedService(); ok {
			return v, func() tea.Msg {
				return BuildMsg{ProjectPath: v.project.Path, Service: svc.Name}
			}
		}
	case ui.MatchKey(msg, keys.BuildAll):
		return v, func() tea.Msg {
			return BuildMsg{ProjectPath: v.project.Path, Service: ""}
		}
	case ui.MatchKey(msg, keys.Copy):
		if svc, ok := v.selectedService(); ok && len(svc.Containers) > 0 {
			id := svc.Containers[0].ID
			return v, func() tea.Msg { return CopyMsg{Text: id, Label: id} }
		}
	case ui.MatchKey(msg, keys.EnvFile):
		return v, func() tea.Msg {
			return SwitchToEnvMsg{ProjectName: v.project.Name, ProjectPath: v.project.Path}
		}
	default:
		if k := msg.String(); len(k) == 1 && k >= "1" && k <= "9" {
			if cmd := v.matchCustomCommand(k, filtered); cmd != nil {
				return v, cmd
			}
			if k <= "7" {
				col := int(k[0]-'0') - 1
				if v.sortColumn == col {
					v.sortReverse = !v.sortReverse
				} else {
					v.sortColumn = col
					v.sortReverse = false
				}
			}
		}
	}

	return v, nil
}

func (v ServicesView) matchCustomCommand(key string, filtered []docker.Service) tea.Cmd {
	for _, cc := range v.customCommands {
		if cc.Key != key {
			continue
		}
		if v.cursor >= len(filtered) {
			return nil
		}
		svc := filtered[v.cursor]
		if len(svc.Containers) == 0 {
			return nil
		}
		ct := svc.Containers[0]
		vars := config.CommandVars{
			ContainerID:   ct.ID,
			ContainerName: ct.Name,
			Image:         svc.Image,
			ProjectName:   v.project.Name,
			ProjectPath:   v.project.Path,
		}
		rendered, err := cc.Render(vars)
		if err != nil {
			return nil
		}
		name := cc.Name
		return func() tea.Msg {
			return CustomCommandMsg{Command: rendered, Name: name}
		}
	}
	return nil
}

func (v ServicesView) View() string {
	title := ui.ProjectTitleStyle.Render("Project: " + v.project.Name)
	filtered := v.filtered()

	if len(v.project.Services) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			title,
			ui.ContentStyle.Render("No services found"),
		)
	}

	if len(filtered) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			title,
			ui.MutedStyle.Render(fmt.Sprintf("No matches for '%s'", v.filterText)),
		)
	}

	colName := 14
	colCont := 25
	colStatus := 10
	colHealth := 2
	colCPU := 8
	colMem := 10
	colImage := 20

	cols := []string{"SERVICE", "CONTAINER", "STATUS", "H", "CPU", "MEM", "IMAGE", "PORTS"}
	for i := range cols {
		if i == v.sortColumn {
			if v.sortReverse {
				cols[i] += "▼"
			} else {
				cols[i] += "▲"
			}
		}
	}

	header := ui.HeaderRowStyle.Render(
		fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s %s",
			colName, cols[0], colCont, cols[1], colStatus, cols[2], colHealth, cols[3],
			colCPU, cols[4], colMem, cols[5],
			colImage, cols[6], cols[7]),
	)

	var rows []string
	rows = append(rows, title, header)

	for i, svc := range filtered {
		ports := formatPorts(svc)
		cpu, mem := v.svcStats(svc)
		ctName := svcContainerName(svc)

		if i == v.cursor {
			plainRow := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s %s",
				colName, truncate(svc.Name, colName),
				colCont, truncate(ctName, colCont),
				colStatus, statusTextPlain(svc.Status),
				colHealth, svcHealthPlain(v.health, svc),
				colCPU, cpu,
				colMem, mem,
				colImage, truncate(svc.Image, colImage),
				ports,
			)
			rows = append(rows, renderSelectedRow(plainRow, v.width-2))
			continue
		}

		statusStr := statusText(svc.Status)
		healthStr := v.svcHealthIndicator(svc)
		row := fmt.Sprintf("%-*s %-*s %s %s %-*s %-*s %-*s %s",
			colName, truncate(svc.Name, colName),
			colCont, truncate(ctName, colCont),
			padRight(statusStr, colStatus),
			padRight(healthStr, colHealth),
			colCPU, cpu,
			colMem, mem,
			colImage, truncate(svc.Image, colImage),
			ports,
		)

		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (v ServicesView) svcStats(svc docker.Service) (string, string) {
	if v.stats == nil {
		return "-", "-"
	}
	var totalCPU float64
	var totalMem uint64
	var totalLimit uint64
	found := false
	for _, ct := range svc.Containers {
		if s, ok := v.stats[ct.ID]; ok {
			totalCPU += s.CPUPercent
			totalMem += s.MemUsage
			totalLimit += s.MemLimit
			found = true
		}
	}
	if !found {
		return "-", "-"
	}
	return formatCPU(totalCPU), formatMemory(totalMem)
}

func (v ServicesView) svcHealthIndicator(svc docker.Service) string {
	if v.health == nil {
		return ui.MutedStyle.Render("-")
	}
	for _, ct := range svc.Containers {
		if s, ok := v.health[ct.ID]; ok {
			switch s {
			case "healthy":
				return ui.RunningStyle.Render("●")
			case "unhealthy":
				return ui.ErrorStyle.Render("✕")
			case "starting":
				return ui.PartialStyle.Render("◌")
			}
		}
	}
	return ui.MutedStyle.Render("-")
}

func svcHealthPlain(health map[string]string, svc docker.Service) string {
	if health == nil {
		return "-"
	}
	for _, ct := range svc.Containers {
		if s, ok := health[ct.ID]; ok {
			switch s {
			case "healthy":
				return "●"
			case "unhealthy":
				return "✕"
			case "starting":
				return "◌"
			}
		}
	}
	return "-"
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

func formatPorts(svc docker.Service) string {
	seen := make(map[string]bool)
	var parts []string
	for _, c := range svc.Containers {
		for _, p := range c.Ports {
			if p.HostPort == 0 {
				continue
			}
			s := fmt.Sprintf("%d:%d", p.HostPort, p.ContPort)
			if !seen[s] {
				seen[s] = true
				parts = append(parts, s)
			}
		}
	}
	return strings.Join(parts, ", ")
}

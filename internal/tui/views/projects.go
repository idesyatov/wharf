package views

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/config"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/ui"
)

type CopyMsg struct {
	Text  string
	Label string
}
type BookmarkToggleMsg struct{ ProjectName string }

type ProjectsLoadedMsg struct{ Projects []docker.Project }
type ProjectsErrorMsg struct{ Err error }
type TickMsg struct{}

type ComposeUpMsg struct {
	ProjectPath string
	ProjectName string
}
type ComposeStopMsg struct {
	ProjectPath string
	ProjectName string
}
type ComposeDownMsg struct {
	ProjectPath string
	ProjectName string
}
type ComposeRestartMsg struct {
	ProjectPath string
	ProjectName string
}
type ComposeResultMsg struct {
	Err         error
	Action      string
	ProjectName string
}

type BatchActionMsg struct {
	Action   string
	Projects []docker.Project
}

type ProjectsView struct {
	projects        []docker.Project
	cursor          int
	width           int
	height          int
	pendingG        bool
	err             error
	filterMode      bool
	filterText      string
	pendingDown     bool
	pendingDownName string
	pendingDownPath string
	pollInterval    time.Duration
	cfg             *config.Config
	sortColumn      int
	sortReverse     bool
	selected        map[int]bool
}

func NewProjectsView(pollInterval time.Duration, cfg *config.Config) ProjectsView {
	return ProjectsView{pollInterval: pollInterval, cfg: cfg}
}

func (v ProjectsView) SetSize(w, h int) ProjectsView {
	v.width = w
	v.height = h
	return v
}

func (v ProjectsView) Breadcrumb() string      { return "" }
func (v ProjectsView) FilterMode() bool        { return v.filterMode }
func (v ProjectsView) FilterText() string      { return v.filterText }
func (v ProjectsView) PendingDown() bool       { return v.pendingDown }
func (v ProjectsView) PendingDownName() string { return v.pendingDownName }
func (v ProjectsView) SelectedCount() int      { return len(v.selected) }
func (v ProjectsView) HasSelected() bool       { return len(v.selected) > 0 }
func (v ProjectsView) Projects() []docker.Project { return v.projects }

func LoadProjects(client *docker.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return ProjectsErrorMsg{Err: fmt.Errorf("Docker is not connected")}
		}
		projects, err := client.ListProjects(context.Background())
		if err != nil {
			return ProjectsErrorMsg{Err: err}
		}
		return ProjectsLoadedMsg{Projects: projects}
	}
}

func TickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return TickMsg{}
	})
}

func (v ProjectsView) filtered() []docker.Project {
	var src []docker.Project
	if v.filterText == "" {
		src = make([]docker.Project, len(v.projects))
		copy(src, v.projects)
	} else {
		q := strings.ToLower(v.filterText)
		for _, p := range v.projects {
			if strings.Contains(strings.ToLower(p.Name), q) {
				src = append(src, p)
			}
		}
	}
	// Sort: bookmarked first
	if v.cfg != nil && len(v.cfg.Bookmarks) > 0 {
		sort.SliceStable(src, func(i, j int) bool {
			bi := v.cfg.IsBookmarked(src[i].Name)
			bj := v.cfg.IsBookmarked(src[j].Name)
			if bi != bj {
				return bi
			}
			return false
		})
	}
	// Column sort
	v.applySortProjects(src)
	return src
}

func statusOrder(s docker.ServiceStatus) int {
	switch s {
	case docker.StatusRunning:
		return 0
	case docker.StatusPartial:
		return 1
	default:
		return 2
	}
}

func runningCount(p docker.Project) int {
	n := 0
	for _, s := range p.Services {
		if s.Status == docker.StatusRunning {
			n++
		}
	}
	return n
}

func (v ProjectsView) applySortProjects(ps []docker.Project) {
	sort.SliceStable(ps, func(i, j int) bool {
		var less bool
		switch v.sortColumn {
		case 0: // NAME
			less = ps[i].Name < ps[j].Name
		case 1: // STATUS
			less = statusOrder(ps[i].Status) < statusOrder(ps[j].Status)
		case 2: // SERVICES (running count)
			less = runningCount(ps[i]) < runningCount(ps[j])
		case 3: // PATH
			less = ps[i].Path < ps[j].Path
		default:
			less = ps[i].Name < ps[j].Name
		}
		if v.sortReverse {
			return !less
		}
		return less
	})
}

func (v ProjectsView) Update(msg tea.Msg, keys ui.KeyMap) (ProjectsView, tea.Cmd) {
	switch msg := msg.(type) {
	case ProjectsLoadedMsg:
		v.projects = msg.Projects
		v.err = nil
		filtered := v.filtered()
		if v.cursor >= len(filtered) && len(filtered) > 0 {
			v.cursor = len(filtered) - 1
		}
		return v, nil

	case ProjectsErrorMsg:
		v.err = msg.Err
		return v, nil

	case tea.MouseMsg:
		filtered := v.filtered()
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			row := msg.Y - 4 // info + breadcrumbs + header
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

func (v ProjectsView) handleFilterInput(msg tea.KeyMsg) (ProjectsView, tea.Cmd) {
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

func (v ProjectsView) handleKeyMsg(msg tea.KeyMsg, keys ui.KeyMap) (ProjectsView, tea.Cmd) {
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

func (v ProjectsView) handleNavAndActions(msg tea.KeyMsg, keys ui.KeyMap, filtered []docker.Project) (ProjectsView, tea.Cmd) {
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
	case ui.MatchKey(msg, keys.Right):
		if len(filtered) > 0 {
			p := filtered[v.cursor]
			return v, func() tea.Msg {
				return SwitchToServicesMsg{Project: p}
			}
		}
	case ui.MatchKey(msg, keys.TopView):
		if len(filtered) > 0 {
			p := filtered[v.cursor]
			return v, func() tea.Msg {
				return SwitchToTopProjectMsg{Project: p}
			}
		}
	case ui.MatchKey(msg, keys.Images):
		return v, func() tea.Msg { return SwitchToImagesMsg{} }
	case ui.MatchKey(msg, keys.Events):
		return v, func() tea.Msg { return SwitchToEventsMsg{} }
	case ui.MatchKey(msg, keys.SystemDf):
		return v, func() tea.Msg { return SwitchToSystemMsg{} }
	case msg.String() == "H":
		return v, func() tea.Msg { return SwitchToHostsMsg{} }
	default:
		return v.handleProjectActions(msg, keys, filtered)
	}
	return v, nil
}

func (v ProjectsView) handleProjectActions(msg tea.KeyMsg, keys ui.KeyMap, filtered []docker.Project) (ProjectsView, tea.Cmd) {
	switch {
	case msg.String() == " ":
		if len(filtered) > 0 {
			if v.selected == nil {
				v.selected = make(map[int]bool)
			}
			if v.selected[v.cursor] {
				delete(v.selected, v.cursor)
			} else {
				v.selected[v.cursor] = true
			}
			if v.cursor < len(filtered)-1 {
				v.cursor++
			}
		}
	default:
		return v.handleComposeActions(msg, keys, filtered)
	}
	return v, nil
}

func (v ProjectsView) handleComposeActions(msg tea.KeyMsg, keys ui.KeyMap, filtered []docker.Project) (ProjectsView, tea.Cmd) {
	switch {
	case ui.MatchKey(msg, keys.ComposeUp):
		if len(v.selected) > 0 {
			return v, v.batchAction("up", filtered)
		}
		if len(filtered) > 0 {
			p := filtered[v.cursor]
			return v, func() tea.Msg {
				return ComposeUpMsg{ProjectPath: p.Path, ProjectName: p.Name}
			}
		}
	case ui.MatchKey(msg, keys.ComposeStop):
		if len(v.selected) > 0 {
			return v, v.batchAction("stop", filtered)
		}
		if len(filtered) > 0 {
			p := filtered[v.cursor]
			return v, func() tea.Msg {
				return ComposeStopMsg{ProjectPath: p.Path, ProjectName: p.Name}
			}
		}
	case ui.MatchKey(msg, keys.ComposeDown):
		if len(v.selected) > 0 {
			return v, v.batchAction("down", filtered)
		}
		if len(filtered) > 0 {
			p := filtered[v.cursor]
			v.pendingDown = true
			v.pendingDownName = p.Name
			v.pendingDownPath = p.Path
		}
	case ui.MatchKey(msg, keys.ComposeRestart):
		if len(v.selected) > 0 {
			return v, v.batchAction("restart", filtered)
		}
		if len(filtered) > 0 {
			p := filtered[v.cursor]
			return v, func() tea.Msg {
				return ComposeRestartMsg{ProjectPath: p.Path, ProjectName: p.Name}
			}
		}
	default:
		return v.handleProjectMisc(msg, keys, filtered)
	}
	return v, nil
}

func (v ProjectsView) handleProjectMisc(msg tea.KeyMsg, keys ui.KeyMap, filtered []docker.Project) (ProjectsView, tea.Cmd) {
	switch {
	case ui.MatchKey(msg, keys.Bookmark):
		if len(filtered) > 0 {
			name := filtered[v.cursor].Name
			return v, func() tea.Msg { return BookmarkToggleMsg{ProjectName: name} }
		}
	case ui.MatchKey(msg, keys.Copy):
		if len(filtered) > 0 {
			name := filtered[v.cursor].Name
			return v, func() tea.Msg { return CopyMsg{Text: name, Label: name} }
		}
	case msg.String() >= "1" && msg.String() <= "4":
		col := int(msg.String()[0]-'0') - 1
		if v.sortColumn == col {
			v.sortReverse = !v.sortReverse
		} else {
			v.sortColumn = col
			v.sortReverse = false
		}
	case msg.Type == tea.KeyEsc:
		if len(v.selected) > 0 {
			v.selected = nil
		}
	}

	return v, nil
}

func (v ProjectsView) View() string {
	if v.err != nil {
		return ui.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}

	filtered := v.filtered()

	if len(v.projects) == 0 {
		return ui.ContentStyle.Render("No Docker Compose projects found")
	}

	if len(filtered) == 0 {
		return ui.MutedStyle.Render(fmt.Sprintf("No matches for '%s'", v.filterText))
	}

	colMark := 2
	colName := 20
	colStatus := 12
	colSvc := 12

	cols := []string{"NAME", "STATUS", "SERVICES", "PATH"}
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
		fmt.Sprintf("%-*s%-*s %-*s %-*s %s", colMark, "", colName, cols[0], colStatus, cols[1], colSvc, cols[2], cols[3]),
	)

	var rows []string
	rows = append(rows, header)

	for i, p := range filtered {
		running := 0
		for _, s := range p.Services {
			if s.Status == docker.StatusRunning {
				running++
			}
		}
		svcCount := fmt.Sprintf("%d/%d", running, len(p.Services))

		if i == v.cursor {
			mark := "  "
			if v.HasSelected() {
				if v.selected[i] {
					mark = "✓ "
				} else {
					mark = "  "
				}
			} else if v.cfg != nil && v.cfg.IsBookmarked(p.Name) {
				mark = "* "
			}
			plainRow := fmt.Sprintf("%s%-*s %-*s %-*s %s",
				mark,
				colName, truncate(p.Name, colName),
				colStatus, statusTextPlain(p.Status),
				colSvc, svcCount,
				p.Path,
			)
			rows = append(rows, renderSelectedRow(plainRow, v.width-2))
			continue
		}

		statusStr := statusText(p.Status)
		mark := "  "
		if v.HasSelected() {
			if v.selected[i] {
				mark = ui.RunningStyle.Render("✓") + " "
			} else {
				mark = "  "
			}
		} else if v.cfg != nil && v.cfg.IsBookmarked(p.Name) {
			mark = ui.BookmarkStyle.Render("★") + " "
		}

		row := fmt.Sprintf("%s%-*s %s %-*s %s",
			mark,
			colName, truncate(p.Name, colName),
			padRight(statusStr, colStatus),
			colSvc, svcCount,
			p.Path,
		)

		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func statusText(s docker.ServiceStatus) string {
	switch s {
	case docker.StatusRunning:
		return ui.RunningStyle.Render("running")
	case docker.StatusPartial:
		return ui.PartialStyle.Render("partial")
	case docker.StatusStopped:
		return ui.StoppedStyle.Render("stopped")
	default:
		return string(s)
	}
}

func statusTextPlain(s docker.ServiceStatus) string {
	switch s {
	case docker.StatusRunning:
		return "running"
	case docker.StatusPartial:
		return "partial"
	case docker.StatusStopped:
		return "stopped"
	default:
		return string(s)
	}
}

func renderSelectedRow(text string, width int) string {
	for lipgloss.Width(text) < width {
		text += " "
	}
	return ui.SelectedRowStyle.Render(text)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func padRight(s string, width int) string {
	visible := lipgloss.Width(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

func (v ProjectsView) batchAction(action string, filtered []docker.Project) tea.Cmd {
	var projects []docker.Project
	for idx := range v.selected {
		if idx < len(filtered) {
			projects = append(projects, filtered[idx])
		}
	}
	return func() tea.Msg {
		return BatchActionMsg{Action: action, Projects: projects}
	}
}

func (v ProjectsView) SelectedProjects(filtered []docker.Project) []docker.Project {
	var projects []docker.Project
	for idx := range v.selected {
		if idx < len(filtered) {
			projects = append(projects, filtered[idx])
		}
	}
	return projects
}

type SwitchToServicesMsg struct{ Project docker.Project }
type SwitchToProjectsMsg struct{}

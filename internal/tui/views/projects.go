package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/ui"
)

type ProjectsLoadedMsg struct{ Projects []docker.Project }
type ProjectsErrorMsg struct{ Err error }
type TickMsg struct{}

type ComposeUpMsg struct {
	ProjectPath string
	ProjectName string
}
type ComposeDownMsg struct {
	ProjectPath string
	ProjectName string
}
type ComposeResultMsg struct {
	Err         error
	Action      string
	ProjectName string
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
}

func NewProjectsView(pollInterval time.Duration) ProjectsView {
	return ProjectsView{pollInterval: pollInterval}
}

func (v ProjectsView) SetSize(w, h int) ProjectsView {
	v.width = w
	v.height = h
	return v
}

func (v ProjectsView) FilterMode() bool    { return v.filterMode }
func (v ProjectsView) FilterText() string   { return v.filterText }
func (v ProjectsView) PendingDown() bool    { return v.pendingDown }
func (v ProjectsView) PendingDownName() string { return v.pendingDownName }

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
	if v.filterText == "" {
		return v.projects
	}
	q := strings.ToLower(v.filterText)
	var out []docker.Project
	for _, p := range v.projects {
		if strings.Contains(strings.ToLower(p.Name), q) {
			out = append(out, p)
		}
	}
	return out
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

	case tea.KeyMsg:
		if v.filterMode {
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

		// Confirmation dialog for compose down
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
		case ui.MatchKey(msg, keys.ComposeUp):
			if len(filtered) > 0 {
				p := filtered[v.cursor]
				return v, func() tea.Msg {
					return ComposeUpMsg{ProjectPath: p.Path, ProjectName: p.Name}
				}
			}
		case ui.MatchKey(msg, keys.ComposeDown):
			if len(filtered) > 0 {
				p := filtered[v.cursor]
				v.pendingDown = true
				v.pendingDownName = p.Name
				v.pendingDownPath = p.Path
			}
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

	colName := 20
	colStatus := 12
	colSvc := 12

	header := ui.HeaderRowStyle.Render(
		fmt.Sprintf("%-*s %-*s %-*s %s", colName, "NAME", colStatus, "STATUS", colSvc, "SERVICES", "PATH"),
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
		statusStr := statusText(p.Status)

		row := fmt.Sprintf("%-*s %s %-*s %s",
			colName, truncate(p.Name, colName),
			padRight(statusStr, colStatus),
			colSvc, svcCount,
			p.Path,
		)

		if i == v.cursor {
			row = ui.SelectedRowStyle.Width(v.width - 4).Render(row)
		}

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

type SwitchToServicesMsg struct{ Project docker.Project }
type SwitchToProjectsMsg struct{}

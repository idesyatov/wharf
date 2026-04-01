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

type DetailLoadedMsg struct{ Detail docker.ContainerDetail }
type DetailErrorMsg struct{ Err error }
type SwitchBackFromDetailMsg struct{}

type DetailView struct {
	service  docker.Service
	detail   docker.ContainerDetail
	loaded   bool
	err      error
	sections []string
	scroll   int
	width    int
	height   int
	pendingG bool
}

func NewDetailView(svc docker.Service) DetailView {
	return DetailView{
		service: svc,
	}
}

func (v DetailView) Breadcrumb() string {
	return "› " + v.service.Project + " › " + v.service.Name
}

func (v DetailView) ServiceName() string {
	return v.service.Name
}

func (v DetailView) ProjectName() string {
	return v.service.Project
}

func (v DetailView) SetSize(w, h int) DetailView {
	v.width = w
	v.height = h
	return v
}

func LoadContainerDetail(client *docker.Client, svc docker.Service) tea.Cmd {
	return func() tea.Msg {
		if client == nil || len(svc.Containers) == 0 {
			return DetailErrorMsg{Err: fmt.Errorf("no containers")}
		}
		detail, err := client.InspectContainer(context.Background(), svc.Containers[0].ID)
		if err != nil {
			return DetailErrorMsg{Err: err}
		}
		return DetailLoadedMsg{Detail: detail}
	}
}

func (v DetailView) Update(msg tea.Msg, keys ui.KeyMap) (DetailView, tea.Cmd) {
	switch msg := msg.(type) {
	case DetailLoadedMsg:
		v.detail = msg.Detail
		v.loaded = true
		v.sections = v.buildSections()
		return v, nil

	case DetailErrorMsg:
		v.err = msg.Err
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg, keys)
	}

	return v, nil
}

func (v DetailView) handleKeyMsg(msg tea.KeyMsg, keys ui.KeyMap) (DetailView, tea.Cmd) {
	if v.pendingG {
		v.pendingG = false
		if msg.String() == "g" {
			v.scroll = 0
			return v, nil
		}
	}

	totalLines := len(v.sections)
	visible := v.visibleHeight()

	switch {
	case ui.MatchKey(msg, keys.Down):
		if v.scroll < totalLines-visible {
			v.scroll++
		}
	case ui.MatchKey(msg, keys.Up):
		if v.scroll > 0 {
			v.scroll--
		}
	case ui.MatchKey(msg, keys.Bottom):
		if totalLines > visible {
			v.scroll = totalLines - visible
		}
	case msg.String() == "g":
		v.pendingG = true
	case ui.MatchKey(msg, keys.Left):
		return v, func() tea.Msg { return SwitchBackFromDetailMsg{} }
	case ui.MatchKey(msg, keys.Logs):
		if len(v.service.Containers) > 0 {
			ct := v.service.Containers[0]
			return v, func() tea.Msg { return SwitchToLogsMsg{Container: ct} }
		}
	case ui.MatchKey(msg, keys.Exec):
		if len(v.service.Containers) > 0 {
			ct := v.service.Containers[0]
			if ct.Status != "running" {
				break
			}
			return v, func() tea.Msg {
				return ExecMsg{ContainerID: ct.ID, ContainerName: ct.Name, Image: ct.Image}
			}
		}
	case ui.MatchKey(msg, keys.Copy):
		if v.loaded {
			id := v.detail.ID
			return v, func() tea.Msg { return CopyMsg{Text: id, Label: id} }
		}
	case ui.MatchKey(msg, keys.CopyFull):
		if v.loaded {
			info := fmt.Sprintf("ID: %s\nImage: %s\nStatus: %s", v.detail.ID, v.detail.Image, v.detail.Status)
			return v, func() tea.Msg { return CopyMsg{Text: info, Label: v.detail.ID} }
		}
	}
	return v, nil
}

func (v DetailView) visibleHeight() int {
	h := v.height - 3
	if h < 1 {
		h = 1
	}
	return h
}

func (v DetailView) View() string {
	title := ui.ProjectTitleStyle.Render("Detail: " + v.service.Name)

	if v.err != nil {
		return lipgloss.JoinVertical(lipgloss.Left,
			title,
			ui.ErrorStyle.Render("Error: "+v.err.Error()),
		)
	}

	if !v.loaded {
		return lipgloss.JoinVertical(lipgloss.Left,
			title,
			ui.ContentStyle.Render("Loading..."),
		)
	}

	visible := v.visibleHeight()
	end := v.scroll + visible
	if end > len(v.sections) {
		end = len(v.sections)
	}
	start := v.scroll
	if start > len(v.sections) {
		start = len(v.sections)
	}

	content := strings.Join(v.sections[start:end], "\n")

	return lipgloss.JoinVertical(lipgloss.Left, title, content)
}

func (v DetailView) buildSections() []string {
	d := v.detail
	var lines []string

	lines = append(lines,
		fmt.Sprintf("  %-14s %s", "Container", d.ID),
		fmt.Sprintf("  %-14s %s", "Name", d.Name),
		fmt.Sprintf("  %-14s %s", "Image", d.Image),
		fmt.Sprintf("  %-14s %s", "Status", detailStatusText(docker.ServiceStatus(d.Status)).render()),
		fmt.Sprintf("  %-14s %s", "State", d.State),
		fmt.Sprintf("  %-14s %s", "Created", d.Created.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("  %-14s %s", "Restart", d.RestartPolicy),
		fmt.Sprintf("  %-14s %s", "Network mode", d.NetworkMode),
		"",
	)

	if len(d.Ports) > 0 {
		lines = append(lines, "  Ports")
		for _, p := range d.Ports {
			if p.HostPort > 0 {
				lines = append(lines, fmt.Sprintf("    %d/%s → %s:%d", p.ContPort, p.Proto, p.HostIP, p.HostPort))
			} else {
				lines = append(lines, fmt.Sprintf("    %d/%s", p.ContPort, p.Proto))
			}
		}
		lines = append(lines, "")
	}

	if len(d.Volumes) > 0 {
		lines = append(lines, "  Volumes")
		for _, vol := range d.Volumes {
			mode := vol.Mode
			if mode == "" {
				mode = "rw"
			}
			lines = append(lines, fmt.Sprintf("    %s → %s (%s)", vol.Source, vol.Destination, mode))
		}
		lines = append(lines, "")
	}

	if len(d.Env) > 0 {
		lines = append(lines, "  Environment")
		for _, e := range d.Env {
			lines = append(lines, "    "+e)
		}
		lines = append(lines, "")
	}

	if len(d.Networks) > 0 {
		lines = append(lines, "  Networks")
		for _, n := range d.Networks {
			lines = append(lines, "    "+n)
		}
		lines = append(lines, "")
	}

	if len(d.Cmd) > 0 {
		lines = append(lines, fmt.Sprintf("  %-14s %s", "Cmd", strings.Join(d.Cmd, " ")))
	}
	if len(d.Entrypoint) > 0 {
		lines = append(lines, fmt.Sprintf("  %-14s %s", "Entrypoint", strings.Join(d.Entrypoint, " ")))
	}

	lines = append(lines, v.buildHealthSection()...)

	return lines
}

func (v DetailView) buildHealthSection() []string {
	d := v.detail
	if d.Health.Status == "none" {
		return nil
	}
	var lines []string
	status := d.Health.Status
	if d.Health.FailingStreak > 0 {
		status += fmt.Sprintf(" (failing streak: %d)", d.Health.FailingStreak)
	}
	lines = append(lines, "", "  Health Check")
	lines = append(lines, fmt.Sprintf("    Status: %s", status))
	showLogs := d.Health.Log
	if len(showLogs) > 3 {
		showLogs = showLogs[len(showLogs)-3:]
	}
	for _, entry := range showLogs {
		lines = append(lines, fmt.Sprintf("    %s  exit=%d  %s",
			entry.Start.Format("15:04:05"), entry.ExitCode,
			truncate(entry.Output, 60)))
	}
	return lines
}

type styledStatus struct {
	s string
}

func (ss styledStatus) render() string {
	return ss.s
}

func detailStatusText(s docker.ServiceStatus) styledStatus {
	switch s {
	case docker.StatusRunning:
		return styledStatus{ui.RunningStyle.Render("running")}
	case docker.StatusPartial:
		return styledStatus{ui.PartialStyle.Render("partial")}
	case docker.StatusStopped:
		return styledStatus{ui.StoppedStyle.Render("stopped")}
	default:
		return styledStatus{string(s)}
	}
}

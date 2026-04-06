package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/config"
	"github.com/idesyatov/wharf/internal/ui"
)

type SwitchToHostsMsg struct{}
type SwitchBackFromHostsMsg struct{}
type HostSelectedMsg struct {
	Name string
	URL  string
}
type HostDeleteMsg struct {
	Name string
}
type HostAddMsg struct {
	Name string
	URL  string
}

type HostsView struct {
	hosts         []hostEntry
	cursor        int
	width         int
	height        int
	addMode       bool
	addField      int // 0 = name, 1 = url
	addName       string
	addURL        string
	pendingDelete bool
}

type hostEntry struct {
	name   string
	url    string
	active bool
}

func NewHostsView(hosts []config.HostEntry, activeURL string) HostsView {
	entries := []hostEntry{
		{name: "local", url: "", active: activeURL == ""},
	}
	for _, h := range hosts {
		entries = append(entries, hostEntry{
			name:   h.Name,
			url:    h.URL,
			active: h.URL == activeURL,
		})
	}
	return HostsView{hosts: entries}
}

func (v HostsView) Breadcrumb() string { return "Hosts" }

func (v HostsView) SetSize(w, h int) HostsView {
	v.width = w
	v.height = h
	return v
}

func (v HostsView) AddMode() bool       { return v.addMode }
func (v HostsView) PendingDelete() bool  { return v.pendingDelete }
func (v HostsView) PendingDeleteName() string {
	if v.cursor > 0 && v.cursor < len(v.hosts) {
		return v.hosts[v.cursor].name
	}
	return ""
}

func (v HostsView) Update(msg tea.Msg, keys ui.KeyMap) (HostsView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if v.addMode {
			return v.handleAddInput(msg)
		}
		if v.pendingDelete {
			v.pendingDelete = false
			if msg.String() == "y" || msg.String() == "Y" {
				h := v.hosts[v.cursor]
				return v, func() tea.Msg {
					return HostDeleteMsg{Name: h.name}
				}
			}
			return v, nil
		}
		switch {
		case ui.MatchKey(msg, keys.Left), ui.MatchKey(msg, keys.Help):
			return v, func() tea.Msg { return SwitchBackFromHostsMsg{} }
		case ui.MatchKey(msg, keys.Down):
			if v.cursor < len(v.hosts)-1 {
				v.cursor++
			}
		case ui.MatchKey(msg, keys.Up):
			if v.cursor > 0 {
				v.cursor--
			}
		case ui.MatchKey(msg, keys.Right), msg.Type == tea.KeyEnter:
			h := v.hosts[v.cursor]
			return v, func() tea.Msg {
				return HostSelectedMsg{Name: h.name, URL: h.url}
			}
		case msg.String() == "d", msg.String() == "x":
			if v.cursor > 0 {
				v.pendingDelete = true
				return v, nil
			}
		case msg.String() == "a":
			v.addMode = true
			v.addField = 0
			v.addName = ""
			v.addURL = ""
			return v, nil
		case ui.MatchKey(msg, keys.Bottom):
			v.cursor = len(v.hosts) - 1
		case msg.String() == "g":
			v.cursor = 0
		}
	}
	return v, nil
}

func (v HostsView) handleAddInput(msg tea.KeyMsg) (HostsView, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		v.addMode = false
		return v, nil
	case tea.KeyTab:
		v.addField = (v.addField + 1) % 2
		return v, nil
	case tea.KeyEnter:
		if v.addName != "" && v.addURL != "" {
			v.addMode = false
			name := v.addName
			url := v.addURL
			return v, func() tea.Msg {
				return HostAddMsg{Name: name, URL: url}
			}
		}
		if v.addName != "" && v.addURL == "" {
			v.addField = 1
		}
		return v, nil
	case tea.KeyBackspace:
		if v.addField == 0 && len(v.addName) > 0 {
			v.addName = v.addName[:len(v.addName)-1]
		} else if v.addField == 1 && len(v.addURL) > 0 {
			v.addURL = v.addURL[:len(v.addURL)-1]
		}
		return v, nil
	case tea.KeySpace:
		if v.addField == 1 {
			v.addURL += " "
		}
		return v, nil
	case tea.KeyRunes:
		if v.addField == 0 {
			v.addName += string(msg.Runes)
		} else {
			v.addURL += string(msg.Runes)
		}
		return v, nil
	}
	return v, nil
}

func (v HostsView) View() string {
	colName := 16
	colURL := 40
	colStatus := 12

	header := ui.HeaderRowStyle.Render(
		fmt.Sprintf("  %-*s %-*s %s",
			colName, "NAME",
			colURL, "URL",
			padRight("STATUS", colStatus)),
	)

	var rows []string
	rows = append(rows, header)

	for i, h := range v.hosts {
		url := h.url
		if url == "" {
			url = "unix:///var/run/docker.sock"
		}

		var status string
		if h.active {
			status = ui.RunningStyle.Render("● connected")
		} else {
			status = ui.MutedStyle.Render("○")
		}

		marker := "  "
		if h.active {
			marker = ui.RunningStyle.Render("● ")
		}

		row := fmt.Sprintf("%s%-*s %-*s %s",
			marker,
			colName, truncate(h.name, colName),
			colURL, truncate(url, colURL),
			status,
		)

		if i == v.cursor {
			rows = append(rows, renderSelectedRow(row, v.width-2))
		} else {
			rows = append(rows, row)
		}
	}

	result := lipgloss.JoinVertical(lipgloss.Left, rows...)

	if v.addMode {
		labelStyle := ui.MutedStyle
		inputStyle := lipgloss.NewStyle().Foreground(ui.ColorPrimary)
		cursor := "\u2588"

		nameValue := v.addName
		urlValue := v.addURL
		if v.addField == 0 {
			nameValue += cursor
		} else {
			urlValue += cursor
		}

		form := "\n\n" +
			labelStyle.Render("  Add host:") + "\n" +
			labelStyle.Render("  Name: ") + inputStyle.Render(nameValue) + "\n" +
			labelStyle.Render("  URL:  ") + inputStyle.Render(urlValue) + "\n" +
			ui.MutedStyle.Render("  Tab=switch field  Enter=save  Esc=cancel")

		result += form
	}

	return result
}

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

type SwitchToNetworksMsg struct{ ProjectName string }
type SwitchBackFromNetworksMsg struct{}
type NetworksLoadedMsg struct{ Networks []docker.Network }

type NetworksView struct {
	projectName   string
	networks      []docker.Network
	cursor        int
	width, height int
	pendingG      bool
	err           error
	detail        bool // true = showing detail of selected network
}

func NewNetworksView(projectName string) NetworksView {
	return NetworksView{projectName: projectName}
}

func (v NetworksView) SetSize(w, h int) NetworksView {
	v.width = w
	v.height = h
	return v
}

func (v NetworksView) Breadcrumb() string  { return "› " + v.projectName + " › Networks" }
func (v NetworksView) ProjectName() string { return v.projectName }

func LoadNetworks(client *docker.Client, projectName string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return NetworksLoadedMsg{}
		}
		all, err := client.ListNetworks(context.Background())
		if err != nil {
			return NetworksLoadedMsg{}
		}
		if projectName == "" {
			return NetworksLoadedMsg{Networks: all}
		}
		var filtered []docker.Network
		for _, n := range all {
			if n.Project == projectName {
				filtered = append(filtered, n)
			}
		}
		return NetworksLoadedMsg{Networks: filtered}
	}
}

func (v NetworksView) Update(msg tea.Msg, keys ui.KeyMap) (NetworksView, tea.Cmd) {
	switch msg := msg.(type) {
	case NetworksLoadedMsg:
		v.networks = msg.Networks
		if v.cursor >= len(v.networks) && len(v.networks) > 0 {
			v.cursor = len(v.networks) - 1
		}
		return v, nil

	case tea.MouseMsg:
		if !v.detail {
			if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
				row := msg.Y - 4
				if row >= 0 && row < len(v.networks) {
					v.cursor = row
				}
			}
			if msg.Button == tea.MouseButtonWheelDown {
				if v.cursor < len(v.networks)-1 {
					v.cursor++
				}
			}
			if msg.Button == tea.MouseButtonWheelUp {
				if v.cursor > 0 {
					v.cursor--
				}
			}
		}
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg, keys)
	}
	return v, nil
}

func (v NetworksView) handleKeyMsg(msg tea.KeyMsg, keys ui.KeyMap) (NetworksView, tea.Cmd) {
	if v.pendingG {
		v.pendingG = false
		if msg.String() == "g" {
			v.cursor = 0
			return v, nil
		}
	}

	switch {
	case ui.MatchKey(msg, keys.Down):
		if !v.detail && v.cursor < len(v.networks)-1 {
			v.cursor++
		}
	case ui.MatchKey(msg, keys.Up):
		if !v.detail && v.cursor > 0 {
			v.cursor--
		}
	case ui.MatchKey(msg, keys.Bottom):
		if !v.detail && len(v.networks) > 0 {
			v.cursor = len(v.networks) - 1
		}
	case msg.String() == "g":
		if !v.detail {
			v.pendingG = true
		}
	case ui.MatchKey(msg, keys.Right):
		if !v.detail && len(v.networks) > 0 {
			v.detail = true
		}
	case ui.MatchKey(msg, keys.Left):
		if v.detail {
			v.detail = false
			return v, nil
		}
		return v, func() tea.Msg { return SwitchBackFromNetworksMsg{} }
	}
	return v, nil
}

func (v NetworksView) View() string {
	if v.err != nil {
		return ui.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if len(v.networks) == 0 {
		return ui.MutedStyle.Render("No networks found")
	}

	if v.detail {
		return v.detailView()
	}

	return v.listView()
}

func (v NetworksView) listView() string {
	colName := 28
	colDriver := 10
	colSubnet := 20

	header := ui.HeaderRowStyle.Render(
		fmt.Sprintf("%-*s %-*s %-*s %s", colName, "NAME", colDriver, "DRIVER", colSubnet, "SUBNET", "CONTAINERS"),
	)

	var rows []string
	rows = append(rows, header)

	for i, n := range v.networks {
		containers := strings.Join(n.Containers, ", ")
		if containers == "" {
			containers = "-"
		}
		row := fmt.Sprintf("%-*s %-*s %-*s %s",
			colName, truncate(n.Name, colName),
			colDriver, n.Driver,
			colSubnet, n.Subnet,
			containers,
		)
		if i == v.cursor {
			row = renderSelectedRow(row, v.width-2)
		}
		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (v NetworksView) detailView() string {
	n := v.networks[v.cursor]
	var lines []string

	lines = append(lines,
		fmt.Sprintf("  %-14s %s", "Name", n.Name),
		fmt.Sprintf("  %-14s %s", "ID", n.ID),
		fmt.Sprintf("  %-14s %s", "Driver", n.Driver),
		fmt.Sprintf("  %-14s %s", "Subnet", n.Subnet),
		fmt.Sprintf("  %-14s %s", "Gateway", n.Gateway),
		"",
	)

	if len(n.Containers) > 0 {
		lines = append(lines, "  Containers")
		for _, c := range n.Containers {
			lines = append(lines, "    "+c)
		}
	} else {
		lines = append(lines, "  Containers: none")
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

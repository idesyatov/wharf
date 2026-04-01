package views

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/ui"
)

type SwitchToVolumesMsg struct{ ProjectName string }
type SwitchBackFromVolumesMsg struct{}
type VolumesLoadedMsg struct{ Volumes []docker.Volume }
type VolumeRemovedMsg struct {
	Err        error
	VolumeName string
}
type VolumesPrunedMsg struct {
	Err       error
	Count     int
	Reclaimed uint64
}

type VolumesView struct {
	projectName    string
	volumes        []docker.Volume
	cursor         int
	width, height  int
	pendingG       bool
	err            error
	pendingRemove  bool
	pendingPrune   bool
	pendingVolName string
}

func NewVolumesView(projectName string) VolumesView {
	return VolumesView{projectName: projectName}
}

func (v VolumesView) SetSize(w, h int) VolumesView {
	v.width = w
	v.height = h
	return v
}

func (v VolumesView) Breadcrumb() string     { return "› " + v.projectName + " › Volumes" }
func (v VolumesView) ProjectName() string    { return v.projectName }
func (v VolumesView) PendingRemove() bool    { return v.pendingRemove }
func (v VolumesView) PendingPrune() bool     { return v.pendingPrune }
func (v VolumesView) PendingVolName() string { return v.pendingVolName }

func LoadVolumes(client *docker.Client, projectName string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return VolumesLoadedMsg{}
		}
		all, err := client.ListVolumes(context.Background())
		if err != nil {
			return VolumesLoadedMsg{}
		}
		if projectName == "" {
			return VolumesLoadedMsg{Volumes: all}
		}
		var filtered []docker.Volume
		for _, vol := range all {
			if vol.Project == projectName {
				filtered = append(filtered, vol)
			}
		}
		return VolumesLoadedMsg{Volumes: filtered}
	}
}

func (v VolumesView) Update(msg tea.Msg, keys ui.KeyMap) (VolumesView, tea.Cmd) {
	switch msg := msg.(type) {
	case VolumesLoadedMsg:
		v.volumes = msg.Volumes
		if v.cursor >= len(v.volumes) && len(v.volumes) > 0 {
			v.cursor = len(v.volumes) - 1
		}
		return v, nil

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			row := msg.Y - 4
			if row >= 0 && row < len(v.volumes) {
				v.cursor = row
			}
		}
		if msg.Button == tea.MouseButtonWheelDown {
			if v.cursor < len(v.volumes)-1 {
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
		// Confirmation handlers
		if v.pendingRemove {
			v.pendingRemove = false
			if ui.MatchKey(msg, keys.Confirm) {
				name := v.pendingVolName
				return v, func() tea.Msg { return RemoveVolumeMsg{Name: name} }
			}
			return v, nil
		}
		if v.pendingPrune {
			v.pendingPrune = false
			if ui.MatchKey(msg, keys.Confirm) {
				return v, func() tea.Msg { return PruneVolumesActionMsg{} }
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

		switch {
		case ui.MatchKey(msg, keys.Down):
			if v.cursor < len(v.volumes)-1 {
				v.cursor++
			}
		case ui.MatchKey(msg, keys.Up):
			if v.cursor > 0 {
				v.cursor--
			}
		case ui.MatchKey(msg, keys.Bottom):
			if len(v.volumes) > 0 {
				v.cursor = len(v.volumes) - 1
			}
		case msg.String() == "g":
			v.pendingG = true
		case ui.MatchKey(msg, keys.Remove):
			if len(v.volumes) > 0 {
				v.pendingRemove = true
				v.pendingVolName = v.volumes[v.cursor].Name
			}
		case ui.MatchKey(msg, keys.Prune):
			v.pendingPrune = true
		case ui.MatchKey(msg, keys.Left):
			return v, func() tea.Msg { return SwitchBackFromVolumesMsg{} }
		}
	}
	return v, nil
}

func (v VolumesView) View() string {
	if v.err != nil {
		return ui.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if len(v.volumes) == 0 {
		return ui.MutedStyle.Render("No volumes found")
	}

	colName := 35
	colDriver := 12

	header := ui.HeaderRowStyle.Render(
		fmt.Sprintf("%-*s %-*s %s", colName, "NAME", colDriver, "DRIVER", "MOUNTPOINT"),
	)

	var rows []string
	rows = append(rows, header)

	for i, vol := range v.volumes {
		row := fmt.Sprintf("%-*s %-*s %s",
			colName, truncate(vol.Name, colName),
			colDriver, vol.Driver,
			vol.Mountpoint,
		)
		if i == v.cursor {
			row = renderSelectedRow(row, v.width-2)
		}
		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// Internal messages for App to handle
type RemoveVolumeMsg struct{ Name string }
type PruneVolumesActionMsg struct{}

func RemoveVolume(client *docker.Client, name string) tea.Cmd {
	return func() tea.Msg {
		err := client.RemoveVolume(context.Background(), name)
		return VolumeRemovedMsg{Err: err, VolumeName: name}
	}
}

func PruneVolumes(client *docker.Client) tea.Cmd {
	return func() tea.Msg {
		count, reclaimed, err := client.PruneVolumes(context.Background())
		return VolumesPrunedMsg{Err: err, Count: count, Reclaimed: reclaimed}
	}
}

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

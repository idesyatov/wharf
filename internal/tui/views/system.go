package views

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/ui"
)

type SwitchToSystemMsg struct{}
type SwitchBackFromSystemMsg struct{}
type SystemDfLoadedMsg struct{ Df docker.SystemDf }
type SystemPruneMsg struct{}
type SystemPruneDoneMsg struct{ Err error }

type SystemView struct {
	df           docker.SystemDf
	loaded       bool
	width        int
	height       int
	pendingPrune bool
}

func NewSystemView() SystemView {
	return SystemView{}
}

func (v SystemView) SetSize(w, h int) SystemView {
	v.width = w
	v.height = h
	return v
}

func (v SystemView) Breadcrumb() string { return "› System" }
func (v SystemView) PendingPrune() bool { return v.pendingPrune }

func LoadSystemDf(client *docker.Client) tea.Cmd {
	return func() tea.Msg {
		df, err := client.SystemDiskUsage(context.Background())
		if err != nil {
			return SystemDfLoadedMsg{}
		}
		return SystemDfLoadedMsg{Df: df}
	}
}

func SystemPrune() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("docker", "system", "prune", "-a", "--volumes", "-f")
		err := cmd.Run()
		return SystemPruneDoneMsg{Err: err}
	}
}

func (v SystemView) Update(msg tea.Msg, keys ui.KeyMap) (SystemView, tea.Cmd) {
	switch msg := msg.(type) {
	case SystemDfLoadedMsg:
		v.df = msg.Df
		v.loaded = true
		return v, nil

	case tea.KeyMsg:
		if v.pendingPrune {
			v.pendingPrune = false
			if ui.MatchKey(msg, keys.Confirm) {
				return v, func() tea.Msg { return SystemPruneMsg{} }
			}
			return v, nil
		}

		switch {
		case ui.MatchKey(msg, keys.Left):
			return v, func() tea.Msg { return SwitchBackFromSystemMsg{} }
		case ui.MatchKey(msg, keys.Prune):
			v.pendingPrune = true
		}
	}
	return v, nil
}

func (v SystemView) View() string {
	if !v.loaded {
		return ui.MutedStyle.Render("Loading...")
	}

	colType := 20
	colCount := 10
	colSize := 12

	header := ui.HeaderRowStyle.Render(
		fmt.Sprintf("%-*s %*s %*s", colType, "TYPE", colCount, "COUNT", colSize, "SIZE"),
	)

	total := v.df.ImagesSize + v.df.ContainersSize + v.df.VolumesSize + v.df.BuildCacheSize

	rows := []string{
		header,
		fmt.Sprintf("%-*s %*d %*s", colType, "Images", colCount, v.df.ImagesCount, colSize, FormatBytes(uint64(v.df.ImagesSize))),
		fmt.Sprintf("%-*s %*d %*s", colType, "Containers", colCount, v.df.ContainersCount, colSize, FormatBytes(uint64(v.df.ContainersSize))),
		fmt.Sprintf("%-*s %*d %*s", colType, "Volumes", colCount, v.df.VolumesCount, colSize, FormatBytes(uint64(v.df.VolumesSize))),
		fmt.Sprintf("%-*s %*s %*s", colType, "Build Cache", colCount, "-", colSize, FormatBytes(uint64(v.df.BuildCacheSize))),
		fmt.Sprintf("%-*s %*s %*s", colType, "", colCount, "", colSize, strings.Repeat("-", colSize)),
		fmt.Sprintf("%-*s %*s %*s", colType, "Total", colCount, "", colSize, FormatBytes(uint64(total))),
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

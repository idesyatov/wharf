package views

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/ui"
)

type SwitchToEnvMsg struct {
	ProjectName string
	ProjectPath string
}
type SwitchBackFromEnvMsg struct{}

type EnvFileView struct {
	projectName   string
	filePath      string
	lines         []string
	scroll        int
	width, height int
	err           error
	pendingG      bool
}

var envFiles = []string{".env", ".env.local", ".env.development"}

func NewEnvFileView(projectName, projectPath string) EnvFileView {
	v := EnvFileView{projectName: projectName}

	for _, name := range envFiles {
		p := filepath.Join(projectPath, name)
		data, err := os.ReadFile(p)
		if err == nil {
			v.filePath = name
			v.lines = strings.Split(string(data), "\n")
			return v
		}
	}
	v.err = os.ErrNotExist
	return v
}

func (v EnvFileView) SetSize(w, h int) EnvFileView {
	v.width = w
	v.height = h
	return v
}

func (v EnvFileView) Breadcrumb() string  { return "› " + v.projectName + " › " + v.filePath }
func (v EnvFileView) ProjectName() string { return v.projectName }
func (v EnvFileView) FileName() string    { return v.filePath }

func (v EnvFileView) Update(msg tea.Msg, keys ui.KeyMap) (EnvFileView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if v.pendingG {
			v.pendingG = false
			if msg.String() == "g" {
				v.scroll = 0
				return v, nil
			}
		}

		switch {
		case ui.MatchKey(msg, keys.Down):
			maxScroll := len(v.lines) - v.visibleHeight()
			if maxScroll < 0 {
				maxScroll = 0
			}
			if v.scroll < maxScroll {
				v.scroll++
			}
		case ui.MatchKey(msg, keys.Up):
			if v.scroll > 0 {
				v.scroll--
			}
		case ui.MatchKey(msg, keys.Bottom):
			maxScroll := len(v.lines) - v.visibleHeight()
			if maxScroll < 0 {
				maxScroll = 0
			}
			v.scroll = maxScroll
		case msg.String() == "g":
			v.pendingG = true
		case ui.MatchKey(msg, keys.Left):
			return v, func() tea.Msg { return SwitchBackFromEnvMsg{} }
		}
	}
	return v, nil
}

func (v EnvFileView) visibleHeight() int {
	h := v.height - 2
	if h < 1 {
		h = 1
	}
	return h
}

func (v EnvFileView) View() string {
	if v.err != nil {
		return ui.MutedStyle.Render("No .env file found")
	}

	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorPrimary)

	visible := v.visibleHeight()
	start := v.scroll
	end := start + visible
	if end > len(v.lines) {
		end = len(v.lines)
	}

	var rows []string
	for i := start; i < end; i++ {
		line := v.lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			rows = append(rows, "")
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			rows = append(rows, ui.MutedStyle.Render(line))
			continue
		}
		if idx := strings.Index(line, "="); idx >= 0 {
			key := line[:idx]
			val := line[idx:]
			rows = append(rows, keyStyle.Render(key)+val)
			continue
		}
		rows = append(rows, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

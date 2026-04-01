package views

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/ui"
)

type SwitchToComposeMsg struct {
	ProjectName string
	ProjectPath string
}
type SwitchBackFromComposeMsg struct{}
type EditComposeMsg struct{ FilePath string }
type EditComposeDoneMsg struct {
	Err      error
	FilePath string
}

var composeFileNames = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yml",
	"docker-compose.yaml",
}

type ComposeView struct {
	projectName string
	projectPath string
	fileName    string
	filePath    string
	lines       []string
	scroll      int
	width       int
	height      int
	err         error
	pendingG    bool
}

func NewComposeView(projectName, projectPath string) ComposeView {
	v := ComposeView{projectName: projectName, projectPath: projectPath}

	for _, name := range composeFileNames {
		p := filepath.Join(projectPath, name)
		data, err := os.ReadFile(p)
		if err == nil {
			v.fileName = name
			v.filePath = p
			v.lines = strings.Split(string(data), "\n")
			return v
		}
	}

	v.err = os.ErrNotExist
	return v
}

func (v ComposeView) Breadcrumb() string {
	return "› " + v.projectName + " › " + v.fileName
}

func (v ComposeView) FileName() string {
	return v.fileName
}

func (v ComposeView) ProjectName() string {
	return v.projectName
}

func (v ComposeView) ProjectPath() string {
	return v.projectPath
}

func (v ComposeView) FilePath() string {
	return v.filePath
}

func (v ComposeView) SetSize(w, h int) ComposeView {
	v.width = w
	v.height = h
	return v
}

func (v ComposeView) Update(msg tea.Msg, keys ui.KeyMap) (ComposeView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if v.pendingG {
			v.pendingG = false
			if msg.String() == "g" {
				v.scroll = 0
				return v, nil
			}
		}

		visible := v.visibleHeight()
		maxScroll := len(v.lines) - visible
		if maxScroll < 0 {
			maxScroll = 0
		}

		switch {
		case ui.MatchKey(msg, keys.Down):
			if v.scroll < maxScroll {
				v.scroll++
			}
		case ui.MatchKey(msg, keys.Up):
			if v.scroll > 0 {
				v.scroll--
			}
		case ui.MatchKey(msg, keys.Bottom):
			v.scroll = maxScroll
		case msg.String() == "g":
			v.pendingG = true
		case ui.MatchKey(msg, keys.Edit):
			if v.filePath != "" {
				fp := v.filePath
				return v, func() tea.Msg { return EditComposeMsg{FilePath: fp} }
			}
		case ui.MatchKey(msg, keys.Left):
			return v, func() tea.Msg { return SwitchBackFromComposeMsg{} }
		}
	}
	return v, nil
}

func (v ComposeView) visibleHeight() int {
	h := v.height - 2
	if h < 1 {
		h = 1
	}
	return h
}

func (v ComposeView) View() string {
	if v.err != nil {
		return ui.ErrorStyle.Render("No compose file found")
	}

	visible := v.visibleHeight()
	start := v.scroll
	end := start + visible
	if end > len(v.lines) {
		end = len(v.lines)
	}

	var rendered []string
	for _, line := range v.lines[start:end] {
		rendered = append(rendered, highlightYAML(line))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rendered...)
}

var (
	yamlKeyStyle     = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
	yamlCommentStyle = lipgloss.NewStyle().Foreground(ui.ColorMuted)
)

func highlightYAML(line string) string {
	trimmed := strings.TrimSpace(line)

	if strings.HasPrefix(trimmed, "#") {
		return yamlCommentStyle.Render(line)
	}

	if idx := strings.Index(line, ":"); idx >= 0 {
		// Check it's not inside a string value (simple heuristic: key is before first colon)
		key := line[:idx+1]
		val := line[idx+1:]
		return yamlKeyStyle.Render(key) + val
	}

	return line
}

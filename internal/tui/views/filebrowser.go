package views

import (
	"context"
	"fmt"
	"path"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/ui"
)

type SwitchToFileBrowserMsg struct {
	ContainerID   string
	ContainerName string
}
type SwitchBackFromFileBrowserMsg struct{}

type FileBrowserListMsg struct {
	Entries []FileEntry
	Path    string
	Err     error
}
type FileBrowserReadMsg struct {
	Content string
	Name    string
	Err     error
}

type FileEntry struct {
	Name  string
	IsDir bool
	Size  string
	Perms string
}

type FileBrowserView struct {
	containerID   string
	containerName string
	currentPath   string
	entries       []FileEntry
	cursor        int
	scroll        int
	fileContent   string
	fileName      string
	viewingFile   bool
	width, height int
	loading       bool
	err           error
	pendingG      bool
}

func NewFileBrowserView(containerID, containerName string) FileBrowserView {
	return FileBrowserView{
		containerID:   containerID,
		containerName: containerName,
		currentPath:   "/",
		loading:       true,
	}
}

func (v FileBrowserView) SetSize(w, h int) FileBrowserView {
	v.width = w
	v.height = h
	return v
}

func (v FileBrowserView) ContainerID() string   { return v.containerID }
func (v FileBrowserView) ContainerName() string { return v.containerName }

func (v FileBrowserView) Breadcrumb() string {
	p := v.currentPath
	if v.viewingFile {
		p = path.Join(v.currentPath, v.fileName)
	}
	return "› " + v.containerName + " › Files [" + p + "]"
}

func (v FileBrowserView) Update(msg tea.Msg, keys ui.KeyMap) (FileBrowserView, tea.Cmd) {
	switch msg := msg.(type) {
	case FileBrowserListMsg:
		v.loading = false
		if msg.Err != nil {
			v.err = msg.Err
			return v, nil
		}
		v.entries = msg.Entries
		v.currentPath = msg.Path
		v.cursor = 0
		v.scroll = 0
		v.err = nil
		return v, nil

	case FileBrowserReadMsg:
		v.loading = false
		if msg.Err != nil {
			v.err = msg.Err
			return v, nil
		}
		v.viewingFile = true
		v.fileContent = msg.Content
		v.fileName = msg.Name
		v.scroll = 0
		v.err = nil
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg, keys)
	}
	return v, nil
}

func (v FileBrowserView) handleKeyMsg(msg tea.KeyMsg, keys ui.KeyMap) (FileBrowserView, tea.Cmd) {
	if v.loading {
		if ui.MatchKey(msg, keys.Left) {
			return v, func() tea.Msg { return SwitchBackFromFileBrowserMsg{} }
		}
		return v, nil
	}

	if v.viewingFile {
		return v.handleFileViewKey(msg, keys)
	}
	return v.handleDirViewKey(msg, keys)
}

func (v FileBrowserView) handleFileViewKey(msg tea.KeyMsg, keys ui.KeyMap) (FileBrowserView, tea.Cmd) {
	lines := strings.Split(v.fileContent, "\n")
	visible := v.visibleHeight()
	switch {
	case ui.MatchKey(msg, keys.Down):
		if v.scroll < len(lines)-visible {
			v.scroll++
		}
	case ui.MatchKey(msg, keys.Up):
		if v.scroll > 0 {
			v.scroll--
		}
	case ui.MatchKey(msg, keys.Bottom):
		if len(lines) > visible {
			v.scroll = len(lines) - visible
		}
	case ui.MatchKey(msg, keys.Left):
		v.viewingFile = false
		v.fileContent = ""
		v.fileName = ""
		v.scroll = 0
	}
	return v, nil
}

func (v FileBrowserView) handleDirViewKey(msg tea.KeyMsg, keys ui.KeyMap) (FileBrowserView, tea.Cmd) {
	if v.pendingG {
		v.pendingG = false
		if msg.String() == "g" {
			v.cursor = 0
			v.scroll = 0
			return v, nil
		}
	}

	switch {
	case ui.MatchKey(msg, keys.Down):
		if v.cursor < len(v.entries)-1 {
			v.cursor++
		}
	case ui.MatchKey(msg, keys.Up):
		if v.cursor > 0 {
			v.cursor--
		}
	case ui.MatchKey(msg, keys.Bottom):
		if len(v.entries) > 0 {
			v.cursor = len(v.entries) - 1
		}
	case msg.String() == "g":
		v.pendingG = true
	case ui.MatchKey(msg, keys.Right):
		return v.openSelected()
	case ui.MatchKey(msg, keys.Left):
		return v.goUp()
	}
	return v, nil
}

type FileBrowserNavigateMsg struct {
	ContainerID string
	Path        string
	IsFile      bool
}

func (v FileBrowserView) openSelected() (FileBrowserView, tea.Cmd) {
	if v.cursor >= len(v.entries) {
		return v, nil
	}
	entry := v.entries[v.cursor]
	fullPath := path.Join(v.currentPath, entry.Name)
	v.loading = true
	return v, func() tea.Msg {
		return FileBrowserNavigateMsg{
			ContainerID: v.containerID,
			Path:        fullPath,
			IsFile:      !entry.IsDir,
		}
	}
}

func (v FileBrowserView) goUp() (FileBrowserView, tea.Cmd) {
	if v.currentPath == "/" {
		return v, func() tea.Msg { return SwitchBackFromFileBrowserMsg{} }
	}
	parent := path.Dir(v.currentPath)
	v.loading = true
	return v, func() tea.Msg {
		return FileBrowserNavigateMsg{
			ContainerID: v.containerID,
			Path:        parent,
		}
	}
}

func (v FileBrowserView) visibleHeight() int {
	h := v.height - 3
	if h < 1 {
		h = 1
	}
	return h
}

func (v FileBrowserView) View() string {
	if v.loading {
		return ui.MutedStyle.Render("Loading...")
	}
	if v.err != nil {
		return ui.ErrorStyle.Render("Error: " + v.err.Error())
	}
	if v.viewingFile {
		return v.renderFileView()
	}
	return v.renderDirView()
}

func (v FileBrowserView) renderDirView() string {
	if len(v.entries) == 0 {
		return ui.MutedStyle.Render("Empty directory")
	}

	colPerms := 12
	colSize := 10

	header := ui.HeaderRowStyle.Render(
		fmt.Sprintf("  %-*s %*s  %s", colPerms, "PERMS", colSize, "SIZE", "NAME"),
	)

	visible := v.visibleHeight()
	start := 0
	if v.cursor >= visible {
		start = v.cursor - visible + 1
	}
	end := start + visible
	if end > len(v.entries) {
		end = len(v.entries)
	}

	var rows []string
	rows = append(rows, header)

	for i := start; i < end; i++ {
		e := v.entries[i]
		name := e.Name
		nameStyle := lipgloss.NewStyle()
		if e.IsDir {
			name += "/"
			nameStyle = nameStyle.Foreground(ui.ColorPrimary)
		}

		row := fmt.Sprintf("  %-*s %*s  %s",
			colPerms, e.Perms,
			colSize, e.Size,
			nameStyle.Render(name),
		)

		if i == v.cursor {
			plainRow := fmt.Sprintf("  %-*s %*s  %s",
				colPerms, e.Perms,
				colSize, e.Size,
				name,
			)
			rows = append(rows, renderSelectedRow(plainRow, v.width-2))
		} else {
			rows = append(rows, row)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (v FileBrowserView) renderFileView() string {
	lines := strings.Split(v.fileContent, "\n")
	visible := v.visibleHeight()

	start := v.scroll
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + visible
	if end > len(lines) {
		end = len(lines)
	}

	var rows []string
	for _, line := range lines[start:end] {
		rows = append(rows, "  "+line)
	}

	return strings.Join(rows, "\n")
}

func LoadDirectoryListing(client *docker.Client, containerID, dirPath string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return FileBrowserListMsg{Err: fmt.Errorf("no docker client")}
		}
		ctx := context.Background()
		output, err := client.ExecOutput(ctx, containerID, []string{"ls", "-la", dirPath})
		if err != nil {
			return FileBrowserListMsg{Err: err, Path: dirPath}
		}
		entries := parseLsOutput(output)
		return FileBrowserListMsg{Entries: entries, Path: dirPath}
	}
}

func LoadFileContent(client *docker.Client, containerID, filePath string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return FileBrowserReadMsg{Err: fmt.Errorf("no docker client")}
		}
		ctx := context.Background()
		output, err := client.ExecOutput(ctx, containerID, []string{"cat", filePath})
		if err != nil {
			return FileBrowserReadMsg{Err: err, Name: path.Base(filePath)}
		}
		if len(output) > 100*1024 {
			output = output[:100*1024] + "\n... (truncated)"
		}
		return FileBrowserReadMsg{Content: output, Name: path.Base(filePath)}
	}
}

func parseLsOutput(output string) []FileEntry {
	var entries []FileEntry
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		// Strip docker exec multiplexing header bytes
		if len(line) > 0 && line[0] < 32 {
			if len(line) > 8 {
				line = strings.TrimSpace(line[8:])
			} else {
				continue
			}
		}
		if line == "" || strings.HasPrefix(line, "total") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		name := strings.Join(fields[8:], " ")
		if name == "." || name == ".." {
			continue
		}
		entries = append(entries, FileEntry{
			Name:  name,
			IsDir: fields[0][0] == 'd',
			Size:  fields[4],
			Perms: fields[0],
		})
	}
	return entries
}

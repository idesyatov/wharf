package views

import (
	"context"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/ui"
)

type LogErrorMsg struct{ Err error }
type SwitchBackFromLogsMsg struct{}

type logReaderMsg struct{ reader io.ReadCloser }
type logLineWithReader struct {
	line   string
	reader io.ReadCloser
}

type LogsView struct {
	containerID   string
	containerName string
	lines         []string
	maxLines      int
	follow        bool
	offset        int
	width         int
	height        int
	pendingG      bool
	cancel        context.CancelFunc
	searchMode    bool
	searchText    string
	searchHits    []int
	searchCursor  int
}

func NewLogsView(ct docker.Container, maxLines int) LogsView {
	return LogsView{
		containerID:   ct.ID,
		containerName: ct.Name,
		maxLines:      maxLines,
		follow:        true,
	}
}

func (v LogsView) ContainerName() string {
	return v.containerName
}

func (v LogsView) SetSize(w, h int) LogsView {
	v.width = w
	v.height = h
	return v
}

type SaveLogsMsg struct{ Path string }
type LogsSavedMsg struct {
	Path  string
	Lines int
	Err   error
}

func (v LogsView) Breadcrumb() string { return "› " + v.containerName + " [LOGS]" }
func (v LogsView) Lines() []string    { return v.lines }
func (v LogsView) SearchMode() bool   { return v.searchMode }
func (v LogsView) SearchText() string { return v.searchText }
func (v LogsView) SearchInfo() string {
	if v.searchText == "" || len(v.searchHits) == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d matches", v.searchCursor+1, len(v.searchHits))
}

func (v *LogsView) SetCancel(cancel context.CancelFunc) {
	v.cancel = cancel
}

func (v *LogsView) Close() {
	if v.cancel != nil {
		v.cancel()
		v.cancel = nil
	}
}

func StartLogStream(client *docker.Client, containerID string, tail int) tea.Cmd {
	return func() tea.Msg {
		reader, err := client.ContainerLogs(context.Background(), containerID, tail)
		if err != nil {
			return LogErrorMsg{Err: err}
		}
		return logReaderMsg{reader: reader}
	}
}

func (v LogsView) Update(msg tea.Msg, keys ui.KeyMap) (LogsView, tea.Cmd) {
	switch msg := msg.(type) {
	case logReaderMsg:
		return v, readNextLine(msg.reader)

	case logLineWithReader:
		v = v.processLogLines(msg.line)
		return v, readNextLine(msg.reader)

	case LogErrorMsg:
		if msg.Err != nil {
			v.lines = append(v.lines, ui.ErrorStyle.Render("Error: "+msg.Err.Error()))
		} else {
			v.lines = append(v.lines, ui.MutedStyle.Render("[END OF LOGS]"))
		}
		return v, nil

	case tea.MouseMsg:
		visibleLines := v.visibleHeight()
		if msg.Button == tea.MouseButtonWheelUp {
			if v.offset < len(v.lines)-visibleLines {
				v.offset++
				v.follow = false
			}
		}
		if msg.Button == tea.MouseButtonWheelDown {
			if v.offset > 0 {
				v.offset--
			}
			if v.offset == 0 {
				v.follow = true
			}
		}
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg, keys)
	}

	return v, nil
}

func (v LogsView) handleKeyMsg(msg tea.KeyMsg, keys ui.KeyMap) (LogsView, tea.Cmd) {
	if v.searchMode {
		return v.handleSearchInput(msg)
	}

	if v.pendingG {
		v.pendingG = false
		if msg.String() == "g" {
			v.offset = len(v.lines)
			v.follow = false
			return v, nil
		}
	}

	visibleLines := v.visibleHeight()

	switch {
	case ui.MatchKey(msg, keys.Up):
		if v.offset < len(v.lines)-visibleLines {
			v.offset++
			v.follow = false
		}
	case ui.MatchKey(msg, keys.Down):
		if v.offset > 0 {
			v.offset--
		}
		if v.offset == 0 {
			v.follow = true
		}
	case ui.MatchKey(msg, keys.Bottom):
		v.offset = 0
		v.follow = true
	case msg.String() == "g":
		v.pendingG = true
	case ui.MatchKey(msg, keys.Follow):
		v.follow = !v.follow
		if v.follow {
			v.offset = 0
		}
	case ui.MatchKey(msg, keys.Search):
		v.searchMode = true
		v.searchText = ""
		v.follow = false
	case msg.String() == "n":
		v.nextSearchHit()
	case msg.String() == "N":
		v.prevSearchHit()
	case ui.MatchKey(msg, keys.SaveLogs):
		return v, func() tea.Msg { return SaveLogsMsg{Path: ""} }
	case ui.MatchKey(msg, keys.Left):
		v.Close()
		return v, func() tea.Msg { return SwitchBackFromLogsMsg{} }
	}
	return v, nil
}

func (v LogsView) handleSearchInput(msg tea.KeyMsg) (LogsView, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		v.searchMode = false
		v.applySearch()
	case tea.KeyEsc:
		v.searchMode = false
		v.searchText = ""
		v.searchHits = nil
		v.searchCursor = 0
	case tea.KeyBackspace:
		if len(v.searchText) > 0 {
			v.searchText = v.searchText[:len(v.searchText)-1]
		}
	default:
		if msg.Type == tea.KeyRunes {
			v.searchText += string(msg.Runes)
		}
	}
	return v, nil
}

func (v LogsView) processLogLines(raw string) LogsView {
	for _, l := range strings.Split(raw, "\n") {
		if l != "" {
			v.lines = append(v.lines, l)
			if v.searchText != "" && strings.Contains(strings.ToLower(l), strings.ToLower(v.searchText)) {
				v.searchHits = append(v.searchHits, len(v.lines)-1)
			}
		}
	}
	if v.maxLines > 0 && len(v.lines) > v.maxLines {
		excess := len(v.lines) - v.maxLines
		v.lines = v.lines[excess:]
		adjusted := v.searchHits[:0]
		for _, idx := range v.searchHits {
			if newIdx := idx - excess; newIdx >= 0 {
				adjusted = append(adjusted, newIdx)
			}
		}
		v.searchHits = adjusted
		if v.searchCursor >= len(v.searchHits) && len(v.searchHits) > 0 {
			v.searchCursor = len(v.searchHits) - 1
		}
	}
	if v.follow {
		v.offset = 0
	}
	return v
}

func (v LogsView) visibleHeight() int {
	h := v.height - 3
	if h < 1 {
		h = 1
	}
	return h
}

func (v LogsView) View() string {
	mode := "FOLLOWING"
	if !v.follow {
		mode = "PAUSED"
	}
	title := ui.ProjectTitleStyle.Render("Logs: " + v.containerName + " [" + mode + "]")

	visible := v.visibleHeight()
	end := len(v.lines) - v.offset
	if end < 0 {
		end = 0
	}
	start := end - visible
	if start < 0 {
		start = 0
	}

	var content string
	if len(v.lines) == 0 {
		content = ui.MutedStyle.Render("Waiting for logs...")
	} else {
		var rendered []string
		for i := start; i < end; i++ {
			line := v.lines[i]
			if v.searchText != "" && v.isSearchHit(i) {
				line = ui.SearchHighlightStyle.Render(line)
			}
			rendered = append(rendered, line)
		}
		content = strings.Join(rendered, "\n")
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, content)
}

func (v *LogsView) applySearch() {
	v.searchHits = nil
	v.searchCursor = 0
	if v.searchText == "" {
		return
	}
	q := strings.ToLower(v.searchText)
	for i, line := range v.lines {
		if strings.Contains(strings.ToLower(line), q) {
			v.searchHits = append(v.searchHits, i)
		}
	}
	if len(v.searchHits) > 0 {
		v.scrollToHit()
	}
}

func (v *LogsView) nextSearchHit() {
	if len(v.searchHits) == 0 {
		return
	}
	v.searchCursor = (v.searchCursor + 1) % len(v.searchHits)
	v.scrollToHit()
}

func (v *LogsView) prevSearchHit() {
	if len(v.searchHits) == 0 {
		return
	}
	v.searchCursor--
	if v.searchCursor < 0 {
		v.searchCursor = len(v.searchHits) - 1
	}
	v.scrollToHit()
}

func (v *LogsView) scrollToHit() {
	if len(v.searchHits) == 0 {
		return
	}
	hitLine := v.searchHits[v.searchCursor]
	v.offset = len(v.lines) - hitLine - v.visibleHeight()/2
	if v.offset < 0 {
		v.offset = 0
	}
	if v.offset > len(v.lines)-v.visibleHeight() {
		v.offset = len(v.lines) - v.visibleHeight()
	}
	v.follow = false
}

func (v LogsView) isSearchHit(lineIndex int) bool {
	for _, idx := range v.searchHits {
		if idx == lineIndex {
			return true
		}
	}
	return false
}

func readNextLine(reader io.ReadCloser) tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 8192)
		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				return LogErrorMsg{Err: nil}
			}
			return LogErrorMsg{Err: err}
		}
		lines := parseDockerLogFrames(string(buf[:n]))
		if len(lines) > 0 {
			return logLineWithReader{line: strings.Join(lines, "\n"), reader: reader}
		}
		// Empty frame — try again
		return readNextLine(reader)()
	}
}

func parseDockerLogFrames(data string) []string {
	var lines []string
	for len(data) > 0 {
		if len(data) < 8 {
			lines = append(lines, strings.TrimRight(data, "\n\r"))
			break
		}
		size := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])
		data = data[8:]
		if size > len(data) {
			size = len(data)
		}
		frame := data[:size]
		data = data[size:]
		for _, line := range strings.Split(strings.TrimRight(frame, "\n\r"), "\n") {
			if line != "" {
				lines = append(lines, line)
			}
		}
	}
	return lines
}

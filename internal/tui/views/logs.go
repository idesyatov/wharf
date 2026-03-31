package views

import (
	"context"
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
		// msg.line may contain multiple lines joined by \n
		for _, l := range strings.Split(msg.line, "\n") {
			if l != "" {
				v.lines = append(v.lines, l)
			}
		}
		if v.maxLines > 0 && len(v.lines) > v.maxLines {
			v.lines = v.lines[len(v.lines)-v.maxLines:]
		}
		if v.follow {
			v.offset = 0
		}
		return v, readNextLine(msg.reader)

	case LogErrorMsg:
		if msg.Err != nil {
			v.lines = append(v.lines, ui.ErrorStyle.Render("Error: "+msg.Err.Error()))
		} else {
			v.lines = append(v.lines, ui.MutedStyle.Render("[END OF LOGS]"))
		}
		return v, nil

	case tea.KeyMsg:
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
		case ui.MatchKey(msg, keys.Left):
			v.Close()
			return v, func() tea.Msg { return SwitchBackFromLogsMsg{} }
		}
	}

	return v, nil
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
		content = strings.Join(v.lines[start:end], "\n")
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, content)
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

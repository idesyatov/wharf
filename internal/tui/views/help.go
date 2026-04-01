package views

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idesyatov/wharf/internal/ui"
)

type SwitchToHelpMsg struct{}
type SwitchBackFromHelpMsg struct{}

type HelpView struct {
	width  int
	height int
	scroll int
}

func (v HelpView) Breadcrumb() string { return "Help" }

func NewHelpView() HelpView {
	return HelpView{}
}

func (v HelpView) SetSize(w, h int) HelpView {
	v.width = w
	v.height = h
	return v
}

func (v HelpView) Update(msg tea.Msg, keys ui.KeyMap) (HelpView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case ui.MatchKey(msg, keys.Help), ui.MatchKey(msg, keys.Left):
			return v, func() tea.Msg { return SwitchBackFromHelpMsg{} }
		case ui.MatchKey(msg, keys.Down):
			v.scroll++
		case ui.MatchKey(msg, keys.Up):
			if v.scroll > 0 {
				v.scroll--
			}
		}
	}
	return v, nil
}

var helpText = strings.TrimSpace(`
  Navigation
    j / k / ↑ / ↓    Move cursor up/down
    h / ← / Esc      Go back
    l / → / Enter     Select / drill down
    gg                Jump to top
    G                 Jump to bottom

  Actions (Services view)
    s                 Start service
    S                 Stop service
    r                 Restart service
    L                 View logs
    e                 Exec into container (shell)
    b                 Build service
    B                 Build all services
    u                 Compose up (start project)
    d                 Compose stop (stop, keep containers)
    X                 Compose down (stop and REMOVE containers)
    R                 Compose restart
    c                 View compose file
    v                 View volumes
    n                 View networks

  Projects view
    i                 View images
    D                 System disk usage
    E                 Docker events
    Space             Toggle select (bulk)
    Esc               Clear all selections

  Volumes view
    x                 Remove volume (confirm)
    P                 Prune dangling volumes

  Images view
    p                 Pull image
    P                 Prune unused images

  System view
    P                 Prune all unused resources

  Sorting (Projects / Services view)
    1-6               Sort by column (repeat to reverse)

  Logs view
    f                 Toggle follow mode
    w                 Save logs to file
    j / k             Scroll up/down
    /                 Search in logs
    n                 Next match
    N                 Previous match

  Services view extras
    .                 Preview .env file

  Clipboard
    y                 Copy container ID / project name
    Y                 Copy extended info (Detail view)

  Command mode (:)
    :q                Quit
    :theme dark       Switch to dark theme
    :theme light      Switch to light theme
    :host             Show Docker host
    :version          Show version
    :save [path]      Save logs (in Logs view)
    :help             Show this help

  General
    *                 Toggle bookmark (Projects view)
    /                 Filter (search)
    ?                 Show this help
    q                 Quit

                      Press ? or Esc to close
`)

func (v HelpView) View() string {
	lines := strings.Split(helpText, "\n")
	visible := v.height - 2
	if visible < 1 {
		visible = 1
	}
	start := v.scroll
	if start >= len(lines) {
		start = len(lines) - 1
	}
	end := start + visible
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}

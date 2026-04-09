package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idesyatov/wharf/internal/ui"
)

type SwitchToHelpMsg struct{}
type SwitchBackFromHelpMsg struct{}

type HelpView struct {
	width      int
	height     int
	scroll     int
	searchMode bool
	searchText string
	searchHits []int
	searchCur  int
	pendingG   bool
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

func (v HelpView) SearchMode() bool   { return v.searchMode }
func (v HelpView) SearchText() string { return v.searchText }

func (v HelpView) SearchInfo() string {
	if v.searchText == "" {
		return ""
	}
	if len(v.searchHits) == 0 {
		return "no matches"
	}
	return fmt.Sprintf("%d/%d matches", v.searchCur+1, len(v.searchHits))
}

func (v HelpView) Update(msg tea.Msg, keys ui.KeyMap) (HelpView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if v.searchMode {
			return v.handleSearchInput(msg)
		}
		return v.handleKeyMsg(msg, keys)
	}
	return v, nil
}

func (v HelpView) handleSearchInput(msg tea.KeyMsg) (HelpView, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		v.searchMode = false
		v.applySearch()
	case tea.KeyEsc:
		v.searchMode = false
		v.searchText = ""
		v.searchHits = nil
		v.searchCur = 0
	case tea.KeyBackspace:
		if len(v.searchText) > 0 {
			v.searchText = v.searchText[:len(v.searchText)-1]
		}
	case tea.KeySpace:
		v.searchText += " "
	case tea.KeyRunes:
		v.searchText += string(msg.Runes)
	}
	return v, nil
}

func (v HelpView) handleKeyMsg(msg tea.KeyMsg, keys ui.KeyMap) (HelpView, tea.Cmd) {
	if v.pendingG {
		v.pendingG = false
		if msg.String() == "g" {
			v.scroll = 0
			return v, nil
		}
	}

	switch {
	case ui.MatchKey(msg, keys.Help), ui.MatchKey(msg, keys.Left):
		return v, func() tea.Msg { return SwitchBackFromHelpMsg{} }
	case ui.MatchKey(msg, keys.Down):
		v.scroll++
	case ui.MatchKey(msg, keys.Up):
		if v.scroll > 0 {
			v.scroll--
		}
	case ui.MatchKey(msg, keys.Bottom):
		// handled by clamp below
		v.scroll = len(strings.Split(helpText, "\n"))
	case msg.String() == "g":
		v.pendingG = true
	case msg.String() == "/":
		v.searchMode = true
		v.searchText = ""
		return v, nil
	case msg.String() == "n":
		v.nextMatch()
	case msg.String() == "N":
		v.prevMatch()
	}

	// Clamp scroll
	totalLines := len(strings.Split(helpText, "\n"))
	visible := v.height - 2
	if visible < 1 {
		visible = 1
	}
	maxScroll := totalLines - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if v.scroll > maxScroll {
		v.scroll = maxScroll
	}
	if v.scroll < 0 {
		v.scroll = 0
	}

	return v, nil
}

func (v *HelpView) applySearch() {
	v.searchHits = nil
	v.searchCur = 0
	if v.searchText == "" {
		return
	}
	q := strings.ToLower(v.searchText)
	for i, line := range strings.Split(helpText, "\n") {
		if strings.Contains(strings.ToLower(line), q) {
			v.searchHits = append(v.searchHits, i)
		}
	}
	if len(v.searchHits) > 0 {
		v.scroll = v.searchHits[0]
	}
}

func (v *HelpView) nextMatch() {
	if len(v.searchHits) == 0 {
		return
	}
	v.searchCur = (v.searchCur + 1) % len(v.searchHits)
	v.scroll = v.searchHits[v.searchCur]
}

func (v *HelpView) prevMatch() {
	if len(v.searchHits) == 0 {
		return
	}
	v.searchCur--
	if v.searchCur < 0 {
		v.searchCur = len(v.searchHits) - 1
	}
	v.scroll = v.searchHits[v.searchCur]
}

func (v HelpView) isSearchHit(lineIndex int) bool {
	for _, idx := range v.searchHits {
		if idx == lineIndex {
			return true
		}
	}
	return false
}

var helpText = strings.TrimSpace(`
  Navigation
    j / k / ↑ / ↓    Move cursor up/down
    h / ← / Esc      Go back
    l / → / Enter     Select / drill down
    gg                Jump to top
    G                 Jump to bottom

  Actions (Services view)
    t                 Resource monitor (top) for container
    s                 Start service
    S                 Stop service
    r                 Restart service
    x                 Remove stopped container (confirm)
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
    H                 Host Switcher (saved hosts)
    t                 Resource monitor (top) for project
    i                 View images
    D                 System disk usage
    E                 Docker events
    Space             Toggle select (bulk)
    Esc               Clear all selections

  Host Switcher
    Enter             Connect to selected host
    a                 Add new host
    d                 Delete host (confirm)

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

  Compose view
    e                 Edit compose file ($EDITOR)

  File Browser
    F                 Browse container filesystem
    Enter/l           Open directory or view file
    h/Esc             Go back / exit file view

  Services view extras
    .                 Preview .env file

  Clipboard
    y                 Copy container ID / project name
    Y                 Copy extended info (Detail view)

  Command mode (:)
    :q                Quit
    :theme dark       Switch to dark theme
    :theme light      Switch to light theme
    :host [name/url]  Show / switch Docker host
    :hosts            Open Host Switcher
    :up [profile]     Compose up with profile (:up *, :up debug)
    :down [profile]   Compose down with profile
    :version          Show version
    :save [path]      Save logs (in Logs view)
    :edit             Edit compose file (Compose view)
    :go <name>        Jump to project by name
    :exec <name>      Exec into container by name
    :validate [name]  Validate compose file
    :help             Show this help
    Tab               Autocomplete command

  Custom Commands (from config)
    1-9               Run custom command (see ~/.config/wharf/config.yaml)

  General
    *                 Toggle bookmark (Projects view)
    /                 Filter / search
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

	var result []string
	for i := start; i < end; i++ {
		line := lines[i]
		if v.searchText != "" && v.isSearchHit(i) {
			line = ui.SearchHighlightStyle.Render(line)
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

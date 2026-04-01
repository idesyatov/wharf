package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/ui"
)

type SwitchToEventsMsg struct{}
type SwitchBackFromEventsMsg struct{}
type EventReceivedMsg struct{ Event docker.Event }

type EventsView struct {
	events   []docker.Event
	scroll   int
	width    int
	height   int
	pendingG bool
}

func NewEventsView(events []docker.Event) EventsView {
	return EventsView{events: events}
}

func (v EventsView) Breadcrumb() string { return "› Events" }

func (v EventsView) SetSize(w, h int) EventsView {
	v.width = w
	v.height = h
	return v
}

func (v EventsView) Update(msg tea.Msg, keys ui.KeyMap) (EventsView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if v.pendingG {
			v.pendingG = false
			if msg.String() == "g" {
				v.scroll = 0
				return v, nil
			}
		}

		maxScroll := len(v.events) - v.visibleHeight()
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
		case ui.MatchKey(msg, keys.Left):
			return v, func() tea.Msg { return SwitchBackFromEventsMsg{} }
		}
	}
	return v, nil
}

func (v EventsView) visibleHeight() int {
	h := v.height - 2
	if h < 1 {
		h = 1
	}
	return h
}

func (v EventsView) View() string {
	if len(v.events) == 0 {
		return ui.MutedStyle.Render("No events")
	}

	colTime := 12
	colType := 12
	colAction := 12

	header := ui.HeaderRowStyle.Render(
		fmt.Sprintf("%-*s %-*s %-*s %s", colTime, "TIME", colType, "TYPE", colAction, "ACTION", "ACTOR"),
	)

	visible := v.visibleHeight()
	start := v.scroll
	end := start + visible
	if end > len(v.events) {
		end = len(v.events)
	}

	var rows []string
	rows = append(rows, header)

	for i := start; i < end; i++ {
		e := v.events[i]
		timeStr := e.Time.Format("15:04:05")

		row := fmt.Sprintf("%-*s %-*s %-*s %s",
			colTime, timeStr,
			colType, truncate(e.Type, colType),
			colAction, truncate(e.Action, colAction),
			e.Actor,
		)
		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

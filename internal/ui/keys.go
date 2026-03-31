package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type KeyMap struct {
	Quit      key.Binding
	ForceQuit key.Binding
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Top       key.Binding
	Bottom    key.Binding
	Search    key.Binding
	Help      key.Binding
	Start     key.Binding
	Stop      key.Binding
	Restart   key.Binding
	Logs       key.Binding
	Follow     key.Binding
	ComposeUp   key.Binding
	ComposeDown key.Binding
	Confirm     key.Binding
	Compose     key.Binding
	VolumesKey  key.Binding
	NetworksKey key.Binding
	Remove      key.Binding
	Prune       key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "force quit"),
		),
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("h", "esc", "left"),
			key.WithHelp("h/←/esc", "back"),
		),
		Right: key.NewBinding(
			key.WithKeys("l", "enter", "right"),
			key.WithHelp("l/→/enter", "select"),
		),
		Top: key.NewBinding(
			key.WithHelp("gg", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Start: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start"),
		),
		Stop: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "stop"),
		),
		Restart: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restart"),
		),
		Logs: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "logs"),
		),
		Follow: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "follow"),
		),
		ComposeUp: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "compose up"),
		),
		ComposeDown: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "compose down"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "confirm"),
		),
		Compose: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "compose file"),
		),
		VolumesKey: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "volumes"),
		),
		NetworksKey: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "networks"),
		),
		Remove: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "remove"),
		),
		Prune: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "prune"),
		),
	}
}

func ApplyKeyBindings(km KeyMap, bindings map[string]string) KeyMap {
	if len(bindings) == 0 {
		return km
	}
	apply := func(b *key.Binding, name string) {
		if v, ok := bindings[name]; ok {
			*b = key.NewBinding(key.WithKeys(v), key.WithHelp(v, name))
		}
	}
	apply(&km.Quit, "quit")
	apply(&km.Up, "up")
	apply(&km.Down, "down")
	apply(&km.Left, "back")
	apply(&km.Right, "select")
	apply(&km.Search, "filter")
	apply(&km.Help, "help")
	apply(&km.Start, "start")
	apply(&km.Stop, "stop")
	apply(&km.Restart, "restart")
	apply(&km.Logs, "logs")
	apply(&km.Follow, "follow")
	apply(&km.ComposeUp, "compose_up")
	apply(&km.ComposeDown, "compose_down")
	apply(&km.Compose, "compose")
	apply(&km.VolumesKey, "volumes")
	apply(&km.NetworksKey, "networks")
	apply(&km.Remove, "remove")
	apply(&km.Prune, "prune")
	return km
}

func MatchKey(msg tea.KeyMsg, binding key.Binding) bool {
	for _, k := range binding.Keys() {
		if msg.String() == k {
			return true
		}
	}
	return false
}

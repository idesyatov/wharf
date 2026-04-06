package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCmdModeEnterCancel(t *testing.T) {
	var c CmdMode

	c.Enter()
	if !c.IsActive() {
		t.Error("expected active after Enter")
	}
	if c.Input() != "" {
		t.Error("expected empty input after Enter")
	}

	c.Cancel()
	if c.IsActive() {
		t.Error("expected inactive after Cancel")
	}
}

func TestCmdModeExecute(t *testing.T) {
	var c CmdMode
	c.Enter()
	c.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("host prod")})

	cmd := c.Execute()
	if cmd != "host prod" {
		t.Errorf("expected 'host prod', got %q", cmd)
	}
	if c.IsActive() {
		t.Error("expected inactive after Execute")
	}
	if len(c.history) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(c.history))
	}
}

func TestCmdModeBackspace(t *testing.T) {
	var c CmdMode
	c.Enter()
	c.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("abc")})
	c.HandleKey(tea.KeyMsg{Type: tea.KeyBackspace})

	if c.Input() != "ab" {
		t.Errorf("expected 'ab' after backspace, got %q", c.Input())
	}
}

func TestCmdModeComplete(t *testing.T) {
	var c CmdMode
	c.Enter()
	c.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hos")})
	c.HandleKey(tea.KeyMsg{Type: tea.KeyTab})

	// "hos" matches "host" and "hosts" — should complete to "host" (common prefix)
	input := c.Input()
	if input != "host" {
		t.Errorf("expected 'host' after Tab, got %q", input)
	}
}

func TestCmdModeCompleteUnique(t *testing.T) {
	var c CmdMode
	c.Enter()
	c.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("val")})
	c.HandleKey(tea.KeyMsg{Type: tea.KeyTab})

	// "val" uniquely matches "validate"
	input := c.Input()
	if input != "validate " {
		t.Errorf("expected 'validate ' after Tab, got %q", input)
	}
}

func TestCmdModeHostNameComplete(t *testing.T) {
	var c CmdMode
	c.SetHostNames([]string{"local", "prod", "staging"})

	c.Enter()
	c.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("host pr")})
	c.HandleKey(tea.KeyMsg{Type: tea.KeyTab})

	input := c.Input()
	if input != "host prod" {
		t.Errorf("expected 'host prod' after Tab, got %q", input)
	}
}

func TestCmdModeHistory(t *testing.T) {
	var c CmdMode

	// Execute two commands
	c.Enter()
	c.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("host")})
	c.Execute()

	c.Enter()
	c.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("version")})
	c.Execute()

	// Navigate history
	c.Enter()
	c.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	if c.Input() != "version" {
		t.Errorf("expected 'version' on Up, got %q", c.Input())
	}

	c.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	if c.Input() != "host" {
		t.Errorf("expected 'host' on second Up, got %q", c.Input())
	}

	c.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	if c.Input() != "version" {
		t.Errorf("expected 'version' on Down, got %q", c.Input())
	}
}

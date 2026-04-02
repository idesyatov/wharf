package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var commandList = []string{
	"q",
	"q!",
	"help",
	"host",
	"theme",
	"version",
	"save",
	"edit",
	"go",
	"exec",
	"validate",
}

type CmdMode struct {
	active  bool
	input   string
	history []string
	histIdx int
}

func (c *CmdMode) IsActive() bool { return c.active }
func (c *CmdMode) Input() string  { return c.input }

func (c *CmdMode) Enter() {
	c.active = true
	c.input = ""
	c.histIdx = len(c.history)
}

func (c *CmdMode) Cancel() {
	c.active = false
	c.input = ""
}

func (c *CmdMode) Execute() string {
	cmd := c.input
	c.active = false
	c.input = ""
	if cmd != "" {
		c.history = append(c.history, cmd)
	}
	c.histIdx = len(c.history)
	return cmd
}

func (c *CmdMode) HandleKey(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyBackspace:
		if len(c.input) > 0 {
			c.input = c.input[:len(c.input)-1]
		}
	case tea.KeySpace:
		c.input += " "
	case tea.KeyUp:
		if c.histIdx > 0 {
			c.histIdx--
			c.input = c.history[c.histIdx]
		}
	case tea.KeyDown:
		if c.histIdx < len(c.history)-1 {
			c.histIdx++
			c.input = c.history[c.histIdx]
		} else {
			c.histIdx = len(c.history)
			c.input = ""
		}
	case tea.KeyTab:
		c.complete()
	case tea.KeyRunes:
		c.input += string(msg.Runes)
	}
}

func (c *CmdMode) complete() {
	input := strings.TrimSpace(c.input)
	if input == "" {
		return
	}
	parts := strings.Fields(input)
	if len(parts) > 1 {
		return
	}
	prefix := strings.ToLower(parts[0])
	var matches []string
	for _, cmd := range commandList {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}
	if len(matches) == 1 {
		c.input = matches[0] + " "
	} else if len(matches) > 1 {
		common := commonPrefix(matches)
		if len(common) > len(prefix) {
			c.input = common
		}
	}
}

func commonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

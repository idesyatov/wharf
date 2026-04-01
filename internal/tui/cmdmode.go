package tui

import tea "github.com/charmbracelet/bubbletea"

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
	case tea.KeyRunes:
		c.input += string(msg.Runes)
	}
}

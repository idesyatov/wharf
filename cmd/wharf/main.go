package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idesyatov/wharf/internal/config"
	"github.com/idesyatov/wharf/internal/tui"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--config" || os.Args[1] == "-c") {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(cfg.String())
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(tui.NewApp(cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

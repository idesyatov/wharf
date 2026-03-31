package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idesyatov/wharf/internal/config"
	"github.com/idesyatov/wharf/internal/tui"
	"github.com/idesyatov/wharf/internal/ui"
	"github.com/idesyatov/wharf/internal/version"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("wharf " + version.Full())
		return
	}

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

	if cfg.DockerHost != "" {
		os.Setenv("DOCKER_HOST", cfg.DockerHost)
	}

	theme, themeErr := ui.LoadTheme(cfg.Theme)
	if themeErr != nil {
		fmt.Fprintf(os.Stderr, "Theme warning: %v, using default\n", themeErr)
	} else {
		ui.ApplyTheme(theme)
	}

	p := tea.NewProgram(tui.NewApp(cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/tui/views"
	"github.com/idesyatov/wharf/internal/ui"
	"github.com/idesyatov/wharf/internal/util"
	"github.com/idesyatov/wharf/internal/version"
)

func (a App) executeAction(action string, svc docker.Service) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var err error
		switch action {
		case "start":
			err = a.docker.StartService(ctx, svc)
		case "stop":
			err = a.docker.StopService(ctx, svc)
		case "restart":
			err = a.docker.RestartService(ctx, svc)
		}
		return views.ActionResultMsg{
			Err:         err,
			Action:      action,
			ServiceName: svc.Name,
		}
	}
}

func (a App) executeCompose(action, projectName, projectPath string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var err error
		switch action {
		case "up":
			err = docker.ComposeUp(ctx, projectPath)
		case "stop":
			err = docker.ComposeStop(ctx, projectPath)
		case "down":
			err = docker.ComposeDown(ctx, projectPath)
		case "restart":
			err = docker.ComposeRestart(ctx, projectPath)
		}
		return views.ComposeResultMsg{
			Err:         err,
			Action:      action,
			ProjectName: projectName,
		}
	}
}

func (a App) executeBatchCompose(action string, projects []docker.Project) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var errors []string
		for _, p := range projects {
			var err error
			switch action {
			case "up":
				err = docker.ComposeUp(ctx, p.Path)
			case "stop":
				err = docker.ComposeStop(ctx, p.Path)
			case "down":
				err = docker.ComposeDown(ctx, p.Path)
			case "restart":
				err = docker.ComposeRestart(ctx, p.Path)
			}
			if err != nil {
				errors = append(errors, p.Name+": "+err.Error())
			}
		}
		if len(errors) > 0 {
			return views.ComposeResultMsg{
				Err:         fmt.Errorf("%s", strings.Join(errors, "; ")),
				Action:      action,
				ProjectName: fmt.Sprintf("%d projects", len(projects)),
			}
		}
		return views.ComposeResultMsg{
			Action:      action,
			ProjectName: fmt.Sprintf("%d projects", len(projects)),
		}
	}
}

func (a *App) executeCommand(cmd string) tea.Cmd {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "q", "q!":
		a.cleanup()
		return tea.Quit
	case "help":
		return func() tea.Msg { return views.SwitchToHelpMsg{} }
	case "hosts":
		return func() tea.Msg { return views.SwitchToHostsMsg{} }
	case "host":
		return a.cmdHost(parts)
	case "theme":
		return a.cmdTheme(parts)
	case "version":
		return a.cmdVersion()
	case "save":
		return a.cmdSave(parts)
	case "edit":
		return a.cmdEdit()
	case "go":
		return a.cmdGo(parts)
	case "exec":
		return a.cmdExec(parts)
	case "validate":
		return a.cmdValidate(parts)
	case "up":
		return a.cmdUp(parts)
	case "down":
		return a.cmdDown(parts)
	default:
		return a.notifyErr("Unknown command: "+cmd, 2*time.Second)
	}
}

func (a *App) notifyErr(msg string, dur time.Duration) tea.Cmd {
	a.notification = msg
	a.notificationErr = true
	a.notificationExp = time.Now().Add(dur)
	return tea.Tick(dur, func(time.Time) tea.Msg { return notificationClearMsg{} })
}

func (a *App) cmdHost(parts []string) tea.Cmd {
	if len(parts) < 2 {
		host := "local"
		if a.cfg.DockerHost != "" {
			host = a.cfg.DockerHost
		} else if dh := os.Getenv("DOCKER_HOST"); dh != "" {
			host = dh
		}
		a.notification = "Docker host: " + host
		a.notificationErr = false
		a.notificationExp = time.Now().Add(3 * time.Second)
		return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
	}
	newHost := parts[1]
	if newHost == "local" {
		newHost = ""
	} else if !strings.Contains(newHost, "://") {
		if entry := a.cfg.FindHost(newHost); entry != nil {
			newHost = entry.URL
		} else {
			return a.notifyErr("Unknown host: "+parts[1], 3*time.Second)
		}
	}
	return func() tea.Msg { return switchHostMsg{host: newHost} }
}

func (a *App) cmdTheme(parts []string) tea.Cmd {
	if len(parts) < 2 {
		return a.notifyErr("Usage: :theme dark|light", 3*time.Second)
	}
	theme, err := ui.LoadTheme(parts[1])
	if err != nil {
		a.notification = "Theme error: " + err.Error()
		a.notificationErr = true
	} else {
		ui.ApplyTheme(theme)
		a.notification = "Theme: " + parts[1]
		a.notificationErr = false
	}
	a.notificationExp = time.Now().Add(2 * time.Second)
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
}

func (a *App) cmdVersion() tea.Cmd {
	a.notification = "wharf " + version.Full()
	a.notificationErr = false
	a.notificationExp = time.Now().Add(3 * time.Second)
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
}

func (a *App) cmdSave(parts []string) tea.Cmd {
	if a.state == viewLogs {
		path := ""
		if len(parts) > 1 {
			path = parts[1]
		}
		return func() tea.Msg { return views.SaveLogsMsg{Path: path} }
	}
	return a.notifyErr("save: only available in Logs view", 2*time.Second)
}

func (a *App) cmdEdit() tea.Cmd {
	if a.state == viewCompose && a.composeView.FilePath() != "" {
		editor := util.DetectEditor()
		fp := a.composeView.FilePath()
		c := exec.Command(editor, fp)
		return tea.ExecProcess(c, func(err error) tea.Msg {
			return views.EditComposeDoneMsg{Err: err, FilePath: fp}
		})
	}
	return a.notifyErr("edit: only available in Compose view", 2*time.Second)
}

func (a *App) cmdGo(parts []string) tea.Cmd {
	if len(parts) < 2 {
		return a.notifyErr("Usage: :go <project-name>", 2*time.Second)
	}
	query := strings.ToLower(parts[1])
	for _, p := range a.projectsView.Projects() {
		if strings.Contains(strings.ToLower(p.Name), query) {
			a.state = viewServices
			a.servicesView = views.NewServicesView(p, a.cfg.CustomCommands).SetSize(a.width, a.height-7)
			if p.Path != "" {
				if profiles, err := docker.ParseProfiles(p.Path); err == nil {
					a.servicesView = a.servicesView.SetProfiles(profiles)
				}
			}
			a.notification = "→ " + p.Name
			a.notificationErr = false
			a.notificationExp = time.Now().Add(2 * time.Second)
			return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
		}
	}
	return a.notifyErr("Project not found: "+parts[1], 2*time.Second)
}

func (a *App) cmdExec(parts []string) tea.Cmd {
	if len(parts) < 2 {
		return a.notifyErr("Usage: :exec <container-name>", 2*time.Second)
	}
	ct := a.findContainerByName(parts[1])
	if ct == nil {
		return a.notifyErr("Container not found: "+parts[1], 2*time.Second)
	}
	shell := a.docker.DetectShell(context.Background(), ct.ID)
	banner := fmt.Sprintf(
		"echo '─────────────────────────────────────────' && "+
			"echo '  ⚓ Wharf — Container Shell' && "+
			"echo '  Container: %s' && "+
			"echo '  Image:     %s' && "+
			"echo '  Shell:     %s' && "+
			"echo '  Exit:      type exit or Ctrl+D' && "+
			"echo '─────────────────────────────────────────' && "+
			"exec %s",
		ct.Name, ct.Image, shell, shell,
	)
	c := exec.Command("docker", "exec", "-it", ct.ID, "sh", "-c", banner)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return views.ExecDoneMsg{Err: err}
	})
}

func (a *App) cmdValidate(parts []string) tea.Cmd {
	var projectPath string
	if a.state == viewServices {
		projectPath = a.servicesView.ProjectPath()
	}
	if len(parts) > 1 {
		query := strings.ToLower(parts[1])
		for _, p := range a.projectsView.Projects() {
			if strings.Contains(strings.ToLower(p.Name), query) {
				projectPath = p.Path
				break
			}
		}
	}
	if projectPath == "" {
		return a.notifyErr("validate: navigate to a project or use :validate <name>", 3*time.Second)
	}
	return a.validateCompose(projectPath)
}

func (a *App) cmdUp(parts []string) tea.Cmd {
	if a.state != viewServices {
		return a.notifyErr("up: only in Services view", 3*time.Second)
	}
	projectName := a.servicesView.ProjectName()
	projectPath := a.servicesView.ProjectPath()
	if projectPath == "" {
		return a.notifyErr("up: project path unknown", 2*time.Second)
	}

	if len(parts) < 2 {
		return a.executeCompose("up", projectName, projectPath)
	}

	profile := parts[1]
	var profiles []string
	if profile == "*" {
		if p, err := docker.ParseProfiles(projectPath); err == nil {
			profiles = p.AllProfiles
		}
	} else {
		profiles = []string{profile}
	}
	pName := projectName
	pPath := projectPath
	return func() tea.Msg {
		return views.ComposeResultMsg{
			ProjectName: pName,
			Action:      "up",
			Err:         docker.ComposeUpWithProfiles(context.Background(), pPath, profiles),
		}
	}
}

func (a *App) cmdDown(parts []string) tea.Cmd {
	if a.state != viewServices {
		return a.notifyErr("down: only in Services view", 3*time.Second)
	}
	projectName := a.servicesView.ProjectName()
	projectPath := a.servicesView.ProjectPath()
	if projectPath == "" {
		return a.notifyErr("down: project path unknown", 2*time.Second)
	}

	if len(parts) < 2 {
		return a.executeCompose("down", projectName, projectPath)
	}

	profile := parts[1]
	var profiles []string
	if profile == "*" {
		if p, err := docker.ParseProfiles(projectPath); err == nil {
			profiles = p.AllProfiles
		}
	} else {
		profiles = []string{profile}
	}
	pName := projectName
	pPath := projectPath
	return func() tea.Msg {
		return views.ComposeResultMsg{
			ProjectName: pName,
			Action:      "down",
			Err:         docker.ComposeDownWithProfiles(context.Background(), pPath, profiles),
		}
	}
}

func (a *App) findContainerByName(name string) *docker.Container {
	query := strings.ToLower(name)
	for _, p := range a.projectsView.Projects() {
		for _, svc := range p.Services {
			for i, ct := range svc.Containers {
				if strings.Contains(strings.ToLower(ct.Name), query) {
					return &svc.Containers[i]
				}
			}
		}
	}
	return nil
}

func (a *App) validateCompose(projectPath string) tea.Cmd {
	return func() tea.Msg {
		composePath, err := docker.FindComposeFile(projectPath)
		if err != nil {
			return composeValidateResultMsg{Err: err}
		}
		cmd := exec.Command("docker", "compose", "-f", composePath, "config", "--quiet")
		cmd.Dir = projectPath
		output, err := cmd.CombinedOutput()
		return composeValidateResultMsg{Err: err, Output: string(output)}
	}
}

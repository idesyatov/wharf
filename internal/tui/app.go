package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/config"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/tui/views"
	"github.com/idesyatov/wharf/internal/ui"
)

type viewState int

const (
	viewProjects viewState = iota
	viewServices
	viewDetail
	viewLogs
	viewCompose
	viewVolumes
	viewNetworks
	viewHelp
)

type notificationClearMsg struct{}

type App struct {
	state           viewState
	prevState       viewState
	projectsView    views.ProjectsView
	servicesView    views.ServicesView
	detailView      views.DetailView
	logsView        views.LogsView
	composeView     views.ComposeView
	volumesView     views.VolumesView
	networksView    views.NetworksView
	helpView        views.HelpView
	docker          *docker.Client
	cfg             *config.Config
	width           int
	height          int
	keys            ui.KeyMap
	err             error
	notification    string
	notificationErr bool
	notificationExp time.Time
	pendingColon    bool
}

func NewApp(cfg *config.Config) App {
	client, err := docker.NewClient()
	keys := ui.DefaultKeyMap()
	keys = ui.ApplyKeyBindings(keys, cfg.KeyBindings)
	return App{
		state:        viewProjects,
		projectsView: views.NewProjectsView(cfg.PollInterval),
		keys:         keys,
		docker:       client,
		cfg:          cfg,
		err:          err,
	}
}

func (a App) Init() tea.Cmd {
	if a.err != nil || a.docker == nil {
		return nil
	}
	return tea.Batch(
		views.LoadProjects(a.docker),
		views.TickCmd(a.cfg.PollInterval),
	)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.projectsView = a.projectsView.SetSize(msg.Width, msg.Height-4)
		a.servicesView = a.servicesView.SetSize(msg.Width, msg.Height-4)
		a.detailView = a.detailView.SetSize(msg.Width, msg.Height-4)
		a.logsView = a.logsView.SetSize(msg.Width, msg.Height-4)
		a.composeView = a.composeView.SetSize(msg.Width, msg.Height-4)
		a.volumesView = a.volumesView.SetSize(msg.Width, msg.Height-4)
		a.networksView = a.networksView.SetSize(msg.Width, msg.Height-4)
		a.helpView = a.helpView.SetSize(msg.Width, msg.Height-4)
		return a, nil

	// --- View switching ---

	case views.SwitchToServicesMsg:
		a.state = viewServices
		a.servicesView = views.NewServicesView(msg.Project).SetSize(a.width, a.height-4)
		return a, nil

	case views.SwitchToProjectsMsg:
		a.state = viewProjects
		return a, nil

	case views.SwitchToDetailMsg:
		a.prevState = a.state
		a.state = viewDetail
		a.detailView = views.NewDetailView(msg.Service).SetSize(a.width, a.height-4)
		return a, views.LoadContainerDetail(a.docker, msg.Service)

	case views.SwitchBackFromDetailMsg:
		a.state = viewServices
		return a, nil

	case views.SwitchToLogsMsg:
		a.prevState = a.state
		a.state = viewLogs
		a.logsView = views.NewLogsView(msg.Container, a.cfg.MaxLogLines).SetSize(a.width, a.height-4)
		return a, views.StartLogStream(a.docker, msg.Container.ID, a.cfg.LogTail)

	case views.SwitchBackFromLogsMsg:
		if a.prevState == viewDetail {
			a.state = viewDetail
		} else {
			a.state = viewServices
		}
		return a, nil

	case views.SwitchToComposeMsg:
		a.prevState = a.state
		a.state = viewCompose
		a.composeView = views.NewComposeView(msg.ProjectName, msg.ProjectPath).SetSize(a.width, a.height-4)
		return a, nil

	case views.SwitchBackFromComposeMsg:
		a.state = viewServices
		return a, nil

	case views.SwitchToVolumesMsg:
		a.prevState = a.state
		a.state = viewVolumes
		a.volumesView = views.NewVolumesView(msg.ProjectName).SetSize(a.width, a.height-4)
		return a, views.LoadVolumes(a.docker, msg.ProjectName)

	case views.SwitchBackFromVolumesMsg:
		a.state = viewServices
		return a, nil

	case views.VolumesLoadedMsg:
		a.volumesView, _ = a.volumesView.Update(msg, a.keys)
		return a, nil

	case views.VolumeRemovedMsg:
		if msg.Err != nil {
			a.notification = "remove " + msg.VolumeName + ": " + msg.Err.Error()
			a.notificationErr = true
		} else {
			a.notification = "removed " + msg.VolumeName
			a.notificationErr = false
		}
		a.notificationExp = time.Now().Add(3 * time.Second)
		return a, tea.Batch(
			views.LoadVolumes(a.docker, a.volumesView.ProjectName()),
			tea.Tick(3*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} }),
		)

	case views.VolumesPrunedMsg:
		if msg.Err != nil {
			a.notification = "prune: " + msg.Err.Error()
			a.notificationErr = true
		} else {
			a.notification = fmt.Sprintf("pruned %d volumes, reclaimed %s", msg.Count, views.FormatBytes(msg.Reclaimed))
			a.notificationErr = false
		}
		a.notificationExp = time.Now().Add(3 * time.Second)
		return a, tea.Batch(
			views.LoadVolumes(a.docker, a.volumesView.ProjectName()),
			tea.Tick(3*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} }),
		)

	case views.RemoveVolumeMsg:
		return a, views.RemoveVolume(a.docker, msg.Name)

	case views.PruneVolumesActionMsg:
		return a, views.PruneVolumes(a.docker)

	case views.SwitchToNetworksMsg:
		a.prevState = a.state
		a.state = viewNetworks
		a.networksView = views.NewNetworksView(msg.ProjectName).SetSize(a.width, a.height-4)
		return a, views.LoadNetworks(a.docker, msg.ProjectName)

	case views.SwitchBackFromNetworksMsg:
		a.state = viewServices
		return a, nil

	case views.NetworksLoadedMsg:
		a.networksView, _ = a.networksView.Update(msg, a.keys)
		return a, nil

	case views.SwitchToHelpMsg:
		a.prevState = a.state
		a.state = viewHelp
		a.helpView = views.NewHelpView().SetSize(a.width, a.height-4)
		return a, nil

	case views.SwitchBackFromHelpMsg:
		a.state = a.prevState
		return a, nil

	// --- Data loading ---

	case views.ProjectsLoadedMsg:
		if a.state == viewServices {
			for _, p := range msg.Projects {
				if p.Name == a.servicesView.ProjectName() {
					a.servicesView = a.servicesView.UpdateProject(p)
					break
				}
			}
		}
		var cmd tea.Cmd
		a.projectsView, cmd = a.projectsView.Update(msg, a.keys)
		return a, cmd

	case views.ProjectsErrorMsg:
		a.projectsView, _ = a.projectsView.Update(msg, a.keys)
		return a, nil

	case views.TickMsg:
		cmds := []tea.Cmd{
			views.LoadProjects(a.docker),
			views.TickCmd(a.cfg.PollInterval),
		}
		if a.state == viewServices {
			cmds = append(cmds, views.LoadStats(a.docker, a.servicesView.Project()))
		}
		return a, tea.Batch(cmds...)

	case views.StatsLoadedMsg:
		a.servicesView = a.servicesView.UpdateStats(msg.Stats)
		return a, nil

	// --- Compose ---

	case views.ComposeUpMsg:
		return a, a.executeCompose("up", msg.ProjectName, msg.ProjectPath)
	case views.ComposeDownMsg:
		return a, a.executeCompose("down", msg.ProjectName, msg.ProjectPath)
	case views.ComposeResultMsg:
		if msg.Err != nil {
			a.notification = "compose " + msg.Action + " " + msg.ProjectName + ": " + msg.Err.Error()
			a.notificationErr = true
		} else {
			a.notification = "compose " + msg.Action + " " + msg.ProjectName + ": OK"
			a.notificationErr = false
		}
		a.notificationExp = time.Now().Add(3 * time.Second)
		return a, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return notificationClearMsg{}
		})

	// --- Actions ---

	case views.ActionStartMsg:
		return a, a.executeAction("start", msg.Service)
	case views.ActionStopMsg:
		return a, a.executeAction("stop", msg.Service)
	case views.ActionRestartMsg:
		return a, a.executeAction("restart", msg.Service)

	case views.ActionResultMsg:
		if msg.Err != nil {
			a.notification = msg.Action + " " + msg.ServiceName + ": " + msg.Err.Error()
			a.notificationErr = true
		} else {
			a.notification = msg.Action + " " + msg.ServiceName + ": OK"
			a.notificationErr = false
		}
		a.notificationExp = time.Now().Add(3 * time.Second)
		return a, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return notificationClearMsg{}
		})

	case notificationClearMsg:
		if time.Now().After(a.notificationExp) {
			a.notification = ""
		}
		return a, nil

	// --- Quit ---

	case tea.KeyMsg:
		// In filter mode, don't intercept keys
		if a.isFilterMode() {
			break
		}

		// Handle :q sequence
		if a.pendingColon {
			a.pendingColon = false
			if msg.String() == "q" {
				a.cleanup()
				return a, tea.Quit
			}
			// Not q after colon — ignore and continue
			break
		}

		switch {
		case msg.String() == ":":
			a.pendingColon = true
			return a, nil
		case ui.MatchKey(msg, a.keys.Quit):
			a.cleanup()
			return a, tea.Quit
		case ui.MatchKey(msg, a.keys.ForceQuit):
			a.cleanup()
			return a, tea.Quit
		}
	}

	// Delegate to current view
	var cmd tea.Cmd
	switch a.state {
	case viewProjects:
		a.projectsView, cmd = a.projectsView.Update(msg, a.keys)
	case viewServices:
		a.servicesView, cmd = a.servicesView.Update(msg, a.keys)
	case viewDetail:
		a.detailView, cmd = a.detailView.Update(msg, a.keys)
	case viewLogs:
		a.logsView, cmd = a.logsView.Update(msg, a.keys)
	case viewCompose:
		a.composeView, cmd = a.composeView.Update(msg, a.keys)
	case viewVolumes:
		a.volumesView, cmd = a.volumesView.Update(msg, a.keys)
	case viewNetworks:
		a.networksView, cmd = a.networksView.Update(msg, a.keys)
	case viewHelp:
		a.helpView, cmd = a.helpView.Update(msg, a.keys)
	}
	return a, cmd
}

func (a App) isFilterMode() bool {
	switch a.state {
	case viewProjects:
		return a.projectsView.FilterMode()
	case viewServices:
		return a.servicesView.FilterMode()
	}
	return false
}

func (a App) View() string {
	header := ui.HeaderStyle.Width(a.width).Render(a.headerText())

	var content string
	if a.err != nil {
		content = ui.ContentStyle.
			Width(a.width).
			Height(a.height - 4).
			Render(ui.ErrorStyle.Render("Docker error: " + a.err.Error()))
	} else {
		var viewContent string
		switch a.state {
		case viewProjects:
			viewContent = a.projectsView.View()
		case viewServices:
			viewContent = a.servicesView.View()
		case viewDetail:
			viewContent = a.detailView.View()
		case viewLogs:
			viewContent = a.logsView.View()
		case viewCompose:
			viewContent = a.composeView.View()
		case viewVolumes:
			viewContent = a.volumesView.View()
		case viewNetworks:
			viewContent = a.networksView.View()
		case viewHelp:
			viewContent = a.helpView.View()
		}
		content = ui.ContentStyle.
			Width(a.width).
			Height(a.height - 4).
			Render(viewContent)
	}

	footer := ui.FooterStyle.Width(a.width).Render(a.footerText())

	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

func (a App) headerText() string {
	switch a.state {
	case viewServices:
		return "⚓ Wharf › " + a.servicesView.ProjectName()
	case viewDetail:
		return "⚓ Wharf › " + a.detailView.ProjectName() + " › " + a.detailView.ServiceName()
	case viewLogs:
		return "⚓ Wharf › " + a.servicesView.ProjectName() + " › " + a.logsView.ContainerName() + " [LOGS]"
	case viewCompose:
		return "⚓ Wharf › " + a.composeView.ProjectName() + " › " + a.composeView.FileName()
	case viewVolumes:
		return "⚓ Wharf › " + a.volumesView.ProjectName() + " › Volumes"
	case viewNetworks:
		return "⚓ Wharf › " + a.networksView.ProjectName() + " › Networks"
	case viewHelp:
		return "⚓ Wharf — Help"
	default:
		return "⚓ Wharf"
	}
}

func (a App) footerText() string {
	// Pending colon indicator
	if a.pendingColon {
		return ":"
	}

	// Notification takes priority
	if a.notification != "" && time.Now().Before(a.notificationExp) {
		if a.notificationErr {
			return ui.ErrorStyle.Render(a.notification)
		}
		return ui.RunningStyle.Render(a.notification)
	}

	// Confirmation dialog
	if a.state == viewProjects && a.projectsView.PendingDown() {
		return ui.ErrorStyle.Render("Down project \"" + a.projectsView.PendingDownName() + "\"? Press y to confirm, any key to cancel")
	}
	if a.state == viewServices && a.servicesView.PendingDown() {
		return ui.ErrorStyle.Render("Down project \"" + a.servicesView.PendingDownName() + "\"? Press y to confirm, any key to cancel")
	}

	// Volume/network confirmations
	if a.state == viewVolumes && a.volumesView.PendingRemove() {
		return ui.ErrorStyle.Render("Remove volume \"" + a.volumesView.PendingVolName() + "\"? Press y to confirm, any key to cancel")
	}
	if a.state == viewVolumes && a.volumesView.PendingPrune() {
		return ui.ErrorStyle.Render("Remove all unused volumes? Press y to confirm, any key to cancel")
	}

	// Filter mode input
	if a.state == viewProjects && a.projectsView.FilterMode() {
		return ui.FilterInputStyle.Render("/ " + a.projectsView.FilterText() + "█")
	}
	if a.state == viewServices && a.servicesView.FilterMode() {
		return ui.FilterInputStyle.Render("/ " + a.servicesView.FilterText() + "█")
	}

	switch a.state {
	case viewServices:
		return "j/k navigate • Enter details • L logs • h back • s start • S stop • r restart • u up • d down • / filter • :q quit"
	case viewDetail:
		return "j/k scroll • L logs • h back • :q quit"
	case viewCompose:
		return "j/k scroll • gg/G top/bottom • h back • :q quit"
	case viewVolumes:
		return "j/k navigate • x remove • P prune dangling • h back • :q quit"
	case viewNetworks:
		return "j/k navigate • Enter details • h back • :q quit"
	case viewLogs:
		return "j/k scroll • f follow • G bottom • h back • :q quit"
	case viewHelp:
		return "? or Esc to close"
	default:
		return "j/k navigate • Enter select • u up • d down • / filter • :q quit • ? help"
	}
}

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
		case "down":
			err = docker.ComposeDown(ctx, projectPath)
		}
		return views.ComposeResultMsg{
			Err:         err,
			Action:      action,
			ProjectName: projectName,
		}
	}
}

func (a *App) cleanup() {
	a.logsView.Close()
	if a.docker != nil {
		a.docker.Close()
	}
}

package tui

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/config"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/tui/views"
	"github.com/idesyatov/wharf/internal/ui"
	"github.com/idesyatov/wharf/internal/util"
	"github.com/idesyatov/wharf/internal/version"
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
	viewImages
	viewEvents
	viewSystem
	viewEnv
	viewHelp
	viewTop
	viewFileBrowser
)

type notificationClearMsg struct{}
type switchHostMsg struct{ host string }
type updateAvailableMsg struct {
	Version string
	URL     string
}
type composeValidateResultMsg struct {
	Err    error
	Output string
}

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
	imagesView      views.ImagesView
	eventsView      views.EventsView
	systemView      views.SystemView
	envFileView     views.EnvFileView
	helpView        views.HelpView
	topView         views.TopView
	fileBrowserView views.FileBrowserView
	events          []docker.Event
	eventsNew       int
	eventsChan      <-chan docker.Event
	docker          *docker.Client
	cfg             *config.Config
	width           int
	height          int
	keys            ui.KeyMap
	err             error
	notification    string
	notificationErr bool
	notificationExp time.Time
	cmdMode         CmdMode
	updateAvailable string
}

func NewApp(cfg *config.Config) App {
	var client *docker.Client
	var err error
	if cfg.DockerHost != "" {
		client, err = docker.NewClientWithHost(cfg.DockerHost)
	} else {
		client, err = docker.NewClient()
	}
	keys := ui.DefaultKeyMap()
	keys = ui.ApplyKeyBindings(keys, cfg.KeyBindings)

	var eventsChan <-chan docker.Event
	if client != nil {
		ch, evErr := client.SubscribeEvents(context.Background())
		if evErr == nil {
			eventsChan = ch
		}
	}

	return App{
		state:        viewProjects,
		projectsView: views.NewProjectsView(cfg.PollInterval, cfg),
		keys:         keys,
		docker:       client,
		cfg:          cfg,
		err:          err,
		eventsChan:   eventsChan,
	}
}

func (a App) Init() tea.Cmd {
	if a.err != nil || a.docker == nil {
		return checkUpdateCmd()
	}
	cmds := []tea.Cmd{
		views.LoadProjects(a.docker),
		views.TickCmd(a.cfg.PollInterval),
		checkUpdateCmd(),
	}
	if a.eventsChan != nil {
		cmds = append(cmds, a.listenEvent())
	}
	return tea.Batch(cmds...)
}

func checkUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		newVer, url := version.CheckUpdate()
		if newVer != "" {
			return updateAvailableMsg{Version: newVer, URL: url}
		}
		return nil
	}
}

func (a App) listenEvent() tea.Cmd {
	ch := a.eventsChan
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return views.EventReceivedMsg{Event: ev}
	}
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return a.handleResize(msg)
	case tea.KeyMsg:
		return a.handleGlobalKeyMsg(msg)
	case tea.MouseMsg:
		// fall through to view delegation
	default:
		model, cmd := a.handleMsg(msg)
		if model != nil {
			return model, cmd
		}
	}

	// Delegate to current view
	return a.delegateToView(msg)
}

func (a App) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	a.width = msg.Width
	a.height = msg.Height
	h := msg.Height - 7
	a.projectsView = a.projectsView.SetSize(msg.Width, h)
	a.servicesView = a.servicesView.SetSize(msg.Width, h)
	a.detailView = a.detailView.SetSize(msg.Width, h)
	a.logsView = a.logsView.SetSize(msg.Width, h)
	a.composeView = a.composeView.SetSize(msg.Width, h)
	a.volumesView = a.volumesView.SetSize(msg.Width, h)
	a.networksView = a.networksView.SetSize(msg.Width, h)
	a.imagesView = a.imagesView.SetSize(msg.Width, h)
	a.eventsView = a.eventsView.SetSize(msg.Width, h)
	a.systemView = a.systemView.SetSize(msg.Width, h)
	a.envFileView = a.envFileView.SetSize(msg.Width, h)
	a.helpView = a.helpView.SetSize(msg.Width, h)
	a.topView = a.topView.SetSize(msg.Width, h)
	a.fileBrowserView = a.fileBrowserView.SetSize(msg.Width, h)
	return a, nil
}

func (a App) handleGlobalKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.cmdMode.IsActive() {
		switch msg.Type {
		case tea.KeyEnter:
			cmd := a.cmdMode.Execute()
			return a, a.executeCommand(cmd)
		case tea.KeyEsc:
			a.cmdMode.Cancel()
			return a, nil
		default:
			a.cmdMode.HandleKey(msg)
			return a, nil
		}
	}

	if a.isFilterMode() {
		return a.delegateToView(msg)
	}

	switch {
	case msg.String() == ":":
		a.cmdMode.Enter()
		return a, nil
	case ui.MatchKey(msg, a.keys.Quit):
		a.cleanup()
		return a, tea.Quit
	case ui.MatchKey(msg, a.keys.ForceQuit):
		a.cleanup()
		return a, tea.Quit
	}

	return a.delegateToView(msg)
}

func (a App) delegateToView(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case viewImages:
		a.imagesView, cmd = a.imagesView.Update(msg, a.keys)
	case viewEvents:
		a.eventsView, cmd = a.eventsView.Update(msg, a.keys)
	case viewSystem:
		a.systemView, cmd = a.systemView.Update(msg, a.keys)
	case viewEnv:
		a.envFileView, cmd = a.envFileView.Update(msg, a.keys)
	case viewHelp:
		a.helpView, cmd = a.helpView.Update(msg, a.keys)
	case viewTop:
		a.topView, cmd = a.topView.Update(msg, a.keys)
	case viewFileBrowser:
		a.fileBrowserView, cmd = a.fileBrowserView.Update(msg, a.keys)
	}
	return a, cmd
}

func (a App) handleMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m, c := a.handleViewSwitch(msg); m != nil {
		return m, c
	}
	if m, c := a.handleDataAndTick(msg); m != nil {
		return m, c
	}
	return a.handleActions(msg)
}

func (a App) handleViewSwitch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case views.SwitchToServicesMsg:
		a.state = viewServices
		a.servicesView = views.NewServicesView(msg.Project, a.cfg.CustomCommands).SetSize(a.width, a.height-5)
		return a, nil

	case views.SwitchToProjectsMsg:
		a.state = viewProjects
		return a, nil

	case views.SwitchToDetailMsg:
		a.prevState = a.state
		a.state = viewDetail
		a.detailView = views.NewDetailView(msg.Service).SetSize(a.width, a.height-5)
		return a, views.LoadContainerDetail(a.docker, msg.Service)

	case views.SwitchBackFromDetailMsg:
		a.state = viewServices
		return a, nil

	case views.SwitchToLogsMsg:
		a.prevState = a.state
		a.state = viewLogs
		a.logsView = views.NewLogsView(msg.Container, a.cfg.MaxLogLines).SetSize(a.width, a.height-5)
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
		a.composeView = views.NewComposeView(msg.ProjectName, msg.ProjectPath).SetSize(a.width, a.height-5)
		return a, nil

	case views.SwitchBackFromComposeMsg:
		a.state = viewServices
		return a, nil

	case views.EditComposeMsg:
		editor := util.DetectEditor()
		c := exec.Command(editor, msg.FilePath)
		return a, tea.ExecProcess(c, func(err error) tea.Msg {
			return views.EditComposeDoneMsg{Err: err, FilePath: msg.FilePath}
		})

	case views.EditComposeDoneMsg:
		if msg.Err != nil {
			a.notification = "editor: " + msg.Err.Error()
			a.notificationErr = true
			a.notificationExp = time.Now().Add(3 * time.Second)
			return a, tea.Tick(3*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
		}
		a.composeView = views.NewComposeView(a.composeView.ProjectName(), a.composeView.ProjectPath()).SetSize(a.width, a.height-7)
		a.notification = "Editor closed"
		a.notificationErr = false
		a.notificationExp = time.Now().Add(2 * time.Second)
		return a, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })

	}
	return a.handleResourceMsg(msg)
}

func (a App) handleResourceMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case views.SwitchToVolumesMsg:
		a.prevState = a.state
		a.state = viewVolumes
		a.volumesView = views.NewVolumesView(msg.ProjectName).SetSize(a.width, a.height-5)
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
		a.networksView = views.NewNetworksView(msg.ProjectName).SetSize(a.width, a.height-5)
		return a, views.LoadNetworks(a.docker, msg.ProjectName)

	case views.SwitchBackFromNetworksMsg:
		a.state = viewServices
		return a, nil

	case views.NetworksLoadedMsg:
		a.networksView, _ = a.networksView.Update(msg, a.keys)
		return a, nil

	case views.BookmarkToggleMsg:
		if a.cfg != nil {
			a.cfg.ToggleBookmark(msg.ProjectName)
			_ = a.cfg.Save()
			if a.cfg.IsBookmarked(msg.ProjectName) {
				a.notification = "★ " + msg.ProjectName
			} else {
				a.notification = "☆ " + msg.ProjectName
			}
			a.notificationErr = false
			a.notificationExp = time.Now().Add(2 * time.Second)
			return a, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
		}
		return a, nil

	case views.CopyMsg:
		ui.CopyToClipboard(msg.Text)
		a.notification = "Copied: " + msg.Label
		a.notificationErr = false
		a.notificationExp = time.Now().Add(2 * time.Second)
		return a, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })

	}
	return a.handleToolMsg(msg)
}

func (a App) handleToolMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case views.SwitchToImagesMsg:
		a.prevState = a.state
		a.state = viewImages
		a.imagesView = views.NewImagesView().SetSize(a.width, a.height-5)
		return a, views.LoadImages(a.docker)

	case views.SwitchBackFromImagesMsg:
		a.state = viewProjects
		return a, nil

	case views.ImagesLoadedMsg:
		a.imagesView, _ = a.imagesView.Update(msg, a.keys)
		return a, nil

	case views.PullImageActionMsg:
		a.notification = "pulling " + msg.Ref + "..."
		a.notificationErr = false
		a.notificationExp = time.Now().Add(60 * time.Second)
		return a, views.PullImage(a.docker, msg.Ref)

	case views.ImagePulledMsg:
		if msg.Err != nil {
			a.notification = "pull " + msg.ImageRef + ": " + msg.Err.Error()
			a.notificationErr = true
		} else {
			a.notification = "pulled " + msg.ImageRef
			a.notificationErr = false
		}
		a.notificationExp = time.Now().Add(3 * time.Second)
		return a, tea.Batch(
			views.LoadImages(a.docker),
			tea.Tick(3*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} }),
		)

	case views.PruneImagesActionMsg:
		return a, views.PruneImages(a.docker)

	case views.ImagesPrunedMsg:
		if msg.Err != nil {
			a.notification = "prune images: " + msg.Err.Error()
			a.notificationErr = true
		} else {
			a.notification = fmt.Sprintf("pruned %d images, reclaimed %s", msg.Count, views.FormatBytes(msg.Reclaimed))
			a.notificationErr = false
		}
		a.notificationExp = time.Now().Add(3 * time.Second)
		return a, tea.Batch(
			views.LoadImages(a.docker),
			tea.Tick(3*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} }),
		)

	}
	return a.handleBuildAndEventsMsg(msg)
}

func (a App) handleBuildAndEventsMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case views.BuildMsg:
		composePath := ""
		if msg.ComposePath != "" {
			composePath = msg.ComposePath
		}
		_ = composePath // ComposeBuild finds file itself
		svcName := msg.Service
		if svcName == "" {
			svcName = "all"
		}
		args := []string{"compose"}
		if msg.ProjectPath != "" {
			cf, err := docker.FindComposeFile(msg.ProjectPath)
			if err == nil {
				args = append(args, "-f", cf)
			}
		}
		args = append(args, "build")
		if msg.Service != "" {
			args = append(args, msg.Service)
		}
		c := exec.Command("docker", args...)
		c.Dir = msg.ProjectPath
		return a, tea.ExecProcess(c, func(err error) tea.Msg {
			return views.BuildDoneMsg{Err: err, Service: svcName}
		})

	case views.BuildDoneMsg:
		if msg.Err != nil {
			a.notification = "build " + msg.Service + ": " + msg.Err.Error()
			a.notificationErr = true
		} else {
			a.notification = "build " + msg.Service + ": OK"
			a.notificationErr = false
		}
		a.notificationExp = time.Now().Add(3 * time.Second)
		return a, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return notificationClearMsg{}
		})

	// --- Events ---

	case views.EventReceivedMsg:
		a.events = append(a.events, msg.Event)
		if len(a.events) > 50 {
			a.events = a.events[len(a.events)-50:]
		}
		if a.state != viewEvents {
			a.eventsNew++
		}
		return a, a.listenEvent()

	case views.SwitchToEventsMsg:
		a.prevState = a.state
		a.state = viewEvents
		a.eventsNew = 0
		a.eventsView = views.NewEventsView(a.events).SetSize(a.width, a.height-6)
		return a, nil

	case views.SwitchBackFromEventsMsg:
		a.state = a.prevState
		return a, nil

	}
	return a.handleSystemAndMiscMsg(msg)
}

func (a App) handleSystemAndMiscMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case views.SwitchToSystemMsg:
		a.prevState = a.state
		a.state = viewSystem
		a.systemView = views.NewSystemView().SetSize(a.width, a.height-6)
		return a, views.LoadSystemDf(a.docker)

	case views.SwitchBackFromSystemMsg:
		a.state = a.prevState
		return a, nil

	case views.SystemDfLoadedMsg:
		a.systemView, _ = a.systemView.Update(msg, a.keys)
		return a, nil

	case views.SystemPruneMsg:
		a.notification = "pruning all unused resources..."
		a.notificationErr = false
		a.notificationExp = time.Now().Add(60 * time.Second)
		return a, views.SystemPrune()

	case views.SystemPruneDoneMsg:
		if msg.Err != nil {
			a.notification = "system prune: " + msg.Err.Error()
			a.notificationErr = true
		} else {
			a.notification = "system prune: OK"
			a.notificationErr = false
		}
		a.notificationExp = time.Now().Add(3 * time.Second)
		return a, tea.Batch(
			views.LoadSystemDf(a.docker),
			tea.Tick(3*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} }),
		)

	}
	return a.handleFileAndNavMsg(msg)
}

func (a App) handleFileAndNavMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case views.OpenBrowserMsg:
		err := util.OpenBrowser(msg.URL)
		if err != nil {
			a.notification = "open: " + err.Error()
			a.notificationErr = true
		} else {
			a.notification = "Opening " + msg.URL
			a.notificationErr = false
		}
		a.notificationExp = time.Now().Add(3 * time.Second)
		return a, tea.Tick(3*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })

	// --- Logs save ---

	case views.SaveLogsMsg:
		path := msg.Path
		if path == "" {
			home, _ := os.UserHomeDir()
			dir := filepath.Join(home, "wharf-logs")
			_ = os.MkdirAll(dir, 0755)
			path = filepath.Join(dir, a.logsView.ContainerName()+"-"+time.Now().Format("2006-01-02-150405")+".log")
		}
		lines := a.logsView.Lines()
		err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
		if err != nil {
			a.notification = "save: " + err.Error()
			a.notificationErr = true
		} else {
			a.notification = fmt.Sprintf("Saved %d lines → %s", len(lines), path)
			a.notificationErr = false
		}
		a.notificationExp = time.Now().Add(3 * time.Second)
		return a, tea.Tick(3*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })

	// --- Env file ---

	case views.SwitchToEnvMsg:
		a.prevState = a.state
		a.state = viewEnv
		a.envFileView = views.NewEnvFileView(msg.ProjectName, msg.ProjectPath).SetSize(a.width, a.height-5)
		if a.envFileView.FileName() == "" {
			a.state = a.prevState
			a.notification = "No .env file found"
			a.notificationErr = false
			a.notificationExp = time.Now().Add(2 * time.Second)
			return a, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
		}
		return a, nil

	case views.SwitchBackFromEnvMsg:
		a.state = a.prevState
		return a, nil

	case views.SwitchToTopProjectMsg:
		a.prevState = a.state
		a.state = viewTop
		a.topView = views.NewTopViewProject(msg.Project).SetSize(a.width, a.height-7)
		return a, views.LoadTopStats(a.docker, msg.Project)

	case views.SwitchToTopContainerMsg:
		a.prevState = a.state
		a.state = viewTop
		a.topView = views.NewTopViewContainer(msg.ContainerID, msg.ContainerName, msg.Image).SetSize(a.width, a.height-7)
		return a, views.LoadTopContainerStats(a.docker, msg.ContainerID)

	case views.SwitchBackFromTopMsg:
		a.state = a.prevState
		return a, nil

	case views.TopStatsLoadedMsg:
		a.topView = a.topView.UpdateStats(msg.Stats)
		return a, nil

	case views.SwitchToFileBrowserMsg:
		a.prevState = a.state
		a.state = viewFileBrowser
		a.fileBrowserView = views.NewFileBrowserView(msg.ContainerID, msg.ContainerName).SetSize(a.width, a.height-7)
		return a, views.LoadDirectoryListing(a.docker, msg.ContainerID, "/")

	case views.SwitchBackFromFileBrowserMsg:
		a.state = a.prevState
		return a, nil

	case views.FileBrowserListMsg:
		a.fileBrowserView, _ = a.fileBrowserView.Update(msg, a.keys)
		return a, nil

	case views.FileBrowserReadMsg:
		a.fileBrowserView, _ = a.fileBrowserView.Update(msg, a.keys)
		return a, nil

	case views.FileBrowserNavigateMsg:
		if msg.IsFile {
			return a, views.LoadFileContent(a.docker, msg.ContainerID, msg.Path)
		}
		return a, views.LoadDirectoryListing(a.docker, msg.ContainerID, msg.Path)

	case views.SwitchToHelpMsg:
		if a.state != viewHelp {
			a.prevState = a.state
		}
		a.state = viewHelp
		a.helpView = views.NewHelpView().SetSize(a.width, a.height-5)
		return a, nil

	case views.SwitchBackFromHelpMsg:
		a.state = a.prevState
		return a, nil
	}
	return nil, nil
}

func (a App) handleDataAndTick(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
			cmds = append(cmds, views.LoadHealth(a.docker, a.servicesView.Project()))
		}
		if a.state == viewTop {
			if a.topView.IsProjectMode() {
				cmds = append(cmds, views.LoadTopStats(a.docker, a.topView.Project()))
			} else {
				cmds = append(cmds, views.LoadTopContainerStats(a.docker, a.topView.ContainerID()))
			}
		}
		return a, tea.Batch(cmds...)

	case views.StatsLoadedMsg:
		a.servicesView = a.servicesView.UpdateStats(msg.Stats)
		return a, nil

	// --- Compose ---

	case views.BatchActionMsg:
		a.notification = fmt.Sprintf("compose %s: %d projects...", msg.Action, len(msg.Projects))
		a.notificationErr = false
		a.notificationExp = time.Now().Add(30 * time.Second)
		return a, a.executeBatchCompose(msg.Action, msg.Projects)

	case views.HealthLoadedMsg:
		a.servicesView = a.servicesView.UpdateHealth(msg.Health)
		return a, nil

	case views.ComposeUpMsg:
		return a, a.executeCompose("up", msg.ProjectName, msg.ProjectPath)
	case views.ComposeStopMsg:
		return a, a.executeCompose("stop", msg.ProjectName, msg.ProjectPath)
	case views.ComposeDownMsg:
		return a, a.executeCompose("down", msg.ProjectName, msg.ProjectPath)
	case views.ComposeRestartMsg:
		return a, a.executeCompose("restart", msg.ProjectName, msg.ProjectPath)
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

	}
	return nil, nil
}

func (a App) handleActions(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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

	case updateAvailableMsg:
		a.updateAvailable = msg.Version
		return a, nil

	case composeValidateResultMsg:
		if msg.Err != nil {
			errText := strings.TrimSpace(msg.Output)
			if errText == "" {
				errText = msg.Err.Error()
			}
			a.notification = "✕ " + errText
			a.notificationErr = true
		} else {
			a.notification = "✓ Compose file is valid"
			a.notificationErr = false
		}
		a.notificationExp = time.Now().Add(5 * time.Second)
		return a, tea.Tick(5*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })

	case switchHostMsg:
		var newClient *docker.Client
		var connErr error
		if msg.host != "" {
			newClient, connErr = docker.NewClientWithHost(msg.host)
		} else {
			newClient, connErr = docker.NewClient()
		}
		if connErr != nil {
			a.notification = "Connection failed: " + connErr.Error()
			a.notificationErr = true
			a.notificationExp = time.Now().Add(5 * time.Second)
			return a, tea.Tick(5*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
		}
		if a.docker != nil {
			a.docker.Close()
		}
		a.docker = newClient
		a.cfg.DockerHost = msg.host
		if ch, evErr := newClient.SubscribeEvents(context.Background()); evErr == nil {
			a.eventsChan = ch
		}
		hostName := "local"
		if msg.host != "" {
			hostName = msg.host
		}
		a.notification = "Connected: " + hostName
		a.notificationErr = false
		a.notificationExp = time.Now().Add(3 * time.Second)
		a.state = viewProjects
		return a, tea.Batch(
			views.LoadProjects(a.docker),
			tea.Tick(3*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} }),
		)

	case notificationClearMsg:
		if time.Now().After(a.notificationExp) {
			a.notification = ""
		}
		return a, nil

	// --- Custom commands ---

	case views.CustomCommandMsg:
		c := exec.Command("sh", "-c", msg.Command)
		return a, tea.ExecProcess(c, func(err error) tea.Msg {
			return views.CustomCommandDoneMsg{Err: err, Name: msg.Name}
		})

	case views.CustomCommandDoneMsg:
		if msg.Err != nil {
			a.notification = msg.Name + ": " + msg.Err.Error()
			a.notificationErr = true
		} else {
			a.notification = msg.Name + ": done"
			a.notificationErr = false
		}
		a.notificationExp = time.Now().Add(2 * time.Second)
		return a, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })

	// --- Exec ---

	case views.ExecMsg:
		shell := msg.Shell
		if shell == "" {
			shell = a.docker.DetectShell(context.Background(), msg.ContainerID)
		}
		banner := fmt.Sprintf(
			"echo '─────────────────────────────────────────' && "+
				"echo '  ⚓ Wharf — Container Shell' && "+
				"echo '  Container: %s' && "+
				"echo '  Image:     %s' && "+
				"echo '  Shell:     %s' && "+
				"echo '  Exit:      type exit or Ctrl+D' && "+
				"echo '─────────────────────────────────────────' && "+
				"exec %s",
			msg.ContainerName, msg.Image, shell, shell,
		)
		c := exec.Command("docker", "exec", "-it", msg.ContainerID, "sh", "-c", banner)
		return a, tea.ExecProcess(c, func(err error) tea.Msg {
			return views.ExecDoneMsg{Err: err}
		})

	case views.ExecDoneMsg:
		if msg.Err != nil {
			a.notification = "exec: " + msg.Err.Error()
			a.notificationErr = true
			a.notificationExp = time.Now().Add(3 * time.Second)
			return a, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return notificationClearMsg{}
			})
		}
		return a, nil

	}
	return nil, nil
}

func (a App) isFilterMode() bool {
	switch a.state {
	case viewProjects:
		return a.projectsView.FilterMode()
	case viewServices:
		return a.servicesView.FilterMode()
	case viewLogs:
		return a.logsView.SearchMode()
	case viewHelp:
		return a.helpView.SearchMode()
	}
	return false
}

func (a App) View() string {
	infoBar := a.renderInfoBar()
	breadcrumbs := a.renderBreadcrumbs()
	content := a.renderContent()
	menuBar := a.renderMenuBar()
	statusLine := a.renderStatusLine()

	return lipgloss.JoinVertical(lipgloss.Left, infoBar, breadcrumbs, content, menuBar, statusLine)
}

func (a App) renderInfoBar() string {
	logo := ui.LogoStyle.Render("⚓ Wharf")

	dockerStatus := ui.RunningStyle.Render("●")
	if a.err != nil {
		dockerStatus = ui.ErrorStyle.Render("●")
	}

	host := "local"
	if a.cfg.DockerHost != "" {
		if u, err := url.Parse(a.cfg.DockerHost); err == nil && u.Host != "" {
			host = u.Host
		} else {
			host = a.cfg.DockerHost
		}
	} else if dh := os.Getenv("DOCKER_HOST"); dh != "" {
		if u, err := url.Parse(dh); err == nil && u.Host != "" {
			host = u.Host
		} else {
			host = dh
		}
	}

	ver := ui.MutedStyle.Render(" " + version.String())
	update := ""
	if a.updateAvailable != "" {
		update = " " + ui.PartialStyle.Render("↑"+a.updateAvailable)
	}

	right := ui.InfoBarStyle.Render("Docker: ") + dockerStatus + ui.MutedStyle.Render(" "+host) + ver + update

	gap := a.width - lipgloss.Width(logo) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	pad := lipgloss.NewStyle().Width(gap).Render("")

	return logo + pad + right
}

func (a App) renderBreadcrumbs() string {
	var crumb string
	switch a.state {
	case viewProjects:
		crumb = a.projectsView.Breadcrumb()
	case viewServices:
		crumb = a.servicesView.Breadcrumb()
	case viewDetail:
		crumb = a.detailView.Breadcrumb()
	case viewLogs:
		crumb = a.logsView.Breadcrumb()
	case viewCompose:
		crumb = a.composeView.Breadcrumb()
	case viewVolumes:
		crumb = a.volumesView.Breadcrumb()
	case viewNetworks:
		crumb = a.networksView.Breadcrumb()
	case viewImages:
		crumb = a.imagesView.Breadcrumb()
	case viewEvents:
		crumb = a.eventsView.Breadcrumb()
	case viewSystem:
		crumb = a.systemView.Breadcrumb()
	case viewEnv:
		crumb = a.envFileView.Breadcrumb()
	case viewHelp:
		crumb = a.helpView.Breadcrumb()
	case viewTop:
		crumb = a.topView.Breadcrumb()
	case viewFileBrowser:
		crumb = a.fileBrowserView.Breadcrumb()
	}

	style := lipgloss.NewStyle().
		Foreground(ui.ColorMuted).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(ui.ColorBorder).
		Width(a.width).
		Padding(0, 1)

	return style.Render(crumb)
}

func joinMenuItems(items ...string) string {
	return strings.Join(items, "  ")
}

func (a App) renderMenuBar() string {
	var actionsLine, toolsLine string

	switch a.state {
	case viewProjects:
		if a.projectsView.HasSelected() {
			actionsLine = joinMenuItems(
				fmt.Sprintf("%d selected", a.projectsView.SelectedCount()),
				ui.FormatMenuItem("u", "p all"),
				ui.FormatMenuItem("d", " stop all"),
				ui.FormatMenuItem("X", " down all"),
				ui.FormatMenuItem("R", " restart all"),
				ui.FormatMenuItem("Esc", " clear"),
			)
			toolsLine = joinMenuItems(
				ui.FormatMenuItem("Space", " toggle"),
			)
		} else {
			actionsLine = joinMenuItems(
				ui.FormatMenuItem("u", " compose up"),
				ui.FormatMenuItem("d", " compose stop"),
				ui.FormatMenuItem("X", " compose down"),
				ui.FormatMenuItem("R", " compose restart"),
			)
			toolsLine = joinMenuItems(
				ui.FormatMenuItem("t", "op"),
				ui.FormatMenuItem("i", "mages"),
				ui.FormatMenuItem("E", "vents"),
				ui.FormatMenuItem("D", "isk usage"),
				ui.FormatMenuItem("*", "mark"),
				ui.FormatMenuItem("/", "filter"),
				ui.FormatMenuItem("?", "help"),
			)
		}
	case viewServices:
		actionsLine = joinMenuItems(
			ui.FormatMenuItem("s", "tart"),
			ui.FormatMenuItem("S", "top"),
			ui.FormatMenuItem("r", "estart"),
			ui.FormatMenuItem("e", "xec"),
			ui.FormatMenuItem("L", "ogs"),
		)
		toolsItems := []string{
			ui.FormatMenuItem("t", "op"),
			ui.FormatMenuItem("F", "iles"),
			ui.FormatMenuItem("b", "uild"),
			ui.FormatMenuItem("c", "ompose"),
			ui.FormatMenuItem("v", "ol"),
			ui.FormatMenuItem("n", "et"),
			ui.FormatMenuItem("/", "filter"),
			ui.FormatMenuItem("?", "help"),
		}
		for _, cc := range a.servicesView.CustomCommands() {
			toolsItems = append(toolsItems, ui.FormatMenuItem(cc.Key, " "+cc.Name))
		}
		toolsLine = joinMenuItems(toolsItems...)
	case viewDetail:
		actionsLine = joinMenuItems(
			ui.FormatMenuItem("L", "ogs"),
			ui.FormatMenuItem("e", "xec"),
			ui.FormatMenuItem("F", "iles"),
			ui.FormatMenuItem("y", "copy"),
			ui.FormatMenuItem("Y", "copy+"),
		)
	case viewCompose:
		actionsLine = joinMenuItems(
			ui.FormatMenuItem("e", "dit"),
		)
	case viewLogs:
		actionsLine = joinMenuItems(
			ui.FormatMenuItem("f", "ollow"),
			ui.FormatMenuItem("w", "save"),
		)
	case viewVolumes:
		actionsLine = joinMenuItems(
			ui.FormatMenuItem("x", "remove"),
			ui.FormatMenuItem("P", "rune"),
		)
	case viewImages:
		actionsLine = joinMenuItems(
			ui.FormatMenuItem("p", "ull"),
			ui.FormatMenuItem("P", "rune"),
		)
	case viewEvents:
		actionsLine = ""
	case viewSystem:
		actionsLine = joinMenuItems(
			ui.FormatMenuItem("P", "rune all"),
		)
	}

	content := actionsLine
	if toolsLine != "" {
		content = actionsLine + "\n" + toolsLine
	}

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(ui.ColorBorder).
		Width(a.width).
		Padding(0, 1)

	return style.Render(content)
}

func (a App) renderContent() string {
	contentHeight := a.height - 7 // info 1 + breadcrumbs 2 (border) + menu 3 (border + 2 lines) + status 1
	if contentHeight < 1 {
		contentHeight = 1
	}

	var viewContent string
	if a.err != nil {
		viewContent = ui.ErrorStyle.Render("Docker error: " + a.err.Error())
	} else {
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
		case viewImages:
			viewContent = a.imagesView.View()
		case viewEvents:
			viewContent = a.eventsView.View()
		case viewSystem:
			viewContent = a.systemView.View()
		case viewEnv:
			viewContent = a.envFileView.View()
		case viewHelp:
			viewContent = a.helpView.View()
		case viewTop:
			viewContent = a.topView.View()
		case viewFileBrowser:
			viewContent = a.fileBrowserView.View()
		}
	}

	return ui.ContentStyle.
		Width(a.width).
		Height(contentHeight).
		Render(viewContent)
}

func (a App) renderStatusLine() string {
	if a.cmdMode.IsActive() {
		return ui.CommandStyle.Render(":" + a.cmdMode.Input() + "█")
	}
	if a.notification != "" && time.Now().Before(a.notificationExp) {
		if a.notificationErr {
			return ui.ErrorStyle.Render(a.notification)
		}
		return ui.RunningStyle.Render(a.notification)
	}
	if s := a.renderConfirmDialog(); s != "" {
		return s
	}
	if a.state == viewServices && !a.servicesView.HasStats() {
		return ui.RunningStyle.Render("Loading stats...")
	}
	if a.state == viewTop && !a.topView.HasStats() {
		return ui.RunningStyle.Render("Loading stats...")
	}
	return a.renderFilterStatus()
}

func (a App) renderConfirmDialog() string {
	switch a.state {
	case viewProjects:
		if a.projectsView.PendingDown() {
			return ui.ErrorStyle.Render("Down (REMOVE containers) \"" + a.projectsView.PendingDownName() + "\"? [y/N]")
		}
	case viewServices:
		if a.servicesView.PendingDown() {
			return ui.ErrorStyle.Render("Down (REMOVE containers) \"" + a.servicesView.PendingDownName() + "\"? [y/N]")
		}
	case viewVolumes:
		if a.volumesView.PendingRemove() {
			return ui.ErrorStyle.Render("Remove volume \"" + a.volumesView.PendingVolName() + "\"? [y/N]")
		}
		if a.volumesView.PendingPrune() {
			return ui.ErrorStyle.Render("Remove all unused volumes? [y/N]")
		}
	case viewImages:
		if a.imagesView.PendingPrune() {
			return ui.ErrorStyle.Render("Remove all unused images? [y/N]")
		}
	case viewSystem:
		if a.systemView.PendingPrune() {
			return ui.ErrorStyle.Render("Prune all unused resources (images, containers, volumes, build cache)? [y/N]")
		}
	}
	return ""
}

func (a App) renderFilterStatus() string {
	switch a.state {
	case viewProjects:
		if a.projectsView.FilterMode() {
			return ui.FilterInputStyle.Render("/ " + a.projectsView.FilterText() + "█")
		}
	case viewServices:
		if a.servicesView.FilterMode() {
			return ui.FilterInputStyle.Render("/ " + a.servicesView.FilterText() + "█")
		}
	case viewLogs:
		if a.logsView.SearchMode() {
			return ui.FilterInputStyle.Render("/ " + a.logsView.SearchText() + "█")
		}
		if info := a.logsView.SearchInfo(); info != "" {
			return ui.MutedStyle.Render("/" + a.logsView.SearchText() + "  " + info)
		}
	case viewHelp:
		if a.helpView.SearchMode() {
			return ui.FilterInputStyle.Render("/ " + a.helpView.SearchText() + "█")
		}
		if info := a.helpView.SearchInfo(); info != "" {
			return ui.MutedStyle.Render("/" + a.helpView.SearchText() + "  " + info)
		}
	}
	return ""
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
	case "host":
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
		}
		return func() tea.Msg { return switchHostMsg{host: newHost} }
	case "theme":
		if len(parts) < 2 {
			a.notification = "Usage: :theme dark|light"
			a.notificationErr = true
			a.notificationExp = time.Now().Add(3 * time.Second)
			return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
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
	case "version":
		a.notification = "wharf " + version.Full()
		a.notificationErr = false
		a.notificationExp = time.Now().Add(3 * time.Second)
		return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
	case "save":
		if a.state == viewLogs {
			path := ""
			if len(parts) > 1 {
				path = parts[1]
			}
			return func() tea.Msg { return views.SaveLogsMsg{Path: path} }
		}
		a.notification = "save: only available in Logs view"
		a.notificationErr = true
		a.notificationExp = time.Now().Add(2 * time.Second)
		return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
	case "edit":
		if a.state == viewCompose && a.composeView.FilePath() != "" {
			editor := util.DetectEditor()
			fp := a.composeView.FilePath()
			c := exec.Command(editor, fp)
			return tea.ExecProcess(c, func(err error) tea.Msg {
				return views.EditComposeDoneMsg{Err: err, FilePath: fp}
			})
		}
		a.notification = "edit: only available in Compose view"
		a.notificationErr = true
		a.notificationExp = time.Now().Add(2 * time.Second)
		return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
	case "go":
		if len(parts) < 2 {
			a.notification = "Usage: :go <project-name>"
			a.notificationErr = true
			a.notificationExp = time.Now().Add(2 * time.Second)
			return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
		}
		query := strings.ToLower(parts[1])
		for _, p := range a.projectsView.Projects() {
			if strings.Contains(strings.ToLower(p.Name), query) {
				a.state = viewServices
				a.servicesView = views.NewServicesView(p, a.cfg.CustomCommands).SetSize(a.width, a.height-7)
				a.notification = "→ " + p.Name
				a.notificationErr = false
				a.notificationExp = time.Now().Add(2 * time.Second)
				return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
			}
		}
		a.notification = "Project not found: " + parts[1]
		a.notificationErr = true
		a.notificationExp = time.Now().Add(2 * time.Second)
		return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
	case "exec":
		if len(parts) < 2 {
			a.notification = "Usage: :exec <container-name>"
			a.notificationErr = true
			a.notificationExp = time.Now().Add(2 * time.Second)
			return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
		}
		ct := a.findContainerByName(parts[1])
		if ct == nil {
			a.notification = "Container not found: " + parts[1]
			a.notificationErr = true
			a.notificationExp = time.Now().Add(2 * time.Second)
			return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
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
	case "validate":
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
			a.notification = "validate: navigate to a project or use :validate <name>"
			a.notificationErr = true
			a.notificationExp = time.Now().Add(3 * time.Second)
			return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
		}
		return a.validateCompose(projectPath)
	default:
		a.notification = "Unknown command: " + cmd
		a.notificationErr = true
		a.notificationExp = time.Now().Add(2 * time.Second)
		return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })
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

func (a *App) cleanup() {
	a.logsView.Close()
	if a.docker != nil {
		a.docker.Close()
	}
}

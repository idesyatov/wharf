package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idesyatov/wharf/internal/config"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/tui/views"
	"github.com/idesyatov/wharf/internal/ui"
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
	viewHosts
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
	hostsView       views.HostsView
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

func (a *App) cleanup() {
	a.logsView.Close()
	if a.docker != nil {
		a.docker.Close()
	}
}

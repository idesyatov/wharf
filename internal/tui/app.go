package tui

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/config"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/tui/views"
	"github.com/idesyatov/wharf/internal/ui"
	"github.com/idesyatov/wharf/internal/util"
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
	imagesView      views.ImagesView
	eventsView      views.EventsView
	systemView      views.SystemView
	helpView        views.HelpView
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
	pendingColon    bool
}

func NewApp(cfg *config.Config) App {
	client, err := docker.NewClient()
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
		return nil
	}
	cmds := []tea.Cmd{
		views.LoadProjects(a.docker),
		views.TickCmd(a.cfg.PollInterval),
	}
	if a.eventsChan != nil {
		cmds = append(cmds, a.listenEvent())
	}
	return tea.Batch(cmds...)
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
		a.width = msg.Width
		a.height = msg.Height
		a.projectsView = a.projectsView.SetSize(msg.Width, msg.Height-6)
		a.servicesView = a.servicesView.SetSize(msg.Width, msg.Height-6)
		a.detailView = a.detailView.SetSize(msg.Width, msg.Height-6)
		a.logsView = a.logsView.SetSize(msg.Width, msg.Height-6)
		a.composeView = a.composeView.SetSize(msg.Width, msg.Height-6)
		a.volumesView = a.volumesView.SetSize(msg.Width, msg.Height-6)
		a.networksView = a.networksView.SetSize(msg.Width, msg.Height-6)
		a.imagesView = a.imagesView.SetSize(msg.Width, msg.Height-6)
		a.eventsView = a.eventsView.SetSize(msg.Width, msg.Height-6)
		a.systemView = a.systemView.SetSize(msg.Width, msg.Height-6)
		a.helpView = a.helpView.SetSize(msg.Width, msg.Height-6)
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

	case views.SwitchToImagesMsg:
		a.prevState = a.state
		a.state = viewImages
		a.imagesView = views.NewImagesView().SetSize(a.width, a.height-4)
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

	// --- Build ---

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

	// --- System ---

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

	// --- Browser ---

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

	// --- Exec ---

	case views.ExecMsg:
		shell := msg.Shell
		if shell == "" {
			shell = a.docker.DetectShell(context.Background(), msg.ContainerID)
		}
		c := exec.Command("docker", "exec", "-it", msg.ContainerID, shell)
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
	case viewImages:
		a.imagesView, cmd = a.imagesView.Update(msg, a.keys)
	case viewEvents:
		a.eventsView, cmd = a.eventsView.Update(msg, a.keys)
	case viewSystem:
		a.systemView, cmd = a.systemView.Update(msg, a.keys)
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
	case viewLogs:
		return a.logsView.SearchMode()
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
	if dh := os.Getenv("DOCKER_HOST"); dh != "" {
		if u, err := url.Parse(dh); err == nil {
			host = u.Host
			if host == "" {
				host = dh
			}
		} else {
			host = dh
		}
	}

	right := ui.InfoBarStyle.Render("Docker: ") + dockerStatus + ui.MutedStyle.Render(" "+host)

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
		crumb = ""
	case viewServices:
		crumb = "› " + a.servicesView.ProjectName()
	case viewDetail:
		crumb = "› " + a.detailView.ProjectName() + " › " + a.detailView.ServiceName()
	case viewLogs:
		crumb = "› " + a.servicesView.ProjectName() + " › " + a.logsView.ContainerName() + " [LOGS]"
	case viewCompose:
		crumb = "› " + a.composeView.ProjectName() + " › " + a.composeView.FileName()
	case viewVolumes:
		crumb = "› " + a.volumesView.ProjectName() + " › Volumes"
	case viewNetworks:
		crumb = "› " + a.networksView.ProjectName() + " › Networks"
	case viewImages:
		crumb = "› Images"
	case viewEvents:
		crumb = "› Events"
	case viewSystem:
		crumb = "› System"
	case viewHelp:
		crumb = "Help"
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

func (a App) renderMenuBar() string {
	var items []string

	switch a.state {
	case viewProjects:
		items = []string{
			ui.FormatMenuItem("i", "mages"),
			ui.FormatMenuItem("E", "vents"),
			ui.FormatMenuItem("D", "isk usage"),
			ui.FormatMenuItem("u", "p"),
			ui.FormatMenuItem("d", "own"),
			ui.FormatMenuItem("*", "mark"),
			ui.FormatMenuItem("/", "filter"),
			ui.FormatMenuItem("?", "help"),
		}
	case viewServices:
		items = []string{
			ui.FormatMenuItem("s", "tart"),
			ui.FormatMenuItem("S", "top"),
			ui.FormatMenuItem("r", "estart"),
			ui.FormatMenuItem("e", "xec"),
			ui.FormatMenuItem("b", "uild"),
			ui.FormatMenuItem("u", "p"),
			ui.FormatMenuItem("d", "own"),
			ui.FormatMenuItem("L", "ogs"),
			ui.FormatMenuItem("c", "ompose"),
			ui.FormatMenuItem("v", "ol"),
			ui.FormatMenuItem("n", "et"),
			ui.FormatMenuItem("/", "filter"),
		}
	case viewDetail:
		items = []string{
			ui.FormatMenuItem("L", "ogs"),
			ui.FormatMenuItem("e", "xec"),
			ui.FormatMenuItem("y", "copy"),
			ui.FormatMenuItem("Y", "copy+"),
		}
	case viewLogs:
		items = []string{
			ui.FormatMenuItem("f", "ollow"),
		}
	case viewVolumes:
		items = []string{
			ui.FormatMenuItem("x", "remove"),
			ui.FormatMenuItem("P", "rune"),
		}
	case viewImages:
		items = []string{
			ui.FormatMenuItem("p", "ull"),
			ui.FormatMenuItem("P", "rune"),
		}
	case viewEvents:
		items = []string{}
	case viewSystem:
		items = []string{
			ui.FormatMenuItem("P", "rune all"),
		}
	}

	joined := ""
	for i, item := range items {
		if i > 0 {
			joined += "  "
		}
		joined += item
	}

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(ui.ColorBorder).
		Width(a.width).
		Padding(0, 1)

	return style.Render(joined)
}

func (a App) renderContent() string {
	contentHeight := a.height - 6 // info 1 + breadcrumbs 2 (border) + menu 2 (border) + status 1
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
		case viewHelp:
			viewContent = a.helpView.View()
		}
	}

	return ui.ContentStyle.
		Width(a.width).
		Height(contentHeight).
		Render(viewContent)
}

func (a App) renderStatusLine() string {
	if a.pendingColon {
		return ui.CommandStyle.Render(":")
	}

	if a.notification != "" && time.Now().Before(a.notificationExp) {
		if a.notificationErr {
			return ui.ErrorStyle.Render(a.notification)
		}
		return ui.RunningStyle.Render(a.notification)
	}

	// Confirmation dialogs
	if a.state == viewProjects && a.projectsView.PendingDown() {
		return ui.ErrorStyle.Render("Down project \"" + a.projectsView.PendingDownName() + "\"? [y/N]")
	}
	if a.state == viewServices && a.servicesView.PendingDown() {
		return ui.ErrorStyle.Render("Down project \"" + a.servicesView.PendingDownName() + "\"? [y/N]")
	}
	if a.state == viewVolumes && a.volumesView.PendingRemove() {
		return ui.ErrorStyle.Render("Remove volume \"" + a.volumesView.PendingVolName() + "\"? [y/N]")
	}
	if a.state == viewVolumes && a.volumesView.PendingPrune() {
		return ui.ErrorStyle.Render("Remove all unused volumes? [y/N]")
	}
	if a.state == viewImages && a.imagesView.PendingPrune() {
		return ui.ErrorStyle.Render("Remove all unused images? [y/N]")
	}
	if a.state == viewSystem && a.systemView.PendingPrune() {
		return ui.ErrorStyle.Render("Prune all unused resources (images, containers, volumes, build cache)? [y/N]")
	}

	// Filter mode
	if a.state == viewProjects && a.projectsView.FilterMode() {
		return ui.FilterInputStyle.Render("/ " + a.projectsView.FilterText() + "█")
	}
	if a.state == viewServices && a.servicesView.FilterMode() {
		return ui.FilterInputStyle.Render("/ " + a.servicesView.FilterText() + "█")
	}

	// Log search mode
	if a.state == viewLogs && a.logsView.SearchMode() {
		return ui.FilterInputStyle.Render("/ " + a.logsView.SearchText() + "█")
	}
	// Log search active (not in input mode)
	if a.state == viewLogs {
		info := a.logsView.SearchInfo()
		if info != "" {
			return ui.MutedStyle.Render("/" + a.logsView.SearchText() + "  " + info)
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

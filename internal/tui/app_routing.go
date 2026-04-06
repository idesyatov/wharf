package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/tui/views"
	"github.com/idesyatov/wharf/internal/ui"
	"github.com/idesyatov/wharf/internal/util"
)

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
	a.hostsView = a.hostsView.SetSize(msg.Width, h)
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
		a.cmdMode.SetHostNames(a.cfg.HostNames())
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
	case viewHosts:
		a.hostsView, cmd = a.hostsView.Update(msg, a.keys)
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

	// --- Hosts ---

	case views.SwitchToHostsMsg:
		a.prevState = a.state
		a.state = viewHosts
		a.hostsView = views.NewHostsView(a.cfg.Hosts, a.cfg.DockerHost).SetSize(a.width, a.height-7)
		return a, nil

	case views.SwitchBackFromHostsMsg:
		a.state = a.prevState
		return a, nil

	case views.HostSelectedMsg:
		return a, func() tea.Msg { return switchHostMsg{host: msg.URL} }

	case views.HostDeleteMsg:
		a.cfg.RemoveHost(msg.Name)
		_ = a.cfg.Save()
		a.hostsView = views.NewHostsView(a.cfg.Hosts, a.cfg.DockerHost).SetSize(a.width, a.height-7)
		a.notification = "Host removed: " + msg.Name
		a.notificationErr = false
		a.notificationExp = time.Now().Add(2 * time.Second)
		return a, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })

	case views.HostAddMsg:
		a.cfg.AddHost(msg.Name, msg.URL)
		_ = a.cfg.Save()
		a.hostsView = views.NewHostsView(a.cfg.Hosts, a.cfg.DockerHost).SetSize(a.width, a.height-7)
		a.notification = "Host added: " + msg.Name
		a.notificationErr = false
		a.notificationExp = time.Now().Add(2 * time.Second)
		return a, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return notificationClearMsg{} })

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

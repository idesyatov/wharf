package tui

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/ui"
	"github.com/idesyatov/wharf/internal/version"
)

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
	case viewHosts:
		crumb = a.hostsView.Breadcrumb()
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
		actionsLine, toolsLine = a.menuProjects()
	case viewServices:
		actionsLine, toolsLine = a.menuServices()
	case viewDetail:
		actionsLine = a.menuDetail()
	case viewCompose:
		actionsLine = a.menuCompose()
	case viewLogs:
		actionsLine = a.menuLogs()
	case viewVolumes:
		actionsLine = a.menuVolumes()
	case viewImages:
		actionsLine = a.menuImages()
	case viewEvents:
		actionsLine = ""
	case viewHosts:
		actionsLine = a.menuHosts()
	case viewSystem:
		actionsLine = a.menuSystem()
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

func (a App) menuProjects() (string, string) {
	if a.projectsView.HasSelected() {
		return joinMenuItems(
				fmt.Sprintf("%d selected", a.projectsView.SelectedCount()),
				ui.FormatMenuItem("u", "p all"),
				ui.FormatMenuItem("d", " stop all"),
				ui.FormatMenuItem("X", " down all"),
				ui.FormatMenuItem("R", " restart all"),
				ui.FormatMenuItem("Esc", " clear"),
			), joinMenuItems(
				ui.FormatMenuItem("Space", " toggle"),
			)
	}
	return joinMenuItems(
			ui.FormatMenuItem("u", " compose up"),
			ui.FormatMenuItem("d", " compose stop"),
			ui.FormatMenuItem("X", " compose down"),
			ui.FormatMenuItem("R", " compose restart"),
		), joinMenuItems(
			ui.FormatMenuItem("t", "op"),
			ui.FormatMenuItem("i", "mages"),
			ui.FormatMenuItem("E", "vents"),
			ui.FormatMenuItem("D", "isk usage"),
			ui.FormatMenuItem("H", "osts"),
			ui.FormatMenuItem("*", "mark"),
			ui.FormatMenuItem("/", "filter"),
			ui.FormatMenuItem("?", "help"),
		)
}

func (a App) menuServices() (string, string) {
	actionsLine := joinMenuItems(
		ui.FormatMenuItem("s", "tart"),
		ui.FormatMenuItem("S", "top"),
		ui.FormatMenuItem("r", "estart"),
		ui.FormatMenuItem("x", "remove"),
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
	return actionsLine, joinMenuItems(toolsItems...)
}

func (a App) menuDetail() string {
	return joinMenuItems(
		ui.FormatMenuItem("L", "ogs"),
		ui.FormatMenuItem("e", "xec"),
		ui.FormatMenuItem("F", "iles"),
		ui.FormatMenuItem("y", "copy"),
		ui.FormatMenuItem("Y", "copy+"),
	)
}

func (a App) menuCompose() string {
	return joinMenuItems(
		ui.FormatMenuItem("e", "dit"),
	)
}

func (a App) menuLogs() string {
	return joinMenuItems(
		ui.FormatMenuItem("f", "ollow"),
		ui.FormatMenuItem("w", "save"),
	)
}

func (a App) menuVolumes() string {
	return joinMenuItems(
		ui.FormatMenuItem("x", "remove"),
		ui.FormatMenuItem("P", "rune"),
	)
}

func (a App) menuImages() string {
	return joinMenuItems(
		ui.FormatMenuItem("p", "ull"),
		ui.FormatMenuItem("P", "rune"),
	)
}

func (a App) menuHosts() string {
	return joinMenuItems(
		ui.FormatMenuItem("Enter", " connect"),
		ui.FormatMenuItem("a", "dd"),
		ui.FormatMenuItem("d", "elete"),
	)
}

func (a App) menuSystem() string {
	return joinMenuItems(
		ui.FormatMenuItem("P", "rune all"),
	)
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
		case viewHosts:
			viewContent = a.hostsView.View()
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
		if a.servicesView.PendingRemove() {
			return ui.ErrorStyle.Render("Remove container \"" + a.servicesView.PendingRemoveName() + "\"? [y/N]")
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
	case viewHosts:
		if a.hostsView.PendingDelete() {
			return ui.ErrorStyle.Render("Remove host \"" + a.hostsView.PendingDeleteName() + "\"? [y/N]")
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

package views

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idesyatov/wharf/internal/docker"
	"github.com/idesyatov/wharf/internal/ui"
)

type SwitchToImagesMsg struct{}
type SwitchBackFromImagesMsg struct{}
type ImagesLoadedMsg struct{ Images []docker.Image }
type ImagePulledMsg struct {
	Err      error
	ImageRef string
}
type ImagesPrunedMsg struct {
	Err       error
	Count     int
	Reclaimed uint64
}

type LoadImageLayersMsg struct {
	ImageID string
}
type ImageLayersLoadedMsg struct {
	Layers []docker.ImageLayer
	Ref    string
}
type ImageSaveMsg struct {
	Ref string
}
type ImageSaveDoneMsg struct {
	Err  error
	Path string
	Ref  string
}

type ImagesView struct {
	images        []docker.Image
	cursor        int
	width, height int
	pendingG      bool
	pendingPrune  bool
	err           error
	layersMode    bool
	layers        []docker.ImageLayer
	layerScroll   int
	selectedImage string
}

func NewImagesView() ImagesView {
	return ImagesView{}
}

func (v ImagesView) SetSize(w, h int) ImagesView {
	v.width = w
	v.height = h
	return v
}

func (v ImagesView) Breadcrumb() string {
	if v.layersMode {
		return "› Images › " + v.selectedImage
	}
	return "› Images"
}
func (v ImagesView) PendingPrune() bool { return v.pendingPrune }
func (v ImagesView) LayersMode() bool   { return v.layersMode }

func LoadImages(client *docker.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return ImagesLoadedMsg{}
		}
		images, err := client.ListImages(context.Background())
		if err != nil {
			return ImagesLoadedMsg{}
		}
		return ImagesLoadedMsg{Images: images}
	}
}

func PullImage(client *docker.Client, ref string) tea.Cmd {
	return func() tea.Msg {
		err := client.PullImage(context.Background(), ref)
		return ImagePulledMsg{Err: err, ImageRef: ref}
	}
}

func PruneImages(client *docker.Client) tea.Cmd {
	return func() tea.Msg {
		count, reclaimed, err := client.PruneImages(context.Background())
		return ImagesPrunedMsg{Err: err, Count: count, Reclaimed: reclaimed}
	}
}

func LoadImageLayers(client *docker.Client, imageID string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return ImageLayersLoadedMsg{}
		}
		layers, err := client.ImageHistory(context.Background(), imageID)
		if err != nil {
			return ImageLayersLoadedMsg{}
		}
		return ImageLayersLoadedMsg{Layers: layers, Ref: imageID}
	}
}

func SaveImage(client *docker.Client, ref string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return ImageSaveDoneMsg{Err: fmt.Errorf("no docker client")}
		}
		safeName := strings.ReplaceAll(ref, "/", "_")
		safeName = strings.ReplaceAll(safeName, ":", "_")
		path := safeName + ".tar"
		absPath, _ := filepath.Abs(path)
		err := client.ImageSave(context.Background(), ref, path)
		return ImageSaveDoneMsg{Err: err, Path: absPath, Ref: ref}
	}
}

func (v ImagesView) selectedRef() string {
	if len(v.images) == 0 || v.cursor >= len(v.images) {
		return ""
	}
	img := v.images[v.cursor]
	if len(img.RepoTags) > 0 {
		return img.RepoTags[0]
	}
	return img.ID
}

func (v ImagesView) Update(msg tea.Msg, keys ui.KeyMap) (ImagesView, tea.Cmd) {
	switch msg := msg.(type) {
	case ImagesLoadedMsg:
		v.images = msg.Images
		if v.cursor >= len(v.images) && len(v.images) > 0 {
			v.cursor = len(v.images) - 1
		}
		return v, nil

	case ImageLayersLoadedMsg:
		v.layers = msg.Layers
		v.layersMode = true
		v.layerScroll = 0
		return v, nil

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			row := msg.Y - 4
			if row >= 0 && row < len(v.images) {
				v.cursor = row
			}
		}
		if msg.Button == tea.MouseButtonWheelDown {
			if v.cursor < len(v.images)-1 {
				v.cursor++
			}
		}
		if msg.Button == tea.MouseButtonWheelUp {
			if v.cursor > 0 {
				v.cursor--
			}
		}
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg, keys)
	}
	return v, nil
}

func (v ImagesView) handleKeyMsg(msg tea.KeyMsg, keys ui.KeyMap) (ImagesView, tea.Cmd) {
	if v.layersMode {
		return v.handleLayersKeyMsg(msg, keys)
	}

	if v.pendingPrune {
		v.pendingPrune = false
		if ui.MatchKey(msg, keys.Confirm) {
			return v, func() tea.Msg { return PruneImagesActionMsg{} }
		}
		return v, nil
	}

	if v.pendingG {
		v.pendingG = false
		if msg.String() == "g" {
			v.cursor = 0
			return v, nil
		}
	}

	switch {
	case ui.MatchKey(msg, keys.Down):
		if v.cursor < len(v.images)-1 {
			v.cursor++
		}
	case ui.MatchKey(msg, keys.Up):
		if v.cursor > 0 {
			v.cursor--
		}
	case ui.MatchKey(msg, keys.Bottom):
		if len(v.images) > 0 {
			v.cursor = len(v.images) - 1
		}
	case msg.String() == "g":
		v.pendingG = true
	case ui.MatchKey(msg, keys.Right), msg.Type == tea.KeyEnter:
		if len(v.images) > 0 && v.cursor < len(v.images) {
			img := v.images[v.cursor]
			v.selectedImage = v.selectedRef()
			id := img.ID
			return v, func() tea.Msg {
				return LoadImageLayersMsg{ImageID: id}
			}
		}
	case msg.String() == "w":
		ref := v.selectedRef()
		if ref != "" {
			return v, func() tea.Msg {
				return ImageSaveMsg{Ref: ref}
			}
		}
	case ui.MatchKey(msg, keys.Pull):
		ref := v.selectedRef()
		if ref != "" {
			return v, func() tea.Msg { return PullImageActionMsg{Ref: ref} }
		}
	case ui.MatchKey(msg, keys.Prune):
		v.pendingPrune = true
	case ui.MatchKey(msg, keys.Left):
		return v, func() tea.Msg { return SwitchBackFromImagesMsg{} }
	}
	return v, nil
}

func (v ImagesView) handleLayersKeyMsg(msg tea.KeyMsg, keys ui.KeyMap) (ImagesView, tea.Cmd) {
	switch {
	case ui.MatchKey(msg, keys.Left), msg.Type == tea.KeyEsc:
		v.layersMode = false
		return v, nil
	case ui.MatchKey(msg, keys.Down):
		v.layerScroll++
	case ui.MatchKey(msg, keys.Up):
		if v.layerScroll > 0 {
			v.layerScroll--
		}
	case msg.String() == "w":
		ref := v.selectedImage
		if ref != "" {
			return v, func() tea.Msg {
				return ImageSaveMsg{Ref: ref}
			}
		}
	}
	totalLines := len(v.layers)
	visible := v.height - 6
	if visible < 1 {
		visible = 1
	}
	maxScroll := totalLines - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if v.layerScroll > maxScroll {
		v.layerScroll = maxScroll
	}
	return v, nil
}

type PullImageActionMsg struct{ Ref string }
type PruneImagesActionMsg struct{}

func (v ImagesView) View() string {
	if v.layersMode {
		return v.renderLayers()
	}
	if v.err != nil {
		return ui.ErrorStyle.Render(fmt.Sprintf("Error: %v", v.err))
	}
	if len(v.images) == 0 {
		return ui.MutedStyle.Render("No images found")
	}

	colRepo := 30
	colTag := 15
	colSize := 10

	header := ui.HeaderRowStyle.Render(
		fmt.Sprintf("%-*s %-*s %-*s %s", colRepo, "REPOSITORY", colTag, "TAG", colSize, "SIZE", "CREATED"),
	)

	var rows []string
	rows = append(rows, header)

	for i, img := range v.images {
		repo, tag := parseRepoTag(img)
		size := FormatBytes(uint64(img.Size))
		created := timeAgo(img.Created)

		row := fmt.Sprintf("%-*s %-*s %-*s %s",
			colRepo, truncate(repo, colRepo),
			colTag, truncate(tag, colTag),
			colSize, size,
			created,
		)

		if i == v.cursor {
			row = renderSelectedRow(row, v.width-2)
		}
		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (v ImagesView) renderLayers() string {
	title := ui.MutedStyle.Render("  Image: ") +
		lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render(v.selectedImage)

	header := ui.HeaderRowStyle.Render(
		fmt.Sprintf("  %-10s  %s", "SIZE", "COMMAND"))

	var rows []string
	rows = append(rows, title, "")
	rows = append(rows, header)

	visible := v.height - 6
	if visible < 1 {
		visible = 1
	}
	start := v.layerScroll
	end := start + visible
	if end > len(v.layers) {
		end = len(v.layers)
	}

	for i := start; i < end; i++ {
		layer := v.layers[i]
		size := ""
		if layer.Size > 0 {
			size = FormatBytes(uint64(layer.Size))
		}

		cmd := layer.CreatedBy
		cmd = strings.TrimPrefix(cmd, "/bin/sh -c ")
		cmd = strings.TrimPrefix(cmd, "#(nop) ")
		maxCmd := v.width - 20
		if maxCmd > 0 && len(cmd) > maxCmd {
			cmd = cmd[:maxCmd-3] + "..."
		}

		row := fmt.Sprintf("  %-10s  %s", size, cmd)
		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func parseRepoTag(img docker.Image) (string, string) {
	if len(img.RepoTags) == 0 {
		return "<none>", "<none>"
	}
	rt := img.RepoTags[0]
	for i := len(rt) - 1; i >= 0; i-- {
		if rt[i] == ':' {
			return rt[:i], rt[i+1:]
		}
	}
	return rt, "latest"
}

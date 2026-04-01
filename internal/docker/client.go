package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	typesvolume "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

type Client struct {
	cli *client.Client
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker connect: %w", err)
	}
	return &Client{cli: cli}, nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}

func (c *Client) ListContainers(ctx context.Context) ([]Container, error) {
	raw, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	containers := make([]Container, 0, len(raw))
	for _, r := range raw {
		containers = append(containers, fromAPIContainer(r))
	}

	return containers, nil
}

func (c *Client) StartContainer(ctx context.Context, id string) error {
	if err := c.cli.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container %s: %w", id, err)
	}
	return nil
}

func (c *Client) StopContainer(ctx context.Context, id string) error {
	if err := c.cli.ContainerStop(ctx, id, container.StopOptions{}); err != nil {
		return fmt.Errorf("stop container %s: %w", id, err)
	}
	return nil
}

func (c *Client) RestartContainer(ctx context.Context, id string) error {
	if err := c.cli.ContainerRestart(ctx, id, container.StopOptions{}); err != nil {
		return fmt.Errorf("restart container %s: %w", id, err)
	}
	return nil
}

func (c *Client) StartService(ctx context.Context, svc Service) error {
	var firstErr error
	for _, ct := range svc.Containers {
		if err := c.StartContainer(ctx, ct.ID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (c *Client) StopService(ctx context.Context, svc Service) error {
	var firstErr error
	for _, ct := range svc.Containers {
		if err := c.StopContainer(ctx, ct.ID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (c *Client) RestartService(ctx context.Context, svc Service) error {
	var firstErr error
	for _, ct := range svc.Containers {
		if err := c.RestartContainer(ctx, ct.ID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (c *Client) ContainerLogs(ctx context.Context, containerID string, tail int) (io.ReadCloser, error) {
	return c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       fmt.Sprintf("%d", tail),
		Timestamps: true,
	})
}

func (c *Client) ContainerStats(ctx context.Context, id string) (Stats, error) {
	resp, err := c.cli.ContainerStats(ctx, id, false)
	if err != nil {
		return Stats{}, fmt.Errorf("stats %s: %w", id, err)
	}
	defer resp.Body.Close()

	var v struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
			OnlineCPUs     uint64 `json:"online_cpus"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64            `json:"usage"`
			Limit uint64            `json:"limit"`
			Stats map[string]uint64 `json:"stats"`
		} `json:"memory_stats"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return Stats{}, fmt.Errorf("decode stats %s: %w", id, err)
	}

	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(v.CPUStats.SystemCPUUsage - v.PreCPUStats.SystemCPUUsage)
	cpuPercent := 0.0
	if sysDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / sysDelta) * float64(v.CPUStats.OnlineCPUs) * 100.0
	}

	memUsage := v.MemoryStats.Usage
	if cache, ok := v.MemoryStats.Stats["cache"]; ok {
		memUsage -= cache
	}

	return Stats{
		CPUPercent: cpuPercent,
		MemUsage:   memUsage,
		MemLimit:   v.MemoryStats.Limit,
	}, nil
}

func (c *Client) InspectContainer(ctx context.Context, id string) (ContainerDetail, error) {
	info, err := c.cli.ContainerInspect(ctx, id)
	if err != nil {
		return ContainerDetail{}, fmt.Errorf("inspect container %s: %w", id, err)
	}

	var volumes []VolumeMount
	for _, m := range info.Mounts {
		volumes = append(volumes, VolumeMount{
			Source:      m.Source,
			Destination: m.Destination,
			Mode:        m.Mode,
		})
	}

	var networks []string
	if info.NetworkSettings != nil {
		for name := range info.NetworkSettings.Networks {
			networks = append(networks, name)
		}
		sort.Strings(networks)
	}

	created, _ := time.Parse(time.RFC3339Nano, info.Created)

	var ports []Port
	if info.NetworkSettings != nil {
		for portProto, bindings := range info.NetworkSettings.Ports {
			for _, b := range bindings {
				var hostPort uint16
				if b.HostPort != "" {
					fmt.Sscanf(b.HostPort, "%d", &hostPort)
				}
				ports = append(ports, Port{
					HostIP:   b.HostIP,
					HostPort: hostPort,
					ContPort: uint16(portProto.Int()),
					Proto:    portProto.Proto(),
				})
			}
		}
	}

	restartPolicy := ""
	if info.HostConfig != nil {
		restartPolicy = string(info.HostConfig.RestartPolicy.Name)
	}

	networkMode := ""
	if info.HostConfig != nil {
		networkMode = string(info.HostConfig.NetworkMode)
	}

	health := HealthCheck{Status: "none"}
	if info.State.Health != nil {
		health.Status = info.State.Health.Status
		health.FailingStreak = info.State.Health.FailingStreak
		for _, entry := range info.State.Health.Log {
			health.Log = append(health.Log, HealthLog{
				Start:    entry.Start,
				End:      entry.End,
				ExitCode: entry.ExitCode,
				Output:   strings.TrimSpace(entry.Output),
			})
		}
	}

	return ContainerDetail{
		Container: Container{
			ID:      info.ID[:12],
			Name:    strings.TrimPrefix(info.Name, "/"),
			Image:   info.Config.Image,
			Status:  info.State.Status,
			State:   info.State.Status,
			Ports:   ports,
			Labels:  info.Config.Labels,
			Created: created,
		},
		Env:           info.Config.Env,
		Volumes:       volumes,
		RestartPolicy: restartPolicy,
		NetworkMode:   networkMode,
		Networks:      networks,
		Cmd:           info.Config.Cmd,
		Entrypoint:    info.Config.Entrypoint,
		Health:        health,
	}, nil
}

func (c *Client) ContainerHealthStatus(ctx context.Context, id string) string {
	info, err := c.cli.ContainerInspect(ctx, id)
	if err != nil || info.State.Health == nil {
		return "none"
	}
	return info.State.Health.Status
}

func (c *Client) ListImages(ctx context.Context) ([]Image, error) {
	raw, err := c.cli.ImageList(ctx, types.ImageListOptions{All: false})
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}
	var images []Image
	for _, r := range raw {
		images = append(images, Image{
			ID:       r.ID[7:19], // strip sha256: prefix, take 12 chars
			RepoTags: r.RepoTags,
			Size:     r.Size,
			Created:  time.Unix(r.Created, 0),
		})
	}
	sort.Slice(images, func(i, j int) bool {
		if len(images[i].RepoTags) == 0 || len(images[j].RepoTags) == 0 {
			return len(images[i].RepoTags) > len(images[j].RepoTags)
		}
		return images[i].RepoTags[0] < images[j].RepoTags[0]
	})
	return images, nil
}

func (c *Client) PullImage(ctx context.Context, ref string) error {
	reader, err := c.cli.ImagePull(ctx, ref, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pull %s: %w", ref, err)
	}
	defer reader.Close()
	// Drain reader to complete pull
	buf := make([]byte, 4096)
	for {
		_, err := reader.Read(buf)
		if err != nil {
			break
		}
	}
	return nil
}

func (c *Client) PruneImages(ctx context.Context) (int, uint64, error) {
	report, err := c.cli.ImagesPrune(ctx, filters.Args{})
	if err != nil {
		return 0, 0, fmt.Errorf("prune images: %w", err)
	}
	return len(report.ImagesDeleted), report.SpaceReclaimed, nil
}

func (c *Client) ListVolumes(ctx context.Context) ([]Volume, error) {
	resp, err := c.cli.VolumeList(ctx, typesvolume.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list volumes: %w", err)
	}
	var volumes []Volume
	for _, v := range resp.Volumes {
		proj := ""
		if v.Labels != nil {
			proj = v.Labels["com.docker.compose.project"]
		}
		volumes = append(volumes, Volume{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			Labels:     v.Labels,
			Project:    proj,
		})
	}
	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].Name < volumes[j].Name
	})
	return volumes, nil
}

func (c *Client) RemoveVolume(ctx context.Context, name string) error {
	if err := c.cli.VolumeRemove(ctx, name, false); err != nil {
		return fmt.Errorf("remove volume %s: %w", name, err)
	}
	return nil
}

func (c *Client) PruneVolumes(ctx context.Context) (int, uint64, error) {
	report, err := c.cli.VolumesPrune(ctx, filters.Args{})
	if err != nil {
		return 0, 0, fmt.Errorf("prune volumes: %w", err)
	}
	return len(report.VolumesDeleted), report.SpaceReclaimed, nil
}

func (c *Client) ListNetworks(ctx context.Context) ([]Network, error) {
	raw, err := c.cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}
	var networks []Network
	for _, n := range raw {
		subnet := ""
		gateway := ""
		if len(n.IPAM.Config) > 0 {
			subnet = n.IPAM.Config[0].Subnet
			gateway = n.IPAM.Config[0].Gateway
		}
		var containers []string
		for _, ep := range n.Containers {
			containers = append(containers, ep.Name)
		}
		sort.Strings(containers)
		proj := ""
		if n.Labels != nil {
			proj = n.Labels["com.docker.compose.project"]
		}
		networks = append(networks, Network{
			Name:       n.Name,
			ID:         n.ID[:12],
			Driver:     n.Driver,
			Subnet:     subnet,
			Gateway:    gateway,
			Containers: containers,
			Labels:     n.Labels,
			Project:    proj,
		})
	}
	sort.Slice(networks, func(i, j int) bool {
		return networks[i].Name < networks[j].Name
	})
	return networks, nil
}

func (c *Client) SubscribeEvents(ctx context.Context) (<-chan Event, error) {
	msgs, errs := c.cli.Events(ctx, types.EventsOptions{})
	ch := make(chan Event, 64)
	go func() {
		defer close(ch)
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				actor := ""
				if msg.Actor.Attributes != nil {
					if name, ok := msg.Actor.Attributes["name"]; ok {
						actor = name
					} else if img, ok := msg.Actor.Attributes["image"]; ok {
						actor = img
					}
				}
				if actor == "" {
					actor = msg.Actor.ID
					if len(actor) > 12 {
						actor = actor[:12]
					}
				}
				ch <- Event{
					Time:   time.Unix(msg.Time, msg.TimeNano),
					Type:   string(msg.Type),
					Action: string(msg.Action),
					Actor:  actor,
				}
			case _, ok := <-errs:
				if !ok {
					return
				}
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

func (c *Client) SystemDiskUsage(ctx context.Context) (SystemDf, error) {
	du, err := c.cli.DiskUsage(ctx, types.DiskUsageOptions{})
	if err != nil {
		return SystemDf{}, fmt.Errorf("disk usage: %w", err)
	}

	var imagesSize int64
	for _, img := range du.Images {
		imagesSize += img.Size
	}

	var containersSize int64
	for _, ct := range du.Containers {
		containersSize += ct.SizeRw
	}

	var volumesSize int64
	for _, vol := range du.Volumes {
		if vol.UsageData.Size > 0 {
			volumesSize += vol.UsageData.Size
		}
	}

	var buildCacheSize int64
	if du.BuildCache != nil {
		for _, bc := range du.BuildCache {
			buildCacheSize += bc.Size
		}
	}

	return SystemDf{
		ImagesCount:     len(du.Images),
		ImagesSize:      imagesSize,
		ContainersCount: len(du.Containers),
		ContainersSize:  containersSize,
		VolumesCount:    len(du.Volumes),
		VolumesSize:     volumesSize,
		BuildCacheSize:  buildCacheSize,
	}, nil
}

func (c *Client) DetectShell(ctx context.Context, containerID string) string {
	execCfg, err := c.cli.ContainerExecCreate(ctx, containerID, types.ExecConfig{
		Cmd:          []string{"which", "bash"},
		AttachStdout: true,
	})
	if err == nil {
		resp, err := c.cli.ContainerExecAttach(ctx, execCfg.ID, types.ExecStartCheck{})
		if err == nil {
			resp.Close()
			inspect, err := c.cli.ContainerExecInspect(ctx, execCfg.ID)
			if err == nil && inspect.ExitCode == 0 {
				return "bash"
			}
		}
	}
	return "sh"
}

func fromAPIContainer(r types.Container) Container {
	name := ""
	if len(r.Names) > 0 {
		name = strings.TrimPrefix(r.Names[0], "/")
	}

	ports := make([]Port, 0, len(r.Ports))
	for _, p := range r.Ports {
		ports = append(ports, Port{
			HostIP:   p.IP,
			HostPort: p.PublicPort,
			ContPort: p.PrivatePort,
			Proto:    p.Type,
		})
	}

	return Container{
		ID:      r.ID[:12],
		Name:    name,
		Image:   r.Image,
		Status:  r.State,
		State:   r.Status,
		Ports:   ports,
		Labels:  r.Labels,
		Created: time.Unix(r.Created, 0),
	}
}

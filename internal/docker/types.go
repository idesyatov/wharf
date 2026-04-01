// Package docker provides a wrapper around the Docker SDK
// for managing containers, images, volumes, networks, and compose projects.
package docker

import "time"

// Container represents a Docker container with basic metadata.
type Container struct {
	ID      string
	Name    string
	Image   string
	Status  string
	State   string
	Ports   []Port
	Labels  map[string]string
	Created time.Time
}

// Port represents a port mapping for a container.
type Port struct {
	HostIP   string
	HostPort uint16
	ContPort uint16
	Proto    string
}

// ServiceStatus represents the aggregate status of a compose service.
type ServiceStatus string

const (
	StatusRunning ServiceStatus = "running"
	StatusPartial ServiceStatus = "partial"
	StatusStopped ServiceStatus = "stopped"
)

// Service represents a Docker Compose service with its containers.
type Service struct {
	Name       string
	Project    string
	Image      string
	Status     ServiceStatus
	Containers []Container
}

// Project represents a Docker Compose project discovered through container labels.
type Project struct {
	Name     string
	Path     string
	Status   ServiceStatus
	Services []Service
}

// ContainerDetail contains extended container information from docker inspect.
type ContainerDetail struct {
	Container
	Env           []string
	Volumes       []VolumeMount
	RestartPolicy string
	NetworkMode   string
	Networks      []string
	Cmd           []string
	Entrypoint    []string
	Health        HealthCheck
}

// HealthCheck contains container health check status and log.
type HealthCheck struct {
	Status        string
	FailingStreak int
	Log           []HealthLog
}

// HealthLog represents a single health check execution result.
type HealthLog struct {
	Start    time.Time
	End      time.Time
	ExitCode int
	Output   string
}

// VolumeMount represents a bind mount or volume in a container.
type VolumeMount struct {
	Source      string
	Destination string
	Mode        string
}

// Stats contains CPU, memory, and network usage statistics for a container.
type Stats struct {
	CPUPercent float64
	MemUsage   uint64
	MemLimit   uint64
	NetRx      uint64
	NetTx      uint64
}

// Image represents a Docker image.
type Image struct {
	ID       string
	RepoTags []string
	Size     int64
	Created  time.Time
}

// Event represents a Docker engine event.
type Event struct {
	Time   time.Time
	Type   string
	Action string
	Actor  string
}

// SystemDf contains Docker system disk usage information.
type SystemDf struct {
	ImagesCount     int
	ImagesSize      int64
	ContainersCount int
	ContainersSize  int64
	VolumesCount    int
	VolumesSize     int64
	BuildCacheSize  int64
}

// Volume represents a Docker volume.
type Volume struct {
	Name       string
	Driver     string
	Mountpoint string
	Labels     map[string]string
	Project    string
}

// Network represents a Docker network.
type Network struct {
	Name       string
	ID         string
	Driver     string
	Subnet     string
	Gateway    string
	Containers []string
	Labels     map[string]string
	Project    string
}

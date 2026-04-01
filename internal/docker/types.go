package docker

import "time"

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

type Port struct {
	HostIP   string
	HostPort uint16
	ContPort uint16
	Proto    string
}

type ServiceStatus string

const (
	StatusRunning ServiceStatus = "running"
	StatusPartial ServiceStatus = "partial"
	StatusStopped ServiceStatus = "stopped"
)

type Service struct {
	Name       string
	Project    string
	Image      string
	Status     ServiceStatus
	Containers []Container
}

type Project struct {
	Name     string
	Path     string
	Status   ServiceStatus
	Services []Service
}

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

type HealthCheck struct {
	Status        string
	FailingStreak int
	Log           []HealthLog
}

type HealthLog struct {
	Start    time.Time
	End      time.Time
	ExitCode int
	Output   string
}

type VolumeMount struct {
	Source      string
	Destination string
	Mode        string
}

type Stats struct {
	CPUPercent float64
	MemUsage   uint64
	MemLimit   uint64
}

type Image struct {
	ID       string
	RepoTags []string
	Size     int64
	Created  time.Time
}

type Event struct {
	Time   time.Time
	Type   string
	Action string
	Actor  string
}

type SystemDf struct {
	ImagesCount     int
	ImagesSize      int64
	ContainersCount int
	ContainersSize  int64
	VolumesCount    int
	VolumesSize     int64
	BuildCacheSize  int64
}

type Volume struct {
	Name       string
	Driver     string
	Mountpoint string
	Labels     map[string]string
	Project    string
}

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

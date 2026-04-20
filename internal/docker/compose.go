package docker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

const (
	labelProject    = "com.docker.compose.project"
	labelService    = "com.docker.compose.service"
	labelWorkingDir = "com.docker.compose.project.working_dir"
	labelOneoff     = "com.docker.compose.oneoff"
)

func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	containers, err := c.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	projectMap := make(map[string]*Project)
	serviceMap := make(map[string]map[string]*Service) // project -> service -> *Service

	for _, ct := range containers {
		projName := ct.Labels[labelProject]
		if projName == "" {
			continue
		}
		if ct.Labels[labelOneoff] == "True" {
			continue
		}

		svcName := ct.Labels[labelService]

		_, ok := projectMap[projName]
		if !ok {
			proj := &Project{
				Name: projName,
				Path: ct.Labels[labelWorkingDir],
			}
			projectMap[projName] = proj
			serviceMap[projName] = make(map[string]*Service)
		}

		svc, ok := serviceMap[projName][svcName]
		if !ok {
			svc = &Service{
				Name:    svcName,
				Project: projName,
				Image:   ct.Image,
			}
			serviceMap[projName][svcName] = svc
		}
		svc.Containers = append(svc.Containers, ct)
	}

	projects := make([]Project, 0, len(projectMap))
	for name, proj := range projectMap {
		services := make([]Service, 0, len(serviceMap[name]))
		for _, svc := range serviceMap[name] {
			svc.Status = calcServiceStatus(svc.Containers)
			services = append(services, *svc)
		}
		sort.Slice(services, func(i, j int) bool {
			return services[i].Name < services[j].Name
		})
		proj.Services = services
		proj.Status = calcProjectStatus(services)
		projects = append(projects, *proj)
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects, nil
}

var composeFiles = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yml",
	"docker-compose.yaml",
}

// FindComposeFile searches for a compose file in the given directory.
func FindComposeFile(projectPath string) (string, error) {
	for _, name := range composeFiles {
		p := filepath.Join(projectPath, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no compose file found in %s", projectPath)
}

// composeExec runs `docker compose [--profile p...] <action> [extraArgs...]` in projectPath.
func composeExec(ctx context.Context, projectPath, action string, profiles []string, extraArgs ...string) error {
	composePath, err := FindComposeFile(projectPath)
	if err != nil {
		return err
	}
	args := []string{"compose", "-f", composePath}
	for _, p := range profiles {
		args = append(args, "--profile", p)
	}
	args = append(args, action)
	args = append(args, extraArgs...)
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = projectPath
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("compose %s: %s", action, stderr.String())
		}
		return fmt.Errorf("compose %s: %w", action, err)
	}
	return nil
}

// ComposeUp starts a compose project with docker compose up -d, optionally filtered by profiles.
func ComposeUp(ctx context.Context, projectPath string, profiles ...string) error {
	return composeExec(ctx, projectPath, "up", profiles, "-d")
}

// ComposeStop stops a compose project without removing containers, optionally filtered by profiles.
func ComposeStop(ctx context.Context, projectPath string, profiles ...string) error {
	return composeExec(ctx, projectPath, "stop", profiles)
}

// ComposeDown stops and removes containers for a compose project, optionally filtered by profiles.
func ComposeDown(ctx context.Context, projectPath string, profiles ...string) error {
	return composeExec(ctx, projectPath, "down", profiles)
}

// ComposeRestart restarts services in a compose project, optionally filtered by profiles.
func ComposeRestart(ctx context.Context, projectPath string, profiles ...string) error {
	return composeExec(ctx, projectPath, "restart", profiles)
}

// ComposeBuild builds images for a compose project or a specific service.
func ComposeBuild(ctx context.Context, projectPath string, service string) error {
	composePath, err := FindComposeFile(projectPath)
	if err != nil {
		return err
	}
	args := []string{"compose", "-f", composePath, "build"}
	if service != "" {
		args = append(args, service)
	}
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = projectPath
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("compose build: %s", stderr.String())
		}
		return fmt.Errorf("compose build: %w", err)
	}
	return nil
}

// ComposeProfiles represents profiles extracted from a compose file.
type ComposeProfiles struct {
	AllProfiles     []string
	ServiceProfiles map[string][]string
}

// ParseProfiles reads a compose file and extracts profile information.
func ParseProfiles(projectPath string) (*ComposeProfiles, error) {
	composePath, err := FindComposeFile(projectPath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(composePath)
	if err != nil {
		return nil, fmt.Errorf("read compose file: %w", err)
	}

	var raw struct {
		Services map[string]struct {
			Profiles []string `yaml:"profiles"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse compose file: %w", err)
	}

	profileSet := make(map[string]bool)
	serviceProfiles := make(map[string][]string)
	for name, svc := range raw.Services {
		if len(svc.Profiles) > 0 {
			serviceProfiles[name] = svc.Profiles
			for _, p := range svc.Profiles {
				profileSet[p] = true
			}
		}
	}

	var allProfiles []string
	for p := range profileSet {
		allProfiles = append(allProfiles, p)
	}
	sort.Strings(allProfiles)

	return &ComposeProfiles{
		AllProfiles:     allProfiles,
		ServiceProfiles: serviceProfiles,
	}, nil
}

func calcServiceStatus(containers []Container) ServiceStatus {
	running := 0
	for _, c := range containers {
		if c.Status == "running" {
			running++
		}
	}
	switch {
	case running == len(containers):
		return StatusRunning
	case running > 0:
		return StatusPartial
	default:
		return StatusStopped
	}
}

func calcProjectStatus(services []Service) ServiceStatus {
	running := 0
	for _, s := range services {
		if s.Status == StatusRunning {
			running++
		}
	}
	switch {
	case running == len(services):
		return StatusRunning
	case running > 0:
		return StatusPartial
	default:
		return StatusStopped
	}
}

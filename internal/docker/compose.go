package docker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

		proj, ok := projectMap[projName]
		if !ok {
			proj = &Project{
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

func findComposeFile(projectPath string) (string, error) {
	for _, name := range composeFiles {
		p := filepath.Join(projectPath, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no compose file found in %s", projectPath)
}

func ComposeUp(ctx context.Context, projectPath string) error {
	composePath, err := findComposeFile(projectPath)
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composePath, "up", "-d")
	cmd.Dir = projectPath
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("compose up: %s", stderr.String())
		}
		return fmt.Errorf("compose up: %w", err)
	}
	return nil
}

func ComposeDown(ctx context.Context, projectPath string) error {
	composePath, err := findComposeFile(projectPath)
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composePath, "down")
	cmd.Dir = projectPath
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("compose down: %s", stderr.String())
		}
		return fmt.Errorf("compose down: %w", err)
	}
	return nil
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

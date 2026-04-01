//go:build integration

package docker_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/idesyatov/wharf/internal/docker"
)

var testComposeDir string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "wharf-integration-*")
	if err != nil {
		panic(err)
	}
	testComposeDir = dir

	compose := `services:
  web:
    image: busybox:latest
    command: ["sh", "-c", "echo 'hello from wharf test' && sleep 3600"]
`
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(compose), 0644); err != nil {
		panic(err)
	}

	// Start compose stack
	cmd := exec.Command("docker", "compose", "-f", filepath.Join(dir, "compose.yaml"), "-p", "wharf-test", "up", "-d")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("compose up failed: " + string(out))
	}

	// Wait for container to start
	time.Sleep(2 * time.Second)

	code := m.Run()

	// Cleanup
	cleanup := exec.Command("docker", "compose", "-f", filepath.Join(dir, "compose.yaml"), "-p", "wharf-test", "down", "-v")
	cleanup.Dir = dir
	_ = cleanup.Run()
	os.RemoveAll(dir)

	os.Exit(code)
}

func TestListProjects_Integration(t *testing.T) {
	client, err := docker.NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	projects, err := client.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}

	found := false
	for _, p := range projects {
		if p.Name == "wharf-test" {
			found = true
			if len(p.Services) == 0 {
				t.Error("expected at least 1 service")
			}
			break
		}
	}
	if !found {
		t.Error("project 'wharf-test' not found")
	}
}

func TestListContainers_Integration(t *testing.T) {
	client, err := docker.NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	containers, err := client.ListContainers(context.Background())
	if err != nil {
		t.Fatalf("ListContainers: %v", err)
	}

	found := false
	for _, c := range containers {
		if c.Labels["com.docker.compose.project"] == "wharf-test" {
			found = true
			if c.Status != "running" {
				t.Errorf("expected running, got %s", c.Status)
			}
			break
		}
	}
	if !found {
		t.Error("wharf-test container not found")
	}
}

func TestContainerStats_Integration(t *testing.T) {
	client, err := docker.NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	containers, err := client.ListContainers(context.Background())
	if err != nil {
		t.Fatalf("ListContainers: %v", err)
	}

	for _, c := range containers {
		if c.Labels["com.docker.compose.project"] == "wharf-test" {
			stats, err := client.ContainerStats(context.Background(), c.ID)
			if err != nil {
				t.Fatalf("ContainerStats: %v", err)
			}
			if stats.MemUsage == 0 {
				t.Error("expected MemUsage > 0")
			}
			return
		}
	}
	t.Error("wharf-test container not found for stats")
}

func TestStopStartContainer_Integration(t *testing.T) {
	client, err := docker.NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	containers, err := client.ListContainers(context.Background())
	if err != nil {
		t.Fatalf("ListContainers: %v", err)
	}

	for _, c := range containers {
		if c.Labels["com.docker.compose.project"] == "wharf-test" {
			// Stop
			if err := client.StopContainer(context.Background(), c.ID); err != nil {
				t.Fatalf("StopContainer: %v", err)
			}
			time.Sleep(1 * time.Second)

			// Start
			if err := client.StartContainer(context.Background(), c.ID); err != nil {
				t.Fatalf("StartContainer: %v", err)
			}
			time.Sleep(1 * time.Second)
			return
		}
	}
	t.Error("wharf-test container not found")
}

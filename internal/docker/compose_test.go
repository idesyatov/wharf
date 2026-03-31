package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalcServiceStatus_AllRunning(t *testing.T) {
	got := calcServiceStatus([]Container{
		{Status: "running"},
		{Status: "running"},
	})
	if got != StatusRunning {
		t.Errorf("expected running, got %s", got)
	}
}

func TestCalcServiceStatus_Mixed(t *testing.T) {
	got := calcServiceStatus([]Container{
		{Status: "running"},
		{Status: "exited"},
	})
	if got != StatusPartial {
		t.Errorf("expected partial, got %s", got)
	}
}

func TestCalcServiceStatus_AllStopped(t *testing.T) {
	got := calcServiceStatus([]Container{
		{Status: "exited"},
		{Status: "exited"},
	})
	if got != StatusStopped {
		t.Errorf("expected stopped, got %s", got)
	}
}

func TestCalcServiceStatus_Empty(t *testing.T) {
	// 0 running == 0 total → vacuously "running" (0/0)
	got := calcServiceStatus(nil)
	if got != StatusRunning {
		t.Errorf("expected running for empty (0==0), got %s", got)
	}
}

func TestCalcProjectStatus_AllRunning(t *testing.T) {
	got := calcProjectStatus([]Service{
		{Status: StatusRunning},
		{Status: StatusRunning},
	})
	if got != StatusRunning {
		t.Errorf("expected running, got %s", got)
	}
}

func TestCalcProjectStatus_Mixed(t *testing.T) {
	got := calcProjectStatus([]Service{
		{Status: StatusRunning},
		{Status: StatusStopped},
	})
	if got != StatusPartial {
		t.Errorf("expected partial, got %s", got)
	}
}

func TestCalcProjectStatus_AllStopped(t *testing.T) {
	got := calcProjectStatus([]Service{
		{Status: StatusStopped},
	})
	if got != StatusStopped {
		t.Errorf("expected stopped, got %s", got)
	}
}

func TestFindComposeFile_ComposeYaml(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services:"), 0644)
	got, err := FindComposeFile(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(got) != "compose.yaml" {
		t.Errorf("expected compose.yaml, got %s", filepath.Base(got))
	}
}

func TestFindComposeFile_DockerComposeYml(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services:"), 0644)
	got, err := FindComposeFile(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(got) != "docker-compose.yml" {
		t.Errorf("expected docker-compose.yml, got %s", filepath.Base(got))
	}
}

func TestFindComposeFile_Priority(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services:"), 0644)
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services:"), 0644)
	got, err := FindComposeFile(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(got) != "compose.yaml" {
		t.Errorf("expected compose.yaml (higher priority), got %s", filepath.Base(got))
	}
}

func TestFindComposeFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := FindComposeFile(dir)
	if err == nil {
		t.Error("expected error for missing compose file")
	}
}

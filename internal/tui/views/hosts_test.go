package views

import (
	"testing"

	"github.com/idesyatov/wharf/internal/config"
)

func TestNewHostsView(t *testing.T) {
	hosts := []config.HostEntry{
		{Name: "prod", URL: "ssh://prod.srv"},
		{Name: "staging", URL: "ssh://staging.srv"},
	}
	v := NewHostsView(hosts, "")

	if len(v.hosts) != 3 {
		t.Fatalf("expected 3 entries (local + 2), got %d", len(v.hosts))
	}
	if v.hosts[0].name != "local" {
		t.Errorf("first host should be 'local', got %s", v.hosts[0].name)
	}
	if !v.hosts[0].active {
		t.Error("local should be active when activeURL is empty")
	}
	if v.hosts[1].active {
		t.Error("prod should not be active")
	}
}

func TestNewHostsViewActiveRemote(t *testing.T) {
	hosts := []config.HostEntry{
		{Name: "prod", URL: "ssh://prod.srv"},
	}
	v := NewHostsView(hosts, "ssh://prod.srv")

	if v.hosts[0].active {
		t.Error("local should not be active when remote is connected")
	}
	if !v.hosts[1].active {
		t.Error("prod should be active")
	}
}

func TestHostsViewBreadcrumb(t *testing.T) {
	v := NewHostsView(nil, "")
	if v.Breadcrumb() != "Hosts" {
		t.Errorf("expected 'Hosts', got %s", v.Breadcrumb())
	}
}

func TestHostsViewPendingDeleteLocal(t *testing.T) {
	v := NewHostsView(nil, "")
	// cursor is 0 (local) — PendingDeleteName should return empty (can't delete local)
	if v.PendingDeleteName() != "" {
		t.Error("should not be able to get delete name for local (cursor 0)")
	}
}

func TestHostsViewPendingDeleteRemote(t *testing.T) {
	hosts := []config.HostEntry{
		{Name: "prod", URL: "ssh://prod.srv"},
	}
	v := NewHostsView(hosts, "")
	v.cursor = 1

	name := v.PendingDeleteName()
	if name != "prod" {
		t.Errorf("expected 'prod', got %s", name)
	}
}

func TestHostsViewSetSize(t *testing.T) {
	v := NewHostsView(nil, "")
	v = v.SetSize(120, 40)
	if v.width != 120 || v.height != 40 {
		t.Errorf("expected 120x40, got %dx%d", v.width, v.height)
	}
}

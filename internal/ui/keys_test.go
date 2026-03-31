package ui

import (
	"testing"
)

func TestApplyKeyBindings_Empty(t *testing.T) {
	original := DefaultKeyMap()
	applied := ApplyKeyBindings(original, nil)

	origKeys := original.Quit.Keys()
	appliedKeys := applied.Quit.Keys()
	if len(origKeys) != len(appliedKeys) || origKeys[0] != appliedKeys[0] {
		t.Error("empty bindings should not change KeyMap")
	}
}

func TestApplyKeyBindings_EmptyMap(t *testing.T) {
	original := DefaultKeyMap()
	applied := ApplyKeyBindings(original, map[string]string{})

	origKeys := original.Quit.Keys()
	appliedKeys := applied.Quit.Keys()
	if len(origKeys) != len(appliedKeys) || origKeys[0] != appliedKeys[0] {
		t.Error("empty map should not change KeyMap")
	}
}

func TestApplyKeyBindings_Override(t *testing.T) {
	km := DefaultKeyMap()
	applied := ApplyKeyBindings(km, map[string]string{
		"quit": "ctrl+q",
	})

	keys := applied.Quit.Keys()
	if len(keys) != 1 || keys[0] != "ctrl+q" {
		t.Errorf("expected [ctrl+q], got %v", keys)
	}

	// Other bindings unchanged
	upKeys := applied.Up.Keys()
	if upKeys[0] != "k" {
		t.Errorf("Up should remain 'k', got %s", upKeys[0])
	}
}

func TestApplyKeyBindings_MultipleOverrides(t *testing.T) {
	km := DefaultKeyMap()
	applied := ApplyKeyBindings(km, map[string]string{
		"quit":   "ctrl+q",
		"up":     "w",
		"down":   "s",
		"select": "space",
	})

	if applied.Quit.Keys()[0] != "ctrl+q" {
		t.Error("quit override failed")
	}
	if applied.Up.Keys()[0] != "w" {
		t.Error("up override failed")
	}
	if applied.Down.Keys()[0] != "s" {
		t.Error("down override failed")
	}
	if applied.Right.Keys()[0] != "space" {
		t.Error("select override failed")
	}
}

func TestDefaultKeyMap_HasAllBindings(t *testing.T) {
	km := DefaultKeyMap()

	bindings := []struct {
		name string
		keys []string
	}{
		{"Quit", km.Quit.Keys()},
		{"ForceQuit", km.ForceQuit.Keys()},
		{"Up", km.Up.Keys()},
		{"Down", km.Down.Keys()},
		{"Left", km.Left.Keys()},
		{"Right", km.Right.Keys()},
		{"Start", km.Start.Keys()},
		{"Stop", km.Stop.Keys()},
		{"Restart", km.Restart.Keys()},
		{"Logs", km.Logs.Keys()},
		{"Exec", km.Exec.Keys()},
	}

	for _, b := range bindings {
		if len(b.keys) == 0 {
			t.Errorf("%s has no keys", b.name)
		}
	}
}

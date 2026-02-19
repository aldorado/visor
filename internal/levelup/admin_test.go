package levelup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnableDisableList(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"himalaya", "obsidian"} {
		dir := filepath.Join(root, "levelups", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := "name = \"" + name + "\"\ndisplay_name = \"" + name + "\"\ncompose_file = \"docker-compose.yml\"\n"
		if err := os.WriteFile(filepath.Join(dir, "levelup.toml"), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := Enable(root, []string{"himalaya"}); err != nil {
		t.Fatalf("enable failed: %v", err)
	}

	statuses, err := List(root)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	enabled := map[string]bool{}
	for _, s := range statuses {
		enabled[s.Name] = s.Enabled
	}
	if !enabled["himalaya"] {
		t.Fatal("expected himalaya enabled")
	}
	if enabled["obsidian"] {
		t.Fatal("expected obsidian disabled")
	}

	if err := Disable(root, []string{"himalaya"}); err != nil {
		t.Fatalf("disable failed: %v", err)
	}

	statuses, err = List(root)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	for _, s := range statuses {
		if s.Enabled {
			t.Fatalf("expected all disabled, got enabled: %s", s.Name)
		}
	}
}

func TestEnableUnknownLevelupFails(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "levelups", "himalaya")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "levelup.toml"), []byte("name = \"himalaya\"\ncompose_file = \"x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Enable(root, []string{"missing"}); err == nil {
		t.Fatal("expected error")
	}
}

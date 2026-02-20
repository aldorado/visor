package levelup

import (
	"os"
	"path/filepath"
	"strings"
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

func TestProxyRoutesLifecycleOnEnableDisableReenable(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".levelup.env"), []byte("PROXY_DOMAIN=example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	proxyDir := filepath.Join(root, "levelups", "proxy")
	if err := os.MkdirAll(proxyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	proxyManifest := "name = \"proxy\"\ndisplay_name = \"proxy\"\ncompose_file = \"docker-compose.levelup.proxy.yml\"\n"
	if err := os.WriteFile(filepath.Join(proxyDir, "levelup.toml"), []byte(proxyManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	obsidianDir := filepath.Join(root, "levelups", "obsidian")
	if err := os.MkdirAll(obsidianDir, 0o755); err != nil {
		t.Fatal(err)
	}
	obsidianManifest := "name = \"obsidian\"\ndisplay_name = \"obsidian\"\ncompose_file = \"docker-compose.levelup.obsidian.yml\"\nproxy_service = \"obsidian\"\nproxy_port = 3000\n"
	if err := os.WriteFile(filepath.Join(obsidianDir, "levelup.toml"), []byte(obsidianManifest), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Enable(root, []string{"proxy", "obsidian"}); err != nil {
		t.Fatalf("enable failed: %v", err)
	}
	cfgPath := filepath.Join(root, "data", "levelups", "proxy", "Caddyfile.autogen")
	cfg, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), "obsidian.visor.example.com") {
		t.Fatalf("expected obsidian route after enable, got: %s", string(cfg))
	}

	if err := Disable(root, []string{"obsidian"}); err != nil {
		t.Fatalf("disable failed: %v", err)
	}
	cfg, err = os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cfg), "obsidian.visor.example.com") {
		t.Fatalf("expected obsidian route removed after disable, got: %s", string(cfg))
	}

	if err := Enable(root, []string{"obsidian"}); err != nil {
		t.Fatalf("re-enable failed: %v", err)
	}
	cfg, err = os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), "obsidian.visor.example.com") {
		t.Fatalf("expected obsidian route restored after re-enable, got: %s", string(cfg))
	}
}

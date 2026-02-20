package levelup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildProxyRoutes(t *testing.T) {
	manifests := map[string]Manifest{
		"proxy": {
			Name: "proxy",
		},
		"obsidian": {
			Name:         "obsidian",
			Subdomain:    "vault",
			ProxyService: "obsidian",
			ProxyPort:    3000,
		},
		"echo-stub": {
			Name:         "echo-stub",
			ProxyService: "echo-stub",
			ProxyPort:    5678,
		},
	}

	routes := buildProxyRoutes(manifests, []string{"proxy", "obsidian", "echo-stub"}, "example.com")
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}
	if routes[0].Host != "echo-stub.visor.example.com" {
		t.Fatalf("unexpected first host: %s", routes[0].Host)
	}
	if routes[1].Host != "vault.visor.example.com" {
		t.Fatalf("unexpected second host: %s", routes[1].Host)
	}
	if routes[1].Upstream != "http://obsidian:3000" {
		t.Fatalf("unexpected upstream: %s", routes[1].Upstream)
	}
}

func TestSyncProxyConfigForEnabledWritesFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".levelup.env"), []byte("PROXY_DOMAIN=example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	manifests := map[string]Manifest{
		"proxy": {
			Name: "proxy",
		},
		"obsidian": {
			Name:         "obsidian",
			Subdomain:    "vault",
			ProxyService: "obsidian",
			ProxyPort:    3000,
		},
	}

	if err := SyncProxyConfigForEnabled(root, manifests, []string{"proxy", "obsidian"}); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	bytes, err := os.ReadFile(filepath.Join(root, "data", "levelups", "proxy", "Caddyfile.autogen"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(bytes)
	if !strings.Contains(content, "vault.visor.example.com") {
		t.Fatalf("expected route in caddyfile, got: %s", content)
	}
	if !strings.Contains(content, "handle /_health") {
		t.Fatalf("expected per-subdomain health handler in caddyfile, got: %s", content)
	}
}

func TestSyncProxyConfigForEnabledRequiresDomain(t *testing.T) {
	root := t.TempDir()
	manifests := map[string]Manifest{"proxy": {Name: "proxy"}}
	if err := SyncProxyConfigForEnabled(root, manifests, []string{"proxy"}); err == nil {
		t.Fatal("expected PROXY_DOMAIN error")
	}
}

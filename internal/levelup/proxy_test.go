package levelup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildProxyRoutes(t *testing.T) {
	manifests := map[string]Manifest{
		"proxy": {Name: "proxy"},
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
	env := map[string]string{
		"PROXY_AUTH_VAULT_USER":        "anuar",
		"PROXY_AUTH_VAULT_PASS_BCRYPT": "$2a$14$abc",
		"PROXY_ALLOW_VAULT":            "10.0.0.0/8,127.0.0.1/32",
	}

	routes := buildProxyRoutes(manifests, []string{"proxy", "obsidian", "echo-stub"}, "example.com", env)
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}
	if routes[1].Host != "vault.visor.example.com" {
		t.Fatalf("unexpected second host: %s", routes[1].Host)
	}
	if routes[1].Access.AuthUser != "anuar" {
		t.Fatalf("expected route auth user from env")
	}
	if len(routes[1].Access.AllowCIDRs) != 2 {
		t.Fatalf("expected allow list from env")
	}
}

func TestSyncProxyConfigForEnabledWritesAccessControlsAndDashboard(t *testing.T) {
	root := t.TempDir()
	env := strings.Join([]string{
		"PROXY_DOMAIN=example.com",
		"PROXY_AUTH_VAULT_USER=anuar",
		"PROXY_AUTH_VAULT_PASS_BCRYPT=$2a$14$hash",
		"PROXY_ALLOW_VAULT=127.0.0.1/32",
		"PROXY_ADMIN_SUBDOMAIN=ops",
		"PROXY_ADMIN_AUTH_USER=ops",
		"PROXY_ADMIN_AUTH_PASS_BCRYPT=$2a$14$ops",
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(root, ".levelup.env"), []byte(env), 0o644); err != nil {
		t.Fatal(err)
	}

	manifests := map[string]Manifest{
		"proxy": {Name: "proxy"},
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
		t.Fatalf("expected route host in caddyfile")
	}
	if !strings.Contains(content, "basic_auth") {
		t.Fatalf("expected basic_auth in caddyfile")
	}
	if !strings.Contains(content, "@allow_") {
		t.Fatalf("expected allow matcher in caddyfile")
	}
	if !strings.Contains(content, "ops.visor.example.com") {
		t.Fatalf("expected admin dashboard subdomain in caddyfile")
	}
	if !strings.Contains(content, "handle /metrics") {
		t.Fatalf("expected admin metrics handler")
	}
}

func TestRouteAccessForSubdomainSharedAuth(t *testing.T) {
	env := map[string]string{
		"PROXY_AUTH_SHARED_USER":          "shared",
		"PROXY_AUTH_SHARED_PASS_BCRYPT":   "$2a$14$shared",
		"PROXY_AUTH_OBSIDIAN_USE_SHARED":  "true",
		"PROXY_AUTH_FORGEJO_USE_SHARED":   "false",
		"PROXY_AUTH_OBSIDIAN_PASS_BCRYPT": "",
	}

	obsidian := routeAccessForSubdomain("obsidian", env)
	if obsidian.AuthUser != "shared" || obsidian.AuthPassBcrypt != "$2a$14$shared" {
		t.Fatalf("expected shared auth credentials for obsidian route")
	}

	forgejo := routeAccessForSubdomain("forgejo", env)
	if forgejo.AuthUser != "" || forgejo.AuthPassBcrypt != "" {
		t.Fatalf("expected forgejo auth to stay empty when shared auth not enabled")
	}
}

func TestRouteAccessForSubdomainExplicitAuthOverridesShared(t *testing.T) {
	env := map[string]string{
		"PROXY_AUTH_SHARED_USER":          "shared",
		"PROXY_AUTH_SHARED_PASS_BCRYPT":   "$2a$14$shared",
		"PROXY_AUTH_OBSIDIAN_USE_SHARED":  "true",
		"PROXY_AUTH_OBSIDIAN_USER":        "obsidian",
		"PROXY_AUTH_OBSIDIAN_PASS_BCRYPT": "$2a$14$obsidian",
	}

	access := routeAccessForSubdomain("obsidian", env)
	if access.AuthUser != "obsidian" || access.AuthPassBcrypt != "$2a$14$obsidian" {
		t.Fatalf("expected explicit route auth credentials to override shared")
	}
}

func TestSyncProxyConfigForEnabledRequiresDomain(t *testing.T) {
	root := t.TempDir()
	manifests := map[string]Manifest{"proxy": {Name: "proxy"}}
	if err := SyncProxyConfigForEnabled(root, manifests, []string{"proxy"}); err == nil {
		t.Fatal("expected PROXY_DOMAIN error")
	}
}

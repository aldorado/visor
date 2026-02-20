package levelup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const proxyLevelupName = "proxy"

func SyncProxyConfig(projectRoot string) error {
	manifests, err := DiscoverManifests(projectRoot)
	if err != nil {
		return err
	}
	state, err := LoadState(projectRoot)
	if err != nil {
		return err
	}
	return SyncProxyConfigForEnabled(projectRoot, manifests, state.Enabled)
}

func SyncProxyConfigForEnabled(projectRoot string, manifests map[string]Manifest, enabled []string) error {
	if !isEnabled(enabled, proxyLevelupName) {
		return nil
	}
	env, err := LoadLayeredEnv(projectRoot)
	if err != nil {
		return err
	}
	domain := strings.TrimSpace(env["PROXY_DOMAIN"])
	if domain == "" {
		return fmt.Errorf("PROXY_DOMAIN is required when proxy level-up is enabled")
	}

	routes := buildProxyRoutes(manifests, enabled, domain)
	content := renderCaddyfile(routes)
	path := filepath.Join(projectRoot, "data", "levelups", "proxy", "Caddyfile.autogen")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

type proxyRoute struct {
	Host     string
	Upstream string
}

func buildProxyRoutes(manifests map[string]Manifest, enabled []string, domain string) []proxyRoute {
	routes := make([]proxyRoute, 0, len(enabled))
	for _, name := range enabled {
		if name == proxyLevelupName {
			continue
		}
		m, ok := manifests[name]
		if !ok || m.ProxyPort <= 0 {
			continue
		}
		service := strings.TrimSpace(m.ProxyService)
		if service == "" {
			service = name
		}
		subdomain := strings.TrimSpace(m.Subdomain)
		if subdomain == "" {
			subdomain = name
		}
		host := fmt.Sprintf("%s.visor.%s", subdomain, domain)
		routes = append(routes, proxyRoute{
			Host:     host,
			Upstream: fmt.Sprintf("http://%s:%d", service, m.ProxyPort),
		})
	}
	sort.Slice(routes, func(i, j int) bool { return routes[i].Host < routes[j].Host })
	return routes
}

func renderCaddyfile(routes []proxyRoute) string {
	lines := []string{
		"{",
		"\tauto_https disable_redirects",
		"}",
		"",
		":80 {",
		"\thandle /_proxy/health {",
		"\t\trespond \"ok\" 200",
		"\t}",
		"\trespond \"visor proxy\" 200",
		"}",
	}
	for _, route := range routes {
		lines = append(lines,
			"",
			route.Host+" {",
			"\thandle /_health {",
			"\t\treverse_proxy "+route.Upstream,
			"\t}",
			"\treverse_proxy "+route.Upstream,
			"}",
		)
	}
	return strings.Join(lines, "\n") + "\n"
}

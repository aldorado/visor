package levelup

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const proxyLevelupName = "proxy"

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]+`)

type proxyAccess struct {
	AuthUser       string
	AuthPassBcrypt string
	AllowCIDRs     []string
	DenyCIDRs      []string
}

type proxyRoute struct {
	Host     string
	Upstream string
	Access   proxyAccess
}

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

	routes := buildProxyRoutes(manifests, enabled, domain, env)
	content := renderCaddyfile(routes, domain, env)
	path := filepath.Join(projectRoot, "data", "levelups", "proxy", "Caddyfile.autogen")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func buildProxyRoutes(manifests map[string]Manifest, enabled []string, domain string, env map[string]string) []proxyRoute {
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
			Access:   routeAccessForSubdomain(subdomain, env),
		})
	}
	sort.Slice(routes, func(i, j int) bool { return routes[i].Host < routes[j].Host })
	return routes
}

func routeAccessForSubdomain(subdomain string, env map[string]string) proxyAccess {
	key := envKeySegment(subdomain)
	authUser := strings.TrimSpace(env["PROXY_AUTH_"+key+"_USER"])
	authPass := strings.TrimSpace(env["PROXY_AUTH_"+key+"_PASS_BCRYPT"])
	if authUser == "" || authPass == "" {
		if useSharedAuthForSubdomain(key, env) {
			authUser = strings.TrimSpace(env["PROXY_AUTH_SHARED_USER"])
			authPass = strings.TrimSpace(env["PROXY_AUTH_SHARED_PASS_BCRYPT"])
		}
	}
	return proxyAccess{
		AuthUser:       authUser,
		AuthPassBcrypt: authPass,
		AllowCIDRs:     splitCSV(env["PROXY_ALLOW_"+key]),
		DenyCIDRs:      splitCSV(env["PROXY_DENY_"+key]),
	}
}

func useSharedAuthForSubdomain(key string, env map[string]string) bool {
	if key == "" {
		return false
	}
	if !isTrue(env["PROXY_AUTH_"+key+"_USE_SHARED"]) {
		return false
	}
	return strings.TrimSpace(env["PROXY_AUTH_SHARED_USER"]) != "" && strings.TrimSpace(env["PROXY_AUTH_SHARED_PASS_BCRYPT"]) != ""
}

func isTrue(raw string) bool {
	v, err := strconv.ParseBool(strings.TrimSpace(raw))
	return err == nil && v
}

func renderCaddyfile(routes []proxyRoute, domain string, env map[string]string) string {
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
	for i, route := range routes {
		lines = append(lines, "", route.Host+" {")
		lines = appendRoutePolicy(lines, route, i)
		lines = append(lines, "}")
	}

	adminSubdomain := strings.TrimSpace(env["PROXY_ADMIN_SUBDOMAIN"])
	if adminSubdomain == "" {
		adminSubdomain = "admin"
	}
	adminHost := fmt.Sprintf("%s.visor.%s", adminSubdomain, domain)
	adminAccess := proxyAccess{
		AuthUser:       strings.TrimSpace(env["PROXY_ADMIN_AUTH_USER"]),
		AuthPassBcrypt: strings.TrimSpace(env["PROXY_ADMIN_AUTH_PASS_BCRYPT"]),
		AllowCIDRs:     splitCSV(env["PROXY_ADMIN_ALLOW"]),
		DenyCIDRs:      splitCSV(env["PROXY_ADMIN_DENY"]),
	}
	lines = append(lines, "", adminHost+" {")
	lines = appendAdminPolicy(lines, routes, adminAccess)
	lines = append(lines, "}")

	return strings.Join(lines, "\n") + "\n"
}

func appendRoutePolicy(lines []string, route proxyRoute, idx int) []string {
	if len(route.Access.DenyCIDRs) > 0 {
		lines = append(lines,
			fmt.Sprintf("\t@deny_%d remote_ip %s", idx, strings.Join(route.Access.DenyCIDRs, " ")),
			fmt.Sprintf("\trespond @deny_%d \"forbidden\" 403", idx),
		)
	}
	if len(route.Access.AllowCIDRs) > 0 {
		lines = append(lines,
			fmt.Sprintf("\t@allow_%d remote_ip %s", idx, strings.Join(route.Access.AllowCIDRs, " ")),
			fmt.Sprintf("\thandle @allow_%d {", idx),
		)
		lines = appendAuthAndProxy(lines, route, "\t\t")
		lines = append(lines,
			"\t}",
			"\trespond \"forbidden\" 403",
		)
		return lines
	}
	return appendAuthAndProxy(lines, route, "\t")
}

func appendAdminPolicy(lines []string, routes []proxyRoute, access proxyAccess) []string {
	if len(access.DenyCIDRs) > 0 {
		lines = append(lines,
			fmt.Sprintf("\t@admin_deny remote_ip %s", strings.Join(access.DenyCIDRs, " ")),
			"\trespond @admin_deny \"forbidden\" 403",
		)
	}

	innerIndent := "\t"
	if len(access.AllowCIDRs) > 0 {
		lines = append(lines,
			fmt.Sprintf("\t@admin_allow remote_ip %s", strings.Join(access.AllowCIDRs, " ")),
			"\thandle @admin_allow {",
		)
		innerIndent = "\t\t"
	}

	if access.AuthUser != "" && access.AuthPassBcrypt != "" {
		lines = append(lines,
			innerIndent+"basic_auth {",
			innerIndent+"\t"+access.AuthUser+" "+access.AuthPassBcrypt,
			innerIndent+"}",
		)
	}

	summary := "routes:"
	for _, route := range routes {
		summary += " " + route.Host + "->" + route.Upstream + ";"
	}
	lines = append(lines,
		innerIndent+"handle /status {",
		innerIndent+"\trespond "+strconv.Quote(summary)+" 200",
		innerIndent+"}",
		innerIndent+"handle /metrics {",
		innerIndent+"\trespond "+strconv.Quote(fmt.Sprintf("proxy_routes_total %d", len(routes)))+" 200",
		innerIndent+"}",
		innerIndent+"respond \"visor proxy admin\" 200",
	)

	if len(access.AllowCIDRs) > 0 {
		lines = append(lines,
			"\t}",
			"\trespond \"forbidden\" 403",
		)
	}
	return lines
}

func appendAuthAndProxy(lines []string, route proxyRoute, indent string) []string {
	if route.Access.AuthUser != "" && route.Access.AuthPassBcrypt != "" {
		lines = append(lines,
			indent+"basic_auth {",
			indent+"\t"+route.Access.AuthUser+" "+route.Access.AuthPassBcrypt,
			indent+"}",
		)
	}
	lines = append(lines,
		indent+"handle /_health {",
		indent+"\treverse_proxy "+route.Upstream,
		indent+"}",
		indent+"reverse_proxy "+route.Upstream,
	)
	return lines
}

func envKeySegment(s string) string {
	clean := strings.TrimSpace(s)
	if clean == "" {
		return ""
	}
	clean = strings.ToUpper(clean)
	clean = nonAlnum.ReplaceAllString(clean, "_")
	clean = strings.Trim(clean, "_")
	return clean
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

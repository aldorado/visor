package levelup

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var envPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-([^}]*))?\}`)

func CheckEnabledHTTPHealth(ctx context.Context, projectRoot string) ([]string, error) {
	manifests, err := DiscoverManifests(projectRoot)
	if err != nil {
		return nil, err
	}
	state, err := LoadState(projectRoot)
	if err != nil {
		return nil, err
	}
	env, err := LoadLayeredEnv(projectRoot)
	if err != nil {
		return nil, err
	}
	failed := make([]string, 0)
	for _, name := range state.Enabled {
		m, ok := manifests[name]
		if !ok {
			return nil, fmt.Errorf("enabled level-up %q has no manifest", name)
		}
		h := strings.TrimSpace(m.Healthcheck)
		if h == "" {
			continue
		}
		resolved := expandEnvDefaults(h, env)
		u, err := url.Parse(resolved)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s invalid healthcheck URL", name))
			continue
		}
		host := strings.ToLower(u.Hostname())
		if host != "127.0.0.1" && host != "localhost" {
			// non-host endpoints are skipped in host-native setup flow
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolved, nil)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s healthcheck build failed", name))
			continue
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s healthcheck failed: %v", name, err))
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			failed = append(failed, fmt.Sprintf("%s healthcheck status %d", name, resp.StatusCode))
		}
	}
	return failed, nil
}

func expandEnvDefaults(input string, env map[string]string) string {
	return envPattern.ReplaceAllStringFunc(input, func(m string) string {
		parts := envPattern.FindStringSubmatch(m)
		if len(parts) < 2 {
			return m
		}
		key := parts[1]
		if v, ok := env[key]; ok && strings.TrimSpace(v) != "" {
			return v
		}
		if len(parts) >= 4 {
			return parts[3]
		}
		return ""
	})
}

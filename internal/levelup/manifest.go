package levelup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Manifest struct {
	Name             string   `toml:"name"`
	DisplayName      string   `toml:"display_name"`
	Version          string   `toml:"version"`
	Description      string   `toml:"description"`
	Kind             string   `toml:"kind"`
	EnabledByDefault bool     `toml:"enabled_by_default"`
	ComposeFile      string   `toml:"compose_file"`
	Healthcheck      string   `toml:"healthcheck"`
	Tags             []string `toml:"tags"`
	RequiredEnv      []string `toml:"required_env"`
	Subdomain        string   `toml:"subdomain"`
	ProxyService     string   `toml:"proxy_service"`
	ProxyPort        int      `toml:"proxy_port"`
	Path             string   `toml:"-"`
}

func DiscoverManifests(projectRoot string) (map[string]Manifest, error) {
	if projectRoot == "" {
		return nil, fmt.Errorf("project root is required")
	}

	pattern := filepath.Join(projectRoot, "levelups", "*", "levelup.toml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob manifests: %w", err)
	}

	out := map[string]Manifest{}
	for _, match := range matches {
		var m Manifest
		if _, err := toml.DecodeFile(match, &m); err != nil {
			return nil, fmt.Errorf("decode manifest %s: %w", match, err)
		}
		if m.Name == "" {
			return nil, fmt.Errorf("manifest missing name: %s", match)
		}
		if _, ok := out[m.Name]; ok {
			return nil, fmt.Errorf("duplicate level-up name %q", m.Name)
		}
		m.Path = match
		out[m.Name] = m
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no level-up manifests found in %s", filepath.Join(projectRoot, "levelups"))
	}

	return out, nil
}

func EnsureLevelupDataDir(projectRoot string) (string, error) {
	dir := filepath.Join(projectRoot, "data", "levelups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return dir, nil
}

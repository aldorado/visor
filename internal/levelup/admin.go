package levelup

import (
	"context"
	"fmt"
	"sort"
)

type Status struct {
	Name        string
	DisplayName string
	Enabled     bool
}

func List(projectRoot string) ([]Status, error) {
	manifests, err := DiscoverManifests(projectRoot)
	if err != nil {
		return nil, err
	}

	state, err := LoadState(projectRoot)
	if err != nil {
		return nil, err
	}

	enabledSet := make(map[string]struct{}, len(state.Enabled))
	for _, name := range state.Enabled {
		enabledSet[name] = struct{}{}
	}

	statuses := make([]Status, 0, len(manifests))
	for name, m := range manifests {
		_, enabled := enabledSet[name]
		statuses = append(statuses, Status{
			Name:        name,
			DisplayName: m.DisplayName,
			Enabled:     enabled,
		})
	}

	sort.Slice(statuses, func(i, j int) bool { return statuses[i].Name < statuses[j].Name })
	return statuses, nil
}

func Enable(projectRoot string, names []string) error {
	if len(names) == 0 {
		return fmt.Errorf("at least one level-up name is required")
	}

	manifests, err := DiscoverManifests(projectRoot)
	if err != nil {
		return err
	}

	state, err := LoadState(projectRoot)
	if err != nil {
		return err
	}

	enabled := make(map[string]struct{}, len(state.Enabled))
	for _, name := range state.Enabled {
		enabled[name] = struct{}{}
	}

	for _, name := range names {
		if _, ok := manifests[name]; !ok {
			return fmt.Errorf("unknown level-up: %s", name)
		}
		enabled[name] = struct{}{}
	}

	updated := make([]string, 0, len(enabled))
	for name := range enabled {
		updated = append(updated, name)
	}

	return SaveState(projectRoot, State{Enabled: updated})
}

func Disable(projectRoot string, names []string) error {
	if len(names) == 0 {
		return fmt.Errorf("at least one level-up name is required")
	}

	manifests, err := DiscoverManifests(projectRoot)
	if err != nil {
		return err
	}

	state, err := LoadState(projectRoot)
	if err != nil {
		return err
	}

	enabled := make(map[string]struct{}, len(state.Enabled))
	for _, name := range state.Enabled {
		enabled[name] = struct{}{}
	}

	for _, name := range names {
		if _, ok := manifests[name]; !ok {
			return fmt.Errorf("unknown level-up: %s", name)
		}
		delete(enabled, name)
	}

	updated := make([]string, 0, len(enabled))
	for name := range enabled {
		updated = append(updated, name)
	}

	return SaveState(projectRoot, State{Enabled: updated})
}

func ValidateEnabled(ctx context.Context, projectRoot, baseComposeFile string) error {
	manifests, err := DiscoverManifests(projectRoot)
	if err != nil {
		return err
	}
	state, err := LoadState(projectRoot)
	if err != nil {
		return err
	}

	overlays := make([]string, 0, len(state.Enabled))
	for _, name := range state.Enabled {
		manifest, ok := manifests[name]
		if !ok {
			return fmt.Errorf("enabled level-up %q has no manifest", name)
		}
		overlays = append(overlays, manifest.ComposeFile)
	}

	assembly, err := BuildComposeAssembly(projectRoot, baseComposeFile, overlays)
	if err != nil {
		return err
	}
	env, err := LoadLayeredEnv(projectRoot)
	if err != nil {
		return err
	}

	for _, name := range state.Enabled {
		manifest := manifests[name]
		if err := ValidateRequiredEnv(env, manifest.RequiredEnv); err != nil {
			return fmt.Errorf("level-up %s: %w", name, err)
		}
		if name == "obsidian" {
			if err := ValidateObsidianMounts(env); err != nil {
				return fmt.Errorf("level-up obsidian mounts: %w", err)
			}
		}
	}

	if err := ValidateComposeConfig(ctx, assembly, env); err != nil {
		return err
	}

	if isEnabled(state.Enabled, "obsidian") {
		if smokeURL := env["OBSIDIAN_SMOKE_URL"]; smokeURL != "" {
			if err := CheckObsidianReachable(ctx, smokeURL); err != nil {
				return fmt.Errorf("obsidian smoke check failed: %w", err)
			}
		}
	}

	return nil
}

func isEnabled(enabled []string, name string) bool {
	for _, item := range enabled {
		if item == name {
			return true
		}
	}
	return false
}

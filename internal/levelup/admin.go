package levelup

import (
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

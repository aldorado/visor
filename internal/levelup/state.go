package levelup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type State struct {
	Enabled []string `json:"enabled"`
}

func statePath(projectRoot string) string {
	return filepath.Join(projectRoot, "data", "levelups", "enabled.json")
}

func LoadState(projectRoot string) (State, error) {
	path := statePath(projectRoot)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return State{Enabled: []string{}}, nil
		}
		return State{}, fmt.Errorf("stat %s: %w", path, err)
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return State{}, fmt.Errorf("read %s: %w", path, err)
	}

	var s State
	if err := json.Unmarshal(bytes, &s); err != nil {
		return State{}, fmt.Errorf("decode %s: %w", path, err)
	}

	if s.Enabled == nil {
		s.Enabled = []string{}
	}

	return s, nil
}

func SaveState(projectRoot string, state State) error {
	if _, err := EnsureLevelupDataDir(projectRoot); err != nil {
		return err
	}

	sort.Strings(state.Enabled)
	bytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	bytes = append(bytes, '\n')

	path := statePath(projectRoot)
	if err := os.WriteFile(path, bytes, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"visor/internal/observability"
)

// Manager handles skill lifecycle: loading, CRUD, discovery, and matching.
type Manager struct {
	baseDir  string
	skills   []*Skill
	mu       sync.RWMutex
	executor *Executor
	log      *observability.Logger
}

func NewManager(baseDir string) *Manager {
	return &Manager{
		baseDir:  baseDir,
		executor: NewExecutor(),
		log:      observability.Component("skills.manager"),
	}
}

// Reload loads (or reloads) all skills from the base directory.
func (m *Manager) Reload() error {
	skills, err := LoadAll(m.baseDir)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.skills = skills
	m.mu.Unlock()
	m.log.Info(nil, "skills loaded", "count", len(skills), "base_dir", m.baseDir)
	return nil
}

// All returns a copy of the current skill list.
func (m *Manager) All() []*Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Skill, len(m.skills))
	copy(out, m.skills)
	return out
}

// Get returns a skill by name, or nil if not found.
func (m *Manager) Get(name string) *Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.skills {
		if s.Manifest.Name == name {
			return s
		}
	}
	return nil
}

// Match returns all skills whose triggers match the given text.
func (m *Manager) Match(text string) []*Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return MatchAll(m.skills, text)
}

// Executor returns the skill executor.
func (m *Manager) Exec() *Executor {
	return m.executor
}

// Describe returns a formatted summary of all loaded skills for prompt injection.
func (m *Manager) Describe() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return describeSkills(m.skills)
}

func describeSkills(skills []*Skill) string {
	if len(skills) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("available visor skills:\n")
	for _, s := range skills {
		b.WriteString(fmt.Sprintf("- %s: %s", s.Manifest.Name, s.Manifest.Description))
		if len(s.Manifest.Triggers) > 0 {
			b.WriteString(fmt.Sprintf(" [triggers: %s]", strings.Join(s.Manifest.Triggers, ", ")))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Create writes a new skill directory with skill.toml and optional script file.
func (m *Manager) Create(action CreateAction) error {
	dir := filepath.Join(m.baseDir, action.Name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("skill %q already exists", action.Name)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create skill dir: %w", err)
	}

	manifest := Manifest{
		Name:         action.Name,
		Description:  action.Description,
		Triggers:     action.Triggers,
		Run:          action.Run,
		Dependencies: action.Dependencies,
		Timeout:      action.Timeout,
	}
	if manifest.Timeout == 0 {
		manifest.Timeout = 30
	}
	if manifest.Run == "" {
		manifest.Run = "bash run.sh"
	}

	if err := writeManifest(filepath.Join(dir, "skill.toml"), manifest); err != nil {
		os.RemoveAll(dir)
		return err
	}

	if action.Script != "" {
		scriptName := extractScriptName(manifest.Run)
		if err := os.WriteFile(filepath.Join(dir, scriptName), []byte(action.Script), 0o755); err != nil {
			os.RemoveAll(dir)
			return fmt.Errorf("write script: %w", err)
		}
	}

	m.log.Info(nil, "skill created", "name", action.Name)
	return m.Reload()
}

// Edit modifies an existing skill's manifest and/or script.
func (m *Manager) Edit(action EditAction) error {
	skill := m.Get(action.Name)
	if skill == nil {
		return fmt.Errorf("skill %q not found", action.Name)
	}

	if action.Description != "" {
		skill.Manifest.Description = action.Description
	}
	if action.Triggers != nil {
		skill.Manifest.Triggers = action.Triggers
	}
	if action.Run != "" {
		skill.Manifest.Run = action.Run
	}
	if action.Timeout > 0 {
		skill.Manifest.Timeout = action.Timeout
	}

	if err := writeManifest(filepath.Join(skill.Dir, "skill.toml"), skill.Manifest); err != nil {
		return err
	}

	if action.Script != "" {
		scriptName := extractScriptName(skill.Manifest.Run)
		if err := os.WriteFile(filepath.Join(skill.Dir, scriptName), []byte(action.Script), 0o755); err != nil {
			return fmt.Errorf("write script: %w", err)
		}
	}

	m.log.Info(nil, "skill edited", "name", action.Name)
	return m.Reload()
}

// Delete removes a skill directory entirely.
func (m *Manager) Delete(name string) error {
	skill := m.Get(name)
	if skill == nil {
		return fmt.Errorf("skill %q not found", name)
	}

	if err := os.RemoveAll(skill.Dir); err != nil {
		return fmt.Errorf("remove skill dir: %w", err)
	}

	m.log.Info(nil, "skill deleted", "name", name)
	return m.Reload()
}

func writeManifest(path string, m Manifest) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create skill.toml: %w", err)
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(m)
}

func extractScriptName(runCmd string) string {
	parts := strings.Fields(runCmd)
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return "run.sh"
}

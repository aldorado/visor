package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/BurntSushi/toml"
)

// Manifest is the skill.toml schema.
type Manifest struct {
	Name         string   `toml:"name"`
	Description  string   `toml:"description"`
	Triggers     []string `toml:"triggers"`      // regex patterns matched against incoming messages
	Run          string   `toml:"run"`            // command to execute (e.g. "python3 run.py", "bash run.sh")
	Dependencies []string `toml:"dependencies"`   // required tools/packages
	LevelUps     []string `toml:"level_ups"`      // required level-ups (e.g. "email-himalaya")
	Timeout      int      `toml:"timeout"`        // execution timeout in seconds (default: 30)
}

// Skill is a loaded, ready-to-match skill.
type Skill struct {
	Manifest Manifest
	Dir      string           // absolute path to the skill directory
	Patterns []*regexp.Regexp // compiled trigger patterns
}

// LoadAll reads all skill directories under baseDir and returns loaded skills.
// Each subdirectory must contain a skill.toml to be recognized as a skill.
// Directories without skill.toml are silently skipped (they may be instruction-only skills for AI backends).
func LoadAll(baseDir string) ([]*Skill, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("skills: read dir %s: %w", baseDir, err)
	}

	var skills []*Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(baseDir, entry.Name())
		manifestPath := filepath.Join(skillDir, "skill.toml")

		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			continue
		}

		skill, err := Load(skillDir)
		if err != nil {
			return nil, fmt.Errorf("skills: load %s: %w", entry.Name(), err)
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

// Load reads a single skill from its directory.
func Load(dir string) (*Skill, error) {
	manifestPath := filepath.Join(dir, "skill.toml")

	var m Manifest
	if _, err := toml.DecodeFile(manifestPath, &m); err != nil {
		return nil, fmt.Errorf("parse skill.toml: %w", err)
	}

	if m.Name == "" {
		return nil, fmt.Errorf("skill.toml missing required field: name")
	}
	if m.Run == "" {
		return nil, fmt.Errorf("skill.toml missing required field: run")
	}
	if m.Timeout == 0 {
		m.Timeout = 30
	}

	var patterns []*regexp.Regexp
	for _, pat := range m.Triggers {
		re, err := regexp.Compile("(?i)" + pat)
		if err != nil {
			return nil, fmt.Errorf("invalid trigger pattern %q: %w", pat, err)
		}
		patterns = append(patterns, re)
	}

	return &Skill{
		Manifest: m,
		Dir:      dir,
		Patterns: patterns,
	}, nil
}

// Match returns true if the message text matches any of the skill's trigger patterns.
func (s *Skill) Match(text string) bool {
	for _, re := range s.Patterns {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

// MatchAll returns all skills that match the given message text.
func MatchAll(skills []*Skill, text string) []*Skill {
	var matched []*Skill
	for _, s := range skills {
		if s.Match(text) {
			matched = append(matched, s)
		}
	}
	return matched
}

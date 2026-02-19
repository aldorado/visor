package skills

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"visor/internal/observability"
)

var importLog = observability.Component("skills.import")

// Import clones a skill from a git repo URL into the manager's base directory.
// The repo must contain a skill.toml at its root.
// If ref is non-empty, it checks out that ref (tag, branch, or commit hash).
func (m *Manager) Import(repoURL, ref string) error {
	// derive skill dir name from repo URL
	name := repoNameFromURL(repoURL)
	if name == "" {
		return fmt.Errorf("cannot derive skill name from URL: %s", repoURL)
	}

	dir := filepath.Join(m.baseDir, name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("skill %q already exists, delete it first to reimport", name)
	}

	// git clone
	args := []string{"clone", "--depth", "1"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, repoURL, dir)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// shallow clone with --branch fails for commit hashes, retry without --branch and checkout
		if ref != "" && strings.Contains(string(output), "not found") {
			return m.importWithCheckout(repoURL, ref, dir)
		}
		os.RemoveAll(dir)
		return fmt.Errorf("git clone failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	// if ref was a branch/tag but we want a specific hash, checkout
	if ref != "" && looksLikeHash(ref) {
		return m.importWithCheckout(repoURL, ref, dir)
	}

	return m.finalizeImport(dir, repoURL, ref)
}

func (m *Manager) importWithCheckout(repoURL, ref, dir string) error {
	os.RemoveAll(dir)

	// full clone (no --depth for hash checkout)
	cloneCmd := exec.Command("git", "clone", repoURL, dir)
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		return fmt.Errorf("git clone failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	checkoutCmd := exec.Command("git", "-C", dir, "checkout", ref)
	if output, err := checkoutCmd.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		return fmt.Errorf("git checkout %s failed: %s: %w", ref, strings.TrimSpace(string(output)), err)
	}

	return m.finalizeImport(dir, repoURL, ref)
}

func (m *Manager) finalizeImport(dir, repoURL, ref string) error {
	// verify skill.toml exists
	manifestPath := filepath.Join(dir, "skill.toml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		os.RemoveAll(dir)
		return fmt.Errorf("imported repo has no skill.toml at root")
	}

	// resolve version from git
	version := ref
	if version == "" {
		version = gitHeadHash(dir)
	} else if !looksLikeHash(ref) {
		// ref was a tag/branch, resolve to hash
		hash := gitHeadHash(dir)
		if hash != "" {
			version = ref + " (" + hash[:minInt(8, len(hash))] + ")"
		}
	}

	// stamp source + version into manifest
	skill, err := Load(dir)
	if err != nil {
		os.RemoveAll(dir)
		return fmt.Errorf("imported skill invalid: %w", err)
	}
	skill.Manifest.Source = repoURL
	skill.Manifest.Version = version

	if err := writeManifest(manifestPath, skill.Manifest); err != nil {
		os.RemoveAll(dir)
		return err
	}

	// check dependencies
	if missing := CheckDependencies(skill.Manifest.Dependencies); len(missing) > 0 {
		importLog.Warn(nil, "imported skill has missing dependencies", "skill", skill.Manifest.Name, "missing", missing)
	}

	importLog.Info(nil, "skill imported", "name", skill.Manifest.Name, "source", repoURL, "version", version)
	return m.Reload()
}

// Update re-imports a skill by pulling the latest from its source repo.
func (m *Manager) Update(name string) error {
	skill := m.Get(name)
	if skill == nil {
		return fmt.Errorf("skill %q not found", name)
	}
	if skill.Manifest.Source == "" {
		return fmt.Errorf("skill %q has no source URL, cannot update", name)
	}

	// git pull in the skill directory
	cmd := exec.Command("git", "-C", skill.Dir, "pull", "--ff-only")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	// update version hash
	hash := gitHeadHash(skill.Dir)
	if hash != "" {
		skill.Manifest.Version = hash
		writeManifest(filepath.Join(skill.Dir, "skill.toml"), skill.Manifest)
	}

	importLog.Info(nil, "skill updated", "name", name, "version", hash)
	return m.Reload()
}

// CheckDependencies verifies that required tools are available on PATH.
// Returns a list of missing dependencies.
func CheckDependencies(deps []string) []string {
	var missing []string
	for _, dep := range deps {
		if _, err := exec.LookPath(dep); err != nil {
			missing = append(missing, dep)
		}
	}
	return missing
}

func repoNameFromURL(url string) string {
	// handle https://github.com/user/repo.git or git@github.com:user/repo.git
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimRight(url, "/")

	// get last path segment
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return ""
	}
	name := parts[len(parts)-1]

	// handle git@...:user/repo format
	if idx := strings.LastIndex(name, ":"); idx >= 0 {
		name = name[idx+1:]
	}

	return name
}

func gitHeadHash(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func looksLikeHash(s string) bool {
	if len(s) < 7 || len(s) > 40 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

package skills

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRepoNameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/user/my-skill.git", "my-skill"},
		{"https://github.com/user/my-skill", "my-skill"},
		{"https://github.com/user/my-skill/", "my-skill"},
		{"git@github.com:user/my-skill.git", "my-skill"},
		{"https://example.com/skills/cool-tool.git", "cool-tool"},
	}
	for _, tc := range tests {
		got := repoNameFromURL(tc.url)
		if got != tc.want {
			t.Errorf("repoNameFromURL(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}

func TestLooksLikeHash(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"abc1234", true},
		{"abc123456789012345678901234567890abcdef0", true},
		{"v1.0.0", false},
		{"main", false},
		{"abc12", false},   // too short
		{"ABC1234", false}, // uppercase
	}
	for _, tc := range tests {
		got := looksLikeHash(tc.s)
		if got != tc.want {
			t.Errorf("looksLikeHash(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

func TestCheckDependencies(t *testing.T) {
	// bash should always exist
	missing := CheckDependencies([]string{"bash"})
	if len(missing) != 0 {
		t.Errorf("bash should not be missing, got %v", missing)
	}

	// nonexistent tool
	missing = CheckDependencies([]string{"bash", "totally-fake-tool-xyz"})
	if len(missing) != 1 || missing[0] != "totally-fake-tool-xyz" {
		t.Errorf("expected [totally-fake-tool-xyz], got %v", missing)
	}
}

func TestCheckDependenciesEmpty(t *testing.T) {
	missing := CheckDependencies(nil)
	if len(missing) != 0 {
		t.Errorf("expected empty, got %v", missing)
	}
}

// TestImportFromLocalGitRepo tests the full import flow using a local git repo.
func TestImportFromLocalGitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// create a fake "remote" git repo with a skill.toml
	remote := t.TempDir()
	runGit(t, remote, "init")
	runGit(t, remote, "config", "user.email", "test@test.com")
	runGit(t, remote, "config", "user.name", "test")
	writeFile(t, filepath.Join(remote, "skill.toml"), `
name = "imported-skill"
description = "a skill from git"
run = "bash run.sh"
dependencies = ["bash"]
`)
	writeScript(t, filepath.Join(remote, "run.sh"), "#!/bin/bash\necho imported")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "init")

	// import into manager
	base := t.TempDir()
	m := NewManager(base)
	m.Reload()

	if err := m.Import(remote, ""); err != nil {
		t.Fatal(err)
	}

	// verify skill loaded
	s := m.Get("imported-skill")
	if s == nil {
		t.Fatal("expected imported-skill to be loaded")
	}
	if s.Manifest.Source != remote {
		t.Errorf("source = %q, want %q", s.Manifest.Source, remote)
	}
	if s.Manifest.Version == "" {
		t.Error("expected version to be set (git hash)")
	}

	// verify script exists
	script, _ := os.ReadFile(filepath.Join(base, filepath.Base(remote), "run.sh"))
	if string(script) != "#!/bin/bash\necho imported" {
		t.Errorf("script = %q", string(script))
	}
}

func TestImportDuplicate(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	remote := t.TempDir()
	runGit(t, remote, "init")
	runGit(t, remote, "config", "user.email", "test@test.com")
	runGit(t, remote, "config", "user.name", "test")
	writeFile(t, filepath.Join(remote, "skill.toml"), "name = \"dup\"\nrun = \"echo ok\"\n")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "init")

	base := t.TempDir()
	m := NewManager(base)
	m.Reload()

	if err := m.Import(remote, ""); err != nil {
		t.Fatal(err)
	}

	// second import should fail
	if err := m.Import(remote, ""); err == nil {
		t.Fatal("expected error for duplicate import")
	}
}

func TestImportNoManifest(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	remote := t.TempDir()
	runGit(t, remote, "init")
	runGit(t, remote, "config", "user.email", "test@test.com")
	runGit(t, remote, "config", "user.name", "test")
	writeFile(t, filepath.Join(remote, "README.md"), "# not a skill")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "init")

	base := t.TempDir()
	m := NewManager(base)
	m.Reload()

	err := m.Import(remote, "")
	if err == nil {
		t.Fatal("expected error for repo without skill.toml")
	}
	if !containsStr(err.Error(), "no skill.toml") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateSkill(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// create remote repo
	remote := t.TempDir()
	runGit(t, remote, "init")
	runGit(t, remote, "config", "user.email", "test@test.com")
	runGit(t, remote, "config", "user.name", "test")
	writeFile(t, filepath.Join(remote, "skill.toml"), `
name = "updatable"
run = "bash run.sh"
`)
	writeScript(t, filepath.Join(remote, "run.sh"), "#!/bin/bash\necho v1")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "v1")

	// import
	base := t.TempDir()
	m := NewManager(base)
	m.Reload()
	if err := m.Import(remote, ""); err != nil {
		t.Fatal(err)
	}

	oldVersion := m.Get("updatable").Manifest.Version

	// add a new commit to remote
	writeScript(t, filepath.Join(remote, "run.sh"), "#!/bin/bash\necho v2")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "v2")

	// update
	if err := m.Update("updatable"); err != nil {
		t.Fatal(err)
	}

	newVersion := m.Get("updatable").Manifest.Version
	if newVersion == oldVersion {
		t.Error("expected version to change after update")
	}

	// verify updated script
	cloneDir := filepath.Join(base, filepath.Base(remote))
	script, _ := os.ReadFile(filepath.Join(cloneDir, "run.sh"))
	if string(script) != "#!/bin/bash\necho v2" {
		t.Errorf("script after update = %q", string(script))
	}
}

func TestUpdateNoSource(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "local", "skill.toml"), `
name = "local"
run = "echo ok"
`)

	m := NewManager(base)
	m.Reload()

	err := m.Update("local")
	if err == nil {
		t.Fatal("expected error for skill without source")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE=2026-01-01T00:00:00+00:00", "GIT_COMMITTER_DATE=2026-01-01T00:00:00+00:00")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %s: %v", args, output, err)
	}
}

package promptsync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSyncCopiesSkillsToPiOnly(t *testing.T) {
	repo := t.TempDir()

	mustWrite(t, filepath.Join(repo, ".pi", "SYSTEM.md"), "system")
	mustWrite(t, filepath.Join(repo, "skills", "alpha", "SKILL.md"), "alpha")
	mustWrite(t, filepath.Join(repo, "skills", "beta", "SKILL.md"), "beta")
	mustWrite(t, filepath.Join(repo, ".pi", "skills", "old", "SKILL.md"), "old")

	if err := Sync(repo); err != nil {
		t.Fatal(err)
	}

	if mustRead(t, filepath.Join(repo, ".pi", "skills", "alpha", "SKILL.md")) != "alpha" {
		t.Fatal("pi alpha not synced")
	}
	if mustRead(t, filepath.Join(repo, ".pi", "skills", "beta", "SKILL.md")) != "beta" {
		t.Fatal("pi beta not synced")
	}
	if _, err := os.Stat(filepath.Join(repo, ".pi", "skills", "old")); !os.IsNotExist(err) {
		t.Fatal("stale pi skill should be removed")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

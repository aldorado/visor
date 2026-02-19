package selfevolve

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initGitRepo creates a temp dir with an initialized git repo and a dummy go module.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s %v", args, out, err)
		}
	}

	// initial commit so HEAD exists
	dummy := filepath.Join(dir, "init.txt")
	os.WriteFile(dummy, []byte("init"), 0o644)
	gitAdd := exec.Command("git", "add", "-A")
	gitAdd.Dir = dir
	gitAdd.CombinedOutput()
	gitCommit := exec.Command("git", "commit", "-m", "initial")
	gitCommit.Dir = dir
	gitCommit.CombinedOutput()

	return dir
}

func TestApplyDisabled(t *testing.T) {
	m := New(Config{Enabled: false, RepoDir: t.TempDir()})
	result, err := m.Apply(context.Background(), Request{CommitMessage: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Committed || result.Built {
		t.Error("expected no-op when disabled")
	}
}

func TestApplyNoChanges(t *testing.T) {
	dir := initGitRepo(t)
	m := New(Config{Enabled: true, RepoDir: dir})
	result, err := m.Apply(context.Background(), Request{CommitMessage: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Committed {
		t.Error("expected no commit when no changes")
	}
}

func TestApplyCommitsChanges(t *testing.T) {
	dir := initGitRepo(t)

	// write a valid go file so build succeeds
	writeGoModule(t, dir)
	writeGoMain(t, dir)

	m := New(Config{Enabled: true, RepoDir: dir})
	m.exitFn = func(code int) {} // prevent actual exit

	result, err := m.Apply(context.Background(), Request{CommitMessage: "test commit"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Committed {
		t.Error("expected committed=true")
	}
	if !result.Built {
		t.Error("expected built=true")
	}

	// verify git log contains our message
	cmd := exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	if got := string(out); !contains(got, "test commit") {
		t.Errorf("git log = %q, want 'test commit' in it", got)
	}
}

func TestApplyBuildFailureRollback(t *testing.T) {
	dir := initGitRepo(t)

	// write invalid Go code that won't compile
	writeGoModule(t, dir)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() { broken }\n"), 0o644)

	m := New(Config{Enabled: true, RepoDir: dir})

	result, err := m.Apply(context.Background(), Request{CommitMessage: "bad code"})
	if err != nil {
		t.Fatal(err)
	}
	if result.BuildErr == "" {
		t.Fatal("expected build error")
	}
	if result.Built {
		t.Error("expected built=false on build failure")
	}

	// verify the commit was rolled back
	cmd := exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	if got := string(out); contains(got, "bad code") {
		t.Errorf("commit should have been rolled back, but git log shows: %q", got)
	}

	// verify the changes are still present (unstaged)
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, _ = cmd.CombinedOutput()
	if got := string(out); got == "" {
		t.Error("expected uncommitted changes after rollback")
	}
}

func TestApplyDefaultCommitMessage(t *testing.T) {
	dir := initGitRepo(t)
	writeGoModule(t, dir)
	writeGoMain(t, dir)

	m := New(Config{Enabled: true, RepoDir: dir})
	m.exitFn = func(code int) {}

	result, err := m.Apply(context.Background(), Request{CommitMessage: ""})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Committed {
		t.Fatal("expected commit")
	}

	cmd := exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	if got := string(out); !contains(got, "self-evolution update") {
		t.Errorf("expected default commit message, got: %q", got)
	}
}

func TestReplaceBinarySameDevice(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "new")
	dst := filepath.Join(dir, "old")

	os.WriteFile(src, []byte("new-content"), 0o755)
	os.WriteFile(dst, []byte("old-content"), 0o644)

	if err := replaceBinary(src, dst); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(dst)
	if string(data) != "new-content" {
		t.Errorf("dst content = %q, want 'new-content'", data)
	}

	info, _ := os.Stat(dst)
	if info.Mode().Perm()&0o111 == 0 {
		t.Error("expected executable permission on dst")
	}
}

func TestRestartExitCode(t *testing.T) {
	var exitCode int
	m := New(Config{Enabled: true})
	m.exitFn = func(code int) { exitCode = code }

	m.Restart()
	if exitCode != ExitCodeRestart {
		t.Errorf("exit code = %d, want %d", exitCode, ExitCodeRestart)
	}
}

func TestExitCodeRestartValue(t *testing.T) {
	if ExitCodeRestart != 42 {
		t.Errorf("ExitCodeRestart = %d, want 42", ExitCodeRestart)
	}
}

// helpers

func writeGoModule(t *testing.T, dir string) {
	t.Helper()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.21\n"), 0o644)
}

func writeGoMain(t *testing.T, dir string) {
	t.Helper()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644)
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

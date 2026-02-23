package selfevolve

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// initGitRepo creates a temp dir with an initialized git repo.
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

func TestApplyCommitsAndBuilds(t *testing.T) {
	dir := initGitRepo(t)
	writeGoModule(t, dir)
	writeGoMain(t, dir)

	m := New(Config{Enabled: true, RepoDir: dir, DataDir: filepath.Join(dir, "data")})
	m.exitFn = func(code int) {}

	result, err := m.Apply(context.Background(), Request{CommitMessage: "test commit", Backend: "pi"})
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
	if !strings.Contains(string(out), "test commit") {
		t.Errorf("git log = %q, want 'test commit' in it", out)
	}
}

func TestApplyBuildFailureRollback(t *testing.T) {
	dir := initGitRepo(t)
	writeGoModule(t, dir)
	// code that passes vet but fails to build: missing import
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nimport \"missingpkg123\"\nfunc main() { missingpkg123.Do() }\n"), 0o644)

	m := New(Config{Enabled: true, RepoDir: dir, DataDir: filepath.Join(dir, "data")})

	result, err := m.Apply(context.Background(), Request{CommitMessage: "bad code"})
	if err != nil {
		t.Fatal(err)
	}
	// could fail at vet or build stage — either way, should be rolled back
	if result.BuildErr == "" && result.VetErr == "" {
		t.Fatal("expected build or vet error")
	}
	if result.Built {
		t.Error("expected built=false on failure")
	}

	// verify rollback
	cmd := exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	if strings.Contains(string(out), "bad code") {
		t.Errorf("commit should have been rolled back, but git log shows: %q", out)
	}
}

func TestApplyVetFailureRollback(t *testing.T) {
	dir := initGitRepo(t)
	writeGoModule(t, dir)
	// code that compiles but fails go vet (unreachable code after return)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

import "fmt"

func main() {
	fmt.Printf("%d", "not-a-number")
}
`), 0o644)

	m := New(Config{Enabled: true, RepoDir: dir, DataDir: filepath.Join(dir, "data")})

	result, err := m.Apply(context.Background(), Request{CommitMessage: "vet-fail"})
	if err != nil {
		t.Fatal(err)
	}
	if result.VetErr == "" {
		t.Fatal("expected vet error")
	}
	if result.Built {
		t.Error("expected built=false on vet failure")
	}

	// verify rollback
	cmd := exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	if strings.Contains(string(out), "vet-fail") {
		t.Errorf("commit should have been rolled back, but git log shows: %q", out)
	}
}

func TestApplyDefaultCommitMessage(t *testing.T) {
	dir := initGitRepo(t)
	writeGoModule(t, dir)
	writeGoMain(t, dir)

	m := New(Config{Enabled: true, RepoDir: dir, DataDir: filepath.Join(dir, "data")})
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
	if !strings.Contains(string(out), "self-evolution update") {
		t.Errorf("expected default commit message, got: %q", out)
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

// --- M8-I3: safety rails tests ---

func TestBackupBinaryRotation(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "visor")

	// create a "binary"
	os.WriteFile(bin, []byte("v1"), 0o755)
	if err := backupBinary(bin, 3); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(bin + ".bak.1")
	if string(data) != "v1" {
		t.Errorf("bak.1 = %q, want 'v1'", data)
	}

	// second backup: v1 should shift to .bak.2
	os.WriteFile(bin, []byte("v2"), 0o755)
	backupBinary(bin, 3)
	data, _ = os.ReadFile(bin + ".bak.1")
	if string(data) != "v2" {
		t.Errorf("bak.1 = %q, want 'v2'", data)
	}
	data, _ = os.ReadFile(bin + ".bak.2")
	if string(data) != "v1" {
		t.Errorf("bak.2 = %q, want 'v1'", data)
	}

	// third and fourth backup — should keep max 3
	os.WriteFile(bin, []byte("v3"), 0o755)
	backupBinary(bin, 3)
	os.WriteFile(bin, []byte("v4"), 0o755)
	backupBinary(bin, 3)

	// .bak.1=v4, .bak.2=v3, .bak.3=v2, .bak.4 should NOT exist (pruned)
	data, _ = os.ReadFile(bin + ".bak.1")
	if string(data) != "v4" {
		t.Errorf("bak.1 = %q, want 'v4'", data)
	}
	if _, err := os.Stat(bin + ".bak.4"); err == nil {
		t.Error("bak.4 should have been pruned")
	}
}

func TestLatestBackup(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "visor")

	// no backups
	if got := latestBackup(bin); got != "" {
		t.Errorf("expected empty, got %q", got)
	}

	// create .bak.1
	os.WriteFile(bin+".bak.1", []byte("backup"), 0o755)
	if got := latestBackup(bin); got != bin+".bak.1" {
		t.Errorf("expected .bak.1, got %q", got)
	}
}

func TestShouldAutoRollback(t *testing.T) {
	m := New(Config{Enabled: true})
	m.startedAt = time.Now()
	m.nowFn = func() time.Time { return m.startedAt.Add(10 * time.Second) }

	if !m.ShouldAutoRollback() {
		t.Error("expected true within crash window")
	}

	m.nowFn = func() time.Time { return m.startedAt.Add(31 * time.Second) }
	if m.ShouldAutoRollback() {
		t.Error("expected false after crash window")
	}
}

func TestWriteChangelog(t *testing.T) {
	dir := t.TempDir()
	m := New(Config{Enabled: true, DataDir: dir})
	m.nowFn = func() time.Time { return time.Date(2026, 2, 19, 12, 0, 0, 0, time.UTC) }

	m.writeChangelog(Request{CommitMessage: "test update", ChatID: 123, Backend: "pi"})

	logFile := filepath.Join(dir, "selfevolve.log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "backend=pi") {
		t.Errorf("changelog missing backend: %q", got)
	}
	if !strings.Contains(got, "chat_id=123") {
		t.Errorf("changelog missing chat_id: %q", got)
	}
	if !strings.Contains(got, `message="test update"`) {
		t.Errorf("changelog missing message: %q", got)
	}

	// second write should append
	m.writeChangelog(Request{CommitMessage: "second", Backend: "pi"})
	data, _ = os.ReadFile(logFile)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 changelog lines, got %d", len(lines))
	}
}

func TestConfigDisabledIsAlreadyWired(t *testing.T) {
	// SELF_EVOLUTION_ENABLED=false should result in Enabled=false
	m := New(Config{Enabled: false})
	if m.Enabled() {
		t.Error("expected disabled")
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

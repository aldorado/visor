package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeScript(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSkill(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "skill.toml"), `
name = "test-skill"
description = "a test skill"
triggers = ["^hello", "greet"]
run = "bash run.sh"
timeout = 10
`)

	skill, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if skill.Manifest.Name != "test-skill" {
		t.Errorf("name = %q, want %q", skill.Manifest.Name, "test-skill")
	}
	if skill.Manifest.Timeout != 10 {
		t.Errorf("timeout = %d, want 10", skill.Manifest.Timeout)
	}
	if len(skill.Patterns) != 2 {
		t.Errorf("patterns = %d, want 2", len(skill.Patterns))
	}
}

func TestLoadSkillDefaults(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "skill.toml"), `
name = "minimal"
run = "echo ok"
`)

	skill, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if skill.Manifest.Timeout != 30 {
		t.Errorf("default timeout = %d, want 30", skill.Manifest.Timeout)
	}
}

func TestLoadSkillMissingName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "skill.toml"), `
run = "echo ok"
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestLoadSkillMissingRun(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "skill.toml"), `
name = "no-run"
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for missing run")
	}
}

func TestLoadSkillBadTrigger(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "skill.toml"), `
name = "bad-trigger"
run = "echo ok"
triggers = ["[invalid"]
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid regex trigger")
	}
}

func TestLoadAll(t *testing.T) {
	base := t.TempDir()

	// skill with manifest
	writeFile(t, filepath.Join(base, "greet", "skill.toml"), `
name = "greet"
run = "echo hello"
triggers = ["^hi$"]
`)

	// skill without manifest (instruction-only, should be skipped)
	writeFile(t, filepath.Join(base, "ai-only", "SKILL.md"), "# AI instruction skill")

	// not a directory
	writeFile(t, filepath.Join(base, "random.txt"), "ignored")

	skills, err := LoadAll(base)
	if err != nil {
		t.Fatal(err)
	}

	if len(skills) != 1 {
		t.Fatalf("loaded %d skills, want 1", len(skills))
	}
	if skills[0].Manifest.Name != "greet" {
		t.Errorf("skill name = %q, want %q", skills[0].Manifest.Name, "greet")
	}
}

func TestLoadAllMissingDir(t *testing.T) {
	skills, err := LoadAll("/nonexistent/path")
	if err != nil {
		t.Fatal(err)
	}
	if skills != nil {
		t.Errorf("expected nil skills for missing dir, got %d", len(skills))
	}
}

func TestMatch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "skill.toml"), `
name = "todo"
run = "echo ok"
triggers = ["^todo\\b", "^to-do\\b"]
`)

	skill, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		input string
		want  bool
	}{
		{"todo buy milk", true},
		{"Todo Buy Milk", true}, // case insensitive
		{"to-do clean house", true},
		{"my todo list", false},
		{"nothing", false},
	}

	for _, tc := range tests {
		got := skill.Match(tc.input)
		if got != tc.want {
			t.Errorf("Match(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestMatchAll(t *testing.T) {
	base := t.TempDir()

	writeFile(t, filepath.Join(base, "a", "skill.toml"), `
name = "a"
run = "echo a"
triggers = ["hello"]
`)
	writeFile(t, filepath.Join(base, "b", "skill.toml"), `
name = "b"
run = "echo b"
triggers = ["world"]
`)
	writeFile(t, filepath.Join(base, "c", "skill.toml"), `
name = "c"
run = "echo c"
triggers = ["hello.*world"]
`)

	skills, err := LoadAll(base)
	if err != nil {
		t.Fatal(err)
	}

	matched := MatchAll(skills, "hello world")
	// a and c should match, b should match too ("world" is in the text)
	if len(matched) != 3 {
		names := make([]string, len(matched))
		for i, s := range matched {
			names[i] = s.Manifest.Name
		}
		t.Errorf("matched %d skills %v, want 3", len(matched), names)
	}
}

func TestExecuteSkill(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "skill.toml"), `
name = "echo-test"
run = "bash run.sh"
timeout = 5
`)
	writeScript(t, filepath.Join(dir, "run.sh"), `#!/bin/bash
echo "hello from skill"
echo "msg: $VISOR_USER_MESSAGE"
`)

	skill, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	exec := NewExecutor()
	result, err := exec.Run(context.Background(), skill, Context{
		UserMessage: "test message",
		ChatID:      "12345",
		MessageType: "text",
		Platform:    "telegram",
		DataDir:     "/tmp/data",
		SkillDir:    dir,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
	if got := result.Stdout; got != "hello from skill\nmsg: test message\n" {
		t.Errorf("stdout = %q", got)
	}
}

func TestExecuteSkillStdin(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "skill.toml"), `
name = "stdin-test"
run = "bash run.sh"
timeout = 5
`)
	// read JSON context from stdin
	writeScript(t, filepath.Join(dir, "run.sh"), `#!/bin/bash
cat | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['user_message'])"
`)

	skill, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	exec := NewExecutor()
	result, err := exec.Run(context.Background(), skill, Context{
		UserMessage: "hello from stdin",
		ChatID:      "999",
		MessageType: "text",
		Platform:    "telegram",
		DataDir:     "/tmp",
		SkillDir:    dir,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", result.ExitCode, result.Stderr)
	}
	expected := "hello from stdin\n"
	if result.Stdout != expected {
		t.Errorf("stdout = %q, want %q", result.Stdout, expected)
	}
}

func TestExecuteSkillNonZeroExit(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "skill.toml"), `
name = "fail-test"
run = "bash run.sh"
timeout = 5
`)
	writeScript(t, filepath.Join(dir, "run.sh"), `#!/bin/bash
echo "error output" >&2
exit 1
`)

	skill, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	exec := NewExecutor()
	result, err := exec.Run(context.Background(), skill, Context{
		UserMessage: "test",
		SkillDir:    dir,
	})
	// non-zero exit is not a Go error — it's captured in result
	if err != nil {
		t.Fatal(err)
	}

	if result.ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", result.ExitCode)
	}
	if result.Stderr != "error output\n" {
		t.Errorf("stderr = %q", result.Stderr)
	}
}

func TestExecuteSkillTimeout(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "skill.toml"), `
name = "slow-test"
run = "bash run.sh"
timeout = 1
`)
	writeScript(t, filepath.Join(dir, "run.sh"), `#!/bin/bash
sleep 10
echo "should not reach"
`)

	skill, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	exec := NewExecutor()
	_, err = exec.Run(context.Background(), skill, Context{
		UserMessage: "test",
		SkillDir:    dir,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

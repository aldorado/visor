package levelup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBuildComposeConfigArgs(t *testing.T) {
	assembly := &ComposeAssembly{
		ProjectRoot: "/root/code/visor",
		Files:       []string{"/root/code/visor/docker-compose.yml", "/root/code/visor/docker-compose.levelup.obsidian.yml"},
	}
	envFiles := []string{"/root/code/visor/.env", "/root/code/visor/.levelup.env"}

	args := BuildComposeConfigArgs(assembly, envFiles)
	expected := []string{
		"compose",
		"-f", "/root/code/visor/docker-compose.yml",
		"-f", "/root/code/visor/docker-compose.levelup.obsidian.yml",
		"--project-directory", "/root/code/visor",
		"--env-file", "/root/code/visor/.env",
		"--env-file", "/root/code/visor/.levelup.env",
		"config",
	}

	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("unexpected args\nwant: %#v\ngot:  %#v", expected, args)
	}
}

func TestExistingEnvFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("A=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".levelup.env"), []byte("B=2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := existingEnvFiles(root)
	expected := []string{filepath.Join(root, ".env"), filepath.Join(root, ".levelup.env")}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected env files\nwant: %#v\ngot:  %#v", expected, got)
	}
}

func TestMapToEnvSorted(t *testing.T) {
	got := mapToEnv(map[string]string{"Z": "3", "A": "1", "M": "2"})
	expected := []string{"A=1", "M=2", "Z=3"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected env\nwant: %#v\ngot:  %#v", expected, got)
	}
}

func TestValidateComposeConfigRunnerError(t *testing.T) {
	orig := composeRunner
	defer func() { composeRunner = orig }()

	composeRunner = func(ctx context.Context, projectRoot string, args []string, env map[string]string) ([]byte, error) {
		return []byte("bad compose"), errors.New("exit status 1")
	}

	assembly := &ComposeAssembly{ProjectRoot: t.TempDir(), Files: []string{"/tmp/base.yml"}}
	err := ValidateComposeConfig(context.Background(), assembly, map[string]string{"PATH": os.Getenv("PATH")})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad compose") {
		t.Fatalf("expected stderr in error: %v", err)
	}
}

func TestValidateComposeConfigRunnerSuccess(t *testing.T) {
	orig := composeRunner
	defer func() { composeRunner = orig }()

	composeRunner = func(ctx context.Context, projectRoot string, args []string, env map[string]string) ([]byte, error) {
		return []byte("services:{}"), nil
	}

	assembly := &ComposeAssembly{ProjectRoot: t.TempDir(), Files: []string{"/tmp/base.yml"}}
	if err := ValidateComposeConfig(context.Background(), assembly, map[string]string{"PATH": os.Getenv("PATH")}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

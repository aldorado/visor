package levelup

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBuildComposeAssembly(t *testing.T) {
	root := t.TempDir()
	infraDir := filepath.Join(root, "infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}

	base := filepath.Join(infraDir, "docker-compose.yml")
	overlay1 := filepath.Join(infraDir, "docker-compose.levelup.email-himalaya.yml")
	overlay2 := filepath.Join(infraDir, "docker-compose.levelup.obsidian.yml")

	for _, f := range []string{base, overlay1, overlay2} {
		if err := os.WriteFile(f, []byte("services: {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	assembly, err := BuildComposeAssembly(root, "infra/docker-compose.yml", []string{
		"docker-compose.levelup.email-himalaya.yml",
		"docker-compose.levelup.obsidian.yml",
		"docker-compose.levelup.email-himalaya.yml", // duplicate should be ignored
	})
	if err != nil {
		t.Fatalf("BuildComposeAssembly error: %v", err)
	}

	if assembly.BaseFile != base {
		t.Fatalf("unexpected base file: %s", assembly.BaseFile)
	}

	expectedOverlays := []string{overlay1, overlay2}
	if !reflect.DeepEqual(assembly.OverlayFiles, expectedOverlays) {
		t.Fatalf("unexpected overlays\nwant: %#v\ngot:  %#v", expectedOverlays, assembly.OverlayFiles)
	}

	expectedFiles := []string{base, overlay1, overlay2}
	if !reflect.DeepEqual(assembly.Files, expectedFiles) {
		t.Fatalf("unexpected files\nwant: %#v\ngot:  %#v", expectedFiles, assembly.Files)
	}
}

func TestBuildComposeAssemblyMissingBase(t *testing.T) {
	root := t.TempDir()
	_, err := BuildComposeAssembly(root, "docker-compose.yml", nil)
	if err == nil {
		t.Fatal("expected missing base file error")
	}
	if !strings.Contains(err.Error(), "required file missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildComposeFileArgs(t *testing.T) {
	args := BuildComposeFileArgs([]string{"/tmp/base.yml", "/tmp/overlay.yml"})
	expected := []string{"-f", "/tmp/base.yml", "-f", "/tmp/overlay.yml"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("unexpected args\nwant: %#v\ngot:  %#v", expected, args)
	}
}

package forgejo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadToken_missing(t *testing.T) {
	tok, err := ReadToken(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if tok != "" {
		t.Fatalf("expected empty token, got %q", tok)
	}
}

func TestReadToken_present(t *testing.T) {
	dir := t.TempDir()
	tokenDir := filepath.Join(dir, "levelups", "forgejo")
	if err := os.MkdirAll(tokenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tokenDir, "visor-push.token"), []byte("abc123\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tok, err := ReadToken(dir)
	if err != nil {
		t.Fatal(err)
	}
	if tok != "abc123" {
		t.Fatalf("expected abc123, got %q", tok)
	}
}

func TestBuildRemoteURL(t *testing.T) {
	got := buildRemoteURL("visor", "tok123", "3002", "myrepo")
	want := "http://visor:tok123@localhost:3002/visor/myrepo.git"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

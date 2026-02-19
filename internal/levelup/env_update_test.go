package levelup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateLevelupEnvSetAndUnset(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".levelup.env")
	if err := os.WriteFile(path, []byte("A=1\nB=2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := UpdateLevelupEnv(root, map[string]string{"B": "3", "C": "x"}, []string{"A"})
	if err != nil {
		t.Fatal(err)
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(bytes)
	want := "B=3\nC=x\n"
	if got != want {
		t.Fatalf(".levelup.env\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestUpdateLevelupEnvInvalidKey(t *testing.T) {
	root := t.TempDir()
	err := UpdateLevelupEnv(root, map[string]string{"BAD-KEY": "1"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid env key") {
		t.Fatalf("err=%v", err)
	}
}

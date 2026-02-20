package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateDotEnv(t *testing.T) {
	root := t.TempDir()
	if err := UpdateDotEnv(root, map[string]string{"A": "1"}, nil); err != nil {
		t.Fatal(err)
	}
	bytes, err := os.ReadFile(filepath.Join(root, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if string(bytes) != "A=1\n" {
		t.Fatalf("unexpected content: %q", string(bytes))
	}
}

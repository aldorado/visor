package levelup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadLayeredEnv(t *testing.T) {
	t.Setenv("PROC_OVERRIDE", "process")
	t.Setenv("SHARED", "from-process")

	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, ".env"), []byte("SHARED=from-env\nONLY_ENV=yes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".levelup.env"), []byte("SHARED=from-levelup\nONLY_LEVELUP=yes\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	env, err := LoadLayeredEnv(tmp)
	if err != nil {
		t.Fatalf("LoadLayeredEnv error: %v", err)
	}

	if env["ONLY_ENV"] != "yes" {
		t.Fatalf("expected ONLY_ENV from .env")
	}
	if env["ONLY_LEVELUP"] != "yes" {
		t.Fatalf("expected ONLY_LEVELUP from .levelup.env")
	}
	if env["SHARED"] != "from-process" {
		t.Fatalf("expected process env to override file layers, got %q", env["SHARED"])
	}
}

func TestValidateRequiredEnv(t *testing.T) {
	env := map[string]string{
		"A": "ok",
		"B": "   ",
	}

	err := ValidateRequiredEnv(env, []string{"A", "B", "C"})
	if err == nil {
		t.Fatal("expected error for missing keys")
	}

	msg := err.Error()
	for _, key := range []string{"B", "C"} {
		if !strings.Contains(msg, key) {
			t.Fatalf("expected missing key %s in error: %s", key, msg)
		}
	}
}

func TestParseEnvFileInvalidLine(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".levelup.env")
	if err := os.WriteFile(path, []byte("NO_EQUALS\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := parseEnvFile(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

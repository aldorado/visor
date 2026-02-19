package levelup

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestValidateObsidianMounts(t *testing.T) {
	root := t.TempDir()
	env := map[string]string{
		"OBSIDIAN_CONFIG_PATH": filepath.Join(root, "config"),
		"OBSIDIAN_VAULT_PATH":  filepath.Join(root, "vault"),
	}
	if err := ValidateObsidianMounts(env); err != nil {
		t.Fatalf("validate mounts: %v", err)
	}
}

func TestCheckObsidianReachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := CheckObsidianReachable(context.Background(), srv.URL); err != nil {
		t.Fatalf("reachable check failed: %v", err)
	}
}

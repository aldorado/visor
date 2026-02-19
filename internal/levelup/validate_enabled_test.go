package levelup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateEnabledWithObsidianMounts(t *testing.T) {
	root := t.TempDir()

	// manifests
	mkManifest := func(name, compose string, required []string) {
		dir := filepath.Join(root, "levelups", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		content := "name = \"" + name + "\"\ncompose_file = \"" + compose + "\"\nrequired_env = ["
		for i, k := range required {
			if i > 0 {
				content += ","
			}
			content += "\"" + k + "\""
		}
		content += "]\n"
		if err := os.WriteFile(filepath.Join(dir, "levelup.toml"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mkManifest("obsidian", "docker-compose.levelup.obsidian.yml", []string{"OBSIDIAN_CONFIG_PATH", "OBSIDIAN_VAULT_PATH", "OBSIDIAN_HTTP_PORT", "OBSIDIAN_HTTPS_PORT", "TZ", "OBSIDIAN_PUID", "OBSIDIAN_PGID"})

	base := filepath.Join(root, "docker-compose.yml")
	overlay := filepath.Join(root, "docker-compose.levelup.obsidian.yml")
	if err := os.WriteFile(base, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overlay, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SaveState(root, State{Enabled: []string{"obsidian"}}); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, ".levelup.env"), []byte("TZ=Europe/Vienna\nOBSIDIAN_PUID=1000\nOBSIDIAN_PGID=1000\nOBSIDIAN_CONFIG_PATH="+filepath.Join(root, "cfg")+"\nOBSIDIAN_VAULT_PATH="+filepath.Join(root, "vault")+"\nOBSIDIAN_HTTP_PORT=3010\nOBSIDIAN_HTTPS_PORT=3011\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig := composeRunner
	defer func() { composeRunner = orig }()
	composeRunner = func(ctx context.Context, projectRoot string, args []string, env map[string]string) ([]byte, error) {
		return []byte("ok"), nil
	}

	if err := ValidateEnabled(context.Background(), root, "docker-compose.yml"); err != nil {
		t.Fatalf("validate enabled: %v", err)
	}
}

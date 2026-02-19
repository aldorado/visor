package levelup

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func ValidateObsidianMounts(env map[string]string) error {
	configPath := env["OBSIDIAN_CONFIG_PATH"]
	vaultPath := env["OBSIDIAN_VAULT_PATH"]
	if configPath == "" || vaultPath == "" {
		return fmt.Errorf("OBSIDIAN_CONFIG_PATH and OBSIDIAN_VAULT_PATH are required")
	}

	for _, p := range []string{configPath, vaultPath} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", p, err)
		}
		testFile := filepath.Join(p, ".visor-write-test")
		if err := os.WriteFile(testFile, []byte("ok"), 0o644); err != nil {
			return fmt.Errorf("write test %s: %w", testFile, err)
		}
		_ = os.Remove(testFile)
	}

	return nil
}

func CheckObsidianReachable(ctx context.Context, baseURL string) error {
	if baseURL == "" {
		return fmt.Errorf("obsidian url is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("obsidian request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 399 {
		return fmt.Errorf("obsidian returned status %d", resp.StatusCode)
	}
	return nil
}

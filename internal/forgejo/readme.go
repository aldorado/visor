package forgejo

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// EnsureReadme checks if README.md exists in the Forgejo repo for repoDir
// and creates a minimal one if it doesn't. No-op if token is unavailable or
// Forgejo is unreachable.
func EnsureReadme(ctx context.Context, repoDir, dataDir, adminUser, hostPort string) error {
	token, err := ReadToken(dataDir)
	if err != nil || token == "" {
		return nil
	}

	repoName := filepath.Base(repoDir)
	apiBase := fmt.Sprintf("http://localhost:%s/api/v1", hostPort)
	contentsURL := fmt.Sprintf("%s/repos/%s/%s/contents/README.md", apiBase, adminUser, repoName)

	// check if README exists
	req, err := http.NewRequestWithContext(ctx, "GET", contentsURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "token "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil // forgejo unreachable
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		return nil // already exists (or other error — skip)
	}

	// create README
	content := buildReadme(repoDir, repoName)
	body := map[string]any{
		"message": "init: add README",
		"content": base64.StdEncoding.EncodeToString([]byte(content)),
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err = http.NewRequestWithContext(ctx, "POST", contentsURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil // forgejo unreachable — non-fatal
	}
	resp.Body.Close()
	return nil
}

func buildReadme(repoDir, repoName string) string {
	var sb strings.Builder
	sb.WriteString("# " + repoName + "\n\n")
	sb.WriteString("Visor-authored project.\n")

	// include forge blueprint link if one exists
	forgeFiles, _ := filepath.Glob(filepath.Join(repoDir, "*.forge.md"))
	if len(forgeFiles) > 0 {
		forge := filepath.Base(forgeFiles[0])
		sb.WriteString(fmt.Sprintf("\nForge blueprint: [%s](%s)\n", forge, forge))
	}

	// include current status from README if one already exists locally
	localReadme := filepath.Join(repoDir, "README.md")
	if _, err := os.Stat(localReadme); err == nil {
		// local README exists — don't overwrite with a stub; the API call above
		// would have returned 200 if forgejo already has it, so this case
		// means the local file exists but forgejo doesn't have it yet (first push).
		// Re-read and use local content instead.
		if data, err := os.ReadFile(localReadme); err == nil {
			return string(data)
		}
	}

	return sb.String()
}

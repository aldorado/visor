package forgejo

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"visor/internal/observability"
)

const remoteName = "forgejo"

// TokenPath returns the expected path of the auto-generated push token file.
func TokenPath(dataDir string) string {
	return filepath.Join(dataDir, "levelups", "forgejo", "visor-push.token")
}

// ReadToken reads the Forgejo push token from the token file.
// Returns ("", nil) if the file does not exist (forgejo not yet bootstrapped).
func ReadToken(dataDir string) (string, error) {
	b, err := os.ReadFile(TokenPath(dataDir))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read forgejo token: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}

// SyncRemote adds or removes the "forgejo" git remote on repoDir.
//
// enable=true: sets remote URL using the token file and env values;
// no-op (no error) if the token file does not exist yet.
//
// enable=false: removes the remote (best effort, ignores errors).
func SyncRemote(ctx context.Context, repoDir, dataDir, adminUser, hostPort string, enable bool) error {
	if !enable {
		_ = runGit(ctx, repoDir, "remote", "remove", remoteName) // best effort
		return nil
	}

	token, err := ReadToken(dataDir)
	if err != nil {
		return err
	}
	if token == "" {
		// token not yet written (forgejo not started yet) — remote will be set on first push
		return nil
	}

	repoName := filepath.Base(repoDir)
	url := buildRemoteURL(adminUser, token, hostPort, repoName)

	if remoteExists(ctx, repoDir) {
		return runGit(ctx, repoDir, "remote", "set-url", remoteName, url)
	}
	return runGit(ctx, repoDir, "remote", "add", remoteName, url)
}

// PushBackground pushes to the "forgejo" remote in a background goroutine.
// It is a no-op if the remote is not configured. Errors are logged as warnings.
func PushBackground(ctx context.Context, repoDir string, log *observability.Logger) {
	go func() {
		if !remoteExists(context.Background(), repoDir) {
			return
		}
		if err := runGit(context.Background(), repoDir, "push", remoteName, "HEAD:main"); err != nil {
			log.Warn(ctx, "forgejo push failed (non-blocking)", "repo", repoDir, "error", err.Error())
		} else {
			log.Info(ctx, "forgejo push succeeded", "repo", repoDir)
		}
	}()
}

func buildRemoteURL(adminUser, token, hostPort, repoName string) string {
	return fmt.Sprintf("http://%s:%s@localhost:%s/%s/%s.git",
		adminUser, token, hostPort, adminUser, repoName)
}

func remoteExists(ctx context.Context, repoDir string) bool {
	err := runGit(ctx, repoDir, "remote", "get-url", remoteName)
	return err == nil
}

func runGit(ctx context.Context, repoDir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoDir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(out.String()))
	}
	return nil
}

package selfevolve

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

// ExitCodeRestart is the exit code that signals the supervisor to restart
// with the new binary. Any process manager (systemd, supervisor script)
// should watch for this code.
const ExitCodeRestart = 42

type Config struct {
	Enabled bool
	RepoDir string
	Push    bool
}

type Request struct {
	CommitMessage string
	ChatID        int64
}

// Result describes what happened during Apply.
type Result struct {
	Committed bool
	Built     bool
	BuildErr  string // non-empty if build failed (commit was rolled back)
}

type Manager struct {
	cfg     Config
	exitFn  func(code int) // overridable for testing (defaults to os.Exit)
	log     *observability.Logger
}

func New(cfg Config) *Manager {
	if cfg.RepoDir == "" {
		cfg.RepoDir = "."
	}
	return &Manager{
		cfg:    cfg,
		exitFn: os.Exit,
		log:    observability.Component("selfevolve"),
	}
}

func (m *Manager) Enabled() bool { return m.cfg.Enabled }

// Apply runs the full self-evolution pipeline: commit → build → replace binary.
// If the build fails, it rolls back the commit and returns the error in Result.
func (m *Manager) Apply(ctx context.Context, req Request) (Result, error) {
	if !m.cfg.Enabled {
		return Result{}, nil
	}
	if strings.TrimSpace(req.CommitMessage) == "" {
		req.CommitMessage = "self-evolution update"
	}

	changed, err := hasGitChanges(ctx, m.cfg.RepoDir)
	if err != nil {
		return Result{}, err
	}
	if !changed {
		m.log.Info(ctx, "self-evolve skipped: no git changes")
		return Result{}, nil
	}

	// step 1: commit
	if _, err := run(ctx, m.cfg.RepoDir, "git", "add", "-A"); err != nil {
		return Result{}, fmt.Errorf("git add: %w", err)
	}
	if _, err := run(ctx, m.cfg.RepoDir, "git", "commit", "-m", req.CommitMessage); err != nil {
		return Result{}, fmt.Errorf("git commit: %w", err)
	}
	m.log.Info(ctx, "self-evolve committed", "message", req.CommitMessage)

	// step 2: build
	newBinary := filepath.Join(m.cfg.RepoDir, "visor-new")
	buildOut, buildErr := run(ctx, m.cfg.RepoDir, "go", "build", "-o", newBinary, ".")
	if buildErr != nil {
		m.log.Error(ctx, "self-evolve build failed, rolling back", "error", buildErr.Error(), "output", truncate(buildOut, 500))
		// rollback the commit
		if _, rbErr := run(ctx, m.cfg.RepoDir, "git", "reset", "HEAD~1"); rbErr != nil {
			m.log.Error(ctx, "rollback failed", "error", rbErr.Error())
		} else {
			m.log.Info(ctx, "self-evolve commit rolled back")
		}
		return Result{Committed: true, BuildErr: buildErr.Error()}, nil
	}
	m.log.Info(ctx, "self-evolve build succeeded", "binary", newBinary)

	// step 3: push (if enabled)
	if m.cfg.Push {
		if _, err := run(ctx, m.cfg.RepoDir, "git", "push"); err != nil {
			return Result{Committed: true, Built: true}, fmt.Errorf("git push: %w", err)
		}
		m.log.Info(ctx, "self-evolve pushed")
	}

	// step 4: replace current binary and signal restart
	currentBinary, err := os.Executable()
	if err != nil {
		return Result{Committed: true, Built: true}, fmt.Errorf("find current binary: %w", err)
	}
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return Result{Committed: true, Built: true}, fmt.Errorf("resolve binary symlink: %w", err)
	}

	if err := replaceBinary(newBinary, currentBinary); err != nil {
		return Result{Committed: true, Built: true}, fmt.Errorf("replace binary: %w", err)
	}
	m.log.Info(ctx, "self-evolve binary replaced", "path", currentBinary)

	return Result{Committed: true, Built: true}, nil
}

// Restart exits with ExitCodeRestart, signaling the supervisor to respawn.
// Call this after Apply returns successfully with Built=true.
func (m *Manager) Restart() {
	m.log.Info(nil, "self-evolve restarting", "exit_code", ExitCodeRestart)
	m.exitFn(ExitCodeRestart)
}

// replaceBinary atomically replaces dst with src via rename.
// Falls back to copy if rename fails (cross-device).
func replaceBinary(src, dst string) error {
	// try atomic rename first
	if err := os.Rename(src, dst); err == nil {
		return os.Chmod(dst, 0o755)
	}

	// cross-device fallback: read + write + remove
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read new binary: %w", err)
	}
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		return fmt.Errorf("write binary: %w", err)
	}
	os.Remove(src) // clean up
	return nil
}

func hasGitChanges(ctx context.Context, repoDir string) (bool, error) {
	out, err := run(ctx, repoDir, "git", "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	return strings.TrimSpace(out) != "", nil
}

func run(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("%s %v: %w: %s", name, args, err, strings.TrimSpace(out.String()))
	}
	return out.String(), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

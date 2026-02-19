package selfevolve

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"visor/internal/observability"
)

// ExitCodeRestart is the exit code that signals the supervisor to restart
// with the new binary. Any process manager (systemd, supervisor script)
// should watch for this code.
const ExitCodeRestart = 42

// MaxBackups is the number of old binaries to keep for rollback.
const MaxBackups = 3

// CrashWindow is how long after startup a crash triggers auto-rollback.
const CrashWindow = 30 * time.Second

type Config struct {
	Enabled bool
	RepoDir string
	Push    bool
	DataDir string // for changelog; defaults to RepoDir + "/data"
}

type Request struct {
	CommitMessage string
	ChatID        int64
	Backend       string // which agent backend triggered this
}

// Result describes what happened during Apply.
type Result struct {
	Committed bool
	Built     bool
	BuildErr  string // non-empty if build failed (commit was rolled back)
	VetErr    string // non-empty if go vet failed (commit was rolled back)
}

type Manager struct {
	cfg       Config
	startedAt time.Time
	exitFn    func(code int) // overridable for testing (defaults to os.Exit)
	nowFn     func() time.Time // overridable for testing
	log       *observability.Logger
}

func New(cfg Config) *Manager {
	if cfg.RepoDir == "" {
		cfg.RepoDir = "."
	}
	if cfg.DataDir == "" {
		cfg.DataDir = filepath.Join(cfg.RepoDir, "data")
	}
	return &Manager{
		cfg:       cfg,
		startedAt: time.Now(),
		exitFn:    os.Exit,
		nowFn:     time.Now,
		log:       observability.Component("selfevolve"),
	}
}

func (m *Manager) Enabled() bool { return m.cfg.Enabled }

// Apply runs the full self-evolution pipeline: vet → commit → build → backup → replace binary.
// If vet or build fails, it rolls back the commit and returns the error in Result.
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

	// step 2: go vet
	vetOut, vetErr := run(ctx, m.cfg.RepoDir, "go", "vet", "./...")
	if vetErr != nil {
		m.log.Error(ctx, "self-evolve vet failed, rolling back", "error", vetErr.Error(), "output", truncateStr(vetOut, 500))
		rollback(ctx, m.cfg.RepoDir, m.log)
		return Result{Committed: true, VetErr: vetErr.Error()}, nil
	}

	// step 3: build
	newBinary := filepath.Join(m.cfg.RepoDir, "visor-new")
	buildOut, buildErr := run(ctx, m.cfg.RepoDir, "go", "build", "-o", newBinary, ".")
	if buildErr != nil {
		m.log.Error(ctx, "self-evolve build failed, rolling back", "error", buildErr.Error(), "output", truncateStr(buildOut, 500))
		rollback(ctx, m.cfg.RepoDir, m.log)
		return Result{Committed: true, BuildErr: buildErr.Error()}, nil
	}
	m.log.Info(ctx, "self-evolve build succeeded", "binary", newBinary)

	// step 4: push (if enabled)
	if m.cfg.Push {
		if _, err := run(ctx, m.cfg.RepoDir, "git", "push"); err != nil {
			return Result{Committed: true, Built: true}, fmt.Errorf("git push: %w", err)
		}
		m.log.Info(ctx, "self-evolve pushed")
	}

	// step 5: backup current binary before replacing
	currentBinary, err := os.Executable()
	if err != nil {
		return Result{Committed: true, Built: true}, fmt.Errorf("find current binary: %w", err)
	}
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return Result{Committed: true, Built: true}, fmt.Errorf("resolve binary symlink: %w", err)
	}

	if err := backupBinary(currentBinary, MaxBackups); err != nil {
		m.log.Warn(ctx, "binary backup failed (continuing anyway)", "error", err.Error())
	}

	if err := replaceBinary(newBinary, currentBinary); err != nil {
		return Result{Committed: true, Built: true}, fmt.Errorf("replace binary: %w", err)
	}
	m.log.Info(ctx, "self-evolve binary replaced", "path", currentBinary)

	// step 6: log to changelog
	m.writeChangelog(req)

	return Result{Committed: true, Built: true}, nil
}

// Restart exits with ExitCodeRestart, signaling the supervisor to respawn.
func (m *Manager) Restart() {
	m.log.Info(nil, "self-evolve restarting", "exit_code", ExitCodeRestart)
	m.exitFn(ExitCodeRestart)
}

// ShouldAutoRollback checks if visor crashed within CrashWindow of startup,
// indicating the new binary is broken and should be rolled back.
func (m *Manager) ShouldAutoRollback() bool {
	return m.nowFn().Sub(m.startedAt) < CrashWindow
}

// AutoRollback restores the most recent backup binary and restarts.
func (m *Manager) AutoRollback(ctx context.Context) error {
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find current binary: %w", err)
	}
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return fmt.Errorf("resolve binary symlink: %w", err)
	}

	backup := latestBackup(currentBinary)
	if backup == "" {
		return fmt.Errorf("no backup binary found for rollback")
	}

	if err := replaceBinary(backup, currentBinary); err != nil {
		return fmt.Errorf("rollback replace: %w", err)
	}
	m.log.Info(ctx, "auto-rollback complete", "backup", backup)

	// also rollback the git commit
	rollback(ctx, m.cfg.RepoDir, m.log)

	return nil
}

// writeChangelog appends an entry to the self-evolution changelog.
func (m *Manager) writeChangelog(req Request) {
	logDir := m.cfg.DataDir
	os.MkdirAll(logDir, 0o755)
	logFile := filepath.Join(logDir, "selfevolve.log")

	entry := fmt.Sprintf("[%s] backend=%s chat_id=%d message=%q\n",
		m.nowFn().UTC().Format(time.RFC3339), req.Backend, req.ChatID, req.CommitMessage)

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		m.log.Warn(nil, "changelog write failed", "error", err.Error())
		return
	}
	defer f.Close()
	f.WriteString(entry)
}

func rollback(ctx context.Context, repoDir string, log *observability.Logger) {
	if _, err := run(ctx, repoDir, "git", "reset", "HEAD~1"); err != nil {
		log.Error(ctx, "rollback failed", "error", err.Error())
	} else {
		log.Info(ctx, "self-evolve commit rolled back")
	}
}

// backupBinary copies the current binary to name.bak.N, rotating old backups.
func backupBinary(binaryPath string, maxBackups int) error {
	if _, err := os.Stat(binaryPath); err != nil {
		return nil // nothing to backup
	}

	// shift existing backups: .bak.2 -> .bak.3, .bak.1 -> .bak.2, etc
	for i := maxBackups; i > 1; i-- {
		old := fmt.Sprintf("%s.bak.%d", binaryPath, i-1)
		new := fmt.Sprintf("%s.bak.%d", binaryPath, i)
		os.Rename(old, new) // ignore errors (file might not exist)
	}

	// copy current to .bak.1
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("read binary for backup: %w", err)
	}
	bakPath := fmt.Sprintf("%s.bak.1", binaryPath)
	if err := os.WriteFile(bakPath, data, 0o755); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}

	// clean up old backups beyond maxBackups
	pruneBackups(binaryPath, maxBackups)
	return nil
}

// latestBackup finds the most recent .bak.N file.
func latestBackup(binaryPath string) string {
	for i := 1; i <= MaxBackups+1; i++ {
		path := fmt.Sprintf("%s.bak.%d", binaryPath, i)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// pruneBackups removes .bak.N files beyond maxBackups.
func pruneBackups(binaryPath string, maxBackups int) {
	pattern := binaryPath + ".bak.*"
	matches, _ := filepath.Glob(pattern)
	if len(matches) <= maxBackups {
		return
	}
	sort.Strings(matches)
	for _, m := range matches[maxBackups:] {
		os.Remove(m)
	}
}

// replaceBinary atomically replaces dst with src via rename.
// Falls back to copy if rename fails (cross-device).
func replaceBinary(src, dst string) error {
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
	os.Remove(src)
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

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

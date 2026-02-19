package selfevolve

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"visor/internal/observability"
)

type Config struct {
	Enabled bool
	RepoDir string
	Push    bool
}

type Request struct {
	CommitMessage string
	ChatID        int64
}

type Manager struct {
	cfg Config
	log *observability.Logger
}

func New(cfg Config) *Manager {
	if cfg.RepoDir == "" {
		cfg.RepoDir = "."
	}
	return &Manager{cfg: cfg, log: observability.Component("selfevolve")}
}

func (m *Manager) Enabled() bool { return m.cfg.Enabled }

func (m *Manager) Apply(ctx context.Context, req Request) error {
	if !m.cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(req.CommitMessage) == "" {
		req.CommitMessage = "self-evolution update"
	}

	changed, err := hasGitChanges(ctx, m.cfg.RepoDir)
	if err != nil {
		return err
	}
	if !changed {
		m.log.Info(ctx, "self-evolve skipped: no git changes")
		return nil
	}

	if _, err := run(ctx, m.cfg.RepoDir, "git", "add", "-A"); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	if _, err := run(ctx, m.cfg.RepoDir, "git", "commit", "-m", req.CommitMessage); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	m.log.Info(ctx, "self-evolve committed", "message", req.CommitMessage)

	if m.cfg.Push {
		if _, err := run(ctx, m.cfg.RepoDir, "git", "push"); err != nil {
			return fmt.Errorf("git push: %w", err)
		}
		m.log.Info(ctx, "self-evolve pushed")
	}

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

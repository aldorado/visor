package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"visor/internal/observability"
)

// Context holds the runtime context passed to skill scripts.
type Context struct {
	UserMessage string `json:"user_message"`
	ChatID      string `json:"chat_id"`
	MessageType string `json:"message_type"` // "text", "voice", "photo", etc.
	Platform    string `json:"platform"`     // "telegram"
	DataDir     string `json:"data_dir"`
	SkillDir    string `json:"skill_dir"` // absolute path to this skill's directory
}

// Result holds the output from a skill execution.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// Executor runs skills as subprocesses.
type Executor struct {
	log *observability.Logger
}

func NewExecutor() *Executor {
	return &Executor{
		log: observability.Component("skills.executor"),
	}
}

// Run executes a skill with the given context.
// The skill's `run` command is split on spaces and executed in the skill's directory.
// Context is passed via both env vars and stdin (JSON).
func (e *Executor) Run(ctx context.Context, skill *Skill, sc Context) (*Result, error) {
	timeout := time.Duration(skill.Manifest.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	parts := strings.Fields(skill.Manifest.Run)
	if len(parts) == 0 {
		return nil, fmt.Errorf("skill %s: empty run command", skill.Manifest.Name)
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = skill.Dir

	// pass context as JSON on stdin
	stdinData, err := json.Marshal(sc)
	if err != nil {
		return nil, fmt.Errorf("skill %s: marshal context: %w", skill.Manifest.Name, err)
	}
	cmd.Stdin = bytes.NewReader(stdinData)

	// env vars for context
	cmd.Env = append(cmd.Environ(),
		"VISOR_USER_MESSAGE="+sc.UserMessage,
		"VISOR_CHAT_ID="+sc.ChatID,
		"VISOR_MESSAGE_TYPE="+sc.MessageType,
		"VISOR_PLATFORM="+sc.Platform,
		"VISOR_DATA_DIR="+sc.DataDir,
		"VISOR_SKILL_DIR="+sc.SkillDir,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	e.log.Info(ctx, "skill execution started", "skill", skill.Manifest.Name, "command", skill.Manifest.Run, "timeout_s", skill.Manifest.Timeout)

	runErr := cmd.Run()
	duration := time.Since(start)

	result := &Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
	}

	if runErr != nil {
		// check context first — killed-by-timeout also produces an ExitError
		if ctx.Err() != nil {
			e.log.Warn(ctx, "skill execution timed out", "skill", skill.Manifest.Name, "timeout_s", skill.Manifest.Timeout, "duration_ms", duration.Milliseconds())
			return result, fmt.Errorf("skill %s: timeout after %v", skill.Manifest.Name, timeout)
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return result, fmt.Errorf("skill %s: exec: %w", skill.Manifest.Name, runErr)
		}
	}

	e.log.Info(ctx, "skill execution finished", "skill", skill.Manifest.Name, "exit_code", result.ExitCode, "duration_ms", duration.Milliseconds(), "stdout_len", len(result.Stdout), "stderr_len", len(result.Stderr))
	return result, nil
}

package levelup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

var composeRunner = runDockerCompose

func ValidateComposeConfig(ctx context.Context, assembly *ComposeAssembly, env map[string]string) error {
	if assembly == nil {
		return fmt.Errorf("compose assembly is required")
	}
	if len(assembly.Files) == 0 {
		return fmt.Errorf("compose files are required")
	}

	envFiles := existingEnvFiles(assembly.ProjectRoot)
	args := BuildComposeConfigArgs(assembly, envFiles)

	output, err := composeRunner(ctx, assembly.ProjectRoot, args, env)
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return fmt.Errorf("docker compose config failed: %w", err)
		}
		return fmt.Errorf("docker compose config failed: %w: %s", err, trimmed)
	}

	return nil
}

func BuildComposeConfigArgs(assembly *ComposeAssembly, envFiles []string) []string {
	args := []string{"compose"}
	args = append(args, BuildComposeFileArgs(assembly.Files)...)
	args = append(args, "--project-directory", assembly.ProjectRoot)
	for _, envFile := range envFiles {
		args = append(args, "--env-file", envFile)
	}
	args = append(args, "config")
	return args
}

func existingEnvFiles(projectRoot string) []string {
	candidates := []string{
		filepath.Join(projectRoot, ".env"),
		filepath.Join(projectRoot, ".levelup.env"),
	}

	files := make([]string, 0, len(candidates))
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			files = append(files, path)
		}
	}
	return files
}

func runDockerCompose(ctx context.Context, projectRoot string, args []string, env map[string]string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = projectRoot
	if len(env) > 0 {
		cmd.Env = mapToEnv(env)
	}
	return cmd.CombinedOutput()
}

func mapToEnv(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%s=%s", k, env[k]))
	}
	return out
}

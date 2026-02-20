package levelup

import (
	"context"
	"fmt"
	"strings"
)

func UpEnabled(ctx context.Context, projectRoot, baseComposeFile string) error {
	manifests, err := DiscoverManifests(projectRoot)
	if err != nil {
		return err
	}
	state, err := LoadState(projectRoot)
	if err != nil {
		return err
	}
	overlays := make([]string, 0, len(state.Enabled))
	for _, name := range state.Enabled {
		manifest, ok := manifests[name]
		if !ok {
			return fmt.Errorf("enabled level-up %q has no manifest", name)
		}
		overlays = append(overlays, manifest.ComposeFile)
	}
	assembly, err := BuildComposeAssembly(projectRoot, baseComposeFile, overlays)
	if err != nil {
		return err
	}
	env, err := LoadLayeredEnv(projectRoot)
	if err != nil {
		return err
	}
	if err := ValidateComposeConfig(ctx, assembly, env); err != nil {
		return err
	}
	args := BuildComposeUpArgs(assembly, existingEnvFiles(projectRoot))
	output, err := composeRunner(ctx, assembly.ProjectRoot, args, env)
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return fmt.Errorf("docker compose up failed: %w", err)
		}
		return fmt.Errorf("docker compose up failed: %w: %s", err, trimmed)
	}
	return nil
}

func BuildComposeUpArgs(assembly *ComposeAssembly, envFiles []string) []string {
	args := []string{"compose"}
	args = append(args, BuildComposeFileArgs(assembly.Files)...)
	args = append(args, "--project-directory", assembly.ProjectRoot)
	for _, envFile := range envFiles {
		args = append(args, "--env-file", envFile)
	}
	args = append(args, "up", "-d")
	return args
}

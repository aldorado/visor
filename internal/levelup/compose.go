package levelup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type ComposeAssembly struct {
	ProjectRoot  string
	BaseFile     string
	OverlayFiles []string
	Files        []string
}

func BuildComposeAssembly(projectRoot, baseComposeFile string, selectedOverlays []string) (*ComposeAssembly, error) {
	if projectRoot == "" {
		return nil, errors.New("project root is required")
	}
	if baseComposeFile == "" {
		return nil, errors.New("base compose file is required")
	}

	projectRootAbs, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}

	basePath, err := resolvePath(projectRootAbs, baseComposeFile)
	if err != nil {
		return nil, fmt.Errorf("resolve base compose file: %w", err)
	}
	if err := requireFile(basePath); err != nil {
		return nil, err
	}

	baseDir := filepath.Dir(basePath)
	files := []string{basePath}
	overlays := make([]string, 0, len(selectedOverlays))
	seen := map[string]struct{}{basePath: {}}

	for _, overlay := range selectedOverlays {
		overlayPath, err := resolvePath(baseDir, overlay)
		if err != nil {
			return nil, fmt.Errorf("resolve overlay %q: %w", overlay, err)
		}
		if err := requireFile(overlayPath); err != nil {
			return nil, err
		}
		if _, ok := seen[overlayPath]; ok {
			continue
		}

		seen[overlayPath] = struct{}{}
		overlays = append(overlays, overlayPath)
		files = append(files, overlayPath)
	}

	return &ComposeAssembly{
		ProjectRoot:  projectRootAbs,
		BaseFile:     basePath,
		OverlayFiles: overlays,
		Files:        files,
	}, nil
}

func BuildComposeFileArgs(files []string) []string {
	args := make([]string, 0, len(files)*2)
	for _, file := range files {
		args = append(args, "-f", file)
	}
	return args
}

func resolvePath(baseDir, target string) (string, error) {
	if target == "" {
		return "", errors.New("path is empty")
	}

	path := target
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return filepath.Clean(absPath), nil
}

func requireFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("required file missing: %s", path)
		}
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected file but got directory: %s", path)
	}
	return nil
}

package promptsync

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func Sync(repoRoot string) error {
	if repoRoot == "" {
		repoRoot = "."
	}

	if err := syncSystemFiles(repoRoot); err != nil {
		return err
	}
	if err := syncSkills(repoRoot); err != nil {
		return err
	}
	return nil
}

func syncSystemFiles(repoRoot string) error {
	source := filepath.Join(repoRoot, ".pi", "SYSTEM.md")
	if _, err := os.Stat(source); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", source, err)
	}

	return nil
}

func syncSkills(repoRoot string) error {
	sourceRoot := filepath.Join(repoRoot, "skills")
	if _, err := os.Stat(sourceRoot); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", sourceRoot, err)
	}

	targetRoots := []string{
		filepath.Join(repoRoot, ".pi", "skills"),
	}

	sourceNames, err := listDirNames(sourceRoot)
	if err != nil {
		return err
	}

	for _, targetRoot := range targetRoots {
		if err := os.MkdirAll(targetRoot, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", targetRoot, err)
		}

		targetNames, err := listDirNames(targetRoot)
		if err != nil {
			return err
		}

		for name := range sourceNames {
			src := filepath.Join(sourceRoot, name)
			dst := filepath.Join(targetRoot, name)
			if err := copyTree(src, dst); err != nil {
				return err
			}
		}

		for name := range targetNames {
			if _, ok := sourceNames[name]; ok {
				continue
			}
			if err := os.RemoveAll(filepath.Join(targetRoot, name)); err != nil {
				return fmt.Errorf("remove stale skill %s/%s: %w", targetRoot, name, err)
			}
		}
	}

	return nil
}

func listDirNames(root string) (map[string]struct{}, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", root, err)
	}
	out := make(map[string]struct{})
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		out[e.Name()] = struct{}{}
	}
	return out, nil
}

func copyTree(srcRoot, dstRoot string) error {
	if err := os.RemoveAll(dstRoot); err != nil {
		return fmt.Errorf("clear %s: %w", dstRoot, err)
	}
	if err := os.MkdirAll(dstRoot, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dstRoot, err)
	}

	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return fmt.Errorf("rel %s: %w", path, err)
		}
		if rel == "." {
			return nil
		}

		target := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("mkdir %s: %w", target, err)
			}
			return nil
		}

		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

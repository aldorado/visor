package setup

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func UpdateDotEnv(projectRoot string, set map[string]string, unset []string) error {
	path := filepath.Join(projectRoot, ".env")
	current, err := parseEnvFile(path)
	if err != nil {
		return err
	}
	for k, v := range set {
		key := strings.TrimSpace(k)
		if !envKeyPattern.MatchString(key) {
			return fmt.Errorf("invalid env key: %s", k)
		}
		if strings.Contains(v, "\n") {
			return fmt.Errorf("env value for %s contains newline", key)
		}
		current[key] = v
	}
	for _, k := range unset {
		key := strings.TrimSpace(k)
		if !envKeyPattern.MatchString(key) {
			return fmt.Errorf("invalid env key: %s", k)
		}
		delete(current, key)
	}
	keys := make([]string, 0, len(current))
	for k := range current {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", k, current[k]))
	}
	content := ""
	if len(lines) > 0 {
		content = strings.Join(lines, "\n") + "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func parseEnvFile(path string) (map[string]string, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	out := map[string]string{}
	s := bufio.NewScanner(f)
	lineNo := 0
	for s.Scan() {
		lineNo++
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid env line %s:%d", path, lineNo)
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return out, nil
}

package levelup

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func LoadLayeredEnv(projectRoot string) (map[string]string, error) {
	if projectRoot == "" {
		return nil, errors.New("project root is required")
	}

	env := map[string]string{}
	for _, kv := range os.Environ() {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		env[k] = v
	}

	for _, name := range []string{".env", ".levelup.env"} {
		path := filepath.Join(projectRoot, name)
		fileVars, err := parseEnvFile(path)
		if err != nil {
			return nil, err
		}
		for k, v := range fileVars {
			env[k] = v
		}
	}

	for _, kv := range os.Environ() {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		env[k] = v
	}

	return env, nil
}

func ValidateRequiredEnv(env map[string]string, required []string) error {
	if len(required) == 0 {
		return nil
	}

	missing := make([]string, 0)
	for _, key := range required {
		value, ok := env[key]
		if !ok || strings.TrimSpace(value) == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	sort.Strings(missing)
	return fmt.Errorf("missing required env keys: %s", strings.Join(missing, ", "))
}

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func UpdateLevelupEnv(projectRoot string, set map[string]string, unset []string) error {
	if projectRoot == "" {
		return fmt.Errorf("project root is required")
	}

	path := filepath.Join(projectRoot, ".levelup.env")
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
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func parseEnvFile(path string) (map[string]string, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
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
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid env line %s:%d", path, lineNo)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("empty env key at %s:%d", path, lineNo)
		}

		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		out[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return out, nil
}

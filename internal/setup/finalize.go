package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	personalityBegin = "<!-- SETUP_PERSONALITY_OVERRIDE:BEGIN -->"
	personalityEnd   = "<!-- SETUP_PERSONALITY_OVERRIDE:END -->"
	setupBegin       = "<!-- FIRST_RUN_SETUP:BEGIN -->"
	setupEnd         = "<!-- FIRST_RUN_SETUP:END -->"
)

type SummaryInput struct {
	AgentBackend string
	Levelups     []string
	WebhookURL   string
	HealthOK     bool
}

func ApplyPersonalityOverride(projectRoot, file, choice, content string) error {
	choice = strings.TrimSpace(strings.ToLower(choice))
	if choice == "" || choice == "keep_defaults" || choice == "keep" {
		return nil
	}
	if choice != "custom" {
		return fmt.Errorf("unknown personality choice: %s", choice)
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("personality_content is required for custom choice")
	}
	if strings.TrimSpace(file) == "" {
		file = filepath.Join(projectRoot, ".claude", "CLAUDE.md")
	}
	bytes, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read %s: %w", file, err)
	}
	updated := upsertBlock(string(bytes), personalityBegin, personalityEnd, "\n"+strings.TrimSpace(content)+"\n")
	if err := os.WriteFile(file, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", file, err)
	}
	return nil
}

func CleanupSetupInstructions(projectRoot, file string) error {
	if strings.TrimSpace(file) == "" {
		file = filepath.Join(projectRoot, ".claude", "CLAUDE.md")
	}
	bytes, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", file, err)
	}
	text := string(bytes)
	text = removeBlock(text, setupBegin, setupEnd)
	if err := os.WriteFile(file, []byte(text), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", file, err)
	}
	return nil
}

func WriteSetupSummary(projectRoot string, in SummaryInput) (string, error) {
	dir := filepath.Join(projectRoot, "data", "setup")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	path := filepath.Join(dir, "summary.md")
	content := "# setup summary\n\n"
	content += fmt.Sprintf("agent backend: `%s`\n", in.AgentBackend)
	if strings.TrimSpace(in.WebhookURL) != "" {
		content += fmt.Sprintf("webhook: `%s`\n", in.WebhookURL)
	}
	if len(in.Levelups) > 0 {
		content += fmt.Sprintf("levelups: `%s`\n", strings.Join(in.Levelups, ", "))
	}
	if in.HealthOK {
		content += "health: `ok`\n"
	}
	content += "\nstart: `go build -o bin/visor . && ./bin/visor`\n"
	content += "stop: `ctrl+c` (or stop your service manager)\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

func upsertBlock(text, begin, end, body string) string {
	block := begin + "\n" + body + "\n" + end
	if strings.Contains(text, begin) && strings.Contains(text, end) {
		return replaceBlock(text, begin, end, block)
	}
	if strings.HasSuffix(text, "\n") {
		return text + "\n" + block + "\n"
	}
	return text + "\n\n" + block + "\n"
}

func removeBlock(text, begin, end string) string {
	if !strings.Contains(text, begin) || !strings.Contains(text, end) {
		return text
	}
	return replaceBlock(text, begin, end, "")
}

func replaceBlock(text, begin, end, replacement string) string {
	s := strings.Index(text, begin)
	e := strings.Index(text, end)
	if s == -1 || e == -1 || e < s {
		return text
	}
	e += len(end)
	out := text[:s] + replacement + text[e:]
	out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	return out
}

package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyPersonalityOverride(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPersonalityOverride(root, file, "custom", "be concise"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(file)
	if !strings.Contains(string(b), "SETUP_PERSONALITY_OVERRIDE") {
		t.Fatal("expected override block")
	}
}

func TestWriteSetupSummary(t *testing.T) {
	root := t.TempDir()
	path, err := WriteSetupSummary(root, SummaryInput{AgentBackend: "pi", HealthOK: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

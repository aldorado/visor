package server

import (
	"strings"
	"testing"
	"time"

	"visor/internal/scheduler"
)

func TestBuildScheduledTaskContent_AddsSelfImprovementDateGuard(t *testing.T) {
	task := scheduler.Task{
		ID:        "task-1",
		Recurring: true,
		Prompt:    "Create one new forge-style self-improvement proposal for visor code.",
	}

	now := time.Date(2026, 3, 1, 20, 0, 30, 0, time.UTC) // 21:00:30 Europe/Vienna
	content := buildScheduledTaskContent(task, "Europe/Vienna", now)

	if !strings.Contains(content, "current local date: 2026-03-01") {
		t.Fatalf("content=%q", content)
	}
	if !strings.Contains(content, "filename prefix must be exactly 2026-03-01") {
		t.Fatalf("content=%q", content)
	}
	if !strings.Contains(content, "markdown field `_date_` must be exactly 2026-03-01") {
		t.Fatalf("content=%q", content)
	}
}

func TestBuildScheduledTaskContent_NoDateGuardForRegularPrompt(t *testing.T) {
	task := scheduler.Task{
		ID:        "task-2",
		Recurring: false,
		Prompt:    "Erinnere mich daran, dass meine Kontaktlinsen ablaufen.",
	}

	content := buildScheduledTaskContent(task, "Europe/Vienna", time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC))

	if strings.Contains(content, "date guard (must follow exactly)") {
		t.Fatalf("content=%q", content)
	}
	if !strings.Contains(content, "prompt: Erinnere mich daran") {
		t.Fatalf("content=%q", content)
	}
}

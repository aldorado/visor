package scheduler

import "testing"

func TestExtractActions_None(t *testing.T) {
	clean, actions, err := ExtractActions("hello")
	if err != nil {
		t.Fatal(err)
	}
	if clean != "hello" {
		t.Fatalf("clean=%q", clean)
	}
	if actions != nil {
		t.Fatal("expected nil actions")
	}
}

func TestExtractActions_ParsesAndCleans(t *testing.T) {
	raw := "ok\n\n```json\n{\"schedule_actions\":{\"list\":true,\"create\":[{\"prompt\":\"ping\",\"run_at\":\"2026-02-20T10:00:00Z\"}]}}\n```"
	clean, actions, err := ExtractActions(raw)
	if err != nil {
		t.Fatal(err)
	}
	if clean != "ok" {
		t.Fatalf("clean=%q", clean)
	}
	if actions == nil || !actions.List {
		t.Fatal("expected list action")
	}
	if len(actions.Create) != 1 {
		t.Fatalf("create len=%d", len(actions.Create))
	}
	if actions.Create[0].Prompt != "ping" {
		t.Fatalf("prompt=%q", actions.Create[0].Prompt)
	}
}

func TestExtractActions_InvalidJSON(t *testing.T) {
	raw := "```json\n{\"schedule_actions\":\n```"
	_, _, err := ExtractActions(raw)
	if err == nil {
		t.Fatal("expected error")
	}
}

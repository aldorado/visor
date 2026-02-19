package levelup

import "testing"

func TestExtractActionsNone(t *testing.T) {
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

func TestExtractActionsParses(t *testing.T) {
	raw := "ok\n```json\n{\"levelup_actions\":{\"env_set\":{\"A\":\"1\"},\"enable\":[\"obsidian\"]}}\n```"
	clean, actions, err := ExtractActions(raw)
	if err != nil {
		t.Fatal(err)
	}
	if clean != "ok" {
		t.Fatalf("clean=%q", clean)
	}
	if actions == nil {
		t.Fatal("expected actions")
	}
	if actions.EnvSet["A"] != "1" {
		t.Fatalf("env set A=%q", actions.EnvSet["A"])
	}
	if len(actions.Enable) != 1 || actions.Enable[0] != "obsidian" {
		t.Fatalf("enable=%v", actions.Enable)
	}
}

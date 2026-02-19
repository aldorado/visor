package skills

import (
	"testing"
)

func TestExtractActionsCreate(t *testing.T) {
	response := `ok i'll create that skill for you

` + "```json\n" + `{"skill_actions": {"create": [{"name": "weather", "description": "checks weather", "run": "python3 run.py", "triggers": ["weather", "wetter"], "script": "print('sunny')"}]}}` + "\n```\n" + `
done!`

	clean, actions, err := ExtractActions(response)
	if err != nil {
		t.Fatal(err)
	}
	if actions == nil {
		t.Fatal("expected actions")
	}
	if len(actions.Create) != 1 {
		t.Fatalf("expected 1 create action, got %d", len(actions.Create))
	}
	if actions.Create[0].Name != "weather" {
		t.Errorf("name = %q", actions.Create[0].Name)
	}
	if actions.Create[0].Script != "print('sunny')" {
		t.Errorf("script = %q", actions.Create[0].Script)
	}
	// clean text should not contain the json block
	if containsStr(clean, "skill_actions") {
		t.Errorf("clean text still contains json block: %q", clean)
	}
	if !containsStr(clean, "ok i'll create that skill") {
		t.Errorf("clean text missing response text: %q", clean)
	}
}

func TestExtractActionsEdit(t *testing.T) {
	response := "updated the skill\n```json\n" + `{"skill_actions": {"edit": [{"name": "weather", "description": "new desc"}]}}` + "\n```"

	_, actions, err := ExtractActions(response)
	if err != nil {
		t.Fatal(err)
	}
	if actions == nil || len(actions.Edit) != 1 {
		t.Fatal("expected 1 edit action")
	}
	if actions.Edit[0].Name != "weather" {
		t.Errorf("name = %q", actions.Edit[0].Name)
	}
}

func TestExtractActionsDelete(t *testing.T) {
	response := "removing it\n```json\n" + `{"skill_actions": {"delete": [{"name": "old-skill"}]}}` + "\n```"

	_, actions, err := ExtractActions(response)
	if err != nil {
		t.Fatal(err)
	}
	if actions == nil || len(actions.Delete) != 1 {
		t.Fatal("expected 1 delete action")
	}
	if actions.Delete[0].Name != "old-skill" {
		t.Errorf("name = %q", actions.Delete[0].Name)
	}
}

func TestExtractActionsNoActions(t *testing.T) {
	response := "just a normal response with no json blocks"

	clean, actions, err := ExtractActions(response)
	if err != nil {
		t.Fatal(err)
	}
	if actions != nil {
		t.Fatal("expected nil actions")
	}
	if clean != response {
		t.Errorf("clean = %q, want original", clean)
	}
}

func TestExtractActionsSkipNonSkillJSON(t *testing.T) {
	response := "here's some data\n```json\n" + `{"email_actions": [{"to": "a@b.com"}]}` + "\n```\nand more text"

	clean, actions, err := ExtractActions(response)
	if err != nil {
		t.Fatal(err)
	}
	if actions != nil {
		t.Fatal("expected nil actions for non-skill json")
	}
	if clean != response {
		t.Errorf("clean should be unchanged for non-skill json")
	}
}

func TestExtractActionsMultipleActions(t *testing.T) {
	response := "batch update\n```json\n" + `{"skill_actions": {"create": [{"name": "a", "run": "echo a"}], "delete": [{"name": "b"}]}}` + "\n```"

	_, actions, err := ExtractActions(response)
	if err != nil {
		t.Fatal(err)
	}
	if actions == nil {
		t.Fatal("expected actions")
	}
	if len(actions.Create) != 1 {
		t.Errorf("create = %d, want 1", len(actions.Create))
	}
	if len(actions.Delete) != 1 {
		t.Errorf("delete = %d, want 1", len(actions.Delete))
	}
}

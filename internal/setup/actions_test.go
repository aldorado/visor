package setup

import "testing"

func TestExtractActions(t *testing.T) {
	raw := "ok\n```json\n{\"setup_actions\":{\"env_set\":{\"A\":\"1\"},\"validate_telegram\":true}}\n```"
	clean, actions, err := ExtractActions(raw)
	if err != nil {
		t.Fatal(err)
	}
	if clean != "ok" {
		t.Fatalf("unexpected clean: %q", clean)
	}
	if actions == nil || actions.EnvSet["A"] != "1" || !actions.ValidateTelegram {
		t.Fatal("parsed actions mismatch")
	}
}

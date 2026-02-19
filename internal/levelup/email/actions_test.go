package email

import "testing"

func TestExtractActions(t *testing.T) {
	input := "done\n```json\n{\"email_actions\":[{\"to\":\"a@b.com\",\"subject\":\"hi\",\"body\":\"yo\"}]}\n```"
	clean, actions, err := ExtractActions(input)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if clean != "done" {
		t.Fatalf("unexpected clean text: %q", clean)
	}
	if len(actions) != 1 || actions[0].To != "a@b.com" {
		t.Fatalf("unexpected actions: %#v", actions)
	}
}

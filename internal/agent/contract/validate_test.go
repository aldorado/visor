package contract

import "testing"

func TestValidate_TextRequiredWhenNotVoice(t *testing.T) {
	err := Validate(Response{ResponseText: "", SendVoice: false})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidate_AllowsEmptyTextForVoice(t *testing.T) {
	err := Validate(Response{ResponseText: "", SendVoice: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_CommitMessageRequiredForCodeChanges(t *testing.T) {
	err := Validate(Response{ResponseText: "ok", CodeChanges: true})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestFixDefaults_ResetsConversationFinishedWithoutGoodbye(t *testing.T) {
	resp := Response{ResponseText: "let's keep going", ConversationFinished: true}
	changed := FixDefaults(&resp)
	if !changed {
		t.Fatal("expected change")
	}
	if resp.ConversationFinished {
		t.Fatal("conversation_finished should be false")
	}
}

func TestFixDefaults_KeepsConversationFinishedWithGoodbye(t *testing.T) {
	resp := Response{ResponseText: "ok bye", ConversationFinished: true}
	changed := FixDefaults(&resp)
	if changed {
		t.Fatal("did not expect change")
	}
	if !resp.ConversationFinished {
		t.Fatal("conversation_finished should stay true")
	}
}

func TestParseRaw_Metadata(t *testing.T) {
	raw := "hello\n---\nsend_voice: true\ncode_changes: true\ncommit_message: test\nconversation_finished: false"
	resp := ParseRaw(raw)
	if resp.ResponseText != "hello" {
		t.Fatalf("text=%q", resp.ResponseText)
	}
	if !resp.SendVoice || !resp.CodeChanges {
		t.Fatalf("flags parse failed: %+v", resp)
	}
	if resp.CommitMessage != "test" {
		t.Fatalf("commit_message=%q", resp.CommitMessage)
	}
}

func TestJSONSchema_NotEmpty(t *testing.T) {
	if JSONSchema() == "" {
		t.Fatal("schema should not be empty")
	}
}

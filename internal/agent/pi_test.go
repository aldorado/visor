package agent

import (
	"encoding/json"
	"testing"
)

func TestPiEvent_TextDelta(t *testing.T) {
	line := `{"type":"message_update","assistantMessageEvent":{"type":"text_delta","text":"hello "}}`
	var event piEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if event.Type != "message_update" {
		t.Errorf("type = %q, want message_update", event.Type)
	}
	if event.AssistantMessageEvent == nil {
		t.Fatal("assistantMessageEvent is nil")
	}
	if event.AssistantMessageEvent.Type != "text_delta" {
		t.Errorf("event type = %q, want text_delta", event.AssistantMessageEvent.Type)
	}
	if event.AssistantMessageEvent.Text != "hello " {
		t.Errorf("text = %q, want %q", event.AssistantMessageEvent.Text, "hello ")
	}
}

func TestPiEvent_ResponseSuccess(t *testing.T) {
	line := `{"type":"response","command":"prompt","success":true}`
	var event piEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if event.Type != "response" {
		t.Errorf("type = %q, want response", event.Type)
	}
	if event.Success == nil || !*event.Success {
		t.Error("expected success=true")
	}
}

func TestPiEvent_ResponseFailure(t *testing.T) {
	line := `{"type":"response","command":"prompt","success":false,"error":"invalid prompt"}`
	var event piEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if event.Success == nil || *event.Success {
		t.Error("expected success=false")
	}
	if event.Error != "invalid prompt" {
		t.Errorf("error = %q, want %q", event.Error, "invalid prompt")
	}
}

func TestPiEvent_AgentEnd(t *testing.T) {
	line := `{"type":"agent_end"}`
	var event piEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if event.Type != "agent_end" {
		t.Errorf("type = %q, want agent_end", event.Type)
	}
}

func TestPiEvent_ThinkingDeltaIgnored(t *testing.T) {
	line := `{"type":"message_update","assistantMessageEvent":{"type":"thinking_delta","text":"reasoning..."}}`
	var event piEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// thinking_delta should parse fine but not be text_delta
	if event.AssistantMessageEvent.Type == "text_delta" {
		t.Error("thinking_delta should not be treated as text_delta")
	}
}

func TestPiCommand_Marshal(t *testing.T) {
	cmd := piCommand{Type: "prompt", Message: "hello world"}
	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	expected := `{"type":"prompt","message":"hello world"}`
	if string(data) != expected {
		t.Errorf("got %s, want %s", data, expected)
	}
}

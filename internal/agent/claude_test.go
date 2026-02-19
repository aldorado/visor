package agent

import (
	"encoding/json"
	"testing"
)

func TestClaudeEvent_AssistantText(t *testing.T) {
	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello world"}]}}`
	var event claudeEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if event.Type != "assistant" {
		t.Errorf("type = %q, want assistant", event.Type)
	}

	var msg claudeMessage
	if err := json.Unmarshal(event.Message, &msg); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}
	if msg.Role != "assistant" {
		t.Errorf("role = %q, want assistant", msg.Role)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("content len = %d, want 1", len(msg.Content))
	}
	if msg.Content[0].Type != "text" {
		t.Errorf("block type = %q, want text", msg.Content[0].Type)
	}
	if msg.Content[0].Text != "hello world" {
		t.Errorf("text = %q, want %q", msg.Content[0].Text, "hello world")
	}
}

func TestClaudeEvent_MultipleBlocks(t *testing.T) {
	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"first"},{"type":"text","text":" second"}]}}`
	var event claudeEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var msg claudeMessage
	json.Unmarshal(event.Message, &msg)
	if len(msg.Content) != 2 {
		t.Fatalf("content len = %d, want 2", len(msg.Content))
	}
	if msg.Content[0].Text != "first" {
		t.Errorf("block[0] = %q, want %q", msg.Content[0].Text, "first")
	}
	if msg.Content[1].Text != " second" {
		t.Errorf("block[1] = %q, want %q", msg.Content[1].Text, " second")
	}
}

func TestClaudeEvent_ResultOK(t *testing.T) {
	line := `{"type":"result","result":{"is_error":false,"duration_ms":1234.5}}`
	var event claudeEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if event.Type != "result" {
		t.Errorf("type = %q, want result", event.Type)
	}

	var res claudeResult
	if err := json.Unmarshal(event.Result, &res); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if res.IsError {
		t.Error("expected is_error=false")
	}
}

func TestClaudeEvent_ResultError(t *testing.T) {
	line := `{"type":"result","result":{"is_error":true,"duration_ms":500}}`
	var event claudeEvent
	json.Unmarshal([]byte(line), &event)

	var res claudeResult
	json.Unmarshal(event.Result, &res)
	if !res.IsError {
		t.Error("expected is_error=true")
	}
}

func TestClaudeEvent_SystemIgnored(t *testing.T) {
	line := `{"type":"system","message":"session bootstrap"}`
	var event claudeEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if event.Type != "system" {
		t.Errorf("type = %q, want system", event.Type)
	}
	// system events should parse but not contribute text
}

func TestClaudeEvent_RateLimitIgnored(t *testing.T) {
	line := `{"type":"rate_limit_event","rateLimitType":"token","resetsAt":"2026-01-01T00:00:00Z"}`
	var event claudeEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if event.Type != "rate_limit_event" {
		t.Errorf("type = %q, want rate_limit_event", event.Type)
	}
}

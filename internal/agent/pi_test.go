package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"visor/internal/observability"
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

func TestWithExecutionGuardrail(t *testing.T) {
	input := "check git status"
	out := withExecutionGuardrail(input)
	if !strings.Contains(out, "do not ask the user to run commands") {
		t.Fatalf("guardrail missing in output: %q", out)
	}
	if !strings.HasSuffix(out, input) {
		t.Fatalf("expected original prompt suffix, got: %q", out)
	}
}

func TestNewPiAgent_UsesSessionByDefault(t *testing.T) {
	t.Setenv("PI_NO_SESSION", "")
	a := NewPiAgent(ProcessConfig{})
	joined := strings.Join(a.toolsCfg.Args, " ")
	if strings.Contains(joined, "--no-session") {
		t.Fatalf("expected session mode by default, got: %v", a.toolsCfg.Args)
	}
}

func TestNewPiAgent_RespectsNoSessionEnv(t *testing.T) {
	t.Setenv("PI_NO_SESSION", "true")
	a := NewPiAgent(ProcessConfig{})
	joined := strings.Join(a.toolsCfg.Args, " ")
	if !strings.Contains(joined, "--no-session") {
		t.Fatalf("expected --no-session when PI_NO_SESSION=true, got: %v", a.toolsCfg.Args)
	}
}

func TestLooksLikeDeferral(t *testing.T) {
	if !looksLikeDeferral("kann ich von hier nicht direkt sehen") {
		t.Fatal("expected german deferral phrase to match")
	}
	if !looksLikeDeferral("please run git status and send me output") {
		t.Fatal("expected english deferral phrase to match")
	}
	if looksLikeDeferral("git status: clean, nothing to commit") {
		t.Fatal("status result must not be flagged as deferral")
	}
}

func TestContextWindowTokensFromEnv_Default(t *testing.T) {
	t.Setenv("PI_CONTEXT_WINDOW_TOKENS", "")
	if got := contextWindowTokensFromEnv(); got != 64000 {
		t.Fatalf("got %d want 64000", got)
	}
}

func TestHandoffThresholdFromEnv_Default(t *testing.T) {
	t.Setenv("PI_HANDOFF_THRESHOLD", "")
	if got := handoffThresholdFromEnv(); got != 0.60 {
		t.Fatalf("got %.2f want 0.60", got)
	}
}

func TestBackendLabel_UsesSessionInputTokens(t *testing.T) {
	p := &PiAgent{model: "codex", contextWindowTokens: 1000, sessionInputTokens: 250}
	got := p.BackendLabel()
	if !strings.Contains(got, "ctx 25.0%") {
		t.Fatalf("label=%q want ctx 25.0%%", got)
	}
}

func TestMaybeHandoff_AccumulatesSessionTokens(t *testing.T) {
	p := &PiAgent{
		contextWindowTokens: 1000,
		handoffThreshold:    0.95,
		log:                 observability.Component("agent.pi.test"),
	}

	p.maybeHandoff(context.Background(), nil, 100)
	p.maybeHandoff(context.Background(), nil, 150)

	if p.lastInputTokens != 150 {
		t.Fatalf("lastInputTokens=%d want 150", p.lastInputTokens)
	}
	if p.sessionInputTokens != 250 {
		t.Fatalf("sessionInputTokens=%d want 250", p.sessionInputTokens)
	}
}

func TestSetModel_ResetsTokenCounters(t *testing.T) {
	p := &PiAgent{lastInputTokens: 12, sessionInputTokens: 99}
	if err := p.SetModel("codex"); err != nil {
		t.Fatalf("SetModel error: %v", err)
	}
	if p.lastInputTokens != 0 || p.sessionInputTokens != 0 {
		t.Fatalf("counters not reset: last=%d session=%d", p.lastInputTokens, p.sessionInputTokens)
	}
}

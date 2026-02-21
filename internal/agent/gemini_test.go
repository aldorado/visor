package agent

import "testing"

func TestParseGeminiStreamLine_Message(t *testing.T) {
	line := `{"type":"message","message":{"role":"model","parts":[{"text":"hello "},{"text":"world"}]}}`
	chunk, err := parseGeminiStreamLine(line)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if chunk != "hello world" {
		t.Fatalf("chunk = %q, want %q", chunk, "hello world")
	}
}

func TestParseGeminiStreamLine_Error(t *testing.T) {
	line := `{"type":"error","error":{"message":"rate limit hit"}}`
	_, err := parseGeminiStreamLine(line)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "gemini: rate limit hit" {
		t.Fatalf("err = %q", err.Error())
	}
}

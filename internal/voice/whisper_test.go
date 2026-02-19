package voice

import (
	"encoding/json"
	"testing"
)

func TestWhisperResponseParse(t *testing.T) {
	raw := `{"text":"Hello, how are you doing today?"}`
	var resp whisperResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Text != "Hello, how are you doing today?" {
		t.Errorf("text = %q, want %q", resp.Text, "Hello, how are you doing today?")
	}
}

func TestWhisperResponseParse_Empty(t *testing.T) {
	raw := `{"text":""}`
	var resp whisperResponse
	json.Unmarshal([]byte(raw), &resp)
	if resp.Text != "" {
		t.Errorf("expected empty text, got %q", resp.Text)
	}
}

func TestWhisperResponseParse_Unicode(t *testing.T) {
	raw := `{"text":"Hallo, wie geht es dir? 😊"}`
	var resp whisperResponse
	json.Unmarshal([]byte(raw), &resp)
	if resp.Text != "Hallo, wie geht es dir? 😊" {
		t.Errorf("text = %q", resp.Text)
	}
}

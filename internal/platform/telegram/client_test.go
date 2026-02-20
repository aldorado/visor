package telegram

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendMessage_UsesConfiguredAPIBase(t *testing.T) {
	type msgReq struct {
		ChatID    int64  `json:"chat_id"`
		Text      string `json:"text"`
		ParseMode string `json:"parse_mode"`
	}

	gotPath := ""
	got := msgReq{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	c := NewClientWithOptions("test-token", ts.URL+"/bot", ts.Client())
	if err := c.SendMessage(12345, "hello"); err != nil {
		t.Fatalf("send message: %v", err)
	}

	if gotPath != "/bottest-token/sendMessage" {
		t.Fatalf("path=%q want=%q", gotPath, "/bottest-token/sendMessage")
	}
	if got.ChatID != 12345 {
		t.Fatalf("chat_id=%d want=12345", got.ChatID)
	}
	if got.Text != "hello" {
		t.Fatalf("text=%q want=hello", got.Text)
	}
	if got.ParseMode != "Markdown" {
		t.Fatalf("parse_mode=%q want=Markdown", got.ParseMode)
	}
}

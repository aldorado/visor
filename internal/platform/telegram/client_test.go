package telegram

import (
	"encoding/json"
	"io"
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

func TestSendMessage_FallbackToPlainOnEntityParseError(t *testing.T) {
	type msgReq struct {
		ChatID    int64  `json:"chat_id"`
		Text      string `json:"text"`
		ParseMode string `json:"parse_mode"`
	}

	calls := 0
	var first, second msgReq

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

		var got msgReq
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if calls == 1 {
			first = got
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"ok":false,"error_code":400,"description":"Bad Request: can't parse entities: Can't find end of the entity starting at byte offset 10"}`))
			return
		}

		second = got
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	c := NewClientWithOptions("test-token", ts.URL+"/bot", ts.Client())
	if err := c.SendMessage(12345, "broken _markdown"); err != nil {
		t.Fatalf("send message: %v", err)
	}

	if calls != 2 {
		t.Fatalf("calls=%d want=2", calls)
	}
	if first.ParseMode != "Markdown" {
		t.Fatalf("first parse_mode=%q want=Markdown", first.ParseMode)
	}
	if second.ParseMode != "" {
		t.Fatalf("second parse_mode=%q want empty", second.ParseMode)
	}
	if second.Text != "broken _markdown" {
		t.Fatalf("second text=%q", second.Text)
	}
}

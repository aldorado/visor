package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"visor/internal/agent"
	"visor/internal/config"
	"visor/internal/platform/telegram"
)

func testConfig(secret string) *config.Config {
	return &config.Config{
		TelegramBotToken:      "test-token",
		TelegramWebhookSecret: secret,
		UserChatID:            "12345",
		Port:                  8080,
		AgentBackend:          "echo",
	}
}

func makeUpdate(updateID int, chatID int64, text string) telegram.Update {
	return telegram.Update{
		UpdateID: updateID,
		Message: &telegram.Message{
			MessageID: 1,
			Chat:      telegram.Chat{ID: chatID, Type: "private"},
			Text:      text,
		},
	}
}

func postWebhook(srv *Server, update telegram.Update, headers map[string]string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(update)
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)
	return w
}

func TestHealth(t *testing.T) {
	srv := New(testConfig(""), &agent.EchoAgent{})
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}

func TestWebhook_ValidTextMessage(t *testing.T) {
	srv := New(testConfig(""), &agent.EchoAgent{})
	update := makeUpdate(1, 12345, "hello")
	w := postWebhook(srv, update, nil)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestWebhook_UnauthorizedChat(t *testing.T) {
	srv := New(testConfig(""), &agent.EchoAgent{})
	update := makeUpdate(1, 99999, "hello")
	w := postWebhook(srv, update, nil)

	// should still return 200 (don't leak auth info)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestWebhook_DuplicateUpdate(t *testing.T) {
	srv := New(testConfig(""), &agent.EchoAgent{})
	update := makeUpdate(42, 12345, "hello")

	postWebhook(srv, update, nil)
	w := postWebhook(srv, update, nil)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestWebhook_SignatureValid(t *testing.T) {
	srv := New(testConfig("my-secret"), &agent.EchoAgent{})
	update := makeUpdate(1, 12345, "hello")
	w := postWebhook(srv, update, map[string]string{
		"X-Telegram-Bot-Api-Secret-Token": "my-secret",
	})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestWebhook_SignatureInvalid(t *testing.T) {
	srv := New(testConfig("my-secret"), &agent.EchoAgent{})
	update := makeUpdate(1, 12345, "hello")
	w := postWebhook(srv, update, map[string]string{
		"X-Telegram-Bot-Api-Secret-Token": "wrong-secret",
	})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestWebhook_SignatureMissing(t *testing.T) {
	srv := New(testConfig("my-secret"), &agent.EchoAgent{})
	update := makeUpdate(1, 12345, "hello")
	w := postWebhook(srv, update, nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestWebhook_NoMessage(t *testing.T) {
	srv := New(testConfig(""), &agent.EchoAgent{})
	update := telegram.Update{UpdateID: 1}
	w := postWebhook(srv, update, nil)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestWebhook_BadJSON(t *testing.T) {
	srv := New(testConfig(""), &agent.EchoAgent{})
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestWebhook_VoiceMessage(t *testing.T) {
	srv := New(testConfig(""), &agent.EchoAgent{})
	update := telegram.Update{
		UpdateID: 2,
		Message: &telegram.Message{
			MessageID: 1,
			Chat:      telegram.Chat{ID: 12345, Type: "private"},
			Voice:     &telegram.Voice{FileID: "voice-file-123", Duration: 5},
		},
	}
	w := postWebhook(srv, update, nil)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestWebhook_PhotoMessage(t *testing.T) {
	srv := New(testConfig(""), &agent.EchoAgent{})
	update := telegram.Update{
		UpdateID: 3,
		Message: &telegram.Message{
			MessageID: 1,
			Chat:      telegram.Chat{ID: 12345, Type: "private"},
			Photo: []telegram.PhotoSize{
				{FileID: "small", Width: 100, Height: 100},
				{FileID: "large", Width: 800, Height: 800},
			},
			Caption: "look at this",
		},
	}
	w := postWebhook(srv, update, nil)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestVerifySignature(t *testing.T) {
	if !verifySignature("abc", "abc") {
		t.Error("matching signatures should verify")
	}
	if verifySignature("abc", "xyz") {
		t.Error("different signatures should not verify")
	}
	if verifySignature("", "secret") {
		t.Error("empty vs non-empty should not verify")
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate(%q, 10) = %q", "short", got)
	}
	if got := truncate("this is a long string", 10); got != "this is a ..." {
		t.Errorf("truncate long = %q", got)
	}
}

func TestParseResponse_PlainText(t *testing.T) {
	text, meta := parseResponse("hello world")
	if text != "hello world" {
		t.Errorf("text = %q, want %q", text, "hello world")
	}
	if meta.SendVoice {
		t.Error("sendVoice should be false for plain text")
	}
}

func TestParseResponse_WithSendVoice(t *testing.T) {
	raw := "here is my response\n---\nsend_voice: true"
	text, meta := parseResponse(raw)
	if text != "here is my response" {
		t.Errorf("text = %q, want %q", text, "here is my response")
	}
	if !meta.SendVoice {
		t.Error("sendVoice should be true")
	}
}

func TestParseResponse_SendVoiceNoSpace(t *testing.T) {
	raw := "response text\n---\nsend_voice:true"
	text, meta := parseResponse(raw)
	if text != "response text" {
		t.Errorf("text = %q", text)
	}
	if !meta.SendVoice {
		t.Error("sendVoice should be true without space")
	}
}

func TestParseResponse_MetaWithoutVoice(t *testing.T) {
	raw := "response text\n---\nsome_other_flag: true"
	text, meta := parseResponse(raw)
	if text != "response text" {
		t.Errorf("text = %q", text)
	}
	if meta.SendVoice {
		t.Error("sendVoice should be false when not present in meta")
	}
}

func TestParseResponse_MultilineMeta(t *testing.T) {
	raw := "the actual response\n---\nfoo: bar\nsend_voice: true\nbaz: 42"
	text, meta := parseResponse(raw)
	if text != "the actual response" {
		t.Errorf("text = %q", text)
	}
	if !meta.SendVoice {
		t.Error("sendVoice should be true in multiline meta")
	}
}

func TestParseResponse_EmptyResponse(t *testing.T) {
	text, meta := parseResponse("")
	if text != "" {
		t.Errorf("text = %q, want empty", text)
	}
	if meta.SendVoice {
		t.Error("sendVoice should be false for empty response")
	}
}

func TestParseResponse_CodeChangesAndCommitMessage(t *testing.T) {
	raw := "ok\n---\ncode_changes: true\ncommit_message: self update"
	text, meta := parseResponse(raw)
	if text != "ok" {
		t.Fatalf("text=%q want=ok", text)
	}
	if !meta.CodeChanges {
		t.Fatal("expected code changes true")
	}
	if meta.CommitMessage != "self update" {
		t.Fatalf("commitMessage=%q", meta.CommitMessage)
	}
}

func TestWebhook_E2E_TelegramDelivery(t *testing.T) {
	type msgReq struct {
		ChatID int64  `json:"chat_id"`
		Text   string `json:"text"`
	}

	delivered := make(chan msgReq, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bottest-token/sendMessage" {
			t.Fatalf("path=%q want=/bottest-token/sendMessage", r.URL.Path)
		}
		var payload msgReq
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		delivered <- payload
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	srv := New(testConfig(""), &agent.EchoAgent{})
	srv.tg = telegram.NewClientWithOptions("test-token", ts.URL+"/bot", ts.Client())

	update := makeUpdate(1001, 12345, "hello from webhook")
	w := postWebhook(srv, update, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want=200", w.Code)
	}

	select {
	case got := <-delivered:
		if got.ChatID != 12345 {
			t.Fatalf("chat_id=%d want=12345", got.ChatID)
		}
		if got.Text != "echo: hello from webhook" {
			t.Fatalf("text=%q want=%q", got.Text, "echo: hello from webhook")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for telegram sendMessage call")
	}
}

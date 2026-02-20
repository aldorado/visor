package config

import (
	"os"
	"testing"
)

func clearEnv() {
	os.Unsetenv("TELEGRAM_BOT_TOKEN")
	os.Unsetenv("USER_PHONE_NUMBER")
	os.Unsetenv("PORT")
	os.Unsetenv("TELEGRAM_WEBHOOK_SECRET")
	os.Unsetenv("AGENT_BACKEND")
	os.Unsetenv("SELF_EVOLUTION_ENABLED")
	os.Unsetenv("SELF_EVOLUTION_REPO_DIR")
	os.Unsetenv("SELF_EVOLUTION_PUSH")
	os.Unsetenv("TZ")
}

func TestLoad_MinimalValid(t *testing.T) {
	clearEnv()
	os.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	os.Setenv("USER_PHONE_NUMBER", "12345")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TelegramBotToken != "test-token" {
		t.Errorf("token = %q, want %q", cfg.TelegramBotToken, "test-token")
	}
	if cfg.UserChatID != "12345" {
		t.Errorf("chatID = %q, want %q", cfg.UserChatID, "12345")
	}
	if cfg.Port != 8080 {
		t.Errorf("port = %d, want 8080", cfg.Port)
	}
	if cfg.AgentBackend != "echo" {
		t.Errorf("backend = %q, want %q", cfg.AgentBackend, "echo")
	}
	if cfg.TelegramWebhookSecret != "" {
		t.Errorf("webhook secret = %q, want empty", cfg.TelegramWebhookSecret)
	}
	if cfg.Timezone != "UTC" {
		t.Errorf("timezone = %q, want %q", cfg.Timezone, "UTC")
	}
}

func TestLoad_AllFields(t *testing.T) {
	clearEnv()
	os.Setenv("TELEGRAM_BOT_TOKEN", "tok")
	os.Setenv("USER_PHONE_NUMBER", "999")
	os.Setenv("PORT", "3000")
	os.Setenv("TELEGRAM_WEBHOOK_SECRET", "secret123")
	os.Setenv("AGENT_BACKEND", "pi")
	os.Setenv("TZ", "Europe/Vienna")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 3000 {
		t.Errorf("port = %d, want 3000", cfg.Port)
	}
	if cfg.TelegramWebhookSecret != "secret123" {
		t.Errorf("secret = %q, want %q", cfg.TelegramWebhookSecret, "secret123")
	}
	if cfg.AgentBackend != "pi" {
		t.Errorf("backend = %q, want %q", cfg.AgentBackend, "pi")
	}
	if cfg.Timezone != "Europe/Vienna" {
		t.Errorf("timezone = %q, want %q", cfg.Timezone, "Europe/Vienna")
	}
}

func TestLoad_MissingToken(t *testing.T) {
	clearEnv()
	os.Setenv("USER_PHONE_NUMBER", "12345")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestLoad_MissingChatID(t *testing.T) {
	clearEnv()
	os.Setenv("TELEGRAM_BOT_TOKEN", "tok")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing chat ID")
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	clearEnv()
	os.Setenv("TELEGRAM_BOT_TOKEN", "tok")
	os.Setenv("USER_PHONE_NUMBER", "123")
	os.Setenv("PORT", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid port")
	}
}

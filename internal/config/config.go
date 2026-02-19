package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	TelegramBotToken      string
	TelegramWebhookSecret string
	UserChatID            string
	Port                  int
	AgentBackend          string // "pi", "claude", "echo" (default: "echo")
	OpenAIAPIKey          string
	DataDir               string // base directory for runtime data (default: "data")
	HimalayaEnabled       bool
	HimalayaAccount       string
	HimalayaPollInterval  int // seconds
}

func Load() (*Config, error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}

	userChatID := os.Getenv("USER_PHONE_NUMBER")
	if userChatID == "" {
		return nil, fmt.Errorf("USER_PHONE_NUMBER is required")
	}

	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		var err error
		port, err = strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("PORT must be a number: %w", err)
		}
	}

	backend := os.Getenv("AGENT_BACKEND")
	if backend == "" {
		backend = "echo"
	}

	himalayaEnabled := os.Getenv("HIMALAYA_ENABLED") == "1" || os.Getenv("HIMALAYA_ENABLED") == "true"
	himalayaPollInterval := 60
	if s := os.Getenv("HIMALAYA_POLL_INTERVAL_SECONDS"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("HIMALAYA_POLL_INTERVAL_SECONDS must be a number: %w", err)
		}
		himalayaPollInterval = v
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}

	return &Config{
		TelegramBotToken:      token,
		TelegramWebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
		UserChatID:            userChatID,
		Port:                  port,
		AgentBackend:          backend,
		OpenAIAPIKey:          os.Getenv("OPENAI_API_KEY"),
		DataDir:               dataDir,
		HimalayaEnabled:       himalayaEnabled,
		HimalayaAccount:       os.Getenv("HIMALAYA_ACCOUNT"),
		HimalayaPollInterval:  himalayaPollInterval,
	}, nil
}

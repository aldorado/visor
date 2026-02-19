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
	ElevenLabsAPIKey      string
	ElevenLabsVoiceID     string
	HimalayaEnabled       bool
	HimalayaAccount       string
	HimalayaPollInterval  int // seconds
	LogLevel              string
	LogVerbose            bool
	OTELEnabled           bool
	OTELEndpoint          string
	OTELServiceName       string
	OTELEnvironment       string
	OTELInsecure          bool
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

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logVerbose := os.Getenv("LOG_VERBOSE") == "1" || os.Getenv("LOG_VERBOSE") == "true"

	otelEnabled := os.Getenv("OTEL_ENABLED") == "1" || os.Getenv("OTEL_ENABLED") == "true"
	otelServiceName := os.Getenv("OTEL_SERVICE_NAME")
	if otelServiceName == "" {
		otelServiceName = "visor"
	}
	otelEnvironment := os.Getenv("OTEL_ENVIRONMENT")
	if otelEnvironment == "" {
		otelEnvironment = "dev"
	}
	otelInsecure := os.Getenv("OTEL_INSECURE") == "1" || os.Getenv("OTEL_INSECURE") == "true"

	return &Config{
		TelegramBotToken:      token,
		TelegramWebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
		UserChatID:            userChatID,
		Port:                  port,
		AgentBackend:          backend,
		OpenAIAPIKey:          os.Getenv("OPENAI_API_KEY"),
		ElevenLabsAPIKey:      os.Getenv("ELEVENLABS_API_KEY"),
		ElevenLabsVoiceID:     os.Getenv("ELEVENLABS_VOICE_ID"),
		DataDir:               dataDir,
		HimalayaEnabled:       himalayaEnabled,
		HimalayaAccount:       os.Getenv("HIMALAYA_ACCOUNT"),
		HimalayaPollInterval:  himalayaPollInterval,
		LogLevel:              logLevel,
		LogVerbose:            logVerbose,
		OTELEnabled:           otelEnabled,
		OTELEndpoint:          os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		OTELServiceName:       otelServiceName,
		OTELEnvironment:       otelEnvironment,
		OTELInsecure:          otelInsecure,
	}, nil
}

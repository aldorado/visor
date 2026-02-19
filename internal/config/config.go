package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	TelegramBotToken string
	UserChatID       string
	Port             int
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

	return &Config{
		TelegramBotToken: token,
		UserChatID:       userChatID,
		Port:             port,
	}, nil
}

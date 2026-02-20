package setup

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ActionEnvelope struct {
	EnvSet            map[string]string `json:"env_set,omitempty"`
	EnvUnset          []string          `json:"env_unset,omitempty"`
	ValidateTelegram  bool              `json:"validate_telegram,omitempty"`
	WebhookURL        string            `json:"webhook_url,omitempty"`
	WebhookSecret     string            `json:"webhook_secret,omitempty"`
	CheckHealth       bool              `json:"check_health,omitempty"`
	LevelupEnvSet     map[string]string `json:"levelup_env_set,omitempty"`
	LevelupEnvUnset   []string          `json:"levelup_env_unset,omitempty"`
	EnableLevelups    []string          `json:"enable_levelups,omitempty"`
	DisableLevelups   []string          `json:"disable_levelups,omitempty"`
	ValidateLevelups  bool              `json:"validate_levelups,omitempty"`
	StartLevelups     bool              `json:"start_levelups,omitempty"`
	CheckLevelups     bool              `json:"check_levelups,omitempty"`
	SyncForgejoRemote bool              `json:"sync_forgejo_remote,omitempty"`
}

func ExtractActions(response string) (cleanText string, actions *ActionEnvelope, err error) {
	idx := 0
	for {
		start := strings.Index(response[idx:], "```json")
		if start == -1 {
			return response, nil, nil
		}
		start += idx

		endMarker := strings.Index(response[start+7:], "```")
		if endMarker == -1 {
			return response, nil, nil
		}

		blockStart := start + 7
		blockEnd := blockStart + endMarker
		jsonText := strings.TrimSpace(response[blockStart:blockEnd])
		if !strings.Contains(jsonText, "setup_actions") {
			idx = blockEnd + 3
			continue
		}

		var wrapper struct {
			SetupActions ActionEnvelope `json:"setup_actions"`
		}
		if err := json.Unmarshal([]byte(jsonText), &wrapper); err != nil {
			return response, nil, fmt.Errorf("parse setup_actions json: %w", err)
		}

		clean := strings.TrimSpace(response[:start] + response[blockEnd+3:])
		return clean, &wrapper.SetupActions, nil
	}
}

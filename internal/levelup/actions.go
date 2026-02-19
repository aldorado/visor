package levelup

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ActionEnvelope struct {
	EnvSet   map[string]string `json:"env_set,omitempty"`
	EnvUnset []string          `json:"env_unset,omitempty"`
	Enable   []string          `json:"enable,omitempty"`
	Disable  []string          `json:"disable,omitempty"`
	Validate bool              `json:"validate,omitempty"`
}

// ExtractActions parses levelup_actions JSON blocks from agent response.
// Returns clean text (JSON block removed) and parsed actions.
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
		if !strings.Contains(jsonText, "levelup_actions") {
			idx = blockEnd + 3
			continue
		}

		var wrapper struct {
			LevelupActions ActionEnvelope `json:"levelup_actions"`
		}
		if err := json.Unmarshal([]byte(jsonText), &wrapper); err != nil {
			return response, nil, fmt.Errorf("parse levelup_actions json: %w", err)
		}

		clean := strings.TrimSpace(response[:start] + response[blockEnd+3:])
		return clean, &wrapper.LevelupActions, nil
	}
}

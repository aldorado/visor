package email

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ActionEnvelope struct {
	EmailActions []OutgoingMessage `json:"email_actions"`
}

func ExtractActions(response string) (cleanText string, actions []OutgoingMessage, err error) {
	start := strings.Index(response, "```json")
	if start == -1 {
		return response, nil, nil
	}
	end := strings.Index(response[start+7:], "```")
	if end == -1 {
		return response, nil, nil
	}

	blockStart := start + len("```json")
	blockEnd := blockStart + end
	jsonText := strings.TrimSpace(response[blockStart:blockEnd])
	if jsonText == "" {
		return response, nil, nil
	}

	var envelope ActionEnvelope
	if err := json.Unmarshal([]byte(jsonText), &envelope); err != nil {
		return response, nil, fmt.Errorf("parse email_actions json: %w", err)
	}

	filtered := make([]OutgoingMessage, 0, len(envelope.EmailActions))
	for _, action := range envelope.EmailActions {
		if strings.TrimSpace(action.To) == "" {
			continue
		}
		filtered = append(filtered, action)
	}

	clean := strings.TrimSpace(response[:start] + response[blockEnd+3:])
	return clean, filtered, nil
}

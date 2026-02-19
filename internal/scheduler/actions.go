package scheduler

import (
	"encoding/json"
	"fmt"
	"strings"
)

type CreateAction struct {
	Prompt          string `json:"prompt"`
	RunAt           string `json:"run_at"`
	IntervalSeconds int64  `json:"interval_seconds,omitempty"`
}

type UpdateAction struct {
	ID              string `json:"id"`
	Prompt          string `json:"prompt,omitempty"`
	RunAt           string `json:"run_at,omitempty"`
	Recurring       *bool  `json:"recurring,omitempty"`
	IntervalSeconds *int64 `json:"interval_seconds,omitempty"`
}

type DeleteAction struct {
	ID string `json:"id"`
}

type ActionEnvelope struct {
	Create []CreateAction `json:"create,omitempty"`
	Update []UpdateAction `json:"update,omitempty"`
	Delete []DeleteAction `json:"delete,omitempty"`
	List   bool           `json:"list,omitempty"`
}

// ExtractActions parses schedule_actions JSON blocks from agent response.
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
		if !strings.Contains(jsonText, "schedule_actions") {
			idx = blockEnd + 3
			continue
		}

		var wrapper struct {
			ScheduleActions ActionEnvelope `json:"schedule_actions"`
		}
		if err := json.Unmarshal([]byte(jsonText), &wrapper); err != nil {
			return response, nil, fmt.Errorf("parse schedule_actions json: %w", err)
		}

		clean := strings.TrimSpace(response[:start] + response[blockEnd+3:])
		return clean, &wrapper.ScheduleActions, nil
	}
}

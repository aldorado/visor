package skills

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Action types for agent-authored skill management.

type CreateAction struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Triggers     []string `json:"triggers"`
	Run          string   `json:"run"`
	Script       string   `json:"script"`
	Dependencies []string `json:"dependencies"`
	Timeout      int      `json:"timeout"`
}

type EditAction struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Triggers    []string `json:"triggers,omitempty"`
	Run         string   `json:"run,omitempty"`
	Script      string   `json:"script,omitempty"`
	Timeout     int      `json:"timeout,omitempty"`
}

type DeleteAction struct {
	Name string `json:"name"`
}

type ActionEnvelope struct {
	Create []CreateAction `json:"create,omitempty"`
	Edit   []EditAction   `json:"edit,omitempty"`
	Delete []DeleteAction `json:"delete,omitempty"`
}

// ExtractActions parses skill_actions JSON blocks from agent response.
// Returns clean text (with JSON block removed) and parsed actions.
// Format in agent response:
//
//	```json
//	{"skill_actions": {"create": [...], "edit": [...], "delete": [...]}}
//	```
func ExtractActions(response string) (cleanText string, actions *ActionEnvelope, err error) {
	// find ```json block containing "skill_actions"
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

		if !strings.Contains(jsonText, "skill_actions") {
			idx = blockEnd + 3
			continue
		}

		var wrapper struct {
			SkillActions ActionEnvelope `json:"skill_actions"`
		}
		if err := json.Unmarshal([]byte(jsonText), &wrapper); err != nil {
			return response, nil, fmt.Errorf("parse skill_actions json: %w", err)
		}

		clean := strings.TrimSpace(response[:start] + response[blockEnd+3:])
		return clean, &wrapper.SkillActions, nil
	}
}

package contract

import (
	"encoding/json"
	"strings"
)

func parseMeta(resp *Response, block string) {
	lines := strings.Split(block, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		switch {
		case line == "send_voice: true" || line == "send_voice:true":
			resp.SendVoice = true
		case line == "send_voice: false" || line == "send_voice:false":
			resp.SendVoice = false
		case line == "code_changes: true" || line == "code_changes:true":
			resp.CodeChanges = true
		case line == "code_changes: false" || line == "code_changes:false":
			resp.CodeChanges = false
		case line == "conversation_finished: true" || line == "conversation_finished:true":
			resp.ConversationFinished = true
		case line == "conversation_finished: false" || line == "conversation_finished:false":
			resp.ConversationFinished = false
		case strings.HasPrefix(line, "commit_message:"):
			resp.CommitMessage = strings.TrimSpace(strings.TrimPrefix(line, "commit_message:"))
		case line == "git_push: true" || line == "git_push:true":
			resp.GitPush = true
		case line == "git_push: false" || line == "git_push:false":
			resp.GitPush = false
		case strings.HasPrefix(line, "git_push_dir:"):
			resp.GitPushDir = strings.TrimSpace(strings.TrimPrefix(line, "git_push_dir:"))
		case strings.HasPrefix(line, "memories_to_save:"):
			rest := strings.TrimSpace(strings.TrimPrefix(line, "memories_to_save:"))
			if rest != "" {
				if strings.HasPrefix(rest, "[") {
					var inline []string
					if err := json.Unmarshal([]byte(rest), &inline); err == nil {
						resp.MemoriesToSave = append(resp.MemoriesToSave, inline...)
						continue
					}
				}
				resp.MemoriesToSave = append(resp.MemoriesToSave, rest)
				continue
			}
			for i+1 < len(lines) {
				next := strings.TrimSpace(lines[i+1])
				if next == "" {
					i++
					continue
				}
				if strings.HasPrefix(next, "- ") {
					resp.MemoriesToSave = append(resp.MemoriesToSave, strings.TrimSpace(strings.TrimPrefix(next, "- ")))
					i++
					continue
				}
				break
			}
		}
	}
}

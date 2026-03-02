package contract

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Issues []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("response contract validation failed: %s", strings.Join(e.Issues, "; "))
}

func Validate(resp Response) error {
	issues := make([]string, 0, 4)
	if !resp.SendVoice && strings.TrimSpace(resp.ResponseText) == "" {
		issues = append(issues, "response_text is required when send_voice=false")
	}
	if resp.CodeChanges && strings.TrimSpace(resp.CommitMessage) == "" {
		issues = append(issues, "commit_message is required when code_changes=true")
	}
	for i, m := range resp.MemoriesToSave {
		if strings.TrimSpace(m) == "" {
			issues = append(issues, fmt.Sprintf("memories_to_save[%d] is empty", i))
		}
	}
	if len(issues) == 0 {
		return nil
	}
	return &ValidationError{Issues: issues}
}

func FixDefaults(resp *Response) bool {
	changed := false

	trimmed := strings.TrimSpace(resp.ResponseText)
	if trimmed != resp.ResponseText {
		resp.ResponseText = trimmed
		changed = true
	}

	for i := 0; i < len(resp.MemoriesToSave); i++ {
		m := strings.TrimSpace(resp.MemoriesToSave[i])
		if m == "" {
			resp.MemoriesToSave = append(resp.MemoriesToSave[:i], resp.MemoriesToSave[i+1:]...)
			i--
			changed = true
			continue
		}
		if m != resp.MemoriesToSave[i] {
			resp.MemoriesToSave[i] = m
			changed = true
		}
	}

	if resp.ConversationFinished && !hasGoodbyeIntent(resp.ResponseText) {
		resp.ConversationFinished = false
		changed = true
	}

	// zero-value bool defaults for CodeChanges/ConversationFinished are intentional.
	return changed
}

func hasGoodbyeIntent(text string) bool {
	s := strings.ToLower(strings.TrimSpace(text))
	if s == "" {
		return false
	}
	markers := []string{"bye", "goodbye", "ciao", "tschüss", "tschuess", "bis dann", "see you"}
	for _, marker := range markers {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

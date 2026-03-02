package contract

import (
	"encoding/json"
	"strings"
)

// Response is the canonical structured assistant response contract.
type Response struct {
	ResponseText         string
	SendVoice            bool
	CodeChanges          bool
	ConversationFinished bool
	CommitMessage        string
	GitPush              bool
	GitPushDir           string
	MemoriesToSave       []string
}

// JSONSchema returns a JSON schema for the structured response metadata.
func JSONSchema() string {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title":   "visor.response",
		"type":    "object",
		"properties": map[string]any{
			"response_text":         map[string]any{"type": "string"},
			"send_voice":            map[string]any{"type": "boolean"},
			"code_changes":          map[string]any{"type": "boolean"},
			"conversation_finished": map[string]any{"type": "boolean"},
			"commit_message":        map[string]any{"type": "string"},
			"git_push":              map[string]any{"type": "boolean"},
			"git_push_dir":          map[string]any{"type": "string"},
			"memories_to_save": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
		},
		"required": []string{"response_text", "send_voice", "code_changes", "conversation_finished"},
	}
	b, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}

func ParseRaw(raw string) Response {
	resp := Response{}
	parts := strings.SplitN(raw, "\n---\n", 2)
	resp.ResponseText = parts[0]
	if len(parts) != 2 {
		return resp
	}

	parseMeta(&resp, parts[1])
	return resp
}

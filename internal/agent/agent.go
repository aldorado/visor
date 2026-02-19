package agent

import "context"

// Agent is the interface for any AI backend (pi, claude, gemini, etc.)
type Agent interface {
	SendPrompt(ctx context.Context, prompt string) (string, error)
	Close() error
}

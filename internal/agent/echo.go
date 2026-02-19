package agent

import (
	"context"
	"fmt"
)

// EchoAgent is a stub backend that echoes messages. Used for testing.
type EchoAgent struct{}

func (e *EchoAgent) SendPrompt(_ context.Context, prompt string) (string, error) {
	return fmt.Sprintf("echo: %s", prompt), nil
}

func (e *EchoAgent) Close() error { return nil }

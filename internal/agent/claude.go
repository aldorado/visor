package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// Claude Code stream-json event types

type claudeEvent struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type claudeMessage struct {
	Role    string         `json:"role"`
	Content []claudeBlock  `json:"content,omitempty"`
}

type claudeBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type claudeResult struct {
	IsError bool   `json:"is_error"`
	Duration float64 `json:"duration_ms"`
}

// ClaudeAgent implements Agent using `claude -p --output-format stream-json`.
// Each prompt spawns a new process (no persistent RPC mode available).
type ClaudeAgent struct {
	timeout time.Duration
}

func NewClaudeAgent(timeout time.Duration) *ClaudeAgent {
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	return &ClaudeAgent{timeout: timeout}
}

func (c *ClaudeAgent) SendPrompt(ctx context.Context, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", "-p", "--output-format", "stream-json", "--verbose", prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("claude: stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("claude: start: %w", err)
	}

	var response strings.Builder
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event claudeEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			log.Printf("claude: skipping unparseable line: %.100s", line)
			continue
		}

		switch event.Type {
		case "assistant":
			var msg claudeMessage
			if err := json.Unmarshal(event.Message, &msg); err != nil {
				log.Printf("claude: bad assistant message: %v", err)
				continue
			}
			for _, block := range msg.Content {
				if block.Type == "text" {
					response.WriteString(block.Text)
				}
			}

		case "result":
			var res claudeResult
			if err := json.Unmarshal(event.Result, &res); err == nil && res.IsError {
				return response.String(), fmt.Errorf("claude: result reported error")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return response.String(), fmt.Errorf("claude: read stdout: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return response.String(), fmt.Errorf("claude: timeout")
		}
		return response.String(), fmt.Errorf("claude: exit: %w", err)
	}

	return response.String(), nil
}

func (c *ClaudeAgent) Close() error { return nil }

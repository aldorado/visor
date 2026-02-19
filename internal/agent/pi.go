package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// Pi RPC JSON-lines protocol types

type piCommand struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type piEvent struct {
	Type    string `json:"type"`
	Success *bool  `json:"success,omitempty"`
	Error   string `json:"error,omitempty"`

	// for message_update events
	AssistantMessageEvent *piAssistantEvent `json:"assistantMessageEvent,omitempty"`
}

type piAssistantEvent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// PiAgent implements Agent using `pi --mode rpc`.
type PiAgent struct {
	pm *ProcessManager
	mu sync.Mutex // serialize prompts (one at a time over shared stdin/stdout)
}

func NewPiAgent(cfg ProcessConfig) *PiAgent {
	cfg.Command = "pi"
	cfg.Args = []string{"--mode", "rpc"}
	return &PiAgent{
		pm: NewProcessManager(cfg),
	}
}

func (p *PiAgent) Start() error {
	return p.pm.Start()
}

func (p *PiAgent) SendPrompt(ctx context.Context, prompt string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	timeout := p.pm.cfg.PromptTimeout
	if timeout == 0 {
		timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// send prompt command
	cmd := piCommand{Type: "prompt", Message: prompt}
	data, err := json.Marshal(cmd)
	if err != nil {
		return "", fmt.Errorf("pi: marshal command: %w", err)
	}

	stdin := p.pm.Stdin()
	if stdin == nil {
		return "", fmt.Errorf("pi: process not running")
	}

	if _, err := fmt.Fprintf(stdin, "%s\n", data); err != nil {
		return "", fmt.Errorf("pi: write stdin: %w", err)
	}

	// read events until agent_end
	var response strings.Builder
	scanner := p.pm.Scanner()
	if scanner == nil {
		return "", fmt.Errorf("pi: scanner not available")
	}

	for {
		select {
		case <-ctx.Done():
			return response.String(), fmt.Errorf("pi: timeout waiting for response")
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return response.String(), fmt.Errorf("pi: read stdout: %w", err)
			}
			return response.String(), fmt.Errorf("pi: process closed stdout")
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		var event piEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			log.Printf("pi: skipping unparseable line: %s", line)
			continue
		}

		switch event.Type {
		case "response":
			// ack/nack for our command
			if event.Success != nil && !*event.Success {
				return "", fmt.Errorf("pi: command rejected: %s", event.Error)
			}

		case "message_update":
			if event.AssistantMessageEvent != nil && event.AssistantMessageEvent.Type == "text_delta" {
				response.WriteString(event.AssistantMessageEvent.Text)
			}

		case "agent_end":
			return response.String(), nil
		}
	}
}

func (p *PiAgent) Close() error {
	return p.pm.Stop()
}

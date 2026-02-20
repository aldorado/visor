package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"visor/internal/observability"
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
	Message               *piMessage        `json:"message,omitempty"`
}

type piAssistantEvent struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Delta string `json:"delta,omitempty"`
}

type piMessage struct {
	Role    string           `json:"role"`
	Content []piMessageBlock `json:"content,omitempty"`
}

type piMessageBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// PiAgent implements Agent using `pi --mode rpc`.
type PiAgent struct {
	toolsPM      *ProcessManager
	toolsCfg     ProcessConfig
	toolsMu      sync.Mutex
	toolsStarted bool
	mu           sync.Mutex // serialize prompts (one at a time over shared stdin/stdout)
	log          *observability.Logger
}

func NewPiAgent(cfg ProcessConfig) *PiAgent {
	toolsCfg := cfg
	toolsCfg.Command = "pi"
	toolsCfg.Args = []string{"--mode", "rpc"}

	return &PiAgent{
		toolsCfg: toolsCfg,
		log:      observability.Component("agent.pi"),
	}
}

func (p *PiAgent) Start() error {
	_, err := p.ensureToolsProcess()
	return err
}

func (p *PiAgent) SendPrompt(ctx context.Context, prompt string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	pm, err := p.ensureToolsProcess()
	if err != nil {
		return "", err
	}
	p.log.Debug(ctx, "pi mode selected", "mode", "tools")

	timeout := pm.cfg.PromptTimeout
	if timeout == 0 {
		timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// send prompt command
	cmd := piCommand{Type: "prompt", Message: withExecutionGuardrail(prompt)}
	data, err := json.Marshal(cmd)
	if err != nil {
		return "", fmt.Errorf("pi: marshal command: %w", err)
	}

	stdin := pm.Stdin()
	if stdin == nil {
		return "", fmt.Errorf("pi: process not running")
	}

	if _, err := fmt.Fprintf(stdin, "%s\n", data); err != nil {
		return "", fmt.Errorf("pi: write stdin: %w", err)
	}

	// read events until agent_end
	var response strings.Builder
	scanner := pm.Scanner()
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
			p.log.Warn(ctx, "pi event parse failed", "line_preview", truncateLine(line, 120), "error", err.Error())
			continue
		}

		switch event.Type {
		case "response":
			if event.Success != nil && !*event.Success {
				return "", fmt.Errorf("pi: command rejected: %s", event.Error)
			}
		case "message_update":
			if event.AssistantMessageEvent != nil {
				switch event.AssistantMessageEvent.Type {
				case "text_delta", "output_text_delta":
					if event.AssistantMessageEvent.Text != "" {
						response.WriteString(event.AssistantMessageEvent.Text)
					} else if event.AssistantMessageEvent.Delta != "" {
						response.WriteString(event.AssistantMessageEvent.Delta)
					}
				}
			}
		case "message_end", "turn_end":
			if response.Len() == 0 && event.Message != nil && event.Message.Role == "assistant" {
				for _, block := range event.Message.Content {
					if block.Type == "text" && block.Text != "" {
						if response.Len() > 0 {
							response.WriteString("\n")
						}
						response.WriteString(block.Text)
					}
				}
			}
		case "agent_end":
			return response.String(), nil
		}
	}
}

func (p *PiAgent) ensureToolsProcess() (*ProcessManager, error) {
	p.toolsMu.Lock()
	defer p.toolsMu.Unlock()

	if p.toolsPM == nil {
		p.toolsPM = NewProcessManager(p.toolsCfg)
	}
	if !p.toolsStarted {
		if err := p.toolsPM.Start(); err != nil {
			return nil, fmt.Errorf("pi tools start: %w", err)
		}
		p.toolsStarted = true
	}
	return p.toolsPM, nil
}

func (p *PiAgent) Close() error {
	if p.toolsPM != nil {
		if err := p.toolsPM.Stop(); err != nil {
			return err
		}
	}
	return nil
}

func withExecutionGuardrail(prompt string) string {
	guardrail := "[runtime execution policy]\n" +
		"you have direct access to tools in this runtime (read, bash, edit, write and related capabilities).\n" +
		"do not ask the user to run commands or inspect files when you can do it yourself.\n" +
		"for checks/status questions, run the commands yourself and return results directly.\n" +
		"only ask the user for input that is genuinely unavailable in the runtime (e.g. missing credentials or personal preference).\n\n"
	return guardrail + prompt
}

func truncateLine(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

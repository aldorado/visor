package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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
	Usage   *piUsage         `json:"usage,omitempty"`
}

type piUsage struct {
	Input       int `json:"input"`
	Output      int `json:"output"`
	TotalTokens int `json:"totalTokens"`
}

type piMessageBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// PiAgent implements Agent using `pi --mode rpc`.
type PiAgent struct {
	toolsPM             *ProcessManager
	toolsCfg            ProcessConfig
	toolsMu             sync.Mutex
	toolsStarted        bool
	model               string
	contextWindowTokens int
	handoffThreshold    float64
	handoffContext      string
	lastInputTokens     int
	mu                  sync.Mutex // serialize prompts (one at a time over shared stdin/stdout)
	log                 *observability.Logger
}

func NewPiAgent(cfg ProcessConfig) *PiAgent {
	toolsCfg := cfg
	toolsCfg.Command = "pi"

	model := strings.TrimSpace(os.Getenv("PI_MODEL"))
	toolsCfg.Args = piRPCArgs(model)

	return &PiAgent{
		toolsCfg:            toolsCfg,
		model:               model,
		contextWindowTokens: contextWindowTokensFromEnv(),
		handoffThreshold:    handoffThresholdFromEnv(),
		log:                 observability.Component("agent.pi"),
	}
}

func (p *PiAgent) Start() error {
	_, err := p.ensureToolsProcess()
	return err
}

func (p *PiAgent) SetModel(model string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	model = strings.TrimSpace(model)
	p.toolsMu.Lock()
	defer p.toolsMu.Unlock()

	p.model = model
	p.lastInputTokens = 0
	p.toolsCfg.Args = piRPCArgs(model)

	if p.toolsPM != nil {
		_ = p.toolsPM.Stop()
		p.toolsPM = nil
		p.toolsStarted = false
	}
	return nil
}

func (p *PiAgent) Model() string {
	p.toolsMu.Lock()
	defer p.toolsMu.Unlock()
	return p.model
}

func (p *PiAgent) BackendLabel() string {
	p.toolsMu.Lock()
	model := p.model
	inputTokens := p.lastInputTokens
	ctxWindow := p.contextWindowTokens
	p.toolsMu.Unlock()

	label := "pi"
	if model != "" {
		label = "pi/" + model
	}
	if inputTokens > 0 && ctxWindow > 0 {
		usagePct := 100 * float64(inputTokens) / float64(ctxWindow)
		label += fmt.Sprintf(" · ctx %.1f%%", usagePct)
	}
	return label
}

func (p *PiAgent) SendPrompt(ctx context.Context, prompt string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	pm, err := p.ensureToolsProcess()
	if err != nil {
		return "", err
	}
	p.log.Debug(ctx, "pi mode selected", "mode", "tools", "session", "persistent")

	guarded := withExecutionGuardrail(prompt)
	if strings.TrimSpace(p.handoffContext) != "" {
		guarded = p.handoffContext + "\n\n" + guarded
		p.handoffContext = ""
	}

	response, inputTokens, err := p.sendPromptOnce(ctx, pm, guarded)
	if err != nil {
		return response, err
	}
	if looksLikeDeferral(response) {
		p.log.Warn(ctx, "pi returned deferral-style response, retrying with hard guardrail")
		hard := guarded + "\n[critical enforcement]\n" +
			"your previous answer violated policy. do not ask the user to run commands. " +
			"run the required checks yourself now and return concrete results only."
		response, inputTokens, err = p.sendPromptOnce(ctx, pm, hard)
		if err != nil {
			return response, err
		}
	}

	p.maybeHandoff(ctx, pm, inputTokens)
	return response, nil
}

func (p *PiAgent) sendPromptOnce(ctx context.Context, pm *ProcessManager, prompt string) (string, int, error) {
	timeout := pm.cfg.PromptTimeout
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := piCommand{Type: "prompt", Message: prompt}
	data, err := json.Marshal(cmd)
	if err != nil {
		return "", 0, fmt.Errorf("pi: marshal command: %w", err)
	}

	stdin := pm.Stdin()
	if stdin == nil {
		return "", 0, fmt.Errorf("pi: process not running")
	}
	if _, err := fmt.Fprintf(stdin, "%s\n", data); err != nil {
		return "", 0, fmt.Errorf("pi: write stdin: %w", err)
	}

	var response strings.Builder
	inputTokens := 0

	scanner := pm.Scanner()
	if scanner == nil {
		return "", 0, fmt.Errorf("pi: scanner not available")
	}

	for {
		select {
		case <-ctx.Done():
			return response.String(), inputTokens, fmt.Errorf("pi: timeout waiting for response")
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return response.String(), inputTokens, fmt.Errorf("pi: read stdout: %w", err)
			}
			return response.String(), inputTokens, fmt.Errorf("pi: process closed stdout")
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
				return "", inputTokens, fmt.Errorf("pi: command rejected: %s", event.Error)
			}
		case "message_update":
			if event.AssistantMessageEvent != nil {
				switch event.AssistantMessageEvent.Type {
				case "text_delta", "output_text_delta":
					if event.AssistantMessageEvent.Text != "" {
						response.WriteString(event.AssistantMessageEvent.Text)
						reportProgress(ctx, event.AssistantMessageEvent.Text)
					} else if event.AssistantMessageEvent.Delta != "" {
						response.WriteString(event.AssistantMessageEvent.Delta)
						reportProgress(ctx, event.AssistantMessageEvent.Delta)
					}
				}
			}
		case "message_end", "turn_end":
			if event.Message != nil && event.Message.Usage != nil && event.Message.Usage.Input > 0 {
				inputTokens = event.Message.Usage.Input
			}
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
			return response.String(), inputTokens, nil
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

func (p *PiAgent) freshToolsProcess() (*ProcessManager, error) {
	p.toolsMu.Lock()
	defer p.toolsMu.Unlock()

	if p.toolsPM != nil {
		_ = p.toolsPM.Stop()
		p.toolsPM = nil
		p.toolsStarted = false
	}

	pm := NewProcessManager(p.toolsCfg)
	if err := pm.Start(); err != nil {
		return nil, fmt.Errorf("pi tools fresh start: %w", err)
	}
	p.toolsPM = pm
	p.toolsStarted = true
	return pm, nil
}

func (p *PiAgent) Close() error {
	if p.toolsPM != nil {
		if err := p.toolsPM.Stop(); err != nil {
			return err
		}
	}
	return nil
}

func (p *PiAgent) maybeHandoff(ctx context.Context, pm *ProcessManager, inputTokens int) {
	p.toolsMu.Lock()
	p.lastInputTokens = inputTokens
	ctxWindow := p.contextWindowTokens
	threshold := p.handoffThreshold
	p.toolsMu.Unlock()

	if inputTokens <= 0 || ctxWindow <= 0 {
		return
	}

	usageRatio := float64(inputTokens) / float64(ctxWindow)
	p.log.Info(ctx, "pi context usage", "input_tokens", inputTokens, "context_window_tokens", ctxWindow, "usage_pct", fmt.Sprintf("%.1f", usageRatio*100))
	if usageRatio < threshold {
		return
	}

	p.log.Warn(ctx, "pi context above threshold, running handoff + restart", "threshold_pct", fmt.Sprintf("%.1f", threshold*100))
	handoffPrompt := "create a compact handoff summary for a fresh rpc session. include: user intent, active tasks, constraints, important decisions, and immediate next steps. keep it under 2200 characters and use plain text."
	handoffCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	summary, _, err := p.sendPromptOnce(handoffCtx, pm, handoffPrompt)
	if err != nil {
		p.log.Warn(ctx, "pi handoff summary failed", "error", err.Error())
	} else if strings.TrimSpace(summary) != "" {
		p.handoffContext = "[handoff context from previous rpc session]\n" + strings.TrimSpace(summary)
	}

	if err := pm.Restart(); err != nil {
		p.log.Error(ctx, "pi restart after handoff failed", "error", err.Error())
		return
	}
	p.log.Info(ctx, "pi restarted after handoff")
}

func contextWindowTokensFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("PI_CONTEXT_WINDOW_TOKENS"))
	if raw == "" {
		return 200000
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 200000
	}
	return v
}

func handoffThresholdFromEnv() float64 {
	raw := strings.TrimSpace(os.Getenv("PI_HANDOFF_THRESHOLD"))
	if raw == "" {
		return 0.80
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0.80
	}
	if v <= 0 || v >= 1 {
		return 0.80
	}
	return v
}

func piRPCArgs(model string) []string {
	args := []string{"--mode", "rpc", "--no-session"}
	if model != "" {
		args = append(args, "--model", model)
	}
	return args
}

func withExecutionGuardrail(prompt string) string {
	guardrail := "[runtime execution policy]\n" +
		"you have direct access to tools in this runtime (read, bash, edit, write and related capabilities).\n" +
		"do not ask the user to run commands or inspect files when you can do it yourself.\n" +
		"for checks/status questions, run the commands yourself and return results directly.\n" +
		"only ask the user for input that is genuinely unavailable in the runtime (e.g. missing credentials or personal preference).\n\n"
	return guardrail + prompt
}

func looksLikeDeferral(s string) bool {
	l := strings.ToLower(strings.TrimSpace(s))
	patterns := []string{
		"kann ich von hier nicht direkt sehen",
		"ich kann hier",
		"schick mir",
		"wenn du kurz",
		"run this",
		"please run",
		"i can't access",
		"i can’t access",
	}
	for _, p := range patterns {
		if strings.Contains(l, p) {
			return true
		}
	}
	return false
}

func truncateLine(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

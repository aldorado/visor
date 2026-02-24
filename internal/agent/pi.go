package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Role     string           `json:"role"`
	Content  []piMessageBlock `json:"content,omitempty"`
	Usage    *piUsage         `json:"usage,omitempty"`
	Provider string           `json:"provider,omitempty"`
	Model    string           `json:"model,omitempty"`
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
	modelProvider       string
	modelSource         string
	modelStatePath      string
	contextWindowTokens int
	handoffThreshold    float64
	handoffContext      string
	lastInputTokens     int
	sessionInputTokens  int
	mu                  sync.Mutex // serialize prompts (one at a time over shared stdin/stdout)
	log                 *observability.Logger
}

func NewPiAgent(cfg ProcessConfig) *PiAgent {
	return NewPiAgentWithModelState(cfg, filepath.Join(dataDirFromEnv(), "current-model.json"))
}

func NewPiAgentWithModelState(cfg ProcessConfig, modelStatePath string) *PiAgent {
	toolsCfg := cfg
	toolsCfg.Command = "pi"

	model, provider, err := loadCurrentModel(modelStatePath)
	if err != nil {
		observability.Component("agent.pi").Warn(nil, "load current model failed", "path", modelStatePath, "error", err.Error())
	}
	toolsCfg.Args = piRPCArgs(model)

	source := "runtime"
	if model != "" {
		source = "state-file"
	}

	return &PiAgent{
		toolsCfg:            toolsCfg,
		model:               model,
		modelProvider:       provider,
		modelSource:         source,
		modelStatePath:      modelStatePath,
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
	p.modelProvider = ""
	p.modelSource = "runtime"
	p.lastInputTokens = 0
	p.sessionInputTokens = 0
	p.toolsCfg.Args = piRPCArgs(model)

	if err := saveCurrentModel(p.modelStatePath, model, ""); err != nil {
		return err
	}

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

func (p *PiAgent) ModelStatus() ModelStatus {
	p.toolsMu.Lock()
	status := ModelStatus{
		Backend:  "pi",
		Model:    p.model,
		Provider: p.modelProvider,
		Source:   p.modelSource,
	}
	statePath := p.modelStatePath
	p.toolsMu.Unlock()

	stateModel, stateProvider, stateUpdatedAt, err := readCurrentModelState(statePath)
	if err == nil {
		status.StateModel = stateModel
		status.StateProvider = stateProvider
		status.StateUpdatedAt = stateUpdatedAt
	}
	return status
}

func (p *PiAgent) BackendLabel() string {
	p.toolsMu.Lock()
	model := p.model
	inputTokens := p.sessionInputTokens
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
			if event.Message != nil {
				p.updateModelFromMessage(ctx, event.Message)
			}
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
		p.lastInputTokens = 0
		p.sessionInputTokens = 0
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
	p.lastInputTokens = 0
	p.sessionInputTokens = 0
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
	if inputTokens > 0 {
		p.sessionInputTokens += inputTokens
	}
	sessionTokens := p.sessionInputTokens
	ctxWindow := p.contextWindowTokens
	threshold := p.handoffThreshold
	p.toolsMu.Unlock()

	if sessionTokens <= 0 || ctxWindow <= 0 {
		return
	}

	usageRatio := float64(sessionTokens) / float64(ctxWindow)
	p.log.Info(ctx, "pi context usage",
		"turn_input_tokens", inputTokens,
		"session_input_tokens", sessionTokens,
		"context_window_tokens", ctxWindow,
		"usage_pct", fmt.Sprintf("%.1f", usageRatio*100),
	)
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
	p.toolsMu.Lock()
	p.lastInputTokens = 0
	p.sessionInputTokens = 0
	p.toolsMu.Unlock()
	p.log.Info(ctx, "pi restarted after handoff")
}

func (p *PiAgent) updateModelFromMessage(ctx context.Context, msg *piMessage) {
	if msg == nil {
		return
	}
	model := strings.TrimSpace(msg.Model)
	if model == "" {
		return
	}
	provider := strings.TrimSpace(msg.Provider)

	p.toolsMu.Lock()
	changed := model != p.model || provider != p.modelProvider
	if changed {
		p.model = model
		p.modelProvider = provider
		p.modelSource = "runtime"
		p.toolsCfg.Args = piRPCArgs(model)
	}
	p.toolsMu.Unlock()

	if !changed {
		return
	}
	if err := saveCurrentModel(p.modelStatePath, model, provider); err != nil {
		p.log.Warn(ctx, "save current model failed", "path", p.modelStatePath, "error", err.Error())
	}
}

func dataDirFromEnv() string {
	dataDir := strings.TrimSpace(os.Getenv("DATA_DIR"))
	if dataDir == "" {
		return "data"
	}
	return dataDir
}

type currentModelState struct {
	Model     string `json:"model"`
	Provider  string `json:"provider,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

func loadCurrentModel(path string) (model string, provider string, err error) {
	model, provider, _, err = readCurrentModelState(path)
	return model, provider, err
}

func readCurrentModelState(path string) (model string, provider string, updatedAt string, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", "", "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", "", nil
		}
		return "", "", "", fmt.Errorf("read model state: %w", err)
	}
	var s currentModelState
	if err := json.Unmarshal(data, &s); err != nil {
		return "", "", "", fmt.Errorf("parse model state: %w", err)
	}
	return strings.TrimSpace(s.Model), strings.TrimSpace(s.Provider), strings.TrimSpace(s.UpdatedAt), nil
}

func saveCurrentModel(path, model, provider string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir model state dir: %w", err)
	}
	payload := currentModelState{
		Model:     strings.TrimSpace(model),
		Provider:  strings.TrimSpace(provider),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal model state: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write model state: %w", err)
	}
	return nil
}

func contextWindowTokensFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("PI_CONTEXT_WINDOW_TOKENS"))
	if raw == "" {
		return 64000
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 64000
	}
	return v
}

func handoffThresholdFromEnv() float64 {
	raw := strings.TrimSpace(os.Getenv("PI_HANDOFF_THRESHOLD"))
	if raw == "" {
		return 0.60
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0.60
	}
	if v <= 0 || v >= 1 {
		return 0.60
	}
	return v
}

func piRPCArgs(model string) []string {
	args := []string{"--mode", "rpc"}
	if os.Getenv("PI_NO_SESSION") == "1" || strings.EqualFold(os.Getenv("PI_NO_SESSION"), "true") {
		args = append(args, "--no-session")
	}
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

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"visor/internal/observability"
)

// GeminiAgent implements Agent using Gemini CLI in headless stream-json mode.
// It prefers a local `gemini` binary and falls back to `npx @google/gemini-cli`.
type GeminiAgent struct {
	timeout      time.Duration
	log          *observability.Logger
	mu           sync.Mutex
	lastSuccess  time.Time
	resumeWindow time.Duration
}

func NewGeminiAgent(timeout time.Duration) *GeminiAgent {
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	return &GeminiAgent{
		timeout:      timeout,
		log:          observability.Component("agent.gemini"),
		resumeWindow: geminiResumeWindow(),
	}
}

func (g *GeminiAgent) SendPrompt(ctx context.Context, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	binary, prefix, err := resolveGeminiCommand()
	if err != nil {
		return "", err
	}

	model := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	if model == "" {
		model = "auto-gemini-3"
	}

	useResume := g.shouldResume()
	start := time.Now()
	g.log.Info(ctx, "gemini request start", "model", model, "prompt_len", len(prompt), "resume_latest", useResume, "resume_window_min", int(g.resumeWindow/time.Minute))

	args := append(prefix, "-m", model)
	if useResume {
		args = append(args, "--resume", "latest")
	}
	args = append(args, "-p", prompt, "--output-format", "stream-json")
	cmd := exec.CommandContext(ctx, binary, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("gemini: stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("gemini: stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("gemini: start: %w", err)
	}

	var response strings.Builder
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	firstTokenMs := int64(-1)
	stdoutLines := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		stdoutLines++

		chunk, lineErr := parseGeminiStreamLine(line)
		if lineErr != nil {
			g.log.Warn(ctx, "gemini stream parse error", "error", lineErr.Error())
			return response.String(), lineErr
		}
		if chunk != "" {
			if firstTokenMs < 0 {
				firstTokenMs = time.Since(start).Milliseconds()
				g.log.Info(ctx, "gemini first token", "model", model, "latency_ms", firstTokenMs)
			}
			response.WriteString(chunk)
		}
	}

	if err := scanner.Err(); err != nil {
		return response.String(), fmt.Errorf("gemini: read stdout: %w", err)
	}

	stderrBytes, _ := io.ReadAll(stderr)

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			g.log.Warn(ctx, "gemini request timeout", "model", model, "duration_ms", time.Since(start).Milliseconds())
			return response.String(), fmt.Errorf("gemini: timeout")
		}
		stderrMsg := strings.TrimSpace(string(stderrBytes))
		if stderrMsg != "" {
			g.log.Warn(ctx, "gemini request failed", "model", model, "duration_ms", time.Since(start).Milliseconds(), "stderr_len", len(stderrMsg))
			return response.String(), fmt.Errorf("gemini: %s", stderrMsg)
		}
		return response.String(), fmt.Errorf("gemini: exit: %w", err)
	}

	if response.Len() == 0 {
		stderrMsg := strings.TrimSpace(string(stderrBytes))
		if stderrMsg != "" {
			g.log.Warn(ctx, "gemini empty response with stderr", "model", model, "duration_ms", time.Since(start).Milliseconds(), "stderr_len", len(stderrMsg))
			return "", fmt.Errorf("gemini: %s", stderrMsg)
		}
	}

	g.markSuccess()
	g.log.Info(ctx, "gemini request done", "model", model, "duration_ms", time.Since(start).Milliseconds(), "first_token_ms", firstTokenMs, "stdout_lines", stdoutLines, "response_len", response.Len())
	return response.String(), nil
}

func (g *GeminiAgent) Close() error { return nil }

func (g *GeminiAgent) shouldResume() bool {
	if g.resumeWindow <= 0 {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.lastSuccess.IsZero() {
		return false
	}
	return time.Since(g.lastSuccess) <= g.resumeWindow
}

func (g *GeminiAgent) markSuccess() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.lastSuccess = time.Now()
}

func geminiResumeWindow() time.Duration {
	raw := strings.TrimSpace(os.Getenv("GEMINI_RESUME_WINDOW_MINUTES"))
	if raw == "" {
		return 20 * time.Minute
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return 20 * time.Minute
	}
	return time.Duration(v) * time.Minute
}

func resolveGeminiCommand() (binary string, prefixArgs []string, err error) {
	if _, lookErr := exec.LookPath("gemini"); lookErr == nil {
		return "gemini", nil, nil
	}
	if _, lookErr := exec.LookPath("npx"); lookErr == nil {
		return "npx", []string{"-y", "@google/gemini-cli"}, nil
	}
	return "", nil, fmt.Errorf("gemini: neither 'gemini' nor 'npx' found on PATH")
}

func parseGeminiStreamLine(line string) (string, error) {
	var event struct {
		Type    string          `json:"type"`
		Role    string          `json:"role,omitempty"`
		Content string          `json:"content,omitempty"`
		Message json.RawMessage `json:"message,omitempty"`
		Result  json.RawMessage `json:"result,omitempty"`
		Error   json.RawMessage `json:"error,omitempty"`
		Text    string          `json:"text,omitempty"`
	}

	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return "", fmt.Errorf("gemini event parse failed: %w", err)
	}

	switch event.Type {
	case "error":
		msg := extractJSONText(event.Error)
		if msg == "" {
			msg = extractJSONText([]byte(line))
		}
		if msg == "" {
			msg = "unknown gemini error"
		}
		return "", fmt.Errorf("gemini: %s", msg)
	case "message":
		var out strings.Builder
		if event.Role == "assistant" && event.Content != "" {
			out.WriteString(event.Content)
		}
		if event.Text != "" {
			out.WriteString(event.Text)
		}
		out.WriteString(extractJSONText(event.Message))
		return out.String(), nil
	case "result":
		var out strings.Builder
		if event.Text != "" {
			out.WriteString(event.Text)
		}
		out.WriteString(extractJSONText(event.Result))
		return out.String(), nil
	default:
		if event.Role == "assistant" && event.Content != "" {
			return event.Content, nil
		}
		if event.Text != "" {
			return event.Text, nil
		}
		return "", nil
	}
}

func extractJSONText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}

	var chunks []string
	var walk func(any)
	walk = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			if s, ok := t["text"].(string); ok && s != "" {
				chunks = append(chunks, s)
			}
			if s, ok := t["content"].(string); ok && s != "" {
				chunks = append(chunks, s)
			}
			if s, ok := t["message"].(string); ok && s != "" {
				chunks = append(chunks, s)
			}
			for k, child := range t {
				if k == "text" || k == "content" || k == "message" {
					continue // already extracted above, don't recurse into same key
				}
				walk(child)
			}
		case []any:
			for _, child := range t {
				walk(child)
			}
		}
	}

	walk(value)
	return strings.Join(chunks, "")
}

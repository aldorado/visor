package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"visor/internal/observability"
)

// GeminiAgent implements Agent using Gemini CLI in headless stream-json mode.
// It prefers a local `gemini` binary and falls back to `npx @google/gemini-cli`.
type GeminiAgent struct {
	timeout time.Duration
	log     *observability.Logger
}

func NewGeminiAgent(timeout time.Duration) *GeminiAgent {
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	return &GeminiAgent{timeout: timeout, log: observability.Component("agent.gemini")}
}

func (g *GeminiAgent) SendPrompt(ctx context.Context, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	binary, prefix, err := resolveGeminiCommand()
	if err != nil {
		return "", err
	}

	args := append(prefix, "-p", prompt, "--output-format", "stream-json")
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

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		chunk, lineErr := parseGeminiStreamLine(line)
		if lineErr != nil {
			return response.String(), lineErr
		}
		if chunk != "" {
			response.WriteString(chunk)
		}
	}

	if err := scanner.Err(); err != nil {
		return response.String(), fmt.Errorf("gemini: read stdout: %w", err)
	}

	stderrBytes, _ := io.ReadAll(stderr)

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return response.String(), fmt.Errorf("gemini: timeout")
		}
		stderrMsg := strings.TrimSpace(string(stderrBytes))
		if stderrMsg != "" {
			return response.String(), fmt.Errorf("gemini: %s", stderrMsg)
		}
		return response.String(), fmt.Errorf("gemini: exit: %w", err)
	}

	if response.Len() == 0 {
		stderrMsg := strings.TrimSpace(string(stderrBytes))
		if stderrMsg != "" {
			return "", fmt.Errorf("gemini: %s", stderrMsg)
		}
	}

	return response.String(), nil
}

func (g *GeminiAgent) Close() error { return nil }

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
			for _, child := range t {
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

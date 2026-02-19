package email

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type HimalayaClient struct {
	account string
	runner  func(ctx context.Context, name string, args ...string) ([]byte, error)
}

func NewHimalayaClient(account string) *HimalayaClient {
	return &HimalayaClient{
		account: account,
		runner:  runCommand,
	}
}

type envelopeListItem struct {
	ID      string `json:"id"`
	From    string `json:"from"`
	Subject string `json:"subject"`
	Date    string `json:"date"`
}

func (h *HimalayaClient) Fetch(ctx context.Context) ([]IncomingMessage, error) {
	args := []string{"--output", "json", "envelope", "list", "INBOX", "--max", "10"}
	if h.account != "" {
		args = append([]string{"--account", h.account}, args...)
	}
	out, err := h.runner(ctx, "himalaya", args...)
	if err != nil {
		return nil, fmt.Errorf("himalaya envelope list: %w", err)
	}

	var items []envelopeListItem
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, fmt.Errorf("decode envelope list: %w", err)
	}

	messages := make([]IncomingMessage, 0, len(items))
	for _, it := range items {
		body, err := h.readBody(ctx, it.ID)
		if err != nil {
			return nil, err
		}
		date, _ := time.Parse(time.RFC3339, it.Date)
		messages = append(messages, IncomingMessage{
			ID:      it.ID,
			From:    it.From,
			Subject: it.Subject,
			Body:    body,
			Date:    date,
		})
	}
	return messages, nil
}

func (h *HimalayaClient) Send(ctx context.Context, msg OutgoingMessage) error {
	if strings.TrimSpace(msg.To) == "" {
		return fmt.Errorf("missing recipient")
	}
	args := []string{"message", "send", "--to", msg.To, "--subject", msg.Subject, "--body", msg.Body}
	if h.account != "" {
		args = append([]string{"--account", h.account}, args...)
	}
	_, err := h.runner(ctx, "himalaya", args...)
	if err != nil {
		return fmt.Errorf("himalaya send: %w", err)
	}
	return nil
}

func (h *HimalayaClient) readBody(ctx context.Context, id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("missing message id")
	}
	args := []string{"message", "read", id, "--raw"}
	if h.account != "" {
		args = append([]string{"--account", h.account}, args...)
	}
	out, err := h.runner(ctx, "himalaya", args...)
	if err != nil {
		return "", fmt.Errorf("himalaya message read %s: %w", id, err)
	}
	return string(out), nil
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

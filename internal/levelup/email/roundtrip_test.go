package email

import (
	"context"
	"testing"
)

type fakeSender struct {
	sent []OutgoingMessage
}

func (f *fakeSender) Send(ctx context.Context, msg OutgoingMessage) error {
	f.sent = append(f.sent, msg)
	return nil
}

func TestRoundtripReceiveAgentSend(t *testing.T) {
	inbound := IncomingMessage{ID: "m1", From: "boss@example.com", Subject: "todo", Body: "reply please"}
	agentInput := FormatInboundForAgent(inbound)
	if agentInput == "" {
		t.Fatal("expected formatted agent input")
	}

	agentResponse := "done\n```json\n{\"email_actions\":[{\"to\":\"boss@example.com\",\"subject\":\"re: todo\",\"body\":\"on it\"}]}\n```"
	_, actions, err := ExtractActions(agentResponse)
	if err != nil {
		t.Fatalf("extract actions: %v", err)
	}

	sender := &fakeSender{}
	if err := ExecuteActions(context.Background(), sender, actions); err != nil {
		t.Fatalf("execute actions: %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 outbound email, got %d", len(sender.sent))
	}
	if sender.sent[0].To != "boss@example.com" {
		t.Fatalf("unexpected recipient: %s", sender.sent[0].To)
	}
}

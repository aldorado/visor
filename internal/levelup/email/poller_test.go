package email

import (
	"context"
	"testing"
	"time"
)

type fakeFetcher struct {
	messages []IncomingMessage
}

func (f *fakeFetcher) Fetch(ctx context.Context) ([]IncomingMessage, error) {
	return f.messages, nil
}

func TestPollerTickDedup(t *testing.T) {
	fetcher := &fakeFetcher{messages: []IncomingMessage{{ID: "1", Subject: "s1"}, {ID: "1", Subject: "dup"}, {ID: "2", Subject: "s2"}}}
	seen := []string{}
	p := NewPoller(fetcher, time.Second, func(msg IncomingMessage) {
		seen = append(seen, msg.ID)
	})

	if err := p.Tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}
	if len(seen) != 2 {
		t.Fatalf("expected 2 unique events, got %d", len(seen))
	}

	if err := p.Tick(context.Background()); err != nil {
		t.Fatalf("tick2: %v", err)
	}
	if len(seen) != 2 {
		t.Fatalf("expected no new events after dedup, got %d", len(seen))
	}
}

func TestPollerAllowedSendersFilters(t *testing.T) {
	fetcher := &fakeFetcher{messages: []IncomingMessage{
		{ID: "1", From: "alice@example.com", Subject: "allowed"},
		{ID: "2", From: "bob@spam.com", Subject: "blocked"},
		{ID: "3", From: "Alice <alice@example.com>", Subject: "allowed with name"},
	}}
	var delivered []string
	p := NewPoller(fetcher, time.Second, func(msg IncomingMessage) {
		delivered = append(delivered, msg.ID)
	})
	p.SetAllowedSenders([]string{"alice@example.com"})

	if err := p.Tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}
	if len(delivered) != 2 {
		t.Fatalf("expected 2 delivered (IDs 1,3), got %d: %v", len(delivered), delivered)
	}
	if delivered[0] != "1" || delivered[1] != "3" {
		t.Fatalf("expected IDs [1,3], got %v", delivered)
	}
}

func TestPollerEmptyAllowlistPassesAll(t *testing.T) {
	fetcher := &fakeFetcher{messages: []IncomingMessage{
		{ID: "1", From: "anyone@anywhere.com", Subject: "hi"},
		{ID: "2", From: "random@other.com", Subject: "hey"},
	}}
	var delivered []string
	p := NewPoller(fetcher, time.Second, func(msg IncomingMessage) {
		delivered = append(delivered, msg.ID)
	})
	// no SetAllowedSenders call = empty allowlist = all pass

	if err := p.Tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}
	if len(delivered) != 2 {
		t.Fatalf("expected all 2 delivered, got %d", len(delivered))
	}
}

func TestPollerAllowedSendersCaseInsensitive(t *testing.T) {
	fetcher := &fakeFetcher{messages: []IncomingMessage{
		{ID: "1", From: "Alice@Example.COM", Subject: "mixed case"},
	}}
	var delivered []string
	p := NewPoller(fetcher, time.Second, func(msg IncomingMessage) {
		delivered = append(delivered, msg.ID)
	})
	p.SetAllowedSenders([]string{"alice@example.com"})

	if err := p.Tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}
	if len(delivered) != 1 {
		t.Fatalf("expected 1 delivered (case-insensitive match), got %d", len(delivered))
	}
}

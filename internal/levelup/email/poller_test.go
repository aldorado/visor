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

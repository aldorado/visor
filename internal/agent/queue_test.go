package agent

import (
	"context"
	"sync"
	"testing"
	"time"
)

// slowAgent blocks for a duration before responding.
type slowAgent struct {
	delay time.Duration
}

func (s *slowAgent) SendPrompt(_ context.Context, prompt string) (string, error) {
	time.Sleep(s.delay)
	return "reply:" + prompt, nil
}

func (s *slowAgent) Close() error { return nil }

func TestQueuedAgent_SingleMessage(t *testing.T) {
	var mu sync.Mutex
	var got []string

	qa := NewQueuedAgent(&EchoAgent{}, "echo", func(ctx context.Context, chatID int64, response string, err error) {
		mu.Lock()
		got = append(got, response)
		mu.Unlock()
	})

	qa.Enqueue(context.Background(), Message{ChatID: 1, Content: "hello", Type: "text"})

	// wait for async processing
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 || got[0] != "echo: hello" {
		t.Errorf("got = %v, want [echo: hello]", got)
	}
}

func TestQueuedAgent_QueueWhileBusy(t *testing.T) {
	var mu sync.Mutex
	var got []string

	qa := NewQueuedAgent(&slowAgent{delay: 50 * time.Millisecond}, "slow", func(ctx context.Context, chatID int64, response string, err error) {
		mu.Lock()
		got = append(got, response)
		mu.Unlock()
	})

	qa.Enqueue(context.Background(), Message{ChatID: 1, Content: "first", Type: "text"})
	// give goroutine time to start
	time.Sleep(10 * time.Millisecond)

	qa.Enqueue(context.Background(), Message{ChatID: 1, Content: "second", Type: "text"})
	qa.Enqueue(context.Background(), Message{ChatID: 1, Content: "third", Type: "text"})

	if qa.QueueLen() < 1 {
		t.Error("expected at least 1 message in queue while busy")
	}

	// wait for all to complete
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 3 {
		t.Fatalf("got %d responses, want 3", len(got))
	}
	if got[0] != "reply:first" {
		t.Errorf("got[0] = %q, want %q", got[0], "reply:first")
	}
	if got[1] != "reply:second" {
		t.Errorf("got[1] = %q, want %q", got[1], "reply:second")
	}
	if got[2] != "reply:third" {
		t.Errorf("got[2] = %q, want %q", got[2], "reply:third")
	}
}

func TestQueuedAgent_QueueLenEmptyWhenIdle(t *testing.T) {
	qa := NewQueuedAgent(&EchoAgent{}, "echo", func(context.Context, int64, string, error) {})
	if qa.QueueLen() != 0 {
		t.Errorf("queue len = %d, want 0", qa.QueueLen())
	}
}

func TestEchoAgent(t *testing.T) {
	a := &EchoAgent{}
	resp, err := a.SendPrompt(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "echo: test" {
		t.Errorf("resp = %q, want %q", resp, "echo: test")
	}
}

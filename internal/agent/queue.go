package agent

import (
	"context"
	"log"
	"sync"
)

type Message struct {
	ChatID  int64
	Content string
	Type    string // "text", "voice", "photo"
}

type Response struct {
	Text string
	Err  error
}

type QueuedAgent struct {
	agent    Agent
	mu       sync.Mutex
	busy     bool
	queue    []pendingMsg
	handler  func(chatID int64, response string, err error)
}

type pendingMsg struct {
	ctx context.Context
	msg Message
}

// NewQueuedAgent wraps an Agent with a message queue.
// handler is called with the response for each processed message.
func NewQueuedAgent(agent Agent, handler func(chatID int64, response string, err error)) *QueuedAgent {
	return &QueuedAgent{
		agent:   agent,
		handler: handler,
	}
}

// Enqueue adds a message. If agent is idle, processes immediately.
// If busy, queues it and processes after current prompt finishes.
func (qa *QueuedAgent) Enqueue(ctx context.Context, msg Message) {
	qa.mu.Lock()
	if qa.busy {
		log.Printf("agent: queued message from %d (queue size: %d)", msg.ChatID, len(qa.queue)+1)
		qa.queue = append(qa.queue, pendingMsg{ctx: ctx, msg: msg})
		qa.mu.Unlock()
		return
	}
	qa.busy = true
	qa.mu.Unlock()

	go qa.process(ctx, msg)
}

func (qa *QueuedAgent) process(ctx context.Context, msg Message) {
	response, err := qa.agent.SendPrompt(ctx, msg.Content)
	qa.handler(msg.ChatID, response, err)

	for {
		qa.mu.Lock()
		if len(qa.queue) == 0 {
			qa.busy = false
			qa.mu.Unlock()
			return
		}
		next := qa.queue[0]
		qa.queue = qa.queue[1:]
		qa.mu.Unlock()

		response, err := qa.agent.SendPrompt(next.ctx, next.msg.Content)
		qa.handler(next.msg.ChatID, response, err)
	}
}

// QueueLen returns the number of pending messages.
func (qa *QueuedAgent) QueueLen() int {
	qa.mu.Lock()
	defer qa.mu.Unlock()
	return len(qa.queue)
}

package agent

import (
	"context"
	"sync"
	"time"

	"visor/internal/observability"
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
	agent   Agent
	backend string
	mu      sync.Mutex
	busy    bool
	queue   []pendingMsg
	handler func(ctx context.Context, chatID int64, response string, err error)
	log     *observability.Logger
}

type pendingMsg struct {
	ctx context.Context
	msg Message
}

// NewQueuedAgent wraps an Agent with a message queue.
// handler is called with the response for each processed message.
func NewQueuedAgent(agent Agent, backend string, handler func(ctx context.Context, chatID int64, response string, err error)) *QueuedAgent {
	if backend == "" {
		backend = "unknown"
	}
	return &QueuedAgent{
		agent:   agent,
		backend: backend,
		handler: handler,
		log:     observability.Component("agent.queue"),
	}
}

// Enqueue adds a message. If agent is idle, processes immediately.
// If busy, queues it and processes after current prompt finishes.
func (qa *QueuedAgent) Enqueue(ctx context.Context, msg Message) {
	qa.mu.Lock()
	if qa.busy {
		queueSize := len(qa.queue) + 1
		qa.log.Info(ctx, "message queued", "chat_id", msg.ChatID, "message_type", msg.Type, "queue_size", queueSize)
		qa.queue = append(qa.queue, pendingMsg{ctx: ctx, msg: msg})
		qa.mu.Unlock()
		return
	}
	qa.busy = true
	qa.mu.Unlock()

	qa.log.Debug(ctx, "processing message immediately", "chat_id", msg.ChatID, "message_type", msg.Type)
	go qa.process(ctx, msg)
}

func (qa *QueuedAgent) process(ctx context.Context, msg Message) {
	qa.log.Debug(ctx, "agent prompt start", "chat_id", msg.ChatID, "message_type", msg.Type, "backend", qa.backend)
	startedAt := nowMillis()
	response, err := qa.agent.SendPrompt(ctx, msg.Content)
	duration := nowMillis() - startedAt
	if err != nil {
		qa.log.Error(ctx, "agent prompt error", "chat_id", msg.ChatID, "backend", qa.backend, "duration_ms", duration, "error", err.Error())
	} else {
		qa.log.Info(ctx, "agent prompt processed", "chat_id", msg.ChatID, "backend", qa.backend, "duration_ms", duration)
	}
	qa.handler(ctx, msg.ChatID, response, err)

	for {
		qa.mu.Lock()
		if len(qa.queue) == 0 {
			qa.busy = false
			qa.mu.Unlock()
			qa.log.Debug(ctx, "agent queue idle")
			return
		}
		next := qa.queue[0]
		qa.queue = qa.queue[1:]
		remaining := len(qa.queue)
		qa.mu.Unlock()

		qa.log.Debug(next.ctx, "agent processing queued message", "chat_id", next.msg.ChatID, "message_type", next.msg.Type, "backend", qa.backend, "remaining_queue", remaining)
		startedAt = nowMillis()
		response, err := qa.agent.SendPrompt(next.ctx, next.msg.Content)
		duration = nowMillis() - startedAt
		if err != nil {
			qa.log.Error(next.ctx, "agent prompt error", "chat_id", next.msg.ChatID, "backend", qa.backend, "duration_ms", duration, "error", err.Error())
		} else {
			qa.log.Info(next.ctx, "agent prompt processed", "chat_id", next.msg.ChatID, "backend", qa.backend, "duration_ms", duration)
		}
		qa.handler(next.ctx, next.msg.ChatID, response, err)
	}
}

// QueueLen returns the number of pending messages.
func (qa *QueuedAgent) QueueLen() int {
	qa.mu.Lock()
	defer qa.mu.Unlock()
	return len(qa.queue)
}

func nowMillis() int64 {
	return time.Now().UnixMilli()
}

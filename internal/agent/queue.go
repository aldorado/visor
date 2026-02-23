package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
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
	agent                Agent
	backend              string
	mu                   sync.Mutex
	busy                 bool
	queue                []pendingMsg
	handler              func(ctx context.Context, chatID int64, response string, err error, duration time.Duration)
	longRunningHandler   func(ctx context.Context, chatID int64, elapsed time.Duration, preview string)
	longRunningThreshold time.Duration
	log                  *observability.Logger
}

type pendingMsg struct {
	ctx context.Context
	msg Message
}

// NewQueuedAgent wraps an Agent with a message queue.
// handler is called with the response for each processed message.
func NewQueuedAgent(agent Agent, backend string, handler func(ctx context.Context, chatID int64, response string, err error, duration time.Duration)) *QueuedAgent {
	if backend == "" {
		backend = "unknown"
	}
	return &QueuedAgent{
		agent:                agent,
		backend:              backend,
		handler:              handler,
		longRunningThreshold: 3 * time.Minute,
		log:                  observability.Component("agent.queue"),
	}
}

func (qa *QueuedAgent) SetLongRunningHandler(handler func(ctx context.Context, chatID int64, elapsed time.Duration, preview string)) {
	qa.mu.Lock()
	defer qa.mu.Unlock()
	qa.longRunningHandler = handler
}

func (qa *QueuedAgent) SetLongRunningThreshold(d time.Duration) {
	qa.mu.Lock()
	defer qa.mu.Unlock()
	qa.longRunningThreshold = d
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
	qa.processOne(ctx, msg)

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

		nextCtx, nextSpan := observability.StartSpan(next.ctx, "agent.process", attribute.String("backend", qa.backend), attribute.String("message_type", next.msg.Type))
		qa.log.Debug(nextCtx, "agent processing queued message", "chat_id", next.msg.ChatID, "message_type", next.msg.Type, "backend", qa.backend, "remaining_queue", remaining)
		qa.processOne(nextCtx, next.msg)
		nextSpan.End()
	}
}

func (qa *QueuedAgent) processOne(ctx context.Context, msg Message) {
	ctx, span := observability.StartSpan(ctx, "agent.process", attribute.String("backend", qa.backend), attribute.String("message_type", msg.Type))
	defer span.End()

	qa.log.Debug(ctx, "agent prompt start", "chat_id", msg.ChatID, "message_type", msg.Type, "backend", qa.backend)

	startedAt := time.Now()
	var progressMu sync.Mutex
	progressTail := ""

	reporter := func(delta string) {
		progressMu.Lock()
		defer progressMu.Unlock()
		progressTail = keepTail(progressTail + delta)
	}

	reportCtx := withProgressReporter(ctx, reporter)

	notifyDone := make(chan struct{})
	go func() {
		threshold := qa.getLongRunningThreshold()
		if threshold <= 0 {
			return
		}
		timer := time.NewTimer(threshold)
		defer timer.Stop()
		for {
			select {
			case <-notifyDone:
				return
			case <-timer.C:
				handler := qa.getLongRunningHandler()
				if handler != nil {
					progressMu.Lock()
					preview := strings.TrimSpace(progressTail)
					progressMu.Unlock()
					if preview == "" {
						preview = "(noch kein rpc output)"
					}
					handler(ctx, msg.ChatID, time.Since(startedAt), preview)
				}
				timer.Reset(threshold)
			}
		}
	}()

	response, err := qa.agent.SendPrompt(reportCtx, msg.Content)
	close(notifyDone)

	duration := time.Since(startedAt)
	durationMs := duration.Milliseconds()
	if err != nil {
		qa.log.Error(ctx, "agent prompt error", "chat_id", msg.ChatID, "backend", qa.backend, "duration_ms", durationMs, "error", err.Error())
	} else {
		qa.log.Info(ctx, "agent prompt processed", "chat_id", msg.ChatID, "backend", qa.backend, "duration_ms", durationMs)
	}
	qa.handler(ctx, msg.ChatID, response, err, duration)
}

func (qa *QueuedAgent) getLongRunningHandler() func(ctx context.Context, chatID int64, elapsed time.Duration, preview string) {
	qa.mu.Lock()
	defer qa.mu.Unlock()
	return qa.longRunningHandler
}

func (qa *QueuedAgent) getLongRunningThreshold() time.Duration {
	qa.mu.Lock()
	defer qa.mu.Unlock()
	return qa.longRunningThreshold
}

func keepTail(s string) string {
	const max = 320
	if len(s) <= max {
		return s
	}
	return s[len(s)-max:]
}

// QueueLen returns the number of pending messages.
func (qa *QueuedAgent) QueueLen() int {
	qa.mu.Lock()
	defer qa.mu.Unlock()
	return len(qa.queue)
}

// CurrentBackend returns the current backend label.
// For model-aware backends this can include the selected model (example: pi/codex).
func (qa *QueuedAgent) CurrentBackend() string {
	if reg, ok := qa.agent.(*Registry); ok {
		return reg.ActiveLabel()
	}
	return backendLabel(qa.backend, qa.agent)
}

func (qa *QueuedAgent) CurrentModel() string {
	if reg, ok := qa.agent.(*Registry); ok {
		return reg.ActiveModel()
	}
	return currentModel(qa.agent)
}

// SwitchBackend pins the active backend to the named one.
// Only works if the underlying agent is a Registry.
func (qa *QueuedAgent) SwitchBackend(name string) error {
	reg, ok := qa.agent.(*Registry)
	if !ok {
		return fmt.Errorf("agent does not support backend switching")
	}
	return reg.SetActive(name)
}

func (qa *QueuedAgent) SwitchModel(model string) error {
	if reg, ok := qa.agent.(*Registry); ok {
		return reg.SetModelOnActive(model)
	}
	return switchModel(qa.agent, model)
}

func nowMillis() int64 {
	return time.Now().UnixMilli()
}

package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type QuickActionType int

const (
	ActionDone QuickActionType = iota
	ActionSnooze
	ActionReschedule
)

type QuickAction struct {
	Type    QuickActionType
	RawTime string // for snooze/reschedule: the time expression
}

// ParseQuickAction checks if text is a quick action reply.
// Returns nil if text is not a recognized quick action.
func ParseQuickAction(text string) *QuickAction {
	original := strings.TrimSpace(text)
	lower := strings.ToLower(original)

	if lower == "done" || lower == "ok" || lower == "✓" || lower == "✅" {
		return &QuickAction{Type: ActionDone}
	}

	if strings.HasPrefix(lower, "snooze") {
		rest := strings.TrimSpace(original[len("snooze"):])
		restLower := strings.ToLower(rest)
		if rest == "" {
			rest = "in 15m" // default snooze
		} else if !strings.HasPrefix(restLower, "in ") && !strings.HasPrefix(restLower, "tomorrow") && !strings.HasPrefix(restLower, "mon") && !strings.HasPrefix(restLower, "tue") && !strings.HasPrefix(restLower, "wed") && !strings.HasPrefix(restLower, "thu") && !strings.HasPrefix(restLower, "fri") && !strings.HasPrefix(restLower, "sat") && !strings.HasPrefix(restLower, "sun") {
			rest = "in " + rest // bare duration like "30m" → "in 30m"
		}
		return &QuickAction{Type: ActionSnooze, RawTime: rest}
	}

	if strings.HasPrefix(lower, "reschedule") {
		rest := strings.TrimSpace(original[len("reschedule"):])
		if rest == "" {
			return nil // reschedule needs a time
		}
		return &QuickAction{Type: ActionReschedule, RawTime: rest}
	}

	return nil
}

// TriggerRecord tracks the last triggered task for quick action context.
type TriggerRecord struct {
	TaskID    string
	Prompt    string
	Recurring bool
	FiredAt   time.Time
}

const quickActionWindow = 5 * time.Minute

// QuickActionHandler manages quick action state and execution.
type QuickActionHandler struct {
	mu          sync.Mutex
	lastTrigger *TriggerRecord
	processed   map[string]time.Time // "taskID_firedAt" -> processed time
	scheduler   *Scheduler
	loc         *time.Location
	log         interface {
		Info(ctx context.Context, msg string, args ...any)
	}
}

func NewQuickActionHandler(s *Scheduler, loc *time.Location, log interface {
	Info(ctx context.Context, msg string, args ...any)
}) *QuickActionHandler {
	return &QuickActionHandler{
		processed: make(map[string]time.Time),
		scheduler: s,
		loc:       loc,
		log:       log,
	}
}

// RecordTrigger records a task trigger for quick action context.
func (h *QuickActionHandler) RecordTrigger(task Task) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastTrigger = &TriggerRecord{
		TaskID:    task.ID,
		Prompt:    task.Prompt,
		Recurring: task.Recurring,
		FiredAt:   time.Now().UTC(),
	}
}

// TryHandle checks if the message is a quick action for a recently triggered task.
// Returns (response string, handled bool).
func (h *QuickActionHandler) TryHandle(ctx context.Context, text string) (string, bool) {
	action := ParseQuickAction(text)
	if action == nil {
		return "", false
	}

	h.mu.Lock()
	trigger := h.lastTrigger
	h.mu.Unlock()

	if trigger == nil {
		return "", false
	}

	now := time.Now().UTC()
	if now.Sub(trigger.FiredAt) > quickActionWindow {
		return "", false
	}

	// idempotency check
	key := fmt.Sprintf("%s_%d", trigger.TaskID, trigger.FiredAt.UnixNano())
	h.mu.Lock()
	if _, done := h.processed[key]; done {
		h.mu.Unlock()
		return "already handled 👍", true
	}
	h.processed[key] = now
	// clean old entries
	for k, t := range h.processed {
		if now.Sub(t) > 10*time.Minute {
			delete(h.processed, k)
		}
	}
	h.mu.Unlock()

	switch action.Type {
	case ActionDone:
		return h.handleDone(ctx, trigger)
	case ActionSnooze:
		return h.handleSnooze(ctx, trigger, action.RawTime, now)
	case ActionReschedule:
		return h.handleReschedule(ctx, trigger, action.RawTime, now)
	}

	return "", false
}

func (h *QuickActionHandler) handleDone(ctx context.Context, trigger *TriggerRecord) (string, bool) {
	// for one-shot: already deleted by TriggerDue. nothing to do.
	// for recurring: series continues, just acknowledge.
	h.log.Info(ctx, "quick action: done", "task_id", trigger.TaskID, "prompt", trigger.Prompt)
	return "done ✓", true
}

func (h *QuickActionHandler) handleSnooze(ctx context.Context, trigger *TriggerRecord, rawTime string, now time.Time) (string, bool) {
	t, err := ParseNaturalTime(rawTime, now, h.loc)
	if err != nil {
		return fmt.Sprintf("couldn't parse snooze time: %s", err), true
	}
	if !t.After(now) {
		return "snooze time must be in the future", true
	}

	// always create a new one-shot for snooze (preserves recurring series)
	id, err := h.scheduler.AddOneShot(trigger.Prompt, t)
	if err != nil {
		return fmt.Sprintf("snooze failed: %s", err), true
	}

	h.log.Info(ctx, "quick action: snooze", "task_id", trigger.TaskID, "new_id", id, "snooze_until", t)
	return fmt.Sprintf("snoozed until %s ⏰", t.In(h.loc).Format("15:04")), true
}

func (h *QuickActionHandler) handleReschedule(ctx context.Context, trigger *TriggerRecord, rawTime string, now time.Time) (string, bool) {
	t, err := ParseNaturalTime(rawTime, now, h.loc)
	if err != nil {
		return fmt.Sprintf("couldn't parse time: %s", err), true
	}
	if !t.After(now) {
		return "reschedule time must be in the future", true
	}

	if trigger.Recurring {
		// update the recurring task's next run (series shifts)
		err := h.scheduler.Update(trigger.TaskID, UpdateTaskInput{RunAt: &t})
		if err != nil {
			return fmt.Sprintf("reschedule failed: %s", err), true
		}
		h.log.Info(ctx, "quick action: reschedule recurring", "task_id", trigger.TaskID, "new_time", t)
		return fmt.Sprintf("rescheduled to %s ⏰", t.In(h.loc).Format("15:04")), true
	}

	// one-shot was already deleted, create new one
	id, err := h.scheduler.AddOneShot(trigger.Prompt, t)
	if err != nil {
		return fmt.Sprintf("reschedule failed: %s", err), true
	}
	h.log.Info(ctx, "quick action: reschedule one-shot", "old_id", trigger.TaskID, "new_id", id, "new_time", t)
	return fmt.Sprintf("rescheduled to %s ⏰", t.In(h.loc).Format("15:04")), true
}

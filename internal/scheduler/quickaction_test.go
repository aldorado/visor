package scheduler

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestParseQuickAction(t *testing.T) {
	tests := []struct {
		input    string
		wantType *QuickActionType
		wantTime string
	}{
		{"done", ptr(ActionDone), ""},
		{"ok", ptr(ActionDone), ""},
		{"✅", ptr(ActionDone), ""},
		{"snooze 30m", ptr(ActionSnooze), "in 30m"},
		{"snooze in 1h", ptr(ActionSnooze), "in 1h"},
		{"snooze", ptr(ActionSnooze), "in 15m"},
		{"reschedule tomorrow 09:00", ptr(ActionReschedule), "tomorrow 09:00"},
		{"reschedule in 2h", ptr(ActionReschedule), "in 2h"},
		{"reschedule", nil, ""},        // missing time
		{"hello world", nil, ""},       // not a quick action
		{"do something done", nil, ""}, // not a quick action
	}

	for _, tt := range tests {
		action := ParseQuickAction(tt.input)
		if tt.wantType == nil {
			if action != nil {
				t.Errorf("ParseQuickAction(%q) = %+v, want nil", tt.input, action)
			}
			continue
		}
		if action == nil {
			t.Errorf("ParseQuickAction(%q) = nil, want type=%d", tt.input, *tt.wantType)
			continue
		}
		if action.Type != *tt.wantType {
			t.Errorf("ParseQuickAction(%q).Type = %d, want %d", tt.input, action.Type, *tt.wantType)
		}
		if action.RawTime != tt.wantTime {
			t.Errorf("ParseQuickAction(%q).RawTime = %q, want %q", tt.input, action.RawTime, tt.wantTime)
		}
	}
}

func ptr(t QuickActionType) *QuickActionType { return &t }

type testLogger struct{}

func (l testLogger) Info(_ context.Context, _ string, _ ...any) {}

func TestQuickActionHandler_Done(t *testing.T) {
	tmp := t.TempDir()
	s, err := New(filepath.Join(tmp, "scheduler"), nil)
	if err != nil {
		t.Fatal(err)
	}

	h := NewQuickActionHandler(s, time.UTC, testLogger{})
	h.RecordTrigger(Task{ID: "abc", Prompt: "wake up", Recurring: false, NextRunAt: time.Now().UTC()})

	reply, ok := h.TryHandle(context.Background(), "done")
	if !ok {
		t.Fatal("expected handled")
	}
	if reply != "done ✓" {
		t.Fatalf("reply=%q", reply)
	}
}

func TestQuickActionHandler_Snooze(t *testing.T) {
	tmp := t.TempDir()
	s, err := New(filepath.Join(tmp, "scheduler"), nil)
	if err != nil {
		t.Fatal(err)
	}

	h := NewQuickActionHandler(s, time.UTC, testLogger{})
	h.RecordTrigger(Task{ID: "abc", Prompt: "check email", Recurring: true, NextRunAt: time.Now().UTC()})

	reply, ok := h.TryHandle(context.Background(), "snooze 30m")
	if !ok {
		t.Fatal("expected handled")
	}
	if reply == "" {
		t.Fatal("expected non-empty reply")
	}

	// a new one-shot should be created
	list := s.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 snoozed task, got %d", len(list))
	}
	if list[0].Recurring {
		t.Fatal("snoozed task should be one-shot")
	}
	if list[0].Prompt != "check email" {
		t.Fatalf("prompt=%q", list[0].Prompt)
	}
}

func TestQuickActionHandler_Reschedule_OneShot(t *testing.T) {
	tmp := t.TempDir()
	s, err := New(filepath.Join(tmp, "scheduler"), nil)
	if err != nil {
		t.Fatal(err)
	}

	h := NewQuickActionHandler(s, time.UTC, testLogger{})
	h.RecordTrigger(Task{ID: "abc", Prompt: "dentist", Recurring: false})

	reply, ok := h.TryHandle(context.Background(), "reschedule tomorrow 09:00")
	if !ok {
		t.Fatal("expected handled")
	}
	if reply == "" {
		t.Fatal("expected non-empty reply")
	}

	list := s.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 rescheduled task, got %d", len(list))
	}
	if list[0].Prompt != "dentist" {
		t.Fatalf("prompt=%q", list[0].Prompt)
	}
}

func TestQuickActionHandler_Reschedule_Recurring(t *testing.T) {
	tmp := t.TempDir()
	s, err := New(filepath.Join(tmp, "scheduler"), nil)
	if err != nil {
		t.Fatal(err)
	}

	// add a recurring task
	id, err := s.AddRecurring("standup", time.Now().UTC().Add(1*time.Hour), 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	h := NewQuickActionHandler(s, time.UTC, testLogger{})
	h.RecordTrigger(Task{ID: id, Prompt: "standup", Recurring: true})

	reply, ok := h.TryHandle(context.Background(), "reschedule in 2h")
	if !ok {
		t.Fatal("expected handled")
	}
	if reply == "" {
		t.Fatal("expected non-empty reply")
	}

	// recurring task should be updated, not replaced
	list := s.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 task, got %d", len(list))
	}
	if list[0].ID != id {
		t.Fatalf("task ID changed: got %s, want %s", list[0].ID, id)
	}
	if !list[0].Recurring {
		t.Fatal("task should still be recurring")
	}
}

func TestQuickActionHandler_Idempotency(t *testing.T) {
	tmp := t.TempDir()
	s, err := New(filepath.Join(tmp, "scheduler"), nil)
	if err != nil {
		t.Fatal(err)
	}

	h := NewQuickActionHandler(s, time.UTC, testLogger{})
	h.RecordTrigger(Task{ID: "abc", Prompt: "test", Recurring: false})

	// first call
	reply1, ok1 := h.TryHandle(context.Background(), "done")
	if !ok1 {
		t.Fatal("expected handled")
	}
	if reply1 != "done ✓" {
		t.Fatalf("reply1=%q", reply1)
	}

	// second call (duplicate)
	reply2, ok2 := h.TryHandle(context.Background(), "done")
	if !ok2 {
		t.Fatal("expected handled")
	}
	if reply2 != "already handled 👍" {
		t.Fatalf("reply2=%q", reply2)
	}
}

func TestQuickActionHandler_NoRecentTrigger(t *testing.T) {
	tmp := t.TempDir()
	s, err := New(filepath.Join(tmp, "scheduler"), nil)
	if err != nil {
		t.Fatal(err)
	}

	h := NewQuickActionHandler(s, time.UTC, testLogger{})
	// no trigger recorded

	_, ok := h.TryHandle(context.Background(), "done")
	if ok {
		t.Fatal("expected not handled (no recent trigger)")
	}
}

func TestQuickActionHandler_ExpiredTrigger(t *testing.T) {
	tmp := t.TempDir()
	s, err := New(filepath.Join(tmp, "scheduler"), nil)
	if err != nil {
		t.Fatal(err)
	}

	h := NewQuickActionHandler(s, time.UTC, testLogger{})
	h.RecordTrigger(Task{ID: "abc", Prompt: "test", Recurring: false})
	// manually expire
	h.mu.Lock()
	h.lastTrigger.FiredAt = time.Now().UTC().Add(-10 * time.Minute)
	h.mu.Unlock()

	_, ok := h.TryHandle(context.Background(), "done")
	if ok {
		t.Fatal("expected not handled (trigger expired)")
	}
}

func TestQuickActionHandler_SnoozePastTime(t *testing.T) {
	tmp := t.TempDir()
	s, err := New(filepath.Join(tmp, "scheduler"), nil)
	if err != nil {
		t.Fatal(err)
	}

	h := NewQuickActionHandler(s, time.UTC, testLogger{})
	h.RecordTrigger(Task{ID: "abc", Prompt: "test", Recurring: false})

	// snooze to a past RFC3339 time
	reply, ok := h.TryHandle(context.Background(), "reschedule 2020-01-01T00:00:00Z")
	if !ok {
		t.Fatal("expected handled")
	}
	if reply != "reschedule time must be in the future" {
		t.Fatalf("reply=%q", reply)
	}
}

func TestQuickActionHandler_DefaultSnooze(t *testing.T) {
	tmp := t.TempDir()
	s, err := New(filepath.Join(tmp, "scheduler"), nil)
	if err != nil {
		t.Fatal(err)
	}

	h := NewQuickActionHandler(s, time.UTC, testLogger{})
	h.RecordTrigger(Task{ID: "abc", Prompt: "test", Recurring: false})

	// "snooze" without duration → defaults to 15m
	reply, ok := h.TryHandle(context.Background(), "snooze")
	if !ok {
		t.Fatal("expected handled")
	}
	if reply == "" {
		t.Fatal("expected non-empty reply")
	}

	list := s.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 task, got %d", len(list))
	}
	// should be ~15 minutes from now
	diff := time.Until(list[0].NextRunAt)
	if diff < 14*time.Minute || diff > 16*time.Minute {
		t.Fatalf("snoozed task at %s, expected ~15m from now", list[0].NextRunAt)
	}
}

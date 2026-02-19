package scheduler

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSchedulerOneShotTriggerRemovesTask(t *testing.T) {
	tmp := t.TempDir()
	triggered := 0
	s, err := New(filepath.Join(tmp, "scheduler"), func(ctx context.Context, task Task) {
		triggered++
	})
	if err != nil {
		t.Fatal(err)
	}

	runAt := time.Now().UTC().Add(-1 * time.Second)
	_, err = s.AddOneShot("hello", runAt)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.TriggerDue(context.Background(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	if triggered != 1 {
		t.Fatalf("triggered=%d want=1", triggered)
	}
	if len(s.List()) != 0 {
		t.Fatalf("expected one-shot removed, got %d", len(s.List()))
	}
}

func TestSchedulerRecurringReschedules(t *testing.T) {
	tmp := t.TempDir()
	triggered := 0
	s, err := New(filepath.Join(tmp, "scheduler"), func(ctx context.Context, task Task) {
		triggered++
	})
	if err != nil {
		t.Fatal(err)
	}

	firstRun := time.Now().UTC().Add(-5 * time.Second)
	_, err = s.AddRecurring("ping", firstRun, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	if err := s.TriggerDue(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	if triggered != 1 {
		t.Fatalf("triggered=%d want=1", triggered)
	}

	list := s.List()
	if len(list) != 1 {
		t.Fatalf("expected recurring task retained, got %d", len(list))
	}
	if !list[0].NextRunAt.After(now) {
		t.Fatalf("expected rescheduled next run after now, got %s", list[0].NextRunAt)
	}
}

func TestSchedulerPersistsAcrossRestart(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "scheduler")

	s1, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s1.AddOneShot("persist-me", time.Now().UTC().Add(10*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	s2, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(s2.List()) != 1 {
		t.Fatalf("expected persisted task count=1, got %d", len(s2.List()))
	}
}

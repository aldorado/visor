package server

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"visor/internal/observability"
	"visor/internal/scheduler"
)

func TestExecuteScheduleActions_CreateListUpdateDelete(t *testing.T) {
	tmp := t.TempDir()
	sched, err := scheduler.New(filepath.Join(tmp, "scheduler"), nil)
	if err != nil {
		t.Fatal(err)
	}

	srv := &Server{scheduler: sched, log: observability.Component("server_test")}
	ctx := context.Background()

	runAt := time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339)
	note := srv.executeScheduleActions(ctx, &scheduler.ActionEnvelope{
		Create: []scheduler.CreateAction{{Prompt: "ping", RunAt: runAt}},
		List:   true,
	})
	if !strings.Contains(note, "scheduled ✅") {
		t.Fatalf("note=%q", note)
	}
	if !strings.Contains(note, "scheduled tasks:") {
		t.Fatalf("note=%q", note)
	}

	list := sched.List()
	if len(list) != 1 {
		t.Fatalf("len=%d", len(list))
	}
	id := list[0].ID

	newPrompt := "pong"
	note = srv.executeScheduleActions(ctx, &scheduler.ActionEnvelope{
		Update: []scheduler.UpdateAction{{ID: id, Prompt: newPrompt}},
	})
	if !strings.Contains(note, "schedule updated ✅") {
		t.Fatalf("note=%q", note)
	}
	if sched.List()[0].Prompt != "pong" {
		t.Fatalf("prompt=%q", sched.List()[0].Prompt)
	}

	note = srv.executeScheduleActions(ctx, &scheduler.ActionEnvelope{
		Delete: []scheduler.DeleteAction{{ID: id}},
		List:   true,
	})
	if !strings.Contains(note, "schedule deleted ✅") {
		t.Fatalf("note=%q", note)
	}
	if !strings.Contains(note, "no scheduled tasks") {
		t.Fatalf("note=%q", note)
	}
}

func TestExecuteScheduleActions_InvalidAndUnknown(t *testing.T) {
	tmp := t.TempDir()
	sched, err := scheduler.New(filepath.Join(tmp, "scheduler"), nil)
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{scheduler: sched, log: observability.Component("server_test")}
	ctx := context.Background()

	note := srv.executeScheduleActions(ctx, &scheduler.ActionEnvelope{
		Create: []scheduler.CreateAction{{Prompt: "ping", RunAt: "tomorrow"}},
		Update: []scheduler.UpdateAction{{ID: "missing", Prompt: "x"}},
		Delete: []scheduler.DeleteAction{{ID: "missing"}},
	})
	if !strings.Contains(note, "invalid run_at") {
		t.Fatalf("note=%q", note)
	}
	if !strings.Contains(note, "task not found") {
		t.Fatalf("note=%q", note)
	}
}

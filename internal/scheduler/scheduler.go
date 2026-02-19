package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"visor/internal/observability"
)

type Task struct {
	ID              string    `json:"id"`
	Prompt          string    `json:"prompt"`
	NextRunAt       time.Time `json:"next_run_at"`
	Recurring       bool      `json:"recurring"`
	IntervalSeconds int64     `json:"interval_seconds,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type Scheduler struct {
	mu        sync.Mutex
	tasks     map[string]Task
	storePath string
	onTrigger func(context.Context, Task)
	log       *observability.Logger
}

func New(dataDir string, onTrigger func(context.Context, Task)) (*Scheduler, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("data dir is required")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir data dir: %w", err)
	}

	s := &Scheduler{
		tasks:     map[string]Task{},
		storePath: filepath.Join(dataDir, "tasks.json"),
		onTrigger: onTrigger,
		log:       observability.Component("scheduler"),
	}
	if err := s.loadLocked(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Scheduler) AddOneShot(prompt string, runAt time.Time) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}
	if runAt.IsZero() {
		return "", fmt.Errorf("runAt is required")
	}

	task := Task{
		ID:        uuid.NewString(),
		Prompt:    prompt,
		NextRunAt: runAt,
		Recurring: false,
		CreatedAt: time.Now().UTC(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
	if err := s.saveLocked(); err != nil {
		return "", err
	}
	s.log.Info(context.Background(), "scheduler one-shot added", "task_id", task.ID, "run_at", task.NextRunAt)
	return task.ID, nil
}

func (s *Scheduler) AddRecurring(prompt string, firstRun time.Time, interval time.Duration) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}
	if firstRun.IsZero() {
		return "", fmt.Errorf("firstRun is required")
	}
	if interval <= 0 {
		return "", fmt.Errorf("interval must be > 0")
	}

	task := Task{
		ID:              uuid.NewString(),
		Prompt:          prompt,
		NextRunAt:       firstRun,
		Recurring:       true,
		IntervalSeconds: int64(interval.Seconds()),
		CreatedAt:       time.Now().UTC(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
	if err := s.saveLocked(); err != nil {
		return "", err
	}
	s.log.Info(context.Background(), "scheduler recurring added", "task_id", task.ID, "run_at", task.NextRunAt, "interval_seconds", task.IntervalSeconds)
	return task.ID, nil
}

func (s *Scheduler) Delete(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[taskID]; !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	delete(s.tasks, taskID)
	if err := s.saveLocked(); err != nil {
		return err
	}
	s.log.Info(context.Background(), "scheduler task deleted", "task_id", taskID)
	return nil
}

func (s *Scheduler) List() []Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].NextRunAt.Before(out[j].NextRunAt)
	})
	return out
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	s.log.Info(ctx, "scheduler loop started")
	for {
		if err := s.TriggerDue(ctx, time.Now().UTC()); err != nil {
			s.log.Error(ctx, "scheduler trigger failed", "error", err.Error())
		}
		select {
		case <-ctx.Done():
			s.log.Info(ctx, "scheduler loop stopped")
			return
		case <-ticker.C:
		}
	}
}

func (s *Scheduler) TriggerDue(ctx context.Context, now time.Time) error {
	s.mu.Lock()
	due := make([]Task, 0)
	for _, t := range s.tasks {
		if !t.NextRunAt.After(now) {
			due = append(due, t)
		}
	}

	if len(due) == 0 {
		s.mu.Unlock()
		return nil
	}

	for _, t := range due {
		if t.Recurring {
			next := t.NextRunAt
			interval := time.Duration(t.IntervalSeconds) * time.Second
			for !next.After(now) {
				next = next.Add(interval)
			}
			t.NextRunAt = next
			s.tasks[t.ID] = t
		} else {
			delete(s.tasks, t.ID)
		}
	}

	if err := s.saveLocked(); err != nil {
		s.mu.Unlock()
		return err
	}
	s.mu.Unlock()

	for _, t := range due {
		s.log.Info(ctx, "scheduler task triggered", "task_id", t.ID, "recurring", t.Recurring)
		if s.onTrigger != nil {
			s.onTrigger(ctx, t)
		}
	}
	return nil
}

func (s *Scheduler) loadLocked() error {
	if _, err := os.Stat(s.storePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat scheduler store: %w", err)
	}
	bytes, err := os.ReadFile(s.storePath)
	if err != nil {
		return fmt.Errorf("read scheduler store: %w", err)
	}
	if len(bytes) == 0 {
		return nil
	}
	var tasks []Task
	if err := json.Unmarshal(bytes, &tasks); err != nil {
		return fmt.Errorf("decode scheduler store: %w", err)
	}
	for _, t := range tasks {
		s.tasks[t.ID] = t
	}
	s.log.Info(context.Background(), "scheduler tasks loaded", "count", len(tasks))
	return nil
}

func (s *Scheduler) saveLocked() error {
	list := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		list = append(list, t)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].NextRunAt.Before(list[j].NextRunAt)
	})

	bytes, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("encode scheduler store: %w", err)
	}
	bytes = append(bytes, '\n')
	if err := os.WriteFile(s.storePath, bytes, 0o644); err != nil {
		return fmt.Errorf("write scheduler store: %w", err)
	}
	return nil
}

package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type SessionEntry struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"` // "user" or "assistant"
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"` // unix millis
}

type SessionLogger struct {
	dir       string
	mu        sync.Mutex
	sessionID string
	file      *os.File
}

func NewSessionLogger(dir string) (*SessionLogger, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("session: create dir: %w", err)
	}
	sessionID := uuid.New().String()
	filename := fmt.Sprintf("%s_%s.jsonl", time.Now().Format("2006-01-02"), sessionID[:8])
	path := filepath.Join(dir, filename)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("session: open file: %w", err)
	}

	return &SessionLogger{
		dir:       dir,
		sessionID: sessionID,
		file:      f,
	}, nil
}

func (sl *SessionLogger) Log(role, content string) error {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	entry := SessionEntry{
		ID:        uuid.New().String(),
		SessionID: sl.sessionID,
		Role:      role,
		Content:   content,
		Timestamp: time.Now().UnixMilli(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("session: marshal: %w", err)
	}

	if _, err := fmt.Fprintf(sl.file, "%s\n", data); err != nil {
		return fmt.Errorf("session: write: %w", err)
	}
	return nil
}

func (sl *SessionLogger) SessionID() string {
	return sl.sessionID
}

func (sl *SessionLogger) Close() error {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if sl.file != nil {
		return sl.file.Close()
	}
	return nil
}

// ReadAllSessions reads all session entries from all JSONL files in the directory,
// sorted by timestamp ascending.
func ReadAllSessions(dir string) ([]SessionEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("session: read dir: %w", err)
	}

	var all []SessionEntry
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("session: read %s: %w", path, err)
		}

		for _, line := range splitLines(data) {
			if len(line) == 0 {
				continue
			}
			var entry SessionEntry
			if err := json.Unmarshal(line, &entry); err != nil {
				continue // skip malformed lines
			}
			all = append(all, entry)
		}
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Timestamp < all[j].Timestamp
	})
	return all, nil
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

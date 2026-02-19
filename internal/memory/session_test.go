package memory

import (
	"os"
	"testing"
)

func TestSessionLogger_LogAndRead(t *testing.T) {
	dir, _ := os.MkdirTemp("", "visor-session-test-*")
	defer os.RemoveAll(dir)

	sl, err := NewSessionLogger(dir)
	if err != nil {
		t.Fatal(err)
	}

	sl.Log("user", "hello")
	sl.Log("assistant", "hi there")
	sl.Log("user", "how are you?")
	sl.Close()

	entries, err := ReadAllSessions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
	if entries[0].Role != "user" || entries[0].Content != "hello" {
		t.Errorf("entries[0] = %+v", entries[0])
	}
	if entries[1].Role != "assistant" || entries[1].Content != "hi there" {
		t.Errorf("entries[1] = %+v", entries[1])
	}
	if entries[0].SessionID != entries[1].SessionID {
		t.Error("entries from same session should have same session ID")
	}
	if entries[0].ID == entries[1].ID {
		t.Error("entries should have unique IDs")
	}
	// should be sorted by timestamp
	if entries[0].Timestamp > entries[2].Timestamp {
		t.Error("entries should be sorted by timestamp")
	}
}

func TestSessionLogger_SessionID(t *testing.T) {
	dir, _ := os.MkdirTemp("", "visor-session-test-*")
	defer os.RemoveAll(dir)

	sl, _ := NewSessionLogger(dir)
	defer sl.Close()

	if sl.SessionID() == "" {
		t.Error("session ID should not be empty")
	}
}

func TestReadAllSessions_EmptyDir(t *testing.T) {
	dir, _ := os.MkdirTemp("", "visor-session-test-*")
	defer os.RemoveAll(dir)

	entries, err := ReadAllSessions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestReadAllSessions_MultipleSessions(t *testing.T) {
	dir, _ := os.MkdirTemp("", "visor-session-test-*")
	defer os.RemoveAll(dir)

	sl1, _ := NewSessionLogger(dir)
	sl1.Log("user", "session 1 msg")
	sl1.Close()

	sl2, _ := NewSessionLogger(dir)
	sl2.Log("user", "session 2 msg")
	sl2.Close()

	entries, _ := ReadAllSessions(dir)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].SessionID == entries[1].SessionID {
		t.Error("different sessions should have different session IDs")
	}
}

package telegram

import (
	"testing"
	"time"
)

func TestDedup_FirstSeen(t *testing.T) {
	d := NewDedup(time.Minute)
	if d.IsDuplicate(1) {
		t.Error("first call should not be duplicate")
	}
}

func TestDedup_SecondSeen(t *testing.T) {
	d := NewDedup(time.Minute)
	d.IsDuplicate(1)
	if !d.IsDuplicate(1) {
		t.Error("second call with same ID should be duplicate")
	}
}

func TestDedup_DifferentIDs(t *testing.T) {
	d := NewDedup(time.Minute)
	d.IsDuplicate(1)
	if d.IsDuplicate(2) {
		t.Error("different ID should not be duplicate")
	}
}

func TestDedup_Cleanup(t *testing.T) {
	d := NewDedup(50 * time.Millisecond)
	d.IsDuplicate(1)

	// wait for TTL + cleanup tick
	time.Sleep(120 * time.Millisecond)

	if d.IsDuplicate(1) {
		t.Error("ID should have been cleaned up after TTL")
	}
}

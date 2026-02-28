package memory

import "testing"

func TestManager_RuntimeSelfCheck(t *testing.T) {
	dir := tempDir(t)
	m, err := NewManager(dir, "test-key")
	if err != nil {
		t.Fatal(err)
	}

	if err := m.RuntimeSelfCheck(); err != nil {
		t.Fatalf("runtime self-check failed: %v", err)
	}
}

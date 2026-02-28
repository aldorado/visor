package memory

import (
	"fmt"
)

// RuntimeSelfCheck verifies that memory storage is readable in the current runtime.
// It is intentionally local-only (no network calls) so startup checks stay fast.
func (m *Manager) RuntimeSelfCheck() error {
	if m == nil {
		return fmt.Errorf("memory runtime self-check: manager is nil")
	}

	if _, err := m.store.Count(); err != nil {
		return fmt.Errorf("memory runtime self-check: store unreadable: %w", err)
	}

	return nil
}

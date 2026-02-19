package observability

import (
	"strings"
	"testing"
)

func TestCompactStack(t *testing.T) {
	long := strings.Repeat("frame\n", 40)
	got := compactStack(long)
	if strings.Count(got, "\n") > 16 {
		t.Fatalf("expected compact stack, got %d lines", strings.Count(got, "\n"))
	}
}

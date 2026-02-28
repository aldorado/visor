package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCLI_SelfCheck(t *testing.T) {
	tmp, err := os.MkdirTemp("", "visor-memorylookup-cli-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	cmd := exec.Command("go", "run", ".", "-self-check", "-data-dir", tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\noutput:\n%s", err, string(out))
	}

	if !strings.Contains(string(out), "memory lookup runtime ok") {
		t.Fatalf("unexpected output: %s", string(out))
	}
}

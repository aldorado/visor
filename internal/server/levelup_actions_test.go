package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"visor/internal/config"
	"visor/internal/levelup"
	"visor/internal/observability"
)

func TestExecuteLevelupActionsUpdatesEnvFile(t *testing.T) {
	tmp := t.TempDir()
	srv := &Server{
		cfg: &config.Config{SelfEvolutionRepoDir: tmp},
		log: observability.Component("server_test"),
	}

	note := srv.executeLevelupActions(context.Background(), &levelup.ActionEnvelope{
		EnvSet: map[string]string{
			"A": "1",
			"B": "2",
		},
		EnvUnset: []string{"B"},
	})

	if !strings.Contains(note, ".levelup.env updated") {
		t.Fatalf("note=%q", note)
	}

	bytes, err := os.ReadFile(filepath.Join(tmp, ".levelup.env"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(bytes)
	if got != "A=1\n" {
		t.Fatalf("got=%q", got)
	}
}

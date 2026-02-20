package levelup

import "testing"

func TestBuildComposeUpArgs(t *testing.T) {
	assembly := &ComposeAssembly{ProjectRoot: "/tmp/p", Files: []string{"/tmp/p/base.yml", "/tmp/p/ov.yml"}}
	args := BuildComposeUpArgs(assembly, []string{"/tmp/p/.env", "/tmp/p/.levelup.env"})
	want := []string{"compose", "-f", "/tmp/p/base.yml", "-f", "/tmp/p/ov.yml", "--project-directory", "/tmp/p", "--env-file", "/tmp/p/.env", "--env-file", "/tmp/p/.levelup.env", "up", "-d"}
	if len(args) != len(want) {
		t.Fatalf("args len mismatch: got %d want %d", len(args), len(want))
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d] got %q want %q", i, args[i], want[i])
		}
	}
}

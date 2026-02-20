package levelup

import "testing"

func TestExpandEnvDefaults(t *testing.T) {
	env := map[string]string{"A": "x"}
	got := expandEnvDefaults("http://127.0.0.1:${P:-8080}/h/${A}", env)
	want := "http://127.0.0.1:8080/h/x"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

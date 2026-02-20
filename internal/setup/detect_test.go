package setup

import (
	"os"
	"testing"
)

func TestDetectFirstRun(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("USER_PHONE_NUMBER", "")
	root := t.TempDir()
	state, err := Detect(root, "data")
	if err != nil {
		t.Fatal(err)
	}
	if !state.FirstRun {
		t.Fatal("expected first run true")
	}
}

func TestDetectNotFirstRunWhenBootstrapEnvPresent(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "x")
	t.Setenv("USER_PHONE_NUMBER", "1")
	root := t.TempDir()
	state, err := Detect(root, "data")
	if err != nil {
		t.Fatal(err)
	}
	if state.FirstRun {
		t.Fatal("expected first run false")
	}
	_ = os.Unsetenv("TELEGRAM_BOT_TOKEN")
	_ = os.Unsetenv("USER_PHONE_NUMBER")
}

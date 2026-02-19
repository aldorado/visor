package agent

import (
	"context"
	"fmt"
	"testing"
)

type failAgent struct {
	err error
}

func (f *failAgent) SendPrompt(_ context.Context, _ string) (string, error) {
	return "", f.err
}
func (f *failAgent) Close() error { return nil }

func TestRegistryPrioritySelection(t *testing.T) {
	r := NewRegistry()
	r.Register("echo1", &EchoAgent{}, 2)
	r.Register("echo0", &EchoAgent{}, 0)
	r.Register("echo3", &EchoAgent{}, 3)

	r.HealthCheckAll(context.Background())

	if got := r.Active(); got != "echo0" {
		t.Errorf("active = %q, want 'echo0' (lowest priority = highest preference)", got)
	}
}

func TestRegistrySendPromptProxies(t *testing.T) {
	r := NewRegistry()
	r.Register("echo", &EchoAgent{}, 0)
	r.HealthCheckAll(context.Background())

	resp, err := r.SendPrompt(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if resp != "echo: hello" {
		t.Errorf("response = %q, want 'echo: hello'", resp)
	}
}

func TestRegistryNoHealthyBackend(t *testing.T) {
	r := NewRegistry()
	r.Register("bad", &EchoAgent{}, 0)

	// manually mark unhealthy
	r.MarkUnhealthy("bad", "down")

	_, err := r.SendPrompt(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error when no healthy backend")
	}
}

func TestRegistryMarkUnhealthyFallback(t *testing.T) {
	r := NewRegistry()
	r.Register("primary", &EchoAgent{}, 0)
	r.Register("fallback", &EchoAgent{}, 1)
	r.HealthCheckAll(context.Background())

	if r.Active() != "primary" {
		t.Fatalf("expected primary active, got %q", r.Active())
	}

	r.MarkUnhealthy("primary", "rate limited")

	if r.Active() != "fallback" {
		t.Errorf("after marking primary unhealthy, active = %q, want 'fallback'", r.Active())
	}
}

func TestRegistryMarkHealthyRecovery(t *testing.T) {
	r := NewRegistry()
	r.Register("primary", &EchoAgent{}, 0)
	r.Register("fallback", &EchoAgent{}, 1)
	r.HealthCheckAll(context.Background())

	r.MarkUnhealthy("primary", "down")
	if r.Active() != "fallback" {
		t.Fatalf("expected fallback, got %q", r.Active())
	}

	r.MarkHealthy("primary")
	if r.Active() != "primary" {
		t.Errorf("after recovery, active = %q, want 'primary'", r.Active())
	}
}

func TestRegistryStatus(t *testing.T) {
	r := NewRegistry()
	r.Register("a", &EchoAgent{}, 0)
	r.Register("b", &EchoAgent{}, 1)
	r.HealthCheckAll(context.Background())

	status := r.Status()
	if len(status) != 2 {
		t.Fatalf("status len = %d, want 2", len(status))
	}

	// first should be active
	if !status[0].Active {
		t.Error("expected first backend to be active")
	}
	if status[0].Name != "a" {
		t.Errorf("first backend = %q, want 'a'", status[0].Name)
	}
	if status[1].Active {
		t.Error("expected second backend to not be active")
	}
}

func TestRegistryClose(t *testing.T) {
	r := NewRegistry()
	r.Register("a", &EchoAgent{}, 0)
	r.Register("b", &EchoAgent{}, 1)

	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRegistryCloseError(t *testing.T) {
	r := NewRegistry()
	r.Register("bad", &closeErrorAgent{}, 0)

	err := r.Close()
	if err == nil {
		t.Fatal("expected close error")
	}
}

type closeErrorAgent struct{ EchoAgent }

func (c *closeErrorAgent) Close() error { return fmt.Errorf("close failed") }

func TestCheckHealthEcho(t *testing.T) {
	healthy, reason := checkHealth(context.Background(), "echo")
	if !healthy {
		t.Errorf("echo should always be healthy, got reason: %s", reason)
	}
}

func TestCheckHealthUnknown(t *testing.T) {
	healthy, _ := checkHealth(context.Background(), "unknown-backend")
	if !healthy {
		t.Error("unknown backends should default to healthy")
	}
}

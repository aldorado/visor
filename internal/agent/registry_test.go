package agent

import (
	"context"
	"fmt"
	"testing"
	"time"
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

// --- M7-I2: auto-failover tests ---

// rateLimitAgent returns a retryable error on first call, succeeds after.
type rateLimitAgent struct {
	calls int
}

func (r *rateLimitAgent) SendPrompt(_ context.Context, prompt string) (string, error) {
	r.calls++
	if r.calls == 1 {
		return "", fmt.Errorf("rate limit exceeded (429)")
	}
	return "rateLimitAgent: " + prompt, nil
}
func (r *rateLimitAgent) Close() error { return nil }

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		err  string
		want bool
	}{
		{"rate limit exceeded", true},
		{"rate_limit_event from backend", true},
		{"429 Too Many Requests", true},
		{"quota exhausted for today", true},
		{"server overloaded, try later", true},
		{"resource_exhausted", true},
		{"throttled by upstream", true},
		{"capacity reached", true},
		{"connection refused", false},
		{"timeout waiting for response", false},
		{"pi: process closed stdout", false},
		{"claude: exit: exit status 1", false},
	}
	for _, tc := range tests {
		got := IsRetryableError(fmt.Errorf("%s", tc.err))
		if got != tc.want {
			t.Errorf("IsRetryableError(%q) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

func TestIsRetryableErrorNil(t *testing.T) {
	if IsRetryableError(nil) {
		t.Error("expected false for nil error")
	}
}

func TestSendPromptAutoFailover(t *testing.T) {
	r := NewRegistry()
	r.Register("primary", &failAgent{err: fmt.Errorf("rate limit exceeded (429)")}, 0)
	r.Register("fallback", &EchoAgent{}, 1)
	r.HealthCheckAll(context.Background())

	var switched bool
	r.OnSwitch = func(from, to string) {
		switched = true
		if from != "primary" || to != "fallback" {
			t.Errorf("switch = %s→%s, want primary→fallback", from, to)
		}
	}

	resp, err := r.SendPrompt(context.Background(), "hello")
	if err != nil {
		t.Fatalf("expected failover to succeed, got error: %v", err)
	}
	if resp != "echo: hello" {
		t.Errorf("response = %q, want 'echo: hello'", resp)
	}
	if !switched {
		t.Error("expected OnSwitch to be called")
	}
	if r.Active() != "fallback" {
		t.Errorf("active after failover = %q, want 'fallback'", r.Active())
	}
}

func TestSendPromptNoFailoverOnNonRetryable(t *testing.T) {
	r := NewRegistry()
	r.Register("primary", &failAgent{err: fmt.Errorf("connection refused")}, 0)
	r.Register("fallback", &EchoAgent{}, 1)
	r.HealthCheckAll(context.Background())

	_, err := r.SendPrompt(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for non-retryable failure")
	}
	// primary should still be active (not marked unhealthy)
	if r.Active() != "primary" {
		t.Errorf("active = %q, want 'primary' (non-retryable shouldn't trigger failover)", r.Active())
	}
}

func TestSendPromptAllBackendsExhausted(t *testing.T) {
	r := NewRegistry()
	r.Register("a", &failAgent{err: fmt.Errorf("rate limit exceeded")}, 0)
	r.Register("b", &failAgent{err: fmt.Errorf("quota exhausted")}, 1)
	r.HealthCheckAll(context.Background())

	_, err := r.SendPrompt(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error when all backends fail")
	}
	// after first fails, second becomes active and also fails
	// but the second failure just returns the error (no more backends to try)
}

func TestCooldownRecovery(t *testing.T) {
	r := NewRegistry()
	r.cooldown = 10 * time.Millisecond // short cooldown for testing
	r.Register("primary", &EchoAgent{}, 0)
	r.Register("fallback", &EchoAgent{}, 1)
	r.HealthCheckAll(context.Background())

	r.MarkUnhealthy("primary", "rate limited")
	if r.Active() != "fallback" {
		t.Fatalf("expected fallback, got %q", r.Active())
	}

	// wait for cooldown
	time.Sleep(20 * time.Millisecond)

	// trigger reselection via any operation that calls selectActiveLocked
	// MarkHealthy on fallback is a no-op but triggers reselect
	r.MarkHealthy("fallback")

	if r.Active() != "primary" {
		t.Errorf("after cooldown, active = %q, want 'primary'", r.Active())
	}
}

func TestCooldownNotYetExpired(t *testing.T) {
	r := NewRegistry()
	r.cooldown = 1 * time.Hour // long cooldown — won't expire in test
	r.Register("primary", &EchoAgent{}, 0)
	r.Register("fallback", &EchoAgent{}, 1)
	r.HealthCheckAll(context.Background())

	r.MarkUnhealthy("primary", "rate limited")
	if r.Active() != "fallback" {
		t.Fatalf("expected fallback, got %q", r.Active())
	}

	// trigger reselection — primary should NOT recover yet
	r.MarkHealthy("fallback")
	if r.Active() != "fallback" {
		t.Errorf("before cooldown expires, active = %q, want 'fallback'", r.Active())
	}
}

func TestOnSwitchNotCalledWithoutFailover(t *testing.T) {
	r := NewRegistry()
	r.Register("echo", &EchoAgent{}, 0)
	r.HealthCheckAll(context.Background())

	called := false
	r.OnSwitch = func(_, _ string) { called = true }

	_, err := r.SendPrompt(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Error("OnSwitch should not be called when no failover happens")
	}
}

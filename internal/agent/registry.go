package agent

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"visor/internal/observability"
)

// Backend wraps an Agent with metadata for the registry.
type Backend struct {
	Name     string
	Agent    Agent
	Priority int    // lower = higher priority
	Healthy  bool
	LastErr  string
}

// Registry manages multiple backends with priority-based selection.
// It implements the Agent interface by proxying to the active backend.
type Registry struct {
	backends []*Backend // sorted by priority (lowest first = highest priority)
	active   *Backend
	mu       sync.RWMutex
	log      *observability.Logger
}

func NewRegistry() *Registry {
	return &Registry{
		log: observability.Component("agent.registry"),
	}
}

// Register adds a backend at the given priority. Lower priority = preferred.
func (r *Registry) Register(name string, a Agent, priority int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	b := &Backend{
		Name:     name,
		Agent:    a,
		Priority: priority,
		Healthy:  true, // assume healthy until checked
	}
	r.backends = append(r.backends, b)

	// keep sorted by priority
	for i := len(r.backends) - 1; i > 0; i-- {
		if r.backends[i].Priority < r.backends[i-1].Priority {
			r.backends[i], r.backends[i-1] = r.backends[i-1], r.backends[i]
		}
	}

	r.log.Info(nil, "backend registered", "name", name, "priority", priority)
}

// HealthCheckAll runs health checks on all backends and selects the best one.
func (r *Registry) HealthCheckAll(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, b := range r.backends {
		healthy, reason := checkHealth(ctx, b.Name)
		b.Healthy = healthy
		if !healthy {
			b.LastErr = reason
			r.log.Warn(ctx, "backend unhealthy", "name", b.Name, "reason", reason)
		} else {
			b.LastErr = ""
			r.log.Info(ctx, "backend healthy", "name", b.Name)
		}
	}

	r.selectActiveLocked(ctx)
}

// selectActiveLocked picks the highest-priority healthy backend. Must hold mu.
func (r *Registry) selectActiveLocked(ctx context.Context) {
	old := r.active
	r.active = nil
	for _, b := range r.backends {
		if b.Healthy {
			r.active = b
			break
		}
	}

	if r.active == nil {
		r.log.Error(ctx, "no healthy backends available")
		return
	}

	if old == nil || old.Name != r.active.Name {
		r.log.Info(ctx, "active backend changed", "backend", r.active.Name, "priority", r.active.Priority)
	}
}

// Active returns the name of the currently active backend.
func (r *Registry) Active() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.active == nil {
		return ""
	}
	return r.active.Name
}

// MarkUnhealthy marks a backend as unhealthy and triggers reselection.
func (r *Registry) MarkUnhealthy(name, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, b := range r.backends {
		if b.Name == name {
			b.Healthy = false
			b.LastErr = reason
			r.log.Warn(nil, "backend marked unhealthy", "name", name, "reason", reason)
			break
		}
	}

	r.selectActiveLocked(nil)
}

// MarkHealthy marks a backend as healthy and triggers reselection.
func (r *Registry) MarkHealthy(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, b := range r.backends {
		if b.Name == name {
			b.Healthy = true
			b.LastErr = ""
			break
		}
	}

	r.selectActiveLocked(nil)
}

// Status returns info about all backends.
func (r *Registry) Status() []BackendStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	active := ""
	if r.active != nil {
		active = r.active.Name
	}

	out := make([]BackendStatus, len(r.backends))
	for i, b := range r.backends {
		out[i] = BackendStatus{
			Name:     b.Name,
			Priority: b.Priority,
			Healthy:  b.Healthy,
			Active:   b.Name == active,
			LastErr:  b.LastErr,
		}
	}
	return out
}

type BackendStatus struct {
	Name     string
	Priority int
	Healthy  bool
	Active   bool
	LastErr  string
}

// SendPrompt implements Agent by proxying to the active backend.
func (r *Registry) SendPrompt(ctx context.Context, prompt string) (string, error) {
	r.mu.RLock()
	active := r.active
	r.mu.RUnlock()

	if active == nil {
		return "", fmt.Errorf("no healthy backend available")
	}

	r.log.Debug(ctx, "routing prompt to backend", "backend", active.Name)
	return active.Agent.SendPrompt(ctx, prompt)
}

// Close shuts down all registered backends.
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []string
	for _, b := range r.backends {
		if err := b.Agent.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", b.Name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close backends: %s", strings.Join(errs, "; "))
	}
	return nil
}

// checkHealth verifies a backend is available.
// For CLI-based backends, checks if the binary exists on PATH.
func checkHealth(ctx context.Context, name string) (healthy bool, reason string) {
	switch name {
	case "pi":
		return checkCLI("pi")
	case "claude":
		return checkCLI("claude")
	case "echo":
		return true, ""
	default:
		// unknown backends — assume healthy, let runtime errors handle it
		return true, ""
	}
}

func checkCLI(binary string) (bool, string) {
	path, err := exec.LookPath(binary)
	if err != nil {
		return false, fmt.Sprintf("%s CLI not found on PATH", binary)
	}

	// quick version check to verify it actually runs
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Sprintf("%s --version failed: %s", binary, strings.TrimSpace(string(output)))
	}

	return true, ""
}

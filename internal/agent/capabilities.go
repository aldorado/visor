package agent

import "fmt"

// BackendLabeler exposes a human-readable backend label shown in response footers.
// Example: "pi/codex".
type BackendLabeler interface {
	BackendLabel() string
}

// ModelSwitcher allows runtime model switching for backends that support it.
type ModelSwitcher interface {
	SetModel(model string) error
	Model() string
}

type ModelStatus struct {
	Backend        string
	Model          string
	Provider       string
	Source         string
	StateModel     string
	StateProvider  string
	StateUpdatedAt string
}

type ModelStatusProvider interface {
	ModelStatus() ModelStatus
}

func switchModel(a Agent, model string) error {
	ms, ok := a.(ModelSwitcher)
	if !ok {
		return fmt.Errorf("active backend does not support model switching")
	}
	return ms.SetModel(model)
}

func currentModel(a Agent) string {
	ms, ok := a.(ModelSwitcher)
	if !ok {
		return ""
	}
	return ms.Model()
}

func backendLabel(defaultName string, a Agent) string {
	l, ok := a.(BackendLabeler)
	if !ok {
		return defaultName
	}
	label := l.BackendLabel()
	if label == "" {
		return defaultName
	}
	return label
}

func modelStatus(defaultBackend string, a Agent) ModelStatus {
	msp, ok := a.(ModelStatusProvider)
	if ok {
		ms := msp.ModelStatus()
		if ms.Backend == "" {
			ms.Backend = defaultBackend
		}
		return ms
	}
	return ModelStatus{Backend: defaultBackend, Model: currentModel(a), Source: "runtime"}
}

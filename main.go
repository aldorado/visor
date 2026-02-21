package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"visor/internal/agent"
	"visor/internal/branding"
	"visor/internal/config"
	"visor/internal/observability"
	"visor/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	observability.Init(observability.LogConfig{Level: cfg.LogLevel, Verbose: cfg.LogVerbose})
	log := observability.Component("main")
	fmt.Print(branding.StartupASCII)
	fmt.Println("visor startup sequence engaged 🛸")

	shutdownOTel, err := observability.InitOTel(context.Background(), observability.OTelConfig{
		Enabled:     cfg.OTELEnabled,
		Endpoint:    cfg.OTELEndpoint,
		ServiceName: cfg.OTELServiceName,
		Environment: cfg.OTELEnvironment,
		Insecure:    cfg.OTELInsecure,
	})
	if err != nil {
		log.Error(context.Background(), "otel init failed", "error", err.Error())
		os.Exit(1)
	}
	defer shutdownOTel(context.Background())

	a, err := createAgents(cfg)
	if err != nil {
		log.Error(context.Background(), "agent init failed", "error", err.Error())
		os.Exit(1)
	}
	defer a.Close()

	srv := server.New(cfg, a)
	if err := srv.ListenAndServe(); err != nil {
		log.Error(context.Background(), "server failed", "error", err.Error())
		os.Exit(1)
	}
}

func createAgents(cfg *config.Config) (agent.Agent, error) {
	// single backend (backward compat)
	if len(cfg.AgentBackends) <= 1 {
		return createSingleAgent(cfg.AgentBackend)
	}

	// multi-backend registry
	registry := agent.NewRegistry()
	for i, name := range cfg.AgentBackends {
		a, err := createSingleAgent(name)
		if err != nil {
			return nil, fmt.Errorf("backend %s: %w", name, err)
		}
		registry.Register(name, a, i)
	}
	registry.HealthCheckAll(context.Background())
	return registry, nil
}

func createSingleAgent(name string) (agent.Agent, error) {
	switch name {
	case "pi":
		pi := agent.NewPiAgent(agent.ProcessConfig{
			RestartDelay:  3 * time.Second,
			PromptTimeout: 6 * time.Minute,
		})
		if err := pi.Start(); err != nil {
			return nil, err
		}
		return pi, nil
	case "claude":
		return agent.NewClaudeAgent(5 * time.Minute), nil
	case "echo":
		return &agent.EchoAgent{}, nil
	default:
		return nil, fmt.Errorf("unknown agent backend: %s", name)
	}
}

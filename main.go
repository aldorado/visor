package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"visor/internal/agent"
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

	a, err := createAgent(cfg)
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

func createAgent(cfg *config.Config) (agent.Agent, error) {
	switch cfg.AgentBackend {
	case "pi":
		pi := agent.NewPiAgent(agent.ProcessConfig{
			RestartDelay:  3 * time.Second,
			PromptTimeout: 2 * time.Minute,
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
		return nil, fmt.Errorf("unknown agent backend: %s", cfg.AgentBackend)
	}
}

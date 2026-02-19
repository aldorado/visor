package main

import (
	"log"
	"time"

	"visor/internal/agent"
	"visor/internal/config"
	"visor/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	a, err := createAgent(cfg)
	if err != nil {
		log.Fatalf("agent: %v", err)
	}
	defer a.Close()

	srv := server.New(cfg, a)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
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
		log.Fatalf("unknown agent backend: %s", cfg.AgentBackend)
		return nil, nil
	}
}

package main

import (
	"log"

	"visor/internal/agent"
	"visor/internal/config"
	"visor/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// echo agent for now — will be replaced by pi/claude adapter in M2-I2
	a := &agent.EchoAgent{}

	srv := server.New(cfg, a)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

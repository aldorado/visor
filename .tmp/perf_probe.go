package main

import (
	"context"
	"fmt"
	"time"

	"visor/internal/agent"
)

func main() {
	p := agent.NewPiAgent(agent.ProcessConfig{RestartDelay: 3 * time.Second})
	defer p.Close()

	prompts := []string{
		"antworte nur mit ok",
		"nenn nur eine zahl zwischen 1 und 9",
		"schreib exakt: done",
	}

	for i, prompt := range prompts {
		start := time.Now()
		resp, err := p.SendPrompt(context.Background(), prompt)
		dur := time.Since(start)
		if err != nil {
			fmt.Printf("run %d error after %s: %v\n", i+1, dur, err)
			continue
		}
		if len(resp) > 120 {
			resp = resp[:120] + "..."
		}
		fmt.Printf("run %d duration=%s response=%q\n", i+1, dur, resp)
	}
}

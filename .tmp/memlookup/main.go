package main

import (
	"fmt"
	"os"
	"visor/internal/memory"
)

func main() {
	m, err := memory.NewManager("data", os.Getenv("OPENAI_API_KEY"))
	if err != nil {
		panic(err)
	}
	out, err := m.Lookup("gemini cli readiness visor", 5)
	if err != nil {
		panic(err)
	}
	fmt.Print(out)
}

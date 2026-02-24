package main

import (
	"fmt"
	"os"
	"time"

	"visor/internal/memory"
)

func main() {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		fmt.Println("OPENAI_API_KEY missing; skipping")
		return
	}
	m, err := memory.NewManager("data", key)
	if err != nil {
		panic(err)
	}

	t0 := time.Now()
	err = m.Save([]string{"probe memory speed check " + time.Now().Format(time.RFC3339Nano)})
	fmt.Printf("save: %s err=%v\n", time.Since(t0), err)

	t1 := time.Now()
	ctx, err := m.Lookup("probe memory speed", 5)
	fmt.Printf("lookup: %s err=%v len=%d\n", time.Since(t1), err, len(ctx))
}

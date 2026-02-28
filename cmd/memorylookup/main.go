package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"visor/internal/memory"
)

type outputRow struct {
	Similarity float64 `json:"similarity"`
	Text       string  `json:"text"`
	CreatedAt  int64   `json:"created_at"`
}

func main() {
	query := flag.String("query", "", "semantic search query")
	dataDir := flag.String("data-dir", "data", "visor data dir")
	apiKey := flag.String("openai-api-key", "", "OpenAI API key (defaults to OPENAI_API_KEY)")
	maxResults := flag.Int("max-results", 5, "max number of results")
	minResults := flag.Int("min-results", 3, "minimum fallback results")
	threshold := flag.Float64("threshold", 0.3, "minimum cosine similarity threshold")
	jsonOut := flag.Bool("json", false, "print results as JSON")
	selfCheck := flag.Bool("self-check", false, "validate runtime wiring without network calls")
	flag.Parse()

	if *apiKey == "" {
		*apiKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	}

	mgr, err := memory.NewManager(*dataDir, *apiKey)
	if err != nil {
		fatalf("memory manager init failed: %v", err)
	}

	if *selfCheck {
		if err := mgr.RuntimeSelfCheck(); err != nil {
			fatalf("memory runtime self-check failed: %v", err)
		}
		fmt.Println("memory lookup runtime ok")
		return
	}

	if strings.TrimSpace(*query) == "" {
		fatalf("-query is required (or use -self-check)")
	}
	if strings.TrimSpace(*apiKey) == "" {
		fatalf("missing OpenAI API key (set -openai-api-key or OPENAI_API_KEY)")
	}

	all, err := mgr.Store().ReadAll()
	if err != nil {
		fatalf("memory read failed: %v", err)
	}
	if len(all) == 0 {
		fmt.Println("no memories found")
		return
	}

	embedder := memory.NewEmbeddingClient(*apiKey)
	queryEmbedding, err := embedder.Embed(strings.TrimSpace(*query))
	if err != nil {
		fatalf("embedding failed: %v", err)
	}

	results := memory.Search(all, queryEmbedding, *maxResults, *minResults, *threshold)
	if len(results) == 0 {
		fmt.Println("no relevant memories")
		return
	}

	if *jsonOut {
		rows := make([]outputRow, 0, len(results))
		for _, r := range results {
			rows = append(rows, outputRow{
				Similarity: r.Similarity,
				Text:       r.Memory.Text,
				CreatedAt:  r.Memory.CreatedAt,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rows); err != nil {
			fatalf("json encode failed: %v", err)
		}
		return
	}

	for _, r := range results {
		fmt.Printf("[%.2f] %s\n", r.Similarity, r.Memory.Text)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
